package graph

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchOpsWorkSummarizesWorkAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/tasks":
			w.Write([]byte(`{"tasks":[
			{"id":"task-1","title":"Blocked task","description":"Needs dependency","priority":"high","status":"open","assignee":"","blocked":true,"artifact_count":0,"waived":false,"ready":false,"missing_gates":["definition_of_done"]},
			{"id":"task-2","title":"Active task","description":"Being built","priority":"medium","status":"in_progress","assignee":"agent-1","blocked":false,"artifact_count":2,"waived":false,"ready":true,"missing_gates":[]},
			{"id":"task-3","title":"Completed task","description":"Done","priority":"low","status":"completed","assignee":"agent-2","blocked":false,"artifact_count":1,"waived":true,"ready":true,"missing_gates":[]}
		]}`))
		case "/phase-gates":
			w.Write([]byte(`{"gates":[{"id":"gate-1","phase":"design","title":"Design approval","criteria":["brief"],"status":"approved","summary":"accepted","updated_at":"2026-05-01T10:00:00Z"}]}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	t.Setenv("WORK_API_BASE_URL", srv.URL)
	t.Setenv("WORK_API_KEY", "test-key")
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/work", nil)

	got := fetchOpsWork(req)

	if got.Error != "" {
		t.Fatalf("Error = %q, want empty", got.Error)
	}
	if got.Total != 3 || got.Open != 2 || got.Active != 1 || got.Blocked != 1 || got.Completed != 1 {
		t.Fatalf("counts = total:%d open:%d active:%d blocked:%d completed:%d", got.Total, got.Open, got.Active, got.Blocked, got.Completed)
	}
	if got.HighPriority != 1 || got.Unassigned != 1 || got.EvidenceCount != 3 || got.WaivedCount != 1 {
		t.Fatalf("summary = high:%d unassigned:%d evidence:%d waived:%d", got.HighPriority, got.Unassigned, got.EvidenceCount, got.WaivedCount)
	}
	if got.Ready != 2 {
		t.Fatalf("Ready = %d, want 2", got.Ready)
	}
	if len(got.BlockedTasks) != 1 || got.BlockedTasks[0].ID != "task-1" {
		t.Fatalf("BlockedTasks = %#v, want task-1", got.BlockedTasks)
	}
	if len(got.PhaseGates) != 1 || got.PhaseGates[0].Status != "approved" {
		t.Fatalf("PhaseGates = %#v, want approved gate", got.PhaseGates)
	}
}

func TestHandleOpsWorkDoesNotLinkLegacyDashboard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/tasks":
			w.Write([]byte(`{"tasks":[]}`))
		case "/phase-gates":
			w.Write([]byte(`{"gates":[{"id":"gate-1","phase":"validation","title":"Validation gate","status":"pending"}]}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/work?profile=transpara", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/work: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "Legacy dashboard") || strings.Contains(body, `href="`+srv.URL+`/"`) {
		t.Fatal("GET /ops/work still links to the legacy Work browser dashboard")
	}
	if !strings.Contains(body, "Work summary") {
		t.Fatal("GET /ops/work: body does not contain native Work summary")
	}
	if !strings.Contains(body, "Phase gates") || !strings.Contains(body, "Validation gate") {
		t.Fatal("GET /ops/work: body does not contain phase gate summary")
	}
}

func TestFetchOpsTelemetryIncludesPipelineReport(t *testing.T) {
	var overviewAuth, reportAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/telemetry/overview":
			overviewAuth = r.Header.Get("Authorization")
			w.Write([]byte(`{
				"timestamp":"2026-04-29T15:00:00Z",
				"actors":[{"id":"actor-1","display_name":"Agent","actor_type":"agent","status":"active"}],
				"agents":[{"role":"builder","state":"processing","model":"claude","last_message":"building","errors":0}],
				"recent_events":[{"event_type":"pipeline.phase.completed","actor_role":"builder","summary":"emission done","at":"2026-04-29T15:00:00Z"}],
				"phases":[{"phase":4,"label":"emission","status":"in_progress"}]
			}`))
		case "/telemetry/pipeline/report":
			reportAuth = r.Header.Get("Authorization")
			w.Write([]byte(`{"computed_at":"2026-04-29T15:00:01Z","report":{
				"cycle_id":"pipeline-test",
				"status":"complete",
				"current_stage":"audit",
				"current_phase":"observer",
				"last_outcome":"audit.done",
				"last_summary":"audit completed; 4 tasks remain open",
				"updated_at":"2026-04-29T15:00:00Z",
				"duration_secs":326.6,
				"total_cost_usd":1.3908,
				"total_tokens":28781,
				"open_board_items":4,
				"revise_count":0,
				"emission_complete":true,
				"human_status":"Cycle pipeline-test is complete.",
				"phases":[{"phase":"builder","workflow_stage":"emission","outcome":"task.done","duration_secs":12,"cost_usd":0.5,"summary":"emission completed"}]
			}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	t.Setenv("WORK_API_BASE_URL", srv.URL)
	t.Setenv("WORK_API_KEY", "test-key")
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/telemetry", nil)

	got := fetchOpsTelemetry(req)

	if got.Error != "" || got.PipelineError != "" {
		t.Fatalf("errors = overview:%q pipeline:%q", got.Error, got.PipelineError)
	}
	if overviewAuth != "Bearer test-key" || reportAuth != "Bearer test-key" {
		t.Fatalf("auth headers = overview:%q report:%q", overviewAuth, reportAuth)
	}
	if got.Pipeline == nil {
		t.Fatal("Pipeline = nil, want report")
	}
	if got.Pipeline.CycleID != "pipeline-test" || got.Pipeline.HumanStatus == "" {
		t.Fatalf("pipeline report = %#v", got.Pipeline)
	}
	if len(got.Pipeline.Phases) != 1 || got.Pipeline.Phases[0].WorkflowStage != "emission" {
		t.Fatalf("pipeline phases = %#v", got.Pipeline.Phases)
	}
}

func TestHandleOpsTelemetryDoesNotLinkLegacyDashboard(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/telemetry/overview":
			w.Write([]byte(`{"timestamp":"2026-04-29T15:00:00Z","actors":[],"agents":[],"recent_events":[],"phases":[]}`))
		case "/telemetry/pipeline/report":
			w.Write([]byte(`{"computed_at":"2026-04-29T15:00:01Z","report":null}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/telemetry?profile=transpara", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/telemetry: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "Legacy dashboard") || strings.Contains(body, `href="`+srv.URL+`/telemetry/"`) {
		t.Fatal("GET /ops/telemetry still links to the legacy Work telemetry dashboard")
	}
	if !strings.Contains(body, "Telemetry summary") {
		t.Fatal("GET /ops/telemetry: body does not contain native telemetry summary")
	}
}

func TestHandleOpsHiveRendersNativeSummary(t *testing.T) {
	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/hive?profile=transpara", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/hive: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Hive summary") {
		t.Fatal("GET /ops/hive: body does not contain native Hive summary")
	}
	if strings.Contains(body, "<iframe") {
		t.Fatal("GET /ops/hive: body contains an iframe; operator route should be native")
	}
	if !strings.Contains(body, "Public live build") {
		t.Fatal("GET /ops/hive: body does not link to the public /hive live-build page")
	}
}

func TestFetchOpsHiveOperatorProjection(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hive/operator-projection" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"generated_at":"2026-05-09T12:00:00Z",
			"source":"eventgraph",
			"pending_approvals":[{"request_id":"req-1","requesting_actor":"actor-requester","action_name":"agent.retire","target":"actor-target","environment":"production","justification":"completed mandate","created_at":"2026-05-09T11:00:00Z"}],
			"authority_decisions":[{"decision_id":"decision-1","request_id":"req-2","approver_actor":"actor-approver","outcome":"approved","approved_action":"agent.revoke","approved_target":"actor-revoked","rationale":"valid evidence","created_at":"2026-05-09T10:00:00Z"}],
			"lifecycle":[{"actor_id":"actor-target","display_name":"builder","role":"builder","lifecycle_status":"retired","authority_scope":"hive:read","key_provenance":"generated","updated_at":"2026-05-09T09:00:00Z"}],
			"key_audit_traces":[{"event_id":"event-key","event_type":"agent.key.registered","actor_id":"actor-target","key_provenance":"generated","public_key":"abc","created_at":"2026-05-09T08:00:00Z"}]
		}`))
	}))
	defer srv.Close()

	t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)
	t.Setenv("HIVE_OPS_API_KEY", "ops-key")
	h, _, _ := testHandlers(t)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/hive", nil)

	got := h.fetchOpsHive(req)

	if auth != "Bearer ops-key" {
		t.Fatalf("Authorization = %q, want Bearer ops-key", auth)
	}
	if got.ProjectionError != "" {
		t.Fatalf("ProjectionError = %q, want empty", got.ProjectionError)
	}
	if got.ProjectionSource != "eventgraph" {
		t.Fatalf("ProjectionSource = %q, want eventgraph", got.ProjectionSource)
	}
	if len(got.PendingApprovals) != 1 || got.PendingApprovals[0].ActionName != "agent.retire" {
		t.Fatalf("PendingApprovals = %#v", got.PendingApprovals)
	}
	if len(got.AuthorityDecisions) != 1 || got.AuthorityDecisions[0].Outcome != "approved" {
		t.Fatalf("AuthorityDecisions = %#v", got.AuthorityDecisions)
	}
	if len(got.Lifecycle) != 1 || got.Lifecycle[0].LifecycleStatus != "retired" {
		t.Fatalf("Lifecycle = %#v", got.Lifecycle)
	}
	if len(got.KeyAuditTraces) != 1 || got.KeyAuditTraces[0].EventType != "agent.key.registered" {
		t.Fatalf("KeyAuditTraces = %#v", got.KeyAuditTraces)
	}
}

func TestHandleOpsHiveRendersReadOnlyAuthorityProjection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"generated_at":"2026-05-09T12:00:00Z",
			"source":"eventgraph",
			"pending_approvals":[{"request_id":"req-1","requesting_actor":"actor-requester","action_name":"agent.spawn.persistent","target":"builder","environment":"production","justification":"trial passed","created_at":"2026-05-09T11:00:00Z"}],
			"authority_decisions":[{"decision_id":"decision-1","request_id":"req-2","approver_actor":"actor-approver","outcome":"denied","approved_action":"agent.revoke","approved_target":"actor-revoked","rationale":"insufficient evidence","created_at":"2026-05-09T10:00:00Z"}],
			"lifecycle":[{"actor_id":"actor-builder","display_name":"builder","role":"builder","lifecycle_status":"active","authority_scope":"hive:read","key_provenance":"external","updated_at":"2026-05-09T09:00:00Z"}],
			"key_audit_traces":[{"event_id":"event-key","event_type":"agent.key.registered","actor_id":"actor-builder","key_provenance":"external","public_key":"abc","created_at":"2026-05-09T08:00:00Z"}]
		}`))
	}))
	defer srv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)

	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/hive?profile=transpara", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/hive: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"Authority projection", "Pending approvals", "Authority decisions", "Lifecycle state", "Key provenance", "agent.spawn.persistent", "builder"} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/hive: body does not contain %q", want)
		}
	}
	if strings.Contains(body, `method="post"`) ||
		strings.Contains(body, `action="/ops/hive"`) ||
		strings.Contains(body, `data-authority-action`) {
		t.Fatal("GET /ops/hive exposes authority mutation controls")
	}
}

func TestFetchOpsHiveProjectionFailureIsNonFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no projection", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)

	h, _, _ := testHandlers(t)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/hive", nil)

	got := h.fetchOpsHive(req)

	if got.Error != "" {
		t.Fatalf("Error = %q, want empty runtime summary error", got.Error)
	}
	if got.ProjectionError == "" {
		t.Fatal("ProjectionError is empty, want nonfatal projection error")
	}
}
