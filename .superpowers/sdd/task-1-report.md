# Task 1 Report: Contract types + view-model derivation + timeout + 1.5.0 compat

## Scope

Implemented Task 1 of `docs/superpowers/plans/2026-07-02-intake-recent-rail.md`
on branch `feat/intake-recent-rail` (site repo), stacked on
`feat/console-ux-polish`. TDD: failing tests written first, then
implementation, per `superpowers:test-driven-development`.

## Files changed

- `graph/civilization_recent_runs.go` (new) — `OpsCivilizationRecentIssueScanRuns`
  / `OpsCivilizationRecentIssueScanRun`, mirroring hive's
  `CivilizationRecentIssueScanRuns` / `CivilizationRecentIssueScanRun` JSON
  shape field-for-field and tag-for-tag (verified against
  `/Transpara/transpara-ai/worktrees/hive-b2-recent-intakes/pkg/hive/civilization_recent_issue_scan.go`).
- `graph/civilization.go` — added `RecentIssueScanRuns OpsCivilizationRecentIssueScanRuns
  \`json:"recent_issue_scan_runs,omitempty"\`` to `OpsCivilizationAssemblyProjection`.
  Additive + omitempty: a 1.5.0 payload with no such field decodes to the zero
  value (`Status: ""`), which the derivation treats as unavailable.
- `graph/console.go`:
  - `ConsoleIssueScan` gained a `RecentRuns ConsoleRecentRuns` field.
  - New types `ConsoleRecentRuns` (`Available`, `Truncated`, `Entries`) and
    `ConsoleRecentRunEntry` (`RunID, StageID, IssueLabel, IssueURL, State,
    StyleKind, Age, DrawerURL, Linked` plus `FactoryOrderID, BlockerType,
    RequiredAction` carried through for a future drawer/detail use, per the
    hive contract fields not explicitly excluded by the plan's Produces
    block).
  - `buildConsoleRecentRuns(proj, board, freshness, now) ConsoleRecentRuns` —
    pure, no I/O, no `time.Now()`. Two fail-closed gates, both required:
    surface freshness ∈ {current, stale, partial} (exact same set/decision as
    the board — `consoleRecentRunUsableFreshness`, shared with
    `buildConsoleIssueScan`'s own success path) AND section
    `status == "available"` (exact-match allowlist, not a denylist — absent
    field, `"unavailable"`, empty string, and any unrecognized value all
    deny).
  - `consoleRecentRunStyleKind(state)` — allowlist map `{parked,
    human_action}` → amber; every other state (queued, in_flight, recorded,
    any unknown/future value) → neutral. No `default` branch produces amber.
  - `consoleRecentRunAge(now, lastEventAt)` — thin wrapper reusing the
    existing `humanizeAge` helper (`graph/console_kanban.go`), which already
    returns `""` for empty/unparseable/future timestamps (never fabricates
    "now").
  - Linked/DrawerURL: a site-side `(run_id, stage_id)` index is built from the
    *same* `board OpsCivilizationIssueScanKanban` passed in (the rendered
    board, post `opsCivilizationIssueScanKanban`), using the existing
    `runStageKey` helper. `DrawerURL` is only set when `Linked`, via
    `consoleIssueScanCardURL` — no fabricated targets.
  - `buildConsoleIssueScan` now calls `buildConsoleRecentRuns(proj, board,
    freshness, now)` and wires the result into the returned
    `ConsoleIssueScan.RecentRuns`, sharing the exact `freshness` value already
    derived for the board (CFADA1-adv2: rail and board can never diverge on
    freshness).
- `graph/ops.go` — `civilizationOpsProjectionClient` timeout `8s → 9s` (D2).
- `graph/console_recent_runs_test.go` (new) — the TDD test suite, written
  first.

## Test domain covered

- `TestBuildConsoleRecentRunsAbsentFieldIsCompat15` — the shared
  `hiveCivilizationAssemblyProjectionFixture` (which has NO
  `recent_issue_scan_runs` field) decodes with `Available=false`, zero
  entries. This is the natural 1.5.0-compat regression case per the task
  brief.
- `TestBuildConsoleRecentRunsAvailableRendersProjectedOrder` — available +
  usable renders both entries in projected (source) order; verifies
  StyleKind, Linked/DrawerURL (linked case uses `consoleIssueScanCardURL`
  mechanics exactly; unlinked case has no DrawerURL), IssueLabel, IssueURL,
  and non-empty Age for a valid timestamp.
- `TestBuildConsoleRecentRunsSectionStatusUnavailableOrUnknown` — status ∈
  {"unavailable", "some_future_status", "", "AVAILABLE-ish"} all deny
  (allowlist, not denylist; near-miss strings don't slip through).
- `TestBuildConsoleRecentRunsSurfaceFreshnessGating` — full freshness domain:
  {current, stale, partial} render; {unavailable, an unrecognized future
  value, empty string} all deny, even with a section that itself claims
  "available".
- `TestBuildConsoleRecentRunsUnknownStateNeutralVerbatim` — an unknown state
  (`ready_for_human`, which the hive packet explicitly excludes from v1)
  renders neutral style with the state text preserved verbatim, not
  substituted or hidden.
- `TestBuildConsoleRecentRunsAllowlistStyleKind` — full v1 state domain table:
  parked/human_action → amber; queued/in_flight/recorded → neutral; plus one
  unknown value → neutral.
- `TestBuildConsoleRecentRunsAgeOmittedOnBadTimestamp` — empty, malformed, and
  syntactically-invalid-RFC3339 timestamps all yield `Age == ""`.
- `TestBuildConsoleRecentRunsTruncatedFlag` — `Truncated` mirrors the
  section's own flag both ways (true and the implicit false/omitted case).
- `TestBuildConsoleRecentRunsNilProjectionUnavailable` — nil projection input
  denies (matches `buildConsoleIssueScan`'s own nil handling).
- `TestBuildConsoleIssueScanWiresRecentRuns` — end-to-end through
  `buildConsoleIssueScan`: confirms `RecentRuns` is actually wired into the
  returned `ConsoleIssueScan` and shares the board's freshness verdict.
- `TestCivilizationOpsProjectionClientTimeoutIsNineSeconds` — reads
  `civilizationOpsProjectionClient.Timeout` directly and asserts `9 *
  time.Second`.

## Verification

```
GOFLAGS=-buildvcs=false go test ./graph/ -count=1        # PASS (5.5s, full package, all pre-existing tests included)
GOFLAGS=-buildvcs=false go vet ./graph/...                # clean
GOFLAGS=-buildvcs=false go build ./...                    # clean
gofmt -l graph/civilization_recent_runs.go graph/console.go graph/civilization.go graph/ops.go graph/console_recent_runs_test.go
                                                            # clean (civilization_recent_runs.go was reformatted once, then clean)
```

Note: `graph/console_unblock_test.go`, `graph/mind_test.go`,
`graph/personas.go` show up under `gofmt -l` on the wider `graph/` package but
are pre-existing and untouched by this task (confirmed via `git status
--porcelain` showing no diff on those files) — out of scope.

## Deviations from the plan / notes for Task 2

- The plan's Produces block for `ConsoleRecentRunEntry` lists `RunID, StageID,
  IssueLabel, IssueURL, State, StyleKind, Age, DrawerURL, Linked`. I added
  three more fields (`FactoryOrderID, BlockerType, RequiredAction`) carried
  straight through from the hive contract, unused by any Task 1 test and not
  yet consumed by any template. They cost nothing structurally (plain string
  fields, zero-valued when absent) and give Task 2's rail templ optional
  material without a second contract round-trip; if Task 2's design doesn't
  want them, they're trivial to drop. Flagging explicitly since the plan's
  struct literal didn't name them.
- `graph/civilization_recent_runs.go` is a new file rather than extending
  `graph/civilization_issue_scan.go`, matching the plan's stated file-choice
  alternative ("or a new `graph/civilization_recent_runs.go`").
- No hive-side changes were made or needed; this worktree only mirrors the
  already-committed hive contract types.
