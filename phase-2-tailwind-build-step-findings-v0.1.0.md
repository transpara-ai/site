# Phase 2 — Tailwind Build Step Findings

**Version:** 0.1.0 · **Date:** 2026-04-20
**Author:** Claude (via Claude Code)
**Owner:** Michael Saucier
**Status:** Phase 2 complete — pending PR review
**Companion:** `docs/site-profile-redesign/08-prompt-2-phase-2-tailwind-build-step.md`,
`phase-2-build-plan.md`

---

## A. Audit results (Pass 1)

- **Tailwind version (before):** `@tailwindcss/browser@4` (unpinned major) loaded via
  jsdelivr. 7 CDN script tag locations total: `views/layout.templ`, `graph/hive.templ`, and
  5 places in `graph/views.templ`.
- **`@theme` drift between the two blocks (before):** zero. Both 12-token blocks
  (`views/layout.templ:35–48` and `graph/views.templ:20–33` inside `themeBlock()`) held
  byte-identical values. Single source of truth after Phase 2:
  `static/css/input.css`.
- **State/priority helpers:** `graph/views.templ:5278–5308` (`stateColorHex`,
  `priorityDotHex`). 11 hex returns across 2 functions, 5 call sites across the file.
  Proposed token names from semantic meaning, not value:
  - `--color-state-active` ← `#818cf8` (task in progress)
  - `--color-state-review` ← `#fbbf24` (awaiting review; also reused for `PriorityHigh`)
  - `--color-state-done` ← `#6ec89b` (completed)
  - `--color-priority-urgent` ← `#e07070` (urgent priority)
- **Build-infrastructure plan accepted:** yes, with one deviation — no
  `tailwind.config.ts`; content globs and `@theme` live in `static/css/input.css` via
  `@source` directives, which is the Tailwind v4 idiom (see §F.1).

Full plan: `phase-2-build-plan.md`.

## B. Build infrastructure added

- **package.json dependencies:** `tailwindcss@^4` and `@tailwindcss/cli@^4` as
  devDependencies. Installed resolves to `4.2.2`. Two scripts: `build:css`, `watch:css`.
- **tailwindcss.config.ts:** not added (v4 CSS-first). See §F.1.
- **Content globs (via `@source` in input.css):** `views/**/*.templ`, `views/**/*.go`,
  `graph/**/*.templ`, `graph/**/*.go`, `cmd/**/*.go`, `handlers/**/*.go`.
- **input.css structure:** `@import "tailwindcss"` → 6 `@source` directives → `@theme`
  block (12 existing + 4 new tokens) → component CSS (animations, skeleton, ember-glow,
  prose — all moved in from the two inline `<style>` blocks).
- **Makefile target:** `css` (runs `npx @tailwindcss/cli ... --minify`). `build` now
  depends on `css + generate`. Added `--watch` flag to `dev`.
- **Dockerfile stage count:** 2 → 3. New `css-builder` (node:20-alpine) runs `npm ci` +
  `make css`; compiled CSS is copied into the go builder stage; final alpine image
  unchanged in shape.
- **deploy.sh line-count delta:** +3 lines. Inserts `make css` step immediately before
  `templ generate`.
- **CI changes:** yes. `.github/workflows/ci.yml` gains `actions/setup-node@v4`,
  `npm ci`, and `make css` before the existing templ steps.

## C. Tokens added

All four tokens live in `static/css/input.css:33–36` under the single `@theme` block.
All four are now referenced by the Go helpers at `graph/views.templ:5278–5308`, which
return `var(--color-...)` strings interpolated into inline `style` attributes at 5 UI
call sites.

| Token | Value | Semantic meaning | Files using it |
|---|---|---|---|
| `--color-state-active` | `#818cf8` | `stateColorHex(StateActive)` — task in progress | `static/css/input.css`, `graph/views.templ` |
| `--color-state-review` | `#fbbf24` | `stateColorHex(StateReview)` — awaiting review; also `priorityDotHex(PriorityHigh)` | `static/css/input.css`, `graph/views.templ` |
| `--color-state-done` | `#6ec89b` | `stateColorHex(StateDone)` — completed | `static/css/input.css`, `graph/views.templ` |
| `--color-priority-urgent` | `#e07070` | `priorityDotHex(PriorityUrgent)` — urgent priority | `static/css/input.css`, `graph/views.templ` |

## D. Consolidation results

- **`views/layout.templ`:** inline `<style type="text/tailwindcss">@theme{...}</style>`
  and the big component `<style>` block (.font-display, animations, skeleton,
  ember-glow, prose, reduced-motion media) removed. Head now carries a single
  `<link rel="stylesheet" href="/static/css/site.css">`. `@tailwindcss/browser` CDN
  script removed.
- **`graph/views.templ` (`themeBlock()`):** `@theme` block and inline component
  `<style>` removed. The helper now emits just font preconnect + font stylesheet +
  `<link rel="stylesheet" href="/static/css/site.css">`. 5 additional CDN `<script>`
  tags across the 5 standalone `<head>` sections removed; each still invokes
  `@themeBlock()` and picks up the compiled CSS through it. **0 component rules
  remain in graph/views.templ** — the subset that was there was strictly contained in
  the layout.templ subset, so there was nothing to keep.
- **`graph/hive.templ`:** CDN `<script>` removed; it uses `@themeBlock()` and inherits
  the compiled CSS link.
- **Single source of truth:** `static/css/input.css`.
- **Drift between the two @theme blocks was reconciled in favor of:** neither — both
  were byte-identical. The move to `input.css` preserves all 12 values verbatim, then
  extends with the 4 new state/priority tokens.

## E. Verification results

- **make css:** pass. Raw output 66 295 bytes (65 KB); gzipped 11 010 bytes (11 KB) —
  well under the 100 KB gzipped cap. Build time ≈ 400 ms per run (Tailwind CLI
  self-reported); ≈ 1 s wall clock including npx startup. Far under the 5 s target.
- **templ generate:** pass. All 206 templ files regenerated; no diff after re-running.
- **go build:** pass. `go build ./cmd/site/` produces a working binary.
- **go test:** pass (all packages). `auth`, `graph`, `handlers`, `cmd/gen-persona-status`
  all green; no test broken or added.
- **docker build:** pass end-to-end
  (`docker build -t lovyou-ai-site:phase-2 .`). Final image 601 MB, within rounding
  distance of the pre-Phase-2 image (only change to /static is the 65 KB site.css).
- **deploy.sh dry-run:** no `--dry-run` flag exists; traced mentally. Sequence is
  `make css → templ generate → go build → setcap → systemctl restart → health check`.
  No step diffs the CSS output against the working tree, so adding `make css` does not
  risk a false-positive "uncommitted changes" failure.
- **git diff main -- auth/auth.go:** empty. Out-of-scope file untouched.
- **hex-count grep:**
  `grep -rInE '#[0-9a-fA-F]{3,8}' --include='*.templ' --include='*.css' --include='*.html' cmd/ views/ graph/ static/`
  — excluding the compiled `site.css` (gitignored build artifact) — returns 17
  matches:
  - 16 token definitions in `static/css/input.css:18–36` (12 existing + 4 new).
  - 1 HTML ID selector `#feed-items` at `graph/views.templ:3098` (pre-existing false
    positive flagged in Phase 1).
  Whitelist-only. Zero drift.
- **Visual regression:** null by construction. All hex values removed from Go source
  map to existing-or-new token values that the compiled CSS resolves to the identical
  RGB. Runtime smoke-test (`PORT=8765 ./site`) confirms:
  - `/` and `/blog/the-hive` return 200, link `/static/css/site.css`, contain no
    `@tailwindcss/browser` reference and no inline `<style>` blocks.
  - `/static/css/site.css` serves 200 with `Content-Type: text/css` and the 66 295-byte
    payload.
- **Utility-class smoke test:** pass. Spot-checks of the compiled output confirm
  `bg-void`, `text-brand`, `text-warm-muted`, `bg-surface`, `border-edge`, opacity
  modifiers (`bg-brand/10` through `bg-brand/90`), hover variants, and semantic
  literal classes (`text-indigo-400`, `bg-indigo-500`, `text-amber-400`,
  `text-emerald-400`) all compile into the output. Token definitions for all 12+4
  tokens present in `:root`.
- **Go-helper smoke test:** pass. `graph/views_templ.go:16968–17000` confirms
  `stateColorHex` and `priorityDotHex` now return `"var(--color-...)"` literals; the
  5 call sites interpolate those via `fmt.Sprintf("background-color: %s", ...)` so
  rendered `style` attributes at runtime carry `background-color: var(--color-...)`,
  not hex.

## F. Surprises and judgment calls

1. **No `tailwind.config.ts`.** Tailwind v4 is CSS-first: `@theme` and `@source` live in
   `input.css`. A `tailwind.config.ts` is only useful for JS plugins or v3 `@config`
   migration paths, neither of which this project needs. The prompt listed
   `tailwind.config.ts` in the in-scope list; Approach Hint #3 already acknowledged v4
   is CSS-first. I omitted the file and documented it in the build plan §4.2. If
   Michael wants an explicit config file (e.g., for future plugins), adding one later
   is a one-line input.css change (`@config "./tailwind.config.ts"`) plus the new file.
2. **`PriorityHigh` reuses `--color-state-review`.** The two Go cases both resolve to
   `#fbbf24`. Rather than define `--color-priority-high` as a 5th token that
   literally aliases `--color-state-review`, `priorityDotHex(PriorityHigh)` returns
   `var(--color-state-review)` with an inline comment. This holds the token count to
   exactly 4 (the prompt's target) and keeps the semantic relationship explicit where
   a future profile swap would diverge the two if desired.
3. **`@themeBlock()` stays as a partial.** The prompt suggested removing the
   `@theme` duplicate from `graph/views.templ`. The helper was more than just the
   `@theme` block — it also emitted font preconnect links and a plain `<style>` with
   `.font-display`, `.brand-breathe`, `.skeleton`, and reduced-motion rules. All of
   those rules are a strict subset of what lived in `views/layout.templ`'s inline
   `<style>`. Consolidating all of it into `input.css` means the helper is now a
   4-line partial (preconnects + fonts + link to `site.css`), which all 6 remaining
   callers (HivePage, appLayout, Welcome, and 3 other `<head>` sections in
   `graph/views.templ`) use unchanged. No component rules needed to stay in templ land.
4. **Stale gitignore entry preserved.** `/site` gitignore already had
   `static/css/output.css` — a leftover from an earlier abandoned attempt. I added
   `static/css/site.css` next to it and left `output.css` alone; removing the stale
   entry is cosmetic and out-of-scope. A later sweep can tidy it.
5. **Node is new on the deploy host.** `deploy.sh` now calls `make css`, which
   requires node + npm to be installed on the deploy target. The existing flow
   already requires templ in `$HOME/go/bin`, so the deploy host is not minimal
   infrastructure; node alongside is a reasonable ask. If the deploy host does not
   yet have node, `deploy.sh` will fail fast on the `make css` step — no partial
   state, no silent degradation.
6. **Helper function names unchanged.** `stateColorHex` / `priorityDotHex` now return
   CSS `var()` expressions, not hex. The names are mild misnomers. Renaming would
   touch 5 call sites and all 206 regenerated `_templ.go` files; the prompt chose to
   keep them, and so did I. Worth flagging as rename candidates for a future pass.

## G. Open questions for Michael

1. **Accept `tailwind.config.ts` omission?** Per §F.1. If you'd prefer the file be
   present even if empty (for future plugins / discoverability), say so and I'll add
   it in a follow-up commit. No impact on the build output either way.
2. **Ship `PriorityHigh`-reuses-`state-review` as-is?** Per §F.2. The alternative
   (introduce `--color-priority-high` as a true-alias 5th token) is one line each in
   input.css and priorityDotHex. I don't see value in it but it's a design call.
3. **Node on deploy host.** Per §F.5. Confirm the deploy host has node + npm, or the
   first PR-merge deploy will fail on `make css`. If this needs to be solved
   differently (e.g., commit the compiled CSS, or use a helper that installs node
   on-demand), I can adjust.
4. **Stale gitignore entry for `static/css/output.css`.** Per §F.4. Remove in a
   follow-up, or leave alone?

---

## Five-line summary

1. Build tool: Tailwind v4.2.2 via `@tailwindcss/cli` — npm toolchain, no
   `tailwind.config.ts` (CSS-first v4 idiom); config lives in `static/css/input.css`.
2. Output CSS size: 66 295 bytes raw / 11 010 bytes gzipped; build time ≈ 400 ms per
   `make css` run (well under 100 KB / 5 s targets).
3. Tokens added: `--color-state-active`, `--color-state-review`, `--color-state-done`,
   `--color-priority-urgent`.
4. Verification: all-pass. `make css`, `templ generate`, `go build`, `go test`,
   `docker build`, hex-count grep, utility-class smoke, Go-helper smoke, and runtime
   smoke (`curl`-driven) all green; visual regression null by construction.
5. Ready for Phase 3: yes — pending PR review and any §G resolutions.
