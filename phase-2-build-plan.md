# Phase 2 — Build Plan (Pass 1 audit)

**Date:** 2026-04-20
**Scope:** Audit-only. Produced before any edits in Pass 2. Captures the current state of the
Tailwind CDN wiring, `@theme` block duplication, Go state/priority helpers, and the proposed
build-infrastructure shape. Reviewed against Phase 1 findings and the Prompt 2 spec.

---

## 1. Current Tailwind version in use

- **CDN script:** `https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4`
- **Version pinning:** major-only (`@4`). jsdelivr resolves to the latest Tailwind v4 release at
  load time. No exact version pin exists anywhere in the repo.
- **Match strategy for the build step:** pin npm dependencies to the latest Tailwind v4 minor
  available at `npm install` time (`"tailwindcss": "^4"` + `"@tailwindcss/cli": "^4"`). Because
  the CDN was unpinned, any recent v4.x release is a legitimate "match" and no bit-for-bit
  reproduction is possible.
- **CDN locations to remove (7 total):**
  - `views/layout.templ:32`
  - `graph/hive.templ:14`
  - `graph/views.templ:62, 643, 891, 1064, 5112`

## 2. Current `@theme` block — both copies, drift check

Two `@theme` blocks live in the tree. Both are 12 tokens, identical values.

| Token | `views/layout.templ:35–48` | `graph/views.templ:20–33` (`themeBlock()`) | Drift? |
|---|---|---|---|
| `--color-brand` | `#e8a0b8` | `#e8a0b8` | — |
| `--color-brand-dark` | `#d4899f` | `#d4899f` | — |
| `--color-void` | `#09090b` | `#09090b` | — |
| `--color-surface` | `#111113` | `#111113` | — |
| `--color-elevated` | `#18181b` | `#18181b` | — |
| `--color-edge` | `#1e1e22` | `#1e1e22` | — |
| `--color-edge-mid` | `#2a2a2e` | `#2a2a2e` | — |
| `--color-edge-strong` | `#3a3a3f` | `#3a3a3f` | — |
| `--color-warm` | `#f0ede8` | `#f0ede8` | — |
| `--color-warm-secondary` | `#c8c4bc` | `#c8c4bc` | — |
| `--color-warm-muted` | `#78756e` | `#78756e` | — |
| `--color-warm-faint` | `#4a4844` | `#4a4844` | — |

**No drift.** Both blocks are byte-equal. Safe to pick either as the source of truth; we pick
`views/layout.templ` and move it into `static/css/input.css`.

Non-@theme CSS co-located with both blocks (to be consolidated into `input.css`):
- `views/layout.templ:50–128` — `.font-display`, `@keyframes breathe / fadeUp / ember-pulse /
  skeleton-pulse`, `.brand-breathe`, `.skeleton`, `.reveal`, `.reveal-scroll`, `.ember-glow +
  ::before`, `@media (prefers-reduced-motion: reduce)`, and the full `.prose` ruleset.
- `graph/views.templ:35–48` (`themeBlock()` inner `<style>`) — a subset: `.font-display`,
  `@keyframes breathe / skeleton-pulse`, `.brand-breathe`, `.skeleton`, reduced-motion media.

The `themeBlock()` subset is a strict subset of the `layout.templ` block; consolidating into a
single `input.css` eliminates duplication.

## 3. Current `stateColorHex` and `priorityDotHex` helpers

- **File:** `graph/views.templ` (lives alongside the templ helpers; regenerated into
  `graph/views_templ.go` by `templ generate`).
- **Lines:** `5278–5293` (`stateColorHex`), `5295–5308` (`priorityDotHex`).
- **Constants:** defined in `graph/store.go:28–41` as plain string literals
  (`StateOpen = "open"`, etc.).
- **Callers (inline `style="background-color: %s"`):**
  - `stateColorHex`: `graph/views.templ:1741` (board column dot).
  - `priorityDotHex`: `graph/views.templ:813, 1938, 3021, 4642` (priority dots on task rows and
    dashboard cards).

**Hex values and semantics:**

| Helper | Case | Hex | Semantics | Resolution |
|---|---|---|---|---|
| `stateColorHex` | `StateOpen` | `#78756e` | idle / backlog | maps to existing `--color-warm-muted` |
| `stateColorHex` | `StateActive` | `#818cf8` | task in progress (indigo) | **NEW: `--color-state-active`** |
| `stateColorHex` | `StateReview` | `#fbbf24` | awaiting review (amber) | **NEW: `--color-state-review`** |
| `stateColorHex` | `StateDone` | `#6ec89b` | completed (green) | **NEW: `--color-state-done`** |
| `stateColorHex` | `StateClosed` | `#4a4844` | closed / archived | maps to existing `--color-warm-faint` |
| `stateColorHex` | default | `#78756e` | fallback | maps to existing `--color-warm-muted` |
| `priorityDotHex` | `PriorityUrgent` | `#e07070` | urgent (red) | **NEW: `--color-priority-urgent`** |
| `priorityDotHex` | `PriorityHigh` | `#fbbf24` | high (amber, same as review) | reuse `--color-state-review` |
| `priorityDotHex` | `PriorityMedium` | `#e8a0b8` | medium (brand pink) | maps to existing `--color-brand` |
| `priorityDotHex` | `PriorityLow` | `#78756e` | low | maps to existing `--color-warm-muted` |
| `priorityDotHex` | default | `#78756e` | fallback | maps to existing `--color-warm-muted` |

**Proposed 4 new tokens (total):**

| Token | Value | Semantic role |
|---|---|---|
| `--color-state-active` | `#818cf8` | in-progress state color; indigo accent |
| `--color-state-review` | `#fbbf24` | review state color; also reused for `PriorityHigh` |
| `--color-state-done` | `#6ec89b` | done state color; success green |
| `--color-priority-urgent` | `#e07070` | urgent priority signal; red |

**Judgment call (flag in §F of findings):** `PriorityHigh` and `StateReview` are the same amber.
Rather than introduce a redundant `--color-priority-high` alias, `priorityDotHex(PriorityHigh)`
will reference `var(--color-state-review)` with an inline comment explaining the intentional
reuse. This holds the token count to exactly 4, matches the prompt's "4 semantic tokens", and
keeps the semantic relationship explicit in the code.

## 4. Build infrastructure plan

### 4.1 Toolchain choice — `npm`

npm is the simplest and comes preinstalled on every Node image. No pnpm/yarn benefit justifies
the extra config for a single-package dev-dep project. Commit `package.json` and
`package-lock.json`. No `node_modules/`.

### 4.2 Tailwind v4 config — CSS-first, no `tailwind.config.ts`

Tailwind v4 radically simplified config: `@theme` blocks live in input CSS, content globs are
declared via `@source "..."` directives in the same input CSS, and `tailwind.config.ts` is
optional (only needed for JS plugins or `@config` legacy migration). This project needs neither
plugins nor a v3 migration path, so **no `tailwind.config.ts` is created**. All config lives in
`static/css/input.css`.

**Deviation from prompt:** The prompt's "add `tailwind.config.ts`" reflects v3 thinking (Approach
Hint #3 acknowledges v4 is CSS-first). Skipping the file is the v4 idiom. Flagged in findings §F.

### 4.3 `static/css/input.css` structure

```css
@import "tailwindcss";

/* Content sources — globs scanning for utility classes. Go files included because
   templ literals carry class strings in string form. */
@source "../../views/**/*.templ";
@source "../../views/**/*.go";
@source "../../graph/**/*.templ";
@source "../../graph/**/*.go";
@source "../../cmd/**/*.go";
@source "../../handlers/**/*.go";

@theme {
  --color-brand: #e8a0b8;
  --color-brand-dark: #d4899f;
  --color-void: #09090b;
  --color-surface: #111113;
  --color-elevated: #18181b;
  --color-edge: #1e1e22;
  --color-edge-mid: #2a2a2e;
  --color-edge-strong: #3a3a3f;
  --color-warm: #f0ede8;
  --color-warm-secondary: #c8c4bc;
  --color-warm-muted: #78756e;
  --color-warm-faint: #4a4844;
  /* Phase 2: state + priority semantic tokens (see stateColorHex / priorityDotHex). */
  --color-state-active: #818cf8;
  --color-state-review: #fbbf24;
  --color-state-done: #6ec89b;
  --color-priority-urgent: #e07070;
}

/* Component-scoped rules moved from views/layout.templ and graph/views.templ themeBlock(). */
.font-display { font-family: 'Source Serif 4', Georgia, 'Times New Roman', serif; }
@keyframes breathe { ... }
@keyframes fadeUp { ... }
@keyframes ember-pulse { ... }
@keyframes skeleton-pulse { ... }
.brand-breathe { ... }
.skeleton { ... }
.reveal { ... }
.reveal-scroll { ... }
.reveal-scroll.in { ... }
.ember-glow { ... }
.ember-glow::before { ... }
@media (prefers-reduced-motion: reduce) { ... }
.prose { ... } /* full prose ruleset */
```

### 4.4 Makefile target sketch

```makefile
.PHONY: css generate build run dev deploy

css:
	npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/site.css --minify

generate:
	templ generate

build: css generate
	go build -o site ./cmd/site/

run: build
	./site

dev:
	npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/site.css --watch &
	templ generate --watch &
	go run ./cmd/site/

deploy:
	fly deploy
```

### 4.5 Dockerfile stage sketch (multi-stage)

```dockerfile
# 1) CSS build stage — node only for tailwind
FROM node:20-alpine AS css-builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY static ./static
COPY views ./views
COPY graph ./graph
COPY cmd ./cmd
COPY handlers ./handlers
RUN npx @tailwindcss/cli -i ./static/css/input.css -o ./static/css/site.css --minify

# 2) Go build stage
FROM golang:1.25-alpine AS go-builder
RUN go install github.com/a-h/templ/cmd/templ@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=css-builder /app/static/css/site.css ./static/css/site.css
RUN templ generate
RUN CGO_ENABLED=0 go build -o /site ./cmd/site/

# 3) Final stage — unchanged shape, just pulls built CSS via the go stage's /app/static
FROM alpine:3.21
RUN apk add --no-cache ca-certificates nodejs npm
RUN npm install -g @anthropic-ai/claude-code
COPY --from=go-builder /site /site
COPY --from=go-builder /app/static /static

EXPOSE 8080
CMD ["/site"]
```

Node is used only in the `css-builder` stage; final Alpine image size should not grow.

### 4.6 `deploy.sh` insertion point

Insert `make css` as a new step immediately before the existing `templ generate` step. The
ordering matches Makefile `build: css generate`. No other changes to `deploy.sh`.

## 5. CI impact

Current `.github/workflows/ci.yml` does not run npm. Add before the `Install templ` step:

```yaml
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
          cache: "npm"

      - name: Install npm deps
        run: npm ci

      - name: Build CSS
        run: make css
```

The existing `Check for uncommitted generated file changes` step must not flag
`static/css/site.css` — the file is gitignored (see §6) so it will never appear in
`git diff --exit-code -- '*_templ.go' '*_gen.go'`. Safe.

## 6. `.gitignore` updates

Add:
```
/node_modules/
/static/css/site.css
```

Existing stale line `static/css/output.css` is left alone (it refers to a path we will not
produce; removing it is cosmetic and out-of-scope for Phase 2).

## 7. Expected output size + build time

- **Output size:** 20–40 KB gzipped per Prompt 2 constraint; well under the 100 KB cap. Exact
  number filled in findings §E after first build.
- **Build time:** <5 s on modern hardware per Prompt 2 constraint. Exact number filled in
  findings §E.

## 8. Risks carried from Prompt 2

- **R1 — v4 CSS-first vs v3 JS-first.** Resolved by skipping `tailwind.config.ts`; input.css is
  the single config surface. Flagged in findings §F.
- **R2 — `color-mix()` support.** Phase 1 introduced `color-mix` in ember-glow. All evergreen
  browsers support it since 2023; project is not aimed at legacy browsers.
- **R3 — Content glob completeness.** Globs in §4.3 cover `.templ` and `.go` under `views/`,
  `graph/`, `cmd/`, `handlers/`. If a Tailwind class lives elsewhere, it will be dropped from
  the output. No such location is known. Smoke test in Pass 3 will catch a class dropped from
  visible UI.
- **R4 — Docker image size.** Node is confined to the `css-builder` stage; final Alpine image
  unchanged. Verified by not copying `node_modules` into the go or final stages.
- **R5 — `deploy.sh` idempotency.** `deploy.sh` does no `git diff --exit-code` on the CSS file,
  so adding `make css` does not risk false-positive "uncommitted changes" errors.

## 9. Open questions for Michael (go / no-go before edits)

None that block progress. Three judgment calls are being made and will be re-reported in
findings §F:

1. **No `tailwind.config.ts`** — v4 CSS-first idiom; `@source` lives in `input.css`.
2. **`PriorityHigh` reuses `--color-state-review`** rather than introducing a redundant alias
   token. Holds new tokens to exactly 4.
3. **Consolidated `@themeBlock()`** — its component CSS (`.font-display`, `.brand-breathe`,
   `.skeleton`, `@keyframes breathe/skeleton-pulse`, reduced-motion media) is a strict subset
   of the `layout.templ` CSS. After consolidation into `input.css`, both heads link the same
   `site.css` and the `themeBlock()` helper becomes a thin partial that emits font preconnects
   + font stylesheet link + `<link rel="stylesheet" href="/static/css/site.css">`.

Proceeding to Pass 2.
