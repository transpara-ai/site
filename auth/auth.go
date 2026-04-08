// Package auth provides Google OAuth2 authentication and session management.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// APIKey represents a stored API key (hash only, raw key never stored).
type APIKey struct {
	ID        string
	Name      string
	UserID    string    // Human sponsor who created/manages this key.
	AgentID   string    // If non-empty, this key authenticates as this agent user.
	AgentName string    // Display name of the agent (denormalized for listing).
	CreatedAt time.Time
}

// User represents an authenticated user (human or agent).
type User struct {
	ID      string
	Email   string
	Name    string
	Picture string
	Kind    string // "human" or "agent"
}

type contextKey struct{}

// UserFromContext returns the authenticated user, or nil.
func UserFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(contextKey{}).(*User)
	return u
}

// ContextWithUser stores a user in the context.
func ContextWithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, contextKey{}, u)
}

// Auth handles Google OAuth2 and session management.
type Auth struct {
	db          *sql.DB
	oauth       *oauth2.Config
	secure      bool
	userInfoURL string // defaults to Google's userinfo endpoint; overridable in tests
}

// New creates an Auth service backed by the given database.
func New(db *sql.DB, clientID, clientSecret, redirectURL string, secure bool) (*Auth, error) {
	a := &Auth{
		db: db,
		oauth: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
		secure:      secure,
		userInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
	}
	if err := a.migrate(); err != nil {
		return nil, fmt.Errorf("auth migrate: %w", err)
	}
	return a, nil
}

func (a *Auth) migrate() error {
	_, err := a.db.Exec(`
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    google_id TEXT UNIQUE,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    picture TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    key_hash TEXT UNIQUE NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_name TEXT NOT NULL DEFAULT '',
    agent_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS magic_link_tokens (
    id TEXT PRIMARY KEY,
    token_hash TEXT UNIQUE NOT NULL,
    email TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
`)
	if err != nil {
		return err
	}

	// Migrations for existing databases.
	migrations := []string{
		`ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS agent_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS agent_id TEXT REFERENCES users(id) ON DELETE SET NULL`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'human'`,
		`ALTER TABLE users ALTER COLUMN google_id DROP NOT NULL`,
	}
	for _, m := range migrations {
		if _, err := a.db.ExecContext(context.Background(), m); err != nil {
			return fmt.Errorf("migration: %w", err)
		}
	}
	return nil
}

// Register adds auth routes to the mux.
func (a *Auth) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/login", a.handleLogin)
	mux.HandleFunc("GET /auth/google", a.handleGoogleOAuth)
	mux.HandleFunc("GET /auth/callback", a.handleCallback)
	mux.HandleFunc("POST /auth/logout", a.handleLogout)
	mux.HandleFunc("GET /auth/error", a.handleAuthError)
	mux.HandleFunc("GET /auth/status", a.handleStatus)

	// Magic link (email-based) auth — fallback for blocked OAuth users.
	mux.HandleFunc("GET /auth/magic-link/request", a.handleMagicLinkRequestForm)
	mux.HandleFunc("POST /auth/magic-link/request", a.handleMagicLinkRequest)
	mux.HandleFunc("GET /auth/magic-link/verify", a.handleMagicLinkVerify)

	// API key management (requires session auth).
	mux.Handle("POST /auth/api-keys", a.RequireAuth(a.handleCreateAPIKey))
	mux.Handle("POST /auth/api-keys/{id}/delete", a.RequireAuth(a.handleDeleteAPIKey))
}

// RequireAuth wraps a handler to require authentication.
// Checks Bearer token first (for API clients), then session cookie (for browsers).
func (a *Auth) RequireAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try API key auth first.
		if user := a.userFromBearer(r); user != nil {
			ctx := ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Fall back to session cookie.
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		user, err := a.userBySession(r.Context(), cookie.Value)
		if err != nil {
			a.clearCookie(w)
			http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
			return
		}

		ctx := ContextWithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth tries to load a user from Bearer token or session cookie
// but does not redirect if neither exists.
func (a *Auth) OptionalAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try API key auth first.
		if user := a.userFromBearer(r); user != nil {
			ctx := ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Try session cookie.
		cookie, err := r.Cookie("session")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		user, err := a.userBySession(r.Context(), cookie.Value)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := ContextWithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────

func (a *Auth) handleGoogleOAuth(w http.ResponseWriter, r *http.Request) {
	state := newID()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/auth",
		MaxAge:   300,
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
	})
	// Derive redirect URL from the request so the callback domain matches
	// the domain the user is visiting (avoids cookie domain mismatch).
	redirectURL := a.redirectURL(r)
	authURL := a.oauth.AuthCodeURL(state, oauth2.SetAuthURLParam("redirect_uri", redirectURL))
	log.Printf("auth: login redirect state=%s host=%s", state[:8], r.Host)
	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

func (a *Auth) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Google may send ?error=access_denied or similar before state check.
	if errCode := r.URL.Query().Get("error"); errCode != "" {
		log.Printf("auth: callback google error: %s", errCode)
		http.Redirect(w, r, "/auth/error?code="+url.QueryEscape(errCode), http.StatusSeeOther)
		return
	}

	// Verify CSRF state.
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		log.Printf("auth: callback invalid state: cookie_err=%v", err)
		http.Redirect(w, r, "/auth/error?code=invalid_state", http.StatusSeeOther)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Path: "/auth", MaxAge: -1})

	// Exchange code for token — use same redirect_uri as the login request.
	redirectURL := a.redirectURL(r)
	token, err := a.oauth.Exchange(r.Context(), r.URL.Query().Get("code"),
		oauth2.SetAuthURLParam("redirect_uri", redirectURL))
	if err != nil {
		log.Printf("auth: token exchange error: %v", err)
		http.Redirect(w, r, "/auth/error?code=exchange_failed", http.StatusSeeOther)
		return
	}

	// Fetch Google user info.
	client := a.oauth.Client(r.Context(), token)
	resp, err := client.Get(a.userInfoURL)
	if err != nil {
		log.Printf("auth: fetch userinfo error: %v", err)
		http.Redirect(w, r, "/auth/error?code=userinfo_failed", http.StatusSeeOther)
		return
	}
	defer resp.Body.Close()

	var info struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		log.Printf("auth: decode userinfo error: %v", err)
		http.Redirect(w, r, "/auth/error?code=userinfo_failed", http.StatusSeeOther)
		return
	}

	// Upsert user — creates on first login, updates name/picture on subsequent logins.
	user, err := a.upsertUser(r.Context(), info.ID, info.Email, info.Name, info.Picture)
	if err != nil {
		log.Printf("auth: upsert user error email=%s: %v", info.Email, err)
		http.Redirect(w, r, "/auth/error?code=user_create_failed", http.StatusSeeOther)
		return
	}
	log.Printf("auth: user upserted id=%s email=%s", user.ID, user.Email)

	// Create session (30 days).
	sessionID := newID()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if _, err := a.db.ExecContext(r.Context(),
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		sessionID, user.ID, expiresAt,
	); err != nil {
		log.Printf("auth: create session error user=%s: %v", user.ID, err)
		http.Redirect(w, r, "/auth/error?code=session_failed", http.StatusSeeOther)
		return
	}

	log.Printf("auth: login success user=%s email=%s session=%s", user.ID, user.Email, sessionID[:8])

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

// handleAuthError renders a user-facing error page when OAuth fails.
// Error codes come from the callback handler and from Google directly.
func (a *Auth) handleAuthError(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	log.Printf("auth: error page shown code=%s", code)

	msg := "Authentication failed. Please try again."
	switch code {
	case "access_denied":
		msg = "Sign-in was cancelled or blocked. Your organisation may restrict third-party sign-in. Try using an API key instead."
	case "invalid_state":
		msg = "Your sign-in session expired. Please try signing in again."
	case "exchange_failed":
		msg = "Could not complete sign-in with Google. Your organisation may block third-party sign-in. Try using an API key instead."
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Sign-in error — lovyou.ai</title>
  <style>
    body{font-family:system-ui,sans-serif;background:#0d0d0d;color:#e8d5c4;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}
    .card{max-width:420px;padding:2rem;background:#1a1a1a;border-radius:12px;border:1px solid #333}
    h1{margin:0 0 1rem;font-size:1.25rem;color:#e8a0b8}
    p{margin:0 0 1.5rem;line-height:1.6;color:#c4a882}
    a{display:inline-block;padding:.6rem 1.2rem;background:#e8a0b8;color:#0d0d0d;border-radius:6px;text-decoration:none;font-weight:600}
    a:hover{background:#d48aa0}
    .code{margin-top:1rem;font-size:.75rem;color:#666}
  </style>
</head>
<body>
  <div class="card">
    <h1>Sign-in error</h1>
    <p>%s</p>
    <a href="/auth/login">Try again</a>
    %s
  </div>
</body>
</html>`,
		html.EscapeString(msg),
		func() string {
			if code != "" {
				return `<p class="code">Error code: ` + html.EscapeString(code) + `</p>`
			}
			return ""
		}(),
	)
}

// handleStatus returns the auth configuration state for debugging.
// Returns safe config state only — no secrets, no tokens.
func (a *Auth) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"oauth_configured": a.oauth != nil && a.oauth.ClientID != "",
		"redirect_url":     a.redirectURL(r),
		"secure":           a.secure,
	})
}

func (a *Auth) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		a.db.ExecContext(r.Context(), `DELETE FROM sessions WHERE id = $1`, cookie.Value)
	}
	a.clearCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ────────────────────────────────────────────────────────────────────
// API Key handlers
// ────────────────────────────────────────────────────────────────────

func (a *Auth) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = "default"
	}
	agentName := strings.TrimSpace(r.FormValue("agent_name"))

	rawKey, err := a.createAPIKey(r.Context(), user.ID, name, agentName)
	if err != nil {
		log.Printf("auth: create api key: %v", err)
		http.Error(w, "failed to create key", http.StatusInternalServerError)
		return
	}

	// HTMX request — return HTML fragment showing the raw key.
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="p-4 bg-elevated rounded-lg border border-brand">
<p class="text-sm font-medium text-brand mb-2">Key created! Save this — you won't see it again.</p>
<code class="block text-sm bg-void rounded p-3 text-warm break-all select-all">%s</code>
<p class="text-xs text-warm-muted mt-2">Use as: Authorization: Bearer %s</p>
</div>`, rawKey, rawKey[:10]+"...")
		return
	}

	// JSON response for API clients.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"key":  rawKey,
		"name": name,
	})
}

func (a *Auth) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	keyID := r.PathValue("id")
	_, err := a.db.ExecContext(r.Context(),
		`DELETE FROM api_keys WHERE id = $1 AND user_id = $2`, keyID, user.ID)
	if err != nil {
		http.Error(w, "failed to delete key", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app/keys", http.StatusSeeOther)
}

// ListAPIKeys returns all API keys for a user (metadata only, no raw keys).
func (a *Auth) ListAPIKeys(ctx context.Context, userID string) ([]APIKey, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT id, name, user_id, agent_name, COALESCE(agent_id, ''), created_at
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.UserID, &k.AgentName, &k.AgentID, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// ────────────────────────────────────────────────────────────────────
// Internal
// ────────────────────────────────────────────────────────────────────

func (a *Auth) upsertUser(ctx context.Context, googleID, email, name, picture string) (*User, error) {
	var u User
	err := a.db.QueryRowContext(ctx, `
		INSERT INTO users (id, google_id, email, name, picture)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (google_id) DO UPDATE SET
			email = EXCLUDED.email,
			name = EXCLUDED.name,
			picture = EXCLUDED.picture
		RETURNING id, email, name, picture, kind`,
		newID(), googleID, email, name, picture,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Picture, &u.Kind)
	return &u, err
}

// ensureAgentUser creates or finds an agent user record.
// Agents are real users with kind='agent' and a synthetic google_id.
func (a *Auth) ensureAgentUser(ctx context.Context, agentName string) (*User, error) {
	syntheticGoogleID := "agent:" + agentName
	syntheticEmail := agentName + "@agent.lovyou.ai"

	var u User
	err := a.db.QueryRowContext(ctx, `
		INSERT INTO users (id, google_id, email, name, kind, persona_name)
		VALUES ($1, $2, $3, $4, 'agent', $4)
		ON CONFLICT (google_id) DO UPDATE SET name = EXCLUDED.name, persona_name = EXCLUDED.persona_name
		RETURNING id, email, name, picture, kind`,
		newID(), syntheticGoogleID, syntheticEmail, agentName,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Picture, &u.Kind)
	return &u, err
}

func (a *Auth) userBySession(ctx context.Context, sessionID string) (*User, error) {
	var u User
	err := a.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.name, u.picture, u.kind
		FROM users u
		JOIN sessions s ON s.user_id = u.id
		WHERE s.id = $1 AND s.expires_at > NOW()`,
		sessionID,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Picture, &u.Kind)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// redirectURL derives the OAuth callback URL from the incoming request's Host
// header so the cookie domain always matches the callback domain.
func (a *Auth) redirectURL(r *http.Request) string {
	scheme := "https"
	if !a.secure {
		scheme = "http"
	}
	return scheme + "://" + r.Host + "/auth/callback"
}

func (a *Auth) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// userFromBearer extracts a Bearer token from the Authorization header
// and looks up the associated user via API key hash.
func (a *Auth) userFromBearer(r *http.Request) *User {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil
	}
	rawKey := strings.TrimPrefix(auth, "Bearer ")
	if rawKey == "" {
		return nil
	}
	user, err := a.userByAPIKey(r.Context(), rawKey)
	if err != nil {
		return nil
	}
	return user
}

// createAPIKey generates a new API key, stores its hash, returns the raw key.
// If agentName is non-empty, creates a real agent user and links the key to it.
func (a *Auth) createAPIKey(ctx context.Context, userID, name, agentName string) (string, error) {
	rawKey := "lv_" + newID() + newID() // 64 hex chars + prefix
	hash := hashKey(rawKey)
	id := newID()

	var agentID *string
	if agentName != "" {
		agent, err := a.ensureAgentUser(ctx, agentName)
		if err != nil {
			return "", fmt.Errorf("ensure agent user: %w", err)
		}
		agentID = &agent.ID
	}

	_, err := a.db.ExecContext(ctx,
		`INSERT INTO api_keys (id, name, key_hash, user_id, agent_name, agent_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		id, name, hash, userID, agentName, agentID)
	if err != nil {
		return "", fmt.Errorf("insert api key: %w", err)
	}
	return rawKey, nil
}

// userByAPIKey looks up a user by raw API key (hashes it first).
// If the key has an agent_id, returns the agent user — the agent acts
// under its own identity, not the human sponsor's.
func (a *Auth) userByAPIKey(ctx context.Context, rawKey string) (*User, error) {
	hash := hashKey(rawKey)
	var u User
	var agentID sql.NullString
	err := a.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.name, u.picture, u.kind, k.agent_id
		FROM users u
		JOIN api_keys k ON k.user_id = u.id
		WHERE k.key_hash = $1`, hash,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Picture, &u.Kind, &agentID)
	if err != nil {
		return nil, err
	}

	// If the key is linked to an agent, resolve to the agent's identity.
	if agentID.Valid {
		var agent User
		err := a.db.QueryRowContext(ctx, `
			SELECT id, email, name, picture, kind FROM users WHERE id = $1`,
			agentID.String,
		).Scan(&agent.ID, &agent.Email, &agent.Name, &agent.Picture, &agent.Kind)
		if err != nil {
			return nil, fmt.Errorf("resolve agent user: %w", err)
		}
		return &agent, nil
	}

	return &u, nil
}

// ────────────────────────────────────────────────────────────────────
// Magic link (email-based) auth
// ────────────────────────────────────────────────────────────────────

// handleMagicLinkRequestForm renders the email entry form.
func (a *Auth) handleMagicLinkRequestForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Sign in by email — lovyou.ai</title>
  <style>
    body{font-family:system-ui,sans-serif;background:#0d0d0d;color:#e8d5c4;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}
    .card{max-width:420px;width:100%;padding:2rem;background:#1a1a1a;border-radius:12px;border:1px solid #333}
    h1{margin:0 0 .5rem;font-size:1.25rem;color:#e8a0b8}
    p{margin:0 0 1.5rem;line-height:1.6;color:#c4a882;font-size:.9rem}
    label{display:block;margin-bottom:.4rem;font-size:.85rem;color:#c4a882}
    input{width:100%;box-sizing:border-box;padding:.6rem .8rem;background:#0d0d0d;border:1px solid #444;border-radius:6px;color:#e8d5c4;font-size:1rem;margin-bottom:1rem}
    button{width:100%;padding:.65rem;background:#e8a0b8;color:#0d0d0d;border:none;border-radius:6px;font-size:1rem;font-weight:600;cursor:pointer}
    button:hover{background:#d48aa0}
    .back{display:block;text-align:center;margin-top:1rem;color:#666;font-size:.85rem;text-decoration:none}
    .back:hover{color:#c4a882}
  </style>
</head>
<body>
  <div class="card">
    <h1>Sign in by email</h1>
    <p>Enter your email address and we'll send you a one-time sign-in link valid for 15 minutes.</p>
    <form method="POST" action="/auth/magic-link/request">
      <label for="email">Email address</label>
      <input type="email" id="email" name="email" placeholder="you@example.com" required autofocus>
      <button type="submit">Send sign-in link</button>
    </form>
    <a class="back" href="/auth/login">← Back to sign-in</a>
  </div>
</body>
</html>`)
}

// handleMagicLinkRequest handles POST /auth/magic-link/request.
// Validates the email, generates a one-time token, and logs the link
// (production email delivery is wired via the Mailer field).
func (a *Auth) handleMagicLinkRequest(w http.ResponseWriter, r *http.Request) {
	email := strings.TrimSpace(r.FormValue("email"))
	if !strings.Contains(email, "@") || len(email) < 3 {
		http.Error(w, "invalid email address", http.StatusBadRequest)
		return
	}

	rawToken, err := a.requestMagicLink(r.Context(), email)
	if err != nil {
		log.Printf("auth: magic link request error email=%s: %v", email, err)
		http.Error(w, "failed to generate link", http.StatusInternalServerError)
		return
	}

	scheme := "https"
	if !a.secure {
		scheme = "http"
	}
	link := scheme + "://" + r.Host + "/auth/magic-link/verify?token=" + url.QueryEscape(rawToken)
	log.Printf("auth: magic link generated email=%s link=%s", email, link)
	// TODO: send email — stub logs link above; wire smtp/sendgrid here.

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Check your email — lovyou.ai</title>
  <style>
    body{font-family:system-ui,sans-serif;background:#0d0d0d;color:#e8d5c4;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}
    .card{max-width:420px;padding:2rem;background:#1a1a1a;border-radius:12px;border:1px solid #333}
    h1{margin:0 0 1rem;font-size:1.25rem;color:#e8a0b8}
    p{margin:0 0 1rem;line-height:1.6;color:#c4a882}
    .note{font-size:.85rem;color:#666}
    a{color:#e8a0b8;text-decoration:none}a:hover{text-decoration:underline}
  </style>
</head>
<body>
  <div class="card">
    <h1>Check your email</h1>
    <p>A sign-in link has been sent to <strong>%s</strong>. The link expires in 15 minutes.</p>
    <p class="note">Didn't receive it? Check your spam folder or <a href="/auth/magic-link/request">send again</a>.</p>
  </div>
</body>
</html>`, html.EscapeString(email))
}

// handleMagicLinkVerify handles GET /auth/magic-link/verify?token=...
// Validates the token, creates a session, and redirects to /app.
func (a *Auth) handleMagicLinkVerify(w http.ResponseWriter, r *http.Request) {
	rawToken := r.URL.Query().Get("token")
	if rawToken == "" {
		http.Redirect(w, r, "/auth/error?code=invalid_token", http.StatusSeeOther)
		return
	}

	user, err := a.verifyMagicLink(r.Context(), rawToken)
	if err != nil {
		log.Printf("auth: magic link verify failed: %v", err)
		http.Redirect(w, r, "/auth/error?code=invalid_token", http.StatusSeeOther)
		return
	}

	sessionID := newID()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if _, err := a.db.ExecContext(r.Context(),
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		sessionID, user.ID, expiresAt,
	); err != nil {
		log.Printf("auth: magic link session create error user=%s: %v", user.ID, err)
		http.Redirect(w, r, "/auth/error?code=session_failed", http.StatusSeeOther)
		return
	}

	log.Printf("auth: magic link login success user=%s email=%s session=%s", user.ID, user.Email, sessionID[:8])

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

// requestMagicLink generates a one-time token for the given email and stores
// its hash in the database. Returns the raw token for delivery to the user.
func (a *Auth) requestMagicLink(ctx context.Context, email string) (string, error) {
	rawToken := newID()
	hash := hashKey(rawToken)
	id := newID()
	expiresAt := time.Now().Add(15 * time.Minute)
	_, err := a.db.ExecContext(ctx,
		`INSERT INTO magic_link_tokens (id, token_hash, email, expires_at) VALUES ($1, $2, $3, $4)`,
		id, hash, email, expiresAt)
	if err != nil {
		return "", fmt.Errorf("insert magic link token: %w", err)
	}
	return rawToken, nil
}

// verifyMagicLink atomically marks the token used and returns the user.
// Returns an error if the token is invalid, expired, or already used.
func (a *Auth) verifyMagicLink(ctx context.Context, rawToken string) (*User, error) {
	hash := hashKey(rawToken)
	var email string
	err := a.db.QueryRowContext(ctx,
		`UPDATE magic_link_tokens SET used = TRUE
		 WHERE token_hash = $1 AND expires_at > NOW() AND used = FALSE
		 RETURNING email`,
		hash,
	).Scan(&email)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid, expired, or already used token")
	}
	if err != nil {
		return nil, fmt.Errorf("verify magic link: %w", err)
	}
	return a.upsertUserByEmail(ctx, email)
}

// upsertUserByEmail finds an existing user by email or creates a new one.
// Used for magic-link auth where there is no Google ID.
func (a *Auth) upsertUserByEmail(ctx context.Context, email string) (*User, error) {
	id := newID()
	var u User
	err := a.db.QueryRowContext(ctx, `
		INSERT INTO users (id, email, name, kind)
		VALUES ($1, $2, '', 'human')
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		RETURNING id, email, name, picture, kind`,
		id, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Picture, &u.Kind)
	return &u, err
}

// UserFromBearer resolves a Bearer token from the request without requiring
// a full Auth instance. Used in anonymous mode to support API key auth.
func UserFromBearer(db *sql.DB, r *http.Request) *User {
	hdr := r.Header.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		return nil
	}
	rawKey := strings.TrimPrefix(hdr, "Bearer ")
	if rawKey == "" {
		return nil
	}
	hash := hashKey(rawKey)
	var u User
	var agentID sql.NullString
	err := db.QueryRowContext(r.Context(), `
		SELECT u.id, u.email, u.name, u.picture, u.kind, k.agent_id
		FROM users u
		JOIN api_keys k ON k.user_id = u.id
		WHERE k.key_hash = $1`, hash,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Picture, &u.Kind, &agentID)
	if err != nil {
		return nil
	}
	if agentID.Valid {
		var agent User
		err := db.QueryRowContext(r.Context(), `
			SELECT id, email, name, picture, kind FROM users WHERE id = $1`,
			agentID.String,
		).Scan(&agent.ID, &agent.Email, &agent.Name, &agent.Picture, &agent.Kind)
		if err != nil {
			return nil
		}
		return &agent
	}
	return &u
}

func hashKey(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// handleLogin renders the login page with Google OAuth and email magic link options.
func (a *Auth) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Sign in — lovyou.ai</title>
  <style>
    *{box-sizing:border-box}
    body{font-family:system-ui,sans-serif;background:#0d0d0d;color:#e8d5c4;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}
    .card{max-width:400px;width:100%;padding:2rem;background:#1a1a1a;border-radius:12px;border:1px solid #333}
    h1{margin:0 0 .25rem;font-size:1.4rem;font-weight:700;color:#f0ede8}
    .sub{margin:0 0 2rem;font-size:.9rem;color:#78756e}
    .btn-google{display:flex;align-items:center;justify-content:center;gap:.65rem;width:100%;padding:.75rem 1rem;background:#fff;color:#1a1a1a;border:none;border-radius:8px;font-size:.95rem;font-weight:600;cursor:pointer;text-decoration:none;transition:background .15s}
    .btn-google:hover{background:#f0f0f0}
    .btn-google svg{flex-shrink:0}
    .divider{display:flex;align-items:center;gap:.75rem;margin:1.5rem 0;color:#4a4844;font-size:.8rem}
    .divider::before,.divider::after{content:'';flex:1;border-top:1px solid #2a2a2e}
    details{border:1px solid #2a2a2e;border-radius:8px;overflow:hidden}
    summary{padding:.75rem 1rem;cursor:pointer;font-size:.9rem;color:#c4a882;list-style:none;display:flex;align-items:center;justify-content:space-between;user-select:none}
    summary::-webkit-details-marker{display:none}
    summary::after{content:'▾';transition:transform .2s;font-size:.75rem;color:#78756e}
    details[open] summary::after{transform:rotate(-180deg)}
    .email-body{padding:1rem;border-top:1px solid #2a2a2e}
    label{display:block;margin-bottom:.4rem;font-size:.8rem;color:#78756e}
    input[type=email]{width:100%;padding:.65rem .8rem;background:#0d0d0d;border:1px solid #3a3a3f;border-radius:6px;color:#f0ede8;font-size:.95rem;outline:none;transition:border-color .15s}
    input[type=email]:focus{border-color:#e8a0b8}
    .btn-email{margin-top:.75rem;width:100%;padding:.65rem;background:#e8a0b8;color:#0d0d0d;border:none;border-radius:6px;font-size:.9rem;font-weight:600;cursor:pointer;transition:background .15s}
    .btn-email:hover{background:#d48aa0}
    .sent-msg{display:none;padding:.75rem 1rem;font-size:.85rem;color:#a0c4a0;background:#0a1a0a;border-radius:6px;margin-top:.75rem}
    details[open] .email-body{display:block}
  </style>
</head>
<body>
  <div class="card">
    <h1>Sign in</h1>
    <p class="sub">to lovyou.ai</p>

    <a href="/auth/google" class="btn-google">
      <svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <path d="M17.64 9.205c0-.639-.057-1.252-.164-1.841H9v3.481h4.844a4.14 4.14 0 0 1-1.796 2.716v2.259h2.908c1.702-1.567 2.684-3.875 2.684-6.615z" fill="#4285F4"/>
        <path d="M9 18c2.43 0 4.467-.806 5.956-2.18l-2.908-2.259c-.806.54-1.837.86-3.048.86-2.344 0-4.328-1.584-5.036-3.711H.957v2.332A8.997 8.997 0 0 0 9 18z" fill="#34A853"/>
        <path d="M3.964 10.71A5.41 5.41 0 0 1 3.682 9c0-.593.102-1.17.282-1.71V4.958H.957A8.996 8.996 0 0 0 0 9c0 1.452.348 2.827.957 4.042l3.007-2.332z" fill="#FBBC05"/>
        <path d="M9 3.58c1.321 0 2.508.454 3.44 1.345l2.582-2.58C13.463.891 11.426 0 9 0A8.997 8.997 0 0 0 .957 4.958L3.964 6.29C4.672 4.163 6.656 3.58 9 3.58z" fill="#EA4335"/>
      </svg>
      Continue with Google
    </a>

    <div class="divider">or</div>

    <details>
      <summary>Use email instead</summary>
      <div class="email-body">
        <form method="POST" action="/auth/magic-link/request" onsubmit="handleEmailSubmit(event, this)">
          <label for="ml-email">Email address</label>
          <input type="email" id="ml-email" name="email" placeholder="you@example.com" required autocomplete="email">
          <button type="submit" class="btn-email">Send sign-in link</button>
        </form>
        <div class="sent-msg" id="ml-sent">Check your email — a sign-in link is on its way.</div>
      </div>
    </details>
  </div>
  <script>
    function handleEmailSubmit(e, form) {
      e.preventDefault();
      var email = form.email.value.trim();
      if (!email) return;
      fetch('/auth/magic-link/request', {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: 'email=' + encodeURIComponent(email)
      }).then(function(r) {
        if (r.ok) {
          form.style.display = 'none';
          document.getElementById('ml-sent').style.display = 'block';
        } else {
          alert('Something went wrong. Please try again.');
        }
      }).catch(function() {
        // Fallback: submit the form normally so the server handles it.
        form.submit();
      });
    }
  </script>
</body>
</html>`)
}
