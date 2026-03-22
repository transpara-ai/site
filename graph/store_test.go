package graph

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// testDB returns a test database connection. Skips if DATABASE_URL is not set.
func testDB(t *testing.T) (*sql.DB, *Store) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set (run: docker compose up -d && DATABASE_URL=postgres://site:site@localhost:5433/site?sslmode=disable go test ./graph/)")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return db, store
}

func TestCreateAndGetSpace(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-space", "Test Space", "A test space", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	if space.Slug != "test-space" {
		t.Errorf("slug = %q, want %q", space.Slug, "test-space")
	}
	if space.Visibility != "public" {
		t.Errorf("visibility = %q, want %q", space.Visibility, "public")
	}

	got, err := store.GetSpaceBySlug(ctx, "test-space")
	if err != nil {
		t.Fatalf("get space: %v", err)
	}
	if got.ID != space.ID {
		t.Errorf("ID = %q, want %q", got.ID, space.ID)
	}
}

func TestCreateAndListNodes(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-nodes", "Test Nodes", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	// Create a task.
	task, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindTask,
		Title:      "Test Task",
		Body:       "Do the thing",
		Author:     "tester",
		AuthorID:   "tester-id",
		AuthorKind: "human",
		Priority:   PriorityHigh,
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.Kind != KindTask {
		t.Errorf("kind = %q, want %q", task.Kind, KindTask)
	}
	if task.Priority != PriorityHigh {
		t.Errorf("priority = %q, want %q", task.Priority, PriorityHigh)
	}

	// Create a post.
	post, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindPost,
		Title:      "Test Post",
		Body:       "Hello world",
		Author:     "tester",
		AuthorID:   "tester-id",
		AuthorKind: "human",
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}

	// List all nodes.
	nodes, err := store.ListNodes(ctx, ListNodesParams{SpaceID: space.ID})
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("got %d nodes, want 2", len(nodes))
	}

	// List only tasks.
	tasks, err := store.ListNodes(ctx, ListNodesParams{SpaceID: space.ID, Kind: KindTask})
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("got %d tasks, want 1", len(tasks))
	}

	// Get node by ID.
	got, err := store.GetNode(ctx, task.ID)
	if err != nil {
		t.Fatalf("get node: %v", err)
	}
	if got.Title != "Test Task" {
		t.Errorf("title = %q, want %q", got.Title, "Test Task")
	}

	// Test child count.
	_, err = store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		ParentID:   post.ID,
		Kind:       KindComment,
		Body:       "A comment",
		Author:     "commenter",
		AuthorID:   "commenter-id",
		AuthorKind: "human",
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}

	got, err = store.GetNode(ctx, post.ID)
	if err != nil {
		t.Fatalf("get post: %v", err)
	}
	if got.ChildCount != 1 {
		t.Errorf("child_count = %d, want 1", got.ChildCount)
	}
}

func TestConversations(t *testing.T) {
	db, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-convos", "Test Convos", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	// Create a conversation. Tags store user IDs.
	convo, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindConversation,
		Title:      "Test Chat",
		Body:       "Let's discuss",
		Author:     "alice",
		AuthorID:   "alice-id",
		AuthorKind: "human",
		Tags:       []string{"alice-id", "bob-id"},
	})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}

	// List conversations for participant (by userID).
	convos, err := store.ListConversations(ctx, space.ID, "alice-id")
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}
	if len(convos) != 1 {
		t.Fatalf("got %d conversations, want 1", len(convos))
	}
	if convos[0].Title != "Test Chat" {
		t.Errorf("title = %q, want %q", convos[0].Title, "Test Chat")
	}

	// Non-participant shouldn't see it.
	convos, err = store.ListConversations(ctx, space.ID, "charlie-id")
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}
	if len(convos) != 0 {
		t.Errorf("got %d conversations for non-participant, want 0", len(convos))
	}

	// Add messages and check last message preview.
	_, err = store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		ParentID: convo.ID,
		Kind:     KindComment,
		Body:     "Hello!",
		Author:   "alice",
		AuthorID: "alice-id",
	})
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	convos, err = store.ListConversations(ctx, space.ID, "alice-id")
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}
	if convos[0].LastBody != "Hello!" {
		t.Errorf("last_body = %q, want %q", convos[0].LastBody, "Hello!")
	}
	if convos[0].ChildCount != 1 {
		t.Errorf("child_count = %d, want 1", convos[0].ChildCount)
	}

	// HasAgentParticipant should return false (no agents — tags are user IDs).
	hasAgent, err := store.HasAgentParticipant(ctx, convo.Tags)
	if err != nil {
		t.Fatalf("has agent: %v", err)
	}
	if hasAgent {
		t.Errorf("should not have agent participant")
	}

	// Create an agent user and add to conversation.
	db.ExecContext(ctx, `INSERT INTO users (id, google_id, email, name, kind)
		VALUES ('agent-test-1', 'agent:TestBot', 'testbot@agent.lovyou.ai', 'TestBot', 'agent')
		ON CONFLICT (google_id) DO NOTHING`)
	t.Cleanup(func() { db.ExecContext(ctx, `DELETE FROM users WHERE id = 'agent-test-1'`) })

	// HasAgentParticipant now matches on user ID.
	hasAgent, err = store.HasAgentParticipant(ctx, []string{"alice-id", "agent-test-1"})
	if err != nil {
		t.Fatalf("has agent: %v", err)
	}
	if !hasAgent {
		t.Errorf("should have agent participant")
	}
}

func TestOps(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-ops", "Test Ops", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	node, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		Title:    "Task for ops",
		Author:   "tester",
		AuthorID: "tester-id",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	// Record an op.
	op, err := store.RecordOp(ctx, space.ID, node.ID, "tester", "tester-id", "intend", nil)
	if err != nil {
		t.Fatalf("record op: %v", err)
	}
	if op.Op != "intend" {
		t.Errorf("op = %q, want %q", op.Op, "intend")
	}

	// List ops.
	ops, err := store.ListOps(ctx, space.ID, 10)
	if err != nil {
		t.Fatalf("list ops: %v", err)
	}
	if len(ops) != 1 {
		t.Errorf("got %d ops, want 1", len(ops))
	}
}

func TestUpdateAndDeleteNode(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-mutations", "Test Mutations", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	node, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		Title:    "Mutable Task",
		Author:   "tester",
		AuthorID: "tester-id",
		State:    StateOpen,
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	// Update state.
	if err := store.UpdateNodeState(ctx, node.ID, StateDone); err != nil {
		t.Fatalf("update state: %v", err)
	}
	got, _ := store.GetNode(ctx, node.ID)
	if got.State != StateDone {
		t.Errorf("state = %q, want %q", got.State, StateDone)
	}

	// Update fields.
	newTitle := "Updated Title"
	if err := store.UpdateNode(ctx, node.ID, &newTitle, nil, nil, nil); err != nil {
		t.Fatalf("update node: %v", err)
	}
	got, _ = store.GetNode(ctx, node.ID)
	if got.Title != "Updated Title" {
		t.Errorf("title = %q, want %q", got.Title, "Updated Title")
	}

	// Delete.
	if err := store.DeleteNode(ctx, node.ID); err != nil {
		t.Fatalf("delete node: %v", err)
	}
	_, err = store.GetNode(ctx, node.ID)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestPublicSpaces(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	// Create a public and private space.
	pub, err := store.CreateSpace(ctx, "test-public", "Public Space", "visible", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create public space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, pub.ID) })

	priv, err := store.CreateSpace(ctx, "test-private", "Private Space", "hidden", "owner-1", "project", "private")
	if err != nil {
		t.Fatalf("create private space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, priv.ID) })

	// Add a node to the public space.
	_, err = store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  pub.ID,
		Kind:     KindPost,
		Title:    "Public Post",
		Author:   "tester",
		AuthorID: "tester-id",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	// ListPublicSpaces should include public but not private.
	spaces, err := store.ListPublicSpaces(ctx)
	if err != nil {
		t.Fatalf("list public spaces: %v", err)
	}

	foundPublic := false
	foundPrivate := false
	for _, sp := range spaces {
		if sp.Slug == "test-public" {
			foundPublic = true
			if sp.NodeCount != 1 {
				t.Errorf("node_count = %d, want 1", sp.NodeCount)
			}
		}
		if sp.Slug == "test-private" {
			foundPrivate = true
		}
	}
	if !foundPublic {
		t.Errorf("should find public space")
	}
	if foundPrivate {
		t.Errorf("should not find private space")
	}
}
