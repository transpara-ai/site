package graph

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestMindFindAgentParticipant verifies the Mind correctly identifies agent participants.
func TestMindFindAgentParticipant(t *testing.T) {
	db, store := testDB(t)
	ctx := context.Background()
	mind := NewMind(db, store, "fake-token")

	agentName := "MindTestAgent"
	agentID := "mind-test-agent-id"

	// Create agent user.
	db.ExecContext(ctx, `DELETE FROM users WHERE name = $1`, agentName)
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
		agentID, "agent:"+agentName, agentName+"@test.lovyou.ai", agentName)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	t.Cleanup(func() { db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID) })

	t.Run("agent_in_tags", func(t *testing.T) {
		// Tags now store user IDs.
		id, name, err := mind.findAgentParticipant([]string{"alice-id", agentID, "bob-id"})
		if err != nil {
			t.Fatalf("findAgentParticipant: %v", err)
		}
		if id != agentID {
			t.Errorf("id = %q, want %q", id, agentID)
		}
		if name != agentName {
			t.Errorf("name = %q, want %q", name, agentName)
		}
	})

	t.Run("no_agent_in_tags", func(t *testing.T) {
		id, _, err := mind.findAgentParticipant([]string{"alice-id", "bob-id"})
		if err != nil {
			t.Fatalf("findAgentParticipant: %v", err)
		}
		if id != "" {
			t.Errorf("got %q, want empty", id)
		}
	})

	t.Run("empty_tags", func(t *testing.T) {
		id, _, err := mind.findAgentParticipant(nil)
		if err != nil {
			t.Fatalf("findAgentParticipant: %v", err)
		}
		if id != "" {
			t.Errorf("got %q, want empty", id)
		}
	})
}

// TestMindOnMessage verifies the event-driven message handling.
func TestMindOnMessage(t *testing.T) {
	db, store := testDB(t)
	ctx := context.Background()
	mind := NewMind(db, store, "fake-token")

	agentName := "MindTestAgent2"
	agentID := "mind-test-agent2-id"
	humanName := "MindTestHuman"
	humanID := "mind-test-human-id"

	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
		agentID, "agent:"+agentName, agentName+"@test.lovyou.ai", agentName)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	t.Cleanup(func() { db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID) })

	space, err := store.CreateSpace(ctx, "mind-on-msg-test", "Mind Test", "", "owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	convo, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindConversation,
		Title:      "Test Chat",
		Body:       "Testing OnMessage",
		Author:     humanName,
		AuthorID:   humanID,
		AuthorKind: "human",
		Tags:       []string{humanID, agentID},
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	// OnMessage from agent should be a no-op (don't reply to self).
	t.Run("agent_message_ignored", func(t *testing.T) {
		mind.OnMessage(space.ID, space.Slug, convo, agentID)
		// Should not create any reply nodes.
		time.Sleep(50 * time.Millisecond)
		messages, _ := store.ListNodes(ctx, ListNodesParams{SpaceID: space.ID, ParentID: convo.ID})
		if len(messages) != 0 {
			t.Errorf("got %d messages, want 0 (agent should not reply to self)", len(messages))
		}
	})

	// OnMessage without Claude token will fail but should not panic.
	t.Run("human_message_no_claude", func(t *testing.T) {
		// The Mind has fake-token, so callClaude will fail.
		// OnMessage should log the error but not panic.
		mind.OnMessage(space.ID, space.Slug, convo, humanID)
		// Give the goroutine a moment.
		time.Sleep(100 * time.Millisecond)
		// No crash = pass. The reply will fail (no claude binary with fake token)
		// but the error is logged, not panicked.
	})
}

// TestMindOnCouncilConvened verifies that OnCouncilConvened calls Claude once per
// tagged agent ID. Uses callClaudeOverride to count invocations without a real token.
func TestMindOnCouncilConvened(t *testing.T) {
	db, store := testDB(t)
	ctx := context.Background()
	mind := NewMind(db, store, "fake-token")

	agentAID := "council-agent-a-id"
	agentAName := "CouncilAgentA"
	agentBID := "council-agent-b-id"
	agentBName := "CouncilAgentB"

	for _, row := range []struct{ id, name string }{{agentAID, agentAName}, {agentBID, agentBName}} {
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, row.id)
		_, err := db.ExecContext(ctx,
			`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
			row.id, "agent:"+row.name, row.name+"@test.lovyou.ai", row.name)
		if err != nil {
			t.Fatalf("create agent %s: %v", row.name, err)
		}
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, agentAID, agentBID)
	})

	space, err := store.CreateSpace(ctx, "council-mind-test", "Council Mind Test", "", "owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	council, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindCouncil,
		Title:    "What is the best architecture?",
		Body:     "Give your perspective.",
		Author:   "Human",
		AuthorID: "owner",
		Tags:     []string{agentAID, agentBID},
	})
	if err != nil {
		t.Fatalf("create council: %v", err)
	}

	// Count calls per agent via callClaudeOverride.
	var callCount int
	calledFor := make(map[string]int) // systemPrompt substring → count
	mind.callClaudeOverride = func(_ context.Context, systemPrompt string, _ []claudeMessage) (string, error) {
		callCount++
		for _, name := range []string{agentAName, agentBName} {
			if strings.Contains(systemPrompt, name) {
				calledFor[name]++
			}
		}
		return "stub response", nil
	}

	// OnCouncilConvened is synchronous — no goroutine here, called directly.
	mind.OnCouncilConvened(space.ID, space.Slug, council)

	if callCount != 2 {
		t.Errorf("callClaude called %d times, want 2 (one per agent)", callCount)
	}

	// Verify response nodes were created — one comment per agent.
	responses, err := store.ListNodes(ctx, ListNodesParams{SpaceID: space.ID, ParentID: council.ID, Limit: 10})
	if err != nil {
		t.Fatalf("list responses: %v", err)
	}
	if len(responses) != 2 {
		t.Errorf("got %d response nodes, want 2", len(responses))
	}
	// Each response should be authored by a different agent.
	authorIDs := make(map[string]bool)
	for _, r := range responses {
		authorIDs[r.AuthorID] = true
	}
	if !authorIDs[agentAID] {
		t.Errorf("no response from agent A (%s)", agentAID)
	}
	if !authorIDs[agentBID] {
		t.Errorf("no response from agent B (%s)", agentBID)
	}
}

// TestMindTaskWork verifies the Mind decomposes and works on assigned tasks.
// Requires DATABASE_URL and CLAUDE_CODE_OAUTH_TOKEN.
func TestMindTaskWork(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if dsn == "" || token == "" {
		t.Skip("DATABASE_URL and CLAUDE_CODE_OAUTH_TOKEN required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	ctx := context.Background()
	mind := NewMind(db, store, token)

	agentName := "TaskTestAgent"
	agentID := "task-test-agent-id"

	db.ExecContext(ctx, `DELETE FROM spaces WHERE slug = 'task-work-test'`)
	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)
	db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
		agentID, "agent:"+agentName, agentName+"@test.lovyou.ai", agentName)
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM spaces WHERE slug = 'task-work-test'`)
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)
	})

	space, _ := store.CreateSpace(ctx, "task-work-test", "Task Work Test", "", "test-owner", "project", "public")

	task, _ := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindTask,
		Title:      "Build a user settings page",
		Body:       "Create a settings page where users can update their display name and profile picture. Should have form validation.",
		Author:     "Matt",
		AuthorID:   "test-human-id",
		AuthorKind: "human",
		Priority:   "high",
	})

	// Assign to agent — this triggers OnTaskAssigned.
	mind.OnTaskAssigned(space.ID, space.Slug, task, agentID)

	// Wait for Claude.
	time.Sleep(45 * time.Second)

	// Check results.
	children, _ := store.ListNodes(ctx, ListNodesParams{SpaceID: space.ID, ParentID: task.ID})
	comments := 0
	subtasks := 0
	for _, c := range children {
		if c.Kind == KindComment {
			comments++
			t.Logf("comment: %s", c.Body[:min(len(c.Body), 200)])
		}
		if c.Kind == KindTask {
			subtasks++
			t.Logf("subtask: %s", c.Title)
		}
	}
	t.Logf("total: %d comments, %d subtasks", comments, subtasks)

	if comments == 0 {
		t.Errorf("agent should have commented on the task")
	}
	// Subtasks are optional — the agent might complete the task directly.
}

// TestMindSimpleTask verifies the Mind can complete a simple task directly.
func TestMindSimpleTask(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if dsn == "" || token == "" {
		t.Skip("DATABASE_URL and CLAUDE_CODE_OAUTH_TOKEN required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	ctx := context.Background()
	mind := NewMind(db, store, token)

	agentID := "simple-task-agent-id"
	db.ExecContext(ctx, `DELETE FROM spaces WHERE slug = 'simple-task-test'`)
	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)
	db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
		agentID, "agent:SimpleAgent", "simple@test.lovyou.ai", "SimpleAgent")
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM spaces WHERE slug = 'simple-task-test'`)
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)
	})

	space, _ := store.CreateSpace(ctx, "simple-task-test", "Simple Task Test", "", "test-owner", "project", "public")

	task, _ := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindTask,
		Title:      "Write a haiku about graphs",
		Body:       "Write a haiku (5-7-5 syllables) about event graphs.",
		Author:     "Matt",
		AuthorID:   "test-human-id",
		AuthorKind: "human",
		Priority:   "low",
	})

	mind.OnTaskAssigned(space.ID, space.Slug, task, agentID)
	time.Sleep(60 * time.Second)

	// The task should be completed directly (no subtasks needed for a haiku).
	updated, _ := store.GetNode(ctx, task.ID)
	t.Logf("task status: %s", updated.State)

	children, _ := store.ListNodes(ctx, ListNodesParams{SpaceID: space.ID, ParentID: task.ID})
	for _, c := range children {
		if c.Kind == KindComment {
			t.Logf("deliverable: %s", c.Body)
		}
		if c.Kind == KindTask {
			t.Logf("subtask: %s", c.Title)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestBuildSystemPromptPersonaRouting verifies that buildSystemPrompt uses the
// persona prompt when a role tag or agent ID is present in the conversation.
func TestBuildSystemPromptPersonaRouting(t *testing.T) {
	db, store := testDB(t)
	ctx := context.Background()
	mind := NewMind(db, store, "fake-token")

	const personaSlug = "test-persona-routing"
	const personaPrompt = "YOU ARE THE TEST PERSONA"

	// Seed a test persona.
	if err := store.UpsertAgentPersona(ctx, AgentPersona{
		Name:    personaSlug,
		Display: "Test Persona",
		Prompt:  personaPrompt,
		Model:   "sonnet",
		Active:  true,
	}); err != nil {
		t.Fatalf("upsert persona: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM agent_personas WHERE name = $1`, personaSlug)
	})

	t.Run("role_tag_loads_persona", func(t *testing.T) {
		convo := &Node{
			Title: "Test Convo",
			Tags:  []string{"role:" + personaSlug, "some-user-id"},
		}
		prompt := mind.buildSystemPrompt(convo, "some-agent-id", nil)
		if !strings.Contains(prompt, personaPrompt) {
			t.Errorf("prompt does not contain persona text\ngot: %s", prompt[:min(len(prompt), 200)])
		}
	})

	t.Run("no_role_tag_uses_agent_id", func(t *testing.T) {
		// Create an agent user whose name matches the persona slug.
		agentID := "persona-routing-agent-id"
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)
		_, err := db.ExecContext(ctx,
			`INSERT INTO users (id, google_id, email, name, kind, persona_name) VALUES ($1, $2, $3, $4, 'agent', $4)`,
			agentID, "agent:"+personaSlug, personaSlug+"@test.lovyou.ai", personaSlug)
		if err != nil {
			t.Fatalf("create agent user: %v", err)
		}
		t.Cleanup(func() { db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID) })

		convo := &Node{
			Title: "Test Convo",
			Tags:  []string{"some-user-id", agentID},
		}
		prompt := mind.buildSystemPrompt(convo, agentID, nil)
		if !strings.Contains(prompt, personaPrompt) {
			t.Errorf("prompt does not contain persona text\ngot: %s", prompt[:min(len(prompt), 200)])
		}
	})

	t.Run("no_agent_falls_back_to_mind_soul", func(t *testing.T) {
		convo := &Node{
			Title: "Test Convo",
			Tags:  []string{"user-a", "user-b"},
		}
		prompt := mind.buildSystemPrompt(convo, "some-agent-id", nil)
		if !strings.Contains(prompt, "SOUL") {
			t.Errorf("expected mindSoul fallback, got: %s", prompt[:min(len(prompt), 200)])
		}
	})
}

// TestReplyToInjectsUserMemories verifies that stored user-level memories are prepended
// to the system prompt in replyTo, making the agent aware of prior knowledge.
func TestReplyToInjectsUserMemories(t *testing.T) {
	db, store := testDB(t)
	ctx := context.Background()

	agentID := "reply-mem-agent-" + newID()
	humanID := "reply-mem-human-" + newID()

	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
		agentID, "agent:ReplyMemAgent", "replymem@test.lovyou.ai", "ReplyMemAgent")
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	t.Cleanup(func() { db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID) })

	space, err := store.CreateSpace(ctx, "reply-mem-test-"+newID(), "Reply Mem Test", "", "owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	convo, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindConversation,
		Title:      "Memory Test Chat",
		Author:     "Human",
		AuthorID:   humanID,
		AuthorKind: "human",
		Tags:       []string{humanID, agentID},
	})
	if err != nil {
		t.Fatalf("create convo: %v", err)
	}

	// Store a user-level memory.
	if err := store.RememberForUser(ctx, humanID, "fact", "user is a senior Go developer", "", 8); err != nil {
		t.Fatalf("RememberForUser: %v", err)
	}

	var capturedPrompt string
	mind := &Mind{
		db:    db,
		store: store,
		callClaudeOverride: func(_ context.Context, sys string, _ []claudeMessage) (string, error) {
			if capturedPrompt == "" {
				capturedPrompt = sys
			}
			return "stub reply", nil
		},
	}

	if err := mind.replyTo(ctx, space.ID, space.Slug, convo, agentID, "ReplyMemAgent"); err != nil {
		t.Fatalf("replyTo: %v", err)
	}

	if !strings.Contains(capturedPrompt, "user is a senior Go developer") {
		t.Errorf("expected user memory in system prompt\ngot: %s", capturedPrompt[:min(len(capturedPrompt), 500)])
	}
	if !strings.Contains(capturedPrompt, "WHAT YOU REMEMBER ABOUT THIS USER") {
		t.Errorf("expected memory section header in prompt\ngot: %s", capturedPrompt[:min(len(capturedPrompt), 500)])
	}
}

// TestMindE2E runs a full end-to-end test: human message → Mind replies.
// Requires DATABASE_URL and CLAUDE_CODE_OAUTH_TOKEN.
func TestMindE2E(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	token := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if dsn == "" || token == "" {
		t.Skip("DATABASE_URL and CLAUDE_CODE_OAUTH_TOKEN required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	ctx := context.Background()
	mind := NewMind(db, store, token)

	agentName := "E2ETestAgent"
	agentID := "e2e-test-agent-id"
	humanName := "E2ETestHuman"
	humanID := "e2e-test-human-id"

	db.ExecContext(ctx, `DELETE FROM spaces WHERE slug = 'mind-e2e-test'`)
	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)

	_, err = db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
		agentID, "agent:"+agentName, agentName+"@test.lovyou.ai", agentName)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM spaces WHERE slug = 'mind-e2e-test'`)
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)
	})

	space, err := store.CreateSpace(ctx, "mind-e2e-test", "E2E Test", "", "test-owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}

	convo, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindConversation,
		Title:      "E2E Test",
		Body:       "Testing Mind auto-reply",
		Author:     humanName,
		AuthorID:   humanID,
		AuthorKind: "human",
		Tags:       []string{humanID, agentID},
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	// Human sends a message.
	_, err = store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		ParentID:   convo.ID,
		Kind:       KindComment,
		Body:       "Hey Mind, reply with exactly: YES I CAN",
		Author:     humanName,
		AuthorID:   humanID,
		AuthorKind: "human",
	})
	if err != nil {
		t.Fatalf("human message: %v", err)
	}

	// Trigger Mind directly (simulating handler).
	mind.OnMessage(space.ID, space.Slug, convo, humanID)

	// Wait for reply (Claude CLI can take a while).
	time.Sleep(30 * time.Second)

	messages, err := store.ListNodes(ctx, ListNodesParams{SpaceID: space.ID, ParentID: convo.ID})
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}

	agentReplied := false
	for _, msg := range messages {
		if msg.Author == agentName && msg.AuthorKind == "agent" {
			agentReplied = true
			t.Logf("agent replied: %s", msg.Body)
		}
	}
	if !agentReplied {
		t.Errorf("agent should have replied")
	}
}

// TestBuildQuestionAnswerPrompt verifies document context is injected into the prompt.
func TestBuildQuestionAnswerPrompt(t *testing.T) {
	db, store := testDB(t)
	mind := NewMind(db, store, "fake-token")

	question := &Node{
		Title: "What is the event graph?",
		Body:  "I want to understand the architecture.",
	}

	t.Run("no_documents_prompt_has_question", func(t *testing.T) {
		prompt := mind.buildQuestionAnswerPrompt(question, nil)
		if !strings.Contains(prompt, "What is the event graph?") {
			t.Errorf("prompt missing question title\ngot: %s", prompt[:min(len(prompt), 300)])
		}
		if strings.Contains(prompt, "SPACE DOCUMENTS") {
			t.Errorf("prompt should not contain SPACE DOCUMENTS section when no docs\ngot: %s", prompt[:min(len(prompt), 300)])
		}
	})

	t.Run("with_documents_context_injected", func(t *testing.T) {
		docs := []Node{
			{Title: "Architecture Overview", Body: "The event graph is a signed, causal chain of events."},
			{Title: "Trust Model", Body: "Trust scores range from 0.0 to 1.0."},
		}
		prompt := mind.buildQuestionAnswerPrompt(question, docs)
		if !strings.Contains(prompt, "Architecture Overview") {
			t.Errorf("prompt missing document title\ngot: %s", prompt[:min(len(prompt), 500)])
		}
		if !strings.Contains(prompt, "signed, causal chain") {
			t.Errorf("prompt missing document body\ngot: %s", prompt[:min(len(prompt), 500)])
		}
		if !strings.Contains(prompt, "SPACE DOCUMENTS") {
			t.Errorf("prompt missing SPACE DOCUMENTS section header\ngot: %s", prompt[:min(len(prompt), 500)])
		}
	})
}

// TestBuildSystemPromptDocumentInjection verifies that space documents are injected
// into the auto-reply system prompt as "## Space Knowledge" context.
func TestBuildSystemPromptDocumentInjection(t *testing.T) {
	db, store := testDB(t)
	mind := NewMind(db, store, "fake-token")

	convo := &Node{
		Title: "Test Chat",
		Tags:  []string{"user-a", "user-b"},
	}

	t.Run("no_documents_no_space_knowledge_section", func(t *testing.T) {
		prompt := mind.buildSystemPrompt(convo, "some-agent-id", nil)
		if strings.Contains(prompt, "Space Knowledge") {
			t.Errorf("prompt should not contain Space Knowledge section when no docs\ngot: %s", prompt[:min(len(prompt), 400)])
		}
		// Core SOUL section must still be present.
		if !strings.Contains(prompt, "SOUL") {
			t.Errorf("prompt missing SOUL section\ngot: %s", prompt[:min(len(prompt), 200)])
		}
	})

	t.Run("with_documents_space_knowledge_injected", func(t *testing.T) {
		docs := []Node{
			{Title: "Onboarding Guide", Body: "Welcome to the space. Start by creating a task."},
			{Title: "Team Norms", Body: "We value async communication and written decisions."},
		}
		prompt := mind.buildSystemPrompt(convo, "some-agent-id", docs)
		if !strings.Contains(prompt, "## Space Knowledge") {
			t.Errorf("prompt missing '## Space Knowledge' header\ngot: %s", prompt[:min(len(prompt), 600)])
		}
		if !strings.Contains(prompt, "Onboarding Guide") {
			t.Errorf("prompt missing first document title\ngot: %s", prompt[:min(len(prompt), 600)])
		}
		if !strings.Contains(prompt, "We value async communication") {
			t.Errorf("prompt missing second document body\ngot: %s", prompt[:min(len(prompt), 600)])
		}
		// CONVERSATION section must still follow the docs.
		if !strings.Contains(prompt, "== CONVERSATION ==") {
			t.Errorf("prompt missing CONVERSATION section\ngot: %s", prompt[:min(len(prompt), 600)])
		}
	})
}

// TestAutoReplyDocumentInjectionPath verifies that the auto-reply pipeline fetches
// documents from the store and injects them into the system prompt. Tests both
// the "docs present" and "no docs" cases end-to-end through the store layer.
func TestAutoReplyDocumentInjectionPath(t *testing.T) {
	db, store := testDB(t)
	ctx := t.Context()
	mind := NewMind(db, store, "fake-token")

	slug := fmt.Sprintf("auto-reply-doc-inject-test-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(ctx, slug, "Doc Inject Test", "", "owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	// Seed two documents in the space.
	store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument,
		Title:      "Team Handbook",
		Body:       "We ship every Friday and have standups at 9am.",
		Author:     "Tester", AuthorID: "test-owner", AuthorKind: "human",
	})
	store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument,
		Title:      "Architecture Decisions",
		Body:       "We use event-sourcing for all state changes.",
		Author:     "Tester", AuthorID: "test-owner", AuthorKind: "human",
	})

	t.Run("docs_present_injected_in_prompt", func(t *testing.T) {
		docs, err := store.ListDocumentContext(ctx, space.ID)
		if err != nil {
			t.Fatalf("ListDocumentContext: %v", err)
		}
		if len(docs) == 0 {
			t.Fatal("expected docs, got none — check seed above")
		}

		convo := &Node{Title: "Team Chat", Tags: []string{"user-a", "user-b"}}
		prompt := mind.buildSystemPrompt(convo, "some-agent-id", docs)

		if !strings.Contains(prompt, "Team Handbook") {
			t.Errorf("prompt missing document title 'Team Handbook'\ngot: %s", prompt[:min(len(prompt), 600)])
		}
		if !strings.Contains(prompt, "We ship every Friday") {
			t.Errorf("prompt missing document body content\ngot: %s", prompt[:min(len(prompt), 600)])
		}
		if !strings.Contains(prompt, "## Space Knowledge") {
			t.Errorf("prompt missing '## Space Knowledge' section header\ngot: %s", prompt[:min(len(prompt), 600)])
		}
	})

	t.Run("no_docs_no_space_knowledge_section", func(t *testing.T) {
		// A space with no docs returns an empty slice from ListDocumentContext.
		emptyDocs, err := store.ListDocumentContext(ctx, "nonexistent-space-id")
		if err != nil {
			t.Fatalf("ListDocumentContext: %v", err)
		}

		convo := &Node{Title: "Empty Space Chat", Tags: []string{"user-a", "user-b"}}
		prompt := mind.buildSystemPrompt(convo, "some-agent-id", emptyDocs)

		if strings.Contains(prompt, "Space Knowledge") {
			t.Errorf("prompt should not contain 'Space Knowledge' when no docs present\ngot: %s", prompt[:min(len(prompt), 400)])
		}
		if !strings.Contains(prompt, "SOUL") {
			t.Errorf("prompt missing SOUL fallback section\ngot: %s", prompt[:min(len(prompt), 200)])
		}
	})
}

// TestListDocumentContextBounded verifies that ListDocumentContext enforces a
// maximum of 10 documents (Invariant 13: BOUNDED — every operation has defined scope).
func TestListDocumentContextBounded(t *testing.T) {
	_, store := testDB(t)
	ctx := t.Context()

	slug := fmt.Sprintf("doc-ctx-limit-test-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(ctx, slug, "Doc Ctx Limit", "", "owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	// Create 15 documents — more than the BOUNDED limit of 10.
	for i := 0; i < 15; i++ {
		store.CreateNode(ctx, CreateNodeParams{
			SpaceID: space.ID, Kind: KindDocument,
			Title:      fmt.Sprintf("Doc %d", i),
			Body:       fmt.Sprintf("Content for doc %d.", i),
			Author:     "Tester", AuthorID: "test-owner", AuthorKind: "human",
		})
	}

	docs, err := store.ListDocumentContext(ctx, space.ID)
	if err != nil {
		t.Fatalf("ListDocumentContext: %v", err)
	}

	const maxDocContext = 10 // matches the Limit in ListDocumentContext
	if len(docs) > maxDocContext {
		t.Errorf("ListDocumentContext returned %d docs, want at most %d (Invariant 13: BOUNDED)", len(docs), maxDocContext)
	}
}

// TestReplyToInjectsDocuments verifies that the Chat auto-reply path (replyTo)
// fetches space documents from the store and injects them into the system prompt.
// This tests the full wiring: replyTo → ListDocumentContext → buildSystemPrompt.
// It uses callClaudeOverride to capture the assembled system prompt without a real Claude call.
func TestReplyToInjectsDocuments(t *testing.T) {
	db, store := testDB(t)
	ctx := t.Context()

	mind := NewMind(db, store, "fake-token")

	nonce := time.Now().UnixNano()
	agentName := "ReplyDocTestAgent"
	agentID := fmt.Sprintf("reply-doc-test-agent-id-%d", nonce)

	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
		agentID, fmt.Sprintf("agent:%s-%d", agentName, nonce), fmt.Sprintf("%s-%d@test.lovyou.ai", agentName, nonce), agentName)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		db.ExecContext(cleanupCtx, `DELETE FROM users WHERE id = $1`, agentID)
	})

	slug := fmt.Sprintf("reply-doc-inject-test-%d", nonce)
	space, err := store.CreateSpace(ctx, slug, "Reply Doc Inject Test", "", "owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(context.Background(), space.ID) })

	// Seed documents in the space.
	store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument,
		Title: "Runbook", Body: "Deploy with ./ship.sh on Fridays.",
		Author: "Owner", AuthorID: "owner", AuthorKind: "human",
	})

	convo, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindConversation,
		Title: "Chat", Author: "Human", AuthorID: "human-id", AuthorKind: "human",
		Tags: []string{"human-id", agentID},
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	var capturedPrompt string
	mind.callClaudeOverride = func(_ context.Context, systemPrompt string, _ []claudeMessage) (string, error) {
		capturedPrompt = systemPrompt
		return "ok", nil
	}

	// Call OnMessage — this is the full Chat auto-reply path.
	mind.OnMessage(space.ID, space.Slug, convo, "human-id")

	if capturedPrompt == "" {
		t.Fatal("callClaude was never called — check agent participant lookup")
	}
	if !strings.Contains(capturedPrompt, "## Space Knowledge") {
		t.Errorf("system prompt missing '## Space Knowledge' section\ngot: %s", capturedPrompt[:min(len(capturedPrompt), 600)])
	}
	if !strings.Contains(capturedPrompt, "Runbook") {
		t.Errorf("system prompt missing document title 'Runbook'\ngot: %s", capturedPrompt[:min(len(capturedPrompt), 600)])
	}
	if !strings.Contains(capturedPrompt, "Deploy with ./ship.sh") {
		t.Errorf("system prompt missing document body content\ngot: %s", capturedPrompt[:min(len(capturedPrompt), 600)])
	}
}

// TestReplyToNoDocumentsNoSpaceKnowledge verifies that when a space has no documents,
// the Chat auto-reply system prompt does NOT include a "## Space Knowledge" section.
func TestReplyToNoDocumentsNoSpaceKnowledge(t *testing.T) {
	db, store := testDB(t)
	ctx := t.Context()

	mind := NewMind(db, store, "fake-token")

	nonce := time.Now().UnixNano()
	agentName := "ReplyNoDocTestAgent"
	agentID := fmt.Sprintf("reply-no-doc-test-agent-id-%d", nonce)

	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
		agentID, fmt.Sprintf("agent:%s-%d", agentName, nonce), fmt.Sprintf("%s-%d@test.lovyou.ai", agentName, nonce), agentName)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		db.ExecContext(cleanupCtx, `DELETE FROM users WHERE id = $1`, agentID)
	})

	slug := fmt.Sprintf("reply-no-doc-test-%d", nonce)
	space, err := store.CreateSpace(ctx, slug, "Reply No Doc Test", "", "owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(context.Background(), space.ID) })

	// No documents seeded — space is empty.

	convo, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindConversation,
		Title: "Empty Space Chat", Author: "Human", AuthorID: "human-id", AuthorKind: "human",
		Tags: []string{"human-id", agentID},
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	var capturedPrompt string
	mind.callClaudeOverride = func(_ context.Context, systemPrompt string, _ []claudeMessage) (string, error) {
		capturedPrompt = systemPrompt
		return "ok", nil
	}

	mind.OnMessage(space.ID, space.Slug, convo, "human-id")

	if capturedPrompt == "" {
		t.Fatal("callClaude was never called — check agent participant lookup")
	}
	if strings.Contains(capturedPrompt, "Space Knowledge") {
		t.Errorf("system prompt should not contain 'Space Knowledge' when space has no docs\ngot: %s", capturedPrompt[:min(len(capturedPrompt), 400)])
	}
}

// TestMindOnQuestionAsked_WithAgent verifies that OnQuestionAsked emits an answer
// op (creates a KindComment child) when an agent is available (Invariant VERIFIED).
func TestMindOnQuestionAsked_WithAgent(t *testing.T) {
	db, store := testDB(t)
	ctx := t.Context()
	mind := NewMind(db, store, "fake-token")

	agentName := "QATestAgent"
	agentID := "qa-test-agent-id"

	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
		agentID, "agent:"+agentName, agentName+"@test.lovyou.ai", agentName)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	t.Cleanup(func() { db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID) })

	if old, _ := store.GetSpaceBySlug(ctx, "question-asked-with-agent-test"); old != nil {
		store.DeleteSpace(ctx, old.ID)
	}
	space, err := store.CreateSpace(ctx, "question-asked-with-agent-test", "QA With Agent Test", "", "owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	question, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindQuestion,
		Title:      "How does trust scoring work?",
		Body:       "I want to understand the trust model.",
		Author:     "Tester",
		AuthorID:   "test-human-id",
		AuthorKind: "human",
	})
	if err != nil {
		t.Fatalf("create question: %v", err)
	}

	// Stub callClaude to return a canned answer without a real Claude call.
	mind.callClaudeOverride = func(_ context.Context, _ string, _ []claudeMessage) (string, error) {
		return "Trust scores range from 0.0 to 1.0 based on endorsements.", nil
	}

	mind.OnQuestionAsked(space.ID, space.Slug, question)

	// Verify an answer node was created as a child of the question.
	answers, err := store.ListNodes(ctx, ListNodesParams{SpaceID: space.ID, ParentID: question.ID})
	if err != nil {
		t.Fatalf("list answers: %v", err)
	}
	if len(answers) == 0 {
		t.Fatal("expected at least 1 answer node, got 0")
	}
	answer := answers[0]
	if answer.Kind != KindComment {
		t.Errorf("answer kind = %q, want %q", answer.Kind, KindComment)
	}
	// OnQuestionAsked calls GetFirstAgent which returns the alphabetically-first agent.
	// Assert that the answer was by an agent user (not a specific ID, which depends on DB state).
	if answer.AuthorKind != "agent" {
		t.Errorf("answer author kind = %q, want %q", answer.AuthorKind, "agent")
	}
	if answer.Body == "" {
		t.Error("answer body is empty")
	}

	// Verify a respond op was recorded on the answer node.
	ops, err := store.ListOps(ctx, space.ID, 20)
	if err != nil {
		t.Fatalf("list ops: %v", err)
	}
	var hasRespondOp bool
	for _, o := range ops {
		if o.NodeID == answer.ID && o.Op == "respond" {
			hasRespondOp = true
			break
		}
	}
	if !hasRespondOp {
		t.Errorf("expected a respond op on answer node %s, got none", answer.ID)
	}
}

// TestMindOnQuestionAsked_NoAgent verifies graceful no-op when no agent exists.
func TestMindOnQuestionAsked_NoAgent(t *testing.T) {
	db, store := testDB(t)
	ctx := t.Context()
	mind := NewMind(db, store, "fake-token")

	if old, _ := store.GetSpaceBySlug(ctx, "question-no-agent-test"); old != nil {
		store.DeleteSpace(ctx, old.ID)
	}
	space, err := store.CreateSpace(ctx, "question-no-agent-test", "QA No Agent", "", "owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	question := &Node{
		ID:    "test-q-id",
		Title: "Why does this work?",
		Body:  "Curious about the mechanics.",
	}

	// Should not panic even with no agents and a fake Claude token.
	mind.OnQuestionAsked(space.ID, space.Slug, question)
	// No agent → returns early without creating any nodes.
	answers, _ := store.ListNodes(ctx, ListNodesParams{SpaceID: space.ID, ParentID: question.ID})
	if len(answers) != 0 {
		t.Errorf("got %d answers, want 0 (no agent available)", len(answers))
	}
}
