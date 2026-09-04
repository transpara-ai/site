package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"golang.org/x/oauth2"
)

func TestAllowedEmailDomainsAreExactAndCaseInsensitive(t *testing.T) {
	a := &Auth{}
	if err := a.SetAllowedEmailDomains([]string{"Transpara.COM", "transpara.ai"}); err != nil {
		t.Fatal(err)
	}
	for _, email := range []string{"operator@transpara.com", "OPERATOR@TRANSPARA.AI"} {
		if !a.emailAllowed(email) {
			t.Fatalf("expected %q to be allowed", email)
		}
	}
	for _, email := range []string{"operator@example.com", "operator@sub.transpara.com", "operator@transpara.com@example.com"} {
		if a.emailAllowed(email) {
			t.Fatalf("expected %q to be rejected", email)
		}
	}
}

func TestAllowedEmailDomainsRejectInvalidConfiguration(t *testing.T) {
	for _, domain := range []string{"", "*.transpara.com", "@transpara.com", ".transpara.com", "transpara.com."} {
		a := &Auth{}
		if err := a.SetAllowedEmailDomains([]string{domain}); err == nil {
			t.Fatalf("expected %q to be rejected", domain)
		}
	}
}

func TestDisabledMagicLinksAreNotRegisteredOrAdvertised(t *testing.T) {
	a := &Auth{}
	a.SetMagicLinksEnabled(false)
	mux := http.NewServeMux()
	a.Register(mux)

	login := httptest.NewRecorder()
	mux.ServeHTTP(login, httptest.NewRequest(http.MethodGet, "/auth/login", nil))
	if strings.Contains(login.Body.String(), "magic-link") || !strings.Contains(login.Body.String(), "/auth/google") {
		t.Fatalf("login body = %q", login.Body.String())
	}

	request := httptest.NewRecorder()
	mux.ServeHTTP(request, httptest.NewRequest(http.MethodPost, "/auth/magic-link/request", nil))
	if request.Code != http.StatusNotFound {
		t.Fatalf("magic-link status = %d, want 404", request.Code)
	}
}

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

func TestProductionRedirectURLIgnoresRequestHost(t *testing.T) {
	a := &Auth{
		secure: true,
		oauth:  &oauth2.Config{RedirectURL: "https://civilization.internal.example/auth/callback"},
	}
	req := httptest.NewRequest(http.MethodGet, "https://attacker.example/auth/status", nil)
	req.Host = "attacker.example"
	if got := a.redirectURL(req); got != a.oauth.RedirectURL {
		t.Fatalf("redirectURL = %q, want %q", got, a.oauth.RedirectURL)
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

// ────────────────────────────────────────────────────────────────────
// OAuth happy path and exchange failure tests
// ────────────────────────────────────────────────────────────────────

// TestOAuthHappyPath exercises the full OAuth callback flow using mocked Google
// token exchange and userinfo endpoints. Requires a real database.
func TestOAuthHappyPath(t *testing.T) {
	a, db := testAuth(t)
	ctx := context.Background()

	testEmail := "oauth-happy@test.invalid"
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE email = $1)`, testEmail)
		db.ExecContext(ctx, `DELETE FROM users WHERE email = $1`, testEmail)
	})

	// Mock Google userinfo endpoint.
	userInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":             "google-oauth-test-123",
			"email":          testEmail,
			"verified_email": true,
			"name":           "OAuth Happy Tester",
			"picture":        "https://example.com/pic.jpg",
		})
	}))
	defer userInfoServer.Close()

	// Mock Google token exchange endpoint — returns a minimal valid token response.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	// Override oauth config and userinfo URL so no real Google traffic happens.
	a.oauth.Endpoint = oauth2.Endpoint{
		TokenURL:  tokenServer.URL + "/token",
		AuthURL:   tokenServer.URL + "/auth",
		AuthStyle: oauth2.AuthStyleInParams,
	}
	a.userInfoURL = userInfoServer.URL + "/userinfo"

	req := httptest.NewRequest("GET", "/auth/callback?state=teststate&code=testcode", nil)
	req.Host = "localhost"
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "teststate"})
	w := httptest.NewRecorder()
	a.handleCallback(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d (redirect to /app)", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != "/app" {
		t.Errorf("redirect location = %q, want /app", loc)
	}

	// Verify session cookie was set.
	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie in response")
	}

	// Verify session resolves to the correct user.
	user, err := a.userBySession(ctx, sessionCookie.Value)
	if err != nil {
		t.Fatalf("userBySession: %v", err)
	}
	if user.Email != testEmail {
		t.Errorf("user.Email = %q, want %q", user.Email, testEmail)
	}
	if user.Kind != "human" {
		t.Errorf("user.Kind = %q, want human", user.Kind)
	}
}

// TestOAuthCallbackTokenExchangeFailure verifies that when the token exchange
// fails (e.g. Workspace blocks OAuth), the handler redirects to exchange_failed.
func TestOAuthCallbackTokenExchangeFailure(t *testing.T) {
	// Mock token endpoint that rejects the code.
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
	}))
	defer tokenServer.Close()

	a := &Auth{
		secure: false,
		oauth: &oauth2.Config{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			Endpoint: oauth2.Endpoint{
				TokenURL:  tokenServer.URL + "/token",
				AuthURL:   tokenServer.URL + "/auth",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		userInfoURL: "http://should-not-be-called",
	}

	req := httptest.NewRequest("GET", "/auth/callback?state=mystate&code=testcode", nil)
	req.Host = "localhost"
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "mystate"})
	w := httptest.NewRecorder()
	a.handleCallback(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "exchange_failed") {
		t.Errorf("expected redirect to exchange_failed, got %q", loc)
	}
}

// TestOAuthCallbackMissingCode verifies that an empty code causes exchange_failed.
func TestOAuthCallbackMissingCode(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_request"})
	}))
	defer tokenServer.Close()

	a := &Auth{
		secure: false,
		oauth: &oauth2.Config{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			Endpoint: oauth2.Endpoint{
				TokenURL:  tokenServer.URL + "/token",
				AuthURL:   tokenServer.URL + "/auth",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		userInfoURL: "http://should-not-be-called",
	}

	// No code parameter in URL.
	req := httptest.NewRequest("GET", "/auth/callback?state=mystate", nil)
	req.Host = "localhost"
	req.AddCookie(&http.Cookie{Name: "oauth_state", Value: "mystate"})
	w := httptest.NewRecorder()
	a.handleCallback(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "/auth/error") {
		t.Errorf("expected redirect to /auth/error, got %q", loc)
	}
}

// ────────────────────────────────────────────────────────────────────
// Session lifecycle tests
// ────────────────────────────────────────────────────────────────────

// TestSessionExpired verifies that an expired session is rejected.
func TestSessionExpired(t *testing.T) {
	a, db := testAuth(t)
	ctx := context.Background()

	userID := "session-expire-test"
	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'human')`,
		userID, "google:expire-test", "expire@test.invalid", "Expire Tester"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	// Insert an already-expired session.
	sessionID := newID()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, NOW() - INTERVAL '1 second')`,
		sessionID, userID); err != nil {
		t.Fatalf("insert expired session: %v", err)
	}

	_, err := a.userBySession(ctx, sessionID)
	if err == nil {
		t.Error("expired session should be rejected")
	}
}

// TestLogout verifies that logout deletes the session and clears the cookie.
func TestLogout(t *testing.T) {
	a, db := testAuth(t)
	ctx := context.Background()

	userID := "logout-test"
	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'human')`,
		userID, "google:logout-test", "logout@test.invalid", "Logout Tester"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	sessionID := newID()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, NOW() + INTERVAL '30 days')`,
		sessionID, userID); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	// Session is valid before logout.
	if _, err := a.userBySession(ctx, sessionID); err != nil {
		t.Fatalf("session should be valid before logout: %v", err)
	}

	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	w := httptest.NewRecorder()
	a.handleLogout(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("logout status = %d, want %d", w.Code, http.StatusSeeOther)
	}

	// Session must be invalid after logout.
	if _, err := a.userBySession(ctx, sessionID); err == nil {
		t.Error("session should be invalid after logout")
	}

	// Response must clear the session cookie (MaxAge = -1).
	var cleared bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "session" && c.MaxAge == -1 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("expected session cookie to be cleared (MaxAge=-1) after logout")
	}
}

// ────────────────────────────────────────────────────────────────────
// API key revoke test
// ────────────────────────────────────────────────────────────────────

// TestAPIKeyRevoke verifies that a deleted API key can no longer authenticate.
func TestAPIKeyRevoke(t *testing.T) {
	a, db := testAuth(t)
	ctx := context.Background()

	userID := "revoke-test"
	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'human')`,
		userID, "google:revoke-test", "revoke@test.invalid", "Revoke Tester"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() {
		db.ExecContext(ctx, `DELETE FROM api_keys WHERE user_id = $1`, userID)
		db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	rawKey, err := a.createAPIKey(ctx, userID, "revoke-key", "")
	if err != nil {
		t.Fatalf("create key: %v", err)
	}

	// Key authenticates before revocation.
	if _, err := a.userByAPIKey(ctx, rawKey); err != nil {
		t.Fatalf("key should authenticate before revocation: %v", err)
	}

	// Find the key ID.
	keys, err := a.ListAPIKeys(ctx, userID)
	if err != nil || len(keys) == 0 {
		t.Fatalf("list keys: err=%v len=%d", err, len(keys))
	}
	keyID := keys[0].ID

	// Revoke via direct DB delete (same operation as handleDeleteAPIKey).
	if _, err := db.ExecContext(ctx,
		`DELETE FROM api_keys WHERE id = $1 AND user_id = $2`, keyID, userID); err != nil {
		t.Fatalf("revoke key: %v", err)
	}

	// Key must not authenticate after revocation.
	if _, err := a.userByAPIKey(ctx, rawKey); err == nil {
		t.Error("revoked key should no longer authenticate")
	}

	// List should be empty.
	remaining, err := a.ListAPIKeys(ctx, userID)
	if err != nil {
		t.Fatalf("list after revoke: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected 0 keys after revoke, got %d", len(remaining))
	}
}
