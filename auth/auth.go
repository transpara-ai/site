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
	"log"
	"net/http"
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
	db     *sql.DB
	oauth  *oauth2.Config
	secure bool
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
		secure: secure,
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
    google_id TEXT UNIQUE NOT NULL,
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
`)
	if err != nil {
		return err
	}

	// Migrations for existing databases.
	migrations := []string{
		`ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS agent_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS agent_id TEXT REFERENCES users(id) ON DELETE SET NULL`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'human'`,
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
	mux.HandleFunc("GET /auth/callback", a.handleCallback)
	mux.HandleFunc("POST /auth/logout", a.handleLogout)

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

func (a *Auth) handleLogin(w http.ResponseWriter, r *http.Request) {
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
	log.Printf("auth: login redirect to %s", authURL)
	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

func (a *Auth) handleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify CSRF state.
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Path: "/auth", MaxAge: -1})

	// Exchange code for token — use same redirect_uri as the login request.
	redirectURL := a.redirectURL(r)
	token, err := a.oauth.Exchange(r.Context(), r.URL.Query().Get("code"),
		oauth2.SetAuthURLParam("redirect_uri", redirectURL))
	if err != nil {
		log.Printf("auth: oauth exchange: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	// Fetch Google user info.
	client := a.oauth.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("auth: fetch userinfo: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
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
		log.Printf("auth: decode userinfo: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	// Upsert user.
	user, err := a.upsertUser(r.Context(), info.ID, info.Email, info.Name, info.Picture)
	if err != nil {
		log.Printf("auth: upsert user: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

	// Create session (30 days).
	sessionID := newID()
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if _, err := a.db.ExecContext(r.Context(),
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		sessionID, user.ID, expiresAt,
	); err != nil {
		log.Printf("auth: create session: %v", err)
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}

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
