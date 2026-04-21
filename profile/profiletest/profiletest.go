// Package profiletest provides test helpers for constructing contexts
// that carry a Profile. Keeping these out of the production profile
// package avoids forcing a dependency on the testing package into
// non-test builds.
//
// Today's profile-aware templates (simpleHeader, simpleFooter, and the
// four §G orphan pages) do not read any field on the Profile, so tests
// that omit WithDefault still pass. This helper is a Phase-2 safety net:
// the moment a layout starts reading p.Slug, any test that renders it
// against a bare context.Background() will nil-panic. WithDefault is the
// one-liner those tests can use to stay green — plug it into any
// existing *testing.T-bearing context construction, no imports beyond
// this package and standard context.
package profiletest

import (
	"context"
	"testing"

	"github.com/lovyou-ai/site/profile"
)

// WithDefault returns a context carrying the default Profile. Use in
// tests that render profile-aware templates to avoid nil panics when
// (not if) layouts begin reading fields on *profile.Profile.
func WithDefault(t *testing.T, ctx context.Context) context.Context {
	t.Helper()
	return profile.WithProfile(ctx, profile.Default())
}
