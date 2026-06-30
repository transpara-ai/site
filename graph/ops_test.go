package graph

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
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

func TestOpsControlActionsUseQueueOnlyVocabulary(t *testing.T) {
	for _, action := range opsControlActions() {
		if action.ActionLabel == "" {
			t.Fatalf("action %s has empty action label", action.Kind)
		}
		for _, value := range []string{action.ActionLabel, action.Label, action.Description, action.Placeholder, action.Status} {
			if opsControlContainsForbiddenVerb(value) {
				t.Fatalf("control action %s contains forbidden execution vocabulary in %q", action.Kind, value)
			}
		}
	}
}

func TestBuildOpsObservationDataDoesNotMarkUnavailableCivilizationProjectionCurrent(t *testing.T) {
	now := time.Date(2026, 6, 29, 14, 0, 0, 0, time.UTC)
	telemetry := &OpsTelemetryData{GeneratedAt: now.Add(-time.Minute).Format(time.RFC3339)}
	hive := &OpsHiveData{GeneratedAt: now.Add(-2 * time.Minute).Format(time.RFC3339)}
	canonical := &OpsGitHubCanonicalData{GeneratedAt: now.Format(time.RFC3339), ProjectionState: "available"}

	for _, tt := range []struct {
		name        string
		projection  string
		wantValue   string
		wantState   string
		wantContext string
	}{
		{
			name:        "unavailable projection stays degraded",
			projection:  opsCivilizationProjectionStatusUnavailable,
			wantValue:   "degraded",
			wantState:   "degraded",
			wantContext: "Civilization projection unavailable",
		},
		{
			name:        "complete projection is current",
			projection:  opsCivilizationProjectionStatusComplete,
			wantValue:   "current",
			wantState:   "current",
			wantContext: "sources checked",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			civilization := &OpsCivilizationAssemblyData{
				GeneratedAt:      now.Add(-3 * time.Minute).Format(time.RFC3339),
				ProjectionStatus: tt.projection,
			}
			data := buildOpsObservationData(now, telemetry, hive, civilization, canonical, nil)
			var health *OpsObservationMetric
			for i := range data.Metrics {
				if data.Metrics[i].Label == "Civilization health" {
					health = &data.Metrics[i]
					break
				}
			}
			if health == nil {
				t.Fatalf("Civilization health metric missing: %#v", data.Metrics)
			}
			if health.Value != tt.wantValue || health.State != tt.wantState || health.Context != tt.wantContext {
				t.Fatalf("health metric = value:%q state:%q context:%q, want value:%q state:%q context:%q", health.Value, health.State, health.Context, tt.wantValue, tt.wantState, tt.wantContext)
			}
		})
	}
}

func TestBuildOpsObservationDataSummarizesLevel1Canary(t *testing.T) {
	now := time.Date(2026, 6, 30, 10, 30, 0, 0, time.UTC)
	civilization := &OpsCivilizationAssemblyData{
		GeneratedAt:      now.Add(-time.Minute).Format(time.RFC3339),
		ProjectionStatus: opsCivilizationProjectionStatusComplete,
		IssueIntake: OpsCivilizationIssueIntake{
			Issues: []OpsCivilizationIssueIntakeProjected{
				{Repo: "transpara-ai/docs", Number: 226, Title: "Authorize live production EventGraph and runtime operation path"},
				{Repo: "transpara-ai/operation", Number: 45, Title: "Produce deployed public proof"},
			},
		},
		IssueScanKanban: OpsCivilizationIssueScanKanban{
			Status: opsCivilizationFieldAvailable,
			Columns: []OpsCivilizationIssueScanKanbanColumn{
				{
					State: "human_action",
					Label: "Human Action",
					Cards: []OpsCivilizationIssueScanKanbanCard{
						{
							CurrentState: "human_action",
							SelectedIssue: OpsCivilizationIssueRef{
								Repo:   "transpara-ai/docs",
								Number: 226,
								Title:  "Authorize live production EventGraph and runtime operation path",
							},
							Blockers: []OpsCivilizationIssueScanBlockerProjected{
								{
									BlockerType:    "protected_action",
									RequiredAction: "human must authorize the protected-action boundary before Hive may continue",
									EvidenceRefs:   []string{"019f17bf-f8cc-7a3f-a305-7d045eb7786b"},
								},
							},
						},
						{
							CurrentState: "pr_ready",
							SelectedIssue: OpsCivilizationIssueRef{
								Repo:   "transpara-ai/operation",
								Number: 45,
								Title:  "Produce deployed public proof",
							},
							EvidenceRefs: []string{"github:transpara-ai/operation#45"},
						},
					},
				},
			},
		},
	}

	data := buildOpsObservationData(now, nil, nil, civilization, &OpsGitHubCanonicalData{}, nil)
	if data.Canary.DiscoveredIssueCount != 2 || data.Canary.PRReadyIssueCount != 1 || data.Canary.ParkedIssueCount != 1 || data.Canary.HumanActionCount != 1 || data.Canary.ProtectedActionCount != 1 {
		t.Fatalf("canary counts = discovered:%d prReady:%d parked:%d human:%d protected:%d",
			data.Canary.DiscoveredIssueCount,
			data.Canary.PRReadyIssueCount,
			data.Canary.ParkedIssueCount,
			data.Canary.HumanActionCount,
			data.Canary.ProtectedActionCount,
		)
	}
	if len(data.Canary.Rows) != 2 {
		t.Fatalf("canary rows = %d, want 2", len(data.Canary.Rows))
	}
	if data.Canary.Rows[0].EvidenceRef == "" || data.Canary.Boundary == "" || !strings.Contains(data.Canary.Summary, "2 issue(s) discovered") {
		t.Fatalf("canary summary/evidence not populated: %#v", data.Canary)
	}
}

func TestBuildOpsObservationDataDedupesLevel1CanaryIssueCards(t *testing.T) {
	now := time.Date(2026, 6, 30, 10, 45, 0, 0, time.UTC)
	civilization := &OpsCivilizationAssemblyData{
		GeneratedAt:      now.Format(time.RFC3339),
		ProjectionStatus: opsCivilizationProjectionStatusComplete,
		IssueIntake: OpsCivilizationIssueIntake{
			Issues: []OpsCivilizationIssueIntakeProjected{
				{Repo: "transpara-ai/docs", Number: 226, Title: "Authorize live production EventGraph and runtime operation path"},
			},
		},
		IssueScanKanban: OpsCivilizationIssueScanKanban{
			Status: opsCivilizationFieldAvailable,
			Columns: []OpsCivilizationIssueScanKanbanColumn{
				{
					State: "pr_ready",
					Label: "PR-ready",
					Cards: []OpsCivilizationIssueScanKanbanCard{
						{
							CurrentState: "pr_ready",
							SelectedIssue: OpsCivilizationIssueRef{
								Repo:   "transpara-ai/docs",
								Number: 226,
								Title:  "Authorize live production EventGraph and runtime operation path",
							},
							EvidenceRefs: []string{"github:transpara-ai/docs#226"},
						},
					},
				},
				{
					State: "human_action",
					Label: "Human Action",
					Cards: []OpsCivilizationIssueScanKanbanCard{
						{
							CurrentState: "human_action",
							SelectedIssue: OpsCivilizationIssueRef{
								Repo:   "transpara-ai/docs",
								Number: 226,
								Title:  "Authorize live production EventGraph and runtime operation path",
							},
							Blockers: []OpsCivilizationIssueScanBlockerProjected{
								{
									BlockerType:    "protected_action",
									RequiredAction: "human must authorize the protected-action boundary before Hive may continue",
									EvidenceRefs:   []string{"eventgraph:parked-docs-226"},
								},
							},
						},
					},
				},
			},
		},
	}

	data := buildOpsObservationData(now, nil, nil, civilization, &OpsGitHubCanonicalData{}, nil)
	if data.Canary.DiscoveredIssueCount != 1 || data.Canary.PRReadyIssueCount != 0 || data.Canary.ParkedIssueCount != 1 || data.Canary.HumanActionCount != 1 || data.Canary.ProtectedActionCount != 1 {
		t.Fatalf("deduped canary counts = discovered:%d prReady:%d parked:%d human:%d protected:%d",
			data.Canary.DiscoveredIssueCount,
			data.Canary.PRReadyIssueCount,
			data.Canary.ParkedIssueCount,
			data.Canary.HumanActionCount,
			data.Canary.ProtectedActionCount,
		)
	}
	if len(data.Canary.Rows) != 1 {
		t.Fatalf("canary rows = %d, want 1: %#v", len(data.Canary.Rows), data.Canary.Rows)
	}
	if data.Canary.Rows[0].State != "human_action" || data.Canary.Rows[0].EvidenceRef != "eventgraph:parked-docs-226" {
		t.Fatalf("canary row = %#v, want human_action protected evidence row", data.Canary.Rows[0])
	}
}

func TestOpsControlIntentParamsRejectsForbiddenExecutionVocabulary(t *testing.T) {
	form := url.Values{}
	form.Set("intent_kind", "council_meeting")
	form.Set("title", "Invoke Council")
	form.Set("content", "Queue a review discussion.")
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/control/intents", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if _, err := opsControlIntentParamsFromForm(req, "operator", opsModelTargetValueSet(opsControlModelTargetOptions(nil))); err == nil {
		t.Fatal("opsControlIntentParamsFromForm accepted forbidden execution vocabulary")
	}
}

func TestOpsControlIntentParamsUsesWordBoundaryVocabularyGuard(t *testing.T) {
	form := url.Values{}
	form.Set("intent_kind", "model_policy")
	form.Set("title", "Dataset reviewer budget request")
	form.Set("target", "reviewer")
	form.Set("requested_by", "operator")
	form.Set("content", "Request a model change for the dataset reviewer.")
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/control/intents", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	params, err := opsControlIntentParamsFromForm(req, "operator", opsModelTargetValueSet(opsControlModelTargetOptions(nil)))
	if err != nil {
		t.Fatalf("opsControlIntentParamsFromForm rejected safe text: %v", err)
	}
	var meta opsControlIntentMeta
	if err := json.Unmarshal([]byte(params.Detail), &meta); err != nil {
		t.Fatalf("control intent detail is not JSON: %v", err)
	}
	if meta.IntentKind != "model_policy" || meta.Target != "reviewer" || meta.RequestedBy != "operator" {
		t.Fatalf("meta = %#v, want model_policy/reviewer/operator", meta)
	}
}

func TestOpsControlIntentParamsRejectsFreeTextAgentRoleTarget(t *testing.T) {
	form := url.Values{}
	form.Set("intent_kind", "model_policy")
	form.Set("title", "Dataset reviewer model request")
	form.Set("target", "dataset-reviewer")
	form.Set("requested_by", "operator")
	form.Set("content", "Request a model change for the dataset reviewer.")
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/control/intents", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if _, err := opsControlIntentParamsFromForm(req, "operator", opsModelTargetValueSet(opsControlModelTargetOptions(nil))); err == nil {
		t.Fatal("opsControlIntentParamsFromForm accepted a model-policy target outside the agent/role dropdown catalog")
	}
	if _, err := opsControlIntentParamsFromForm(req, "operator", map[string]bool{}); err == nil {
		t.Fatal("opsControlIntentParamsFromForm accepted model-policy target with an empty dropdown catalog")
	}
}

func TestOpsControlModelTargetOptionsIncludeEmergentRoles(t *testing.T) {
	options := opsControlModelTargetOptions(&OpsHiveData{
		ModelSelection: OpsHiveModelSelection{Assignments: []OpsHiveModelRoleAssignment{{Role: "runtime-specialist", CanOperate: true}}},
		Lifecycle:      []OpsHiveLifecycle{{ActorID: "actor-runtime-specialist", Role: "runtime-specialist", LifecycleStatus: "planned"}},
	})
	values := opsModelTargetValueSet(options)
	for _, want := range []string{"strategist", "reviewer", "designer", "legal", "runtime-specialist", "actor-runtime-specialist"} {
		if !values[want] {
			t.Fatalf("model target options missing %q in %#v", want, options)
		}
	}
	for _, notAvailable := range []string{"scribe", "budget"} {
		if values[notAvailable] {
			t.Fatalf("model target options included unavailable role %q in %#v", notAvailable, options)
		}
	}
}

func TestOpsHiveLaunchableIntakeSourcesExcludesControlAndFactoryKinds(t *testing.T) {
	got := opsHiveLaunchableIntakeSources([]OpsHiveIntakeSource{
		{Kind: opsControlIntentSourceKind, Title: "Control intent"},
		{Kind: opsMarkdownArtifactSourceKind, Title: "Human artifact"},
		{Kind: "Text", Title: "Launchable source"},
		{Kind: "draft_note", Title: "Unknown internal draft"},
	})
	if len(got) != 1 || got[0].Title != "Launchable source" {
		t.Fatalf("launchable sources = %#v, want only ordinary intake source", got)
	}
}

func TestOpsHiveClassifiedIntakeKindsAreLaunchable(t *testing.T) {
	cases := []struct {
		name    string
		kind    string
		title   string
		content string
	}{
		{name: "url", kind: "url", content: "https://example.com/reference"},
		{name: "repo", kind: "repo", content: "transpara-ai/site"},
		{name: "prd", kind: "text", title: "Checkout PRD", content: "Acceptance criteria and requirements."},
		{name: "spec", kind: "text", title: "API contract", content: "Schema and API contract."},
		{name: "plan", kind: "text", title: "Milestone plan", content: "Roadmap and milestone plan."},
		{name: "text", kind: "text", title: "Operator notes", content: "Plain source notes."},
		{name: "unknown raw kind normalizes to text", kind: "draft_note", title: "Draft note", content: "Plain source notes."},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{
				"source_kind": {tt.kind},
				"title":       {tt.title},
				"content":     {tt.content},
			}
			req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/sources", strings.NewReader(values.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			params, err := opsHiveIntakeSourceParamsFromForm(req)
			if err != nil {
				t.Fatalf("opsHiveIntakeSourceParamsFromForm: %v", err)
			}
			if !opsHiveLaunchableIntakeKind(params.Kind) {
				t.Fatalf("classified kind %q is not launchable", params.Kind)
			}
		})
	}
}

func TestListLaunchableOpsHiveIntakeSourcesFiltersBeforeLimit(t *testing.T) {
	_, store, _ := testHandlers(t)
	profileSlug := fmt.Sprintf("launchable-limit-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = store.db.ExecContext(t.Context(), `DELETE FROM ops_hive_intake_sources WHERE profile_slug = $1`, profileSlug)
	})

	launchable, err := store.CreateOpsHiveIntakeSource(t.Context(), CreateOpsHiveIntakeSourceParams{
		ProfileSlug: profileSlug,
		Kind:        "Text",
		Title:       "Launchable source",
		Content:     "Issue brief and acceptance criteria.",
		Status:      "parsed",
	})
	if err != nil {
		t.Fatalf("create launchable source: %v", err)
	}
	if _, err := store.db.ExecContext(t.Context(), `UPDATE ops_hive_intake_sources SET created_at = $1 WHERE id = $2`, time.Now().Add(-time.Hour), launchable.ID); err != nil {
		t.Fatalf("age launchable source: %v", err)
	}
	for i := 0; i < opsHiveLaunchableIntakeListLimit+5; i++ {
		if _, err := store.CreateOpsHiveIntakeSource(t.Context(), CreateOpsHiveIntakeSourceParams{
			ProfileSlug: profileSlug,
			Kind:        opsControlIntentSourceKind,
			Title:       fmt.Sprintf("Internal control intent %03d", i),
			Content:     "Site-local intent.",
			Status:      "Queued",
		}); err != nil {
			t.Fatalf("create internal source %d: %v", i, err)
		}
	}

	sources, err := store.ListLaunchableOpsHiveIntakeSources(t.Context(), profileSlug, 1)
	if err != nil {
		t.Fatalf("ListLaunchableOpsHiveIntakeSources: %v", err)
	}
	if len(sources) != 1 || sources[0].ID != launchable.ID {
		t.Fatalf("launchable sources = %#v, want only aged launchable source despite newer internal rows", sources)
	}
}

func TestHandleOpsControlAndFactoryRecordsDoNotEnterHiveLaunchIntake(t *testing.T) {
	h, store, _ := testHandlers(t)
	if _, err := store.db.ExecContext(t.Context(), `DELETE FROM ops_hive_intake_sources`); err != nil {
		t.Fatalf("clear intake sources: %v", err)
	}
	t.Cleanup(func() {
		_, _ = store.db.ExecContext(t.Context(), `DELETE FROM ops_hive_intake_sources`)
	})
	mux := http.NewServeMux()
	h.Register(mux)

	controlForm := url.Values{}
	controlForm.Set("intent_kind", "model_policy")
	controlForm.Set("title", "Dataset reviewer model request")
	controlForm.Set("target", "reviewer")
	controlForm.Set("requested_by", "operator")
	controlForm.Set("content", "Request a model change for the dataset reviewer.")
	controlReq := httptest.NewRequest(http.MethodPost, "http://site.test/ops/control/intents", strings.NewReader(controlForm.Encode()))
	controlReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	controlResp := httptest.NewRecorder()
	mux.ServeHTTP(controlResp, controlReq)
	if controlResp.Code != http.StatusOK {
		t.Fatalf("POST /ops/control/intents status = %d, want 200; body: %s", controlResp.Code, controlResp.Body.String())
	}
	if !strings.Contains(controlResp.Body.String(), "Queued intent recorded") {
		t.Fatalf("control response missing queued confirmation: %s", controlResp.Body.String())
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("submitter_name", "Ada Operator")
	_ = writer.WriteField("title", "Factory artifact")
	part, err := writer.CreateFormFile("artifact", "factory.md")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("# Factory artifact\n\nHuman-submitted artifact.")); err != nil {
		t.Fatalf("write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close multipart writer: %v", err)
	}
	factoryReq := httptest.NewRequest(http.MethodPost, "http://site.test/factory/artifacts", body)
	factoryReq.Header.Set("Content-Type", writer.FormDataContentType())
	factoryResp := httptest.NewRecorder()
	mux.ServeHTTP(factoryResp, factoryReq)
	if factoryResp.Code != http.StatusOK {
		t.Fatalf("POST /factory/artifacts status = %d, want 200; body: %s", factoryResp.Code, factoryResp.Body.String())
	}
	if !strings.Contains(factoryResp.Body.String(), "Artifact submitted. FactoryOrder conversion requires separate governed review.") {
		t.Fatalf("factory response missing governed-review confirmation: %s", factoryResp.Body.String())
	}

	sources, err := store.ListOpsHiveIntakeSources(t.Context(), "transpara-ai", 25)
	if err != nil {
		t.Fatalf("ListOpsHiveIntakeSources: %v", err)
	}
	if len(sources) != 2 {
		t.Fatalf("stored sources = %#v, want two Site-local records", sources)
	}
	intakeReq := httptest.NewRequest(http.MethodGet, "http://site.test/ops/hive/intake?profile=transpara-ai", nil)
	intake := h.buildOpsHiveIntakeView(intakeReq)
	if len(intake.Sources) != 0 {
		t.Fatalf("Hive intake surfaced internal control/factory records: %#v", intake.Sources)
	}
	launchReq := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/launch?profile=transpara-ai", strings.NewReader("target_repos=transpara-ai/site&max_iterations=1&max_cost_usd=1"))
	launchReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := launchReq.ParseForm(); err != nil {
		t.Fatalf("ParseForm: %v", err)
	}
	if _, _, err := h.buildOpsHiveRunLaunchPayload(launchReq, "transpara-ai"); err == nil {
		t.Fatal("buildOpsHiveRunLaunchPayload accepted internal control/factory records as launchable sources")
	}
}

func TestNewOperatorMutationRoutesRejectCrossOriginPost(t *testing.T) {
	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)
	factoryBody := &bytes.Buffer{}
	factoryWriter := multipart.NewWriter(factoryBody)
	_ = factoryWriter.WriteField("submitter_name", "Ada Operator")
	_ = factoryWriter.WriteField("title", "Factory artifact")
	factoryPart, err := factoryWriter.CreateFormFile("artifact", "factory.md")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := factoryPart.Write([]byte("# Factory artifact")); err != nil {
		t.Fatalf("write multipart content: %v", err)
	}
	if err := factoryWriter.Close(); err != nil {
		t.Fatalf("Close multipart writer: %v", err)
	}

	for _, tt := range []struct {
		name        string
		path        string
		body        io.Reader
		contentType string
	}{
		{
			name:        "control intent",
			path:        "/ops/control/intents",
			body:        strings.NewReader("intent_kind=model_policy&title=Reviewer+model&target=reviewer&content=Request+model+review"),
			contentType: "application/x-www-form-urlencoded",
		},
		{
			name:        "factory artifact",
			path:        "/factory/artifacts",
			body:        factoryBody,
			contentType: factoryWriter.FormDataContentType(),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://site.test"+tt.path, tt.body)
			req.Header.Set("Content-Type", tt.contentType)
			req.Header.Set("Origin", "https://evil.test")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			if w.Code != http.StatusForbidden {
				t.Fatalf("POST %s status = %d, want 403; body: %s", tt.path, w.Code, w.Body.String())
			}
		})
	}
}

func TestOpsFactorySubmissionParamsAcceptsMarkdownArtifact(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("submitter_name", "Ada Operator")
	_ = writer.WriteField("submitter_email", "ada@example.invalid")
	_ = writer.WriteField("submitter_org", "Factory Test")
	_ = writer.WriteField("title", "Artifact brief")
	part, err := writer.CreateFormFile("artifact", "brief.md")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	content := []byte("# Brief\n\nConvert this into a governed FactoryOrder candidate.")
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://site.test/factory/artifacts", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	params, err := opsFactorySubmissionParamsFromRequest(req)
	if err != nil {
		t.Fatalf("opsFactorySubmissionParamsFromRequest: %v", err)
	}
	if params.Kind != opsMarkdownArtifactSourceKind || params.Status != "submitted" {
		t.Fatalf("params = %#v, want markdown_artifact/submitted", params)
	}
	sum := sha256.Sum256(content)
	if !strings.Contains(params.Detail, hex.EncodeToString(sum[:])) {
		t.Fatalf("params.Detail = %q, want sha256", params.Detail)
	}
}

func TestOpsFactorySubmissionParamsRejectsNonMarkdownArtifact(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("submitter_name", "Ada Operator")
	part, err := writer.CreateFormFile("artifact", "brief.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write([]byte("not markdown")); err != nil {
		t.Fatalf("write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://site.test/factory/artifacts", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if _, err := opsFactorySubmissionParamsFromRequest(req); err == nil {
		t.Fatal("opsFactorySubmissionParamsFromRequest accepted a non-Markdown artifact")
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
	h, store, _ := testHandlers(t)
	if _, err := store.db.ExecContext(t.Context(), `DELETE FROM ops_hive_intake_sources`); err != nil {
		t.Fatalf("clear intake sources: %v", err)
	}
	t.Cleanup(func() {
		_, _ = store.db.ExecContext(t.Context(), `DELETE FROM ops_hive_intake_sources`)
	})
	mux := http.NewServeMux()
	h.Register(mux)

	tests := []struct {
		path      string
		want      []string
		forbidden []string
	}{
		{
			path:      "/ops/hive/intake?profile=transpara",
			want:      []string{"Hive intake", "Source input", "Live interpretation", "Add source", "Factory brief", "No scoped sources yet.", "No intake sources saved yet"},
			forbidden: []string{"<iframe", "Hive operator projection source is not configured", "checkout-redesign.md"},
		},
		{
			path:      "/ops/hive/runs?profile=transpara",
			want:      []string{"Hive runs", "Run tower", "run_static_001", "Build onboarding control surface"},
			forbidden: []string{"<iframe", `method="post"`, `action="/ops/hive`, "Hive operator projection source is not configured"},
		},
		{
			path:      "/ops/hive/agents?profile=transpara",
			want:      []string{"Hive agents", "Agent topology", "guardian", "architect"},
			forbidden: []string{"<iframe", `method="post"`, `action="/ops/hive`, "Hive operator projection source is not configured"},
		},
		{
			path:      "/ops/hive/resources?profile=transpara",
			want:      []string{"Hive resources", "Run budget", "Approval queue", "Read-only mode"},
			forbidden: []string{"<iframe", `method="post"`, `action="/ops/hive`, "Hive operator projection source is not configured"},
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
			for _, forbidden := range tt.forbidden {
				if strings.Contains(body, forbidden) {
					t.Fatalf("GET %s: body contains forbidden %q", tt.path, forbidden)
				}
			}
		})
	}
}

func TestHandleOpsHiveIntakePersistsSources(t *testing.T) {
	h, store, _ := testHandlers(t)
	if _, err := store.db.ExecContext(t.Context(), `DELETE FROM ops_hive_intake_sources`); err != nil {
		t.Fatalf("clear intake sources: %v", err)
	}
	t.Cleanup(func() {
		_, _ = store.db.ExecContext(t.Context(), `DELETE FROM ops_hive_intake_sources`)
	})
	mux := http.NewServeMux()
	h.Register(mux)

	postSource := func(profileSlug string, values url.Values) {
		t.Helper()
		values.Set("profile", profileSlug)
		req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/sources", strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusSeeOther {
			t.Fatalf("POST /ops/hive/intake/sources: status = %d, want 303; body: %s", w.Code, w.Body.String())
		}
		if got, want := w.Header().Get("Location"), "/ops/hive/intake?profile="+profileSlug; got != want {
			t.Fatalf("POST /ops/hive/intake/sources: Location = %q, want %q", got, want)
		}
	}

	postSource("transpara", url.Values{
		"source_kind": {"text"},
		"title":       {"Checkout PRD"},
		"content":     {"PRD\nAcceptance criteria: operator can review intake sources before launch."},
	})
	postSource("transpara", url.Values{
		"source_kind": {"url"},
		"content":     {"https://example.com/customer-notes"},
	})
	postSource("transpara", url.Values{
		"source_kind": {"repo"},
		"content":     {"transpara-ai/site"},
	})
	postSource("transpara-ai", url.Values{
		"source_kind": {"text"},
		"title":       {"Default profile only"},
		"content":     {"Default-profile material should not appear in the Transpara intake."},
	})

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/hive/intake?profile=transpara", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/hive/intake: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"Checkout PRD", "PRD", "parsed", "example.com/customer-notes", "URL", "classified", "transpara-ai/site", "Repo", "scoped", "draft ready", "full product pipeline", "persisted", "Factory brief", "Acceptance criteria: operator can review intake sources before launch.", "URL reference: ready", "Repo context: ready", "Budget cap: warning", "Readiness: draft ready / full product pipeline"} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/hive/intake: body does not contain %q", want)
		}
	}
	for _, forbidden := range []string{`name="brief_title"`, `name="brief_objective"`, `name="brief_scope"`, `name="brief_acceptance"`, `name="brief_risks"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("GET /ops/hive/intake rendered submittable brief preview control %q", forbidden)
		}
	}
	if !strings.Contains(body, "cap pending") {
		t.Fatal("GET /ops/hive/intake: body does not show pending budget cap")
	}
	if strings.Contains(body, "$18.00") {
		t.Fatal("GET /ops/hive/intake rendered a numeric budget estimate for draft sources")
	}
	if strings.Contains(body, "checkout-redesign.md") || strings.Contains(body, "No intake sources saved yet") {
		t.Fatal("GET /ops/hive/intake rendered static or empty-state sources after persisted sources")
	}
	if strings.Contains(body, "Default profile only") {
		t.Fatal("GET /ops/hive/intake rendered source from another profile")
	}
	formStart := strings.Index(body, `action="/ops/hive/intake/sources"`)
	if formStart < 0 {
		t.Fatal("GET /ops/hive/intake did not render the source form")
	}
	formEnd := strings.Index(body[formStart:], "</form>")
	if formEnd < 0 {
		t.Fatal("GET /ops/hive/intake source form is not closed")
	}
	sourceForm := body[formStart : formStart+formEnd]
	if strings.Contains(sourceForm, `name="brief_`) {
		t.Fatal("GET /ops/hive/intake nested brief preview controls inside the source POST form")
	}
}

func TestHandleOpsHiveIntakeLaunchQueuesRun(t *testing.T) {
	h, store, _ := testHandlers(t)
	clearOpsHiveLaunchTables(t, store)
	mux := http.NewServeMux()
	h.Register(mux)

	if _, err := store.CreateOpsHiveIntakeSource(t.Context(), CreateOpsHiveIntakeSourceParams{
		ProfileSlug: "transpara",
		Kind:        "Spec",
		Title:       "Factory launch spec",
		Content:     "Objective: queue a governed Hive run request from Site intake.",
		Status:      "parsed",
	}); err != nil {
		t.Fatalf("create intake source: %v", err)
	}

	var gotAuth string
	var gotPayload struct {
		OperatorID string `json:"operator_id"`
		IntakeID   string `json:"intake_id"`
		Title      string `json:"title"`
		Brief      struct {
			Title     string   `json:"title"`
			Objective string   `json:"objective"`
			Readiness string   `json:"readiness"`
			Missing   []string `json:"missing"`
		} `json:"brief"`
		Sources []struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Ref   string `json:"ref"`
			Title string `json:"title"`
		} `json:"sources"`
		Authority struct {
			InitialLevel string `json:"initial_level"`
			Scope        string `json:"scope"`
			PolicyRef    string `json:"policy_ref"`
			Rationale    string `json:"rationale"`
		} `json:"authority"`
		Budget struct {
			MaxIterations int     `json:"max_iterations"`
			MaxCostUSD    float64 `json:"max_cost_usd"`
		} `json:"budget"`
		TargetRepos []string `json:"target_repos"`
	}
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hive/runs" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode run launch payload: %v", err)
		}
		if gotPayload.OperatorID != "site_operator_test-user-1" {
			http.Error(w, "operator_id is not the authenticated Site user", http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(gotPayload.IntakeID, "site_transpara_") {
			http.Error(w, "intake_id must be unique and profile scoped", http.StatusBadRequest)
			return
		}
		if gotPayload.Authority.InitialLevel != "Required" || gotPayload.Authority.Scope != "operator-launch" || gotPayload.Authority.PolicyRef == "" {
			http.Error(w, "authority packet does not match Hive run launch contract", http.StatusBadRequest)
			return
		}
		if gotPayload.Budget.MaxIterations <= 0 || gotPayload.Budget.MaxCostUSD < 0 {
			http.Error(w, "budget does not match Hive run launch contract", http.StatusBadRequest)
			return
		}
		if len(gotPayload.TargetRepos) == 0 || !strings.Contains(gotPayload.TargetRepos[0], "/") {
			http.Error(w, "target_repos does not match Hive run launch contract", http.StatusBadRequest)
			return
		}
		if len(gotPayload.Sources) == 0 || gotPayload.Sources[0].Type == "" || gotPayload.Sources[0].Ref == "" {
			http.Error(w, "sources do not match Hive run launch contract", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"run_id":"run_queued_site","status":"queued","first_event_id":"event_source"}`))
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)
	t.Setenv("HIVE_OPS_API_KEY", "secret")

	form := url.Values{
		"profile":        {"transpara"},
		"target_repos":   {"transpara-ai/hive, transpara-ai/work"},
		"max_iterations": {"4"},
		"max_cost_usd":   {"12.50"},
	}
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/launch", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("POST /ops/hive/intake/launch: status = %d, want 303; body: %s", w.Code, w.Body.String())
	}
	if got, want := w.Header().Get("Location"), "/ops/hive/runs?profile=transpara&run_id=run_queued_site"; got != want {
		t.Fatalf("POST /ops/hive/intake/launch: Location = %q, want %q", got, want)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q, want Bearer secret", gotAuth)
	}
	if gotPayload.OperatorID != "site_operator_test-user-1" || !strings.HasPrefix(gotPayload.IntakeID, "site_transpara_") {
		t.Fatalf("identity = operator:%q intake:%q", gotPayload.OperatorID, gotPayload.IntakeID)
	}
	if gotPayload.Title != "Factory launch spec" || gotPayload.Brief.Title != "Factory launch spec" {
		t.Fatalf("title = payload:%q brief:%q", gotPayload.Title, gotPayload.Brief.Title)
	}
	if gotPayload.Brief.Readiness != "draft ready / brief drafting" || !strings.Contains(gotPayload.Brief.Objective, "queue a governed Hive run request") {
		t.Fatalf("brief = %#v", gotPayload.Brief)
	}
	if len(gotPayload.Sources) != 1 || gotPayload.Sources[0].Type != "spec" || !strings.HasPrefix(gotPayload.Sources[0].Ref, "site-intake-source:") {
		t.Fatalf("sources = %#v", gotPayload.Sources)
	}
	if gotPayload.Authority.InitialLevel != "Required" || gotPayload.Authority.Scope != "operator-launch" || !strings.Contains(gotPayload.Authority.Rationale, "queued launch") {
		t.Fatalf("authority = %#v", gotPayload.Authority)
	}
	if gotPayload.Budget.MaxIterations != 4 || gotPayload.Budget.MaxCostUSD != 12.50 {
		t.Fatalf("budget = %#v", gotPayload.Budget)
	}
	if len(gotPayload.TargetRepos) != 2 || gotPayload.TargetRepos[0] != "transpara-ai/hive" || gotPayload.TargetRepos[1] != "transpara-ai/work" {
		t.Fatalf("target_repos = %#v", gotPayload.TargetRepos)
	}

	launches, err := store.ListOpsHiveRunLaunches(t.Context(), "transpara", 10)
	if err != nil {
		t.Fatalf("list run launches: %v", err)
	}
	if len(launches) != 1 {
		t.Fatalf("launches = %#v, want one stored launch", launches)
	}
	if launches[0].RunID != "run_queued_site" || launches[0].Status != "queued" || launches[0].FirstEventID != "event_source" {
		t.Fatalf("stored launch = %#v", launches[0])
	}
	if launches[0].OperatorID != "site_operator_test-user-1" || launches[0].IntakeID != gotPayload.IntakeID {
		t.Fatalf("stored identity = operator:%q intake:%q", launches[0].OperatorID, launches[0].IntakeID)
	}
	if launches[0].BudgetMaxIterations != 4 || launches[0].BudgetMaxCostUSD != 12.50 {
		t.Fatalf("stored budget = %#v", launches[0])
	}

	req = httptest.NewRequest(http.MethodGet, "http://site.test"+w.Header().Get("Location"), nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/hive/runs: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"Queued Hive run requests", "run_queued_site", "event_source", "site_operator_test-user-1", "transpara-ai/hive, transpara-ai/work", "4 iter / $12.50", "not runtime-start evidence"} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/hive/runs: body does not contain %q", want)
		}
	}
}

func TestHandleOpsHiveIntakeLaunchRendersHiveError(t *testing.T) {
	h, store, _ := testHandlers(t)
	clearOpsHiveLaunchTables(t, store)
	mux := http.NewServeMux()
	h.Register(mux)

	if _, err := store.CreateOpsHiveIntakeSource(t.Context(), CreateOpsHiveIntakeSourceParams{
		ProfileSlug: "transpara",
		Kind:        "Text",
		Title:       "Operator notes",
		Content:     "Queue launch only when Hive accepts the authority packet.",
		Status:      "parsed",
	}); err != nil {
		t.Fatalf("create intake source: %v", err)
	}

	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("target_repos[0] must be a safe owner/repo name"))
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	form := url.Values{
		"profile":        {"transpara"},
		"target_repos":   {"bad/repo"},
		"max_iterations": {"4"},
		"max_cost_usd":   {"12.50"},
	}
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/launch", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /ops/hive/intake/launch: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"Hive intake", "Queue Hive run request", "hive returned 400 Bad Request: target_repos[0] must be a safe owner/repo name"} {
		if !strings.Contains(body, want) {
			t.Fatalf("POST /ops/hive/intake/launch: body does not contain %q", want)
		}
	}
	launches, err := store.ListOpsHiveRunLaunches(t.Context(), "transpara", 10)
	if err != nil {
		t.Fatalf("list run launches: %v", err)
	}
	if len(launches) != 0 {
		t.Fatalf("stored launches after failed Hive response = %#v, want none", launches)
	}
}

func TestHandleOpsHiveIntakeLaunchSurfacesStoreFailureWithRunID(t *testing.T) {
	h, store, _ := testHandlers(t)
	clearOpsHiveLaunchTables(t, store)
	mux := http.NewServeMux()
	h.Register(mux)

	if _, err := store.CreateOpsHiveIntakeSource(t.Context(), CreateOpsHiveIntakeSourceParams{
		ProfileSlug: "transpara",
		Kind:        "Text",
		Title:       "Operator notes",
		Content:     "Queue launch only when Hive accepts the authority packet.",
		Status:      "parsed",
	}); err != nil {
		t.Fatalf("create intake source: %v", err)
	}
	if _, err := store.CreateOpsHiveRunLaunch(t.Context(), CreateOpsHiveRunLaunchParams{
		ProfileSlug:         "transpara",
		OperatorID:          "site_operator_test-user-1",
		IntakeID:            "site_transpara_existing",
		RunID:               "run_existing",
		Status:              "queued",
		FirstEventID:        "event_existing",
		Title:               "Existing launch",
		TargetRepos:         []string{"transpara-ai/hive"},
		BudgetMaxIterations: 1,
		BudgetMaxCostUSD:    1,
	}); err != nil {
		t.Fatalf("seed existing launch: %v", err)
	}

	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"run_id":"run_existing","status":"queued","first_event_id":"event_new"}`))
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	form := url.Values{
		"profile":        {"transpara"},
		"target_repos":   {"transpara-ai/hive"},
		"max_iterations": {"4"},
		"max_cost_usd":   {"12.50"},
	}
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/launch", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /ops/hive/intake/launch: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Hive accepted queued run run_existing but Site could not store queued run proof") {
		t.Fatalf("POST /ops/hive/intake/launch: body does not surface accepted run id: %s", body)
	}
	launches, err := store.ListOpsHiveRunLaunches(t.Context(), "transpara", 10)
	if err != nil {
		t.Fatalf("list run launches: %v", err)
	}
	if len(launches) != 1 {
		t.Fatalf("stored launches after duplicate run_id = %#v, want existing launch only", launches)
	}
}

func TestHandleOpsHiveIntakeLaunchRequiresSource(t *testing.T) {
	h, store, _ := testHandlers(t)
	clearOpsHiveLaunchTables(t, store)
	mux := http.NewServeMux()
	h.Register(mux)

	form := url.Values{
		"profile":        {"transpara"},
		"target_repos":   {"transpara-ai/hive"},
		"max_iterations": {"4"},
		"max_cost_usd":   {"12.50"},
	}
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/launch", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST /ops/hive/intake/launch: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "add at least one intake source before queueing a Hive run") {
		t.Fatalf("POST /ops/hive/intake/launch: body = %s", w.Body.String())
	}
}

func TestHandleOpsHiveIntakeLaunchRejectsInvalidFormValuesBeforeHivePost(t *testing.T) {
	h, store, _ := testHandlers(t)
	clearOpsHiveLaunchTables(t, store)
	mux := http.NewServeMux()
	h.Register(mux)

	if _, err := store.CreateOpsHiveIntakeSource(t.Context(), CreateOpsHiveIntakeSourceParams{
		ProfileSlug: "transpara",
		Kind:        "Text",
		Title:       "Operator notes",
		Content:     "Queue launch only when budget and target fields are valid.",
		Status:      "parsed",
	}); err != nil {
		t.Fatalf("create intake source: %v", err)
	}
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Hive must not be called when launch form validation fails")
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	tests := []struct {
		name          string
		targetRepos   string
		maxIterations string
		maxCostUSD    string
		want          string
	}{
		{name: "empty targets", targetRepos: " \t ", maxIterations: "4", maxCostUSD: "12.50", want: "target_repos is required"},
		{name: "zero iterations", targetRepos: "transpara-ai/hive", maxIterations: "0", maxCostUSD: "12.50", want: "budget.max_iterations must be greater than zero"},
		{name: "negative cost", targetRepos: "transpara-ai/hive", maxIterations: "4", maxCostUSD: "-0.01", want: "budget.max_cost_usd must be zero or greater"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{
				"profile":        {"transpara"},
				"target_repos":   {tt.targetRepos},
				"max_iterations": {tt.maxIterations},
				"max_cost_usd":   {tt.maxCostUSD},
			}
			req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/launch", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("POST /ops/hive/intake/launch: status = %d, want 200; body: %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tt.want) {
				t.Fatalf("POST /ops/hive/intake/launch: body does not contain %q: %s", tt.want, w.Body.String())
			}
		})
	}
}

func TestHandleOpsHiveIntakeLaunchRejectsCrossOriginPost(t *testing.T) {
	h, store, _ := testHandlers(t)
	clearOpsHiveLaunchTables(t, store)
	mux := http.NewServeMux()
	h.Register(mux)

	form := url.Values{
		"profile":        {"transpara"},
		"target_repos":   {"transpara-ai/hive"},
		"max_iterations": {"4"},
		"max_cost_usd":   {"12.50"},
	}
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/launch", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.test")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("POST /ops/hive/intake/launch: status = %d, want 403; body: %s", w.Code, w.Body.String())
	}
}

func TestOpsHiveRunLaunchStoreEnforcesGlobalRunIDAndProfileScope(t *testing.T) {
	_, store, _ := testHandlers(t)
	clearOpsHiveLaunchTables(t, store)

	if _, err := store.CreateOpsHiveRunLaunch(t.Context(), CreateOpsHiveRunLaunchParams{
		ProfileSlug:         "transpara",
		OperatorID:          "site_operator_a",
		IntakeID:            "site_transpara_a",
		RunID:               "run_global",
		Status:              "queued",
		FirstEventID:        "event_a",
		Title:               "Profile A launch",
		TargetRepos:         []string{"transpara-ai/hive"},
		BudgetMaxIterations: 4,
		BudgetMaxCostUSD:    12.50,
	}); err != nil {
		t.Fatalf("create profile A launch: %v", err)
	}
	if _, err := store.CreateOpsHiveRunLaunch(t.Context(), CreateOpsHiveRunLaunchParams{
		ProfileSlug:         "transpara-other",
		OperatorID:          "site_operator_b",
		IntakeID:            "site_transpara_b",
		RunID:               "run_global",
		Status:              "queued",
		FirstEventID:        "event_b",
		Title:               "Profile B duplicate launch",
		TargetRepos:         []string{"transpara-ai/hive"},
		BudgetMaxIterations: 4,
		BudgetMaxCostUSD:    12.50,
	}); err == nil {
		t.Fatal("CreateOpsHiveRunLaunch accepted duplicate Hive run_id across profiles")
	}
	if _, err := store.CreateOpsHiveRunLaunch(t.Context(), CreateOpsHiveRunLaunchParams{
		ProfileSlug:         "transpara-other",
		OperatorID:          "site_operator_b",
		IntakeID:            "site_transpara_b",
		RunID:               "run_other",
		Status:              "queued",
		FirstEventID:        "event_b",
		Title:               "Profile B launch",
		TargetRepos:         []string{"transpara-ai/hive"},
		BudgetMaxIterations: 2,
		BudgetMaxCostUSD:    3.50,
	}); err != nil {
		t.Fatalf("create profile B launch: %v", err)
	}

	launches, err := store.ListOpsHiveRunLaunches(t.Context(), "transpara", 10)
	if err != nil {
		t.Fatalf("list profile A launches: %v", err)
	}
	if len(launches) != 1 || launches[0].RunID != "run_global" {
		t.Fatalf("profile A launches = %#v, want only run_global", launches)
	}
	otherLaunches, err := store.ListOpsHiveRunLaunches(t.Context(), "transpara-other", 10)
	if err != nil {
		t.Fatalf("list profile B launches: %v", err)
	}
	if len(otherLaunches) != 1 || otherLaunches[0].RunID != "run_other" {
		t.Fatalf("profile B launches = %#v, want only run_other", otherLaunches)
	}
}

func clearOpsHiveLaunchTables(t *testing.T, store *Store) {
	t.Helper()
	for _, stmt := range []string{
		`DELETE FROM ops_hive_run_launches`,
		`DELETE FROM ops_hive_intake_sources`,
	} {
		if _, err := store.db.ExecContext(t.Context(), stmt); err != nil {
			t.Fatalf("clear ops hive launch tables: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = store.db.ExecContext(t.Context(), `DELETE FROM ops_hive_run_launches`)
		_, _ = store.db.ExecContext(t.Context(), `DELETE FROM ops_hive_intake_sources`)
	})
}

func TestHandleOpsHiveIntakeRejectsMissingSourceContent(t *testing.T) {
	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/sources", strings.NewReader("source_kind=text"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("POST /ops/hive/intake/sources: status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "source content is required") {
		t.Fatalf("POST /ops/hive/intake/sources: body = %q", w.Body.String())
	}
}

func TestHandleOpsHiveIntakeRejectsOversizedSourceContent(t *testing.T) {
	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	values := url.Values{
		"source_kind": {"text"},
		"content":     {strings.Repeat("x", opsHiveIntakeMaxContentBytes+1)},
	}
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/sources", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("POST /ops/hive/intake/sources: status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "source content must be 20000 bytes or less") {
		t.Fatalf("POST /ops/hive/intake/sources: body = %q", w.Body.String())
	}
}

func TestHandleOpsHiveIntakeRendersBriefDefaultsOnSourceLoadError(t *testing.T) {
	h, store, _ := testHandlers(t)
	if err := store.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/hive/intake?profile=transpara", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/hive/intake: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"Could not load persisted intake sources.", "Factory brief draft", "Awaiting source material.", "No scoped sources yet.", "Readiness: intake pending"} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/hive/intake: body does not contain %q", want)
		}
	}
}

func TestOpsHiveIntakeSourceParamsTruncatesProvidedTitle(t *testing.T) {
	values := url.Values{
		"source_kind": {"text"},
		"title":       {strings.Repeat("A", 80)},
		"content":     {"requirements for the intake card"},
	}
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/sources", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	params, err := opsHiveIntakeSourceParamsFromForm(req)
	if err != nil {
		t.Fatalf("opsHiveIntakeSourceParamsFromForm: %v", err)
	}
	if len(params.Title) != 72 || !strings.HasSuffix(params.Title, "...") {
		t.Fatalf("Title = %q, want 72-char truncated title with ellipsis", params.Title)
	}
}

func TestOpsHiveIntakeSourceParamsTruncatesMultibyteTitleOnRuneBoundary(t *testing.T) {
	values := url.Values{
		"source_kind": {"text"},
		"title":       {strings.Repeat("界", 80)},
		"content":     {"requirements for the intake card"},
	}
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/hive/intake/sources", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	params, err := opsHiveIntakeSourceParamsFromForm(req)
	if err != nil {
		t.Fatalf("opsHiveIntakeSourceParamsFromForm: %v", err)
	}
	if !utf8.ValidString(params.Title) {
		t.Fatalf("Title is invalid UTF-8: %q", params.Title)
	}
	if utf8.RuneCountInString(params.Title) != 72 || !strings.HasSuffix(params.Title, "...") {
		t.Fatalf("Title = %q, want 72-rune truncated title with ellipsis", params.Title)
	}
}

func TestOpsHiveBriefPreviewDerivesSourcePriorityAndOverflow(t *testing.T) {
	sources := []OpsHiveSourceView{
		{Kind: "URL", Title: "reference", Content: "https://example.com/reference"},
		{Kind: "Repo", Title: "transpara-ai/site", Content: "transpara-ai/site"},
		{Kind: "Spec", Title: "API contract", Content: strings.Repeat("contract ", 60)},
		{Kind: "Plan", Title: "milestone plan", Content: "launch plan"},
		{Kind: "Text", Title: "operator notes", Content: "operator notes"},
		{Kind: "Repo", Title: "transpara-ai/hive", Content: "transpara-ai/hive"},
	}
	missing := []OpsHiveMissingFieldView{
		{Label: "Source material", Detail: "captured", Status: "ready"},
		{Label: "Budget cap", Detail: "Confirm max spend before run launch.", Status: "warning"},
	}

	brief := opsHiveBriefPreview(sources, missing, "draft ready", "full product pipeline")

	if brief.Title != "API contract" {
		t.Fatalf("Title = %q, want primary text-like source title", brief.Title)
	}
	if !strings.HasSuffix(brief.Objective, "...") || utf8.RuneCountInString(brief.Objective) > 260 {
		t.Fatalf("Objective = %q, want excerpt no longer than 260 runes with ellipsis", brief.Objective)
	}
	if !strings.Contains(brief.Scope, "Additional sources: 1") {
		t.Fatalf("Scope = %q, want additional-source count", brief.Scope)
	}
	if !strings.Contains(brief.Acceptance, "Resolve Budget cap") {
		t.Fatalf("Acceptance = %q, want unresolved missing field", brief.Acceptance)
	}
	if !strings.Contains(brief.Risks, "not runtime-start evidence") {
		t.Fatalf("Risks = %q, want runtime-evidence boundary", brief.Risks)
	}
	if brief.Readiness != "draft ready / full product pipeline" {
		t.Fatalf("Readiness = %q, want status and mode", brief.Readiness)
	}
}

func TestOpsHiveBriefExcerptHandlesSmallLimit(t *testing.T) {
	got := opsHiveBriefExcerpt("brief content", 3)
	if got != "brief content" {
		t.Fatalf("opsHiveBriefExcerpt = %q, want unmodified content for small limit", got)
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
					"brief_kind":"transpara_ai_github_issue_scan",
					"lifecycle_version":"civilization_issue_to_human_ready_pr_v0.3",
					"lifecycle_evidence_kind":"expected_lifecycle_not_runtime_progress",
					"development_lifecycle":[{
						"id":"research_issue_and_repo_context",
						"name":"Research issue and repository context",
						"required_roles":["strategist","planner"],
						"required_evidence":["selected_issue","repo_context"],
						"authority_boundary":"read_only_repo_research",
						"completion_gate":"issue_context_ready_for_debate",
						"evidence_status":"expected_not_observed"
					},{
						"id":"surface_ready_for_Human_result_PR",
						"name":"Surface ready-for-Human result PR",
						"required_roles":["strategist","reviewer","guardian"],
						"required_evidence":["ready_pr_url","ready_state_review","human_ready_summary"],
						"authority_boundary":"human_approval_required_no_merge",
						"completion_gate":"ready_pr_waiting_for_human_approval",
						"evidence_status":"expected_not_observed"
					}],
					"agent_execution_plan":[{
						"id":"10_implement_on_branch_implementer",
						"stage_id":"implement_on_branch",
						"role":"implementer",
						"can_operate":true,
						"objective":"Implement the selected approach on a branch.",
						"required_inputs":["selected_approach"],
						"required_outputs":["branch_commit","validation_output"],
						"authority_boundary":"branch_only_no_merge",
						"completion_gate":"implementation_ready_for_review",
						"evidence_status":"expected_not_observed"
					},{
						"id":"11_run_adversarial_review_reviewer",
						"stage_id":"run_adversarial_review",
						"role":"reviewer",
						"can_operate":false,
						"objective":"Run exact-head adversarial review.",
						"required_inputs":["draft_pr_url"],
						"required_outputs":["exact_head_review_artifact"],
						"authority_boundary":"review_only_no_merge",
						"completion_gate":"review_findings_disposed",
						"evidence_status":"expected_not_observed"
					}],
					"evidence_kind":"queued_request_not_runtime_start",
					"created_at":"2026-05-09T05:59:00Z"
				},
				"artifacts":[{
					"event_id":"event-artifact",
					"run_id":"run_123",
					"artifact_id":"artifact_factory_brief",
					"label":"factory_brief",
					"title":"Factory brief",
					"media_type":"text/markdown",
					"uri":"eventgraph://artifact/artifact_factory_brief",
					"summary":"brief ready for operator inspection",
					"producer_actor_id":"actor_builder",
					"causes":[{"event_id":"event-spawn","event_type":"hive.agent.spawned","scope":"run"}],
					"cause_status":"caused",
					"created_at":"2026-05-09T06:01:30Z"
				}],
				"run_events":[{
					"event_id":"event-artifact",
					"event_type":"factory.artifact.created",
					"conversation_id":"conv_runtime",
					"created_at":"2026-05-09T06:01:30Z",
					"causes":["event-spawn"],
					"inspector_kind":"curated_eventgraph_event",
					"content":{"artifact_id":"artifact_factory_brief","label":"factory_brief","media_type":"text/markdown"}
				},{
					"event_id":"event-request",
					"event_type":"factory.run.requested",
					"conversation_id":"conv_runtime",
					"created_at":"2026-05-09T05:59:00Z",
					"causes":["event-source"],
					"inspector_kind":"curated_eventgraph_event",
					"content":{"secret":"should-not-render"},
					"content_error":"content omitted: factory.run.requested is not in the runtime inspector allowlist"
				}],
				"causal_graph":{
					"scope":"latest_run_conversation",
					"conversation_id":"conv_runtime",
					"limit":50,
					"truncated":false,
					"nodes":[{"event_id":"event-spawn","event_type":"hive.agent.spawned","label":"Agent spawned: implementer/builder","scope":"run"},{"event_id":"event-artifact","event_type":"factory.artifact.created","label":"Artifact: Factory brief","artifact_id":"artifact_factory_brief","scope":"run"}],
					"edges":[{"from_event_id":"event-spawn","to_event_id":"event-artifact","scope":"run"}]
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
	queued := got.RuntimeEvidence.LastQueuedRunRequest
	if queued.BriefKind != "transpara_ai_github_issue_scan" || queued.LifecycleVersion != "civilization_issue_to_human_ready_pr_v0.3" || queued.LifecycleEvidenceKind != "expected_lifecycle_not_runtime_progress" {
		t.Fatalf("RuntimeEvidence queued lifecycle metadata = %#v", queued)
	}
	if len(queued.DevelopmentLifecycle) != 2 || queued.DevelopmentLifecycle[1].AuthorityBoundary != "human_approval_required_no_merge" {
		t.Fatalf("RuntimeEvidence queued lifecycle stages = %#v", queued.DevelopmentLifecycle)
	}
	if len(queued.AgentExecutionPlan) != 2 || !queued.AgentExecutionPlan[0].CanOperate || queued.AgentExecutionPlan[1].CanOperate {
		t.Fatalf("RuntimeEvidence queued agent execution plan = %#v", queued.AgentExecutionPlan)
	}
	if len(got.RuntimeEvidence.Artifacts) != 1 || got.RuntimeEvidence.Artifacts[0].ArtifactID != "artifact_factory_brief" || got.RuntimeEvidence.Artifacts[0].CauseStatus != "caused" {
		t.Fatalf("RuntimeEvidence artifacts = %#v", got.RuntimeEvidence.Artifacts)
	}
	if len(got.RuntimeEvidence.RunEvents) != 2 || got.RuntimeEvidence.RunEvents[1].ContentError == "" {
		t.Fatalf("RuntimeEvidence run events = %#v", got.RuntimeEvidence.RunEvents)
	}
	if got.RuntimeEvidence.CausalGraph.Scope != "latest_run_conversation" || len(got.RuntimeEvidence.CausalGraph.Edges) != 1 {
		t.Fatalf("RuntimeEvidence causal graph = %#v", got.RuntimeEvidence.CausalGraph)
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
				"status":"completed",
				"last_run":{"started_event_id":"event-run-start","completed_event_id":"event-run-complete","conversation_id":"conv_runtime","started_at":"2026-05-09T06:00:00Z","completed_at":"2026-05-09T06:03:00Z","seed_idea":"render runtime evidence","repo_path":"/repos/hive","agent_count":1,"duration_ms":120000,"total_cost":0},
				"agent_events":{
					"scope":"events_since_latest_hive.run.started",
					"spawned":2,
					"stopped":0,
					"observed_active":1,
					"active_agents":[{"name":"builder","role":"implementer","model":"claude-sonnet-4-6","actor_id":"actor-runtime-evidence-only","spawned_event_id":"event-spawn","spawned_at":"2026-05-09T06:01:00Z"}]
				},
				"last_queued_run_request":{
					"event_id":"event-request",
					"conversation_id":"conv_request",
					"run_id":"run_123",
					"title":"Queued run",
					"status":"queued",
					"target_repos":["transpara-ai/hive"],
					"authority_initial_level":"Required",
					"budget_max_iterations":0,
					"budget_max_cost_usd":0,
					"brief_kind":"transpara_ai_github_issue_scan",
					"lifecycle_version":"civilization_issue_to_human_ready_pr_v0.3",
					"lifecycle_evidence_kind":"expected_lifecycle_not_runtime_progress",
					"development_lifecycle":[{
						"id":"research_issue_and_repo_context",
						"name":"Research issue and repository context",
						"required_roles":["strategist","planner"],
						"required_evidence":["selected_issue","repo_context"],
						"authority_boundary":"read_only_repo_research",
						"completion_gate":"issue_context_ready_for_debate",
						"evidence_status":"expected_not_observed"
					},{
						"id":"surface_ready_for_Human_result_PR",
						"name":"Surface ready-for-Human result PR",
						"required_roles":["strategist","reviewer","guardian"],
						"required_evidence":["ready_pr_url","ready_state_review","human_ready_summary"],
						"authority_boundary":"human_approval_required_no_merge",
						"completion_gate":"ready_pr_waiting_for_human_approval",
						"evidence_status":"expected_not_observed"
					}],
					"agent_execution_plan":[{
						"id":"10_implement_on_branch_implementer",
						"stage_id":"implement_on_branch",
						"role":"implementer",
						"can_operate":true,
						"objective":"Implement the selected approach on a branch.",
						"required_inputs":["selected_approach"],
						"required_outputs":["branch_commit","validation_output"],
						"authority_boundary":"branch_only_no_merge",
						"completion_gate":"implementation_ready_for_review",
						"evidence_status":"expected_not_observed"
					},{
						"id":"11_run_adversarial_review_reviewer",
						"stage_id":"run_adversarial_review",
						"role":"reviewer",
						"can_operate":false,
						"objective":"Run exact-head adversarial review.",
						"required_inputs":["draft_pr_url"],
						"required_outputs":["exact_head_review_artifact"],
						"authority_boundary":"review_only_no_merge",
						"completion_gate":"review_findings_disposed",
						"evidence_status":"expected_not_observed"
					}],
					"evidence_kind":"queued_request_not_runtime_start",
					"created_at":"2026-05-09T05:59:00Z"
				},
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
	for _, want := range []string{"Authority projection", "Runtime evidence", "Queued launch intent", "queued_request_not_runtime_start", "runtime-start proof", "Issue-scan lifecycle", "expected_lifecycle_not_runtime_progress", "Research issue and repository context", "Surface ready-for-Human result PR", "human_approval_required_no_merge", "Agent execution plan", "implement_on_branch", "exact_head_review_artifact", "branch_only_no_merge", "review_only_no_merge", "operate", "review", "completed", "2026-05-09 06:03:00", "2.0m", "$0.00", "0 iter / $0.00", "actor-runtime-evidence-only", "Pending approvals", "Authority decisions", "Lifecycle state", "Key provenance", "Model selection", "startup-static", "guardian", "subscription", "agent.spawn.persistent", "builder", `action="/ops/hive/model-policy"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/hive: body does not contain %q", want)
		}
	}
	if strings.Contains(body, `data-authority-action`) {
		t.Fatal("GET /ops/hive exposes authority mutation controls")
	}
}

func TestHandleOpsHiveSuppressesLifecycleSectionWhenQueuedRequestHasNoLifecycle(t *testing.T) {
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
				"status":"queued",
				"last_queued_run_request":{
					"event_id":"event-request",
					"conversation_id":"conv_request",
					"run_id":"run_123",
					"title":"Queued run",
					"status":"queued",
					"target_repos":["transpara-ai/hive"],
					"authority_initial_level":"Required",
					"evidence_kind":"queued_request_not_runtime_start",
					"created_at":"2026-05-09T05:59:00Z"
				},
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
	if !strings.Contains(body, "Queued launch intent") || !strings.Contains(body, "queued_request_not_runtime_start") {
		t.Fatal("GET /ops/hive did not render queued request evidence")
	}
	if strings.Contains(body, "Issue-scan lifecycle") || strings.Contains(body, "Agent execution plan") {
		t.Fatal("GET /ops/hive rendered lifecycle section for queued request without lifecycle fields")
	}
	if strings.Contains(body, `data-authority-action`) {
		t.Fatal("GET /ops/hive exposes authority mutation controls")
	}
}

func TestHandleOpsApprovalsRendersProjectedQueue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hive/operator-projection" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"generated_at":"2026-05-09T12:00:00Z",
			"source":"eventgraph",
			"pending_approvals":[
				{"event_id":"ev-pr","request_id":"req-pr","requesting_actor":"actor-guardian","action_name":"pull_request.create","target":"transpara-ai/site codex/branch","environment":"pre-live","justification":"draft PR is ready","risk_summary":"bounded PR creation","created_at":"2026-05-09T11:00:00Z"},
				{"event_id":"ev-spawn","request_id":"req-spawn","requesting_actor":"actor-guardian","action_name":"agent.spawn.persistent","target":"builder","environment":"pre-live","justification":"persistent role trial","proposed_operation":"spawn builder","created_at":"2026-05-09T10:00:00Z"}
			],
			"authority_decisions":[{"event_id":"ev-decision","decision_id":"decision-1","request_id":"req-old","approver_actor":"actor-human","outcome":"approved","approved_action":"pull_request.create","approved_target":"transpara-ai/site codex/old","rationale":"evidence checked","created_at":"2026-05-09T09:00:00Z"}],
			"key_audit_traces":[{"event_id":"audit-key","event_type":"agent.key.registered","actor_id":"actor-builder","authority_request":"req-spawn","decision_event":"decision-1","created_at":"2026-05-09T08:00:00Z"}]
		}`))
	}))
	defer srv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)

	h := testOpsDecisionHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/approvals?profile=transpara", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/approvals: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"Authority approval queue",
		"Pending",
		"req-pr",
		"ev-pr",
		"actor-guardian",
		"Review decision",
		`/ops/decision?profile=transpara&amp;request_id=req-pr`,
		"req-spawn",
		"projected only",
		"Hive decision POST currently resolves draft-PR requests only",
		"Recent authority decisions",
		"decision-1",
		"evidence checked",
		"Guardian audit refs",
		"audit-key",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/approvals: body does not contain %q; body: %s", want, body)
		}
	}
	if got := strings.Count(body, "Review decision"); got != 1 {
		t.Fatalf("GET /ops/approvals: Review decision link count = %d, want 1", got)
	}
	for _, forbidden := range []string{"<form", "<button", `name="decision"`, `method="post"`, `action="/ops/approvals"`, `data-authority-action`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("GET /ops/approvals rendered mutation control %q", forbidden)
		}
	}
}

func TestHandleOpsApprovalsRendersPartialProjectionWarningWithQueue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"source":"eventgraph",
			"errors":["key audit projection timed out"],
			"pending_approvals":[{"event_id":"ev-pr","request_id":"req-pr","requesting_actor":"actor-guardian","action_name":"pull_request.create","target":"transpara-ai/site codex/branch","justification":"draft PR is ready"}],
			"authority_decisions":[],
			"key_audit_traces":[]
		}`))
	}))
	defer srv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)

	h := testOpsDecisionHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/approvals", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/approvals: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"key audit projection timed out",
		"req-pr",
		"Review decision",
		"Pending",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/approvals: body does not contain %q; body: %s", want, body)
		}
	}
}

func TestHandleOpsApprovalsProjectionErrorRendersWithoutQueue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "projection unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)

	h := testOpsDecisionHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/approvals", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/approvals: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "hive operator projection returned 503 Service Unavailable") {
		t.Fatalf("GET /ops/approvals: body does not contain projection error; body: %s", body)
	}
	for _, forbidden := range []string{"Pending approvals", "Review decision", "No pending authority requests projected"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("GET /ops/approvals: body contains queue content %q despite fatal projection error; body: %s", forbidden, body)
		}
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

func TestHandleOpsHiveRendersArtifactsGraphAndEventInspector(t *testing.T) {
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
				"status":"running",
				"last_run":{"started_event_id":"event-run-start","conversation_id":"conv_runtime","started_at":"2026-05-09T06:00:00Z","seed_idea":"artifact graph run"},
				"agent_events":{"scope":"events_since_latest_hive.run.started","spawned":1,"stopped":0,"observed_active":1},
				"artifacts":[{
					"event_id":"event-artifact",
					"run_id":"run_123",
					"artifact_id":"artifact_factory_brief",
					"label":"factory_brief",
					"title":"Factory brief",
					"media_type":"text/markdown",
					"uri":"eventgraph://artifact/artifact_factory_brief",
					"summary":"brief ready for operator inspection",
					"producer_actor_id":"actor_builder",
					"causes":[{"event_id":"event-spawn","event_type":"hive.agent.spawned","scope":"run"}],
					"cause_status":"caused",
					"created_at":"2026-05-09T06:01:30Z"
				}],
				"run_events":[{
					"event_id":"event-artifact",
					"event_type":"factory.artifact.created",
					"conversation_id":"conv_runtime",
					"created_at":"2026-05-09T06:01:30Z",
					"causes":["event-spawn"],
					"inspector_kind":"curated_eventgraph_event",
					"content":{"artifact_id":"artifact_factory_brief","label":"factory_brief","media_type":"text/markdown"}
				},{
					"event_id":"event-request",
					"event_type":"factory.run.requested",
					"conversation_id":"conv_runtime",
					"created_at":"2026-05-09T05:59:00Z",
					"causes":["event-source"],
					"inspector_kind":"curated_eventgraph_event",
					"content":{"secret":"should-not-render"},
					"content_error":"content omitted: factory.run.requested is not in the runtime inspector allowlist"
				}],
				"causal_graph":{
					"scope":"latest_run_conversation",
					"conversation_id":"conv_runtime",
					"limit":50,
					"truncated":true,
					"nodes":[{"event_id":"event-spawn","event_type":"hive.agent.spawned","label":"Agent spawned: implementer/builder","scope":"run"},{"event_id":"event-artifact","event_type":"factory.artifact.created","label":"Artifact: Factory brief","artifact_id":"artifact_factory_brief","scope":"run"}],
					"edges":[{"from_event_id":"event-spawn","to_event_id":"event-artifact","scope":"run"}]
				},
				"limitations":["artifact and causal graph projections are bounded by the operator projection limit"]
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
	for _, want := range []string{"Artifacts", "Factory brief", "Run-local causal graph", "Event inspector", "factory.artifact.created", "media_type", "content omitted: factory.run.requested", "event-spawn", "event-artifact", "truncated"} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/hive: body does not contain %q", want)
		}
	}
	if strings.Contains(body, `method="post"`) ||
		strings.Contains(body, `action="/ops/hive"`) ||
		strings.Contains(body, `data-authority-action`) {
		t.Fatal("GET /ops/hive artifact graph view exposes mutation controls")
	}
	if strings.Contains(body, "should-not-render") {
		t.Fatal("GET /ops/hive rendered event content despite content_error")
	}
}

func TestOpsHiveRuntimeEvidenceEmptyPreservesAgentMetadata(t *testing.T) {
	if opsHiveRuntimeEvidenceEmpty(OpsHiveRuntimeEvidence{
		AgentEvents: OpsHiveRuntimeAgentEvents{Scope: "events_since_latest_hive.run.started"},
	}) {
		t.Fatal("runtime evidence with agent-event scope was classified as empty")
	}
	if opsHiveRuntimeEvidenceEmpty(OpsHiveRuntimeEvidence{
		AgentEvents: OpsHiveRuntimeAgentEvents{LastAgentEventID: "event-agent"},
	}) {
		t.Fatal("runtime evidence with last agent event id was classified as empty")
	}
	if opsHiveRuntimeEvidenceEmpty(OpsHiveRuntimeEvidence{
		AgentEvents: OpsHiveRuntimeAgentEvents{LastAgentEventAt: "2026-05-09T06:01:00Z"},
	}) {
		t.Fatal("runtime evidence with last agent event time was classified as empty")
	}
	if opsHiveRuntimeEvidenceEmpty(OpsHiveRuntimeEvidence{
		Artifacts: []OpsHiveRuntimeArtifact{{ArtifactID: "artifact_1"}},
	}) {
		t.Fatal("runtime evidence with artifacts was classified as empty")
	}
	if opsHiveRuntimeEvidenceEmpty(OpsHiveRuntimeEvidence{
		RunEvents: []OpsHiveRuntimeEvent{{EventID: "event_1"}},
	}) {
		t.Fatal("runtime evidence with run events was classified as empty")
	}
	if opsHiveRuntimeEvidenceEmpty(OpsHiveRuntimeEvidence{
		CausalGraph: OpsHiveCausalGraph{Scope: "latest_run_conversation"},
	}) {
		t.Fatal("runtime evidence with causal graph was classified as empty")
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
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
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
	if gotMethod != http.MethodGet {
		t.Fatalf("projection request method = %q, want GET", gotMethod)
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

func TestHandleOpsPublicProofRendersDisplayOnlyEvidenceLedger(t *testing.T) {
	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/public-proof?profile=transpara", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/public-proof: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"Public Proof",
		`data-public-proof="display-only"`,
		"Public-reader and public-correction proof",
		"no fake green lights",
		"Operation-approved public-proof evidence reference displayed from static Site records",
		"reference_generated_at",
		"reference_fresh_until",
		"Operation public-proof packet",
		"OPERATION-PUBLIC-PROOF-REFERENCE-2026-06-29",
		"operation-approved-public-proof-reference",
		"operation-reference",
		"2026-06-29 08:10:49",
		"2026-07-06 08:10:49",
		"transpara-ai/operation/docs/operations/public-proof-evidence/public-proof-reference-2026-06-29.md",
		"https://github.com/transpara-ai/operation/blob/7ab3929ff88d0b75c48d53a80e92db74ec523482/docs/operations/public-proof-evidence/public-proof-reference-2026-06-29.md",
		"Site scope decision",
		"Deployed public URL evidence",
		"Live-reader proof",
		"Public-correction proof",
		"Telemetry precedent",
		"transpara-ai/site#191 scope comment",
		"https://github.com/transpara-ai/site/issues/191#issuecomment-4826247687",
		"transpara-ai/operation#45 pending",
		"docs/designs/telemetry-mission-control-design-v0.4.1.md",
		"unavailable",
		"stale",
		"fixture/local",
		"projection-only",
		"deployed-reference",
		"live-reader-proof",
		"public-correction-proof",
		"not live deployed public-reader proof or public-correction proof",
		"No deploy, private fetch, runtime execution, EventGraph write, Hive wake, Test 001 GREEN or closure",
		"operation#45 closure",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/public-proof: body does not contain %q", want)
		}
	}
	surfaceStart := strings.Index(body, `data-public-proof="display-only"`)
	if surfaceStart < 0 {
		t.Fatal("GET /ops/public-proof: missing display-only marker")
	}
	surface := body[surfaceStart:]
	if end := strings.Index(surface, "</main>"); end >= 0 {
		surface = surface[:end]
	}
	assertNoCivilizationMutationControls(t, surface)
}

func TestBuildOpsPublicProofDataMarksOperationPacketStaleAfterFreshUntil(t *testing.T) {
	fresh := buildOpsPublicProofData(time.Date(2026, 7, 6, 8, 10, 48, 0, time.UTC))
	freshRecord := findOpsPublicProofRecord(t, fresh, "Operation public-proof packet")
	if freshRecord.State != "projection-only" {
		t.Fatalf("fresh operation packet state = %q, want projection-only", freshRecord.State)
	}
	if !stringSliceContains(freshRecord.Labels, "operation-reference") {
		t.Fatalf("fresh operation packet labels = %#v, want operation-reference", freshRecord.Labels)
	}

	stale := buildOpsPublicProofData(time.Date(2026, 7, 6, 8, 10, 49, 0, time.UTC))
	staleRecord := findOpsPublicProofRecord(t, stale, "Operation public-proof packet")
	if staleRecord.State != "stale" {
		t.Fatalf("expired operation packet state = %q, want stale", staleRecord.State)
	}
	if !stringSliceContains(staleRecord.Labels, "stale") {
		t.Fatalf("expired operation packet labels = %#v, want stale", staleRecord.Labels)
	}
}

func TestBuildOpsPublicProofDataKeepsReaderAndCorrectionProofUnavailable(t *testing.T) {
	data := buildOpsPublicProofData(time.Date(2026, 6, 29, 8, 30, 0, 0, time.UTC))
	for _, category := range []string{"Live-reader proof", "Public-correction proof"} {
		record := findOpsPublicProofRecord(t, data, category)
		if record.State != "unavailable" {
			t.Fatalf("%s state = %q, want unavailable", category, record.State)
		}
		if !stringSliceContains(record.Labels, "unavailable") {
			t.Fatalf("%s labels = %#v, want unavailable", category, record.Labels)
		}
	}
}

func findOpsPublicProofRecord(t *testing.T, data *OpsPublicProofData, category string) OpsPublicProofRecord {
	t.Helper()
	for _, record := range data.Records {
		if record.Category == category {
			return record
		}
	}
	t.Fatalf("missing public proof record category %q", category)
	return OpsPublicProofRecord{}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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
	if len(data.Actions) != 0 {
		t.Fatalf("Actions = %d, want 0 for blocked non-draft-PR request", len(data.Actions))
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

func TestOpsDecisionSubmitRequiresReasonForApproveDeny(t *testing.T) {
	hivePostHits := 0
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			hivePostHits++
			t.Fatal("hive POST /api/hive/operator-decision must not be called without a decision reason")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"source":"test","pending_approvals":[]}`))
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := testOpsDecisionHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	for _, decision := range []string{opsDecisionApprove, opsDecisionDeny} {
		t.Run(decision, func(t *testing.T) {
			form := strings.NewReader("request_id=req-civic-roles&decision=" + decision)
			req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/decision", form)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if hivePostHits != 0 {
				t.Fatal("hive POST was hit")
			}
			if w.Code >= 400 {
				t.Fatalf("site returned %d", w.Code)
			}
			body := w.Body.String()
			for _, want := range []string{
				"Governance POST failed: decision_reason is required for approve/deny",
				"effect = none",
				"decision_reason is required for approve/deny",
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("response body does not contain %q; body: %s", want, body)
				}
			}
		})
	}
}

func TestOpsDecisionSubmitIgnoresPostedApprover(t *testing.T) {
	var gotBody string
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			bs, _ := io.ReadAll(r.Body)
			gotBody = string(bs)
		}
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"source":"test","pending_approvals":[]}`))
		}
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := testOpsDecisionHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	form := strings.NewReader("request_id=req-civic-roles&decision=" + opsDecisionApprove + "&approver=actor_attacker&reason=reviewed")
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/decision", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if gotBody == "" {
		t.Fatal("hive POST body is empty, want forwarded decision")
	}
	if strings.Contains(gotBody, "actor_attacker") || strings.Contains(gotBody, "approver") {
		t.Fatalf("forwarded body must not include operator-supplied approver, got %q", gotBody)
	}
	if !strings.Contains(gotBody, `"reason":"reviewed"`) {
		t.Fatalf("forwarded body = %q, want reason", gotBody)
	}
	if w.Code >= 400 {
		t.Fatalf("site returned %d", w.Code)
	}
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

func TestOpsDecisionRequestMoreEvidenceBypassesReasonRequiredInHTML(t *testing.T) {
	h := testOpsDecisionHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/decision?action=request-more-evidence&target_ref=pr://transpara-ai/site/82&reason=operator+review", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/decision: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	moreEvidenceValue := `value="` + opsDecisionMoreEvidence + `"`
	moreEvidencePos := strings.Index(body, moreEvidenceValue)
	if moreEvidencePos < 0 {
		t.Fatalf("GET /ops/decision: body does not contain %s; body: %s", moreEvidenceValue, body)
	}
	buttonEnd := strings.Index(body[moreEvidencePos:], ">")
	if buttonEnd < 0 {
		t.Fatalf("GET /ops/decision: more-evidence button is malformed; body: %s", body)
	}
	buttonMarkup := body[moreEvidencePos : moreEvidencePos+buttonEnd]
	if !strings.Contains(buttonMarkup, "formnovalidate") {
		t.Fatalf("request-more-evidence button markup = %q, want formnovalidate", buttonMarkup)
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
