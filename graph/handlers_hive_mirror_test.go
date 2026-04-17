package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// setupMirrorTest creates a handler + space + op that can be stamped with a
// hive_chain_ref. Returns the seed op ID.
func setupMirrorTest(t *testing.T) (*Handlers, *Store, string, func()) {
	t.Helper()
	h, store, _ := testHandlers(t)
	ctx := context.Background()

	slug := fmt.Sprintf("mirror-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(ctx, slug, "Mirror Test", "", "test-user-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	node, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "t", Author: "Tester", AuthorID: "test-user-1",
	})
	if err != nil {
		t.Fatalf("create node: %v", err)
	}
	op, err := store.RecordOp(ctx, space.ID, node.ID, "Tester", "test-user-1", "complete", nil)
	if err != nil {
		t.Fatalf("record op: %v", err)
	}
	cleanup := func() { store.DeleteSpace(ctx, space.ID) }
	return h, store, op.ID, cleanup
}

func postMirror(t *testing.T, h *Handlers, body any) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	h.Register(mux)

	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest("POST", "/api/hive/mirror", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestHiveMirror_StampsHiveChainRef(t *testing.T) {
	h, store, opID, cleanup := setupMirrorTest(t)
	t.Cleanup(cleanup)

	w := postMirror(t, h, map[string]string{
		"op_id":          opID,
		"hive_chain_ref": "cas://event/abc123",
		"event_type":     "hive.op.observed",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	got, err := store.GetOp(context.Background(), opID)
	if err != nil {
		t.Fatalf("get op: %v", err)
	}
	if got.HiveChainRef != "cas://event/abc123" {
		t.Errorf("hive_chain_ref = %q, want cas://event/abc123", got.HiveChainRef)
	}
}

func TestHiveMirror_MissingFieldsReturns400(t *testing.T) {
	h, _, opID, cleanup := setupMirrorTest(t)
	t.Cleanup(cleanup)

	cases := []map[string]string{
		{"hive_chain_ref": "cas://x", "event_type": "hive.op.observed"},                     // no op_id
		{"op_id": opID, "event_type": "hive.op.observed"},                                   // no hive_chain_ref
		{"op_id": opID, "hive_chain_ref": "cas://x"},                                        // no event_type
	}
	for i, body := range cases {
		w := postMirror(t, h, body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("case %d: status = %d, want 400; body: %s", i, w.Code, w.Body.String())
		}
	}
}

// Unknown op_id is a silent no-op (200) — by design, the hive may echo ops
// that predate this column or have been evicted.
func TestHiveMirror_UnknownOpIsSilentNoOp(t *testing.T) {
	h, _, _, cleanup := setupMirrorTest(t)
	t.Cleanup(cleanup)

	w := postMirror(t, h, map[string]string{
		"op_id":          "does-not-exist",
		"hive_chain_ref": "cas://x",
		"event_type":     "hive.op.observed",
	})
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (silent no-op); body: %s", w.Code, w.Body.String())
	}
}

func TestHiveMirror_IdempotentSameRef(t *testing.T) {
	h, store, opID, cleanup := setupMirrorTest(t)
	t.Cleanup(cleanup)

	body := map[string]string{
		"op_id":          opID,
		"hive_chain_ref": "cas://event/same",
		"event_type":     "hive.op.observed",
	}
	for i := 0; i < 3; i++ {
		w := postMirror(t, h, body)
		if w.Code != http.StatusOK {
			t.Fatalf("call %d: status = %d, want 200; body: %s", i, w.Code, w.Body.String())
		}
	}
	got, _ := store.GetOp(context.Background(), opID)
	if got.HiveChainRef != "cas://event/same" {
		t.Errorf("hive_chain_ref = %q, want cas://event/same", got.HiveChainRef)
	}
}

// Latest-wins: a new ref overwrites a previous stamp.
func TestHiveMirror_LatestRefWins(t *testing.T) {
	h, store, opID, cleanup := setupMirrorTest(t)
	t.Cleanup(cleanup)

	w1 := postMirror(t, h, map[string]string{
		"op_id":          opID,
		"hive_chain_ref": "cas://event/first",
		"event_type":     "hive.op.observed",
	})
	if w1.Code != http.StatusOK {
		t.Fatalf("first stamp: status = %d", w1.Code)
	}

	w2 := postMirror(t, h, map[string]string{
		"op_id":          opID,
		"hive_chain_ref": "cas://event/second",
		"event_type":     "hive.op.observed",
	})
	if w2.Code != http.StatusOK {
		t.Fatalf("second stamp: status = %d, want 200 (latest-wins)", w2.Code)
	}

	got, _ := store.GetOp(context.Background(), opID)
	if got.HiveChainRef != "cas://event/second" {
		t.Errorf("hive_chain_ref = %q, want cas://event/second", got.HiveChainRef)
	}
}

func TestHiveMirror_InvalidJSONReturns400(t *testing.T) {
	h, _, _, cleanup := setupMirrorTest(t)
	t.Cleanup(cleanup)

	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest("POST", "/api/hive/mirror", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// hive.spec.completed events should drive derived node state to "completed".
func TestHiveMirror_UpdatesDerivedNodeForSpecCompleted(t *testing.T) {
	h, store, opID, cleanup := setupMirrorTest(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	opBefore, _ := store.GetOp(ctx, opID)
	nodeID := opBefore.NodeID
	if nodeID == "" {
		t.Fatal("seed op has no node_id — cannot test derived-node update")
	}

	w := postMirror(t, h, map[string]string{
		"op_id":          opID,
		"hive_chain_ref": "cas://spec/abc",
		"event_type":     "hive.spec.completed",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	got, err := store.GetNode(ctx, nodeID)
	if err != nil {
		t.Fatalf("get node: %v", err)
	}
	if got.State != "completed" {
		t.Errorf("node state = %q, want completed", got.State)
	}
}
