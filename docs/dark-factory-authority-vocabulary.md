# Dark Factory Authority Vocabulary

Date: 2026-05-08

Source of truth: `transpara-ai/docs` `dark-factory/DF-SOP-0001-authority-gated-side-effects.md`.

Site is the operator-facing surface. It must render and route authority-gated side effects using the shared vocabulary instead of UI-specific aliases.

## Authority Outcomes

```text
Autonomous
Notify
ApprovalRequired
Forbidden
```

## Protected Actions

```text
production.deploy
repo.create
repo.delete
repo.push.default_branch
repo.merge.main
repo.mutate.cross_repo
agent.spawn.persistent
agent.retire
agent.escalate_permissions
policy.change
secret.access
external_communication.company_voice
data.delete
self_modification.activate
billing.spend_above_threshold
license.change
```

## Local Alignment Notes

- Production deployment controls must use `production.deploy`.
- Secret-backed operator surfaces must use `secret.access` for explicit secret access.
- Policy or auth guardrail changes must use `policy.change` when represented as protected actions.
- Cross-repo actions displayed in `/ops/*` must use `repo.mutate.cross_repo`, not `repo.mutate_cross_repo`.
