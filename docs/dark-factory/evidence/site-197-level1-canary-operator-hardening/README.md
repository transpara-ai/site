# Site 197 Level 1 Canary Operator Hardening Evidence

Generated: 2026-06-30

This packet records visual evidence for the Site-only Level 1 Dark Factory canary operator display hardening.

## Scope

- `/ops/observation` renders a compact Level 1 canary panel.
- The panel shows discovered, PR-ready, parked, human-action, and protected-action counts.
- The panel renders issue rows for `transpara-ai/docs#226`, `transpara-ai/operation#45`, and `transpara-ai/operation#26`.
- `/ops/control` remains queue-intent-only and display/intent scoped.
- Long Civilization source-reference lists are summarized in the operator view while full refs remain available to model/test consumers.

## Live Local State Used For Capture

- `site.service`: active
- `hive-ops-api.service`: active
- `hive.service`: inactive
- `/ops/observation` canary summary: `3 issue(s) discovered: 0 PR-ready, 3 parked, 3 awaiting human action.`
- Boundary: display-only Site operator shell; no Hive wake, issue closure, PR merge, deploy, Test 001 GREEN, value allocation, or autonomy increase.

## Capture Commands

```sh
npx --yes playwright screenshot --full-page --viewport-size=1440,1200 http://127.0.0.1/ops/observation docs/dark-factory/evidence/site-197-level1-canary-operator-hardening/observation-canary-desktop.png
npx --yes playwright screenshot --full-page --viewport-size=390,844 http://127.0.0.1/ops/observation docs/dark-factory/evidence/site-197-level1-canary-operator-hardening/observation-canary-mobile.png
npx --yes playwright screenshot --full-page --viewport-size=1440,1200 http://127.0.0.1/ops/control docs/dark-factory/evidence/site-197-level1-canary-operator-hardening/control-desktop.png
npx --yes playwright screenshot --full-page --viewport-size=390,844 http://127.0.0.1/ops/control docs/dark-factory/evidence/site-197-level1-canary-operator-hardening/control-mobile.png
```

## Files

- `observation-canary-desktop.png`
- `observation-canary-mobile.png`
- `control-desktop.png`
- `control-mobile.png`

## Validation

```sh
git diff --check HEAD~1..HEAD
go test ./...
```

The evidence is visual/operator evidence only. It is not production proof, deployed public proof, runtime authorization, issue closure authority, merge authority, or Test 001 GREEN evidence.
