# Phase 1 — Token Refactor Findings

**Version:** 0.1.0 · **Date:** 2026-04-20
**Author:** Claude (via Claude Code)
**Owner:** Michael Saucier
**Status:** Phase 1 complete — pending PR review
**Companion:** `prompt-1-phase-1-token-refactor-v0.1.0.md`, `phase-1-token-map.md`

## A. Audit results (Pass 1)

- **Total hex values found in scope:** 20 (19 hex + 2 rgba).
- **Files touched:**
  - `views/layout.templ` — 15 hex + 2 rgba literals (prose classes + .skeleton + ember-glow).
  - `graph/views.templ` — 1 hex (.skeleton in `themeBlock()`).
- **Map summary:** 20 exact matches to existing tokens, 0 new tokens required.
- **Additional hex found but deferred:** 11 values inside Go helper functions `stateColorHex` / `priorityDotHex` (`graph/views.templ` lines 5278–5308). See §E.
- **HTML-ID false positive:** `hx-target="#feed-items"` at `graph/views.templ:3131` — CSS selector, not a color.

Full map: `phase-1-token-map.md`.

## B. Token additions

**None.** All in-scope values mapped to one of the existing 12 tokens. Two rgba literals were converted to `color-mix(in srgb, var(--color-brand) N%, transparent)` per the prompt hint, which avoids introducing alpha-variant tokens.

## C. Edits applied (Pass 2)

| Commit | Subject | Files |
|---|---|---|
| `f11529e` | `refactor(layout): token-ify prose and skeleton chrome` | `views/layout.templ` (14 hex) |
| `4264727` | `refactor(layout): token-ify ember-glow rgba literals` | `views/layout.templ` (2 rgba → color-mix) |
| `a9b296d` | `refactor(graph/views): token-ify skeleton chrome` | `graph/views.templ` (1 hex) |

Generated `*_templ.go` files updated alongside each commit (templ regenerated after every edit, co-located in the same commit).

## D. Verification results (Pass 3)

- **templ generate:** pass. Generated files committed alongside each edit.
- **go build ./cmd/site/:** pass. No warnings.
- **go test ./...:** pass. All existing tests green (`auth`, `graph`, `handlers`, `cmd/gen-persona-status`).
- **hex-count grep** (`grep -rInE '#[0-9a-fA-F]{3,8}' --include='*.templ' --include='*.css' --include='*.html' cmd/ views/ graph/`): remaining matches comprise only:
  - 24 `@theme` token definitions (12 × 2 duplicate blocks — see §E).
  - 1 HTML ID selector `#feed-items` (false positive).
  - 11 hex values inside Go helper functions (deferred, see §E).
- **Out-of-scope files:** `git diff main -- auth/auth.go Dockerfile Makefile deploy.sh fly.toml` is empty. Confirmed untouched.
- **Visual regression:** **not empirically verified** in this session (no browser available). However, every substitution is an exact-value swap (token values are byte-identical to the replaced hex strings), and the two rgba literals map to `color-mix(in srgb, #e8a0b8 N%, transparent)` which is equivalent to `rgba(232, 160, 184, N/100)` in every mainstream browser that has shipped `color-mix()` (all evergreen browsers since ~2023). The changes are provably visually null. A manual screenshot pass by the reviewer is still recommended before merge.

## E. Surprises and judgment calls

1. **Duplicate `@theme` block.** `graph/views.templ` declares its own `@theme` block inside `themeBlock()` (lines 19–34) that mirrors the one in `views/layout.templ` (lines 35–49) exactly. This drift risk is pre-existing. Phase 1 left both blocks intact; a later phase should either extract to a shared fragment or drop the duplicate once the Tailwind build step (Phase 2) centralizes token resolution.

2. **Go-interpolated hex constants deferred.** `graph/views.templ` contains two Go helpers — `stateColorHex` and `priorityDotHex` — whose return values are interpolated into inline `style={ fmt.Sprintf(...) }` attributes. Five callers across the file. The 11 hex values returned include:
   - **6 that match existing tokens** (`#78756e`, `#4a4844`, `#e8a0b8`).
   - **4 semantic state/priority colors that have no existing token** (`#818cf8` indigo active, `#fbbf24` amber review/high, `#6ec89b` green done, `#e07070` red urgent).
   - Per the prompt's Risk 2 note, Go-string-interpolated values "must NOT be naively swept." These were left untouched for Phase 1. The right home for state/priority colors — dedicated `--color-state-*` / `--color-priority-*` tokens vs. semantic CSS classes vs. a separate palette — is a design question that should be answered alongside the profile-swap machinery in Phase 3+, not decided mid-refactor.

3. **rgba → color-mix choice.** The ember-glow rgba literals (`rgba(232, 160, 184, 0.08)` and `..., 0.03)`) were rewritten as `color-mix(in srgb, var(--color-brand) 8%/3%, transparent)` rather than introducing brand-alpha tokens. This keeps the `@theme` block lean, as suggested. `color-mix(in srgb, ...)` produces the identical sRGB output as `rgba()` with the same alpha; no visual change.

4. **`views/layout.templ:130+` uses Tailwind utility classes (`bg-void`, `text-warm-secondary`, `border-edge`, `text-brand`, etc.)** — these are already token-backed via the `@theme` block and required no changes. Phase 1 left them alone, confirming the refactor scope was correctly identified.

## F. Open questions for Michael

1. **Are state/priority colors (`#818cf8`, `#fbbf24`, `#6ec89b`, `#e07070`) brand-level or semantic-level?** If they should follow profile swaps, they need `--color-state-*` tokens in `@theme` — defer to Phase 3 profile design. If they should stay constant across all profiles (universal semantic signals), they could live as Go constants or a separate non-profile CSS layer.
2. **Should the duplicate `@theme` block in `graph/views.templ` be deduplicated now or as part of Phase 2 (Tailwind build step)?** Phase 2's proper build pipeline is a natural home for centralized token definition.
3. **Visual regression verification** — I could not run browser screenshots in this session. Is manual review + the exact-value-swap argument in §D sufficient, or do you want a follow-up commit with screenshots before merge?

---

## Five-line summary

1. Files touched: 2 (`views/layout.templ`, `graph/views.templ`) + generated `*_templ.go`; ~18 lines changed net.
2. Token additions: none.
3. Biggest surprise: duplicate `@theme` block in `graph/views.templ` mirrors `views/layout.templ` — pre-existing drift risk to address later.
4. Verification status: all-pass (templ generate, go build, go test); visual regression not empirically run (browser unavailable) but provably null by exact-value-swap + color-mix equivalence.
5. Ready for Phase 2: yes — blocked only by reviewer sign-off.
