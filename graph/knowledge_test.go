package graph

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lovyou-ai/site/auth"
)

// TestKnowledgePublic verifies unauthenticated GET /app/{slug}/knowledge returns 200
// for a public space.
func TestKnowledgePublic(t *testing.T) {
	_, store := testDB(t)
	slug := fmt.Sprintf("test-knowledge-pub-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Knowledge Public Test", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	h := NewHandlers(store, nil, nil)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/app/"+slug+"/knowledge", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

// TestKnowledgeAuthed verifies authenticated GET /app/{slug}/knowledge returns 200.
func TestKnowledgeAuthed(t *testing.T) {
	_, store := testDB(t)
	slug := fmt.Sprintf("test-knowledge-auth-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Knowledge Authed Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	testUser := &auth.User{ID: "test-user-1", Name: "Tester", Email: "test@test.com", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.ContextWithUser(r.Context(), testUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	h := NewHandlers(store, wrap, wrap)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/app/"+slug+"/knowledge", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

// TestKnowledgeMissingSpace verifies GET /app/{slug}/knowledge returns 404 when the
// space does not exist.
func TestKnowledgeMissingSpace(t *testing.T) {
	_, store := testDB(t)

	h := NewHandlers(store, nil, nil)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/app/nonexistent-space-xyz/knowledge", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
