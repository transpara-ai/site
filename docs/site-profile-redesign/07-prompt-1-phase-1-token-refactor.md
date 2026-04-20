# Prompt 1 — Phase 1: Token Refactor

**Version:** 0.1.0 · **Date:** 2026-04-20
**Author:** Claude Opus 4.7
**Owner:** Michael Saucier
**Status:** Ready to execute — copy into Claude Code session
**Versioning:** Versioned as part of the site-profile-redesign set (01–07). Major for structural changes to the prompt scope; minor for additional work items; patch for corrections and clarifications.
**Companion:** `01-site-map-discovery.md` (v0.2.0), `02-display-profile-system.md` (v0.3.0), `03-transpara-profile-design.md` (v0.3.0), `04-transpara-profile-wireframes.md` (v0.3.0), `05-transpara-home-prototype.html` (v0.1.2), `06-site-profile-redesign-recon-prompt.md` (v0.1.0), `site-profile-redesign-recon-findings-v0.1.0.md`

---

### Revision History

| Version | Date | Description |
|---------|------|-------------|
| 0.1.0 | 2026-04-20 | Initial Phase 1 prompt. Scope: token refactor in `lovyou-ai-site` only. Target: rewrite ~40 hardcoded hex values to `var(--color-...)` references against the existing 12-token `@theme` block. No build-system changes, no new abstractions, no visual regression. |

---

## Where this sits in the plan

```
Prompt 0 (recon) ─────────────► findings v0.1.0
                                    │
                                    ▼
v0.3.0 design corrections ────► Artifacts 01–05 updated
                                    │
                                    ▼
Prompt 1 (THIS) ──────────────► Phase 1: token refactor  ◄── you are here
Prompt 2 ──────────────────────► Phase 2: Tailwind build step
Prompt 3 ──────────────────────► Phase 3: Profile context
Prompt 4 ──────────────────────► Phase 4: three-shell abstraction
...and so on through Phase 8 (test matrix)
```

Phase 1 is **deliberately small**. It's the safest possible starting move: no behavior change, no build change, no new abstractions. Just token-purity in the CSS. This unblocks every later phase without locking in any premature choices.

---

## How to launch

1. **Attach the design set** (Artifacts 01–05 + recon findings) to your Claude Code session.
2. **Enable skills:**
   - `frontend-design` — for context on token patterns.
   - `hive-lifecycle` — not needed for Phase 1 but keep available.
3. **Launch from the `lovyou-ai-site` repo root** (Phase 1 is single-repo; no need to launch from parent).
4. **Paste the PROMPT block below.**

---

## PROMPT — copy everything between `─── BEGIN ───` and `─── END ───` markers below

```
─── BEGIN PROMPT 1 — PHASE 1: TOKEN REFACTOR ───

ROLE
You are executing Phase 1 of the site profile redesign for the
lovyou-ai-site repo. This is the first implementation step after
the recon-and-design phase. You are working against a well-defined
spec (Artifacts 01–05 attached) with recon findings that have
already been incorporated into v0.3.0 of the design.

GOAL
Rewrite every hardcoded hex color value in the CSS surface of
lovyou-ai-site to use CSS custom properties (`var(--color-...)`)
from the existing 12-token @theme block in views/layout.templ.
No visual change should be detectable. This is a pure refactor.

WHY THIS PHASE EXISTS
The site already has a 12-token @theme block — recon confirmed this.
But ~40 hardcoded hex values still appear throughout the codebase,
shadowing the tokens. Before we introduce profile-swap machinery in
Phase 3, the CSS has to consistently reference tokens, not raw hex.
Otherwise a profile swap would miss those 40 spots and the Transpara
skin would leak lovyou-ai colors.

SCOPE

IN SCOPE:
- views/layout.templ — ~15 raw hex in prose classes shadow token
  values; two ember-glow rgba() rules use brand colors as literals.
- graph/views.templ — ~25 hardcoded color matches.
- Any other .templ, .css, or inline <style> block containing a
  hardcoded hex value in the CSS surface.
- Adding new tokens to the @theme block ONLY if a color is used
  repeatedly but has no existing token. Flag each addition.

OUT OF SCOPE (do not touch):
- auth/auth.go — recon noted 62 hex matches here but those are
  test/seed data, not styling. Leave alone.
- The @theme block itself — tokens stay as-is. You may ADD new
  tokens if justified, but do not modify existing ones.
- Tailwind classes that use hex values in arbitrary-value syntax
  like `bg-[#1a2230]`. Leave these for Phase 2 when the build
  step lands; arbitrary values require the build pipeline anyway.
- The cmd/site/main.go file and other Go logic — pure CSS refactor.
- `lovyou-ai-work` and `lovyou-ai-summary` repos — Phase 1 is
  single-repo.
- Build infrastructure (Dockerfile, Makefile, deploy.sh) —
  Phase 2 work.

CONSTRAINTS (non-negotiable)
1. NO VISUAL CHANGE. The rendered site must look byte-identical to
   before the refactor. You will verify this with screenshots.
2. NO NEW TOKENS without justification. If you add a token, record
   WHY in the findings doc (§E) — what existing token could not
   cover the use case, what you named it, and what value you used.
3. NO BUILD-STEP CHANGES. The @tailwindcss/browser JIT CDN must
   continue to work. If a change requires a real Tailwind build to
   function, stop and flag it in findings.
4. NO MAIN-BRANCH COMMITS. All work on a feature branch:
   `feat/phase-1-token-refactor`
5. NO UPSTREAM PUSH. Push only to `origin` (transpara-ai fork).
6. DON'T DEPLOY. close.sh / deploy.sh must not run. Do not touch
   production.
7. CO-AUTHORED-BY TAG. Every commit ends with:
   `Co-Authored-By: Paperclip <noreply@paperclip.ing>`
8. PR, DO NOT MERGE. Open a PR against origin/main and stop.
   Michael reviews and merges.

WORK PLAN

Phase 1 is executed in three passes:

PASS 1 — AUDIT (non-destructive)

Before touching any file, produce a token map. For every hardcoded
hex / rgb / rgba / hsl value found in the in-scope files:

  file:line | current value     | proposed token        | notes
  ──────────┼───────────────────┼───────────────────────┼──────────────
  layout.templ:143 | #1a2230     | var(--color-void)     | exact match
  layout.templ:187 | #e8a0b8     | var(--color-warm)     | brand glow, exact
  graph/views.templ:42 | #f7f8fa | var(--color-surface)  | exact match
  graph/views.templ:89 | #ff6b2c | NEW: --color-signal  | no existing match

Save this map as phase-1-token-map.md at the repo root. Review it
carefully yourself before proceeding. Every "NEW:" row needs an
explicit justification line: what existing token was insufficient?

If the map contains more than 5 "NEW:" rows, STOP and surface in
findings — that's a signal the @theme block is underspecified and
Phase 1 should expand the token set before the refactor, or the
design needs revisiting.

PASS 2 — EDIT

Work in small, reviewable commits. Suggested commit sequence:

  commit 1: `refactor(layout): token-ify prose class colors`
  commit 2: `refactor(layout): token-ify ember-glow rgba literals`
  commit 3: `refactor(graph/views): token-ify app chrome colors`
  commit 4: `refactor(graph/views): token-ify hive chrome colors`
  commit 5 (optional): `feat(theme): add [--color-signal] token`
                       (one commit per new token, if any)

After each commit: run `templ generate` to ensure the .templ changes
compile. Run `go build ./cmd/site/` to ensure nothing else broke.
Neither should produce errors or warnings.

PASS 3 — VERIFY

1. `templ generate` succeeds with no diff in committed output.
   (Generated files may update; commit them in a separate commit
   titled `chore(gen): regenerate templ after token refactor`.)
2. `go build ./cmd/site/` succeeds.
3. `go test ./...` — all existing tests pass.
4. `git diff main -- auth/auth.go` is empty (auth.go untouched).
5. `git diff main -- Dockerfile Makefile deploy.sh fly.toml`
   is empty (no build/deploy changes).
6. Hex count check: `grep -rInE '#[0-9a-fA-F]{3,8}'
   --include='*.templ' --include='*.css' --include='*.html'
   cmd/ views/ graph/ static/` should return only:
   - The 12 token definitions in @theme
   - Any NEW tokens added to @theme (with justification)
   - Tailwind arbitrary-value `[#...]` syntax (out of scope)
   - False positives in comments (acceptable)
7. Visual regression: launch the site locally and screenshot at
   least three representative routes in both themes. Currently
   the site is single-theme (lovyou-ai dark), so screenshot in
   the current theme only:
   - `/` (home)
   - `/hive` (Phase Timeline)
   - `/blog/the-hive` (blog post)
   Compare to pre-refactor screenshots. Diff must be visually
   null — pixel-perfect not required, but nothing should look
   different to the human eye.

GIT DISCIPLINE

Branch: `feat/phase-1-token-refactor`

Remote: `origin` (transpara-ai fork, NEVER upstream)

Commit format: Conventional commits, lowercase subject, imperative:
  refactor(layout): token-ify prose class colors

  Replace 15 raw hex values in prose-class CSS with var(--color-...)
  references against the existing @theme block. No visual change.

  Co-Authored-By: Paperclip <noreply@paperclip.ing>

Before pushing: verify `git log --oneline origin/main..HEAD` lists
only commits with the Paperclip co-author tag.

PR: open against `origin/main` titled
  "refactor(theme): Phase 1 — token refactor (hardcoded hex → var)"

PR body must include:
  - Link to this prompt (if filed in repos/designs/)
  - Summary of files changed + line counts
  - The token map (or link to phase-1-token-map.md in the branch)
  - Any new tokens added, with justification
  - Confirmation that templ generate / go build / go test all pass
  - Screenshots of /, /hive, /blog/the-hive before and after

DELIVERABLE

Create a findings file at
`$REPO_ROOT/phase-1-token-refactor-findings-v0.1.0.md`
with the structure below. Do not skip sections; if a section is
empty, say "None" explicitly.

  # Phase 1 — Token Refactor Findings

  **Version:** 0.1.0 · **Date:** <today>
  **Author:** Claude (via Claude Code)
  **Owner:** Michael Saucier
  **Status:** Phase 1 complete — pending PR review
  **Companion:** prompt-1-phase-1-token-refactor-v0.1.0.md

  ## A. Audit results (Pass 1)
     - Total hex values found (in scope): N
     - Files touched: list
     - Map summary: X exact matches, Y new tokens needed

  ## B. Token additions
     For each new token: name, value, justification, files using it.
     Should be 0–3 additions for a clean refactor.

  ## C. Edits applied (Pass 2)
     Commit list with short descriptions. 3–6 commits typical.

  ## D. Verification results (Pass 3)
     - templ generate: pass/fail
     - go build: pass/fail
     - go test: pass/fail (existing test count, none broken)
     - hex-count grep: remaining occurrences (should match the
       whitelist in Pass 3 step 6)
     - Visual regression: pass/fail/concerns

  ## E. Surprises and judgment calls
     Anything that required a decision: an ambiguous color, a
     token you had to propose, a file that was harder than
     expected. Be honest — this is how Phase 2 gets tuned.

  ## F. Open questions for Michael
     Things that could not be resolved without his call.
     One question per line.

  ---

  ## Five-line summary

  Line 1: Files touched + total line-diff count
  Line 2: Token additions (count + names, or "none")
  Line 3: Biggest surprise (or "none")
  Line 4: Verification status (all-pass / any failures)
  Line 5: Ready for Phase 2 (yes / blocked by X)

APPROACH HINTS

1. Start by reading the recon findings §A.2 (theming surface). It
   already has the counts and the top-file list. Don't re-derive
   what's already known.
2. Read views/layout.templ FIRST. The @theme block is the source
   of truth. Understand what tokens exist before proposing new ones.
3. The 12 existing tokens:
     --color-brand, --color-brand-dark, --color-void,
     --color-surface, --color-elevated, --color-edge,
     --color-edge-mid, --color-edge-strong, --color-warm,
     --color-warm-secondary, --color-warm-muted, --color-warm-faint
   Most hardcoded values in-scope will map cleanly to one of these.
4. For rgba() literals with alpha (e.g. ember-glow rules), you can
   use color-mix(in srgb, var(--color-warm) NN%, transparent) OR
   add a dedicated token if the same alpha appears repeatedly.
   Prefer color-mix — it keeps the @theme block lean.
5. Commit often. If you break something, git bisect is cheap when
   commits are small.
6. When in doubt, leave a value as-is and flag it in findings §E.
   Undocumented token additions are worse than un-refactored hex.

FINAL CHECK BEFORE YOU START

- [ ] Working directory is lovyou-ai-site repo root
- [ ] Git status is clean (or at least, nothing you will lose)
- [ ] Artifacts 01–05 + recon findings are attached
- [ ] Current branch is NOT main; create feat/phase-1-token-refactor
- [ ] You understand: no visual change, no build change, no deploy
- [ ] You will produce phase-1-token-map.md in Pass 1
- [ ] You will produce findings doc at the path specified
- [ ] You will open a PR and STOP — not merge

Begin Phase 1.

─── END PROMPT 1 — PHASE 1: TOKEN REFACTOR ───
```

---

## What happens after this runs

1. Claude Code produces `phase-1-token-map.md` (in Pass 1), then edits, then produces `phase-1-token-refactor-findings-v0.1.0.md` at the repo root.
2. Claude Code opens a PR on `origin/main` and stops.
3. You review the PR. If the map is clean, the hex grep is empty (modulo whitelist), and the screenshots match, merge it.
4. Bring the findings doc back into this Claude.ai design thread.
5. We discuss any §E surprises or §F open questions.
6. Draft **Prompt 2: Phase 2 — Tailwind build step** for the next Claude Code session.

---

## Why this prompt is deliberately boring

Phase 1 is infrastructure hygiene, not design work. The exciting stuff — profiles, dark mode, iframe wiring, the Transpara aesthetic — all comes later. If Phase 1 feels underwhelming to read, the prompt is probably correctly scoped. The worst outcome here isn't "boring" — it's "ambitious": token additions that don't pay off, layout changes that break things, premature Tailwind-build-step work that belongs to Phase 2.

Keep it tight. Token-ify the CSS. Ship the PR. Move on.

---

## Risk notes (for your situational awareness, not for the prompt)

**Risk 1 — Tailwind browser-JIT quirks.** The CDN JIT resolves `var(--color-...)` for utility classes like `bg-[var(--color-brand)]` only if the variable is declared in the `@theme` block. If the refactor accidentally moves a variable outside `@theme`, utility classes break silently. Claude Code should leave the `@theme` block structure alone.

**Risk 2 — Inline style attributes.** templ syntax allows `style="color: #123456"` inline. These count as hardcoded hex and should be token-ified. But some inline styles use dynamic values (`style={ fmt.Sprintf(...) }`) — those are Go-string-interpolated and must NOT be naively swept. The audit in Pass 1 should flag them.

**Risk 3 — "almost-matching" colors.** A hex value might be close to but not identical to an existing token (say `#1a2230` vs `--color-void: #1b2232`). The correct move is to use the token and drop the two-digit variance — not to add a new token. The brand intent is one color; the codebase has drifted. Phase 1 re-converges.

**Risk 4 — Test snapshots.** If the repo has visual regression tests or CSS snapshot tests, token refactors can legitimately break them because the generated CSS bytes change. Update snapshots as part of Phase 1 — don't let that drift into Phase 2.

If Claude Code hits any of these, it should flag in findings §E and ask in §F. The recon discipline carries forward.
