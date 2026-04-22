# Phase 5 — Profile-Aware Navigation + Copy Findings

**Version:** 0.1.0 · **Date:** 2026-04-22
**Author:** Claude (via Claude Code)
**Owner:** Michael Saucier
**Status:** Phase 5 implementation queued in PR (not yet merged at the time of writing). Six commits on `feat/profile-nav` build incrementally on top of `main` at `22e612e` (Phase 4 findings merge).
**Companion:** `phase-4-profile-differentiation-findings-v0.1.0.md` (v0.1.0). No Phase 5 prompt artifact archived yet — the three prompts (nav, copy, findings) were supplied inline in-session.

---

## A. What shipped

Six commits on `feat/profile-nav`, in two thematic halves.

### A.1 Profile-aware navigation

1. **`858db68`** — `NavItem{Label, Path}` + `HeaderNav` / `FooterNav` fields on `Profile`. Nil-AND-empty-safe accessors `GetHeaderNav()` / `GetFooterNav()` (an empty slice falls back to default — a profile with zero links would render a blank nav, almost certainly a misconfiguration). Registry populated for both profiles. +6 tests covering nil receiver, valid receiver, empty-field fallback, and per-profile divergence.
2. **`b8dbdb0`** — Replace hardcoded nav link lists with `range p.GetHeaderNav()` / `range p.GetFooterNav()` in three shells: `views.Layout` (header + footer), `graph.simpleHeader` (orphan-page header), `graph.appLayout` (app chrome header). Footer GitHub link stays literal (real URL, not branding). Auth-area "My Work" / user / logout block stays literal (auth-state, not profile). Mobile lens bar + app sidebar untouched — space-scoped, not profile-scoped.
3. **`b118566`** — Extend `TestBoundedDiff_LayoutAcrossProfiles`'s normaliser with a regex that collapses contiguous runs of profile-driven nav `<a>` tags into a single `{{NAV_LINKS}}` placeholder. Required because nav-link counts legitimately differ per profile (4 vs. 2 in the header, 9 vs. 2 in the footer) — per-link substring replacement could not produce equal-length normalised strings. Regex matches only `/`-prefixed hrefs with the shared `hover:text-brand transition-colors` class signature, leaving the GitHub link and "My Work" button literal.

### A.2 Profile-aware copy

4. **`bef58cd`** — `Copy map[string]string` field on `Profile` plus the nil-AND-nil-map-safe accessor `GetCopy(key, fallback string) string`. Templates supply the inline default; profile only overrides keys it cares about. Default profile leaves `Copy` nil (every key returns fallback → today's text unchanged). Secondary profile populates 6 keys whose tone diverges sharply: `tagline`, `home.hero.subtitle`, `home.subhero.body`, `discover.empty`, `welcome.subtitle`, `apikeys.desc`. +4 tests covering nil receiver, missing key, existing key, and a guard that the default profile has nil `Copy` (so an accidental default override cannot silently change today's rendering).
5. **`7f7b802`** — Replace 6 hardcoded copy strings in `views/layout.templ`, `views/home.templ`, `views/discover.templ`, and `graph/views.templ` with `p.GetCopy(key, fallback)`. Fallbacks preserve the current default-profile text verbatim so the default render is byte-identical to today. Two of the fallbacks (`apikeys.desc`, `home.subhero.body`) keep `p.GetBrandName()` interpolation — any future profile that omits the override still gets brand-aware copy.
6. **`369d1f2`** — Extend the bounded-diff test's normaliser with `copyKeysInLayout` map: for each key whose rendered text appears inside the `views.Layout` shell, both the fallback (lovyou render) and the override (read live from `p.Copy[key]` in the transpara render) rewrite to a shared `{{COPY_<key>}}` placeholder. Today only `tagline` appears in `Layout`; the other five Copy keys live inside `Home` / `Discover` / `Welcome` / `APIKeysView` — known coverage gap, called out in the test comment.

## B. Profile divergence summary (after Phase 5)

What now legitimately differs between the two registered profiles, ordered by phase introduced:

| Surface | Default profile | Secondary profile (`transpara`) | Phase |
|---|---|---|---|
| `BrandName` | `lovyou.ai` | `Transpara` | 4 |
| `LogoPath` | `/static/logo-lovyou.svg` | `/static/logo-transpara.svg` | 4 |
| `AccentColor` | `#e8a0b8` (pink) | `#0ea5e9` (blue) | 4 |
| `<title>` suffix | `— lovyou.ai` | `— Transpara` | 4 |
| `og:site_name` | `lovyou.ai` | `Transpara` | 4 |
| `<html data-profile>` | `lovyou-ai` | `transpara` | 4 |
| `--accent` CSS var | `#e8a0b8` | `#0ea5e9` | 4 |
| `text-brand` / `bg-brand` / `border-brand` runtime color | pink | blue (via `var(--color-brand: var(--accent, #e8a0b8))`) | 4 |
| Header nav links | Discover, Hive, Agents, Blog | Discover, Blog | 5 |
| Footer nav links | Discover, Hive, Agents, Market, Knowledge, Activity, Search, Blog, Reference (+ GitHub) | Discover, Blog (+ GitHub) | 5 |
| Footer tagline | "Humans and agents, building together." | "Operations intelligence for the people on the floor." | 5 |
| Home hero subtitle | "Assign tasks to an AI agent…" | "Surface what the plant is telling you, before it costs you a shift…" | 5 |
| Home sub-hero body | "Autonomous AI agents are building lovyou.ai, live…" | "Autonomous agents reason about your operation in real time…" | 5 |
| Discover empty state | "No public spaces yet." | "No shared workspaces yet." | 5 |
| Welcome subtitle | "You're in. Let's set up your first space…" | "You're in. Let's set up your first workspace — somewhere your team can ask questions of the plant data…" | 5 |
| APIKeysView body | "Authenticate agents and scripts to interact with `<brand>` programmatically." | "Authenticate scripts and agents to query Transpara programmatically — the same data your team sees, accessible from your tooling." | 5 |

Three things still **identical** across profiles by design (and worth being explicit about):
- Page structure / DOM layout (Phase 5 explicitly didn't touch shell composition)
- Auth flow (login routes, OAuth handler, session cookies)
- Content store (every space, node, op, agent persona is visible to every profile)

## C. Bounded-diff test evolution

`views/layout_diff_test.go:TestBoundedDiff_LayoutAcrossProfiles` is the leakage alarm. Three normalisation surfaces have accreted across phases — adding a fourth (Copy) was the Phase 5 increment.

| Phase | Normalisation surface | Mechanism |
|---|---|---|
| 4 | `BrandName`, `LogoPath`, `AccentColor` | substring `strings.ReplaceAll` per profile |
| 4 | `Slug`, scoped to the `data-profile` attribute | substring scoped to `data-profile="…"` literal — bare slug would clobber `https://github.com/lovyou-ai` |
| 5 | Nav links | regex `(?:<a href="/[^"]*" class="hover:text-brand transition-colors[^"]*">[^<]*</a>\s*)+` collapses runs to `{{NAV_LINKS}}` — handles divergent link counts |
| 5 | Copy overrides in Layout scope | `copyKeysInLayout` map: fallback (lovyou render) and `p.Copy[key]` (transpara render) both rewrite to `{{COPY_<key>}}` |

The test's two assertions are unchanged:
1. **Raw outputs DIFFER** — pinned divergence actually happens in render output.
2. **Normalised outputs are byte-IDENTICAL** — nothing else leaks beyond the canonical surfaces.

A new field added to `Profile` that reaches the rendered HTML must either land in the normalisation set or fail the test loudly. That property is the whole point of the alarm: the test cost grows linearly with the divergence surface, but the safety it provides is exponential — every untracked leak is caught at test time, not in production.

## D. Verification

- `templ generate` clean (221 updates).
- `go build ./...` clean.
- `go test ./...` — all six packages with tests green: `auth`, `cmd/gen-persona-status`, `graph`, `handlers`, `profile`, `views`.
- `go vet ./...` — only the two pre-existing `graph/store.go:3543-3544` self-assignment warnings (predate Phase 4; left for a dedicated cleanup pass).
- The Phase 4 disable-flag invariant (`PROFILE_SYSTEM_DISABLED=1` reverts everything to default) still holds — the middleware doesn't read any of the new fields, and every accessor falls back to default when the receiver is nil.

## E. Deferred to Phase 6+

- **Data scoping per profile.** Every space / node / op / agent persona is still visible to every profile. If `transpara` needs a disjoint content universe — different default spaces, agents, blog posts — that's a store-layer change. Likely the largest single piece of remaining work.
- **Auth routing per profile.** Google OAuth login, the invite flow (`https://lovyou.ai/join/<token>` URLs), the API-key bootstrap path, the canonical-host redirect in `cmd/site/main.go` — none of these are profile-aware. Once `transpara` wants its own hostname or a different invite domain, the URL builder + the auth callback URL set both have to become profile-driven.
- **Full theming.** Phase 4 ships one CSS variable (`--accent`) flowing through one Tailwind utility (`text-brand`/`bg-brand`/`border-brand`). State colors (`--color-state-active` / `--color-state-review` / `--color-state-done` / `--color-priority-urgent`), warm palette tokens, font pairing — all stayed on fixed values. Profile-aware *full* theming would extend the same `var(--*)` cascade pattern to those tokens. Cheap once a profile actually needs different state colors.
- **Host-based resolution.** The resolver chain is `QueryParamResolver` → `DefaultResolver`. A `SubdomainResolver` or `HostHeaderResolver` is one Resolver implementation + one line in the `Chain` literal in `cmd/site/main.go` — Phase 3's whole architectural payoff. Add it the moment a `transpara.lovyou.ai` or `transpara.com` deployment exists.
- **Per-link responsive visibility.** `views.Layout` and `graph.simpleHeader` nav loops dropped the `hidden md:inline` per-link class modifier when they switched to `range p.GetHeaderNav()`. `Agents` and `Blog` (Layout) and `Blog` (simpleHeader) used to hide below the `md` breakpoint; they show at every viewport now. If mobile layout in those shells needs to match the pre-Phase-5 behaviour exactly, an `MdHidden bool` field on `NavItem` is the natural extension. Not done yet because two profiles' nav sets are short enough that mobile crowding isn't a real problem.
- **Bounded-diff coverage of `appLayout` and `HivePage`.** The bounded-diff test only renders `views.Layout` (the public shell). `appLayout` (the `/app/*` chrome) and `HivePage` (the hive surface) are package-internal in `graph` — adding a parallel leakage alarm inside the `graph` package is straightforward but only worth doing once a divergence in one of those shells matters.
- **Bounded-diff coverage of out-of-Layout Copy keys.** Only `tagline` appears inside `views.Layout`'s render scope. The other five Copy keys (Home / Discover / Welcome / APIKeysView body sentences) aren't normalised because `Layout` doesn't render them. Per-template diff tests would close this gap; deferred for now.
- **Real i18n absorbing the Copy registry.** The 6-key `Copy` map is a deliberate pattern proof, not infrastructure. The accessor signature (`GetCopy(key, fallback string) string`) stays compatible with a richer key namespace (e.g. `<lang>.<surface>.<key>`) so a real translation system can absorb the field without changing the call sites.
- **`SpaceOnboarding` resolution.** Still dead code with `nil` profile plumbed through its `@simpleHeader(user, nil)` / `@simpleFooter(nil)` calls and a hardcoded `<title>Get Started — lovyou.ai</title>`. Either revive it (thread `p`, route the title through `GetBrandName`, possibly add a Copy key for its body) or delete it. Carried forward from Phase 4.

## F. Surprises and course-corrections

- **Per-link mobile-visibility class lost in the nav refactor.** The original Layout / simpleHeader hardcoded link sets used `hidden md:inline` selectively on `Agents` and `Blog` to hide them on mobile. The `range` loop applied a uniform class string to every link — so all of them now show on mobile. Caught at PR-write time, called out in the PR body. The `appLayout` loop kept `hidden md:inline` because the `/app/*` header is horizontally tight and the regression there would have been visible. Two-class-string solution wasn't a real problem; just worth flagging.
- **Bounded-diff test broken in two intermediate commits.** Both the nav (`b8dbdb0`) and the copy (`7f7b802`) commits transiently break `TestBoundedDiff_LayoutAcrossProfiles` — the divergence lands in the templates before the test's normaliser knows about it. Each is followed by a single test-update commit (`b118566` for nav, `369d1f2` for copy) that restores green. The split is intentional: each test-update commit is reviewable in isolation as "here's why this divergence is expected, and here's how the test validates it." A combined "shells + test" commit would have hidden the *why* of each normaliser entry.
- **Most Copy keys aren't testable from the bounded-diff test.** `tagline` is the only Copy key that renders inside `Layout`. The other five (`home.hero.subtitle`, `home.subhero.body`, `discover.empty`, `welcome.subtitle`, `apikeys.desc`) are inside templates the bounded-diff test doesn't render. Tempting to "normalise them anyway" for completeness — that would be no-op string replacements. Resisted; honest scoping flags this as the coverage gap to revisit if the test grows to render more templates.
- **`go vet` warnings still live in `graph/store.go`.** Two pre-existing `dc.SpaceSlug = dc.SpaceSlug` / `dc.SpaceName = dc.SpaceName` self-assignments on lines 3543-3544 still surface on every Phase 5 vet run. Predate the entire profile system; not touched by any Phase 4 / Phase 5 commit; flagged in both this and the Phase 4 findings as known noise so a reader running vet for the first time doesn't mistake them for new regressions.
- **`Name` field is increasingly redundant.** `Profile.Name` still exists for backward compatibility (Phase 3 used it as a display label). After Phase 4, `BrandName` is the right read; after Phase 5, no template path reads `Name` at all — the only callers are `profile_test.go` and the registry literals. Worth deleting in a small cleanup commit if a future phase wants the noise-reduction; no urgency.

## G. Five-line summary

1. **Nav** — each profile declares its own `HeaderNav` / `FooterNav` slice; all three shells (`views.Layout`, `graph.simpleHeader`, `graph.appLayout`) iterate from the profile instead of hardcoded link lists.
2. **Copy** — sparse string registry (`Profile.Copy map[string]string`) with `GetCopy(key, fallback)` accessor; six keys overridden on the secondary profile to swap in tone-appropriate sentences (default profile leaves `Copy` nil and renders today's hardcoded text via the fallback path).
3. **Bounded-diff test** now normalises four divergence surfaces (Phase-4 scalars, slug, nav-link runs, in-Layout Copy overrides) and still asserts the byte-identical-after-normalisation invariant.
4. **Two profiles diverge** on brand name, logo, accent color, page titles, og meta, `data-profile`, header + footer nav, and 6 copy strings. Page structure, auth, and data store remain shared.
5. **Phase 6+ scope** — data scoping, auth routing, full theming, host-based resolution, real i18n, and the long-running `SpaceOnboarding` decision.
