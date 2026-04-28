package profile

import (
	"net/http"
	"os"
)

// disableEnvVar is the environment variable that, when set to any
// non-empty value, short-circuits profile resolution and attaches
// the default profile to every request. This is the cheap-revert
// surface called out in Artifact 09 constraint #4 and approach
// hint #5: if anything downstream misbehaves, setting this env
// var restores always-default behaviour without a code change.
const disableEnvVar = "PROFILE_SYSTEM_DISABLED"

// Middleware returns an HTTP middleware that resolves a Profile for
// each request using the given resolver chain and attaches it to the
// request context. The returned handler always attaches a non-nil
// Profile before invoking next — Chain.Resolve falls back to the
// default if nothing matches, and the disable flag short-circuits
// to the default without invoking the chain at all.
//
// If resolver is nil, the middleware behaves as if PROFILE_SYSTEM_DISABLED
// were set: every request gets the default profile. This is the safest
// failure mode — any configuration mistake defaults to the current
// always-default behaviour rather than a nil dereference downstream.
func Middleware(resolver Resolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var p *Profile
			if resolver == nil || os.Getenv(disableEnvVar) != "" {
				p = Default()
			} else {
				p = resolver.Resolve(r)
				if p == nil {
					p = Default()
				}
				persistQueryProfile(w, r)
			}
			next.ServeHTTP(w, r.WithContext(WithProfile(r.Context(), p)))
		})
	}
}

func persistQueryProfile(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Query().Get("profile")
	if slug == "" {
		return
	}
	p := Lookup(slug)
	if p == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    p.Slug,
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 90,
		SameSite: http.SameSiteLaxMode,
	})
}
