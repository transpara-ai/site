# Visual Regression Checks

The operator UI surfaces should be visually checked after changes that affect
`/ops/*` or `/app/{space}/refinery`.

Canonical local targets on Nucbuntu:

| Surface | URL |
| --- | --- |
| Operations shell | `http://127.0.0.1:8201/ops?profile=transpara` |
| Work summary | `http://127.0.0.1:8201/ops/work?profile=transpara` |
| Telemetry summary | `http://127.0.0.1:8201/ops/telemetry?profile=transpara` |
| Hive summary | `http://127.0.0.1:8201/ops/hive?profile=transpara` |
| Refinery | `http://127.0.0.1:8201/app/journey-test/refinery?profile=transpara` |

Minimum checks:

- The page loads without console errors that block rendering.
- Navigation preserves the active `profile`.
- `/ops/work` and `/ops/telemetry` do not expose legacy Work dashboard links.
- The refinery board remains left-to-right: Inbox, Refining, Review, Ready, Done.
- Refinery execution filters for All, Building, Blocked, and Assigned are visible.
- Clicking a refinery card opens the right-hand detail panel instead of navigating.
