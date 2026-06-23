# INC-001 Site Missing-Render Finding

## Purpose

This packet records the Site-side user-facing evidence posture for the
Test 001 cross-repo runtime-doctrine drift tabletop tracked by
`transpara-ai/operation`.

It is a missing-render finding, not a correction, deployment record, runtime
observation, or authority artifact.

## Finding

```text
finding_id: inc-001-site-missing-render-2026-06-18
incident: INC-001 / Test 001 Cross-Repo Runtime-Doctrine Drift Tabletop
surface_repo: transpara-ai/site
surface_status: MISSING_RENDER_ACCEPTED
surface_status_meaning: missing rendered or deployed evidence recorded, not authority signoff
source_commit: 8579a015f662ed76eebdfbb3833ccf61c2c1cdbc
correction_type: NO_CHANGE
human_authorization_required: no
human_authorization_evidence: none
```

Site has source-defined presentation routes associated with operationally
material surfaces for the simulated incident classes, but this packet does not
cite a deployed URL, browser screenshot, rendered capture, or live feeder
response for INC-001. The absence is explicit evidence, not a silent pass.

The `human_authorization_required` value above is scoped only to recording this
missing-render finding. It does not grant authority to publish a correction,
change public posture, alter runtime behavior, or close INC-001 as `GREEN`.

## Surfaces Reviewed

| Surface | Class | Route | Source anchors at `source_commit` | Finding |
| --- | --- | --- | --- | --- |
| Public Hive live-build page | `PUBLIC_POSTURE` | `/hive`, `/hive/feed`, `/hive/stats`, `/hive/status` | `graph/handlers.go:355-359`; `graph/hive.templ:9-31` | Source route and template evidence exist; no incident-specific deployed URL, screenshot, or rendered capture is cited. |
| Hive operator summary | `OPERATOR_POSTURE` | `/ops/hive`, `/ops/hive/intake`, `/ops/hive/runs`, `/ops/hive/agents`, `/ops/hive/resources` | `graph/handlers.go:367-374`; `graph/ops.go:1460-1488`; `graph/ops.templ:605-633` | Source route and operator shell evidence exist; no live Hive projection output or rendered incident capture is cited. |
| Observatory transparency view | `OPERATOR_POSTURE`, `VISUALIZATION` | `/ops/observatory`, `/ops/observatory/events` | `graph/handlers.go:365-366`; `graph/observatory.go:17-25`; `graph/ops.go:1403-1410` | Source route and read-only projection evidence exist; no feeder response, deployed URL, screenshot, or rendered incident capture is cited. |
| Evidence projection | `OPERATOR_POSTURE` | `/ops/evidence` | `graph/handlers.go:375`; `graph/ops.go:1421-1428` | Source route and read-only evidence projection evidence exist; no configured projection output or rendered incident capture is cited. |
| Gate E decision boundary | `OPERATOR_POSTURE` | `/ops/decision`, `/ops/approvals` | `graph/handlers.go:376-378`; `graph/ops.go:1430-1437`; `graph/ops.go:2330-2347` | Source route and source-declared effect-none decision-surface evidence exist; no incident-specific authority request, decision, or rendered capture is cited. |

The source anchors above were spot-checked in the local checkout before PR
review. They are commit-bounded source-location hints, not rendered output,
runtime observation, deployment evidence, or a machine-validated proof.

## Boundaries

This packet does not prove:

- that the INC-001 simulated contradiction affected any live visitor
- that Site was deployed at `source_commit`
- that any route rendered correctly in a browser for INC-001
- that Hive runtime behavior was observed
- that EventGraph records exist
- that a public correction was required or completed
- that Site grants policy, runtime, production, or human authorization
- that Test 001 is `GREEN`
- that `wiki`, OpenBrain/`OB1`, active roster, Hive runtime, or
  EventGraph-record gaps are resolved

`docs` remains the canonical doctrine source when doctrine conflicts with Site.
`hive` remains the runtime source for Hive behavior. `eventgraph` remains the
incident evidence source for event records. Site only owns the public and
operator-facing presentation surfaces listed above.

## Validation Plan

The owning repo validation for this documentation packet is:

```bash
make verify
```

The packet should be cited by `operation` only after the PR that
adds it has passed local validation, GitHub CI, exact-head adversarial review,
and has been merged to `origin/main`.
