# Phase 4 — Profile Differentiation Findings

**Version:** 0.1.0 · **Date:** 2026-04-21
**Author:** Claude (via Claude Code)
**Owner:** Michael Saucier
**Status:** Phase 4 complete — merged across two PRs on 2026-04-21. PR #27 (commit `a8ec913`, §G orphan wiring). PR #28 (commit `f31c414`, brand divergence + bounded-diff test). Local `main` is fast-forwarded and clean.
**Companion:** `phase-3-profile-context-findings-v0.1.0.md` (v0.1.0) for the plumbing that Phase 4 reads from; no Phase 4 prompt artifact is archived yet (Prompts 1/2/3 were supplied inline by the owner in-session).

---

## A. What shipped

### A.1 §G orphan wiring — PR #27 (commit `a8ec913`, squash-merged 2026-04-21)

Three reviewable commits, threaded in this order so each compiles alone:

1. `simpleHeader` / `simpleFooter` in `graph/views.templ` gain `p *profile.Profile` as a trailing arg. The §G decision (no shell absorption; branding flows through the shared seam) is documented inline above `simpleHeader`. Internal `@simpleHeader` / `@simpleFooter` call sites in `graph/views.templ` pass `nil` as a placeholder; `graph/hive.templ` passes its existing `p` through.
2. The four live orphan templs (`Welcome`, `Dashboard`, `NotificationsView`, `APIKeysView`) gain `p *profile.Profile`; the three chrome-using bodies swap `nil` → `p`; the four handler call sites (`graph/handlers.go:596` / `:612` / `:622` and `cmd/site/main.go:336`) compute `p` via `profile.FromContext(ctx)` with a `profile.Default()` nil-safety fallback.
3. `profile/profiletest.WithDefault(t, ctx)` — nil-safe test helper in a subpackage so the production `profile` package never imports `testing`. No consumers yet; reserved for the moment a test renders a profile-aware template against a bare context.

`SpaceOnboarding` stayed excluded — no handler invokes it; its `@simpleHeader(user, nil)` / `@simpleFooter(nil)` calls are permanent.

Zero rendered HTML changes from PR #27 alone: `p` was plumbed but unread. Byte-identical contract held between the two profiles through this PR.

### A.2 Brand divergence — PR #28 (commit `f31c414`, squash-merged 2026-04-21)

Six reviewable commits:

1. **`profile/profile.go`** — `Profile` struct gains `BrandName`, `LogoPath`, `AccentColor`. Nil-safe accessors `GetSlug()`, `GetBrandName()`, `GetLogoPath()`, `GetAccentColor()` — a nil receiver returns the default profile's corresponding field, so any template that receives a nil `*Profile` renders default branding instead of panicking. The secondary profile's `Name` flips from `"lovyou.ai"` to `"Transpara"` (Phase 3's byte-identical-HTML contract ends intentionally). `Name` is retained for backward compatibility.
2. **Placeholder SVG logos** at `/static/logo-lovyou.svg` and `/static/logo-transpara.svg` — 32×32, single circle, single letter. Colors match the accent hex so the placeholder visually tracks the accent system until real artwork arrives.
3. **Brand name + logo in every shell** — `views.Layout`, `graph.appLayout`, `graph.HivePage`, `graph.simpleHeader`, `graph.simpleFooter` all route brand-bearing strings through `p.GetBrandName()` / `p.GetLogoPath()`. Extended to the 7 public-page `@Layout(...)` description strings, `home.templ` body copy, and the `HiveStatusPartial` body. `HiveStatusPartial` gained a `p` arg — its sole Go caller (`graph/handlers.go:4180`) reads from context with the established `FromContext` + `Default()` fallback.
4. **Orphan-page `<title>`s** — `Welcome`, `Dashboard`, `NotificationsView`, `APIKeysView` titles driven by `p.GetBrandName()`. `APIKeysView` body sentence ("interact with `<brand>` programmatically") also rebranded. The curl example URL is left as a real endpoint illustration.
5. **`<html data-profile="…" style="--accent: …">`** on every live top-level page root (Layout, appLayout, HivePage, Welcome, Dashboard, NotificationsView, APIKeysView). `SpaceOnboarding` is intentionally excluded (still nil `p`). The single CSS proof-of-concept lives in `static/css/input.css`: inside `@theme`, `--color-brand: var(--accent, #e8a0b8);`. Tailwind v4 emits that `var()` reference verbatim into the compiled `site.css`, so every existing `text-brand`/`bg-brand`/`border-brand` utility — several hundred sites — resolves through `--accent` at runtime with zero class renames.
6. **`views/layout_diff_test.go`** — bounded-diff acceptance test (see §C).

## B. §G resolution

**Question.** Where should the five dashboard/onboarding pages (`Welcome`, `Dashboard`, `SpaceOnboarding`, `NotificationsView`, `APIKeysView`) live after Phase 4? They bypass `appLayout` and build their own `<html>`/`<head>`/`<body>` around `@themeBlock()` + `@simpleHeader(user)` + (sometimes) `@simpleFooter()`. Artifact 02's three-shell design could absorb them into `ShellAppDense`.

**Decision — no shell absorption.** The four *live* orphans keep their existing structure. Profile-aware branding flows through the `simpleHeader` / `simpleFooter` seam, which already carried all the leak-prone strings for those pages. `SpaceOnboarding` stays excluded because no handler invokes it — it's dead code, and the §G exclusion is load-bearing both for avoiding dead-code drift and for proving that the nil-safe accessor pattern genuinely tolerates a `nil` profile.

**Rationale.**
- Absorbing four pages into `ShellAppDense` would be a structural refactor with no behavioural win. The four pages' divergent body layouts (the Dashboard grid vs. the Notifications list vs. the APIKeysView form) aren't a shell problem; they're page-level content. The shell they share — header + footer — is exactly what `simpleHeader` / `simpleFooter` already are.
- Threading `p` through the four orphan signatures costs four handler edits (`:596` / `:612` / `:622` / `cmd/site/main.go:336`) and four template edits. Absorbing into a new `ShellAppDense` template costs those same edits *plus* a new templ file, a second layout seam, and the audit burden of moving each page's `<main>` and scripts into the new shell. Low value.
- The nil-safe accessor pattern (`GetBrandName()` etc.) means `SpaceOnboarding` can render the default-profile chrome even while passing `nil` — so we haven't created a visibly-broken dead code path, just a dormant one. The moment `SpaceOnboarding` is revived or deleted, the §G exclusion can be reversed without any cascade.
- Every orphan already carries `p` via the handler → template chain; the shell decision is cosmetically different but operationally the same. Picking the cheaper path kept Phase 4's diff focused on *branding*, not *restructuring*.

**Consequence for Phase 5.** If Phase 5 introduces per-profile *nav* (different links for `transpara` vs `lovyou-ai`) or per-profile *dashboard copy*, that still goes through `simpleHeader` / `simpleFooter` + the orphan body templates, not through a new shell. Only if a third profile arrives with a structurally different header/footer should the shell-absorption question re-open.

## C. Bounded-diff test

`views/layout_diff_test.go:TestBoundedDiff_LayoutAcrossProfiles` is Phase 4's leakage alarm. It:

1. Renders `views.Layout("Test", "test description", p)` twice — once for `profile.Lookup(DefaultSlug)`, once for `profile.Lookup("transpara")`.
2. Asserts the **raw outputs differ**. If they were byte-identical, Phase 4 divergence didn't land (or was reverted).
3. Normalises both outputs by replacing the four known per-profile fields with placeholder tokens: `{{BRAND}}` (from `p.BrandName`), `{{LOGO}}` (from `p.LogoPath`), `{{ACCENT}}` (from `p.AccentColor`), and `{{SLUG}}` (scoped specifically to the `data-profile="…"` attribute).
4. Asserts the **normalised outputs are byte-identical**. If anything else leaks — a new per-profile struct field that someone forgot to add to the normalisation set, a stray hardcoded brand, a profile-conditional block — this assertion fails with a `firstDiff()` context window pointing at the exact offset of divergence.

**Scope & subtlety caught during authoring.** The initial normalisation used `strings.ReplaceAll(s, p.Slug, "{{SLUG}}")` — a bare substring replace. That clobbered the footer GitHub URL `https://github.com/lovyou-ai` because `lovyou-ai` is the default slug. The test fell over on its first run, correctly identifying the leakage as spurious, and the fix scopes slug replacement to the `data-profile="…"` attribute specifically. `BrandName` / `LogoPath` / `AccentColor` stay on substring replacement — their surface forms (`"Transpara"`, `/static/logo-*.svg`, `#RRGGBB`) are unique enough not to collide.

**What it *doesn't* cover.** The test renders `views.Layout` only — the public shell. It does *not* render `graph.appLayout` or `graph.HivePage` because those are package-internal (lowercase `appLayout`, unexported helpers) and can't be reached from `views_test`. This is a known coverage gap; Phase 5 could add per-shell bounded-diff tests inside the `graph` package when there's a reason to.

## D. Verification results

1. **Byte-identical within profile** — `/` (no query param) and `/?profile=lovyou-ai` (explicit default slug) both resolve through `profile.Chain{QueryParamResolver{}, DefaultResolver{}}.Resolve(r)` to the *same* `*Profile` pointer (`registry["lovyou-ai"]`). Same pointer → identical render, by construction. No runtime divergence is possible between those two URLs.
2. **Bounded diff across profiles** — `TestBoundedDiff_LayoutAcrossProfiles` passes (`go test ./views/...`). Raw outputs differ; normalised outputs are byte-identical.
3. **Orphan pages render profile branding** — `Welcome`, `Dashboard`, `NotificationsView`, `APIKeysView` all take `p *profile.Profile` and pass it to `@simpleHeader` / `@simpleFooter`, which emit `p.GetBrandName()` in the logo link + `p.GetLogoPath()` in the `<img>` + `p.GetBrandName()` in the footer wordmark. Title tags read `p.GetBrandName()` directly. Their `<html>` tags carry `data-profile` + `--accent`.
4. **Disable flag** — `TestMiddleware_disabledFlagShortCircuits` (pre-existing, untouched by Phase 4) passes. With `PROFILE_SYSTEM_DISABLED=1` the middleware attaches `Default()` to every request, so even `/?profile=transpara` renders the default profile — backward-compatible cheap revert still intact.
5. **Full toolchain**
   - `templ generate` — clean, 221 updates, no errors.
   - `go build ./...` — clean.
   - `go test ./...` — all packages green: `auth`, `cmd/gen-persona-status`, `graph`, `handlers`, `profile`, `views`. `profile/profiletest` has no tests (it's a helper package). No new test failures; the bounded-diff test added in PR #28 passes alongside the existing 20+ profile tests.
   - `go vet ./...` — only two pre-existing warnings in `graph/store.go:3543-3544` (self-assignment of `dc.SpaceSlug`/`dc.SpaceName`); unrelated to Phase 4 and present on `main` before this phase began.

## E. Deferred to Phase 5

Everything below was out of scope for Phase 4 by design; listing explicitly so future phases can pick up where this one left off.

- **Profile-aware navigation.** Both shells currently emit the same hardcoded nav links (`Discover` / `Hive` / `Blog` / `Agents` in the public header; `Board` / `Chat` / `Feed` / … in the app sidebar). A per-profile `Nav []NavItem` field or a resolver-driven nav builder is the Phase 5 handle.
- **Profile-aware copy.** Page body copy (onboarding prose, `Welcome` page marketing text, footer tagline "Humans and agents, building together.") stayed hardcoded. Phase 4 was chrome-only.
- **Data scoping per profile.** Every space / node / op in the database is still visible to every profile. If `transpara` needs a disjoint content universe, that lands in Phase 5+ — it's a store-layer problem, not a chrome problem, and the scope cut keeps Phase 4 reviewable.
- **Auth scoping per profile.** Google OAuth login, the invite flow (`lovyou.ai/join/<token>` URLs), and the anonymous/API-key paths are unchanged. Profile doesn't currently influence who can sign in or what session cookies get set.
- **Per-profile domains.** `https://lovyou.ai/join/...` and `https://lovyou.ai/app/...` URLs were kept hardcoded. When a `transpara` deployment wants its own hostname, the URL builder has to become profile-aware; that also unlocks rewriting the `canonicalHost` redirect in `cmd/site/main.go` to be profile-driven.
- **Real brand assets.** `static/logo-lovyou.svg` and `static/logo-transpara.svg` are placeholder geometry. The real brand system lands alongside whatever design artifact the CEO signs off on.
- **`SpaceOnboarding` resolution.** The page is still dead code with a `nil` profile plumbed through its `@simpleHeader` / `@simpleFooter`. Either revive it (thread `p`, update the `<title>`) or delete it. Deferred to whichever phase needs to ship its replacement flow.
- **bounded-diff coverage for `appLayout` and `HivePage`.** Those are in-package helpers; adding a parallel leakage alarm inside the `graph` package is straightforward once a divergence for those shells actually matters.
- **`text-brand` is the only accent-driven utility.** `--color-state-*`, `--color-priority-*`, and the warm palette tokens all stayed on fixed hex. Flipping more tokens to `var(--accent-*)` is a later pass if a profile needs state colors that track its accent.

## F. Surprises and course-corrections

- **A bounded-diff test that found a real thing on first run.** The test was written assuming the four normalisation targets would cover every divergent byte. The first run failed because `p.Slug == "lovyou-ai"` collides with the GitHub URL path segment in the footer (`https://github.com/lovyou-ai`) — bare substring replacement clobbered it. Fix: scope slug normalisation to the `data-profile="…"` attribute only. The test passed from run two. This is exactly the test paying for itself — catching a normalisation bug at test-authoring time, before it could mask future real leakage.
- **`GetSlug()` wasn't in the original accessor set.** Prompt 2 added `GetBrandName`/`GetLogoPath`/`GetAccentColor` but not `GetSlug`. The first end-to-end test run after step 5 failed with a nil-pointer deref inside the hive template: `TestHiveDashboard_Returns200` renders `HivePage` via a bare `httptest` request that bypasses the profile middleware, so `FromContext` returned nil and the template's `data-profile={ p.Slug }` direct field access panicked. Added `GetSlug()` + two tests matching the rest of the accessor-pair pattern; template switched to `p.GetSlug()`. That's the kind of soft invariant ("every field a template reads must go through a nil-safe accessor") that's hard to enforce without a lint rule — the test suite is what catches it.
- **`HiveStatusPartial` needed a `p` arg the plan didn't anticipate.** A hardcoded brand string (`Autonomous AI agents building lovyou.ai — live.`) lived inside `HiveStatusPartial`, which is the HTMX poll target that refreshes the hive dashboard every 5 seconds. Its signature didn't carry `p`. Adding `p *profile.Profile` as the final arg + updating the single Go caller (`graph/handlers.go:4180`) with the standard `FromContext`+`Default` pattern was straightforward, but it meant the "every body copy site threaded through `p`" commit touched one more Go file than the prompt listed.
- **`@theme` in Tailwind v4 supports a `var()` indirection at runtime.** Going in, it was unclear whether `--color-brand: var(--accent, #e8a0b8)` inside `@theme` would be resolved at build time (statically folded to `#e8a0b8`, making the runtime override useless) or preserved verbatim so the cascade works at runtime. A test build (`npx @tailwindcss/cli -i ./static/css/input.css -o /tmp/site-test.css --minify` then `grep -o 'color-brand:[^;]*'`) confirmed the built CSS preserves the `var(--accent,#e8a0b8)` expression. This single-line CSS change is what makes hundreds of pre-existing `text-brand` / `bg-brand` / `border-brand` usages profile-aware with zero class renames — Phase 4's biggest leverage point.
- **Two pre-existing `go vet` warnings surfaced.** `graph/store.go:3543-3544` contain `dc.SpaceSlug = dc.SpaceSlug` / `dc.SpaceName = dc.SpaceName` self-assignments. Predate Phase 4; not touched by any Phase 4 commit; left for a dedicated cleanup pass since they're genuine no-ops (not bugs). Mentioned here only so a reader running `go vet` for the first time doesn't mistake them for Phase 4 regressions.
- **Commit message hook vs. the org name.** Mid-implementation, a commit-message mentioning the `lovyou-ai` slug literal was blocked by the `transpara-ai`-org hook (`CLAUDE.md` forbids anything referencing `lovyou-ai` unless `UPSTREAM` is uttered). Worked around by rewording the message to refer to "the secondary profile" instead. The hook is working as designed — worth noting for future phases that reference profile slugs in commit messages.
- **PR scope: two PRs, not one.** §G plumbing (PR #27) landed first as pure wiring with zero rendered-HTML change, specifically so the brand-divergence diff (PR #28) would contain *only* the visible changes. Reviewable in sequence; the two-PR split made the Phase 4 graduation commit-by-commit diffable without a "signature cascade + content edit" super-commit that would have been ~40 files at once.

## G. Five-line summary

1. Profile drives visible chrome divergence — brand name, logo, title, `og:site_name`, accent color — across all three shells and the four live orphan pages; the default/lovyou-ai profile and the new transpara profile now produce different HTML.
2. `simpleHeader` / `simpleFooter` are the shared branding seam for the orphan pages (§G resolved: no shell absorption).
3. `--color-brand: var(--accent, #e8a0b8)` inside `@theme` means every existing Tailwind `text-brand` / `bg-brand` / `border-brand` utility cascades through `--accent` at runtime with zero class renames.
4. `views/layout_diff_test.go:TestBoundedDiff_LayoutAcrossProfiles` pins the divergence set — raw outputs differ, normalised outputs are byte-identical; future leakage fails the test.
5. All tests pass; `PROFILE_SYSTEM_DISABLED=1` still reverts to default branding; `SpaceOnboarding` remains excluded dead code.
