# Mission Control — Intake Design (two channels)

> doc_id: SITE-MISSION-CONTROL-INTAKE-001
> realizes: the **Intake** surface (need #3) of SITE-MISSION-CONTROL-DESIGN-001
> authority: design-only. No runtime, governed write, EventGraph write, or autonomy increase is authorized here. Both builds below are **read-only** on the console (zero writes).
> status: **DRAFT — Build-1 approved for planning (2026-06-30). Adds an optional-gate control model (§Optional adversarial gates) as a separate, backend-touching build ("Build G") pending owner ✓ on two decisions + authorization.**

## Purpose

Intake is the operator's view of **what is entering the factory**, in two channels:

- **Channel A — human request.** A free-form/structured requirement is iterated (via a **managed structured flow** now; interactive AI assist later) into a well-formed factory-order draft. **Stops at a ready-to-submit draft** — the governed submit that seeds the order is deferred (separately authorized).
- **Channel B — factory issue-scan.** The factory autonomously scans a list of repos, takes possession of GitHub issues, and drives each through its lifecycle. The console **surfaces** that (state, owner, working agents, blockers) — read-only.

**Build order (owner-decided):** **Build 1 = Channel B** (fast: UI-only over a ready backend), **Build 2 = Channel A** (the compose wizard). Both live under the **Intake** tab.

## Backend readiness verdict (read-only assessment, 2026-06-30)

The issue-scan backend is **built end-to-end**; the console gap is **UI-only**. Evidence:

- **Autonomy + repo list (WIRED).** `hive civilization daemon --issue-scan-interval <d> --issue-scan-repo … | --issue-scan-registry` autonomously polls GitHub across a repo list and queues the top-ranked `cc:pr-ready` issue as a factory run. Guardrails present: one-active guard, hard duration cap, kill-switch file, max-new-runs, per-stage `AuthorityLevelRequired`.
- **Lifecycle + possession (WIRED).** 7-stage pipeline (research/design → CFADA=`debate_with_correct_civic_roles` → design-select → implement → CFAR=`run_adversarial_review` → drive-blockers → surface-ready-PR), possession/stage/working-agents recorded as **real EventGraph events**, no-merge enforced at the finalizer (`HumanApprovalRequired=true`, `NoMergeOrDeployClaim=true`).
- **Read path (WIRED).** `GET /api/hive/civilization/assembly-projection` emits runs/stages/blockers/lineage with per-stage `CurrentState`, `AssignedAgentIDs`/`TouchingAgentIDs`, target issue, and evidence — from real events. Site already fetches (`fetchOpsCivilizationProjection`) and **renders an issue-scan Kanban** on the legacy `/ops/civilization` page via the `civilization_issue_scan.go` builder.

**Calibration caveats (recorded, not blocking):**
- "Autonomous" = the *scan + issue selection* is autonomous on a timer; driving a run through the stages to a ready PR *fully unattended* depends on the stage-runner flags being wired and the approval config (per-stage authority is Required by default; scanner mode can't combine with blanket auto-approve). The console **surfaces state**; it does not change this.
- **One issue per run** by design (one-active guard) — not batch.
- **No `IAR` stage** exists in code. Owner to clarify what IAR denotes so states are labeled correctly. *(open)*

## Build 1 — Channel B: console issue-scan surface (+ retire the /ops board)

**Zero backend changes. Reuses the existing data builder. Read-only.**

### Data
- Fetch: `fetchOpsCivilizationProjection(r)` (already exists) → `OpsCivilizationAssemblyProjection`.
- View-model: **reuse the existing `OpsCivilizationIssueScanKanban` builder** in `graph/civilization_issue_scan.go` (~631 LOC, already produces columns-by-state + per-stage cards with assigned/touching agents, blockers, lineage, target issue, evidence). Do **not** rewrite it.
- Freshness: derive honest staleness from the projection's `GeneratedAt` (reuse the console `deriveFreshness` machine); a down/absent projection → explicit `unavailable`, never a comforting default.

### View (under the Intake tab)
- A **lifecycle board** of issue-scan runs: columns by run/stage state (queued · dispatched · running · blocked · parked · human_action · ready_for_human · superseded · completed · projection_only).
- **Cards lead with the issue** (`repo#number` + title) and carry: current **stage**, **working agent(s)** (`AssignedAgentIDs`/`TouchingAgentIDs`), **blocker** (type + required action) when present, and the ready/no-merge state made explicit ("ready for human — not merged").
- **Details drawer** (reuse the Plan-2b drawer pattern): stage lineage, evidence refs, authority boundary, target issue link. Possession facts (who took it, which agents) front-and-center — that's the operator's question for Channel B.
- Honest-staleness + no-fabrication throughout (a parked/blocked run *looks* blocked; a missing field renders as explicit "unavailable", never invented).

### Retirement of the legacy /ops board (owner: "replace as part of this build")
Precise, minimal surgery — **retire the board, keep the builder**:
- **Remove** the visual issue-scan sections from `graph/civilization.templ` on the `/ops/civilization` page: `#issue-scan-kanban` (line ~221), the "Queued issue-scan lifecycle" section (~521), and the issue-scan stage-evidence table (~544). Replace with a short pointer: *"Issue-scan moved to Mission Control → /console/intake."*
- **Keep** `OpsCivilizationIssueScanKanban` + the `civilization_issue_scan.go` builder — it is **shared** with the observation/canary surface (`ops.go:2246-2460`) and now the console. Removing only the rendering, not the data, avoids breaking the canary.
- `handleOpsCivilization` keeps its other sections (boundary, status, factory orders, issue readiness) — those aren't superseded yet (D1: retire `/ops/*` view-by-view).

### Wiring
- Enable the Intake tab: `consoleTab("intake", …, true)` in `console.templ`.
- `ConsolePageData` gains `IssueScan *ConsoleIssueScan` (or reuse the ops kanban type directly).
- Routes `GET /console/intake` (+ `/console/intake/fragment` for HTMX refresh) in **both** `Register` (via `writeWrap`) and `RegisterReadOnlyConsole`; handler mirrors `handleConsoleKanban` (fetch → build → render).

### Tests (mirror console_kanban_test.go)
- Handler/render: `GET /console/intake` with a mocked `assembly-projection` renders the scan board — asserts a card shows the issue ref, stage, a working agent, and (when present) a blocker; the ready state renders "not merged".
- Honest-staleness: projection error/absent → `unavailable` state, zero fabricated cards.
- Retirement: `/ops/civilization` no longer renders the issue-scan section and shows the pointer; the canary/observation path (`opsObservationIssueScanCards`) still builds (builder retained) — a regression guard.
- Per MFOF-001: desktop + mobile screenshots of the console Intake board.

## Build 2 — Channel A: human-request compose wizard (summary; later)

Carries the earlier decisions: a **managed structured flow** (guided, validated wizard → title, definition-of-done, acceptance criteria, expected outputs, risk, repo), **stop at a ready-to-submit draft** (no write), verbatim request preserved, requestor = current operator, **AI-assist deferred** (no free-text structurer exists today — `work.BuildFactoryOrderDevelopmentProposal` is a *proof-of-work assembler of structured input*, not a free-text structurer; confirmed in the assessment). Both the AI-assist and the governed submit render as explicit deferred seams. Detailed in its own plan when Build 1 lands.

## Open items for owner
1. **Confirm the Build-1 retirement scope:** remove the three visual issue-scan sections from `/ops/civilization` + pointer, **keep** the shared builder. (Recommended — minimal, non-breaking.)
2. **IAR** — what does it denote in your lifecycle? (affects state labels on the board.)
3. Anything you want on the **drawer** beyond lineage/evidence/authority-boundary/possession for Build 1.

## Optional adversarial gates (control surface + backend-wiring proposal)

Owner intent: the four adversarial gates should be **optional per workflow**, so prototype work isn't charged the full production build time. The canonical set is a **2×2** — a design phase and a review phase, each an *internal* pass feeding a *cross-family* pass:

| Gate | Phase | Meaning | In backend today? |
|------|-------|---------|-------------------|
| **IADA** — Internal Adversarial Design Assessment | design | model self-assesses the design | **ABSENT** → owner asked for a **no-op placeholder** |
| **CFADA** — Cross-Family Adversarial Design Assessment | design | cross-family design debate | **WIRED** = `debate_with_correct_civic_roles` (stage 2/7) |
| **IAR** — Internal Adversarial Review | review | model self-reviews, drives *self*-blockers to 0, before CFAR | **ABSENT** (open: real stage vs placeholder) |
| **CFAR** — Cross-Family Adversarial Review | review | cross-family review | **WIRED** = `run_adversarial_review` (stage 5/7) |

The pipeline is a **fixed 7-stage hardcoded slice** (`hive/pkg/hive/issue_intake.go:625-712`) with **no notion of optional stages** and **no gate policy** today. Making gates optional is therefore a **backend change** (hive, + projection), not UI-only — it is its own build.

### Fail-safe posture (non-negotiable — corrects the naive design)

A quick proposal modeled `ApplyIADA/ApplyCFADA/… bool` "disabled by default for backwards compatibility." That is **fail-open**: Go bools zero to `false`, so an absent/empty policy would **skip every gate**. Inverted per the owner's fail-safe standard — *the permissive outcome is the explicitly-proven branch; the default and every ambiguity deny*:

- **Default = all gates ON (production).** The zero value / absent / unparseable policy runs **every** gate. Skipping is never the default.
- **Skipping is an allowlist, gated twice.** A gate is skipped only when the policy is **explicitly `mode: prototype`** *and* the gate is named in an explicit `skip_gates` allowlist *and* the stage is one of the four **optional gate stages** (core stages — research, design-select, implement, drive-blockers, ready-PR — are **never** skippable). Any other value, unknown gate name, or read error → **run the gate**.
- **Honest surfacing.** The projection emits the resolved per-gate disposition (`ran` / `skipped-prototype`); the console shows a skipped gate **explicitly** ("CFADA — skipped (prototype)"), never hidden. A skipped gate must *look* skipped.

```go
// GatePolicy is fail-safe by construction: the zero value runs EVERY gate.
type GatePolicy struct {
    Mode      string   `json:"mode,omitempty"`       // ""/"production" => all gates; only "prototype" permits skips
    SkipGates []string `json:"skip_gates,omitempty"` // allowlist of gate stage IDs; honored ONLY when Mode=="prototype"
    Rationale string   `json:"rationale,omitempty"`
    SetBy     string   `json:"set_by,omitempty"`     // operator identity (provenance)
}
// gateSkipped fails closed: false unless everything below is explicitly proven.
func gateSkipped(p GatePolicy, stageID string) bool {
    if !isOptionalGateStage(stageID) { return false }                 // core stages never skip
    if strings.ToLower(strings.TrimSpace(p.Mode)) != "prototype" { return false } // default/unknown => run
    for _, g := range p.SkipGates { if g == stageID { return true } } // explicit allowlist only
    return false                                                       // fall-through denies the skip
}
```

### Backend wiring (grounded proposal)

1. **Attach point — per-order policy.** Add `GatePolicy` to `FactoryRunRequestedContent` (`hive/pkg/hive/events.go:176-191`), beside the existing `Authority` / `Budget` / `ModelOverrides` policy envelopes (same precedent). Additive/optional field.
2. **Hook point — one choke.** `selectIssueScanStageAdvanceTarget` (`hive/pkg/hive/issue_scan_stage_advance.go:155-196`) is the single place "next stage" is chosen; insert the fail-closed `gateSkipped(...)` check in its loop (~line 187) so a skipped gate stage is passed over. All 8 advance callers funnel through here.
3. **Placeholder stages.** Add **IADA** (and, per decision, **IAR**) to `issueScanDevelopmentLifecycle()` as **no-op placeholder stages** — a runner that records `*_placeholder_noop` evidence and auto-passes — so they exist in the model/UI now and can be filled with a real runner later. IADA before CFADA; IAR before CFAR.
4. **Projection exposure.** Add the `GatePolicy` + resolved per-gate disposition to `CivilizationIssueScanRunProjected` / `CivilizationIssueScanStageProjected` (`civilization_assembly_projection.go:291-318`) so the read-only console can show gate state honestly.

### UI / control surface

- **Where the flags live: at intake, on the order (both channels).** For **Channel A** (compose wizard, Build 2) the flags are part of the draft — a **production/prototype preset** plus per-gate toggles — read-only until the (deferred, governed) submit carries them. Setting review rigor is part of defining the order. For **Channel B**, a scan run inherits a default policy (per-repo or global); the console **shows** each run's policy + resolved dispositions (read); changing an in-flight run's policy is a governed action (deferred).
- A skipped gate renders as an explicit muted "skipped (prototype)" chip on the board/drawer — honest-staleness applied to process, not just data.

### Two decisions this needs (owner)
- **Prototype-preset composition:** what does "prototype" actually skip? (e.g. keep **CFAR** as the irreducible minimum bar and make {IADA, CFADA, IAR} optional — my recommendation — vs. allow skipping CFAR too.)
- **IAR:** build it as a **real stage** now (internal self-review → drive self-blockers to zero, before CFAR — a backend build), or a **no-op placeholder** like IADA for now?

## Build sequencing

1. **Build 1 — Channel B read-only surface + /ops retirement.** Approved; UI-only; **no dependency on the gate work.** Proceeds to plan now. (It shows the stages a run *has* run; it cannot show gate *policy* until the projection carries it — that arrives with Build G.)
2. **Build G — optional-gate control.** Backend (hive: GatePolicy field + fail-closed skip hook + IADA/IAR placeholder stages + projection exposure) **and** UI (intake flags). Multi-repo, **changes factory behavior**, so it needs its own explicit authorization + a fail-closed review pass. Design/proposal above is ready; sequenced after Build 1.
3. **Build 2 — Channel A compose wizard** (managed structured flow, stop-at-draft). Naturally carries the Build-G gate flags once they exist.

## Precedent & evidence index
- `SITE-MISSION-CONTROL-DESIGN-001` — parent console design (this realizes + refines its Intake surface).
- Backend: `hive/cmd/hive/factory_issue_scan_scanner.go`, `hive/pkg/hive/issue_intake.go`, `issue_scan_*` (7-stage lifecycle), `civilization_assembly_projection.go`; `GET /api/hive/civilization/assembly-projection`.
- Site read path (to port/retire): `graph/civilization_issue_scan.go` (builder, reuse), `graph/civilization.templ` §issue-scan (retire), `fetchOpsCivilizationProjection` (ops.go).
- Merged console foundation: `graph/console.go`, `console.templ`, `handlers.go` (Plan 1 #198, Plan 2 #199).
