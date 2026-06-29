# CFADA Packet: site#195 Civilization Operator UI Rebuild

## Packet Identifier

- Repository: `transpara-ai/site`
- Issue: `https://github.com/transpara-ai/site/issues/195`
- Branch: `codex/site-195-civilization-operator-ui`
- Base head at packet creation: `55521a3e4e21cf4ba3a7cc5804952e4e283d7a39`
- Packet created: `2026-06-29T12:07:17Z`

## Required Reviewer Skills

Claude must use these skills while reviewing this design:

- `simplify`
- `frontend-design:frontend-design`

## Source Of Intent

Michael approved a complete workflow for rebuilding the Civilization operator UI around three screens:

1. Observation: monitoring, high-level telemetry, Factory statistics, Civilization health, live agents/actors, and runs or expenses by agent/role.
2. Control: setting which LLM model each Agent/Role will use, setting budgets, editing or setting the Council agenda, invoking ad-hoc Council Meetings, and determining actions to evolve the Civilization.
3. Non-admin Human Usage: uploading Markdown files as artifacts intended to be converted to Factory Orders, viewing the active Factory in Motion, and filtering by Human Submitter.

Michael also approved the Codex skill stack and required that CFADA instruct Claude to use `simplify` and `frontend-design:frontend-design`.

## Canonical Context

- Site is the display/operator shell.
- EventGraph remains truth.
- Hive owns runtime/governance orchestration.
- Work owns work-item semantics.
- Docs owns canonical governance records.
- Current accepted Civilization/Dark Factory arc remains `dark-factory/v4.0`.
- Telemetry `0.4.1` is visualization precedent only.

Primary local design input:

- `docs/designs/civilization-minimum-functioning-operational-front-end-v0.1.0.md`
- `docs/dark-factory/evidence/civilization-mfof-20260627/README.md`

## Proposed Design Scope

Implement a Site-only UI rebuild:

- `/ops`: compact operator task hub with direct paths to Observation, Control, and Human Factory usage.
- `/ops/observation`: compact operator monitoring screen with health, Factory throughput, agent liveness, run/cost/token posture, blockers, freshness, and evidence refs.
- `/ops/control`: bounded admin control room that queues intent records or renders disabled controls. It must not execute protected actions.
- `/factory`: non-admin human workspace for Markdown artifact submission, Factory-in-motion display, and submitter filtering.

Existing dense routes remain available as drilldowns:

- `/ops/telemetry`
- `/ops/observatory`
- `/ops/civilization`
- `/ops/github-canonical`
- `/ops/evidence`
- `/ops/public-proof`
- `/ops/review-console`
- `/ops/hive/intake`

## Design Requirements

- Reduce interpretive burden. Operators should see state, source, freshness, next allowed action, and blocker posture without reading long evidence prose.
- Use Tufte-style high data-to-ink visual design: compact tables, small multiples, sparklines or inline bars where useful, no ornamental gradients, no color-only status encoding, direct labels.
- Distinguish current, stale, unavailable, and fixture/projection-only data in text and visual state.
- Any monitoring card or row should show source, last update, boundary, and evidence ref where available.
- Surface parked human-scope blockers without implying PR-readiness, runtime readiness, or autonomy.
- Missing services, missing scanner artifacts, and rejected projections must render as unavailable/degraded states.
- Mobile and desktop views must remain usable and must receive visual evidence before final approval.

## Required UI Vocabulary

The implementation must pin action language so the screen says exactly what
happens.

### `/ops/control`

Allowed action verbs:

- `Request`
- `Queue`
- `Draft`
- `Propose`
- `Review`

Forbidden action verbs in button labels, form legends, HTMX action names,
confirmation copy, or prominent status text:

- `Invoke`
- `Set`
- `Execute`
- `Start`
- `Launch`
- `Deploy`
- `Write`
- `Trigger`
- `Approve`
- `Merge`

Required control labels:

| Intent | Required copy pattern |
| --- | --- |
| Role or agent model policy | `Request model change` |
| Budget policy | `Propose budget change` |
| Council agenda | `Draft agenda update` |
| Ad-hoc Council meeting | `Queue Council Meeting request` |
| Civilization evolution action | `Propose evolution action` |

Queued item state labels must use one of:

- `Queued`
- `Pending human approval`
- `Blocked: protected action required`
- `Rejected`

The page root must include a machine-readable boundary marker:

`data-authority="queue-intent-only"`

### `/factory`

The upload affordance must not say `Create FactoryOrder`, `Submit FactoryOrder`,
or equivalent. Required copy pattern:

`Submit artifact for governed FactoryOrder review`

Post-upload confirmation copy must say:

`Artifact submitted. FactoryOrder conversion requires separate governed review.`

The Factory-in-Motion display must visually separate:

- `Submitted artifact`
- `FactoryOrder candidate`
- `Confirmed FactoryOrder`

The conversion actor must be identified as the governed review process, not the
human submitter and not the Site upload form.

The page root must include:

`data-factory-boundary="artifact-intake-only"`

## Consolidation And Evidence Requirements

- `/ops` must consolidate the current flat surface list into the three primary
  paths. Existing detailed surfaces stay visible only as secondary drilldowns.
  The implementation must not increase the primary hub card count.
- Projection or fixture data must never render as a green success state. It must
  render as `projection-only`, `fixture`, `stale`, or `unavailable` in text.
- `/factory` is not a read-only ops route and must not be added to
  `RegisterReadOnlyOps`. No-DB mode must render upload as disabled or
  unavailable.
- Visual evidence must include desktop width `1440px` and mobile width `390px`
  screenshots for `/ops`, `/ops/observation`, `/ops/control`, and `/factory`.

## Protected Action Boundary

Forbidden by this packet and by `site#195`:

- no protected action execution
- no Hive runtime start or wake
- no EventGraph production writes
- no Work mutation
- no service restart or deploy
- no settings, secrets, repo permission, or branch protection changes
- no PR merge by an autonomous agent
- no Test 001 GREEN or closure
- no docs#172 closure
- no residual-risk closure
- no value allocation
- no autonomy increase

Control screen actions must be phrased and implemented as queue/request/draft/review intent only. They must not imply execution.

## CFADA Questions

1. Does this design scope correctly separate Site display/control-intent UI from EventGraph truth, Hive orchestration, Work semantics, and Docs governance?
2. Does the Control screen scope risk implying protected action authority even if controls only queue intent?
3. Does the proposed `/factory` non-admin flow incorrectly imply that uploaded Markdown becomes a FactoryOrder without a governed conversion step?
4. Does the Observation design avoid fake green lights and preserve current/stale/unavailable/projection-only states?
5. Does the design materially simplify operator usage compared with the current dense evidence route index?
6. Are the issue labels and `cc:protected-action` posture consistent with the protected-action-adjacent UI scope?
7. Is the proposed implementation ready for CFAR after code, validation, and visual evidence, or is there a design blocker?

## Expected CFADA Result

Return a Markdown verdict with:

- packet identifier
- source-of-intent records used
- canonical docs/code records used
- claims audited
- blocker count
- residual risks left open
- explicit non-authorizations
- readiness for implementation and later CFAR
