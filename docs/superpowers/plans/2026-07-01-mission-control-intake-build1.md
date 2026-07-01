# Mission Control Intake — Build 1 (Channel B: console issue-scan surface) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only issue-scan lifecycle board at `/console/intake` under the Mission Control Intake tab, reusing the existing `OpsCivilizationIssueScanKanban` builder, and retire the three legacy issue-scan sections from `/ops/civilization` (keeping the shared builder).

**Architecture:** Pure projection re-render. A new console view-model (`ConsoleIssueScan`) wraps the *existing, unchanged* `opsCivilizationIssueScanKanban(projection)` builder and adds an honest-staleness freshness state. Handlers mirror `handleConsoleKanban` (fetch → build → render). The Intake tab is enabled; routes are registered in both `Register` and `RegisterReadOnlyConsole`. Retirement removes only the *rendering* of three sections in `civilization.templ`; the builder, its type, the handler's data population, and the observation/canary consumer (`opsObservationIssueScanCards`) are all retained.

**Tech Stack:** Go (net/http, stdlib), Templ (regenerate `*_templ.go` after every `.templ` edit), HTMX (CDN, already wired), `go test ./graph/...`.

## Global Constraints

- **ZERO backend changes.** No edits under `../hive`, `../work`, or `../eventgraph`. Site is a read-only projection consumer.
- **Read-only. No writes, no persistence, no new mutation controls.** Every route is `GET`.
- **Honest-staleness / no-fabrication.** A `nil` projection, a `failed`-status projection, or a missing/zero `generated_at` renders as explicit `unavailable` — never a comforting default and never invented cards. Fail closed.
- **Reuse, do not rewrite** `graph/civilization_issue_scan.go` (~631 LOC) and the `OpsCivilizationIssueScanKanban` type. Consume its entry point `opsCivilizationIssueScanKanban(projection *OpsCivilizationAssemblyProjection)`.
- **Keep the shared builder alive after retirement.** `OpsCivilizationIssueScanKanban` + `opsCivilizationIssueScanKanban` + `opsObservationIssueScanCards` are shared with the observation/canary surface (`graph/ops.go:2246`). Retirement removes rendering only.
- **Always run `templ generate` after editing any `.templ` file.** The generated `*_templ.go` files are committed. Never hand-edit them.
- **Freshness constants and helpers already exist** in `graph/console.go`: `ConsoleFreshness`, `FreshnessCurrent/Stale/Partial/Unavailable`, `deriveFreshness(generatedAt string, fetchErr error, hasPartialErrors bool, now time.Time, staleWindow time.Duration)`, `consoleStaleWindow`. Reuse them; do not redefine.
- **Verify (run from `repos/site`):** `templ generate && go build ./... && go vet ./... && go test ./graph/...`
- **Conventional commits**, lowercase imperative subject, no trailing period. Commit at the end of each task.
- **Branch:** work stays on `feat/mission-control-intake-design` (already checked out; carries the design commits f4fa9b8, a884f74, a3511e3). Do NOT commit to `main`. Do NOT push.

---

## File Structure

- `graph/console.go` (modify) — add `ConsoleIssueScan` type, `ConsolePageData.IssueScan` field, `buildConsoleIssueScan`, the intake handlers, the drawer handler, and the small card-rendering helper funcs. Add `"strings"` to imports.
- `graph/console.templ` (modify) — enable the Intake tab; render `data.IssueScan`; add `consoleIssueScanFragment`, `consoleIssueScan`, `consoleIssueScanCard`, `consoleIssueScanDrawer`. Regenerate `graph/console_templ.go`.
- `graph/handlers.go` (modify) — register the three intake routes in both `Register` (via `writeWrap`) and `RegisterReadOnlyConsole` (via `HandleFunc`).
- `graph/civilization.templ` (modify) — remove the three issue-scan render sections; add a pointer. Regenerate `graph/civilization_templ.go`.
- `graph/console_intake_test.go` (create) — unit tests for `buildConsoleIssueScan` + handler render/staleness/drawer tests, mirroring `console_kanban_test.go`.
- `graph/handlers_test.go` (modify, Task 4 only) — reconcile the `/ops/civilization` assertion list; add a retirement regression test.

---

## Task 1: Console issue-scan view-model + builder

Pure function that wraps the existing kanban builder with an honest-staleness freshness state. No HTTP, no templ — unit-tested in isolation.

**Files:**
- Modify: `graph/console.go` (add import `"strings"`; add `ConsoleIssueScan` type; add `IssueScan *ConsoleIssueScan` to `ConsolePageData`; add `buildConsoleIssueScan`)
- Test: `graph/console_intake_test.go` (create)

**Interfaces:**
- Consumes (already exist): `opsCivilizationIssueScanKanban(projection *OpsCivilizationAssemblyProjection) OpsCivilizationIssueScanKanban`; `OpsCivilizationAssemblyProjection` (fields `GeneratedAt time.Time`, `DerivationStatus string`, `FailureReasons []string`); constants `opsCivilizationProjectionStatusFailed`, `opsCivilizationProjectionStatusPartial`; `deriveFreshness`, `consoleStaleWindow`, `FreshnessUnavailable`.
- Produces (later tasks rely on these exact names/types): `type ConsoleIssueScan struct { Freshness ConsoleFreshness; GeneratedAt string; Status string; Summary string; Board OpsCivilizationIssueScanKanban; Notices []string }` and `func buildConsoleIssueScan(proj *OpsCivilizationAssemblyProjection, now time.Time) ConsoleIssueScan`; field `ConsolePageData.IssueScan *ConsoleIssueScan`.

- [ ] **Step 1: Write the failing tests**

Create `graph/console_intake_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./graph/ -run TestBuildConsoleIssueScan -v`
Expected: FAIL — `undefined: buildConsoleIssueScan` (and `ConsoleIssueScan`).

- [ ] **Step 3: Add the import, view-model type, page-data field, and builder**

In `graph/console.go`, add `"strings"` to the import block (keep `"net/http"`, `"time"`, `"github.com/transpara-ai/site/profile"`):

```go
import (
	"net/http"
	"strings"
	"time"

	"github.com/transpara-ai/site/profile"
)
```

Add `IssueScan` to `ConsolePageData` (place after `Kanban`):

```go
type ConsolePageData struct {
	Title     string
	Active    string // health | kanban | intake | config
	Health    *ConsoleHealthWall
	Kanban    *ConsoleKanban
	IssueScan *ConsoleIssueScan
}
```

Add the view-model and builder (place near the other console builders):

```go
// ConsoleIssueScan is the Intake-tab view-model: the reused issue-scan kanban
// board plus an explicit freshness state derived from the civilization
// projection. It fails closed — a nil, failed, or timestamp-less projection is
// rendered as unavailable with a human-readable notice and no invented cards.
type ConsoleIssueScan struct {
	Freshness   ConsoleFreshness
	GeneratedAt string
	Status      string
	Summary     string
	Board       OpsCivilizationIssueScanKanban
	Notices     []string
}

// buildConsoleIssueScan maps the civilization assembly projection (or its
// absence/failure) into the Intake view-model. It reuses the existing
// opsCivilizationIssueScanKanban builder verbatim and derives honest staleness.
func buildConsoleIssueScan(proj *OpsCivilizationAssemblyProjection, now time.Time) ConsoleIssueScan {
	board := opsCivilizationIssueScanKanban(proj) // nil-safe: returns an unavailable board
	if proj == nil {
		return ConsoleIssueScan{
			Freshness: FreshnessUnavailable,
			Status:    board.Status,
			Summary:   board.Summary,
			Board:     board,
			Notices:   []string{"civilization projection unavailable to Site"},
		}
	}
	if strings.EqualFold(strings.TrimSpace(proj.DerivationStatus), opsCivilizationProjectionStatusFailed) {
		return ConsoleIssueScan{
			Freshness: FreshnessUnavailable,
			Status:    board.Status,
			Summary:   board.Summary,
			Board:     board,
			Notices:   append([]string(nil), proj.FailureReasons...),
		}
	}
	if proj.GeneratedAt.IsZero() {
		return ConsoleIssueScan{
			Freshness: FreshnessUnavailable,
			Status:    board.Status,
			Summary:   board.Summary,
			Board:     board,
			Notices:   []string{"projection missing generated_at timestamp"},
		}
	}
	generatedAt := proj.GeneratedAt.UTC().Format(time.RFC3339)
	hasPartial := strings.EqualFold(strings.TrimSpace(proj.DerivationStatus), opsCivilizationProjectionStatusPartial)
	return ConsoleIssueScan{
		Freshness:   deriveFreshness(generatedAt, nil, hasPartial, now, consoleStaleWindow),
		GeneratedAt: generatedAt,
		Status:      board.Status,
		Summary:     board.Summary,
		Board:       board,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./graph/ -run TestBuildConsoleIssueScan -v`
Expected: PASS (all five).

- [ ] **Step 5: Commit**

```bash
git add graph/console.go graph/console_intake_test.go
git commit -m "feat: add console issue-scan view-model with honest staleness"
```

---

## Task 2: Intake board surface — templ, handlers, routes, tab

Renders the board at `GET /console/intake` (+ `/fragment`), enables the Intake tab, and wires routes in both registrars. Cards lead with the issue, stage, working agents, blocker, and the ready/no-merge state. The drawer wiring is added in Task 3; here the card is a plain (non-clickable) block so this task is independently reviewable.

**Files:**
- Modify: `graph/console.templ` (enable tab line ~25; render `data.IssueScan` in `ConsolePage`; add `consoleIssueScanFragment`, `consoleIssueScan`, `consoleIssueScanCard`)
- Modify: `graph/console.go` (add handlers `handleConsoleIntake`, `handleConsoleIntakeFragment`; add card-render helper funcs)
- Modify: `graph/handlers.go` (routes in `Register` and `RegisterReadOnlyConsole`)
- Regenerate: `graph/console_templ.go` (via `templ generate`)
- Test: `graph/console_intake_test.go` (append handler tests)

**Interfaces:**
- Consumes: `ConsoleIssueScan`, `buildConsoleIssueScan`, `ConsolePageData.IssueScan` (Task 1); existing `fetchOpsCivilizationProjection(r *http.Request) *OpsCivilizationAssemblyProjection`; existing helpers `opsCivilizationIssueRefLabel`, `opsCivilizationValue`, `noticeText`; existing templ `consoleFreshnessBadge`.
- Produces: handlers `handleConsoleIntake`, `handleConsoleIntakeFragment`; helper funcs `consoleIssueScanCardIssue`, `consoleIssueScanCardTitle`, `consoleIssueScanCardAgents`, `consoleIssueScanCardBlocker`, `consoleIssueScanCardReady`; templ funcs `consoleIssueScanFragment`, `consoleIssueScan`, `consoleIssueScanCard`.

- [ ] **Step 1: Write the failing handler tests**

Append to `graph/console_intake_test.go`:

```go
func TestConsoleIntakeRendersIssueScanBoard(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hive/civilization/assembly-projection" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, hiveCivilizationAssemblyProjectionFixture)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	// Issue ref, a working agent, a stage, and a blocker from the fixture.
	for _, want := range []string{"transpara-ai/docs#172", "agent_reviewer", "run_adversarial_review", "duplicate_chain"} {
		if !strings.Contains(body, want) {
			t.Errorf("intake board missing %q", want)
		}
	}
	// This shared fixture has NO ready_for_human card (states are parked /
	// human_action only), so the "not merged" affordance is asserted in the
	// dedicated card-render tests below, not here.
}

func TestConsoleIssueScanCardStatesNoMergeWhenReady(t *testing.T) {
	card := OpsCivilizationIssueScanKanbanCard{
		RunID:        "run_ready",
		StageID:      "surface_ready_for_human_result_pr",
		CurrentState: "ready_for_human",
		TargetIssue:  OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 400},
	}
	var buf bytes.Buffer
	if err := consoleIssueScanCard(card).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), "not merged") {
		t.Error("a ready_for_human card must state the no-merge boundary")
	}
}

func TestConsoleIssueScanCardOmitsNoMergeWhenNotReady(t *testing.T) {
	card := OpsCivilizationIssueScanKanbanCard{RunID: "r", StageID: "s", CurrentState: "parked"}
	var buf bytes.Buffer
	if err := consoleIssueScanCard(card).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(buf.String(), "not merged") {
		t.Error("a non-ready card must not claim a no-merge boundary")
	}
}

func TestConsoleIntakeUnavailableWhenProjectionAbsent(t *testing.T) {
	t.Setenv("HIVE_OPS_API_BASE_URL", "") // no upstream configured -> nil projection

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "unavailable") {
		t.Error("absent projection must render an explicit unavailable state")
	}
	// No fabricated cards from the fixture may appear.
	if strings.Contains(body, "transpara-ai/docs#172") {
		t.Error("unavailable board must not fabricate issue-scan cards")
	}
}

func TestConsoleIntakeTabEnabled(t *testing.T) {
	t.Setenv("HIVE_OPS_API_BASE_URL", "")
	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()
	// The enabled Intake tab is an anchor to /console/intake, not a disabled span.
	if !strings.Contains(body, `href="/console/intake"`) {
		t.Error("Intake tab must be enabled (anchor to /console/intake)")
	}
}
```

Add these imports to the top of `graph/console_intake_test.go` (merge with the existing block; `bytes` and `context` are for the direct card-render tests):

```go
import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./graph/ -run TestConsoleIntake -v`
Expected: FAIL — `undefined: handleConsoleIntake` / route not registered / tab still disabled.

- [ ] **Step 3: Add the card-render helper funcs**

In `graph/console.go`, add:

```go
// consoleIssueScanCardIssue renders the leading issue reference (repo#number,
// falling back to URL/title) for an issue-scan card, preferring the target
// issue and falling back to the selected candidate.
func consoleIssueScanCardIssue(card OpsCivilizationIssueScanKanbanCard) string {
	ref := card.TargetIssue
	if ref.Repo == "" && ref.Number == 0 {
		ref = card.SelectedIssue
	}
	return opsCivilizationIssueRefLabel(ref)
}

func consoleIssueScanCardTitle(card OpsCivilizationIssueScanKanbanCard) string {
	if card.TargetIssue.Title != "" {
		return card.TargetIssue.Title
	}
	return card.SelectedIssue.Title
}

// consoleIssueScanCardAgents lists the possessing agents — assigned first, then
// touching — or "unassigned" when the projection names none. Never invented.
func consoleIssueScanCardAgents(card OpsCivilizationIssueScanKanbanCard) string {
	if len(card.AssignedAgentIDs) > 0 {
		return strings.Join(card.AssignedAgentIDs, ", ")
	}
	if len(card.TouchingAgentIDs) > 0 {
		return strings.Join(card.TouchingAgentIDs, ", ")
	}
	return "unassigned"
}

func consoleIssueScanCardBlocker(card OpsCivilizationIssueScanKanbanCard) string {
	if len(card.Blockers) == 0 {
		return ""
	}
	b := card.Blockers[0]
	if b.RequiredAction != "" {
		return b.BlockerType + " — " + b.RequiredAction
	}
	return b.BlockerType
}

// consoleIssueScanCardReady reports whether the card is in the terminal
// ready-for-human state, so the board can state the no-merge boundary honestly.
func consoleIssueScanCardReady(card OpsCivilizationIssueScanKanbanCard) bool {
	return strings.EqualFold(strings.TrimSpace(card.CurrentState), "ready_for_human")
}
```

- [ ] **Step 4: Add the handlers**

In `graph/console.go`, add (mirroring `handleConsoleKanban`):

```go
func (h *Handlers) handleConsoleIntake(w http.ResponseWriter, r *http.Request) {
	proj := fetchOpsCivilizationProjection(r)
	scan := buildConsoleIssueScan(proj, time.Now().UTC())
	h.renderConsole(w, r, ConsolePageData{Title: "Intake", Active: "intake", IssueScan: &scan})
}

func (h *Handlers) handleConsoleIntakeFragment(w http.ResponseWriter, r *http.Request) {
	proj := fetchOpsCivilizationProjection(r)
	scan := buildConsoleIssueScan(proj, time.Now().UTC())
	consoleIssueScanFragment(scan).Render(r.Context(), w)
}
```

- [ ] **Step 5: Enable the tab and render the board in `console.templ`**

In `graph/console.templ`, change the Intake tab's `enabled` argument from `false` to `true` (line ~25):

```templ
					@consoleTab("intake", "Intake", "/console/intake", data.Active, true)
```

In `ConsolePage`, add an `IssueScan` render block after the existing `data.Kanban` block (the drawer div is a sibling of the polled fragment — Plan-2b lesson: never inside the polled subtree):

```templ
				if data.IssueScan != nil {
					@consoleIssueScanFragment(*data.IssueScan)
					<div id="console-intake-drawer"></div>
				}
```

Add the board templ funcs (place after `consoleKanban` / near the other console fragments). Note `strconv` is already imported in `console.templ`:

```templ
templ consoleIssueScanFragment(s ConsoleIssueScan) {
	<div id="console-intake" hx-get="/console/intake/fragment" hx-trigger="every 10s" hx-swap="outerHTML">
		@consoleIssueScan(s)
	</div>
}

templ consoleIssueScan(s ConsoleIssueScan) {
	<section class="space-y-4" data-console-surface="intake">
		<div class="flex items-center justify-between gap-3">
			<div>
				<h2 class="text-lg font-medium text-warm">Intake — issue-scan lifecycle</h2>
				if s.Summary != "" {
					<p class="text-xs text-warm-muted mt-1">{ s.Summary }</p>
				}
			</div>
			@consoleFreshnessBadge(s.Freshness, s.GeneratedAt)
		</div>
		if s.Freshness == FreshnessUnavailable {
			<p class="text-sm text-warm-muted" data-state="unavailable">unavailable — { noticeText(s.Notices) }</p>
		} else if len(s.Board.Columns) == 0 {
			<p class="text-sm text-warm-muted">No issue-scan runs projected.</p>
		} else {
			<div class="flex gap-4 overflow-x-auto pb-2">
				for _, col := range s.Board.Columns {
					<section class="min-w-[18rem] flex-shrink-0 space-y-2">
						<h3 class="text-xs uppercase tracking-wide text-warm-muted">{ col.Label } · { strconv.Itoa(len(col.Cards)) }</h3>
						for _, card := range col.Cards {
							@consoleIssueScanCard(card)
						}
					</section>
				}
			</div>
		}
	</section>
}

templ consoleIssueScanCard(card OpsCivilizationIssueScanKanbanCard) {
	<div class="w-full rounded border border-edge bg-surface/40 p-3 space-y-1">
		<div class="flex items-center justify-between gap-2">
			<span class="text-sm text-warm font-medium break-words">{ consoleIssueScanCardIssue(card) }</span>
			if card.StageNumber > 0 {
				<span class="text-[10px] text-warm-muted font-mono whitespace-nowrap">{ fmt.Sprintf("%d/%d", card.StageNumber, card.StageCount) }</span>
			}
		</div>
		if consoleIssueScanCardTitle(card) != "" {
			<p class="text-xs text-warm-muted break-words">{ consoleIssueScanCardTitle(card) }</p>
		}
		<div class="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-warm-muted">
			<span>stage: { opsCivilizationValue(card.StageID, "—") }</span>
			<span>agents: { consoleIssueScanCardAgents(card) }</span>
		</div>
		if len(card.Blockers) > 0 {
			<p class="text-xs text-amber-300 break-words">blocked: { consoleIssueScanCardBlocker(card) }</p>
		}
		if consoleIssueScanCardReady(card) {
			<p class="text-xs text-emerald-300">ready for human — not merged</p>
		}
	</div>
}
```

Confirm `fmt` is imported in `console.templ`; if not, add it to the templ import block. (Check: `grep -n '"fmt"' graph/console.templ`. If absent, add `"fmt"`.)

- [ ] **Step 6: Register the routes in `handlers.go`**

In `Register` (after the `/console/kanban/order/{id}` line, ~396):

```go
	mux.Handle("GET /console/intake", h.writeWrap(h.handleConsoleIntake))
	mux.Handle("GET /console/intake/fragment", h.writeWrap(h.handleConsoleIntakeFragment))
```

In `RegisterReadOnlyConsole` (after the `/console/kanban/order/{id}` line, ~433):

```go
	mux.HandleFunc("GET /console/intake", h.handleConsoleIntake)
	mux.HandleFunc("GET /console/intake/fragment", h.handleConsoleIntakeFragment)
```

(The `/console/intake/card` route is added in Task 3.)

- [ ] **Step 7: Regenerate templ and run the build + tests**

Run: `templ generate && go build ./... && go test ./graph/ -run 'TestConsoleIntake|TestBuildConsoleIssueScan' -v`
Expected: `templ generate` succeeds; build succeeds; all tests PASS.

- [ ] **Step 8: Commit**

```bash
git add graph/console.go graph/console.templ graph/console_templ.go graph/handlers.go graph/console_intake_test.go
git commit -m "feat: render console intake issue-scan board under the intake tab"
```

---

## Task 3: Issue-scan run details drawer

Makes each card clickable, opening a drawer (mirroring the Plan-2b kanban drawer) with possession, lineage, evidence, authority boundary, and the target issue. Keyed by `run`+`stage` query params. Fails closed to an honest not-found.

**Files:**
- Modify: `graph/console.go` (add `handleConsoleIntakeCard`)
- Modify: `graph/console.templ` (make `consoleIssueScanCard` a button targeting the drawer; add `consoleIssueScanDrawer`)
- Modify: `graph/handlers.go` (route `/console/intake/card` in both registrars)
- Regenerate: `graph/console_templ.go`
- Test: `graph/console_intake_test.go` (append drawer tests)

**Interfaces:**
- Consumes: `opsCivilizationIssueScanKanban`, `fetchOpsCivilizationProjection`, `OpsCivilizationIssueScanKanbanCard`, helpers `opsCivilizationValue`, `opsCivilizationJoin`, `opsCivilizationEvidenceStatusValue`, `consoleIssueScanCardIssue` (Task 2).
- Produces: `func (h *Handlers) handleConsoleIntakeCard`; templ `consoleIssueScanDrawer(card OpsCivilizationIssueScanKanbanCard, found bool)`.

- [ ] **Step 1: Write the failing drawer tests**

Append to `graph/console_intake_test.go`:

```go
func TestConsoleIntakeCardDrawerRendersPossessionAndLineage(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hive/civilization/assembly-projection" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, hiveCivilizationAssemblyProjectionFixture)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	// run_docs_172 / run_adversarial_review is a parked stage with agent_reviewer in the fixture.
	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake/card?run=run_docs_172&stage=run_adversarial_review", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"agent_reviewer", "run_adversarial_review", "Run details"} {
		if !strings.Contains(body, want) {
			t.Errorf("drawer missing %q", want)
		}
	}
}

func TestConsoleIntakeCardDrawerUnknownIsHonestNotFound(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, hiveCivilizationAssemblyProjectionFixture)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake/card?run=nope&stage=nope", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not found") {
		t.Error("unknown card must render an honest not-found drawer")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./graph/ -run TestConsoleIntakeCardDrawer -v`
Expected: FAIL — route unregistered / `undefined: handleConsoleIntakeCard`.

- [ ] **Step 3: Add the drawer handler**

In `graph/console.go`:

```go
func (h *Handlers) handleConsoleIntakeCard(w http.ResponseWriter, r *http.Request) {
	run := strings.TrimSpace(r.URL.Query().Get("run"))
	stage := strings.TrimSpace(r.URL.Query().Get("stage"))
	board := opsCivilizationIssueScanKanban(fetchOpsCivilizationProjection(r))
	for _, col := range board.Columns {
		for _, card := range col.Cards {
			if card.RunID == run && card.StageID == stage {
				consoleIssueScanDrawer(card, true).Render(r.Context(), w)
				return
			}
		}
	}
	// Not found or upstream error: honest empty drawer, never a fabricated run.
	consoleIssueScanDrawer(OpsCivilizationIssueScanKanbanCard{RunID: run, StageID: stage}, false).Render(r.Context(), w)
}
```

- [ ] **Step 4: Make the card clickable and add the drawer templ**

In `graph/console.templ`, replace the `consoleIssueScanCard` outer `<div>` with a `<button>` that targets the drawer (keep the inner content identical). The `hx-get` value is built from the run/stage IDs, which are sanitized alphanumeric tokens; `hx-*` is an HTML attribute so templ's attribute escaping is the correct protection (same precedent as `consoleOrderCard`):

```templ
templ consoleIssueScanCard(card OpsCivilizationIssueScanKanbanCard) {
	<button
		type="button"
		class="w-full text-left rounded border border-edge bg-surface/40 p-3 space-y-1 hover:border-brand transition-colors"
		hx-get={ "/console/intake/card?run=" + card.RunID + "&stage=" + card.StageID }
		hx-target="#console-intake-drawer"
		hx-swap="innerHTML"
	>
		<div class="flex items-center justify-between gap-2">
			<span class="text-sm text-warm font-medium break-words">{ consoleIssueScanCardIssue(card) }</span>
			if card.StageNumber > 0 {
				<span class="text-[10px] text-warm-muted font-mono whitespace-nowrap">{ fmt.Sprintf("%d/%d", card.StageNumber, card.StageCount) }</span>
			}
		</div>
		if consoleIssueScanCardTitle(card) != "" {
			<p class="text-xs text-warm-muted break-words">{ consoleIssueScanCardTitle(card) }</p>
		}
		<div class="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-warm-muted">
			<span>stage: { opsCivilizationValue(card.StageID, "—") }</span>
			<span>agents: { consoleIssueScanCardAgents(card) }</span>
		</div>
		if len(card.Blockers) > 0 {
			<p class="text-xs text-amber-300 break-words">blocked: { consoleIssueScanCardBlocker(card) }</p>
		}
		if consoleIssueScanCardReady(card) {
			<p class="text-xs text-emerald-300">ready for human — not merged</p>
		}
	</button>
}

templ consoleIssueScanDrawer(card OpsCivilizationIssueScanKanbanCard, found bool) {
	<aside class="rounded border border-edge bg-surface/60 p-4 space-y-3" role="dialog" aria-label="Issue-scan run details">
		<div class="flex items-center justify-between gap-2">
			<h3 class="text-sm font-medium text-warm">Run details</h3>
			<button type="button" class="text-xs text-warm-muted hover:text-warm" onclick="document.getElementById('console-intake-drawer').innerHTML='';return false;">close</button>
		</div>
		if !found {
			<p class="text-sm text-warm-muted">Run { opsCivilizationValue(card.RunID, "—") } stage { opsCivilizationValue(card.StageID, "—") } is unavailable (not found).</p>
		} else {
			<dl class="grid grid-cols-[9rem,1fr] gap-y-1 text-xs">
				<dt class="text-warm-muted">Issue</dt><dd class="text-warm break-words">{ consoleIssueScanCardIssue(card) }</dd>
				<dt class="text-warm-muted">Stage</dt><dd class="break-all">{ opsCivilizationValue(card.StageID, "—") }</dd>
				<dt class="text-warm-muted">State</dt><dd>{ opsCivilizationValue(card.CurrentState, "—") }</dd>
				<dt class="text-warm-muted">Assigned</dt><dd class="break-all">{ opsCivilizationJoin(card.AssignedAgentIDs, "none projected") }</dd>
				<dt class="text-warm-muted">Touching</dt><dd class="break-all">{ opsCivilizationJoin(card.TouchingAgentIDs, "none projected") }</dd>
				<dt class="text-warm-muted">Authority</dt><dd class="break-words">{ opsCivilizationValue(card.AuthorityBoundary, "not projected") }</dd>
				<dt class="text-warm-muted">Run</dt><dd class="break-all">{ opsCivilizationValue(card.RunID, "—") }</dd>
				<dt class="text-warm-muted">Canonical task</dt><dd class="break-all">{ opsCivilizationValue(card.CanonicalTaskID, opsCivilizationValue(card.TaskID, "not projected")) }</dd>
			</dl>
			if len(card.Blockers) > 0 {
				<div class="border-t border-edge pt-3 space-y-2">
					<p class="text-[10px] text-warm-faint uppercase tracking-widest">blockers</p>
					for _, b := range card.Blockers {
						<div>
							<p class="text-xs text-warm break-words">{ opsCivilizationEvidenceStatusValue(b.BlockerType, "blocker") }</p>
							<p class="text-[10px] text-warm-muted break-words">{ opsCivilizationValue(b.RequiredAction, "action not projected") }</p>
						</div>
					}
				</div>
			}
			if card.HasLineage {
				<div class="border-t border-edge pt-3">
					<p class="text-[10px] text-warm-faint uppercase tracking-widest">lineage</p>
					<p class="text-[10px] text-warm-muted break-all mt-1">primary { opsCivilizationValue(card.Lineage.PrimaryTaskID, "not projected") }</p>
					<p class="text-[10px] text-warm-faint break-all mt-1">duplicates { opsCivilizationJoin(card.Lineage.DuplicateTaskIDs, "none") }</p>
				</div>
			}
			if len(card.EvidenceRefs) > 0 {
				<div class="border-t border-edge pt-3">
					<p class="text-[10px] text-warm-faint uppercase tracking-widest">evidence</p>
					<p class="text-[10px] text-warm-muted break-all mt-1">{ opsCivilizationJoin(card.EvidenceRefs, "none projected") }</p>
				</div>
			}
		}
	</aside>
}
```

- [ ] **Step 5: Register the drawer route in `handlers.go`**

In `Register`:

```go
	mux.Handle("GET /console/intake/card", h.writeWrap(h.handleConsoleIntakeCard))
```

In `RegisterReadOnlyConsole`:

```go
	mux.HandleFunc("GET /console/intake/card", h.handleConsoleIntakeCard)
```

- [ ] **Step 6: Regenerate templ and run the intake tests**

Run: `templ generate && go build ./... && go test ./graph/ -run 'TestConsoleIntake|TestBuildConsoleIssueScan' -v`
Expected: build succeeds; all intake tests PASS (board + drawer). If `TestConsoleIntakeCardDrawerRendersPossessionAndLineage` fails on the exact `run`/`stage` tokens, confirm the fixture's run/stage IDs with `grep -n '"run_id"\|"stage_id"' graph/handlers_test.go` in the `issue_scan_projection` block and adjust the query params to a real (run, stage) pair; do not change production code to match a wrong fixture assumption.

- [ ] **Step 7: Commit**

```bash
git add graph/console.go graph/console.templ graph/console_templ.go graph/handlers.go graph/console_intake_test.go
git commit -m "feat: add console intake run details drawer with possession and lineage"
```

---

## Task 4: Retire the legacy /ops/civilization issue-scan sections

Remove the three issue-scan *render* sections from `/ops/civilization`; add a pointer to `/console/intake`. Keep the builder, the type, the handler's data population, and the observation/canary consumer. Reconcile the existing `/ops/civilization` test and add a retirement regression test.

**Files:**
- Modify: `graph/civilization.templ` (remove `#issue-scan-kanban` section; remove the `if data.QueuedRunRequest != nil { … }` section, which contains the stage-evidence table; add a pointer)
- Regenerate: `graph/civilization_templ.go`
- Modify: `graph/handlers_test.go` (reconcile `TestHandleOpsCivilizationConsumesHiveProjection`; add `TestHandleOpsCivilizationRetiresIssueScanBoard`)

**Interfaces:**
- Consumes: nothing new. Must NOT touch `graph/civilization_issue_scan.go`, `opsObservationIssueScanCards`, `opsCivilizationIssueScanKanban`, or `handleOpsCivilization`'s data population.
- Produces: no new exports.

- [ ] **Step 1: Write the retirement regression test first**

Append to `graph/handlers_test.go`:

```go
func TestHandleOpsCivilizationRetiresIssueScanBoard(t *testing.T) {
	h, _, _ := testHandlers(t)
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, hiveCivilizationAssemblyProjectionFixture)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)
	t.Setenv("HIVE_OPS_API_KEY", "secret")

	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/civilization", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	// Retired: the three issue-scan render sections are gone.
	for _, gone := range []string{
		`data-civilization-issue-scan-kanban="read-only"`,
		`data-civilization-wide-table="issue-scan-kanban"`,
		`data-civilization-wide-table="issue-scan-stage-evidence"`,
		"Queued issue-scan lifecycle",
	} {
		if strings.Contains(body, gone) {
			t.Errorf("/ops/civilization still renders retired section marker %q", gone)
		}
	}
	// Added: an explicit pointer to the console surface.
	if !strings.Contains(body, "/console/intake") {
		t.Error("/ops/civilization must point issue-scan to /console/intake")
	}
}

func TestObservationCanaryStillBuildsAfterRetirement(t *testing.T) {
	// Regression guard: the shared builder must remain wired to the canary path.
	board := opsCivilizationIssueScanKanban(&OpsCivilizationAssemblyProjection{
		IssueScanProjection: OpsCivilizationIssueScanProjection{
			Runs: []OpsCivilizationIssueScanRunProjected{{
				RunID:       "run_c",
				TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 300},
			}},
		},
	})
	if cards := opsObservationIssueScanCards(board); len(cards) == 0 {
		t.Fatal("observation/canary consumer must still build cards from the retained builder")
	}
}
```

- [ ] **Step 2: Run the new regression test to verify it fails**

Run: `go test ./graph/ -run 'TestHandleOpsCivilizationRetiresIssueScanBoard|TestObservationCanaryStillBuildsAfterRetirement' -v`
Expected: `TestHandleOpsCivilizationRetiresIssueScanBoard` FAILS (sections still present, pointer absent). `TestObservationCanaryStillBuildsAfterRetirement` PASSES (builder still wired — this is the guard that must keep passing).

- [ ] **Step 3: Remove the `#issue-scan-kanban` section and add the pointer**

In `graph/civilization.templ`, delete the entire `<section id="issue-scan-kanban" …> … </section>` block (opens at the line containing `id="issue-scan-kanban"`, ~221; closes at its matching `</section>`, ~335) and replace it with:

```templ
		<section class="border border-edge bg-surface rounded-lg p-4" data-civilization-issue-scan-moved="console">
			<h2 class="text-sm font-medium text-warm">Issue-scan</h2>
			<p class="text-xs text-warm-muted mt-1">
				Issue-scan moved to Mission Control →
				<a href="/console/intake" class="text-brand hover:underline">/console/intake</a>.
			</p>
		</section>
```

- [ ] **Step 4: Remove the queued issue-scan lifecycle + stage-evidence section**

In `graph/civilization.templ`, delete the entire `if data.QueuedRunRequest != nil { <section …> … </section> }` block (opens ~517, closes ~574). This removes both the "Queued issue-scan lifecycle" section and the nested `issue-scan-stage-evidence` table. Do not add a second pointer — the pointer from Step 3 covers the retired surface. Leave the surrounding sections (`ReferenceGroups`, `Role topology`) untouched.

- [ ] **Step 5: Regenerate templ and rebuild**

Run: `templ generate && go build ./... && go vet ./...`
Expected: success. (Unused templ helper `opsHiveQueuedRunLifecycle` and unused struct fields do not break `go build`/`go vet`. If `go vet` is clean, proceed.)

- [ ] **Step 6: Reconcile the existing `/ops/civilization` assertion list**

The large `want` slice in `TestHandleOpsCivilizationConsumesHiveProjection` (~line 600–724) asserts many tokens; some are produced *only* by the now-retired sections and will now be absent, while others are produced by retained sections (Issue readiness, Issue intake projection, factory orders, work tasks/artifacts, role topology) and must remain asserted.

Do this empirically — do not guess:

1. Run `go test ./graph/ -run TestHandleOpsCivilizationConsumesHiveProjection -v` and capture the failure output (it names the missing tokens).
2. For each token the test now reports missing, remove that string from the `want` slice ONLY. Definite removals (retired-section markers): `"Issue-scan Kanban"`, `` `data-civilization-issue-scan-kanban="read-only"` ``, `` `data-civilization-wide-table="issue-scan-kanban"` ``, `` `data-civilization-wide-table="issue-scan-stage-evidence"` ``, `"Queued issue-scan lifecycle"`, and the `w-full min-w-[…]` width markers unique to the removed wide-tables if they are now absent.
3. A token still present in the body (produced by a retained section) will NOT be reported missing — leave it. This is the discriminator: presence in the rendered body, verified by re-running, not by inspection.
4. Re-run until `TestHandleOpsCivilizationConsumesHiveProjection` passes. Do not add tokens; only remove the ones the retirement genuinely dropped.
5. If the same test also has an *absence* assertion block for issue-scan (search the test body for `strings.Contains(body, "issue-scan"` negations), leave the retirement-specific absence assertions to `TestHandleOpsCivilizationRetiresIssueScanBoard` (Step 1) and keep this test focused on retained content.

- [ ] **Step 7: Full graph test run**

Run: `go test ./graph/...`
Expected: PASS, including `TestHandleOpsCivilizationRetiresIssueScanBoard`, `TestObservationCanaryStillBuildsAfterRetirement`, and the reconciled `TestHandleOpsCivilizationConsumesHiveProjection`.

- [ ] **Step 8: Commit**

```bash
git add graph/civilization.templ graph/civilization_templ.go graph/handlers_test.go
git commit -m "refactor: retire /ops/civilization issue-scan board in favor of /console/intake"
```

---

## Task 5: Full verification, visual evidence, and SDD ledger

**Files:**
- Modify: `.superpowers/sdd/progress.md` (append Build-1 ledger entry)
- Create: desktop + mobile screenshots of `/console/intake` (per MFOF-001)

- [ ] **Step 1: Full verify from `repos/site`**

Run: `templ generate && go build ./... && go vet ./... && go test ./graph/...`
Expected: `templ generate` produces no diff beyond the already-committed generated files (run `git status` — `console_templ.go` / `civilization_templ.go` should be clean if committed in prior tasks); build, vet, and all graph tests PASS. If `templ generate` produces an uncommitted diff, commit it (`git add -A graph/*_templ.go && git commit -m "chore: regenerate templ output"`).

- [ ] **Step 2: Capture desktop + mobile screenshots of the Intake board**

Use the project's run/screenshot path (see `/run` skill or `make dev`). Render `/console/intake` against a fixture or live projection at desktop (~1280px) and mobile (~390px) widths. Save under `.visual-evidence/` (the repo already uses this dir). Confirm the board shows honest columns, issue-leading cards, a working agent, a blocker, and the ready/no-merge affordance; and that an unavailable projection shows the explicit unavailable state (no fabricated cards).

- [ ] **Step 3: Update the SDD ledger**

Append a Build-1 entry to `.superpowers/sdd/progress.md` recording: the five tasks, the files touched, the verify command output summary, and the retirement scope (sections removed + pointer, builder kept). Reference this plan path.

- [ ] **Step 4: Commit**

```bash
git add .superpowers/sdd/progress.md .visual-evidence
git commit -m "docs: record mission control intake build 1 verification and evidence"
```

---

## Self-Review

**Spec coverage** (against `docs/superpowers/specs/2026-06-30-mission-control-intake-design.md` §"Build 1"):
- Data: reuse `opsCivilizationIssueScanKanban` builder + `fetchOpsCivilizationProjection` → Task 1. ✓
- Freshness from `GeneratedAt` via `deriveFreshness`, down/absent → `unavailable` → Task 1 (with failed-status + zero-timestamp fail-closed guards). ✓
- View: lifecycle board, columns by state, cards lead with issue + stage + working agents + blocker + "not merged" → Task 2. ✓
- Details drawer (lineage/evidence/authority-boundary/possession/target issue) → Task 3. ✓
- Honest-staleness + no-fabrication throughout → Tasks 1–3 tests. ✓
- Retirement: remove 3 sections from `civilization.templ` + pointer; keep builder + type (shared with canary) → Task 4. ✓
- Wiring: enable Intake tab; `ConsolePageData.IssueScan`; routes in both `Register` and `RegisterReadOnlyConsole` → Tasks 1–3. ✓
- Tests mirror `console_kanban_test.go`; retirement regression guard; canary still builds; desktop + mobile screenshots → Tasks 2–5. ✓

**Placeholder scan:** No TBD/TODO/"handle edge cases"/"similar to Task N". All code shown in full. ✓

**Type consistency:** `ConsoleIssueScan`, `buildConsoleIssueScan`, `ConsolePageData.IssueScan`, `handleConsoleIntake`/`handleConsoleIntakeFragment`/`handleConsoleIntakeCard`, `consoleIssueScanFragment`/`consoleIssueScan`/`consoleIssueScanCard`/`consoleIssueScanDrawer`, and the `consoleIssueScanCard*` helper names are used identically across tasks. The reused builder entry `opsCivilizationIssueScanKanban(projection)` and the `OpsCivilizationIssueScanKanbanCard` field names (`RunID`, `StageID`, `StageNumber`, `StageCount`, `CurrentState`, `TargetIssue`, `SelectedIssue`, `AssignedAgentIDs`, `TouchingAgentIDs`, `Blockers`, `Lineage`, `HasLineage`, `EvidenceRefs`, `AuthorityBoundary`, `CanonicalTaskID`, `TaskID`) match `graph/civilization_issue_scan.go`. ✓
