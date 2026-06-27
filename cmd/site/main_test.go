package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/transpara-ai/site/auth"
	"github.com/transpara-ai/site/profile"
	"github.com/transpara-ai/site/views"
)

func TestValidateProductionAuthConfigFailsClosedWithoutOAuth(t *testing.T) {
	env := map[string]string{
		"SITE_ENV": "production",
	}

	err := validateProductionAuthConfig(mapGetter(env))
	if err == nil {
		t.Fatal("expected production auth config error")
	}
	if !strings.Contains(err.Error(), "GOOGLE_CLIENT_ID") || !strings.Contains(err.Error(), "GOOGLE_CLIENT_SECRET") {
		t.Fatalf("error = %q, want missing Google auth config", err.Error())
	}
}

func TestNoEnvDefaultsToNonProduction(t *testing.T) {
	// With no environment declared, default to non-production (on-prem opt-in).
	env := map[string]string{}

	if err := validateProductionAuthConfig(mapGetter(env)); err != nil {
		t.Fatalf("empty env must default to non-production: %v", err)
	}
}

func TestValidateProductionAuthConfigAllowsProductionWithOAuth(t *testing.T) {
	env := map[string]string{
		"APP_ENV":              "production",
		"GOOGLE_CLIENT_ID":     "client-id",
		"GOOGLE_CLIENT_SECRET": "client-secret",
	}

	if err := validateProductionAuthConfig(mapGetter(env)); err != nil {
		t.Fatalf("validateProductionAuthConfig: %v", err)
	}
}

func TestValidateProductionAuthConfigAllowsLocalAnonymousMode(t *testing.T) {
	env := map[string]string{
		"APP_ENV": "development",
	}

	if err := validateProductionAuthConfig(mapGetter(env)); err != nil {
		t.Fatalf("validateProductionAuthConfig: %v", err)
	}
}

func TestNoDatabaseRoutesExposeReadOnlyCivilization(t *testing.T) {
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("home"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/civilization", nil)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/civilization without DATABASE_URL status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		`data-civilization-assembly="read-only"`,
		`id="issue-intake"`,
		`id="issue-scan-kanban"`,
		"Civilization Assembly",
		"projection unavailable",
		"this page has no execution authority",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/civilization without DATABASE_URL body missing %q", want)
		}
	}
	assertNoMutationControls(t, "/ops/civilization", body)
}

func TestNoDatabaseRoutesExposeReadOnlyOps(t *testing.T) {
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("home"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops", nil)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops without DATABASE_URL status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"Operations",
		"site shell",
		"Operator surfaces",
		"Civilization",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops without DATABASE_URL body missing %q", want)
		}
	}
	assertNoMutationControls(t, "/ops", body)
}

func TestNoDatabaseHomeExposesMFOFMonitoringSurfaces(t *testing.T) {
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		views.Home(views.HomeStats{}, profile.FromContext(r.Context())).Render(r.Context(), w)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://site.test/", nil)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET / without DATABASE_URL status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		`aria-label="Civilization monitoring surfaces"`,
		`href="/ops/civilization"`,
		`href="/ops/observatory"`,
		`href="/ops/telemetry"`,
		`href="/ops/civilization#issue-intake"`,
		`href="/ops/github-canonical"`,
		`href="/ops/civilization#issue-scan-kanban"`,
		`href="/ops/github-canonical#test-001-posture"`,
		`href="/ops/review-console"`,
		`href="/ops/evidence"`,
		"YELLOW/open",
		"projection only",
		"scanner evidence",
		"read-only",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET / without DATABASE_URL body missing %q", want)
		}
	}
	assertNoMutationControls(t, "/", body)
}

func TestNoDatabaseRoutesExposeReadOnlyGitHubCanonicalTest001Posture(t *testing.T) {
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("home"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/github-canonical", nil)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/github-canonical without DATABASE_URL status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		`id="test-001-posture"`,
		`data-github-canonical-test-001-posture="read-only"`,
		"Test 001 carried-evidence posture",
		"YELLOW",
		"transpara-ai/operation#26",
		"STILL_UNAVAILABLE_YELLOW_KEEPING",
		"does not authorize Test 001 GREEN",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/github-canonical without DATABASE_URL body missing %q", want)
		}
	}
	assertNoMutationControls(t, "/ops/github-canonical", body)
}

func TestNoDatabaseRoutesExposeReadOnlyMonitoringSurfaces(t *testing.T) {
	feeder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "feeder unavailable in test", http.StatusServiceUnavailable)
	}))
	defer feeder.Close()
	t.Setenv("WORK_API_BASE_URL", feeder.URL)
	t.Setenv("HIVE_OPS_API_BASE_URL", feeder.URL)

	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		path string
		want []string
	}{
		{
			path: "/ops/telemetry",
			want: []string{
				"Telemetry",
				"Telemetry summary",
				"work telemetry returned 503 Service Unavailable",
			},
		},
		{
			path: "/ops/observatory",
			want: []string{
				"Observatory",
				"Civilization vitals",
				"Vitals unavailable",
				"feeder returned 503 Service Unavailable",
				"read-only projection",
			},
		},
		{
			path: "/ops/review-console",
			want: []string{
				"External Committee Review Console",
				"Gate W was closed by the docs#186 evidence-decision accepting Site PR #90 evidence only for Event 13 Level 0 read-only review-console display evidence.",
				"bounded Gate W evidence decision merged by docs#186",
				"read-only",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "http://site.test"+tt.path, nil)
			mux.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("GET %s without DATABASE_URL status = %d, want 200; body: %s", tt.path, w.Code, w.Body.String())
			}
			body := w.Body.String()
			for _, want := range tt.want {
				if !strings.Contains(body, want) {
					t.Fatalf("GET %s without DATABASE_URL body missing %q", tt.path, want)
				}
			}
			assertNoMutationControls(t, tt.path, body)
		})
	}
}

func TestNoDatabaseRoutesExposeReadOnlyObservatoryEvents(t *testing.T) {
	feeder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "feeder unavailable in test", http.StatusServiceUnavailable)
	}))
	defer feeder.Close()
	t.Setenv("WORK_API_BASE_URL", feeder.URL)

	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("home fallback"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/observatory/events", nil)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("GET /ops/observatory/events without DATABASE_URL status = %d, want 502; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "home fallback") {
		t.Fatal("GET /ops/observatory/events without DATABASE_URL fell through to home fallback")
	}
	if !strings.Contains(body, "event pulse feeder returned 503 Service Unavailable") {
		t.Fatalf("GET /ops/observatory/events without DATABASE_URL body missing feeder failure; body: %s", body)
	}
}

func TestNoDatabaseCivilizationRejectsMutationMethod(t *testing.T) {
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops/civilization", nil)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /ops/civilization without DATABASE_URL status = %d, want 405; body: %s", w.Code, w.Body.String())
	}
}

func TestNoDatabaseOpsRejectsMutationMethod(t *testing.T) {
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, path := range []string{"/ops", "/ops/telemetry", "/ops/observatory", "/ops/observatory/events", "/ops/civilization", "/ops/github-canonical", "/ops/review-console"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "http://site.test"+path, nil)
			mux.ServeHTTP(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Fatalf("POST %s without DATABASE_URL status = %d, want 405; body: %s", path, w.Code, w.Body.String())
			}
		})
	}
}

func TestNoDatabaseReadOnlyOpsToleratesUserContextWithoutStore(t *testing.T) {
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, path := range []string{"/ops", "/ops/telemetry", "/ops/observatory", "/ops/civilization", "/ops/github-canonical"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "http://site.test"+path, nil)
			req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{
				ID:      "user_test",
				Name:    "Test Operator",
				Picture: "https://example.invalid/avatar.png",
			}))
			mux.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("GET %s with user context and no store status = %d, want 200; body: %s", path, w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "Test Operator") {
				t.Fatalf("GET %s with user context body missing operator name", path)
			}
		})
	}
}

func TestNoDatabaseReviewConsoleToleratesUserContextWithoutStore(t *testing.T) {
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/review-console", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{
		ID:      "user_test",
		Name:    "Test Operator",
		Picture: "https://example.invalid/avatar.png",
	}))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/review-console with user context and no store status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"External Committee Review Console",
		"data-review-console=\"read-only\"",
		"Gate W was closed by the docs#186 evidence-decision accepting Site PR #90 evidence only for Event 13 Level 0 read-only review-console display evidence.",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/review-console with user context body missing %q", want)
		}
	}
}

func assertNoMutationControls(t *testing.T, path string, body string) {
	t.Helper()
	lower := strings.ToLower(body)
	for _, forbidden := range []string{
		`method="post"`,
		`hx-post=`,
		`hx-put=`,
		`hx-patch=`,
		`hx-delete=`,
		`formaction=`,
		`data-action="approve"`,
		`data-action="merge"`,
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("GET %s without DATABASE_URL exposed mutation marker %q", path, forbidden)
		}
	}
}

func mapGetter(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
