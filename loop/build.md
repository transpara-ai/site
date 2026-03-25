# Build: Add last message preview to Chat lens conversation list

## What changed

### `graph/store.go`
- Added `ALTER TABLE nodes ADD COLUMN IF NOT EXISTS last_message_preview TEXT NOT NULL DEFAULT ''` to `migrate()`
- Added `UpdateLastMessagePreview(ctx, conversationID, body string) error` — truncates to 100 chars and updates the column + `updated_at`
- Updated `ListConversations` query: uses `COALESCE(NULLIF(n.last_message_preview, ''), lm.body)` so new messages use the column directly while existing conversations fall back to the lateral join

### `graph/handlers.go`
- In the `respond` op handler, after notifying conversation participants, calls `h.store.UpdateLastMessagePreview(ctx, parentID, body)` when the parent is a conversation

### `graph/mind.go`
- After the agent reply node is created and `RecordOp` is called, also calls `m.store.UpdateLastMessagePreview(ctx, convo.ID, cleanResponse)` so agent replies update the preview too

## What was already done
The template (`ConversationsView` in `views.templ`) already displayed `convo.LastBody` with author attribution — no template changes were needed.

## Verification
- `go.exe build -buildvcs=false ./...` — clean
- `go.exe test ./...` — all pass (including `TestConversation` which checks `LastBody`)
