package graph

import (
	"encoding/json"
	"io"
	"net/http"
)

// --- Webhook endpoints (called by hive membrane agents) ---

// handleBridgeActionWebhook receives a new pending action from a membrane agent.
// POST /api/bridge/action
func (h *Handlers) handleBridgeActionWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}

	var req struct {
		AgentName   string          `json:"agent_name"`
		ActionType  string          `json:"action_type"`
		Summary     string          `json:"summary"`
		Authority   string          `json:"authority"`
		TargetHuman string          `json:"target_human"`
		DomainData  json.RawMessage `json:"domain_data"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.AgentName == "" || req.TargetHuman == "" {
		http.Error(w, "agent_name and target_human required", http.StatusBadRequest)
		return
	}

	action, err := h.store.CreateBridgeAction(r.Context(), BridgeAction{
		AgentName:   req.AgentName,
		ActionType:  req.ActionType,
		Summary:     req.Summary,
		Authority:   req.Authority,
		TargetHuman: req.TargetHuman,
		Status:      "pending",
		DomainData:  req.DomainData,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok", "action_id": action.ID})
}

// handleBridgeEventWebhook receives a membrane event (poll results, mode changes, etc).
// POST /api/bridge/event
func (h *Handlers) handleBridgeEventWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}

	var req struct {
		AgentName string          `json:"agent_name"`
		EventType string          `json:"event_type"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.AgentName == "" || req.EventType == "" {
		http.Error(w, "agent_name and event_type required", http.StatusBadRequest)
		return
	}

	if err := h.store.AppendBridgeEvent(r.Context(), req.AgentName, req.EventType, body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

// handleBridgeDecisionsAPI returns decided actions for a given agent (polled by hive).
// GET /api/bridge/decisions?agent=sdr
func (h *Handlers) handleBridgeDecisionsAPI(w http.ResponseWriter, r *http.Request) {
	agent := r.URL.Query().Get("agent")
	if agent == "" {
		http.Error(w, "agent query parameter required", http.StatusBadRequest)
		return
	}

	actions, err := h.store.ListDecidedBridgeActions(r.Context(), agent, 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, actions)
}

// --- Human-facing endpoints (browser, requires auth) ---

// handleBridgeIndex renders the personal action queue.
// GET /bridge/
func (h *Handlers) handleBridgeIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := h.userID(r)

	pending, err := h.store.ListPendingBridgeActions(ctx, uid, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	recent, err := h.store.ListRecentBridgeActions(ctx, uid, 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	BridgeIndexPage(pending, recent, h.viewUser(r)).Render(ctx, w)
}

// handleBridgeActionDetail renders a single action for review.
// GET /bridge/actions/{id}
func (h *Handlers) handleBridgeActionDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	action, err := h.store.GetBridgeAction(r.Context(), id)
	if err != nil {
		http.Error(w, "action not found", http.StatusNotFound)
		return
	}

	BridgeActionDetailPage(action, h.viewUser(r)).Render(r.Context(), w)
}

// handleBridgeDecide records a human decision.
// POST /bridge/actions/{id}/decide
func (h *Handlers) handleBridgeDecide(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	decision := r.FormValue("decision")
	notes := r.FormValue("notes")
	uid := h.userID(r)

	switch decision {
	case "approved", "rejected":
	default:
		http.Error(w, "decision must be 'approved' or 'rejected'", http.StatusBadRequest)
		return
	}

	if err := h.store.DecideBridgeAction(r.Context(), id, decision, uid, notes); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Redirect back to bridge index
	http.Redirect(w, r, "/bridge/", http.StatusSeeOther)
}

// handleBridgeAgents lists all membrane agents with their recent activity.
// GET /bridge/agents
func (h *Handlers) handleBridgeAgents(w http.ResponseWriter, r *http.Request) {
	// For now, query distinct agent names from bridge_actions
	ctx := r.Context()
	agents, err := h.store.ListBridgeAgentNames(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	BridgeAgentsPage(agents, h.viewUser(r)).Render(ctx, w)
}

// handleBridgeAgentDetail shows activity for a specific membrane agent.
// GET /bridge/agents/{name}
func (h *Handlers) handleBridgeAgentDetail(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ctx := r.Context()

	events, err := h.store.ListBridgeEvents(ctx, name, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	BridgeAgentDetailPage(name, events, h.viewUser(r)).Render(ctx, w)
}

// handleBridgeAgentDomain renders a domain-specific view for a membrane agent.
// GET /bridge/agents/{name}/domain
func (h *Handlers) handleBridgeAgentDomain(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ctx := r.Context()

	actions, err := h.store.ListAllBridgeActions(ctx, name, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	events, err := h.store.ListBridgeEvents(ctx, name, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	BridgeAgentDomainPage(name, actions, events, h.viewUser(r)).Render(ctx, w)
}

// handleBridgePreferences renders notification preference settings.
// GET /bridge/preferences
func (h *Handlers) handleBridgePreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := h.userID(r)

	prefs, err := h.store.GetBridgeNotifyPreferences(ctx, uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	BridgePreferencesPage(prefs, h.viewUser(r)).Render(ctx, w)
}

// handleBridgeSavePreferences saves notification preferences.
// POST /bridge/preferences
func (h *Handlers) handleBridgeSavePreferences(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	uid := h.userID(r)
	ctx := r.Context()

	emailEnabled := r.FormValue("email") == "on"
	teamsEnabled := r.FormValue("teams") == "on"

	if err := h.store.SetBridgeNotifyPreference(ctx, uid, "email", emailEnabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.store.SetBridgeNotifyPreference(ctx, uid, "teams", teamsEnabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/bridge/preferences", http.StatusSeeOther)
}

// handleBridgeFeed returns the HTMX partial for live action feed polling.
// GET /bridge/feed
func (h *Handlers) handleBridgeFeed(w http.ResponseWriter, r *http.Request) {
	uid := h.userID(r)
	pending, err := h.store.ListPendingBridgeActions(r.Context(), uid, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	BridgeActionFeed(pending).Render(r.Context(), w)
}
