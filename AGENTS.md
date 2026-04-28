# AGENTS.md

## Purpose
Transpara AI site and operator-facing product surface. It presents embedded content, graph UI, auth, hive dashboard, and public pages.

## Commands
- CSS: `make css`
- Generate templates: `make generate`
- Build: `make build`
- Test: `make test`
- Verify: `make verify`
- Dev server: `make dev`

## Rules
- Run `templ generate` after editing `.templ` files; never edit generated `*_templ.go` files directly.
- Preserve embedded content behavior and route registration patterns.
- UI changes should keep the operational graph legible and efficient, not just visually changed.
- Do not push to `upstream`; `origin` is the writable fork.

## Exit Criteria
- `make verify` passes, or the blocker is explicit.
- Template changes regenerate outputs.
- Public, auth, or dashboard behavior changes include tests or a clear validation note.
