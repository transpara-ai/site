package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://site.test/ops", nil)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /ops without DATABASE_URL status = %d, want 405; body: %s", w.Code, w.Body.String())
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
