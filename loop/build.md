# Build: Fix persona-aware routing — invariant 11 (identity)

## Gap
`GetAgentPersonaForConversation` joined `users.name → agent_personas.name` to resolve a persona for an agent participant. `users.name` is the display name — mutable and semantically distinct from the persona slug. This violates invariant 11 (join on IDs/stable identifiers, not mutable names).

## Changes

### `graph/store.go`
- Schema: `ALTER TABLE users ADD COLUMN IF NOT EXISTS persona_name TEXT` — explicit stable slug column, distinct from display name
- `GetAgentPersonaForConversation`: now selects `persona_name` (not `name`) and gates on `persona_name IS NOT NULL`, so old agent rows without the column set don't match spuriously

### `auth/auth.go`
- `ensureAgentUser`: INSERT and ON CONFLICT now also set `persona_name = agentName` (the slug — stable, never the mutable display name)

### `graph/mind_test.go`
- `no_role_tag_uses_agent_id` test now inserts agent user with `persona_name = personaSlug`, matching the new lookup path

## Verification
- `go.exe build -buildvcs=false ./...` — clean
- `go.exe test -short ./...` — all pass (auth, graph)
