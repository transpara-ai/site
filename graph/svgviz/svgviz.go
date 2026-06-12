// Package svgviz renders deterministic, dependency-free SVG fragments for the
// /ops/observatory transparency surface.
//
// Contract (binding, see dark-factory transparency contract T7 and the
// visualization plan's "fail legible, not blank" principle): every renderer
// either draws the input honestly or returns "" so the caller renders an
// explicit "unavailable" block. No renderer guesses, clamps silently (except
// the over-cap gauge fill, which clamps visually while stating the real
// numbers in its title), or styles unknown states as known ones. All labels
// are escaped — feeder data is untrusted at this boundary.
package svgviz

import (
	"html"
	"math"
	"strconv"
	"strings"
)

// Span is one segment of a state timeline (e.g. an agent's FSM state over a
// window). Kind selects a CSS class from an allowlist; unknown kinds render
// with the explicit viz-unknown class.
type Span struct {
	Label   string
	Seconds float64
	Kind    string
}

// Step is one node of a causal chain (e.g. one work.task.* audit event).
type Step struct {
	Label string
	Sub   string
}

const pad = 4.0

// kindClass is the allowlist of span kinds. Anything not listed renders as
// viz-unknown — an unknown state must look unknown, never borrow a known style.
var kindClass = map[string]string{
	"processing": "viz-processing",
	"idle":       "viz-idle",
	"stuck":      "viz-stuck",
	"retired":    "viz-retired",
	"suspended":  "viz-suspended",
	"error":      "viz-error",
}

func classForKind(kind string) string {
	if c, ok := kindClass[strings.ToLower(kind)]; ok {
		return c
	}
	return "viz-unknown"
}

func bad(f float64) bool { return math.IsNaN(f) || math.IsInf(f, 0) }

func f2(f float64) string { return strconv.FormatFloat(f, 'f', 2, 64) }

func esc(s string) string { return html.EscapeString(s) }

// Gauge renders value against cap as a horizontal fill on a track. It fails
// closed (returns "") when the input cannot be drawn honestly: unknown or
// non-positive cap, negative or non-finite value, or a degenerate canvas.
// When value exceeds cap the fill clamps to the track and the viz-over class
// marks the overrun; the title always states the real numbers.
func Gauge(value, cap float64, w, h int) string {
	if bad(value) || bad(cap) || cap <= 0 || value < 0 || w <= 0 || h <= 0 {
		return ""
	}
	track := float64(w) - 2*pad
	if track <= 0 {
		return ""
	}
	ratio := value / cap
	over := ratio > 1
	if over {
		ratio = 1
	}
	fill := track * ratio
	cls := "viz-gauge"
	if over {
		cls += " viz-over"
	}
	title := f2(value) + " of " + f2(cap)
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-label="gauge: ` + esc(title) + `" viewBox="0 0 ` + strconv.Itoa(w) + ` ` + strconv.Itoa(h) + `" class="` + cls + `">`)
	b.WriteString(`<title>` + esc(title) + `</title>`)
	b.WriteString(`<rect class="viz-track" x="` + f2(pad) + `" y="` + f2(pad) + `" width="` + f2(track) + `" height="` + f2(float64(h)-2*pad) + `" rx="2" fill="none" stroke="currentColor" opacity="0.35"/>`)
	b.WriteString(`<rect class="viz-fill" x="` + f2(pad) + `" y="` + f2(pad) + `" width="` + f2(fill) + `" height="` + f2(float64(h)-2*pad) + `" rx="2" fill="currentColor"/>`)
	b.WriteString(`</svg>`)
	return b.String()
}

// Bars renders a proportional bar series with one label per value. It fails
// closed on an empty series, a length mismatch, any negative or non-finite
// value, an all-zero series, or a degenerate canvas.
func Bars(values []float64, labels []string, w, h int) string {
	if len(values) == 0 || len(values) != len(labels) || w <= 0 || h <= 0 {
		return ""
	}
	max := 0.0
	for _, v := range values {
		if bad(v) || v < 0 {
			return ""
		}
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		return ""
	}
	usableH := float64(h) - 2*pad
	usableW := float64(w) - 2*pad
	if usableH <= 0 || usableW <= 0 {
		return ""
	}
	const gap = 2.0
	barW := (usableW - gap*float64(len(values)-1)) / float64(len(values))
	if barW <= 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-label="bar chart, ` + strconv.Itoa(len(values)) + ` bars" viewBox="0 0 ` + strconv.Itoa(w) + ` ` + strconv.Itoa(h) + `" class="viz-bars">`)
	b.WriteString(`<title>bar chart, ` + strconv.Itoa(len(values)) + ` bars</title>`)
	for i, v := range values {
		bh := usableH * (v / max)
		x := pad + float64(i)*(barW+gap)
		y := pad + (usableH - bh)
		b.WriteString(`<rect x="` + f2(x) + `" y="` + f2(y) + `" width="` + f2(barW) + `" height="` + f2(bh) + `" fill="currentColor">`)
		b.WriteString(`<title>` + esc(labels[i]) + `: ` + f2(v) + `</title>`)
		b.WriteString(`</rect>`)
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// SpanStrip renders consecutive state spans as a single horizontal strip with
// widths proportional to duration. It fails closed on an empty list, any
// negative or non-finite duration, a non-positive total, or a degenerate
// canvas.
func SpanStrip(spans []Span, w, h int) string {
	if len(spans) == 0 || w <= 0 || h <= 0 {
		return ""
	}
	total := 0.0
	for _, s := range spans {
		if bad(s.Seconds) || s.Seconds < 0 {
			return ""
		}
		total += s.Seconds
	}
	if total <= 0 {
		return ""
	}
	track := float64(w) - 2*pad
	if track <= 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-label="state timeline, ` + strconv.Itoa(len(spans)) + ` spans" viewBox="0 0 ` + strconv.Itoa(w) + ` ` + strconv.Itoa(h) + `" class="viz-spanstrip">`)
	b.WriteString(`<title>state timeline, ` + strconv.Itoa(len(spans)) + ` spans</title>`)
	x := pad
	for _, s := range spans {
		sw := track * (s.Seconds / total)
		b.WriteString(`<rect class="` + classForKind(s.Kind) + `" x="` + f2(x) + `" y="` + f2(pad) + `" width="` + f2(sw) + `" height="` + f2(float64(h)-2*pad) + `" fill="currentColor">`)
		b.WriteString(`<title>` + esc(s.Label) + ` (` + f2(s.Seconds) + `s)</title>`)
		b.WriteString(`</rect>`)
		x += sw
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// Staircase renders an ordered causal chain as stepped nodes left-to-right,
// top-to-bottom. It fails closed on an empty chain or a degenerate canvas.
func Staircase(steps []Step, w, h int) string {
	if len(steps) == 0 || w <= 0 || h <= 0 {
		return ""
	}
	usableW := float64(w) - 2*pad
	usableH := float64(h) - 2*pad
	if usableW <= 0 || usableH <= 0 {
		return ""
	}
	n := float64(len(steps))
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-label="causal chain, ` + strconv.Itoa(len(steps)) + ` steps" viewBox="0 0 ` + strconv.Itoa(w) + ` ` + strconv.Itoa(h) + `" class="viz-staircase">`)
	b.WriteString(`<title>causal chain, ` + strconv.Itoa(len(steps)) + ` steps</title>`)
	for i, s := range steps {
		fi := float64(i)
		x := pad + usableW*fi/n
		y := pad + usableH*fi/n
		b.WriteString(`<circle cx="` + f2(x+3) + `" cy="` + f2(y+3) + `" r="3" fill="currentColor">`)
		b.WriteString(`<title>` + esc(s.Label) + ` — ` + esc(s.Sub) + `</title>`)
		b.WriteString(`</circle>`)
		b.WriteString(`<text x="` + f2(x+10) + `" y="` + f2(y+7) + `" class="viz-step-label" fill="currentColor">` + esc(s.Label) + `</text>`)
		if i > 0 {
			px := pad + usableW*(fi-1)/n
			py := pad + usableH*(fi-1)/n
			b.WriteString(`<line x1="` + f2(px+3) + `" y1="` + f2(py+3) + `" x2="` + f2(x+3) + `" y2="` + f2(y+3) + `" stroke="currentColor" opacity="0.5"/>`)
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}
