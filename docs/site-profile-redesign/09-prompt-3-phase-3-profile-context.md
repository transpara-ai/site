# Prompt 3 — Phase 3: Profile Context

**Version:** 0.1.0 · **Date:** 2026-04-21
**Author:** Claude Opus 4.7
**Owner:** Michael Saucier
**Status:** Ready to execute — copy into Claude Code session after Phase 2 PR merges
**Versioning:** Versioned as part of the site-profile-redesign set. Major for structural changes to the prompt scope; minor for additional work items; patch for corrections and clarifications.
**Companion:** `01-site-map-discovery.md` (v0.2.0), `02-display-profile-system.md` (v0.3.0), `03-transpara-profile-design.md` (v0.3.0), `04-transpara-profile-wireframes.md` (v0.3.0), `05-transpara-home-prototype.html` (v0.1.2), `06a-site-profile-redesign-recon-prompt.md` (v0.1.0), `06b-site-profile-redesign-recon-findings-v0.1.0.md`, `07-prompt-1-phase-1-token-refactor.md` (v0.1.0), `08-prompt-2-phase-2-tailwind-build-step.md` (v0.1.0), `phase-1-token-refactor-findings-v0.1.0.md`, `phase-2-tailwind-build-step-findings-v0.1.0.md`

---

### Revision History

| Version | Date | Description |
|---------|------|-------------|
| 0.1.0 | 2026-04-21 | Initial Phase 3 prompt. Scope: introduce a `Profile` abstraction with request-time resolution via pluggable resolver (CEO decision #4), plumb `Profile` through context to templ layouts as struct param, register two profiles (`lovyou-ai`, `transpara`) with identical rendering, gate behind `PROFILE_SYSTEM_DISABLED` feature flag. No visual change. **Commit trailer convention updated:** Paperclip is removed from this project; Phase 3 uses `Co-Authored-By: transpara-ai <transpara-ai@transpara.com>`. |

---

## Where this sits in the plan

```
Prompt 0 (recon) ─────────────► findings v0.1.0
Prompt 1 (Phase 1) ───────────► token refactor — MERGED (PR #18)
Prompt 2 (Phase 2) ───────────► tailwind build step — MERGED (PR #19)
                                    │
                                    ▼
Prompt 3 (THIS) ──────────────► Phase 3: Profile context  ◄── you are here
Prompt 4 ──────────────────────► Phase 4: three-shell abstraction
Prompt 5 ──────────────────────► Phase 5: route visibility + logo/wordmark
Prompt 6 ──────────────────────► Phase 6: Transpara visual fork
...through Phase 8 (test matrix)
```

Phase 3 is pure plumbing. The `Profile` object gets defined, resolved per request, attached to context, pulled out in handlers, and passed to layout components as a struct parameter. Both profiles exist in the registry. Neither profile branches visually yet — Phase 6 is where the tokens actually fork. If anything looks different between `/` and `/?profile=transpara` after Phase 3, something is wrong.

Phase 3 also locks in CEO decision #4: the resolver is pluggable. Phase 3 ships with `QueryParamResolver` + `DefaultResolver`. Swapping in a `SubdomainResolver`, `CookieResolver`, or `HostHeaderResolver` later must be a one-line change in chain construction — not a refactor.

---

## Precondition

- PR #18 (Phase 1 token refactor) MUST be merged to `origin/main`.
- PR #19 (Phase 2 Tailwind build step) MUST be merged to `origin/main`.
- PR #20 (revert of accidentally-merged PR #16) MUST be in `origin/main`.
- CI must be green on `main` — site repo pinned to hive submodule at commit `486db00`, submodule-path fix confirmed at site commit `8d7d809`.
- Current branch at launch: `main` at HEAD, `git status` clean.
- `docs/site-profile-redesign/phase-1-token-refactor-findings-v0.1.0.md` and `docs/site-profile-redesign/phase-2-tailwind-build-step-findings-v0.1.0.md` should both be readable.
- `go test ./...` passes on `main`.
- `make css` (from Phase 2) produces valid `static/css/site.css` on `main`.

If any of these are not true, stop and notify Michael.

---

## How to launch

1. **Attach the design set + Phase 1 and Phase 2 findings** to your Claude Code session (Artifacts 01–05 + 07 + 08 + phase-1 findings + phase-2 findings).
2. **Enable skills:**
   - `frontend-design` — for context on layout and templ patterns.
   - `hive-lifecycle` — not needed, keep available.
3. **Launch from `lovyou-ai-site` repo root.**
4. **Verify PR #19 is merged** (`git log --oneline origin/main | head -5` should show the Phase 2 merge commit and the PR #20 revert).
5. **Paste the PROMPT block below.**

---

## PROMPT — copy everything between `─── BEGIN ───` and `─── END ───` markers below

```
─── BEGIN PROMPT 3 — PHASE 3: PROFILE CONTEXT ───

ROLE
You are executing Phase 3 of the site profile redesign for the
lovyou-ai-site repo. Phase 1 (token refactor) and Phase 2 (Tailwind
build step + @theme consolidation + 4 state/priority tokens) are
complete and merged to main. You have the design set (Artifacts
01–05), Prompts 1 and 2 (Artifacts 07 and 08), and both Phase 1 and
Phase 2 findings docs attached.

GOAL
Introduce a Profile abstraction with request-time resolution. Define
a Profile struct in a new profile/ package, add a pluggable Resolver
interface with QueryParamResolver + DefaultResolver implementations,
write resolution middleware that attaches Profile to context.Context,
update templ layout components to accept *Profile as a struct param,
update handlers to extract Profile from context and pass it through.
Register two profiles (lovyou-ai, transpara) with identical rendering.
Gate the entire system behind a PROFILE_SYSTEM_DISABLED feature flag.
No visual change of any kind.

WHY THIS PHASE EXISTS
Every subsequent phase (shell abstraction, route visibility, logo
component, visual fork) depends on the request knowing which profile
it's rendering for. Phase 3 establishes that plumbing end-to-end so
Phases 4–6 can branch on profile.Slug without rewiring anything.

CEO decision #4 (locked): the resolver is pluggable and starts with
query-param (?profile=transpara) for Phase 3. The interface must be
designed so a future SubdomainResolver, CookieResolver, or
HostHeaderResolver drops in with a one-line change to chain
construction. This is the entire value of this phase.

The feature flag exists as a cheap revert. If Phase 3 ships and
anything downstream misbehaves, PROFILE_SYSTEM_DISABLED=1 falls back
to always-lovyou-ai behavior without code changes or rollback.

SCOPE

IN SCOPE:
- New package at repo root: profile/
    profile/profile.go      — Profile struct + registry + two
                              profiles (lovyou-ai default, transpara)
    profile/resolver.go     — Resolver interface + QueryParamResolver
                              + DefaultResolver + chain composition
    profile/context.go      — unexported context key type,
                              WithProfile(ctx, *Profile),
                              FromContext(ctx) *Profile
    profile/middleware.go   — HTTP middleware; honors
                              PROFILE_SYSTEM_DISABLED env var
    profile/profile_test.go — unit tests covering the resolver
                              chain, middleware behavior, feature
                              flag, default fallback, unknown-slug
                              fallback
- Templ layout signature updates: every layout component that
  renders chrome for public, /app, or /hive surfaces accepts
  *profile.Profile as a struct parameter.
- Handler updates: every handler that renders a layout extracts
  Profile from request context and passes it through.
- Router wiring: profile middleware registered at the correct
  point in the HTTP stack, before any handler that needs it.
- Templ regeneration: if layout signatures change, regenerate .go
  files and commit them in the same commit as the signature change.

OUT OF SCOPE (do not touch):
- Shell abstraction (Phase 4). Layouts do not branch on
  profile.Slug yet. They receive Profile but ignore it.
- Route visibility, per-profile route overrides (Phase 5).
- Logo, wordmark, or any per-brand component (Phase 5).
- Any visual divergence between the two profiles (Phase 6).
  Token forks, chrome differences, copy differences — none of it.
- Subdomain, cookie, or host-header resolvers. Design the
  Resolver interface so these drop in later, but do not implement
  them now.
- /hive iframe wrapping behavior (CEO decision #1). Confirm
  Profile reaches the handler that emits the iframe, but do not
  modify the iframe itself.
- auth/auth.go — still out of scope as in Phase 1.
- lovyou-ai-work and lovyou-ai-summary repos — single-repo phase.
- Fly.io deployment. deploy.sh runs unchanged.
- Unrelated fixes, cleanups, or refactors. If you notice
  something sketchy, note it in findings §F and move on.

CONSTRAINTS (non-negotiable)
1. NO VISUAL CHANGE. Both profiles must render byte-identical HTML
   when the query param is absent. /?profile=transpara must render
   byte-identical HTML to /. Use curl + diff to verify in Pass 3.
2. THE PROFILE STRUCT IS MINIMAL. Slug and Name only, unless Pass 1
   audit surfaces a concrete Phase 4 need. Resist speculative fields.
3. UNEXPORTED CONTEXT KEY TYPE. Never use a string literal as a
   context key. Define an unexported custom type; expose WithProfile
   and FromContext helpers.
4. FEATURE FLAG SHORT-CIRCUITS IN THE MIDDLEWARE. If
   PROFILE_SYSTEM_DISABLED is set, attach the default profile and
   skip resolution. This keeps the revert surface small and obvious.
5. RESOLVER CHAIN IS THE POINT. Structure it so adding a
   SubdomainResolver is one line in chain construction. If it's
   not, the design is wrong.
6. NO MAIN-BRANCH COMMITS. Feature branch:
   feat/phase-3-profile-context
7. NO UPSTREAM PUSH. Origin only.
8. DON'T DEPLOY. deploy.sh must not run. Do not touch production.
9. CO-AUTHORED-BY TAG on every commit:
   Co-Authored-By: transpara-ai <transpara-ai@transpara.com>
   (Paperclip is removed from this project — do NOT use the
   Paperclip trailer from Phases 1 and 2.)
10. PR, DO NOT MERGE. Michael reviews and merges.

WORK PLAN

Phase 3 is executed in three passes:

PASS 1 — AUDIT + PLAN (non-destructive)

Do not edit any Go, templ, CSS, or config file in this pass. The
output of Pass 1 is a written plan and nothing else.

Produce phase-3-profile-context-plan.md at the repo root covering:

1. Router inventory:
   - HTTP router entry point (likely cmd/site/main.go).
   - Every middleware registration site.
   - Exactly where the profile middleware must register to run
     before any handler that reads Profile from context.

2. Layout inventory:
   - Every layout templ component for the three chrome surfaces:
     public, /app, /hive.
   - Current signatures of each.
   - Proposed new signatures with *profile.Profile added.

3. Handler inventory:
   - Every handler that invokes a layout component.
   - Grouped by surface (public / app / hive).
   - For each: which layout it calls, what it passes today, what
     it needs to pass after the refactor.

4. Env-var convention:
   - Where existing env-var parsing lives in the codebase.
   - Confirm PROFILE_SYSTEM_DISABLED fits the established
     convention. If the repo uses positive-enable flags
     (PROFILE_SYSTEM_ENABLED=1 default on), flip to match.

5. Templ toolchain:
   - templ version in use (`templ version`).
   - Whether generated .go files are committed (should be — spot
     check a recent commit that modified a .templ).
   - Any version drift between local toolchain and go.mod pin.

6. Profile struct shape:
   - Proposed fields (should be Slug, Name — justify anything
     beyond that).
   - Registry structure (map? slice of constants?).
   - Default profile selection.

7. Resolver interface:
   - Interface shape.
   - QueryParamResolver behavior: reads ?profile=<slug>, returns
     the matching Profile or nil if unknown.
   - DefaultResolver behavior: always returns lovyou-ai.
   - Chain composition: a slice of resolvers tried in order,
     first non-nil wins. Default is terminal.

8. Commit sequence:
   - Propose the exact commit list (target 7–9 commits).
   - Each commit compiles, each commit passes existing tests,
     each commit is reviewable in isolation.

If anything in the plan requires a design call Michael hasn't made,
stop and flag in findings §G before proceeding. Do not assume.

PASS 2 — EDIT

Suggested commit sequence (order matters for bisect-ability):

  1. `feat(profile): add Profile struct and registry`
     (profile/profile.go — struct definition, lovyou-ai and
      transpara profiles registered, default selector, no wiring)

  2. `feat(profile): add Resolver interface and implementations`
     (profile/resolver.go — Resolver interface, QueryParamResolver,
      DefaultResolver, chain composition helper)

  3. `feat(profile): add context plumbing`
     (profile/context.go — unexported key type, WithProfile,
      FromContext)

  4. `feat(profile): add resolution middleware with feature flag`
     (profile/middleware.go — HTTP middleware using the resolver
      chain, short-circuits on PROFILE_SYSTEM_DISABLED)

  5. `test(profile): unit tests for resolver chain and middleware`
     (profile/profile_test.go — default path, query-param override,
      unknown slug fallback, feature-flag disabled, context
      round-trip)

  6. `refactor(layouts): accept Profile as layout parameter`
     (update every layout component to take *profile.Profile;
      commit regenerated .go files in this same commit)

  7. `refactor(handlers): extract Profile from context and pass
      through to layouts`
     (update every handler call site identified in Pass 1)

  8. `feat(router): register profile middleware in HTTP stack`
     (wire middleware at the point identified in Pass 1)

After EACH commit:
- Run `templ generate` (commits 6+). No unexpected diff.
- Run `go build ./cmd/site/`. Must succeed.
- Run `go test ./...`. Must pass.

If any commit breaks a later one, reorder. Bisect-friendliness is
more important than the suggested order.

PASS 3 — VERIFY

1. `templ generate` produces no unexpected diff in committed output.
2. `go build ./cmd/site/` succeeds.
3. `go test ./... -race` — all tests pass, including new
   profile package tests.
4. `go vet ./...` clean.
5. `make css` still produces valid output (Phase 2 should be
   untouched; this is a sanity check).
6. `git diff main -- auth/auth.go` is empty.
7. `git diff main -- Dockerfile Makefile deploy.sh fly.toml
   static/css/input.css tailwind.config.ts` is empty. Phase 3
   does not touch Phase 1 or Phase 2 surfaces.
8. Local smoke test, documented step by step in findings §E:
   a. Boot the site locally. Request `/` with no query param.
      Verify lovyou-ai profile is attached to the request
      context (temporary debug log acceptable; remove before
      final commit).
   b. Request `/?profile=transpara`. Verify transpara profile
      attaches.
   c. Request `/?profile=nonsense`. Verify fallback to lovyou-ai
      (DefaultResolver terminates the chain).
   d. Set PROFILE_SYSTEM_DISABLED=1, restart. Verify lovyou-ai
      attaches regardless of query param.
9. Byte-identical HTML check (the bar for "no visual change"):
   `curl -s http://localhost:PORT/ > /tmp/a.html`
   `curl -s 'http://localhost:PORT/?profile=transpara' > /tmp/b.html`
   `diff /tmp/a.html /tmp/b.html`
   Output must be empty. Paste the empty diff into findings §E
   as proof.
10. Hex-count grep sanity: `grep -rInE '#[0-9a-fA-F]{3,8}'
    --include='*.templ' --include='*.css' --include='*.html'
    cmd/ views/ graph/ static/` — survivors should match the
    Phase 2 whitelist exactly. Phase 3 adds no hex values.
11. Repo-scoped diff review: `git diff main --stat` should show
    changes only in profile/, templ layout files, handler files,
    router wiring, and the findings doc. Anything else is out
    of scope for Phase 3 and must be justified or reverted.

GIT DISCIPLINE

Branch: `feat/phase-3-profile-context`

Remote: `origin` (transpara-ai fork, NEVER upstream)

Commit format: Conventional commits, lowercase subject, imperative:
  feat(profile): add Profile struct and registry

  Introduce profile/profile.go with the Profile struct (Slug, Name
  fields), a package-level registry of known profiles, and two
  entries: lovyou-ai (default) and transpara. No wiring to HTTP or
  handlers in this commit — struct and registry only.

  Co-Authored-By: transpara-ai <transpara-ai@transpara.com>

Before pushing: verify `git log --oneline origin/main..HEAD` lists
only commits with the transpara-ai co-author trailer. No Paperclip
lines (that convention is retired as of Phase 3).

PR: open against `origin/main` titled
  "feat(profile): Phase 3 — profile context + pluggable resolver"

PR body must include:
- Link to this prompt (Artifact 09)
- Link to findings doc
- Before/after summary:
  - Before: no Profile concept; layouts and handlers are
    profile-agnostic
  - After: Profile struct, resolver chain, context plumbing,
    middleware, two profiles registered, feature-flag gated —
    all rendering identically
- Proof of byte-identical HTML between / and /?profile=transpara
  (the empty diff output)
- Proof that PROFILE_SYSTEM_DISABLED=1 reverts to default behavior
- Any §F/§G items from findings that need Michael's attention

DELIVERABLE

Create `$REPO_ROOT/phase-3-profile-context-findings-v0.1.0.md`:

  # Phase 3 — Profile Context Findings

  **Version:** 0.1.0 · **Date:** <today>
  **Author:** Claude (via Claude Code)
  **Owner:** Michael Saucier
  **Status:** Phase 3 complete — pending PR review
  **Companion:** 09-prompt-3-phase-3-profile-context.md

  ## A. Audit results (Pass 1)
     - Router inventory (entry point, middleware sites)
     - Layout inventory (components per chrome surface)
     - Handler inventory (call sites per surface)
     - Env-var convention (existing pattern + chosen flag name)
     - Templ toolchain (version, regen status)
     - Profile struct proposal (fields + justification)
     - Resolver interface proposal (shape + chain construction)
     - Commit sequence proposal (accepted as-is / modified)

  ## B. Package layout
     For each new file under profile/:
     - Path
     - Public API (types + exported functions)
     - Line count
     - One-sentence purpose

  ## C. Layout + handler updates
     - Layout components touched (count, list)
     - Handler call sites touched (count, grouped by surface)
     - Templ regeneration diff summary (lines added/removed in
       generated .go)

  ## D. Router wiring
     - Middleware registration point (file + function + order
       relative to existing middleware)
     - Any middleware ordering concerns surfaced

  ## E. Verification results (Pass 3)
     - templ generate: pass/fail
     - go build: pass/fail
     - go test -race: pass/fail (count, none broken, new tests
       from this phase)
     - go vet: pass/fail
     - make css: pass/fail (sanity — should be untouched)
     - Smoke test results (4 scenarios from Pass 3 step 8)
     - Byte-identical HTML diff output (should be empty — paste it)
     - Hex-count grep: survivors (should equal Phase 2 whitelist)
     - Repo-scoped diff: only expected paths touched (yes/no + list)

  ## F. Surprises and judgment calls
     Anything that required a decision not explicit in the prompt.
     Be honest — this is how Phase 4 gets tuned.

  ## G. Open questions for Michael
     Things that could not be resolved without his call.
     One question per line.

  ---

  ## Five-line summary

  Line 1: Files added + files modified + total line-diff
  Line 2: Profile struct field list + number of resolvers implemented
  Line 3: Biggest surprise (or "none")
  Line 4: Verification status (all-pass / any failures)
  Line 5: Ready for Phase 4 (yes / blocked by X)

APPROACH HINTS

1. Read the design set before touching anything. Artifact 02 §6
   step 3 is the canonical spec for this phase — re-read it even
   if you've seen it before. Artifact 07 and 08 establish the
   three-pass discipline and commit-style patterns this phase
   inherits.
2. The Profile struct should be deliberately minimal. Slug and
   Name cover Phase 3. More fields land when Phase 4 needs them
   — not speculatively. It's cheap to add a field later and
   expensive to prune one that callers have started using.
3. The resolver chain is the whole point of this phase. Write it
   so swapping QueryParamResolver for a future SubdomainResolver
   is ONE LINE in chain construction. If your chain composition
   requires rewriting three files to swap a resolver, redesign.
4. templ does not plumb context.Context natively. Handlers do.
   Pattern: handler reads ctx → handler extracts *Profile via
   profile.FromContext(ctx) → handler passes *Profile as a
   struct param to the layout templ component. No globals,
   no thread-locals, no clever tricks.
5. Feature flag short-circuits in the middleware, not in the
   resolver. If PROFILE_SYSTEM_DISABLED is set, the middleware
   attaches the default profile and does not invoke the chain
   at all. Small revert surface, obvious behavior.
6. Context-key hygiene: unexported type, never a string literal.
   Go 101, but silent collision bugs eat hours. Do it right.
7. Middleware ordering matters silently. If the profile
   middleware registers after a handler-level middleware that
   reads Profile from context, FromContext returns nil and
   nothing obvious breaks until a handler actually uses a
   Profile field. Pass 1 audit must surface the correct
   registration point.
8. Byte-identical HTML is the verification bar. If / and
   /?profile=transpara produce any diff — even whitespace —
   something in the rendering path is branching on profile
   and shouldn't be yet. Fix it before declaring Phase 3 done.
9. Commit the plan in Pass 1 before editing. Push the plan
   commit to origin so Michael can review mid-run if he wants.
   Do not begin Pass 2 until the plan is written.

FINAL CHECK BEFORE YOU START

- [ ] Working directory is lovyou-ai-site repo root
- [ ] Current branch is main at the PR #19 merge commit (or
      later, if PR #20 revert is present)
- [ ] Git status is clean
- [ ] Artifacts 01–05 + 07 + 08 + Phase 1 findings + Phase 2
      findings are attached
- [ ] You will create feat/phase-3-profile-context
- [ ] You understand: NO visual change, NO shell abstraction,
      NO token fork, NO route overrides, NO logo component
- [ ] You will produce phase-3-profile-context-plan.md in Pass 1
- [ ] You will produce findings doc at the path specified
- [ ] You will use the transpara-ai co-author trailer, NOT
      the Paperclip trailer from Phases 1 and 2
- [ ] You will open a PR and STOP — not merge

Begin Phase 3.

─── END PROMPT 3 — PHASE 3: PROFILE CONTEXT ───
```

---

## What happens after this runs

1. Claude Code produces `phase-3-profile-context-plan.md` (Pass 1), commits and pushes it, then executes Pass 2 commits, then produces `phase-3-profile-context-findings-v0.1.0.md`.
2. Claude Code opens PR against `origin/main` and stops.
3. You review the PR — particular attention to:
   - The resolver chain. Does adding a hypothetical `SubdomainResolver` actually take one line? Sketch it mentally; if it doesn't, kick back.
   - Middleware registration point. Is it before every handler that reads `Profile` from context?
   - Byte-identical HTML proof. The empty diff output should be in the PR body. If it isn't, the verification wasn't done.
   - `PROFILE_SYSTEM_DISABLED=1` actually working end-to-end. This is the cheap revert; it must work.
4. Manual visual check on `/`, `/app`, `/hive`, `/blog/the-hive` with and without `?profile=transpara`. Nothing should change visually.
5. If §F surfaces anything interesting or §G has open questions, bring findings back here.
6. We draft **Prompt 4: Phase 4 — three-shell abstraction** for the next session. Phase 4 is where layouts actually start branching on `profile.Slug`, using the plumbing Phase 3 laid down.

---

## Why this phase is boring-by-design

Phase 3 introduces a new package, a new interface, a new middleware, changes every layout signature, and touches every handler that renders chrome. That sounds like a lot. It is a lot. But none of it is visible to a user. The entire point is to lay down the plumbing so Phase 4 can turn on the water.

Resist every urge to "just quickly" fork a token, branch a component, or pre-populate a Phase 4 struct field. Every one of those shortcuts:
- Hides real bugs in the context plumbing behind visual noise.
- Commits Phase 3 to design choices that belong to Phase 4.
- Makes the byte-identical-HTML verification impossible.

If Phase 3 feels underwhelming to review, it's probably correctly scoped. If it feels ambitious, something leaked in from a later phase.

---

## Risk notes (for your situational awareness, not for the prompt)

**Risk 1 — templ regeneration drift.** Signature changes on templ components require `templ generate`. The generated `.go` files must be committed in the same commit as the signature change. If Claude Code's local templ version doesn't match `go.mod`, regeneration produces diff noise and commits won't be clean. The plan must confirm templ version alignment in Pass 1.

**Risk 2 — Middleware ordering silence.** If the profile middleware registers after a handler-level middleware that reads `Profile` from context, `FromContext` returns nil and nothing obviously breaks until a handler uses a `Profile` field. Pass 1 audit of the router must surface the correct registration point and Pass 3 verification must exercise every chrome surface (public, /app, /hive) to confirm the context is populated on all paths.

**Risk 3 — Context-key collisions.** A string-literal context key collides with any other string-literal context key using the same value. Unexported custom type, always. This is called out in the prompt but worth reinforcing: silent collision bugs are multi-hour debugging sessions.

**Risk 4 — Templ output non-determinism.** The byte-identical-HTML bar assumes templ output is deterministic. It should be — but if any component uses `time.Now()`, a request ID, a random value, or any other non-stable input in rendered HTML, the diff won't be clean. Pass 1 audit should flag any such components. If found, the verification bar softens to "structurally identical modulo timestamps/request IDs," documented in findings §F.

**Risk 5 — Feature-flag behavior on `/hive`.** CEO decision #1 makes `/hive` under Transpara an iframe to `work:8080/telemetry/`. When `PROFILE_SYSTEM_DISABLED=1`, the site is always lovyou-ai, which means `/hive` should render the lovyou-ai chrome around the iframe (current behavior). Pass 3 verification must include the disabled-flag case on `/hive` specifically to confirm no regression.

**Risk 6 — Co-author trailer change.** Phases 1 and 2 used `Co-Authored-By: Paperclip <noreply@paperclip.ing>`. Phase 3 switches to `Co-Authored-By: transpara-ai <transpara-ai@transpara.com>` because Paperclip is removed from this project. Claude Code inheriting pattern-matching from Phase 1/2 findings may reach for the old trailer. The prompt explicitly forbids this, but it's worth watching the first commit to confirm compliance.

**Risk 7 — `Profile` struct field creep.** While touching every layout signature, there's a temptation to pre-populate `Profile` with fields Phase 4 will need (e.g., `ShellConfig`, `LogoComponent`, `RouteVisibility`). Resist. Every speculative field becomes a callsite that Phase 4 has to revise when the real shape emerges. Ship minimal; expand when the need is concrete.

If Claude Code hits any of these, findings §F. Same discipline as Phases 1 and 2.

---

## One tactical note on the two-profile registration

The `transpara` profile in Phase 3 is a placeholder — same `Name` field value as `lovyou-ai`, identical rendering, no visual or structural differences. It exists only so the resolver has a valid non-default target to resolve to. This is deliberate: it proves the chain works end-to-end without introducing any Phase 6 scope.

The temptation will be to put `"Transpara"` in the `Name` field of the Transpara profile. Don't — if `Name` is used anywhere in rendering (a page `<title>`, a footer, an `alt` attribute), the byte-identical-HTML check will fail. For Phase 3, both profiles carry whatever `Name` value the current site uses (likely the lovyou-ai display name). Phase 4 is where `Name` differentiates, because Phase 4 is where layouts begin branching on profile.

Tiny thing, easy to overlook, catches the byte-identical check in an ugly way if missed.
