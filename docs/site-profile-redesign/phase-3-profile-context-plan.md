# Phase 3 — Profile Context Plan

**Version:** 0.1.0 · **Date:** 2026-04-21
**Author:** Claude (via Claude Code)
**Owner:** Michael Saucier
**Status:** Pass 1 complete — ready for Pass 2 execution
**Companion:** `09-prompt-3-phase-3-profile-context.md` (v0.1.1)

---

## 1. Router inventory

**Entry point:** `cmd/site/main.go` — `func main()` constructs `mux := http.NewServeMux()`.

**Existing global middleware chain (outer → inner):**

1. `canonicalHost(next http.Handler) http.Handler` — `cmd/site/main.go` (~line 955). Validates/redirects host.
2. `noCache(next http.Handler) http.Handler` — `cmd/site/main.go` (~line 981). Sets `Cache-Control` headers.
3. *(per-route auth via `authService.RequireAuth(...)`, `readWrap(...)`, `writeWrap(...)` — not in the global chain; applied at `mux.Handle(...)` registration sites in the 290–759 range.)*

The global wrap at the bottom of `main()` composes as `canonicalHost(noCache(mux))` before `http.ListenAndServe`.

**Profile middleware registration point:** wrap the mux as the innermost global middleware so that it runs *after* host-canonicalization and cache-header work but *before* any route handler. Insertion pattern:

```go
handler := canonicalHost(noCache(profile.Middleware(mux)))
http.ListenAndServe(addr, handler)
```

This ordering:
- Runs `canonicalHost` first (cheapest rejection path — drop bad-host requests before allocating profile state).
- Runs `noCache` second (header-only, no allocations).
- Runs profile resolution third, attaching `*Profile` to `r.Context()` exactly once per request.
- Every per-route auth wrapper runs inside the resolved-profile context — so if auth handlers ever need `profile.FromContext(ctx)`, it's already populated.

Rationale for not making profile the outermost: short-circuits before profile resolution (bad-host 4xx, cached 304) should not pay the profile-resolution cost.

---

## 2. Layout inventory

Three chrome surfaces, one layout component per surface.

### Public surface

| Component | File:Line | Current signature | Proposed signature |
|---|---|---|---|
| `Layout` | `views/layout.templ:10` | `Layout(title, description string, user ...SiteUser)` | `Layout(title, description string, p *profile.Profile, user ...SiteUser)` |

`Layout` is `@Layout(...)`-embedded by **23 public page components** (Home, BlogIndex, BlogPost, VisionPage, VisionLayerPage, VisionGoalPage, ReferenceIndex, BaseGrammarPage, CognitiveGrammarPage, CodeGraphPage, HigherOrderOpsPage, LayerPage, GrammarIndex, GrammarPage, AgentPrimitivesPage, PrimitivePage, SearchPage, DiscoverPage, AgentsPage, AgentProfilePage, ProfilePage, GlobalActivityPage, MarketPage, KnowledgePage). Every embed gains the new `p` argument.

### `/app` surface

| Component | File:Line | Current signature | Proposed signature |
|---|---|---|---|
| `appLayout` | `graph/views.templ:26` | `appLayout(space Space, spaces []Space, activeLens string, user ViewUser)` | `appLayout(space Space, spaces []Space, activeLens string, p *profile.Profile, user ViewUser)` |

`appLayout` is `@appLayout(...)`-embedded by ~25 lens/view templates (Board, Feed, Threads, Conversations, People, Agents, Activity, Knowledge, Changelog, Projects, Goals, GoalDetail, Roles, Teams, Policies, Documents, DocumentEdit, Questions, QuestionDetail, CouncilList, CouncilDetail, NotificationsView, SettingsView, SpaceDefaultView, Dashboard, Welcome, plus InviteCodeRow — count ≈ 26 call sites).

### `/hive` surface

| Component | File:Line | Current signature | Proposed signature |
|---|---|---|---|
| `HivePage` | `graph/hive.templ:7` | `HivePage(ls LoopState, entries []DiagEntry, commits []RecentCommit, user ViewUser)` | `HivePage(ls LoopState, entries []DiagEntry, commits []RecentCommit, p *profile.Profile, user ViewUser)` |

**Not layouts (exempt from Phase 3 signature change):**
- `simpleHeader(user ViewUser)` — embedded component.
- `commandPalette()`, partial HTMX snippets — leaf components; no chrome responsibility.
- HTMX partial handlers that write a fragment directly (no `@Layout` wrap) — profile does not need to reach them for Phase 3.

**Arg placement rationale:** Placing `*profile.Profile` immediately before `user` (or `user ...SiteUser`) minimizes churn at each call site: the call site already has easy access to `r.Context()`, so `profile.FromContext(ctx)` inserts cleanly before the user-resolution code, and `@Layout(title, desc, p, user)` reads naturally.

---

## 3. Handler inventory

Total handler call sites that invoke a layout: **≈49** (23 public + 25 `/app` + 1 `/hive`). Each gains one line `p := profile.FromContext(r.Context())` (or uses it directly as `profile.FromContext(r.Context())`) and passes `p` to the layout render.

### Public handlers (`cmd/site/main.go`)

23 handlers calling `Layout` or a wrapper that calls it, across routes:
- `/` (handleHome)
- `/blog`, `/blog/{slug}` (handleBlogIndex, handleBlogPost)
- `/vision`, `/vision/layer/{num}`, `/vision/goal/{id}`
- `/reference`, `/reference/grammar`, `/reference/cognitive-grammar`, `/reference/higher-order-ops`, `/reference/code-graph`, `/reference/layers/{num}`, `/reference/grammars`, `/reference/grammars/{slug}`, `/reference/agents`, `/reference/primitives/{slug}`
- `/search`, `/discover`
- `/agents`, `/agents/{name}`
- `/user/{name}`
- `/activity`, `/market`, `/knowledge`

**Note — variable-name collision:** `/user/{name}` currently binds a local `profile` struct (user profile view model). That local must rename to `userProfile` (or the like) before the `*profile.Profile` argument arrives, to avoid shadowing the imported package.

### `/app` handlers (`graph/handlers.go`)

≈25 handlers registered via `graph.Register(mux)` — per the audit (`graph/handlers.go:595`, 611, 621, 697, 1016, 1168, 1201, 1265, 1319, 1356, 1443, 1507, 1544, 1577, 1638, 1716, 1749, 1793, 1826, 1860, 1914, 1948, 1994, 2021, 2067 …). Each calls a view template that embeds `appLayout`; the handler extracts `p` and forwards it.

### `/hive` handler (`handlers/hive.go`)

1 handler — `hive.go:244` — renders `HivePage`.

### HTMX partial handlers (exempt)

Handlers that render *only* a fragment (no layout embed) — e.g., `/api/palette`, `/api/members`, and other `api/*` endpoints used for inline HTMX swaps. These don't need profile for Phase 3. If a future phase adds profile-branched partials, they can lift profile out of context at that point — the context plumbing is the same.

---

## 4. Env-var convention

**Existing env-var usage sites** (all in `cmd/site/main.go` unless noted):

| Var | Pattern |
|---|---|
| `PORT` | flag default + env override (line ~30) |
| `DATABASE_URL` | `if dsn := os.Getenv("DATABASE_URL"); dsn != ""` (line ~293) |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | non-empty check gates OAuth (~305–306) |
| `AUTH_REDIRECT_URL` | non-empty override (~309) |
| `HIVE_WEBHOOK_URL` | non-empty override (~373) |
| `CLAUDE_CODE_OAUTH_TOKEN` | non-empty check (~379) |
| `HIVE_REPO_PATH` | `handlers/hive.go:51` — env override with literal fallback |

**Observation:** all existing env vars are *configuration values* (DSNs, secrets, URLs, paths), not *feature flags*. The repo has no precedent for either `X_ENABLED` or `X_DISABLED` flags. The convention is "presence-means-configured."

**Decision for Phase 3:** keep `PROFILE_SYSTEM_DISABLED` as specified in the prompt. Semantics: if `os.Getenv("PROFILE_SYSTEM_DISABLED") != ""`, skip resolver chain and attach the default profile directly. Any non-empty value counts as disable — no "true/false" parsing, matching how the repo treats env presence elsewhere.

**Flagged in §F** — there is no prior convention to match, so the prompt's choice stands; documenting so future readers aren't surprised.

---

## 5. Templ toolchain

- **go.mod pin:** `github.com/a-h/templ v0.3.1001`.
- **Generated files committed:** yes — spot-check against Phase 2's merge commit (`graph/views_templ.go` committed alongside `graph/views.templ`).
- **Version drift:** `templ version` CLI may not be installed globally in the session environment; will `go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate` (or `go tool` equivalent) to stay pinned. If the local `templ` binary version matches `v0.3.1001`, use it directly.

---

## 6. Profile struct shape

Minimal per constraint #2:

```go
type Profile struct {
    Slug string
    Name string
}
```

**Registry:** `map[string]*Profile` keyed by `Slug`, with a `DefaultSlug` constant.

```go
const DefaultSlug = "lovyou-ai"

var registry = map[string]*Profile{
    "lovyou-ai": {Slug: "lovyou-ai", Name: "lovyou.ai"},
    "transpara": {Slug: "transpara", Name: "Transpara"},
}

func Default() *Profile        { return registry[DefaultSlug] }
func Lookup(slug string) *Profile // returns registry[slug] (nil if unknown)
func All() []*Profile           // iteration helper for tests/future admin
```

**Both registry entries carry the lovyou-ai display `Name` value** — per Artifact 09 "one tactical note on the two-profile registration": if the Name differs, any layout that renders Name (page title, footer, alt text) will produce divergent HTML and break the byte-identical-HTML verification. Phase 3 explicitly holds Name identical. Phase 4 is where Name differentiates once layouts begin branching on profile.

Actually — Phase 3 layouts receive `p` but *must not read `p.Name` anywhere*. If any existing template reads from a field added to Profile, byte-identical-HTML breaks. The safest stance: the Phase 3 struct ships with Slug + Name, layouts ignore both fields entirely, and Phase 4 introduces the first read. To belt-and-brace this: both registered profiles hold identical Name values until Phase 4 intentionally forks.

**Justification against more fields:** Artifact 02 §4 shows the eventual target has identity/theme/layout/nav/overrides. Adding those now would (a) force speculative design choices before Phase 4 surfaces real needs and (b) create layout call sites that would break when struct shape settles. Ship minimal; expand on concrete demand.

---

## 7. Resolver interface

```go
type Resolver interface {
    Resolve(r *http.Request) *Profile
}
```

**Implementations shipped in Phase 3:**

1. `QueryParamResolver` — reads `?profile=<slug>` from URL query. Returns `Lookup(slug)` or nil (nil on empty or unknown).
2. `DefaultResolver` — returns `Default()`; terminal in the chain.

**Chain:**

```go
type Chain []Resolver

func (c Chain) Resolve(r *http.Request) *Profile {
    for _, res := range c {
        if p := res.Resolve(r); p != nil {
            return p
        }
    }
    return Default()
}
```

A `Chain` is just a slice — composition is a literal:

```go
chain := profile.Chain{
    profile.QueryParamResolver{},
    profile.DefaultResolver{},
}
```

**One-line extensibility check:** adding a `SubdomainResolver` in Phase 4+ is:

```go
chain := profile.Chain{
    profile.SubdomainResolver{},      // ← added
    profile.QueryParamResolver{},
    profile.DefaultResolver{},
}
```

One line in chain construction, zero changes elsewhere. Passes constraint #5.

*Design note:* using `Chain []Resolver` (a slice type that itself implements `Resolver`) is slightly neater than a wrapper struct with a constructor — it eliminates `NewChain(...)` ceremony and makes test-time composition trivial. Semantically identical.

---

## 8. Commit sequence

**Rework from the prompt's proposed 8-commit sequence:** the prompt's commits 6 ("all layouts") and 7 ("all handlers") split by *layer*. That split breaks the build between commits 6 and 7, because changing a layout signature without updating its callers won't compile. Phase 3 prompt permits reordering for bisect-ability.

**Revised sequence (9 commits, within the 7–9 target):**

| # | Type | Subject | Touches |
|---|---|---|---|
| 1 | feat | `feat(profile): add Profile struct and registry` | `profile/profile.go` |
| 2 | feat | `feat(profile): add Resolver interface and implementations` | `profile/resolver.go` |
| 3 | feat | `feat(profile): add context plumbing` | `profile/context.go` |
| 4 | feat | `feat(profile): add resolution middleware with feature flag` | `profile/middleware.go` |
| 5 | test | `test(profile): unit tests for resolver chain and middleware` | `profile/profile_test.go` |
| 6 | refactor | `refactor(public): thread Profile through Layout + public handlers` | `views/layout.templ`, `views/*_templ.go` (regen), `cmd/site/main.go` public handlers, `views/*.templ` that embed Layout |
| 7 | refactor | `refactor(app): thread Profile through appLayout + /app handlers` | `graph/views.templ`, `graph/views_templ.go` (regen), `graph/handlers.go` lens handlers |
| 8 | refactor | `refactor(hive): thread Profile through HivePage handler` | `graph/hive.templ`, `graph/hive_templ.go` (regen), `handlers/hive.go` |
| 9 | feat | `feat(router): register profile middleware in HTTP stack` | `cmd/site/main.go` (ListenAndServe wiring) |

**Per-surface refactor split (6/7/8)** keeps each commit self-contained: one layout signature change plus every handler and template that calls it. Every commit compiles. Bisect points at a surface, not a layer.

**After every commit, the required verification (from Phase 3 prompt):**
- `templ generate` (commits 6+): no unexpected diff
- `go build ./cmd/site/`: must succeed
- `go test ./...`: must pass

**Commit 9 is last** so the middleware only activates after all surfaces can accept a non-nil Profile — avoids any window where the middleware is live but layouts still have old signatures.

---

## §F — Judgment calls (to document in findings)

1. **Env-var naming (`PROFILE_SYSTEM_DISABLED`) vs. repo convention.** Repo has no existing feature flags; all env vars are config values. Keeping the prompt's name; documenting so future feature flags align.
2. **Commit-sequence restructure (by surface instead of by layer).** 9 commits instead of 8. Prompt explicitly permits reordering for bisect-ability; this is that exit.
3. **`Chain []Resolver` vs. `NewChain(...)`.** Picked the slice-as-interface pattern for simpler composition; one-line SubdomainResolver swap still holds.
4. **Profile.Name identical across both entries.** Required to pass byte-identical-HTML if any template touches `p.Name`; belt-and-brace measure — Phase 3 layouts are instructed not to read the field at all.

## §G — Open questions (none blocking)

No design calls outstanding. Proceeding to Pass 2.
