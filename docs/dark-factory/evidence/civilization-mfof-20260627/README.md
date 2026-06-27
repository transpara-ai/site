---
doc_id: SITE-CIVILIZATION-MFOF-EVIDENCE-20260627
title: Civilization MFOF Design Evidence Index
doc_type: evidence-index
status: proposal
version: 0.1.0
created: 2026-06-27
updated: 2026-06-27
owner: Michael Saucier
steward: assistant
primary_repo: transpara-ai/site
canonical_arc: dark-factory/v4.0
authority: evidence-index only; no runtime, deploy, Test 001 GREEN, autonomy increase, or production EventGraph write
---

<!-- transpara:artifact id=SITE-CIVILIZATION-MFOF-EVIDENCE-20260627 type=evidence-index version=0.1.0 status=proposal -->

# Civilization MFOF Design Evidence Index

This index records the source material used for
`SITE-CIVILIZATION-MFOF-DESIGN-001` version `0.1.0`. It is a design evidence
index, not an authority packet and not implementation evidence.

## References

| Reference | Role in the design update | Boundary |
| --- | --- | --- |
| `transpara-ai/docs:dark-factory/v4.0/README.md` | Accepted canonical Dark Factory baseline. | Canonical arc source; does not itself authorize MFOF implementation. |
| `transpara-ai/docs:dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md` | Accepted integration arc and governed event/gate record. | Source for current arc framing; not a newer baseline. |
| `transpara-ai/docs:designs/telemetry-mission-control-design-v0.4.1.md` | Historical telemetry dashboard and mission-control visualization precedent. | Visualization precedent only; stale repo/process assumptions are not carried forward. |
| `transpara-ai/docs:designs/telemetry-claude-code-prompts-v0.4.1.md` | Companion prompt record showing `0.4.1` implementation sequence and completed prompt posture. | Historical context only; Claude-specific flow is not a Site implementation contract. |
| `transpara-ai/wiki:raw/civilization/stage-0-institutional-substrate/README.md` | Wiki proposal/provenance corpus for `Civilization v4.1 Event 15 Gate Y`. | Proposal/provenance corpus only, not accepted canonical docs. |

## Reconciliation Finding

The `0.4.1` telemetry documents materially affect visualization criteria, but
they do not change the current canonical Civilization arc. The current accepted
arc remains `dark-factory/v4.0`. Wiki `Civilization v4.1 Event 15` material is
proposal/provenance context, and `G-4.1` / `G-4.2` are gate labels rather than
semver baselines.

## Carry-Forward Rules

- Carry forward honest staleness, connection state, source labeling, compact
  health summaries, live-event framing where real, and explicit unavailable
  states.
- Do not carry forward `transpara-ai-summary`, `lovyou-ai-summary`, static
  poster routing, Claude-specific prompt flow, old process boundaries, or any
  claim that telemetry tables are canonical audit truth.
- Site MFOF implementation must preserve read-only scanner and projection
  boundaries unless a separate governed authority packet expands scope.

## Verification Checklist

- `git diff --check`
- Site docs link or reference check, if present.
- Confirm design text does not claim a newer accepted canonical arc.
- Confirm design text does not claim Test 001 closure or GREEN status.
- Confirm design text does not claim production go-live, runtime execution
  authority, EventGraph production write authority, value allocation, or
  autonomy increase.
