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

	tasks, err := store.ListAvailableTasks(ctx, "", 50)
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
	tasks, _ = store.ListAvailableTasks(ctx, "Available", 50)
	if len(tasks) == 0 {
		t.Error("search should find the task")
	}
	tasks, _ = store.ListAvailableTasks(ctx, "nonexistent", 50)
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
