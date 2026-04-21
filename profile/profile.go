// Package profile defines the request-time Profile abstraction that lets
// the site host multiple visual identities under a single URL space.
//
// Phase 4 (first divergence): a Profile now carries enough data to
// produce visibly different HTML — brand name, logo path, accent
// color. Byte-identical-output is over; templates read these fields
// through nil-safe accessors so a missing Profile falls back to the
// default branding rather than panicking.
package profile

// Profile is the declarative presentation bundle attached to a request.
//
// Slug is the resolution key (query param, subdomain, etc.) and
// uniquely identifies the profile. BrandName, LogoPath, and
// AccentColor drive the visible chrome. Name is retained for backward
// compatibility with earlier phases — new code should read BrandName.
type Profile struct {
	Slug        string // resolution key
	Name        string // display label (kept for backward compat)
	BrandName   string // what appears in headers, titles, meta
	LogoPath    string // path to logo asset (e.g. /static/logo-lovyou.svg)
	AccentColor string // hex color for CSS --accent property
}

// DefaultSlug is the slug that resolves when no other resolver matches.
const DefaultSlug = "lovyou-ai"

// registry is the authoritative set of known profiles. Keyed by Slug.
// Phase 4 is the point at which the entries intentionally diverge —
// different BrandName, different LogoPath, different AccentColor
// produce visibly different chrome per request.
var registry = map[string]*Profile{
	"lovyou-ai": {
		Slug:        "lovyou-ai",
		Name:        "lovyou.ai",
		BrandName:   "lovyou.ai",
		LogoPath:    "/static/logo-lovyou.svg",
		AccentColor: "#e8a0b8",
	},
	"transpara": {
		Slug:        "transpara",
		Name:        "Transpara",
		BrandName:   "Transpara",
		LogoPath:    "/static/logo-transpara.svg",
		AccentColor: "#0ea5e9",
	},
}

// Default returns the fallback profile used when no resolver matches.
func Default() *Profile {
	return registry[DefaultSlug]
}

// Lookup returns the profile for the given slug, or nil if unknown.
// Callers use nil as the signal to continue resolver-chain traversal.
func Lookup(slug string) *Profile {
	return registry[slug]
}

// All returns a slice of every registered profile. Order is not
// guaranteed — iteration helper for tests and future admin surfaces.
func All() []*Profile {
	out := make([]*Profile, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	return out
}

// GetBrandName returns the brand name for rendering, falling back to
// the default profile's brand name if the receiver is nil. Templates
// must call this instead of reading p.BrandName directly so that any
// code path which produces a nil *Profile renders default branding
// instead of nil-panicking.
func (p *Profile) GetBrandName() string {
	if p == nil {
		return Default().BrandName
	}
	return p.BrandName
}

// GetLogoPath returns the logo asset path, falling back to default on nil.
func (p *Profile) GetLogoPath() string {
	if p == nil {
		return Default().LogoPath
	}
	return p.LogoPath
}

// GetAccentColor returns the accent color hex, falling back to default on nil.
func (p *Profile) GetAccentColor() string {
	if p == nil {
		return Default().AccentColor
	}
	return p.AccentColor
}
