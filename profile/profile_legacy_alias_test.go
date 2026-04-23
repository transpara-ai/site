package profile

import (
	"bytes"
	"log/slog"
	"testing"
)

// TestLegacyAliasResolvesToCanonical — Lookup of a legacy slug must return
// the same *Profile as Lookup of the canonical slug, and the returned
// Profile's Slug field must be the canonical post-migration identifier
// (never the requested legacy string).
func TestLegacyAliasResolvesToCanonical(t *testing.T) {
	legacy := Lookup("lovyou-ai")
	canonical := Lookup("transpara-ai")

	if legacy == nil {
		t.Fatal("legacy slug 'lovyou-ai' should resolve via alias, got nil")
	}
	if canonical == nil {
		t.Fatal("canonical slug 'transpara-ai' should resolve, got nil")
	}
	if legacy != canonical {
		t.Errorf("legacy alias should return the same *Profile as canonical: legacy=%p canonical=%p", legacy, canonical)
	}
	if legacy.Slug != "transpara-ai" {
		t.Errorf("legacy resolution must surface canonical slug, got %q", legacy.Slug)
	}
}

// TestLegacyAliasEmitsWarning — every legacy-slug resolution must emit a
// structured deprecation warning so operators can grep logs for the
// request source and migrate it. Uses slog.SetDefault to capture output
// into an in-test buffer.
func TestLegacyAliasEmitsWarning(t *testing.T) {
	var buf bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(original) })

	_ = Lookup("lovyou-ai")

	if !bytes.Contains(buf.Bytes(), []byte("deprecated profile slug resolved via legacy alias")) {
		t.Errorf("expected deprecation warning in log output, got: %q", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte(`requested=lovyou-ai`)) {
		t.Errorf("warning should name the requested slug, got: %q", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte(`canonical=transpara-ai`)) {
		t.Errorf("warning should name the canonical target, got: %q", buf.String())
	}
}

// TestLookupCanonicalDoesNotWarn — a direct match on the canonical slug must
// NOT emit the deprecation warning. Guards against a regression where the
// alias log line fires on every lookup (noise) instead of only on legacy ones.
func TestLookupCanonicalDoesNotWarn(t *testing.T) {
	var buf bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(original) })

	_ = Lookup("transpara-ai")

	if bytes.Contains(buf.Bytes(), []byte("deprecated profile slug")) {
		t.Errorf("canonical lookup should not emit deprecation warning, got: %q", buf.String())
	}
}

// TestLookupUnknownSlugReturnsNil — unchanged resolver-chain semantics: an
// unknown slug (not in registry, not in legacyAliases) returns nil so the
// Chain can fall through to DefaultResolver. Regression guard for the
// "alias map absorbs all misses" failure mode.
func TestLookupUnknownSlugReturnsNil(t *testing.T) {
	if p := Lookup("nonexistent-slug-xyz"); p != nil {
		t.Errorf("unknown slug should return nil, got profile with Slug=%q", p.Slug)
	}
}

// TestLegacyAliasesPointToExistingProfiles — alias-map hygiene: every
// canonical target must actually exist in the registry. Prevents a typo in
// legacyAliases from silently routing a legacy slug to a nil Profile.
func TestLegacyAliasesPointToExistingProfiles(t *testing.T) {
	for legacy, canonical := range legacyAliases {
		if _, ok := registry[canonical]; !ok {
			t.Errorf("legacyAliases[%q] points to %q which is not in registry", legacy, canonical)
		}
	}
}
