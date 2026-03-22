package graph

import (
	"context"
	"database/sql"
	"os"
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
