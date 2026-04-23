# Profile Migration Audit — 2026-04-22

**Version:** 0.1.0 · **Date:** 2026-04-22
**Author:** Claude (via Claude Code)
**Owner:** Michael Saucier
**Status:** Post-Phase-5 audit. `main` is at commit `22e612e` (Phase 4 findings merge). Phase 5 implementation queued in open PR #30 (`feat/profile-nav`). Audit ran from `main` with a fresh `git checkout main && git pull --ff-only`, so the findings here reflect what ships today, not what's on the feat branch.
**Companion:** `phase-1-token-refactor-findings-v0.1.0.md`, `phase-2-tailwind-build-step-findings-v0.1.0.md`, `phase-3-profile-context-findings-v0.1.0.md`, `phase-3-profile-context-plan.md`, `phase-4-profile-differentiation-findings-v0.1.0.md`. Phase 5 findings live on branch `feat/profile-nav` and will land when PR #30 merges.

---

## Open PRs

| Repo | PR | Title | Status |
|---|---|---|---|
| `transpara-ai/site` | [#30](https://github.com/transpara-ai/site/pull/30) | Phase 5: profile-aware navigation + copy | Open, CI green, code-review comment posted flagging `simpleFooter` omission — awaits merge |
| `transpara-ai/hive` | #74 | docs(debt): track JSONB expression index follow-up | Open — unrelated to migration (debt tracker from PR #73) |
| `transpara-ai/summary` | #49 | fix(dashboard): light phase dots for checkpoint-verifier and task-curator | Open — unrelated to migration |
| `transpara-ai/work` | — | none | clean |
| `transpara-ai/eventgraph` | — | none | clean |
| `transpara-ai/agent` | — | none | clean |

## Stale Branches — `transpara-ai/site` only

`git branch -r --merged origin/main` returned empty, but the following remote branches correspond to already-merged PRs and are candidates for deletion via `gh api -X DELETE repos/transpara-ai/site/git/refs/heads/<name>`:

- `origin/chore/relocate-phase-3-docs` (PR #26)
- `origin/docs/artifact-09-path-correction` (PR #24)
- `origin/docs/phase-1-findings-touchup`, `origin/docs/phase-2-status-refresh` (merged into PR #21)
- `origin/docs/phase-4-findings` (PR #29)
- `origin/feat/phase-1-token-refactor` (PR #18)
- `origin/feat/phase-2-tailwind-build-step` (PR #19)
- `origin/feat/phase-3-profile-context` (PR #25)
- `origin/feat/profile-brand-divergence` (PR #28)
- `origin/feat/profile-orphan-pages` (PR #27)
- `origin/fix/auth-persona-name-migration` (PR #23)
- `origin/fix/graph-test-category-e-g` (PR #22)
- `origin/fix/test-suite-hygiene`
- `origin/feat/site-bridge-state-transitions-and-mirror` (superseded)
- `origin/revert/16-hive-bridge`

Keep `origin/feat/profile-nav` (legitimately unmerged, carries PR #30).

Local-only branches on the Michael-primary checkout that are safe to `git branch -d`: `docs/artifact-09-path-correction`, `docs/phase-1-findings-touchup`, `docs/phase-2-status-refresh`, `feat/phase-1-token-refactor`, `feat/phase-2-tailwind-build-step`, `feat/site-bridge-state-transitions-and-mirror`, `fix/test-suite-hygiene`, `pr25`, `revert/16-hive-bridge`.

## Doc Inventory — `docs/site-profile-redesign/`

| Expected | Status |
|---|---|
| 01 — site map discovery | present |
| 02 — display profile system | present |
| 03 — Transpara profile design | present |
| 04 — Transpara profile wireframes | present |
| 05 — Transpara home prototype (HTML) | present |
| 06a recon prompt / 06b recon findings | present |
| 07 prompt-1 Phase 1 | present |
| 08 prompt-2 Phase 2 | present |
| 09 prompt-3 Phase 3 | present |
| Phase 1 findings + token map | present |
| Phase 2 findings + build plan | present |
| Phase 3 findings + plan | present |
| Phase 4 findings | present |
| Phase 5 findings | **absent on main** — lives on `feat/profile-nav`, ships with PR #30 |
| 10 prompt-Phase-4 | **absent** — Phase 4 prompt was supplied inline; no artifact archived |
| 11 prompt-Phase-5 | **absent** — Phase 5 prompts (nav + copy + findings) were supplied inline; no artifact archived |
| `session-2-artifact-09-path-correction-prompt.md` | on disk but **untracked** — has been untracked since before Phase 3 |

## Stale Files at Repo Root

Root `*.md`: only `CLAUDE.md`. Clean. PR #26 relocated the Phase-3 files out of the root; Phases 4/5 findings were authored directly into `docs/site-profile-redesign/` so no cleanup needed.

## Build Status

| Check | Result |
|---|---|
| `templ generate` | pass, 221 updates, no errors |
| `go build ./...` | clean |
| `go test ./...` | all packages green (`auth`, `cmd/gen-persona-status`, `graph`, `handlers`, `profile`, `views`) |
| `go vet ./...` | 2 pre-existing warnings at `graph/store.go:3543-3544` (self-assignment of `dc.SpaceSlug` / `dc.SpaceName`); predate Phase 4; no new warnings |
| `TestBoundedDiff_LayoutAcrossProfiles` (isolated) | PASS — normalises `data-profile` attribute, `BrandName`, `LogoPath`, `AccentColor` |

## Hardcoded `lovyou.ai` in Templates

| File:Line | String | Classification |
|---|---|---|
| `graph/views.templ:864` | `<title>Get Started — lovyou.ai</title>` | DEAD CODE — `SpaceOnboarding`, no handler call site (§G exclusion) |
| `graph/views.templ:883` | `placeholder="e.g. lovyou.ai"` | DEAD CODE — same `SpaceOnboarding` form input |
| `graph/views.templ:1265` | `value={ "https://lovyou.ai/join/" + inviteToken }` | EXPECTED — real invite URL (noted Phase 4 §A as "revisit when per-profile domains land") |
| `graph/views.templ:4012` | `value={ "https://lovyou.ai/join/" + inv.Token }` | EXPECTED — same invite pattern, different flow |
| `graph/views.templ:5150` | `https://lovyou.ai/app/your-space/feed` | EXPECTED — documentation example endpoint in `APIKeysView` curl snippet |

No `SHOULD BE PROFILE-DRIVEN` items on `main`. The only hardcoded branding that actually reaches render output is inside `graph.simpleFooter` (nav links + tagline), flagged separately in the PR #30 code review.

## Profile Registry (on `main`, pre-Phase-5)

| Profile | BrandName | Logo exists on disk | Accent | HeaderNav | FooterNav | Copy |
|---|---|---|---|---|---|---|
| `lovyou-ai` | `lovyou.ai` | `static/logo-lovyou.svg` ✓ | `#e8a0b8` | field not defined on `main` | field not defined on `main` | field not defined on `main` |
| `transpara` | `Transpara` | `static/logo-transpara.svg` ✓ | `#0ea5e9` | field not defined on `main` | field not defined on `main` | field not defined on `main` |

Both logo files exist at the paths the registry references. No empty or missing fields within Phase 4 scope. `HeaderNav` / `FooterNav` / `Copy` land with PR #30.

## Cross-Repo Findings

| Repo | Relevant hits | Assessment |
|---|---|---|
| `hive` | Internal loop / council docs reference `lovyou.ai` | clean — no user-facing templates |
| `work` | Design docs mention the civilization; `dashboard/dashboard.html:802` hardcodes `<span class="topbar-brand">Transpara-AI</span>` | clean — dashboard already Transpara-branded (internal mission-control board, not the public site) |
| `eventgraph` | Blog posts, CODE_OF_CONDUCT, SECURITY reference `lovyou.ai` | clean — public docs reflect the org identity by design; not profile-configurable |
| `agent` | Only agent persona design prompts | clean |
| `summary` | `dashboard.html:802` shows `Transpara-AI` topbar; `README.md:1` is `# lovyou.ai — Summary`; design docs reference the civilization | clean — static architecture poster, explicitly separate from the profile system per `work/docs/designs/telemetry-mission-control-design-v0.4.1.md:26` ("not the branded LovYou deployment on Fly.io, and not the static GitHub Pages architecture poster") |

No cross-repo surfaces need the profile system plumbed through. Each external repo has a clear identity and is a separate deployment.

## Recommended Actions

Prioritised:

1. **Merge PR #30 (Phase 5 nav + copy + findings).** Gate: address the `simpleFooter` omission flagged in the code-review comment, either by threading `GetFooterNav()` + `GetCopy("tagline", ...)` through `graph.simpleFooter`, or by explicitly scoping it out in the §G comment block / findings doc as intentional. As-is, orphan-page footers under `?profile=transpara` will render lovyou.ai-flavored nav, contradicting the PR's "all three shells" scope claim.
2. **Clean up stale remote branches on `transpara-ai/site`.** Fourteen remote branches correspond to already-merged PRs and can be deleted. `feat/profile-nav` stays; everything else in the Stale Branches list can go.
3. **Local branch cleanup on the site checkout.** `git branch -d` the nine merged/abandoned local branches. Keep `main` and `feat/profile-nav`.
4. **Decide on `SpaceOnboarding`.** Still dead code — hardcoded title, hardcoded placeholder, `@simpleHeader(user, nil)` / `@simpleFooter(nil)`, no handler calls anywhere. Either revive the onboarding flow or delete the template. Carried forward since Phase 4.
5. **Archive Phase 4 and Phase 5 prompt artifacts.** Prompts for Phases 4 and 5 were supplied inline in-session; prior phases all have a `0N-prompt-<name>.md` archive file. Consider back-filling `10-prompt-phase-4-profile-differentiation.md` and `11-prompt-phase-5-nav-and-copy.md` (or equivalent) for session-continuity.
6. **Fix `go vet` self-assignment warnings** at `graph/store.go:3543-3544`. Two lines, pre-existing (predate Phase 4). Trivial cleanup in a dedicated commit.
7. *(Optional, Phase 6+ deferred)* `SubdomainResolver` / `HostHeaderResolver` in `profile/resolver.go` — one Resolver implementation + one-line `Chain` addition in `cmd/site/main.go`. Unblocks `transpara.lovyou.ai` or `transpara.com` deployments. Not urgent until a separate-host deployment actually exists.
8. *(Optional)* `Profile.Name` is increasingly redundant — no template reads it post-Phase-4; all callers are tests + registry literals. Small noise-reduction commit.

---

## Methodology

Single-session audit driven by an inline prompt covering four parts:

- **Part 1** — `gh pr list` + `git branch -r` / `git branch` across all six `lovyou-ai-*` repos.
- **Part 2** — deep audit on `site`: branch hygiene on `main`, doc inventory, repo-root cleanliness, profile-package integrity, full `templ generate` + `go build` + `go test` + `go vet` toolchain, hardcoded-`lovyou.ai` grep across all `*.templ`, bounded-diff test in isolation, registry completeness check.
- **Part 3** — cross-repo grep for `lovyou.ai` / `brand` / `profile` in `*.go`, `*.md`, `*.html`, `*.tsx`, `*.ts` on the other five repos, filtering for user-facing surfaces.
- **Part 4** — structured report.

All checks ran from `main` at `22e612e`. Phase 5 scope (HeaderNav / FooterNav / Copy) is explicitly out of scope for this audit since it's unmerged on `feat/profile-nav` — those findings are documented in the forthcoming `phase-5-nav-and-copy-findings-v0.1.0.md` that lands with PR #30.

No fixes applied during the audit; this document is the deliverable.
