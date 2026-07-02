# Console UX Polish — Design Packet

- **doc_id:** SITE-CONSOLE-UX-POLISH-DESIGN-001
- **version:** v0.5.0 (CFADA rounds 1-3 + CFAR round 1 resolved)
- **status:** CFADA-clean pending round-4 confirmation; building under TDD
- **issue:** https://github.com/transpara-ai/site/issues/202
- **base:** site main @ b68e214
- **scope:** `graph/console*` only (+ committed generated `*_templ.go`); zero backend changes; console stays a read-only projection surface.

## 1. Problem

The four `/console` tabs (Health wall, Kanban, Intake, Config) are functionally honest but presentationally raw:

1. Intake blocker cards state the blocker (`needs_human_scope — human must clarify scope…`) but leave the operator to reconstruct the label protocol and the parked-run-is-terminal gotcha from memory.
2. Empty states shrug (`No orders.`) instead of teaching the flow.
3. Visual language drifted per tab (badge/notice/card styles inconsistent, lens nav is plain links).

## 2. Decisions

### D1 — Actionability lives in the Intake drawer; the card gets a hint, not a command

Cards stay scannable (board density is a feature). A blocked card shows its existing amber blocker summary plus a small `unblock available` hint **only when** an unblock plan exists (see D2 gate). The drawer gains an **Unblock** section rendering: the exact `gh issue edit` command, the label chips, the terminal-run explanation, and the rescan command.

### D2 — Unblock plan is a pure, fail-closed derivation from projected data

Two pure functions (no I/O) with **exact signatures** (CFADA2-1: the single-arg form could not see sibling blockers; the index is a required parameter, not an option):

```go
func consoleIssueScanRunBlockerTypes(board OpsCivilizationIssueScanKanban) map[string][]string // skips blank RunIDs
func consoleIssueScanUnblockPlan(card OpsCivilizationIssueScanKanbanCard, runBlockerTypes map[string][]string) (consoleUnblockPlan, bool)
type consoleUnblockPlan struct{ ScopeCommand, ProtectedCommand, RescanCommand string }
```

Precondition (CFADA2-3): `strings.TrimSpace(card.RunID) != ""` or `ok=false`; the index builder likewise skips blank RunIDs so unrelated blank-run cards can never alias as siblings under the empty key.

- **Gate (allowlist; every condition must hold or return `ok=false`):**
  - the card has ≥1 blocker AND **every** blocker type projected for the card's `RunID` anywhere on the board (across all sibling cards/columns, via a runID→blocker-types index built during the board walk) ∈ {`not_pr_ready`, `needs_human_scope`, `protected_action`} (IADA-1 + CFADA1-1: sibling cards of the same run can carry non-label blockers — e.g. a `protected_action` card whose run also has a `stale_target` blocker on another card must NOT offer a command; relabeling would not unblock the run)
  - issue ref selected with the **same preference the existing card helpers use** (target issue preferred, else selected issue), all fields (repo, number, labels) taken from that one ref — never mixed (IADA-2)
  - ref `Repo` matches `^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*$`
  - ref `Number > 0`
  - **evidence binding per blocker type (CFADA1-2)** — each blocker type on the run must be corroborated by the label evidence on the chosen ref, else `ok=false`:
    - `needs_human_scope` → requires labels ∩ {`cc:needs-human-scope`, `cc:pr-deferred`} ≠ ∅ (hive maps both to this blocker, `issue_scan_parking.go:190-201`)
    - `protected_action` → requires `cc:protected-action` present
    - `not_pr_ready` → requires `cc:pr-ready` absent
    - blocker projected without its corroborating label evidence = projection/label mismatch → no command (fail closed; the projected `RequiredAction` still renders)
- **Label comparison** normalizes with `strings.ToLower(strings.TrimSpace(l))` — identical to hive's `issueScanLabelSet` (`pkg/hive/issue_scan_parking.go:367-376`) and to the admission gate `IssueScanCandidatePRReady` (`pkg/hive/issue_intake.go:328-349`: requires `cc:pr-ready` present AND none of the three deny labels), so the site's view of the gate matches the gate (IADA-3, CFADA1-adv2).
- **Command construction — split by authority class (CFADA2-2):** hive parks first-match (`issue_scan_parking.go:185-207`: stale → needs-human-scope → pr-deferred → protected-action → duplicate), so a projected blocker type witnesses only the *first* matching deny condition; the full boundary state lives in the observed labels. Three policies were rejected: remove-everything-in-one-command over-authorizes (clears a protected boundary without projected evidence — codex's round-2 scenario); remove-only-corroborated emits commands that provably cannot admit the issue (dishonest); refuse-on-any-uncorroborated-label denies every live card (label-superset is the normal first-match state). Resolution — two commands, granular authority:
  - **Scope command** (rendered when its change set is non-empty): `gh issue edit <number> --repo <repo>` + `--remove-label cc:pr-deferred` / `--remove-label cc:needs-human-scope` per presence (fixed order) + `--add-label cc:pr-ready` iff absent.
  - **Protected-action command** (rendered iff `cc:protected-action` present, always as a separate block): `gh issue edit <number> --repo <repo> --remove-label cc:protected-action`, preceded by mandatory copy: *"This second command clears a protected-action boundary — run it only if you authorize the protected action."* The human is the sole owner of that boundary; the separate block makes exercising it a deliberate, informed act rather than a side-effect.
  - `ok=false` when both change sets are empty. Labels come from the fixed constant set, never echoed from projection free-text.
- **Rescan command:** `hive factory scan-issues --human YOUR_NAME --repo <repo>` (repo passes the same gate). Verified (`cmd/hive/factory_scan_issues.go:32`): `--dispatch` defaults false and is the extra step that turns the queued run into a FactoryOrder task — copy presents the bare scan as the re-queue step and mentions `--dispatch` as the optional immediate-launch flag (IADA-4). `YOUR_NAME` is a deliberately shell-inert placeholder (CFADA1-4: angle-bracket placeholders are POSIX redirection metacharacters inside a copyable block): operator identity is not projected, and fabricating one would violate the honesty contract. A one-line note says to replace it and mentions the daemon alternative (`hive factory daemon --issue-scan-interval 15m`).
- **No fall-through emits a command.** `stale_target`, `duplicate_chain`, empty, and any unknown/future blocker type render the projected `RequiredAction` text only. There is no `default:` that constructs a command.

Rationale: hive owns label semantics and never mutates labels itself (`pkg/hive/issue_intake.go:18-21`); the human performs the labeling act. Rendering the command is presentation of governance, not a write path.

### D3 — Terminal-run explanation is mandatory copy wherever a command renders

Exact copy (drawer):

> **A parked run is terminal.** Fixing labels does not resume this run — the parked event is final and idempotent. After relabeling, a fresh scan cycle re-queues the issue: run the command below, or wait for the scan daemon's next interval.

### D4 — Purposeful empty states (rendered only when freshness ≠ unavailable)

| Tab | Today | New copy (exact) |
|---|---|---|
| Kanban | `No orders.` | `No factory orders yet. Work enters the factory at Intake: issues labeled cc:pr-ready are admitted by the issue scanner and become orders tracked here.` + anchor `Go to Intake →` (`/console/intake`) |
| Intake | `No issue-scan runs projected.` | `No issue-scan runs projected. Runs appear after a scan cycle: one-shot hive factory scan-issues, or the scan daemon on its interval. Issues need the cc:pr-ready label to be admitted.` |
| Health | `No active agents reported.` | `No active agents. Agents report here while a civilization run is in flight; between runs an empty roster is the honest state.` |
| Config (assignments) | `No role assignments projected.` | unchanged string + trailing context line: `Role routing appears once the hive projects its model-selection policy.` |
| Config (models) | `No catalog models projected.` | unchanged string + trailing context line: `The catalog is hive-owned (HIVE_OPS_CATALOG) and hot-reloaded.` |

Unavailable states are untouched: no links, no teaching copy layered onto an outage (a link to Intake under "Kanban unavailable" would imply the system is navigable-healthy when it is not — keep the honest notice alone).

### D5 — Visual pass, token-system only

No new dependencies, no new JS beyond existing HTMX patterns, Tailwind v4 semantic tokens only.

- **Shell/tabs:** tab row gets consistent spacing + `aria-current="page"` on active tab; keep border-bottom active style.
- **Freshness badges:** unify into one component style — dot + label (`live`/`stale`/`partial`/`unavailable`), colors as today (emerald/amber/amber/muted); timestamps in `font-mono text-[11px]`.
- **Kanban:** lens nav becomes a segmented control (single bordered group, active segment `bg-elevated text-warm`, inactive `text-warm-muted`); column headers get count pills (`bg-elevated rounded-full px-2`); cards keep fields, tighten hierarchy (title `text-warm`, meta row `text-[11px] text-warm-muted`).
- **Intake:** cards get label chips row (deny labels amber-tinted chip `border-amber-500/40 text-amber-300`, `cc:pr-ready` emerald chip, other `cc:*` labels neutral `border-edge text-warm-muted`; chips only for labels matching `^cc:[a-z0-9-]+$`, others fall back to existing plain text — allowlist, escaped either way; chips read from the **same chosen issue ref as the command** (CFADA1-adv1: card-level `Labels` can differ from the ref's labels)); blocked cards show `unblock available` hint per D1 iff the plan gate passes; drawer gets sectioned layout (Issue / State / Possession / **Unblock** / Blockers / Lineage / Evidence) with `text-[10px] uppercase tracking-wide text-warm-faint` section headers.
- **Config:** stat cards + tables keep content; normalize table header style with intake drawer section headers; deprecated flag stays amber.
- **Health:** stat cards normalized to the same style as Config's; approvals/notices boxes use the shared notice style (amber box for degraded, neutral for info).
- **Command blocks (new):** `<pre><code>` with `bg-elevated border border-edge rounded p-2 text-[11px] font-mono text-warm-secondary select-all` — selectable text, **not** a button; zero write affordances.

### D6 — TDD plan (tests first, whole-domain coverage)

1. **Unblock plan domain table** (`console_unblock_test.go`): every known blocker type × {deny labels present/absent, pr-ready present/absent, valid/invalid repo, number 0/negative/positive, blank RunID} + `""` + `"future_blocker_v2"`. Assert: `ScopeCommand`/`ProtectedCommand` exact-match for the actionable types with corroborating label evidence — including the split rows (`cc:protected-action` present → protected command in its own field; run projecting only `not_pr_ready` with protected label → scope add-only + separate protected command, never combined); `ok=false` for everything else — **blocker/label mismatch** rows (e.g. `protected_action` blocker without `cc:protected-action` label), **same-run sibling blocker** rows (actionable card whose RunID carries a `stale_target`/`duplicate_chain`/unknown blocker on another board card), and **blank RunID** (CFADA1-1/2, CFADA2-1/2/3). Render test asserts the protected-action warning copy appears iff `ProtectedCommand` renders. This is the fix-the-class test over the full input domain.
2. **Render tests:** drawer shows command + terminal-run copy + rescan for an actionable card; shows only `RequiredAction` for `stale_target`/`duplicate_chain`/unknown; card hint appears iff plan exists.
3. **Read-only extension:** Intake surface (page + fragment + drawer with unblock section) asserts no `<form>`, no `hx-post/put/delete/patch`, no `<button type="submit">`, no write endpoints (mirror of `TestConsoleConfigRendersNoWriteControls`).
4. **Hostile-projection extension:** hostile repo (`transpara-ai/docs" onmouseover=…`, `../../etc`, `javascript:` URL), hostile labels, number 0 — assert raw payloads never survive AND no command renders (gate, not just escaping).
5. **Empty states:** Kanban zero-cards renders flow copy + `/console/intake` anchor; unavailable renders no anchor; other tabs' copy asserted.
6. **Existing guards must stay green unchanged in semantics** (status allowlist, drawer OOB reset, metacharacter round-trip, config guards).

## 3. Non-goals

- No recent-intakes rail (B2), no work-server timeout/perf changes (B3), no catalog/provenance changes (B4).
- No governed writes, no new endpoints, no changes under `/ops`.
- No new JS framework, CSS lib, or icon set.

## 4. Risks / IADA seeds

- R-1: Copy asserting system behavior must match hive reality (terminal parked runs, label gate) — sourced from verified `pkg/hive` anchors, not memory.
- R-2: A rendered shell command built from projected data is an injection surface for the *operator's terminal* — mitigated by the D2 allowlist gate (fixed label strings, strict repo regex, integer number) plus templ escaping.
- R-3: Changing empty-state strings breaks tests that assert them — tests updated in the same commit as the strings (TDD order: test first).
- R-4: Visual pass must not reduce information density or drop projected fields — every currently rendered field remains rendered.

## 5. Rollout

Single PR to `transpara-ai/site` off `feat/console-ux-polish`. Before/after screenshots of all four tabs in the PR body — captured against a local branch build with fixture-backed upstream mocks (deterministic populated/blocked/empty states; labeled as fixture-backed in the PR) plus live-box captures where the live projections cooperate. `make verify` green; `templ generate` no-drift.

## 6. IADA record (v0.1.0, 2026-07-02)

Adversarial pass by the authoring session before CFADA:

- **IADA-1 (blocker mixing, fixed in D2):** gating on the *primary* blocker type would offer a label command on cards that also carry non-label blockers (`duplicate_chain` etc.), where relabeling cannot unblock. Gate now requires *every* blocker to be label-actionable. Fail-closed: under-offering is safe, over-offering lies.
- **IADA-2 (ref mixing, fixed in D2):** command fields must come from one issue ref chosen by the existing target-preferred/selected-fallback helper order; mixing repo from one ref with number from another could emit a command against the wrong issue.
- **IADA-3 (case drift, fixed in D2):** hive normalizes labels ToLower/TrimSpace; naive exact compare would diverge from the real gate on cased labels.
- **IADA-4 (dispatch overreach, fixed in D2):** `--dispatch` verified optional (default false); presenting it as part of the mandatory rescan would push the operator into immediately spending agent tokens. Bare scan re-queues; `--dispatch` documented as the optional launch step.
- **IADA-5 (screenshot honesty, fixed in §5):** the live box serves main, not the branch; branch screenshots come from a local branch build with fixture mocks, and are labeled as such — no passing fixture renders off as live-box captures.

## 7. CFADA record

### Round 1 (codex, 2026-07-02) — VERDICT: BLOCKERS (4) → all resolved in v0.2.0

- **CFADA1-1 (same-run sibling blockers):** per-card gate could offer a command while the same run carries a non-label blocker on a sibling card. Resolved: gate aggregates blocker types per RunID across the whole board.
- **CFADA1-2 (evidence binding):** `cc:pr-ready` absence alone could produce an add-only command for `protected_action`/`needs_human_scope` cards whose deny labels are not observed. Resolved: each blocker type must be corroborated by its specific label evidence on the chosen ref, else no command.
- **CFADA1-3 (issue/packet dispatch divergence):** companion issue #202 R1 still showed `--dispatch` in the rescan command. Resolved: issue body updated to match the packet (bare scan re-queues; `--dispatch` documented as the optional immediate-launch flag).
- **CFADA1-4 (`<name>` is a shell metacharacter):** placeholder replaced with shell-inert `YOUR_NAME` everywhere a copyable block renders it.
- Advisories 1-3 adopted: chips read from the chosen ref; `IssueScanCandidatePRReady` cited; sibling-blocker and mismatch test rows added to D6.

### Round 2 (codex, 2026-07-02, via its CFADA governance skill) — VERDICT: BLOCKERS (3) → all resolved in v0.3.0

- **CFADA2-1 (API hole):** packet named a single-arg `consoleIssueScanUnblockPlan(card)` that could not see sibling blockers. Resolved: exact two-function API specified in D2 (index builder + two-arg plan function).
- **CFADA2-2 (one-way evidence binding / protected-boundary overreach):** a run projecting only `not_pr_ready` while the ref carries `cc:protected-action` would have had its protected boundary cleared as a side-effect of one combined command. Resolved: command split by authority class — scope command vs. a separate protected-action command with mandatory authorization warning copy (rationale and rejected alternatives recorded in D2).
- **CFADA2-3 (blank RunID fail-open):** aggregation under the empty key could alias unrelated cards. Resolved: non-empty-RunID precondition on the plan; index skips blank RunIDs.
- Advisories adopted: issue's upstream-facts section reworded to shell-inert placeholders; "copy-paste correct" wording tightened (label commands exact; rescan requires replacing `YOUR_NAME`).

### Round 3 (codex, 2026-07-02) — VERDICT: BLOCKERS (1) → resolved in v0.4.0

- **CFADA3-1 (plan tests were skeletons):** the implementation plan's Task 2 render tests were comment-only and referenced nonexistent helpers — unexecutable evidence. Resolved: replaced with complete, executable tests driven by the existing shared fixture `hiveCivilizationAssemblyProjectionFixture` (`graph/handlers_test.go:1289`), which already encodes the adversarial scenarios (run_site_115 sibling `protected_action`+`stale_target`; run_docs_172 `duplicate_chain`; run_docs_172_scope cleanly label-parked), plus direct component-render tests for the split protected command.
- Advisory 2 (round 3): codex explicitly accepted the split-command authority posture: "a separately warned protected-action removal command is an acceptable read-only authority posture for a human authority owner."
- Advisory 3 adopted: when both commands render, copy states the scope command alone does not admit until the protected boundary is separately authorized.
- Advisory 4 adopted: plan header provenance updated to the current packet version.

### CFAR round 1 (codex, on PR #203) — 1 blocker → resolved in v0.5.0

- **CFAR1-1 (stale-projection command suppression):** `buildConsoleIssueScan` stamped `UnblockAvailable` — and `handleConsoleIntakeCard` derived/rendered the unblock plan — whenever freshness was anything other than `FreshnessUnavailable`. That let `FreshnessStale` (and `FreshnessPartial`) boards and drawers offer exact `gh issue edit` label-surgery commands derived from an out-of-date label snapshot: the projection had already fallen outside `consoleStaleWindow` (or come from a degraded/partial source set), so the labels a command would remove/add were no longer a verified-current fact. Resolved with a two-tier, fail-closed gate:
  - **Tier 1 — finding the card at all** (`handleConsoleIntakeCard`): unchanged, still gated to `scan.Freshness != FreshnessUnavailable`. A stale or partial drawer still shows the real run and its projected `RequiredAction` honestly — hiding a genuine run because its timestamp aged out would itself be dishonest.
  - **Tier 2 — offering hints/commands**: tightened to an explicit allowlist, `scan.Freshness == FreshnessCurrent` only. `buildConsoleIssueScan` now computes freshness *before* the per-card stamping loop and only stamps `UnblockAvailable` when it is exactly `FreshnessCurrent`; every other value (stale, partial, unavailable, and any future enum addition) leaves every card's `UnblockAvailable` at its zero value (`false`) by falling through rather than by an explicit per-case deny — the class of "not verified current" is denied by construction, not enumerated. `handleConsoleIntakeCard` mirrors this: it only calls `consoleIssueScanUnblockPlan` when `scan.Freshness == FreshnessCurrent`; otherwise it renders `consoleUnblockPlan{}, false`, matching the not-found path's behavior for the plan argument.
  - Partial is deliberately included in the suppression, not just stale: a partial derivation is a degraded source set (some records missing/failed), which is not a verified-current label snapshot either — allowlisting only `FreshnessCurrent` covers both without needing a separate partial-specific carve-out.
  - Test coverage: `graph/console_intake_test.go` adds `freshHiveCivilizationAssemblyProjectionFixture()` (rewrites the shared fixture's baked-in stale `generated_at` to `time.Now()` for tests that need `FreshnessCurrent` to assert real command rendering) and a new `TestConsoleIntakeStaleProjectionSuppressesUnblock`, which serves the shared fixture unmodified (stale) and asserts the board still renders `transpara-ai/docs#172` and the drawer still shows "human must clarify issue scope before runtime continues," but neither the board's "unblock available" hint nor any "gh issue edit" command appears anywhere. `TestConsoleIntakeDrawerNoCommandForNonLabelBlocker` was switched to the fresh fixture with a comment clarifying it asserts the sibling-blocker gate, not the freshness gate (which now has its own dedicated test).
