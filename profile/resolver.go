package profile

import "net/http"

// CookieName stores the operator-selected profile for subsequent same-host
// navigation. QueryParamResolver remains the explicit override; this cookie
// is only the persistence layer after that override is chosen.
const CookieName = "lovyou_profile"

// Resolver maps an incoming request to a Profile, or returns nil to
// defer to the next resolver in a chain. The interface is deliberately
// small: every future resolver (subdomain, cookie, host header, …)
// implements this one method and slots into the chain unchanged.
type Resolver interface {
	Resolve(r *http.Request) *Profile
}

// QueryParamResolver resolves a Profile from the "profile" URL query
// parameter. Empty or unknown slugs return nil so the chain may
// continue.
type QueryParamResolver struct{}

// Resolve reads ?profile=<slug> and looks it up in the registry.
func (QueryParamResolver) Resolve(r *http.Request) *Profile {
	slug := r.URL.Query().Get("profile")
	if slug == "" {
		return nil
	}
	return Lookup(slug)
}

// CookieResolver resolves a Profile from the profile persistence cookie.
// Empty, missing, or unknown cookie values return nil so the chain may
// continue to the default resolver.
type CookieResolver struct{}

// Resolve reads the persisted profile slug from CookieName.
func (CookieResolver) Resolve(r *http.Request) *Profile {
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	return Lookup(c.Value)
}

// DefaultResolver always returns the default profile. Place it last in
// a Chain so it terminates resolution for any request that no earlier
// resolver recognised.
type DefaultResolver struct{}

// Resolve always returns the registry's default profile.
func (DefaultResolver) Resolve(*http.Request) *Profile {
	return Default()
}

// Chain is a sequence of Resolvers tried in order; the first non-nil
// result wins. Chain itself implements Resolver, so a chain composes
// into a larger chain without wrapping.
//
// Adding a future resolver (subdomain, cookie, host header) is one
// line at the chain's construction site; the rest of the HTTP stack
// stays unchanged. That property is the whole point of this phase.
type Chain []Resolver

// Resolve walks the chain in order and returns the first non-nil
// profile produced. If every resolver returns nil, Resolve falls
// back to the default profile so callers never observe a nil Profile
// from a properly-constructed chain.
func (c Chain) Resolve(r *http.Request) *Profile {
	for _, res := range c {
		if p := res.Resolve(r); p != nil {
			return p
		}
	}
	return Default()
}
