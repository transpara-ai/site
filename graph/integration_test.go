package graph

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lovyou-ai/site/auth"
)

// TestNewUserJourney simulates a complete new user experience:
// sign in → create space → create content → create conversation → view conversation.
func TestNewUserJourney(t *testing.T) {
	_, store := testDB(t)
	ctx := t.Context()

	// Simulate a logged-in user.
	user := &auth.User{ID: "journey-user-1", Name: "Journey Tester", Email: "journey@test.com", Kind: "human"}
	wrap := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	h := NewHandlers(store, wrap, wrap)
	mux := http.NewServeMux()
	h.Register(mux)

	// Clean up from prior runs.
	if old, _ := store.GetSpaceBySlug(ctx, "journey-test"); old != nil {
		store.DeleteSpace(ctx, old.ID)
	}

	// Step 1: Create a space.
	resp := jsonPost(t, mux, "/app/new", `{"name":"Journey Test","description":"E2E journey","kind":"community","visibility":"public"}`)
	if resp.Code != http.StatusCreated && resp.Code != http.StatusSeeOther {
		t.Fatalf("create space: status %d, body: %s", resp.Code, resp.Body.String())
	}

	space, err := store.GetSpaceBySlug(ctx, "journey-test")
	if err != nil {
		t.Fatalf("space not created: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	// Step 2: Create a task (intend).
	resp = jsonOp(t, mux, "journey-test", `{"op":"intend","title":"Build something","priority":"high"}`)
	if resp.Code != http.StatusCreated {
		t.Fatalf("intend: status %d, body: %s", resp.Code, resp.Body.String())
	}
	var intendResult map[string]any
	json.NewDecoder(resp.Body).Decode(&intendResult)
	taskNode := intendResult["node"].(map[string]any)
	if taskNode["author_id"] != user.ID {
		t.Errorf("task author_id = %v, want %s", taskNode["author_id"], user.ID)
	}

	// Step 3: Create a post (express).
	resp = jsonOp(t, mux, "journey-test", `{"op":"express","title":"First update","body":"Hello world"}`)
	if resp.Code != http.StatusCreated {
		t.Fatalf("express: status %d", resp.Code)
	}

	// Step 4: Create a conversation (converse).
	resp = jsonOp(t, mux, "journey-test", `{"op":"converse","title":"Team Chat","body":"Let's plan"}`)
	if resp.Code != http.StatusCreated {
		t.Fatalf("converse: status %d, body: %s", resp.Code, resp.Body.String())
	}
	var converseResult map[string]any
	json.NewDecoder(resp.Body).Decode(&converseResult)
	convoNode := converseResult["node"].(map[string]any)
	convoID := convoNode["id"].(string)

	// Verify conversation has the creator's ID in tags.
	tags := convoNode["tags"].([]any)
	found := false
	for _, tag := range tags {
		if tag.(string) == user.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("conversation tags should contain creator's user ID %s, got %v", user.ID, tags)
	}

	// Step 5: Send a message in the conversation (respond).
	resp = jsonOp(t, mux, "journey-test", `{"op":"respond","parent_id":"`+convoID+`","body":"Let's do it"}`)
	if resp.Code != http.StatusCreated {
		t.Fatalf("respond: status %d, body: %s", resp.Code, resp.Body.String())
	}

	// Step 6: View the conversation detail.
	req := httptest.NewRequest("GET", "/app/journey-test/conversation/"+convoID, nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("conversation detail: status %d", w.Code)
	}

	var detail map[string]any
	json.NewDecoder(w.Body).Decode(&detail)
	messages := detail["messages"].([]any)
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}
	msg := messages[0].(map[string]any)
	if msg["author_id"] != user.ID {
		t.Errorf("message author_id = %v, want %s", msg["author_id"], user.ID)
	}

	// Step 7: List conversations — should find ours.
	req = httptest.NewRequest("GET", "/app/journey-test/conversations", nil)
	req.Header.Set("Accept", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list conversations: status %d", w.Code)
	}

	var listResult map[string]any
	json.NewDecoder(w.Body).Decode(&listResult)
	convos := listResult["conversations"].([]any)
	// handleCreateSpace auto-creates a "Welcome" conversation, so there will be
	// at least 2 conversations here (Welcome + the Team Chat we created). Assert
	// that our Team Chat is in the list rather than pinning the total count.
	foundOurs := false
	for _, c := range convos {
		if c.(map[string]any)["id"] == convoID {
			foundOurs = true
			break
		}
	}
	if !foundOurs {
		t.Errorf("conversation %s not found in list of %d conversations", convoID, len(convos))
	}
}

func jsonPost(t *testing.T, mux *http.ServeMux, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func jsonOp(t *testing.T, mux *http.ServeMux, spaceSlug, body string) *httptest.ResponseRecorder {
	t.Helper()
	return jsonPost(t, mux, "/app/"+spaceSlug+"/op", body)
}
