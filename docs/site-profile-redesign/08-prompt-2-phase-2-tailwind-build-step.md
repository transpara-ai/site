# Prompt 2 — Phase 2: Tailwind Build Step

**Version:** 0.1.0 · **Date:** 2026-04-20
**Author:** Claude Opus 4.7
**Owner:** Michael Saucier
**Status:** Ready to execute — copy into Claude Code session after Phase 1 PR merges
**Versioning:** Versioned as part of the site-profile-redesign set. Major for structural changes to the prompt scope; minor for additional work items; patch for corrections and clarifications.
**Companion:** `01-site-map-discovery.md` (v0.2.0), `02-display-profile-system.md` (v0.3.0), `03-transpara-profile-design.md` (v0.3.0), `04-transpara-profile-wireframes.md` (v0.3.0), `05-transpara-home-prototype.html` (v0.1.2), `06-site-profile-redesign-recon-prompt.md` (v0.1.0), `07-prompt-1-phase-1-token-refactor.md` (v0.1.0), `phase-1-token-refactor-findings-v0.1.0.md`

---

### Revision History

| Version | Date | Description |
|---------|------|-------------|
| 0.1.0 | 2026-04-20 | Initial Phase 2 prompt. Scope: replace `@tailwindcss/browser` CDN JIT with a proper Tailwind v4 build step producing a real CSS file. Includes two Phase 1 punts: (a) consolidate the duplicate `@theme` block from `graph/views.templ` into a single `static/css/input.css` source, (b) promote the 4 Go-interpolated state/priority colors (`stateColorHex`, `priorityDotHex`) to semantic tokens in `@theme` and update helpers to return `var()` references. No visual change. |

---

## Where this sits in the plan

```
Prompt 0 (recon) ─────────────► findings v0.1.0
Prompt 1 (Phase 1) ───────────► token refactor — MERGED (PR #18)
                                    │
                                    ▼
Prompt 2 (THIS) ──────────────► Phase 2: Tailwind build step  ◄── you are here
Prompt 3 ──────────────────────► Phase 3: Profile context
Prompt 4 ──────────────────────► Phase 4: three-shell abstraction
...through Phase 8 (test matrix)
```

Phase 2 replaces the `@tailwindcss/browser` CDN JIT with a proper Tailwind v4 build step that outputs a real CSS file. Three deliverables rolled into one phase because they all touch the same surface:

1. **Build infrastructure** — `package.json`, `tailwind.config.ts`, `static/css/input.css`, Makefile target, Dockerfile stage, `deploy.sh` update.
2. **@theme consolidation** — move tokens from the two templ-embedded `@theme` blocks into `static/css/input.css`. The duplicate in `graph/views.templ` goes away. The one in `views/layout.templ` becomes a plain `<link rel="stylesheet">` reference.
3. **State/priority token promotion** — add 4 semantic tokens (names derived from helper function semantics) to the consolidated `@theme`, update `stateColorHex` / `priorityDotHex` Go helpers to return `var(--color-...)` strings instead of raw hex.

Phase 1 findings confirmed the token surface is well-defined (0 new tokens needed beyond what @theme already held), so Phase 2 inherits a clean foundation. The 4 state/priority tokens are the only additions, and they have clear semantic names.

---

## Precondition

- PR #18 (Phase 1 token refactor) MUST be merged to `origin/main` before starting Phase 2.
- Current branch after PR #18 merge: `main` at the merge commit.
- `phase-1-token-refactor-findings-v0.1.0.md` should be readable in the repo root (it was committed on the feature branch and persists after merge).

If PR #18 is not yet merged, stop and notify Michael.

---

## How to launch

1. **Attach the design set + Phase 1 findings** to your Claude Code session (Artifacts 01–05 + 07 + Phase 1 findings doc).
2. **Enable skills:**
   - `frontend-design` — for context on Tailwind build patterns.
   - `hive-lifecycle` — not needed, keep available.
3. **Launch from `site` repo root.**
4. **Verify PR #18 is merged** (`git log --oneline origin/main | head -5` should show the merge commit).
5. **Paste the PROMPT block below.**

---

## PROMPT — copy everything between `─── BEGIN ───` and `─── END ───` markers below

```
─── BEGIN PROMPT 2 — PHASE 2: TAILWIND BUILD STEP ───

ROLE
You are executing Phase 2 of the site profile redesign for the
site repo. Phase 1 (token refactor) is complete and
merged to main; Phase 2 builds on that foundation. You have the
design set (Artifacts 01–05), Prompt 1 (Artifact 07), and the
Phase 1 findings doc attached.

GOAL
Replace the @tailwindcss/browser CDN JIT with a proper Tailwind v4
build step that produces a compiled CSS file. While at it, consolidate
the duplicate @theme block and promote the 4 Go-interpolated
state/priority colors to semantic tokens. No visual change.

WHY THIS PHASE EXISTS
The CEO decided (2026-04-20) to move from the browser-JIT CDN to
a real build step — part of the site-profile-redesign scope. This
unlocks proper CSS purging, predictable output, and eliminates the
FOUC risk of client-side JIT. It also enables CSS-file-based profile
swapping in Phase 3.

Phase 1 surfaced two issues that naturally belong to Phase 2:
- graph/views.templ has its own @theme block mirroring
  views/layout.templ. Once we have a single compiled CSS file
  loaded globally, the duplicate becomes redundant and harmful
  (risk of drift).
- stateColorHex and priorityDotHex in Go code return 4 hardcoded
  hex values. These should be var(--color-...) references so they
  participate in profile swaps.

SCOPE

IN SCOPE:
- Add package.json + tailwind.config.ts + postcss.config if needed.
- Add static/css/input.css as the single source of truth for @theme.
- Add Makefile target: `make css` → runs tailwind build.
- Update Dockerfile: add node build stage that runs `make css` before
  `go build`.
- Update deploy.sh: run `make css` before `templ generate` (or in
  parallel — whatever keeps deploy.sh readable).
- Update views/layout.templ: replace
  `<script src="https://cdn.jsdelivr.net/npm/@tailwindcss/browser@...">`
  with `<link rel="stylesheet" href="/static/css/site.css">`. Remove
  the inline `<style>@theme {...}</style>` block (tokens now live in
  input.css).
- Update graph/views.templ: remove its duplicate @theme block. Keep
  any component-scoped CSS rules (non-token styling).
- Update graph/hive.templ if it has its own @theme or <head> — route
  it through the same compiled CSS or its own minimal include.
- Add 4 semantic tokens to @theme (names derived from the actual
  semantic meaning of stateColorHex / priorityDotHex — read the
  Go source to name them correctly).
- Update stateColorHex / priorityDotHex to return
  `var(--color-state-X)` strings instead of hex literals.
- Add node/npm to CI if needed (.github/workflows/ci.yml).
- Add node_modules/, static/css/site.css, and any build artifacts
  to .gitignore.
- Commit package.json + package-lock.json (or pnpm-lock.yaml, match
  whatever toolchain you pick — npm is the simple default).

OUT OF SCOPE (do not touch):
- Profile context, shell abstraction, route overrides — Phase 3+.
- Any visual design change beyond what the token promotion produces
  (which should be visually null if tokens are chosen correctly).
- New components or new routes.
- work and summary repos — single-repo phase.
- Fly.io deployment. deploy.sh changes must keep the existing flow
  intact; you're inserting `make css` as a new step, not restructuring.
- Production deploy — do not run deploy.sh or fly deploy.

CONSTRAINTS (non-negotiable)
1. NO VISUAL CHANGE. Screenshots before and after must match to the
   human eye. `color-mix` rendering may have sub-perceptual deltas —
   ignore those, flag anything visible.
2. TAILWIND V4. Match the CDN version already in use (check the
   current CDN URL for the version number). Using v3 would be a
   structural regression.
3. ZERO NEW TAILWIND FEATURES. Don't introduce new utility classes
   or new config features. This phase changes HOW Tailwind compiles,
   not WHAT it produces.
4. CSS BUNDLE SANITY. Output CSS file must be under 100 KB gzipped.
   Tailwind v4 JIT with a well-specified @theme produces 20–40 KB
   gzipped for a site of this size. If the output is much larger,
   something is wrong — stop and investigate.
5. BUILD SPEED. `make css` should complete in under 5 seconds on
   a modern machine. Tailwind v4 is fast; if it isn't, config is
   broken.
6. NO MAIN-BRANCH COMMITS. Feature branch:
   `feat/phase-2-tailwind-build-step`
7. NO UPSTREAM PUSH. Origin only.
8. DON'T DEPLOY. deploy.sh must not run. Do not touch production.
9. CO-AUTHORED-BY TAG on every commit:
   `Co-Authored-By: Paperclip <noreply@paperclip.ing>`
10. PR, DO NOT MERGE. Michael reviews and merges.

WORK PLAN

Phase 2 is executed in three passes:

PASS 1 — AUDIT + PLAN (non-destructive)

Produce `phase-2-build-plan.md` at the repo root covering:

1. Current Tailwind version in use (read CDN script URL in
   views/layout.templ, extract version number).
2. Current @theme block contents — both copies. Diff them. Note any
   drift between views/layout.templ and graph/views.templ.
3. Current stateColorHex and priorityDotHex helpers:
   - File + line location.
   - The 4 hex values they return.
   - The semantic meaning of each (node state? priority level?
     agent state? Read the surrounding code to understand).
   - Proposed token names for each, following the existing
     --color-<semantic-role> pattern.
4. Build infrastructure plan:
   - Toolchain choice (npm vs pnpm — npm is simpler, use unless
     there's a strong reason otherwise).
   - tailwind.config.ts structure: content globs (must cover
     *.templ, *.go templ literals, static HTML), output path.
   - static/css/input.css structure: @import, @theme block,
     optional @layer rules.
   - Makefile target sketch.
   - Dockerfile stage sketch (multi-stage: node build → go build).
   - deploy.sh insertion point.
5. CI impact: does .github/workflows/ci.yml need a node setup step?
   If yes, plan the update.
6. Expected output size + build time on first run.

If anything in the plan requires a design call Michael hasn't made,
stop and flag in findings §G before proceeding. Do not assume.

PASS 2 — EDIT

Suggested commit sequence (order matters for bisect-ability):

  1. `feat(build): add tailwind v4 build infrastructure`
     (package.json, package-lock.json, tailwind.config.ts,
      static/css/input.css with complete @theme, Makefile target,
      .gitignore updates)

  2. `feat(theme): add 4 semantic state/priority tokens`
     (extend @theme in input.css; keep existing 12 tokens
      unchanged in name and value)

  3. `refactor(graph): use state/priority tokens in Go helpers`
     (update stateColorHex and priorityDotHex to return
      var(--color-...) strings; update tests if any)

  4. `refactor(layout): swap CDN script for compiled stylesheet`
     (remove @tailwindcss/browser script tag and inline @theme
      block from views/layout.templ; add
      <link rel="stylesheet" href="/static/css/site.css">)

  5. `refactor(graph): consolidate @theme — remove duplicate`
     (remove @theme block from graph/views.templ and
      graph/hive.templ if present; keep any component-scoped
      <style> rules)

  6. `chore(docker): add node build stage for tailwind`
     (Dockerfile multi-stage update)

  7. `chore(deploy): run make css before templ generate`
     (deploy.sh update)

  8. `chore(ci): add node setup + make css to ci.yml`
     (only if CI runs frontend builds today — otherwise skip)

After EACH commit:
- Run `make css` (commits 1+). Output file must exist and be
  parseable CSS.
- Run `templ generate` (commits 4+). No error, no unexpected diff.
- Run `go build ./cmd/site/` (all commits). Must succeed.
- Run `go test ./...` (all commits). Must pass.

PASS 3 — VERIFY

1. `make css` produces `static/css/site.css`. File exists, is
   syntactically valid CSS, and is under 100 KB gzipped.
   Check: `gzip -c static/css/site.css | wc -c`
2. `templ generate` produces no unexpected diff in committed
   output.
3. `go build ./cmd/site/` succeeds.
4. `go test ./...` — all existing tests pass.
5. Docker build: `docker build -t site:phase-2 .`
   succeeds end-to-end. Do not run the resulting image against
   production anything.
6. `deploy.sh --dry-run` (or equivalent; if deploy.sh has no
   dry-run flag, read it and mentally trace the sequence — don't
   actually execute).
7. `git diff main -- auth/auth.go` is empty.
8. Hex-count grep across the CSS surface (same command as Phase 1):
   `grep -rInE '#[0-9a-fA-F]{3,8}' --include='*.templ' --include='*.css'
    --include='*.html' cmd/ views/ graph/ static/`
   Expected survivors: the 12+4 tokens in input.css, any Tailwind
   arbitrary-value `[#...]` syntax (now resolvable by the real
   build), comments.
9. Visual regression: launch the site locally with the compiled
   CSS (not the CDN). Screenshot `/`, `/hive`, `/blog/the-hive`.
   Compare to post-Phase-1 state. Diff must be visually null.
10. Tailwind utility smoke test: inspect element on a few utility
    classes in the browser. `bg-brand`, `text-warm`, etc. should
    resolve to the token values in computed styles.
11. Go-helper smoke test: find a page that renders a state badge
    or priority dot. Inspect element. `style` attribute should
    contain `var(--color-...)`, not a hex string.

GIT DISCIPLINE

Branch: `feat/phase-2-tailwind-build-step`

Remote: `origin` (transpara-ai fork, NEVER upstream)

Commit format: Conventional commits, lowercase subject, imperative:
  feat(build): add tailwind v4 build infrastructure

  Introduce package.json with Tailwind v4 as the sole dependency.
  Add tailwind.config.ts with content globs covering .templ, .go,
  and static HTML. Add static/css/input.css containing the
  complete @theme block. Add `make css` Makefile target. Update
  .gitignore for node_modules/ and static/css/site.css.

  Co-Authored-By: Paperclip <noreply@paperclip.ing>

PR: open against `origin/main` titled
  "feat(build): Phase 2 — Tailwind build step + @theme consolidation"

PR body must include:
- Link to this prompt (if filed)
- Before/after comparison:
  - Before: CDN JIT + 2 inline @theme blocks + 4 hex-returning Go helpers
  - After: compiled CSS file + 1 @theme block + 4 var()-returning helpers
- Output CSS file size (raw + gzipped)
- Build time on your machine
- Screenshots of /, /hive, /blog/the-hive before and after
- Confirmation that docker build succeeds
- Any §F/§G items from findings that need Michael's attention

DELIVERABLE

Create `$REPO_ROOT/phase-2-tailwind-build-step-findings-v0.1.0.md`:

  # Phase 2 — Tailwind Build Step Findings

  **Version:** 0.1.0 · **Date:** <today>
  **Author:** Claude (via Claude Code)
  **Owner:** Michael Saucier
  **Status:** Phase 2 complete — pending PR review
  **Companion:** prompt-2-phase-2-tailwind-build-step-v0.1.0.md

  ## A. Audit results (Pass 1)
     - Tailwind version (before)
     - @theme drift between the two blocks (before)
     - State/priority helpers: locations + values + proposed names
     - Build-infrastructure plan accepted as-is / modified

  ## B. Build infrastructure added
     - package.json dependencies (list)
     - tailwind.config.ts content globs
     - input.css structure
     - Makefile target
     - Dockerfile stage count (before/after)
     - deploy.sh line-count delta
     - CI changes (if any)

  ## C. Tokens added
     For each of the 4 new tokens:
     - Token name
     - Value (copied from Go helper hex)
     - Semantic meaning (what state/priority it represents)
     - Files now using the token

  ## D. Consolidation results
     - graph/views.templ: @theme removed, N component rules kept
     - graph/hive.templ: @theme removed (if applicable)
     - Single source of truth: static/css/input.css
     - Any drift between the two @theme blocks reconciled in favor of: <rationale>

  ## E. Verification results
     - make css: pass/fail, output size raw/gzipped, build time
     - templ generate: pass/fail
     - go build: pass/fail
     - go test: pass/fail (count, none broken)
     - docker build: pass/fail
     - deploy.sh dry-run: pass/fail
     - hex-count grep: survivors (list or "whitelist only")
     - Visual regression: null / noted differences
     - Utility-class smoke test: pass/fail
     - Go-helper smoke test: pass/fail

  ## F. Surprises and judgment calls
     Anything that required a decision not explicit in the prompt.

  ## G. Open questions for Michael
     Things that need his call.

  ---

  ## Five-line summary

  Line 1: Build tool + version + toolchain (npm/pnpm)
  Line 2: Output CSS size (raw / gzipped) + build time
  Line 3: Token additions (4 names in list form)
  Line 4: Verification status (all-pass / any failures)
  Line 5: Ready for Phase 3 (yes / blocked by X)

APPROACH HINTS

1. Start with `cat views/layout.templ | grep -A1 tailwindcss` to
   identify the current CDN version. Tailwind v4 via the browser
   script at the time of this writing is typically served from
   cdn.jsdelivr.net — match that exact version.
2. Read the recon findings §A.2 before designing the build. It has
   the current @theme block contents and the top-file hex counts.
3. Tailwind v4 uses CSS-first config (`@theme` block in input.css)
   rather than JS-first. `tailwind.config.ts` is still useful for
   content globs, but @theme lives in input.css now. This is the
   opposite of v3 and worth confirming before writing config.
4. Content globs must include Go files because templ literals can
   contain Tailwind classes as strings. A safe glob pattern is
   `['./views/**/*.templ', './graph/**/*.templ', './cmd/**/*.go',
     './views/**/*.go', './graph/**/*.go']`.
5. For the 4 state/priority tokens, name them by semantic meaning,
   not by value. "--color-state-claimed" beats "--color-green".
   Read the Go helpers and their call sites to understand what each
   state/priority represents before naming.
6. Multi-stage Dockerfile: keep the `FROM node:... AS build-css`
   stage minimal — only install Tailwind, copy the input CSS and
   content glob sources, run `make css`, copy the output to the
   final Alpine stage. Don't install the full templ toolchain in
   the node stage.
7. Cache-busting: the CSS output filename is `site.css`. If the
   site uses cache-busting URL suffixes (e.g. `site.css?v=abc123`),
   keep them. If it doesn't, Phase 2 doesn't add them — that's a
   separate concern.

FINAL CHECK BEFORE YOU START

- [ ] Working directory is site repo root
- [ ] Current branch is main at the PR #18 merge commit
- [ ] Artifacts 01–05 + 07 + Phase 1 findings are attached
- [ ] You will create feat/phase-2-tailwind-build-step
- [ ] You understand: no visual change, v4 only, single repo,
      don't deploy
- [ ] You will produce phase-2-build-plan.md in Pass 1
- [ ] You will produce findings doc at the path specified
- [ ] You will open a PR and STOP — not merge

Begin Phase 2.

─── END PROMPT 2 — PHASE 2: TAILWIND BUILD STEP ───
```

---

## What happens after this runs

1. Claude Code produces `phase-2-build-plan.md` (Pass 1), executes commits, produces `phase-2-tailwind-build-step-findings-v0.1.0.md`.
2. Claude Code opens PR against `origin/main` and stops.
3. You review the PR — particular attention to:
   - Output CSS size (should be 20–40 KB gzipped).
   - Docker build succeeds and image size hasn't ballooned.
   - Visual regression — this is the phase most likely to cause surprises because build output is different bytes even when the rendered CSS is equivalent.
4. Manual visual check on `/`, `/hive`, `/blog/the-hive` before merging.
5. If §F surfaces anything interesting or §G has open questions, bring findings back here.
6. We draft **Prompt 3: Phase 3 — Profile context** for the next session.

---

## Why this phase is bigger than Phase 1

Phase 1 was pure CSS hygiene — no structural change. Phase 2 introduces a new build tool, a new Dockerfile stage, a new CI dependency, a new generated artifact, and consolidates structural duplication. The blast radius is larger. That's why:

- Eight suggested commits instead of 4–5.
- Verification adds `docker build` and bundle-size checks.
- The audit pass produces a plan document before editing, not just a token map.
- CI changes are explicit (the top of many build-step migrations quietly breaks CI until someone notices).

The Tailwind v4 CSS-first config story is genuinely pleasant — the `@theme` block you already have IS the Tailwind v4 theme config. You're basically moving one `<style>` block from a templ file into `static/css/input.css` and adding a build target that compiles it. The surface area is small even though the moving parts are many.

---

## Risk notes (for your situational awareness, not for the prompt)

**Risk 1 — Tailwind v4 CSS-first vs JS-first config.** Tailwind v4 radically simplified configuration — `@theme` blocks in CSS are now the primary way to declare design tokens, and `tailwind.config.ts` is reduced to content globs and plugins. If Claude Code reaches for v3-style JS configuration patterns, the build will work but be idiomatically wrong. The prompt flags this in Approach Hint #3.

**Risk 2 — `color-mix()` usage from Phase 1.** The two rgba-literal ember-glow rules in Phase 1 were converted to `color-mix(in srgb, var(--color-warm) N%, transparent)`. Tailwind v4 supports `color-mix` natively in compiled output, but some older browsers may not. If the site supports legacy browsers, this becomes a concern. (My read: nucbuntu-dashboard-class users are on Chrome/Edge/Safari current — not a real concern — but worth noting.)

**Risk 3 — Content glob completeness.** Tailwind's JIT only generates CSS for classes it sees in content globs. If the glob misses `graph/**/*.go` and a class appears only in Go templ-literal strings, that class won't compile into the output CSS and will be visually broken. The prompt's glob pattern covers `.go` in three directories; adjust if Phase 1 findings show classes living elsewhere.

**Risk 4 — Docker image size.** Adding a node build stage to a multi-stage Dockerfile doesn't affect the final image size (the node stage is discarded after the copy), but careless stage authoring can leak node_modules into the final Alpine image. Claude Code should verify the final image size hasn't increased meaningfully.

**Risk 5 — deploy.sh idempotency.** If `deploy.sh` is run multiple times in quick succession (e.g. on manual deploys), `make css` running each time is fine; but if the script does `git diff --exit-code` checks against the compiled CSS file, adding the build step might trigger false-positive "uncommitted changes" errors. Verify deploy.sh isn't diff-checking generated assets.

If Claude Code hits any of these, findings §F. Same discipline as Phase 1.

---

## One tactical note on the state/priority token names

I'm deliberately not naming the 4 tokens in this prompt because I haven't read the Go helpers. Claude Code's Pass 1 audit will read `stateColorHex` and `priorityDotHex`, determine what each color represents, and propose names in the plan document. Likely candidates based on the /market filters from Artifact 01 (`?priority=urgent|high|medium|low`):

- `--color-priority-urgent`
- `--color-priority-high`
- `--color-priority-medium`
- `--color-priority-low`

But the helpers might be about node states (`claimed|challenged|verified|retracted` from the /knowledge filter) or agent states (`idle|processing|retiring|retired`). Claude Code makes the call from the actual code. You veto or accept in findings §G.
