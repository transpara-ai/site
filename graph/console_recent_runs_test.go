package graph

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// spliceRecentIssueScanRunsSection inserts a raw `"recent_issue_scan_runs": {...}`
// JSON fragment into hiveCivilizationAssemblyProjectionFixture (which has NO
// such field — making it the natural 1.5.0-compat fixture) by string-splicing
// before the fixture's final closing brace. This keeps the shared fixture
// untouched for every other test while giving these tests a 1.6.0-shaped
// payload to decode.
func spliceRecentIssueScanRunsSection(t *testing.T, fixture string, section string) string {
	t.Helper()
	trimmed := strings.TrimRight(fixture, "\n")
	lastBrace := strings.LastIndex(trimmed, "}")
	if lastBrace == -1 {
		t.Fatal("spliceRecentIssueScanRunsSection: no closing brace found in fixture")
	}
	return trimmed[:lastBrace] + `,"recent_issue_scan_runs":` + section + trimmed[lastBrace:]
}

func decodeProjectionFixture(t *testing.T, raw string) *OpsCivilizationAssemblyProjection {
	t.Helper()
	var proj OpsCivilizationAssemblyProjection
	if err := json.Unmarshal([]byte(raw), &proj); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return &proj
}

func TestBuildConsoleRecentRunsAbsentFieldIsCompat15(t *testing.T) {
	// hiveCivilizationAssemblyProjectionFixture has NO recent_issue_scan_runs
	// field at all — this is the 1.5.0 compat case. Available must be false
	// and no entries must be invented.
	proj := decodeProjectionFixture(t, hiveCivilizationAssemblyProjectionFixture)
	now := time.Date(2026, 6, 23, 9, 30, 5, 0, time.UTC)
	board := opsCivilizationIssueScanKanban(proj)
	got := buildConsoleRecentRuns(proj, board, FreshnessCurrent, now)
	if got.Available {
		t.Fatalf("Available = true for a 1.5.0 payload with no recent_issue_scan_runs field, want false")
	}
	if len(got.Entries) != 0 {
		t.Fatalf("Entries = %d, want 0 for absent section", len(got.Entries))
	}
}

func TestBuildConsoleRecentRunsAvailableRendersProjectedOrder(t *testing.T) {
	section := `{
		"status": "available",
		"summary": "2 recent issue-scan run(s) projected.",
		"runs": [
			{"run_id": "run_a", "repo": "transpara-ai/site", "issue_number": 10, "issue_url": "https://github.com/transpara-ai/site/issues/10", "issue_title": "First", "state": "parked", "last_event_at": "2026-06-23T09:00:00Z", "stage_id": "stage_a", "source_refs": ["evt_a"]},
			{"run_id": "run_b", "repo": "transpara-ai/site", "issue_number": 11, "state": "queued", "last_event_at": "2026-06-23T09:05:00Z", "source_refs": ["evt_b"]}
		]
	}`
	raw := spliceRecentIssueScanRunsSection(t, hiveCivilizationAssemblyProjectionFixture, section)
	proj := decodeProjectionFixture(t, raw)
	board := OpsCivilizationIssueScanKanban{
		Columns: []OpsCivilizationIssueScanKanbanColumn{
			{Cards: []OpsCivilizationIssueScanKanbanCard{
				{RunID: "run_a", StageID: "stage_a"},
			}},
		},
	}
	now := time.Date(2026, 6, 23, 9, 30, 5, 0, time.UTC)
	got := buildConsoleRecentRuns(proj, board, FreshnessCurrent, now)
	if !got.Available {
		t.Fatalf("Available = false, want true for available+usable section")
	}
	if len(got.Entries) != 2 {
		t.Fatalf("Entries = %d, want 2", len(got.Entries))
	}
	// Projected order preserved: run_a first, run_b second.
	if got.Entries[0].RunID != "run_a" || got.Entries[1].RunID != "run_b" {
		t.Fatalf("order = [%s, %s], want [run_a, run_b] (projected order preserved)", got.Entries[0].RunID, got.Entries[1].RunID)
	}
	// run_a: parked -> amber, linked (on board), drawer URL set.
	a := got.Entries[0]
	if a.State != "parked" || a.StyleKind != "amber" {
		t.Errorf("run_a state/style = %q/%q, want parked/amber", a.State, a.StyleKind)
	}
	if !a.Linked {
		t.Error("run_a must be Linked: (run_a, stage_a) is on the rendered board")
	}
	if a.DrawerURL == "" {
		t.Error("run_a DrawerURL must be set when Linked")
	}
	if a.DrawerURL != consoleIssueScanCardURL(OpsCivilizationIssueScanKanbanCard{RunID: "run_a", StageID: "stage_a"}) {
		t.Errorf("run_a DrawerURL = %q, want consoleIssueScanCardURL mechanics", a.DrawerURL)
	}
	if a.IssueLabel != "transpara-ai/site#10" {
		t.Errorf("run_a IssueLabel = %q, want transpara-ai/site#10", a.IssueLabel)
	}
	if a.IssueURL != "https://github.com/transpara-ai/site/issues/10" {
		t.Errorf("run_a IssueURL = %q", a.IssueURL)
	}
	if a.Age == "" {
		t.Error("run_a Age must be populated for a valid RFC3339 last_event_at")
	}

	// run_b: queued -> neutral, NOT linked (not on board), no drawer URL.
	b := got.Entries[1]
	if b.State != "queued" || b.StyleKind != "neutral" {
		t.Errorf("run_b state/style = %q/%q, want queued/neutral", b.State, b.StyleKind)
	}
	if b.Linked {
		t.Error("run_b must NOT be Linked: (run_b, \"\") is not on the rendered board")
	}
	if b.DrawerURL != "" {
		t.Errorf("run_b DrawerURL = %q, want empty when not Linked", b.DrawerURL)
	}
}

func TestBuildConsoleRecentRunsSectionStatusUnavailableOrUnknown(t *testing.T) {
	now := time.Date(2026, 6, 23, 9, 30, 5, 0, time.UTC)
	for _, status := range []string{"unavailable", "some_future_status", "", "AVAILABLE-ish"} {
		section := `{"status": ` + jsonQuote(status) + `, "runs": [{"run_id": "run_x", "repo": "transpara-ai/site", "issue_number": 1, "state": "parked"}]}`
		raw := spliceRecentIssueScanRunsSection(t, hiveCivilizationAssemblyProjectionFixture, section)
		proj := decodeProjectionFixture(t, raw)
		board := opsCivilizationIssueScanKanban(proj)
		got := buildConsoleRecentRuns(proj, board, FreshnessCurrent, now)
		if got.Available {
			t.Errorf("status %q: Available = true, want false (fail-closed allowlist)", status)
		}
		if len(got.Entries) != 0 {
			t.Errorf("status %q: Entries = %d, want 0", status, len(got.Entries))
		}
	}
}

func TestBuildConsoleRecentRunsSurfaceFreshnessGating(t *testing.T) {
	section := `{"status": "available", "runs": [{"run_id": "run_x", "repo": "transpara-ai/site", "issue_number": 1, "state": "parked"}]}`
	raw := spliceRecentIssueScanRunsSection(t, hiveCivilizationAssemblyProjectionFixture, section)
	proj := decodeProjectionFixture(t, raw)
	board := opsCivilizationIssueScanKanban(proj)
	now := time.Date(2026, 6, 23, 9, 30, 5, 0, time.UTC)

	usable := []ConsoleFreshness{FreshnessCurrent, FreshnessStale, FreshnessPartial}
	for _, fr := range usable {
		got := buildConsoleRecentRuns(proj, board, fr, now)
		if !got.Available {
			t.Errorf("freshness %q: Available = false, want true (usable surface set)", fr)
		}
	}

	unusable := []ConsoleFreshness{FreshnessUnavailable, ConsoleFreshness("future-state"), ConsoleFreshness("")}
	for _, fr := range unusable {
		got := buildConsoleRecentRuns(proj, board, fr, now)
		if got.Available {
			t.Errorf("freshness %q: Available = true, want false (not in usable set)", fr)
		}
		if len(got.Entries) != 0 {
			t.Errorf("freshness %q: Entries = %d, want 0", fr, len(got.Entries))
		}
	}
}

func TestBuildConsoleRecentRunsUnknownStateNeutralVerbatim(t *testing.T) {
	section := `{"status": "available", "runs": [{"run_id": "run_x", "repo": "transpara-ai/site", "issue_number": 1, "state": "ready_for_human"}]}`
	raw := spliceRecentIssueScanRunsSection(t, hiveCivilizationAssemblyProjectionFixture, section)
	proj := decodeProjectionFixture(t, raw)
	board := opsCivilizationIssueScanKanban(proj)
	now := time.Date(2026, 6, 23, 9, 30, 5, 0, time.UTC)
	got := buildConsoleRecentRuns(proj, board, FreshnessCurrent, now)
	if !got.Available {
		t.Fatal("Available = false, want true")
	}
	if len(got.Entries) != 1 {
		t.Fatalf("Entries = %d, want 1", len(got.Entries))
	}
	e := got.Entries[0]
	if e.StyleKind != "neutral" {
		t.Errorf("StyleKind = %q, want neutral for unknown state ready_for_human (no healthy default)", e.StyleKind)
	}
	if e.State != "ready_for_human" {
		t.Errorf("State = %q, want verbatim ready_for_human", e.State)
	}
}

func TestBuildConsoleRecentRunsAllowlistStyleKind(t *testing.T) {
	now := time.Date(2026, 6, 23, 9, 30, 5, 0, time.UTC)
	cases := map[string]string{
		"parked":       "amber",
		"human_action": "amber",
		"queued":       "neutral",
		"in_flight":    "neutral",
		"recorded":     "neutral",
		"unknown_xyz":  "neutral",
	}
	for state, want := range cases {
		section := `{"status": "available", "runs": [{"run_id": "run_x", "repo": "transpara-ai/site", "issue_number": 1, "state": ` + jsonQuote(state) + `}]}`
		raw := spliceRecentIssueScanRunsSection(t, hiveCivilizationAssemblyProjectionFixture, section)
		proj := decodeProjectionFixture(t, raw)
		board := opsCivilizationIssueScanKanban(proj)
		got := buildConsoleRecentRuns(proj, board, FreshnessCurrent, now)
		if len(got.Entries) != 1 {
			t.Fatalf("state %q: Entries = %d, want 1", state, len(got.Entries))
		}
		if got.Entries[0].StyleKind != want {
			t.Errorf("state %q: StyleKind = %q, want %q", state, got.Entries[0].StyleKind, want)
		}
	}
}

func TestBuildConsoleRecentRunsAgeOmittedOnBadTimestamp(t *testing.T) {
	now := time.Date(2026, 6, 23, 9, 30, 5, 0, time.UTC)
	for _, ts := range []string{"", "not-a-timestamp", "2026-99-99T99:99:99Z"} {
		var tsField string
		if ts == "" {
			tsField = ""
		} else {
			tsField = `, "last_event_at": ` + jsonQuote(ts)
		}
		section := `{"status": "available", "runs": [{"run_id": "run_x", "repo": "transpara-ai/site", "issue_number": 1, "state": "parked"` + tsField + `}]}`
		raw := spliceRecentIssueScanRunsSection(t, hiveCivilizationAssemblyProjectionFixture, section)
		proj := decodeProjectionFixture(t, raw)
		board := opsCivilizationIssueScanKanban(proj)
		got := buildConsoleRecentRuns(proj, board, FreshnessCurrent, now)
		if len(got.Entries) != 1 {
			t.Fatalf("timestamp %q: Entries = %d, want 1", ts, len(got.Entries))
		}
		if got.Entries[0].Age != "" {
			t.Errorf("timestamp %q: Age = %q, want empty (never fabricate 'now')", ts, got.Entries[0].Age)
		}
	}
}

func TestBuildConsoleRecentRunsTruncatedFlag(t *testing.T) {
	now := time.Date(2026, 6, 23, 9, 30, 5, 0, time.UTC)
	section := `{"status": "available", "truncated": true, "runs": [{"run_id": "run_x", "repo": "transpara-ai/site", "issue_number": 1, "state": "parked"}]}`
	raw := spliceRecentIssueScanRunsSection(t, hiveCivilizationAssemblyProjectionFixture, section)
	proj := decodeProjectionFixture(t, raw)
	board := opsCivilizationIssueScanKanban(proj)
	got := buildConsoleRecentRuns(proj, board, FreshnessCurrent, now)
	if !got.Truncated {
		t.Error("Truncated = false, want true")
	}

	sectionFalse := `{"status": "available", "runs": [{"run_id": "run_x", "repo": "transpara-ai/site", "issue_number": 1, "state": "parked"}]}`
	rawFalse := spliceRecentIssueScanRunsSection(t, hiveCivilizationAssemblyProjectionFixture, sectionFalse)
	projFalse := decodeProjectionFixture(t, rawFalse)
	boardFalse := opsCivilizationIssueScanKanban(projFalse)
	gotFalse := buildConsoleRecentRuns(projFalse, boardFalse, FreshnessCurrent, now)
	if gotFalse.Truncated {
		t.Error("Truncated = true, want false when the section did not set it")
	}
}

func TestBuildConsoleRecentRunsNilProjectionUnavailable(t *testing.T) {
	now := time.Date(2026, 6, 23, 9, 30, 5, 0, time.UTC)
	got := buildConsoleRecentRuns(nil, OpsCivilizationIssueScanKanban{}, FreshnessCurrent, now)
	if got.Available {
		t.Fatal("Available = true for nil projection, want false")
	}
	if len(got.Entries) != 0 {
		t.Fatalf("Entries = %d, want 0 for nil projection", len(got.Entries))
	}
}

func TestBuildConsoleIssueScanWiresRecentRuns(t *testing.T) {
	// End-to-end through buildConsoleIssueScan: RecentRuns must be derived and
	// attached, sharing the same freshness decision as the board.
	section := `{"status": "available", "runs": [{"run_id": "run_x", "repo": "transpara-ai/site", "issue_number": 1, "state": "parked"}]}`
	raw := spliceRecentIssueScanRunsSection(t, hiveCivilizationAssemblyProjectionFixture, section)
	proj := decodeProjectionFixture(t, raw)
	// Rewrite generated_at to be fresh relative to `now` passed below.
	now := time.Now().UTC()
	proj.GeneratedAt = now.Add(-5 * time.Second)
	scan := buildConsoleIssueScan(proj, now)
	if scan.Freshness != FreshnessCurrent {
		t.Fatalf("freshness = %q, want current", scan.Freshness)
	}
	if !scan.RecentRuns.Available {
		t.Fatal("RecentRuns.Available = false, want true wired through buildConsoleIssueScan")
	}
	if len(scan.RecentRuns.Entries) != 1 {
		t.Fatalf("RecentRuns.Entries = %d, want 1", len(scan.RecentRuns.Entries))
	}
}

func TestCivilizationOpsProjectionClientTimeoutIsNineSeconds(t *testing.T) {
	if civilizationOpsProjectionClient.Timeout != 9*time.Second {
		t.Fatalf("civilizationOpsProjectionClient.Timeout = %s, want 9s", civilizationOpsProjectionClient.Timeout)
	}
}

func jsonQuote(s string) string {
	// Minimal JSON string quoting sufficient for test fixture construction
	// (no embedded quotes/backslashes in the test inputs above).
	return `"` + s + `"`
}
