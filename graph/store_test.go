package graph

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

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
	if err := store.UpdateNode(ctx, node.ID, &newTitle, nil, nil, nil, nil); err != nil {
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

func TestUpdateNodeStateChildGate(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-child-gate", "Child Gate", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	parent, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		Title:    "Parent Task",
		Author:   "tester",
		AuthorID: "tester-id",
		State:    StateActive,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	child, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		ParentID: parent.ID,
		Kind:     KindTask,
		Title:    "Child Task",
		Author:   "tester",
		AuthorID: "tester-id",
		State:    StateActive,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	// Completing parent with incomplete child must auto-cascade and succeed.
	if err := store.UpdateNodeState(ctx, parent.ID, StateDone); err != nil {
		t.Fatalf("complete parent: %v", err)
	}

	// Parent must be done.
	gotParent, _ := store.GetNode(ctx, parent.ID)
	if gotParent.State != StateDone {
		t.Errorf("parent state = %q, want %q", gotParent.State, StateDone)
	}

	// Child must have been auto-closed by the cascade.
	gotChild, _ := store.GetNode(ctx, child.ID)
	if gotChild.State != StateDone {
		t.Errorf("child state = %q, want %q (expected cascade close)", gotChild.State, StateDone)
	}
}

// TestUpdateNodeStateChildGateLeafNode verifies that leaf nodes (no children)
// can be completed without the gate blocking them.
func TestUpdateNodeStateChildGateLeafNode(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-child-gate-leaf", "Child Gate Leaf", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	leaf, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		Title:    "Leaf Task",
		Author:   "tester",
		AuthorID: "tester-id",
		State:    StateActive,
	})
	if err != nil {
		t.Fatalf("create leaf: %v", err)
	}

	// Leaf node (no children) must be completable directly.
	if err := store.UpdateNodeState(ctx, leaf.ID, StateDone); err != nil {
		t.Fatalf("complete leaf: %v", err)
	}
	got, _ := store.GetNode(ctx, leaf.ID)
	if got.State != StateDone {
		t.Errorf("leaf state = %q, want %q", got.State, StateDone)
	}
}

// TestUpdateNodeStateChildGateMultipleChildren verifies partial child completion
// still blocks parent completion, and all-done children allow it.
func TestUpdateNodeStateChildGateMultipleChildren(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-child-gate-multi", "Child Gate Multi", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	parent, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		Title:    "Parent",
		Author:   "tester",
		AuthorID: "tester-id",
		State:    StateActive,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	child1, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		ParentID: parent.ID,
		Kind:     KindTask,
		Title:    "Child 1",
		Author:   "tester",
		AuthorID: "tester-id",
		State:    StateActive,
	})
	if err != nil {
		t.Fatalf("create child1: %v", err)
	}

	child2, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		ParentID: parent.ID,
		Kind:     KindTask,
		Title:    "Child 2",
		Author:   "tester",
		AuthorID: "tester-id",
		State:    StateActive,
	})
	if err != nil {
		t.Fatalf("create child2: %v", err)
	}

	// Complete only child1 — completing parent must auto-cascade and close child2.
	if err := store.UpdateNodeState(ctx, child1.ID, StateDone); err != nil {
		t.Fatalf("complete child1: %v", err)
	}
	if err := store.UpdateNodeState(ctx, parent.ID, StateDone); err != nil {
		t.Fatalf("complete parent with one open child: %v", err)
	}

	// child2 must have been auto-closed by the cascade.
	gotChild2, _ := store.GetNode(ctx, child2.ID)
	if gotChild2.State != StateDone {
		t.Errorf("child2 state = %q, want %q (expected cascade close)", gotChild2.State, StateDone)
	}

	// Parent must be done.
	gotParent, _ := store.GetNode(ctx, parent.ID)
	if gotParent.State != StateDone {
		t.Errorf("parent state = %q, want %q", gotParent.State, StateDone)
	}
}

// TestCascadeCloseChildrenDeep verifies that completing a grandparent also closes
// grandchildren — the recursive cascade traverses all descendant levels.
func TestCascadeCloseChildrenDeep(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-cascade-deep", "Cascade Deep", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	grandparent, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Grandparent",
		Author: "tester", AuthorID: "tester-id", State: StateActive,
	})
	if err != nil {
		t.Fatalf("create grandparent: %v", err)
	}

	parent, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, ParentID: grandparent.ID, Kind: KindTask, Title: "Parent",
		Author: "tester", AuthorID: "tester-id", State: StateActive,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	grandchild, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, ParentID: parent.ID, Kind: KindTask, Title: "Grandchild",
		Author: "tester", AuthorID: "tester-id", State: StateActive,
	})
	if err != nil {
		t.Fatalf("create grandchild: %v", err)
	}

	if err := store.UpdateNodeState(ctx, grandparent.ID, StateDone); err != nil {
		t.Fatalf("complete grandparent: %v", err)
	}

	for label, id := range map[string]string{
		"grandparent": grandparent.ID,
		"parent":      parent.ID,
		"grandchild":  grandchild.ID,
	} {
		got, err := store.GetNode(ctx, id)
		if err != nil {
			t.Fatalf("get %s: %v", label, err)
		}
		if got.State != StateDone {
			t.Errorf("%s state = %q, want %q (expected cascade close)", label, got.State, StateDone)
		}
	}
}

// TestUpdateNodeStateNonDoneSkipsGate verifies that the child-completion gate
// only fires when transitioning to StateDone — other state changes are unblocked.
func TestUpdateNodeStateNonDoneSkipsGate(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-child-gate-nondone", "Child Gate NonDone", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	parent, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		Title:    "Parent",
		Author:   "tester",
		AuthorID: "tester-id",
		State:    StateActive,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	// Add an incomplete child.
	_, err = store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		ParentID: parent.ID,
		Kind:     KindTask,
		Title:    "Incomplete Child",
		Author:   "tester",
		AuthorID: "tester-id",
		State:    StateActive,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	// Setting to StateReview (non-done) must not trigger the gate.
	if err := store.UpdateNodeState(ctx, parent.ID, StateReview); err != nil {
		t.Fatalf("set parent to review with incomplete child: %v", err)
	}
}

// TestCascadeDepthBoundary verifies that cascadeCloseChildren returns an error
// when depth exceeds maxCascadeDepth (50). Uses a parent+child pair but calls
// cascadeCloseChildren directly at depth=maxCascadeDepth so the recursive call
// hits depth=51, triggering the bound — without needing a 52-level tree.
func TestCascadeDepthBoundary(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-cascade-bound", "Cascade Bound", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	parent, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Parent",
		Author: "tester", AuthorID: "tester-id", State: StateActive,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}

	_, err = store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, ParentID: parent.ID, Kind: KindTask, Title: "Child",
		Author: "tester", AuthorID: "tester-id", State: StateActive,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	// Call at maxCascadeDepth: the child recurse call hits depth=51, which exceeds
	// the cap. Cascade must fail with a depth-exceeded error.
	err = store.cascadeCloseChildren(ctx, parent.ID, maxCascadeDepth)
	if err == nil {
		t.Fatal("expected depth-exceeded error, got nil")
	}
	if !strings.Contains(err.Error(), "cascade depth exceeded") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "cascade depth exceeded")
	}
}

func TestDependencies(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-deps", "Test Deps", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	taskA, _ := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Task A", Author: "tester", AuthorID: "tester-id",
	})
	taskB, _ := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Task B", Author: "tester", AuthorID: "tester-id",
	})

	// B depends on A.
	if err := store.AddDependency(ctx, taskB.ID, taskA.ID); err != nil {
		t.Fatalf("add dependency: %v", err)
	}

	// B should have 1 blocker (A is not done).
	b, _ := store.GetNode(ctx, taskB.ID)
	if b.BlockerCount != 1 {
		t.Errorf("blocker_count = %d, want 1", b.BlockerCount)
	}

	// List blockers for B.
	blockers, err := store.ListBlockers(ctx, taskB.ID)
	if err != nil {
		t.Fatalf("list blockers: %v", err)
	}
	if len(blockers) != 1 {
		t.Fatalf("got %d blockers, want 1", len(blockers))
	}
	if blockers[0].Title != "Task A" {
		t.Errorf("blocker title = %q, want %q", blockers[0].Title, "Task A")
	}

	// Complete A — B should have 0 blockers.
	store.UpdateNodeState(ctx, taskA.ID, StateDone)
	b, _ = store.GetNode(ctx, taskB.ID)
	if b.BlockerCount != 0 {
		t.Errorf("after completing A, blocker_count = %d, want 0", b.BlockerCount)
	}

	// Duplicate dependency should be ignored.
	if err := store.AddDependency(ctx, taskB.ID, taskA.ID); err != nil {
		t.Fatalf("duplicate dependency should not error: %v", err)
	}
}

func TestMembership(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, _ := store.CreateSpace(ctx, "test-membership", "Membership Test", "", "owner-1", "project", "public")
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	// Not a member initially.
	if store.IsMember(ctx, space.ID, "user-1") {
		t.Error("should not be a member initially")
	}
	if store.MemberCount(ctx, space.ID) != 0 {
		t.Errorf("member count = %d, want 0", store.MemberCount(ctx, space.ID))
	}

	// Join.
	if err := store.JoinSpace(ctx, space.ID, "user-1", "Alice"); err != nil {
		t.Fatalf("join: %v", err)
	}
	if !store.IsMember(ctx, space.ID, "user-1") {
		t.Error("should be a member after joining")
	}
	if store.MemberCount(ctx, space.ID) != 1 {
		t.Errorf("member count = %d, want 1", store.MemberCount(ctx, space.ID))
	}

	// Duplicate join is a no-op.
	store.JoinSpace(ctx, space.ID, "user-1", "Alice")
	if store.MemberCount(ctx, space.ID) != 1 {
		t.Errorf("member count = %d, want 1 after duplicate join", store.MemberCount(ctx, space.ID))
	}

	// Leave.
	store.LeaveSpace(ctx, space.ID, "user-1")
	if store.IsMember(ctx, space.ID, "user-1") {
		t.Error("should not be a member after leaving")
	}
}

func TestNodeMembership(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, _ := store.CreateSpace(ctx, "test-node-membership", "Node Membership Test", "", "owner-1", "project", "public")
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	team, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindTeam,
		Title:    "Engineering",
		AuthorID: "owner-1",
		Author:   "Owner",
	})
	if err != nil {
		t.Fatalf("create team: %v", err)
	}

	// Not a member initially.
	if store.IsNodeMember(ctx, team.ID, "user-1") {
		t.Error("should not be a node member initially")
	}
	if store.NodeMemberCount(ctx, team.ID) != 0 {
		t.Errorf("node member count = %d, want 0", store.NodeMemberCount(ctx, team.ID))
	}

	// Join.
	if err := store.JoinNodeMember(ctx, team.ID, "user-1"); err != nil {
		t.Fatalf("join node: %v", err)
	}
	if !store.IsNodeMember(ctx, team.ID, "user-1") {
		t.Error("should be a node member after joining")
	}
	if store.NodeMemberCount(ctx, team.ID) != 1 {
		t.Errorf("node member count = %d, want 1", store.NodeMemberCount(ctx, team.ID))
	}

	// Duplicate join is a no-op.
	store.JoinNodeMember(ctx, team.ID, "user-1")
	if store.NodeMemberCount(ctx, team.ID) != 1 {
		t.Errorf("node member count = %d, want 1 after duplicate join", store.NodeMemberCount(ctx, team.ID))
	}

	// ListTeamMembers.
	members, err := store.ListTeamMembers(ctx, space.ID, team.ID)
	if err != nil {
		t.Fatalf("list team members: %v", err)
	}
	if len(members) != 1 {
		t.Errorf("team members = %d, want 1", len(members))
	}
	if members[0].UserID != "user-1" {
		t.Errorf("member user_id = %q, want %q", members[0].UserID, "user-1")
	}

	// Leave.
	store.LeaveNodeMember(ctx, team.ID, "user-1")
	if store.IsNodeMember(ctx, team.ID, "user-1") {
		t.Error("should not be a node member after leaving")
	}
	if store.NodeMemberCount(ctx, team.ID) != 0 {
		t.Errorf("node member count = %d, want 0 after leave", store.NodeMemberCount(ctx, team.ID))
	}
}

func TestAvailableTasks(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, _ := store.CreateSpace(ctx, "test-market", "Market Test", "", "owner-1", "project", "public")
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	// Create an open, unassigned task.
	store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Available Task",
		Author: "tester", AuthorID: "tester-id", State: StateOpen,
	})
	// Create an assigned task (should not appear).
	store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Taken Task",
		Author: "tester", AuthorID: "tester-id", State: StateOpen, Assignee: "someone",
	})

	tasks, err := store.ListAvailableTasks(ctx, "", "", 50)
	if err != nil {
		t.Fatalf("list available: %v", err)
	}

	found := false
	for _, task := range tasks {
		if task.Title == "Available Task" {
			found = true
		}
		if task.Title == "Taken Task" {
			t.Error("assigned task should not appear in available list")
		}
	}
	if !found {
		t.Error("should find the available task")
	}

	// Search.
	tasks, _ = store.ListAvailableTasks(ctx, "Available", "", 50)
	if len(tasks) == 0 {
		t.Error("search should find the task")
	}
	tasks, _ = store.ListAvailableTasks(ctx, "nonexistent", "", 50)
	for _, task := range tasks {
		if task.Title == "Available Task" {
			t.Error("search for nonexistent should not find the task")
		}
	}
}

func TestEndorsements(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	// No endorsements initially.
	if store.CountEndorsements(ctx, "user-b") != 0 {
		t.Error("should have 0 endorsements initially")
	}
	if store.HasEndorsed(ctx, "user-a", "user-b") {
		t.Error("should not have endorsed initially")
	}

	// Endorse.
	if err := store.Endorse(ctx, "user-a", "user-b"); err != nil {
		t.Fatalf("endorse: %v", err)
	}
	if !store.HasEndorsed(ctx, "user-a", "user-b") {
		t.Error("should have endorsed after Endorse")
	}
	if store.CountEndorsements(ctx, "user-b") != 1 {
		t.Errorf("endorsement count = %d, want 1", store.CountEndorsements(ctx, "user-b"))
	}

	// Duplicate endorse is a no-op.
	store.Endorse(ctx, "user-a", "user-b")
	if store.CountEndorsements(ctx, "user-b") != 1 {
		t.Errorf("endorsement count after duplicate = %d, want 1", store.CountEndorsements(ctx, "user-b"))
	}

	// Unendorse.
	store.Unendorse(ctx, "user-a", "user-b")
	if store.HasEndorsed(ctx, "user-a", "user-b") {
		t.Error("should not have endorsed after Unendorse")
	}
	if store.CountEndorsements(ctx, "user-b") != 0 {
		t.Errorf("endorsement count after unendorse = %d, want 0", store.CountEndorsements(ctx, "user-b"))
	}
}

func TestReportsAndResolve(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, _ := store.CreateSpace(ctx, "test-reports", "Reports Test", "", "owner-1", "project", "public")
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	node, _ := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindPost, Title: "Flagged Post",
		Author: "author", AuthorID: "author-id",
	})

	// Report the node.
	store.RecordOp(ctx, space.ID, node.ID, "reporter", "reporter-id", "report",
		[]byte(`{"reason":"spam"}`))

	// Should appear in unresolved reports.
	reports, err := store.ListReports(ctx, space.ID)
	if err != nil {
		t.Fatalf("list reports: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("got %d reports, want 1", len(reports))
	}
	if reports[0].NodeTitle != "Flagged Post" {
		t.Errorf("node title = %q, want %q", reports[0].NodeTitle, "Flagged Post")
	}
	if reports[0].Reason != "spam" {
		t.Errorf("reason = %q, want %q", reports[0].Reason, "spam")
	}

	// Resolve the report (dismiss).
	store.RecordOp(ctx, space.ID, node.ID, "owner", "owner-1", "resolve",
		[]byte(`{"action":"dismiss"}`))

	// Should no longer appear.
	reports, _ = store.ListReports(ctx, space.ID)
	if len(reports) != 0 {
		t.Errorf("got %d reports after resolve, want 0", len(reports))
	}
}

func TestDashboardQueries(t *testing.T) {
	db, store := testDB(t)
	ctx := context.Background()

	space, _ := store.CreateSpace(ctx, "test-dash", "Dashboard Test", "", "owner-1", "project", "public")
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	// Create a user for assignee resolution.
	db.ExecContext(ctx, `INSERT INTO users (id, google_id, email, name, kind)
		VALUES ('dash-user', 'google:dash', 'dash@test.com', 'DashUser', 'human')
		ON CONFLICT (google_id) DO NOTHING`)
	t.Cleanup(func() { db.ExecContext(ctx, `DELETE FROM users WHERE id = 'dash-user'`) })

	// Create a task assigned to the user.
	store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "My Task",
		Author: "other", AuthorID: "other-id", Assignee: "DashUser", AssigneeID: "dash-user",
	})
	// Create a task by the user (author).
	store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Authored Task",
		Author: "DashUser", AuthorID: "dash-user",
	})
	// Create a done task (should not appear).
	node, _ := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Done Task",
		Author: "DashUser", AuthorID: "dash-user",
	})
	store.UpdateNodeState(ctx, node.ID, StateDone)

	tasks, err := store.ListUserTasks(ctx, "dash-user", "", 50)
	if err != nil {
		t.Fatalf("list user tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2 (assigned + authored)", len(tasks))
	}
	for _, task := range tasks {
		if task.Title == "Done Task" {
			t.Error("done task should not appear in dashboard")
		}
		if task.SpaceName != "Dashboard Test" {
			t.Errorf("space name = %q, want %q", task.SpaceName, "Dashboard Test")
		}
	}

	// User conversations.
	store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindConversation, Title: "My Chat",
		Author: "DashUser", AuthorID: "dash-user", Tags: []string{"dash-user", "other-id"},
	})

	convos, err := store.ListUserConversations(ctx, "dash-user", 50)
	if err != nil {
		t.Fatalf("list user conversations: %v", err)
	}
	if len(convos) != 1 {
		t.Errorf("got %d conversations, want 1", len(convos))
	}
}

func TestSearch(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, _ := store.CreateSpace(ctx, "test-search", "Searchable Space", "find me", "owner-1", "project", "public")
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindPost, Title: "Findable Post",
		Body: "unique content xyz123", Author: "tester", AuthorID: "tester-id",
	})

	// Search spaces by name.
	results := store.Search(ctx, "Searchable", 10)
	if len(results.Spaces) == 0 {
		t.Error("should find space by name")
	}

	// Search nodes by body.
	results = store.Search(ctx, "xyz123", 10)
	if len(results.Nodes) == 0 {
		t.Error("should find node by body content")
	}

	// No results for nonsense.
	results = store.Search(ctx, "zzzznotfound9999", 10)
	if len(results.Spaces) != 0 || len(results.Nodes) != 0 {
		t.Error("should find nothing for nonsense query")
	}

	// Empty query returns nothing.
	results = store.Search(ctx, "", 10)
	if len(results.Spaces) != 0 || len(results.Nodes) != 0 {
		t.Error("empty query should return nothing")
	}
}

func TestKnowledgeClaims(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, _ := store.CreateSpace(ctx, "test-knowledge", "Knowledge Test", "", "owner-1", "project", "public")
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	// Create a claim.
	claim, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindClaim, Title: "Earth is round",
		Body: "Verified by observation", Author: "scientist", AuthorID: "sci-id",
		State: ClaimClaimed,
	})
	if err != nil {
		t.Fatalf("create claim: %v", err)
	}
	if claim.Kind != KindClaim {
		t.Errorf("kind = %q, want %q", claim.Kind, KindClaim)
	}

	// List claims.
	claims, err := store.ListKnowledgeClaims(ctx, "", "", 50)
	if err != nil {
		t.Fatalf("list claims: %v", err)
	}
	found := false
	for _, c := range claims {
		if c.ID == claim.ID {
			found = true
			if c.Title != "Earth is round" {
				t.Errorf("title = %q, want %q", c.Title, "Earth is round")
			}
		}
	}
	if !found {
		t.Error("should find the claim in ListKnowledgeClaims")
	}

	// Challenge it.
	store.RecordOp(ctx, space.ID, claim.ID, "skeptic", "skeptic-id", "challenge",
		[]byte(`{"reason":"needs evidence"}`))
	store.UpdateNodeState(ctx, claim.ID, ClaimChallenged)

	// Filter by challenged state.
	claims, _ = store.ListKnowledgeClaims(ctx, ClaimChallenged, "", 50)
	found = false
	for _, c := range claims {
		if c.ID == claim.ID {
			found = true
			if c.Challenges != 1 {
				t.Errorf("challenges = %d, want 1", c.Challenges)
			}
		}
	}
	if !found {
		t.Error("challenged claim should appear in filtered list")
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

func TestFollows(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	// No follows initially.
	if store.IsFollowing(ctx, "user-a", "user-b") {
		t.Error("should not be following initially")
	}
	if store.CountFollowers(ctx, "user-b") != 0 {
		t.Error("should have 0 followers initially")
	}
	if store.CountFollowing(ctx, "user-a") != 0 {
		t.Error("should be following 0 initially")
	}

	// Follow.
	if err := store.Follow(ctx, "user-a", "user-b"); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if !store.IsFollowing(ctx, "user-a", "user-b") {
		t.Error("should be following after Follow")
	}
	if store.CountFollowers(ctx, "user-b") != 1 {
		t.Errorf("follower count = %d, want 1", store.CountFollowers(ctx, "user-b"))
	}
	if store.CountFollowing(ctx, "user-a") != 1 {
		t.Errorf("following count = %d, want 1", store.CountFollowing(ctx, "user-a"))
	}

	// Duplicate follow is no-op.
	store.Follow(ctx, "user-a", "user-b")
	if store.CountFollowers(ctx, "user-b") != 1 {
		t.Errorf("follower count after duplicate = %d, want 1", store.CountFollowers(ctx, "user-b"))
	}

	// ListFollowedIDs.
	ids := store.ListFollowedIDs(ctx, "user-a")
	if len(ids) != 1 || ids[0] != "user-b" {
		t.Errorf("ListFollowedIDs = %v, want [user-b]", ids)
	}

	// Unfollow.
	store.Unfollow(ctx, "user-a", "user-b")
	if store.IsFollowing(ctx, "user-a", "user-b") {
		t.Error("should not be following after Unfollow")
	}
	if store.CountFollowers(ctx, "user-b") != 0 {
		t.Errorf("follower count after unfollow = %d, want 0", store.CountFollowers(ctx, "user-b"))
	}
}

func TestReposts(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	// Create a space and post for reposting.
	sp, _ := store.CreateSpace(ctx, "repost-test", "Repost Test", "", "test-owner", "project", "public")
	post, _ := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: sp.ID, Kind: KindPost, Title: "Test Post", Body: "body", Author: "tester", AuthorID: "test-owner",
	})

	// Not reposted initially.
	if store.HasReposted(ctx, "user-a", post.ID) {
		t.Error("should not have reposted initially")
	}

	// Repost.
	if err := store.Repost(ctx, "user-a", post.ID); err != nil {
		t.Fatalf("repost: %v", err)
	}
	if !store.HasReposted(ctx, "user-a", post.ID) {
		t.Error("should have reposted after Repost")
	}

	// Bulk counts.
	counts := store.GetBulkRepostCounts(ctx, []string{post.ID})
	if counts[post.ID] != 1 {
		t.Errorf("repost count = %d, want 1", counts[post.ID])
	}

	// Bulk user reposts.
	userReposts := store.GetBulkUserReposts(ctx, "user-a", []string{post.ID})
	if !userReposts[post.ID] {
		t.Error("user should have reposted this post")
	}

	// Duplicate repost is no-op.
	store.Repost(ctx, "user-a", post.ID)
	counts = store.GetBulkRepostCounts(ctx, []string{post.ID})
	if counts[post.ID] != 1 {
		t.Errorf("repost count after duplicate = %d, want 1", counts[post.ID])
	}

	// Unrepost.
	store.Unrepost(ctx, "user-a", post.ID)
	if store.HasReposted(ctx, "user-a", post.ID) {
		t.Error("should not have reposted after Unrepost")
	}
}

func TestQuotePost(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	sp, _ := store.CreateSpace(ctx, "quote-test", "Quote Test", "", "test-owner", "project", "public")

	// Create original post.
	original, _ := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: sp.ID, Kind: KindPost, Title: "Original", Body: "original body", Author: "alice", AuthorID: "alice-id",
	})

	// Create quote post.
	quote, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: sp.ID, Kind: KindPost, Title: "My Quote", Body: "quoting this", Author: "bob", AuthorID: "bob-id",
		QuoteOfID: original.ID,
	})
	if err != nil {
		t.Fatalf("create quote: %v", err)
	}
	if quote.QuoteOfID != original.ID {
		t.Errorf("QuoteOfID = %q, want %q", quote.QuoteOfID, original.ID)
	}

	// GetNode resolves quote fields.
	got, err := store.GetNode(ctx, quote.ID)
	if err != nil {
		t.Fatalf("get quote: %v", err)
	}
	if got.QuoteOfID != original.ID {
		t.Errorf("GetNode QuoteOfID = %q, want %q", got.QuoteOfID, original.ID)
	}
	if got.QuoteOfAuthor != "alice" {
		t.Errorf("QuoteOfAuthor = %q, want %q", got.QuoteOfAuthor, "alice")
	}
	if got.QuoteOfTitle != "Original" {
		t.Errorf("QuoteOfTitle = %q, want %q", got.QuoteOfTitle, "Original")
	}
	if got.QuoteOfBody == "" {
		t.Error("QuoteOfBody should not be empty")
	}
}

func TestMessageSearch(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	sp, _ := store.CreateSpace(ctx, "msgsearch-test", "Msg Search Test", "", "test-owner", "project", "public")

	// Create a conversation with messages.
	convo, _ := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: sp.ID, Kind: KindConversation, Title: "Test Convo", Author: "alice", AuthorID: "alice-id", Tags: []string{"alice-id"},
	})
	store.CreateNode(ctx, CreateNodeParams{
		SpaceID: sp.ID, Kind: KindComment, ParentID: convo.ID, Body: "hello world from alice", Author: "alice", AuthorID: "alice-id",
	})
	store.CreateNode(ctx, CreateNodeParams{
		SpaceID: sp.ID, Kind: KindComment, ParentID: convo.ID, Body: "goodbye world from bob", Author: "bob", AuthorID: "bob-id",
	})

	// Search by body.
	results, err := store.SearchMessages(ctx, sp.ID, "hello", "", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("search 'hello' returned %d results, want 1", len(results))
	}
	if len(results) > 0 && results[0].ConvoTitle != "Test Convo" {
		t.Errorf("ConvoTitle = %q, want %q", results[0].ConvoTitle, "Test Convo")
	}

	// Search by from: author.
	results, err = store.SearchMessages(ctx, sp.ID, "", "bob", 10)
	if err != nil {
		t.Fatalf("search from: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("search from:bob returned %d results, want 1", len(results))
	}

	// Search with no matches.
	results, _ = store.SearchMessages(ctx, sp.ID, "nonexistent", "", 10)
	if len(results) != 0 {
		t.Errorf("search 'nonexistent' returned %d results, want 0", len(results))
	}
}

func TestBulkEndorsements(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	sp, _ := store.CreateSpace(ctx, "bulkendorse-test", "Bulk Endorse Test", "", "test-owner", "project", "public")
	post1, _ := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: sp.ID, Kind: KindPost, Title: "Post 1", Body: "body1", Author: "alice", AuthorID: "alice-id",
	})
	post2, _ := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: sp.ID, Kind: KindPost, Title: "Post 2", Body: "body2", Author: "alice", AuthorID: "alice-id",
	})

	// Endorse post1.
	store.Endorse(ctx, "user-a", post1.ID)
	store.Endorse(ctx, "user-b", post1.ID)

	// Bulk counts.
	counts := store.GetBulkEndorsementCounts(ctx, []string{post1.ID, post2.ID})
	if counts[post1.ID] != 2 {
		t.Errorf("post1 endorsements = %d, want 2", counts[post1.ID])
	}
	if counts[post2.ID] != 0 {
		t.Errorf("post2 endorsements = %d, want 0", counts[post2.ID])
	}

	// Bulk user endorsements.
	userEndorsed := store.GetBulkUserEndorsements(ctx, "user-a", []string{post1.ID, post2.ID})
	if !userEndorsed[post1.ID] {
		t.Error("user-a should have endorsed post1")
	}
	if userEndorsed[post2.ID] {
		t.Error("user-a should not have endorsed post2")
	}
}

func TestGetAgentPersonasForConversations(t *testing.T) {
	db, store := testDB(t)
	ctx := context.Background()

	// Seed a persona.
	persona := AgentPersona{
		Name:        "test-agent-persona",
		Display:     "Test Agent",
		Description: "for testing",
		Category:    "product",
		Prompt:      "# Test Agent\nA test persona.",
		Model:       "sonnet",
		Active:      true,
	}
	if err := store.UpsertAgentPersona(ctx, persona); err != nil {
		t.Fatalf("upsert persona: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM agent_personas WHERE name = $1`, persona.Name)
	})

	// Insert an agent user referencing that persona.
	agentID := newID()
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, google_id, email, name, kind, persona_name)
		VALUES ($1, $2, $3, $4, 'agent', $5)
		ON CONFLICT (google_id) DO NOTHING`,
		agentID, "agent:test-agent-persona", "test-agent-persona@agent.lovyou.ai", "test-agent-persona", persona.Name,
	)
	if err != nil {
		t.Fatalf("insert agent user: %v", err)
	}
	t.Cleanup(func() { db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID) })

	humanID := newID()
	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, google_id, email, name, kind)
		VALUES ($1, $2, $3, $4, 'human')
		ON CONFLICT (google_id) DO NOTHING`,
		humanID, "human:test-"+humanID, "human@example.com", "human-user",
	)
	if err != nil {
		t.Fatalf("insert human user: %v", err)
	}
	t.Cleanup(func() { db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, humanID) })

	// Build two fake conversations: one with the agent, one without.
	convoWithAgent := ConversationSummary{Node: Node{ID: newID(), Tags: []string{humanID, agentID}}}
	convoHuman := ConversationSummary{Node: Node{ID: newID(), Tags: []string{humanID}}}
	convos := []ConversationSummary{convoWithAgent, convoHuman}

	result := store.GetAgentPersonasForConversations(ctx, convos)

	if p := result[convoWithAgent.ID]; p == nil {
		t.Error("expected persona for convo with agent, got nil")
	} else if p.Name != persona.Name {
		t.Errorf("persona name = %q, want %q", p.Name, persona.Name)
	} else if p.Display != persona.Display {
		t.Errorf("persona display = %q, want %q", p.Display, persona.Display)
	}

	if p := result[convoHuman.ID]; p != nil {
		t.Errorf("expected nil persona for human-only convo, got %+v", p)
	}

	// Empty input must not panic.
	empty := store.GetAgentPersonasForConversations(ctx, nil)
	if len(empty) != 0 {
		t.Errorf("empty input: expected 0 results, got %d", len(empty))
	}
}

// TestListDocuments verifies that ListDocuments returns only KindDocument nodes
// and that the limit parameter is enforced (Invariant 13: BOUNDED).
func TestListDocuments(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "list-docs-limit-test", "List Docs Test", "", "owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	// Seed 3 documents and 1 question (should not appear in document results).
	for i := range 3 {
		if _, err := store.CreateNode(ctx, CreateNodeParams{
			SpaceID: space.ID, Kind: KindDocument,
			Title: fmt.Sprintf("Doc %d", i), Author: "Tester", AuthorID: "owner",
		}); err != nil {
			t.Fatalf("create doc %d: %v", i, err)
		}
	}
	if _, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindQuestion,
		Title: "A question (should not appear in docs)", Author: "Tester", AuthorID: "owner",
	}); err != nil {
		t.Fatalf("create question: %v", err)
	}

	t.Run("returns_only_documents", func(t *testing.T) {
		docs, err := store.ListDocuments(ctx, space.ID, 50)
		if err != nil {
			t.Fatalf("ListDocuments: %v", err)
		}
		if len(docs) != 3 {
			t.Errorf("got %d docs, want 3", len(docs))
		}
		for _, d := range docs {
			if d.Kind != KindDocument {
				t.Errorf("got kind %q, want %q", d.Kind, KindDocument)
			}
		}
	})

	t.Run("limit_enforced", func(t *testing.T) {
		docs, err := store.ListDocuments(ctx, space.ID, 2)
		if err != nil {
			t.Fatalf("ListDocuments with limit: %v", err)
		}
		if len(docs) > 2 {
			t.Errorf("got %d docs with limit=2, want at most 2 (Invariant 13: BOUNDED)", len(docs))
		}
	})
}

// TestListQuestions verifies that ListQuestions returns only KindQuestion nodes
// and that the limit parameter is enforced (Invariant 13: BOUNDED).
func TestListQuestions(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "list-questions-limit-test", "List Questions Test", "", "owner", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	// Seed 3 questions and 1 document (should not appear in question results).
	for i := range 3 {
		if _, err := store.CreateNode(ctx, CreateNodeParams{
			SpaceID: space.ID, Kind: KindQuestion,
			Title: fmt.Sprintf("Question %d", i), Author: "Tester", AuthorID: "owner",
		}); err != nil {
			t.Fatalf("create question %d: %v", i, err)
		}
	}
	if _, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument,
		Title: "A document (should not appear in questions)", Author: "Tester", AuthorID: "owner",
	}); err != nil {
		t.Fatalf("create document: %v", err)
	}

	t.Run("returns_only_questions", func(t *testing.T) {
		questions, err := store.ListQuestions(ctx, space.ID, 50)
		if err != nil {
			t.Fatalf("ListQuestions: %v", err)
		}
		if len(questions) != 3 {
			t.Errorf("got %d questions, want 3", len(questions))
		}
		for _, q := range questions {
			if q.Kind != KindQuestion {
				t.Errorf("got kind %q, want %q", q.Kind, KindQuestion)
			}
		}
	})

	t.Run("limit_enforced", func(t *testing.T) {
		questions, err := store.ListQuestions(ctx, space.ID, 2)
		if err != nil {
			t.Fatalf("ListQuestions with limit: %v", err)
		}
		if len(questions) > 2 {
			t.Errorf("got %d questions with limit=2, want at most 2 (Invariant 13: BOUNDED)", len(questions))
		}
	})
}

// TestListHiveActivity_FiltersAndLimits verifies that ListHiveActivity respects
// the author_id filter and the LIMIT parameter (invariant 13: BOUNDED).
func TestListHiveActivity_FiltersAndLimits(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "hive-activity-test", "Hive Activity Test", "", "owner-ha-test", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	const agentID = "test-hive-agent-bounded-id"
	const otherID = "test-other-agent-bounded-id"

	// Create 3 posts by agentID and 2 by otherID.
	for i := range 3 {
		if _, err := store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindPost,
			Title:      fmt.Sprintf("hive post %d", i),
			Author:     "hive-agent",
			AuthorID:   agentID,
			AuthorKind: "agent",
		}); err != nil {
			t.Fatalf("create hive post %d: %v", i, err)
		}
	}
	for i := range 2 {
		if _, err := store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindPost,
			Title:      fmt.Sprintf("other post %d", i),
			Author:     "other-agent",
			AuthorID:   otherID,
			AuthorKind: "agent",
		}); err != nil {
			t.Fatalf("create other post %d: %v", i, err)
		}
	}

	// author_id filter: only returns posts matching agentID.
	posts, err := store.ListHiveActivity(ctx, agentID, 100)
	if err != nil {
		t.Fatalf("list hive activity: %v", err)
	}
	if len(posts) != 3 {
		t.Errorf("author_id filter: got %d posts, want 3", len(posts))
	}
	for _, p := range posts {
		if p.AuthorID != agentID {
			t.Errorf("got post with author_id %q, want %q", p.AuthorID, agentID)
		}
	}

	// LIMIT is enforced: requesting 2 returns at most 2.
	limited, err := store.ListHiveActivity(ctx, agentID, 2)
	if err != nil {
		t.Fatalf("list hive activity with limit: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("LIMIT: got %d posts, want 2", len(limited))
	}
}

func TestInviteCode(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "invite-code-test", "Invite Code Test", "", "owner-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	t.Run("create_and_get_happy_path", func(t *testing.T) {
		token, err := store.CreateInviteCode(ctx, space.ID, "owner-1", nil, 0)
		if err != nil {
			t.Fatalf("create invite code: %v", err)
		}
		got, err := store.GetInviteCode(ctx, token)
		if err != nil {
			t.Fatalf("get invite code: %v", err)
		}
		if got == nil {
			t.Fatal("expected invite code, got nil")
		}
		if got.SpaceID != space.ID {
			t.Errorf("space_id = %q, want %q", got.SpaceID, space.ID)
		}
		if got.ExpiresAt != nil {
			t.Errorf("expires_at should be nil for unlimited invite")
		}
		if got.MaxUses != 0 {
			t.Errorf("max_uses = %d, want 0 (unlimited)", got.MaxUses)
		}
	})

	t.Run("get_nonexistent", func(t *testing.T) {
		got, err := store.GetInviteCode(ctx, "no-such-token")
		if err != nil {
			t.Fatalf("get invite code: %v", err)
		}
		if got != nil {
			t.Error("expected nil for nonexistent token")
		}
	})

	t.Run("expired", func(t *testing.T) {
		past := time.Now().Add(-time.Hour)
		token, err := store.CreateInviteCode(ctx, space.ID, "owner-1", &past, 0)
		if err != nil {
			t.Fatalf("create invite code: %v", err)
		}
		got, err := store.GetInviteCode(ctx, token)
		if err != nil {
			t.Fatalf("get invite code: %v", err)
		}
		if got != nil {
			t.Error("expected nil for expired invite, got non-nil")
		}
	})

	t.Run("exhausted", func(t *testing.T) {
		token, err := store.CreateInviteCode(ctx, space.ID, "owner-1", nil, 1)
		if err != nil {
			t.Fatalf("create invite code: %v", err)
		}
		if err := store.UseInviteCode(ctx, token, "user-a"); err != nil {
			t.Fatalf("use invite code: %v", err)
		}
		got, err := store.GetInviteCode(ctx, token)
		if err != nil {
			t.Fatalf("get invite code: %v", err)
		}
		if got != nil {
			t.Error("expected nil for exhausted invite (max_uses=1, use_count=1)")
		}
	})

	t.Run("use_idempotent", func(t *testing.T) {
		token, err := store.CreateInviteCode(ctx, space.ID, "owner-1", nil, 5)
		if err != nil {
			t.Fatalf("create invite code: %v", err)
		}
		// Same user uses twice — should only count once.
		if err := store.UseInviteCode(ctx, token, "user-b"); err != nil {
			t.Fatalf("first use: %v", err)
		}
		if err := store.UseInviteCode(ctx, token, "user-b"); err != nil {
			t.Fatalf("second use: %v", err)
		}
		got, err := store.GetInviteCode(ctx, token)
		if err != nil {
			t.Fatalf("get invite code: %v", err)
		}
		if got == nil {
			t.Fatal("invite should still be valid")
		}
		if got.UseCount != 1 {
			t.Errorf("use_count = %d, want 1 (idempotent)", got.UseCount)
		}
		// Different user uses — should count independently.
		if err := store.UseInviteCode(ctx, token, "user-c"); err != nil {
			t.Fatalf("use by different user: %v", err)
		}
		got2, err := store.GetInviteCode(ctx, token)
		if err != nil {
			t.Fatalf("get invite code after second user: %v", err)
		}
		if got2 == nil {
			t.Fatal("invite should still be valid")
		}
		if got2.UseCount != 2 {
			t.Errorf("use_count = %d, want 2", got2.UseCount)
		}
	})
}

func TestListInvitesAndRevoke(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "list-invites-test", "List Invites Test", "", "owner-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	t.Run("empty_list", func(t *testing.T) {
		codes, err := store.ListInvites(ctx, space.ID)
		if err != nil {
			t.Fatalf("list invites: %v", err)
		}
		if len(codes) != 0 {
			t.Errorf("expected 0 invites, got %d", len(codes))
		}
	})

	t.Run("lists_created_invites", func(t *testing.T) {
		tok1, err := store.CreateInviteCode(ctx, space.ID, "owner-1", nil, 0)
		if err != nil {
			t.Fatalf("create invite 1: %v", err)
		}
		tok2, err := store.CreateInviteCode(ctx, space.ID, "owner-1", nil, 0)
		if err != nil {
			t.Fatalf("create invite 2: %v", err)
		}
		t.Cleanup(func() {
			store.RevokeInvite(ctx, tok1)
			store.RevokeInvite(ctx, tok2)
		})

		codes, err := store.ListInvites(ctx, space.ID)
		if err != nil {
			t.Fatalf("list invites: %v", err)
		}
		if len(codes) != 2 {
			t.Errorf("expected exactly 2 invites, got %d", len(codes))
		}
	})

	t.Run("revoke_removes_invite", func(t *testing.T) {
		tok, err := store.CreateInviteCode(ctx, space.ID, "owner-1", nil, 0)
		if err != nil {
			t.Fatalf("create invite: %v", err)
		}

		if err := store.RevokeInvite(ctx, tok); err != nil {
			t.Fatalf("revoke invite: %v", err)
		}

		got, err := store.GetInviteCode(ctx, tok)
		if err != nil {
			t.Fatalf("get invite after revoke: %v", err)
		}
		if got != nil {
			t.Error("expected nil after revoke, got non-nil")
		}
	})
}

func TestUpdateNodeCauses(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	space, err := store.CreateSpace(ctx, "test-update-causes", "Update Causes", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	node, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindClaim,
		Title:    "Test Claim",
		Author:   "tester",
		AuthorID: "tester-id",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	// Node starts with no causes.
	if len(node.Causes) != 0 {
		t.Errorf("initial causes = %v, want []", node.Causes)
	}

	// Set causes.
	causeID := "task-node-abc123"
	if err := store.UpdateNodeCauses(ctx, node.ID, []string{causeID}); err != nil {
		t.Fatalf("UpdateNodeCauses: %v", err)
	}

	// Verify causes persisted.
	got, err := store.GetNode(ctx, node.ID)
	if err != nil {
		t.Fatalf("get node: %v", err)
	}
	if len(got.Causes) != 1 || got.Causes[0] != causeID {
		t.Errorf("causes = %v, want [%q]", got.Causes, causeID)
	}

	// Update with multiple causes.
	causeIDs := []string{"task-aaa", "task-bbb"}
	if err := store.UpdateNodeCauses(ctx, node.ID, causeIDs); err != nil {
		t.Fatalf("UpdateNodeCauses multi: %v", err)
	}
	got, _ = store.GetNode(ctx, node.ID)
	if len(got.Causes) != 2 {
		t.Errorf("causes = %v, want 2 entries", got.Causes)
	}

	// Update with nil causes — should store empty slice, not error.
	if err := store.UpdateNodeCauses(ctx, node.ID, nil); err != nil {
		t.Fatalf("UpdateNodeCauses nil: %v", err)
	}
	got, _ = store.GetNode(ctx, node.ID)
	if len(got.Causes) != 0 {
		t.Errorf("causes after nil update = %v, want []", got.Causes)
	}

	// Non-existent node returns ErrNotFound.
	err = store.UpdateNodeCauses(ctx, "nonexistent-id", []string{"x"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing node, got %v", err)
	}
}
