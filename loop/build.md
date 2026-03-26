# Build Report — Fix: Multi-agent auto-response trigger on convene

## Gap
Critic found three violations in commit 64338c2 related to the `convene` op and council Mind logic.

## Changes

### graph/handlers.go — convene op
**Invariant 11 (IDENTITY) fix.** Removed the fallback that stored a display name in `agentIDs` when `ResolveUserID` returned empty. Unresolvable names are now skipped silently. A name is not an ID — smuggling it into a field that expects an ID corrupts `Tags` and downstream `AuthorID` in `CreateNode`.

### graph/mind.go — OnCouncilConvened
**Invariant 13 (BOUNDED) fix.** Added a cap of 10 agents per council. If `council.Tags` exceeds 10, the excess is dropped and a log line is emitted. This prevents unbounded sequential Claude API calls in a single goroutine.

### graph/mind.go — buildCouncilPrompt
**Context propagation fix.** Changed `buildCouncilPrompt` to accept a `ctx context.Context` parameter and pass it to `m.store.GetAgentPersona`. Previously `context.Background()` was used, bypassing the `replyTimeout` set in `OnCouncilConvened`. A stalled `GetAgentPersona` call will now be cancelled when the parent context times out.

## Verification
- `go.exe build -buildvcs=false ./...` — clean
- `go.exe test ./...` — all pass (graph: 2.784s)
