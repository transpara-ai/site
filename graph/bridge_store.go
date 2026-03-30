package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// BridgeAction represents a pending (or decided) human action.
type BridgeAction struct {
	ID            string          `json:"id"`
	AgentName     string          `json:"agent_name"`
	ActionType    string          `json:"action_type"`    // "approval", "handoff", "escalation"
	Summary       string          `json:"summary"`
	Authority     string          `json:"authority"`      // "required", "recommended", "notification"
	TargetHuman   string          `json:"target_human"`   // user ID
	Status        string          `json:"status"`         // "pending", "approved", "rejected", "auto_approved", "timeout"
	DecidedBy     string          `json:"decided_by"`
	DecidedAt     *time.Time      `json:"decided_at,omitempty"`
	DecisionNotes string          `json:"decision_notes"`
	DomainData    json.RawMessage `json:"domain_data"`
	CreatedAt     time.Time       `json:"created_at"`
}

// BridgeEvent is a membrane event ingested via webhook.
type BridgeEvent struct {
	ID        string          `json:"id"`
	AgentName string          `json:"agent_name"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// NotifyPreference stores per-user notification channel preferences.
type NotifyPreference struct {
	UserID  string          `json:"user_id"`
	Channel string          `json:"channel"` // "email", "teams"
	Enabled bool            `json:"enabled"`
	Config  json.RawMessage `json:"config"` // channel-specific: {"webhook_url":"..."} or {"email":"..."}
}

// CreateBridgeAction inserts a new pending action.
func (s *Store) CreateBridgeAction(ctx context.Context, a BridgeAction) (*BridgeAction, error) {
	a.ID = newID()
	if a.DomainData == nil {
		a.DomainData = json.RawMessage(`{}`)
	}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO bridge_actions (id, agent_name, action_type, summary, authority, target_human, status, domain_data)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING created_at`,
		a.ID, a.AgentName, a.ActionType, a.Summary, a.Authority, a.TargetHuman, a.Status, a.DomainData,
	).Scan(&a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create bridge action: %w", err)
	}
	return &a, nil
}

// GetBridgeAction retrieves a single action by ID.
func (s *Store) GetBridgeAction(ctx context.Context, id string) (*BridgeAction, error) {
	var a BridgeAction
	err := s.db.QueryRowContext(ctx,
		`SELECT id, agent_name, action_type, summary, authority, target_human, status,
		        decided_by, decided_at, decision_notes, domain_data, created_at
		 FROM bridge_actions WHERE id = $1`, id,
	).Scan(&a.ID, &a.AgentName, &a.ActionType, &a.Summary, &a.Authority, &a.TargetHuman,
		&a.Status, &a.DecidedBy, &a.DecidedAt, &a.DecisionNotes, &a.DomainData, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get bridge action: %w", err)
	}
	return &a, nil
}

// ListPendingBridgeActions returns pending actions for a specific human.
func (s *Store) ListPendingBridgeActions(ctx context.Context, userID string, limit int) ([]BridgeAction, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_name, action_type, summary, authority, target_human, status,
		        decided_by, decided_at, decision_notes, domain_data, created_at
		 FROM bridge_actions
		 WHERE target_human = $1 AND status = 'pending'
		 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending bridge actions: %w", err)
	}
	defer rows.Close()

	var actions []BridgeAction
	for rows.Next() {
		var a BridgeAction
		if err := rows.Scan(&a.ID, &a.AgentName, &a.ActionType, &a.Summary, &a.Authority,
			&a.TargetHuman, &a.Status, &a.DecidedBy, &a.DecidedAt, &a.DecisionNotes,
			&a.DomainData, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan bridge action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

// ListRecentBridgeActions returns recent decided actions for a user (activity feed).
func (s *Store) ListRecentBridgeActions(ctx context.Context, userID string, limit int) ([]BridgeAction, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_name, action_type, summary, authority, target_human, status,
		        decided_by, decided_at, decision_notes, domain_data, created_at
		 FROM bridge_actions
		 WHERE target_human = $1 AND status != 'pending'
		 ORDER BY decided_at DESC NULLS LAST LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent bridge actions: %w", err)
	}
	defer rows.Close()

	var actions []BridgeAction
	for rows.Next() {
		var a BridgeAction
		if err := rows.Scan(&a.ID, &a.AgentName, &a.ActionType, &a.Summary, &a.Authority,
			&a.TargetHuman, &a.Status, &a.DecidedBy, &a.DecidedAt, &a.DecisionNotes,
			&a.DomainData, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan bridge action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

// DecideBridgeAction records a human decision on a pending action.
func (s *Store) DecideBridgeAction(ctx context.Context, actionID, decision, decidedBy, notes string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE bridge_actions
		 SET status = $1, decided_by = $2, decided_at = NOW(), decision_notes = $3
		 WHERE id = $4 AND status = 'pending'`,
		decision, decidedBy, notes, actionID)
	if err != nil {
		return fmt.Errorf("decide bridge action: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("action %s not found or already decided", actionID)
	}
	return nil
}

// ListUndecidedBridgeActions returns pending actions for a given agent (polled by hive).
func (s *Store) ListUndecidedBridgeActions(ctx context.Context, agentName string) ([]BridgeAction, error) {
	return s.listBridgeActionsByAgentAndStatus(ctx, agentName, "pending")
}

// ListDecidedBridgeActions returns recently decided actions for a given agent (polled by hive).
func (s *Store) ListDecidedBridgeActions(ctx context.Context, agentName string, limit int) ([]BridgeAction, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_name, action_type, summary, authority, target_human, status,
		        decided_by, decided_at, decision_notes, domain_data, created_at
		 FROM bridge_actions
		 WHERE agent_name = $1 AND status != 'pending'
		 ORDER BY decided_at DESC NULLS LAST LIMIT $2`, agentName, limit)
	if err != nil {
		return nil, fmt.Errorf("list decided bridge actions: %w", err)
	}
	defer rows.Close()

	var actions []BridgeAction
	for rows.Next() {
		var a BridgeAction
		if err := rows.Scan(&a.ID, &a.AgentName, &a.ActionType, &a.Summary, &a.Authority,
			&a.TargetHuman, &a.Status, &a.DecidedBy, &a.DecidedAt, &a.DecisionNotes,
			&a.DomainData, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan bridge action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (s *Store) listBridgeActionsByAgentAndStatus(ctx context.Context, agentName, status string) ([]BridgeAction, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_name, action_type, summary, authority, target_human, status,
		        decided_by, decided_at, decision_notes, domain_data, created_at
		 FROM bridge_actions
		 WHERE agent_name = $1 AND status = $2
		 ORDER BY created_at ASC`, agentName, status)
	if err != nil {
		return nil, fmt.Errorf("list bridge actions: %w", err)
	}
	defer rows.Close()

	var actions []BridgeAction
	for rows.Next() {
		var a BridgeAction
		if err := rows.Scan(&a.ID, &a.AgentName, &a.ActionType, &a.Summary, &a.Authority,
			&a.TargetHuman, &a.Status, &a.DecidedBy, &a.DecidedAt, &a.DecisionNotes,
			&a.DomainData, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan bridge action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

// AppendBridgeEvent stores a membrane event received via webhook.
func (s *Store) AppendBridgeEvent(ctx context.Context, agentName, eventType string, payload []byte) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO bridge_events (id, agent_name, event_type, payload)
		 VALUES ($1, $2, $3, $4)`,
		newID(), agentName, eventType, payload)
	return err
}

// ListBridgeEvents returns recent events for an agent.
func (s *Store) ListBridgeEvents(ctx context.Context, agentName string, limit int) ([]BridgeEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_name, event_type, payload, created_at
		 FROM bridge_events
		 WHERE agent_name = $1
		 ORDER BY created_at DESC LIMIT $2`, agentName, limit)
	if err != nil {
		return nil, fmt.Errorf("list bridge events: %w", err)
	}
	defer rows.Close()

	var events []BridgeEvent
	for rows.Next() {
		var e BridgeEvent
		if err := rows.Scan(&e.ID, &e.AgentName, &e.EventType, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan bridge event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// SetBridgeNotifyPreference upserts a notification channel preference.
func (s *Store) SetBridgeNotifyPreference(ctx context.Context, userID, channel string, enabled bool) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO bridge_notify_preferences (user_id, channel, enabled, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (user_id, channel)
		 DO UPDATE SET enabled = $3, updated_at = NOW()`,
		userID, channel, enabled)
	return err
}

// GetBridgeNotifyPreferences returns all notification preferences for a user.
func (s *Store) GetBridgeNotifyPreferences(ctx context.Context, userID string) ([]NotifyPreference, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT user_id, channel, enabled, config
		 FROM bridge_notify_preferences
		 WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("get bridge notify preferences: %w", err)
	}
	defer rows.Close()

	var prefs []NotifyPreference
	for rows.Next() {
		var p NotifyPreference
		if err := rows.Scan(&p.UserID, &p.Channel, &p.Enabled, &p.Config); err != nil {
			return nil, fmt.Errorf("scan notify preference: %w", err)
		}
		prefs = append(prefs, p)
	}
	return prefs, rows.Err()
}

// ListAllBridgeActions returns recent actions for an agent regardless of status.
func (s *Store) ListAllBridgeActions(ctx context.Context, agentName string, limit int) ([]BridgeAction, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, agent_name, action_type, summary, authority, target_human, status,
		        decided_by, decided_at, decision_notes, domain_data, created_at
		 FROM bridge_actions
		 WHERE agent_name = $1
		 ORDER BY created_at DESC LIMIT $2`, agentName, limit)
	if err != nil {
		return nil, fmt.Errorf("list all bridge actions: %w", err)
	}
	defer rows.Close()

	var actions []BridgeAction
	for rows.Next() {
		var a BridgeAction
		if err := rows.Scan(&a.ID, &a.AgentName, &a.ActionType, &a.Summary, &a.Authority,
			&a.TargetHuman, &a.Status, &a.DecidedBy, &a.DecidedAt, &a.DecisionNotes,
			&a.DomainData, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan bridge action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

// ListBridgeAgentNames returns distinct agent names that have actions.
func (s *Store) ListBridgeAgentNames(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT agent_name FROM bridge_actions ORDER BY agent_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}
