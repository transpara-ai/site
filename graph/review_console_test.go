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
		"Gate S approval artifact residual",
		"accepted_with_residual",
		"Gate W closeout evidence",
		`data-evidence-state="missing"`,
		"Test 001 remains YELLOW",
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
	var missing, residual bool
	for _, item := range data.Items {
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
	start := strings.Index(body, `data-review-console="read-only"`)
	if start < 0 {
		t.Fatal("review console marker missing")
	}
	surface := body[start:]
	forbidden := []string{
		"<form",
		"<button",
		`method="post"`,
		`action="`,
		`formaction="`,
		"api.github.com",
		"/repos/",
		"GitHub token",
		"secret",
		"Deploy",
		"RuntimeBroker execution control",
		"EventGraph write control",
	}
	for _, f := range forbidden {
		if strings.Contains(surface, f) {
			t.Fatalf("review console contains mutation/control marker %q", f)
		}
	}
}
