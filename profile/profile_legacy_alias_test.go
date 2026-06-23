package profile

import "testing"

func TestLegacySlugDoesNotResolve(t *testing.T) {
	if p := Lookup("lovyou-ai"); p != nil {
		t.Fatalf("legacy slug should not resolve to renderable profile UI, got %q", p.Slug)
	}
}

func TestLegacySlugIsDocumentedAsQuarantined(t *testing.T) {
	if got := quarantinedLegacySlugs["lovyou-ai"]; got != "transpara-ai" {
		t.Fatalf("quarantined legacy slug target = %q, want %q", got, "transpara-ai")
	}
}

func TestLookupUnknownSlugReturnsNil(t *testing.T) {
	if p := Lookup("nonexistent-slug-xyz"); p != nil {
		t.Errorf("unknown slug should return nil, got profile with Slug=%q", p.Slug)
	}
}
