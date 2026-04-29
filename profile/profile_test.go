package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGetHeaderNav_nilProfile_returnsDefault(t *testing.T) {
	var p *Profile
	got := p.GetHeaderNav()
	want := Default().HeaderNav
	if len(got) == 0 {
		t.Fatal("nil.GetHeaderNav() returned empty slice; expected default fallback")
	}
	if len(got) != len(want) {
		t.Fatalf("nil.GetHeaderNav() len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("nil.GetHeaderNav()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestGetHeaderNav_validProfile_returnsField(t *testing.T) {
	p := Lookup("transpara")
	got := p.GetHeaderNav()
	if len(got) == 0 {
		t.Fatal("transpara.GetHeaderNav() returned empty slice")
	}
	// transpara is intentionally narrower than default — if this
	// test ever flips to equal-length, something merged the two
	// nav sets by accident.
	if len(got) >= len(Default().HeaderNav) {
		t.Errorf("transpara header nav len = %d, expected fewer than default's %d", len(got), len(Default().HeaderNav))
	}
	if got[0].Path != "/discover" {
		t.Errorf("transpara header nav[0].Path = %q, want %q", got[0].Path, "/discover")
	}
}

func TestGetFooterNav_nilProfile_returnsDefault(t *testing.T) {
	var p *Profile
	got := p.GetFooterNav()
	want := Default().FooterNav
	if len(got) == 0 {
		t.Fatal("nil.GetFooterNav() returned empty slice; expected default fallback")
	}
	if len(got) != len(want) {
		t.Fatalf("nil.GetFooterNav() len = %d, want %d", len(got), len(want))
	}
}

func TestGetFooterNav_validProfile_returnsField(t *testing.T) {
	p := Lookup("transpara")
	got := p.GetFooterNav()
	if len(got) == 0 {
		t.Fatal("transpara.GetFooterNav() returned empty slice")
	}
	if len(got) >= len(Default().FooterNav) {
		t.Errorf("transpara footer nav len = %d, expected fewer than default's %d", len(got), len(Default().FooterNav))
	}
}

func TestHeaderNav_emptyFieldFallsBackToDefault(t *testing.T) {
	// A registered profile with an empty HeaderNav slice should
	// render the default profile's nav rather than an empty nav
	// bar. Protects against a configuration mistake producing a
	// visibly broken page.
	p := &Profile{Slug: "test", HeaderNav: nil}
	got := p.GetHeaderNav()
	if len(got) != len(Default().HeaderNav) {
		t.Errorf("empty-HeaderNav profile returned len %d, want default len %d", len(got), len(Default().HeaderNav))
	}
}

func TestHeaderNav_profilesDiverge(t *testing.T) {
	// The whole point of Phase 5 is that different profiles have
	// different nav. If this fails, the registry somehow tied.
	l := Lookup(DefaultSlug).HeaderNav
	p := Lookup("transpara").HeaderNav
	if len(l) == len(p) {
		t.Errorf("registered HeaderNav lengths collide: %d == %d", len(l), len(p))
	}
}

func TestGetCopy_nilProfile_returnsFallback(t *testing.T) {
	var p *Profile
	if got, want := p.GetCopy("any.key", "fallback text"), "fallback text"; got != want {
		t.Errorf("nil.GetCopy() = %q, want %q", got, want)
	}
}

func TestGetCopy_missingKey_returnsFallback(t *testing.T) {
	p := Lookup("transpara")
	if got, want := p.GetCopy("definitely.not.a.real.key", "the inline default"), "the inline default"; got != want {
		t.Errorf("transpara.GetCopy(missing) = %q, want %q", got, want)
	}
}

func TestGetCopy_existingKey_returnsValue(t *testing.T) {
	p := Lookup("transpara")
	got := p.GetCopy("tagline", "FALLBACK_NOT_USED")
	if got == "" {
		t.Fatal("transpara.GetCopy('tagline') returned empty")
	}
	if got == "FALLBACK_NOT_USED" {
		t.Fatal("transpara.GetCopy('tagline') returned the fallback; expected the override")
	}
	if strings.Contains(got, "Humans and agents") {
		t.Errorf("transpara.GetCopy('tagline') returned default-profile copy %q; expected Transpara override", got)
	}
}

func TestGetCopy_defaultProfile_relyOnFallback(t *testing.T) {
	// The default profile intentionally has no Copy map — every key
	// returns the fallback, which matches today's hardcoded text.
	// This guards against accidentally adding a default-profile
	// override that would silently change copy.
	p := Lookup(DefaultSlug)
	if p.Copy != nil {
		t.Errorf("default profile has %d Copy overrides; expected nil so fallbacks render unchanged", len(p.Copy))
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

func TestCookieResolver(t *testing.T) {
	cases := []struct {
		name     string
		cookie   *http.Cookie
		wantNil  bool
		wantSlug string
	}{
		{name: "missing cookie", wantNil: true},
		{name: "empty cookie", cookie: &http.Cookie{Name: CookieName, Value: ""}, wantNil: true},
		{name: "known slug", cookie: &http.Cookie{Name: CookieName, Value: "transpara"}, wantSlug: "transpara"},
		{name: "unknown slug", cookie: &http.Cookie{Name: CookieName, Value: "nonsense"}, wantNil: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			p := CookieResolver{}.Resolve(req)
			if tc.wantNil {
				if p != nil {
					t.Errorf("CookieResolver.Resolve = %+v, want nil", p)
				}
				return
			}
			if p == nil || p.Slug != tc.wantSlug {
				t.Errorf("CookieResolver.Resolve = %+v, want slug %q", p, tc.wantSlug)
			}
		})
	}
}

func TestChain_firstMatchWins(t *testing.T) {
	chain := Chain{
		QueryParamResolver{},
		CookieResolver{},
		DefaultResolver{},
	}
	req := httptest.NewRequest(http.MethodGet, "/?profile=transpara", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: DefaultSlug})
	p := chain.Resolve(req)
	if p == nil || p.Slug != "transpara" {
		t.Errorf("chain.Resolve = %+v, want slug 'transpara'", p)
	}
}

func TestChain_usesCookieWhenQueryMissing(t *testing.T) {
	chain := Chain{
		QueryParamResolver{},
		CookieResolver{},
		DefaultResolver{},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "transpara"})
	p := chain.Resolve(req)
	if p == nil || p.Slug != "transpara" {
		t.Errorf("chain.Resolve with cookie = %+v, want slug 'transpara'", p)
	}
}

func TestChain_fallsBackToDefault(t *testing.T) {
	chain := Chain{
		QueryParamResolver{},
		CookieResolver{},
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
	chain := Chain{QueryParamResolver{}, CookieResolver{}, DefaultResolver{}}
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

func TestMiddleware_persistsValidQueryProfile(t *testing.T) {
	chain := Chain{QueryParamResolver{}, CookieResolver{}, DefaultResolver{}}
	handler := Middleware(chain)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?profile=transpara", nil)
	handler.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("middleware set %d cookies, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != CookieName || c.Value != "transpara" {
		t.Fatalf("cookie = %s=%s, want %s=transpara", c.Name, c.Value, CookieName)
	}
	if c.Path != "/" || c.MaxAge <= 0 || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie attributes = path %q maxAge %d httpOnly %v sameSite %v", c.Path, c.MaxAge, c.HttpOnly, c.SameSite)
	}
}

func TestMiddleware_doesNotPersistUnknownQueryProfile(t *testing.T) {
	chain := Chain{QueryParamResolver{}, CookieResolver{}, DefaultResolver{}}
	handler := Middleware(chain)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?profile=nonsense", nil)
	handler.ServeHTTP(rec, req)

	if cookies := rec.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("middleware set cookies for unknown profile: %+v", cookies)
	}
}

func TestMiddleware_attachesCookieProfile(t *testing.T) {
	chain := Chain{QueryParamResolver{}, CookieResolver{}, DefaultResolver{}}
	var captured *Profile
	handler := Middleware(chain)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = FromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CookieName, Value: "transpara"})
	handler.ServeHTTP(rec, req)

	if captured == nil || captured.Slug != "transpara" {
		t.Errorf("cookie middleware attached = %+v, want transpara", captured)
	}
}

func TestMiddleware_defaultOnUnknownSlug(t *testing.T) {
	chain := Chain{QueryParamResolver{}, CookieResolver{}, DefaultResolver{}}
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
