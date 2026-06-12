package graph

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuildObsAgentsJoinsHistoryAndSorts(t *testing.T) {
	trust := 0.42
	agents := []obsStatusAgent{
		{Role: "scout", ActorID: "b-2", State: "processing", CostUSD: 1.5, TrustScore: &trust, LastEventAt: time.Date(2026, 6, 13, 1, 2, 3, 0, time.UTC)},
		{Role: "builder", ActorID: "a-1", State: "idle", CostUSD: 0},
	}
	histories := map[string]obsAgentHistory{
		"b-2": {ActorID: "b-2", States: []obsStateSpan{{State: "processing", Duration: 60, EnteredAt: time.Now()}}},
	}
	views := buildObsAgents(agents, histories)
	if len(views) != 2 {
		t.Fatalf("want 2 views, got %d", len(views))
	}
	if views[0].Role != "builder" || views[1].Role != "scout" {
		t.Errorf("views must sort by role: got %s, %s", views[0].Role, views[1].Role)
	}
	if views[0].Trust != "unknown" {
		t.Errorf("nil trust must render as unknown, got %q", views[0].Trust)
	}
	if views[1].Trust != "0.42" {
		t.Errorf("trust must format, got %q", views[1].Trust)
	}
	if views[0].TimelineSVG != "" {
		t.Error("agent without history must have empty timeline (template renders the reason)")
	}
	if views[1].TimelineSVG == "" {
		t.Error("agent with history must have a timeline strip")
	}
	if views[1].CostUSD != "1.50" {
		t.Errorf("cost must format to cents, got %q", views[1].CostUSD)
	}
}

func TestPhaseCostBarsFailsClosed(t *testing.T) {
	if svg, _, _ := phaseCostBars(nil); svg != "" {
		t.Error("nil report must render nothing")
	}
	report := &OpsPipelineReport{Phases: []OpsPipelinePhase{{Phase: "scout", CostUSD: -1}}}
	if svg, labels, total := phaseCostBars(report); svg != "" || labels != nil || total != 0 {
		t.Error("negative cost must fail closed entirely")
	}
	zero := &OpsPipelineReport{Phases: []OpsPipelinePhase{{Phase: "scout", CostUSD: 0}}}
	if svg, labels, _ := phaseCostBars(zero); svg != "" || len(labels) != 1 {
		t.Error("all-zero costs: no chart (svgviz fails closed) but labels preserved")
	}
}

func TestFetchObservatoryStatusDecodesAndAuths(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/telemetry/status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"agents":[{"role":"scout","actor_id":"a1","state":"processing","cost_usd":2.25,"trust_score":null}],"hive":{"active_agents":3,"total_actors":7,"chain_length":1234,"chain_ok":true,"event_rate":null,"daily_cost":11.5,"daily_cap":25,"severity":"ok"},"timestamp":"2026-06-13T01:00:00Z"}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)
	t.Setenv("WORK_API_KEY", "")

	r := httptest.NewRequest(http.MethodGet, "http://site/ops/observatory", nil)
	payload, url, err := fetchObservatoryStatus(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(url, "/telemetry/status") {
		t.Errorf("provenance URL must be the real endpoint, got %s", url)
	}
	if gotAuth != "Bearer dev" {
		t.Errorf("must send work auth header, got %q", gotAuth)
	}
	if payload.Hive == nil || payload.Hive.DailyCap == nil || *payload.Hive.DailyCap != 25 {
		t.Error("hive vitals must decode")
	}
	if payload.Hive.EventRate != nil {
		t.Error("null event_rate must stay nil (renders as unknown)")
	}
	if len(payload.Agents) != 1 || payload.Agents[0].TrustScore != nil {
		t.Error("agents must decode with null trust preserved")
	}
}

func TestFetchObservatoryHistoryErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	r := httptest.NewRequest(http.MethodGet, "http://site/ops/observatory", nil)
	if _, _, err := fetchObservatoryHistory(r); err == nil {
		t.Fatal("a 500 from the feeder must surface as an error, never an empty success")
	}
}

func TestFetchObservatoryTracePreservesOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/tasks/") || !strings.HasSuffix(r.URL.Path, "/events") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"task_id":"t-1","events":[{"id":"e1","type":"work.task.created","source":"actor-a","timestamp":"2026-06-13T00:00:01Z"},{"id":"e2","type":"work.task.completed","source":"actor-b","timestamp":"2026-06-13T00:05:00Z"}]}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	r := httptest.NewRequest(http.MethodGet, "http://site/ops/observatory?task=t-1", nil)
	steps, _, err := fetchObservatoryTrace(r, "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 2 || steps[0].Label != "work.task.created" || steps[1].Label != "work.task.completed" {
		t.Errorf("steps must preserve feeder order, got %+v", steps)
	}
	if steps[0].Sub != "actor-a" {
		t.Errorf("step source must carry through, got %q", steps[0].Sub)
	}
}
