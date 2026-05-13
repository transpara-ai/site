# Site v3.9 Minimal Evidence UI Recon

Date: 2026-05-13
Scope: design-only recon for Dark Factory v3.9 Stage 7 minimal Site evidence UI.

## Sources

- `/home/transpara/transpara-ai/repos/docs/dark-factory/v3.9/02-kernel-schema-and-state-v3.9.md`
- `/home/transpara/transpara-ai/repos/docs/dark-factory/v3.9/05-verification-audit-risk-eval-v3.9.md`
- `/home/transpara/transpara-ai/repos/docs/dark-factory/v3.9/08-implementation-workflow-checklist-v3.9.md`
- Current Site routing, graph, operator UI, store, refinery, and ops code.
- `transpara-ai/docs#44`: Dark Factory v3.9 Base Slice 0 implementation tracker, including Lane D guidance that Site work is design-only until EventGraph and Work schemas exist.

## 1. Current Site Operator/UI Model

Site is currently a Go `net/http` application with Templ-rendered views and Tailwind classes. Public content, reference pages, auth, `/app/*` graph UI, `/hive*` public runtime status, and `/ops/*` operator shell are registered from `cmd/site/main.go` and `graph/handlers.go`.

The active operator model is split across two surfaces:

- `/app/{slug}/*`: Site-owned graph workspace UI backed by Site Postgres tables for `Space`, `Node`, `Op`, users, notifications, reactions, dependencies, votes, and hive mirrors. This is operational collaboration UI, not v3.9 EventGraph truth.
- `/ops/*`: authenticated operator shell that reads external summaries from Work and Hive-like projection endpoints, then renders compact evidence/status summaries.

The current Site graph model is intentionally broad and simple:

- `Node` is the universal unit for task, post, thread, comment, claim, proposal, project, goal, role, team, policy, document, question, and council.
- `Op` records grammar operations such as `intend`, `decompose`, `assign`, `complete`, `progress`, `review`, `challenge`, `verify`, `retract`, governance operations, and other user or agent actions.
- The current task lifecycle is `open`, `active`, `review`, `blocked`, `done`, `closed`, with child/dependency-derived blocking.
- This differs from v3.9 Work `Task` states: `created`, `ready`, `running`, `blocked`, `failed`, `repair_required`, `repair_running`, `repaired`, `verification_running`, `verified`, `certified`, `rejected`, `superseded`, and `policy_blocked`.

The current operator UI already has a useful boundary: Site can render operator summaries without owning the underlying execution model. That boundary should be preserved for v3.9.

## 2. Current Relevant Routes/Views/Stores

Current route families:

- `/app`: authenticated dashboard and space index.
- `/app/{slug}` and `/app/{slug}/board`: workspace landing and task board.
- `/app/{slug}/refinery`: simplified intake/design review surface over Site tasks.
- `/api/refinery/{slug}/projection`: JSON projection from Site tasks into the current refinery state model.
- `/app/{slug}/activity`: recent Site `Op` activity.
- `/app/{slug}/knowledge`: documents, questions, and claims.
- `/app/{slug}/governance`: proposals, votes, and delegation.
- `/app/{slug}/node/{id}`: node detail with children, dependencies, dependents, ops, comments, progress, review, evidence-style operations, and state controls.
- `/app/{slug}/op`: grammar operation dispatcher and write path.
- `/hive`, `/hive/feed`, `/hive/stats`, `/hive/status`: public Hive dashboard and HTMX partials.
- `/ops`, `/ops/work`, `/ops/telemetry`, `/ops/hive`, `/ops/refinery`: authenticated operator shell.
- `/api/hive/diagnostic`, `/api/hive/escalation`, `/api/hive/mirror`, `/api/hive/site-ops`: Site-owned Hive integration/mirror endpoints.

Relevant view/store code:

- `graph/store.go`: Site Postgres model for spaces, nodes, ops, dependencies, governance, hive diagnostics, and search.
- `graph/handlers.go`: route registration and main workspace handlers.
- `graph/views.templ`: main workspace UI including dashboard, board, node detail, activity, knowledge, governance, and Hive partials.
- `graph/refinery.go` and `graph/refinery.templ`: current Site refinery projection and view.
- `graph/ops.go` and `graph/ops.templ`: current operator shell and Work/Hive/telemetry projection rendering.

Relevant current projection patterns:

- Work summary reads `WORK_API_BASE_URL` or `WORK_UI_BASE_URL` and expects `/tasks`, `/phase-gates`, `/telemetry/overview`, and `/telemetry/pipeline/report`.
- Hive authority summary reads `HIVE_OPS_API_BASE_URL` and expects `/api/hive/operator-projection`.
- Refinery projection is currently derived locally from Site `Node` tasks.

## 3. Minimal v3.9 Evidence UI Requirements

The minimal evidence UI should be an operator evidence console for v3.9 Base Slice 0. It should expose enough trace, gate, release, failure, repair, and audit state for an operator to determine whether a FactoryOrder is certifiable or blocked. It should not create, mutate, repair, certify, reject, execute, or infer kernel truth.

### FactoryOrder Timeline

Show a chronological timeline for one `FactoryOrder`:

- status transitions: `draft`, `interpreted`, `accepted`, `decomposed`, `in_production`, `verification`, `certified`, `rejected`, `superseded`;
- source intent ref and hash;
- risk class and release policy;
- linked requirements, tasks, runtime invocations, artifacts, gate results, release candidates, certifications/rejections, failures, repair attempts, and audit reports;
- event IDs, actor IDs, timestamps, and immutable refs for each row.

The timeline should prefer event rows from EventGraph, with Work task state projected into the timeline only when linked back to EventGraph records.

### Requirement -> AcceptanceCriterion -> Artifact Trace

Show a trace matrix from each accepted `Requirement` to its `AcceptanceCriterion` records and produced evidence:

- requirement text, source, status, risk class;
- criterion text, verification method, required evidence type, owner role, waiver policy;
- linked `TestCase`, `TestRun`, `GateResult`, `Artifact`, `CodeChange`, and `ActorInvocation`;
- missing links called out per row.

The view should make explicit that unrelated tests do not satisfy an acceptance criterion.

### RuntimeEnvelope / RuntimeResult View

Show invocation details for each `ActorInvocation`:

- `RuntimeEnvelope`: adapter ID/version, FactoryRuntimeVersion ref, task ID, actor ID, authority decision ref, allowed and denied files, allowed and denied commands, network policy, secrets policy, working directory, timeout, resource limits, expected outputs, output contract, trace required paths, validation plan, envelope hash.
- `RuntimeResult`: start/complete times, exit status, stdout/stderr refs, artifact refs, changed files, command log, network access log, secret access log, policy decision refs, error summary, post-run validation refs.

The operator should be able to inspect policy-blocked and timed-out results without treating them as successful work.

### Gate Evidence Table

Show product gates for a release candidate:

- v3.9 minimum gates: `unit_tests`, `integration_tests`, `e2e_tests`, `build`, `migration_check`, `secret_scan`, `dependency_vulnerability_scan`, `dependency_license_scan`, `sast`, `auth_flow_security_check`, `configuration_security_check`, optional `container_or_build_artifact_scan`, `trace_completeness`, `factory_runtime_bom_check`, and `audit_report_check`;
- gate status: `pass`, `fail`, `error`, `skipped`, or `waived`;
- evidence refs, waiver ref, failure refs, and certification-blocking status;
- clear grouping for required, waived, failed, and missing gates.

### ReleaseCandidate Evidence View

Show one `ReleaseCandidate` with:

- FactoryOrder link;
- FactoryRuntimeVersion link and BOM summary;
- artifact refs;
- status: `draft`, `verification`, `certified`, `rejected`, or `superseded`;
- trace completeness result;
- product gate table;
- certification or rejection record;
- audit report status.

The FactoryRuntimeVersion BOM section should summarize capabilities, runtimes, models, policies, environment, and repos, with drift/failure refs when available.

### Certification / Rejection View

v3.9 defines `Certification` with status `certified` or `rejected`; no separate `Rejection` node schema was found in the required v3.9 schema doc. Site should therefore render "Certification / Rejection" as the terminal decision view over `Certification.status` and `ReleaseCandidate.status`.

Show:

- release candidate ID;
- certifier actor ID;
- decision status;
- reason;
- evidence refs;
- linked TraceCompletenessGate result;
- linked GateResult rows;
- blocking missing paths or unresolved high/critical failures, if rejected.

### AuditReport View

Show the finalized or in-progress `AuditReport` for a FactoryOrder or ReleaseCandidate:

- target type and ID;
- status: `complete`, `incomplete`, or `failed`;
- trace score;
- missing links;
- answers to the v3.9 audit questions: request, actor, requirements, changes, runtime, factory version, tests/gates, failures, repairs, waivers, protected approvals, certification/rejection reason, memory/knowledge influence, and rollback/restoration path.

For minimal Stage 7, capability audit questions can be omitted unless the target type is capability-related.

### Failure And Repair Timeline

Show failures and repairs as a timeline:

- `Failure`: class, severity, summary, status, FactoryOrder/task/gate/test refs;
- `RepairAttempt`: failure ID, task ID, actor invocation ID, status;
- linked rerun `TestRun` or `GateResult` evidence;
- unresolved high/critical failures highlighted as certification blockers;
- waiver and accepted-risk records shown but not hidden.

### Missing Provenance Report

Show a dedicated missing-provenance report using TraceCompletenessGate output:

- target type and target ID;
- score;
- required paths total and present;
- missing nodes;
- missing edges;
- missing paths;
- certification-blocking boolean;
- summary;
- suggested owning projection/source for each missing item when the projection provides enough metadata.

This report must be generated from EventGraph/Work projections and must not be inferred by Site from free text.

## 4. Projection Dependencies On EventGraph/Work

Stage 7 depends on typed projections, not direct Site schema invention. Required projection families:

- `factory_order_projection`: FactoryOrder summary, lifecycle events, linked requirements, tasks, releases, failures, repairs, gates, and audits.
- `requirement_trace_projection`: Requirement -> AcceptanceCriterion -> TestCase/TestRun/GateResult/Artifact/CodeChange/ActorInvocation paths.
- `runtime_invocation_projection`: ActorInvocation -> RuntimeEnvelope -> RuntimeResult, with policy, command, file, network, secret, artifact, and validation refs.
- `gate_evidence_projection`: product gate status rows and evidence refs.
- `release_trace_projection`: ReleaseCandidate, FactoryRuntimeVersion/BOM, Certification or rejection decision, GateResult refs, TraceCompletenessGate result, AuditReport.
- `failure_repair_projection`: Failure and RepairAttempt paths to TestRun/GateResult evidence.
- `missing_provenance_projection`: TraceCompletenessGate result, missing nodes, missing edges, missing paths, and certification-blocking status.
- `audit_report_projection`: AuditReport fields and structured answers to required audit questions.

EventGraph owns Tier 0 schema, append-only records, required paths, trace source data, authority/evidence/release/capability records, and projection rebuildability. Work owns Task lifecycle, DAG/readiness/blocking/repair projections, and FactoryOrder-to-Task operational flow. Site consumes the projections and renders them.

Minimum projection contracts should include:

- stable IDs for every row;
- source record type and source record ID;
- event IDs and timestamps;
- actor IDs;
- immutable refs or hashes for evidence;
- target IDs for drill-down;
- status enums matching v3.9;
- an explicit `projection_generated_at`;
- an explicit `projection_source`;
- a machine-readable `projection_errors` list.

## 5. Proposed Route/View Structure

Add no routes until Stage 1/3/5/6 projections exist. Proposed eventual route structure:

- `/ops/evidence`: overview of recent FactoryOrders, release candidates, certification blockers, and missing provenance.
- `/ops/evidence/factory-orders/{factory_order_id}`: FactoryOrder timeline.
- `/ops/evidence/factory-orders/{factory_order_id}/trace`: Requirement -> AcceptanceCriterion -> Artifact trace.
- `/ops/evidence/factory-orders/{factory_order_id}/failures`: failure and repair timeline.
- `/ops/evidence/runtime/{actor_invocation_id}`: RuntimeEnvelope / RuntimeResult detail.
- `/ops/evidence/release-candidates/{release_candidate_id}`: release candidate evidence view.
- `/ops/evidence/release-candidates/{release_candidate_id}/gates`: gate evidence table.
- `/ops/evidence/release-candidates/{release_candidate_id}/decision`: certification/rejection decision.
- `/ops/evidence/audit-reports/{audit_report_id}`: AuditReport view.
- `/ops/evidence/missing-provenance?target_type=...&target_id=...`: missing provenance report.

Proposed view layout:

- Keep this inside the existing authenticated `/ops` shell.
- Add "Evidence" as a fifth operator surface only when projections exist.
- Use compact, table-first layouts for gates, trace rows, missing provenance, and runtime logs.
- Use timelines only where sequence matters: FactoryOrder lifecycle and failure/repair.
- Keep IDs, refs, hashes, and paths visible and copyable enough for operators, but avoid turning the page into raw JSON by default.
- Provide raw projection JSON only as an inspection affordance after the projection contract is stable.

## 6. Non-Goals

- Site does not execute work.
- Site does not become truth.
- Site does not become authority.
- Site does not certify, reject, waive, repair, plan, or mutate v3.9 records in the minimal UI.
- Site does not infer missing provenance from text.
- Site does not replace TraceCompletenessGate, AuthorityDecision, ExecutionReceipt, Certification, Rejection/terminal decision, or AuditReport records.
- No external runtime UI yet.
- No Hermes/OpenManus/OpenClaw adapter UI yet.
- No capability evolution UI in this minimal Stage 7 design, except leaving route space for later.
- No resurrection of deleted `site/refactor/solo-phase-gates` or `site/feat/refinery-hive-site-ops` branches.

## 7. Implementation Plan After Stage 1/3/5/6 Land

1. Confirm Stage 1 EventGraph Tier 0 schema and required path query APIs are merged and versioned.
2. Confirm Stage 3 Work task lifecycle and FactoryOrder/Requirement/AcceptanceCriterion/Task linkage projections are stable.
3. Confirm Stage 5 TraceCompletenessGate and FactoryRuntimeVersion/BOM projections exist.
4. Confirm Stage 6 ReleaseCandidate, Certification or rejected decision, and AuditReport records/projections exist.
5. Write a Site projection client layer for the stable API endpoints with typed structs and fixture-backed tests.
6. Add `/ops/evidence` route registration and one overview view using fixture data first.
7. Add FactoryOrder timeline and requirement trace views.
8. Add RuntimeEnvelope/RuntimeResult and gate evidence views.
9. Add ReleaseCandidate, decision, AuditReport, failure/repair, and missing provenance views.
10. Add tests for handler routing, projection error rendering, empty states, blocked/certification statuses, and enum coverage.
11. Run `templ generate` after `.templ` edits.
12. Run `make verify`; if UI behavior changed, add browser/screenshot validation against fixture projections.

## 8. Risks

- Projection schemas may drift while EventGraph/Work are still landing; implementation should wait for stable contracts.
- Site currently has its own broad Node/Op graph model, which can confuse operators if v3.9 evidence is blended with Site-local collaboration records.
- Existing `/ops/work` and `/ops/telemetry` expect legacy Work endpoints; v3.9 may require separate endpoints or versioned paths.
- Current refinery view maps Site task states into a simplified FSM; reusing it directly would hide v3.9 task states such as `policy_blocked`, `repair_required`, `repair_running`, `verification_running`, `verified`, `certified`, and `rejected`.
- If Site presents missing provenance as advice instead of gate output, it can accidentally become an authority-like interpreter.
- Raw evidence volume may make the UI noisy; minimal views should prioritize blockers, required paths, and drill-downs.
- FactoryRuntimeVersion BOM drift needs exact source hashes and refs; vague repo labels are not enough for operator trust.
- Certification and rejection terminology must stay aligned with v3.9. The schema currently models rejection through `Certification.status = rejected` and `ReleaseCandidate.status = rejected`, not a separate Rejection node.
- Public `/hive*` surfaces are unauthenticated today; v3.9 evidence views should remain under authenticated `/ops/*` unless a later policy explicitly allows public evidence.
