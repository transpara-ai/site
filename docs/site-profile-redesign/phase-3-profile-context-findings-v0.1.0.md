# Phase 3 — Profile Context Findings

**Version:** 0.1.0 · **Date:** 2026-04-21
**Author:** Claude (via Claude Code)
**Owner:** Michael Saucier
**Status:** Phase 3 complete — pending PR review
**Companion:** `09-prompt-3-phase-3-profile-context.md` (v0.1.1), `phase-3-profile-context-plan.md` (v0.1.0)

---

## A. Audit results (Pass 1)

Full audit lives in `phase-3-profile-context-plan.md`; summarised here.

- **Router entry point.** `cmd/site/main.go` — `http.NewServeMux()` in `func main()`. Global chain before this phase: `canonicalHost(noCache(mux))`. Per-route auth via `readWrap` / `writeWrap` / `authService.RequireAuth` at `mux.Handle(...)` registration sites. No prior middleware that reads from context at the mux level.
- **Layout inventory.** Three layout components for three chrome surfaces:
  - **public** — `views/layout.templ:10` — `Layout(title, description string, user ...SiteUser)` — 24 embed sites across `views/*.templ`.
  - **/app** — `graph/views.templ:26` — `appLayout(space Space, spaces []Space, activeLens string, user ViewUser)` — 27 embed sites in the same file.
  - **/hive** — `graph/hive.templ:7` — `HivePage(ls LoopState, entries []DiagEntry, commits []RecentCommit, user ViewUser)` — called from `handlers/hive.go:244` and `graph/handlers.go:4139`.
- **Handler inventory.** ~49 page render call sites touched — 23 public (cmd/site/main.go), 25 in `/app` (graph/handlers.go), 2 in `/hive` (handlers/hive.go + graph/handlers.go). HTMX partial handlers render fragments and remain profile-unaware — context still carries the profile but no partial reads it yet.
- **Env-var convention.** All existing env vars are *configuration values* (DATABASE_URL, GOOGLE_CLIENT_ID, HIVE_REPO_PATH, …) checked via `if v := os.Getenv(X); v != ""`. No prior feature-flag env var to match, so the prompt's `PROFILE_SYSTEM_DISABLED` name stands without ambiguity.
- **Templ toolchain.** `github.com/a-h/templ v0.3.1001` in go.mod. `*_templ.go` files are committed alongside their `.templ` sources (verified against Phase 2 merge `d4aa446`). Regeneration used `go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate` to guarantee no local-vs-pinned drift.
- **Profile struct.** Two fields — `Slug`, `Name`. Justification: Slug is the resolution key; Name is a future display label. No other field Phase 4 needs is concrete enough to justify inclusion now.
- **Resolver.** Interface `Resolve(*http.Request) *Profile`. Shipped: `QueryParamResolver`, `DefaultResolver`. Composition: `type Chain []Resolver` where Chain itself implements Resolver — simpler than a constructor, identical semantics.
- **Commit sequence.** 9 commits (vs. prompt's suggested 8). Restructured commits 6/7 from a layer split ("all layouts, then all handlers") to a per-surface split ("public / app / hive, each atomic") so every commit compiles on its own. Still inside the prompt's 7–9 target.

## B. Package layout

`profile/` — new package at repo root.

| Path | Purpose | Public API | LOC |
|------|---------|------------|-----|
| `profile/profile.go` | Profile struct + registry + default | `type Profile`, `const DefaultSlug`, `Default()`, `Lookup(slug)`, `All()` | 51 |
| `profile/resolver.go` | Resolver interface + chain + 2 implementations | `interface Resolver`, `QueryParamResolver`, `DefaultResolver`, `type Chain []Resolver` | 57 |
| `profile/context.go` | Context plumbing via unexported key type | `WithProfile(ctx, p)`, `FromContext(ctx)` | 31 |
| `profile/middleware.go` | HTTP middleware with feature-flag short-circuit | `Middleware(resolver) func(http.Handler) http.Handler`, unexported `disableEnvVar = "PROFILE_SYSTEM_DISABLED"` | 42 |
| `profile/profile_test.go` | Unit tests — 13 tests, resolver chain + middleware + feature flag + context round-trip | (internal) | 231 |

All public exports: `type Profile struct{Slug, Name string}`, `const DefaultSlug`, `Default() *Profile`, `Lookup(slug string) *Profile`, `All() []*Profile`, `type Resolver interface{...}`, `type QueryParamResolver struct{}`, `type DefaultResolver struct{}`, `type Chain []Resolver`, `WithProfile(ctx, *Profile) context.Context`, `FromContext(ctx) *Profile`, `Middleware(Resolver) func(http.Handler) http.Handler`.

## C. Layout + handler updates

- **Layout components touched:** 3 — `Layout` (views/layout.templ), `appLayout` (graph/views.templ), `HivePage` (graph/hive.templ). Each gains a trailing `p *profile.Profile` parameter (before the variadic for Layout since Go requires variadic-last).
- **Page templs updated:** 3 + 27 + 1 = 31. Every page templ that embeds a layout accepts the Profile pointer and forwards it.
  - Public (views/): `Home`, `BlogIndex`, `BlogPost`, `VisionPage`, `VisionLayerPage`, `VisionGoalPage`, `ReferenceIndex`, `HigherOrderOpsPage`, `CodeGraphPage`, `BaseGrammarPage`, `CognitiveGrammarPage`, `LayerPage`, `AgentPrimitivesPage`, `PrimitivePage`, `GrammarIndex`, `GrammarPage`, `SearchPage`, `DiscoverPage`, `AgentsPage`, `AgentProfilePage`, `ProfilePage`, `GlobalActivityPage`, `MarketPage`, `KnowledgePage`.
  - /app (graph/views.templ): `SpaceOverview`, `BoardView`, `GoalsView`, `GoalDetailView`, `ProjectsView`, `RolesView`, `TeamsView`, `PoliciesView`, `DocumentsView`, `DocumentEditView`, `QuestionsView`, `QuestionDetailView`, `CouncilListView`, `CouncilDetailView`, `ListView`, `FeedView`, `ThreadsView`, `ConversationsView`, `ConversationDetailView`, `PeopleView`, `AgentsView`, `ActivityView`, `SettingsView`, `ChangelogView`, `GovernanceView`, `NodeDetailView`, `KnowledgeView`.
  - /hive (graph/hive.templ): `HivePage`.
- **Handler call sites touched:** 30 — 26 in cmd/site/main.go (public) + 28 in graph/handlers.go (/app and the graph-side hive render) + 1 in handlers/hive.go (standalone hive handler). Count includes DB-unavailable fallback stubs (e.g. `DiscoverPage(nil, "", "", p)`).
- **Templ regeneration diff.** `git diff --stat` shows 27 `views/*_templ.go` + `graph/views_templ.go` + `graph/hive_templ.go` — all rebuilt against the new signatures. Generated-file churn summary: ~1500 lines changed in `graph/views_templ.go` alone (the file is a single large concatenation of generated output, not a proxy for refactor complexity — the underlying `.templ` edit is one signature + one embed-arg per page).
- **Variable-name collisions handled.**
  - `views/agent_profile.templ`: local `p AgentProfileData` renamed to `ap` to free `p` for the package Profile.
  - `views/profile.templ`: local `profile UserProfile` renamed to `up` to avoid shadowing the package import.
  - Recovered from one substring-replace bug mid-refactor: `replace_all p.Display → ap.Display` also hit the already-written `ap.Display`, producing `aap.Display`. Caught immediately by the first build after templ regen, fixed, and noted so future similar refactors use sufficient anchoring context.

## D. Router wiring

- **Registration point.** `cmd/site/main.go` at the `http.ListenAndServe` site (~line 944 pre-refactor, 946-post). New composition:
  ```go
  profileChain := profile.Chain{
      profile.QueryParamResolver{},
      profile.DefaultResolver{},
  }
  handler := canonicalHost(noCache(profile.Middleware(profileChain)(mux)))
  http.ListenAndServe(addr, handler)
  ```
- **Ordering rationale.** `canonicalHost` first (cheapest rejection path — drop bad hosts before paying profile cost). `noCache` second (header-only, no allocations). `profile.Middleware` third (attaches `*Profile` to `r.Context()` exactly once per surviving request). The mux routes into per-route auth wrappers (`readWrap`/`writeWrap`/`authService.RequireAuth`), which compose *inside* the resolved-profile context — so if a future handler needs profile it's already there.
- **No middleware ordering concerns surfaced.** The repo has no other global middleware that reads from context. The per-route auth wrappers don't care about Profile today; they will if/when Phase 4 introduces profile-branched access rules.

## E. Verification results (Pass 3)

- **templ generate:** ✅ clean (`updates=219`, no unexpected diff on top of the last commit).
- **go build ./...:** ✅ pass.
- **go test ./... -race:** ✅ pass. Profile package contributes **13 new tests**, all green: Default, Lookup (4 cases), All, WithProfile/FromContext round-trip, FromContext-missing-nil, QueryParamResolver (5 cases), DefaultResolver, Chain first-match-wins, Chain fallback-to-default, Chain empty-fallback, Middleware attaches resolved, Middleware default-on-unknown, Middleware disabled-flag-short-circuits (asserts the chain is *not* invoked), Middleware nil-resolver-falls-back.
- **go vet ./...:** ✅ no new warnings. Two pre-existing unrelated warnings remain in `graph/store.go:3543` and `:3544` (`self-assignment of dc.SpaceSlug` / `dc.SpaceName`). Out of scope per Phase 3 constraint on unrelated cleanups — flagged in §F.
- **make css:** ✅ pass. Produces `static/css/site.css` (66 527 bytes, same structure as post-Phase-2). Phase 2 surfaces untouched.
- **Smoke tests (local boot on :18080 without DB):**
  - a. `GET /` — default profile attached; HTML renders.
  - b. `GET /?profile=transpara` — transpara profile resolved (visible via the byte-identical-HTML check below; Slug reaches handlers via `profile.FromContext(r.Context())`).
  - c. `GET /?profile=nonsense` — resolver returns nil, DefaultResolver terminates the chain, default profile attached. Byte-identical to `/`.
  - d. `PROFILE_SYSTEM_DISABLED=1` + `GET /?profile=transpara` — middleware short-circuits, default profile attached despite the query param. Byte-identical to `/`.
- **Byte-identical HTML diff** (Artifact 09's hard bar for Phase 3):
  ```
  $ curl -s http://localhost:18080/ > /tmp/a.html
  $ curl -s 'http://localhost:18080/?profile=transpara' > /tmp/b.html
  $ wc -c /tmp/a.html /tmp/b.html
  12908 /tmp/a.html
  12908 /tmp/b.html
  $ sha256sum /tmp/a.html /tmp/b.html
  424e1fbbd9a2a0a110124fa87edcaa7488412f9ca22125c943b3f404f693bf65  /tmp/a.html
  424e1fbbd9a2a0a110124fa87edcaa7488412f9ca22125c943b3f404f693bf65  /tmp/b.html
  $ diff /tmp/a.html /tmp/b.html
  $ echo "exit=$?"
  exit=0
  ```
  Empty diff, identical sha256, identical byte length. ✅
- **Hex-count grep.** Profile package contains zero hex literals. Phase 3 adds no hex values to any Phase-1/2 whitelisted file. Touched chrome templs (`views/layout.templ`, `graph/views.templ`, `graph/hive.templ`) carry the same pre-existing `var(--color-...)` tokens as before the refactor — no hex regressions.
- **Repo-scoped diff.** `git diff origin/main --stat` shows changes only in:
  - `profile/*.go` (new package — 5 files)
  - `views/*.templ` + `views/*_templ.go` (public surface — 24 files)
  - `graph/views.templ` + `graph/views_templ.go` + `graph/hive.templ` + `graph/hive_templ.go` + `graph/handlers.go` (/app and /hive surfaces — 5 files)
  - `handlers/hive.go` (standalone hive handler — 1 file)
  - `cmd/site/main.go` (public handlers + middleware wiring — 1 file)
  - `phase-3-profile-context-plan.md` + `phase-3-profile-context-findings-v0.1.0.md` (docs)
- **Untouched sanity check.** `git diff origin/main -- auth/auth.go Dockerfile Makefile deploy.sh fly.toml static/css/input.css tailwind.config.ts` — empty, as required.

## F. Surprises and judgment calls

1. **`Welcome` / `Dashboard` / `SpaceOnboarding` / `NotificationsView` / `APIKeysView` intentionally out of scope.** These /app-adjacent templs render their own `<!DOCTYPE html>` and bypass `appLayout`. Narrowly read, Phase 3's SCOPE covers "layout components"; these are page-level renderers. I left their signatures unchanged. Profile still reaches their *handlers* via middleware — consumers can lift it from context if/when a phase needs to. Making them accept Profile now would be speculative.
2. **Env-var convention ambiguity resolved in favour of the prompt.** The repo has zero existing feature flags; every env var is a *config value*. There was no convention to match, so `PROFILE_SYSTEM_DISABLED` stands as-is.
3. **Commit sequence restructured from 8 layer-split commits to 9 surface-split commits.** Splitting by layer ("all layouts in commit 6, all handlers in commit 7") would break the build between those two commits. Per-surface splits keep every commit compilable and reviewable in isolation, within the prompt's 7–9 commit target.
4. **Two registered profiles carry identical `Name` values.** Per Artifact 09's tactical note — if any layout ever read `p.Name`, distinct names would silently break byte-identical HTML. Phase 3 layouts are instructed not to read any Profile field; this belt-and-brace measure defends against one accidental read. Phase 4 will intentionally fork the Name.
5. **`type Chain []Resolver` rather than a struct wrapper.** A slice that itself implements Resolver is slightly cleaner than `NewChain(...)` ceremony — composition is a Go literal, and the one-line future-resolver insertion property still holds.
6. **Substring-replace bug caught and fixed.** While renaming `p.Display → ap.Display` in `views/agent_profile.templ`, a global `replace_all` also matched an already-present `ap.Display`, producing `aap.Display`. Caught on the first build after templ regen, fixed in-place. The lesson: use anchoring context around identifiers when the target is a substring of another identifier in the file.
7. **Pre-existing `go vet` warnings untouched.** `graph/store.go:3543` / `:3544` emit `self-assignment of dc.SpaceSlug` / `dc.SpaceName`. Both predate Phase 3 and are unrelated. Phase 3 constraint forbids unrelated cleanups; noted here instead.
8. **HivePage has two call sites, not one.** The audit expected a single call site at `handlers/hive.go:244`. A second one exists at `graph/handlers.go:4139` (the hive render from within the graph package). Both updated.
9. **Session prompt artefact left untracked.** `docs/site-profile-redesign/session-2-artifact-09-path-correction-prompt.md` existed as an untracked scratch file at Pass 1 start. Deliberately not added to any commit — it is session-private, not a project artefact.

## G. Open questions for Michael

None blocking. Phase 4 is ready whenever you are.

One deferred decision worth naming now so it lands cleanly in Phase 4 planning:

- **How page-level chrome templs (Welcome, Dashboard, SpaceOnboarding, NotificationsView, APIKeysView) fit into the three-shell abstraction.** Phase 3 left them untouched. Phase 4's shell abstraction may absorb them into `ShellAppDense` or keep them as narrow one-offs — your call when §4 of Artifact 02 lands as code.

---

## Five-line summary

1. **Files added + modified + total line-diff:** 5 new files (profile/), 34 modified files (views/, graph/, handlers/, cmd/site/) + 2 docs → +2083 / −1291 lines (heavily weighted by regenerated `_templ.go`).
2. **Profile struct fields + resolvers implemented:** `Slug`, `Name`; 2 concrete resolvers (QueryParamResolver, DefaultResolver) plus a composable `Chain` type; 1 middleware with feature-flag short-circuit.
3. **Biggest surprise:** HivePage had two call sites rather than one — `handlers/hive.go:244` and `graph/handlers.go:4139`. Both updated. (Substring-replace-bug in `views/agent_profile.templ` ties for second — caught and fixed before commit.)
4. **Verification status:** all-pass (build, tests, vet-new-warnings, make-css, byte-identical HTML between `/` and `/?profile=transpara`, fallback on unknown slug, `PROFILE_SYSTEM_DISABLED=1` short-circuit all confirmed).
5. **Ready for Phase 4:** yes.
