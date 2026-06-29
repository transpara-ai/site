---
doc_id: SITE-195-CIVILIZATION-OPERATOR-UI-EVIDENCE
title: Site 195 Civilization Operator UI Visual Evidence
doc_type: evidence-index
status: draft
created: 2026-06-29
owner: Michael Saucier
primary_repo: transpara-ai/site
source_issue: transpara-ai/site#195
authority: visual evidence only; no runtime, deploy, Hive wake, EventGraph production write, Test 001 closure, residual-risk closure, value allocation, or autonomy increase
---

# Site 195 Civilization Operator UI Visual Evidence

This packet records desktop and mobile browser evidence for the Site-only
Civilization operator UI rebuild.

## Capture Context

- No-DB local server: `PORT=53210 ./site`
- No-DB store mode: no `DATABASE_URL`; screens render degraded/unavailable states
- DB-backed local server: `DATABASE_URL=postgres://site:site@localhost:5433/site?sslmode=disable PORT=53211 ./site`
- DB-backed store mode: local Docker Postgres only; no deploy, runtime start, Hive wake, EventGraph production write, or service restart
- `/ops/hive` model-selection evidence: local fixture projection at
  `HIVE_OPS_API_BASE_URL=http://127.0.0.1:53212`; fixture-only JSON, no Hive
  runtime, no external adapter, no protected action
- Desktop viewport: `1440x1200`
- Mobile viewport: `390x1200`
- Browser: cached Playwright Chromium
- CFADA artifact: `.adversarial-design/20260629T120717Z-site195-cfada/cfada.result.md`
- CFADA verdict: `0 blockers; approved for implementation`
- Claude skill status: `frontend-design:frontend-design` loaded; `simplify` failed to load and was applied from principles

## Screenshots

| Route | Mode | Desktop | Mobile |
| --- | --- | --- | --- |
| `/ops` | no DB | `ops-desktop.png` | `ops-mobile.png` |
| `/ops` | DB-backed | `ops-db-desktop.png` | `ops-db-mobile.png` |
| `/ops/observation` | no DB | `observation-desktop.png` | `observation-mobile.png` |
| `/ops/control` | no DB | `control-desktop.png` | `control-mobile.png` |
| `/ops/control` | DB-backed | `control-db-desktop.png` | `control-db-mobile.png` |
| `/factory` | no DB | `factory-desktop.png` | `factory-mobile.png` |
| `/factory` | DB-backed | `factory-db-desktop.png` | `factory-db-mobile.png` |
| `/ops/hive` | DB-backed local fixture | `hive-model-selection-db-desktop.png` | `hive-model-selection-db-mobile.png` |

## Boundaries Checked

- `/ops` presents three primary paths instead of a flat route wall.
- `/ops/observation` shows unavailable/projection-only state explicitly.
- `/ops/control` renders queue-intent-only controls and disabled no-DB state.
- `/factory` renders artifact-intake-only state and does not claim Markdown
  upload creates a FactoryOrder.
- DB-backed control and Factory artifact records stay Site-local and are
  excluded from Hive intake launch candidates by tests.
- DB-backed `/factory` requires write-authorized access before exposing
  submitted artifact rows.
- Role/agent assignment controls render as dropdown selections for model policy,
  budget policy, Council request targeting, Hive model policy, task assignment,
  Council agent selection, and API key agent identity selection.
