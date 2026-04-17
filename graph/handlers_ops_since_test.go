package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// seedOpsSpace creates a space with N ops spread over time.
func seedOpsSpace(t *testing.T, store *Store, n int) (*Space, []Op) {
	t.Helper()
	ctx := context.Background()

	slug := fmt.Sprintf("ops-since-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(ctx, slug, "Ops Since", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	node, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "x", Author: "Tester", AuthorID: "test-user-1",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	ops := make([]Op, 0, n)
	for i := 0; i < n; i++ {
		op, err := store.RecordOp(ctx, space.ID, node.ID, "Tester", "test-user-1", "progress", nil)
		if err != nil {
			t.Fatalf("record op %d: %v", i, err)
		}
		ops = append(ops, *op)
		// Ensure monotonic created_at ordering even on fast clocks.
		time.Sleep(2 * time.Millisecond)
	}
	return space, ops
}

func getOpsSince(t *testing.T, h *Handlers, slug string, query string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	mux := http.NewServeMux()
	h.Register(mux)

	url := fmt.Sprintf("/app/%s/ops", slug)
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var body map[string]any
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &body)
	}
	return w, body
}

func TestOpsSince_ReturnsOpsAfterTimestamp(t *testing.T) {
	h, store, _ := testHandlers(t)
	space, ops := seedOpsSpace(t, store, 5)
	if len(ops) < 5 {
		t.Fatalf("need 5 ops, got %d", len(ops))
	}

	// Request ops strictly after ops[1].CreatedAt — this should include ops[1..4].
	cutoff := ops[1].CreatedAt.Add(-1 * time.Microsecond).UTC().Format(time.RFC3339Nano)
	w, body := getOpsSince(t, h, space.Slug, "since="+cutoff)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	count, _ := body["count"].(float64)
	if int(count) < 4 {
		t.Errorf("count = %v, want >= 4 ops after cutoff", count)
	}
}

func TestOpsSince_SortsAscending(t *testing.T) {
	h, store, _ := testHandlers(t)
	space, _ := seedOpsSpace(t, store, 4)

	w, body := getOpsSince(t, h, space.Slug, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	opsRaw, _ := body["ops"].([]any)
	if len(opsRaw) < 2 {
		t.Fatalf("need >=2 ops to check ordering, got %d", len(opsRaw))
	}

	prev := ""
	for i, o := range opsRaw {
		m := o.(map[string]any)
		ts, _ := m["created_at"].(string)
		if i > 0 && ts < prev {
			t.Errorf("ops[%d].created_at %q < prev %q — not ascending", i, ts, prev)
		}
		prev = ts
	}
}

func TestOpsSince_HonorsLimit(t *testing.T) {
	h, store, _ := testHandlers(t)
	space, _ := seedOpsSpace(t, store, 6)

	w, body := getOpsSince(t, h, space.Slug, "limit=3")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	count, _ := body["count"].(float64)
	if int(count) != 3 {
		t.Errorf("count = %v, want 3", count)
	}
}

func TestOpsSince_UnknownSlugReturns404(t *testing.T) {
	h, _, _ := testHandlers(t)
	w, _ := getOpsSince(t, h, "no-such-space-xyz", "")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestOpsSince_BadTimestampReturns400(t *testing.T) {
	h, store, _ := testHandlers(t)
	space, _ := seedOpsSpace(t, store, 1)

	w, _ := getOpsSince(t, h, space.Slug, "since=not-a-date")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestOpsSince_EmptyResultReturnsEmptyArray(t *testing.T) {
	h, store, _ := testHandlers(t)
	space, _ := seedOpsSpace(t, store, 1)

	// Far-future cutoff — no ops match.
	future := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339Nano)
	w, body := getOpsSince(t, h, space.Slug, "since="+future)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	opsRaw, ok := body["ops"].([]any)
	if !ok {
		t.Fatalf("ops missing or wrong type: %T", body["ops"])
	}
	if len(opsRaw) != 0 {
		t.Errorf("len(ops) = %d, want 0", len(opsRaw))
	}
}
