# Scout Report — Iteration 351

## The Gap

**The hive dashboard work (in-progress on site) violates the Director Mandate.** The Director explicitly mandated "Engine before paint" — prioritize hive/pipeline foundation work (decision tree integration) before building new site features. This dashboard is paint. The engine isn't ready.

## Evidence

**From hive/loop/backlog.md (lines 101-119), DIRECTOR MANDATE: Engine before paint:**

> "Do NOT create feature tasks (site UI, new entity kinds, social features) until step 1 is in progress."
>
> Step 1: "Decision tree integration (eventgraph/go/pkg/decision/ → pipeline)"

**Current site state (git status):**
- `handlers/hive.go` — new HTTP handlers for reading hive loop state
- `graph/hive.templ`, `graph/hive_feed.templ` — new dashboard UI components
- `graph/handlers.go` (modified) — routes for `/hive` and `/hive/feed`

**The work in progress:** Spectator view of the hive's Scout/Builder/Critic/Reflector pipeline. Shows iteration counter, phase pill, recent commits, cost, diagnostics. Uses HTMX polling every 5s to update.

**Why this is premature:** The pipeline cannot make progress without the decision tree engine. Current failures vanish into logs. The PM cannot measure failure rates. The pipeline has no root-cause tracing. Until step 1 (decision tree integration) is complete, new features built on top of the broken pipeline add more complexity to an unreliable foundation.

## The Real Gap

The hive/pipeline needs:

1. **Decision tree integration** (highest priority, blocking everything)
   - `eventgraph/go/pkg/decision/` already exists (tree.go, evaluate.go, evolve.go)
   - Pipeline currently uses a for-loop with no failure tracing
   - Every failure is silent; no root-cause attribution
   - Decision tree: makes failures traceable, enables cost attribution, allows self-healing

2. **MCP knowledge server** (secondary, unblocks agent memory)
   - Agents currently have no search/persist across conversations
   - Knowledge layer already has assert/challenge/verify/retract ops

3. **Autorun remote** (tertiary, unblocks autonomous execution)
   - Pipeline currently runs manually via cli

## Impact

**If we ship the hive dashboard without the decision tree engine:**
- The dashboard shows pretty iteration numbers, but the hive behind it is fragile
- PM continues optimizing for visible output (the dashboard) instead of invisible reliability
- Backlog issue: "PM created duplicate test-fix directives for already-resolved issues → 3 wasted cycles"
- PM cannot see why cycles are wasted because the engine doesn't trace failures
- Product velocity appears healthy (lots of commits) but hides dead cycles (silent failures)

**Cost:** The backlog documents wastage already happening: "Architect parser failed silently 8+ times → $0.50/cycle wasted." With a decision tree, each silent failure becomes a traceable event. The hive fixes itself. The PM naturally prioritizes the engine once cost attribution is visible.

## Scope

**What's currently in progress on site:**

- Handlers to read `hive/loop/state.md`, `loop/build.md`, `loop/diagnostics.jsonl`
- `HivePage` templ component (Tailwind, dark theme, rose accent)
- `HiveDiagFeed` templ component (phase timeline)
- Routes: `GET /hive` (page), `GET /hive/feed` (JSON partial for HTMX polling)

**This is 100% site-side feature work.** It is:
- ✓ Complete in scope (dashboard works end-to-end)
- ✗ Violates the director mandate (must pause until hive foundation is done)

## Suggestion

**Three options:**

### Option 1: PAUSE this work. Shift to hive pipeline (recommended).
- Stash `handlers/hive.go`, `graph/hive*.templ` changes
- Move the team to `hive/pkg/runner/`
- Goal: Integrate `eventgraph/go/pkg/decision/` into the pipeline
- Timeline: Unknown (foundational work, no estimate)
- Risk: Low. This is a forced architectural simplification, not new code.

### Option 2: SHIP this work. Accept the mandate violation.
- Commit the dashboard as-is
- Simultaneously kick off decision tree integration work in parallel (hive repo)
- Risk: Decision tree work becomes secondary. The dashboard will demand fixes, features, polish. The mandate will be forgotten.

### Option 3: MERGE and DOCUMENT the mandate.
- Commit the dashboard
- Add an entry to CLAUDE.md: "Before next site feature: complete hive decision tree integration"
- Rely on discipline to enforce it next iteration
- Risk: Same as Option 2. Discipline hasn't worked (the PM already violated the mandate without realizing it).

## Recommendation

**Option 1: Pause and pivot to hive.**

The mandate is explicit. The engine is in danger. The dashboard is ship-quality but premature. Stash the code (keep it in git so it's not lost) and shift focus to making the pipeline self-aware before adding more layers on top.
