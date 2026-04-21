// Package profile defines the request-time Profile abstraction that lets
// the site host multiple visual identities under a single URL space.
//
// A Profile is a minimal, declarative bundle. Phase 3 ships with Slug and
// Name only — later phases will add theme tokens, layout shell selection,
// navigation, and route overrides as concrete need surfaces.
package profile

// Profile is the declarative presentation bundle attached to a request.
// Phase 3 holds the struct deliberately minimal: the Slug is the
// resolution key (query param, subdomain, etc.); the Name is a display
// label for future UI/logging. Layouts accept *Profile but do not yet
// read any field — byte-identical HTML between profiles is the Phase 3
// bar.
type Profile struct {
	Slug string
	Name string
}

// DefaultSlug is the slug that resolves when no other resolver matches.
const DefaultSlug = "lovyou-ai"

// registry is the authoritative set of known profiles. Keyed by Slug.
// Both entries intentionally carry the same Name value for Phase 3 so
// that any accidental read of p.Name by a layout cannot produce
// divergent HTML. Phase 4 is where Name differentiates.
var registry = map[string]*Profile{
	"lovyou-ai": {Slug: "lovyou-ai", Name: "lovyou.ai"},
	"transpara": {Slug: "transpara", Name: "lovyou.ai"},
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
