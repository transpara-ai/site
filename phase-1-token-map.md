# Phase 1 — Token Map (Pass 1 audit)

**Date:** 2026-04-20
**Scope:** Hardcoded hex/rgba values in the CSS surface (`.templ`, `.css`, inline `<style>`) that shadow the existing 12-token `@theme` block.

## Existing tokens (source of truth, `views/layout.templ` @theme block)

| Token | Value |
|---|---|
| `--color-brand` | `#e8a0b8` |
| `--color-brand-dark` | `#d4899f` |
| `--color-void` | `#09090b` |
| `--color-surface` | `#111113` |
| `--color-elevated` | `#18181b` |
| `--color-edge` | `#1e1e22` |
| `--color-edge-mid` | `#2a2a2e` |
| `--color-edge-strong` | `#3a3a3f` |
| `--color-warm` | `#f0ede8` |
| `--color-warm-secondary` | `#c8c4bc` |
| `--color-warm-muted` | `#78756e` |
| `--color-warm-faint` | `#4a4844` |

## views/layout.templ

| file:line | current value | proposed token | notes |
|---|---|---|---|
| layout.templ:69 | `#18181b` | `var(--color-elevated)` | `.skeleton` bg, exact |
| layout.templ:96 | `rgba(232, 160, 184, 0.08)` | `color-mix(in srgb, var(--color-brand) 8%, transparent)` | ember-glow, brand rgb |
| layout.templ:96 | `rgba(232, 160, 184, 0.03)` | `color-mix(in srgb, var(--color-brand) 3%, transparent)` | ember-glow, brand rgb |
| layout.templ:107 | `#c8c4bc` | `var(--color-warm-secondary)` | prose body, exact |
| layout.templ:109 | `#f0ede8` | `var(--color-warm)` | prose h2, exact |
| layout.templ:110 | `#f0ede8` | `var(--color-warm)` | prose h3, exact |
| layout.templ:115 | `#e8a0b8` | `var(--color-brand)` | prose blockquote border, exact |
| layout.templ:115 | `#78756e` | `var(--color-warm-muted)` | prose blockquote text, exact |
| layout.templ:116 | `#18181b` | `var(--color-elevated)` | prose code bg, exact |
| layout.templ:116 | `#f0ede8` | `var(--color-warm)` | prose code text, exact |
| layout.templ:117 | `#09090b` | `var(--color-void)` | prose pre bg, exact |
| layout.templ:117 | `#c8c4bc` | `var(--color-warm-secondary)` | prose pre text, exact |
| layout.templ:117 | `#1e1e22` | `var(--color-edge)` | prose pre border, exact |
| layout.templ:119 | `#e8a0b8` | `var(--color-brand)` | prose link, exact |
| layout.templ:120 | `#d4899f` | `var(--color-brand-dark)` | prose link hover, exact |
| layout.templ:121 | `#1e1e22` | `var(--color-edge)` | prose hr, exact |
| layout.templ:123 | `#1e1e22` | `var(--color-edge)` | prose th/td border, exact |
| layout.templ:124 | `#111113` | `var(--color-surface)` | prose th bg, exact |
| layout.templ:124 | `#f0ede8` | `var(--color-warm)` | prose th text, exact |
| layout.templ:126 | `#f0ede8` | `var(--color-warm)` | prose strong, exact |

**Subtotal:** 18 hex values + 2 rgba literals. All map to existing tokens. **Zero new tokens needed.**

## graph/views.templ (CSS surface)

| file:line | current value | proposed token | notes |
|---|---|---|---|
| graph/views.templ:43 | `#18181b` | `var(--color-elevated)` | `.skeleton` bg in `themeBlock()`, exact |

**Subtotal:** 1 hex value. Exact match to existing token. **Zero new tokens needed.**

## graph/views.templ (Go helper functions — OUT OF SCOPE for Phase 1)

The following hex values appear inside Go functions `stateColorHex` (lines 5278–5293) and `priorityDotHex` (lines 5295–5308). They are returned as strings and interpolated into inline `style={ fmt.Sprintf(...) }` attributes. Per the prompt's Risk 2 note, dynamic Go-string values "must NOT be naively swept."

| file:line | current value | would-be-token | notes |
|---|---|---|---|
| graph/views.templ:5281 | `#78756e` | `var(--color-warm-muted)` | state open — maps to existing token |
| graph/views.templ:5283 | `#818cf8` | — | state active (indigo) — NO existing token; semantic state color |
| graph/views.templ:5285 | `#fbbf24` | — | state review (amber) — NO existing token; semantic state color |
| graph/views.templ:5287 | `#6ec89b` | — | state done (green) — NO existing token; semantic state color |
| graph/views.templ:5289 | `#4a4844` | `var(--color-warm-faint)` | state closed — maps to existing token |
| graph/views.templ:5291 | `#78756e` | `var(--color-warm-muted)` | state default — maps to existing token |
| graph/views.templ:5298 | `#e07070` | — | priority urgent (red) — NO existing token; semantic priority color |
| graph/views.templ:5300 | `#fbbf24` | — | priority high (amber) — duplicates state review |
| graph/views.templ:5302 | `#e8a0b8` | `var(--color-brand)` | priority medium — maps to existing token |
| graph/views.templ:5304 | `#78756e` | `var(--color-warm-muted)` | priority low — maps to existing token |
| graph/views.templ:5306 | `#78756e` | `var(--color-warm-muted)` | priority default — maps to existing token |

**Reason to defer:**
1. Four of these (indigo/amber/green/red) are semantic **state/priority** signals, not brand chrome. Adding 4 new tokens (`--color-state-active`, `--color-state-review`, `--color-state-done`, `--color-priority-urgent`) is a design decision, not a pure refactor — it commits Phase 1 to a "state colors live in @theme" choice that should be made with the profile-system design instead (will state colors need to differ across profiles?).
2. The prompt caps new tokens at 5 with a STOP signal. Even including just the 4 semantic ones pushes right to the cap and adds risk, given that the two skeleton/prose changes already demonstrate the pattern cleanly.
3. Per Risk 2 hint: Go-interpolated values are out-of-scope for a naive sweep. Flagging in findings §E for Phase 2 or a dedicated "semantic color tokens" task is the safer path.

**Action:** leave `stateColorHex` / `priorityDotHex` untouched; document in findings §E and §F.

## Summary

- **In-scope hex values to rewrite:** 19 (18 in layout.templ + 1 in graph/views.templ)
- **In-scope rgba literals to rewrite:** 2 (ember-glow, both mapped via `color-mix`)
- **New tokens required:** 0
- **Go-side hex constants deferred:** 11 call sites across 2 functions; design question for Phase 2+
- **STOP signal triggered:** No (0 new tokens, under the 5-row cap)
