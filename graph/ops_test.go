package graph

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
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
	for _, want := range []string{`href="/ops/hive/intake"`, `href="/ops/hive/runs"`, `href="/ops/hive/agents"`, `href="/ops/hive/resources"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/hive: body does not contain child route link %q", want)
		}
	}
}

func TestHandleOpsHiveStaticChildRoutesRender(t *testing.T) {
	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	tests := []struct {
		path string
		want []string
	}{
		{
			path: "/ops/hive/intake?profile=transpara",
			want: []string{"Hive intake", "Source input", "Live interpretation", "checkout-redesign.md"},
		},
		{
			path: "/ops/hive/runs?profile=transpara",
			want: []string{"Hive runs", "Run tower", "run_static_001", "Build onboarding control surface"},
		},
		{
			path: "/ops/hive/agents?profile=transpara",
			want: []string{"Hive agents", "Agent topology", "guardian", "architect"},
		},
		{
			path: "/ops/hive/resources?profile=transpara",
			want: []string{"Hive resources", "Run budget", "Approval queue", "Read-only mode"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://site.test"+tt.path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("GET %s: status = %d, want 200; body: %s", tt.path, w.Code, w.Body.String())
			}
			body := w.Body.String()
			for _, want := range tt.want {
				if !strings.Contains(body, want) {
					t.Fatalf("GET %s: body does not contain %q", tt.path, want)
				}
			}
			for _, forbidden := range []string{"<iframe", `method="post"`, `action="/ops/hive`, "Hive operator projection source is not configured"} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("GET %s: body contains forbidden %q", tt.path, forbidden)
				}
			}
		})
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
			"key_audit_traces":[{"event_id":"event-key","event_type":"agent.key.registered","actor_id":"actor-target","key_provenance":"generated","public_key":"abc","created_at":"2026-05-09T08:00:00Z"}],
			"runtime_evidence":{
				"source":"eventgraph",
				"status":"completed",
				"last_run":{
					"started_event_id":"event-run-start",
					"conversation_id":"conv_runtime",
					"started_at":"2026-05-09T06:00:00Z",
					"seed_idea":"prove runtime evidence",
					"repo_path":"/repos/hive",
					"completed_event_id":"event-run-complete",
					"completed_at":"2026-05-09T06:02:00Z",
					"agent_count":1,
					"duration_ms":120000,
					"total_cost":0
				},
				"agent_events":{"scope":"events_since_latest_hive.run.started","spawned":1,"stopped":1,"observed_active":0},
				"last_queued_run_request":{
					"event_id":"event-request",
					"conversation_id":"conv_request",
					"run_id":"run_123",
					"title":"Queued run",
					"operator_id":"site-ops",
					"status":"queued",
					"target_repos":["transpara-ai/hive"],
					"authority_initial_level":"Required",
					"budget_max_iterations":0,
					"budget_max_cost_usd":0,
					"evidence_kind":"queued_request_not_runtime_start",
					"created_at":"2026-05-09T05:59:00Z"
				},
				"limitations":["factory.run.requested is queued launch intent, not runtime-start proof"]
			},
			"model_selection":{
				"source":"hive",
				"catalog_source":"embedded-defaults",
				"loaded_at":"2026-05-09T07:00:00Z",
				"reload_mode":"hot-reload",
				"hot_reload":true,
				"models":[{"id":"api-claude-sonnet-4-6","provider":"anthropic","auth_mode":"api-key","tier":"execution","capabilities":["reasoning"],"context_window":200000,"max_output_tokens":8192}],
				"assignments":[{"role":"guardian","model":"api-claude-sonnet-4-6","provider":"anthropic","auth_mode":"api-key","preferred_tier":"execution","required_capabilities":["reasoning"],"selection_strategy":"balanced"}]
			}
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
	if got.RuntimeEvidence.Status != "completed" || got.RuntimeEvidence.LastRun == nil || got.RuntimeEvidence.LastRun.TotalCost == nil || *got.RuntimeEvidence.LastRun.TotalCost != 0 {
		t.Fatalf("RuntimeEvidence = %#v", got.RuntimeEvidence)
	}
	if got.RuntimeEvidence.LastQueuedRunRequest == nil || got.RuntimeEvidence.LastQueuedRunRequest.BudgetMaxCostUSD == nil || *got.RuntimeEvidence.LastQueuedRunRequest.BudgetMaxCostUSD != 0 {
		t.Fatalf("RuntimeEvidence queued request = %#v", got.RuntimeEvidence.LastQueuedRunRequest)
	}
	if got.ModelSelection.ReloadMode != "hot-reload" || !got.ModelSelection.HotReload {
		t.Fatalf("ModelSelection reload metadata = %#v", got.ModelSelection)
	}
	if len(got.ModelSelection.Assignments) != 1 || got.ModelSelection.Assignments[0].AuthMode != "api-key" {
		t.Fatalf("ModelSelection assignments = %#v", got.ModelSelection.Assignments)
	}
}

func TestFetchOpsHiveOperatorProjectionToleratesMissingModelSelection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"generated_at":"2026-05-09T12:00:00Z",
			"source":"eventgraph",
			"pending_approvals":[],
			"authority_decisions":[],
			"lifecycle":[],
			"key_audit_traces":[]
		}`))
	}))
	defer srv.Close()

	t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)
	h, _, _ := testHandlers(t)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/hive", nil)

	got := h.fetchOpsHive(req)

	if got.ProjectionError != "" {
		t.Fatalf("ProjectionError = %q, want empty", got.ProjectionError)
	}
	if got.ModelSelection.Source != "" || len(got.ModelSelection.Assignments) != 0 || len(got.ModelSelection.Models) != 0 {
		t.Fatalf("ModelSelection = %#v, want zero value for older Hive projection", got.ModelSelection)
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
			"key_audit_traces":[{"event_id":"event-key","event_type":"agent.key.registered","actor_id":"actor-builder","key_provenance":"external","public_key":"abc","created_at":"2026-05-09T08:00:00Z"}],
			"runtime_evidence":{
				"source":"eventgraph",
				"status":"running",
				"last_run":{"started_event_id":"event-run-start","conversation_id":"conv_runtime","started_at":"2026-05-09T06:00:00Z","seed_idea":"render runtime evidence","repo_path":"/repos/hive"},
				"agent_events":{
					"scope":"events_since_latest_hive.run.started",
					"spawned":2,
					"stopped":0,
					"observed_active":1,
					"active_agents":[{"name":"builder","role":"implementer","model":"claude-sonnet-4-6","actor_id":"actor-runtime-evidence-only","spawned_event_id":"event-spawn","spawned_at":"2026-05-09T06:01:00Z"}]
				},
				"last_queued_run_request":{"event_id":"event-request","conversation_id":"conv_request","run_id":"run_123","title":"Queued run","status":"queued","target_repos":["transpara-ai/hive"],"authority_initial_level":"Required","budget_max_iterations":0,"budget_max_cost_usd":0,"evidence_kind":"queued_request_not_runtime_start","created_at":"2026-05-09T05:59:00Z"},
				"limitations":["factory.run.requested is queued launch intent, not runtime-start proof","hive.run.started and hive.run.completed prove Hive runtime event emission, not production deployment"]
			},
			"model_selection":{
				"source":"hive",
				"catalog_source":"embedded-defaults",
				"loaded_at":"2026-05-09T07:00:00Z",
				"reload_mode":"startup-static",
				"hot_reload":false,
				"models":[{"id":"claude-sonnet-4-6","provider":"claude-cli","auth_mode":"subscription","tier":"execution","capabilities":["reasoning","coding"],"context_window":200000,"max_output_tokens":8192}],
				"assignments":[{"role":"guardian","model":"claude-sonnet-4-6","provider":"claude-cli","auth_mode":"subscription","preferred_tier":"execution","required_capabilities":["reasoning"],"selection_strategy":"balanced"}]
			}
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
	for _, want := range []string{"Authority projection", "Runtime evidence", "Queued launch intent", "queued_request_not_runtime_start", "runtime-start proof", "0 iter / $0.00", "actor-runtime-evidence-only", "Pending approvals", "Authority decisions", "Lifecycle state", "Key provenance", "Model selection", "startup-static", "guardian", "subscription", "agent.spawn.persistent", "builder", `action="/ops/hive/model-policy"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/hive: body does not contain %q", want)
		}
	}
	if strings.Contains(body, `data-authority-action`) {
		t.Fatal("GET /ops/hive exposes authority mutation controls")
	}
}

func TestOpsHiveModelPolicySubmitForwardsToHive(t *testing.T) {
	var gotPath, gotAuth string
	var gotPayload map[string]any
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode forwarded payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"recorded"}`))
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)
	t.Setenv("HIVE_OPS_API_KEY", "secret")

	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)
	form := url.Values{}
	form.Set("role", "guardian")
	form.Set("model", "api-sonnet")
	form.Set("auth_mode", "api-key")
	form.Set("profile", "judgment")
	form.Set("preferred_tier", "execution")
	form.Add("required_capability", "reasoning")
	form.Set("max_cost_per_call_usd", "2.75")
	form.Set("operator_id", "site-ops")
	form.Set("reason", "operator selected metered guardian")
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/model-policy", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d body=%s", w.Code, http.StatusSeeOther, w.Body.String())
	}
	if gotPath != "/api/hive/model-selection/role-policy" {
		t.Fatalf("forwarded path = %q", gotPath)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("forwarded auth = %q", gotAuth)
	}
	for key, want := range map[string]string{
		"role":           "guardian",
		"model":          "api-sonnet",
		"auth_mode":      "api-key",
		"profile":        "judgment",
		"preferred_tier": "execution",
		"operator_id":    "site-ops",
		"reason":         "operator selected metered guardian",
	} {
		if gotPayload[key] != want {
			t.Fatalf("payload[%s] = %#v, want %q (payload=%#v)", key, gotPayload[key], want, gotPayload)
		}
	}
	caps, ok := gotPayload["required_capabilities"].([]any)
	if !ok || len(caps) != 1 || caps[0] != "reasoning" {
		t.Fatalf("required_capabilities = %#v, want [reasoning]", gotPayload["required_capabilities"])
	}
	if gotPayload["max_cost_per_call_usd"] != 2.75 {
		t.Fatalf("max_cost_per_call_usd = %#v, want 2.75", gotPayload["max_cost_per_call_usd"])
	}
}

func TestHandleOpsHiveRendersModelSelectionEmptyStateForOlderProjection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"generated_at":"2026-05-09T12:00:00Z",
			"source":"eventgraph",
			"pending_approvals":[],
			"authority_decisions":[],
			"lifecycle":[],
			"key_audit_traces":[]
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
	if !strings.Contains(body, "No model-selection projection returned.") {
		t.Fatalf("GET /ops/hive: body does not contain model-selection empty state")
	}
	if !strings.Contains(body, "No runtime evidence projection returned.") {
		t.Fatalf("GET /ops/hive: body does not contain runtime-evidence empty state")
	}
	if strings.Contains(body, `method="post"`) ||
		strings.Contains(body, `action="/ops/hive"`) ||
		strings.Contains(body, `data-authority-action`) {
		t.Fatal("GET /ops/hive older projection exposes mutation controls")
	}
}

func TestHandleOpsHiveRendersRuntimeEvidenceBoundariesWithoutRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"generated_at":"2026-05-09T12:00:00Z",
			"source":"eventgraph",
			"pending_approvals":[],
			"authority_decisions":[],
			"lifecycle":[],
			"key_audit_traces":[],
			"runtime_evidence":{
				"source":"eventgraph",
				"status":"not_observed",
				"agent_events":{"scope":"none","spawned":0,"stopped":0,"observed_active":0},
				"limitations":["factory.run.requested is queued launch intent, not runtime-start proof"]
			}
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
	for _, want := range []string{"Runtime evidence", "not_observed", "No hive.run.started event observed.", "Evidence boundaries", "queued launch intent"} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/hive: body does not contain %q", want)
		}
	}
	if strings.Contains(body, "No runtime evidence projection returned.") {
		t.Fatal("GET /ops/hive hid current runtime-evidence boundaries behind the older-projection empty state")
	}
	if strings.Contains(body, `method="post"`) ||
		strings.Contains(body, `action="/ops/hive"`) ||
		strings.Contains(body, `data-authority-action`) {
		t.Fatal("GET /ops/hive runtime-evidence boundaries expose mutation controls")
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

func TestFetchOpsHiveProjectionTimeoutIsNonFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)

	oldClient := hiveOpsProjectionClient
	hiveOpsProjectionClient = &http.Client{Timeout: 10 * time.Millisecond}
	t.Cleanup(func() { hiveOpsProjectionClient = oldClient })

	h, _, _ := testHandlers(t)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/hive", nil)

	got := h.fetchOpsHive(req)

	if got.Error != "" {
		t.Fatalf("Error = %q, want empty runtime summary error", got.Error)
	}
	if got.ProjectionError == "" {
		t.Fatal("ProjectionError is empty, want nonfatal timeout error")
	}
}

func TestFetchOpsEvidenceProjection(t *testing.T) {
	var gotFactoryOrder, gotReleaseCandidate string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFactoryOrder = r.URL.Query().Get("factory_order_id")
		gotReleaseCandidate = r.URL.Query().Get("release_candidate_id")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(opsEvidenceFixtureJSON()))
	}))
	defer srv.Close()

	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", srv.URL+"/projection?existing=1")
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/evidence?factory_order_id=fo_001&release_candidate_id=rc_001", nil)

	got := fetchOpsEvidence(req)

	if got.ProjectionError != "" {
		t.Fatalf("ProjectionError = %q, want empty", got.ProjectionError)
	}
	if gotFactoryOrder != "fo_001" || gotReleaseCandidate != "rc_001" {
		t.Fatalf("forwarded query = factory_order_id:%q release_candidate_id:%q", gotFactoryOrder, gotReleaseCandidate)
	}
	if got.Source != "eventgraph-work-projection" {
		t.Fatalf("Source = %q, want fixture source", got.Source)
	}
	if got.FactoryOrder == nil || got.FactoryOrder.ID != "fo_001" {
		t.Fatalf("FactoryOrder = %#v, want fo_001", got.FactoryOrder)
	}
	if got.ReleaseCandidate == nil || got.ReleaseCandidate.ID != "rc_001" {
		t.Fatalf("ReleaseCandidate = %#v, want rc_001", got.ReleaseCandidate)
	}
	if len(got.Timeline) != 2 || got.Timeline[1].NodeID != "gate_001" {
		t.Fatalf("Timeline = %#v, want gate_001 event", got.Timeline)
	}
	if len(got.GateEvidence) != 1 || got.GateEvidence[0].GateName != "unit_tests" {
		t.Fatalf("GateEvidence = %#v, want unit_tests", got.GateEvidence)
	}
	if len(got.ReleaseEvidence) != 1 || len(got.ReleaseEvidence[0].RuntimeRefs) != 1 {
		t.Fatalf("ReleaseEvidence = %#v, want runtime refs", got.ReleaseEvidence)
	}
	if len(got.FailuresRepairs) != 1 || got.FailuresRepairs[0].RepairID != "rep_001" {
		t.Fatalf("FailuresRepairs = %#v, want rep_001", got.FailuresRepairs)
	}
	if got.AuditReport == nil || got.AuditReport.ID != "aud_001" {
		t.Fatalf("AuditReport = %#v, want aud_001", got.AuditReport)
	}
	if len(got.MissingProvenance) != 1 || got.MissingProvenance[0].PathName == "" {
		t.Fatalf("MissingProvenance = %#v, want one path", got.MissingProvenance)
	}
	if got.ProofOfWorkPacket == nil || got.ProofOfWorkPacket.ID != "pow_001" {
		t.Fatalf("ProofOfWorkPacket = %#v, want pow_001", got.ProofOfWorkPacket)
	}
	if got.ProofOfWorkPacket.WorkItem == nil || got.ProofOfWorkPacket.WorkItem.EventGraphRefs[0] != "eg://task/tsk_001" {
		t.Fatalf("ProofOfWorkPacket.WorkItem = %#v, want task EventGraph ref", got.ProofOfWorkPacket.WorkItem)
	}
	if len(got.ProofOfWorkPacket.SecurityScanResults) != 1 || got.ProofOfWorkPacket.SecurityScanResults[0].Status != "pass" {
		t.Fatalf("ProofOfWorkPacket.SecurityScanResults = %#v, want passing scan", got.ProofOfWorkPacket.SecurityScanResults)
	}
}

func TestHandleOpsEvidenceRendersReadOnlyProjection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(opsEvidenceFixtureJSON()))
	}))
	defer srv.Close()
	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", srv.URL)

	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/evidence?profile=transpara&view=forensic", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/evidence: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"Evidence projection",
		"FactoryOrder timeline",
		"Gate evidence",
		"Release evidence",
		"Failures and repairs",
		"Audit evidence",
		"Missing provenance",
		"Proof-of-work packet",
		"Work item",
		"Runtime invocation",
		"Changed files",
		"Tests run",
		"CI status",
		"Review feedback",
		"Security scan results",
		"Screenshots and walkthrough artifacts",
		"Known failures",
		"Operator decision",
		"fo_001",
		"pow_001",
		"unit_tests",
		"go test -count=1 ./graph",
		"CodeQL scan",
		"ops evidence walkthrough",
		"traceability_gap",
		"missing RuntimeResult rr_001",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/evidence: body does not contain %q", want)
		}
	}
	start := strings.Index(body, "Evidence projection")
	if start < 0 {
		t.Fatal("GET /ops/evidence: could not locate evidence surface")
	}
	evidenceSurface := body[start:]
	if end := strings.Index(evidenceSurface, "</main>"); end >= 0 {
		evidenceSurface = evidenceSurface[:end]
	}
	for _, forbidden := range []string{
		"<form",
		"<button",
		`method="post"`,
		`action="/ops/evidence"`,
		`formaction="/ops/evidence"`,
		`hx-post="/ops/evidence"`,
		`hx-put="/ops/evidence"`,
		`hx-patch="/ops/evidence"`,
		`hx-delete="/ops/evidence"`,
		`data-evidence-action`,
		"Certify release",
		"Reject release",
		"Approve authority",
		"Repair work",
		"Waive failure",
		"Retry work",
		"Unblock work",
		"Execute work",
	} {
		if strings.Contains(evidenceSurface, forbidden) {
			t.Fatalf("GET /ops/evidence evidence surface contains mutation control marker %q", forbidden)
		}
	}
}

func TestHandleOpsEvidenceUnconfiguredRendersEmptyState(t *testing.T) {
	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", "")
	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/evidence?profile=transpara&view=forensic", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/evidence: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Evidence projection") || !strings.Contains(body, "projection URL is not configured") {
		t.Fatalf("GET /ops/evidence: body does not contain unconfigured empty state; body: %s", body)
	}
}

func TestHandleOpsEvidenceMissingProofOfWorkPacketIsNonFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(opsEvidenceFixtureWithoutProofOfWorkPacketJSON()))
	}))
	defer srv.Close()
	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", srv.URL)

	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/evidence?profile=transpara&view=forensic", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/evidence: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"Proof-of-work packet", "No proof-of-work packet returned.", "FactoryOrder timeline"} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/evidence missing packet body does not contain %q", want)
		}
	}
	evidenceSurface := body[strings.Index(body, "Evidence projection"):]
	if end := strings.Index(evidenceSurface, "</main>"); end >= 0 {
		evidenceSurface = evidenceSurface[:end]
	}
	if strings.Contains(evidenceSurface, "<form") || strings.Contains(evidenceSurface, "<button") {
		t.Fatal("GET /ops/evidence missing packet state contains mutation controls")
	}
}

func TestFetchOpsEvidenceProjectionFailureIsNonFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no projection", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", srv.URL)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/evidence", nil)
	got := fetchOpsEvidence(req)

	if got.ProjectionError == "" {
		t.Fatal("ProjectionError is empty, want nonfatal projection error")
	}
}

func TestFetchOpsEvidenceProjectionInvalidProofOfWorkPacketIsNonFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"generated_at":"2026-05-14T19:00:00Z","source":"eventgraph-work-projection","proof_of_work_packet":"not an object"}`))
	}))
	defer srv.Close()
	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", srv.URL)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/evidence", nil)
	got := fetchOpsEvidence(req)

	if got.ProjectionError == "" {
		t.Fatal("ProjectionError is empty, want nonfatal proof_of_work_packet decode error")
	}
	if got.ProofOfWorkPacket != nil {
		t.Fatalf("ProofOfWorkPacket = %#v, want nil after type-invalid packet", got.ProofOfWorkPacket)
	}
}

func TestFetchOpsEvidenceProjectionTimeoutIsNonFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", srv.URL)

	oldClient := evidenceOpsProjectionClient
	evidenceOpsProjectionClient = &http.Client{Timeout: 10 * time.Millisecond}
	t.Cleanup(func() { evidenceOpsProjectionClient = oldClient })

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/evidence", nil)
	got := fetchOpsEvidence(req)

	if got.ProjectionError == "" {
		t.Fatal("ProjectionError is empty, want nonfatal timeout error")
	}
}

func TestFetchOpsEvidenceProjectionBadJSONIsNonFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"generated_at":`))
	}))
	defer srv.Close()
	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", srv.URL)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/evidence", nil)
	got := fetchOpsEvidence(req)

	if got.ProjectionError == "" {
		t.Fatal("ProjectionError is empty, want nonfatal JSON error")
	}
}

func TestHandleOpsDecisionRendersNonExecutingBoundary(t *testing.T) {
	h := testOpsDecisionHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/decision?profile=transpara&action=approve&target_type=pull_request&repo=transpara-ai/site&target_ref=pr://transpara-ai/site/75&reason=reviewed+packet+evidence", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/decision: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"Gate E decision surface",
		"Site is non-executing",
		"Site does not directly execute work, produce protected side effects, rely on policy-adapter decisions, run runner/worktree protected execution, or grant production autonomy.",
		"transpara-ai/docs#75 / a50190ce470e8686a561e0e0b6e62ef0c5f5bb13",
		"Approve",
		"Deny",
		"Request more evidence",
		"correlation_id",
		"trace_id",
		"requested_action",
		"decision_reason",
		"target_ref",
		"pr://transpara-ai/site/75",
		"governance_scope",
		"non-executing Site projection; effect = none",
		"accepted_for_review",
		"effect = none",
		"direct_execution_requested",
		"protected_side_effect_requested",
		"policy_adapter_reliance_requested",
		"runner_worktree_execution_requested",
		"production_autonomy_requested",
		"ExecutionReceipt production path",
		"execution_receipt_requested",
		"policy-bundle reliance",
		"policy_bundle_requested",
		"R-001",
		"R-002",
		"R-003",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/decision: body does not contain %q", want)
		}
	}
	assertOpsDecisionNoMutationControls(t, body)
}

func TestHandleOpsDecisionActionsStayInsideGovernedBoundary(t *testing.T) {
	h := testOpsDecisionHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	for _, action := range []string{"approve", "deny", "request-more-evidence", "request_more_evidence"} {
		t.Run(action, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/decision?action="+url.QueryEscape(action)+"&target_type=issue&repo=transpara-ai/site&target_ref=issue://transpara-ai/site/123&reason=operator+review", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("GET /ops/decision: status = %d, want 200; body: %s", w.Code, w.Body.String())
			}
			body := w.Body.String()
			if !strings.Contains(body, "accepted_for_review") || !strings.Contains(body, "effect = none") {
				t.Fatalf("GET /ops/decision: action %q left governed boundary; body: %s", action, body)
			}
			if action == "request_more_evidence" {
				if !strings.Contains(body, "request-more-evidence") {
					t.Fatalf("GET /ops/decision: underscore action did not normalize to request-more-evidence; body: %s", body)
				}
			} else if !strings.Contains(body, action) {
				t.Fatalf("GET /ops/decision: body does not contain action %q", action)
			}
			assertOpsDecisionNoMutationControls(t, body)
		})
	}
}

func TestHandleOpsDecisionFailsClosedForMissingMalformedAndForbiddenScope(t *testing.T) {
	h := testOpsDecisionHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	cases := []struct {
		name string
		url  string
		want string
	}{
		{"missing inputs", "http://site.test/ops/decision", "missing requested_action"},
		{"invalid action", "http://site.test/ops/decision?action=merge&target_ref=pr://transpara-ai/site/75&reason=operator+review", "requested_action is outside the Gate E decision boundary"},
		{"direct execution", "http://site.test/ops/decision?action=approve&target_ref=pr://transpara-ai/site/75&reason=operator+review&direct_execution=true", "direct execution is forbidden for this Site slice"},
		{"protected side effect", "http://site.test/ops/decision?action=approve&target_ref=pr://transpara-ai/site/75&reason=operator+review&protected_side_effect=true", "protected side effects are forbidden for this Site slice"},
		{"policy adapter", "http://site.test/ops/decision?action=approve&target_ref=pr://transpara-ai/site/75&reason=operator+review&policy_adapter_reliance=true", "policy-adapter reliance is forbidden for this Site slice"},
		{"runner worktree", "http://site.test/ops/decision?action=approve&target_ref=pr://transpara-ai/site/75&reason=operator+review&runner_worktree_execution=true", "runner/worktree protected execution is forbidden for this Site slice"},
		{"production autonomy", "http://site.test/ops/decision?action=approve&target_ref=pr://transpara-ai/site/75&reason=operator+review&production_autonomy=true", "production autonomy is forbidden for this Site slice"},
		{"execution receipt", "http://site.test/ops/decision?action=approve&target_ref=pr://transpara-ai/site/75&reason=operator+review&execution_receipt=true", "ExecutionReceipt production behavior is forbidden for this Site slice"},
		{"policy bundle", "http://site.test/ops/decision?action=approve&target_ref=pr://transpara-ai/site/75&reason=operator+review&policy_bundle=true", "policy-bundle reliance is forbidden for this Site slice"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("GET /ops/decision: status = %d, want 200; body: %s", w.Code, w.Body.String())
			}
			body := w.Body.String()
			for _, want := range []string{"blocked", "effect = none", "blocked_reason", "Request failed closed inside Site. No downstream action was executed.", tc.want} {
				if !strings.Contains(body, want) {
					t.Fatalf("GET /ops/decision: body does not contain %q; body: %s", want, body)
				}
			}
			assertOpsDecisionNoMutationControls(t, body)
		})
	}
}

func TestOpsDecisionLoadsRealPendingRequest(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"source": "eventgraph",
			"pending_approvals": []map[string]any{{
				"event_id": "ev1", "request_id": "req-civic-roles", "requesting_actor": "guardian",
				"action_name": "pull_request.create", "target": "transpara-ai/docs codex/civic-roles",
				"justification": "Draft PR for civic-roles.md", "created_at": "2026-06-05T12:00:00Z",
			}},
		})
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := testOpsDecisionHandlers()
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/decision?request_id=req-civic-roles", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()
	for _, want := range []string{"pull_request.create", "codex/civic-roles", "Draft PR for civic-roles.md"} {
		if !strings.Contains(body, want) {
			t.Fatalf("decision page missing %q", want)
		}
	}
	assertOpsDecisionNoMutationControls(t, body)
}

// TestOpsDecisionRejectsNonDraftPRRequest verifies the Gate-E decision surface
// governs ONLY pull_request.create (Codex P1-a). A pending request for any other
// protected action must render as blocked (effect none) and must NOT be
// mislabeled as an approvable pull_request — previously
// buildOpsDecisionDataFromProjection hardcoded TargetType:"pull_request" for any
// pending action.
func TestOpsDecisionRejectsNonDraftPRRequest(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"source": "eventgraph",
			"pending_approvals": []map[string]any{{
				"event_id": "ev1", "request_id": "req-spawn", "requesting_actor": "guardian",
				"action_name": "agent.spawn.persistent", "target": "builder",
				"justification": "persistent identity trial", "created_at": "2026-06-05T12:00:00Z",
			}},
		})
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/decision?request_id=req-spawn", nil)
	data := buildOpsDecisionDataFromProjection(req, "req-spawn")

	if data.Status != "blocked" {
		t.Fatalf("Status = %q, want blocked (a non-draft-PR action must not be approvable here)", data.Status)
	}
	if data.Effect != "none" {
		t.Fatalf("Effect = %q, want none", data.Effect)
	}
	if data.TargetType == "pull_request" {
		t.Fatalf("TargetType = %q: a non-pull_request.create action must not be mislabeled as a pull_request", data.TargetType)
	}
	if len(data.BlockedReasons) == 0 {
		t.Fatal("BlockedReasons is empty, want a reason explaining the action is out of scope")
	}
}

func TestOpsDecisionSubmitForwardsToHive(t *testing.T) {
	// Posts the REAL UI wire value (opsDecisionApprove = "approve") and asserts
	// that Site normalises it to the hive-canonical past-tense form ("approved")
	// before forwarding. Also covers the deny subtest.
	//
	// The re-render step calls /api/hive/operator-projection (GET); we track the
	// governance POST separately.
	newHiveSrv := func(t *testing.T) (srv *httptest.Server, path, auth, body *string) {
		t.Helper()
		var p, a, b string
		path, auth, body = &p, &a, &b
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				*path, *auth = r.URL.Path, r.Header.Get("Authorization")
				bs, _ := io.ReadAll(r.Body)
				*body = string(bs)
			}
			w.WriteHeader(http.StatusOK)
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"source":"test","pending_approvals":[]}`))
			}
		}))
		return srv, path, auth, body
	}

	t.Run("approve wire value normalised to approved", func(t *testing.T) {
		srv, gotPath, gotAuth, gotBody := newHiveSrv(t)
		defer srv.Close()
		t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)
		t.Setenv("HIVE_OPS_API_KEY", "secret")

		h := testOpsDecisionHandlers()
		mux := http.NewServeMux()
		h.Register(mux)
		// POST the real UI wire value ("approve"), not "approved".
		form := strings.NewReader("request_id=req-civic-roles&decision=" + opsDecisionApprove + "&reason=reviewed")
		req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/decision", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if *gotPath != "/api/hive/operator-decision" {
			t.Fatalf("forwarded path = %q", *gotPath)
		}
		if *gotAuth != "Bearer secret" {
			t.Fatalf("forwarded auth = %q", *gotAuth)
		}
		// Site must normalise "approve" → "approved" in the forwarded body.
		if !strings.Contains(*gotBody, "req-civic-roles") || !strings.Contains(*gotBody, `"approved"`) {
			t.Fatalf("forwarded body = %q; want req-civic-roles + \"approved\"", *gotBody)
		}
		if w.Code >= 400 {
			t.Fatalf("site returned %d", w.Code)
		}
	})

	t.Run("deny wire value normalised to denied", func(t *testing.T) {
		srv, gotPath, _, gotBody := newHiveSrv(t)
		defer srv.Close()
		t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)
		t.Setenv("HIVE_OPS_API_KEY", "secret")

		h := testOpsDecisionHandlers()
		mux := http.NewServeMux()
		h.Register(mux)
		// POST the real UI wire value ("deny"), not "denied".
		form := strings.NewReader("request_id=req-civic-roles&decision=" + opsDecisionDeny + "&reason=insufficient+evidence")
		req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/decision", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if *gotPath != "/api/hive/operator-decision" {
			t.Fatalf("forwarded path = %q", *gotPath)
		}
		// Site must normalise "deny" → "denied" in the forwarded body.
		if !strings.Contains(*gotBody, "req-civic-roles") || !strings.Contains(*gotBody, `"denied"`) {
			t.Fatalf("forwarded body = %q; want req-civic-roles + \"denied\"", *gotBody)
		}
		if w.Code >= 400 {
			t.Fatalf("site returned %d", w.Code)
		}
	})
}

// TestOpsDecisionButtonConstantsMatchHandler is a round-trip guard: it asserts
// that opsDecisionActions()[0].WireValue (the approve button's emitted value)
// equals opsDecisionApprove — the constant the handler switch also uses.
// Any future vocabulary drift between the template data and the handler is caught here.
func TestOpsDecisionButtonConstantsMatchHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/decision", nil)
	actions := opsDecisionActions(req)
	if len(actions) < 3 {
		t.Fatalf("opsDecisionActions returned %d actions, want 3", len(actions))
	}
	if actions[0].WireValue != opsDecisionApprove {
		t.Errorf("approve button WireValue = %q, want opsDecisionApprove = %q", actions[0].WireValue, opsDecisionApprove)
	}
	if actions[1].WireValue != opsDecisionDeny {
		t.Errorf("deny button WireValue = %q, want opsDecisionDeny = %q", actions[1].WireValue, opsDecisionDeny)
	}
	if actions[2].WireValue != opsDecisionMoreEvidence {
		t.Errorf("more-evidence button WireValue = %q, want opsDecisionMoreEvidence = %q", actions[2].WireValue, opsDecisionMoreEvidence)
	}
}

func TestOpsDecisionRequestMoreEvidenceIsHandledLocally(t *testing.T) {
	// Verify that submitting the real wire value (opsDecisionMoreEvidence =
	// "request-more-evidence") does NOT call hive's /api/hive/operator-decision
	// endpoint. The fake hive POST handler must not be hit; the page must show
	// the "no decision recorded / more evidence" note.
	hivePostHit := false
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			hivePostHit = true
			w.WriteHeader(http.StatusBadRequest) // hive would 400 this value anyway
			return
		}
		// GET /api/hive/operator-projection → minimal valid JSON for re-render.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"source":"test","pending_approvals":[]}`))
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)
	t.Setenv("HIVE_OPS_API_KEY", "secret")

	h := testOpsDecisionHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	// POST the real UI wire value for more-evidence (not a made-up string).
	form := strings.NewReader("request_id=req-civic-roles&decision=" + opsDecisionMoreEvidence + "&reason=need+more+evidence")
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/decision", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if hivePostHit {
		t.Fatal("hive POST /api/hive/operator-decision was called for request-more-evidence — it must not be")
	}
	if w.Code >= 400 {
		t.Fatalf("site returned %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"More evidence requested",
		"no decision recorded",
		"effect = none",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response body does not contain %q; body: %s", want, body)
		}
	}
}

func testOpsDecisionHandlers() *Handlers {
	return NewHandlers(nil, nil, nil)
}

// assertOpsDecisionNoMutationControls asserts the decision surface contains no EXECUTOR control
// and that every HTML form action or formaction in the region targets ONLY /ops/decision.
//
// Region extraction: start from <main> (to exclude the <head> meta-description tags that also
// contain the phrase "Gate E decision surface"), then find the heading text within that region.
//
// Denylist (executor/mutation phrases and GitHub API paths):
//
//	"Merge PR", "Deploy", "Access secret", "Activate capability", "Execute work",
//	"api.github.com", "/repos/"
//
// Allowlist (every action= / formaction= attribute value in the region):
//
//	MUST be exactly "/ops/decision". Any other target — external URL, absolute
//	http(s)://, or any other path — causes a test failure.
//
// External / absolute href and formaction values (http:// or https://) are also forbidden.
func assertOpsDecisionNoMutationControls(t *testing.T, body string) {
	t.Helper()
	// Narrow to the <main> body first so the <head> meta-description tags
	// (which also contain "Gate E decision surface") don't pollute the check.
	mainStart := strings.Index(body, "<main")
	if mainStart < 0 {
		mainStart = 0
	}
	mainBody := body[mainStart:]
	start := strings.Index(mainBody, "Gate E decision surface")
	if start < 0 {
		t.Fatal("/ops/decision: could not locate decision surface inside <main>")
	}
	decisionSurface := mainBody[start:]
	if end := strings.Index(decisionSurface, "</main>"); end >= 0 {
		decisionSurface = decisionSurface[:end]
	}

	// Denylist: executor control phrases and GitHub API paths.
	forbidden := []string{
		"Merge PR",
		"Deploy",
		"Access secret",
		"Activate capability",
		"Execute work",
		"api.github.com",
		"/repos/", // no GitHub mutation target on the console
	}
	for _, f := range forbidden {
		if strings.Contains(decisionSurface, f) {
			t.Fatalf("decision surface must not contain executor control %q", f)
		}
	}

	// Governance posture invariant.
	if !strings.Contains(decisionSurface, "effect = none") {
		t.Fatalf("decision surface must still declare effect = none")
	}

	// Allowlist: every action="..." and formaction="..." must be exactly /ops/decision.
	// No external URLs, no absolute http(s):// targets, no other paths permitted.
	for _, attrPrefix := range []string{`action="`, `formaction="`} {
		search := decisionSurface
		for {
			idx := strings.Index(search, attrPrefix)
			if idx < 0 {
				break
			}
			rest := search[idx+len(attrPrefix):]
			end := strings.IndexByte(rest, '"')
			if end < 0 {
				break
			}
			val := rest[:end]
			if val != "/ops/decision" {
				t.Fatalf("decision surface contains disallowed %s%s\" (only /ops/decision is permitted)", attrPrefix, val)
			}
			search = rest[end+1:]
		}
	}

	// No external / absolute href or formaction values (http:// or https://).
	for _, absPrefix := range []string{`href="http://`, `href="https://`, `formaction="http://`, `formaction="https://`} {
		if strings.Contains(decisionSurface, absPrefix) {
			t.Fatalf("decision surface contains absolute/external link or action starting with %q", absPrefix)
		}
	}
}

func opsEvidenceFixtureJSON() string {
	return `{
		"generated_at":"2026-05-14T19:00:00Z",
		"source":"eventgraph-work-projection",
		"factory_order":{"id":"fo_001","version":1,"status":"draft","source_intent_hash":"sha256:intent","source_intent_ref":"issue://55","risk_class":"medium","release_policy":"human_approval_required"},
		"release_candidate":{"id":"rc_001","status":"certified","factory_order_id":"fo_001","factory_runtime_version_id":"frv_001","artifact_refs":["art_001"]},
		"decision":{"kind":"certification","id":"cert_001","actor_id":"act_human","reason":"all required evidence present","evidence_refs":["gate_001"],"status":"certified","created_at":"2026-05-14T18:30:00Z"},
		"audit_report":{"id":"aud_001","target_type":"release_candidate","target_id":"rc_001","status":"incomplete","trace_score":0.75,"missing_links":["missing RuntimeResult rr_001"]},
		"timeline":[
			{"label":"FactoryOrder recorded","kind":"FactoryOrder","status":"draft","node_id":"fo_001","created_at":"2026-05-14T18:00:00Z","summary":"operator request accepted"},
			{"label":"Gate evaluated","kind":"GateResult","status":"pass","node_id":"gate_001","created_at":"2026-05-14T18:10:00Z","summary":"unit tests passed"}
		],
		"gate_evidence":[{"gate_name":"unit_tests","status":"pass","gate_result_id":"gate_001","evidence_refs":["tr_001"],"waiver_ref":"","missing_refs":[]}],
		"release_evidence":[{"label":"packaged artifact path","status":"complete","artifact_refs":["art_001"],"runtime_refs":["frv_001"],"bom_refs":["bom_001"],"required_path_refs":["path_release_artifact"],"missing_refs":[]}],
		"failures_repairs":[{"failure_id":"fail_001","failure_class":"traceability_gap","severity":"high","summary":"missing evidence fixture","task_id":"tsk_001","gate_result_id":"gate_fail_001","test_run_id":"tr_001","repair_id":"rep_001","repair_status":"planned","actor_invocation_id":"inv_001"}],
		"missing_provenance":[{"path_name":"Task -> RuntimeEnvelope -> RuntimeResult","node_ids":["tsk_001"],"edge_ids":[],"missing":["missing RuntimeResult rr_001"],"completed":false}],
		"proof_of_work_packet":{
			"id":"pow_001",
			"status":"complete",
			"summary":"Read-only Site packet for a bounded D0b evidence view.",
			"event_graph_refs":["eg://factory_order/fo_001","eg://release_candidate/rc_001"],
			"work_item":{"label":"D0b Site proof-of-work packet view","status":"done","summary":"Display projected packet evidence without mutation controls.","artifact_ref":"task://tsk_001","event_graph_refs":["eg://task/tsk_001"]},
			"runtime_invocation":{"label":"local deterministic RuntimeBroker invocation","status":"done","summary":"Bounded worker invocation completed under local policy.","artifact_ref":"runtime://inv_001","event_graph_refs":["eg://actor_invocation/inv_001","eg://runtime_result/rr_001"]},
			"changed_files":[{"label":"graph/ops.go","status":"changed","summary":"Projection decode structs extended.","artifact_ref":"artifact://codechange_001","event_graph_refs":["eg://code_change/cc_001"]}],
			"tests_run":[{"label":"go test -count=1 ./graph","status":"pass","summary":"Graph package validation passed.","artifact_ref":"test://tr_001","event_graph_refs":["eg://test_run/tr_001"]}],
			"ci_status":{"label":"GitHub Build & Test","status":"pass","summary":"Required CI checks passed.","artifact_ref":"ci://run_001","event_graph_refs":["eg://gate_result/gate_001"]},
			"review_feedback":[{"label":"Claude review","status":"addressed","summary":"No blocking findings remain.","artifact_ref":"review://rev_001","event_graph_refs":["eg://review/rev_001"]}],
			"security_scan_results":[{"label":"CodeQL scan","status":"pass","summary":"No high or critical findings.","artifact_ref":"security://scan_001","event_graph_refs":["eg://security_scan/sec_001"]}],
			"screenshots_walkthrough_artifacts":[{"label":"ops evidence walkthrough","status":"recorded","summary":"Operator walkthrough artifact captured.","artifact_ref":"screenshot://pow_001","event_graph_refs":["eg://artifact/shot_001"]}],
			"known_failures":[{"label":"missing RuntimeResult rr_001","status":"open","summary":"Fixture keeps one known traceability gap visible.","artifact_ref":"failure://fail_001","event_graph_refs":["eg://failure/fail_001"]}],
			"operator_decision":{"label":"projected operator decision","status":"recorded","summary":"Decision is displayed from projection only.","artifact_ref":"decision://cert_001","event_graph_refs":["eg://certification/cert_001"]}
		},
		"errors":[]
	}`
}

func TestOpsRoleTimelineRendersSocietyView(t *testing.T) {
	// Stand up a fake hive operator projection server returning lifecycle roles
	// + one approved pull_request.create authority decision.
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"generated_at":"2026-06-05T10:00:00Z",
			"source":"eventgraph",
			"lifecycle":[
				{"actor_id":"act-strategist","display_name":"strategist","role":"strategist","lifecycle_status":"active","updated_at":"2026-06-05T10:00:00Z"},
				{"actor_id":"act-planner","display_name":"planner","role":"planner","lifecycle_status":"active","updated_at":"2026-06-05T10:01:00Z"},
				{"actor_id":"act-implementer","display_name":"implementer","role":"implementer","lifecycle_status":"active","updated_at":"2026-06-05T10:02:00Z"},
				{"actor_id":"act-reviewer","display_name":"reviewer","role":"reviewer","lifecycle_status":"active","updated_at":"2026-06-05T10:03:00Z"},
				{"actor_id":"act-guardian","display_name":"guardian","role":"guardian","lifecycle_status":"active","updated_at":"2026-06-05T10:04:00Z"}
			],
			"authority_decisions":[
				{"decision_id":"dec-001","request_id":"req-001","approver_actor":"act-human","outcome":"approved","approved_action":"pull_request.create","approved_target":"transpara-ai/docs","rationale":"reviewed","created_at":"2026-06-05T10:05:00Z"}
			]
		}`))
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)
	// Evidence projection not needed for the role timeline; set to empty so it
	// renders its unconfigured state without blocking the timeline.
	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", "")

	// Use NewHandlers(nil,nil,nil) so this test runs without a database.
	h := NewHandlers(nil, nil, nil)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/evidence?profile=transpara", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/evidence: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()

	// The seven canonical labels must all appear in the body.
	labels := []string{"strategist", "planner", "implementer", "reviewer", "guardian", "human (Site approval)", "draft PR"}
	for _, label := range labels {
		if !strings.Contains(body, label) {
			t.Fatalf("role timeline: body does not contain role label %q", label)
		}
	}

	// They must appear IN ORDER (each label's index must be strictly increasing).
	prev := -1
	for _, label := range labels {
		idx := strings.Index(body, label)
		if idx <= prev {
			t.Fatalf("role timeline: label %q appears at index %d, expected > %d (out of order)", label, idx, prev)
		}
		prev = idx
	}

	// Evidence surface must remain read-only (no mutation controls).
	evidenceStart := strings.Index(body, "Society view")
	if evidenceStart < 0 {
		t.Fatal("role timeline: body does not contain 'Society view' header")
	}
	surface := body[evidenceStart:]
	if end := strings.Index(surface, "</main>"); end >= 0 {
		surface = surface[:end]
	}
	for _, forbidden := range []string{"<form", "<button", `method="post"`, `action="/ops/evidence"`, "Certify release", "Execute work"} {
		if strings.Contains(surface, forbidden) {
			t.Fatalf("role timeline: evidence surface contains mutation control %q", forbidden)
		}
	}

	// The four guardrail strings must appear in the society-view surface.
	for _, guardrail := range []string{
		"Site does not write the graph or call GitHub",
		"effect = none",
		"EventGraph is truth",
		"Site cannot certify",
	} {
		if !strings.Contains(surface, guardrail) {
			t.Fatalf("role timeline: society view surface missing guardrail %q", guardrail)
		}
	}
}

func opsEvidenceFixtureWithoutProofOfWorkPacketJSON() string {
	return strings.Replace(opsEvidenceFixtureJSON(), `,
		"proof_of_work_packet":{
			"id":"pow_001",
			"status":"complete",
			"summary":"Read-only Site packet for a bounded D0b evidence view.",
			"event_graph_refs":["eg://factory_order/fo_001","eg://release_candidate/rc_001"],
			"work_item":{"label":"D0b Site proof-of-work packet view","status":"done","summary":"Display projected packet evidence without mutation controls.","artifact_ref":"task://tsk_001","event_graph_refs":["eg://task/tsk_001"]},
			"runtime_invocation":{"label":"local deterministic RuntimeBroker invocation","status":"done","summary":"Bounded worker invocation completed under local policy.","artifact_ref":"runtime://inv_001","event_graph_refs":["eg://actor_invocation/inv_001","eg://runtime_result/rr_001"]},
			"changed_files":[{"label":"graph/ops.go","status":"changed","summary":"Projection decode structs extended.","artifact_ref":"artifact://codechange_001","event_graph_refs":["eg://code_change/cc_001"]}],
			"tests_run":[{"label":"go test -count=1 ./graph","status":"pass","summary":"Graph package validation passed.","artifact_ref":"test://tr_001","event_graph_refs":["eg://test_run/tr_001"]}],
			"ci_status":{"label":"GitHub Build & Test","status":"pass","summary":"Required CI checks passed.","artifact_ref":"ci://run_001","event_graph_refs":["eg://gate_result/gate_001"]},
			"review_feedback":[{"label":"Claude review","status":"addressed","summary":"No blocking findings remain.","artifact_ref":"review://rev_001","event_graph_refs":["eg://review/rev_001"]}],
			"security_scan_results":[{"label":"CodeQL scan","status":"pass","summary":"No high or critical findings.","artifact_ref":"security://scan_001","event_graph_refs":["eg://security_scan/sec_001"]}],
			"screenshots_walkthrough_artifacts":[{"label":"ops evidence walkthrough","status":"recorded","summary":"Operator walkthrough artifact captured.","artifact_ref":"screenshot://pow_001","event_graph_refs":["eg://artifact/shot_001"]}],
			"known_failures":[{"label":"missing RuntimeResult rr_001","status":"open","summary":"Fixture keeps one known traceability gap visible.","artifact_ref":"failure://fail_001","event_graph_refs":["eg://failure/fail_001"]}],
			"operator_decision":{"label":"projected operator decision","status":"recorded","summary":"Decision is displayed from projection only.","artifact_ref":"decision://cert_001","event_graph_refs":["eg://certification/cert_001"]}
		}`, "", 1)
}
