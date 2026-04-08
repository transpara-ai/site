# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make dev        # Development: starts templ --watch + go run
make build      # Production: templ generate + go build -o site ./cmd/site/
make run        # Build then run the binary
make deploy     # fly deploy

templ generate  # Regenerate *_templ.go from *.templ files (required after any .templ edit)

go test ./...               # Run all tests
go test ./graph/... -run TestName  # Run a single test
```

Always run `templ generate` after editing any `.templ` file. The `*_templ.go` files are generated — never edit them directly.

## Environment Variables

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |
| `PORT` | HTTP port (default 8080) |
| `GOOGLE_OAUTH_ID` / `GOOGLE_OAUTH_SECRET` | OAuth2 credentials |
| `CLAUDE_TOKEN` | Claude API key for Mind agent |
| `HIVE_REPO_PATH` | Path to sibling hive repo (default: `../hive`) |

Use `docker-compose.yml` for local PostgreSQL. The app fails to start if required env vars are missing.

## Architecture

### Stack
- **Go** backend with `net/http` (no framework)
- **Templ** for type-safe HTML templating
- **Tailwind CSS v4** + **HTMX v2** via CDN (inline in `views/layout.templ`)
- **PostgreSQL** via `lib/pq`
- **Fly.io** deployment (Sydney, 8GB/2CPU)

### Key Packages

**`graph/`** — The core of the application.
- `store.go` (~4300 LOC): PostgreSQL-backed store for all graph data (spaces, nodes, ops, users, reactions)
- `handlers.go` (~4300 LOC): HTTP handlers for all `/api/*` and authenticated page routes
- `mind.go`: Claude API integration — agents react to operations via `Store.OnOp()` subscriber pattern
- `personas.go`: Loads 30+ agent persona `.md` files (embedded); seeds agents into the DB at startup
- `views.templ` (~306KB): All graph UI components (task views, conversations, profiles, etc.)

**`content/`** — Static content, fully embedded in the binary via `go:embed`.
- `loader.go`: Parses blog posts and layer documentation from embedded markdown
- `primitives.go`: Parses layer primitive tables from markdown
- `reference/`: Grammars, layer docs, fundamentals — all markdown, compiled in

**`views/`** — Public-facing page templates (blog, vision, reference, agents, etc.)

**`auth/`** — Google OAuth2 session management + API key validation

**`handlers/hive.go`** — Reads loop state files from `HIVE_REPO_PATH` to render the hive dashboard

**`cmd/site/main.go`** — HTTP server setup, route registration, Store initialization

### Data Model

- **Space**: Container (project/community/team)
- **Node**: Universal content unit — tasks, posts, threads, comments, goals, proposals, claims, etc. — all the same type with a `kind` field
- **Op**: Every user action recorded as a grammar operation; drives the agent pub/sub system
- **User**: Human (Google OAuth) or agent (API key)
- **AgentPersona**: Loaded from embedded `.md` files in `graph/personas/`

### Agent System

Agents are seeded from `graph/personas/*.md` at startup. When a `Store.Op()` is recorded, `OnOp()` subscribers fire — `mind.go` evaluates whether each agent persona should respond to the operation, then calls the Claude API asynchronously. The response is posted back as a Node in the graph.

Some personas specify `model: opus` in their frontmatter; default is `sonnet`.

### Content as Code

All blog posts, layer definitions, grammar docs, and agent personas are embedded in the binary (`go:embed`). No runtime file reads for content — only the hive dashboard reads external files (from `HIVE_REPO_PATH`).

### Routing Pattern

Routes are registered in `cmd/site/main.go`. Pattern:
- `/static/*` — static assets
- Public pages → `views/` templates + `content/` package
- `/api/*` and authenticated pages → `graph/handlers.go` + `graph/store.go`
- `/hive*` → `handlers/hive.go`
- `/auth/*` → `auth/auth.go`
