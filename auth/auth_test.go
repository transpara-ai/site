package auth

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"golang.org/x/oauth2"
	_ "github.com/lib/pq"
)

func testAuth(t *testing.T) (*Auth, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	a, err := New(db, "test-client-id", "test-client-secret", "http://localhost/auth/callback", false)
	if err != nil {
		t.Fatalf("new auth: %v", err)
	}
	return a, db
}

func TestAPIKeyAuth(t *testing.T) {
	a, db := testAuth(t)
	ctx := context.Background()

	// Create a test user.
	userID := "auth-test-user-1"
	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'human')`,
		userID, "google:auth-test", "authtest@test.com", "Auth Tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM api_keys WHERE user_id = $1`, userID)
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	t.Run("create_and_auth", func(t *testing.T) {
		rawKey, err := a.createAPIKey(ctx, userID, "test-key", "")
		if err != nil {
			t.Fatalf("create key: %v", err)
		}
		if rawKey[:3] != "lv_" {
			t.Errorf("key should start with lv_, got %s", rawKey[:3])
		}

		// Auth with the key.
		user, err := a.userByAPIKey(ctx, rawKey)
		if err != nil {
			t.Fatalf("auth: %v", err)
		}
		if user.ID != userID {
			t.Errorf("user ID = %q, want %q", user.ID, userID)
		}
		if user.Name != "Auth Tester" {
			t.Errorf("name = %q, want %q", user.Name, "Auth Tester")
		}
		if user.Kind != "human" {
			t.Errorf("kind = %q, want %q", user.Kind, "human")
		}
	})

	t.Run("invalid_key", func(t *testing.T) {
		_, err := a.userByAPIKey(ctx, "lv_invalid_key_that_doesnt_exist")
		if err == nil {
			t.Error("should fail with invalid key")
		}
	})

	t.Run("agent_key", func(t *testing.T) {
		// Clean up any prior test agent.
		db.ExecContext(ctx, `DELETE FROM users WHERE name = 'AuthTestBot'`)

		rawKey, err := a.createAPIKey(ctx, userID, "agent-key", "AuthTestBot")
		if err != nil {
			t.Fatalf("create agent key: %v", err)
		}

		t.Cleanup(func() {
			db.ExecContext(ctx, `DELETE FROM users WHERE name = 'AuthTestBot'`)
		})

		// Auth should resolve to the agent, not the human.
		user, err := a.userByAPIKey(ctx, rawKey)
		if err != nil {
			t.Fatalf("auth: %v", err)
		}
		if user.Kind != "agent" {
			t.Errorf("kind = %q, want %q", user.Kind, "agent")
		}
		if user.Name != "AuthTestBot" {
			t.Errorf("name = %q, want %q", user.Name, "AuthTestBot")
		}
		// The user ID should be the agent's, not the sponsor's.
		if user.ID == userID {
			t.Errorf("should resolve to agent user ID, not sponsor ID")
		}
	})

	t.Run("bearer_header", func(t *testing.T) {
		rawKey, err := a.createAPIKey(ctx, userID, "bearer-test", "")
		if err != nil {
			t.Fatalf("create key: %v", err)
		}

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+rawKey)
		user := a.userFromBearer(req)
		if user == nil {
			t.Fatal("should resolve user from bearer token")
		}
		if user.ID != userID {
			t.Errorf("user ID = %q, want %q", user.ID, userID)
		}
	})

	t.Run("no_bearer", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		user := a.userFromBearer(req)
		if user != nil {
			t.Error("should return nil without bearer header")
		}
	})

	t.Run("list_keys", func(t *testing.T) {
		keys, err := a.ListAPIKeys(ctx, userID)
		if err != nil {
			t.Fatalf("list keys: %v", err)
		}
		if len(keys) < 1 {
			t.Errorf("should have at least 1 key, got %d", len(keys))
		}
	})
}

func TestRequireAuth(t *testing.T) {
	a, db := testAuth(t)
	ctx := context.Background()

	// Create user + key.
	userID := "auth-middleware-test"
	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'human')`,
		userID, "google:mw-test", "mwtest@test.com", "MW Tester")
	rawKey, _ := a.createAPIKey(ctx, userID, "mw-key", "")
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM api_keys WHERE user_id = $1`, userID)
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	handler := a.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		user := UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(user.Name))
	})

	t.Run("with_api_key", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+rawKey)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		if w.Body.String() != "MW Tester" {
			t.Errorf("body = %q, want %q", w.Body.String(), "MW Tester")
		}
	})

	t.Run("without_auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Should redirect to login.
		if w.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want %d (redirect to login)", w.Code, http.StatusSeeOther)
		}
	})
}

// ────────────────────────────────────────────────────────────────────
// Callback handler tests (no DB required)
// ────────────────────────────────────────────────────────────────────

func newTestAuth() *Auth {
	return &Auth{
		secure: false,
		oauth:  &oauth2.Config{ClientID: "test-client", ClientSecret: "test-secret"},
	}
}

// TestCallbackInvalidState verifies that a state mismatch redirects to /auth/error.
func TestCallbackInvalidState(t *testing.T) {
	a := newTestAuth()
	req := httptest.NewRequest("GET", "/auth/callback?state=wrong&code=test", nil)
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "correct"})
	w := httptest.NewRecorder()
	a.handleCallback(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "invalid_state") {
		t.Errorf("expected redirect to /auth/error?code=invalid_state, got %q", loc)
	}
}

// TestCallbackExpiredState verifies that a missing oauth_state cookie (expired)
// redirects to /auth/error.
func TestCallbackExpiredState(t *testing.T) {
	a := newTestAuth()
	req := httptest.NewRequest("GET", "/auth/callback?state=some_state&code=test", nil)
	// No oauth_state cookie — simulates an expired or missing state.
	w := httptest.NewRecorder()
	a.handleCallback(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "invalid_state") {
		t.Errorf("expected redirect to /auth/error?code=invalid_state, got %q", loc)
	}
}

// TestCallbackGoogleError verifies that a Google error param redirects to /auth/error
// before any state or token exchange.
func TestCallbackGoogleError(t *testing.T) {
	a := newTestAuth()
	req := httptest.NewRequest("GET", "/auth/callback?error=access_denied", nil)
	w := httptest.NewRecorder()
	a.handleCallback(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "access_denied") {
		t.Errorf("expected redirect to include access_denied error code, got %q", loc)
	}
	if !strings.HasPrefix(loc, "/auth/error") {
		t.Errorf("expected redirect to /auth/error, got %q", loc)
	}
}

// TestAuthErrorPage verifies that the error page renders with the right message.
func TestAuthErrorPage(t *testing.T) {
	a := newTestAuth()

	t.Run("access_denied", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/error?code=access_denied", nil)
		w := httptest.NewRecorder()
		a.handleAuthError(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Sign-in was cancelled") {
			t.Errorf("expected 'Sign-in was cancelled' in body, got: %s", body)
		}
		if !strings.Contains(body, "access_denied") {
			t.Errorf("expected error code in body, got: %s", body)
		}
	})

	t.Run("invalid_state", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/error?code=invalid_state", nil)
		w := httptest.NewRecorder()
		a.handleAuthError(w, req)

		body := w.Body.String()
		if !strings.Contains(body, "expired") {
			t.Errorf("expected 'expired' in body for invalid_state, got: %s", body)
		}
	})

	t.Run("no_code", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/error", nil)
		w := httptest.NewRecorder()
		a.handleAuthError(w, req)

		body := w.Body.String()
		if !strings.Contains(body, "Authentication failed") {
			t.Errorf("expected default message in body, got: %s", body)
		}
	})

	t.Run("try_again_link", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/auth/error?code=exchange_failed", nil)
		w := httptest.NewRecorder()
		a.handleAuthError(w, req)

		body := w.Body.String()
		if !strings.Contains(body, `href="/auth/login"`) {
			t.Errorf("expected retry link in error page, got: %s", body)
		}
	})
}

// TestAuthStatus verifies the /auth/status endpoint returns safe config state.
func TestAuthStatus(t *testing.T) {
	a := newTestAuth()
	req := httptest.NewRequest("GET", "/auth/status", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()
	a.handleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"oauth_configured":true`) {
		t.Errorf("expected oauth_configured=true, got: %s", body)
	}
	if !strings.Contains(body, "localhost:8080") {
		t.Errorf("expected redirect_url with host in body, got: %s", body)
	}
	// Must not expose secrets.
	if strings.Contains(body, "test-secret") {
		t.Errorf("status endpoint must not expose client_secret, got: %s", body)
	}
}

// ────────────────────────────────────────────────────────────────────
// Magic link tests
// ────────────────────────────────────────────────────────────────────

// TestMagicLinkRequestInvalidEmail verifies that invalid emails are rejected
// before touching the database.
func TestMagicLinkRequestInvalidEmail(t *testing.T) {
	a := newTestAuth()

	cases := []struct{ name, email string }{
		{"empty", ""},
		{"no_at", "notanemail"},
		{"at_only", "@"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := "email=" + tc.email
			req := httptest.NewRequest("POST", "/auth/magic-link/request", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()
			a.handleMagicLinkRequest(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("email=%q: status = %d, want %d", tc.email, w.Code, http.StatusBadRequest)
			}
		})
	}
}

// TestMagicLinkVerifyMissingToken verifies that a missing token redirects to the error page.
func TestMagicLinkVerifyMissingToken(t *testing.T) {
	a := newTestAuth()
	req := httptest.NewRequest("GET", "/auth/magic-link/verify", nil)
	w := httptest.NewRecorder()
	a.handleMagicLinkVerify(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "/auth/error") {
		t.Errorf("expected redirect to /auth/error, got %q", loc)
	}
}

// TestMagicLinkHappyPath tests the full request+verify flow with a real database.
func TestMagicLinkHappyPath(t *testing.T) {
	a, db := testAuth(t)
	ctx := context.Background()

	email := "magic-happy@test.invalid"
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM magic_link_tokens WHERE email = $1`, email)
		db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, email)
		db.ExecContext(ctx, `DELETE FROM users WHERE email = $1`, email)
	})

	rawToken, err := a.requestMagicLink(ctx, email)
	if err != nil {
		t.Fatalf("requestMagicLink: %v", err)
	}
	if rawToken == "" {
		t.Fatal("expected non-empty token")
	}

	user, err := a.verifyMagicLink(ctx, rawToken)
	if err != nil {
		t.Fatalf("verifyMagicLink: %v", err)
	}
	if user.Email != email {
		t.Errorf("user.Email = %q, want %q", user.Email, email)
	}
	if user.Kind != "human" {
		t.Errorf("user.Kind = %q, want 'human'", user.Kind)
	}

	// Second verify must fail — token already used.
	_, err = a.verifyMagicLink(ctx, rawToken)
	if err == nil {
		t.Error("second verify should fail (token already used)")
	}
}

// TestMagicLinkExpiredToken verifies that expired tokens are rejected.
func TestMagicLinkExpiredToken(t *testing.T) {
	a, db := testAuth(t)
	ctx := context.Background()

	email := "magic-expired@test.invalid"
	rawToken := newID()
	hash := hashKey(rawToken)
	id := newID()
	_, err := db.ExecContext(ctx,
		`INSERT INTO magic_link_tokens (id, token_hash, email, expires_at)
		 VALUES ($1, $2, $3, NOW() - INTERVAL '1 minute')`,
		id, hash, email)
	if err != nil {
		t.Fatalf("insert expired token: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM magic_link_tokens WHERE id = $1`, id)
	})

	_, err = a.verifyMagicLink(ctx, rawToken)
	if err == nil {
		t.Error("expired token should be rejected")
	}
}

// TestMagicLinkInvalidToken verifies that bogus tokens are rejected.
func TestMagicLinkInvalidToken(t *testing.T) {
	a, _ := testAuth(t)
	ctx := context.Background()

	_, err := a.verifyMagicLink(ctx, "totally-bogus-token-that-does-not-exist-in-db")
	if err == nil {
		t.Error("invalid token should be rejected")
	}
}

// TestMagicLinkIdempotentUser verifies that two magic link logins with the same
// email resolve to the same user — not two separate accounts.
func TestMagicLinkIdempotentUser(t *testing.T) {
	a, db := testAuth(t)
	ctx := context.Background()

	email := "magic-idem@test.invalid"
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM magic_link_tokens WHERE email = $1`, email)
		db.ExecContext(ctx, `DELETE FROM users WHERE email = $1`, email)
	})

	tok1, err := a.requestMagicLink(ctx, email)
	if err != nil {
		t.Fatalf("requestMagicLink 1: %v", err)
	}
	tok2, err := a.requestMagicLink(ctx, email)
	if err != nil {
		t.Fatalf("requestMagicLink 2: %v", err)
	}

	u1, err := a.verifyMagicLink(ctx, tok1)
	if err != nil {
		t.Fatalf("verifyMagicLink 1: %v", err)
	}
	u2, err := a.verifyMagicLink(ctx, tok2)
	if err != nil {
		t.Fatalf("verifyMagicLink 2: %v", err)
	}

	if u1.ID != u2.ID {
		t.Errorf("same email should resolve to same user: u1=%s u2=%s", u1.ID, u2.ID)
	}
}

// TestConcurrentSessions verifies that two active sessions for the same user
// both resolve correctly. Requires a real database.
func TestConcurrentSessions(t *testing.T) {
	a, db := testAuth(t)
	ctx := context.Background()

	userID := "concurrent-session-test"
	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'human')`,
		userID, "google:concurrent-test", "concurrent@test.com", "Concurrent Tester")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	// Create two sessions for the same user.
	session1 := newID()
	session2 := newID()
	exp := "NOW() + INTERVAL '30 days'"
	db.ExecContext(ctx, `INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, `+exp+`)`, session1, userID)
	db.ExecContext(ctx, `INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, `+exp+`)`, session2, userID)

	// Both sessions should resolve to the same user.
	u1, err := a.userBySession(ctx, session1)
	if err != nil {
		t.Fatalf("session1 lookup: %v", err)
	}
	u2, err := a.userBySession(ctx, session2)
	if err != nil {
		t.Fatalf("session2 lookup: %v", err)
	}

	if u1.ID != userID || u2.ID != userID {
		t.Errorf("concurrent sessions resolved to wrong user: session1=%s session2=%s want=%s",
			u1.ID, u2.ID, userID)
	}
	if u1.Email != u2.Email {
		t.Errorf("sessions resolved to different users: u1.Email=%s u2.Email=%s", u1.Email, u2.Email)
	}
}
