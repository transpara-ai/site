---
doc_id: SITE-CIVILIZATION-MFOF-DESIGN-001
title: Civilization Minimum Functioning Operational Front End
doc_type: design
status: proposal
version: 0.1.0
created: 2026-06-27
updated: 2026-06-27
owner: Michael Saucier
steward: assistant
primary_repo: transpara-ai/site
canonical_arc: dark-factory/v4.0
telemetry_precedent: transpara-ai/docs:designs/telemetry-mission-control-design-v0.4.1.md
authority: design-only; no runtime, deploy, Test 001 GREEN, autonomy increase, or production EventGraph write
---

<!-- transpara:artifact id=SITE-CIVILIZATION-MFOF-DESIGN-001 type=design version=0.1.0 status=proposal -->

# Civilization Minimum Functioning Operational Front End

## Purpose

This document defines the minimum functioning operational front end for the
Civilization operator surfaces in `transpara-ai/site`. It incorporates the
2026-06-27 development-arc sweep and the telemetry mission-control `0.4.1`
findings as visualization precedent, while preserving `dark-factory/v4.0` as
the current canonical arc.

This is a design input for the Site implementation. It does not authorize
runtime execution, deployment, production EventGraph writes, Hive wake/start,
Work mutations, Test 001 GREEN, residual-risk closure, value allocation,
protected settings changes, or autonomy increase.

## Version And Source Reconciliation

| Source | Current disposition | Implementation consequence |
| --- | --- | --- |
| `transpara-ai/docs:dark-factory/v4.0/README.md` | Accepted canonical Dark Factory baseline. | Site must treat this as the canonical arc source until a later accepted baseline supersedes it. |
| `transpara-ai/docs:dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md` | Accepted integration arc for the current governed event/gate record. | Site operator UI must not imply a newer canonical arc when rendering Civilization status. |
| Wiki `Civilization v4.1 Event 15` material | Proposal/provenance corpus copied into Wiki raw sources, not accepted canonical docs. | Site may cite it only as provenance or proposal context, not as authority. |
| `G-4.1` / `G-4.2` references | Gate or packet labels in older deployment/reunification material. | Do not treat these labels as semver baselines. |
| Telemetry mission-control `0.4.1` | Historical observability design and dashboard precedent. | Carry forward visualization lessons only; do not carry forward stale repo, routing, or process assumptions. |

## Telemetry v0.4.1 Carry-Forward

The `0.4.1` telemetry mission-control design is relevant because it captured
operator-facing observability patterns before the current Civilization operator
stack existed. Its durable ideas should inform the MFOF visualization work:

- Show honest freshness: every runtime, monitor, scanner, or projection surface
  needs a clear current, stale, unavailable, or projection-only state.
- Keep connection and polling state visible when a view depends on a service,
  artifact, or live endpoint.
- Summarize health before detail: compact status cards should precede dense
  event, gate, task, or scan tables.
- Preserve a live-event framing where data exists, but avoid pretending that
  static fixtures or scanner artifacts are live event streams.
- Apply the old "no fake green lights" rule everywhere: unavailable truth
  sources must render as unavailable, not as success.

The following `0.4.1` assumptions are stale and must not be inherited:

- `transpara-ai-summary` or `lovyou-ai-summary` as the operator front-end home.
- Static architecture-poster routing as the production operator surface.
- Claude-specific prompt sequencing as an implementation contract.
- The old three-repo telemetry process boundary as current architecture.
- Any implication that telemetry tables are canonical audit truth.

## MFOF Surface Requirements

The first operational front end is a clean Site-owned operator experience across
control, observatory, ingestion, scan monitoring, and Kanban surfaces. It should
be dense enough for repeated operator use, but every dense table or chart must
keep source, freshness, and authority boundaries visible.

| Surface | Minimum behavior | Required boundary |
| --- | --- | --- |
| Control | Show operator entry points, current blockers, protected-action waits, and next allowed non-protected movement. | No approve, merge, wake, deploy, restart, write, or autonomy controls unless separately authorized. |
| Observatory | Show health, telemetry, runtime, and projection status with compact cards, timelines, or small multiples where useful. | Runtime unavailable and stale states must be explicit. No fixture data may render as live truth. |
| Ingestion | Show incoming source records, issue intake, proposal/provenance records, and malformed or paused inputs. | Intake display is not acceptance, authority, or source-of-truth elevation. |
| Scan monitoring | Show scanner artifact state, PR-ready counts, parked human-scope blockers, and evidence refs. | Scanner output is read-only evidence and does not mutate GitHub or authorize implementation. |
| Kanban | Show issue/task lanes with progressive disclosure for evidence, labels, blockers, and refs. | Parked or human-scope cards must not imply PR-readiness, runtime readiness, or autonomy. |

## Visualization Acceptance Criteria

- Observatory and control pages distinguish current, stale, unavailable, and
  fixture/projection-only data in text and visual state.
- Monitoring cards show source, last update, boundary, and evidence link or ref
  wherever the source exposes one.
- Kanban and scan views surface parked human-scope blockers without implying
  that the blocker is PR-ready, merged, executable, or autonomous.
- Charts, sparklines, tables, and timelines follow a Tufte-style approach:
  compact labels, high data-to-ink ratio, no ornamental gradients, and no color
  that carries status without text.
- Empty and degraded states are first-class. A missing service, missing scanner
  artifact, or rejected projection must be visible and actionable without
  reading logs.
- Mobile views preserve the operator facts: state, source, freshness, blocker,
  and next allowed action must remain readable.

## Implementation Notes

- Prefer existing Site operator patterns in `graph/ops.go`, `graph/ops.templ`,
  and adjacent projection helpers.
- Keep Site as the display and interaction shell. EventGraph remains truth,
  Hive owns runtime/governance orchestration, Work owns work-item semantics, and
  Docs owns canonical governance records.
- Do not introduce a new charting dependency unless a specific visualization
  cannot be expressed with the existing Templ/Tailwind/SVG patterns.
- Any service-dependent view must tolerate missing endpoints and render the
  failure as an explicit unavailable state.
- Any implementation PR must include desktop and mobile screenshots for changed
  operator pages before asking for human approval.

## Evidence Index

Supporting evidence for this design update is recorded in:

`docs/dark-factory/evidence/civilization-mfof-20260627/README.md`
