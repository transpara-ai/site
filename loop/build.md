# Build Report — Public Demo Space

## Gap
Anonymous visitors had no way to experience the product before signing in. The landing page linked to a blog post instead of a live demo.

## Changes

### `graph/store.go`
- Added `"log"` import
- Added `DemoSpaceSlug = "demo"` constant
- Added `SeedDemoSpace(ctx context.Context) string` method:
  - Idempotent: checks if demo space exists first, returns slug immediately if so
  - Upserts a `kind='agent'` user with `google_id = "system:demo-agent"` as demo owner
  - Creates public project space: slug `"demo"`, name `"lovyou.ai Demo"`, visibility `"public"`
  - Seeds 4 tasks (1 done, 1 active, 2 open) to populate the board
  - Seeds 1 post to populate the feed
  - Seeds 1 conversation + 1 agent reply to populate chat
  - Records ops for each node (intend/start/complete/express/converse/respond)
  - Returns `""` on any error

### `views/home.templ`
- Added `DemoSlug string` field to `HomeStats` struct
- Updated hero CTA buttons: if `DemoSlug != ""`, shows "See how it works" link to `/app/{DemoSlug}/board`; otherwise falls back to "How it works" blog link

### `cmd/site/main.go`
- Added `"context"` import
- After `graph.NewStore(db)`, calls `graphStore.SeedDemoSpace(context.Background())` → `demoSlug`
- Passes `DemoSlug: demoSlug` to `HomeStats` in the home handler

## Read-only mode
No handler changes needed. The existing system already enforces read-only for anonymous users:
- `spaceForRead` sets `isOwner = false` for anonymous users
- Templates gate all write forms on `user.Name != "Anonymous"` (board `canWrite`, feed compose, etc.)
- POST routes use `writeWrap` which requires authentication

## Verification
- `templ generate` — 13 updates, no errors
- `go.exe build -buildvcs=false ./...` — no errors
- `go.exe test ./...` — all pass (auth, graph packages)
