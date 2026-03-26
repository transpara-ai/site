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

	space, err := store.CreateSpace(t.Context(), "join-test-space", "Join Test", "", "owner-1", "project", "private")
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

		req := httptest.NewRequest("GET", "/app/join/"+token, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
		}
		want := "/auth/login?next=%2Fapp%2Fjoin%2F" + token
		if loc := w.Header().Get("Location"); loc != want {
			t.Errorf("Location = %q, want %q", loc, want)
		}
	})

	t.Run("valid_code_joins_user", func(t *testing.T) {
		h := NewHandlers(store, authWrap, authWrap)
		mux := http.NewServeMux()
		h.Register(mux)

		req := httptest.NewRequest("GET", "/app/join/"+token, nil)
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

		req := httptest.NewRequest("GET", "/app/join/nonexistent-token", nil)
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

	space, err := store.CreateSpace(t.Context(), "htmx-invite-test", "HTMX Invite Test", "", "test-user-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("owner_creates_invite_returns_html", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/app/htmx-invite-test/invites", nil)
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

		req := httptest.NewRequest("POST", "/app/htmx-invite-test/invites", nil)
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

		req := httptest.NewRequest("POST", "/app/htmx-invite-test/invites", nil)
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

	space, err := store.CreateSpace(t.Context(), "revoke-invite-test", "Revoke Invite Test", "", "test-user-1", "project", "private")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(t.Context(), space.ID) })

	t.Run("revoke_existing_invite", func(t *testing.T) {
		tok, err := store.CreateInviteCode(t.Context(), space.ID, "test-user-1", nil, 0)
		if err != nil {
			t.Fatalf("create invite: %v", err)
		}

		req := httptest.NewRequest("DELETE", "/app/revoke-invite-test/invites/"+tok, nil)
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
		req := httptest.NewRequest("DELETE", "/app/revoke-invite-test/invites/no-such-token", nil)
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

		req := httptest.NewRequest("DELETE", "/app/revoke-invite-test/invites/"+tok, nil)
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

		req := httptest.NewRequest("DELETE", "/app/revoke-invite-test/invites/"+tok, nil)
		w := httptest.NewRecorder()
		mux2.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d (unauthenticated should be rejected)", w.Code, http.StatusNotFound)
		}
	})
}
