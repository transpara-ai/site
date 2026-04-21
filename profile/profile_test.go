package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefault_returnsLovyouAI(t *testing.T) {
	p := Default()
	if p == nil {
		t.Fatal("Default() returned nil")
	}
	if p.Slug != DefaultSlug {
		t.Errorf("Default().Slug = %q, want %q", p.Slug, DefaultSlug)
	}
}

func TestLookup(t *testing.T) {
	cases := []struct {
		name     string
		slug     string
		wantNil  bool
		wantSlug string
	}{
		{name: "default slug", slug: DefaultSlug, wantSlug: DefaultSlug},
		{name: "second registered slug", slug: "transpara", wantSlug: "transpara"},
		{name: "unknown slug", slug: "nonsense", wantNil: true},
		{name: "empty slug", slug: "", wantNil: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := Lookup(tc.slug)
			if tc.wantNil {
				if p != nil {
					t.Errorf("Lookup(%q) = %+v, want nil", tc.slug, p)
				}
				return
			}
			if p == nil {
				t.Fatalf("Lookup(%q) returned nil", tc.slug)
			}
			if p.Slug != tc.wantSlug {
				t.Errorf("Lookup(%q).Slug = %q, want %q", tc.slug, p.Slug, tc.wantSlug)
			}
		})
	}
}

func TestAll_includesRegisteredProfiles(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range All() {
		seen[p.Slug] = true
	}
	if !seen[DefaultSlug] {
		t.Errorf("All() missing default slug %q", DefaultSlug)
	}
	if !seen["transpara"] {
		t.Error("All() missing the second registered profile")
	}
}

func TestRegistry_fieldsPopulated(t *testing.T) {
	for _, p := range All() {
		if p.BrandName == "" {
			t.Errorf("profile %q has empty BrandName", p.Slug)
		}
		if p.LogoPath == "" {
			t.Errorf("profile %q has empty LogoPath", p.Slug)
		}
		if p.AccentColor == "" {
			t.Errorf("profile %q has empty AccentColor", p.Slug)
		}
	}
}

func TestRegistry_profilesDiverge(t *testing.T) {
	// Phase 4 precondition: the two registered profiles must differ
	// on every brand-bearing field, otherwise the divergence tests
	// downstream (bounded-diff) would silently pass against a tie.
	l := Lookup(DefaultSlug)
	p := Lookup("transpara")
	if l.BrandName == p.BrandName {
		t.Errorf("registry BrandNames collide: %q", l.BrandName)
	}
	if l.LogoPath == p.LogoPath {
		t.Errorf("registry LogoPaths collide: %q", l.LogoPath)
	}
	if l.AccentColor == p.AccentColor {
		t.Errorf("registry AccentColors collide: %q", l.AccentColor)
	}
}

func TestGetSlug_nilProfile_returnsDefault(t *testing.T) {
	var p *Profile
	if got, want := p.GetSlug(), Default().Slug; got != want {
		t.Errorf("nil.GetSlug() = %q, want %q", got, want)
	}
}

func TestGetSlug_validProfile_returnsField(t *testing.T) {
	p := Lookup("transpara")
	if got, want := p.GetSlug(), "transpara"; got != want {
		t.Errorf("transpara.GetSlug() = %q, want %q", got, want)
	}
}

func TestGetBrandName_nilProfile_returnsDefault(t *testing.T) {
	var p *Profile
	if got, want := p.GetBrandName(), Default().BrandName; got != want {
		t.Errorf("nil.GetBrandName() = %q, want %q", got, want)
	}
}

func TestGetBrandName_validProfile_returnsField(t *testing.T) {
	p := Lookup("transpara")
	if got, want := p.GetBrandName(), "Transpara"; got != want {
		t.Errorf("transpara.GetBrandName() = %q, want %q", got, want)
	}
}

func TestGetLogoPath_nilProfile_returnsDefault(t *testing.T) {
	var p *Profile
	if got, want := p.GetLogoPath(), Default().LogoPath; got != want {
		t.Errorf("nil.GetLogoPath() = %q, want %q", got, want)
	}
}

func TestGetLogoPath_validProfile_returnsField(t *testing.T) {
	p := Lookup("transpara")
	if got, want := p.GetLogoPath(), "/static/logo-transpara.svg"; got != want {
		t.Errorf("transpara.GetLogoPath() = %q, want %q", got, want)
	}
}

func TestGetAccentColor_nilProfile_returnsDefault(t *testing.T) {
	var p *Profile
	if got, want := p.GetAccentColor(), Default().AccentColor; got != want {
		t.Errorf("nil.GetAccentColor() = %q, want %q", got, want)
	}
}

func TestGetAccentColor_validProfile_returnsField(t *testing.T) {
	p := Lookup("transpara")
	if got, want := p.GetAccentColor(), "#0ea5e9"; got != want {
		t.Errorf("transpara.GetAccentColor() = %q, want %q", got, want)
	}
}

func TestContext_roundTrip(t *testing.T) {
	p := Default()
	ctx := WithProfile(context.Background(), p)
	got := FromContext(ctx)
	if got != p {
		t.Errorf("FromContext() = %p, want %p", got, p)
	}
}

func TestFromContext_missingReturnsNil(t *testing.T) {
	if got := FromContext(context.Background()); got != nil {
		t.Errorf("FromContext(empty ctx) = %+v, want nil", got)
	}
}

func TestQueryParamResolver(t *testing.T) {
	cases := []struct {
		name     string
		url      string
		wantNil  bool
		wantSlug string
	}{
		{name: "no param", url: "/", wantNil: true},
		{name: "empty param", url: "/?profile=", wantNil: true},
		{name: "default slug", url: "/?profile=" + DefaultSlug, wantSlug: DefaultSlug},
		{name: "second slug", url: "/?profile=transpara", wantSlug: "transpara"},
		{name: "unknown slug", url: "/?profile=nonsense", wantNil: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			p := QueryParamResolver{}.Resolve(req)
			if tc.wantNil {
				if p != nil {
					t.Errorf("Resolve(%q) = %+v, want nil", tc.url, p)
				}
				return
			}
			if p == nil {
				t.Fatalf("Resolve(%q) returned nil", tc.url)
			}
			if p.Slug != tc.wantSlug {
				t.Errorf("Resolve(%q).Slug = %q, want %q", tc.url, p.Slug, tc.wantSlug)
			}
		})
	}
}

func TestDefaultResolver_alwaysDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?profile=nonsense", nil)
	p := DefaultResolver{}.Resolve(req)
	if p == nil {
		t.Fatal("DefaultResolver returned nil")
	}
	if p.Slug != DefaultSlug {
		t.Errorf("DefaultResolver.Slug = %q, want %q", p.Slug, DefaultSlug)
	}
}

func TestChain_firstMatchWins(t *testing.T) {
	chain := Chain{
		QueryParamResolver{},
		DefaultResolver{},
	}
	req := httptest.NewRequest(http.MethodGet, "/?profile=transpara", nil)
	p := chain.Resolve(req)
	if p == nil || p.Slug != "transpara" {
		t.Errorf("chain.Resolve = %+v, want slug 'transpara'", p)
	}
}

func TestChain_fallsBackToDefault(t *testing.T) {
	chain := Chain{
		QueryParamResolver{},
		DefaultResolver{},
	}
	req := httptest.NewRequest(http.MethodGet, "/?profile=nonsense", nil)
	p := chain.Resolve(req)
	if p == nil || p.Slug != DefaultSlug {
		t.Errorf("chain.Resolve unknown = %+v, want default slug", p)
	}
}

func TestChain_emptyFallsBackToDefault(t *testing.T) {
	var chain Chain
	req := httptest.NewRequest(http.MethodGet, "/?profile=transpara", nil)
	p := chain.Resolve(req)
	if p == nil || p.Slug != DefaultSlug {
		t.Errorf("empty chain.Resolve = %+v, want default slug", p)
	}
}

func TestMiddleware_attachesResolvedProfile(t *testing.T) {
	chain := Chain{QueryParamResolver{}, DefaultResolver{}}
	var captured *Profile
	handler := Middleware(chain)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = FromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?profile=transpara", nil)
	handler.ServeHTTP(rec, req)

	if captured == nil {
		t.Fatal("middleware did not attach a profile to context")
	}
	if captured.Slug != "transpara" {
		t.Errorf("attached profile slug = %q, want 'transpara'", captured.Slug)
	}
}

func TestMiddleware_defaultOnUnknownSlug(t *testing.T) {
	chain := Chain{QueryParamResolver{}, DefaultResolver{}}
	var captured *Profile
	handler := Middleware(chain)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = FromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?profile=nonsense", nil)
	handler.ServeHTTP(rec, req)

	if captured == nil || captured.Slug != DefaultSlug {
		t.Errorf("unknown-slug middleware attached = %+v, want default", captured)
	}
}

func TestMiddleware_disabledFlagShortCircuits(t *testing.T) {
	t.Setenv(disableEnvVar, "1")
	chain := Chain{
		// A resolver that would panic if called — proves the chain
		// is not invoked when the disable flag is set.
		resolverFunc(func(*http.Request) *Profile { t.Fatal("resolver should not run when disabled"); return nil }),
	}
	var captured *Profile
	handler := Middleware(chain)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = FromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?profile=transpara", nil)
	handler.ServeHTTP(rec, req)

	if captured == nil || captured.Slug != DefaultSlug {
		t.Errorf("disabled middleware attached = %+v, want default", captured)
	}
}

func TestMiddleware_nilResolverFallsBackToDefault(t *testing.T) {
	var captured *Profile
	handler := Middleware(nil)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = FromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?profile=transpara", nil)
	handler.ServeHTTP(rec, req)

	if captured == nil || captured.Slug != DefaultSlug {
		t.Errorf("nil-resolver middleware attached = %+v, want default", captured)
	}
}

// resolverFunc is a test-only helper that adapts a plain func to the
// Resolver interface.
type resolverFunc func(r *http.Request) *Profile

func (f resolverFunc) Resolve(r *http.Request) *Profile { return f(r) }
