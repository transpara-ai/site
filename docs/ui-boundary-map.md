# UI Boundary Map

Date: 2026-04-29

This document defines where browser UI belongs across `site`, `work`, `hive`, `agent`, `eventgraph`, and `docs`.

The goal is to make the UI surface maintainable: `site` owns browser-facing product and operator screens, `work` owns task/workflow APIs and event streams, and `hive` owns runtime/event emission. EventGraph remains the source of durable records and projections.

## Boundary Rules

1. Browser UI routes live in `site`.
2. `work` serves APIs, SSE streams, and workflow state. It should not grow new browser UI.
3. `hive` emits runtime events, diagnostics, status, and telemetry. It should not serve application UI.
4. `eventgraph` stores durable facts and projection inputs. UI should consume projections and show source/freshness when status can lag.
5. `agent` executes or coordinates work. Agent status belongs in projections exposed through `work` or `hive`, then rendered by `site`.
6. `docs` owns human-readable design notes, operating guides, and architecture records.

## Current Route Inventory

| Host | Route or surface | Current role | Disposition |
| --- | --- | --- | --- |
| `site` | `/`, `/discover`, `/blog`, `/reference`, `/agents`, `/market`, `/knowledge`, `/activity`, `/search` | Public/product site and shared discovery surfaces | Keep in `site` |
| `site` | `/app`, `/app/{slug}`, `/app/{slug}/*` | Main product workspace UI | Keep in `site` |
| `site` | `/app/{slug}/board`, refinery/journey views when enabled | Work intake, design, and status review UI | Keep in `site`; simplify FSM in a follow-up change |
| `site` | `/hive`, `/hive/feed`, `/hive/stats`, `/hive/status` | Public live-build page and HTMX partials for the phase timeline | Keep as public/product UI; route operator status through `/ops/hive` |
| `site` | `/api/palette`, `/api/members` | Site UI helper APIs | Keep in `site` |
| `site` | `/api/hive/diagnostic`, `/api/hive/escalation`, `/api/mind-state` | Service ingress and status endpoints used by UI/runtime | Keep short term; document as ingress APIs before moving or renaming |
| `work` | `/` | Inline work dashboard | Legacy UI; migrate to `site` `/ops/work` |
| `work` | `/w/{workspace}` | Workspace task dashboard | Legacy UI; migrate to `site` `/ops/work` or `/app/{slug}/work` |
| `work` | `/telemetry/` | Embedded mission-control telemetry dashboard | Legacy UI; migrate to `site` `/ops/telemetry` |
| `work` | `/tasks`, `/tasks/{id}`, task mutation routes | Task API | Keep in `work` |
| `work` | `/telemetry/status`, `/telemetry/agents`, `/telemetry/stream`, `/telemetry/phases`, `/telemetry/pipeline/report`, `/telemetry/health`, `/telemetry/sse`, `/telemetry/roles`, `/telemetry/actors`, `/telemetry/layers`, `/telemetry/overview` | Telemetry APIs and streams | Keep in `work` |
| `work` | `/events/subscribe` | Event stream | Keep in `work` |
| `work` | `/health` | Service health | Keep in `work` |
| `hive` | Runtime loop files, diagnostics, events, status emission | Runtime plane | No browser UI |
| `agent` | Agent execution and coordination | Worker/control plane | No browser UI |
| `eventgraph` | Records, graph facts, projection inputs | Durable data plane | No browser UI |
| `docs` | Architecture and operating documentation | Human documentation | Keep docs only |

## Target Route Map

```text
site
  /app
  /app/{slug}
  /app/{slug}/...
  /ops
  /ops/work
  /ops/telemetry
  /ops/hive
  /ops/refinery

work
  /health
  /tasks
  /tasks/{id}
  /tasks/{id}/...
  /telemetry/status
  /telemetry/agents
  /telemetry/stream
  /telemetry/phases
  /telemetry/pipeline/report
  /telemetry/health
  /telemetry/sse
  /telemetry/roles
  /telemetry/actors
  /telemetry/layers
  /telemetry/overview
  /events/subscribe

hive
  event emission only
  runtime diagnostics only

agent
  execution only

eventgraph
  records and projections only
```

## Operator UI Ownership

`site` should provide one operator shell under `/ops`. Operator pages can initially link to or proxy existing `work` surfaces while native `site` implementations are built.

| Target page | Owner | Backing data |
| --- | --- | --- |
| `/ops` | `site` | Aggregated links and high-level health |
| `/ops/work` | `site` | `work` task APIs and task events |
| `/ops/telemetry` | `site` | `work` telemetry APIs and SSE |
| `/ops/hive` | `site` | `hive` diagnostics/status projected through existing APIs |
| `/ops/refinery` | `site` | EventGraph/work projections for intake, design, execution, and completion state |

## Work UI Deprecation Plan

1. Add `/ops` navigation in `site` that exposes current operator surfaces in one place.
2. Move telemetry rendering into `site` while continuing to consume `work` telemetry APIs.
3. Move workspace task rendering into `site` while continuing to consume `work` task APIs.
4. Mark `work` browser routes as legacy in code and documentation.
5. Stop adding new UI features to `work`.
6. Remove `work` browser routes or guard them behind an explicit debug/development flag after replacement pages are live.

## Refinery FSM Placement

The refinery UI belongs in `site`. `work` should expose execution status and evidence as API/projection data, not as browser state.

The simplified refinery FSM should be:

```text
Inbox -> Refining -> Review -> Ready -> Done
```

Execution activity such as assigned, building, blocked, and reviewing is not a top-level refinery state. It should be shown as execution status on `Ready` items or in an active-work swimlane/filter.

| Execution condition | Display location |
| --- | --- |
| Ready but unassigned | `Ready`, execution status `unassigned` |
| Assigned but not started | `Ready`, execution status `assigned` |
| Actively building | `Ready`, execution status `building`; also visible in active-work filters |
| Blocked while building | `Ready`, execution status `blocked` |
| Awaiting implementation review | `Ready`, execution status `reviewing` |
| Completed with evidence | `Done` |

## Instrumentation Requirements

Every operator or refinery page rendered by `site` should expose enough state for dashboards and human status reporting:

| Field | Purpose |
| --- | --- |
| `source_system` | Identifies whether the status came from `site`, `work`, `hive`, `agent`, or `eventgraph` |
| `source_id` | Links UI rows back to the durable task, node, actor, or event |
| `projected_at` | Shows when the UI projection was generated |
| `last_event_at` | Shows when the underlying work last moved |
| `state` | Shows the simplified user-facing state |
| `execution_status` | Shows active execution status without expanding the FSM |
| `blocked_reason` | Provides a human-readable explanation when stuck |
| `owner` | Shows actor, role, or service responsible for the next step |
| `next_action` | Provides the specific action required to move forward |
| `evidence_count` | Makes completion and review evidence visible |

## Open Risks

1. The live refinery route at `http://nucbuntu:8201/app/journey-test/refinery?profile=transpara` must be reconciled with the checked-out `site` source before code changes are made to that page.
2. `/hive` is public/product-facing. Operator Hive status belongs under `/ops/hive`.
3. The migration needs an auth and proxy decision for `site` pages that consume `work` APIs from the browser.
4. Existing dashboards may link directly to `work` routes. Keep redirects or compatibility links during migration.

## Acceptance Criteria

1. Every current browser UI route has a documented owner and disposition.
2. Future browser UI work defaults to `site`.
3. New `work` and `hive` UI routes require an explicit architecture exception.
4. Work/refinery dashboards can report `source_system`, `source_id`, `projected_at`, `last_event_at`, `state`, `execution_status`, `blocked_reason`, `owner`, `next_action`, and `evidence_count`.
5. The simplified refinery FSM is documented as the target model before implementation begins.
