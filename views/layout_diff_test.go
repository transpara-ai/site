package views_test

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/transpara-ai/site/profile"
	"github.com/transpara-ai/site/views"
)

// navLinkRunRE matches one or more consecutive nav-style <a> tags
// (internal-path href + the shared `hover:text-brand transition-colors`
// class signature that every profile-driven nav link carries). It does
// NOT match the footer GitHub link (href starts with "https://", not
// "/") or the "My Work" auth button (different class string), so those
// always-present-in-both-outputs anchors stay visible in the
// normalized HTML and pin the surrounding structure.
var navLinkRunRE = regexp.MustCompile(
	`(?:<a href="/[^"]*" class="hover:text-brand transition-colors[^"]*">[^<]*</a>\s*)+`,
)

// copyKeysInLayout enumerates the (key, fallback) pairs whose rendered
// output actually appears inside views.Layout. The test renders only
// Layout — Welcome / Home / APIKeysView / Discover / etc. each carry
// their own Copy keys, but their text doesn't appear in the Layout
// shell, so the normaliser doesn't need to handle them. Adding more
// keys here is the right move when the bounded-diff test grows to
// render additional templates.
//
// Each entry records the FALLBACK that the lovyou-ai (default)
// profile renders. The transpara override for the same key is read
// from the registry at test time. Both sides get rewritten to the
// same {{COPY_<key>}} placeholder so they normalise identically.
var copyKeysInLayout = map[string]string{
	"tagline": "Humans and agents, building together.",
}

// TestBoundedDiff_LayoutAcrossProfiles is the Phase 4/5 leakage alarm.
//
// It renders views.Layout twice — once per registered profile — and
// asserts two things:
//
//  1. Raw outputs DIFFER. If the two profiles produce byte-identical
//     HTML, profile divergence didn't actually land (or a future
//     refactor accidentally undid it).
//  2. After normalizing every known per-profile rendering surface to
//     a placeholder token, the outputs are byte-IDENTICAL. If
//     anything else leaks into the HTML (a new per-profile field
//     someone added without updating this test, a stray hardcoded
//     brand string, a profile-conditional block no one documented),
//     the normalized outputs diverge and this test fails loudly.
//
// The normalization set is the canonical list of divergence points:
//   - BrandName, LogoPath, AccentColor (Phase 4 scalars)
//   - Slug, scoped to the data-profile attribute (Phase 4)
//   - Contiguous runs of profile-driven nav links (Phase 5) — regex
//     collapses the whole run to {{NAV_LINKS}} because the number
//     of links legitimately differs between profiles, so per-link
//     replacement cannot produce equal-length normalized strings.
//   - Copy overrides (Phase 5) — for each key whose output appears
//     in Layout's render scope, the fallback (rendered by lovyou)
//     and the override (rendered by transpara) both rewrite to
//     {{COPY_<key>}}.
//
// New fields added to Profile must either be added here or
// explicitly justified as non-rendering.
func TestBoundedDiff_LayoutAcrossProfiles(t *testing.T) {
	lovyou := profile.Lookup(profile.DefaultSlug)
	if lovyou == nil {
		t.Fatalf("default profile %q missing from registry", profile.DefaultSlug)
	}
	transpara := profile.Lookup("transpara")
	if transpara == nil {
		t.Fatal(`secondary profile "transpara" missing from registry`)
	}

	render := func(p *profile.Profile) string {
		var buf bytes.Buffer
		if err := views.Layout("Test", "test description", p).Render(context.Background(), &buf); err != nil {
			t.Fatalf("render(%s) failed: %v", p.Slug, err)
		}
		return buf.String()
	}

	lHTML := render(lovyou)
	tHTML := render(transpara)

	if lHTML == tHTML {
		t.Fatal("Layout output is byte-identical across profiles; Phase 4 divergence didn't land, or was reverted")
	}

	normalize := func(s string, p *profile.Profile) string {
		// Slug is scoped to the data-profile attribute only. Bare
		// substring replacement would clobber unrelated URLs that
		// contain the slug as a path segment (e.g. the GitHub link
		// https://github.com/lovyou-ai in the footer).
		s = strings.ReplaceAll(s, `data-profile="`+p.Slug+`"`, `data-profile="{{SLUG}}"`)
		s = strings.ReplaceAll(s, p.BrandName, "{{BRAND}}")
		s = strings.ReplaceAll(s, p.LogoPath, "{{LOGO}}")
		s = strings.ReplaceAll(s, p.AccentColor, "{{ACCENT}}")
		// Copy overrides: for each key whose rendered text appears
		// in the Layout shell, replace both the fallback (rendered
		// by lovyou) and the override (rendered by transpara) with
		// the same placeholder. The override comes from the live
		// registry so a registry edit never silently desyncs the
		// test; the fallback is hardcoded above as the contract
		// pin between Layout's call site and this test.
		for key, fallback := range copyKeysInLayout {
			placeholder := "{{COPY_" + key + "}}"
			s = strings.ReplaceAll(s, fallback, placeholder)
			if override, ok := p.Copy[key]; ok {
				s = strings.ReplaceAll(s, override, placeholder)
			}
		}
		// Collapse any run of profile-driven nav links into a single
		// placeholder — link counts differ legitimately per profile,
		// so per-link replacement can't produce equal-length output.
		// Applied after the scalar normalizations so a future brand
		// name that happens to equal an href path segment still gets
		// canonicalised before the regex runs.
		s = navLinkRunRE.ReplaceAllString(s, "{{NAV_LINKS}}")
		return s
	}

	lNorm := normalize(lHTML, lovyou)
	tNorm := normalize(tHTML, transpara)

	if lNorm != tNorm {
		t.Fatalf("profile leakage — Layout normalized outputs differ\n%s", firstDiff(lNorm, tNorm))
	}
}

// firstDiff returns a short context window around the first byte at
// which a and b disagree, formatted for test output. Makes it easy to
// eyeball what's leaking without dumping a ~20KB full-page HTML diff.
func firstDiff(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			start := i - 40
			if start < 0 {
				start = 0
			}
			endA := i + 40
			if endA > len(a) {
				endA = len(a)
			}
			endB := i + 40
			if endB > len(b) {
				endB = len(b)
			}
			var buf strings.Builder
			buf.WriteString("  first divergence at offset ")
			buf.WriteString(intToString(i))
			buf.WriteString(":\n    a: ")
			buf.WriteString(quote(a[start:endA]))
			buf.WriteString("\n    b: ")
			buf.WriteString(quote(b[start:endB]))
			return buf.String()
		}
	}
	return "  lengths differ: " + intToString(len(a)) + " vs " + intToString(len(b))
}

func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\n':
			b.WriteString("\\n")
		case '\t':
			b.WriteString("\\t")
		case '"':
			b.WriteString("\\\"")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}
