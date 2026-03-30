package graph

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBridgeActionWebhook(t *testing.T) {
	_, store := testDB(t)
	h := NewHandlers(store, nil, nil)

	body := `{
		"agent_name": "sdr",
		"action_type": "approval",
		"summary": "Outbound email to Sarah Chen",
		"authority": "required",
		"target_human": "user-matt",
		"domain_data": {"lead_id": "lead-001", "score": 62}
	}`

	req := httptest.NewRequest("POST", "/api/bridge/action", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.handleBridgeActionWebhook(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

func TestBridgeEventWebhook(t *testing.T) {
	_, store := testDB(t)
	h := NewHandlers(store, nil, nil)

	body := `{
		"agent_name": "sdr",
		"event_type": "membrane.service.polled",
		"payload": {"endpoint": "/api/leads", "events_found": 3}
	}`

	req := httptest.NewRequest("POST", "/api/bridge/event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.handleBridgeEventWebhook(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

func TestBridgeDecisionsAPI(t *testing.T) {
	_, store := testDB(t)
	ctx := httptest.NewRequest("GET", "/", nil).Context()
	h := NewHandlers(store, nil, nil)

	// Create a pending action
	store.CreateBridgeAction(ctx, BridgeAction{
		AgentName: "sdr", ActionType: "approval", Summary: "Email",
		Authority: "required", TargetHuman: "user-matt", Status: "pending",
	})

	// Decide it
	actions, _ := store.ListPendingBridgeActions(ctx, "user-matt", 10)
	store.DecideBridgeAction(ctx, actions[0].ID, "approved", "user-matt", "ok")

	// Poll for decisions
	req := httptest.NewRequest("GET", "/api/bridge/decisions?agent=sdr", nil)
	w := httptest.NewRecorder()

	h.handleBridgeDecisionsAPI(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}
