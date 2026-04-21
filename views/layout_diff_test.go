package views_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/lovyou-ai/site/profile"
	"github.com/lovyou-ai/site/views"
)

// TestBoundedDiff_LayoutAcrossProfiles is the Phase 4 leakage alarm.
//
// It renders views.Layout twice — once per registered profile — and
// asserts two things:
//
//  1. Raw outputs DIFFER. If the two profiles produce byte-identical
//     HTML, Phase 4's visible divergence didn't actually land
//     (or a future refactor accidentally undid it).
//  2. After normalizing every known per-profile field to a placeholder
//     token, the outputs are byte-IDENTICAL. If anything else leaks
//     into the HTML (a new per-profile field someone added without
//     updating this test, a stray hardcoded brand string, a
//     profile-conditional block no one documented), the normalized
//     outputs diverge and this test fails loudly.
//
// The normalization set is the canonical list of divergence points
// for Phase 4: BrandName, LogoPath, AccentColor, Slug. New fields
// added to Profile must either be added here or explicitly
// justified as non-rendering.
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
