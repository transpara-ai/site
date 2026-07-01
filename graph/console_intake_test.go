package graph

import (
	"strings"
	"testing"
	"time"
)

func TestBuildConsoleIssueScanNilProjectionIsUnavailable(t *testing.T) {
	scan := buildConsoleIssueScan(nil, time.Now().UTC())
	if scan.Freshness != FreshnessUnavailable {
		t.Fatalf("freshness = %q, want unavailable", scan.Freshness)
	}
	if len(scan.Board.Columns) != 0 {
		t.Fatalf("nil projection must yield zero columns, got %d", len(scan.Board.Columns))
	}
	if len(scan.Notices) == 0 {
		t.Fatal("nil projection must carry an explicit notice")
	}
}

func TestBuildConsoleIssueScanFailedProjectionIsUnavailable(t *testing.T) {
	proj := &OpsCivilizationAssemblyProjection{
		DerivationStatus: opsCivilizationProjectionStatusFailed,
		GeneratedAt:      time.Now().UTC(), // failed sentinel carries a NON-zero timestamp
		FailureReasons:   []string{"hive civilization projection returned 503 Service Unavailable"},
	}
	scan := buildConsoleIssueScan(proj, time.Now().UTC())
	if scan.Freshness != FreshnessUnavailable {
		t.Fatalf("freshness = %q, want unavailable for failed status", scan.Freshness)
	}
	if len(scan.Notices) == 0 || !strings.Contains(scan.Notices[0], "503") {
		t.Fatalf("failed projection must surface its failure reason, got %v", scan.Notices)
	}
}

func TestBuildConsoleIssueScanZeroTimestampIsUnavailable(t *testing.T) {
	proj := &OpsCivilizationAssemblyProjection{DerivationStatus: "complete"} // GeneratedAt zero
	scan := buildConsoleIssueScan(proj, time.Now().UTC())
	if scan.Freshness != FreshnessUnavailable {
		t.Fatalf("freshness = %q, want unavailable for zero generated_at", scan.Freshness)
	}
}

func TestBuildConsoleIssueScanStaleTimestamp(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	proj := &OpsCivilizationAssemblyProjection{
		DerivationStatus: "complete",
		GeneratedAt:      now.Add(-2 * time.Minute), // older than consoleStaleWindow (30s)
	}
	scan := buildConsoleIssueScan(proj, now)
	if scan.Freshness != FreshnessStale {
		t.Fatalf("freshness = %q, want stale", scan.Freshness)
	}
}

func TestBuildConsoleIssueScanCurrentPassesBoardThrough(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	proj := &OpsCivilizationAssemblyProjection{
		DerivationStatus: "complete",
		GeneratedAt:      now.Add(-5 * time.Second),
		IssueScanProjection: OpsCivilizationIssueScanProjection{
			Runs: []OpsCivilizationIssueScanRunProjected{{
				RunID:       "run_x",
				TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 200, Title: "Do the thing"},
			}},
		},
	}
	scan := buildConsoleIssueScan(proj, now)
	if scan.Freshness != FreshnessCurrent {
		t.Fatalf("freshness = %q, want current", scan.Freshness)
	}
	if len(scan.Board.Columns) == 0 {
		t.Fatal("a projected run must produce at least one board column")
	}
}
