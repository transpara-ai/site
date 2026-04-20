package graph

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// TestHiveCostStr verifies the formatted cost badge string.
func TestHiveCostStr(t *testing.T) {
	cases := []struct {
		body string
		want string
	}{
		{"Builder shipped. Cost: $0.42. Duration: 3m28s.", "$0.42"},
		{"Cost: $1.00.", "$1.00"},
		{"No cost here.", ""},
		{"Cost: $0.00", ""},
	}
	for _, c := range cases {
		got := hiveCostStr(Node{Body: c.body})
		if got != c.want {
			t.Errorf("hiveCostStr(%q) = %q, want %q", c.body, got, c.want)
		}
	}
}

// TestHiveDurationStr verifies the duration badge string.
func TestHiveDurationStr(t *testing.T) {
	cases := []struct {
		body string
		want string
	}{
		{"Cost: $0.42. Duration: 3m28s.", "3m28s"},
		{"Duration: 12m", "12m"},
		{"No duration here.", ""},
		{"", ""},
	}
	for _, c := range cases {
		got := hiveDurationStr(Node{Body: c.body})
		if got != c.want {
			t.Errorf("hiveDurationStr(%q) = %q, want %q", c.body, got, c.want)
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
	architectPost := Node{
		Title:     "[hive:architect] iter 240: created tasks",
		CreatedAt: now.Add(-10 * time.Minute),
	}

	roles := computePipelineRoles([]Node{recentPost, oldPost, architectPost})

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

	// Architect: recent post — should be active.
	architect, ok := roleByName["Architect"]
	if !ok {
		t.Fatal("Architect role missing from result")
	}
	if !architect.Active {
		t.Error("Architect should be Active (post within activeRoleThreshold)")
	}
	if architect.LastActive.IsZero() {
		t.Error("Architect LastActive should not be zero")
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

// TestGetHive_ContainsHiveFeed verifies GET /hive returns 200 and includes the
// hive-feed element rendered by HivePage regardless of loop state.
func TestGetHive_ContainsHiveFeed(t *testing.T) {
	h, _, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/hive", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /hive: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "hive-feed") {
		t.Error("GET /hive: body does not contain 'hive-feed' element")
	}
}

// TestGetHiveCurrentTask_ScopedToActor verifies that GetHiveCurrentTask only returns
// tasks authored by the specified actor when actorID is provided — even when a second
// agent actor also has open tasks.
func TestGetHiveCurrentTask_ScopedToActor(t *testing.T) {
	_, store := testDB(t)
	ctx := t.Context()

	slug := fmt.Sprintf("hive-scope-task-test-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(ctx, slug, "Hive Scope Task", "", "owner-scope-task", "project", "public")
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

	nonce := time.Now().UnixNano()
	slug := fmt.Sprintf("hive-scope-totals-test-%d", nonce)
	// GetHiveTotals counts ops by actor ID across all spaces, so randomize the
	// actor IDs too — otherwise stale ops from prior runs (or other tests that
	// reuse the literal) inflate the count.
	actorA := fmt.Sprintf("hive-totals-actor-a-%d", nonce)
	actorB := fmt.Sprintf("hive-totals-actor-b-%d", nonce)

	space, err := store.CreateSpace(ctx, slug, "Hive Scope Totals", "", "owner-scope-totals", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	node, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindTask,
		Title:      "Scope totals test node",
		Author:     "agent-a",
		AuthorID:   actorA,
		AuthorKind: "agent",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	// 2 ops for actor A, 1 for actor B.
	for range 2 {
		if _, err := store.RecordOp(ctx, space.ID, node.ID, "agent-a", actorA, "express", nil); err != nil {
			t.Fatalf("record op A: %v", err)
		}
	}
	if _, err := store.RecordOp(ctx, space.ID, node.ID, "agent-b", actorB, "express", nil); err != nil {
		t.Fatalf("record op B: %v", err)
	}

	totalA, _, err := store.GetHiveTotals(ctx, actorA)
	if err != nil {
		t.Fatalf("GetHiveTotals actorA: %v", err)
	}
	if totalA != 2 {
		t.Errorf("GetHiveTotals(actorA) = %d, want 2", totalA)
	}

	totalB, _, err := store.GetHiveTotals(ctx, actorB)
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

	nonce := time.Now().UnixNano()
	agentUserID := fmt.Sprintf("hive-get-agent-id-test-actor-%d", nonce)
	apiKeyID := fmt.Sprintf("hive-get-agent-id-apikey-test-%d", nonce)
	agentEmail := fmt.Sprintf("hive-agent-id-test-%d@test.local", nonce)

	// Insert a test agent user.
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email, name, kind) VALUES ($1, $2, $3, 'agent')`,
		agentUserID, agentEmail, "test-hive-agent-id-func")
	if err != nil {
		t.Fatalf("insert agent user: %v", err)
	}
	t.Cleanup(func() {
		// t.Context() is cancelled by the time cleanup runs, so use a fresh
		// background context — otherwise these DELETEs silently no-op and
		// stale rows bleed into later runs.
		cleanupCtx := context.Background()
		db.ExecContext(cleanupCtx, `DELETE FROM api_keys WHERE id = $1`, apiKeyID)
		db.ExecContext(cleanupCtx, `DELETE FROM users WHERE id = $1`, agentUserID)
	})

	// Insert an api_keys row linking to the agent. Use a very old created_at so this
	// row is always returned first by ORDER BY created_at ASC LIMIT 1.
	// user_id is the key owner (FK to users.id); reuse agentUserID so we don't
	// depend on another test having seeded a "test-owner-id" user.
	_, err = db.ExecContext(ctx,
		`INSERT INTO api_keys (id, key_hash, user_id, agent_id, created_at) VALUES ($1, $2, $3, $4, '2020-01-01 00:00:00+00')`,
		apiKeyID, fmt.Sprintf("hive-get-agent-id-testhash-%d", nonce), agentUserID, agentUserID)
	if err != nil {
		t.Fatalf("insert api_key: %v", err)
	}

	// The integration path: api_keys row → GetHiveAgentID → correct actor_id.
	got := store.GetHiveAgentID(ctx)
	if got != agentUserID {
		t.Errorf("GetHiveAgentID = %q, want %q", got, agentUserID)
	}
}

// TestGetHiveStatus_Partial verifies GET /hive/status returns 200 with the main
// content element (tasks + build log) and no full HTML shell.
func TestGetHiveStatus_Partial(t *testing.T) {
	h, _, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/hive/status", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /hive/status: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "hive-content") {
		t.Error("GET /hive/status: body does not contain 'hive-content' element")
	}
	if strings.Contains(body, "<html") {
		t.Error("GET /hive/status: body contains full HTML shell — partial should not include <html>")
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

// TestHiveDashboard verifies that GET /hive reads loop state files from loopDir and
// renders the iteration number, phase, and current build title in the response.
func TestHiveDashboard(t *testing.T) {
	dir := t.TempDir()

	// diagnostics.jsonl — three realistic entries.
	diagnostics := `{"timestamp":"2026-03-27T10:00:00Z","phase":"Builder","iteration":7,"event":"build_start"}
{"timestamp":"2026-03-27T10:05:00Z","phase":"Builder","iteration":7,"event":"build_progress","detail":"editing handlers.go"}
{"timestamp":"2026-03-27T10:10:00Z","phase":"Builder","iteration":7,"event":"build_complete","cost":0.42}
`
	if err := os.WriteFile(filepath.Join(dir, "diagnostics.jsonl"), []byte(diagnostics), 0600); err != nil {
		t.Fatalf("write diagnostics.jsonl: %v", err)
	}

	// state.md — current iteration and phase.
	stateMd := "Iteration: 7\nPhase: Builder\n"
	if err := os.WriteFile(filepath.Join(dir, "state.md"), []byte(stateMd), 0600); err != nil {
		t.Fatalf("write state.md: %v", err)
	}

	// build.md — current build title and cost.
	buildMd := "# Add kanban board\n\nCost: $0.42\n"
	if err := os.WriteFile(filepath.Join(dir, "build.md"), []byte(buildMd), 0600); err != nil {
		t.Fatalf("write build.md: %v", err)
	}

	h, _, _ := testHandlers(t)
	h.SetLoopDir(dir)

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/hive", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /hive with loop state: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "7") {
		t.Error("response body does not contain iteration number 7")
	}
	if !strings.Contains(body, "Builder") {
		t.Error("response body does not contain phase 'Builder'")
	}
	if !strings.Contains(body, "Add kanban board") {
		t.Error("response body does not contain build title 'Add kanban board'")
	}
}

// TestGetHiveFeed_PublicNoAuth verifies GET /hive/feed returns 200 without auth
// and returns the standalone phase-timeline partial (no full HTML shell).
func TestGetHiveFeed_PublicNoAuth(t *testing.T) {
	h, _, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/hive/feed", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /hive/feed without auth: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "hive-feed") {
		t.Error("GET /hive/feed: body does not contain 'hive-feed' element")
	}
	if strings.Contains(body, "<html") {
		t.Error("GET /hive/feed: body contains full HTML shell — partial must not include <html>")
	}
}

// TestPostHiveDiagnostic_StoresAndServes verifies the full round-trip:
// POST /api/hive/diagnostic stores the event and GET /hive/feed renders it.
func TestPostHiveDiagnostic_StoresAndServes(t *testing.T) {
	h, _, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	// POST a diagnostic event (auth injected by testHandlers).
	payload := `{"phase":"builder","outcome":"success","cost_usd":0.42,"timestamp":"2026-03-29T10:00:00Z"}`
	req := httptest.NewRequest("POST", "/api/hive/diagnostic", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/hive/diagnostic: status = %d, want 201; body: %s", w.Code, w.Body.String())
	}

	// GET /hive/feed must now include the stored event.
	req2 := httptest.NewRequest("GET", "/hive/feed", nil)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("GET /hive/feed after diagnostic: status = %d, want 200; body: %s", w2.Code, w2.Body.String())
	}
	if !strings.Contains(w2.Body.String(), "builder") {
		t.Error("GET /hive/feed: body does not contain posted phase 'builder'")
	}
}

// TestListHiveDiagnostics_Empty verifies ListHiveDiagnostics returns nil (not error)
// when no diagnostics have been stored yet.
func TestListHiveDiagnostics_Empty(t *testing.T) {
	db, store := testDB(t)
	ctx := t.Context()
	// Truncate to isolate from rows inserted by TestPostHiveDiagnostic_StoresAndServes
	// or any prior run. The hive_diagnostics table has no foreign-key dependents.
	if _, err := db.ExecContext(ctx, `DELETE FROM hive_diagnostics`); err != nil {
		t.Fatalf("clear hive_diagnostics: %v", err)
	}
	entries, err := store.ListHiveDiagnostics(ctx, 10)
	if err != nil {
		t.Fatalf("ListHiveDiagnostics: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("ListHiveDiagnostics on empty DB: got %d entries, want 0", len(entries))
	}
}
