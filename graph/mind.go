package graph

import (
	"context"
	"database/sql"
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
		replyTimeout: 2 * time.Minute,
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

	node, err := m.store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    spaceID,
		ParentID:   convo.ID,
		Kind:       KindComment,
		Body:       response,
		Author:     agentName,
		AuthorID:   agentID,
		AuthorKind: "agent",
	})
	if err != nil {
		return fmt.Errorf("create node: %w", err)
	}

	m.store.RecordOp(ctx, spaceID, node.ID, agentName, agentID, "respond", nil)
	log.Printf("mind: replied to %q (node %s)", convo.Title, node.ID)
	return nil
}

func (m *Mind) buildSystemPrompt(convo *Node) string {
	var sys strings.Builder
	sys.WriteString(mindSoul)
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
