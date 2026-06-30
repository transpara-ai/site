---
doc_id: SITE-MISSION-CONTROL-DESIGN-001
title: Civilization Mission Control Console
doc_type: design
status: proposal
version: 0.1.0
created: 2026-06-30
updated: 2026-06-30
owner: Michael Saucier
steward: assistant
primary_repo: transpara-ai/site
canonical_arc: dark-factory/v4.0
extends: SITE-CIVILIZATION-MFOF-DESIGN-001
consumes:
  - transpara-ai/docs:dark-factory/v4.0/implementation/epics/epic-10-site-civilization-projection-consumer
  - transpara-ai/docs:dark-factory/v4.0 (Epic 12 progress-evidence, Epic 13 review console)
  - transpara-ai/hive:hive-ops-api (operator-projection, civilization assembly, model-policy, runs)
  - transpara-ai/work:work-server (/tasks, /telemetry/*)
telemetry_precedent: transpara-ai/docs:designs/telemetry-mission-control-design-v0.4.1.md
authority: design-only; no runtime, deploy, Test 001 GREEN, autonomy increase, governed write, or production EventGraph write is authorized by this document
---

<!-- transpara:artifact id=SITE-MISSION-CONTROL-DESIGN-001 type=design version=0.1.0 status=proposal -->

# Civilization Mission Control Console

## Purpose

This document defines a single, coherent operator console — **Mission Control** —
for the Civilization in `transpara-ai/site`. It is the concrete visual and
information-architecture realization of `SITE-CIVILIZATION-MFOF-DESIGN-001`
(Minimum Functioning Operational Front End): MFOF-001 set the *principles* for the
operator surfaces (honest freshness, no fake green lights, Site as display shell,
authority boundary); this document specifies the *console* that delivers them as
one surface with one visual language.

It replaces the scattered, divergent `/ops/*` dashboards (observatory, civilization
assembly, refinery, review console, hive dashboard, board) — which today use
roughly five competing dashboard idioms rendered in very large generated Templ
files — with one console whose four views share components, navigation, and a
single data contract.

This is a design input for Site implementation. It authorizes no runtime
execution, deployment, production EventGraph write, Hive wake/start, Work mutation,
Test 001 GREEN, residual-risk closure, protected settings change, or autonomy
increase. The governed-write actions described in §7 are **target capability**,
gated behind separate explicit authorization and excluded from the Phase 1 slice.

## Operator Needs (scope)

The console serves four operator needs, confirmed with the owner:

1. **Monitor the running Civilization** — a glanceable health wall.
2. **Alter config parameters** — chiefly which LLM model is used for which
   agent/role.
3. **Ingest a human request** and convert it, with assistance, into a proper
   factory order (triage → structured order).
4. **Track factory orders on a Kanban** from multiple viewpoints (status, agent
   activity, risk/aging, source).

## Decisions Captured

These decisions were settled with the owner during design and are binding inputs
to the implementation plan:

| # | Decision | Rationale |
| --- | --- | --- |
| D1 | One **unified console** ("Mission Control"), clean concept; retire the scattered `/ops/*` dashboards view-by-view as each is superseded. | Removes the "five competing idioms" problem; the mess is presentation-layer, not data-layer. |
| D2 | **Site owns all four operational surfaces; Wiki stays the static narrative substrate.** | Wiki is a Python-compiled static knowledge layer by design; making it live would fight the architecture. Resolves "Site or Wiki" → Site for all live monitoring/ops/control/config. |
| D3 | The **Health Wall is the home surface** and the center of gravity; monitoring is optimized for a 5-second glance, with detail one click deeper. | Owner's primary monitoring mode is "glanceable health wall." |
| D4 | Intake is **draft → human approves → submit**. AI structures the free-text request; nothing runs until the human approves. The human remains the **requestor/owner** throughout; the drafting agent is credited subordinately. | Matches "Site proposes, Civilization decides," the authority gates, and the owner's fail-safe-by-default standard. |
| D5 | The Kanban supports **four first-class lenses** — by status, by agent activity, by risk+aging, by source/requester — over the same card set. Default lens is **risk+aging ("what needs me")**. | The console should open on the operator's attention, not a neutral inventory. |
| D6 | **Render stack: Site-native** (Templ + HTMX + SSE + a shared Tailwind component library) for Phase 1, to perfect elements and workflow. A later increment may add client-side islands or an SPA for polish. The console BFF emits JSON view-models so the view layer can be upgraded without touching data wiring. | "Minimum code, match surrounding code"; the Kanban is read-mostly, so the Site-native ceiling is high enough. No new build chain in a Go repo for Phase 1. |
| D7 | Phase 1 is a **full vertical slice** across all four views (reads + intake draft + read-only config), then deepen. Governed writes are a separate, explicitly-authorized increment. | Validates the whole concept end-to-end; honors MFOF-001's "no write controls unless separately authorized." |

## Architecture

### Boundary (non-negotiable)

Per `dark-factory/v4.0` ADR-001 Decision 8 and Epic-10 REQ-T2: **Site is a
read-only projection consumer that may submit commands to governed APIs.** It must
not write EventGraph records, own authority, execute work directly, or mutate
Work/Hive/Agent state as truth. Every "write" in this console is a governed command
POSTed to `hive-ops-api` or `work-server`, which record the resulting events. Site
proposes; the Civilization decides and records.

### The Console BFF (the spine)

A dedicated backend-for-frontend layer in Site is the antidote to the current
tangle (handlers each doing their own ad-hoc fetching). It is the single place that
fetches from the upstream services and shapes the result into view-models.

- **Inputs:** `hive-ops-api` (`/api/hive/operator-projection`,
  `/api/hive/civilization/assembly-projection`), `work-server` (`/tasks`,
  `/telemetry/*`), and EventGraph chain/health reads.
- **Output:** typed JSON view-models, one per view, each carrying an explicit
  **freshness/derivation status** (`current | stale | partial | unavailable |
  fixture-only`) and a `generated_at` + source reference, per Epic-10's consumer
  contract.
- **Rendering:** Phase 1 renders these view-models with Templ; SSE pushes wall
  updates; HTMX handles fragment refresh and navigation. Because the contract is
  JSON view-models, a future islands/SPA layer consumes the same spine unchanged.

### Honest-Staleness Contract (applies to every surface)

This is MFOF-001's "no fake green lights" made structural. Every data element
resolves to one of:

- **current** — live, within freshness window. Normal color.
- **stale** — last update older than the window. Shown explicitly ("last seen
  53s ago"), muted, with a hollow indicator. Never rendered as healthy/idle.
- **partial** — some sources resolved, some did not. The view says so.
- **unavailable** — source down or projection rejected. First-class empty state,
  actionable without reading logs.
- **fixture-only** — fallback/typed data, never dressed as a live stream.

A dead data pipe must *look* dead. A view never substitutes a comforting default
for missing truth.

## Shared Components

The console is built from a small set of reused components — this is what keeps the
four views coherent and prevents idiom drift:

1. **App shell + tab nav** — Health wall · Kanban · Intake · Config. Global status
   pill (operational/degraded), live heartbeat with `updated Ns ago`.
2. **Order card** — leads with the **human title**; `fo_id` is a small mono tag.
   Carries submitter avatar, risk chip, and aging. Used on the Kanban and anywhere
   an order is listed.
3. **Order details drawer** — one component, opened from any surface (a wall agent
   row, a Kanban card, an intake result). Shows: human title, submitter + submitted
   date, original request text, effort (to-date and predicted-to-complete) as a
   progress bar, and a meta grid (started, predicted-done, risk, cell, agent,
   iterations, **linked PR**). Footer links to causal trace and acceptance criteria.
4. **Agent row** — role · model badge · live state · current action · spend. Drives
   the Health Wall roster and the by-agent Kanban lens.
5. **Freshness/state badge** — the visual vocabulary for the staleness contract.
6. **Governed-action control** (Phase 2+) — buttons that POST to a governed API,
   always paired with the fail-safe statement of what will happen.

## The Four Surfaces

### Health Wall (home, need #1)

Answers one question in five seconds: *is the Civilization alive, healthy, within
budget, and does anything need me?*

- **Status strip:** global operational pill + live heartbeat (`updated Ns ago`).
- **KPI row:** active agents (n/total), daily spend vs cap (meter), event rate,
  chain integrity (verified + event count), "needs you" (pending approvals count,
  amber).
- **Agent roster (centerpiece):** one row per agent — role, model, live state,
  *what it is doing this second*, spend. Honest staleness per row (a no-telemetry
  agent shows "last seen Ns ago · stale", hollow dot — never green/idle).
- **Side panels:** pending approvals (read in Phase 1; approve/deny is a governed
  action in Phase 2), and a recent civilization-events feed.
- **Data:** roster ← `telemetry_agent_snapshots`; KPIs ← `telemetry_hive_snapshots`
  + EventGraph chain verification; approvals ← `operator-projection.PendingApprovals`.

### Kanban (need #4)

Same card set, four lenses; **defaults to risk+aging**.

- **Lenses:** by status (lifecycle columns), by agent activity (swimlanes per
  agent), by risk+aging (triage — high-risk/stuck surface to top), by source
  (trace a request to the order(s) it spawned).
- **Cards** lead with the human title; carry `fo_id` tag, submitter, risk chip,
  aging. Read-mostly — orders advance because agents advance them, not by drag.
- **Click → order details drawer** (shared component).
- **Status columns** map to `work.TaskStatus`. MVP grouping:
  created → ready → running → verifying → certified, with blocked/failed/rejected
  surfaced (and emphasized in the risk+aging lens).
- **Data:** `work-server /tasks` (+ filters), assignment events for the agent lens,
  runtime-result events for effort.

### Intake (need #3)

Draft → review → submit; the human owns the order throughout.

- **Step 1 Request:** operator pastes free-text. Recorded with the operator as
  **requestor/owner**.
- **Step 2 Review draft:** an agent (e.g. Strategist) structures the request into a
  FactoryOrder draft. The original request is preserved verbatim. All fields are
  editable: title, risk class (suggested, editable), target repo, **definition of
  done (visible checklist)**, acceptance criteria, **expected outputs (artifact
  inventory)**. Provenance shows the human as requestor; the agent credited
  subordinately ("structured by …").
- **Step 3 Submit (governed, Phase 2):** "Approve & submit order" is the single
  gated write. The fail-safe statement is explicit: nothing runs until approval; on
  submit the order is recorded as governed and owned by the requestor.
- **Data/flow:** draft ← `work.BuildFactoryOrderDevelopmentProposal` (pure, no side
  effects); submit → `POST hive/runs` (or `work` seed) as a governed command.

### Config (need #2)

Role × model matrix.

- **Rows:** roles (Strategist, Planner, Implementer, Guardian, CTO, SysMon, …).
- **Columns:** model (picker from catalog), tier (frontier/balanced/volume), per-
  call cost. `can_operate` roles marked.
- **Model picker** lists catalog models with tier + cost — cost-aware selection.
- **Governed change (Phase 2):** a model change is recorded as a
  `model_role_policy_updated` event and **applies on the agent's next iteration,
  never mid-task**. The matrix shows pending changes explicitly.
- **Phase 1:** read-only view of the current role→model assignment + catalog.
- **Data:** `catalog-mixed.yaml` / `modelconfig` catalog (read); current policy ←
  `operator-projection.ModelSelection`; change → `POST
  hive/model-selection/role-policy`.

## Governed-Write Boundary (deferred increment)

The following are the *only* write actions in the target console. Each is a
governed command to an existing endpoint that records an event; none mutate Site
state as truth. All are **excluded from Phase 1** and require separate explicit
authorization (consistent with MFOF-001 and the owner's fail-safe standard):

| Action | Surface | Governed endpoint | Safety property |
| --- | --- | --- | --- |
| Approve / deny a pending authority request | Health Wall, Kanban | `POST /api/hive/operator-decision` | Idempotent; decides only the gated action. |
| Submit a drafted factory order | Intake | `POST /api/hive/runs` (queued) | Human-approved; queued, not immediately executed. |
| Change a role's model | Config | `POST /api/hive/model-selection/role-policy` | Applies on next iteration, never mid-task. |

Fail-safe framing: the permissive outcome (an order entering the factory, a model
changing, an action approved) is always the explicitly-proven branch behind one
deliberate click; the default is to show, not to act.

## Non-Goals

- Not a replacement for the Wiki narrative substrate.
- Not a new authority source of truth; no autonomy increase.
- No new charting dependency unless a visualization cannot be expressed with the
  existing Templ/Tailwind/SVG patterns (MFOF-001 implementation note).
- No public/cloud exposure; on-prem operator surface, behind the firewall.

## Phased Build Order

**Phase 1 — Full vertical slice (reads + intake draft).**
- Console BFF spine + view-model contracts with the honest-staleness states.
- Shared components: app shell/nav, order card, order details drawer, agent row,
  freshness badge.
- Health Wall (read), Kanban (read; risk+aging default + at least the status lens),
  Intake (request + AI draft + editable review; **no submit yet**), Config
  (read-only matrix + catalog).
- Proves data plumbing and the single visual language end-to-end without any write.

**Phase 2 — Governed writes (separately authorized).**
- Approve/deny, submit order, apply model change — each behind the governed-write
  boundary above and explicit authorization. Deepen Kanban lenses (agent, source)
  and the drawer.

**Phase 3 — Polish.**
- Optional client-side islands/SPA over the same BFF view-models; richer live
  interactions; retire remaining legacy `/ops/*` surfaces.

## Open Questions

1. **Predicted effort-to-complete** is the one genuinely new computation (effort
   *to date* is a sum over runtime-result events, which exist). Phase 1 = simple
   heuristic (average of comparable past orders by cell/risk); refine later.
2. **Operators / auth:** scope is effectively the External Committee (owner) plus a
   small internal group; confirm whether the console needs more than the existing
   Site OAuth + read/write wrap before Phase 2 writes land.
3. **SSE vs polling fallback:** SSE for the live wall, with HTMX polling as the
   degraded fallback when SSE is unavailable (must itself render honest staleness).

## Testing Approach

- View-model unit tests assert the freshness state machine: missing/partial/stale
  inputs must yield explicit unavailable/stale states, never a healthy default
  (test the whole input domain, including error paths).
- Component snapshot/handler tests for the shared order card and details drawer
  across surfaces (one component, consistent output).
- Phase 2 governed actions: tests assert the write is a governed POST and that the
  permissive branch is reachable only via the explicit, proven path.
- Per MFOF-001: any implementation PR includes desktop and mobile screenshots for
  changed operator pages before human approval.

## Precedent & Evidence Index

- `SITE-CIVILIZATION-MFOF-DESIGN-001` — surface principles this design realizes.
- `transpara-ai/docs:dark-factory/v4.0` Epic-10/12/13 — Site civilization
  projection consumer, progress-evidence display, External Committee review console.
- `transpara-ai/docs:designs/telemetry-mission-control-design-v0.4.1.md` —
  observability/visualization precedent (carry-forward per MFOF-001 §"Carry-Forward").
- Survey of current state (2026-06-30): `site` operator surfaces, `hive`
  hive-ops-api projections + model-policy/runs endpoints, `work` factory-order model
  + telemetry, `eventgraph` as source of truth.
