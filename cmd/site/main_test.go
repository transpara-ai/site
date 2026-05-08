package main

import (
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

func TestValidateProductionAuthConfigTreatsFlyAsProduction(t *testing.T) {
	env := map[string]string{
		"FLY_APP_NAME": "transpara-site",
	}

	if err := validateProductionAuthConfig(mapGetter(env)); err == nil {
		t.Fatal("expected Fly runtime without OAuth config to fail closed")
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

func mapGetter(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
