package graph

import (
	"context"
	"database/sql"
	"testing"
)

func cleanBridgeTables(t *testing.T, db *sql.DB) {
	t.Helper()
	db.Exec("TRUNCATE bridge_actions, bridge_events, bridge_notify_preferences")
}

func TestCreateAndGetBridgeAction(t *testing.T) {
	db, store := testDB(t)
	cleanBridgeTables(t, db)
	ctx := context.Background()

	action, err := store.CreateBridgeAction(ctx, BridgeAction{
		AgentName:   "sdr",
		ActionType:  "approval",
		Summary:     "Outbound email to Sarah Chen",
		Authority:   "required",
		TargetHuman: "user-matt",
		Status:      "pending",
		DomainData:  []byte(`{"lead_id":"lead-001","score":62}`),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if action.ID == "" {
		t.Fatal("action ID should not be empty")
	}

	got, err := store.GetBridgeAction(ctx, action.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AgentName != "sdr" {
		t.Errorf("agent = %q, want sdr", got.AgentName)
	}
	if got.Status != "pending" {
		t.Errorf("status = %q, want pending", got.Status)
	}
}

func TestListPendingBridgeActions(t *testing.T) {
	db, store := testDB(t)
	cleanBridgeTables(t, db)
	ctx := context.Background()

	// Create 2 actions for matt, 1 for someone else
	store.CreateBridgeAction(ctx, BridgeAction{
		AgentName: "sdr", ActionType: "approval", Summary: "Email 1",
		Authority: "required", TargetHuman: "user-matt", Status: "pending",
	})
	store.CreateBridgeAction(ctx, BridgeAction{
		AgentName: "sdr", ActionType: "handoff", Summary: "Handoff 1",
		Authority: "required", TargetHuman: "user-matt", Status: "pending",
	})
	store.CreateBridgeAction(ctx, BridgeAction{
		AgentName: "sdr", ActionType: "approval", Summary: "Email 2",
		Authority: "required", TargetHuman: "user-other", Status: "pending",
	})

	actions, err := store.ListPendingBridgeActions(ctx, "user-matt", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(actions) != 2 {
		t.Errorf("got %d actions, want 2", len(actions))
	}
}

func TestDecideBridgeAction(t *testing.T) {
	db, store := testDB(t)
	cleanBridgeTables(t, db)
	ctx := context.Background()

	action, _ := store.CreateBridgeAction(ctx, BridgeAction{
		AgentName: "sdr", ActionType: "approval", Summary: "Email",
		Authority: "required", TargetHuman: "user-matt", Status: "pending",
	})

	err := store.DecideBridgeAction(ctx, action.ID, "approved", "user-matt", "looks good")
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	got, _ := store.GetBridgeAction(ctx, action.ID)
	if got.Status != "approved" {
		t.Errorf("status = %q, want approved", got.Status)
	}
	if got.DecidedBy != "user-matt" {
		t.Errorf("decided_by = %q, want user-matt", got.DecidedBy)
	}
}

func TestBridgeNotifyPreferences(t *testing.T) {
	db, store := testDB(t)
	cleanBridgeTables(t, db)
	ctx := context.Background()

	err := store.SetBridgeNotifyPreference(ctx, "user-matt", "email", true)
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	err = store.SetBridgeNotifyPreference(ctx, "user-matt", "teams", true)
	if err != nil {
		t.Fatalf("set teams: %v", err)
	}

	prefs, err := store.GetBridgeNotifyPreferences(ctx, "user-matt")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(prefs) != 2 {
		t.Errorf("got %d prefs, want 2", len(prefs))
	}
}
