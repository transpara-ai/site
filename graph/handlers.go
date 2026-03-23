package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/lovyou-ai/site/auth"
)

// ViewUser holds user info for templates.
type ViewUser struct {
	Name    string
	Picture string
}

// ViewAPIKey holds API key info for templates.
type ViewAPIKey struct {
	ID        string
	Name      string
	AgentName string
	CreatedAt string
}

// Handlers serves the unified product HTTP endpoints.
type Handlers struct {
	store     *Store
	mind      *Mind // optional — triggers auto-reply on conversation messages
	readWrap  func(http.HandlerFunc) http.Handler // optional auth (reads)
	writeWrap func(http.HandlerFunc) http.Handler // required auth (writes)
}

// NewHandlers creates handlers with auth middleware.
// readWrap allows anonymous access (for public spaces), writeWrap requires auth.
func NewHandlers(store *Store, readWrap, writeWrap func(http.HandlerFunc) http.Handler) *Handlers {
	noop := func(hf http.HandlerFunc) http.Handler { return hf }
	if readWrap == nil {
		readWrap = noop
	}
	if writeWrap == nil {
		writeWrap = noop
	}
	return &Handlers{store: store, readWrap: readWrap, writeWrap: writeWrap}
}

// SetMind enables auto-reply on conversation messages.
func (h *Handlers) SetMind(m *Mind) { h.mind = m }

// Register adds all /app routes to the mux.
func (h *Handlers) Register(mux *http.ServeMux) {
	// Space management (requires auth).
	mux.Handle("GET /app", h.writeWrap(h.handleSpaceIndex))
	mux.Handle("GET /app/notifications", h.writeWrap(h.handleNotifications))
	mux.Handle("POST /app/new", h.writeWrap(h.handleCreateSpace))

	// Space settings (requires auth, owner only).
	mux.Handle("GET /app/{slug}/settings", h.writeWrap(h.handleSpaceSettings))
	mux.Handle("POST /app/{slug}/settings", h.writeWrap(h.handleUpdateSpace))
	mux.Handle("POST /app/{slug}/delete", h.writeWrap(h.handleDeleteSpace))

	// Space invites.
	mux.Handle("POST /app/{slug}/invite", h.writeWrap(h.handleCreateInvite))
	mux.Handle("GET /join/{token}", h.writeWrap(h.handleJoinViaInvite))

	// Space lenses (optional auth — public spaces readable by anyone).
	mux.Handle("GET /app/{slug}", h.readWrap(h.handleSpaceDefault))
	mux.Handle("GET /app/{slug}/board", h.readWrap(h.handleBoard))
	mux.Handle("GET /app/{slug}/feed", h.readWrap(h.handleFeed))
	mux.Handle("GET /app/{slug}/threads", h.readWrap(h.handleThreads))
	mux.Handle("GET /app/{slug}/conversations", h.readWrap(h.handleConversations))
	mux.Handle("GET /app/{slug}/people", h.readWrap(h.handlePeople))
	mux.Handle("GET /app/{slug}/activity", h.readWrap(h.handleActivity))
	mux.Handle("GET /app/{slug}/knowledge", h.readWrap(h.handleKnowledge))
	mux.Handle("GET /app/{slug}/governance", h.readWrap(h.handleGovernance))
	mux.Handle("GET /app/{slug}/changelog", h.readWrap(h.handleChangelog))

	// Conversation detail (optional auth).
	mux.Handle("GET /app/{slug}/conversation/{id}", h.readWrap(h.handleConversationDetail))
	mux.Handle("GET /app/{slug}/conversation/{id}/messages", h.readWrap(h.handleConversationMessages))

	// Node detail (optional auth — public spaces readable by anyone).
	mux.Handle("GET /app/{slug}/node/{id}", h.readWrap(h.handleNodeDetail))

	// Grammar operations (requires auth).
	mux.Handle("POST /app/{slug}/op", h.writeWrap(h.handleOp))

	// Mind state (requires auth — used by cmd/post to sync loop state).
	mux.Handle("PUT /api/mind-state", h.writeWrap(h.handleSetMindState))

	// Node children (for HTMX polling).
	mux.Handle("GET /app/{slug}/node/{id}/children", h.readWrap(h.handleNodeChildren))

	// Node mutations (requires auth).
	mux.Handle("POST /app/{slug}/node/{id}/state", h.writeWrap(h.handleNodeState))
	mux.Handle("POST /app/{slug}/node/{id}/update", h.writeWrap(h.handleNodeUpdate))
	mux.Handle("DELETE /app/{slug}/node/{id}", h.writeWrap(h.handleNodeDelete))
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) viewUser(r *http.Request) ViewUser {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return ViewUser{Name: "Anonymous"}
	}
	return ViewUser{Name: u.Name, Picture: u.Picture}
}

func (h *Handlers) userID(r *http.Request) string {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return "anonymous"
	}
	return u.ID
}

func (h *Handlers) userName(r *http.Request) string {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return "anonymous"
	}
	return u.Name
}

func (h *Handlers) userKind(r *http.Request) string {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return "human"
	}
	return u.Kind
}

// notify creates a notification for a user (skips if targetID is the actor).
func (h *Handlers) notify(ctx context.Context, targetID, actorName, opID, spaceID, message string) {
	if targetID == "" || targetID == "anonymous" {
		return
	}
	h.store.CreateNotification(ctx, targetID, opID, spaceID, actorName+": "+message)
}

// spaceFromRequest returns a space for write operations.
// Owners can always write. Authenticated users can write to public spaces.
func (h *Handlers) spaceFromRequest(r *http.Request) (*Space, error) {
	slug := r.PathValue("slug")
	space, err := h.store.GetSpaceBySlug(r.Context(), slug)
	if err != nil {
		return nil, err
	}
	uid := h.userID(r)
	if space.OwnerID == uid {
		return space, nil
	}
	if space.Visibility == VisibilityPublic && uid != "anonymous" {
		return space, nil
	}
	return nil, ErrNotFound
}

// spaceOwnerOnly returns a space only if the current user owns it.
func (h *Handlers) spaceOwnerOnly(r *http.Request) (*Space, error) {
	slug := r.PathValue("slug")
	space, err := h.store.GetSpaceBySlug(r.Context(), slug)
	if err != nil {
		return nil, err
	}
	if space.OwnerID != h.userID(r) {
		return nil, ErrNotFound
	}
	return space, nil
}

// spaceForRead returns a space if the user owns it OR it's public (for reads).
func (h *Handlers) spaceForRead(r *http.Request) (*Space, bool, error) {
	slug := r.PathValue("slug")
	space, err := h.store.GetSpaceBySlug(r.Context(), slug)
	if err != nil {
		return nil, false, err
	}
	isOwner := space.OwnerID == h.userID(r)
	if !isOwner && space.Visibility != VisibilityPublic {
		return nil, false, ErrNotFound
	}
	return space, isOwner, nil
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// populateFormFromJSON parses a JSON request body into r.Form
// so r.FormValue() works for both form-encoded and JSON requests.
func populateFormFromJSON(r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return
	}
	var m map[string]string
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		return
	}
	if r.Form == nil {
		r.Form = url.Values{}
	}
	for k, v := range m {
		r.Form.Set(k, v)
	}
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "space"
	}
	return s
}

// ────────────────────────────────────────────────────────────────────
// Space handlers
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleSpaceIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := h.userID(r)

	spaces, err := h.store.ListSpaces(ctx, uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"spaces": spaces})
		return
	}

	if len(spaces) == 0 {
		// New users: redirect to discover so they see existing spaces first.
		http.Redirect(w, r, "/discover", http.StatusSeeOther)
		return
	}

	tasks, _ := h.store.ListUserTasks(ctx, uid, 10)
	convos, _ := h.store.ListUserConversations(ctx, uid, 5)
	agentOps, _ := h.store.ListUserAgentActivity(ctx, uid, 10)
	agents, _ := h.store.ListAgentNames(ctx)

	defaultSlug := ""
	if len(spaces) > 0 {
		defaultSlug = spaces[0].Slug
	}

	unread := h.store.UnreadCount(ctx, uid)
	Dashboard(spaces, tasks, convos, agentOps, h.viewUser(r), defaultSlug, agents, unread).Render(ctx, w)
}

func (h *Handlers) handleNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := h.userID(r)

	notifs, _ := h.store.ListNotifications(ctx, uid, 50)
	h.store.MarkNotificationsRead(ctx, uid)

	NotificationsView(notifs, h.viewUser(r)).Render(ctx, w)
}

func (h *Handlers) handleCreateSpace(w http.ResponseWriter, r *http.Request) {
	populateFormFromJSON(r)
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))
	kind := r.FormValue("kind")
	if kind == "" {
		kind = SpaceProject
	}
	visibility := r.FormValue("visibility")
	if visibility != VisibilityPublic {
		visibility = VisibilityPrivate
	}

	slug := slugify(name)
	// Ensure unique slug by appending random suffix on conflict.
	space, err := h.store.CreateSpace(r.Context(), slug, name, description, h.userID(r), kind, visibility)
	if err != nil {
		// Try with random suffix.
		slug = slug + "-" + newID()[:6]
		space, err = h.store.CreateSpace(r.Context(), slug, name, description, h.userID(r), kind, visibility)
		if err != nil {
			log.Printf("graph: create space: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusCreated, map[string]any{"space": space})
		return
	}

	http.Redirect(w, r, "/app/"+space.Slug, http.StatusSeeOther)
}

func (h *Handlers) handleSpaceSettings(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceOwnerOnly(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))
	reports, _ := h.store.ListReports(r.Context(), space.ID)
	inviteToken := r.URL.Query().Get("invite")
	if inviteToken == "" {
		inviteToken = h.store.GetInviteToken(r.Context(), space.ID)
	}
	SettingsView(*space, spaces, reports, h.viewUser(r), "", inviteToken).Render(r.Context(), w)
}

func (h *Handlers) handleUpdateSpace(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceOwnerOnly(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))
		reports, _ := h.store.ListReports(r.Context(), space.ID)
		SettingsView(*space, spaces, reports, h.viewUser(r), "Name cannot be empty.", "").Render(r.Context(), w)
		return
	}

	description := strings.TrimSpace(r.FormValue("description"))
	visibility := r.FormValue("visibility")
	if visibility != VisibilityPublic {
		visibility = VisibilityPrivate
	}

	if err := h.store.UpdateSpace(r.Context(), space.ID, name, description, visibility); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app/"+space.Slug+"/settings", http.StatusSeeOther)
}

func (h *Handlers) handleDeleteSpace(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceOwnerOnly(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	confirm := strings.TrimSpace(r.FormValue("confirm"))
	if confirm != space.Name {
		spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))
		reports, _ := h.store.ListReports(r.Context(), space.ID)
		SettingsView(*space, spaces, reports, h.viewUser(r), "Type the space name to confirm deletion.", "").Render(r.Context(), w)
		return
	}

	if err := h.store.DeleteSpace(r.Context(), space.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (h *Handlers) handleCreateInvite(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceOwnerOnly(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Reuse existing token if one exists.
	token := h.store.GetInviteToken(r.Context(), space.ID)
	if token == "" {
		token, err = h.store.CreateInvite(r.Context(), space.ID, h.userID(r))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]string{"token": token})
		return
	}
	http.Redirect(w, r, "/app/"+space.Slug+"/settings?invite="+token, http.StatusSeeOther)
}

func (h *Handlers) handleJoinViaInvite(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	spaceID := h.store.GetInviteSpaceID(r.Context(), token)
	if spaceID == "" {
		http.Error(w, "invalid or expired invite", http.StatusNotFound)
		return
	}

	uid := h.userID(r)
	uname := h.userName(r)
	h.store.JoinSpace(r.Context(), spaceID, uid, uname)
	h.store.RecordOp(r.Context(), spaceID, "", uname, uid, "join", nil)

	space, err := h.store.GetSpaceByID(r.Context(), spaceID)
	if err != nil {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/"+space.Slug, http.StatusSeeOther)
}

func (h *Handlers) handleSpaceDefault(w http.ResponseWriter, r *http.Request) {
	space, isOwner, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space})
		return
	}

	ctx := r.Context()
	uid := h.userID(r)

	spaces, _ := h.store.ListSpaces(ctx, uid)
	pinned, _ := h.store.ListPinnedNodes(ctx, space.ID)
	memberCount := h.store.MemberCount(ctx, space.ID)
	recentOps, _ := h.store.ListOps(ctx, space.ID, 5)

	// Count tasks by state.
	allTasks, _ := h.store.ListNodes(ctx, ListNodesParams{SpaceID: space.ID, Kind: KindTask})
	openTasks, activeTasks, doneTasks := 0, 0, 0
	for _, t := range allTasks {
		switch t.State {
		case StateOpen:
			openTasks++
		case StateActive, StateReview:
			activeTasks++
		case StateDone:
			doneTasks++
		}
	}

	SpaceOverview(*space, spaces, pinned, recentOps, h.viewUser(r), isOwner,
		memberCount, openTasks, activeTasks, doneTasks).Render(ctx, w)
}

// ────────────────────────────────────────────────────────────────────
// Lens handlers
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleBoard(w http.ResponseWriter, r *http.Request) {
	space, isOwner, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))

	tasks, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		ParentID: "root",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply filters from query params.
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	assigneeFilter := strings.TrimSpace(r.URL.Query().Get("assignee"))
	if q != "" || assigneeFilter != "" {
		var filtered []Node
		for _, t := range tasks {
			if q != "" && !strings.Contains(strings.ToLower(t.Title), q) && !strings.Contains(strings.ToLower(t.Body), q) {
				continue
			}
			if assigneeFilter != "" && t.Assignee != assigneeFilter {
				continue
			}
			filtered = append(filtered, t)
		}
		tasks = filtered
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "nodes": tasks})
		return
	}

	columns := groupByState(tasks)
	agents, _ := h.store.ListAgentNames(r.Context())
	BoardView(*space, spaces, columns, h.viewUser(r), isOwner, agents, q, assigneeFilter).Render(r.Context(), w)
}

func (h *Handlers) handleFeed(w http.ResponseWriter, r *http.Request) {
	space, isOwner, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))

	posts, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindPost,
		ParentID: "root",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "nodes": posts})
		return
	}

	agents, _ := h.store.ListAgentNames(r.Context())
	FeedView(*space, spaces, posts, h.viewUser(r), isOwner, len(agents) > 0).Render(r.Context(), w)
}

func (h *Handlers) handleThreads(w http.ResponseWriter, r *http.Request) {
	space, isOwner, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))

	threads, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindThread,
		ParentID: "root",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "nodes": threads})
		return
	}

	ThreadsView(*space, spaces, threads, h.viewUser(r), isOwner).Render(r.Context(), w)
}

func (h *Handlers) handleConversations(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))

	convos, err := h.store.ListConversations(r.Context(), space.ID, h.userID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "conversations": convos, "me": h.userName(r)})
		return
	}

	agents, _ := h.store.ListAgentNames(r.Context())
	// Collect all tag IDs and resolve to display names.
	var allIDs []string
	for _, c := range convos {
		allIDs = append(allIDs, c.Tags...)
	}
	nameMap := h.store.ResolveUserNames(r.Context(), allIDs)
	ConversationsView(*space, spaces, convos, h.viewUser(r), agents, nameMap).Render(r.Context(), w)
}

func (h *Handlers) handlePeople(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))

	ops, err := h.store.ListOps(r.Context(), space.ID, 1000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Aggregate by actor.
	memberMap := map[string]*Member{}
	for _, o := range ops {
		m, ok := memberMap[o.Actor]
		if !ok {
			m = &Member{Name: o.Actor, Kind: o.ActorKind}
			memberMap[o.Actor] = m
		}
		m.OpCount++
		m.LastSeen = o.CreatedAt.Format("Jan 2")
	}
	var members []Member
	for _, m := range memberMap {
		members = append(members, *m)
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "members": members})
		return
	}

	PeopleView(*space, spaces, members, h.viewUser(r)).Render(r.Context(), w)
}

func (h *Handlers) handleActivity(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))

	ops, err := h.store.ListOps(r.Context(), space.ID, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "ops": ops})
		return
	}

	ActivityView(*space, spaces, ops, h.viewUser(r)).Render(r.Context(), w)
}

func (h *Handlers) handleKnowledge(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))

	claims, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindClaim,
		ParentID: "root",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Count challenges per claim.
	challengeCounts := make(map[string]int, len(claims))
	for _, c := range claims {
		challengeCounts[c.ID] = h.store.CountChallenges(r.Context(), c.ID)
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "claims": claims})
		return
	}

	KnowledgeView(*space, spaces, claims, challengeCounts, h.viewUser(r)).Render(r.Context(), w)
}

func (h *Handlers) handleChangelog(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))
	entries, err := h.store.ListChangelog(r.Context(), space.ID, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "changelog": entries})
		return
	}

	ChangelogView(*space, spaces, entries, h.viewUser(r)).Render(r.Context(), w)
}

func (h *Handlers) handleGovernance(w http.ResponseWriter, r *http.Request) {
	space, isOwner, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))
	proposals, err := h.store.ListProposals(r.Context(), space.ID, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "proposals": proposals})
		return
	}

	GovernanceView(*space, spaces, proposals, h.viewUser(r), isOwner).Render(r.Context(), w)
}

// ────────────────────────────────────────────────────────────────────
// Node detail
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleConversationDetail(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeID := r.PathValue("id")
	node, err := h.store.GetNode(r.Context(), nodeID)
	if errors.Is(err, ErrNotFound) || (err == nil && node.Kind != KindConversation) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	messages, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		ParentID: nodeID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "conversation": node, "messages": messages})
		return
	}

	hasAgent, _ := h.store.HasAgentParticipant(r.Context(), node.Tags)

	user := h.viewUser(r)
	nameMap := h.store.ResolveUserNames(r.Context(), node.Tags)
	ConversationDetailView(*space, *node, messages, user, h.userID(r), hasAgent, nameMap).Render(r.Context(), w)
}

// handleConversationMessages returns new messages since the given timestamp.
// Used by HTMX polling: GET /app/{slug}/conversation/{id}/messages?after=RFC3339
func (h *Handlers) handleConversationMessages(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeID := r.PathValue("id")
	afterStr := r.URL.Query().Get("after")
	if afterStr == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	after, err := time.Parse(time.RFC3339Nano, afterStr)
	if err != nil {
		http.Error(w, "invalid after timestamp", http.StatusBadRequest)
		return
	}

	messages, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		ParentID: nodeID,
		After:    &after,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"messages": messages})
		return
	}

	// Return chatMessage fragments for HTMX polling.
	uid := h.userID(r)
	for _, msg := range messages {
		chatMessage(msg, uid).Render(r.Context(), w)
	}
}

func (h *Handlers) handleNodeDetail(w http.ResponseWriter, r *http.Request) {
	space, isOwner, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeID := r.PathValue("id")
	node, err := h.store.GetNode(r.Context(), nodeID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	children, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		ParentID: nodeID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ops, err := h.store.ListNodeOps(r.Context(), nodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "node": node, "children": children, "ops": ops})
		return
	}

	NodeDetailView(*space, *node, children, ops, h.viewUser(r), isOwner).Render(r.Context(), w)
}

// ────────────────────────────────────────────────────────────────────
// Grammar operation dispatcher
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleOp(w http.ResponseWriter, r *http.Request) {
	populateFormFromJSON(r)

	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	op := r.FormValue("op")
	ctx := r.Context()
	actor := h.userName(r)
	actorID := h.userID(r)
	actorKind := h.userKind(r)

	switch op {
	case "intend":
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		assigneeName := strings.TrimSpace(r.FormValue("assignee"))
		assigneeID := ""
		if assigneeName != "" {
			assigneeID = h.store.ResolveUserID(ctx, assigneeName)
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindTask,
			Title:      title,
			Body:       strings.TrimSpace(r.FormValue("description")),
			Priority:   r.FormValue("priority"),
			Assignee:   assigneeName,
			AssigneeID: assigneeID,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		op, _ := h.store.RecordOp(ctx, space.ID, node.ID, actor, actorID, "intend", nil)

		// Notify assignee if task was created with one.
		if assigneeID != "" && assigneeID != actorID && op != nil {
			h.notify(ctx, assigneeID, actor, op.ID, space.ID, "created a task for you: "+node.Title)
		}

		// Trigger Mind if task was created with an agent assignee.
		if h.mind != nil && assigneeID != "" {
			go h.mind.OnTaskAssigned(space.ID, space.Slug, node, assigneeID)
		}

		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "intend"})
			return
		}
		if isHTMX(r) {
			TaskCard(*node, space.Slug).Render(ctx, w)
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/board", http.StatusSeeOther)

	case "decompose":
		title := strings.TrimSpace(r.FormValue("title"))
		parentID := r.FormValue("parent_id")
		if title == "" || parentID == "" {
			http.Error(w, "title and parent_id required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			ParentID:   parentID,
			Kind:       KindTask,
			Title:      title,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		op, _ := h.store.RecordOp(ctx, space.ID, node.ID, actor, actorID, "decompose", nil)

		// Notify parent task author when an agent decomposes their task.
		if actorKind == "agent" && op != nil {
			if parent, _ := h.store.GetNode(ctx, parentID); parent != nil && parent.AuthorID != actorID {
				h.notify(ctx, parent.AuthorID, actor, op.ID, space.ID, "broke down your task: "+parent.Title)
			}
		}

		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "decompose"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, parentID), http.StatusSeeOther)

	case "express":
		title := strings.TrimSpace(r.FormValue("title"))
		body := strings.TrimSpace(r.FormValue("body"))
		if title == "" && body == "" {
			http.Error(w, "title or body required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindPost,
			Title:      title,
			Body:       body,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, actorID, "express", nil)

		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "express"})
			return
		}
		if isHTMX(r) {
			FeedCard(*node, space.Slug).Render(ctx, w)
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/feed", http.StatusSeeOther)

	case "discuss":
		title := strings.TrimSpace(r.FormValue("title"))
		body := strings.TrimSpace(r.FormValue("body"))
		if title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindThread,
			Title:      title,
			Body:       body,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, actorID, "discuss", nil)

		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "discuss"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, node.ID), http.StatusSeeOther)

	case "respond":
		parentID := r.FormValue("parent_id")
		body := strings.TrimSpace(r.FormValue("body"))
		if parentID == "" || body == "" {
			http.Error(w, "parent_id and body required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			ParentID:   parentID,
			Kind:       KindComment,
			Body:       body,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		op, _ := h.store.RecordOp(ctx, space.ID, node.ID, actor, actorID, "respond", nil)

		// Notify conversation participants (except the sender).
		if op != nil {
			if parent, _ := h.store.GetNode(ctx, parentID); parent != nil && parent.Kind == KindConversation {
				for _, tagID := range parent.Tags {
					if tagID != actorID {
						h.notify(ctx, tagID, actor, op.ID, space.ID, "sent a message")
					}
				}
			}
		}

		// Trigger Mind auto-reply if a non-agent messaged in a conversation.
		if h.mind != nil && actorKind != "agent" {
			if parent, _ := h.store.GetNode(ctx, parentID); parent != nil && parent.Kind == KindConversation {
				go h.mind.OnMessage(space.ID, space.Slug, parent, actorID)
			}
		}

		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "respond"})
			return
		}
		if isHTMX(r) {
			parent, _ := h.store.GetNode(ctx, parentID)
			if parent != nil && parent.Kind == KindConversation {
				chatMessage(*node, actorID).Render(ctx, w)
			} else {
				CommentItem(*node).Render(ctx, w)
			}
			return
		}
		if parent, _ := h.store.GetNode(ctx, parentID); parent != nil && parent.Kind == KindConversation {
			http.Redirect(w, r, fmt.Sprintf("/app/%s/conversation/%s", space.Slug, parentID), http.StatusSeeOther)
		} else {
			http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, parentID), http.StatusSeeOther)
		}

	case "complete":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateNodeState(ctx, nodeID, StateDone); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		op, _ := h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "complete", nil)

		// Notify task author when an agent completes their task.
		if actorKind == "agent" && op != nil {
			if node, _ := h.store.GetNode(ctx, nodeID); node != nil && node.AuthorID != actorID {
				h.notify(ctx, node.AuthorID, actor, op.ID, space.ID, "completed your task: "+node.Title)
			}
		}

		if wantsJSON(r) {
			node, _ := h.store.GetNode(ctx, nodeID)
			writeJSON(w, http.StatusOK, map[string]any{"node": node, "op": "complete"})
			return
		}

		node, _ := h.store.GetNode(ctx, nodeID)
		if isHTMX(r) && node != nil {
			TaskCard(*node, space.Slug).Render(ctx, w)
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/board", http.StatusSeeOther)

	case "assign":
		nodeID := r.FormValue("node_id")
		assignee := strings.TrimSpace(r.FormValue("assignee"))
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		// Resolve assignee name to ID.
		assigneeID := h.store.ResolveUserID(ctx, assignee)
		if err := h.store.UpdateNode(ctx, nodeID, nil, nil, nil, &assignee, &assigneeID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		op, _ := h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "assign", nil)

		// Notify assignee.
		if assigneeID != actorID && op != nil {
			h.notify(ctx, assigneeID, actor, op.ID, space.ID, "assigned you a task")
		}

		// Trigger Mind if task was assigned to an agent.
		if h.mind != nil && assigneeID != "" {
			if node, _ := h.store.GetNode(ctx, nodeID); node != nil {
				go h.mind.OnTaskAssigned(space.ID, space.Slug, node, assigneeID)
			}
		}

		if wantsJSON(r) {
			node, _ := h.store.GetNode(ctx, nodeID)
			writeJSON(w, http.StatusOK, map[string]any{"node": node, "op": "assign"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "claim":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateNode(ctx, nodeID, nil, nil, nil, &actor, &actorID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "claim", nil)

		// Trigger Mind if an agent claimed the task.
		if h.mind != nil && actorKind == "agent" {
			if node, _ := h.store.GetNode(ctx, nodeID); node != nil {
				go h.mind.OnTaskAssigned(space.ID, space.Slug, node, actorID)
			}
		}

		if wantsJSON(r) {
			node, _ := h.store.GetNode(ctx, nodeID)
			writeJSON(w, http.StatusOK, map[string]any{"node": node, "op": "claim"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "prioritize":
		nodeID := r.FormValue("node_id")
		priority := r.FormValue("priority")
		if nodeID == "" || priority == "" {
			http.Error(w, "node_id and priority required", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateNode(ctx, nodeID, nil, nil, &priority, nil, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "prioritize", nil)

		if wantsJSON(r) {
			node, _ := h.store.GetNode(ctx, nodeID)
			writeJSON(w, http.StatusOK, map[string]any{"node": node, "op": "prioritize"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "converse":
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		// Participants: store ONLY user IDs in tags.
		// Resolve participant names to IDs via lookup.
		participants := []string{actorID}
		if p := strings.TrimSpace(r.FormValue("participants")); p != "" {
			for _, name := range strings.Split(p, ",") {
				name = strings.TrimSpace(name)
				if name == "" || name == actor || name == actorID {
					continue
				}
				// Try to resolve name to user ID.
				resolved := h.store.ResolveUserID(ctx, name)
				if resolved != "" {
					if resolved != actorID {
						participants = append(participants, resolved)
					}
				} else {
					// Fallback: store raw string for invited users who don't exist yet.
					participants = append(participants, name)
				}
			}
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindConversation,
			Title:      title,
			Body:       strings.TrimSpace(r.FormValue("body")),
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
			Tags:       participants,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, actorID, "converse", nil)

		// Trigger Mind if a human created a conversation with an agent.
		if h.mind != nil && actorKind != "agent" {
			go h.mind.OnMessage(space.ID, space.Slug, node, actorID)
		}

		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "converse"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/conversation/%s", space.Slug, node.ID), http.StatusSeeOther)

	case "join":
		if err := h.store.JoinSpace(ctx, space.ID, actorID, actor); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, "", actor, actorID, "join", nil)
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "join", "status": "joined"})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug, http.StatusSeeOther)

	case "leave":
		if err := h.store.LeaveSpace(ctx, space.ID, actorID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, "", actor, actorID, "leave", nil)
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "leave", "status": "left"})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug, http.StatusSeeOther)

	case "report":
		nodeID := r.FormValue("node_id")
		reason := strings.TrimSpace(r.FormValue("reason"))
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		if reason == "" {
			reason = "flagged by user"
		}
		payload, _ := json.Marshal(map[string]string{"reason": reason})
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "report", payload)

		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "report", "status": "recorded"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "resolve":
		nodeID := r.FormValue("node_id")
		action := r.FormValue("action") // "dismiss" or "remove"
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		// Only space owner can resolve reports.
		if space.OwnerID != actorID {
			http.Error(w, "only space owner can resolve reports", http.StatusForbidden)
			return
		}
		payload, _ := json.Marshal(map[string]string{"action": action})
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "resolve", payload)

		if action == "remove" {
			h.store.DeleteNode(ctx, nodeID)
		}

		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "resolve", "action": action})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/settings", http.StatusSeeOther)

	case "depend":
		nodeID := r.FormValue("node_id")
		dependsOn := r.FormValue("depends_on")
		if nodeID == "" || dependsOn == "" {
			http.Error(w, "node_id and depends_on required", http.StatusBadRequest)
			return
		}
		if err := h.store.AddDependency(ctx, nodeID, dependsOn); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "depend", nil)

		if wantsJSON(r) {
			node, _ := h.store.GetNode(ctx, nodeID)
			writeJSON(w, http.StatusOK, map[string]any{"node": node, "op": "depend"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "assert":
		title := strings.TrimSpace(r.FormValue("title"))
		body := strings.TrimSpace(r.FormValue("body"))
		if title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindClaim,
			Title:      title,
			Body:       body,
			State:      ClaimClaimed,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, actorID, "assert", nil)

		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "assert"})
			return
		}
		if isHTMX(r) {
			KnowledgeCard(*node, space.Slug, 0).Render(ctx, w)
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/knowledge", http.StatusSeeOther)

	case "challenge":
		nodeID := r.FormValue("node_id")
		reason := strings.TrimSpace(r.FormValue("reason"))
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		// Only claims can be challenged.
		node, err := h.store.GetNode(ctx, nodeID)
		if err != nil || node == nil {
			http.NotFound(w, r)
			return
		}
		if node.Kind != KindClaim {
			http.Error(w, "can only challenge claims", http.StatusBadRequest)
			return
		}
		if reason == "" {
			reason = "challenged"
		}
		payload, _ := json.Marshal(map[string]string{"reason": reason})
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "challenge", payload)

		// Update claim state to challenged.
		if err := h.store.UpdateNodeState(ctx, nodeID, ClaimChallenged); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "challenge", "status": "recorded"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "verify":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		node, err := h.store.GetNode(ctx, nodeID)
		if err != nil || node == nil {
			http.NotFound(w, r)
			return
		}
		if node.Kind != KindClaim {
			http.Error(w, "can only verify claims", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateNodeState(ctx, nodeID, ClaimVerified); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "verify", nil)
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "verify"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "retract":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		node, err := h.store.GetNode(ctx, nodeID)
		if err != nil || node == nil {
			http.NotFound(w, r)
			return
		}
		if node.Kind != KindClaim {
			http.Error(w, "can only retract claims", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateNodeState(ctx, nodeID, ClaimRetracted); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "retract", nil)
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "retract"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "reflect":
		body := strings.TrimSpace(r.FormValue("body"))
		if body == "" {
			http.Error(w, "body required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindPost,
			Title:      "Reflection",
			Body:       body,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, actorID, "reflect", nil)
		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "reflect"})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/feed", http.StatusSeeOther)

	case "pin":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		if err := h.store.SetPinned(ctx, nodeID, true); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "pin", nil)
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "pin"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "unpin":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		if err := h.store.SetPinned(ctx, nodeID, false); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "unpin", nil)
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "unpin"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "propose":
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindProposal,
			Title:      title,
			Body:       strings.TrimSpace(r.FormValue("description")),
			State:      ProposalOpen,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, actorID, "propose", nil)

		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "propose"})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/governance", http.StatusSeeOther)

	case "vote":
		nodeID := r.FormValue("node_id")
		vote := r.FormValue("vote") // "yes" or "no"
		if nodeID == "" || (vote != "yes" && vote != "no") {
			http.Error(w, "node_id and vote (yes/no) required", http.StatusBadRequest)
			return
		}
		// Only proposals can be voted on.
		node, err := h.store.GetNode(ctx, nodeID)
		if err != nil || node == nil {
			http.NotFound(w, r)
			return
		}
		if node.Kind != KindProposal {
			http.Error(w, "can only vote on proposals", http.StatusBadRequest)
			return
		}
		if node.State != ProposalOpen {
			http.Error(w, "proposal is no longer open", http.StatusBadRequest)
			return
		}
		// One vote per user per proposal.
		if h.store.HasVoted(ctx, nodeID, actorID) {
			http.Error(w, "already voted", http.StatusConflict)
			return
		}
		payload, _ := json.Marshal(map[string]string{"vote": vote})
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "vote", payload)

		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "vote", "vote": vote})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/governance", http.StatusSeeOther)

	case "close_proposal":
		nodeID := r.FormValue("node_id")
		outcome := r.FormValue("outcome") // "passed" or "rejected"
		if nodeID == "" || (outcome != "passed" && outcome != "rejected") {
			http.Error(w, "node_id and outcome (passed/rejected) required", http.StatusBadRequest)
			return
		}
		// Only space owner can close proposals.
		if space.OwnerID != actorID {
			http.Error(w, "only space owner can close proposals", http.StatusForbidden)
			return
		}
		node, err := h.store.GetNode(ctx, nodeID)
		if err != nil || node == nil {
			http.NotFound(w, r)
			return
		}
		if node.Kind != KindProposal {
			http.Error(w, "can only close proposals", http.StatusBadRequest)
			return
		}
		newState := ProposalPassed
		if outcome == "rejected" {
			newState = ProposalFailed
		}
		if err := h.store.UpdateNodeState(ctx, nodeID, newState); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		payload, _ := json.Marshal(map[string]string{"outcome": outcome})
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "close_proposal", payload)

		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "close_proposal", "outcome": outcome})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/governance", http.StatusSeeOther)

	default:
		http.Error(w, fmt.Sprintf("unknown op: %s", op), http.StatusBadRequest)
	}
}

// ────────────────────────────────────────────────────────────────────
// Node mutation handlers (non-op convenience routes)
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) handleNodeState(w http.ResponseWriter, r *http.Request) {
	populateFormFromJSON(r)
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeID := r.PathValue("id")
	newState := r.FormValue("state")

	node, err := h.store.GetNode(r.Context(), nodeID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.store.UpdateNodeState(r.Context(), nodeID, newState); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Record as appropriate grammar op.
	opName := "progress"
	if newState == StateDone {
		opName = "complete"
	} else if newState == StateReview {
		opName = "review"
	}
	h.store.RecordOp(r.Context(), space.ID, nodeID, h.userName(r), h.userID(r), opName, nil)

	node, _ = h.store.GetNode(r.Context(), nodeID)
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"node": node, "op": opName})
		return
	}
	if isHTMX(r) && node != nil {
		TaskCard(*node, space.Slug).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/app/"+space.Slug+"/board", http.StatusSeeOther)
}

func (h *Handlers) handleNodeUpdate(w http.ResponseWriter, r *http.Request) {
	populateFormFromJSON(r)
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeID := r.PathValue("id")

	var title, body, priority, assignee *string
	if v := r.FormValue("title"); v != "" {
		title = &v
	}
	if r.Form != nil && r.Form.Has("body") {
		v := r.FormValue("body")
		body = &v
	}
	if v := r.FormValue("priority"); v != "" {
		priority = &v
	}
	var assigneeIDPtr *string
	if r.Form != nil && r.Form.Has("assignee") {
		v := r.FormValue("assignee")
		assignee = &v
		aid := h.store.ResolveUserID(r.Context(), v)
		assigneeIDPtr = &aid
	}

	if err := h.store.UpdateNode(r.Context(), nodeID, title, body, priority, assignee, assigneeIDPtr); err != nil {
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	node, _ := h.store.GetNode(r.Context(), nodeID)
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"node": node})
		return
	}
	if isHTMX(r) && node != nil {
		TaskCard(*node, space.Slug).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)
}

func (h *Handlers) handleNodeDelete(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeID := r.PathValue("id")
	if err := h.store.DeleteNode(r.Context(), nodeID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"deleted": nodeID})
		return
	}
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/app/"+space.Slug+"/board")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/app/"+space.Slug+"/board", http.StatusSeeOther)
}

// ────────────────────────────────────────────────────────────────────
// Board helpers
// ────────────────────────────────────────────────────────────────────

// BoardColumn holds nodes grouped by state for the kanban board.
type BoardColumn struct {
	State string
	Label string
	Nodes []Node
}

func groupByState(nodes []Node) []BoardColumn {
	columns := []BoardColumn{
		{State: StateOpen, Label: "Open"},
		{State: StateActive, Label: "Active"},
		{State: StateReview, Label: "Review"},
		{State: StateDone, Label: "Done"},
	}
	byState := map[string]*BoardColumn{}
	for i := range columns {
		byState[columns[i].State] = &columns[i]
	}
	for _, n := range nodes {
		if col, ok := byState[n.State]; ok {
			col.Nodes = append(col.Nodes, n)
		}
	}
	return columns
}

// Member holds aggregated activity data for the People lens.
type Member struct {
	Name     string
	Kind     string // "human" or "agent"
	OpCount  int
	LastSeen string
}

// handleNodeChildren returns child nodes for HTMX polling.
func (h *Handlers) handleNodeChildren(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nodeID := r.PathValue("id")
	children, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		ParentID: nodeID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"children": children})
		return
	}

	// Render children as HTML fragments for HTMX.
	for _, child := range children {
		if child.Kind == KindComment {
			CommentItem(child).Render(r.Context(), w)
		} else {
			childRow(child, space.Slug).Render(r.Context(), w)
		}
	}
}

// handleSetMindState sets a key-value pair for the Mind's context.
// PUT /api/mind-state with JSON body {"key": "...", "value": "..."}.
func (h *Handlers) handleSetMindState(w http.ResponseWriter, r *http.Request) {
	populateFormFromJSON(r)
	key := strings.TrimSpace(r.FormValue("key"))
	value := r.FormValue("value")
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	if err := h.store.SetMindState(r.Context(), key, value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"key": key, "status": "ok"})
}
