# Build Report — First Completion Toast

## Gap
After an agent completes a task for the first time in a space, the user has no nudge to explore Chat. Users miss the conversational dimension of the product.

## Changes

### `graph/store.go`
- **Migration**: `ALTER TABLE spaces ADD COLUMN IF NOT EXISTS first_completion_at TIMESTAMPTZ;`
- **Space struct**: Added `FirstCompletionAt *time.Time` field
- **GetSpaceByID / GetSpaceBySlug**: Updated SELECT + Scan to include `first_completion_at`
- **MarkFirstCompletion(ctx, spaceID) (bool, error)**: Atomically sets `first_completion_at = NOW()` WHERE NULL; returns true if updated (i.e., this was the first completion)

### `graph/handlers.go`
- **handleNodeState**: After recording the `complete` op, calls `MarkFirstCompletion`. If it's the first, redirects to `/app/{slug}/board?first_completion=1` instead of bare board URL. Also adds `first_completion` to JSON response.
- **handleOp "complete"**: Same — calls `MarkFirstCompletion`, adjusts redirect/JSON.
- **handleBoard**: Reads `?first_completion=1` query param, passes `showFirstCompletionToast bool` to `BoardView`.

### `graph/views.templ`
- **firstCompletionToast(spaceSlug)**: New component. Fixed-position toast with: checkmark icon, "Your AI teammate just helped!", "Try Chat →" link to conversations lens, dismiss button, 8-second auto-dismiss via JS.
- **BoardView**: Added `showFirstCompletionToast bool` param. Renders toast above board content when true.

## Verification
- `templ generate`: 13 updates, no errors
- `go build -buildvcs=false ./...`: clean
- `go test ./...`: all pass (auth, graph)
