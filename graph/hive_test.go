package graph

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestParseCostDollars verifies cost extraction from post bodies.
func TestParseCostDollars(t *testing.T) {
	cases := []struct {
		body string
		want float64
	}{
		{"Builder shipped the feature. Cost: $0.53. Duration: 3m28s.", 0.53},
		{"No cost here.", 0},
		{"Multiple $1.00 and $2.50 — first wins.", 1.00},
		{"Cost: $0.00", 0},
	}
	for _, c := range cases {
		got := parseCostDollars(c.body)
		if got != c.want {
			t.Errorf("parseCostDollars(%q) = %.2f, want %.2f", c.body, got, c.want)
		}
	}
}

// TestParseDurationStr verifies duration extraction from post bodies.
func TestParseDurationStr(t *testing.T) {
	cases := []struct {
		body string
		want string
	}{
		{"Cost: $0.53. Duration: 3m28s.", "3m28s"},
		{"Duration: 12m", "12m"},
		{"No duration here.", ""},
		{"Duration: 0m5s.", "0m5s"},
	}
	for _, c := range cases {
		got := parseDurationStr(c.body)
		if got != c.want {
			t.Errorf("parseDurationStr(%q) = %q, want %q", c.body, got, c.want)
		}
	}
}

// TestComputeHiveStats verifies aggregate cost metrics.
func TestComputeHiveStats(t *testing.T) {
	posts := []Node{
		{Body: "Shipped X. Cost: $1.00. Duration: 2m."},
		{Body: "Shipped Y. Cost: $0.50."},
		{Body: "No cost here."},
	}
	stats := computeHiveStats(posts)
	if stats.Features != 2 {
		t.Errorf("Features = %d, want 2", stats.Features)
	}
	if stats.TotalCost != 1.50 {
		t.Errorf("TotalCost = %.2f, want 1.50", stats.TotalCost)
	}
	want := 0.75
	if stats.AvgCost != want {
		t.Errorf("AvgCost = %.2f, want %.2f", stats.AvgCost, want)
	}
}

// TestComputePipelineRoles verifies active/idle state and last-active timestamps.
func TestComputePipelineRoles(t *testing.T) {
	now := time.Now()
	recentPost := Node{
		Title:     "[hive:builder] iter 240: shipped feature",
		CreatedAt: now.Add(-5 * time.Minute),
	}
	oldPost := Node{
		Title:     "[hive:scout] iter 238: scouted gap",
		CreatedAt: now.Add(-2 * time.Hour),
	}

	roles := computePipelineRoles([]Node{recentPost, oldPost})

	roleByName := make(map[string]PipelineRole, len(roles))
	for _, r := range roles {
		roleByName[r.Name] = r
	}

	// Builder: recent post — should be active.
	builder, ok := roleByName["Builder"]
	if !ok {
		t.Fatal("Builder role missing from result")
	}
	if !builder.Active {
		t.Error("Builder should be Active (post within activeRoleThreshold)")
	}
	if builder.LastActive.IsZero() {
		t.Error("Builder LastActive should not be zero")
	}

	// Scout: old post — should be inactive.
	scout, ok := roleByName["Scout"]
	if !ok {
		t.Fatal("Scout role missing from result")
	}
	if scout.Active {
		t.Error("Scout should not be Active (post older than activeRoleThreshold)")
	}
	if scout.LastActive.IsZero() {
		t.Error("Scout LastActive should not be zero")
	}

	// Critic: no posts — should be idle with zero LastActive.
	critic, ok := roleByName["Critic"]
	if !ok {
		t.Fatal("Critic role missing from result")
	}
	if critic.Active {
		t.Error("Critic should not be Active (no posts)")
	}
	if !critic.LastActive.IsZero() {
		t.Error("Critic LastActive should be zero (never seen)")
	}
}

// TestGetHive_PublicNoAuth verifies GET /hive returns 200 without an auth cookie.
func TestGetHive_PublicNoAuth(t *testing.T) {
	h, _, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/hive", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /hive without auth: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

// TestGetHive_RendersMetrics verifies the hive page contains stat card text
// when seeded with hive agent posts.
func TestGetHive_RendersMetrics(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	// Create a public space to house the hive agent posts.
	space, err := store.CreateSpace(t.Context(), "hive-metrics-test", "Hive Metrics Test", "", "owner-hive-metrics", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	// Seed two posts by a hive agent (author_kind = "agent") with cost metadata.
	for i := range 2 {
		_, err := store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindPost,
			Title:      fmt.Sprintf("[hive:builder] iter %d: shipped feature", i+230),
			Body:       fmt.Sprintf("Builder shipped the feature. Cost: $0.42. Duration: %dm30s.", i+3),
			Author:     "hive-builder",
			AuthorID:   "hive-agent-metrics-test-id",
			AuthorKind: "agent",
		})
		if err != nil {
			t.Fatalf("create post %d: %v", i, err)
		}
	}

	req := httptest.NewRequest("GET", "/hive", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, label := range []string{"Features shipped", "Total autonomous spend", "Avg cost"} {
		if !strings.Contains(body, label) {
			t.Errorf("response body does not contain stat card label %q", label)
		}
	}
}

// TestGetHive_RendersCurrentlyBuilding verifies the "Currently building" section
// shows a task title when an open agent task exists, and "Idle" when none exists.
func TestGetHive_RendersCurrentlyBuilding(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	// Without any agent tasks, "Idle" should appear.
	req := httptest.NewRequest("GET", "/hive", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Idle") {
		t.Error("expected 'Idle' when no agent tasks exist")
	}

	// Create a space and seed an open agent task.
	space, err := store.CreateSpace(t.Context(), "hive-task-test", "Hive Task Test", "", "owner-hive-task", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	_, err = store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindTask,
		Title:      "Build the knowledge layer",
		Body:       "Layer 6 implementation.",
		Author:     "hive-strategist",
		AuthorID:   "hive-agent-task-test-id",
		AuthorKind: "agent",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	req2 := httptest.NewRequest("GET", "/hive", nil)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w2.Code, w2.Body.String())
	}
	if !strings.Contains(w2.Body.String(), "Build the knowledge layer") {
		t.Error("expected task title in 'Currently building' section")
	}
}

// TestGetHiveCurrentTask_ScopedToActor verifies that GetHiveCurrentTask only returns
// tasks authored by the specified actor when actorID is provided — even when a second
// agent actor also has open tasks.
func TestGetHiveCurrentTask_ScopedToActor(t *testing.T) {
	_, store := testDB(t)
	ctx := t.Context()

	space, err := store.CreateSpace(ctx, "hive-scope-task-test", "Hive Scope Task", "", "owner-scope-task", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	_, err = store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindTask,
		Title:      "Task from actor A",
		Author:     "agent-a",
		AuthorID:   "hive-scope-actor-a",
		AuthorKind: "agent",
	})
	if err != nil {
		t.Fatalf("create task actor A: %v", err)
	}
	_, err = store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindTask,
		Title:      "Task from actor B",
		Author:     "agent-b",
		AuthorID:   "hive-scope-actor-b",
		AuthorKind: "agent",
	})
	if err != nil {
		t.Fatalf("create task actor B: %v", err)
	}

	nodeA, err := store.GetHiveCurrentTask(ctx, "hive-scope-actor-a")
	if err != nil {
		t.Fatalf("GetHiveCurrentTask actor A: %v", err)
	}
	if nodeA == nil {
		t.Fatal("expected task for actor A, got nil")
	}
	if nodeA.Title != "Task from actor A" {
		t.Errorf("actor A task title = %q, want %q", nodeA.Title, "Task from actor A")
	}

	nodeB, err := store.GetHiveCurrentTask(ctx, "hive-scope-actor-b")
	if err != nil {
		t.Fatalf("GetHiveCurrentTask actor B: %v", err)
	}
	if nodeB == nil {
		t.Fatal("expected task for actor B, got nil")
	}
	if nodeB.Title != "Task from actor B" {
		t.Errorf("actor B task title = %q, want %q", nodeB.Title, "Task from actor B")
	}
}

// TestGetHiveTotals_ScopedToActor verifies that GetHiveTotals counts only ops
// by the specified actor when actorID is provided.
func TestGetHiveTotals_ScopedToActor(t *testing.T) {
	_, store := testDB(t)
	ctx := t.Context()

	space, err := store.CreateSpace(ctx, "hive-scope-totals-test", "Hive Scope Totals", "", "owner-scope-totals", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	node, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindTask,
		Title:      "Scope totals test node",
		Author:     "agent-a",
		AuthorID:   "hive-totals-actor-a",
		AuthorKind: "agent",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	// 2 ops for actor A, 1 for actor B.
	for range 2 {
		if _, err := store.RecordOp(ctx, space.ID, node.ID, "agent-a", "hive-totals-actor-a", "express", nil); err != nil {
			t.Fatalf("record op A: %v", err)
		}
	}
	if _, err := store.RecordOp(ctx, space.ID, node.ID, "agent-b", "hive-totals-actor-b", "express", nil); err != nil {
		t.Fatalf("record op B: %v", err)
	}

	totalA, _, err := store.GetHiveTotals(ctx, "hive-totals-actor-a")
	if err != nil {
		t.Fatalf("GetHiveTotals actorA: %v", err)
	}
	if totalA != 2 {
		t.Errorf("GetHiveTotals(actorA) = %d, want 2", totalA)
	}

	totalB, _, err := store.GetHiveTotals(ctx, "hive-totals-actor-b")
	if err != nil {
		t.Fatalf("GetHiveTotals actorB: %v", err)
	}
	if totalB != 1 {
		t.Errorf("GetHiveTotals(actorB) = %d, want 1", totalB)
	}
}

// TestGetHiveAgentID_IntegrationPath verifies that GetHiveAgentID returns the actor ID
// of the agent linked via the api_keys table: api_keys row → GetHiveAgentID → correct actor_id.
func TestGetHiveAgentID_IntegrationPath(t *testing.T) {
	db, store := testDB(t)
	ctx := t.Context()

	const agentUserID = "hive-get-agent-id-test-actor"
	const apiKeyID = "hive-get-agent-id-apikey-test"

	// Insert a test agent user.
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, kind) VALUES ($1, $2, $3, 'agent')`,
		agentUserID, "hive-agent-id-test@test.local", "test-hive-agent-id-func")
	if err != nil {
		t.Fatalf("insert agent user: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM api_keys WHERE id = $1`, apiKeyID)
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentUserID)
	})

	// Insert an api_keys row linking to the agent. Use a very old created_at so this
	// row is always returned first by ORDER BY created_at ASC LIMIT 1.
	_, err = db.ExecContext(ctx,
		`INSERT INTO api_keys (id, key_hash, user_id, agent_id, created_at) VALUES ($1, $2, $3, $4, '2020-01-01 00:00:00+00')`,
		apiKeyID, "hive-get-agent-id-testhash", "test-owner-id", agentUserID)
	if err != nil {
		t.Fatalf("insert api_key: %v", err)
	}

	// The integration path: api_keys row → GetHiveAgentID → correct actor_id.
	got := store.GetHiveAgentID(ctx)
	if got != agentUserID {
		t.Errorf("GetHiveAgentID = %q, want %q", got, agentUserID)
	}
}

// TestGetHiveStats_Partial verifies GET /hive/stats returns 200 with stats bar HTML.
func TestGetHiveStats_Partial(t *testing.T) {
	h, _, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/hive/stats", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /hive/stats: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "total ops") {
		t.Error("expected 'total ops' in /hive/stats response")
	}
}
