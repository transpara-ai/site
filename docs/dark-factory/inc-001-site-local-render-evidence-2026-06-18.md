# INC-001 Site Local Render Evidence

## Purpose

This packet records local, deterministic Site render evidence for selected
user-facing and operator-facing surfaces associated with the Test 001
cross-repo runtime-doctrine drift tabletop tracked by
`transpara-ai/civilization-operation`.

It narrows the earlier missing-render posture. It is not deployment evidence,
live feeder evidence, runtime observation, correction evidence, public
disclosure, authority signoff, or Test 001 closure.

## Finding

```text
finding_id: inc-001-site-local-render-evidence-2026-06-18
incident: INC-001 / Test 001 Cross-Repo Runtime-Doctrine Drift Tabletop
surface_repo: transpara-ai/site
surface_status: LOCAL_RENDER_EVIDENCE_RECORDED
surface_status_meaning: selected routes rendered in local HTTP tests; downstream citation still waits for validation, review, and merge
source_commit: 613261f16f67a1d1de0aa9d5966ab52c8f1d3f75
correction_type: NO_CHANGE
human_authorization_required: no
human_authorization_evidence: none
```

The `human_authorization_required` value above is scoped only to recording
local render evidence for existing Site behavior. It does not authorize a public
correction, deployment, runtime action, policy change, incident closure, or
production posture change.

The `source_commit` is the base commit containing the unchanged route handlers
and templates under test. The PR that adds this packet contains only test and
documentation changes. `LOCAL_RENDER_EVIDENCE_RECORDED` is the Site-side
evidence posture recorded by this packet, not a claim that the PR is accepted,
merged, deployed, or ready for downstream citation.

## Evidence Collected

Validation command:

```bash
go test -count=1 ./graph -run 'TestGetHive_PublicNoAuth|TestGetHive_ContainsHiveFeed|TestGetHiveStatus_Partial|TestGetHiveStats_Partial|TestGetHiveFeed_PublicNoAuth|TestHandleOpsHiveStaticChildRoutesRender|TestHandleOpsHiveRendersRuntimeEvidenceBoundariesWithoutRun|TestHandleOpsHiveRendersArtifactsGraphAndEventInspector|TestHandleOpsObservatoryRendersReadOnlyProjection|TestHandleOpsEvidenceRendersReadOnlyProjection|TestHandleOpsDecisionRendersNonExecutingBoundary'
```

Observed result:

```text
ok  	github.com/transpara-ai/site/graph	0.018s
```

Additional route-render test added by this packet:

```text
TestHandleOpsObservatoryRendersReadOnlyProjection
```

That test renders `GET /ops/observatory` through the registered Site route with
a local deterministic Work telemetry feeder. It asserts the route returns
`200 OK`, shows the Observatory, civilization vitals, live event pulse,
authority projection, and the read-only/EventGraph-truth boundary, and does not render
mutation-control markers inside the Observatory surface.

## Surface Evidence

| Surface | Route | Local render evidence | Remaining limitation |
| --- | --- | --- | --- |
| Public Hive live-build page | `/hive`, `/hive/feed`, `/hive/stats`, `/hive/status` | `TestGetHive_PublicNoAuth`, `TestGetHive_ContainsHiveFeed`, `TestGetHiveFeed_PublicNoAuth`, `TestGetHiveStats_Partial`, and `TestGetHiveStatus_Partial` render the public page and partials through HTTP handlers. | No deployed URL, live visitor evidence, or actual Hive runtime observation is cited. |
| Hive operator summary | `/ops/hive` plus static child routes | `TestHandleOpsHiveStaticChildRoutesRender` renders the operator shell and child routes; `TestHandleOpsHiveRendersRuntimeEvidenceBoundariesWithoutRun` renders explicit `not_observed` runtime boundaries; `TestHandleOpsHiveRendersArtifactsGraphAndEventInspector` renders artifact, causal graph, and event-inspector projection content from a local deterministic feeder. | The feeder is test-local. It is not production Hive runtime evidence, deployment evidence, or EventGraph chain verification. |
| Observatory transparency view | `/ops/observatory` | `TestHandleOpsObservatoryRendersReadOnlyProjection` renders the Observatory route through registered handlers with a local empty-state telemetry feeder and checks the read-only/EventGraph-truth boundary. | The feeder is test-local and empty-state only. No populated telemetry render, deployed Observatory page, live SSE stream, or production telemetry source is cited. |
| Evidence projection | `/ops/evidence` | `TestHandleOpsEvidenceRendersReadOnlyProjection` renders the read-only evidence projection from a local deterministic feeder and asserts proof-of-work, tests, CI, review, security-scan, screenshot/walkthrough, known-failure, and operator-decision fields appear without mutation controls. | The fixture is local and synthetic. It is not a real FactoryOrder, live EventGraph record, deployment, or production evidence packet. |
| Gate E decision boundary | `/ops/decision` | `TestHandleOpsDecisionRendersNonExecutingBoundary` renders the non-executing decision surface and asserts the governed boundary, effect-none posture, and residual-risk fields. | This proves local rendering of the boundary, not a real authority decision or protected side effect. |

## Boundaries

This packet does not prove:

- that Site was deployed at `source_commit`
- that any route was reached by a live visitor
- that Hive runtime behavior was observed
- that EventGraph records or chain verification exist for INC-001
- that a public correction was required or completed
- that an external feeder response was live or production-backed
- that Site grants policy, runtime, production, or human authorization
- that `civilization-wiki`, OpenBrain/`OB1`, active roster, Hive runtime, or
  EventGraph-record gaps are resolved
- that Test 001 is `GREEN`

`docs` remains the canonical doctrine source when doctrine conflicts with Site.
`hive` remains the runtime source for Hive behavior. `eventgraph` remains the
incident evidence source for event records. This packet only records selected
local Site render evidence.

## Relationship To The Missing-Render Finding

[inc-001-site-missing-render-finding-2026-06-18.md](inc-001-site-missing-render-finding-2026-06-18.md)
remains valid for deployment, live feeder, live visitor, correction, runtime,
and EventGraph evidence gaps. This packet narrows only the local render portion
for selected existing Site surfaces.

## Validation Plan

The owning repo validation for this evidence packet is:

```bash
make verify
```

The packet should be cited by `civilization-operation` only after the PR that
adds it has passed local validation, GitHub CI, exact-head adversarial review,
and has been merged to `origin/main`.
