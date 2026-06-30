# Mission Control — Intake (Phase 1) Design

> doc_id: SITE-MISSION-CONTROL-INTAKE-001
> realizes: the **Intake** surface (need #3) of SITE-MISSION-CONTROL-DESIGN-001
> authority: design-only. No runtime, governed write, EventGraph write, or autonomy increase is authorized by this document. Phase-1 Intake performs **zero writes**.
> status: **DRAFT for owner review** (authored solo by Claude while owner away; the key decisions below need an owner ✓ before a plan is written).

## Purpose

Build the **Intake** view at `/console/intake` on the merged Plan-1/Plan-2 console foundation: a human pastes a free-text request and reviews it as an editable, structured **factory-order draft** (title, risk, target repo, cell, definition-of-done checklist, acceptance criteria, expected-outputs inventory). The human is the requestor/owner throughout. In Phase 1 there is **no submit** — the governed write that seeds the order is deferred to Phase 2.

## The load-bearing finding (please read first)

The parent design doc (D4 / §Intake) names the draft data source as `work.BuildFactoryOrderDevelopmentProposal` and describes "an agent (e.g. Strategist) structures the request into a FactoryOrder draft." Grounding against the code shows this **is not** a free-text → structure step:

- `work.BuildFactoryOrderDevelopmentProposal` (`work/factory_order_proposal.go:186`) is a **pure assembler of already-structured input**. It *requires* non-empty `ChangedFileIntent`, `ValidationPlan`, and `AuthorityBoundary`, plus pre-minted `fo_/req_/ac_/tsk_` IDs, and returns a proof-of-work / audit proposal (`ProofOfWorkPacket`, `AuditReport`, changed-file intent, authority boundaries). It is a *downstream* "assemble the proof-of-work proposal for an already-designed change" tool — **not** an intake-triage structurer. It cannot take a paragraph of free text and propose a title + DoD + acceptance criteria.
- **No work-server HTTP endpoint** exposes any free-text → draft structuring (the only task endpoints are CRUD on `/tasks`). So **Intake has no live upstream to consume** — unlike Health (operator-projection) and Kanban (`/tasks`).
- The shape an editable order draft actually maps to is the simpler **`FactoryOrder` seed DTO** (`work/factory_order.go:39`): `Title, Intent, Cell, RiskClass, DefinitionOfDone, AcceptanceCriteria, TestPlan, ExpectedOutputs`. But its consumer `SeedFactoryOrder` **writes** (creates the seed task + readiness gates) — that is the Phase-2 governed submit.

**Consequence:** an honest Phase-1 Intake **cannot show an AI-structured draft**, because no function or endpoint structures free text today. Fabricating one would violate the console's no-fabrication contract. Phase-1 Intake is therefore a **manual structured compose + live draft preview**, with the AI-assist *and* the submit both rendered as explicit, deferred seams.

This is the central thing for the owner to confirm or redirect (see Open Questions).

## Decisions (made solo; rationale given — owner may override)

| # | Decision | Rationale |
|---|----------|-----------|
| DI-1 | **Phase-1 Intake = manual structured compose + draft preview. No AI structuring; no submit.** The human authors the structured fields; the system renders a live factory-order draft preview. | The only honest option given the finding above — there is no free-text structurer to call, and faking one breaks no-fabrication. |
| DI-2 | **Draft model = the `FactoryOrder` seed shape** (site-side view-model mirroring `Title/RequestText(verbatim)/RiskClass/TargetRepo/Cell/DefinitionOfDone[]/AcceptanceCriteria[]/ExpectedOutputs[]`), **not** `BuildFactoryOrderDevelopmentProposal`. Site does **not** import the `work` package. | `FactoryOrder` is the intake-draft shape; the development-proposal is a downstream proof artifact. Keeping the view-model site-local preserves the read-only-consumer boundary and avoids embedding `work` domain logic in Site. |
| DI-3 | **Requestor = the current operator** (`auth.User.Name` via `viewUser`), shown as owner; provenance line reads "structured by: **manual** — AI assist deferred." | D4: the human stays requestor/owner; the agent is credited subordinately. With no agent involved in Phase 1, provenance says so honestly. |
| DI-4 | **Two-step HTMX flow, no persistence.** `GET /console/intake` renders Step-1 (free-text + optional title). An `hx-post` to `/console/intake/draft` returns the Step-2 editable review fragment (verbatim request + declared defaults + empty editable checklists + live preview + the two deferred seams). The draft is **not** saved server-side. | The compose→review→(deferred) UX is the vertical-slice proof. A POST that returns a *computed scaffold* is a pure read-shaped op (no EventGraph record, no state mutation), so it stays inside the read-only charter. No write ⇒ no persistence in Phase 1. |
| DI-5 | **Both writes are explicit deferred seams, fail-safe framed.** "Approve & submit order" is a **disabled** control with the statement *"nothing runs; submit is a Phase-2 governed action."* "Structure with AI assist" is a **disabled** affordance labeled *"deferred — no structurer available yet."* | Fail-safe by default: the permissive outcome (an order entering the factory) is never reachable in Phase 1; the default is to show, not act. |
| DI-6 | **No fabrication in the scaffold.** The verbatim request is preserved exactly; defaults (risk `low`, repo `transpara-ai/work`, cell `implementation`) are clearly labeled as editable defaults; DoD / acceptance / expected-outputs start as **empty editable lists** — never invented content. | The scaffold helps the human compose without asserting facts it doesn't have. |
| DI-7 | **Enable the Intake nav tab** (`consoleTab("intake", …, true)`); add `Intake *ConsoleIntakeDraft` to `ConsolePageData`; mirror the Health/Kanban handler + route (both `Register` and `RegisterReadOnlyConsole`) + templ pattern. | Match surrounding code; one coherent console idiom. |

## Approaches considered

- **A — Manual structured compose + preview (RECOMMENDED, DI-1).** Site-only, no LLM, no `work` import, no write. Ships the real intake UX and the factory-order draft shape; leaves two clean, explicit Phase-2 seams (AI-assist, submit). Honest by construction. *Cost:* the draft is not AI-assisted or saveable in Phase 1 — it is a compose-and-preview tool, the thinnest of the four Phase-1 views.
- **B — Add a work-server "build draft" endpoint + site consumer (two-repo, like 2a/2b).** Would require a **new** free-text → structure function in `work` (heuristic or LLM); `BuildFactoryOrderDevelopmentProposal` does not fit (it needs structured input). Heavier, and an LLM structurer is arguably itself a Phase-2/agent concern. *Rejected for Phase 1* — it builds the deferred AI-assist seam now, out of slice.
- **C — Site-side LLM structuring** via Site's `intelligence`/`mind` package. Makes "agent structures the request" real, but injects an LLM call (latency, cost, nondeterminism) into the read-only console, deviates from the doc's "draft ← work pure function" data-flow decision, and complicates the no-fabrication contract. *Rejected for Phase 1.*

## Architecture (for Approach A)

**View-model (site-local, no `work` import):**
```
ConsoleIntakeDraft {
  Requestor       string      // current operator (owner)
  StructuredBy    string      // "manual — AI assist deferred"
  RequestText     string      // verbatim, never mutated
  Title           string
  TargetRepo      string      // default "transpara-ai/work" (editable)
  RiskClass       string      // default "low"; one of low|medium|high|critical
  Cell            string      // default "implementation"
  DefinitionOfDone   []string // empty → editable checklist
  AcceptanceCriteria []string
  ExpectedOutputs    []string
  SubmitDeferred  bool        // always true in Phase 1 (drives the disabled control + fail-safe note)
  AIAssistDeferred bool       // always true in Phase 1
  Notices         []string
}
```
A small pure builder `buildConsoleIntakeDraft(requestText, requestor string) ConsoleIntakeDraft` applies the declared defaults and preserves the request verbatim. (No freshness state machine here — Intake consumes no live projection; the honest seams are the two deferred controls, not a staleness badge.)

**Handlers / routes** (mirror Kanban):
- `GET /console/intake` → Step-1 page (free-text + optional title; `hx-post` to the draft route).
- `POST /console/intake/draft` → builds the view-model from the posted text and renders the Step-2 review **fragment** (editable form + live preview + the two disabled seams). Registered in **both** `Register` (via `writeWrap`) and `RegisterReadOnlyConsole`.
- Enable the Intake tab in `console.templ`.

**Templ components:** `consoleIntake` (Step-1 form), `consoleIntakeDraft` (Step-2 review/preview), reusing the existing console shell, freshness vocabulary where relevant, and — if natural — the shared order card/drawer for the preview.

**Honest-data application:** verbatim request preserved; no invented draft content; the two deferred writes are visibly disabled with fail-safe statements; if the request is empty, the draft step returns an honest "nothing to draft" rather than a hollow form.

## Testing approach

- `buildConsoleIntakeDraft` unit tests: verbatim request preserved exactly; declared defaults applied; empty request → honest empty result (no fabricated fields); risk default is a valid class.
- Handler/render tests (mirror `console_kanban_test.go`): `GET /console/intake` renders the compose form; `POST /console/intake/draft` with sample text renders the review fragment containing the verbatim request, the requestor's name, and **both** deferred-seam statements ("Phase-2 governed action" + "AI assist deferred"); the submit control is rendered **disabled** (assert it is not an active form submit / governed POST).
- Per MFOF-001: the implementation PR will include desktop + mobile screenshots of the Intake compose and review screens.

## What this is explicitly NOT (Phase 1)

- **No AI structuring** of the free text (deferred seam — needs an upstream structurer that does not exist yet).
- **No submit / no governed write / no order seeded** (deferred to Phase 2 `SeedFactoryOrder` via a governed endpoint).
- **No persistence** of the draft (no write ⇒ the draft lives only in the request/response).
- **No `work` package import**; no LLM call; no EventGraph or Work/Hive/Agent state mutation.

## Open Questions (need an owner decision)

1. **Accept the manual-compose framing (Approach A) for Phase-1 Intake?** Given there is no free-text structurer to call, this is the only honest option without building new structuring logic. If you'd rather the AI-assist be real now, that's Approach B/C and a larger, separately-scoped effort (and arguably a Phase-2/agent concern). **Recommendation: A.**
2. **Resequence?** Intake is the **thinnest** Phase-1 slice — no live upstream, and both writes deferred. **Config** (read-only role×model matrix) has a real upstream (`operator-projection.ModelSelection` + the `modelconfig` catalog) and may be the higher-value next build. Do you want **Config (Plan 4) before Intake**? **Recommendation: consider yes** — but I designed Intake first because that's what you pointed me at.
3. **POST for a pure compute** inside the read-only console — comfortable with `POST /console/intake/draft` returning a computed scaffold (no write), or prefer a GET-with-text variant? **Recommendation: POST** (cleaner for a textarea; it is not a governed write).

## Precedent & evidence index

- `SITE-MISSION-CONTROL-DESIGN-001` — parent console design (this realizes its Intake surface; **amends** its Intake data-source claim per the finding above).
- `work/factory_order_proposal.go:186` `BuildFactoryOrderDevelopmentProposal` — pure proof-of-work assembler (requires structured input; not a free-text structurer).
- `work/factory_order.go:39` `FactoryOrder` seed DTO + `SeedFactoryOrder` (the Phase-2 governed write).
- Merged console foundation: `site/graph/console.go`, `console.templ`, `handlers.go` (Plan 1 #198, Plan 2 #199).
