# Auth: email magic link for Workspace-blocked users

## What changed

**auth/auth.go** — all changes in one file:

1. **Register()**: Added `GET /auth/google` route mapped to `handleGoogleOAuth`.

2. **Renamed `handleLogin` → `handleGoogleOAuth`**: The existing login handler (which immediately redirects to Google OAuth) is now at `/auth/google`. This keeps the existing OAuth flow intact.

3. **New `handleLogin`**: Renders a proper login page at `GET /auth/login` with:
   - Google OAuth button linking to `/auth/google`
   - Collapsible `<details>` "Use email instead" section
   - Email form that posts to `/auth/magic-link/request`
   - JS-enhanced submit: shows inline "Check your email" confirmation; falls back to plain form POST if fetch fails

4. **Already present (iter 306, not changed)**:
   - `magic_link_tokens` table in `migrate()`
   - `handleMagicLinkRequestForm`, `handleMagicLinkRequest`, `handleMagicLinkVerify`
   - `requestMagicLink`, `verifyMagicLink`, `upsertUserByEmail`
   - Full test suite: happy path, expired token, used token, invalid token, idempotent user

## Verification

```
go.exe build -buildvcs=false ./...   ✅
go.exe test -buildvcs=false ./...    ✅ (all packages pass)
```

## Flow after this change

1. User visits `/auth/login` → sees Google button + "Use email instead" collapsible
2. Google-blocked users expand the section, enter email, click send
3. Server generates token, stores hash, logs the link (email delivery stubbed)
4. User clicks link → `GET /auth/magic-link/verify?token=...` → session created → redirect to `/app`
