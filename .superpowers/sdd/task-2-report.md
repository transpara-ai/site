# Task 2 Report: Rail templ + guards + full verify

## Scope

Implemented Task 2 (final task) of
`docs/superpowers/plans/2026-07-02-intake-recent-rail.md` on branch
`feat/intake-recent-rail` (site repo), stacked on `feat/console-ux-polish`.
TDD: failing render tests written first, then implementation, per
`superpowers:test-driven-development`.

## Field decision (Task 1 handoff)

Task 1's report flagged three extra pass-through fields it added to
`ConsoleRecentRunEntry` (`FactoryOrderID`, `BlockerType`, `RequiredAction`),
asking Task 2 to keep or drop them based on what's actually rendered. The
rail markup (per the plan) renders only IssueLabel, state text, and Age —
none of the three extras appear in any chip title or tooltip attribute. All
three were dropped, along with their assignments in `buildConsoleRecentRuns`:

- `graph/console.go`: `ConsoleRecentRunEntry` no longer declares
  `FactoryOrderID`/`BlockerType`/`RequiredAction`; the corresponding struct-
  literal assignments in `buildConsoleRecentRuns` were removed. No dead
  fields remain.

## Files changed

- `graph/console.templ`:
  - `consoleIssueScan`: inserted `if s.RecentRuns.Available { @consoleRecentRunsRail(s.RecentRuns) }`
    between the freshness header block and the unavailable/empty/board
    branches, as specified by the plan.
  - New `consoleRecentRunsRail(rail ConsoleRecentRuns)` — container
    `<div data-console-rail>`, section header `<h3 class="text-[10px]
    uppercase tracking-widest text-warm-faint">recent intakes</h3>`,
    horizontal scroll strip (`flex gap-2 overflow-x-auto`), trailing
    `… truncated` marker (`text-[10px] text-warm-faint`) iff `rail.Truncated`.
  - New `consoleRecentRunChip(entry ConsoleRecentRunEntry)` — `<a>` with
    `hx-get={entry.DrawerURL}`, `hx-target="#console-intake-drawer"`,
    `hx-swap="innerHTML"` when `entry.Linked`; otherwise a `<span>` with
    identical container styling and no hx-get, so an unlinked rail entry can
    never fabricate a drawer target the board doesn't expose.
  - New `consoleRecentRunChipContent(entry ConsoleRecentRunEntry)` — shared
    inner content: `IssueLabel` (`text-xs text-warm`), state text
    (`text-[10px]`, `text-amber-300` when `StyleKind == "amber"` else
    `text-warm-muted`), `Age` (`text-[10px] font-mono text-warm-muted`) when
    non-empty.
- `graph/console_templ.go` — regenerated via `templ generate`; committed,
  not hand-edited. Verified idempotent (`sha256sum` before/after a second
  `templ generate` run is identical — no drift).
- `graph/console.go` — dropped the three unused `ConsoleRecentRunEntry`
  fields and their assignments (see Field decision above).
- `graph/console_intake_test.go` — new rail render tests, plus two extended
  pre-existing tests (see below).

## Test domain covered

New tests (`graph/console_intake_test.go`):

- `TestConsoleRecentRunsRailRendersChipsWithDrawerLinkAndPlainSpan` — a
  board-present (linked) entry renders an `<a>` with `hx-get` (HTML-attribute-
  escaped `&`→`&amp;` matched explicitly), `hx-target="#console-intake-drawer"`,
  `hx-swap="innerHTML"`, IssueLabel, state; an unlinked entry renders its
  IssueLabel but never an `hx-get` for its own URL within the rail region.
  Also asserts `data-console-rail` and the `recent intakes` header text.
- `TestConsoleRecentRunsRailAbsentWhenUnavailableByteIdentical` — the plan's
  byte-equality guard: `RecentRuns{}` (zero value) vs
  `RecentRuns{Available:false, Entries:[stray populated entry]}` render
  byte-identical HTML; `data-console-rail` and the stray entry's IssueLabel
  are both absent. Proves the template gates on `Available`, not
  `len(Entries)`.
- `TestConsoleRecentRunsRailUnknownStateNeutralVerbatim` — an unknown state
  (`ready_for_human`) renders escaped verbatim with `text-warm-muted`, never
  `text-amber-300`.
- `TestConsoleRecentRunsRailTruncatedMarker` — `Truncated=true` renders the
  marker; `Truncated=false` does not.
- `TestConsoleIntakeSurfaceEscapesHostileProjectionData` (extended) — added
  a hostile `RecentIssueScanRuns` section (`repo`, `issue_title`, `state`
  each carrying an XSS payload: `<svg onload=...>`, `<script>...</script>`,
  `<img src=x onerror=...>`) to the existing hostile-projection struct
  literal; all three raw payloads are asserted absent from the combined
  board+drawer output (the rail renders as part of the board render).
- `TestConsoleIntakeSurfaceRendersNoWriteControls` (extended) — the fixture
  now splices in an available `recent_issue_scan_runs` section (via
  `spliceRecentIssueScanRunsSection`, reused from
  `console_recent_runs_test.go`, same package) so the rail actually renders
  on the page and fragment responses; sanity assertions confirm
  `data-console-rail` is present on both before checking for
  form/hx-post/put/delete/patch/`type="submit"`/write-route leakage — so the
  extended test isn't vacuous.

## Verification

```
GOFLAGS=-buildvcs=false go test ./graph/ -count=1        # PASS (229 subtests, 5.5-6.1s, 0 failures)
GOFLAGS=-buildvcs=false go vet ./graph/...                # clean
gofmt -l graph/console.go graph/console_templ.go graph/console_intake_test.go graph/console_recent_runs_test.go
                                                            # clean
templ generate                                             # idempotent: sha256sum(console_templ.go) unchanged across two consecutive runs
GOFLAGS=-buildvcs=false make verify                        # PASS: verify-canonical-paths, build, verify-public-shell-clean,
                                                            #   go test ./... (all packages ok), go vet ./...
```

Submodule (`third_party/hive`) was already initialized in this worktree, so
no `git submodule update --init --recursive` was needed.

## Notes / deviations

- The plan's Task 2 description mentions "state text ... amber
  `text-amber-300` when StyleKind amber else `text-warm-muted`" without
  spelling out font size for the state span explicitly (only Age is called
  out as `text-[10px] font-mono`); implemented state text at `text-[10px]`
  to match the compact-chip visual rhythm (consistent with Age and the
  section header, both `text-[10px]`) — this is the plan's own wording in
  the Global rendering description ("state text (`text-[10px]`...)").
- `href="#"` on the linked `<a>` mirrors an existing repo pattern
  (`graph/views.templ:1656`, an htmx-triggered anchor) rather than inventing
  a new idiom; the real navigation is via `hx-get`/`hx-target`/`hx-swap`, so
  the anchor never performs a full-page navigation.
- No hive-side changes were made or needed; Task 1 already covers the
  contract types and this task only adds template rendering.
