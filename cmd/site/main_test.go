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
		`href="/ops/telemetry"`,
		`href="/ops/observatory"`,
		`href="/ops/civilization"`,
		`href="/ops/github-canonical"`,
		`href="/ops/public-proof"`,
		`href="/ops/review-console"`,
		`href="/ops/hive/intake"`,
		`href="/ops/evidence"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops without DATABASE_URL body missing %q", want)
		}
	}
	for _, forbidden := range []string{
		`href="/ops/work"`,
		`href="/ops/hive"`,
		`href="/ops/decision"`,
		`href="/ops/approvals"`,
		`href="/ops/refinery"`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("GET /ops without DATABASE_URL body contains unavailable route link %q", forbidden)
		}
	}
	assertNoMutationControls(t, "/ops", body)
}

func TestNoDatabaseRoutesExposeReadOnlyOpsControlAlias(t *testing.T) {
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("home"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/control", nil)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/control without DATABASE_URL status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"Operations",
		"site shell",
		"Operator surfaces",
		`href="/ops/civilization"`,
		`href="/ops/hive/intake"`,
		`href="/ops/evidence"`,
		`href="/ops/public-proof"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/control without DATABASE_URL body missing %q", want)
		}
	}
	if strings.Contains(body, `href="/ops/control"`) {
		t.Fatal("GET /ops/control without DATABASE_URL exposed /ops/control as canonical navigation")
	}
	assertNoMutationControls(t, "/ops/control", body)
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
		`href="/ops/hive/intake"`,
		`href="/ops/github-canonical"`,
		`href="/ops/civilization#issue-scan-kanban"`,
		`href="/ops/github-canonical#test-001-posture"`,
		`href="/ops/review-console"`,
		`href="/ops/evidence"`,
		`href="/ops/public-proof"`,
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
	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", "")

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
		{
			path: "/ops/evidence",
			want: []string{
				"Evidence-surface guardrails",
				"Read-only FactoryOrder evidence projection",
				"Operator shell",
			},
		},
		{
			path: "/ops/evidence?view=forensic",
			want: []string{
				"Evidence projection",
				"projection URL is not configured",
				"FactoryOrder timeline",
				"Read-only",
			},
		},
		{
			path: "/ops/public-proof",
			want: []string{
				"Public Proof",
				`data-public-proof="display-only"`,
				"Public-reader and public-correction proof",
				"unavailable",
				"stale",
				"fixture/local",
				"projection-only",
				"deployed-reference",
				"live-reader-proof",
				"public-correction-proof",
				"no fake green lights",
			},
		},
		{
			path: "/ops/hive/intake",
			want: []string{
				"Hive intake",
				"Ingestion unavailable in this shell.",
				"graph store unavailable",
				"Run request queueing is unavailable",
				"read-only degraded shell",
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
			if tt.path == "/ops/hive/intake" {
				for _, forbidden := range []string{
					`action="/ops/hive/intake/sources"`,
					`action="/ops/hive/intake/launch"`,
					`href="/ops/hive/runs"`,
					`href="/ops/hive/agents"`,
					`href="/ops/hive/resources"`,
					"Add source",
					"Queue Hive run request",
				} {
					if strings.Contains(body, forbidden) {
						t.Fatalf("GET %s without DATABASE_URL exposed disabled ingestion control/link %q", tt.path, forbidden)
					}
				}
			}
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

	for _, path := range []string{"/ops", "/ops/control", "/ops/telemetry", "/ops/observatory", "/ops/observatory/events", "/ops/civilization", "/ops/github-canonical", "/ops/public-proof", "/ops/review-console", "/ops/hive/intake", "/ops/evidence"} {
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

func TestNoDatabaseHiveIntakeMutationSubroutesUnavailable(t *testing.T) {
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, path := range []string{"/ops/hive/intake/sources", "/ops/hive/intake/launch"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "http://site.test"+path, strings.NewReader("source_kind=text&content=x"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			mux.ServeHTTP(w, req)
			if w.Code >= 200 && w.Code < 400 {
				t.Fatalf("POST %s without DATABASE_URL status = %d, want non-success/non-redirect; headers: %v body: %s", path, w.Code, w.Header(), w.Body.String())
			}
			if location := w.Header().Get("Location"); location != "" {
				t.Fatalf("POST %s without DATABASE_URL returned redirect Location %q", path, location)
			}
		})
	}
}

func TestNoDatabaseReadOnlyOpsToleratesUserContextWithoutStore(t *testing.T) {
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, path := range []string{"/ops", "/ops/control", "/ops/telemetry", "/ops/observatory", "/ops/civilization", "/ops/github-canonical", "/ops/public-proof", "/ops/hive/intake"} {
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

func TestNoDatabaseEvidenceToleratesUserContextWithoutStore(t *testing.T) {
	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", "")
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/evidence", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{
		ID:      "user_test",
		Name:    "Test Operator",
		Picture: "https://example.invalid/avatar.png",
	}))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/evidence with user context and no store status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"Evidence-surface guardrails",
		"Read-only FactoryOrder evidence projection",
		"Operator shell",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/evidence with user context body missing %q", want)
		}
	}
}

func TestNoDatabaseEvidenceForensicToleratesUserContextWithoutStore(t *testing.T) {
	t.Setenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL", "")
	mux := http.NewServeMux()
	registerNoDatabaseRoutes(mux, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/evidence?view=forensic", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), &auth.User{
		ID:      "user_test",
		Name:    "Test Operator",
		Picture: "https://example.invalid/avatar.png",
	}))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/evidence?view=forensic with user context and no store status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"Evidence projection",
		"projection URL is not configured",
		"Projection unavailable",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/evidence?view=forensic with user context body missing %q", want)
		}
	}
	evidenceSurface := body
	if start := strings.Index(body, "Evidence projection"); start >= 0 {
		evidenceSurface = body[start:]
	}
	assertNoMutationControls(t, "/ops/evidence?view=forensic evidence surface", evidenceSurface)
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
