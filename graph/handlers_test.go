package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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

	mux := http.NewServeMux()
	h.Register(mux)

	// handleCreateSpace derives the slug from slugify(name) and ignores any
	// explicit slug form field, so make the name unique to keep the slug unique.
	name := fmt.Sprintf("Handler Test %d", time.Now().UnixNano())
	expectedSlug := slugify(name)

	form := url.Values{}
	form.Set("name", name)
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

	space, err := store.GetSpaceBySlug(t.Context(), expectedSlug)
	if err != nil {
		t.Fatalf("get space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	if space.Name != name {
		t.Errorf("name = %q, want %q", space.Name, name)
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

	t.Run("intend_body_field", func(t *testing.T) {
		// body key must be read (not silently dropped as with description-only fallback)
		payload := `{"op":"intend","title":"Body Field Task","body":"from body key"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(payload))
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
		if node["body"] != "from body key" {
			t.Errorf("body = %v, want 'from body key'", node["body"])
		}
	})

	t.Run("intend_description_fallback", func(t *testing.T) {
		// when body key is absent, description key must be used for the node body
		payload := `{"op":"intend","title":"Fallback Task","description":"from description key"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(payload))
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
		if node["body"] != "from description key" {
			t.Errorf("body = %v, want 'from description key'", node["body"])
		}
	})

	t.Run("intend_body_beats_description", func(t *testing.T) {
		// when both body and description are present, body wins
		payload := `{"op":"intend","title":"Priority Task","body":"from body","description":"from description"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(payload))
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
		if node["body"] != "from body" {
			t.Errorf("body = %v, want 'from body'", node["body"])
		}
	})

	t.Run("intend_kind_proposal", func(t *testing.T) {
		// kind=proposal must not be silently dropped to kind=task
		payload := `{"op":"intend","title":"Test Proposal","kind":"proposal"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(payload))
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
		if node["kind"] != KindProposal {
			t.Errorf("kind = %v, want %v", node["kind"], KindProposal)
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

	t.Run("report_json", func(t *testing.T) {
		parent, _ := store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindPost, Title: "Flaggable", Author: "Tester", AuthorID: "test-user-1",
		})
		body := `{"op":"report","node_id":"` + parent.ID + `","reason":"inappropriate content"}`
		req := httptest.NewRequest("POST", "/app/handler-op-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
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

func TestParseMessageSearch(t *testing.T) {
	tests := []struct {
		input      string
		wantBody   string
		wantFrom   string
	}{
		{"hello world", "hello world", ""},
		{"from:alice", "", "alice"},
		{"hello from:alice", "hello", "alice"},
		{"from:bob goodbye", "goodbye", "bob"},
		{"hello from:alice world", "hello world", "alice"},
		{"", "", ""},
	}

	for _, tt := range tests {
		body, from := parseMessageSearch(tt.input)
		if body != tt.wantBody {
			t.Errorf("parseMessageSearch(%q) body = %q, want %q", tt.input, body, tt.wantBody)
		}
		if from != tt.wantFrom {
			t.Errorf("parseMessageSearch(%q) from = %q, want %q", tt.input, from, tt.wantFrom)
		}
	}
}

func TestHandlerDocumentEdit(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-doc-edit-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	// Private space owned by test-user-1 (member).
	space, err := store.CreateSpace(t.Context(), "handler-doc-edit-test", "Doc Edit Test", "", "test-user-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	doc, err := store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument, Title: "Original Title",
		Body: "Original body.", Author: "Tester", AuthorID: "test-user-1",
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	t.Run("get_edit_form_member", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/app/handler-doc-edit-test/document/"+doc.ID+"/edit", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}
	})

	t.Run("post_edit_member", func(t *testing.T) {
		form := url.Values{}
		form.Set("title", "Updated Title")
		form.Set("body", "Updated body content.")

		req := httptest.NewRequest("POST", "/app/handler-doc-edit-test/document/"+doc.ID+"/edit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["title"] != "Updated Title" {
			t.Errorf("title = %v, want Updated Title", node["title"])
		}
		if node["body"] != "Updated body content." {
			t.Errorf("body = %v, want Updated body content.", node["body"])
		}
	})

	t.Run("non_member_rejected", func(t *testing.T) {
		// Create a second handler set with a different user who is not the space owner.
		otherUser := &auth.User{ID: "other-user-99", Name: "Other", Email: "other@test.com", Kind: "human"}
		otherWrap := func(next http.HandlerFunc) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := auth.ContextWithUser(r.Context(), otherUser)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		}
		otherH := NewHandlers(store, otherWrap, otherWrap)
		otherMux := http.NewServeMux()
		otherH.Register(otherMux)

		form := url.Values{}
		form.Set("title", "Should Not Update")
		form.Set("body", "Should not be saved.")

		req := httptest.NewRequest("POST", "/app/handler-doc-edit-test/document/"+doc.ID+"/edit", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		otherMux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (non-member should be rejected)", w.Code, http.StatusNotFound)
		}
	})
}

func TestHandlerDocuments(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-docs-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-docs-test", "Docs Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("create_document", func(t *testing.T) {
		body := `{"op":"intend","title":"Architecture Guide","description":"Our system design","kind":"document"}`
		req := httptest.NewRequest("POST", "/app/handler-docs-test/op", strings.NewReader(body))
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
		if node["kind"] != KindDocument {
			t.Errorf("kind = %v, want %q", node["kind"], KindDocument)
		}
		if node["title"] != "Architecture Guide" {
			t.Errorf("title = %v, want Architecture Guide", node["title"])
		}
	})

	t.Run("list_documents", func(t *testing.T) {
		store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindDocument, Title: "List Doc",
			Author: "Tester", AuthorID: "test-user-1",
		})

		req := httptest.NewRequest("GET", "/app/handler-docs-test/documents", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		docs := result["documents"].([]any)
		if len(docs) == 0 {
			t.Error("got 0 documents, want at least 1")
		}
	})

	t.Run("document_detail", func(t *testing.T) {
		doc, err := store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindDocument, Title: "Detail Doc",
			Body: "# Hello\nThis is content.", Author: "Tester", AuthorID: "test-user-1",
		})
		if err != nil {
			t.Fatalf("create document: %v", err)
		}

		req := httptest.NewRequest("GET", "/app/handler-docs-test/node/"+doc.ID, nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["node"].(map[string]any)
		if node["id"] != doc.ID {
			t.Errorf("id = %v, want %v", node["id"], doc.ID)
		}
		if node["kind"] != KindDocument {
			t.Errorf("kind = %v, want %q", node["kind"], KindDocument)
		}
	})

	t.Run("search_documents", func(t *testing.T) {
		store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindDocument, Title: "Searchable Wiki Page",
			Body: "Contains the word searchable", Author: "Tester", AuthorID: "test-user-1",
		})
		store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindDocument, Title: "Other Doc",
			Author: "Tester", AuthorID: "test-user-1",
		})

		req := httptest.NewRequest("GET", "/app/handler-docs-test/documents?q=Searchable+Wiki", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		docs := result["documents"].([]any)
		if len(docs) == 0 {
			t.Error("got 0 results for search, want at least 1")
		}
		first := docs[0].(map[string]any)
		if first["title"] != "Searchable Wiki Page" {
			t.Errorf("first result title = %v, want Searchable Wiki Page", first["title"])
		}
	})
}

func TestHandlerQuestions(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-qa-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-qa-test", "Q&A Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("create_question", func(t *testing.T) {
		body := `{"op":"intend","title":"How does the event graph work?","description":"Looking for an overview of the architecture","kind":"question"}`
		req := httptest.NewRequest("POST", "/app/handler-qa-test/op", strings.NewReader(body))
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
		if node["kind"] != KindQuestion {
			t.Errorf("kind = %v, want %q", node["kind"], KindQuestion)
		}
		if node["title"] != "How does the event graph work?" {
			t.Errorf("title = %v, want How does the event graph work?", node["title"])
		}
	})

	t.Run("list_questions", func(t *testing.T) {
		store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindQuestion, Title: "What is causality?",
			Author: "Tester", AuthorID: "test-user-1",
		})

		req := httptest.NewRequest("GET", "/app/handler-qa-test/questions", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		questions := result["questions"].([]any)
		if len(questions) == 0 {
			t.Error("got 0 questions, want at least 1")
		}
	})

	t.Run("question_detail", func(t *testing.T) {
		q, err := store.CreateNode(t.Context(), CreateNodeParams{
			SpaceID: space.ID, Kind: KindQuestion, Title: "Why use signed events?",
			Body: "Curious about the integrity model.", Author: "Tester", AuthorID: "test-user-1",
		})
		if err != nil {
			t.Fatalf("create question: %v", err)
		}

		req := httptest.NewRequest("GET", "/app/handler-qa-test/questions/"+q.ID, nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		node := result["question"].(map[string]any)
		if node["id"] != q.ID {
			t.Errorf("id = %v, want %v", node["id"], q.ID)
		}
		if node["kind"] != KindQuestion {
			t.Errorf("kind = %v, want %q", node["kind"], KindQuestion)
		}
	})
}

// TestHandlerExpressQuestion verifies that express op with kind=question creates a KindQuestion.
func TestHandlerExpressQuestion(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-express-qa-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-express-qa-test", "Express Q&A Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("express_kind_question_creates_question", func(t *testing.T) {
		body := `{"op":"express","title":"What is an event graph?","body":"Looking for a brief overview","kind":"question"}`
		req := httptest.NewRequest("POST", "/app/handler-express-qa-test/op", strings.NewReader(body))
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
		if node["kind"] != KindQuestion {
			t.Errorf("kind = %v, want %q", node["kind"], KindQuestion)
		}
		if node["title"] != "What is an event graph?" {
			t.Errorf("title = %v, want 'What is an event graph?'", node["title"])
		}
		if result["op"] != "express" {
			t.Errorf("op = %v, want 'express'", result["op"])
		}
	})

	t.Run("express_no_kind_creates_post", func(t *testing.T) {
		body := `{"op":"express","body":"A post without a kind"}`
		req := httptest.NewRequest("POST", "/app/handler-express-qa-test/op", strings.NewReader(body))
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
		if node["kind"] != KindPost {
			t.Errorf("kind = %v, want %q", node["kind"], KindPost)
		}
	})
}

// TestHandlerKnowledgeLens verifies the knowledge lens returns documents and questions
// alongside claims, with LIMIT bounds applied (Invariant 13: BOUNDED).
func TestHandlerKnowledgeLens(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-knowledge-lens-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-knowledge-lens-test", "Knowledge Lens Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	// Seed a document, a question, and a claim.
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument, Title: "Team Playbook",
		Body: "How we work.", Author: "Tester", AuthorID: "test-user-1", AuthorKind: "human",
	})
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindQuestion, Title: "What is the mission?",
		Author: "Tester", AuthorID: "test-user-1", AuthorKind: "human",
	})
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindClaim, Title: "Event graphs are scalable",
		Author: "Tester", AuthorID: "test-user-1", AuthorKind: "human",
	})

	req := httptest.NewRequest("GET", "/app/handler-knowledge-lens-test/knowledge", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)

	docs, ok := result["documents"].([]any)
	if !ok {
		t.Fatal("knowledge lens response missing 'documents' field")
	}
	if len(docs) == 0 {
		t.Error("knowledge lens: got 0 documents, want at least 1")
	}

	questions, ok := result["questions"].([]any)
	if !ok {
		t.Fatal("knowledge lens response missing 'questions' field")
	}
	if len(questions) == 0 {
		t.Error("knowledge lens: got 0 questions, want at least 1")
	}

	// Invariant 13 (BOUNDED): both lists must be within the declared LIMIT of 50.
	const knowledgeLensLimit = 50
	if len(docs) > knowledgeLensLimit {
		t.Errorf("knowledge lens: documents count %d exceeds BOUNDED limit %d", len(docs), knowledgeLensLimit)
	}
	if len(questions) > knowledgeLensLimit {
		t.Errorf("knowledge lens: questions count %d exceeds BOUNDED limit %d", len(questions), knowledgeLensLimit)
	}
}

// TestHandlerKnowledgeTabs verifies that the /knowledge route handles ?tab=docs
// and ?tab=qa routing correctly — returns 200 and no server errors.
func TestHandlerKnowledgeTabs(t *testing.T) {
	h, store, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "knowledge-tabs-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "knowledge-tabs-test", "Knowledge Tabs Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	// Seed a document and a question.
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument, Title: "Test Doc",
		Body: "Content.", Author: "Tester", AuthorID: "test-user-1",
	})
	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindQuestion, Title: "Test Question",
		Author: "Tester", AuthorID: "test-user-1",
	})

	for _, tab := range []string{"docs", "qa", "claims", ""} {
		tab := tab
		t.Run("tab_"+tab, func(t *testing.T) {
			path := "/app/knowledge-tabs-test/knowledge"
			if tab != "" {
				path += "?tab=" + tab
			}
			req := httptest.NewRequest("GET", path, nil)
			// HTML request — no application/json Accept header.
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("tab=%q: status = %d, want %d", tab, w.Code, http.StatusOK)
			}
		})
	}
}

// TestHandlerKnowledgeDocsTabRows confirms GET ?tab=docs returns 200 and
// renders the seeded document title in the HTML body.
func TestHandlerKnowledgeDocsTabRows(t *testing.T) {
	h, store, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "knowledge-docs-rows-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "knowledge-docs-rows-test", "Knowledge Docs Rows", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindDocument, Title: "Unique Doc Row Title",
		Body: "Excerpt content for the document row.", Author: "Tester", AuthorID: "test-user-1", AuthorKind: "human",
	})

	req := httptest.NewRequest("GET", "/app/knowledge-docs-rows-test/knowledge?tab=docs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("docs tab: status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Unique Doc Row Title") {
		t.Error("docs tab: seeded document title not found in HTML response")
	}
}

// TestHandlerKnowledgeQATabRows confirms GET ?tab=qa returns 200 and renders
// the seeded question title with an Answered or Awaiting badge in the HTML body.
func TestHandlerKnowledgeQATabRows(t *testing.T) {
	h, store, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "knowledge-qa-rows-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "knowledge-qa-rows-test", "Knowledge QA Rows", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindQuestion, Title: "Unique QA Row Question",
		Author: "Tester", AuthorID: "test-user-1", AuthorKind: "human",
	})

	req := httptest.NewRequest("GET", "/app/knowledge-qa-rows-test/knowledge?tab=qa", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("qa tab: status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Unique QA Row Question") {
		t.Error("qa tab: seeded question title not found in HTML response")
	}
	if !strings.Contains(body, "Answered") && !strings.Contains(body, "Awaiting") {
		t.Error("qa tab: neither Answered nor Awaiting badge found in HTML response")
	}
}

func TestHandlerJoinViaInvite(t *testing.T) {
	_, store := testDB(t)

	testUser := &auth.User{ID: "joiner-1", Name: "Joiner", Email: "joiner@test.com", Kind: "human"}
	authWrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.ContextWithUser(r.Context(), testUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	// Passes request through without a user — handler sees "anonymous" and redirects with ?next=.
	anonWrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	slug := fmt.Sprintf("join-test-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Join Test", "", "owner-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	token, err := store.CreateInviteCode(t.Context(), space.ID, "owner-1", nil, 0)
	if err != nil {
		t.Fatalf("create invite code: %v", err)
	}

	t.Run("unauthenticated_redirect", func(t *testing.T) {
		h := NewHandlers(store, anonWrap, anonWrap)
		mux := http.NewServeMux()
		h.Register(mux)

		req := httptest.NewRequest("GET", "/join/"+token, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
		}
		want := "/auth/login?next=%2Fjoin%2F" + token
		if loc := w.Header().Get("Location"); loc != want {
			t.Errorf("Location = %q, want %q", loc, want)
		}
	})

	t.Run("valid_code_joins_user", func(t *testing.T) {
		h := NewHandlers(store, authWrap, authWrap)
		mux := http.NewServeMux()
		h.Register(mux)

		req := httptest.NewRequest("GET", "/join/"+token, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
		}
		want := "/app/" + space.Slug + "/board"
		if loc := w.Header().Get("Location"); loc != want {
			t.Errorf("Location = %q, want %q", loc, want)
		}
		if !store.IsMember(t.Context(), space.ID, testUser.ID) {
			t.Error("user should be a member after joining via invite")
		}
	})

	t.Run("invalid_code_404", func(t *testing.T) {
		h := NewHandlers(store, authWrap, authWrap)
		mux := http.NewServeMux()
		h.Register(mux)

		req := httptest.NewRequest("GET", "/join/nonexistent-token", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})
}

func TestHandlerCreateInviteHTMX(t *testing.T) {
	h, store, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	slug := fmt.Sprintf("htmx-invite-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "HTMX Invite Test", "", "test-user-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("owner_creates_invite_returns_html", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/app/"+slug+"/invites", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		body := w.Body.String()
		if body == "" {
			t.Error("expected HTML fragment in response body, got empty")
		}

		// Verify an invite was actually created in the store.
		invites, err := store.ListInvites(t.Context(), space.ID)
		if err != nil {
			t.Fatalf("list invites: %v", err)
		}
		if len(invites) == 0 {
			t.Error("expected invite to be created in store")
		}
	})

	t.Run("nonexistent_space_404", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/app/no-such-space/invites", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("non_owner_rejected", func(t *testing.T) {
		otherUser := &auth.User{ID: "other-user-99", Name: "Other", Email: "other@test.com", Kind: "human"}
		otherWrap := func(next http.HandlerFunc) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := auth.ContextWithUser(r.Context(), otherUser)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		}
		h2 := NewHandlers(store, otherWrap, otherWrap)
		mux2 := http.NewServeMux()
		h2.Register(mux2)

		req := httptest.NewRequest("POST", "/app/"+slug+"/invites", nil)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (non-owner should be rejected)", w.Code, http.StatusNotFound)
		}
	})

	t.Run("unauthenticated_rejected", func(t *testing.T) {
		anonWrap := func(next http.HandlerFunc) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
		h2 := NewHandlers(store, anonWrap, anonWrap)
		mux2 := http.NewServeMux()
		h2.Register(mux2)

		req := httptest.NewRequest("POST", "/app/"+slug+"/invites", nil)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (unauthenticated should be rejected)", w.Code, http.StatusNotFound)
		}
	})
}

func TestHandlerRevokeInvite(t *testing.T) {
	h, store, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	slug := fmt.Sprintf("revoke-invite-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(t.Context(), slug, "Revoke Invite Test", "", "test-user-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("revoke_existing_invite", func(t *testing.T) {
		tok, err := store.CreateInviteCode(t.Context(), space.ID, "test-user-1", nil, 0)
		if err != nil {
			t.Fatalf("create invite: %v", err)
		}

		req := httptest.NewRequest("DELETE", "/app/"+slug+"/invites/"+tok, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		// Verify it is gone from the store.
		got, err := store.GetInviteCode(t.Context(), tok)
		if err != nil {
			t.Fatalf("get invite after revoke: %v", err)
		}
		if got != nil {
			t.Error("expected invite to be deleted, still present")
		}
	})

	t.Run("revoke_nonexistent_token_404", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/app/"+slug+"/invites/no-such-token", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("non_owner_cannot_revoke", func(t *testing.T) {
		tok, err := store.CreateInviteCode(t.Context(), space.ID, "test-user-1", nil, 0)
		if err != nil {
			t.Fatalf("create invite: %v", err)
		}
		t.Cleanup(func() { store.RevokeInvite(t.Context(), tok) })

		otherUser := &auth.User{ID: "other-user-99", Name: "Other", Email: "other@test.com", Kind: "human"}
		otherWrap := func(next http.HandlerFunc) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := auth.ContextWithUser(r.Context(), otherUser)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		}
		h2 := NewHandlers(store, otherWrap, otherWrap)
		mux2 := http.NewServeMux()
		h2.Register(mux2)

		req := httptest.NewRequest("DELETE", "/app/"+slug+"/invites/"+tok, nil)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (non-owner should be rejected)", w.Code, http.StatusNotFound)
		}
	})

	t.Run("unauthenticated_cannot_revoke", func(t *testing.T) {
		tok, err := store.CreateInviteCode(t.Context(), space.ID, "test-user-1", nil, 0)
		if err != nil {
			t.Fatalf("create invite: %v", err)
		}
		t.Cleanup(func() { store.RevokeInvite(t.Context(), tok) })

		anonWrap := func(next http.HandlerFunc) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}
		h2 := NewHandlers(store, anonWrap, anonWrap)
		mux2 := http.NewServeMux()
		h2.Register(mux2)

		req := httptest.NewRequest("DELETE", "/app/"+slug+"/invites/"+tok, nil)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (unauthenticated should be rejected)", w.Code, http.StatusNotFound)
		}
	})
}

// TestHandlerConveneOp verifies that a convene op creates a KindCouncil node
// with the correct body and tags (agent IDs resolved by name).
func TestHandlerConveneOp(t *testing.T) {
	_, store := testDB(t)
	ctx := t.Context()

	// Create two agent users for the council.
	agentAID := "convene-agent-a-id"
	agentAName := "ConveneAgentA"
	agentBID := "convene-agent-b-id"
	agentBName := "ConveneAgentB"
	for _, row := range []struct{ id, name string }{{agentAID, agentAName}, {agentBID, agentBName}} {
		store.db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, row.id)
		_, err := store.db.ExecContext(ctx,
			`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
			row.id, "agent:"+row.name, row.name+"@test.lovyou.ai", row.name)
		if err != nil {
			t.Fatalf("create agent %s: %v", row.name, err)
		}
	}
	t.Cleanup(func() {
		store.db.ExecContext(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, agentAID, agentBID)
	})

	testUser := &auth.User{ID: "convene-human-1", Name: "ConveneTester", Email: "convene@test.com", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(auth.ContextWithUser(r.Context(), testUser))
			next.ServeHTTP(w, r)
		})
	}
	h := NewHandlers(store, wrap, wrap)
	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(ctx, "convene-op-test"); old != nil {
		store.DeleteSpace(ctx, old.ID)
	}
	space, err := store.CreateSpace(ctx, "convene-op-test", "Convene Op Test", "", testUser.ID, "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	body := `{"op":"convene","title":"What should we build next?","body":"Please share your perspective.","agents":"` + agentAName + `,` + agentBName + `"}`
	req := httptest.NewRequest("POST", "/app/convene-op-test/op", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["op"] != "convene" {
		t.Errorf("op = %v, want convene", result["op"])
	}
	node := result["node"].(map[string]any)
	if node["kind"] != KindCouncil {
		t.Errorf("kind = %v, want %q", node["kind"], KindCouncil)
	}
	if node["body"] != "Please share your perspective." {
		t.Errorf("body = %v, want 'Please share your perspective.'", node["body"])
	}
	if node["title"] != "What should we build next?" {
		t.Errorf("title = %v, want 'What should we build next?'", node["title"])
	}
	tags, _ := node["tags"].([]any)
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag.(string)] = true
	}
	if !tagSet[agentAID] {
		t.Errorf("tags missing agent A ID %q; got %v", agentAID, tags)
	}
	if !tagSet[agentBID] {
		t.Errorf("tags missing agent B ID %q; got %v", agentBID, tags)
	}
}

// TestHandlerCouncilDetail verifies that GET /app/{slug}/council/{id}
// returns 200 with response rows when Mind response nodes exist.
func TestHandlerCouncilDetail(t *testing.T) {
	h, store, _ := testHandlers(t)
	ctx := t.Context()

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(ctx, "council-detail-test"); old != nil {
		store.DeleteSpace(ctx, old.ID)
	}
	space, err := store.CreateSpace(ctx, "council-detail-test", "Council Detail Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	council, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindCouncil,
		Title:    "Should we prioritize performance?",
		Body:     "Looking for agent perspectives.",
		Author:   "Tester",
		AuthorID: "test-user-1",
		Tags:     []string{"agent-resp-1"},
	})
	if err != nil {
		t.Fatalf("create council: %v", err)
	}

	// Simulate two Mind responses as child KindComment nodes.
	for i, name := range []string{"AgentX", "AgentY"} {
		_, err := store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			ParentID:   council.ID,
			Kind:       KindComment,
			Body:       "Response from " + name,
			Author:     name,
			AuthorID:   "agent-resp-" + string(rune('1'+i)),
			AuthorKind: "agent",
		})
		if err != nil {
			t.Fatalf("create response %s: %v", name, err)
		}
	}

	req := httptest.NewRequest("GET", "/app/council-detail-test/council/"+council.ID, nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	session := result["session"].(map[string]any)
	if session["id"] != council.ID {
		t.Errorf("session.id = %v, want %v", session["id"], council.ID)
	}
	if session["kind"] != KindCouncil {
		t.Errorf("session.kind = %v, want %q", session["kind"], KindCouncil)
	}
	responses, _ := result["responses"].([]any)
	if len(responses) != 2 {
		t.Errorf("got %d responses, want 2", len(responses))
	}
}

// TestHandlerCouncilDetail_NotFound verifies that a wrong kind or missing node returns 404.
func TestHandlerCouncilDetail_NotFound(t *testing.T) {
	h, store, _ := testHandlers(t)
	ctx := t.Context()

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(ctx, "council-notfound-test"); old != nil {
		store.DeleteSpace(ctx, old.ID)
	}
	space, err := store.CreateSpace(ctx, "council-notfound-test", "Council NotFound Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	t.Run("nonexistent_id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/app/council-notfound-test/council/no-such-id", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
		}
	})

	t.Run("wrong_kind_returns_404", func(t *testing.T) {
		task, err := store.CreateNode(ctx, CreateNodeParams{
			SpaceID: space.ID, Kind: KindTask, Title: "Not a council",
			Author: "Tester", AuthorID: "test-user-1",
		})
		if err != nil {
			t.Fatalf("create task: %v", err)
		}
		req := httptest.NewRequest("GET", "/app/council-notfound-test/council/"+task.ID, nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (wrong kind should 404)", w.Code, http.StatusNotFound)
		}
	})
}

// TestHandlerQuestionAutoAnswer verifies that creating a KindQuestion via the intend op
// triggers Mind.OnQuestionAsked and results in a respond op on the answer node.
func TestHandlerQuestionAutoAnswer(t *testing.T) {
	db, store := testDB(t)
	ctx := t.Context()

	agentID := "qa-auto-answer-agent-id"
	agentName := "QAAutoAnswerAgent"
	db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID)
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, google_id, email, name, kind) VALUES ($1, $2, $3, $4, 'agent')`,
		agentID, "agent:"+agentName, agentName+"@test.lovyou.ai", agentName)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	t.Cleanup(func() { db.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, agentID) })

	mind := NewMind(db, store, "fake-token")
	mind.callClaudeOverride = func(_ context.Context, _ string, _ []claudeMessage) (string, error) {
		return "Auto-answer from mind.", nil
	}

	testUser := &auth.User{ID: "qa-auto-human-1", Name: "QATester", Email: "qa@test.com", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(auth.ContextWithUser(r.Context(), testUser))
			next.ServeHTTP(w, r)
		})
	}
	h := NewHandlers(store, wrap, wrap)
	h.SetMind(mind)
	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(ctx, "qa-auto-answer-test"); old != nil {
		store.DeleteSpace(ctx, old.ID)
	}
	space, err := store.CreateSpace(ctx, "qa-auto-answer-test", "QA Auto Answer Test", "", testUser.ID, "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	body := `{"op":"intend","title":"What is the event graph?","description":"Looking for a quick overview","kind":"question"}`
	req := httptest.NewRequest("POST", "/app/qa-auto-answer-test/op", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var result map[string]any
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	node := result["node"].(map[string]any)
	questionID := node["id"].(string)

	// OnQuestionAsked runs in a goroutine; poll briefly for the respond op.
	deadline := time.Now().Add(2 * time.Second)
	var hasRespondOp bool
	for time.Now().Before(deadline) {
		ops, err := store.ListOps(ctx, space.ID, 50)
		if err != nil {
			t.Fatalf("list ops: %v", err)
		}
		for _, o := range ops {
			if o.Op == "respond" && o.NodeID != questionID {
				hasRespondOp = true
				break
			}
		}
		if hasRespondOp {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !hasRespondOp {
		t.Errorf("expected a respond op on the answer node after creating KindQuestion %s, got none", questionID)
	}
}

// TestHivePage issues GET /hive and asserts HTTP 200 and the body contains
// the "Phase timeline" section heading that the template always renders.
func TestHivePage(t *testing.T) {
	h, _, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest("GET", "/hive", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /hive: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Phase timeline") {
		t.Error("GET /hive: body does not contain 'Phase timeline'")
	}
}

// TestListHiveActivity calls ListHiveActivity with kind=post and the hive agent's user ID,
// asserts the result is non-nil and bounded to ≤10 items.
func TestListHiveActivity(t *testing.T) {
	_, store := testDB(t)
	ctx := t.Context()

	agentUserID := fmt.Sprintf("hive-list-activity-test-agent-%d", time.Now().UnixNano())
	slug := fmt.Sprintf("hive-list-activity-test-%d", time.Now().UnixNano())

	space, err := store.CreateSpace(ctx, slug, "Hive List Activity Test", "", "owner-list-activity", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	for i := range 3 {
		_, err := store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindPost,
			Title:      fmt.Sprintf("[hive:builder] iter %d: shipped", i),
			Body:       "Builder shipped. Cost: $0.42.",
			Author:     "hive-builder",
			AuthorID:   agentUserID,
			AuthorKind: "agent",
		})
		if err != nil {
			t.Fatalf("create post %d: %v", i, err)
		}
	}

	nodes, err := store.ListHiveActivity(ctx, agentUserID, 10)
	if err != nil {
		t.Fatalf("ListHiveActivity: %v", err)
	}
	if nodes == nil {
		t.Fatal("ListHiveActivity: result is nil, want non-nil slice")
	}
	if len(nodes) > 10 {
		t.Errorf("ListHiveActivity: %d items, want ≤10", len(nodes))
	}
}

// TestHandlerCompleteOpChildrenIncomplete verifies that POST /app/{slug}/op
// with op=complete returns 422 when the target node has incomplete children.
func TestHandlerCompleteOpChildrenIncomplete(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-complete-gate-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-complete-gate-test", "Complete Gate Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	parent, err := store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Parent",
		Author: "Tester", AuthorID: "test-user-1", State: StateActive,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	_, err = store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, ParentID: parent.ID, Kind: KindTask, Title: "Child",
		Author: "Tester", AuthorID: "test-user-1", State: StateActive,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	// Attempt to complete parent via op=complete with incomplete child.
	body := `{"op":"complete","node_id":"` + parent.ID + `"}`
	req := httptest.NewRequest("POST", "/app/handler-complete-gate-test/op", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d (422); body: %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

// TestHandlerNodeStateChildrenIncomplete verifies that POST /app/{slug}/node/{id}/state
// with state=done returns 422 when the node has incomplete children.
func TestHandlerNodeStateChildrenIncomplete(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-nodestate-gate-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-nodestate-gate-test", "NodeState Gate Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	parent, err := store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Parent",
		Author: "Tester", AuthorID: "test-user-1", State: StateActive,
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	_, err = store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID: space.ID, ParentID: parent.ID, Kind: KindTask, Title: "Child",
		Author: "Tester", AuthorID: "test-user-1", State: StateActive,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	// Attempt to mark parent done via node state route with incomplete child.
	form := url.Values{}
	form.Set("state", StateDone)
	req := httptest.NewRequest("POST", "/app/handler-nodestate-gate-test/node/"+parent.ID+"/state", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d (422); body: %s", w.Code, http.StatusUnprocessableEntity, w.Body.String())
	}
}

// TestHandlerOpEditCauses verifies that op=edit with a causes field updates the
// node's causes via UpdateNodeCauses. This is the handler path used by
// backfillClaimCauses in cmd/post to retroactively satisfy Invariant 2 (CAUSALITY).
func TestHandlerOpEditCauses(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	if old, _ := store.GetSpaceBySlug(t.Context(), "handler-edit-causes-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "handler-edit-causes-test", "Edit Causes Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	// Create a claim node with no causes (simulates the 103 legacy claims).
	node, err := store.CreateNode(t.Context(), CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindClaim,
		Title:    "Legacy Claim",
		Author:   "Tester",
		AuthorID: "test-user-1",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	t.Run("edit_causes_single", func(t *testing.T) {
		body := `{"op":"edit","node_id":"` + node.ID + `","causes":"task-node-abc123"}`
		req := httptest.NewRequest("POST", "/app/handler-edit-causes-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		// Verify the causes were persisted.
		got, err := store.GetNode(t.Context(), node.ID)
		if err != nil {
			t.Fatalf("get node: %v", err)
		}
		if len(got.Causes) != 1 || got.Causes[0] != "task-node-abc123" {
			t.Errorf("causes = %v, want [task-node-abc123]", got.Causes)
		}
	})

	t.Run("edit_causes_multiple_comma_separated", func(t *testing.T) {
		body := `{"op":"edit","node_id":"` + node.ID + `","causes":"task-aaa,task-bbb,task-ccc"}`
		req := httptest.NewRequest("POST", "/app/handler-edit-causes-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}

		got, err := store.GetNode(t.Context(), node.ID)
		if err != nil {
			t.Fatalf("get node: %v", err)
		}
		if len(got.Causes) != 3 {
			t.Errorf("causes = %v, want 3 entries", got.Causes)
		}
	})

	t.Run("edit_requires_body_or_causes", func(t *testing.T) {
		// op=edit with neither body nor causes should fail.
		body := `{"op":"edit","node_id":"` + node.ID + `"}`
		req := httptest.NewRequest("POST", "/app/handler-edit-causes-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (400) when neither body nor causes provided; body: %s",
				w.Code, http.StatusBadRequest, w.Body.String())
		}
	})
}

// TestPopulateFormFromJSON is a pure unit test for the populateFormFromJSON helper.
// No database required.
func TestPopulateFormFromJSON(t *testing.T) {
	makeReq := func(contentType, body string) *http.Request {
		r, _ := http.NewRequest("POST", "/", strings.NewReader(body))
		r.Header.Set("Content-Type", contentType)
		return r
	}

	t.Run("array causes to CSV", func(t *testing.T) {
		r := makeReq("application/json", `{"op":"assert","causes":["id1","id2"]}`)
		populateFormFromJSON(r)
		if got := r.FormValue("op"); got != "assert" {
			t.Errorf("op = %q, want %q", got, "assert")
		}
		if got := r.FormValue("causes"); got != "id1,id2" {
			t.Errorf("causes = %q, want %q", got, "id1,id2")
		}
	})

	t.Run("string value pass-through", func(t *testing.T) {
		r := makeReq("application/json", `{"title":"hello world"}`)
		populateFormFromJSON(r)
		if got := r.FormValue("title"); got != "hello world" {
			t.Errorf("title = %q, want %q", got, "hello world")
		}
	})

	t.Run("non-JSON content-type is no-op", func(t *testing.T) {
		r := makeReq("application/x-www-form-urlencoded", `{"op":"assert"}`)
		populateFormFromJSON(r)
		if r.Form != nil && r.FormValue("op") != "" {
			t.Errorf("expected no form population for non-JSON content-type")
		}
	})

	t.Run("invalid JSON is no-op (no panic)", func(t *testing.T) {
		r := makeReq("application/json", `{not valid json`)
		populateFormFromJSON(r) // must not panic
		if r.Form != nil && len(r.Form) != 0 {
			t.Errorf("expected empty form for invalid JSON")
		}
	})

	t.Run("empty array produces empty string", func(t *testing.T) {
		r := makeReq("application/json", `{"causes":[]}`)
		populateFormFromJSON(r)
		if got := r.FormValue("causes"); got != "" {
			t.Errorf("causes = %q, want empty string for empty array", got)
		}
	})

	t.Run("null value is skipped", func(t *testing.T) {
		r := makeReq("application/json", `{"causes":null,"op":"assert"}`)
		populateFormFromJSON(r)
		if got := r.FormValue("causes"); got != "" {
			t.Errorf("causes = %q, want empty (null should be skipped)", got)
		}
		if got := r.FormValue("op"); got != "assert" {
			t.Errorf("op = %q, want %q", got, "assert")
		}
	})

	t.Run("numeric value via fmt.Sprintf", func(t *testing.T) {
		r := makeReq("application/json", `{"priority":5}`)
		populateFormFromJSON(r)
		if got := r.FormValue("priority"); got != "5" {
			t.Errorf("priority = %q, want %q", got, "5")
		}
	})

	t.Run("content-type with charset suffix", func(t *testing.T) {
		r := makeReq("application/json; charset=utf-8", `{"op":"intend"}`)
		populateFormFromJSON(r)
		if got := r.FormValue("op"); got != "intend" {
			t.Errorf("op = %q, want %q", got, "intend")
		}
	})

	t.Run("array with non-string items drops non-strings", func(t *testing.T) {
		r := makeReq("application/json", `{"causes":["id1",42,"id2"]}`)
		populateFormFromJSON(r)
		// 42 is a number, not a string — dropped silently; id1 and id2 are kept
		if got := r.FormValue("causes"); got != "id1,id2" {
			t.Errorf("causes = %q, want %q", got, "id1,id2")
		}
	})

	// Ensure the body is consumed only once — calling on a request with no body
	// should not panic.
	t.Run("empty body is no-op", func(t *testing.T) {
		r, _ := http.NewRequest("POST", "/", nil)
		r.Header.Set("Content-Type", "application/json")
		r.Body = io.NopCloser(strings.NewReader(""))
		populateFormFromJSON(r) // must not panic
	})

	t.Run("array with null item drops null keeps strings", func(t *testing.T) {
		r := makeReq("application/json", `{"causes":["id1",null,"id2"]}`)
		populateFormFromJSON(r)
		if got := r.FormValue("causes"); got != "id1,id2" {
			t.Errorf("causes = %q, want %q", got, "id1,id2")
		}
	})
}

func TestHandlerGovernanceDelegation(t *testing.T) {
	h, store, _ := testHandlers(t)

	mux := http.NewServeMux()
	h.Register(mux)

	// Clean up stale space.
	if old, _ := store.GetSpaceBySlug(t.Context(), "gov-handler-test"); old != nil {
		store.DeleteSpace(t.Context(), old.ID)
	}
	space, err := store.CreateSpace(t.Context(), "gov-handler-test", "Gov Handler Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("propose_with_quorum_pct", func(t *testing.T) {
		body := `{"op":"propose","title":"Quorum Proposal","quorum_pct":"51","voting_body":"all"}`
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
		}
		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		if result["op"] != "propose" {
			t.Errorf("op = %v, want propose", result["op"])
		}
	})

	t.Run("delegate_op", func(t *testing.T) {
		body := `{"op":"delegate","delegate_id":"other-user-999"}`
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}
		var result map[string]any
		json.NewDecoder(w.Body).Decode(&result)
		if result["op"] != OpDelegate {
			t.Errorf("op = %v, want %v", result["op"], OpDelegate)
		}
		if !store.HasDelegated(t.Context(), space.ID, "test-user-1") {
			t.Error("HasDelegated after delegate op = false, want true")
		}
	})

	t.Run("vote_blocked_when_delegated", func(t *testing.T) {
		// test-user-1 has delegated (from previous subtest), so vote should fail.
		proposals, _ := store.ListProposals(t.Context(), space.ID, "open", 10)
		if len(proposals) == 0 {
			t.Skip("no open proposals in space")
		}
		body := fmt.Sprintf(`{"op":"vote","node_id":%q,"vote":"yes"}`, proposals[0].ID)
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("status = %d, want %d (delegated user cannot vote directly)", w.Code, http.StatusConflict)
		}
	})

	t.Run("undelegate_op", func(t *testing.T) {
		body := `{"op":"undelegate"}`
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
		}
		if store.HasDelegated(t.Context(), space.ID, "test-user-1") {
			t.Error("HasDelegated after undelegate op = true, want false")
		}
	})

	t.Run("delegate_missing_delegate_id", func(t *testing.T) {
		body := `{"op":"delegate"}`
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d (missing delegate_id)", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("vote_after_undelegate", func(t *testing.T) {
		// test-user-1 has no delegation (removed by undelegate_op).
		// Create a fresh proposal, then vote — must succeed.
		propBody := `{"op":"propose","title":"Vote After Undelegate"}`
		req := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(propBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("propose: status = %d; body: %s", w.Code, w.Body.String())
		}
		var propResult map[string]any
		json.NewDecoder(w.Body).Decode(&propResult)
		nodeMap, _ := propResult["node"].(map[string]any)
		nodeID, _ := nodeMap["id"].(string)
		if nodeID == "" {
			t.Skip("could not extract node ID from propose response")
		}

		voteBody := fmt.Sprintf(`{"op":"vote","node_id":%q,"vote":"yes"}`, nodeID)
		req2 := httptest.NewRequest("POST", "/app/gov-handler-test/op", strings.NewReader(voteBody))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Accept", "application/json")
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)
		if w2.Code != http.StatusOK {
			t.Errorf("vote after undelegate: status = %d, want %d; body: %s", w2.Code, http.StatusOK, w2.Body.String())
		}
	})
}
