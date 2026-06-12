package svgviz

import (
	"strings"
	"testing"
)

// The transparency surface renders feeder data it does not control. Every
// renderer must (1) fail closed on inputs it cannot honestly draw — returning
// "" so the caller renders an explicit "unavailable" block instead of a
// misleading visual — and (2) escape every label, because labels arrive from
// remote APIs.

func TestGaugeFailsClosedOnUndrawableInput(t *testing.T) {
	cases := []struct {
		name       string
		value, cap float64
		w, h       int
	}{
		{"zero cap", 10, 0, 200, 24},
		{"negative cap", 10, -5, 200, 24},
		{"negative value", -1, 25, 200, 24},
		{"nan value", nan(), 25, 200, 24},
		{"nan cap", 10, nan(), 200, 24},
		{"zero width", 10, 25, 0, 24},
		{"zero height", 10, 25, 200, 0},
		{"height eats padding exactly", 10, 25, 200, 8},
		{"height below padding", 10, 25, 200, 6},
		{"width below padding", 10, 25, 7, 24},
	}
	for _, c := range cases {
		if got := Gauge(c.value, c.cap, c.w, c.h); got != "" {
			t.Errorf("%s: Gauge must return empty string, got %q", c.name, got)
		}
	}
}

func TestGaugeNormalProportions(t *testing.T) {
	// value 10 of cap 25 → fill is 40% of the inner track width.
	svg := Gauge(10, 25, 200, 24)
	if svg == "" {
		t.Fatal("expected gauge output")
	}
	for _, want := range []string{
		`<svg`, `role="img"`, `<title>10.00 of 25.00</title>`,
		`width="76.80"`, // (200-2*4) * 0.40 = 76.80
	} {
		if !strings.Contains(svg, want) {
			t.Errorf("gauge missing %q in:\n%s", want, svg)
		}
	}
	if strings.Contains(svg, "viz-over") {
		t.Error("under-cap gauge must not carry the over-cap class")
	}
}

func TestGaugeOverCapIsExplicit(t *testing.T) {
	svg := Gauge(30, 25, 200, 24)
	if svg == "" {
		t.Fatal("expected gauge output")
	}
	if !strings.Contains(svg, "viz-over") {
		t.Error("over-cap gauge must carry the viz-over class")
	}
	if !strings.Contains(svg, "<title>30.00 of 25.00</title>") {
		t.Error("over-cap gauge must state the real numbers")
	}
	// Fill must clamp to the track, never overflow the drawing.
	if !strings.Contains(svg, `width="192.00"`) { // full inner track 200-8
		t.Errorf("over-cap fill must clamp to track width, got:\n%s", svg)
	}
}

func TestBarsFailClosed(t *testing.T) {
	if Bars(nil, nil, 200, 60) != "" {
		t.Error("empty input must render nothing")
	}
	if Bars([]float64{1, 2}, []string{"only one"}, 200, 60) != "" {
		t.Error("length mismatch must render nothing")
	}
	if Bars([]float64{0, 0}, []string{"a", "b"}, 200, 60) != "" {
		t.Error("all-zero series must render nothing (no fake bars)")
	}
	if Bars([]float64{1, nan()}, []string{"a", "b"}, 200, 60) != "" {
		t.Error("NaN in series must render nothing")
	}
	if Bars([]float64{1, -2}, []string{"a", "b"}, 200, 60) != "" {
		t.Error("negative value must render nothing")
	}
}

func TestBarsProportionalAndEscaped(t *testing.T) {
	svg := Bars([]float64{1, 4}, []string{"phase <b>one</b>", "two"}, 100, 50)
	if svg == "" {
		t.Fatal("expected bars output")
	}
	if strings.Contains(svg, "<b>") {
		t.Error("labels must be escaped")
	}
	if !strings.Contains(svg, "phase &lt;b&gt;one&lt;/b&gt;") {
		t.Error("escaped label must appear in a <title>")
	}
	// max=4 fills the full usable height; value 1 is a quarter of it.
	if !strings.Contains(svg, `height="42.00"`) || !strings.Contains(svg, `height="10.50"`) {
		t.Errorf("bar heights must be proportional, got:\n%s", svg)
	}
}

func TestSpanStripWidthsAndUnknownKind(t *testing.T) {
	spans := []Span{
		{Label: "processing", Seconds: 30, Kind: "processing"},
		{Label: "mystery", Seconds: 10, Kind: "definitely-not-a-state"},
	}
	svg := SpanStrip(spans, 200, 12)
	if svg == "" {
		t.Fatal("expected span strip output")
	}
	// 3:1 split of the inner track (192): 144 and 48.
	if !strings.Contains(svg, `width="144.00"`) || !strings.Contains(svg, `width="48.00"`) {
		t.Errorf("span widths must be proportional, got:\n%s", svg)
	}
	// Unknown kinds map to an explicit unknown class, never silently styled.
	if !strings.Contains(svg, "viz-unknown") {
		t.Error("unknown kind must render the viz-unknown class")
	}
	if SpanStrip(nil, 200, 12) != "" {
		t.Error("empty spans must render nothing")
	}
	if SpanStrip([]Span{{Label: "x", Seconds: 0, Kind: "idle"}}, 200, 12) != "" {
		t.Error("zero-duration total must render nothing")
	}
	if SpanStrip([]Span{{Label: "x", Seconds: 5, Kind: "idle"}}, 200, 8) != "" {
		t.Error("height that leaves no inner track must render nothing")
	}
	if SpanStrip([]Span{{Label: "x", Seconds: 5, Kind: "idle"}}, 200, 6) != "" {
		t.Error("height below padding must render nothing")
	}
}

func TestStaircaseOrderAndEscaping(t *testing.T) {
	steps := []Step{
		{Label: "work.task.created", Sub: "actor-1"},
		{Label: "<script>alert(1)</script>", Sub: "actor-2"},
		{Label: "work.task.completed", Sub: "actor-1"},
	}
	svg := Staircase(steps, 400, 120)
	if svg == "" {
		t.Fatal("expected staircase output")
	}
	if strings.Contains(svg, "<script>") {
		t.Error("labels must be escaped")
	}
	created := strings.Index(svg, "work.task.created")
	completed := strings.Index(svg, "work.task.completed")
	if created == -1 || completed == -1 || created > completed {
		t.Error("steps must render in input order")
	}
	if Staircase(nil, 400, 120) != "" {
		t.Error("empty steps must render nothing")
	}
}

func TestEveryRendererEmitsAccessibleRoot(t *testing.T) {
	outputs := map[string]string{
		"gauge":     Gauge(10, 25, 200, 24),
		"bars":      Bars([]float64{1, 2}, []string{"a", "b"}, 100, 50),
		"spanstrip": SpanStrip([]Span{{Label: "idle", Seconds: 5, Kind: "idle"}}, 200, 12),
		"staircase": Staircase([]Step{{Label: "e", Sub: "s"}}, 400, 120),
	}
	for name, svg := range outputs {
		if svg == "" {
			t.Errorf("%s: expected output for valid input", name)
			continue
		}
		for _, want := range []string{`role="img"`, "<title>", "aria-label="} {
			if !strings.Contains(svg, want) {
				t.Errorf("%s: missing %s", name, want)
			}
		}
	}
}

func nan() float64 {
	var z float64
	return z / z
}
