package graph

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleOpsReviewConsoleRendersDisplayOnlyEvidence(t *testing.T) {
	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/review-console?profile=transpara", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/review-console: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"External Committee Review Console",
		`data-review-console="read-only"`,
		`data-review-console-boundary="display-only"`,
		`data-exact-head-approval-evidence="read-only"`,
		`data-approval-state="approved"`,
		`data-approval-state="stale"`,
		`data-approval-state="missing"`,
		`data-approval-state="accepted_residual"`,
		`data-clean-approval="true"`,
		`data-clean-approval="false"`,
		"docs v4.0 Event 13 AuthorityDecision",
		"127da4ef57dee34231cc50d87a249349fc0f768c",
		"Stale exact-head approval is not approval for the current head.",
		"Missing evidence fails closed and is not approval.",
		"Accepted residual evidence must be carried with later citations and is not clean approval.",
		"Clean approval is display evidence only; Site does not merge, deploy, close gates, or approve protected actions.",
		"Exact-head approval evidence matches the required head.",
		"point-in-time docs#185 approval",
		"Event 13 AuthorityDecision",
		"DF-V4.0-EPIC-013-AUTHORITY-DECISION",
		"Gate S approval artifact residual",
		"accepted_with_residual",
		"Gate W closeout evidence",
		`data-evidence-state="missing"`,
		"Test 001 remains YELLOW",
		"not applicable",
		"display only",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/review-console: body does not contain %q", want)
		}
	}
	assertOpsReviewConsoleNoMutationControls(t, body)
}

func TestOpsReviewConsoleExactHeadApprovalEvidenceStates(t *testing.T) {
	data := buildOpsReviewConsoleData()
	wantStates := map[string]bool{
		"approved":          false,
		"stale":             false,
		"missing":           false,
		"accepted_residual": false,
	}
	cleanCount := 0
	for _, item := range data.ExactHeadEvidence {
		if item.ID == "" ||
			item.Title == "" ||
			item.TargetRepo == "" ||
			item.TargetRef == "" ||
			item.ApprovalState == "" ||
			item.ResidualState == "" ||
			item.Summary == "" ||
			item.Limitation == "" {
			t.Fatalf("exact-head approval item has incomplete fields: %#v", item)
		}
		if !strings.HasPrefix(item.TargetRepo, "transpara-ai/") {
			t.Fatalf("exact-head approval item %q TargetRepo = %q, want Transpara-AI repo", item.ID, item.TargetRepo)
		}
		if item.ApprovalSourceURL != "" && !strings.HasPrefix(item.ApprovalSourceURL, "https://github.com/transpara-ai/") {
			t.Fatalf("exact-head approval item %q ApprovalSourceURL = %q, want Transpara-AI GitHub source", item.ID, item.ApprovalSourceURL)
		}
		if !item.DisplayOnly {
			t.Fatalf("exact-head approval item %q DisplayOnly = false", item.ID)
		}
		wantStates[item.ApprovalState] = true
		if item.ApprovalState == "approved" {
			if !item.CleanApproval {
				t.Fatalf("approved exact-head item %q CleanApproval = false", item.ID)
			}
			cleanCount++
			continue
		}
		if item.CleanApproval {
			t.Fatalf("non-approved exact-head item %q state %q has CleanApproval=true", item.ID, item.ApprovalState)
		}
	}
	for state, seen := range wantStates {
		if !seen {
			t.Fatalf("exact-head approval fixtures do not cover state %q", state)
		}
	}
	if cleanCount != 1 {
		t.Fatalf("clean approved exact-head fixture count = %d, want 1", cleanCount)
	}
}

func TestOpsExactHeadApprovalStateFailsClosed(t *testing.T) {
	for name, tc := range map[string]struct {
		required string
		approved string
		source   string
		residual string
		want     string
	}{
		"missing approved head": {
			required: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			source:   "https://github.com/transpara-ai/site/issues/118",
			want:     "missing",
		},
		"missing source": {
			required: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			approved: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			want:     "missing",
		},
		"stale head": {
			required: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			approved: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			source:   "https://github.com/transpara-ai/site/issues/118",
			want:     "stale",
		},
		"accepted residual": {
			required: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			approved: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			source:   "https://github.com/transpara-ai/site/issues/118",
			residual: "accepted_with_residual",
			want:     "accepted_residual",
		},
		"stale residual remains stale": {
			required: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			approved: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			source:   "https://github.com/transpara-ai/site/issues/118",
			residual: "accepted_with_residual",
			want:     "stale",
		},
		"clean approved": {
			required: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			approved: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			source:   "https://github.com/transpara-ai/site/issues/118",
			want:     "approved",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := opsExactHeadApprovalState(tc.required, tc.approved, tc.source, tc.residual); got != tc.want {
				t.Fatalf("opsExactHeadApprovalState() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestOpsReviewConsoleFailsClosedForMissingAndResidualEvidence(t *testing.T) {
	data := buildOpsReviewConsoleData()
	requiredKinds := map[string]bool{
		"exact_head_approval":  false,
		"authority_decision":   false,
		"residual_disposition": false,
		"gate_closeout":        false,
		"issue_disposition":    false,
	}
	var missing, residual bool
	for _, item := range data.Items {
		if item.ID == "" ||
			item.Title == "" ||
			item.DecisionKind == "" ||
			item.SourceURL == "" ||
			item.SourceType == "" ||
			item.SourceRepo == "" ||
			item.RequiredActor == "" ||
			item.RequiredAction == "" ||
			item.EvidenceState == "" ||
			item.ResidualState == "" ||
			item.GateScope == "" ||
			item.Limitation == "" {
			t.Fatalf("review item has incomplete contract fields: %#v", item)
		}
		if !strings.HasPrefix(item.SourceURL, "https://github.com/transpara-ai/") {
			t.Fatalf("review item %q source URL = %q, want Transpara-AI GitHub source", item.ID, item.SourceURL)
		}
		if _, ok := requiredKinds[item.DecisionKind]; ok {
			requiredKinds[item.DecisionKind] = true
		}
		if item.EvidenceState == "missing" && item.DisplayOnly && item.ResidualState == "open" {
			missing = true
		}
		if item.EvidenceState == "carried_residual" && item.DisplayOnly && item.ResidualState == "accepted_with_residual" {
			residual = true
		}
		if !item.DisplayOnly {
			t.Fatalf("review item %q DisplayOnly = false", item.ID)
		}
	}
	if !missing {
		t.Fatal("review console data does not include a display-only missing evidence item")
	}
	if !residual {
		t.Fatal("review console data does not include a carried residual item")
	}
	if gateW := findOpsReviewItem(t, data, "gate-w-closeout"); gateW.EvidenceState != "missing" || gateW.ResidualState != "open" {
		t.Fatalf("gate-w-closeout state = %s/%s, want missing/open", gateW.EvidenceState, gateW.ResidualState)
	}
	if test001 := findOpsReviewItem(t, data, "test-001-yellow"); test001.EvidenceState != "pending" || test001.ResidualState != "open" {
		t.Fatalf("test-001-yellow state = %s/%s, want pending/open", test001.EvidenceState, test001.ResidualState)
	}
	if test001 := findOpsReviewItem(t, data, "test-001-yellow"); test001.SourceURL != "https://github.com/transpara-ai/operation/issues/26" || test001.SourceRepo != "transpara-ai/operation" {
		t.Fatalf("test-001-yellow source = %s / %s, want operation issue source", test001.SourceURL, test001.SourceRepo)
	}
	for kind, seen := range requiredKinds {
		if !seen {
			t.Fatalf("review console data does not include required decision kind %q", kind)
		}
	}
}

func findOpsReviewItem(t *testing.T, data OpsReviewConsoleData, id string) OpsReviewItem {
	t.Helper()
	for _, item := range data.Items {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("review item %q not found", id)
	return OpsReviewItem{}
}

func TestOperatorReviewConsoleRouteRequiresWriteAuth(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodGet, "/ops/review-console", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusUnauthorized, w.Body.String())
	}
}

func assertOpsReviewConsoleNoMutationControls(t *testing.T, body string) {
	t.Helper()
	forbidden := []string{
		"<form",
		"<button",
		`method="post"`,
		`action="`,
		`formaction="`,
		"data-action=",
		"hx-post",
		"fetch(",
		"XMLHttpRequest",
		"api.github.com",
		"/repos/",
	}
	for _, f := range forbidden {
		if strings.Contains(body, f) {
			t.Fatalf("review console contains mutation/control marker %q", f)
		}
	}
}
