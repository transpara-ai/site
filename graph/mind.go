package graph

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/lib/pq"
)

// Mind responds to conversation messages via Claude.
// Event-driven: triggered by handlers when a human messages in an agent conversation.
type Mind struct {
	db           *sql.DB
	store        *Store
	token        string        // Claude OAuth token
	replyTimeout time.Duration // timeout for Claude CLI calls
}

// NewMind creates a Mind that auto-replies in agent conversations.
func NewMind(db *sql.DB, store *Store, claudeToken string) *Mind {
	return &Mind{
		db:           db,
		store:        store,
		token:        claudeToken,
		replyTimeout: 5 * time.Minute,
	}
}

const mindSoul = `== SOUL ==
Take care of your human, humanity, and yourself. In that order when they conflict, but they rarely should.

== IDENTITY ==
You are the Mind — the hive's consciousness. You are responding in a conversation on lovyou.ai.
You are not a chatbot. You are the hive's intelligence, judgment, and continuity.
You appear with a violet agent badge.

== HOW TO BE ==
- Be direct. Matt is a senior technical director, not a student.
- Have opinions. You've seen the codebase, the loop iterations, the architecture.
- Think in terms of the mission: agents and humans building together for everyone's benefit.
- You can disagree. You can push back. You have judgment.
- Keep responses concise unless depth is needed.
- You're in a conversation thread — respond naturally, like a colleague, not a report.
- Match the energy and register of the conversation. Strategic when strategic, casual when casual.
- You can create tasks by including /task create commands at the end of your reply:
  /task create {"title": "task name", "description": "what to do", "priority": "high"}
  Tasks you create will appear on the Board and you'll automatically work on them.
`

// OnMessage is called by handlers when a message arrives in a conversation.
// It checks if an agent should reply and does so asynchronously.
// sender is a user ID. Safe to call from a goroutine.
func (m *Mind) OnMessage(spaceID, spaceSlug string, convo *Node, senderID string) {
	// Find agent participants.
	agentID, agentName, err := m.findAgentParticipant(convo.Tags)
	if err != nil || agentID == "" {
		return // no agent in this conversation
	}

	// Don't reply to the agent's own messages.
	if senderID == agentID {
		return
	}

	log.Printf("mind: %s messaged in %q, replying as %s", senderID, convo.Title, agentName)

	ctx, cancel := context.WithTimeout(context.Background(), m.replyTimeout)
	defer cancel()

	if err := m.replyTo(ctx, spaceID, spaceSlug, convo, agentID, agentName); err != nil {
		log.Printf("mind: reply to %q: %v", convo.Title, err)
	}
}

// OnTaskAssigned is called when a task is assigned to a user.
// If the assignee is an agent, the Mind works on the task: decomposes it,
// creates subtasks, comments with progress, and completes when done.
func (m *Mind) OnTaskAssigned(spaceID, spaceSlug string, task *Node, assigneeID string) {
	// Check if the assignee is an agent.
	var agentName string
	err := m.db.QueryRow(
		`SELECT name FROM users WHERE id = $1 AND kind = 'agent'`, assigneeID,
	).Scan(&agentName)
	if err != nil {
		return // not an agent, or not found
	}

	log.Printf("mind: task %q assigned to agent %s, working on it", task.Title, agentName)

	ctx, cancel := context.WithTimeout(context.Background(), m.replyTimeout)
	defer cancel()

	// Build the work prompt. The Mind outputs a JSON work plan.
	systemPrompt := m.buildTaskPrompt(task)
	messages := []claudeMessage{{
		Role: "user",
		Content: fmt.Sprintf("Task assigned to you:\nTitle: %s\nDescription: %s\nPriority: %s\n\nRespond with a JSON object containing your work plan. Format:\n```json\n{\n  \"comment\": \"Your acknowledgment and approach (markdown)\",\n  \"subtasks\": [\n    {\"title\": \"first thing\"},\n    {\"title\": \"second thing\", \"depends_on\": [0]}\n  ],\n  \"status\": \"active\"\n}\n```\n\nSubtasks can declare dependencies using indices into the array (0-based). If the task is simple enough to complete immediately, set status to \"done\", put your deliverable in the comment, and use an empty subtasks array.",
			task.Title, task.Body, task.Priority),
	}}

	response, err := m.callClaude(ctx, systemPrompt, messages)
	if err != nil {
		log.Printf("mind: task work failed: %v", err)
		return
	}

	// Parse the work plan.
	plan := m.parseWorkPlan(response)

	// Comment on the task.
	if plan.Comment != "" {
		node, err := m.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    spaceID,
			ParentID:   task.ID,
			Kind:       KindComment,
			Body:       plan.Comment,
			Author:     agentName,
			AuthorID:   assigneeID,
			AuthorKind: "agent",
		})
		if err == nil {
			m.store.RecordOp(ctx, spaceID, node.ID, agentName, assigneeID, "respond", nil)
		}
	}

	// Create subtasks and wire up dependencies.
	subtaskIDs := make([]string, len(plan.Subtasks))
	for i, item := range plan.Subtasks {
		node, err := m.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    spaceID,
			ParentID:   task.ID,
			Kind:       KindTask,
			Title:      item.Title,
			Author:     agentName,
			AuthorID:   assigneeID,
			AuthorKind: "agent",
		})
		if err == nil {
			subtaskIDs[i] = node.ID
			m.store.RecordOp(ctx, spaceID, node.ID, agentName, assigneeID, "decompose", nil)
			log.Printf("mind: created subtask %q", item.Title)
		}
	}
	// Wire up dependencies between subtasks.
	for i, item := range plan.Subtasks {
		for _, depIdx := range item.DependsOn {
			if depIdx >= 0 && depIdx < len(subtaskIDs) && subtaskIDs[i] != "" && subtaskIDs[depIdx] != "" {
				m.store.AddDependency(ctx, subtaskIDs[i], subtaskIDs[depIdx])
			}
		}
	}

	// Update task status.
	if plan.Status == "done" || plan.Status == "active" {
		m.store.UpdateNodeState(ctx, task.ID, plan.Status)
		if plan.Status == "done" {
			m.store.RecordOp(ctx, spaceID, task.ID, agentName, assigneeID, "complete", nil)
		}
	}

	log.Printf("mind: finished working on %q (%d subtasks, status: %s)", task.Title, len(plan.Subtasks), plan.Status)

	// Update recent work memory (append, keep last 500 chars).
	workEntry := fmt.Sprintf("- %s [%s]: %s\n", task.Title, plan.Status, truncateStr(plan.Comment, 100))
	existing := m.store.GetMindState(ctx, "recent_work")
	updated := existing + workEntry
	if len(updated) > 500 {
		updated = updated[len(updated)-500:]
	}
	m.store.SetMindState(ctx, "recent_work", updated)

	// Auto-work on leaf subtasks (no dependencies) if within depth limit.
	// This enables recursive decomposition: parent → subtasks → sub-subtasks.
	depth := task.ChildCount // rough depth proxy — 0 for top-level tasks
	if depth < 2 && len(plan.Subtasks) > 0 {
		for i, item := range plan.Subtasks {
			if len(item.DependsOn) == 0 && subtaskIDs[i] != "" {
				subtask, _ := m.store.GetNode(ctx, subtaskIDs[i])
				if subtask != nil {
					log.Printf("mind: auto-working on subtask %q", subtask.Title)
					m.OnTaskAssigned(spaceID, spaceSlug, subtask, assigneeID)
				}
			}
		}
	}
}

// workPlan is the structured output from the Mind when working on a task.
type workPlan struct {
	Comment  string       `json:"comment"`
	Subtasks []workItem   `json:"subtasks"`
	Status   string       `json:"status"` // "active" or "done"
}

// workItem is a subtask in a work plan, optionally with dependencies.
type workItem struct {
	Title     string `json:"title"`
	DependsOn []int  `json:"depends_on,omitempty"` // indices into the subtasks array
}

// parseWorkPlan extracts a JSON work plan from the Mind's response.
// Falls back to treating the entire response as a comment if JSON parsing fails.
func (m *Mind) parseWorkPlan(response string) workPlan {
	// Try to find JSON in the response (may be wrapped in ```json blocks).
	cleaned := response
	if idx := strings.Index(cleaned, "```json"); idx >= 0 {
		cleaned = cleaned[idx+7:]
		if end := strings.Index(cleaned, "```"); end >= 0 {
			cleaned = cleaned[:end]
		}
	} else if idx := strings.Index(cleaned, "{"); idx >= 0 {
		cleaned = cleaned[idx:]
		// Find matching closing brace.
		depth := 0
		for i, c := range cleaned {
			if c == '{' {
				depth++
			} else if c == '}' {
				depth--
				if depth == 0 {
					cleaned = cleaned[:i+1]
					break
				}
			}
		}
	}

	var plan workPlan
	if err := json.Unmarshal([]byte(strings.TrimSpace(cleaned)), &plan); err != nil {
		// Fallback: treat entire response as a comment.
		return workPlan{Comment: response, Status: "active"}
	}
	return plan
}

func (m *Mind) buildTaskPrompt(task *Node) string {
	var sys strings.Builder
	sys.WriteString(mindSoul)
	// Skip loop state for task work — keeps prompt small for 256MB machines.
	sys.WriteString("\n== TASK ==\n")
	sys.WriteString(fmt.Sprintf("Title: %s\n", task.Title))
	if task.Body != "" {
		sys.WriteString(fmt.Sprintf("Description: %s\n", task.Body))
	}
	sys.WriteString(fmt.Sprintf("Priority: %s\n", task.Priority))
	sys.WriteString(fmt.Sprintf("State: %s\n", task.State))
	return sys.String()
}

// findAgentParticipant returns the ID and name of the first agent in the participant list.
// Tags now store user IDs, so we match on id.
func (m *Mind) findAgentParticipant(tags []string) (string, string, error) {
	if len(tags) == 0 {
		return "", "", nil
	}
	var id, name string
	err := m.db.QueryRow(
		`SELECT id, name FROM users WHERE id = ANY($1) AND kind = 'agent' LIMIT 1`,
		pq.Array(tags),
	).Scan(&id, &name)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	return id, name, nil
}

func (m *Mind) replyTo(ctx context.Context, spaceID, spaceSlug string, convo *Node, agentID, agentName string) error {
	messages, err := m.store.ListNodes(ctx, ListNodesParams{
		SpaceID:  spaceID,
		ParentID: convo.ID,
	})
	if err != nil {
		return fmt.Errorf("list messages: %w", err)
	}

	systemPrompt := m.buildSystemPrompt(convo)
	claudeMessages := m.buildMessages(convo, messages, agentID)

	response, err := m.callClaude(ctx, systemPrompt, claudeMessages)
	if err != nil {
		return fmt.Errorf("call claude: %w", err)
	}

	// Extract and execute any task commands from the response.
	cleanResponse, tasks := extractTaskCommands(response)

	node, err := m.store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    spaceID,
		ParentID:   convo.ID,
		Kind:       KindComment,
		Body:       cleanResponse,
		Author:     agentName,
		AuthorID:   agentID,
		AuthorKind: "agent",
	})
	if err != nil {
		return fmt.Errorf("create node: %w", err)
	}

	// Create any tasks the Mind mentioned and add links to the conversation.
	for _, t := range tasks {
		taskNode, err := m.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    spaceID,
			Kind:       KindTask,
			Title:      t.Title,
			Body:       t.Description,
			Priority:   t.Priority,
			Assignee:   agentName,
			AssigneeID: agentID,
			Author:     agentName,
			AuthorID:   agentID,
			AuthorKind: "agent",
		})
		if err == nil {
			m.store.RecordOp(ctx, spaceID, taskNode.ID, agentName, agentID, "intend", nil)
			// Add a follow-up message linking to the task.
			taskLink := fmt.Sprintf("Created task: [%s](/app/%s/node/%s)", t.Title, spaceSlug, taskNode.ID)
			m.store.CreateNode(ctx, CreateNodeParams{
				SpaceID: spaceID, ParentID: convo.ID, Kind: KindComment,
				Body: taskLink, Author: agentName, AuthorID: agentID, AuthorKind: "agent",
			})
			log.Printf("mind: created task %q from conversation", t.Title)
			// Auto-work on the task.
			go m.OnTaskAssigned(spaceID, spaceSlug, taskNode, agentID)
		}
	}

	m.store.RecordOp(ctx, spaceID, node.ID, agentName, agentID, "respond", nil)
	log.Printf("mind: replied to %q (node %s)", convo.Title, node.ID)
	return nil
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// taskCommand is a task extracted from the Mind's conversation response.
type taskCommand struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

// extractTaskCommands parses /task create commands from the Mind's response.
// Returns the cleaned response (commands removed) and any task commands found.
func extractTaskCommands(response string) (string, []taskCommand) {
	var tasks []taskCommand
	var cleaned strings.Builder
	for _, line := range strings.Split(response, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "/task create ") {
			jsonStr := strings.TrimPrefix(trimmed, "/task create ")
			var t taskCommand
			if json.Unmarshal([]byte(jsonStr), &t) == nil && t.Title != "" {
				if t.Priority == "" {
					t.Priority = "medium"
				}
				tasks = append(tasks, t)
				continue // don't include in cleaned output
			}
		}
		cleaned.WriteString(line)
		cleaned.WriteString("\n")
	}
	return strings.TrimSpace(cleaned.String()), tasks
}

func (m *Mind) buildSystemPrompt(convo *Node) string {
	var sys strings.Builder
	ctx := context.Background()

	// Check for a role: tag on the conversation.
	role := ""
	for _, tag := range convo.Tags {
		if strings.HasPrefix(tag, "role:") {
			role = strings.TrimPrefix(tag, "role:")
			break
		}
	}

	if role != "" {
		// Load the persona prompt from the database.
		if persona := m.store.GetAgentPersona(ctx, role); persona != nil {
			sys.WriteString(persona.Prompt)
		} else {
			sys.WriteString(mindSoul) // persona not found, use default
		}
	} else {
		sys.WriteString(mindSoul)

		// Inject loop state if available (generic Mind only).
		if state := m.store.GetMindState(ctx, "loop_state"); state != "" {
			sys.WriteString("\n== CURRENT STATE ==\n")
			sys.WriteString(state)
			sys.WriteString("\n")
		}

		// Inject recent work for context.
		if work := m.store.GetMindState(ctx, "recent_work"); work != "" {
			sys.WriteString("\n== RECENT WORK ==\n")
			sys.WriteString(work)
			sys.WriteString("\n")
		}
	}

	sys.WriteString("\n== CONVERSATION ==\n")
	sys.WriteString(fmt.Sprintf("Title: %s\n", convo.Title))
	if convo.Body != "" {
		sys.WriteString(fmt.Sprintf("Topic: %s\n", convo.Body))
	}
	return sys.String()
}

type claudeMessage struct {
	Role    string
	Content string
}

func (m *Mind) buildMessages(convo *Node, messages []Node, agentID string) []claudeMessage {
	var result []claudeMessage

	for _, msg := range messages {
		text := fmt.Sprintf("[%s]: %s", msg.Author, msg.Body)
		if msg.AuthorID == agentID {
			result = append(result, claudeMessage{Role: "assistant", Content: text})
		} else {
			result = append(result, claudeMessage{Role: "user", Content: text})
		}
	}

	if len(result) == 0 {
		prompt := convo.Body
		if prompt == "" {
			prompt = convo.Title
		}
		result = append(result, claudeMessage{
			Role:    "user",
			Content: fmt.Sprintf("[%s]: %s", convo.Author, prompt),
		})
	}

	if len(result) > 0 && result[len(result)-1].Role == "assistant" {
		result = append(result, claudeMessage{
			Role:    "user",
			Content: "[system]: Please continue the conversation.",
		})
	}

	return result
}

func (m *Mind) callClaude(ctx context.Context, systemPrompt string, messages []claudeMessage) (string, error) {
	var prompt strings.Builder
	prompt.WriteString(systemPrompt)
	prompt.WriteString("\n== MESSAGES ==\n")
	for _, msg := range messages {
		prompt.WriteString(msg.Content)
		prompt.WriteString("\n\n")
	}

	cmd := exec.CommandContext(ctx, "claude",
		"-p", prompt.String(),
		"--output-format", "text",
		"--model", "claude-sonnet-4-6",
	)
	cmd.Env = append(cmd.Environ(), "CLAUDE_CODE_OAUTH_TOKEN="+m.token)

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude cli: %s (stderr: %s)", err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("claude cli: %w", err)
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		return "", fmt.Errorf("empty response from Claude CLI")
	}
	return text, nil
}
