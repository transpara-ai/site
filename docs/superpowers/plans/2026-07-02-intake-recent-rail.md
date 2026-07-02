# Intake Recent-Intakes Rail Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render a recent-intakes rail on `/console/intake` from hive's new `recent_issue_scan_runs` projection section (hive#240, schema 1.6.0), with strict old-hive (1.5.0) byte-compatibility, plus the civilization-client timeout bump 8s→9s.

**Architecture:** Spec = site issue https://github.com/transpara-ai/site/issues/204 + hive packet `docs/designs/recent-issue-scan-runs-projection-v0.1.0.md` on hive branch `feat/ops-recent-intake-runs` (D5 is the site half; read it from /Transpara/transpara-ai/worktrees/hive-b2-recent-intakes/docs/designs/). Branch stacks on `feat/console-ux-polish` (site PR #203) — all B1 invariants hold. Contract types mirror hive's `CivilizationRecentIssueScanRuns`/`CivilizationRecentIssueScanRun` JSON (verify tags against the hive worktree file `pkg/hive/civilization_recent_issue_scan.go`).

**Tech Stack:** Go + templ (`templ generate` after any .templ edit; commit generated file; GOFLAGS=-buildvcs=false; worktree needs `git submodule update --init --recursive` once for make verify).

## Global Constraints

- Rail renders ONLY when surface freshness ∈ {current, stale, partial} (the board's exact usable set — one shared decision) AND section `status == "available"`. Absent field / unknown status / unavailable → NO rail, board byte-identical to today.
- State→style map is an allowlist over {parked, human_action: amber; queued, in_flight, recorded: neutral}; ANY other state value renders escaped text with neutral style. No healthy color by default.
- Drawer links only for rail entries whose (run_id, stage_id) exists on the rendered board (site-side index); else unlinked span. `consoleIssueScanCardURL` mechanics for links.
- Relative age from `last_event_at`; unparseable/absent → age omitted (never "now").
- Read-only; hostile-projection escaping guards extended over the new fields.
- Civilization client timeout: `graph/ops.go` `civilizationOpsProjectionClient` 8s → 9s.
- Commits conventional, ending `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

---

### Task 1: Contract types + view-model derivation + timeout + 1.5.0 compat (TDD)

**Files:** Modify `graph/civilization_issue_scan.go` (or a new `graph/civilization_recent_runs.go`) for the projection types; `graph/console.go` (`ConsoleIssueScan` gains `RecentRuns ConsoleRecentRuns`; `buildConsoleIssueScan` derives it fail-closed); `graph/ops.go:1119` timeout. Tests in `graph/console_intake_test.go`.

**Produces:** `type ConsoleRecentRuns struct { Available bool; Truncated bool; Entries []ConsoleRecentRunEntry }`; `type ConsoleRecentRunEntry struct { RunID, StageID, IssueLabel, IssueURL, State, StyleKind, Age, DrawerURL string; Linked bool }` where StyleKind ∈ {"amber","neutral"} via allowlist and DrawerURL only set when Linked. Derivation helper `buildConsoleRecentRuns(proj *OpsCivilizationAssemblyProjection, board OpsCivilizationIssueScanKanban, freshness ConsoleFreshness, now time.Time) ConsoleRecentRuns`.

Steps: failing tests first — table over: available+usable renders entries in projected order with correct StyleKind/Linked; absent section (1.5.0 payload — extend the shared fixture WITHOUT the field) → Available=false; section status unavailable/unknown → false; surface unavailable → false; unknown state → neutral + verbatim; linked only when board has the (run,stage) pair; age omitted on bad timestamp; truncated flag. Then implement; `go test ./graph/ -run 'RecentRun' -count=1` green; timeout change + a test asserting the client timeout value (read the var). Commit.

### Task 2: Rail templ + guards + full verify (TDD)

**Files:** `graph/console.templ` (rail strip above the board inside `consoleIssueScan`, rendered iff `s.RecentRuns.Available`), regenerate `console_templ.go`; tests in `graph/console_intake_test.go`.

Rail markup: horizontal scroll strip, compact chips: `<a>` (when Linked, hx-get DrawerURL, hx-target #console-intake-drawer, hx-swap innerHTML) or `<span>`; each chip shows IssueLabel, state text (amber `text-amber-300` when StyleKind amber else `text-warm-muted`), Age in `text-[10px] font-mono`. Section header `recent intakes` in the drawer-section style (`text-[10px] uppercase tracking-widest text-warm-faint`). Trailing `… truncated` marker when Truncated. Zero write affordances.

Steps: failing render tests (rail renders chips + drawer link for a board-present run; no rail when Available=false — assert byte-identical intake surface vs a build without the section, i.e. render both and require equal HTML; unknown state neutral; truncated marker; hostile issue_title/state/repo escaped — extend TestConsoleIntakeSurfaceEscapesHostileProjectionData; read-only guard extension over the rail). Implement + `templ generate`. Full: `GOFLAGS=-buildvcs=false go test ./graph/ -count=1`, `go vet ./graph/`, `templ generate` no-drift, `make verify` (submodule first). Commit.
