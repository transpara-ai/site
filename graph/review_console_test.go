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
		"docs v4.0 Event 13 AuthorityDecision",
		"127da4ef57dee34231cc50d87a249349fc0f768c",
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
