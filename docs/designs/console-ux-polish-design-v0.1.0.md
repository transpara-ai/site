# Console UX Polish — Design Packet

- **doc_id:** SITE-CONSOLE-UX-POLISH-DESIGN-001
- **version:** v0.1.0
- **status:** draft → IADA → CFADA
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

New pure function (no I/O), `consoleIssueScanUnblockPlan(card) (consoleUnblockPlan, bool)`:

- **Gate (allowlist; every condition must hold or return `ok=false`):**
  - the card has ≥1 blocker AND **every** blocker type on the card ∈ {`not_pr_ready`, `needs_human_scope`, `protected_action`} (IADA-1: a card that also carries `stale_target`/`duplicate_chain`/unknown must NOT offer a label command — relabeling would not unblock it and the offer would be a lie)
  - issue ref selected with the **same preference the existing card helpers use** (target issue preferred, else selected issue), all fields (repo, number, labels) taken from that one ref — never mixed (IADA-2)
  - ref `Repo` matches `^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*$`
  - ref `Number > 0`
  - action needed: at least one deny label present, OR `cc:pr-ready` absent
- **Label comparison** normalizes with `strings.ToLower(strings.TrimSpace(l))` — identical to hive's `issueScanLabelSet` (`pkg/hive/issue_scan_parking.go:367-376`), so the site's view of the gate matches the gate (IADA-3).
- **Command construction:** `gh issue edit <number> --repo <repo>` + one `--remove-label <l>` per present deny label (`cc:pr-deferred`, `cc:needs-human-scope`, `cc:protected-action`, in that fixed order) + `--add-label cc:pr-ready` iff absent. Labels come from the fixed constant set, never echoed from projection free-text.
- **Rescan command:** `hive factory scan-issues --human <name> --repo <repo>` (repo passes the same gate). Verified (`cmd/hive/factory_scan_issues.go:32`): `--dispatch` defaults false and is the extra step that turns the queued run into a FactoryOrder task — copy presents the bare scan as the re-queue step and mentions `--dispatch` as the optional immediate-launch flag (IADA-4). `<name>` is a documented placeholder: operator identity is not projected, and fabricating one would violate the honesty contract. A one-line note says to replace it and mentions the daemon alternative (`hive factory daemon --issue-scan-interval 15m`).
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
- **Intake:** cards get label chips row (deny labels amber-tinted chip `border-amber-500/40 text-amber-300`, `cc:pr-ready` emerald chip, other `cc:*` labels neutral `border-edge text-warm-muted`; chips only for labels matching `^cc:[a-z0-9-]+$`, others fall back to existing plain text — allowlist, escaped either way); blocked cards show `unblock available` hint per D1; drawer gets sectioned layout (Issue / State / Possession / **Unblock** / Blockers / Lineage / Evidence) with `text-[10px] uppercase tracking-wide text-warm-faint` section headers.
- **Config:** stat cards + tables keep content; normalize table header style with intake drawer section headers; deprecated flag stays amber.
- **Health:** stat cards normalized to the same style as Config's; approvals/notices boxes use the shared notice style (amber box for degraded, neutral for info).
- **Command blocks (new):** `<pre><code>` with `bg-elevated border border-edge rounded p-2 text-[11px] font-mono text-warm-secondary select-all` — selectable text, **not** a button; zero write affordances.

### D6 — TDD plan (tests first, whole-domain coverage)

1. **Unblock plan domain table** (`console_intake_test.go`): every known blocker type × {deny labels present/absent, pr-ready present/absent, valid/invalid repo, number 0/negative/positive} + `""` + `"future_blocker_v2"`. Assert: command exact-match for the three actionable types with valid data; `ok=false` for everything else. This is the fix-the-class test over the full input domain.
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
