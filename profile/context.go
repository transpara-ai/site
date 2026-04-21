package profile

import "context"

// contextKey is an unexported type used as the map key for the Profile
// attached to a context.Context. An unexported custom type cannot
// collide with any other package's context key, even one that happens
// to use the same literal value — the key type itself is the namespace.
// Never use a string literal as a context key; silent collisions waste
// hours.
type contextKey struct{}

// profileKey is the sole instance of contextKey used by this package.
// All WithProfile/FromContext lookups go through it.
var profileKey = contextKey{}

// WithProfile returns a child context that carries the given Profile.
// The Profile is attached under an unexported key so no other package
// can read or overwrite it.
func WithProfile(ctx context.Context, p *Profile) context.Context {
	return context.WithValue(ctx, profileKey, p)
}

// FromContext returns the Profile attached to ctx, or nil if none was
// attached. Callers that require a non-nil Profile (typical layout
// render path) should treat nil as a bug in middleware ordering — the
// middleware attaches a Profile for every request reaching a handler.
func FromContext(ctx context.Context) *Profile {
	p, _ := ctx.Value(profileKey).(*Profile)
	return p
}
