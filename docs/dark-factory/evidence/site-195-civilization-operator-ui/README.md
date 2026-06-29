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

- Local server: `PORT=53210 ./site`
- Store mode: no `DATABASE_URL`; screens render degraded/unavailable states
- Desktop viewport: `1440x1200`
- Mobile viewport: `390x1200`
- Browser: cached Playwright Chromium
- CFADA artifact: `.adversarial-design/20260629T120717Z-site195-cfada/cfada.result.md`
- CFADA verdict: `0 blockers; approved for implementation`
- Claude skill status: `frontend-design:frontend-design` loaded; `simplify` failed to load and was applied from principles

## Screenshots

| Route | Desktop | Mobile |
| --- | --- | --- |
| `/ops` | `ops-desktop.png` | `ops-mobile.png` |
| `/ops/observation` | `observation-desktop.png` | `observation-mobile.png` |
| `/ops/control` | `control-desktop.png` | `control-mobile.png` |
| `/factory` | `factory-desktop.png` | `factory-mobile.png` |

## Boundaries Checked

- `/ops` presents three primary paths instead of a flat route wall.
- `/ops/observation` shows unavailable/projection-only state explicitly.
- `/ops/control` renders queue-intent-only controls and disabled no-DB state.
- `/factory` renders artifact-intake-only state and does not claim Markdown
  upload creates a FactoryOrder.
