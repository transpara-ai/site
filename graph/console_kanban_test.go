package graph

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFetchConsoleWorkDecodesTasks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tasks":[
			{"id":"task_1","title":"Build civic-roles doc","status":"running",
			 "assignee":"implementer","created_by":"michael","risk_class":"high",
			 "cell":"cell_a","factory_order_id":"fo_42","created_at":"2026-06-30T12:00:00Z"}
		]}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban", nil)
	res := fetchConsoleWork(req)
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if len(res.Tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(res.Tasks))
	}
	if res.Tasks[0].RiskClass != "high" || res.Tasks[0].CreatedBy != "michael" {
		t.Errorf("enriched fields not decoded: %+v", res.Tasks[0])
	}
}

func TestFetchConsoleWorkReportsUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban", nil)
	res := fetchConsoleWork(req)
	if res.Err == nil {
		t.Fatal("want an error on non-2xx upstream, got nil")
	}
	if len(res.Tasks) != 0 {
		t.Fatalf("want zero tasks on error, got %d", len(res.Tasks))
	}
}

func TestOpsWorkTaskDecodesKanbanFields(t *testing.T) {
	const body = `{
		"id": "task_1", "title": "Build civic-roles doc",
		"status": "running", "assignee": "implementer",
		"created_by": "michael", "risk_class": "high",
		"cell": "cell_a", "factory_order_id": "fo_42",
		"created_at": "2026-06-30T12:00:00Z"
	}`
	var task OpsWorkTask
	if err := json.Unmarshal([]byte(body), &task); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if task.CreatedBy != "michael" {
		t.Errorf("CreatedBy = %q, want michael", task.CreatedBy)
	}
	if task.RiskClass != "high" {
		t.Errorf("RiskClass = %q, want high", task.RiskClass)
	}
	if task.Cell != "cell_a" {
		t.Errorf("Cell = %q, want cell_a", task.Cell)
	}
	if task.FactoryOrderID != "fo_42" {
		t.Errorf("FactoryOrderID = %q, want fo_42", task.FactoryOrderID)
	}
	if task.CreatedAt != "2026-06-30T12:00:00Z" {
		t.Errorf("CreatedAt = %q, want the RFC3339 string", task.CreatedAt)
	}
}

func sampleKanbanTasks() []OpsWorkTask {
	return []OpsWorkTask{
		{ID: "t1", Title: "Alpha", Status: "running", Assignee: "implementer",
			CreatedBy: "michael", RiskClass: "high", CreatedAt: "2026-06-29T12:00:00Z"},
		{ID: "t2", Title: "Bravo", Status: "blocked", Assignee: "",
			CreatedBy: "codex", RiskClass: "critical", CreatedAt: "2026-06-28T12:00:00Z"},
		{ID: "t3", Title: "Charlie", Status: "running", Assignee: "implementer",
			CreatedBy: "michael", RiskClass: "", CreatedAt: ""},
	}
}

func columnKeys(k ConsoleKanban) []string {
	keys := make([]string, 0, len(k.Columns))
	for _, c := range k.Columns {
		keys = append(keys, c.Key)
	}
	return keys
}

func TestBuildConsoleKanbanFetchErrorIsUnavailable(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	k := buildConsoleKanban(nil, fmt.Errorf("boom"), LensRisk, now)
	if k.Freshness != FreshnessUnavailable {
		t.Errorf("freshness = %q, want unavailable", k.Freshness)
	}
	if k.TotalCards != 0 || len(k.Columns) != 0 {
		t.Errorf("want zero cards/columns on error, got %d cards %d cols", k.TotalCards, len(k.Columns))
	}
	if len(k.Notices) == 0 {
		t.Error("want a notice explaining the unavailable state")
	}
}

func TestBuildConsoleKanbanRiskLensOrderAndUnclassified(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	k := buildConsoleKanban(sampleKanbanTasks(), nil, LensRisk, now)
	if k.Freshness != FreshnessCurrent {
		t.Errorf("freshness = %q, want current on a clean fetch", k.Freshness)
	}
	// critical, high present; low/medium absent (omitted); unclassified (empty risk) last.
	if got := columnKeys(k); !reflect.DeepEqual(got, []string{"critical", "high", "unclassified"}) {
		t.Errorf("risk columns = %v, want [critical high unclassified]", got)
	}
	if k.TotalCards != 3 {
		t.Errorf("TotalCards = %d, want 3", k.TotalCards)
	}
}

func TestBuildConsoleKanbanStatusLensOrder(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	k := buildConsoleKanban(sampleKanbanTasks(), nil, LensStatus, now)
	// lifecycle order: running before blocked.
	if got := columnKeys(k); !reflect.DeepEqual(got, []string{"running", "blocked"}) {
		t.Errorf("status columns = %v, want [running blocked]", got)
	}
}

func TestBuildConsoleKanbanAgentLensUnassignedLast(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	k := buildConsoleKanban(sampleKanbanTasks(), nil, LensAgent, now)
	if got := columnKeys(k); !reflect.DeepEqual(got, []string{"implementer", "unassigned"}) {
		t.Errorf("agent columns = %v, want [implementer unassigned]", got)
	}
}

func TestBuildConsoleKanbanSourceLens(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	k := buildConsoleKanban(sampleKanbanTasks(), nil, LensSource, now)
	if got := columnKeys(k); !reflect.DeepEqual(got, []string{"codex", "michael"}) {
		t.Errorf("source columns = %v, want [codex michael]", got)
	}
}

func TestHumanizeAge(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name, in, want string
	}{
		{"days", "2026-06-28T12:00:00Z", "2d"},
		{"hours", "2026-06-30T09:00:00Z", "3h"},
		{"minutes", "2026-06-30T11:30:00Z", "30m"},
		{"empty", "", ""},
		{"unparseable", "not-a-time", ""},
		{"future", "2026-07-01T12:00:00Z", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := humanizeAge(now, c.in); got != c.want {
				t.Errorf("humanizeAge(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestParseLensDefaultsToRisk(t *testing.T) {
	for _, raw := range []string{"", "bogus", "RISK?"} {
		if got := parseLens(raw); got != LensRisk {
			t.Errorf("parseLens(%q) = %q, want risk", raw, got)
		}
	}
	if parseLens("status") != LensStatus {
		t.Error("parseLens(status) should be status")
	}
}

func TestBuildConsoleKanbanStatusLensEmptyIsUnknownColumn(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	tasks := []OpsWorkTask{
		{ID: "a", Title: "A", Status: "running", CreatedBy: "michael", CreatedAt: "2026-06-29T12:00:00Z"},
		{ID: "b", Title: "B", Status: "", CreatedBy: "michael", CreatedAt: "2026-06-28T12:00:00Z"},
	}
	k := buildConsoleKanban(tasks, nil, LensStatus, now)
	if got := columnKeys(k); !reflect.DeepEqual(got, []string{"running", "unknown"}) {
		t.Errorf("status columns = %v, want [running unknown]", got)
	}
	if k.TotalCards != 2 {
		t.Errorf("TotalCards = %d, want 2 (no card dropped)", k.TotalCards)
	}
}

func TestBuildConsoleKanbanSourceLensEmptyIsUnknownColumn(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	tasks := []OpsWorkTask{
		{ID: "a", Title: "A", Status: "running", CreatedBy: "michael", CreatedAt: "2026-06-29T12:00:00Z"},
		{ID: "b", Title: "B", Status: "running", CreatedBy: "", CreatedAt: "2026-06-28T12:00:00Z"},
	}
	k := buildConsoleKanban(tasks, nil, LensSource, now)
	if got := columnKeys(k); !reflect.DeepEqual(got, []string{"michael", "unknown"}) {
		t.Errorf("source columns = %v, want [michael unknown]", got)
	}
	if k.TotalCards != 2 {
		t.Errorf("TotalCards = %d, want 2 (no card dropped)", k.TotalCards)
	}
}

func TestBuildConsoleKanbanWithinColumnOldestFirst(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	tasks := []OpsWorkTask{
		{ID: "newer", Title: "Newer", RiskClass: "high", CreatedAt: "2026-06-29T12:00:00Z"},
		{ID: "older", Title: "Older", RiskClass: "high", CreatedAt: "2026-06-27T12:00:00Z"},
	}
	k := buildConsoleKanban(tasks, nil, LensRisk, now)
	if len(k.Columns) != 1 {
		t.Fatalf("got %d columns, want 1 (both high)", len(k.Columns))
	}
	cards := k.Columns[0].Cards
	if len(cards) != 2 || cards[0].ID != "older" || cards[1].ID != "newer" {
		t.Errorf("within-column order = %v, want [older newer] (oldest-first)", []string{cards[0].ID, cards[1].ID})
	}
}

func TestConsoleKanbanRendersOrderCards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tasks":[
			{"id":"task_1","title":"Build civic-roles doc","status":"running",
			 "assignee":"implementer","created_by":"michael","risk_class":"high",
			 "cell":"cell_a","factory_order_id":"fo_42","created_at":"2026-06-29T12:00:00Z"}
		]}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"Build civic-roles doc", "fo_42", "michael", "implementer", "high"} {
		if !strings.Contains(body, want) {
			t.Errorf("kanban body missing %q", want)
		}
	}
}

func TestConsoleKanbanUpstreamErrorIsHonest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unavailable") {
		t.Error("expected an explicit unavailable state on upstream error")
	}
}

func TestConsoleOrderDrawerShowsUnavailableForEffortAndPR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tasks":[
			{"id":"task_1","title":"Build civic-roles doc","status":"running",
			 "assignee":"implementer","created_by":"michael","risk_class":"high",
			 "factory_order_id":"fo_42","created_at":"2026-06-29T12:00:00Z"}
		]}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban/order/task_1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Build civic-roles doc") {
		t.Error("drawer missing order title")
	}
	// effort + linked PR are genuinely not in any projection — must read "unavailable", never fabricated.
	if strings.Count(body, "unavailable") < 2 {
		t.Errorf("drawer must mark effort and linked-PR unavailable; body: %s", body)
	}
}

func TestConsoleOrderDrawerUnknownIDIsHonest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tasks":[]}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban/order/nope", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unavailable") && !strings.Contains(w.Body.String(), "not found") {
		t.Error("unknown order must render an honest not-found/unavailable drawer, not a fabricated order")
	}
}
