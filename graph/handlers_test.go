package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/transpara-ai/site/auth"
)

// testHandlers creates handlers with a test user in context.
func testHandlers(t *testing.T) (*Handlers, *Store, func(http.HandlerFunc) http.Handler) {
	t.Helper()
	_, store := testDB(t)

	// Auth wrapper that injects a test user.
	testUser := &auth.User{ID: "test-user-1", Name: "Tester", Email: "test@test.com", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.ContextWithUser(r.Context(), testUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	h := NewHandlers(store, wrap, wrap)
	return h, store, wrap
}

func TestOperatorAndHiveOpsRoutesRequireWriteAuth(t *testing.T) {
	requireAuth := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "auth required", http.StatusUnauthorized)
		})
	}
	optionalAuth := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatalf("route %s used optional auth wrapper; want required auth", r.URL.Path)
		})
	}
	h := NewHandlers(nil, optionalAuth, requireAuth)
	mux := http.NewServeMux()
	h.Register(mux)

	for _, path := range []string{
		"/ops",
		"/ops/work",
		"/ops/telemetry",
		"/ops/observatory",
		"/ops/observatory/events",
		"/ops/civilization",
		"/ops/github-canonical",
		"/ops/hive",
		"/ops/evidence",
		"/ops/decision",
		"/ops/refinery",
		"/api/hive/site-ops?space=hive",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusUnauthorized, w.Body.String())
			}
		})
	}
}

func TestHandleOpsCivilizationRendersReadOnlyAssembly(t *testing.T) {
	h, _, _ := testHandlers(t)
	t.Setenv("HIVE_OPS_API_BASE_URL", "")

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/civilization", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/civilization: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"Civilization Assembly",
		`data-civilization-assembly="read-only"`,
		"docs v4.0 Event 10 Site Civilization projection-consumer AuthorityDecision",
		"EventGraph Civilization Assembly projection unavailable to Site",
		"EventGraph Civilization Assembly projection",
		"Projection facts",
		"Issue intake projection",
		`data-civilization-issue-intake="read-only"`,
		"No issue intake records are projected.",
		"No-write limit",
		"cannot edit GitHub issues, start Hive, write EventGraph, queue runs, create PRs, merge, deploy, or approve protected actions",
		"Issue readiness",
		`data-civilization-issue-readiness="read-only"`,
		"recommendations are not PR, merge, deploy, or authority approval",
		"PR-Ready-When",
		"cc:intake",
		"cc:pr-deferred",
		"cc:aggregate-candidate",
		"recommendation-only",
		"cc:civilization-presence",
		"cc:protected-action",
		"FactoryOrder evidence",
		"No Work FactoryOrder seed tasks are projected.",
		"derivation status",
		"Registered method",
		"GET only",
		"No mutation handler is registered for this page.",
		"EventGraph projection-shaped input was not available",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/civilization: body does not contain %q", want)
		}
	}
	assertNoCivilizationMutationControls(t, civilizationAssemblySurface(t, body))
}

func TestHandleOpsGitHubCanonicalRendersReadOnlyMigrationSurface(t *testing.T) {
	calledHive := false
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledHive = true
		http.Error(w, "unexpected hive call", http.StatusTeapot)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := NewHandlers(nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterReadOnlyOps(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/github-canonical?profile=transpara", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/github-canonical: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if calledHive {
		t.Fatal("GET /ops/github-canonical called Hive; want static read-only Site fixture")
	}
	body := w.Body.String()
	for _, want := range []string{
		"GitHub-canonical migration",
		`data-github-canonical-migration="read-only"`,
		"transpara-ai/docs#197",
		"transpara-ai/work#61",
		"transpara-ai/work#62",
		"transpara-ai/work#63",
		"transpara-ai/site#127",
		"transpara-ai/site#129",
		"transpara-ai/platform#5",
		"transpara-ai/.github#3",
		"transpara-ai/eventgraph#63",
		"transpara-ai/eventgraph#62",
		"transpara-ai/eventgraph#59",
		"transpara-ai/eventgraph#61",
		"transpara-ai/hive#220",
		"transpara-ai/hive#221",
		"transpara-ai/hive#222",
		"transpara-ai/hive#223",
		"transpara-ai/operation#34",
		"completed",
		"deferred",
		"needs-human-scope",
		"protected-action",
		"legacy-evidence-only",
		"legacy-markdown",
		"Typed evidence records",
		`data-github-canonical-evidence-records="read-only"`,
		"evidence.testrun.recorded",
		"evidence.gateresult.recorded",
		"evidence.auditreport.recorded",
		"tests.pass",
		"gate.partial",
		"closeout.blocked",
		"source_issue_refs",
		"validation_refs",
		"cfar_refs",
		"authority_boundary_refs",
		"residual_risk_refs",
		"provenance_refs",
		"trace_score_basis_points",
		"arc_issue_scan:Findings=0",
		"merge:c6f261a27a193a470a9e287d15580a05d1b0fafc",
		"merge:523181b83ad8540fba747a64a12975996db170a4",
		"merge:0c0fdb5f9c116cef99ed87ed9f31bfc5cbd9e10e",
		"merge:326f90a49d986e66d171e0eb0b5be23b8e64324c",
		"merge:c3dc3a63eb16eafed490b7e6be28affe3469f7ea",
		"work#71 merge:f118276665c0bbbea282be7803070948b8d8e297",
		"https://github.com/transpara-ai/docs/issues/197#issuecomment-4803515276",
		"https://github.com/transpara-ai/site/issues/129",
		"https://github.com/transpara-ai/site/issues/131",
		"https://github.com/transpara-ai/site/issues/133",
		"https://github.com/transpara-ai/operation/pull/37",
		"operation PR #37 CFAR PASS",
		"Operation continuity",
		"authority_recommendation_policy",
		"human_required_classification_policy",
		"role_separation_policy",
		"autonomy_guard_policy",
		"projection-store truth and production write path still not authorized",
		"production write path still human-scope blocked",
		"effect = none",
		"No Hive wake",
		"GitHub Issues are the live source-of-intent target",
		"no live GitHub fetch or mutation",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/github-canonical: body does not contain %q", want)
		}
	}
	for _, stale := range []string{
		"operator continuity procedure not updated for GitHub-canonical cutover",
		"runbook scope handoff evidence pending",
		"remaining EventGraph/docs/operation protected lanes open",
	} {
		if strings.Contains(body, stale) {
			t.Fatalf("GET /ops/github-canonical: body contains stale operation blocker %q", stale)
		}
	}
	assertNoCivilizationMutationControls(t, githubCanonicalSurface(t, body))
}

func TestHandleOpsCivilizationConsumesHiveProjection(t *testing.T) {
	h, _, _ := testHandlers(t)
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hive/civilization/assembly-projection" {
			t.Fatalf("path = %q, want civilization assembly projection", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("authorization header = %q, want bearer secret", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, hiveCivilizationAssemblyProjectionFixture)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)
	t.Setenv("HIVE_OPS_API_KEY", "secret")

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/civilization", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/civilization: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"civ-runtime-001",
		"actor_builder",
		"implementer",
		"runtime projection from Hive",
		"FactoryOrder evidence",
		"task rows can include runtime-evidence and completed-stage signals without merge or deployment authority",
		"Issue readiness",
		`data-civilization-issue-readiness="read-only"`,
		"PR-ready state",
		"pending: Debate with correct civic roles",
		"PR-Ready-When",
		"First pending gate",
		"Debate with correct civic roles",
		"Recommendation state",
		"recommendation-only rank 1 of 3 by scanner_order_first_candidate_v0.1",
		"Grouping recommendation",
		"Grouping remains advisory until the matching repo, touched substrate, risk class, acceptance path, and PR-readiness condition are verified.",
		"Issue intake projection",
		`data-civilization-issue-intake="read-only"`,
		"2 issue(s), 2 advisory group(s) projected.",
		"transpara-ai/site#114",
		"transpara-ai/site#116",
		"https://github.com/transpara-ai/site/issues/114",
		"https://github.com/transpara-ai/site/issues/116",
		"github:transpara-ai/site#114",
		"github:transpara-ai/site#116",
		"site operator ui projection",
		"site civilization display data source",
		"no substrate inputs projected",
		"inputs: none projected",
		"no readiness inputs projected",
		"scanner boundary metadata not projected",
		"do not group until protected-action scope is explicitly authorized",
		"protected-action issue requires separate authority scope before grouped implementation",
		"No-write limit",
		"cc:intake",
		"Durable source-of-intent and scope evidence",
		"cc:pr-deferred",
		"cc:aggregate-candidate",
		"cc:civilization-presence",
		"cc:protected-action",
		"Issue-scan Kanban",
		`data-civilization-issue-scan-kanban="read-only"`,
		"3 run(s), 3 stage(s), 4 blocker(s), 1 lineage record(s) projected.",
		"run_docs_172",
		"run_site_115",
		"run_docs_172_scope",
		"transpara-ai/docs#172",
		"transpara-ai/site#115",
		"Parked",
		"Human Action",
		"Blocked",
		"run_adversarial_review",
		"surface_ready_for_human_result_pr",
		"research_issue_and_repo_context",
		"tsk_docs_172_run_adversarial_review",
		"agent_reviewer",
		"agent_blocker_repair",
		"agent_guardian",
		"duplicate chain",
		"needs human scope",
		"protected action",
		"stale target",
		"collapse duplicate canonical stage chain",
		"human must clarify issue scope before runtime continues",
		"human must authorize protected repo action",
		"confirm target issue is still live",
		"lineage",
		"duplicates",
		"fo_run_issue_scan_001",
		"work_task_seeded",
		"human_required_before_merge",
		"evt_work_task_001",
		"evt_work_task_stage_research_001",
		"Issue-scan stage: Research issue and repo context",
		"tsk_run_issue_scan_001_research_issue_and_repo_context",
		"repo_context_packet",
		"strategist, planner",
		"evt_work_task_stage_role_contract_research_001",
		"evt_work_task_stage_output_contract_research_001",
		"issue_snapshot, repo_context, risk_and_scope_notes",
		"stage step completed runtime evidence recorded",
		"issue_priority_rationale",
		"evt_work_task_stage_debate_001",
		"Issue-scan stage: Debate with correct civic roles",
		"evt_work_dependency_debate_after_research_001",
		"work task seeded",
		"evt_work_task_artifact_001",
		"issue_scan_execution_plan",
		"evt_work_task_stage_artifact_001",
		"issue_scan_lifecycle_stage_research_issue_and_repo_context",
		"declared pending runtime evidence",
		"runtime=stage completed runtime evidence recorded",
		"artifact_runtime_research_001",
		"stage declared pending runtime evidence",
		"application/json",
		`data-civilization-wide-table="factory-orders"`,
		`data-civilization-wide-table="work-tasks"`,
		`data-civilization-wide-table="work-artifacts"`,
		`data-civilization-wide-table="issue-scan-stage-evidence"`,
		`data-civilization-wide-table="issue-scan-kanban"`,
		`data-civilization-wide-table="role-topology"`,
		`w-full min-w-[72rem]`,
		`w-full min-w-[96rem]`,
		`w-full min-w-[64rem]`,
		`w-full min-w-[84rem]`,
		"test_run_001",
		"gate_result_001",
		"audit_report_001",
		"Queued issue-scan lifecycle",
		"not runtime completion proof",
		"expected evidence, not runtime progress",
		"expected not observed",
		"Issue selection",
		"scanner_order_first_candidate_v0.1",
		"scanner_return_order",
		"Resolve transpara-ai/hive#321",
		"research_issue_and_repo_context",
		"debate_with_correct_civic_roles",
		"select_and_design_approach",
		"implement_on_branch",
		"run_adversarial_review",
		"drive_blockers_to_zero",
		"surface_ready_for_Human_result_PR",
		"human_approval_boundary_check",
		"EventGraph Civilization Assembly projection civ-runtime-001",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/civilization: body does not contain %q", want)
		}
	}
	if strings.Contains(body, "EventGraph projection-shaped input was not available") {
		t.Fatal("GET /ops/civilization rendered unavailable fallback despite Hive projection")
	}
	assertNoCivilizationMutationControls(t, civilizationAssemblySurface(t, body))
}

func TestOpsCivilizationEvidenceStatusValue(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   string
		fallback string
		want     string
	}{
		{
			name:     "fallback for empty",
			status:   "",
			fallback: "expected",
			want:     "expected",
		},
		{
			name:     "fallback for whitespace",
			status:   " \t\n ",
			fallback: "not projected",
			want:     "not projected",
		},
		{
			name:     "trims and replaces underscores",
			status:   "  declared_pending_runtime_evidence  ",
			fallback: "expected",
			want:     "declared pending runtime evidence",
		},
		{
			name:     "replaces every underscore",
			status:   "stage_completed_runtime_evidence_recorded",
			fallback: "expected",
			want:     "stage completed runtime evidence recorded",
		},
		{
			name:     "keeps spaced value",
			status:   "expected not observed",
			fallback: "expected",
			want:     "expected not observed",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := opsCivilizationEvidenceStatusValue(tc.status, tc.fallback); got != tc.want {
				t.Fatalf("opsCivilizationEvidenceStatusValue(%q, %q) = %q, want %q", tc.status, tc.fallback, got, tc.want)
			}
		})
	}
}

func TestOpsCivilizationEvidenceObserved(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   bool
	}{
		{status: "", want: false},
		{status: "unavailable", want: false},
		{status: "not_available", want: false},
		{status: "not available", want: false},
		{status: "incomplete", want: false},
		{status: "not_complete", want: false},
		{status: "not_completed", want: false},
		{status: "not_green", want: false},
		{status: "not_recorded", want: false},
		{status: "not passed", want: false},
		{status: "nonzero_blockers", want: false},
		{status: "declared_pending_runtime_evidence", want: false},
		{status: "expected_not_observed", want: false},
		{status: "stage_completed_runtime_evidence_recorded", want: true},
		{status: "stage completed runtime evidence recorded", want: true},
		{status: "available", want: true},
		{status: "passed", want: true},
	} {
		t.Run(tc.status, func(t *testing.T) {
			if got := opsCivilizationEvidenceObserved(tc.status); got != tc.want {
				t.Fatalf("opsCivilizationEvidenceObserved(%q) = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}

func TestOpsCivilizationIssueReadinessBranches(t *testing.T) {
	selection := &OpsHiveQueuedRunSelectionPolicy{
		PolicyID:       "scanner_order_first_candidate_v0.1",
		SelectedRank:   1,
		CandidateCount: 2,
		RankingInputs:  []string{"repo_scan_order"},
		Rationale:      "Select the first validated candidate.",
	}

	t.Run("unavailable lifecycle and runtime evidence do not advance readiness", func(t *testing.T) {
		projection := &OpsCivilizationAssemblyProjection{
			QueuedRunRequest: &OpsHiveQueuedRunRequest{
				RunID:           "run_unavailable",
				SelectionPolicy: selection,
				DevelopmentLifecycle: []OpsHiveQueuedRunLifecycleStage{
					{ID: "research", Name: "Research", EvidenceStatus: "expected_not_observed"},
				},
			},
			WorkEvidenceSummary: OpsCivilizationAssemblyWorkEvidence{
				Tasks: []OpsCivilizationAssemblyTaskEvidence{
					{LifecycleStageID: "research", RuntimeEvidenceStatus: "unavailable"},
				},
			},
		}
		data := buildOpsCivilizationAssemblyDataFromProjection(projection, time.Now().UTC())
		if data.IssueReadiness.Status != "pending: Research" || data.IssueReadiness.FirstPendingStage != "Research" {
			t.Fatalf("issue readiness = %+v, want pending Research", data.IssueReadiness)
		}
	})

	t.Run("runtime evidence can complete a pending lifecycle stage", func(t *testing.T) {
		projection := &OpsCivilizationAssemblyProjection{
			QueuedRunRequest: &OpsHiveQueuedRunRequest{
				RunID:           "run_runtime_complete",
				SelectionPolicy: selection,
				DevelopmentLifecycle: []OpsHiveQueuedRunLifecycleStage{
					{ID: "research", Name: "Research", EvidenceStatus: "expected_not_observed"},
				},
			},
			WorkEvidenceSummary: OpsCivilizationAssemblyWorkEvidence{
				Tasks: []OpsCivilizationAssemblyTaskEvidence{
					{LifecycleStageID: "research", RuntimeEvidenceStatus: "stage_completed_runtime_evidence_recorded"},
				},
			},
		}
		data := buildOpsCivilizationAssemblyDataFromProjection(projection, time.Now().UTC())
		if data.IssueReadiness.Status != "ready-for-Human PR evidence projected" || data.IssueReadiness.FirstPendingStage != "none" {
			t.Fatalf("issue readiness = %+v, want ready from runtime evidence", data.IssueReadiness)
		}
	})

	t.Run("all observed stages are ready", func(t *testing.T) {
		projection := &OpsCivilizationAssemblyProjection{
			QueuedRunRequest: &OpsHiveQueuedRunRequest{
				RunID:           "run_ready",
				SelectionPolicy: selection,
				DevelopmentLifecycle: []OpsHiveQueuedRunLifecycleStage{
					{ID: "research", Name: "Research", EvidenceStatus: "stage_completed_runtime_evidence_recorded"},
					{ID: "review", Name: "Review", EvidenceStatus: "stage_completed_runtime_evidence_recorded"},
				},
			},
		}
		data := buildOpsCivilizationAssemblyDataFromProjection(projection, time.Now().UTC())
		if data.IssueReadiness.Status != "ready-for-Human PR evidence projected" || data.IssueReadiness.FirstPendingStage != "none" {
			t.Fatalf("issue readiness = %+v, want ready", data.IssueReadiness)
		}
	})

	t.Run("queued run without selection policy remains recommendation incomplete", func(t *testing.T) {
		projection := &OpsCivilizationAssemblyProjection{
			QueuedRunRequest: &OpsHiveQueuedRunRequest{
				RunID: "run_no_policy",
				DevelopmentLifecycle: []OpsHiveQueuedRunLifecycleStage{
					{ID: "research", Name: "Research", EvidenceStatus: "stage_completed_runtime_evidence_recorded"},
				},
			},
		}
		data := buildOpsCivilizationAssemblyDataFromProjection(projection, time.Now().UTC())
		if data.IssueReadiness.RecommendationState != "queued issue-scan projected without a selection policy" {
			t.Fatalf("recommendation state = %q", data.IssueReadiness.RecommendationState)
		}
		if data.IssueReadiness.Status != "ready-for-Human PR evidence projected" {
			t.Fatalf("issue readiness = %+v, want ready from observed lifecycle", data.IssueReadiness)
		}
	})

	t.Run("queued run without lifecycle is not projected", func(t *testing.T) {
		projection := &OpsCivilizationAssemblyProjection{
			QueuedRunRequest: &OpsHiveQueuedRunRequest{
				RunID:           "run_no_lifecycle",
				SelectionPolicy: selection,
			},
		}
		data := buildOpsCivilizationAssemblyDataFromProjection(projection, time.Now().UTC())
		if data.IssueReadiness.Status != "not projected" || data.IssueReadiness.FirstPendingStage != "lifecycle stages not projected" {
			t.Fatalf("issue readiness = %+v, want lifecycle not projected", data.IssueReadiness)
		}
	})
}

func TestOpsCivilizationIssueScanKanbanDocs172Site115TypedProjection(t *testing.T) {
	projection := &OpsCivilizationAssemblyProjection{
		IssueScanProjection: OpsCivilizationIssueScanProjection{
			Runs: []OpsCivilizationIssueScanRunProjected{
				{
					RunID:            "run_docs_172",
					FactoryOrderID:   "fo_docs_172",
					LifecycleVersion: "civilization_issue_to_human_ready_pr_v0.4",
					State:            "parked",
					TargetIssue:      OpsCivilizationIssueRef{Repo: "transpara-ai/docs", Number: 172, State: "open"},
					SelectedIssue:    OpsCivilizationIssueRef{Repo: "transpara-ai/docs", Number: 172, State: "open"},
					CandidateIssues: []OpsCivilizationIssueRef{
						{Repo: "transpara-ai/docs", Number: 172},
						{Repo: "transpara-ai/site", Number: 115},
					},
				},
				{
					RunID:          "run_site_115",
					FactoryOrderID: "fo_site_115",
					State:          "human_action",
					TargetIssue:    OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 115, State: "open"},
					SelectedIssue:  OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 115, State: "open"},
				},
				{
					RunID:          "run_docs_172_scope",
					FactoryOrderID: "fo_docs_172_scope",
					State:          "human_action",
					TargetIssue:    OpsCivilizationIssueRef{Repo: "transpara-ai/docs", Number: 172, State: "open", Labels: []string{"cc:needs-human-scope"}},
					SelectedIssue:  OpsCivilizationIssueRef{Repo: "transpara-ai/docs", Number: 172, State: "open", Labels: []string{"cc:needs-human-scope"}},
				},
			},
			Stages: []OpsCivilizationIssueScanStageProjected{
				{
					RunID:             "run_docs_172",
					FactoryOrderID:    "fo_docs_172",
					StageID:           "run_adversarial_review",
					StageNumber:       5,
					StageCount:        7,
					CanonicalTaskID:   "tsk_docs_172_run_adversarial_review",
					TaskID:            "task-docs-review",
					CurrentState:      "parked",
					CompletionGate:    "exact-head adversarial review returns zero blockers",
					AuthorityBoundary: "review only; no merge or deploy",
					AssignedAgentIDs:  []string{"agent_reviewer"},
					TouchingAgentIDs:  []string{"agent_blocker_repair", "agent_reviewer"},
				},
				{
					RunID:             "run_site_115",
					FactoryOrderID:    "fo_site_115",
					StageID:           "surface_ready_for_human_result_pr",
					StageNumber:       7,
					StageCount:        7,
					CanonicalTaskID:   "tsk_site_115_surface_ready_for_human_result_pr",
					TaskID:            "task-site-surface",
					CurrentState:      "human_action",
					CompletionGate:    "ready PR waits for human approval",
					AuthorityBoundary: "human approval required; no merge",
					AssignedAgentIDs:  []string{"agent_guardian"},
					TouchingAgentIDs:  []string{"agent_guardian"},
				},
				{
					RunID:             "run_docs_172_scope",
					FactoryOrderID:    "fo_docs_172_scope",
					StageID:           "select_and_design_approach",
					StageNumber:       3,
					StageCount:        7,
					CanonicalTaskID:   "tsk_docs_172_scope_select_and_design_approach",
					TaskID:            "task-docs-scope",
					CurrentState:      "human_action",
					CompletionGate:    "human scope decision recorded",
					AuthorityBoundary: "human scope clarification required",
					AssignedAgentIDs:  []string{"agent_guardian"},
					TouchingAgentIDs:  []string{"agent_guardian"},
				},
			},
			Blockers: []OpsCivilizationIssueScanBlockerProjected{
				{RunID: "run_docs_172", StageID: "run_adversarial_review", BlockerType: "duplicate_chain", RequiredAction: "collapse duplicate canonical stage chain"},
				{RunID: "run_site_115", StageID: "surface_ready_for_human_result_pr", BlockerType: "protected_action", RequiredAction: "human must authorize protected repo action"},
				{RunID: "run_site_115", StageID: "research_issue_and_repo_context", BlockerType: "stale_target", RequiredAction: "confirm target issue is still live"},
				{RunID: "run_docs_172_scope", StageID: "select_and_design_approach", BlockerType: "needs_human_scope", RequiredAction: "human must clarify issue scope before runtime continues"},
			},
			Lineage: []OpsCivilizationIssueScanLineageProjected{
				{
					RunID:            "run_docs_172",
					StageID:          "run_adversarial_review",
					CanonicalTaskID:  "tsk_docs_172_run_adversarial_review",
					PrimaryTaskID:    "task-docs-review",
					TaskIDs:          []string{"task-docs-review", "task-docs-review-duplicate"},
					DuplicateTaskIDs: []string{"task-docs-review-duplicate"},
					DuplicateOf:      "task-docs-review",
				},
			},
		},
	}

	data := buildOpsCivilizationAssemblyDataFromProjection(projection, time.Now().UTC())
	if data.IssueScanKanban.Status != opsCivilizationFieldAvailable {
		t.Fatalf("kanban status = %q, want available", data.IssueScanKanban.Status)
	}
	if len(data.IssueScanKanban.Columns) != 3 {
		t.Fatalf("kanban columns = %+v, want blocked, parked, human_action", data.IssueScanKanban.Columns)
	}
	docsCard := issueScanKanbanCardByStage(data.IssueScanKanban, "run_docs_172", "run_adversarial_review")
	if docsCard == nil {
		t.Fatalf("missing docs#172 review card: %+v", data.IssueScanKanban)
	}
	if docsCard.CurrentState != "parked" || len(docsCard.Blockers) != 1 || docsCard.Blockers[0].BlockerType != "duplicate_chain" {
		t.Fatalf("docs#172 card state/blockers = %+v", docsCard)
	}
	if !docsCard.HasLineage || !sliceContains(docsCard.Lineage.DuplicateTaskIDs, "task-docs-review-duplicate") {
		t.Fatalf("docs#172 duplicate lineage missing: %+v", docsCard)
	}
	if !sliceContains(docsCard.AssignedAgentIDs, "agent_reviewer") || !sliceContains(docsCard.TouchingAgentIDs, "agent_blocker_repair") {
		t.Fatalf("docs#172 agent touch state missing: %+v", docsCard)
	}
	if docsCard.SelectedIssue.Repo != "transpara-ai/docs" || docsCard.SelectedIssue.Number != 172 || len(docsCard.CandidateIssues) != 2 {
		t.Fatalf("docs#172 issue refs not typed: %+v", docsCard)
	}

	siteCard := issueScanKanbanCardByStage(data.IssueScanKanban, "run_site_115", "surface_ready_for_human_result_pr")
	if siteCard == nil || siteCard.CurrentState != "human_action" || len(siteCard.Blockers) != 1 || siteCard.Blockers[0].BlockerType != "protected_action" {
		t.Fatalf("site#115 human-action card = %+v", siteCard)
	}
	staleCard := issueScanKanbanCardByStage(data.IssueScanKanban, "run_site_115", "research_issue_and_repo_context")
	if staleCard == nil || staleCard.CurrentState != "blocked" || staleCard.Blockers[0].BlockerType != "stale_target" {
		t.Fatalf("site#115 stale-target blocker card = %+v", staleCard)
	}
	scopeCard := issueScanKanbanCardByStage(data.IssueScanKanban, "run_docs_172_scope", "select_and_design_approach")
	if scopeCard == nil || scopeCard.CurrentState != "human_action" || len(scopeCard.Blockers) != 1 || scopeCard.Blockers[0].BlockerType != "needs_human_scope" {
		t.Fatalf("docs#172 needs-human-scope card = %+v", scopeCard)
	}
}

func TestOpsCivilizationIssueScanKanbanRunLevelBlockerAttachesWithoutOrphan(t *testing.T) {
	projection := &OpsCivilizationAssemblyProjection{
		IssueScanProjection: OpsCivilizationIssueScanProjection{
			Runs: []OpsCivilizationIssueScanRunProjected{
				{
					RunID:         "run_docs_172",
					TargetIssue:   OpsCivilizationIssueRef{Repo: "transpara-ai/docs", Number: 172, State: "open"},
					SelectedIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/docs", Number: 172, State: "open"},
				},
			},
			Stages: []OpsCivilizationIssueScanStageProjected{
				{
					RunID:           "run_docs_172",
					StageID:         "run_adversarial_review",
					StageNumber:     5,
					CanonicalTaskID: "tsk_docs_172_run_adversarial_review",
				},
			},
			Blockers: []OpsCivilizationIssueScanBlockerProjected{
				{RunID: "run_docs_172", BlockerType: "needs_human_scope", RequiredAction: "human must clarify issue scope before runtime continues"},
			},
		},
	}

	data := buildOpsCivilizationAssemblyDataFromProjection(projection, time.Now().UTC())
	if got := issueScanKanbanCardCount(data.IssueScanKanban); got != 1 {
		t.Fatalf("kanban card count = %d, want exactly one stage card: %+v", got, data.IssueScanKanban)
	}
	card := issueScanKanbanCardByStage(data.IssueScanKanban, "run_docs_172", "run_adversarial_review")
	if card == nil || card.CurrentState != "blocked" || len(card.Blockers) != 1 || card.Blockers[0].BlockerType != "needs_human_scope" {
		t.Fatalf("stage card did not absorb run-level blocker: %+v", card)
	}
	if orphan := issueScanKanbanCardByStage(data.IssueScanKanban, "run_docs_172", ""); orphan != nil {
		t.Fatalf("run-level blocker rendered an orphan card: %+v", orphan)
	}
}

func TestSortIssueScanCardsHasDeterministicTiebreakers(t *testing.T) {
	cards := []OpsCivilizationIssueScanKanbanCard{
		{RunID: "run", StageNumber: 1, StageID: "stage", CanonicalTaskID: "canonical", FactoryOrderID: "factory-b", TaskID: "task-b"},
		{RunID: "run", StageNumber: 1, StageID: "stage", CanonicalTaskID: "canonical", FactoryOrderID: "factory-a", TaskID: "task-a"},
	}

	sortIssueScanCards(cards)

	if cards[0].FactoryOrderID != "factory-a" || cards[1].FactoryOrderID != "factory-b" {
		t.Fatalf("cards sorted by fallback identity = %+v, want factory-a before factory-b", cards)
	}
}

// Raw Hive-shaped fixture for the contract between transpara-ai/hive#169's
// Civilization Assembly projection endpoint and this Site consumer. Keep this as
// literal JSON so the test exercises wire keys and enum strings rather than
// re-encoding Site's own Go struct.
const hiveCivilizationAssemblyProjectionFixture = `{
  "projection_id": "civ-runtime-001",
  "projection_schema_version": "1.4.0",
  "projection_subject": "civilization_assembly",
  "generated_at": "2026-06-23T09:30:00Z",
  "source_eventgraph_head_or_state_version": "evt_head_runtime_001",
  "source_event_ids_or_query_window": ["evt_runtime_001"],
  "derivation_status": "complete",
  "authority_state": {
    "status": "available",
    "summary": "authority state projected by Hive",
    "authority_requests": [
      {
        "id": "req_pr_001",
        "actor_id": "actor_builder",
        "actor_role": "implementer",
        "action": "pull_request.create",
        "target_type": "pr",
        "target_id": "pr://transpara-ai/site/94",
        "risk_class": "medium",
        "status": "pending"
      }
    ],
    "authority_decisions": [],
    "source_refs": ["evt_authority_001"]
  },
  "external_committee_state": {
    "status": "unavailable",
    "summary": "External Committee approval records are not independently projected by Hive operator state.",
    "committee_roles": ["External Committee"]
  },
  "actor_roster": [
    {
      "id": "actor_summary_001",
      "actor_id": "actor_builder",
      "actor_type": "agent",
      "identity_mode": "runtime",
      "status": "active"
    }
  ],
  "role_bindings": [
    {
      "actor_id": "actor_builder",
      "role": "implementer",
      "source_ref": "evt_runtime_001",
      "source_type": "hive.agent.spawned"
    }
  ],
  "agent_lifecycle_summary": [
    {
      "id": "evt_runtime_001",
      "actor_id": "actor_builder",
      "to_state": "active",
      "status": "active"
    }
  ],
  "factory_order_summary": [
    {
      "id": "fo_run_issue_scan_001",
      "status": "work_task_seeded",
      "risk_class": "high",
      "release_policy": "human_required_before_merge",
      "requirement_refs": ["req_run_issue_scan_001"],
      "acceptance_criterion_refs": ["ac_run_issue_scan_001"],
      "task_refs": ["evt_work_task_001", "evt_work_task_stage_research_001", "evt_work_task_stage_debate_001"]
    }
  ],
  "work_evidence_summary": {
    "status": "available",
    "summary": "runtime projection from Hive",
    "task_refs": ["evt_work_task_001", "evt_work_task_stage_research_001", "evt_work_task_stage_debate_001"],
    "tasks": [
      {
        "id": "evt_work_task_001",
        "factory_order_id": "fo_run_issue_scan_001",
        "title": "Resolve transpara-ai/hive#321",
        "cell": "implementation",
        "risk_class": "high",
        "status": "work_task_seeded",
        "ready": false,
        "blocked": false,
        "requirement_refs": ["req_run_issue_scan_001"],
        "acceptance_criterion_refs": ["ac_run_issue_scan_001"],
        "expected_outputs": ["ready-for-Human result PR"],
        "source_refs": ["evt_work_task_001"]
      },
      {
        "id": "evt_work_task_stage_research_001",
        "canonical_task_id": "tsk_run_issue_scan_001_research_issue_and_repo_context",
        "factory_order_id": "fo_run_issue_scan_001",
        "lifecycle_stage_id": "research_issue_and_repo_context",
        "title": "Issue-scan stage: Research issue and repo context",
        "cell": "planning",
        "risk_class": "high",
        "status": "work_task_seeded",
        "ready": false,
        "blocked": false,
        "requirement_refs": ["req_run_issue_scan_001"],
        "acceptance_criterion_refs": ["ac_run_issue_scan_001"],
        "expected_outputs": ["stage declaration artifact remains pending runtime evidence", "repo_context_packet"],
        "required_evidence": ["issue_snapshot", "repo_context", "risk_and_scope_notes"],
        "required_roles": ["strategist", "planner"],
        "role_contract_refs": ["evt_work_task_stage_role_contract_research_001"],
        "output_contract_refs": ["evt_work_task_stage_output_contract_research_001"],
        "runtime_evidence_refs": ["artifact_runtime_research_001"],
        "runtime_evidence_status": "stage_completed_runtime_evidence_recorded",
        "role_output_contracts": [
          {
            "role": "strategist",
            "can_operate": false,
            "required_outputs": ["issue_priority_rationale", "risk_and_scope_notes"],
            "authority_boundary": "read_only",
            "completion_gate": "context_packet_recorded",
            "evidence_status": "stage_step_completed_runtime_evidence_recorded"
          },
          {
            "role": "planner",
            "can_operate": false,
            "required_outputs": ["repo_context_packet", "candidate_validation_commands"],
            "authority_boundary": "read_only",
            "completion_gate": "context_packet_recorded",
            "evidence_status": "stage_step_completed_runtime_evidence_recorded"
          }
        ],
        "agent_execution_plan": [
          {
            "id": "01_research_issue_and_repo_context_strategist",
            "stage_id": "research_issue_and_repo_context",
            "role": "strategist",
            "can_operate": false,
            "required_outputs": ["issue_priority_rationale", "risk_and_scope_notes"],
            "authority_boundary": "read_only",
            "completion_gate": "context_packet_recorded",
            "evidence_status": "expected_not_observed"
          },
          {
            "id": "02_research_issue_and_repo_context_planner",
            "stage_id": "research_issue_and_repo_context",
            "role": "planner",
            "can_operate": false,
            "required_outputs": ["repo_context_packet", "candidate_validation_commands"],
            "authority_boundary": "read_only",
            "completion_gate": "context_packet_recorded",
            "evidence_status": "expected_not_observed"
          }
        ],
        "source_refs": ["evt_work_task_stage_research_001"]
      },
      {
        "id": "evt_work_task_stage_debate_001",
        "canonical_task_id": "tsk_run_issue_scan_001_debate_with_correct_civic_roles",
        "factory_order_id": "fo_run_issue_scan_001",
        "lifecycle_stage_id": "debate_with_correct_civic_roles",
        "title": "Issue-scan stage: Debate with correct civic roles",
        "cell": "planning",
        "risk_class": "high",
        "status": "work_task_seeded",
        "ready": false,
        "blocked": false,
        "requirement_refs": ["req_run_issue_scan_001"],
        "acceptance_criterion_refs": ["ac_run_issue_scan_001"],
        "expected_outputs": ["stage declaration artifact remains pending runtime evidence", "decision_record"],
        "depends_on_refs": ["evt_work_task_stage_research_001"],
        "source_refs": ["evt_work_task_stage_debate_001", "evt_work_dependency_debate_after_research_001"]
      }
    ],
    "artifact_refs": ["evt_work_task_artifact_001", "evt_work_task_stage_role_contract_research_001", "evt_work_task_stage_output_contract_research_001", "artifact_runtime_research_001"],
    "artifacts": [
      {
        "id": "evt_work_task_artifact_001",
        "task_ref": "evt_work_task_001",
        "label": "issue_scan_execution_plan",
        "media_type": "application/json",
        "source_refs": ["evt_work_task_artifact_001"]
      },
      {
        "id": "evt_work_task_stage_artifact_001",
        "task_ref": "evt_work_task_001",
        "label": "issue_scan_lifecycle_stage_research_issue_and_repo_context",
        "media_type": "application/json",
        "source_refs": ["evt_work_task_stage_artifact_001"]
      },
      {
        "id": "evt_work_task_stage_role_contract_research_001",
        "task_ref": "evt_work_task_stage_research_001",
        "label": "issue_scan_stage_role_contract",
        "media_type": "application/json",
        "source_refs": ["evt_work_task_stage_role_contract_research_001"]
      },
      {
        "id": "evt_work_task_stage_output_contract_research_001",
        "task_ref": "evt_work_task_stage_research_001",
        "label": "issue_scan_stage_output_contract",
        "media_type": "application/json",
        "source_refs": ["evt_work_task_stage_output_contract_research_001"]
      },
      {
        "id": "artifact_runtime_research_001",
        "task_ref": "evt_work_task_stage_research_001",
        "label": "issue_scan_stage_runtime_evidence",
        "media_type": "application/json",
        "source_refs": ["evt_work_task_stage_runtime_evidence_research_001"]
      }
    ],
    "test_run_refs": ["test_run_001"],
    "gate_result_refs": ["gate_result_001"],
    "audit_report_refs": ["audit_report_001"],
    "source_refs": ["evt_runtime_001"]
  },
  "queued_run_request": {
    "event_id": "evt_factory_run_requested_001",
    "conversation_id": "conv_issue_scan_001",
    "run_id": "run_issue_scan_001",
    "title": "Resolve transpara-ai/hive#321",
    "operator_id": "operator_michael",
    "status": "queued",
    "target_repos": ["transpara-ai/hive"],
    "authority_initial_level": "Required",
    "authority_scope": "transpara-ai issue scan to ready-for-Human PR; no merge or deploy",
    "budget_max_iterations": 12,
    "budget_max_cost_usd": 25,
    "source_event_id": "evt_source_issue_001",
    "brief_event_id": "evt_brief_issue_001",
    "brief_kind": "transpara_ai_github_issue_scan",
    "lifecycle_version": "civilization_issue_to_human_ready_pr_v0.4",
    "lifecycle_evidence_kind": "expected_lifecycle_not_runtime_progress",
    "selection_policy": {
      "policy_id": "scanner_order_first_candidate_v0.1",
      "selected_rank": 1,
      "candidate_count": 3,
      "ranking_inputs": ["validated_transpara_ai_repo_scope", "scanner_return_order", "label_filters", "repo_scan_order"],
      "rationale": "Select the first validated candidate after the scanner has applied repo, label, state, and limit filters; preserve all candidates and ranks for later civic debate."
    },
    "development_lifecycle": [
      {
        "id": "research_issue_and_repo_context",
        "name": "Research issue and repo context",
        "required_roles": ["strategist", "planner"],
        "required_evidence": ["issue_snapshot", "repo_context", "risk_and_scope_notes"],
        "authority_boundary": "read_only",
        "completion_gate": "context_packet_recorded",
        "evidence_status": "declared_pending_runtime_evidence"
      },
      {
        "id": "debate_with_correct_civic_roles",
        "name": "Debate with correct civic roles",
        "required_roles": ["strategist", "planner", "reviewer", "guardian"],
        "required_evidence": ["role_positions", "decision_record", "dissent_or_no_dissent_record"],
        "authority_boundary": "proposal_only_no_mutation",
        "completion_gate": "decision_record_has_reviewer_and_guardian_disposition",
        "evidence_status": "expected_not_observed"
      },
      {
        "id": "select_and_design_approach",
        "name": "Select and design approach",
        "required_roles": ["planner", "reviewer", "guardian"],
        "required_evidence": ["selected_approach", "definition_of_done", "acceptance_criteria", "test_plan"],
        "authority_boundary": "implementation_waits_for_authorized_task",
        "completion_gate": "implementation_task_has_readiness_artifacts",
        "evidence_status": "expected_not_observed"
      },
      {
        "id": "implement_on_branch",
        "name": "Implement on branch",
        "required_roles": ["implementer"],
        "required_evidence": ["branch_name", "commit_sha", "changed_files", "validation_output"],
        "authority_boundary": "no_merge_no_deploy",
        "completion_gate": "implementation_changes_validated_on_branch",
        "evidence_status": "expected_not_observed"
      },
      {
        "id": "run_adversarial_review",
        "name": "Run adversarial review",
        "required_roles": ["reviewer", "guardian"],
        "required_evidence": ["exact_head_review_artifact", "finding_disposition"],
        "authority_boundary": "review_is_blocking",
        "completion_gate": "review_artifact_returned_and_findings_classified",
        "evidence_status": "expected_not_observed"
      },
      {
        "id": "drive_blockers_to_zero",
        "name": "Drive blockers to zero",
        "required_roles": ["implementer", "reviewer", "guardian"],
        "required_evidence": ["blocker_fixes", "rerun_validation", "rerun_review"],
        "authority_boundary": "accepted_findings_resolved_or_rejected_with_evidence",
        "completion_gate": "exact_head_review_has_zero_blockers",
        "evidence_status": "expected_not_observed"
      },
      {
        "id": "surface_ready_for_Human_result_PR",
        "name": "Surface ready-for-Human result PR",
        "required_roles": ["strategist", "reviewer", "guardian"],
        "required_evidence": ["ready_pr_url", "ready_state_review", "human_ready_summary"],
        "authority_boundary": "human_approval_required_no_merge",
        "completion_gate": "ready_pr_has_exact_head_evidence_and_waits_for_human",
        "evidence_status": "expected_not_observed"
      }
    ],
    "agent_execution_plan": [
      {
        "id": "01_research_issue_and_repo_context_strategist",
        "stage_id": "research_issue_and_repo_context",
        "role": "strategist",
        "can_operate": false,
        "objective": "Decide why this issue should enter the Civilization factory loop now.",
        "required_outputs": ["issue_priority_rationale", "risk_and_scope_notes"],
        "authority_boundary": "read_only",
        "completion_gate": "context_packet_recorded",
        "evidence_status": "stage_declared_pending_runtime_evidence"
      },
      {
        "id": "10_implement_on_branch_implementer",
        "stage_id": "implement_on_branch",
        "role": "implementer",
        "can_operate": true,
        "objective": "Implement the selected approach on a branch and record exact validation evidence.",
        "required_outputs": ["branch_name", "commit_sha", "validation_output"],
        "authority_boundary": "no_merge_no_deploy",
        "completion_gate": "implementation_changes_validated_on_branch",
        "evidence_status": "expected_not_observed"
      },
      {
        "id": "11_run_adversarial_review_reviewer",
        "stage_id": "run_adversarial_review",
        "role": "reviewer",
        "can_operate": false,
        "objective": "Run an exact-head adversarial review and produce a durable review artifact.",
        "required_outputs": ["exact_head_review_artifact"],
        "authority_boundary": "review_is_blocking",
        "completion_gate": "review_artifact_returned_and_findings_classified",
        "evidence_status": "expected_not_observed"
      },
      {
        "id": "18_surface_ready_for_Human_result_PR_guardian",
        "stage_id": "surface_ready_for_Human_result_PR",
        "role": "guardian",
        "can_operate": false,
        "objective": "Confirm the result PR waits for Human approval and does not merge itself.",
        "required_outputs": ["human_approval_boundary_check"],
        "authority_boundary": "human_approval_required_no_merge",
        "completion_gate": "ready_pr_has_exact_head_evidence_and_waits_for_human",
        "evidence_status": "expected_not_observed"
      }
    ],
    "evidence_kind": "queued_request_not_runtime_start",
    "created_at": "2026-06-23T09:35:00Z"
  },
  "issue_intake_projection": {
    "source_refs": ["change-control-scan:2026-06-25T15:59:37Z"],
    "issues": [
      {
        "repo": "transpara-ai/site",
        "number": 114,
        "url": "https://github.com/transpara-ai/site/issues/114",
        "title": "Read-only issue intake aggregation projection UI",
        "state": "open",
        "labels": ["cc:intake", "cc:aggregate-candidate", "cc:civilization-presence", "cc:pr-ready"],
        "primary_repo": "transpara-ai/site",
        "touched_substrate": "site operator ui projection",
        "risk_class": "normal",
        "readiness": "cc:pr-ready",
        "pr_ready_when": "pr-ready after fixture/read-model source, limitation labels, and no-write behavior are defined",
        "authority_boundary": "read-only issue record; no implementation authority",
        "updated_at": "2026-06-25T15:59:37Z",
        "source_refs": ["github:transpara-ai/site#114"]
      },
      {
        "repo": "transpara-ai/site",
        "number": 116,
        "url": "https://github.com/transpara-ai/site/issues/116",
        "title": "Replace temporary Civilization fixture display with issue and EventGraph projection inputs",
        "state": "open",
        "labels": ["cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"],
        "primary_repo": "transpara-ai/site",
        "touched_substrate": "site civilization display data source",
        "risk_class": "protected-action",
        "readiness": "cc:pr-deferred",
        "pr_ready_when": "pr-ready only after EventGraph/issue read inputs and unavailable-state behavior are scoped",
        "authority_boundary": "protected-action sensitive; issue record is not authority",
        "updated_at": "2026-06-25T04:35:54Z",
        "source_refs": ["github:transpara-ai/site#116"]
      }
    ]
  },
  "issue_scan_projection": {
    "runs": [
      {
        "run_id": "run_docs_172",
        "factory_order_id": "fo_docs_172",
        "lifecycle_version": "civilization_issue_to_human_ready_pr_v0.4",
        "state": "parked",
        "target_issue": {
          "repo": "transpara-ai/docs",
          "number": 172,
          "url": "https://github.com/transpara-ai/docs/issues/172",
          "state": "open",
          "labels": ["cc:pr-ready"]
        },
        "selected_issue": {
          "repo": "transpara-ai/docs",
          "number": 172,
          "url": "https://github.com/transpara-ai/docs/issues/172",
          "state": "open"
        },
        "candidate_issues": [
          {
            "repo": "transpara-ai/docs",
            "number": 172,
            "labels": ["cc:pr-ready"]
          },
          {
            "repo": "transpara-ai/site",
            "number": 115,
            "labels": ["cc:pr-ready", "cc:protected-action"]
          }
        ],
        "source_refs": ["github:transpara-ai/docs#172"]
      },
      {
        "run_id": "run_site_115",
        "factory_order_id": "fo_site_115",
        "lifecycle_version": "civilization_issue_to_human_ready_pr_v0.4",
        "state": "human_action",
        "target_issue": {
          "repo": "transpara-ai/site",
          "number": 115,
          "url": "https://github.com/transpara-ai/site/issues/115",
          "state": "open",
          "labels": ["cc:protected-action"]
        },
        "selected_issue": {
          "repo": "transpara-ai/site",
          "number": 115,
          "url": "https://github.com/transpara-ai/site/issues/115",
          "state": "open"
        },
        "candidate_issues": [
          {
            "repo": "transpara-ai/site",
            "number": 115,
            "labels": ["cc:protected-action"]
          }
        ],
        "source_refs": ["github:transpara-ai/site#115"]
      },
      {
        "run_id": "run_docs_172_scope",
        "factory_order_id": "fo_docs_172_scope",
        "lifecycle_version": "civilization_issue_to_human_ready_pr_v0.4",
        "state": "human_action",
        "target_issue": {
          "repo": "transpara-ai/docs",
          "number": 172,
          "url": "https://github.com/transpara-ai/docs/issues/172",
          "state": "open",
          "labels": ["cc:needs-human-scope"]
        },
        "selected_issue": {
          "repo": "transpara-ai/docs",
          "number": 172,
          "url": "https://github.com/transpara-ai/docs/issues/172",
          "state": "open",
          "labels": ["cc:needs-human-scope"]
        },
        "candidate_issues": [
          {
            "repo": "transpara-ai/docs",
            "number": 172,
            "labels": ["cc:needs-human-scope"]
          }
        ],
        "source_refs": ["github:transpara-ai/docs#172"]
      }
    ],
    "stages": [
      {
        "run_id": "run_docs_172",
        "factory_order_id": "fo_docs_172",
        "stage_id": "run_adversarial_review",
        "stage_number": 5,
        "stage_count": 7,
        "canonical_task_id": "tsk_docs_172_run_adversarial_review",
        "task_id": "019c0000-0000-7000-8000-000000001172",
        "current_state": "parked",
        "completion_gate": "exact-head adversarial review returns zero blockers",
        "authority_boundary": "review only; no merge or deploy",
        "assigned_agent_ids": ["agent_reviewer"],
        "touching_agent_ids": ["agent_blocker_repair", "agent_reviewer"],
        "evidence_refs": ["code.review.submitted:docs-172"]
      },
      {
        "run_id": "run_site_115",
        "factory_order_id": "fo_site_115",
        "stage_id": "surface_ready_for_human_result_pr",
        "stage_number": 7,
        "stage_count": 7,
        "canonical_task_id": "tsk_site_115_surface_ready_for_human_result_pr",
        "task_id": "019c0000-0000-7000-8000-000000001115",
        "current_state": "human_action",
        "completion_gate": "ready PR waits for human approval",
        "authority_boundary": "human approval required; no merge",
        "assigned_agent_ids": ["agent_guardian"],
        "touching_agent_ids": ["agent_guardian"],
        "evidence_refs": ["human_ready_summary:site-115"]
      },
      {
        "run_id": "run_docs_172_scope",
        "factory_order_id": "fo_docs_172_scope",
        "stage_id": "select_and_design_approach",
        "stage_number": 3,
        "stage_count": 7,
        "canonical_task_id": "tsk_docs_172_scope_select_and_design_approach",
        "task_id": "019c0000-0000-7000-8000-000000001174",
        "current_state": "human_action",
        "completion_gate": "human scope decision recorded",
        "authority_boundary": "human scope clarification required",
        "assigned_agent_ids": ["agent_guardian"],
        "touching_agent_ids": ["agent_guardian"],
        "evidence_refs": ["github:transpara-ai/docs#172"]
      }
    ],
    "blockers": [
      {
        "run_id": "run_docs_172",
        "factory_order_id": "fo_docs_172",
        "stage_id": "run_adversarial_review",
        "blocker_type": "duplicate_chain",
        "reason": "canonical task chain duplicated",
        "required_action": "collapse duplicate canonical stage chain",
        "evidence_refs": ["work:duplicate-stage-chain"]
      },
      {
        "run_id": "run_site_115",
        "factory_order_id": "fo_site_115",
        "stage_id": "surface_ready_for_human_result_pr",
        "blocker_type": "protected_action",
        "reason": "ready result PR requires human approval",
        "required_action": "human must authorize protected repo action",
        "evidence_refs": ["github:transpara-ai/site#115"]
      },
      {
        "run_id": "run_site_115",
        "factory_order_id": "fo_site_115",
        "stage_id": "research_issue_and_repo_context",
        "blocker_type": "stale_target",
        "reason": "target state must be refreshed before runtime continues",
        "required_action": "confirm target issue is still live",
        "evidence_refs": ["github:transpara-ai/site#115"]
      },
      {
        "run_id": "run_docs_172_scope",
        "factory_order_id": "fo_docs_172_scope",
        "stage_id": "select_and_design_approach",
        "blocker_type": "needs_human_scope",
        "reason": "issue requires human scope clarification",
        "required_action": "human must clarify issue scope before runtime continues",
        "evidence_refs": ["github:transpara-ai/docs#172"]
      }
    ],
    "lineage": [
      {
        "run_id": "run_docs_172",
        "factory_order_id": "fo_docs_172",
        "stage_id": "run_adversarial_review",
        "canonical_task_id": "tsk_docs_172_run_adversarial_review",
        "primary_task_id": "019c0000-0000-7000-8000-000000001172",
        "task_ids": [
          "019c0000-0000-7000-8000-000000001172",
          "019c0000-0000-7000-8000-000000001173"
        ],
        "duplicate_task_ids": ["019c0000-0000-7000-8000-000000001173"],
        "duplicate_of": "019c0000-0000-7000-8000-000000001172",
        "source_refs": ["work:duplicate-stage-chain"]
      }
    ]
  },
  "site_consumer_status": {
    "status": "available",
    "summary": "Site consumed Hive read-only endpoint",
    "source_refs": ["GET /api/hive/civilization/assembly-projection"]
  },
  "open_gate_summary": [],
  "residual_risk_summary": [],
  "withheld_or_unavailable_fields": [
    {
      "field": "external_committee_state",
      "status": "unavailable",
      "reason": "Hive operator projection does not independently certify External Committee approval."
    }
  ],
  "boundary_flags": ["read_only_site_consumer", "no_runtime_execution"],
  "provenance_refs": ["evt_runtime_001"],
  "validation_refs": ["GET /api/hive/civilization/assembly-projection"]
}`

func TestHandleOpsCivilizationFailsClosedWhenHiveProjectionFails(t *testing.T) {
	h, _, _ := testHandlers(t)
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "projection unavailable", http.StatusBadGateway)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)
	t.Setenv("HIVE_OPS_API_KEY", "")

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/civilization", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/civilization: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"failed",
		"hive civilization projection returned 502 Bad Gateway",
		"failed_closed",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/civilization: body does not contain %q", want)
		}
	}
	assertNoCivilizationMutationControls(t, civilizationAssemblySurface(t, body))
}

func TestHandleOpsCivilizationFailsClosedForInvalidHiveProjectionPayloads(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "malformed json",
			body: `{not-json`,
			want: "invalid character",
		},
		{
			name: "empty object",
			body: `{}`,
			want: "unsupported projection schema version",
		},
		{
			name: "unsupported schema",
			body: strings.Replace(hiveCivilizationAssemblyProjectionFixture, `"projection_schema_version": "1.4.0"`, `"projection_schema_version": "2.0.0"`, 1),
			want: `unsupported projection schema version &#34;2.0.0&#34;`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, _ := testHandlers(t)
			hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tt.body)
			}))
			defer hiveSrv.Close()
			t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

			mux := http.NewServeMux()
			h.Register(mux)

			req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/civilization", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("GET /ops/civilization: status = %d, want 200; body: %s", w.Code, w.Body.String())
			}
			body := w.Body.String()
			for _, want := range []string{"failed", "failed_closed", tt.want} {
				if !strings.Contains(body, want) {
					t.Fatalf("GET /ops/civilization: body does not contain %q", want)
				}
			}
			assertNoCivilizationMutationControls(t, civilizationAssemblySurface(t, body))
		})
	}
}

func TestBuildOpsCivilizationConsumesCompleteProjection(t *testing.T) {
	generatedAt := time.Date(2026, 6, 20, 10, 30, 0, 0, time.UTC)
	now := generatedAt.Add(30 * time.Minute)
	projection := &OpsCivilizationAssemblyProjection{
		ProjectionID:                       "civ-proj-001",
		ProjectionSchemaVersion:            "1.0.0",
		ProjectionSubject:                  "civilization_assembly",
		GeneratedAt:                        generatedAt,
		SourceEventGraphHeadOrStateVersion: "sha256:abc123",
		SourceEventIDsOrQueryWindow:        []string{"authority_decision_001", "site_consumer_civ_001"},
		DerivationStatus:                   opsCivilizationProjectionStatusComplete,
		AuthorityState: OpsCivilizationAssemblyAuthorityState{
			Status:     opsCivilizationFieldAvailable,
			Summary:    "authority chain derived from EventGraph records",
			SourceRefs: []string{"authority_decision_001"},
		},
		ExternalCommitteeState: OpsCivilizationAssemblyCommitteeState{
			Status:         opsCivilizationFieldAvailable,
			Summary:        "External Committee approval records present",
			ApprovalRefs:   []string{"human_approval_001"},
			CommitteeRoles: []string{"External Committee"},
		},
		ActorRoster: []OpsCivilizationAssemblyActorSummary{
			{ID: "actor_identity_001", ActorID: "civilization-operator", ActorType: "human", IdentityMode: "verified", Status: "active"},
		},
		RoleBindings: []OpsCivilizationAssemblyRoleBinding{
			{ActorID: "civilization-operator", Role: "External Committee", SourceRef: "authority_decision_001", SourceType: "AuthorityDecision"},
		},
		AgentLifecycleSummary: []OpsCivilizationAssemblyLifecycleSummary{
			{ID: "lifecycle_001", ActorID: "civilization-operator", FromState: "candidate", ToState: "active", Status: "verified"},
		},
		WorkEvidenceSummary: OpsCivilizationAssemblyWorkEvidence{
			Status:   opsCivilizationFieldAvailable,
			Summary:  "work evidence derived from task and gate records",
			TaskRefs: []string{"evt_work_task_001", "evt_work_task_stage_research_001"},
			Tasks: []OpsCivilizationAssemblyTaskEvidence{
				{
					ID:              "evt_work_task_001",
					FactoryOrderID:  "fo_run_issue_scan_001",
					Title:           "Resolve transpara-ai/hive#321",
					Cell:            "implementation",
					RiskClass:       "high",
					Status:          "work_task_seeded",
					ExpectedOutputs: []string{"ready-for-Human result PR"},
					RequirementRefs: []string{"req_run_issue_scan_001"},
					SourceRefs:      []string{"evt_work_task_001"},
				},
				{
					ID:                      "evt_work_task_stage_research_001",
					CanonicalTaskID:         "tsk_run_issue_scan_001_research_issue_and_repo_context",
					FactoryOrderID:          "fo_run_issue_scan_001",
					LifecycleStageID:        "research_issue_and_repo_context",
					Title:                   "Issue-scan stage: Research issue and repo context",
					Cell:                    "planning",
					RiskClass:               "high",
					Status:                  "work_task_seeded",
					RequirementRefs:         []string{"req_run_issue_scan_001"},
					AcceptanceCriterionRefs: []string{"ac_run_issue_scan_001"},
					ExpectedOutputs:         []string{"stage declaration artifact remains pending runtime evidence", "repo_context_packet"},
					RequiredRoles:           []string{"strategist", "planner"},
					RoleContractRefs:        []string{"evt_work_task_stage_role_contract_research_001"},
					RequiredEvidence:        []string{"issue_snapshot", "repo_context", "risk_and_scope_notes"},
					OutputContractRefs:      []string{"evt_work_task_stage_output_contract_research_001"},
					RuntimeEvidenceRefs:     []string{"artifact_runtime_research_001"},
					RuntimeEvidenceStatus:   "stage_completed_runtime_evidence_recorded",
					RoleOutputContracts: []OpsCivilizationRoleOutputContract{
						{
							Role:              "strategist",
							RequiredOutputs:   []string{"issue_priority_rationale", "risk_and_scope_notes"},
							AuthorityBoundary: "read_only",
							CompletionGate:    "context_packet_recorded",
							EvidenceStatus:    "stage_step_completed_runtime_evidence_recorded",
						},
						{
							Role:              "planner",
							RequiredOutputs:   []string{"repo_context_packet", "candidate_validation_commands"},
							AuthorityBoundary: "read_only",
							CompletionGate:    "context_packet_recorded",
							EvidenceStatus:    "stage_step_completed_runtime_evidence_recorded",
						},
					},
					AgentExecutionPlan: []OpsHiveQueuedRunAgentPlanStep{
						{
							ID:                "01_research_issue_and_repo_context_strategist",
							StageID:           "research_issue_and_repo_context",
							Role:              "strategist",
							RequiredOutputs:   []string{"issue_priority_rationale"},
							AuthorityBoundary: "read_only",
						},
					},
					SourceRefs: []string{"evt_work_task_stage_research_001"},
				},
			},
			ArtifactRefs: []string{"evt_work_task_artifact_001", "evt_work_task_stage_output_contract_research_001", "artifact_runtime_research_001"},
			Artifacts: []OpsCivilizationAssemblyArtifactEvidence{
				{
					ID:         "evt_work_task_artifact_001",
					TaskRef:    "evt_work_task_001",
					Label:      "issue_scan_execution_plan",
					MediaType:  "application/json",
					SourceRefs: []string{"evt_work_task_artifact_001"},
				},
				{
					ID:         "evt_work_task_stage_artifact_001",
					TaskRef:    "evt_work_task_001",
					Label:      "issue_scan_lifecycle_stage_research_issue_and_repo_context",
					MediaType:  "application/json",
					SourceRefs: []string{"evt_work_task_stage_artifact_001"},
				},
				{
					ID:         "evt_work_task_stage_output_contract_research_001",
					TaskRef:    "evt_work_task_stage_research_001",
					Label:      "issue_scan_stage_output_contract",
					MediaType:  "application/json",
					SourceRefs: []string{"evt_work_task_stage_output_contract_research_001"},
				},
				{
					ID:         "artifact_runtime_research_001",
					TaskRef:    "evt_work_task_stage_research_001",
					Label:      "issue_scan_stage_runtime_evidence",
					MediaType:  "application/json",
					SourceRefs: []string{"evt_work_task_stage_runtime_evidence_research_001"},
				},
			},
			TestRunRefs: []string{"test_run_001"},
		},
		QueuedRunRequest: &OpsHiveQueuedRunRequest{
			RunID:                 "run_issue_scan_001",
			Title:                 "Resolve transpara-ai/hive#321",
			Status:                "queued",
			TargetRepos:           []string{"transpara-ai/hive"},
			AuthorityInitialLevel: "Required",
			AuthorityScope:        "transpara-ai issue scan to ready-for-Human PR; no merge or deploy",
			LifecycleVersion:      "civilization_issue_to_human_ready_pr_v0.4",
			LifecycleEvidenceKind: "expected_lifecycle_not_runtime_progress",
			EvidenceKind:          "queued_request_not_runtime_start",
			SelectionPolicy: &OpsHiveQueuedRunSelectionPolicy{
				PolicyID:       "scanner_order_first_candidate_v0.1",
				SelectedRank:   1,
				CandidateCount: 3,
				RankingInputs:  []string{"validated_transpara_ai_repo_scope", "scanner_return_order", "label_filters", "repo_scan_order"},
				Rationale:      "Select the first validated candidate after the scanner has applied repo, label, state, and limit filters; preserve all candidates and ranks for later civic debate.",
			},
			DevelopmentLifecycle: []OpsHiveQueuedRunLifecycleStage{
				{
					ID:                "research_issue_and_repo_context",
					Name:              "Research issue and repo context",
					RequiredRoles:     []string{"strategist", "planner"},
					AuthorityBoundary: "read_only",
					CompletionGate:    "context_packet_recorded",
					EvidenceStatus:    "declared_pending_runtime_evidence",
				},
				{
					ID:                "surface_ready_for_Human_result_PR",
					Name:              "Surface ready-for-Human result PR",
					RequiredRoles:     []string{"strategist", "reviewer", "guardian"},
					AuthorityBoundary: "human_approval_required_no_merge",
					CompletionGate:    "ready_pr_has_exact_head_evidence_and_waits_for_human",
					EvidenceStatus:    "expected_not_observed",
				},
			},
			AgentExecutionPlan: []OpsHiveQueuedRunAgentPlanStep{
				{
					ID:                "18_surface_ready_for_Human_result_PR_guardian",
					StageID:           "surface_ready_for_Human_result_PR",
					Role:              "guardian",
					CanOperate:        false,
					RequiredOutputs:   []string{"human_approval_boundary_check"},
					AuthorityBoundary: "human_approval_required_no_merge",
					CompletionGate:    "ready_pr_has_exact_head_evidence_and_waits_for_human",
					EvidenceStatus:    "expected_not_observed",
				},
			},
		},
		FactoryOrderSummary: []OpsCivilizationAssemblyFactoryOrder{
			{
				ID:                      "fo_run_issue_scan_001",
				Status:                  "work_task_seeded",
				RiskClass:               "high",
				ReleasePolicy:           "human_required_before_merge",
				RequirementRefs:         []string{"req_run_issue_scan_001"},
				AcceptanceCriterionRefs: []string{"ac_run_issue_scan_001"},
				TaskRefs:                []string{"evt_work_task_001"},
			},
		},
		SiteConsumerStatus: OpsCivilizationAssemblyFieldStatus{
			Status:     opsCivilizationFieldAvailable,
			Summary:    "read-only Civilization Assembly Site consumer evidence present",
			SourceRefs: []string{"site_consumer_civ_001"},
		},
		BoundaryFlags:  []string{"read_only_site_consumer", "no_runtime_execution"},
		ProvenanceRefs: []string{"authority_decision_001", "site_consumer_civ_001"},
		ValidationRefs: []string{"test_run_001"},
	}

	data := buildOpsCivilizationAssemblyDataFromProjection(projection, now)

	if data.ProjectionStatus != opsCivilizationProjectionStatusComplete {
		t.Fatalf("projection status = %q, want complete", data.ProjectionStatus)
	}
	if data.ProjectionFreshness != "current" {
		t.Fatalf("projection freshness = %q, want current", data.ProjectionFreshness)
	}
	if !strings.Contains(data.ProjectionSource, "civ-proj-001") {
		t.Fatalf("projection source does not include projection id: %q", data.ProjectionSource)
	}
	if got := statusRowValue(data, "projection generated"); got != "2026-06-20 10:30:00 UTC" {
		t.Fatalf("projection generated = %q", got)
	}
	if got := statusRowValue(data, "authority state"); !strings.Contains(got, "available - authority chain") {
		t.Fatalf("authority state row = %q", got)
	}
	if got := boundaryState(data, "Authority state"); got != opsCivilizationFieldAvailable {
		t.Fatalf("authority boundary = %q, want available", got)
	}
	if got := boundaryState(data, "Site consumer"); got != opsCivilizationFieldAvailable {
		t.Fatalf("site consumer boundary = %q, want available", got)
	}
	if len(data.Civilization.Roster) != 1 {
		t.Fatalf("roster len = %d, want 1: %+v", len(data.Civilization.Roster), data.Civilization.Roster)
	}
	row := data.Civilization.Roster[0]
	if row.Role != "External Committee" || row.Agent != "civilization-operator" || row.Origin != "AuthorityDecision" {
		t.Fatalf("unexpected projected roster row: %+v", row)
	}
	if !referenceGroupContains(data, "validation refs", "test_run_001") {
		t.Fatalf("validation refs missing test_run_001: %+v", data.ReferenceGroups)
	}
	if len(data.FactoryOrders) != 1 || data.FactoryOrders[0].ID != "fo_run_issue_scan_001" {
		t.Fatalf("factory orders = %+v, want projected FactoryOrder", data.FactoryOrders)
	}
	if !sliceContains(data.WorkEvidence.TaskRefs, "evt_work_task_001") {
		t.Fatalf("work evidence task refs = %+v, want evt_work_task_001", data.WorkEvidence.TaskRefs)
	}
	stageTask := civilizationTaskByID(data.WorkEvidence.Tasks, "evt_work_task_stage_research_001")
	if stageTask == nil || stageTask.LifecycleStageID != "research_issue_and_repo_context" {
		t.Fatalf("work evidence stage task = %+v, all tasks = %+v", stageTask, data.WorkEvidence.Tasks)
	}
	if stageTask.CanonicalTaskID != "tsk_run_issue_scan_001_research_issue_and_repo_context" || !sliceContains(stageTask.ExpectedOutputs, "repo_context_packet") {
		t.Fatalf("work evidence stage task metadata = %+v", stageTask)
	}
	if opsCivilizationEvidenceStatusValue(stageTask.Status, "missing") != "work task seeded" {
		t.Fatalf("work evidence stage task status = %q", stageTask.Status)
	}
	if !sliceContains(stageTask.RequiredRoles, "strategist") || !sliceContains(stageTask.RequiredRoles, "planner") {
		t.Fatalf("work evidence stage task roles = %+v", stageTask.RequiredRoles)
	}
	if !sliceContains(stageTask.RoleContractRefs, "evt_work_task_stage_role_contract_research_001") {
		t.Fatalf("work evidence stage task role contract refs = %+v", stageTask.RoleContractRefs)
	}
	if !sliceContains(stageTask.RequiredEvidence, "repo_context") || !sliceContains(stageTask.OutputContractRefs, "evt_work_task_stage_output_contract_research_001") {
		t.Fatalf("work evidence stage task output contract refs/evidence = %+v / %+v", stageTask.OutputContractRefs, stageTask.RequiredEvidence)
	}
	if stageTask.RuntimeEvidenceStatus != "stage_completed_runtime_evidence_recorded" || !sliceContains(stageTask.RuntimeEvidenceRefs, "artifact_runtime_research_001") {
		t.Fatalf("work evidence stage task runtime evidence = %q / %+v", stageTask.RuntimeEvidenceStatus, stageTask.RuntimeEvidenceRefs)
	}
	if len(stageTask.RoleOutputContracts) != 2 || !roleOutputContractsContain(stageTask.RoleOutputContracts, "strategist", "risk_and_scope_notes") {
		t.Fatalf("work evidence stage task role output contracts = %+v", stageTask.RoleOutputContracts)
	}
	if opsCivilizationEvidenceStatusValue(stageTask.RoleOutputContracts[0].EvidenceStatus, "missing") != "stage step completed runtime evidence recorded" {
		t.Fatalf("work evidence stage task role output status = %+v", stageTask.RoleOutputContracts)
	}
	if len(stageTask.AgentExecutionPlan) != 1 || stageTask.AgentExecutionPlan[0].Role != "strategist" || !sliceContains(stageTask.AgentExecutionPlan[0].RequiredOutputs, "issue_priority_rationale") {
		t.Fatalf("work evidence stage task agent plan = %+v", stageTask.AgentExecutionPlan)
	}
	if !sliceContains(data.WorkEvidence.ArtifactRefs, "evt_work_task_artifact_001") {
		t.Fatalf("work evidence artifact refs = %+v, want evt_work_task_artifact_001", data.WorkEvidence.ArtifactRefs)
	}
	if civilizationArtifactByLabel(data.WorkEvidence.Artifacts, "issue_scan_execution_plan") == nil {
		t.Fatalf("work evidence artifacts = %+v, want issue_scan_execution_plan", data.WorkEvidence.Artifacts)
	}
	stageArtifact := civilizationArtifactByLabel(data.WorkEvidence.Artifacts, "issue_scan_lifecycle_stage_research_issue_and_repo_context")
	if stageArtifact == nil || stageArtifact.MediaType != "application/json" {
		t.Fatalf("work evidence stage artifact = %+v, all artifacts = %+v", stageArtifact, data.WorkEvidence.Artifacts)
	}
	if civilizationArtifactByLabel(data.WorkEvidence.Artifacts, "issue_scan_stage_output_contract") == nil {
		t.Fatalf("work evidence artifacts = %+v, want issue_scan_stage_output_contract", data.WorkEvidence.Artifacts)
	}
	if civilizationArtifactByLabel(data.WorkEvidence.Artifacts, "issue_scan_stage_runtime_evidence") == nil {
		t.Fatalf("work evidence artifacts = %+v, want issue_scan_stage_runtime_evidence", data.WorkEvidence.Artifacts)
	}
	if !sliceContains(data.WorkEvidence.TestRunRefs, "test_run_001") {
		t.Fatalf("work evidence test refs = %+v, want test_run_001", data.WorkEvidence.TestRunRefs)
	}
	if len(data.IssueScanStageEvidence) != 1 {
		t.Fatalf("stage evidence = %+v, want one projected issue-scan stage artifact", data.IssueScanStageEvidence)
	}
	stageEvidence := data.IssueScanStageEvidence[0]
	if stageEvidence.StageID != "research_issue_and_repo_context" || stageEvidence.ArtifactID != "evt_work_task_stage_artifact_001" {
		t.Fatalf("stage evidence = %+v, want research stage artifact", stageEvidence)
	}
	if stageEvidence.StageName != "Research issue and repo context" || stageEvidence.EvidenceStatus != "declared pending runtime evidence" {
		t.Fatalf("stage evidence status = %+v", stageEvidence)
	}
	if data.QueuedRunRequest == nil || data.QueuedRunRequest.RunID != "run_issue_scan_001" {
		t.Fatalf("queued run request = %+v, want run_issue_scan_001", data.QueuedRunRequest)
	}
	if len(data.QueuedRunRequest.AgentExecutionPlan) != 1 || data.QueuedRunRequest.AgentExecutionPlan[0].RequiredOutputs[0] != "human_approval_boundary_check" {
		t.Fatalf("queued run agent plan = %+v", data.QueuedRunRequest.AgentExecutionPlan)
	}
	if data.QueuedRunRequest.SelectionPolicy == nil || data.QueuedRunRequest.SelectionPolicy.PolicyID != "scanner_order_first_candidate_v0.1" || !sliceContains(data.QueuedRunRequest.SelectionPolicy.RankingInputs, "scanner_return_order") {
		t.Fatalf("queued run selection policy = %+v", data.QueuedRunRequest.SelectionPolicy)
	}
	if data.IssueReadiness.Status != "pending: Surface ready-for-Human result PR" || data.IssueReadiness.FirstPendingStage != "Surface ready-for-Human result PR" {
		t.Fatalf("issue readiness = %+v, want pending surface-ready stage", data.IssueReadiness)
	}
	if !strings.Contains(data.IssueReadiness.PRReadyWhen, "exact-head CFAR") {
		t.Fatalf("issue PR-Ready-When = %q, want exact-head CFAR boundary", data.IssueReadiness.PRReadyWhen)
	}
	if !strings.Contains(data.IssueReadiness.RecommendationState, "recommendation-only rank 1 of 3") {
		t.Fatalf("issue recommendation state = %q", data.IssueReadiness.RecommendationState)
	}
	if !sliceContains(data.IssueReadiness.GroupingInputs, "scanner_return_order") {
		t.Fatalf("issue grouping inputs = %+v, want scanner_return_order", data.IssueReadiness.GroupingInputs)
	}
	if !issueGuardrailContains(data.IssueReadiness.Guardrails, "cc:aggregate-candidate", "recommendation-only") {
		t.Fatalf("issue guardrails = %+v, want aggregate-candidate recommendation guardrail", data.IssueReadiness.Guardrails)
	}
	if findingContains(data, "fallback") {
		t.Fatal("projection consumer retained a fallback finding")
	}
}

func TestOpsCivilizationProjectionRenderEscapesHostileReadOnlyData(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	projection := &OpsCivilizationAssemblyProjection{
		ProjectionID:                       `civ-proj-<button onclick="x">`,
		ProjectionSchemaVersion:            "1.0.0",
		ProjectionSubject:                  "civilization_assembly",
		GeneratedAt:                        now,
		SourceEventGraphHeadOrStateVersion: "sha256:hostile",
		SourceEventIDsOrQueryWindow:        []string{`<form method="post">`},
		DerivationStatus:                   opsCivilizationProjectionStatusComplete,
		AuthorityState: OpsCivilizationAssemblyAuthorityState{
			Status:  opsCivilizationFieldAvailable,
			Summary: `<button onclick="steal()">approve</button>`,
		},
		ActorRoster: []OpsCivilizationAssemblyActorSummary{
			{ID: `actor_<input>`, ActorID: `actor-<script>`, ActorType: `human`, Status: `active`},
		},
		RoleBindings: []OpsCivilizationAssemblyRoleBinding{
			{ActorID: `actor-<script>`, Role: `<button onclick="x">operator</button>`, SourceRef: `<a hx-post="/mutate">`, SourceType: "AuthorityDecision"},
		},
		FactoryOrderSummary: []OpsCivilizationAssemblyFactoryOrder{
			{
				ID:                      `<script>alert(1)</script>`,
				Status:                  `<button onclick="x">seeded</button>`,
				RiskClass:               `<form action="/mutate">`,
				ReleasePolicy:           `<a href="/deploy">deploy</a>`,
				RequirementRefs:         []string{`<select><option>req</option></select>`},
				AcceptanceCriterionRefs: []string{`<script>accept()</script>`},
				TaskRefs:                []string{`<input name="task">`},
			},
		},
		WorkEvidenceSummary: OpsCivilizationAssemblyWorkEvidence{
			Status:  opsCivilizationFieldAvailable,
			Summary: `<script>alert("work")</script>`,
			Tasks: []OpsCivilizationAssemblyTaskEvidence{
				{
					ID:                    `task_<script>alert(5)</script>`,
					CanonicalTaskID:       `canonical_<input name="task">`,
					LifecycleStageID:      `stage_<script>alert(6)</script>`,
					Title:                 `<button onclick="x">task</button>`,
					Cell:                  `<form action="/mutate">cell</form>`,
					RiskClass:             `"><img src=x onerror=alert(7)>`,
					Status:                `work_task_<script>seeded</script>`,
					DependsOnRefs:         []string{`<a hx-post="/mutate">depends</a>`},
					ExpectedOutputs:       []string{`<textarea>task output</textarea>`},
					SourceRefs:            []string{`<a hx-post="/mutate">task source</a>`},
					RequiredRoles:         []string{`<input name="role">`},
					RoleContractRefs:      []string{`role_contract_<script>alert(8)</script>`},
					RequiredEvidence:      []string{`<select>required evidence</select>`},
					OutputContractRefs:    []string{`output_contract_<script>alert(9)</script>`},
					RuntimeEvidenceRefs:   []string{`runtime_ref_<script>alert(13)</script>`},
					RuntimeEvidenceStatus: `runtime_<script>recorded</script>`,
					RoleOutputContracts: []OpsCivilizationRoleOutputContract{
						{
							Role:              `<button onclick="x">output role</button>`,
							RequiredOutputs:   []string{`<textarea>output contract</textarea>`},
							AuthorityBoundary: `<form action="/authority">output boundary</form>`,
							CompletionGate:    `<a hx-post="/gate">output gate</a>`,
							EvidenceStatus:    `required_<script>not_observed</script>`,
						},
					},
					AgentExecutionPlan: []OpsHiveQueuedRunAgentPlanStep{
						{
							Role:              `<button onclick="x">role</button>`,
							RequiredOutputs:   []string{`<textarea>role output</textarea>`},
							AuthorityBoundary: `<form action="/merge">role boundary</form>`,
						},
					},
				},
				{
					ID:               `task_plan_<script>alert(10)</script>`,
					Title:            `<button onclick="x">fallback plan</button>`,
					LifecycleStageID: `fallback_stage_<script>alert(11)</script>`,
					Status:           `fallback_task_<script>seeded</script>`,
					AgentExecutionPlan: []OpsHiveQueuedRunAgentPlanStep{
						{
							Role:              `<button onclick="x">role</button>`,
							RequiredOutputs:   []string{`<textarea>role output</textarea>`},
							AuthorityBoundary: `<form action="/merge">role boundary</form>`,
						},
					},
				},
			},
			Artifacts: []OpsCivilizationAssemblyArtifactEvidence{
				{
					ID:         `artifact_<script>alert(1)</script>`,
					TaskRef:    `<form action="/mutate">task</form>`,
					Label:      `<button onclick="x">artifact</button>`,
					MediaType:  `"><img src=x onerror=alert(1)>`,
					SourceRefs: []string{`<a hx-post="/mutate">artifact ref</a>`},
				},
				{
					ID:         `stage_artifact_<script>alert(2)</script>`,
					TaskRef:    `stage_task_<form action="/mutate">task</form>`,
					Label:      `issue_scan_lifecycle_stage_stage_<script>alert(3)</script>`,
					MediaType:  `application/<img src=x onerror=alert(4)>`,
					SourceRefs: []string{`<a hx-post="/mutate">stage artifact ref</a>`},
				},
			},
		},
		QueuedRunRequest: &OpsHiveQueuedRunRequest{
			EventID:               `evt_<script>alert("event")</script>`,
			RunID:                 `run_<script>alert("run")</script>`,
			Title:                 `<button onclick="x">queued issue</button>`,
			Status:                `<form method="post">queued</form>`,
			TargetRepos:           []string{`transpara-ai/<script>site</script>`},
			AuthorityInitialLevel: `<input name="authority">`,
			AuthorityScope:        `<a hx-post="/approve">scope</a>`,
			SourceEventID:         `source_<script>alert("source")</script>`,
			BriefEventID:          `brief_<script>alert("brief")</script>`,
			EvidenceKind:          `<select><option>queued</option></select>`,
			BriefKind:             `<script>brief</script>`,
			LifecycleVersion:      `v<script>3</script>`,
			LifecycleEvidenceKind: `<button onclick="x">expected</button>`,
			SelectionPolicy: &OpsHiveQueuedRunSelectionPolicy{
				PolicyID:       `policy_<script>alert(12)</script>`,
				SelectedRank:   1,
				CandidateCount: 2,
				RankingInputs:  []string{`<a hx-post="/select">ranking</a>`, `<input name="rank">`},
				Rationale:      `<form action="/select">rationale</form>`,
			},
			DevelopmentLifecycle: []OpsHiveQueuedRunLifecycleStage{
				{
					ID:                `<script>stage</script>`,
					Name:              `<button onclick="x">Stage</button>`,
					RequiredRoles:     []string{`<input name="role">`},
					AuthorityBoundary: `<form action="/merge">boundary</form>`,
					CompletionGate:    `<a hx-post="/gate">gate</a>`,
					EvidenceStatus:    `<script>expected</script>`,
				},
			},
			AgentExecutionPlan: []OpsHiveQueuedRunAgentPlanStep{
				{
					ID:                `<script>step</script>`,
					StageID:           `<script>stage</script>`,
					Role:              `<button onclick="x">implementer</button>`,
					CanOperate:        true,
					Objective:         `<img src=x onerror=alert(1)>`,
					RequiredOutputs:   []string{`<textarea>output</textarea>`},
					AuthorityBoundary: `<form action="/merge">boundary</form>`,
					CompletionGate:    `<a hx-post="/gate">gate</a>`,
					EvidenceStatus:    `<script>expected</script>`,
				},
			},
		},
		IssueIntakeProjection: OpsCivilizationIssueIntakeProjection{
			Status:            `<script>available</script>`,
			Summary:           `<button onclick="x">issue summary</button>`,
			SourceRefs:        []string{`issue_source_<script>alert("issue-source")</script>`},
			ScannerBoundaries: []string{`<form action="/scanner">scanner boundary</form>`},
			Issues: []OpsCivilizationIssueIntakeProjected{
				{
					Repo:              `transpara-ai/<script>issue</script>`,
					Number:            114,
					PrimaryRepo:       `transpara-ai/<input name="repo">`,
					TouchedSubstrate:  `<form action="/substrate">substrate</form>`,
					TouchedSubstrates: []string{`<input name="substrate-input">`},
					RiskClass:         `<select><option>risk</option></select>`,
					RiskClasses:       []string{`<form action="/risk-input">risk input</form>`},
					UnrecognizedRisk:  []string{`<script>unknownRisk()</script>`},
					Readiness:         `<script>ready</script>`,
					ReadinessStates:   []string{`<button onclick="x">readiness input</button>`},
					AuthorityBoundary: `<a hx-post="/authority">authority</a>`,
					SourceRefs:        []string{`github:<script>source</script>`},
				},
			},
			Groups: []OpsCivilizationIssueIntakeGroupProjected{
				{
					GroupID:          `group_<script>alert("group")</script>`,
					Summary:          `<button onclick="x">group summary</button>`,
					PrimaryRepo:      `transpara-ai/<input name="group-repo">`,
					TouchedSubstrate: `<form action="/group-substrate">group substrate</form>`,
					RiskClass:        `<select><option>group risk</option></select>`,
					Readiness:        `<script>group ready</script>`,
					Recommendation:   `<a hx-post="/group">group recommendation</a>`,
					IssueRefs:        []OpsCivilizationIssueRef{{Repo: `transpara-ai/<script>group-issue</script>`, Number: 116}},
					Blockers:         []string{`<form action="/blocker">group blocker</form>`},
				},
			},
		},
		IssueScanProjection: OpsCivilizationIssueScanProjection{
			Runs: []OpsCivilizationIssueScanRunProjected{
				{
					RunID:            `kanban_run_<script>alert("kanban")</script>`,
					FactoryOrderID:   `kanban_fo_<input name="fo">`,
					LifecycleVersion: `v<script>4</script>`,
					State:            `human_action`,
					TargetIssue:      OpsCivilizationIssueRef{Repo: `transpara-ai/<script>docs</script>`, Number: 172},
					SelectedIssue:    OpsCivilizationIssueRef{Repo: `transpara-ai/<script>docs</script>`, Number: 172},
					CandidateIssues:  []OpsCivilizationIssueRef{{Repo: `transpara-ai/<script>site</script>`, Number: 115}},
					SourceRefs:       []string{`kanban_source_<script>alert("source")</script>`},
					EvidenceRefs:     []string{`kanban_evidence_<script>alert("evidence")</script>`},
				},
			},
			Stages: []OpsCivilizationIssueScanStageProjected{
				{
					RunID:             `kanban_run_<script>alert("kanban")</script>`,
					FactoryOrderID:    `kanban_fo_<input name="fo">`,
					StageID:           `stage_<script>alert("stage")</script>`,
					StageNumber:       1,
					StageCount:        7,
					CanonicalTaskID:   `kanban_task_<input name="stage">`,
					TaskID:            `kanban_event_<script>alert("task")</script>`,
					CurrentState:      `human_action`,
					CompletionGate:    `<a hx-post="/wake">gate</a>`,
					AuthorityBoundary: `<form action="/hive">boundary</form>`,
					AssignedAgentIDs:  []string{`<button onclick="x">agent</button>`},
					TouchingAgentIDs:  []string{`<script>touch()</script>`},
					EvidenceRefs:      []string{`stage_evidence_<script>alert("stage-evidence")</script>`},
					SourceRefs:        []string{`stage_source_<script>alert("stage-source")</script>`},
				},
			},
			Blockers: []OpsCivilizationIssueScanBlockerProjected{
				{
					RunID:          `kanban_run_<script>alert("kanban")</script>`,
					StageID:        `stage_<script>alert("stage")</script>`,
					BlockerType:    `needs_human_scope`,
					Reason:         `<script>reason()</script>`,
					RequiredAction: `<button onclick="x">clarify</button>`,
					EvidenceRefs:   []string{`blocker_evidence_<script>alert("blocker-evidence")</script>`},
					SourceRefs:     []string{`blocker_source_<script>alert("blocker-source")</script>`},
				},
			},
			Lineage: []OpsCivilizationIssueScanLineageProjected{
				{
					RunID:            `kanban_run_<script>alert("kanban")</script>`,
					StageID:          `stage_<script>alert("stage")</script>`,
					CanonicalTaskID:  `kanban_task_<input name="stage">`,
					PrimaryTaskID:    `primary_<script>task</script>`,
					TaskIDs:          []string{`primary_<script>task</script>`, `dup_<input name="dup">`},
					DuplicateTaskIDs: []string{`dup_<input name="dup">`},
					DuplicateOf:      `<form action="/canonical">canonical</form>`,
					SourceRefs:       []string{`lineage_source_<script>alert("lineage")</script>`},
				},
			},
		},
		WithheldOrUnavailableFields: []OpsCivilizationAssemblyUnavailableField{
			{Field: `authority_state`, Status: opsCivilizationFieldUnavailable, Reason: `<select><option>missing</option></select>`},
		},
		ValidationRefs: []string{`validation_<script>alert("validation")</script>`},
	}

	data := buildOpsCivilizationAssemblyDataFromProjection(projection, now)
	var body strings.Builder
	if err := opsCivilizationAssembly(data).Render(context.Background(), &body); err != nil {
		t.Fatalf("render projection component: %v", err)
	}
	html := body.String()
	assertNoCivilizationMutationControls(t, html)
	for _, escaped := range []string{"&lt;button", "&lt;form", "&lt;select", "&lt;a", "&lt;script", "&lt;input"} {
		if !strings.Contains(html, escaped) {
			t.Fatalf("rendered HTML does not include escaped hostile marker %q: %s", escaped, html)
		}
	}
	for _, escaped := range []string{"task_&lt;script&gt;", "canonical_&lt;input name=&#34;task&#34;&gt;", "stage_&lt;script&gt;", "&lt;button onclick=&#34;x&#34;&gt;task", "&lt;form action=&#34;/mutate&#34;&gt;cell", "&#34;&gt;&lt;img src=x onerror=alert(7)&gt;", "work task &lt;script&gt;seeded&lt;/script&gt;", "runtime &lt;script&gt;recorded&lt;/script&gt;", "runtime_ref_&lt;script&gt;alert(13)&lt;/script&gt;", "&lt;a hx-post=&#34;/mutate&#34;&gt;depends", "&lt;textarea&gt;task output&lt;/textarea&gt;", "&lt;a hx-post=&#34;/mutate&#34;&gt;task source", "&lt;input name=&#34;role&#34;&gt;", "role_contract_&lt;script&gt;", "&lt;button onclick=&#34;x&#34;&gt;role", "&lt;textarea&gt;role output&lt;/textarea&gt;", "&lt;form action=&#34;/merge&#34;&gt;role boundary&lt;/form&gt;", "&lt;select&gt;required evidence&lt;/select&gt;", "output_contract_&lt;script&gt;", "&lt;button onclick=&#34;x&#34;&gt;output role", "&lt;textarea&gt;output contract&lt;/textarea&gt;", "&lt;form action=&#34;/authority&#34;&gt;output boundary&lt;/form&gt;", "&lt;a hx-post=&#34;/gate&#34;&gt;output gate&lt;/a&gt;", "required &lt;script&gt;not observed&lt;/script&gt;", "artifact_&lt;script&gt;", "stage_artifact_&lt;script&gt;", "stage_&lt;script&gt;", "stage_task_&lt;form", "&lt;button onclick=&#34;x&#34;&gt;artifact", "&#34;&gt;&lt;img src=x onerror=alert(1)&gt;", "application/&lt;img src=x onerror=alert(4)&gt;", "&lt;a hx-post=&#34;/mutate&#34;&gt;artifact ref", "evt_&lt;script&gt;", "run_&lt;script&gt;", "source_&lt;script&gt;", "brief_&lt;script&gt;", "validation_&lt;script&gt;", "&lt;button onclick=&#34;x&#34;&gt;queued issue", "transpara-ai/&lt;script&gt;site", "policy_&lt;script&gt;", "&lt;a hx-post=&#34;/select&#34;&gt;ranking&lt;/a&gt;", "&lt;input name=&#34;rank&#34;&gt;", "&lt;form action=&#34;/select&#34;&gt;rationale&lt;/form&gt;", "&lt;button onclick=&#34;x&#34;&gt;issue summary&lt;/button&gt;", "transpara-ai/&lt;script&gt;issue&lt;/script&gt;#114", "transpara-ai/&lt;input name=&#34;repo&#34;&gt;", "&lt;form action=&#34;/substrate&#34;&gt;substrate&lt;/form&gt;", "&lt;input name=&#34;substrate-input&#34;&gt;", "&lt;form action=&#34;/risk-input&#34;&gt;risk input&lt;/form&gt;", "&lt;script&gt;unknownRisk()&lt;/script&gt;", "&lt;button onclick=&#34;x&#34;&gt;readiness input&lt;/button&gt;", "&lt;a hx-post=&#34;/authority&#34;&gt;authority&lt;/a&gt;", "&lt;form action=&#34;/scanner&#34;&gt;scanner boundary&lt;/form&gt;", "group_&lt;script&gt;alert(&#34;group&#34;)&lt;/script&gt;", "&lt;a hx-post=&#34;/group&#34;&gt;group recommendation&lt;/a&gt;", "&lt;form action=&#34;/blocker&#34;&gt;group blocker&lt;/form&gt;", "&lt;textarea&gt;output&lt;/textarea&gt;", "&lt;img src=x onerror=alert(1)&gt;", "kanban_run_&lt;script&gt;alert(&#34;kanban&#34;)&lt;/script&gt;", "kanban_task_&lt;input name=&#34;stage&#34;&gt;", "&lt;a hx-post=&#34;/wake&#34;&gt;gate&lt;/a&gt;", "&lt;form action=&#34;/hive&#34;&gt;boundary&lt;/form&gt;", "&lt;button onclick=&#34;x&#34;&gt;agent&lt;/button&gt;", "&lt;script&gt;touch()&lt;/script&gt;", "&lt;button onclick=&#34;x&#34;&gt;clarify&lt;/button&gt;", "&lt;script&gt;reason()&lt;/script&gt;", "primary_&lt;script&gt;task&lt;/script&gt;", "dup_&lt;input name=&#34;dup&#34;&gt;", "&lt;form action=&#34;/canonical&#34;&gt;canonical&lt;/form&gt;"} {
		if !strings.Contains(html, escaped) {
			t.Fatalf("rendered HTML does not include escaped queued lifecycle marker %q: %s", escaped, html)
		}
	}
	for _, raw := range []string{"<script", "<button", "<form", "<input", "<select", "<textarea", "<img", "<a "} {
		if strings.Contains(html, raw) {
			t.Fatalf("rendered HTML contains unescaped hostile tag %q: %s", raw, html)
		}
	}
}

func TestBuildOpsCivilizationPreservesUnavailablePartialFailedAndStaleStates(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		projection    *OpsCivilizationAssemblyProjection
		wantStatus    string
		wantFreshness string
		wantFinding   string
		wantRefGroup  string
		wantRef       string
	}{
		{
			name:          "missing projection",
			projection:    nil,
			wantStatus:    opsCivilizationProjectionStatusUnavailable,
			wantFreshness: "unknown",
			wantFinding:   "EventGraph projection-shaped input was not available",
		},
		{
			name: "partial projection",
			projection: &OpsCivilizationAssemblyProjection{
				ProjectionID:     "civ-proj-partial",
				GeneratedAt:      now.Add(-time.Hour),
				DerivationStatus: opsCivilizationProjectionStatusPartial,
				OpenGateSummary: []OpsCivilizationAssemblyGateSummary{
					{ID: "gate_t", GateName: "Gate T", Status: "open"},
				},
				WithheldOrUnavailableFields: []OpsCivilizationAssemblyUnavailableField{
					{Field: "authority_state", Status: opsCivilizationFieldUnavailable, Reason: "source records missing"},
				},
			},
			wantStatus:    opsCivilizationProjectionStatusPartial,
			wantFreshness: "current",
			wantFinding:   "1 open gate(s) remain projected",
			wantRefGroup:  "withheld/unavailable fields",
			wantRef:       "authority_state: source records missing",
		},
		{
			name: "failed projection",
			projection: &OpsCivilizationAssemblyProjection{
				ProjectionID:     "civ-proj-failed",
				GeneratedAt:      now.Add(-time.Hour),
				DerivationStatus: opsCivilizationProjectionStatusFailed,
				FailureReasons:   []string{"dangling authority reference"},
			},
			wantStatus:    opsCivilizationProjectionStatusFailed,
			wantFreshness: "current",
			wantFinding:   "Projection failure reason: dangling authority reference",
		},
		{
			name: "stale complete projection",
			projection: &OpsCivilizationAssemblyProjection{
				ProjectionID:     "civ-proj-stale",
				GeneratedAt:      now.Add(-25 * time.Hour),
				DerivationStatus: opsCivilizationProjectionStatusComplete,
			},
			wantStatus:    opsCivilizationProjectionStatusComplete,
			wantFreshness: "stale",
			wantFinding:   "Projection freshness: stale.",
		},
		{
			name: "future dated projection",
			projection: &OpsCivilizationAssemblyProjection{
				ProjectionID:     "civ-proj-skewed",
				GeneratedAt:      now.Add(10 * time.Minute),
				DerivationStatus: opsCivilizationProjectionStatusComplete,
			},
			wantStatus:    opsCivilizationProjectionStatusComplete,
			wantFreshness: "skewed",
			wantFinding:   "Projection freshness: skewed.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := buildOpsCivilizationAssemblyDataFromProjection(tt.projection, now)
			if data.ProjectionStatus != tt.wantStatus {
				t.Fatalf("projection status = %q, want %q", data.ProjectionStatus, tt.wantStatus)
			}
			if data.ProjectionFreshness != tt.wantFreshness {
				t.Fatalf("projection freshness = %q, want %q", data.ProjectionFreshness, tt.wantFreshness)
			}
			if !findingContains(data, tt.wantFinding) {
				t.Fatalf("findings do not contain %q: %+v", tt.wantFinding, data.Civilization.Findings)
			}
			if tt.wantRefGroup != "" && !referenceGroupContains(data, tt.wantRefGroup, tt.wantRef) {
				t.Fatalf("reference group %q missing %q: %+v", tt.wantRefGroup, tt.wantRef, data.ReferenceGroups)
			}
		})
	}
}

func TestOpsCivilizationRouteIsGetOnly(t *testing.T) {
	h, _, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/civilization", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /ops/civilization: status = %d, want %d; body: %s", w.Code, http.StatusMethodNotAllowed, w.Body.String())
	}
}

func statusRowValue(data *OpsCivilizationAssemblyData, label string) string {
	for _, row := range data.StatusRows {
		if row.Label == label {
			return row.Value
		}
	}
	return ""
}

func boundaryState(data *OpsCivilizationAssemblyData, label string) string {
	for _, item := range data.Boundary {
		if item.Label == label {
			return item.State
		}
	}
	return ""
}

func referenceGroupContains(data *OpsCivilizationAssemblyData, label string, ref string) bool {
	for _, group := range data.ReferenceGroups {
		if group.Label != label {
			continue
		}
		for _, item := range group.Refs {
			if item == ref {
				return true
			}
		}
	}
	return false
}

func sliceContains(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}

func civilizationArtifactByLabel(items []OpsCivilizationAssemblyArtifactEvidence, label string) *OpsCivilizationAssemblyArtifactEvidence {
	for i := range items {
		if items[i].Label == label {
			return &items[i]
		}
	}
	return nil
}

func civilizationTaskByID(items []OpsCivilizationAssemblyTaskEvidence, id string) *OpsCivilizationAssemblyTaskEvidence {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

func roleOutputContractsContain(items []OpsCivilizationRoleOutputContract, role string, output string) bool {
	for _, item := range items {
		if item.Role == role && sliceContains(item.RequiredOutputs, output) {
			return true
		}
	}
	return false
}

func issueGuardrailContains(items []OpsCivilizationIssueGuardrail, label string, state string) bool {
	for _, item := range items {
		if item.Label == label && item.State == state {
			return true
		}
	}
	return false
}

func findingContains(data *OpsCivilizationAssemblyData, needle string) bool {
	for _, finding := range data.Civilization.Findings {
		if strings.Contains(finding, needle) {
			return true
		}
	}
	return false
}

func issueScanKanbanCardByStage(kanban OpsCivilizationIssueScanKanban, runID string, stageID string) *OpsCivilizationIssueScanKanbanCard {
	for ci := range kanban.Columns {
		for i := range kanban.Columns[ci].Cards {
			card := &kanban.Columns[ci].Cards[i]
			if card.RunID == runID && card.StageID == stageID {
				return card
			}
		}
	}
	return nil
}

func issueScanKanbanCardCount(kanban OpsCivilizationIssueScanKanban) int {
	total := 0
	for _, column := range kanban.Columns {
		total += len(column.Cards)
	}
	return total
}

func assertNoCivilizationMutationControls(t *testing.T, surface string) {
	t.Helper()
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`(?i)<form\b`),
		regexp.MustCompile(`(?i)<button\b`),
		regexp.MustCompile(`(?i)<(input|select|textarea)\b`),
		regexp.MustCompile(`(?i)<[^>]+hx-[a-z-]+\s*=`),
		regexp.MustCompile(`(?i)<[^>]+method\s*=\s*['"]?post`),
		regexp.MustCompile(`(?i)<[^>]+\son[a-z]+\s*=`),
		regexp.MustCompile(`(?i)<a\b[^>]*\bhx-[a-z-]+\s*=`),
	}
	for _, re := range forbidden {
		if re.MatchString(surface) {
			t.Fatalf("civilization assembly body contains mutation control marker %q", re)
		}
	}
}

func civilizationAssemblySurface(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `data-civilization-assembly="read-only"`)
	if start < 0 {
		t.Fatal("GET /ops/civilization: missing read-only assembly marker")
	}
	surface := body[start:]
	if end := strings.Index(surface, "</main>"); end >= 0 {
		surface = surface[:end]
	}
	return surface
}

func githubCanonicalSurface(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `data-github-canonical-migration="read-only"`)
	if start < 0 {
		t.Fatal("GET /ops/github-canonical: missing read-only migration marker")
	}
	surface := body[start:]
	if end := strings.Index(surface, "</main>"); end >= 0 {
		surface = surface[:end]
	}
	return surface
}

func TestHandlerCreateSpace(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	// handleCreateSpace derives the slug from slugify(name) and ignores any
	// explicit slug form field, so make the name unique to keep the slug unique.
	name := fmt.Sprintf("Handler Test %d", time.Now().UnixNano())
	expectedSlug := slugify(name)

	form := url.Values{}
	form.Set("name", name)
	form.Set("description", "Testing handlers")
	form.Set("kind", "project")
	form.Set("visibility", "public")

	req := httptest.NewRequest("POST", "/app/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusSeeOther, w.Body.String())
	}

	space, err := store.GetSpaceBySlug(t.Context(), expectedSlug)
	if err != nil {
		t.Fatalf("get space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	if space.Name != name {
		t.Errorf("name = %q, want %q", space.Name, name)
	}
	if space.Visibility != "public" {
		t.Errorf("visibility = %q, want %q", space.Visibility, "public")
	}
}

func TestHandlerOp(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	// Create a space first (clean up stale data from prior runs).
	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-op-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-op-test", "Op Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("intend_json", func(t *testing.T) {
		body := `{"op":"intend","title":"Test Task","description":"A task","priority":"high"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		if result["op"] != "intend" {
			t.Errorf("op = %v, want intend", result["op"])
		}
		node := result["node"].(map[string]any)
		if node["title"] != "Test Task" {
			t.Errorf("title = %v, want Test Task", node["title"])
		}
		if node["priority"] != "high" {
			t.Errorf("priority = %v, want high", node["priority"])
		}
	})

	t.Run("intend_body_field", func(t *testing.T) {
		// body key must be read (not silently dropped as with description-only fallback)
		payload := `{"op":"intend","title":"Body Field Task","body":"from body key"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["body"] != "from body key" {
			t.Errorf("body = %v, want 'from body key'", node["body"])
		}
	})

	t.Run("intend_description_fallback", func(t *testing.T) {
		// when body key is absent, description key must be used for the node body
		payload := `{"op":"intend","title":"Fallback Task","description":"from description key"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["body"] != "from description key" {
			t.Errorf("body = %v, want 'from description key'", node["body"])
		}
	})

	t.Run("intend_body_beats_description", func(t *testing.T) {
		// when both body and description are present, body wins
		payload := `{"op":"intend","title":"Priority Task","body":"from body","description":"from description"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["body"] != "from body" {
			t.Errorf("body = %v, want 'from body'", node["body"])
		}
	})

	t.Run("intend_kind_proposal", func(t *testing.T) {
		// kind=proposal must not be silently dropped to kind=task
		payload := `{"op":"intend","title":"Test Proposal","kind":"proposal"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["kind"] != KindProposal {
			t.Errorf("kind = %v, want %v", node["kind"], KindProposal)
		}
	})

	t.Run("express_json", func(t *testing.T) {
		body := `{"op":"express","title":"Test Post","body":"Hello world"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
	})

	t.Run("converse_json", func(t *testing.T) {
		body := `{"op":"converse","title":"Test Chat","body":"Let's discuss","participants":"alice,bob"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["title"] != "Test Chat" {
			t.Errorf("title = %v, want Test Chat", node["title"])
		}
	})

	t.Run("respond_json", func(t *testing.T) {
		// Create a parent first.
		parent, _ := store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindThread, Title: "Parent", Author: "Tester", AuthorID: "test-user-1",
		})

		body := `{"op":"respond","parent_id":"` + parent.ID + `","body":"A reply"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
	})

	t.Run("report_json", func(t *testing.T) {
		parent, _ := store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindPost, Title: "Flaggable", Author: "Tester", AuthorID: "test-user-1",
		})
		body := `{"op":"report","node_id":"` + parent.ID + `","reason":"inappropriate content"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}
	})

	t.Run("unknown_op", func(t *testing.T) {
		body := `{"op":"nonexistent"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestHandleHiveSiteOpsReturnsMarkdownIntake(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	slug := fmt.Sprintf("site-ops-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Site Ops", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	markdown := "# Kickoff\n\nBuild the Civilization ingestion bridge."
	node, err := store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		Title:    "Kickoff idea",
		Body:     markdown,
		Priority: PriorityHigh,
		Author:   "Tester",
		AuthorID: "test-user-1",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if _, err := store.RecordOp(t.Context(), space.ID, node.ID, "Tester", "test-user-1", "intend", nil); err != nil {
		t.Fatalf("record op: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/hive/site-ops?space="+url.QueryEscape(slug), nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result struct {
		Ops []struct {
			ID        string          `json:"id"`
			SpaceID   string          `json:"space_id"`
			NodeID    string          `json:"node_id"`
			NodeTitle string          `json:"node_title"`
			ActorKind string          `json:"actor_kind"`
			Op        string          `json:"op"`
			Payload   json.RawMessage `json:"payload"`
		} `json:"ops"`
	}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result.Ops) != 1 {
		t.Fatalf("ops = %d, want 1; body: %s", len(result.Ops), w.Body.String())
	}
	got := result.Ops[0]
	if got.SpaceID != space.ID || got.NodeID != node.ID || got.NodeTitle != "Kickoff idea" || got.ActorKind != "human" || got.Op != "intend" {
		t.Fatalf("op metadata = %#v", got)
	}
	var payload map[string]string
	if err := json.Unmarshal(got.Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["body"] != markdown {
		t.Fatalf("payload body = %q, want Markdown body", payload["body"])
	}
	if payload["priority"] != PriorityHigh {
		t.Fatalf("payload priority = %q, want %q", payload["priority"], PriorityHigh)
	}
}

func TestHandleHiveSiteOpsRejectsPrivateSpaceForNonMember(t *testing.T) {
	_, store, _ := testHandlers(t)

	otherUser := &auth.User{ID: "test-user-2", Name: "Other", Email: "other@test.com", Kind: "human"}
	otherWrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.ContextWithUser(r.Context(), otherUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	h := NewHandlers(store, otherWrap, otherWrap)

	mux := http.NewServeMux()
	h.Register(mux)

	slug := fmt.Sprintf("site-ops-private-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Private Site Ops", "", "test-user-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	node, err := store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		Title:    "Private kickoff",
		Body:     "# Secret kickoff",
		Priority: PriorityHigh,
		Author:   "Tester",
		AuthorID: "test-user-1",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	if _, err := store.RecordOp(t.Context(), space.ID, node.ID, "Tester", "test-user-1", "intend", nil); err != nil {
		t.Fatalf("record op: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/hive/site-ops?space="+url.QueryEscape(slug), nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusNotFound, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "Secret kickoff") {
		t.Fatalf("private payload leaked in response: %s", w.Body.String())
	}
}

func TestHandlerConversationDetail(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-convo-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-convo-test", "Convo Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	convo, err := store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindConversation, Title: "Test Convo",
		Author: "Tester", AuthorID: "test-user-1", Tags: []string{"test-user-1"},
	})
	if err != nil {
		t.Fatalf("create convo: %v", err)
	}

	// Add a message.
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, ParentID: convo.ID, Kind: KindComment,
		Body: "Hello!", Author: "Tester", AuthorID: "test-user-1",
	})

	// GET conversation detail as JSON.
	req := httptest.NewRequest("GET", "/app/handler-convo-test/conversation/"+convo.ID, nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)
	messages := result["messages"].([]any)
	if len(messages) != 1 {
		t.Errorf("got %d messages, want 1", len(messages))
	}
}

func TestParseMessageSearch(t *testing.T) {
	tests := []struct {
		input    string
		wantBody string
		wantFrom string
	}{
		{"hello world", "hello world", ""},
		{"from:alice", "", "alice"},
		{"hello from:alice", "hello", "alice"},
		{"from:bob goodbye", "goodbye", "bob"},
		{"hello from:alice world", "hello world", "alice"},
		{"", "", ""},
	}

	for _, tt := range tests {
		body, from := parseMessageSearch(tt.input)
		if body != tt.wantBody {
			t.Errorf("parseMessageSearch(%q) body = %q, want %q", tt.input, body, tt.wantBody)
		}
		if from != tt.wantFrom {
			t.Errorf("parseMessageSearch(%q) from = %q, want %q", tt.input, from, tt.wantFrom)
		}
	}
}

func TestHandlerDocumentEdit(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-doc-edit-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	// Private space owned by test-user-1 (member).
	space, err := store.CreateSpace(t.Context(), "handler-doc-edit-test", "Doc Edit Test", "", "test-user-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	doc, err := store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument, Title: "Original Title",
		Body: "Original body.", Author: "Tester", AuthorID: "test-user-1",
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	t.Run("get_edit_form_member", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/app/handler-doc-edit-test/document/"+doc.ID+"/edit", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}
	})

	t.Run("post_edit_member", func(t *testing.T) {
		form := url.Values{}
		form.Set("title", "Updated Title")
		form.Set("body", "Updated body content.")

		req := httptest.NewRequest("POST", "/app/handler-doc-edit-test/document/"+doc.ID+"/edit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["title"] != "Updated Title" {
			t.Errorf("title = %v, want Updated Title", node["title"])
		}
		if node["body"] != "Updated body content." {
			t.Errorf("body = %v, want Updated body content.", node["body"])
		}
	})

	t.Run("non_member_rejected", func(t *testing.T) {
		// Create a second handler set with a different user who is not the space owner.
		otherUser := &auth.User{ID: "other-user-99", Name: "Other", Email: "other@test.com", Kind: "human"}
		otherWrap := func(next http.HandlerFunc) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := auth.ContextWithUser(r.Context(), otherUser)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		}
		otherH := NewHandlers(store, otherWrap, otherWrap)
		otherMux := http.NewServeMux()
		otherH.Register(otherMux)

		form := url.Values{}
		form.Set("title", "Should Not Update")
		form.Set("body", "Should not be saved.")

		req := httptest.NewRequest("POST", "/app/handler-doc-edit-test/document/"+doc.ID+"/edit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		otherMux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (non-member should be rejected)", w.Code, http.StatusNotFound)
		}
	})
}

func TestHandlerDocuments(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-docs-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-docs-test", "Docs Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("create_document", func(t *testing.T) {
		body := `{"op":"intend","title":"Architecture Guide","description":"Our system design","kind":"document"}`
		req := httptest.NewRequest("POST", "/app/handler-docs-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["kind"] != KindDocument {
			t.Errorf("kind = %v, want %q", node["kind"], KindDocument)
		}
		if node["title"] != "Architecture Guide" {
			t.Errorf("title = %v, want Architecture Guide", node["title"])
		}
	})

	t.Run("list_documents", func(t *testing.T) {
		store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindDocument, Title: "List Doc",
			Author: "Tester", AuthorID: "test-user-1",
		})

		req := httptest.NewRequest("GET", "/app/handler-docs-test/documents", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		docs := result["documents"].([]any)
		if len(docs) == 0 {
			t.Error("got 0 documents, want at least 1")
		}
	})

	t.Run("document_detail", func(t *testing.T) {
		doc, err := store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindDocument, Title: "Detail Doc",
			Body: "# Hello\nThis is content.", Author: "Tester", AuthorID: "test-user-1",
		})
		if err != nil {
			t.Fatalf("create document: %v", err)
		}

		req := httptest.NewRequest("GET", "/app/handler-docs-test/node/"+doc.ID, nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["id"] != doc.ID {
			t.Errorf("id = %v, want %v", node["id"], doc.ID)
		}
		if node["kind"] != KindDocument {
			t.Errorf("kind = %v, want %q", node["kind"], KindDocument)
		}
	})

	t.Run("search_documents", func(t *testing.T) {
		store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindDocument, Title: "Searchable Wiki Page",
			Body: "Contains the word searchable", Author: "Tester", AuthorID: "test-user-1",
		})
		store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindDocument, Title: "Other Doc",
			Author: "Tester", AuthorID: "test-user-1",
		})

		req := httptest.NewRequest("GET", "/app/handler-docs-test/documents?q=Searchable+Wiki", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		docs := result["documents"].([]any)
		if len(docs) == 0 {
			t.Error("got 0 results for search, want at least 1")
		}
		first := docs[0].(map[string]any)
		if first["title"] != "Searchable Wiki Page" {
			t.Errorf("first result title = %v, want Searchable Wiki Page", first["title"])
		}
	})
}

func TestHandlerQuestions(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-qa-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-qa-test", "Q&A Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("create_question", func(t *testing.T) {
		body := `{"op":"intend","title":"How does the event graph work?","description":"Looking for an overview of the architecture","kind":"question"}`
		req := httptest.NewRequest("POST", "/app/handler-qa-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["kind"] != KindQuestion {
			t.Errorf("kind = %v, want %q", node["kind"], KindQuestion)
		}
		if node["title"] != "How does the event graph work?" {
			t.Errorf("title = %v, want How does the event graph work?", node["title"])
		}
	})

	t.Run("list_questions", func(t *testing.T) {
		store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindQuestion, Title: "What is causality?",
			Author: "Tester", AuthorID: "test-user-1",
		})

		req := httptest.NewRequest("GET", "/app/handler-qa-test/questions", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		questions := result["questions"].([]any)
		if len(questions) == 0 {
			t.Error("got 0 questions, want at least 1")
		}
	})

	t.Run("question_detail", func(t *testing.T) {
		q, err := store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindQuestion, Title: "Why use signed events?",
			Body: "Curious about the integrity model.", Author: "Tester", AuthorID: "test-user-1",
		})
		if err != nil {
			t.Fatalf("create question: %v", err)
		}

		req := httptest.NewRequest("GET", "/app/handler-qa-test/questions/"+q.ID, nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["question"].(map[string]any)
		if node["id"] != q.ID {
			t.Errorf("id = %v, want %v", node["id"], q.ID)
		}
		if node["kind"] != KindQuestion {
			t.Errorf("kind = %v, want %q", node["kind"], KindQuestion)
		}
	})
}

// TestHandlerExpressQuestion verifies that express op with kind=question creates a KindQuestion.
func TestHandlerExpressQuestion(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-express-qa-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-express-qa-test", "Express Q&A Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("express_kind_question_creates_question", func(t *testing.T) {
		body := `{"op":"express","title":"What is an event graph?","body":"Looking for a brief overview","kind":"question"}`
		req := httptest.NewRequest("POST", "/app/handler-express-qa-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["kind"] != KindQuestion {
			t.Errorf("kind = %v, want %q", node["kind"], KindQuestion)
		}
		if node["title"] != "What is an event graph?" {
			t.Errorf("title = %v, want 'What is an event graph?'", node["title"])
		}
		if result["op"] != "express" {
			t.Errorf("op = %v, want 'express'", result["op"])
		}
	})

	t.Run("express_no_kind_creates_post", func(t *testing.T) {
		body := `{"op":"express","body":"A post without a kind"}`
		req := httptest.NewRequest("POST", "/app/handler-express-qa-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["kind"] != KindPost {
			t.Errorf("kind = %v, want %q", node["kind"], KindPost)
		}
	})
}

// TestHandlerKnowledgeLens verifies the knowledge lens returns documents and questions
// alongside claims, with LIMIT bounds applied (Invariant 13: BOUNDED).
func TestHandlerKnowledgeLens(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-knowledge-lens-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-knowledge-lens-test", "Knowledge Lens Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	// Seed a document, a question, and a claim.
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument, Title: "Team Playbook",
		Body: "How we work.", Author: "Tester", AuthorID: "test-user-1", AuthorKind: "human",
	})
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindQuestion, Title: "What is the mission?",
		Author: "Tester", AuthorID: "test-user-1", AuthorKind: "human",
	})
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindClaim, Title: "Event graphs are scalable",
		Author: "Tester", AuthorID: "test-user-1", AuthorKind: "human",
	})

	req := httptest.NewRequest("GET", "/app/handler-knowledge-lens-test/knowledge", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)

	docs, ok := result["documents"].([]any)
	if !ok {
		t.Fatal("knowledge lens response missing 'documents' field")
	}
	if len(docs) == 0 {
		t.Error("knowledge lens: got 0 documents, want at least 1")
	}

	questions, ok := result["questions"].([]any)
	if !ok {
		t.Fatal("knowledge lens response missing 'questions' field")
	}
	if len(questions) == 0 {
		t.Error("knowledge lens: got 0 questions, want at least 1")
	}

	// Invariant 13 (BOUNDED): both lists must be within the declared LIMIT of 50.
	const knowledgeLensLimit = 50
	if len(docs) > knowledgeLensLimit {
		t.Errorf("knowledge lens: documents count %d exceeds BOUNDED limit %d", len(docs), knowledgeLensLimit)
	}
	if len(questions) > knowledgeLensLimit {
		t.Errorf("knowledge lens: questions count %d exceeds BOUNDED limit %d", len(questions), knowledgeLensLimit)
	}
}

// TestHandlerKnowledgeTabs verifies that the /knowledge route handles ?tab=docs
// and ?tab=qa routing correctly — returns 200 and no server errors.
func TestHandlerKnowledgeTabs(t *testing.T) {
	h, store, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "knowledge-tabs-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "knowledge-tabs-test", "Knowledge Tabs Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	// Seed a document and a question.
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument, Title: "Test Doc",
		Body: "Content.", Author: "Tester", AuthorID: "test-user-1",
	})
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindQuestion, Title: "Test Question",
		Author: "Tester", AuthorID: "test-user-1",
	})

	for _, tab := range []string{"docs", "qa", "claims", ""} {
		tab := tab
		t.Run("tab_"+tab, func(t *testing.T) {
			path := "/app/knowledge-tabs-test/knowledge"
			if tab != "" {
				path += "?tab=" + tab
			}
			req := httptest.NewRequest("GET", path, nil)
			// HTML request — no application/json Accept header.
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("tab=%q: status = %d, want %d", tab, w.Code, http.StatusOK)
			}
		})
	}
}

// TestHandlerKnowledgeDocsTabRows confirms GET ?tab=docs returns 200 and
// renders the seeded document title in the HTML body.
func TestHandlerKnowledgeDocsTabRows(t *testing.T) {
	h, store, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "knowledge-docs-rows-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "knowledge-docs-rows-test", "Knowledge Docs Rows", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument, Title: "Unique Doc Row Title",
		Body: "Excerpt content for the document row.", Author: "Tester", AuthorID: "test-user-1", AuthorKind: "human",
	})

	req := httptest.NewRequest("GET", "/app/knowledge-docs-rows-test/knowledge?tab=docs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("docs tab: status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Unique Doc Row Title") {
		t.Error("docs tab: seeded document title not found in HTML response")
	}
}

// TestHandlerKnowledgeQATabRows confirms GET ?tab=qa returns 200 and renders
// the seeded question title with an Answered or Awaiting badge in the HTML body.
func TestHandlerKnowledgeQATabRows(t *testing.T) {
	h, store, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "knowledge-qa-rows-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "knowledge-qa-rows-test", "Knowledge QA Rows", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindQuestion, Title: "Unique QA Row Question",
		Author: "Tester", AuthorID: "test-user-1", AuthorKind: "human",
	})

	req := httptest.NewRequest("GET", "/app/knowledge-qa-rows-test/knowledge?tab=qa", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("qa tab: status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Unique QA Row Question") {
		t.Error("qa tab: seeded question title not found in HTML response")
	}
	if !strings.Contains(body, "Answered") && !strings.Contains(body, "Awaiting") {
		t.Error("qa tab: neither Answered nor Awaiting badge found in HTML response")
	}
}

func TestHandlerJoinViaInvite(t *testing.T) {
	_, store := testDB(t)

	testUser := &auth.User{ID: "joiner-1", Name: "Joiner", Email: "joiner@test.com", Kind: "human"}
	authWrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.ContextWithUser(r.Context(), testUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	// Passes request through without a user — handler sees "anonymous" and redirects with ?next=.
	anonWrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	slug := fmt.Sprintf("join-test-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Join Test", "", "owner-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	token, err := store.CreateInviteCode(t.Context(), space.ID, "owner-1", nil, 0)
	if err != nil {
		t.Fatalf("create invite code: %v", err)
	}

	t.Run("unauthenticated_redirect", func(t *testing.T) {
		h := NewHandlers(store, anonWrap, anonWrap)
		mux := http.NewServeMux()
		h.Register(mux)

		req := httptest.NewRequest("GET", "/join/"+token, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
		}
		want := "/auth/login?next=%2Fjoin%2F" + token
		if loc := w.Header().Get("Location"); loc != want {
			t.Errorf("Location = %q, want %q", loc, want)
		}
	})

	t.Run("valid_code_joins_user", func(t *testing.T) {
		h := NewHandlers(store, authWrap, authWrap)
		mux := http.NewServeMux()
		h.Register(mux)

		req := httptest.NewRequest("GET", "/join/"+token, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
		}
		want := "/app/" + space.Slug + "/board"
		if loc := w.Header().Get("Location"); loc != want {
			t.Errorf("Location = %q, want %q", loc, want)
		}
		if !store.IsMember(t.Context(), space.ID, testUser.ID) {
			t.Error("user should be a member after joining via invite")
		}
	})

	t.Run("invalid_code_404", func(t *testing.T) {
		h := NewHandlers(store, authWrap, authWrap)
		mux := http.NewServeMux()
		h.Register(mux)

		req := httptest.NewRequest("GET", "/join/nonexistent-token", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestHandlerCreateInviteHTMX(t *testing.T) {
	h, store, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	slug := fmt.Sprintf("htmx-invite-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "HTMX Invite Test", "", "test-user-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("owner_creates_invite_returns_html", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/app/"+slug+"/invites", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if body == "" {
			t.Error("expected HTML fragment in response body, got empty")
		}

		// Verify an invite was actually created in the store.
		invites, err := store.ListInvites(t.Context(), space.ID)
		if err != nil {
			t.Fatalf("list invites: %v", err)
		}
		if len(invites) == 0 {
			t.Error("expected invite to be created in store")
		}
	})

	t.Run("nonexistent_space_404", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/app/no-such-space/invites", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("non_owner_rejected", func(t *testing.T) {
		otherUser := &auth.User{ID: "other-user-99", Name: "Other", Email: "other@test.com", Kind: "human"}
		otherWrap := func(next http.HandlerFunc) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := auth.ContextWithUser(r.Context(), otherUser)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		}
		h2 := NewHandlers(store, otherWrap, otherWrap)
		mux2 := http.NewServeMux()
		h2.Register(mux2)

		req := httptest.NewRequest("POST", "/app/"+slug+"/invites", nil)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (non-owner should be rejected)", w.Code, http.StatusNotFound)
		}
	})

	t.Run("unauthenticated_rejected", func(t *testing.T) {
		anonWrap := func(next http.HandlerFunc) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
		h2 := NewHandlers(store, anonWrap, anonWrap)
		mux2 := http.NewServeMux()
		h2.Register(mux2)

		req := httptest.NewRequest("POST", "/app/"+slug+"/invites", nil)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (unauthenticated should be rejected)", w.Code, http.StatusNotFound)
		}
	})
}

func TestHandlerRevokeInvite(t *testing.T) {
	h, store, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	slug := fmt.Sprintf("revoke-invite-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Revoke Invite Test", "", "test-user-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("revoke_existing_invite", func(t *testing.T) {
		tok, err := store.CreateInviteCode(t.Context(), space.ID, "test-user-1", nil, 0)
		if err != nil {
			t.Fatalf("create invite: %v", err)
		}

		req := httptest.NewRequest("DELETE", "/app/"+slug+"/invites/"+tok, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		// Verify it is gone from the store.
		got, err := store.GetInviteCode(t.Context(), tok)
		if err != nil {
			t.Fatalf("get invite after revoke: %v", err)
		}
		if got != nil {
			t.Error("expected invite to be deleted, still present")
		}
	})

	t.Run("revoke_nonexistent_token_404", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/app/"+slug+"/invites/no-such-token", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("non_owner_cannot_revoke", func(t *testing.T) {
		tok, err := store.CreateInviteCode(t.Context(), space.ID, "test-user-1", nil, 0)
		if err != nil {
			t.Fatalf("create invite: %v", err)
		}
		t.Cleanup(func() { store.RevokeInvite(t.Context(), tok) })

		otherUser := &auth.User{ID: "other-user-99", Name: "Other", Email: "other@test.com", Kind: "human"}
		otherWrap := func(next http.HandlerFunc) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := auth.ContextWithUser(r.Context(), otherUser)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		}
		h2 := NewHandlers(store, otherWrap, otherWrap)
		mux2 := http.NewServeMux()
		h2.Register(mux2)

		req := httptest.NewRequest("DELETE", "/app/"+slug+"/invites/"+tok, nil)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (non-owner should be rejected)", w.Code, http.StatusNotFound)
		}
	})

	t.Run("unauthenticated_cannot_revoke", func(t *testing.T) {
		tok, err := store.CreateInviteCode(t.Context(), space.ID, "test-user-1", nil, 0)
		if err != nil {
			t.Fatalf("create invite: %v", err)
		}
		t.Cleanup(func() { store.RevokeInvite(t.Context(), tok) })

		anonWrap := func(next http.HandlerFunc) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
		h2 := NewHandlers(store, anonWrap, anonWrap)
		mux2 := http.NewServeMux()
		h2.Register(mux2)

		req := httptest.NewRequest("DELETE", "/app/"+slug+"/invites/"+tok, nil)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (unauthenticated should be rejected)", w.Code, http.StatusNotFound)
		}
	})
}

// TestHandlerConveneOp verifies that a convene op creates a KindCouncil node
// with the correct body and tags (agent IDs resolved by name).
func TestHandlerConveneOp(t *testing.T) {
	_, store := testDB(t)
	ctx := t.Context()

	// Create two agent users for the council.
	agentAID := "convene-agent-a-id"
	agentAName := "ConveneAgentA"
	agentBID := "convene-agent-b-id"
	agentBName := "ConveneAgentB"
	for _, row := range []struct{ id, name string }{{agentAID, agentAName}, {agentBID, agentBName}} {
		store.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, row.id)
		_, err := store.db.ExecContext(ctx,
			`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
			row.id, "agent:"+row.name, row.name+"@test.transpara.ai", row.name)
		if err != nil {
			t.Fatalf("create agent %s: %v", row.name, err)
		}
	}
	t.Cleanup(func() {
		store.db.ExecContext(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, agentAID, agentBID)
	})

	testUser := &auth.User{ID: "convene-human-1", Name: "ConveneTester", Email: "convene@test.com", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(auth.ContextWithUser(r.Context(), testUser))
			next.ServeHTTP(w, r)
		})
	}
	h := NewHandlers(store, wrap, wrap)
	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(ctx, "convene-op-test"); old != nil {
		store.DeleteSpace(ctx, old.ID)
	}
	space, err := store.CreateSpace(ctx, "convene-op-test", "Convene Op Test", "", testUser.ID, "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	body := `{"op":"convene","title":"What should we build next?","body":"Please share your perspective.","agents":"` + agentAName + `,` + agentBName + `"}`
	req := httptest.NewRequest("POST", "/app/convene-op-test/op", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["op"] != "convene" {
		t.Errorf("op = %v, want convene", result["op"])
	}
	node := result["node"].(map[string]any)
	if node["kind"] != KindCouncil {
		t.Errorf("kind = %v, want %q", node["kind"], KindCouncil)
	}
	if node["body"] != "Please share your perspective." {
		t.Errorf("body = %v, want 'Please share your perspective.'", node["body"])
	}
	if node["title"] != "What should we build next?" {
		t.Errorf("title = %v, want 'What should we build next?'", node["title"])
	}
	tags, _ := node["tags"].([]any)
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag.(string)] = true
	}
	if !tagSet[agentAID] {
		t.Errorf("tags missing agent A ID %q; got %v", agentAID, tags)
	}
	if !tagSet[agentBID] {
		t.Errorf("tags missing agent B ID %q; got %v", agentBID, tags)
	}
}

// TestHandlerCouncilDetail verifies that GET /app/{slug}/council/{id}
// returns 200 with response rows when Mind response nodes exist.
func TestHandlerCouncilDetail(t *testing.T) {
	h, store, _ := testHandlers(t)
	ctx := t.Context()

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(ctx, "council-detail-test"); old != nil {
		store.DeleteSpace(ctx, old.ID)
	}
	space, err := store.CreateSpace(ctx, "council-detail-test", "Council Detail Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	council, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindCouncil,
		Title:    "Should we prioritize performance?",
		Body:     "Looking for agent perspectives.",
		Author:   "Tester",
		AuthorID: "test-user-1",
		Tags:     []string{"agent-resp-1"},
	})
	if err != nil {
		t.Fatalf("create council: %v", err)
	}

	// Simulate two Mind responses as child KindComment nodes.
	for i, name := range []string{"AgentX", "AgentY"} {
		_, err := store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			ParentID:   council.ID,
			Kind:       KindComment,
			Body:       "Response from " + name,
			Author:     name,
			AuthorID:   "agent-resp-" + string(rune('1'+i)),
			AuthorKind: "agent",
		})
		if err != nil {
			t.Fatalf("create response %s: %v", name, err)
		}
	}

	req := httptest.NewRequest("GET", "/app/council-detail-test/council/"+council.ID, nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	session := result["session"].(map[string]any)
	if session["id"] != council.ID {
		t.Errorf("session.id = %v, want %v", session["id"], council.ID)
	}
	if session["kind"] != KindCouncil {
		t.Errorf("session.kind = %v, want %q", session["kind"], KindCouncil)
	}
	responses, _ := result["responses"].([]any)
	if len(responses) != 2 {
		t.Errorf("got %d responses, want 2", len(responses))
	}
}

// TestHandlerCouncilDetail_NotFound verifies that a wrong kind or missing node returns 404.
func TestHandlerCouncilDetail_NotFound(t *testing.T) {
	h, store, _ := testHandlers(t)
	ctx := t.Context()

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(ctx, "council-notfound-test"); old != nil {
		store.DeleteSpace(ctx, old.ID)
	}
	space, err := store.CreateSpace(ctx, "council-notfound-test", "Council NotFound Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	t.Run("nonexistent_id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/app/council-notfound-test/council/no-such-id", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("wrong_kind_returns_404", func(t *testing.T) {
		task, err := store.CreateNode(ctx, CreateNodeParams{
			SpaceID: space.ID, Kind: KindTask, Title: "Not a council",
			Author: "Tester", AuthorID: "test-user-1",
		})
		if err != nil {
			t.Fatalf("create task: %v", err)
		}
		req := httptest.NewRequest("GET", "/app/council-notfound-test/council/"+task.ID, nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (wrong kind should 404)", w.Code, http.StatusNotFound)
		}
	})
}

// TestHandlerQuestionAutoAnswer verifies that creating a KindQuestion via the intend op
// triggers Mind.OnQuestionAsked and results in a respond op on the answer node.
func TestHandlerQuestionAutoAnswer(t *testing.T) {
	db, store := testDB(t)
	ctx := t.Context()

	agentID := "qa-auto-answer-agent-id"
	agentName := "QAAutoAnswerAgent"
	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
		agentID, "agent:"+agentName, agentName+"@test.transpara.ai", agentName)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	t.Cleanup(func() { db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID) })

	mind := NewMind(db, store, nil)
	mind.callProviderOverride = func(_ context.Context, _ string, _ []providerMessage) (string, error) {
		return "Auto-answer from mind.", nil
	}

	testUser := &auth.User{ID: "qa-auto-human-1", Name: "QATester", Email: "qa@test.com", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(auth.ContextWithUser(r.Context(), testUser))
			next.ServeHTTP(w, r)
		})
	}
	h := NewHandlers(store, wrap, wrap)
	h.SetMind(mind)
	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(ctx, "qa-auto-answer-test"); old != nil {
		store.DeleteSpace(ctx, old.ID)
	}
	space, err := store.CreateSpace(ctx, "qa-auto-answer-test", "QA Auto Answer Test", "", testUser.ID, "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	body := `{"op":"intend","title":"What is the event graph?","description":"Looking for a quick overview","kind":"question"}`
	req := httptest.NewRequest("POST", "/app/qa-auto-answer-test/op", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	node := result["node"].(map[string]any)
	questionID := node["id"].(string)

	// OnQuestionAsked runs in a goroutine; poll briefly for the respond op.
	deadline := time.Now().Add(2 * time.Second)
	var hasRespondOp bool
	for time.Now().Before(deadline) {
		ops, err := store.ListOps(ctx, space.ID, 50)
		if err != nil {
			t.Fatalf("list ops: %v", err)
		}
		for _, o := range ops {
			if o.Op == "respond" && o.NodeID != questionID {
				hasRespondOp = true
				break
			}
		}
		if hasRespondOp {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !hasRespondOp {
		t.Errorf("expected a respond op on the answer node after creating KindQuestion %s, got none", questionID)
	}
}

// TestHivePage issues GET /hive and asserts HTTP 200 and the body contains
// the "Phase timeline" section heading that the template always renders.
func TestHivePage(t *testing.T) {
	h, _, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/hive", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /hive: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Phase timeline") {
		t.Error("GET /hive: body does not contain 'Phase timeline'")
	}
}

// TestListHiveActivity calls ListHiveActivity with kind=post and the hive agent's user ID,
// asserts the result is non-nil and bounded to ≤10 items.
func TestListHiveActivity(t *testing.T) {
	_, store := testDB(t)
	ctx := t.Context()

	agentUserID := fmt.Sprintf("hive-list-activity-test-agent-%d", time.Now().UnixNano())
	slug := fmt.Sprintf("hive-list-activity-test-%d", time.Now().UnixNano())

	space, err := store.CreateSpace(ctx, slug, "Hive List Activity Test", "", "owner-list-activity", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	for i := range 3 {
		_, err := store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindPost,
			Title:      fmt.Sprintf("[hive:builder] iter %d: shipped", i),
			Body:       "Builder shipped. Cost: $0.42.",
			Author:     "hive-builder",
			AuthorID:   agentUserID,
			AuthorKind: "agent",
		})
		if err != nil {
			t.Fatalf("create post %d: %v", i, err)
		}
	}

	nodes, err := store.ListHiveActivity(ctx, agentUserID, 10)
	if err != nil {
		t.Fatalf("ListHiveActivity: %v", err)
	}
	if nodes == nil {
		t.Fatal("ListHiveActivity: result is nil, want non-nil slice")
	}
	if len(nodes) > 10 {
		t.Errorf("ListHiveActivity: %d items, want ≤10", len(nodes))
	}
}

// TestHandlerOpEditCauses verifies that op=edit with a causes field updates the
// node's causes via UpdateNodeCauses. This is the handler path used by
// backfillClaimCauses in cmd/post to retroactively satisfy Invariant 2 (CAUSALITY).
func TestHandlerOpEditCauses(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-edit-causes-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-edit-causes-test", "Edit Causes Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	// Create a claim node with no causes (simulates the 103 legacy claims).
	node, err := store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindClaim,
		Title:    "Legacy Claim",
		Author:   "Tester",
		AuthorID: "test-user-1",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	t.Run("edit_causes_single", func(t *testing.T) {
		body := `{"op":"edit","node_id":"` + node.ID + `","causes":"task-node-abc123"}`
		req := httptest.NewRequest("POST", "/app/handler-edit-causes-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		// Verify the causes were persisted.
		got, err := store.GetNode(t.Context(), node.ID)
		if err != nil {
			t.Fatalf("get node: %v", err)
		}
		if len(got.Causes) != 1 || got.Causes[0] != "task-node-abc123" {
			t.Errorf("causes = %v, want [task-node-abc123]", got.Causes)
		}
	})

	t.Run("edit_causes_multiple_comma_separated", func(t *testing.T) {
		body := `{"op":"edit","node_id":"` + node.ID + `","causes":"task-aaa,task-bbb,task-ccc"}`
		req := httptest.NewRequest("POST", "/app/handler-edit-causes-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		got, err := store.GetNode(t.Context(), node.ID)
		if err != nil {
			t.Fatalf("get node: %v", err)
		}
		if len(got.Causes) != 3 {
			t.Errorf("causes = %v, want 3 entries", got.Causes)
		}
	})

	t.Run("edit_requires_body_or_causes", func(t *testing.T) {
		// op=edit with neither body nor causes should fail.
		body := `{"op":"edit","node_id":"` + node.ID + `"}`
		req := httptest.NewRequest("POST", "/app/handler-edit-causes-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (400) when neither body nor causes provided; body: %s",
				w.Code, http.StatusBadRequest, w.Body.String())
		}
	})
}

// TestPopulateFormFromJSON is a pure unit test for the populateFormFromJSON helper.
// No database required.
func TestPopulateFormFromJSON(t *testing.T) {
	makeReq := func(contentType, body string) *http.Request {
		r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", contentType)
		return r
	}

	t.Run("array causes to CSV", func(t *testing.T) {
		r := makeReq("application/json", `{"op":"assert","causes":["id1","id2"]}`)
		populateFormFromJSON(r)
		if got := r.FormValue("op"); got != "assert" {
			t.Errorf("op = %q, want %q", got, "assert")
		}
		if got := r.FormValue("causes"); got != "id1,id2" {
			t.Errorf("causes = %q, want %q", got, "id1,id2")
		}
	})

	t.Run("string value pass-through", func(t *testing.T) {
		r := makeReq("application/json", `{"title":"hello world"}`)
		populateFormFromJSON(r)
		if got := r.FormValue("title"); got != "hello world" {
			t.Errorf("title = %q, want %q", got, "hello world")
		}
	})

	t.Run("non-JSON content-type is no-op", func(t *testing.T) {
		r := makeReq("application/x-www-form-urlencoded", `{"op":"assert"}`)
		populateFormFromJSON(r)
		if r.Form != nil && r.FormValue("op") != "" {
			t.Errorf("expected no form population for non-JSON content-type")
		}
	})

	t.Run("invalid JSON is no-op (no panic)", func(t *testing.T) {
		r := makeReq("application/json", `{not valid json`)
		populateFormFromJSON(r) // must not panic
		if r.Form != nil && len(r.Form) != 0 {
			t.Errorf("expected empty form for invalid JSON")
		}
	})

	t.Run("empty array produces empty string", func(t *testing.T) {
		r := makeReq("application/json", `{"causes":[]}`)
		populateFormFromJSON(r)
		if got := r.FormValue("causes"); got != "" {
			t.Errorf("causes = %q, want empty string for empty array", got)
		}
	})

	t.Run("null value is skipped", func(t *testing.T) {
		r := makeReq("application/json", `{"causes":null,"op":"assert"}`)
		populateFormFromJSON(r)
		if got := r.FormValue("causes"); got != "" {
			t.Errorf("causes = %q, want empty (null should be skipped)", got)
		}
		if got := r.FormValue("op"); got != "assert" {
			t.Errorf("op = %q, want %q", got, "assert")
		}
	})

	t.Run("numeric value via fmt.Sprintf", func(t *testing.T) {
		r := makeReq("application/json", `{"priority":5}`)
		populateFormFromJSON(r)
		if got := r.FormValue("priority"); got != "5" {
			t.Errorf("priority = %q, want %q", got, "5")
		}
	})

	t.Run("content-type with charset suffix", func(t *testing.T) {
		r := makeReq("application/json; charset=utf-8", `{"op":"intend"}`)
		populateFormFromJSON(r)
		if got := r.FormValue("op"); got != "intend" {
			t.Errorf("op = %q, want %q", got, "intend")
		}
	})

	t.Run("array with non-string items drops non-strings", func(t *testing.T) {
		r := makeReq("application/json", `{"causes":["id1",42,"id2"]}`)
		populateFormFromJSON(r)
		// 42 is a number, not a string — dropped silently; id1 and id2 are kept
		if got := r.FormValue("causes"); got != "id1,id2" {
			t.Errorf("causes = %q, want %q", got, "id1,id2")
		}
	})

	// Ensure the body is consumed only once — calling on a request with no body
	// should not panic.
	t.Run("empty body is no-op", func(t *testing.T) {
		r, _ := http.NewRequest("POST", "/", nil)
		r.Header.Set("Content-Type", "application/json")
		r.Body = io.NopCloser(strings.NewReader(""))
		populateFormFromJSON(r) // must not panic
	})

	t.Run("array with null item drops null keeps strings", func(t *testing.T) {
		r := makeReq("application/json", `{"causes":["id1",null,"id2"]}`)
		populateFormFromJSON(r)
		if got := r.FormValue("causes"); got != "id1,id2" {
			t.Errorf("causes = %q, want %q", got, "id1,id2")
		}
	})
}

func TestHandlerGovernanceDelegation(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	// Clean up stale space.
	if old, _ := store.GetSpaceBySlug(t.Context(), "gov-handler-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "gov-handler-test", "Gov Handler Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("propose_with_quorum_pct", func(t *testing.T) {
		body := `{"op":"propose","title":"Quorum Proposal","quorum_pct":"51","voting_body":"all"}`
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		if result["op"] != "propose" {
			t.Errorf("op = %v, want propose", result["op"])
		}
	})

	t.Run("delegate_op", func(t *testing.T) {
		body := `{"op":"delegate","delegate_id":"other-user-999"}`
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}
		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		if result["op"] != OpDelegate {
			t.Errorf("op = %v, want %v", result["op"], OpDelegate)
		}
		if !store.HasDelegated(t.Context(), space.ID, "test-user-1") {
			t.Error("HasDelegated after delegate op = false, want true")
		}
	})

	t.Run("vote_blocked_when_delegated", func(t *testing.T) {
		// test-user-1 has delegated (from previous subtest), so vote should fail.
		proposals, _ := store.ListProposals(t.Context(), space.ID, "open", 10)
		if len(proposals) == 0 {
			t.Skip("no open proposals in space")
		}
		body := fmt.Sprintf(`{"op":"vote","node_id":%q,"vote":"yes"}`, proposals[0].ID)
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("status = %d, want %d (delegated user cannot vote directly)", w.Code, http.StatusConflict)
		}
	})

	t.Run("undelegate_op", func(t *testing.T) {
		body := `{"op":"undelegate"}`
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}
		if store.HasDelegated(t.Context(), space.ID, "test-user-1") {
			t.Error("HasDelegated after undelegate op = true, want false")
		}
	})

	t.Run("delegate_missing_delegate_id", func(t *testing.T) {
		body := `{"op":"delegate"}`
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (missing delegate_id)", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("vote_after_undelegate", func(t *testing.T) {
		// test-user-1 has no delegation (removed by undelegate_op).
		// Create a fresh proposal, then vote — must succeed.
		propBody := `{"op":"propose","title":"Vote After Undelegate"}`
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(propBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("propose: status = %d; body: %s", w.Code, w.Body.String())
		}
		var propResult map[string]any
		json.NewDecoder(w.Body).Decode(&propResult)
		nodeMap, _ := propResult["node"].(map[string]any)
		nodeID, _ := nodeMap["id"].(string)
		if nodeID == "" {
			t.Skip("could not extract node ID from propose response")
		}

		voteBody := fmt.Sprintf(`{"op":"vote","node_id":%q,"vote":"yes"}`, nodeID)
		req2 := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(voteBody))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Accept", "application/json")
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)
		if w2.Code != http.StatusOK {
			t.Errorf("vote after undelegate: status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
		}
	})
}
