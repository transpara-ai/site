package graph

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/lovyou-ai/site/auth"
)

// testHandlers creates handlers with a test user in context.
func testHandlers(t *testing.T) (*Handlers, *Store, func(http.HandlerFunc) http.Handler) {
	t.Helper()
	_, store := testDB(t)

	// Auth wrapper that injects a test user.
	testUser := &auth.User{ID: "test-user-1", Name: "Tester", Email: "test@test.com", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.ContextWithUser(r.Context(), testUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	h := NewHandlers(store, wrap, wrap)
	return h, store, wrap
}

func TestHandlerCreateSpace(t *testing.T) {
	h, store, _ := testHandlers(t)
	ctx := t

	mux := http.NewServeMux()
	h.Register(mux)

	// Create space via form POST.
	form := url.Values{}
	form.Set("slug", "handler-test")
	form.Set("name", "Handler Test")
	form.Set("description", "Testing handlers")
	form.Set("kind", "project")
	form.Set("visibility", "public")

	req := httptest.NewRequest("POST", "/app/new", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusSeeOther, w.Body.String())
	}

	// Verify space was created.
	space, err := store.GetSpaceBySlug(t.Context(), "handler-test")
	if err != nil {
		t.Fatalf("get space: %v", err)
	}
	_ = ctx
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	if space.Name != "Handler Test" {
		t.Errorf("name = %q, want %q", space.Name, "Handler Test")
	}
	if space.Visibility != "public" {
		t.Errorf("visibility = %q, want %q", space.Visibility, "public")
	}
}

func TestHandlerOp(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	// Create a space first (clean up stale data from prior runs).
	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-op-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-op-test", "Op Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("intend_json", func(t *testing.T) {
		body := `{"op":"intend","title":"Test Task","description":"A task","priority":"high"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		if result["op"] != "intend" {
			t.Errorf("op = %v, want intend", result["op"])
		}
		node := result["node"].(map[string]any)
		if node["title"] != "Test Task" {
			t.Errorf("title = %v, want Test Task", node["title"])
		}
		if node["priority"] != "high" {
			t.Errorf("priority = %v, want high", node["priority"])
		}
	})

	t.Run("express_json", func(t *testing.T) {
		body := `{"op":"express","title":"Test Post","body":"Hello world"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
	})

	t.Run("converse_json", func(t *testing.T) {
		body := `{"op":"converse","title":"Test Chat","body":"Let's discuss","participants":"alice,bob"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["title"] != "Test Chat" {
			t.Errorf("title = %v, want Test Chat", node["title"])
		}
	})

	t.Run("respond_json", func(t *testing.T) {
		// Create a parent first.
		parent, _ := store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindThread, Title: "Parent", Author: "Tester", AuthorID: "test-user-1",
		})

		body := `{"op":"respond","parent_id":"` + parent.ID + `","body":"A reply"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
	})

	t.Run("unknown_op", func(t *testing.T) {
		body := `{"op":"nonexistent"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestHandlerConversationDetail(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-convo-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-convo-test", "Convo Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	convo, err := store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindConversation, Title: "Test Convo",
		Author: "Tester", AuthorID: "test-user-1", Tags: []string{"test-user-1"},
	})
	if err != nil {
		t.Fatalf("create convo: %v", err)
	}

	// Add a message.
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, ParentID: convo.ID, Kind: KindComment,
		Body: "Hello!", Author: "Tester", AuthorID: "test-user-1",
	})

	// GET conversation detail as JSON.
	req := httptest.NewRequest("GET", "/app/handler-convo-test/conversation/"+convo.ID, nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)
	messages := result["messages"].([]any)
	if len(messages) != 1 {
		t.Errorf("got %d messages, want 1", len(messages))
	}
}
