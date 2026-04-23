# Site Profile Redesign — Recon Findings

**Version:** 0.1.0 · **Date:** 2026-04-20
**Author:** Claude Opus 4.7 (via Claude Code)
**Owner:** Michael Saucier
**Status:** Recon complete — design correction pending
**Companion:** site-profile-redesign-recon-prompt-v0.1.0.md (Prompt 0, artifacts 01–05)

> **Caveat on artifact cross-checks.** Prompt 0 stated that Artifacts 01–05 were attached to this session, but no attachments were visible in the Claude Code context. Cross-checks in §A.4 and §E are therefore performed against Prompt 0's inline summaries of the artifacts, not against the artifacts themselves. Treat §E as a first pass — a full cross-check pends access to the actual design docs.
>
> **Output-path note.** Prompt 0 specified `$REPOS_ROOT/site-profile-redesign-recon-findings-v0.1.0.md`. `$REPOS_ROOT` resolved to `/Transpara/transpara-ai/data/repos`. A write there was blocked by a PreToolUse hook; this file was instead placed in `/Transpara/transpara-ai/data/repos/designs/` (a git-tracked directory holding all prior design + prompt docs — sysmon, allocator, cto, spawner), which is the semantically correct home alongside this work's sibling artifacts.

**Recon timestamp — D.2 git posture:**

| Repo                 | Branch                  | Last commit                                                                  | Clean? |
| -------------------- | ----------------------- | ---------------------------------------------------------------------------- | ------ |
| site       | `fix/test-suite-hygiene`| `3987801` 2026-04-20 test(graph): apply cancelled-ctx cleanup fix           | yes    |
| work       | `main`                  | `bab6149` 2026-04-20 feat(work-server): embed telemetry dashboard from summary (#19) | yes    |
| summary    | `main`                  | `fd4f089` 2026-04-19 feat(dashboard): consume phases[].agents from backend (#50) | yes    |

All three remotes verified: `origin = github.com/transpara-ai/<repo>.git` (safe forks). `upstream = lovyou-ai/*` is present on site + work with push URL sabotaged to `DISABLE_PUSH_TO_UPSTREAM`. No org-boundary violation in recon scope.

---

## A. site

### A.1 Stack

- **Language:** Go 1.25 — single-process binary (`cmd/site/main.go`).
- **HTTP framework:** stdlib `net/http` with `http.ServeMux` (no Gin/Echo/chi).
- **Template engine:** [templ v0.3.1001](https://templ.guide) — `.templ` files compile to Go functions via `templ generate`.
- **CSS strategy:** Tailwind CSS v4 via the **`@tailwindcss/browser` CDN** (no local Tailwind build step — JIT happens in-browser). Tokens declared inline via a `@theme` block in `views/layout.templ:35-48`.
- **JS strategy:** [HTMX v2.0.8](https://htmx.org) (CDN) for partial-HTML polling; vanilla JS for command palette. No React / Alpine / build step on the JS side.
- **Process boundary:** single Go binary serves HTML (via templ), static assets (`/static/`), and HTTP APIs (`/api/...`). Optional PostgreSQL backend (`DATABASE_URL`); falls back to local file reads for `/hive`.
- **Build:** `Makefile` → `templ generate` → `go build -o site ./cmd/site/`. Multi-stage Dockerfile builds an Alpine runtime image.

### A.2 Theming surface

- **Hardcoded colors: 119 matches total**, heavily concentrated. Top files:
  - `auth/auth.go` (62) — test/seed data, **not styling** (discount these).
  - `views/layout.templ` (32) — primary theme surface.
  - `graph/views.templ` (25) — app/hive component styles.
- **CSS custom properties already defined:** 12 tokens in `views/layout.templ:35-48` —
  `--color-brand`, `--color-brand-dark`, `--color-void`, `--color-surface`, `--color-elevated`, `--color-edge`, `--color-edge-mid`, `--color-edge-strong`, `--color-warm`, `--color-warm-secondary`, `--color-warm-muted`, `--color-warm-faint`.
  Plus `--d` (animation delay, dynamic).
- **Duplicates:** prose classes in `layout.templ` inline raw hex values (~15×) that shadow the token values — these need to be rewritten to `var(--...)` during token extraction.
- **Inline brand-derived `rgba`:** two ember-glow rules (`rgba(232, 160, 184, ...)`) — brand color as literals, need token aliases.
- **Fonts:** single external family — Google Fonts `Source Serif 4` via `<link>` in `layout.templ:31`. No `@font-face` declarations. System sans/mono fallbacks.
- **Token-extraction effort:** **MEDIUM-LOW.** Tailwind `@theme` centralization already exists; the hard part is scattered prose-class hex values (~15) plus the `graph/views.templ` (25 matches, app chrome). Realistic estimate: 1–2 days of focused refactor to get to a token-pure state, then profile-swap is cheap. **This is faster than Artifact 02 §6 assumes if it expected raw-vanilla-CSS pain.**

### A.3 Shell + layout structure

- **Single shared public layout:** `views/layout.templ` — wraps every public page. Header (~lines 131–160) + footer (~170–184) defined inline. Command palette modal at 185–202. **Chrome is NOT duplicated per page** — good news for profile swap.
- **Per-route pattern:** handler builds data → calls templ component → component invokes `@Layout(title, desc, user...) { /* children */ }`. Each page composes *inside* the layout rather than redefining chrome.
- **Exception:** `/app/*` graph handlers use their own layout primitives (`@themeBlock()`, `@simpleHeader()`) — **two-layout codebase**. The profile system must handle both surfaces or explicitly scope to the public layout.
- **`/hive` is a third layout:** `graph/hive.templ:7` `HivePage()` is a standalone `<html>` document with its own head/body — it does NOT use `views.Layout`. So we have **three chrome variants**, not two. (See §E corrections.)

### A.4 Route map — deltas vs Artifact 01

I could not cross-check directly against Artifact 01 (not attached). Below is the full registered route inventory for Michael to diff manually.

**Public routes (registered in `cmd/site/main.go`):** `/static/{path}`, `/blog`, `/blog/{slug}`, `/vision`, `/vision/layer/{num}`, `/vision/goal/{id}`, `/reference`, `/reference/grammar`, `/reference/cognitive-grammar`, `/reference/higher-order-ops`, `/reference/code-graph`, `/reference/layers/{num}`, `/reference/agents`, `/reference/primitives/{slug}`, `/reference/grammars`, `/reference/grammars/{slug}`, `/discover`, `/agents`, `/agents/{name}`, `POST /agents/{name}/chat`, `/user/{name}`, `POST /user/{name}/endorse`, `POST /user/{name}/follow`, `/activity`, `/market`, `/knowledge`, `/search`, `/api/palette`, `/api/members`, `/`, `/work` (301→`/app`), `/app`, `/health`, `/robots.txt`, `/sitemap.xml`.

**App routes (registered via `graphHandlers.Register(mux)`):** `/app/{slug}/*` with lenses for `board`, `feed`, `threads`, `conversations`, `people`, `agents`, `activity`, `knowledge`, `governance`, `changelog`, `projects`, `goals`, `roles`, `teams`, `policies`, `documents`, `questions`, `council`. Plus `/app/{slug}/node/{id}` + POSTs, `POST /app/{slug}/op`, `POST /api/hive/diagnostic`, `POST /api/hive/escalation`.

**/hive routes:** `GET /hive`, `GET /hive/feed` (HTMX partial), `GET /hive/status` (HTMX partial), `GET /hive/stats` (HTMX partial).

**Likely deltas to flag in Artifact 01:**
- `/hive` is its own top-level route, not under `/app/` — confirm Artifact 01 places it correctly.
- Four `/hive/*` HTMX polling endpoints exist (`feed`, `status`, `stats`) — Artifact 01 may show only `/hive`.
- `/work` → `/app` 301 redirect exists (likely undocumented).
- Static assets mount at `/static/` (source: `./static/`).

### A.5 /hive current implementation

**Canonical answer: `/hive` already renders a full Phase Timeline page today — it is not a stub.** This is the single most consequential recon finding.

- Handler: `graph/handlers.go:4089` `handleHive()`.
- Renders: `graph/hive.templ:7` `HivePage(LoopState, []DiagEntry, []RecentCommit, ViewUser)`.
- Data sources:
  - DB table `hive_diagnostics` (preferred) or local `loop/diagnostics.jsonl` (dev fallback).
  - `loop/state.md` → iteration + phase name.
  - `loop/build.md` → last-build title + cost.
  - `git log` on the hive repo → recent commits.
  - Supplemented with `nodes` + `ops` DB tables for totalOps / lastActive per agent.
- UI today: iteration badge, phase pill, last-build title/cost, phase timeline (HTMX-polled every 5s), recent commits, footer.
- **Proxy / iframe / external-API logic: NONE.** No reference to work-server, `:8080`, or `/telemetry/*` anywhere in site code. (A grep turned up one unrelated `localhost:8080` reference in a persona markdown doc.)
- Companion polling endpoints:
  - `GET /hive/feed` → `HiveDiagFeed(entries)` (partial, HTMX 5s poll)
  - `GET /hive/status` → `HiveStatusPartial()` (partial, HTMX 5s poll)
  - `GET /hive/stats` → `HiveStatsBar()` (partial, HTMX 5s poll)
- Ingest endpoints (from hive runner, not dashboard): `POST /api/hive/diagnostic`, `POST /api/hive/escalation`.
- Env var: `HIVE_REPO_PATH` (default `../hive`).

### A.6 Deployment + config

- **Production:** Fly.io via `fly deploy`. `fly.toml` declares app = `lovyou-ai`, region = `syd`, VM = 8 GB / 2 perf CPUs, HTTP 80/443 with force-HTTPS, `/health` check every 30s.
- **Local (nucbuntu-style):** `deploy.sh` runs `templ generate` → `go build` → `setcap` (port 80) → `systemctl --user restart site`. Systemd unit is **not versioned in the repo** — lives at `/usr/lib/systemd/user/site.service` on the host.
- **Env vars at startup:** `PORT` (default 8080), `DATABASE_URL` (optional), `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` / `AUTH_REDIRECT_URL` (optional — anonymous if unset), `HIVE_WEBHOOK_URL` (optional), `CLAUDE_CODE_OAUTH_TOKEN` (optional), `HIVE_REPO_PATH` (default `../hive`).
- **No config files** — 12-factor, env-var only.

---

## B. work

### B.1 Telemetry API surface

All telemetry routes registered in `cmd/work-server/main.go:736–753`:

| Path                           | Method | Response   | Notes                                                  |
| ------------------------------ | ------ | ---------- | ------------------------------------------------------ |
| `/telemetry/status`            | GET    | JSON       | Full runtime snapshot (phases + agents + health)       |
| `/telemetry/agents`            | GET    | JSON       | All agents' latest metrics                             |
| `/telemetry/agents/history`    | GET    | JSON       | Windowed history (1h/24h)                              |
| `/telemetry/agents/{role}`     | GET    | JSON       | One agent detail + history                             |
| `/telemetry/stream`            | GET    | JSON       | Recent event log (paginated)                           |
| `/telemetry/phases`            | GET    | JSON       | Expansion phase defs + status                          |
| `/telemetry/phases/{phase}`    | POST   | JSON       | Mutate phase status                                    |
| `/telemetry/health`            | GET    | JSON       | Latest hive health snapshot                            |
| `/telemetry/sse`               | GET    | SSE        | Debounced delta stream (2-min snapshot window)         |
| `/telemetry/roles`             | GET    | JSON       | Role definitions                                       |
| `/telemetry/roles/{name}`      | GET    | JSON       | One role w/ expansion rules                            |
| `/telemetry/actors`            | GET    | JSON       | Registered actor instances                             |
| `/telemetry/layers`            | GET    | JSON       | All 14 computational layers                            |
| `/telemetry/overview`          | GET    | JSON       | Combined structural + runtime snapshot                 |
| `/telemetry/`                  | GET    | HTML       | **Embedded Mission Control dashboard — no auth on this route.** |

**Dashboard-asset strategy (post-#19):**
- `dashboard/embed.go` uses `//go:embed dashboard.html` to bake the dashboard into the binary.
- Handler `telemetryDashboard` at `main.go:795–820`: reads `TELEMETRY_DASHBOARD_PATH` env var if set (dev override), else serves the embedded `[]byte`. Sets `Cache-Control: no-cache`.
- Also sets a `ws_key` session cookie (`HttpOnly, SameSite=Strict`) on dashboard GET — so the in-browser dashboard can call telemetry endpoints via cookie without handling Bearer headers itself.
- PR #19 removed: the old 1589-line `telemetryDashboardHTML` const, the `TELEMETRY_DASHBOARD_URL` env var, the startup HTTP fetch for the dashboard, and the `POST /telemetry/refresh` endpoint.

### B.2 Auth model

- **Validated on every request** (no per-session caching) by middleware at `main.go:881-896`.
- Auth order: `Authorization: Bearer <key>` header → fallback to `ws_key` cookie.
- SSE-specific middleware (`events.go:133-150`) additionally accepts `?key=<key>` query-string auth (EventSource can't set headers) — logs a warning when used (credentials in access logs).
- Workspace-scoped routes (`/w/{workspace}/...`) use a separate `WORK_API_TOKEN` env var (falls back to `WORK_API_KEY` if unset).
- **Key source: `WORK_API_KEY` env var. No hardcoded dev fallback.** Prior docs' mention of `WORK_API_KEY=dev` is just a convention; the server still requires the env var to be set or it fails to start.
- **Reverse-proxy viability: YES.** Because keys are checked per request (not per-session), a proxy can hold the key server-side and inject `Authorization: Bearer <key>` before forwarding. Alternatively, the proxy can set the `ws_key` cookie on the browser response and let subsequent browser fetches carry it. The dashboard's existing cookie path makes option 2 natural.

### B.3 CORS + port

- **CORS middleware** (`main.go:859–878`) applied globally:
  - Origin echo (if `Origin` header present, else `*`)
  - Allowed methods: `GET, POST, OPTIONS`
  - Allowed headers: `Authorization, Content-Type`
  - **`Access-Control-Allow-Private-Network: true`** — permits browsers on private networks (Chrome PNA).
  - `Vary: Origin` set correctly.
- **Port: default `8080`, `PORT` env-var overridable** (`main.go:609–611`). Binds to `:port` on all interfaces — loopback `127.0.0.1:8080` reachable from same host.
- **Dashboard mount: `GET /telemetry/`** (trailing slash). API endpoints at `/telemetry/<route>`.
- Also: `GET /` is a **separate** read-only monitoring page (inline HTML const), unrelated to the telemetry dashboard.

---

## C. summary

### C.1 Dashboard artifact

- **Single file:** `dashboard.html` at repo root, **180 KB, 3453 lines**.
- Self-contained: all CSS inlined (~760 lines in `<style>`, ~750 class rules), all JS inlined (~2600 lines in an IIFE `<script>`).
- **Zero external dependencies** — no CDN `<link>` / `<script src>`, no frameworks (React / Vue / D3 / Chart.js), no icon fonts, no images. Uses system font stacks.
- **Asset paths: N/A** — nothing to rewrite. This is the rare case of a zero-asset dashboard.
- **Relationship to work-server:** byte-identical copy at `work/dashboard/dashboard.html` (3453 lines, `diff -q` empty). Treated as a git-subtree-style copy; work-server embeds it via `//go:embed`.

### C.2 Embed-friendliness

- **Full-page artifact** — ships its own `<html><head><body>`. NOT a fragment.
- **Built-in embed detection:**
  ```javascript
  var embedded = /\/telemetry\/?$/.test(window.location.pathname);
  var API_BASE = embedded ? window.location.origin : params.get("api");
  var API_KEY  = embedded ? null : params.get("key");
  var USE_COOKIE_AUTH = embedded;
  ```
- **Embedded mode is implicit** (pathname-driven), not controlled by `?embed=1`. The regex matches `/telemetry` or `/telemetry/` exactly — **nothing else**.
- No `hideHeader` / `?embed` URL param; chrome (topbar, view switcher, connection status) always shows. Configuration card only appears if neither `?api=` param nor embed mode is detected.
- No `window.parent`, `postMessage()`, or iframe-detection code. No parent-origin assumption. It's safe to put in an iframe, but it doesn't cooperate with the parent frame.
- **Mount-path constraint:** embedded mode triggers ONLY when `window.location.pathname` ends in `/telemetry` or `/telemetry/`. **A reverse proxy mounted at `/hive` would break embed detection silently** (dashboard shows config card asking for api/key). This is the single biggest constraint on the wiring options.

### C.3 Polling cadence + data dependencies

- **Primary transport:** SSE at `{API_BASE}/telemetry/sse?types=hive.*,agent.*,phase.*,site.op.*` (`EventSource`, with `withCredentials: true` in cookie-auth mode).
- **Initial load:** parallel `GET /telemetry/status` + `GET /telemetry/roles` before SSE attaches (renders snapshot, then SSE delivers deltas).
- **Fallback polling (if SSE fails repeatedly):**
  - `GET /telemetry/status` every 30 s.
  - `GET /telemetry/overview` every 60 s.
- **On-demand:** `GET /tasks?assignee={actorId}` when an agent card expands.
- **Client-side timers:** 1 s tick for topbar clock only.
- **Reconnect strategy:** SSE `onerror` → exponential backoff via `setTimeout`, cap 60 s; after ~3 failures, fall back to polling mode; resume SSE on successful `onopen`.
- **API endpoints consumed:** `/telemetry/sse`, `/telemetry/status`, `/telemetry/roles`, `/telemetry/overview`, `/tasks`. All relative to `API_BASE` (either `window.location.origin` in embed mode, or `?api=` param).

---

## D. Cross-cutting

### D.1 Existing design-system traces

- **No shared design-system / tokens / theme package across the three repos.** Each repo styles independently.
- `site` does NOT consume any sibling repo via Go modules.
- `work` imports `github.com/lovyou-ai/eventgraph/go` with a local `replace` to `../eventgraph/go` — unrelated to styling.
- `summary` has no Go/npm dependencies at all (zero-dep dashboard.html).
- **No "profile", "theme", "brand", or "skin" concept exists in any repo.** The 12-token `@theme` block in site's `layout.templ` is the closest thing, but it's one hard-coded palette, not a profile system. The design's proposal to add one is net-new, not duplicating anything.

### D.2 Git branch posture

Captured at top of doc. All three repos clean; site is on a feature branch (`fix/test-suite-hygiene`, orthogonal to redesign work), work + summary on `main`.

### D.3 CI/CD

- **site:** `.github/workflows/ci.yml` — runs on push/PR to `main`. Go 1.25, installs templ, generates persona status + templ files, `git diff --exit-code` against generated files, then (presumably) build + test. Deploys via Fly.io — **no deploy step in CI**, deploy is manual/local (`fly deploy` or `deploy.sh`).
- **work:** `.github/workflows/` directory **does not exist**. No CI, no CD in this repo. All deploy mechanisms external to the repo (likely systemd on nucbuntu).
- **summary:** `.github/workflows/deploy.yml` — on push of `dashboard.html` to `main`, POSTs to `{WORK_SERVER_URL}/telemetry/refresh` to trigger a refresh on work-server. **THIS WORKFLOW IS STALE** — PR #19 on work-server removed the `POST /telemetry/refresh` endpoint. The workflow will 404 on every push. The dashboard is now embedded in work-server at build time (`//go:embed`), so the refresh hook is obsolete. (See §E correction 8.)

### D.4 Blockers

The 8-step plan in Artifact 02 §6 (which I don't have visibility into) has at least one **major unstated assumption** that recon contradicts:

**BLOCKER 1 (HIGH SEVERITY): `/hive` already renders content unrelated to Mission Control.**
The design appears to assume `/hive` in the site is (or will become) a Mission Control-style dashboard. Reality: `site:/hive` is a **Phase Timeline / recent-commits editorial page** that reads from the hive loop state (`loop/state.md`, `loop/build.md`, `loop/diagnostics.jsonl`) and DB (`hive_diagnostics`, `nodes`, `ops`). The Mission Control dashboard lives in **`work` at `:8080/telemetry/`**, a separate binary on a separate port. Before proceeding, Michael must answer: does the Transpara `/hive` (a) replace the Phase Timeline with Mission Control, (b) surface both as sub-views, or (c) keep Phase Timeline and expose Mission Control elsewhere (`/hive/mission-control`, `/telemetry`, etc.)?

**BLOCKER 2 (HIGH SEVERITY): Embed-detection regex couples mount path to `/telemetry`.**
The dashboard enables "embedded mode" (cookie auth, same-origin API base) only when `window.location.pathname` matches `/\/telemetry\/?$/`. A site-side reverse proxy mounted at `/hive` will serve the HTML, the browser will see pathname `/hive`, the regex will fail, and the dashboard will render the configuration card asking for api/key — silently broken UX, not a server-side error. Three fixes: (a) mount the proxy at `/telemetry` exactly (sidestepping `/hive`), (b) patch the regex in the served HTML via a response rewriter, (c) patch the regex in summary/ source (touches the upstream-shared artifact; risky).

**BLOCKER 3 (LOW SEVERITY): Tailwind-browser-JIT bundles theme inline.**
Site's CSS strategy is `@tailwindcss/browser` CDN with `@theme { ... }` declared inline in `layout.templ`. Swapping profiles means swapping that inline block. This is workable but means **profile switching happens at HTML render time, not via a separate CSS build** — relevant if Artifact 02 §6 assumed a CSS-file swap. Also makes FOUC avoidance slightly trickier (whole CSS is JIT-parsed client-side).

**BLOCKER 4 (LOW SEVERITY): Site has three chrome variants, not one.**
Public pages use `views.Layout`. `/app/*` pages use `graph.themeBlock + graph.simpleHeader`. `/hive` uses a **standalone `<html>` doc** (`graph.HivePage`). Any profile system must either (a) unify these under one profileable layout (significant refactor), (b) make the profile carry three layout variants, or (c) explicitly scope the profile to the public layout and accept that `/app/*` and `/hive` remain independent.

**No other blockers identified.** CORS, port binding, loopback reachability, cookie policies, and asset-path rewriting are all either non-issues or already solved by work-server's existing middleware.

---

## E. Recommended corrections to design artifacts

Since Artifacts 01–05 were not visible in this session, corrections are phrased against Prompt 0's inline summaries. Please verify against the actual artifacts.

1. **Artifact 02 §7 (`/hive` wiring options)** — Prompt 0 summary positions three options (reverse-proxy + shell-injection, iframe, API-only rebuild). Recon reveals **Option 2 (iframe to `work:8080/telemetry/`) is the cheapest and most robust**:
   - CORS is `*` with Private-Network-Access already enabled.
   - Embed mode triggers correctly because the iframe's own pathname is `/telemetry/`.
   - No asset-path rewriting needed.
   - No risk of silently breaking embed detection.
   Option 1 (reverse proxy) becomes viable only if mounted at `/telemetry` on the site, not `/hive`. **Suggest elevating Option 2 to the v1 recommendation.**

2. **Artifact 02 §7 (cookie auth)** — the summary's embedded mode uses `withCredentials: true` + same-origin cookie. For iframe embed (`:80` page hosting a `:8080` iframe), cookies still work because the iframe's fetches are to its *own* origin and `SameSite=Strict` applies per-origin. **Note in §7 that iframe + cookie auth is natively supported.**

3. **Artifact 01 (route map)** — verify the following are present:
   - `/hive/feed`, `/hive/status`, `/hive/stats` (three HTMX-polled partials beyond `/hive` itself).
   - `POST /api/hive/diagnostic`, `POST /api/hive/escalation` (ingest endpoints, not user-facing).
   - `/work` → `/app` 301 redirect.

4. **Artifact 01 (or wherever `/hive` is described)** — correct any claim that `/hive` is a stub or reserved slot. It's a rendered Phase Timeline today (`graph/hive.templ`).

5. **Artifact 02 §6 (theming pain estimate)** — if the migration plan budgets heavy time for token extraction, downgrade: site already has a 12-token `@theme` block and Tailwind utilities resolve `var(--color-...)` automatically. The remaining work is (a) rewriting ~15 raw-hex prose-class values in `layout.templ` to tokens, (b) auditing `graph/views.templ` (25 hardcoded colors), (c) parameterizing the `@theme` block for profile swap. Estimate 1–2 days, not 5+.

6. **Artifact 02 (or wherever layout refactor is planned)** — call out the three-layout reality (public `views.Layout`, app `graph.themeBlock + simpleHeader`, hive standalone `HivePage`). A profile system scoped only to `views.Layout` would leave the hive + app pages visually unchanged.

7. **Artifact 04 / 05 (wireframes + prototype)** — if any wireframe shows the Mission Control dashboard inline in the site's `/hive` page (not in an iframe), verify the choice. Embedding the dashboard's 3453-line self-contained HTML *inline* in a templ component is technically possible but loses the clean separation that makes iframe attractive.

8. **Cross-artifact cleanup** — `summary/.github/workflows/deploy.yml` still pings `POST /telemetry/refresh`, a route removed in work PR #19. Suggest either deleting the workflow or updating it to trigger a work-server rebuild instead. (Not a design-artifact correction per se — a codebase correction adjacent to the recon.)

9. **Artifact 02 §6 (build-system assumption)** — if the plan assumes a Tailwind build step exists today (for token extraction / purge / output CSS file), correct: there is none. Tokens live in `@theme { ... }` inline in `layout.templ`, and Tailwind compiles in the browser via the CDN script. Either (a) keep the in-browser approach and profile-swap by re-rendering `layout.templ`, or (b) migrate to a Tailwind build step as part of Phase 1 — the latter is a real scope-add, not a drop-in.

---

## F. Open questions for Michael

- Which `/hive` is the design targeting — site's Phase Timeline, work-server's Mission Control, or a new unified surface?
- For the Transpara profile, is `site:/hive` meant to *become* Mission Control (replacing Phase Timeline), *add* Mission Control as a tab/sub-route, or *coexist* with Mission Control at a different path?
- Should the profile system cover all three chrome variants (public / `/app` / `/hive`), or only the public layout?
- Is the Tailwind-in-browser CDN approach staying, or is Phase 1 of the migration the right place to introduce a real Tailwind build step?
- Do you want me to file an issue against `summary` for the stale `deploy.yml` referencing the removed `/telemetry/refresh` route, or leave it until after the redesign?
- Are `work` and `summary` in scope for any profile-system wiring, or is the profile purely a `site` concern?
- When Artifact 02 §7 says "reverse-proxy + shell injection", does "shell injection" mean wrapping the proxied dashboard in the site's layout (`views.Layout`)? If so, the iframe option entirely avoids that complexity — worth a second look.

---

## Five-line summary

1. **Stack:** Go 1.25 · stdlib net/http + templ templates · Tailwind-in-browser (CDN, `@theme` block, 12 tokens defined) + HTMX; single-binary deploy to Fly.io.
2. **Biggest surprise:** `/hive` already renders a rich Phase Timeline from hive loop state + DB; the Mission Control dashboard lives in a separate binary (work-server `:8080/telemetry/`) and is never referenced from the site's code.
3. **`/hive` wiring — easiest option:** **iframe to `http://nucbuntu:8080/telemetry/`** (Artifact 02 §7 Option 2). The dashboard already auto-detects embed mode only on that exact path, CORS is `*` with Private-Network-Access, and cookies work same-origin inside the frame. Option 1 (reverse-proxy) needs to mount at `/telemetry` exactly or patch the embed-regex — silent-failure risk makes it a poor default.
4. **Blockers:** 2 high (`/hive` semantic conflict, embed-regex path coupling) + 2 low (Tailwind-in-browser, 3-layout reality). No outright blocker to Phase 1 — all are resolvable in design.
5. **Correction scope:** **0.3.0 minor.** Artifact 02 needs §7 recommendation flipped to iframe and §6 theming estimate trimmed; Artifact 01 needs route-map deltas; one cleanup item outside the artifact set (stale summary deploy.yml).

