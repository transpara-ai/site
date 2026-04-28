package graph

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/transpara-ai/site/auth"
	"github.com/transpara-ai/site/profile"
)

// anonUserID is the sentinel value returned by userID() when no session exists.
const anonUserID = "anonymous"

// ViewUser holds user info for templates.
type ViewUser struct {
	ID          string
	Name        string
	Picture     string
	UnreadCount int
}

// ViewAPIKey holds API key info for templates.
type ViewAPIKey struct {
	ID        string
	Name      string
	AgentName string
	CreatedAt string
}

// LoopState holds the current hive loop iteration state read from loop files on disk.
type LoopState struct {
	Iteration  int
	Phase      string
	BuildTitle string
	BuildCost  float64 // parsed from build.md
}

// readLoopState collects LoopState for the authed hive views from the
// hive repo's loop/ directory. Iteration and Phase come from
// loop/diagnostics.jsonl (newline count + last entry's phase); build
// title and cost still come from loop/build.md. state.md is no longer
// read — the Reflector's narrative format ("Last updated: Iteration N")
// doesn't match the line-prefix parser the site used, which was
// silently zeroing the display on a working hive.
//
// Iteration count matches the hive's own canonical counter
// (pkg/runner/pipeline_tree.go: countDiagnostics — newline bytes). Guard
// for the edge case where the file's final line isn't newline-terminated
// (crash mid-write, or a future batching change in the hive's writer):
// if data is non-empty and doesn't end in \n, count that trailing line.
// Without the guard the dashboard would silently under-report by one.
//
// Returns zero value if dir is empty or files are missing. This is the
// local-dev path; production supplements with DB-backed counts in the
// handler, so a zero here on Fly is expected and fine.
func readLoopState(dir string) LoopState {
	if dir == "" {
		return LoopState{}
	}
	var s LoopState
	if data, err := os.ReadFile(filepath.Join(dir, "diagnostics.jsonl")); err == nil {
		for _, b := range data {
			if b == '\n' {
				s.Iteration++
			}
		}
		if len(data) > 0 && data[len(data)-1] != '\n' {
			s.Iteration++
		}
		lines := strings.Split(string(data), "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			var entry struct {
				Phase string `json:"phase"`
			}
			if err := json.Unmarshal([]byte(line), &entry); err == nil {
				s.Phase = entry.Phase
				break
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, "build.md")); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			if after, ok := strings.CutPrefix(line, "# "); ok && s.BuildTitle == "" {
				s.BuildTitle = strings.TrimSpace(after)
			}
		}
		s.BuildCost = parseCostDollars(string(data))
	}
	return s
}

// DiagEntry is a phase diagnostic event read from loop/diagnostics.jsonl.
type DiagEntry struct {
	Phase     string
	Outcome   string
	CostUSD   float64
	Timestamp time.Time
}

// readDiagnostics reads the last limit entries from loop/diagnostics.jsonl, newest first.
// Returns nil if dir is empty or the file does not exist.
func readDiagnostics(dir string, limit int) []DiagEntry {
	if dir == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(dir, "diagnostics.jsonl"))
	if err != nil {
		return nil
	}
	type raw struct {
		Phase     string  `json:"phase"`
		Outcome   string  `json:"outcome"`
		CostUSD   float64 `json:"cost_usd"`
		Timestamp string  `json:"timestamp"`
	}
	var all []DiagEntry
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		var r raw
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		e := DiagEntry{Phase: r.Phase, Outcome: r.Outcome, CostUSD: r.CostUSD}
		if r.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, r.Timestamp); err == nil {
				e.Timestamp = t
			}
		}
		all = append(all, e)
	}
	start := 0
	if len(all) > limit {
		start = len(all) - limit
	}
	tail := make([]DiagEntry, len(all)-start)
	copy(tail, all[start:])
	// Reverse so newest entry is first.
	for i, j := 0, len(tail)-1; i < j; i, j = i+1, j-1 {
		tail[i], tail[j] = tail[j], tail[i]
	}
	return tail
}

// RecentCommit holds a short git commit hash and subject line.
type RecentCommit struct {
	Hash    string
	Subject string
}

// readRecentCommits runs git log in repoDir and returns up to limit commits.
// Returns nil if repoDir is empty or git is unavailable.
func readRecentCommits(repoDir string, limit int) []RecentCommit {
	if repoDir == "" {
		return nil
	}
	out, err := exec.Command("git", "-C", repoDir, "log", "--oneline", fmt.Sprintf("-%d", limit)).Output()
	if err != nil {
		return nil
	}
	var commits []RecentCommit
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			commits = append(commits, RecentCommit{Hash: parts[0], Subject: parts[1]})
		}
	}
	return commits
}

// hivePhaseClass returns Tailwind badge classes for a pipeline phase name.
func hivePhaseClass(phase string) string {
	switch strings.ToLower(phase) {
	case "scout":
		return "bg-amber-400/10 text-amber-300 border-amber-400/20"
	case "architect":
		return "bg-indigo-400/10 text-indigo-300 border-indigo-400/20"
	case "builder":
		return "bg-emerald-400/10 text-emerald-300 border-emerald-400/20"
	case "critic":
		return "bg-orange-400/10 text-orange-300 border-orange-400/20"
	case "reflector":
		return "bg-violet-400/10 text-violet-300 border-violet-400/20"
	default:
		return "bg-elevated text-warm-muted border-edge"
	}
}

// diagOutcomeIcon returns a symbol representing a diagnostic outcome.
func diagOutcomeIcon(outcome string) string {
	switch outcome {
	case "revise", "revise_blocked":
		return "↻"
	case "failure", "error":
		return "✗"
	case "empty_sections":
		return "○"
	case "":
		return "✓"
	default:
		return "·"
	}
}

// diagOutcomeColor returns a Tailwind text-color class for a diagnostic outcome.
func diagOutcomeColor(outcome string) string {
	switch outcome {
	case "revise", "revise_blocked":
		return "text-amber-400"
	case "failure", "error":
		return "text-red-400"
	case "empty_sections":
		return "text-warm-faint"
	case "":
		return "text-emerald-400"
	default:
		return "text-warm-muted"
	}
}

// maxHiveDiagEntries is the upper bound on diagnostic entries shown on the /hive page.
const maxHiveDiagEntries = 10

// Handlers serves the unified product HTTP endpoints.
type Handlers struct {
	store     *Store
	mind      *Mind                               // optional — triggers auto-reply on conversation messages
	readWrap  func(http.HandlerFunc) http.Handler // optional auth (reads)
	writeWrap func(http.HandlerFunc) http.Handler // required auth (writes)
	loopDir   string                              // optional — path to loop/ dir for reading iteration state
}

// SetLoopDir configures the directory from which loop state files are read.
func (h *Handlers) SetLoopDir(dir string) { h.loopDir = dir }

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
	mux.Handle("POST /app/{slug}/invites", h.writeWrap(h.handleCreateInviteHTMX))
	mux.Handle("DELETE /app/{slug}/invites/{id}", h.writeWrap(h.handleRevokeInvite))
	mux.Handle("GET /join/{code}", h.readWrap(h.handleJoinViaInvite))

	// Space lenses (optional auth — public spaces readable by anyone).
	mux.Handle("GET /app/{slug}", h.readWrap(h.handleSpaceDefault))
	mux.Handle("GET /app/{slug}/board", h.readWrap(h.handleBoard))
	mux.Handle("GET /app/{slug}/refinery", h.readWrap(h.handleRefinery))
	mux.Handle("POST /app/{slug}/refinery/intake", h.writeWrap(h.handleRefineryIntake))
	mux.Handle("POST /app/{slug}/checklist/dismiss", h.writeWrap(h.handleChecklistDismiss))
	mux.Handle("GET /app/{slug}/feed", h.readWrap(h.handleFeed))
	mux.Handle("GET /app/{slug}/threads", h.readWrap(h.handleThreads))
	mux.Handle("GET /app/{slug}/conversations", h.readWrap(h.handleConversations))
	mux.Handle("GET /app/{slug}/people", h.readWrap(h.handlePeople))
	mux.Handle("GET /app/{slug}/agents", h.readWrap(h.handleAgents))
	mux.Handle("PATCH /app/{slug}/agents/{name}/session", h.writeWrap(h.handleAgentSessionUpdate))
	mux.Handle("POST /app/{slug}/agents/{name}/chat", h.writeWrap(h.handleAgentChat))
	mux.Handle("GET /app/{slug}/activity", h.readWrap(h.handleActivity))
	mux.Handle("GET /app/{slug}/knowledge", h.readWrap(h.handleKnowledge))
	mux.Handle("GET /app/{slug}/governance", h.readWrap(h.handleGovernance))
	mux.Handle("GET /app/{slug}/changelog", h.readWrap(h.handleChangelog))
	mux.Handle("GET /app/{slug}/projects", h.readWrap(h.handleProjects))
	mux.Handle("GET /app/{slug}/goals", h.readWrap(h.handleGoals))
	mux.Handle("GET /app/{slug}/goals/{id}", h.readWrap(h.handleGoalDetail))
	mux.Handle("GET /app/{slug}/roles", h.readWrap(h.handleRoles))
	mux.Handle("GET /app/{slug}/teams", h.readWrap(h.handleTeams))
	mux.Handle("GET /app/{slug}/policies", h.readWrap(h.handlePolicies))
	mux.Handle("GET /app/{slug}/documents", h.readWrap(h.handleDocuments))
	mux.Handle("GET /app/{slug}/document/{id}/edit", h.writeWrap(h.handleDocumentEdit))
	mux.Handle("POST /app/{slug}/document/{id}/edit", h.writeWrap(h.handleDocumentEdit))
	mux.Handle("GET /app/{slug}/questions", h.readWrap(h.handleQuestions))
	mux.Handle("GET /app/{slug}/questions/{id}", h.readWrap(h.handleQuestionDetail))
	mux.Handle("GET /app/{slug}/questions/{id}/answers", h.readWrap(h.handleQuestionAnswers))
	mux.Handle("GET /app/{slug}/council", h.readWrap(h.handleCouncil))
	mux.Handle("GET /app/{slug}/council/{id}", h.readWrap(h.handleCouncilDetail))

	// Conversation detail (optional auth).
	mux.Handle("GET /app/{slug}/conversation/{id}", h.readWrap(h.handleConversationDetail))
	mux.Handle("GET /app/{slug}/conversation/{id}/messages", h.readWrap(h.handleConversationMessages))

	// Node detail (optional auth — public spaces readable by anyone).
	mux.Handle("GET /app/{slug}/node/{id}", h.readWrap(h.handleNodeDetail))
	mux.Handle("GET /app/{slug}/node/{id}/activity", h.readWrap(h.handleNodeActivity))

	// Grammar operations (requires auth).
	mux.Handle("POST /app/{slug}/op", h.writeWrap(h.handleOp))

	// Mind state (requires auth — used by cmd/post to sync loop state).
	mux.Handle("PUT /api/mind-state", h.writeWrap(h.handleSetMindState))

	// Hive diagnostics ingestion (requires auth — used by hive runner).
	mux.Handle("POST /api/hive/diagnostic", h.writeWrap(h.handleHiveDiagnostic))
	mux.Handle("POST /api/hive/escalation", h.writeWrap(h.handleHiveEscalation))
	mux.Handle("POST /api/hive/refinery/state", h.writeWrap(h.handleHiveRefineryState))
	mux.Handle("POST /api/hive/mirror", h.writeWrap(h.handleHiveMirror))
	mux.Handle("GET /api/hive/webhook-deliveries", h.readWrap(h.handleHiveWebhookDeliveries))

	// Node children (for HTMX polling).
	mux.Handle("GET /app/{slug}/node/{id}/children", h.readWrap(h.handleNodeChildren))

	// Node mutations (requires auth).
	mux.Handle("POST /app/{slug}/node/{id}/state", h.writeWrap(h.handleNodeState))
	mux.Handle("POST /app/{slug}/node/{id}/update", h.writeWrap(h.handleNodeUpdate))
	mux.Handle("POST /app/{slug}/node/{id}/investigate-gaps", h.writeWrap(h.handleNodeInvestigateGaps))
	mux.Handle("DELETE /app/{slug}/node/{id}", h.writeWrap(h.handleNodeDelete))

	// Hive dashboard — public, no auth required.
	mux.HandleFunc("GET /hive", h.handleHive)
	mux.HandleFunc("GET /hive/feed", h.handleHiveFeed)
	mux.HandleFunc("GET /hive/stats", h.handleHiveStats)
	mux.HandleFunc("GET /hive/status", h.handleHiveStatus)
}

// ────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────

func (h *Handlers) viewUser(r *http.Request) ViewUser {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return ViewUser{Name: "Anonymous"}
	}
	uid := h.userID(r)
	unread := h.store.UnreadCount(r.Context(), uid)
	return ViewUser{ID: u.ID, Name: u.Name, Picture: u.Picture, UnreadCount: unread}
}

func (h *Handlers) userID(r *http.Request) string {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return anonUserID
	}
	return u.ID
}

func (h *Handlers) userName(r *http.Request) string {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return anonUserID
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
	if targetID == "" || targetID == anonUserID {
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
	if space.Visibility == VisibilityPublic && uid != "" {
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

// spaceForRead returns a space if the user owns it, it's public, or the user is a member (for reads).
func (h *Handlers) spaceForRead(r *http.Request) (*Space, bool, error) {
	slug := r.PathValue("slug")
	space, err := h.store.GetSpaceBySlug(r.Context(), slug)
	if err != nil {
		return nil, false, err
	}
	uid := h.userID(r)
	isOwner := space.OwnerID == uid
	if isOwner || space.Visibility == VisibilityPublic {
		return space, isOwner, nil
	}
	if uid != anonUserID && h.store.IsMember(r.Context(), space.ID, uid) {
		return space, false, nil
	}
	return nil, false, ErrNotFound
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

// parseMessageSearch extracts from:user operators from a search query.
// Returns the remaining body query and the from-author filter.
// sortTasks sorts tasks in place by the given sort key.
func sortTasks(tasks []Node, sortBy string) {
	switch sortBy {
	case "priority":
		sort.Slice(tasks, func(i, j int) bool {
			return priorityRank(tasks[i].Priority) < priorityRank(tasks[j].Priority)
		})
	case "due":
		sort.Slice(tasks, func(i, j int) bool {
			if tasks[i].DueDate == nil && tasks[j].DueDate == nil {
				return false
			}
			if tasks[i].DueDate == nil {
				return false
			}
			if tasks[j].DueDate == nil {
				return true
			}
			return tasks[i].DueDate.Before(*tasks[j].DueDate)
		})
	case "created":
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
		})
	case "state":
		sort.Slice(tasks, func(i, j int) bool {
			return stateRank(tasks[i].State) < stateRank(tasks[j].State)
		})
	case "assignee":
		sort.Slice(tasks, func(i, j int) bool {
			return tasks[i].Assignee < tasks[j].Assignee
		})
	default:
		// Default: by priority then created
		sort.Slice(tasks, func(i, j int) bool {
			pi, pj := priorityRank(tasks[i].Priority), priorityRank(tasks[j].Priority)
			if pi != pj {
				return pi < pj
			}
			return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
		})
	}
}

func priorityRank(p string) int {
	switch p {
	case "urgent":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}

func stateRank(s string) int {
	switch s {
	case StateActive:
		return 0
	case StateReview:
		return 1
	case StateOpen:
		return 2
	case StateDone:
		return 3
	default:
		return 4
	}
}

func parseMessageSearch(q string) (body string, fromAuthor string) {
	parts := strings.Fields(q)
	var bodyParts []string
	for _, p := range parts {
		if strings.HasPrefix(p, "from:") {
			fromAuthor = strings.TrimPrefix(p, "from:")
		} else {
			bodyParts = append(bodyParts, p)
		}
	}
	body = strings.Join(bodyParts, " ")
	return
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// populateFormFromJSON parses a JSON request body into r.Form
// so r.FormValue() works for both form-encoded and JSON requests.
// JSON array values (e.g. causes:["id1","id2"]) are converted to CSV
// strings so r.FormValue("causes") returns "id1,id2".
func populateFormFromJSON(r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return
	}
	var m map[string]any
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		return
	}
	if r.Form == nil {
		r.Form = url.Values{}
	}
	for k, v := range m {
		switch val := v.(type) {
		case string:
			r.Form.Set(k, val)
		case []interface{}:
			// Convert JSON array to CSV for form compatibility.
			parts := make([]string, 0, len(val))
			for _, item := range val {
				if s, ok := item.(string); ok {
					parts = append(parts, s)
				}
			}
			r.Form.Set(k, strings.Join(parts, ","))
		case nil:
			// skip
		default:
			r.Form.Set(k, fmt.Sprintf("%v", val))
		}
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

	p := profile.FromContext(ctx)
	if p == nil {
		p = profile.Default()
	}

	if len(spaces) == 0 {
		// New users: show welcome page to create their first space.
		Welcome(h.viewUser(r), p).Render(ctx, w)
		return
	}

	taskFilter := r.URL.Query().Get("tasks")
	tasks, _ := h.store.ListUserTasks(ctx, uid, taskFilter, 20)
	convos, _ := h.store.ListUserConversations(ctx, uid, 5)
	agentOps, _ := h.store.ListUserAgentActivity(ctx, uid, 10)
	agents, _ := h.store.ListAgentNames(ctx)

	defaultSlug := ""
	if len(spaces) > 0 {
		defaultSlug = spaces[0].Slug
	}

	unread := h.store.UnreadCount(ctx, uid)
	Dashboard(spaces, tasks, convos, agentOps, h.viewUser(r), defaultSlug, agents, unread, taskFilter, p).Render(ctx, w)
}

func (h *Handlers) handleNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uid := h.userID(r)

	notifs, _ := h.store.ListNotifications(ctx, uid, 50)
	h.store.MarkNotificationsRead(ctx, uid)

	p := profile.FromContext(ctx)
	if p == nil {
		p = profile.Default()
	}

	NotificationsView(notifs, h.viewUser(r), p).Render(ctx, w)
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

	// Emit agent welcome message into Chat.
	if agentID, agentName, err := h.store.GetFirstAgent(r.Context()); err == nil && agentID != "" {
		userID := h.userID(r)
		convo, cerr := h.store.CreateNode(r.Context(), CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindConversation,
			Title:      "Welcome",
			Body:       "Hey, I'm your AI colleague. I can help with tasks, answer questions, and collaborate with your team.",
			Author:     agentName,
			AuthorID:   agentID,
			AuthorKind: "agent",
			Tags:       []string{userID, agentID},
		})
		if cerr == nil {
			h.store.RecordOp(r.Context(), space.ID, convo.ID, agentName, agentID, "converse", nil)
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
	members, _ := h.store.ListMembers(r.Context(), space.ID, 50)
	invites, _ := h.store.ListInvites(r.Context(), space.ID)
	SettingsView(*space, spaces, reports, h.viewUser(r), "", members, invites, profile.FromContext(r.Context())).Render(r.Context(), w)
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
		SettingsView(*space, spaces, reports, h.viewUser(r), "Name cannot be empty.", nil, nil, profile.FromContext(r.Context())).Render(r.Context(), w)
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
		SettingsView(*space, spaces, reports, h.viewUser(r), "Type the space name to confirm deletion.", nil, nil, profile.FromContext(r.Context())).Render(r.Context(), w)
		return
	}

	if err := h.store.DeleteSpace(r.Context(), space.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

// handleCreateInviteHTMX handles HTMX POST /app/{slug}/invites — creates a new invite code
// and returns an HTML fragment (InviteCodeRow) for inline insertion.
func (h *Handlers) handleCreateInviteHTMX(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceOwnerOnly(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := h.store.CreateInviteCode(r.Context(), space.ID, h.userID(r), nil, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	inv := InviteCode{Token: token, SpaceID: space.ID}
	InviteCodeRow(inv, space.Slug).Render(r.Context(), w)
}

// handleRevokeInvite handles DELETE /app/{slug}/invites/{id} — removes an invite code.
// {id} carries the token string (same value as InviteCode.Token sent by the template).
func (h *Handlers) handleRevokeInvite(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceOwnerOnly(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	code := r.PathValue("id")
	// Verify the token belongs to this space before deleting.
	spaceID := h.store.GetInviteSpaceID(r.Context(), code)
	if spaceID != space.ID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := h.store.RevokeInvite(r.Context(), code); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) handleJoinViaInvite(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	uid := h.userID(r)
	if uid == anonUserID {
		// Use readWrap (not writeWrap) so we can redirect with ?next= here,
		// preserving the invite URL across the login flow.
		next := url.QueryEscape(r.URL.Path)
		http.Redirect(w, r, "/auth/login?next="+next, http.StatusSeeOther)
		return
	}

	inv, err := h.store.GetInviteCode(r.Context(), code)
	if err != nil || inv == nil {
		http.Error(w, "invalid or expired invite", http.StatusNotFound)
		return
	}

	uname := h.userName(r)
	if err := h.store.JoinSpace(r.Context(), inv.SpaceID, uid, uname); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.store.UseInviteCode(r.Context(), code, uid); err != nil {
		log.Printf("UseInviteCode %s user %s: %v", code, uid, err)
	}
	h.store.RecordOp(r.Context(), inv.SpaceID, "", uname, uid, "join", nil)

	space, err := h.store.GetSpaceByID(r.Context(), inv.SpaceID)
	if err != nil {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/app/"+space.Slug+"/board", http.StatusSeeOther)
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

	members, _ := h.store.ListMembers(ctx, space.ID, 10)
	isMember := h.store.IsMember(ctx, space.ID, uid)
	loggedIn := uid != anonUserID
	SpaceOverview(*space, spaces, pinned, recentOps, h.viewUser(r), isOwner,
		memberCount, openTasks, activeTasks, doneTasks, members, isMember, loggedIn, profile.FromContext(ctx)).Render(ctx, w)
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

	// Load projects for filter dropdown.
	projects, _ := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindProject,
		ParentID: "root",
	})

	projectFilter := strings.TrimSpace(r.URL.Query().Get("project"))

	// Load tasks — if project filter is set, load children of that project.
	parentFilter := "root"
	if projectFilter != "" {
		parentFilter = projectFilter
	}
	tasks, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		ParentID: parentFilter,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply text/assignee filters.
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

	agents, _ := h.store.ListAgentPersonas(r.Context())
	viewMode := r.URL.Query().Get("view") // "list" or "" (board)
	sortBy := r.URL.Query().Get("sort")   // "priority", "due", "created", "state", "assignee"

	if viewMode == "list" {
		// Sort tasks for list view.
		sortTasks(tasks, sortBy)
		ListView(*space, spaces, tasks, h.viewUser(r), isOwner, agents, q, assigneeFilter, sortBy, projects, projectFilter, profile.FromContext(r.Context())).Render(r.Context(), w)
		return
	}

	columns := groupByState(tasks)
	showFirstCompletionToast := r.URL.Query().Get("first_completion") == "1"

	// Getting started checklist: show for authenticated users in new spaces (<1 hour old) that haven't dismissed.
	showChecklist := false
	if h.userID(r) != anonUserID && time.Since(space.CreatedAt) < time.Hour {
		if _, err := r.Cookie("checklist_" + space.ID); err != nil {
			showChecklist = true
		}
	}
	hasTask := false
	hasAgentTask := false
	taskCount := 0
	for _, col := range columns {
		for _, t := range col.Nodes {
			hasTask = true
			taskCount++
			if t.AssigneeKind == "agent" {
				hasAgentTask = true
			}
		}
	}
	hasCompletion := space.FirstCompletionAt != nil
	elapsedStr := formatElapsed(time.Since(space.CreatedAt))

	// Welcome modal: show once to non-owner members on their first board visit.
	uid := h.userID(r)
	showWelcome := false
	var welcomeMembers []SpaceMember
	if uid != anonUserID && uid != space.OwnerID && h.store.IsMember(r.Context(), space.ID, uid) {
		showWelcome = h.store.MarkWelcomed(r.Context(), space.ID, uid)
	}
	if showWelcome {
		welcomeMembers, _ = h.store.ListMembers(r.Context(), space.ID, 10)
	}

	// Invite card: show when checklist is complete and space has fewer than 2 members.
	showInviteCard := false
	inviteToken := ""
	if showChecklist && hasTask && hasAgentTask && hasCompletion {
		memberCount := h.store.MemberCount(r.Context(), space.ID)
		if memberCount < 2 {
			showInviteCard = true
			inviteToken = h.store.GetInviteToken(r.Context(), space.ID)
			if inviteToken == "" {
				inviteToken, _ = h.store.CreateInvite(r.Context(), space.ID, uid)
			}
		}
	}

	BoardView(*space, spaces, columns, h.viewUser(r), isOwner, agents, q, assigneeFilter, projects, projectFilter, showFirstCompletionToast, showChecklist, hasTask, hasAgentTask, hasCompletion, taskCount, elapsedStr, showWelcome, welcomeMembers, showInviteCard, inviteToken, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleRefinery(w http.ResponseWriter, r *http.Request) {
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
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	specs, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindSpec,
		ParentID: "root",
		Query:    searchQuery,
		Limit:    200,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "specs": specs})
		return
	}

	lineage := h.refineryLineageByRoot(r.Context(), specs)
	RefineryView(*space, spaces, groupSpecsByRefineryStage(specs), lineage, h.viewUser(r), isOwner, searchQuery, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleRefineryIntake(w http.ResponseWriter, r *http.Request) {
	space, err := h.spaceFromRequest(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := r.ParseMultipartForm(16 << 20); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		http.Error(w, "parse intake: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	question := strings.TrimSpace(r.FormValue("question"))
	outcome := strings.TrimSpace(r.FormValue("outcome"))
	manualBody := strings.TrimSpace(r.FormValue("body"))
	artifacts, err := readMarkdownIntakeArtifacts(r, 5, 2<<20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if question == "" && outcome == "" && manualBody == "" && len(artifacts) == 0 {
		http.Error(w, "question, outcome, pasted markdown, or markdown file required", http.StatusBadRequest)
		return
	}

	title := intakeTitle(question, outcome, manualBody, artifacts)
	duplicates, _ := h.findPotentialDuplicateIntakes(ctx, space.ID, title)
	kind, state, body := classifyRefineryIntake(question, outcome, manualBody, artifacts, duplicates)
	persistedState := SpecStateIntakeRaw
	if kind == KindQuestion {
		persistedState = StateOpen
		body = strings.Replace(body, "- Persisted state: "+SpecStateIntakeRaw, "- Persisted state: "+StateOpen, 1)
	}
	node, err := h.store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       kind,
		Title:      title,
		Body:       body,
		State:      persistedState,
		Priority:   PriorityMedium,
		Author:     h.userName(r),
		AuthorID:   h.userID(r),
		AuthorKind: h.userKind(r),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"classifier_kind":   kind,
		"recommended_state": state,
		"persisted_state":   persistedState,
		"duplicate_count":   len(duplicates),
		"artifact_count":    len(artifacts),
		"question":          question,
		"desired_outcome":   outcome,
		"rationale":         refineryClassifierRationale(kind, state, duplicates, artifacts),
	})
	h.store.RecordOp(ctx, space.ID, node.ID, h.userName(r), h.userID(r), "intake", payload)

	for _, artifact := range artifacts {
		child, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			ParentID:   node.ID,
			Kind:       KindDocument,
			Title:      artifact.Name,
			Body:       artifact.Body,
			State:      StateOpen,
			Priority:   PriorityMedium,
			Author:     h.userName(r),
			AuthorID:   h.userID(r),
			AuthorKind: h.userKind(r),
			Causes:     []string{node.ID},
		})
		if err == nil {
			attachPayload, _ := json.Marshal(map[string]any{
				"parent_id": node.ID,
				"filename":  artifact.Name,
				"hash":      sha256HexString([]byte(artifact.Body)),
			})
			h.store.RecordOp(ctx, space.ID, child.ID, h.userName(r), h.userID(r), "attach", attachPayload)
		}
	}

	if kind == KindQuestion {
		if h.mind != nil {
			h.recordQuestionStatus(ctx, space.ID, node.ID, "Question received. A lightweight answer is being generated.")
			go h.mind.OnQuestionAsked(space.ID, space.Slug, node)
		} else {
			h.recordQuestionStatus(ctx, space.ID, node.ID, "Question received. No answering backend is configured for lightweight Q&A.")
		}
	} else if kind == KindSpec && refineryColumnState(state) == SpecStateInvestigate {
		if task, missing, err := h.ensureRefineryGapInvestigation(ctx, *space, *node, "Agents", "agents", "agent"); err == nil && task != nil {
			log.Printf("[refinery] auto-started gap investigation for %s missing=%s task=%s", node.ID, missing, task.ID)
		} else if err != nil {
			log.Printf("[refinery] auto-start gap investigation failed for %s: %v", node.ID, err)
		}
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusCreated, map[string]any{"node": node, "duplicates": duplicates})
		return
	}
	redirectPath := "/app/" + space.Slug + "/refinery?q=" + url.QueryEscape(node.Title)
	if kind == KindQuestion {
		redirectPath = "/app/" + space.Slug + "/questions/" + node.ID
	}
	http.Redirect(w, r, redirectPath, http.StatusSeeOther)
}

func (h *Handlers) recordQuestionStatus(ctx context.Context, spaceID, questionID, message string) {
	status, err := h.store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    spaceID,
		ParentID:   questionID,
		Kind:       KindComment,
		Body:       message,
		State:      StateOpen,
		Priority:   PriorityLow,
		Author:     "System",
		AuthorID:   "system",
		AuthorKind: "system",
	})
	if err == nil {
		payload, _ := json.Marshal(map[string]string{"status": "question_status"})
		h.store.RecordOp(ctx, spaceID, status.ID, "System", "system", "respond", payload)
	}
}

func (h *Handlers) handleHiveRefineryState(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SpaceID string `json:"space_id"`
		NodeID  string `json:"node_id"`
		State   string `json:"state"`
		Reason  string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	req.SpaceID = strings.TrimSpace(req.SpaceID)
	req.NodeID = strings.TrimSpace(req.NodeID)
	req.State = strings.TrimSpace(req.State)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.SpaceID == "" || req.NodeID == "" || req.State == "" {
		http.Error(w, "space_id, node_id, and state required", http.StatusBadRequest)
		return
	}

	node, err := h.store.GetNode(r.Context(), req.NodeID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if node.SpaceID != req.SpaceID {
		http.Error(w, "node does not belong to requested space", http.StatusBadRequest)
		return
	}
	if node.State == req.State {
		writeJSON(w, http.StatusOK, map[string]any{"node": node, "changed": false})
		return
	}

	if err := h.store.UpdateNodeState(r.Context(), req.NodeID, req.State); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	statePayload, _ := json.Marshal(map[string]string{
		"from_state": node.State,
		"to_state":   req.State,
		"reason":     req.Reason,
		"source":     "hive.refinery.automation",
	})
	opName := "progress"
	if req.State == StateDone || req.State == SpecStateWorkShipped {
		opName = "complete"
	} else if req.State == StateReview {
		opName = "review"
	}
	op, _ := h.store.RecordOp(r.Context(), req.SpaceID, req.NodeID, h.userName(r), h.userID(r), opName, statePayload)
	updated, _ := h.store.GetNode(r.Context(), req.NodeID)
	writeJSON(w, http.StatusOK, map[string]any{"node": updated, "op": op, "changed": true})
}

type intakeArtifact struct {
	Name string
	Body string
}

func readMarkdownIntakeArtifacts(r *http.Request, maxFiles int, maxBytes int64) ([]intakeArtifact, error) {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, nil
	}
	files := r.MultipartForm.File["artifact"]
	if len(files) > maxFiles {
		return nil, fmt.Errorf("too many artifacts: maximum %d", maxFiles)
	}
	artifacts := make([]intakeArtifact, 0, len(files))
	for _, fh := range files {
		name := filepath.Base(fh.Filename)
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".md") && !strings.HasSuffix(lower, ".markdown") {
			return nil, fmt.Errorf("%s is not a markdown file", name)
		}
		if fh.Size > maxBytes {
			return nil, fmt.Errorf("%s exceeds the %d byte artifact limit", name, maxBytes)
		}
		f, err := fh.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", name, err)
		}
		data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		if int64(len(data)) > maxBytes {
			return nil, fmt.Errorf("%s exceeds the %d byte artifact limit", name, maxBytes)
		}
		artifacts = append(artifacts, intakeArtifact{Name: name, Body: string(data)})
	}
	return artifacts, nil
}

func refineryClassifierRationale(kind, state string, duplicates []Node, artifacts []intakeArtifact) string {
	switch {
	case kind == KindQuestion:
		return "trivial question detected; route through lightweight Q&A while preserving raw intake provenance"
	case len(duplicates) > 0:
		return "potential duplicate intake detected; human confirmation is required before autonomous progression"
	case state == SpecStateDraft:
		return "normative sections detected; recommend draft spec while persisted state remains raw intake"
	case state == SpecStateInvestigate:
		return "artifact or substantial scope detected; recommend investigation without blocking ingestion"
	case len(artifacts) > 0:
		return "artifact attached; preserve evidence and classify for refinery progression"
	default:
		return "raw intake requires clarification before drafting"
	}
}

func sha256HexString(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf("sha256:%x", sum[:])
}

func intakeTitle(question, outcome, manualBody string, artifacts []intakeArtifact) string {
	for _, candidate := range []string{question, firstNonEmptyLine(outcome), firstMarkdownHeading(manualBody)} {
		if candidate != "" {
			return truncatePlain(candidate, 120)
		}
	}
	if len(artifacts) > 0 {
		if heading := firstMarkdownHeading(artifacts[0].Body); heading != "" {
			return truncatePlain(heading, 120)
		}
		return truncatePlain(artifacts[0].Name, 120)
	}
	return "Untitled intake"
}

func classifyRefineryIntake(question, outcome, manualBody string, artifacts []intakeArtifact, duplicates []Node) (string, string, string) {
	combined := strings.TrimSpace(strings.Join([]string{question, outcome, manualBody}, "\n\n"))
	for _, artifact := range artifacts {
		combined += "\n\n" + artifact.Body
	}
	kind := KindSpec
	state := SpecStateRequirementClarify
	if isTrivialQuestion(question, outcome, manualBody, artifacts) {
		kind = KindQuestion
		state = StateOpen
	} else if len(duplicates) > 0 {
		state = SpecStateNeedsAttention
	} else if len(artifacts) > 0 || len(combined) > 1200 || looksLikeSpec(combined) {
		state = SpecStateInvestigate
		if hasNormativeSections(combined) {
			state = SpecStateDraft
		}
	}

	var b strings.Builder
	if question != "" {
		b.WriteString("## Intake Question\n\n")
		b.WriteString(question + "\n\n")
	}
	if outcome != "" {
		b.WriteString("## Desired Outcome\n\n")
		b.WriteString(outcome + "\n\n")
	}
	if manualBody != "" {
		b.WriteString("## Pasted Markdown\n\n")
		b.WriteString(manualBody + "\n\n")
	}
	if len(artifacts) > 0 {
		b.WriteString("## Uploaded Artifacts\n\n")
		for _, artifact := range artifacts {
			b.WriteString("- " + artifact.Name + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Classifier Notes\n\n")
	b.WriteString("- Intake kind: " + kind + "\n")
	b.WriteString("- Persisted state: " + SpecStateIntakeRaw + "\n")
	b.WriteString("- Recommended next state: " + state + "\n")
	if len(duplicates) > 0 {
		b.WriteString("- Needs attention: potential redundancy with existing intake. Confirm whether to merge, supersede, or proceed independently.\n")
	}
	if kind == KindSpec && state == SpecStateRequirementClarify {
		b.WriteString("- Clarify: human direction is likely needed before a vetted draft spec can be produced.\n")
	}
	if state == SpecStateInvestigate {
		b.WriteString("- Investigate: artifact or scope appears substantial; agent analysis should continue without blocking ingestion.\n")
	}
	return kind, state, strings.TrimSpace(b.String())
}

func (h *Handlers) findPotentialDuplicateIntakes(ctx context.Context, spaceID, title string) ([]Node, error) {
	needle := normalizeIntakeTitle(title)
	if needle == "" {
		return nil, nil
	}
	var matches []Node
	for _, kind := range []string{KindSpec, KindQuestion} {
		nodes, err := h.store.ListNodes(ctx, ListNodesParams{SpaceID: spaceID, Kind: kind, ParentID: "root", Limit: 200})
		if err != nil {
			return nil, err
		}
		for _, n := range nodes {
			existing := normalizeIntakeTitle(n.Title)
			if existing == needle || (len(needle) > 16 && strings.Contains(existing, needle)) || (len(existing) > 16 && strings.Contains(needle, existing)) {
				matches = append(matches, n)
			}
		}
	}
	return matches, nil
}

func normalizeIntakeTitle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

func isTrivialQuestion(question, outcome, manualBody string, artifacts []intakeArtifact) bool {
	if len(artifacts) > 0 || manualBody != "" || outcome != "" {
		return false
	}
	q := strings.TrimSpace(question)
	words := strings.Fields(q)
	if len(words) == 0 || len(words) > 14 {
		return false
	}
	if strings.HasSuffix(q, "?") {
		return true
	}
	first := strings.ToLower(words[0])
	switch first {
	case "why", "what", "how", "when", "where", "who", "which", "can", "could", "should", "would", "is", "are", "do", "does", "did":
		return true
	default:
		return false
	}
}

func looksLikeSpec(s string) bool {
	lower := strings.ToLower(s)
	for _, marker := range []string{"# ", "definition of done", "acceptance criteria", "test plan", "requirements", "invariants", "architecture"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func hasNormativeSections(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "definition of done") && (strings.Contains(lower, "test plan") || strings.Contains(lower, "acceptance criteria"))
}

func firstMarkdownHeading(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
	}
	return ""
}

func firstNonEmptyLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

func truncatePlain(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max-1]) + "…"
}

func formatElapsed(d time.Duration) string {
	minutes := int(d.Minutes())
	if minutes < 2 {
		return "a minute"
	}
	if minutes < 60 {
		return fmt.Sprintf("%d minutes", minutes)
	}
	hours := int(d.Hours())
	if hours == 1 {
		return "about an hour"
	}
	return fmt.Sprintf("about %d hours", hours)
}

func checklistDoneCount(hasTask, hasAgentTask, hasCompletion bool) int {
	n := 0
	if hasTask {
		n++
	}
	if hasAgentTask {
		n++
	}
	if hasCompletion {
		n++
	}
	return n
}

func (h *Handlers) handleChecklistDismiss(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "checklist_" + space.ID,
		Value:    "1",
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/app/"+space.Slug+"/board", http.StatusSeeOther)
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

	searchQuery := r.URL.Query().Get("q")
	feedTab := r.URL.Query().Get("tab") // "following", "foryou", or "" (all)

	var posts []Node
	if feedTab == "foryou" && searchQuery == "" {
		posts, err = h.store.ListPostsByEngagement(r.Context(), space.ID, 50)
	} else if feedTab == "trending" && searchQuery == "" {
		posts, err = h.store.ListPostsByTrending(r.Context(), space.ID, 50)
	} else {
		posts, err = h.store.ListNodes(r.Context(), ListNodesParams{
			SpaceID:  space.ID,
			Kind:     KindPost,
			ParentID: "root",
			Query:    searchQuery,
		})
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Filter by Following tab: show only posts by followed users + reposted by followed users.
	repostedBy := make(map[string]string) // nodeID → reposter display name
	if feedTab == "following" {
		uid := h.userID(r)
		followedIDs := h.store.ListFollowedIDs(r.Context(), uid)
		followSet := make(map[string]bool, len(followedIDs))
		for _, id := range followedIDs {
			followSet[id] = true
		}
		// Also include posts reposted by followed users.
		repostedNodeIDs := h.store.ListRepostedNodeIDs(r.Context(), followedIDs, 100)
		repostSet := make(map[string]bool, len(repostedNodeIDs))
		for _, id := range repostedNodeIDs {
			repostSet[id] = true
		}
		var filtered []Node
		for _, p := range posts {
			if followSet[p.AuthorID] || repostSet[p.ID] {
				filtered = append(filtered, p)
			}
		}
		posts = filtered

		// Build repost attribution: which followed user reposted each post?
		var repostedIDs []string
		for _, p := range posts {
			if !followSet[p.AuthorID] && repostSet[p.ID] {
				repostedIDs = append(repostedIDs, p.ID)
			}
		}
		if len(repostedIDs) > 0 {
			attrMap := h.store.GetRepostAttribution(r.Context(), followedIDs, repostedIDs)
			// Resolve user IDs to display names.
			var userIDs []string
			for _, uid := range attrMap {
				userIDs = append(userIDs, uid)
			}
			nameMap := h.store.ResolveUserNames(r.Context(), userIDs)
			for nodeID, reposterID := range attrMap {
				if name, ok := nameMap[reposterID]; ok {
					repostedBy[nodeID] = name
				}
			}
		}
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "nodes": posts})
		return
	}

	agents, _ := h.store.ListAgentNames(r.Context())

	// Load endorsement + repost data for all posts.
	var postIDs []string
	for _, p := range posts {
		postIDs = append(postIDs, p.ID)
	}
	endorseCounts := h.store.GetBulkEndorsementCounts(r.Context(), postIDs)
	userEndorsed := h.store.GetBulkUserEndorsements(r.Context(), h.userID(r), postIDs)
	repostCounts := h.store.GetBulkRepostCounts(r.Context(), postIDs)
	userReposted := h.store.GetBulkUserReposts(r.Context(), h.userID(r), postIDs)

	// Quote: load the quoted post for compose form preview.
	var quotePost *Node
	if qid := r.URL.Query().Get("quote"); qid != "" {
		quotePost, _ = h.store.GetNode(r.Context(), qid)
	}

	FeedView(*space, spaces, posts, h.viewUser(r), isOwner, len(agents) > 0, searchQuery, feedTab, endorseCounts, userEndorsed, repostCounts, userReposted, repostedBy, quotePost, profile.FromContext(r.Context())).Render(r.Context(), w)
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

	searchQuery := r.URL.Query().Get("q")
	threads, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindThread,
		ParentID: "root",
		Query:    searchQuery,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "nodes": threads})
		return
	}

	ThreadsView(*space, spaces, threads, h.viewUser(r), isOwner, searchQuery, profile.FromContext(r.Context())).Render(r.Context(), w)
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

	searchQuery := r.URL.Query().Get("q")
	filterMode := r.URL.Query().Get("filter") // "dm" or "group" or ""
	convos, err := h.store.ListConversations(r.Context(), space.ID, h.userID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Filter by query and DM/group.
	var filtered []ConversationSummary
	sq := strings.ToLower(searchQuery)
	for _, c := range convos {
		if searchQuery != "" && !strings.Contains(strings.ToLower(c.Title), sq) {
			continue
		}
		if filterMode == "dm" && len(c.Tags) > 2 {
			continue
		}
		if filterMode == "group" && len(c.Tags) <= 2 {
			continue
		}
		filtered = append(filtered, c)
	}
	convos = filtered

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

	// Build persona map: convo ID → agent persona (nil if no agent in convo).
	// Uses a single batched query to avoid N+1 round-trips.
	personaMap := h.store.GetAgentPersonasForConversations(r.Context(), convos)

	// Message search: when a query is present, also search message bodies.
	var msgResults []MessageSearchResult
	if searchQuery != "" {
		bodyQ, fromAuthor := parseMessageSearch(searchQuery)
		msgResults, _ = h.store.SearchMessages(r.Context(), space.ID, bodyQ, fromAuthor, 20)
	}

	ConversationsView(*space, spaces, convos, h.viewUser(r), agents, nameMap, personaMap, searchQuery, filterMode == "dm", filterMode == "group", msgResults, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handlePeople(w http.ResponseWriter, r *http.Request) {
	// JSON agents endpoint for the hive pipeline.
	if r.URL.Query().Get("format") == "agents" && wantsJSON(r) {
		personas, _ := h.store.ListAgentPersonas(r.Context())
		writeJSON(w, http.StatusOK, map[string]any{"agents": personas})
		return
	}

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
	searchQuery := r.URL.Query().Get("q")
	var members []Member
	for _, m := range memberMap {
		if searchQuery != "" && !strings.Contains(strings.ToLower(m.Name), strings.ToLower(searchQuery)) {
			continue
		}
		members = append(members, *m)
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "members": members})
		return
	}

	PeopleView(*space, spaces, members, h.viewUser(r), searchQuery, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleAgents(w http.ResponseWriter, r *http.Request) {
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
	personas, _ := h.store.ListAgentPersonas(r.Context())

	categoryOrder := []string{"care", "governance", "knowledge", "product", "outward", "resource"}
	categoryLabels := map[string]string{
		"care": "Care", "governance": "Governance", "knowledge": "Knowledge",
		"product": "Product", "outward": "Outward", "resource": "Resource",
	}
	grouped := make(map[string][]AppAgentPersona)
	for _, p := range personas {
		grouped[p.Category] = append(grouped[p.Category], AppAgentPersona{
			Name: p.Name, Display: p.Display, Description: p.Description, Category: p.Category,
		})
	}
	var categories []AppAgentCategoryGroup
	for _, cat := range categoryOrder {
		if items, ok := grouped[cat]; ok {
			categories = append(categories, AppAgentCategoryGroup{
				Name: cat, Label: categoryLabels[cat], Personas: items,
			})
		}
	}

	AgentsView(*space, spaces, categories, h.viewUser(r), profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleAgentSessionUpdate(w http.ResponseWriter, r *http.Request) {
	personaName := r.PathValue("name")
	var body struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SessionID == "" {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}
	if err := h.store.UpdateAgentSession(r.Context(), personaName, body.SessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "session_id": body.SessionID})
}

func (h *Handlers) handleAgentChat(w http.ResponseWriter, r *http.Request) {
	personaName := r.PathValue("name")
	ctx := r.Context()
	persona := h.store.GetAgentPersona(ctx, personaName)
	if persona == nil {
		http.NotFound(w, r)
		return
	}
	agentsSpace, err := h.store.GetSpaceBySlug(ctx, AgentsSpaceSlug)
	if err != nil || agentsSpace == nil {
		http.Error(w, "agents space not available", http.StatusServiceUnavailable)
		return
	}
	actorID, actor, actorKind := h.userID(r), h.userName(r), h.userKind(r)
	node, err := h.store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    agentsSpace.ID,
		Kind:       KindConversation,
		Title:      "Chat with " + persona.Display,
		Author:     actor,
		AuthorID:   actorID,
		AuthorKind: actorKind,
		Tags:       []string{actorID, "role:" + persona.Name},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.store.RecordOp(ctx, agentsSpace.ID, node.ID, actor, actorID, "converse", nil)
	if h.mind != nil {
		go h.mind.OnMessage(agentsSpace.ID, agentsSpace.Slug, node, actorID)
	}
	http.Redirect(w, r, fmt.Sprintf("/app/%s/conversation/%s", agentsSpace.Slug, node.ID), http.StatusSeeOther)
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

	opFilter := r.URL.Query().Get("op")
	ops, err := h.store.ListOps(r.Context(), space.ID, 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if opFilter != "" {
		var filtered []Op
		for _, o := range ops {
			if o.Op == opFilter {
				filtered = append(filtered, o)
			}
		}
		ops = filtered
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "ops": ops})
		return
	}

	ActivityView(*space, spaces, ops, h.viewUser(r), opFilter, profile.FromContext(r.Context())).Render(r.Context(), w)
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

	// ?op=max_lesson — server-side aggregate for NextLessonNumber (O(1), no truncation).
	if r.URL.Query().Get("op") == "max_lesson" {
		max, err := h.store.MaxLessonNumber(r.Context(), space.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"max_lesson": max})
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))

	searchQuery := r.URL.Query().Get("q")
	claims, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindClaim,
		ParentID: "root",
		Query:    searchQuery,
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

	// Fetch documents and questions for the unified knowledge view (BOUNDED: Limit 50 each).
	docs, _ := h.store.ListDocuments(r.Context(), space.ID, 50)
	questions, _ := h.store.ListQuestions(r.Context(), space.ID, 50)

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{
			"space":     space,
			"claims":    claims,
			"documents": docs,
			"questions": questions,
		})
		return
	}

	tab := r.URL.Query().Get("tab")
	if tab != "docs" && tab != "qa" && tab != "claims" {
		tab = "docs"
	}

	KnowledgeView(*space, spaces, claims, challengeCounts, h.viewUser(r), searchQuery, tab, docs, questions, profile.FromContext(r.Context())).Render(r.Context(), w)
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
	searchQuery := r.URL.Query().Get("q")
	entries, err := h.store.ListChangelog(r.Context(), space.ID, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if searchQuery != "" {
		var filtered []ChangelogEntry
		sq := strings.ToLower(searchQuery)
		for _, e := range entries {
			if strings.Contains(strings.ToLower(e.Title), sq) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "changelog": entries})
		return
	}

	ChangelogView(*space, spaces, entries, h.viewUser(r), searchQuery, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleProjects(w http.ResponseWriter, r *http.Request) {
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
	searchQuery := r.URL.Query().Get("q")

	projects, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindProject,
		ParentID: "root",
		Query:    searchQuery,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "projects": projects})
		return
	}

	ProjectsView(*space, spaces, projects, h.viewUser(r), isOwner, searchQuery, profile.FromContext(r.Context())).Render(r.Context(), w)
}

// GoalWithProjects pairs a goal with its child projects.
type GoalWithProjects struct {
	Goal     Node
	Projects []Node
}

// GoalDetail is the full goal view: goal + projects + direct tasks + aggregated counts.
type GoalDetail struct {
	Goal        Node
	Projects    []Node
	DirectTasks []Node
	TotalTasks  int
	DoneTasks   int
}

func (h *Handlers) handleGoals(w http.ResponseWriter, r *http.Request) {
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
	searchQuery := r.URL.Query().Get("q")

	goals, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindGoal,
		ParentID: "root",
		Query:    searchQuery,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	goalsWithProjects := make([]GoalWithProjects, 0, len(goals))
	for _, goal := range goals {
		projects, _ := h.store.ListNodes(r.Context(), ListNodesParams{
			SpaceID:  space.ID,
			Kind:     KindProject,
			ParentID: goal.ID,
		})
		goalsWithProjects = append(goalsWithProjects, GoalWithProjects{
			Goal:     goal,
			Projects: projects,
		})
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "goals": goalsWithProjects})
		return
	}

	GoalsView(*space, spaces, goalsWithProjects, h.viewUser(r), isOwner, searchQuery, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleGoalDetail(w http.ResponseWriter, r *http.Request) {
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
	id := r.PathValue("id")

	goal, err := h.store.GetNode(r.Context(), id)
	if errors.Is(err, ErrNotFound) || (err == nil && goal.Kind != KindGoal) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if goal.SpaceID != space.ID {
		http.NotFound(w, r)
		return
	}

	projects, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindProject,
		ParentID: goal.ID,
		Limit:    200,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	directTasks, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		ParentID: goal.ID,
		Limit:    200,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalTasks := len(directTasks)
	doneTasks := 0
	for _, t := range directTasks {
		if t.State == StateDone {
			doneTasks++
		}
	}
	for _, proj := range projects {
		totalTasks += proj.ChildCount
		doneTasks += proj.ChildDone
	}

	detail := GoalDetail{
		Goal:        *goal,
		Projects:    projects,
		DirectTasks: directTasks,
		TotalTasks:  totalTasks,
		DoneTasks:   doneTasks,
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "goal": detail})
		return
	}

	GoalDetailView(*space, spaces, detail, h.viewUser(r), isOwner, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleRoles(w http.ResponseWriter, r *http.Request) {
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
	searchQuery := r.URL.Query().Get("q")

	roles, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindRole,
		ParentID: "root",
		Query:    searchQuery,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "roles": roles})
		return
	}

	RolesView(*space, spaces, roles, h.viewUser(r), isOwner, searchQuery, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleTeams(w http.ResponseWriter, r *http.Request) {
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
	searchQuery := r.URL.Query().Get("q")

	teams, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindTeam,
		ParentID: "root",
		Query:    searchQuery,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "teams": teams})
		return
	}

	ctx := r.Context()
	uid := h.userID(r)
	memberCounts := make(map[string]int, len(teams))
	isMember := make(map[string]bool, len(teams))
	for _, t := range teams {
		memberCounts[t.ID] = h.store.NodeMemberCount(ctx, t.ID)
		if uid != "" {
			isMember[t.ID] = h.store.IsNodeMember(ctx, t.ID, uid)
		}
	}

	TeamsView(*space, spaces, teams, h.viewUser(r), isOwner, searchQuery, memberCounts, isMember, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handlePolicies(w http.ResponseWriter, r *http.Request) {
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
	searchQuery := r.URL.Query().Get("q")

	policies, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindPolicy,
		ParentID: "root",
		Query:    searchQuery,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "policies": policies})
		return
	}

	PoliciesView(*space, spaces, policies, h.viewUser(r), isOwner, searchQuery, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleDocuments(w http.ResponseWriter, r *http.Request) {
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
	searchQuery := r.URL.Query().Get("q")

	documents, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindDocument,
		ParentID: "root",
		Query:    searchQuery,
		Limit:    50,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "documents": documents})
		return
	}

	DocumentsView(*space, spaces, documents, h.viewUser(r), isOwner, searchQuery, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleDocumentEdit(w http.ResponseWriter, r *http.Request) {
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
	node, err := h.store.GetNode(r.Context(), nodeID)
	if errors.Is(err, ErrNotFound) || node == nil {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if node.Kind != KindDocument {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		title := r.FormValue("title")
		body := r.FormValue("body")
		if err := h.store.UpdateNode(r.Context(), nodeID, &title, &body, nil, nil, nil); err != nil {
			if errors.Is(err, ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if wantsJSON(r) {
			updated, _ := h.store.GetNode(r.Context(), nodeID)
			writeJSON(w, http.StatusOK, map[string]any{"node": updated})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))
	DocumentEditView(*space, spaces, *node, h.viewUser(r), profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleQuestions(w http.ResponseWriter, r *http.Request) {
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
	searchQuery := r.URL.Query().Get("q")

	questions, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		Kind:     KindQuestion,
		ParentID: "root",
		Query:    searchQuery,
		Limit:    50,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "questions": questions})
		return
	}

	QuestionsView(*space, spaces, questions, h.viewUser(r), isOwner, searchQuery, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleQuestionDetail(w http.ResponseWriter, r *http.Request) {
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
	id := r.PathValue("id")

	question, err := h.store.GetNode(r.Context(), id)
	if errors.Is(err, ErrNotFound) || (err == nil && question.Kind != KindQuestion) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if question.SpaceID != space.ID {
		http.NotFound(w, r)
		return
	}

	answers, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		ParentID: id,
		Limit:    200,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "question": question, "answers": answers})
		return
	}

	QuestionDetailView(*space, spaces, *question, answers, h.viewUser(r), isOwner, profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleQuestionAnswers(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id := r.PathValue("id")
	question, err := h.store.GetNode(r.Context(), id)
	if errors.Is(err, ErrNotFound) || (err == nil && question.Kind != KindQuestion) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if question.SpaceID != space.ID {
		http.NotFound(w, r)
		return
	}
	answers, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		ParentID: id,
		Limit:    200,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderQuestionAnswersHTML(w, space.Slug, question.ID, answers)
}

func renderQuestionAnswersHTML(w io.Writer, spaceSlug, questionID string, answers []Node) {
	fmt.Fprintf(w, `<div id="question-answers" class="space-y-4" hx-get="/app/%s/questions/%s/answers" hx-trigger="every 3s" hx-swap="outerHTML">`, html.EscapeString(spaceSlug), html.EscapeString(questionID))
	fmt.Fprintf(w, `<h2 class="text-sm font-semibold text-warm-muted">Answers (%d)</h2>`, len(answers))
	for _, a := range answers {
		body := strings.ReplaceAll(html.EscapeString(a.Body), "\n", "<br>")
		author := html.EscapeString(a.Author)
		date := html.EscapeString(a.CreatedAt.Format("Jan 2, 2006"))
		if a.AuthorKind == "agent" {
			fmt.Fprintf(w, `<div class="bg-violet-950/20 rounded-lg border border-violet-500/20 p-4 space-y-2"><div class="flex items-center gap-2 mb-1"><span class="inline-flex items-center gap-1 text-xs font-medium text-violet-400 bg-violet-500/10 border border-violet-500/20 rounded-full px-2 py-0.5"><svg class="w-2.5 h-2.5" fill="currentColor" viewBox="0 0 8 8"><circle cx="4" cy="4" r="4"></circle></svg>%s</span><span class="text-xs text-warm-faint">agent · %s</span></div><p class="text-sm text-warm-secondary whitespace-pre-wrap">%s</p></div>`, author, date, body)
		} else {
			fmt.Fprintf(w, `<div class="bg-surface rounded-lg border border-edge p-4 space-y-2"><p class="text-sm text-warm-secondary whitespace-pre-wrap">%s</p><div class="flex items-center gap-3 text-xs text-warm-faint"><span>%s</span><span>%s</span></div></div>`, body, author, date)
		}
	}
	if len(answers) == 0 {
		fmt.Fprint(w, `<p class="text-sm text-warm-faint italic">No answers yet.</p>`)
	}
	fmt.Fprint(w, `</div>`)
}

func (h *Handlers) handleCouncil(w http.ResponseWriter, r *http.Request) {
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

	sessions, err := h.store.ListCouncilSessions(r.Context(), space.ID, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "sessions": sessions})
		return
	}

	CouncilListView(*space, spaces, sessions, h.viewUser(r), profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleCouncilDetail(w http.ResponseWriter, r *http.Request) {
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
	id := r.PathValue("id")

	session, err := h.store.GetNode(r.Context(), id)
	if errors.Is(err, ErrNotFound) || (err == nil && session.Kind != KindCouncil) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if session.SpaceID != space.ID {
		http.NotFound(w, r)
		return
	}

	responses, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID:  space.ID,
		ParentID: id,
		Limit:    200,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "session": session, "responses": responses})
		return
	}

	CouncilDetailView(*space, spaces, *session, responses, h.viewUser(r), profile.FromContext(r.Context())).Render(r.Context(), w)
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
	stateFilter := r.URL.Query().Get("state")
	searchQuery := r.URL.Query().Get("q")
	proposals, err := h.store.ListProposals(r.Context(), space.ID, stateFilter, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if searchQuery != "" {
		var filtered []ProposalWithVotes
		sq := strings.ToLower(searchQuery)
		for _, p := range proposals {
			if strings.Contains(strings.ToLower(p.Title), sq) || strings.Contains(strings.ToLower(p.Body), sq) {
				filtered = append(filtered, p)
			}
		}
		proposals = filtered
	}

	actorID := h.userID(r)
	_, delegateName, hasDelegated := h.store.GetUserDelegation(r.Context(), space.ID, actorID)
	delegations, _ := h.store.ListDelegations(r.Context(), space.ID, 20)

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "proposals": proposals})
		return
	}

	GovernanceView(*space, spaces, proposals, h.viewUser(r), isOwner, stateFilter, searchQuery, hasDelegated, delegateName, delegations, profile.FromContext(r.Context())).Render(r.Context(), w)
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
	agentPersona := h.store.GetAgentPersonaForConversation(r.Context(), node.Tags)

	// Mark conversation as read for current user.
	uid := h.userID(r)
	if uid != "" {
		h.store.MarkConversationRead(r.Context(), uid, nodeID)
	}

	// Load reactions for all messages.
	var msgIDs []string
	for _, m := range messages {
		msgIDs = append(msgIDs, m.ID)
	}
	rxnMap := h.store.GetBulkReactions(r.Context(), msgIDs)

	user := h.viewUser(r)
	nameMap := h.store.ResolveUserNames(r.Context(), node.Tags)
	ConversationDetailView(*space, *node, messages, user, h.userID(r), hasAgent, agentPersona, nameMap, rxnMap, profile.FromContext(r.Context())).Render(r.Context(), w)
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
	var msgIDs []string
	for _, m := range messages {
		msgIDs = append(msgIDs, m.ID)
	}
	rxnMap := h.store.GetBulkReactions(r.Context(), msgIDs)
	for _, msg := range messages {
		chatMessage(msg, uid, space.Slug, rxnMap[msg.ID]).Render(r.Context(), w)
	}
}

type nodeActivityEvent struct {
	At      time.Time
	Kind    string
	Actor   string
	Summary string
	Payload json.RawMessage
}

func (h *Handlers) handleNodeActivity(w http.ResponseWriter, r *http.Request) {
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
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if node.SpaceID != space.ID {
		http.NotFound(w, r)
		return
	}
	children, err := h.store.ListNodes(r.Context(), ListNodesParams{SpaceID: space.ID, ParentID: nodeID, Limit: 200})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	visibleChildren, err := h.collectNodeDescendants(r.Context(), space.ID, children, 3, 500)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ops, err := h.store.ListNodeOps(r.Context(), nodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dependencies, _ := h.store.ListDependencies(r.Context(), nodeID)
	blockers, _ := h.store.ListBlockers(r.Context(), nodeID)
	var childOps []Op
	for _, child := range visibleChildren {
		opsForChild, err := h.store.ListNodeOps(r.Context(), child.ID)
		if err == nil {
			childOps = append(childOps, opsForChild...)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderNodeActivityHTML(w, space.Slug, *node, visibleChildren, ops, childOps, dependencies, blockers)
}

func (h *Handlers) collectNodeDescendants(ctx context.Context, spaceID string, roots []Node, maxDepth, limit int) ([]Node, error) {
	if maxDepth <= 1 || len(roots) == 0 {
		return roots, nil
	}
	result := append([]Node{}, roots...)
	frontier := append([]Node{}, roots...)
	seen := make(map[string]bool, len(roots))
	for _, n := range roots {
		seen[n.ID] = true
	}
	for depth := 1; depth < maxDepth && len(frontier) > 0 && len(result) < limit; depth++ {
		var next []Node
		for _, parent := range frontier {
			children, err := h.store.ListNodes(ctx, ListNodesParams{SpaceID: spaceID, ParentID: parent.ID, Limit: 100})
			if err != nil {
				return result, err
			}
			for _, child := range children {
				if seen[child.ID] {
					continue
				}
				seen[child.ID] = true
				result = append(result, child)
				next = append(next, child)
				if len(result) >= limit {
					return result, nil
				}
			}
		}
		frontier = next
	}
	return result, nil
}

func (h *Handlers) handleNodeInvestigateGaps(w http.ResponseWriter, r *http.Request) {
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
	node, err := h.store.GetNode(r.Context(), nodeID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if node.SpaceID != space.ID {
		http.NotFound(w, r)
		return
	}
	if node.State == "deleted" {
		http.Error(w, "cannot investigate a deleted refinery item", http.StatusConflict)
		return
	}

	actor := h.userName(r)
	actorID := h.userID(r)
	actorKind := h.userKind(r)
	task, missing, err := h.ensureRefineryGapInvestigation(r.Context(), *space, *node, actor, actorID, actorKind)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if task != nil {
		agentID := task.AssigneeID
		if agentID != "" {
			h.notify(r.Context(), agentID, actor, "", space.ID, "dispatched a refinery gap investigation to the Hive: "+node.Title)
		}
	}
	if wantsJSON(r) {
		writeJSON(w, http.StatusCreated, map[string]any{"task": task, "missing_gates": missing, "assigned_to": task.Assignee})
		return
	}
	http.Redirect(w, r, "/app/"+space.Slug+"/refinery?q="+url.QueryEscape(node.Title), http.StatusSeeOther)
}

func (h *Handlers) ensureRefineryGapInvestigation(ctx context.Context, space Space, node Node, actor, actorID, actorKind string) (*Node, string, error) {
	agentID, agentName, err := h.store.GetFirstAgent(ctx)
	if err != nil {
		return nil, "", err
	}
	if agentID == "" {
		return nil, "", fmt.Errorf("no agent is available to investigate refinery gaps")
	}

	missing := refineryMissingSections(node.Body)
	if missing == "None detected." {
		missing = "open investigation questions"
	}
	title := "Investigate refinery gaps: " + node.Title
	children, err := h.store.ListNodes(ctx, ListNodesParams{SpaceID: space.ID, ParentID: node.ID, Limit: 200})
	if err != nil {
		return nil, "", err
	}
	for _, child := range children {
		if child.Kind == KindTask && child.State != StateDone && child.State != "deleted" && child.Title == title {
			return &child, missing, nil
		}
	}

	taskBody := buildRefineryGapInvestigationBody(node, missing)
	task, err := h.store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    space.ID,
		ParentID:   node.ID,
		Kind:       KindTask,
		Title:      title,
		Body:       taskBody,
		State:      StateOpen,
		Priority:   PriorityHigh,
		Assignee:   agentName,
		AssigneeID: agentID,
		Author:     actor,
		AuthorID:   actorID,
		AuthorKind: actorKind,
		Causes:     []string{node.ID},
	})
	if err != nil {
		return nil, "", err
	}

	payloadBytes, _ := json.Marshal(map[string]any{
		"source":        "refinery.gap_investigation",
		"kind":          KindTask,
		"priority":      PriorityHigh,
		"description":   taskBody,
		"parent_id":     node.ID,
		"parent_title":  node.Title,
		"task_title":    task.Title,
		"missing_gates": missing,
		"assignee":      agentName,
	})
	_, _ = h.store.RecordOp(ctx, space.ID, task.ID, actor, actorID, "intend", payloadBytes)

	fromState := node.State
	if shouldMoveToInvestigateForGap(fromState) {
		if err := h.store.UpdateNodeState(ctx, node.ID, SpecStateInvestigate); err == nil {
			progressPayload, _ := json.Marshal(map[string]any{
				"from_state": fromState,
				"to_state":   SpecStateInvestigate,
				"reason":     "missing gates were converted into an explicit investigation work order",
				"task_id":    task.ID,
			})
			_, _ = h.store.RecordOp(ctx, space.ID, node.ID, actor, actorID, "progress", progressPayload)
		}
	}
	return task, missing, nil
}

func buildRefineryGapInvestigationBody(node Node, missing string) string {
	return fmt.Sprintf(`## Investigation Trigger
This task was spawned because the parent refinery item has missing or insufficient gates: %s.

## Parent Spec
Title: %s
State at spawn: %s

## Required Investigation Output
1. Inspect the parent spec body and any attached artifacts.
2. Determine whether the missing gates are genuinely absent or only poorly named.
3. Draft concrete content for Definition of Done, Acceptance Criteria, and Test Plan where needed.
4. Identify unresolved ambiguities that must return to Clarify rather than being guessed.
5. Record findings as visible child output under the parent item, with rationale and source evidence.

## Guardrails
Do not treat missing gates as a terminal blocker. Treat them as gap-requirements for investigation. Preserve provenance and make the next recommended FSM state explicit.`, missing, node.Title, node.State)
}

func shouldMoveToInvestigateForGap(state string) bool {
	switch refineryColumnState(state) {
	case SpecStateIntakeRaw, SpecStateRequirementClarify, SpecStateDraft, SpecStateReview, SpecStateNeedsAttention:
		return true
	default:
		return false
	}
}

func renderNodeActivityHTML(w io.Writer, spaceSlug string, node Node, children []Node, ops []Op, childOps []Op, dependencies []Node, blockers []Node) {
	fmt.Fprintf(w, `<div class="space-y-4" data-activity-node="%s">`, html.EscapeString(node.ID))
	fmt.Fprint(w, `<div class="grid grid-cols-2 gap-2 text-xs">`)
	renderActivityStatusCard(w, "Current state", stateLabel(node.State))
	assignee := "No agent or human assigned"
	if strings.TrimSpace(node.Assignee) != "" {
		assignee = node.Assignee
		if node.AssigneeKind != "" {
			assignee += " (" + node.AssigneeKind + ")"
		}
	}
	renderActivityStatusCard(w, "Assigned to", assignee)
	renderActivityStatusCard(w, "Child records", fmt.Sprintf("%d total", len(children)))
	renderActivityStatusCard(w, "Blocking deps", fmt.Sprintf("%d active", len(blockers)))
	fmt.Fprint(w, `</div>`)

	fmt.Fprint(w, `<div class="rounded-xl border border-edge bg-elevated/60 p-3 text-xs">`)
	fmt.Fprint(w, `<p class="font-medium text-warm mb-2">Active work</p>`)
	activeCount := 0
	for _, child := range children {
		if child.Kind == KindTask && child.State != StateDone && child.State != "deleted" {
			activeCount++
			renderActivityChild(w, spaceSlug, child)
		}
	}
	if activeCount == 0 {
		fmt.Fprint(w, `<p class="text-warm-faint">No active work is currently attached. Completed Hive output, artifacts, and historical child records are listed below.</p>`)
	}
	fmt.Fprint(w, `</div>`)

	fmt.Fprint(w, `<div class="rounded-xl border border-edge bg-elevated/60 p-3 text-xs">`)
	fmt.Fprint(w, `<p class="font-medium text-warm mb-2">Artifacts and child records</p>`)
	if len(children) == 0 {
		fmt.Fprint(w, `<p class="text-warm-faint">No child artifacts, tasks, or comments are attached.</p>`)
	} else {
		for _, child := range children {
			renderActivityChild(w, spaceSlug, child)
		}
	}
	fmt.Fprint(w, `</div>`)

	fmt.Fprint(w, `<div class="rounded-xl border border-edge bg-elevated/60 p-3 text-xs">`)
	fmt.Fprint(w, `<p class="font-medium text-warm mb-2">Blockers</p>`)
	missing := refineryMissingSections(node.Body)
	if missing != "None detected." {
		fmt.Fprintf(w, `<p class="mb-2 text-amber-200">Missing spec gates: %s</p>`, html.EscapeString(missing))
	}
	if len(blockers) == 0 && missing == "None detected." {
		fmt.Fprint(w, `<p class="text-warm-faint">No active dependency blockers detected.</p>`)
	}
	for _, blocker := range blockers {
		renderActivityChild(w, spaceSlug, blocker)
	}
	if len(dependencies) > 0 {
		fmt.Fprint(w, `<p class="mt-3 mb-1 text-warm-faint uppercase tracking-wide text-[10px]">All dependencies</p>`)
		for _, dep := range dependencies {
			renderActivityChild(w, spaceSlug, dep)
		}
	}
	fmt.Fprint(w, `</div>`)

	events := make([]nodeActivityEvent, 0, len(ops)+len(childOps))
	for _, op := range ops {
		events = append(events, nodeActivityEvent{At: op.CreatedAt, Kind: op.Op, Actor: op.Actor, Summary: summarizeActivityOp(op), Payload: op.Payload})
	}
	for _, op := range childOps {
		events = append(events, nodeActivityEvent{At: op.CreatedAt, Kind: op.Op, Actor: op.Actor, Summary: summarizeActivityOp(op), Payload: op.Payload})
	}
	sort.Slice(events, func(i, j int) bool { return events[i].At.After(events[j].At) })
	fmt.Fprint(w, `<div class="rounded-xl border border-edge bg-elevated/60 p-3 text-xs">`)
	fmt.Fprint(w, `<p class="font-medium text-warm mb-2">Provenance timeline</p>`)
	if len(events) == 0 {
		fmt.Fprint(w, `<p class="text-warm-faint">No operations recorded for this item yet.</p>`)
	} else {
		fmt.Fprint(w, `<div class="space-y-2">`)
		for _, ev := range events {
			fmt.Fprintf(w, `<div class="border-l border-brand/30 pl-3 py-1"><div class="flex items-center justify-between gap-2"><span class="text-warm">%s</span><span class="text-warm-faint">%s</span></div><p class="text-warm-muted mt-1">%s</p><p class="text-[10px] text-warm-faint mt-1">%s</p></div>`, html.EscapeString(ev.Kind), html.EscapeString(ev.At.Format("Jan 2 15:04:05")), html.EscapeString(ev.Summary), html.EscapeString(nonEmpty(ev.Actor, "system")))
		}
		fmt.Fprint(w, `</div>`)
	}
	fmt.Fprint(w, `</div>`)
	fmt.Fprint(w, `</div>`)
}

func renderActivityStatusCard(w io.Writer, label, value string) {
	fmt.Fprintf(w, `<div class="rounded-lg border border-edge bg-surface p-2"><p class="text-warm-faint uppercase tracking-wide text-[10px]">%s</p><p class="text-warm mt-1">%s</p></div>`, html.EscapeString(label), html.EscapeString(value))
}

func renderActivityChild(w io.Writer, spaceSlug string, child Node) {
	title := child.Title
	if title == "" {
		title = truncatePlain(child.Body, 90)
	}
	if title == "" {
		title = child.Kind
	}
	actor := child.Author
	if child.Assignee != "" {
		actor = "assigned to " + child.Assignee
	}
	fmt.Fprintf(w, `<a href="/app/%s/node/%s" class="block rounded-lg border border-edge bg-surface px-3 py-2 mb-2 hover:border-brand/40"><div class="flex items-center justify-between gap-2"><span class="text-warm">%s</span><span class="text-warm-faint">%s</span></div>`, html.EscapeString(spaceSlug), html.EscapeString(child.ID), html.EscapeString(title), html.EscapeString(stateLabel(child.State)))
	if child.Kind == KindComment && strings.TrimSpace(child.Body) != "" {
		fmt.Fprintf(w, `<p class="mt-1 text-[11px] leading-relaxed text-warm-muted">%s</p>`, html.EscapeString(truncatePlain(child.Body, 260)))
	}
	fmt.Fprintf(w, `<div class="flex items-center gap-2 mt-1"><span class="text-[10px] px-1.5 py-0.5 rounded bg-brand/10 text-brand">%s</span><span class="text-[10px] text-warm-faint">%s</span></div></a>`, html.EscapeString(child.Kind), html.EscapeString(actor))
}

func summarizeActivityOp(op Op) string {
	if len(op.Payload) > 0 && string(op.Payload) != "null" {
		var m map[string]any
		if json.Unmarshal(op.Payload, &m) == nil {
			switch op.Op {
			case "intake":
				return fmt.Sprintf("Classified as %s; recommended %s. %s", valueString(m["classifier_kind"]), valueString(m["recommended_state"]), valueString(m["rationale"]))
			case "attach":
				return fmt.Sprintf("Attached %s with %s.", valueString(m["filename"]), valueString(m["hash"]))
			case "progress", "review", "complete":
				return fmt.Sprintf("Moved from %s to %s. %s", valueString(m["from_state"]), valueString(m["to_state"]), valueString(m["reason"]))
			case "delete":
				return fmt.Sprintf("Soft-delete transition from %s to %s. %s", valueString(m["from_state"]), valueString(m["to_state"]), valueString(m["reason"]))
			case "intend":
				if valueString(m["source"]) == "refinery.gap_investigation" {
					return fmt.Sprintf("Started gap investigation task %q for missing gates: %s. Assigned to %s.", valueString(m["task_title"]), valueString(m["missing_gates"]), valueString(m["assignee"]))
				}
			case "hive_mirror":
				return fmt.Sprintf("Mirrored %s from Hive/EventGraph. Hive task: %s. State: %s.", valueString(m["event_type"]), valueString(m["hive_task_id"]), valueString(m["state"]))
			case "respond":
				return "Recorded a response/comment attached to this item."
			}
		}
	}
	return "Recorded " + op.Op + " operation."
}

func valueString(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
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

	// Fetch dependencies (what this node depends on) and dependents (what depends on this).
	dependencies, _ := h.store.ListDependencies(r.Context(), nodeID)
	dependents, _ := h.store.ListDependents(r.Context(), nodeID)

	// Fetch space tasks for dependency dropdown (tasks only, exclude self + existing deps).
	var spaceTasks []Node
	if node.Kind == KindTask {
		all, _ := h.store.ListNodes(r.Context(), ListNodesParams{SpaceID: space.ID, Kind: KindTask})
		depSet := make(map[string]bool, len(dependencies))
		depSet[nodeID] = true // exclude self
		for _, d := range dependencies {
			depSet[d.ID] = true
		}
		for _, t := range all {
			if !depSet[t.ID] {
				spaceTasks = append(spaceTasks, t)
			}
		}
	}

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"space": space, "node": node, "children": children, "ops": ops, "dependencies": dependencies, "dependents": dependents})
		return
	}

	// Build parent chain for breadcrumbs.
	var parents []Node
	cur := node
	for cur.ParentID != "" && len(parents) < 5 {
		p, err := h.store.GetNode(r.Context(), cur.ParentID)
		if err != nil || p == nil {
			break
		}
		parents = append(parents, *p)
		cur = p
	}
	// Reverse so root parent is first.
	for i, j := 0, len(parents)-1; i < j; i, j = i+1, j-1 {
		parents[i], parents[j] = parents[j], parents[i]
	}

	// Engagement data for the node.
	uid := h.userID(r)
	endorseCount := h.store.CountEndorsements(r.Context(), nodeID)
	endorsed := uid != "" && h.store.HasEndorsed(r.Context(), uid, nodeID)
	repostCounts := h.store.GetBulkRepostCounts(r.Context(), []string{nodeID})
	reposted := uid != "" && h.store.HasReposted(r.Context(), uid, nodeID)

	NodeDetailView(*space, *node, children, ops, h.viewUser(r), isOwner, parents, dependencies, dependents, spaceTasks, endorseCount, endorsed, repostCounts[nodeID], reposted, profile.FromContext(r.Context())).Render(r.Context(), w)
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
		var dueDate *time.Time
		if ds := r.FormValue("due_date"); ds != "" {
			if t, err := time.Parse("2006-01-02", ds); err == nil {
				dueDate = &t
			}
		}
		nodeKind := r.FormValue("kind")
		if nodeKind != KindProject && nodeKind != KindGoal && nodeKind != KindRole && nodeKind != KindTeam && nodeKind != KindPolicy && nodeKind != KindDocument && nodeKind != KindQuestion && nodeKind != KindProposal && nodeKind != KindSpec {
			nodeKind = KindTask // default
		}
		var intentCauses []string
		if causesStr := r.FormValue("causes"); causesStr != "" {
			for _, c := range strings.Split(causesStr, ",") {
				if c = strings.TrimSpace(c); c != "" {
					intentCauses = append(intentCauses, c)
				}
			}
		}
		intentBody := strings.TrimSpace(r.FormValue("body"))
		if intentBody == "" {
			intentBody = strings.TrimSpace(r.FormValue("description"))
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			ParentID:   strings.TrimSpace(r.FormValue("parent_id")),
			Kind:       nodeKind,
			Title:      title,
			Body:       intentBody,
			Priority:   r.FormValue("priority"),
			Assignee:   assigneeName,
			AssigneeID: assigneeID,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
			DueDate:    dueDate,
			Causes:     intentCauses,
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

		// Trigger Mind to auto-answer questions created via intend.
		if h.mind != nil && nodeKind == KindQuestion {
			go h.mind.OnQuestionAsked(space.ID, space.Slug, node)
		}

		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "intend"})
			return
		}
		if isHTMX(r) {
			TaskCard(*node, space.Slug).Render(ctx, w)
			return
		}
		boardURL := "/app/" + space.Slug + "/board"
		if assigneeID != "" {
			if isAgent, _ := h.store.HasAgentParticipant(ctx, []string{assigneeID}); isAgent {
				boardURL += "?aha_agent=1"
			}
		}
		http.Redirect(w, r, boardURL, http.StatusSeeOther)

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

		// express with kind=question creates a KindQuestion and triggers auto-answer.
		if r.FormValue("kind") == KindQuestion {
			node, err := h.store.CreateNode(ctx, CreateNodeParams{
				SpaceID:    space.ID,
				Kind:       KindQuestion,
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

			if h.mind != nil {
				go h.mind.OnQuestionAsked(space.ID, space.Slug, node)
			}

			if wantsJSON(r) {
				writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "express"})
				return
			}
			http.Redirect(w, r, fmt.Sprintf("/app/%s/questions/%s", space.Slug, node.ID), http.StatusSeeOther)
			return
		}

		quoteOfID := strings.TrimSpace(r.FormValue("quote_of_id"))
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindPost,
			Title:      title,
			Body:       body,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
			QuoteOfID:  quoteOfID,
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
			FeedCard(*node, space.Slug, 0, false, 0, false, "").Render(ctx, w)
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
		replyToID := strings.TrimSpace(r.FormValue("reply_to_id"))
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			ParentID:   parentID,
			Kind:       KindComment,
			Body:       body,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
			ReplyToID:  replyToID,
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
				h.store.UpdateLastMessagePreview(ctx, parentID, body)
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
				chatMessage(*node, actorID, space.Slug, nil).Render(ctx, w)
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

	case "claim":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		if err := h.store.ClaimNode(ctx, nodeID, actor, actorID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		op, _ := h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "claim", nil)
		// Notify task author.
		if op != nil {
			if node, _ := h.store.GetNode(ctx, nodeID); node != nil && node.AuthorID != actorID {
				h.notify(ctx, node.AuthorID, actor, op.ID, space.ID, "claimed your task: "+node.Title)
			}
		}
		// Trigger Mind if an agent claims.
		if h.mind != nil {
			if node, _ := h.store.GetNode(ctx, nodeID); node != nil {
				go h.mind.OnTaskAssigned(space.ID, space.Slug, node, actorID)
			}
		}
		if wantsJSON(r) {
			node, _ := h.store.GetNode(ctx, nodeID)
			writeJSON(w, http.StatusOK, map[string]any{"node": node, "op": "claim"})
			return
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)

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

		// Notify task author/assignee when someone completes their task.
		if op != nil {
			if node, _ := h.store.GetNode(ctx, nodeID); node != nil {
				if node.AuthorID != actorID {
					h.notify(ctx, node.AuthorID, actor, op.ID, space.ID, "completed your task: "+node.Title)
				}
				if node.AssigneeID != "" && node.AssigneeID != actorID && node.AssigneeID != node.AuthorID {
					h.notify(ctx, node.AssigneeID, actor, op.ID, space.ID, "completed task: "+node.Title)
				}
			}
		}

		// Recompute reputation for the actor who completed the task.
		h.store.ComputeAndUpdateReputation(ctx, actorID)

		// Track first completion in this space.
		isFirstCompletion, _ := h.store.MarkFirstCompletion(ctx, space.ID)

		if wantsJSON(r) {
			node, _ := h.store.GetNode(ctx, nodeID)
			writeJSON(w, http.StatusOK, map[string]any{"node": node, "op": "complete", "first_completion": isFirstCompletion})
			return
		}

		node, _ := h.store.GetNode(ctx, nodeID)
		if isHTMX(r) && node != nil {
			TaskCard(*node, space.Slug).Render(ctx, w)
			return
		}
		boardURL := "/app/" + space.Slug + "/board"
		if isFirstCompletion {
			boardURL += "?first_completion=1"
		}
		http.Redirect(w, r, boardURL, http.StatusSeeOther)

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
		// Route agent conversations to the agents space.
		// Use identity system (users table, kind='agent') — not content heuristics.
		targetSpace := space
		if hasAgent, err := h.store.HasAgentParticipant(ctx, participants); err != nil {
			log.Printf("converse: check agent participants: %v", err)
		} else if hasAgent {
			agentsSpace, err := h.store.GetSpaceBySlug(ctx, AgentsSpaceSlug)
			if err != nil {
				log.Printf("converse: get agents space %q: %v", AgentsSpaceSlug, err)
			} else if agentsSpace != nil {
				targetSpace = agentsSpace
			}
		}

		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    targetSpace.ID,
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
		h.store.RecordOp(ctx, targetSpace.ID, node.ID, actor, actorID, "converse", nil)

		// Trigger Mind if a human created a conversation with an agent.
		if h.mind != nil && actorKind != "agent" {
			go h.mind.OnMessage(targetSpace.ID, targetSpace.Slug, node, actorID)
		}

		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "converse"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/conversation/%s", targetSpace.Slug, node.ID), http.StatusSeeOther)

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

	case OpJoinTeam:
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		if actorID == "" {
			http.Error(w, "must be logged in to join a team", http.StatusUnauthorized)
			return
		}
		if !h.store.IsMember(ctx, space.ID, actorID) {
			http.Error(w, "must be a space member to join a team", http.StatusForbidden)
			return
		}
		if err := h.store.JoinNodeMember(ctx, nodeID, actorID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, OpJoinTeam, nil)
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": OpJoinTeam, "status": "joined"})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/teams", http.StatusSeeOther)

	case OpLeaveTeam:
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		if actorID == "" {
			http.Error(w, "must be logged in to leave a team", http.StatusUnauthorized)
			return
		}
		// Self or space owner can remove.
		targetID := r.FormValue("user_id")
		if targetID == "" {
			targetID = actorID
		}
		if targetID != actorID && space.OwnerID != actorID {
			http.Error(w, "only space owner can remove others from a team", http.StatusForbidden)
			return
		}
		if err := h.store.LeaveNodeMember(ctx, nodeID, targetID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, OpLeaveTeam, nil)
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": OpLeaveTeam, "status": "left"})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/teams", http.StatusSeeOther)

	case "kick":
		memberID := r.FormValue("member_id")
		if memberID == "" {
			http.Error(w, "member_id required", http.StatusBadRequest)
			return
		}
		if space.OwnerID != actorID {
			http.Error(w, "only space owner can remove members", http.StatusForbidden)
			return
		}
		if memberID == actorID {
			http.Error(w, "cannot remove yourself", http.StatusBadRequest)
			return
		}
		if err := h.store.LeaveSpace(ctx, space.ID, memberID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, "", actor, actorID, "kick", nil)
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "kick"})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/settings", http.StatusSeeOther)

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

	case "undepend":
		nodeID := r.FormValue("node_id")
		dependsOn := r.FormValue("depends_on")
		if nodeID == "" || dependsOn == "" {
			http.Error(w, "node_id and depends_on required", http.StatusBadRequest)
			return
		}
		if err := h.store.RemoveDependency(ctx, nodeID, dependsOn); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "undepend"})
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
		var causes []string
		if causesStr := r.FormValue("causes"); causesStr != "" {
			for _, c := range strings.Split(causesStr, ",") {
				if c = strings.TrimSpace(c); c != "" {
					causes = append(causes, c)
				}
			}
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
			Causes:     causes,
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
		op, _ := h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "challenge", payload)

		// Notify claim author.
		if node.AuthorID != actorID && op != nil {
			h.notify(ctx, node.AuthorID, actor, op.ID, space.ID, "challenged your claim: "+node.Title)
		}

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
		reason := strings.TrimSpace(r.FormValue("reason"))
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
		var payload json.RawMessage
		if reason != "" {
			payload, _ = json.Marshal(map[string]string{"reason": reason})
		}
		op, _ := h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "verify", payload)
		if node.AuthorID != actorID && op != nil {
			h.notify(ctx, node.AuthorID, actor, op.ID, space.ID, "verified your claim: "+node.Title)
		}
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "verify"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "retract":
		nodeID := r.FormValue("node_id")
		reason := strings.TrimSpace(r.FormValue("reason"))
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
		var payload json.RawMessage
		if reason != "" {
			payload, _ = json.Marshal(map[string]string{"reason": reason})
		}
		op, _ := h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "retract", payload)
		if node.AuthorID != actorID && op != nil {
			h.notify(ctx, node.AuthorID, actor, op.ID, space.ID, "retracted your claim: "+node.Title)
		}
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

	case "edit":
		nodeID := r.FormValue("node_id")
		body := strings.TrimSpace(r.FormValue("body"))
		causesStr := r.FormValue("causes")
		if nodeID == "" || (body == "" && causesStr == "") {
			http.Error(w, "node_id and body or causes required", http.StatusBadRequest)
			return
		}
		// Only the author can edit.
		node, err := h.store.GetNode(ctx, nodeID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if node.AuthorID != actorID {
			http.Error(w, "only the author can edit", http.StatusForbidden)
			return
		}
		if body != "" {
			oldBody := node.Body
			if err := h.store.EditNodeBody(ctx, nodeID, body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "edit", json.RawMessage(`{"old_body":`+strconv.Quote(oldBody)+`}`))
		}
		if causesStr != "" {
			var causes []string
			for _, c := range strings.Split(causesStr, ",") {
				if c = strings.TrimSpace(c); c != "" {
					causes = append(causes, c)
				}
			}
			if err := h.store.UpdateNodeCauses(ctx, nodeID, causes); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			meta, _ := json.Marshal(map[string]string{"causes": causesStr})
			h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "edit", json.RawMessage(meta))
		}
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "edit"})
			return
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)

	case "delete":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		// Only the author can delete.
		node, err := h.store.GetNode(ctx, nodeID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if node.AuthorID != actorID {
			http.Error(w, "only the author can delete", http.StatusForbidden)
			return
		}
		if err := h.store.SoftDeleteNode(ctx, nodeID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "delete", json.RawMessage(`{"old_body":`+strconv.Quote(node.Body)+`}`))
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "delete"})
			return
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)

	case "react":
		nodeID := r.FormValue("node_id")
		emoji := r.FormValue("emoji")
		if nodeID == "" || emoji == "" {
			http.Error(w, "node_id and emoji required", http.StatusBadRequest)
			return
		}
		added, err := h.store.ToggleReaction(ctx, nodeID, actorID, emoji)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if added {
			h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "react", json.RawMessage(`{"emoji":"`+emoji+`"}`))
		}
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]any{"op": "react", "added": added, "emoji": emoji})
			return
		}
		if isHTMX(r) {
			reactions := h.store.GetNodeReactions(ctx, nodeID)
			reactionBar(nodeID, space.Slug, reactions, actorID).Render(ctx, w)
			return
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)

	case "endorse":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		// Toggle: endorse if not endorsed, unendorse if already endorsed.
		endorsed := h.store.HasEndorsed(ctx, actorID, nodeID)
		if endorsed {
			h.store.Unendorse(ctx, actorID, nodeID)
		} else {
			h.store.Endorse(ctx, actorID, nodeID)
			h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "endorse", nil)
			// Notify the post author.
			if node, err := h.store.GetNode(ctx, nodeID); err == nil && node.AuthorID != actorID {
				h.notify(ctx, node.AuthorID, actor, nodeID, space.ID, "endorsed your post")
			}
		}
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]any{"op": "endorse", "endorsed": !endorsed})
			return
		}
		if isHTMX(r) {
			count := h.store.CountEndorsements(ctx, nodeID)
			endorseButton(nodeID, space.Slug, count, !endorsed).Render(ctx, w)
			return
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)

	case "repost":
		nodeID := r.FormValue("node_id")
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		reposted := h.store.HasReposted(ctx, actorID, nodeID)
		if reposted {
			h.store.Unrepost(ctx, actorID, nodeID)
		} else {
			h.store.Repost(ctx, actorID, nodeID)
			h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "repost", nil)
			if node, err := h.store.GetNode(ctx, nodeID); err == nil && node.AuthorID != actorID {
				h.notify(ctx, node.AuthorID, actor, nodeID, space.ID, "reposted your post")
			}
		}
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]any{"op": "repost", "reposted": !reposted})
			return
		}
		if isHTMX(r) {
			counts := h.store.GetBulkRepostCounts(ctx, []string{nodeID})
			repostButton(nodeID, space.Slug, counts[nodeID], !reposted).Render(ctx, w)
			return
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)

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
		params := CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindProposal,
			Title:      title,
			Body:       strings.TrimSpace(r.FormValue("description")),
			State:      ProposalOpen,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
		}
		if dl := r.FormValue("deadline"); dl != "" {
			if t, err := time.Parse("2006-01-02", dl); err == nil {
				params.DueDate = &t
			}
		}
		node, err := h.store.CreateNode(ctx, params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Optional quorum configuration.
		if qp := r.FormValue("quorum_pct"); qp != "" {
			var pct int
			if _, err := fmt.Sscanf(qp, "%d", &pct); err == nil && pct > 0 && pct <= 100 {
				vb := r.FormValue("voting_body")
				if vb == "" {
					vb = VotingBodyAll
				}
				h.store.SetProposalConfig(ctx, node.ID, pct, vb)
			}
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
		// Delegated users cannot vote directly — they must undelegate first.
		if h.store.HasDelegated(ctx, space.ID, actorID) {
			http.Error(w, "you have delegated your vote; undelegate before voting directly", http.StatusConflict)
			return
		}
		// One vote per user per proposal.
		if h.store.HasVoted(ctx, nodeID, actorID) {
			http.Error(w, "already voted", http.StatusConflict)
			return
		}
		payload, _ := json.Marshal(map[string]string{"vote": vote})
		op, _ := h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "vote", payload)

		// Notify proposal author.
		if node.AuthorID != actorID && op != nil {
			h.notify(ctx, node.AuthorID, actor, op.ID, space.ID, "voted "+vote+" on your proposal: "+node.Title)
		}

		// Auto-close proposal if quorum is now met.
		h.store.CheckAndAutoCloseProposal(ctx, space.ID, nodeID)

		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "vote", "vote": vote})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/governance", http.StatusSeeOther)

	case "progress":
		// Submit work for review: active → review, with optional progress note.
		nodeID := r.FormValue("node_id")
		body := strings.TrimSpace(r.FormValue("body"))
		if nodeID == "" {
			http.Error(w, "node_id required", http.StatusBadRequest)
			return
		}
		node, err := h.store.GetNode(ctx, nodeID)
		if err != nil || node == nil {
			http.NotFound(w, r)
			return
		}
		if node.Kind != KindTask {
			http.Error(w, "progress op only valid for tasks", http.StatusBadRequest)
			return
		}
		if node.State != StateActive {
			http.Error(w, "task must be in active state to submit for review", http.StatusBadRequest)
			return
		}
		if err := h.store.UpdateNodeState(ctx, nodeID, StateReview); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var progressPayload []byte
		if body != "" {
			progressPayload, _ = json.Marshal(map[string]string{"body": body})
		}
		op, _ := h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "progress", progressPayload)
		// Notify task author that work is ready for review.
		if op != nil && node.AuthorID != actorID {
			h.notify(ctx, node.AuthorID, actor, op.ID, space.ID, "submitted for review: "+node.Title)
		}
		if wantsJSON(r) {
			node, _ := h.store.GetNode(ctx, nodeID)
			writeJSON(w, http.StatusOK, map[string]any{"node": node, "op": "progress"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

	case "review":
		// Structured review with verdict: approve / revise / reject.
		nodeID := r.FormValue("node_id")
		verdict := r.FormValue("verdict") // "approve", "revise", "reject"
		body := strings.TrimSpace(r.FormValue("body"))
		if nodeID == "" || (verdict != "approve" && verdict != "revise" && verdict != "reject") {
			http.Error(w, "node_id and verdict (approve/revise/reject) required", http.StatusBadRequest)
			return
		}
		node, err := h.store.GetNode(ctx, nodeID)
		if err != nil || node == nil {
			http.NotFound(w, r)
			return
		}
		if node.Kind != KindTask {
			http.Error(w, "review op only valid for tasks", http.StatusBadRequest)
			return
		}
		if node.State != StateReview {
			http.Error(w, "task must be in review state", http.StatusBadRequest)
			return
		}
		var newState string
		switch verdict {
		case "approve":
			newState = StateDone
		case "revise":
			newState = StateActive
		default: // "reject"
			newState = StateClosed
		}
		if err := h.store.UpdateNodeState(ctx, nodeID, newState); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reviewPayload, _ := json.Marshal(map[string]string{"verdict": verdict, "body": body})
		op, _ := h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "review", reviewPayload)
		// Notify assignee of review verdict.
		if op != nil && node.AssigneeID != "" && node.AssigneeID != actorID {
			var msg string
			switch verdict {
			case "approve":
				msg = "approved your task: " + node.Title
			case "revise":
				msg = "requested revisions on: " + node.Title
			default:
				msg = "rejected your task: " + node.Title
			}
			h.notify(ctx, node.AssigneeID, actor, op.ID, space.ID, msg)
		}
		// Recompute reputation for the task's assignee (the worker being reviewed).
		if node.AssigneeID != "" {
			h.store.ComputeAndUpdateReputation(ctx, node.AssigneeID)
		}

		if wantsJSON(r) {
			node, _ := h.store.GetNode(ctx, nodeID)
			writeJSON(w, http.StatusOK, map[string]any{"node": node, "op": "review", "verdict": verdict})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/node/%s", space.Slug, nodeID), http.StatusSeeOther)

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
		op, _ := h.store.RecordOp(ctx, space.ID, nodeID, actor, actorID, "close_proposal", payload)

		// Notify proposal author.
		if node.AuthorID != actorID && op != nil {
			h.notify(ctx, node.AuthorID, actor, op.ID, space.ID, outcome+" your proposal: "+node.Title)
		}

		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": "close_proposal", "outcome": outcome})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/governance", http.StatusSeeOther)

	case "delegate":
		delegateID := strings.TrimSpace(r.FormValue("delegate_id"))
		if delegateID == "" {
			http.Error(w, "delegate_id required", http.StatusBadRequest)
			return
		}
		if err := h.store.Delegate(ctx, space.ID, actorID, delegateID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		payload, _ := json.Marshal(map[string]string{"delegate_id": delegateID})
		h.store.RecordOp(ctx, space.ID, "", actor, actorID, OpDelegate, payload)
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": OpDelegate, "delegate_id": delegateID})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/governance", http.StatusSeeOther)

	case "undelegate":
		if err := h.store.Undelegate(ctx, space.ID, actorID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, "", actor, actorID, OpUndelegate, nil)
		if wantsJSON(r) {
			writeJSON(w, http.StatusOK, map[string]string{"op": OpUndelegate})
			return
		}
		http.Redirect(w, r, "/app/"+space.Slug+"/governance", http.StatusSeeOther)

	case "convene":
		title := strings.TrimSpace(r.FormValue("title"))
		body := strings.TrimSpace(r.FormValue("body"))
		if title == "" {
			http.Error(w, "title required", http.StatusBadRequest)
			return
		}
		// Resolve agent IDs from the agents field (comma-separated names or IDs).
		var agentIDs []string
		if a := strings.TrimSpace(r.FormValue("agents")); a != "" {
			for _, name := range strings.Split(a, ",") {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				if resolved := h.store.ResolveUserID(ctx, name); resolved != "" {
					agentIDs = append(agentIDs, resolved)
				}
				// Unresolvable names are skipped — storing a display name where an ID
				// is required violates invariant 11 (IDENTITY).
			}
		}
		node, err := h.store.CreateNode(ctx, CreateNodeParams{
			SpaceID:    space.ID,
			Kind:       KindCouncil,
			Title:      title,
			Body:       body,
			Author:     actor,
			AuthorID:   actorID,
			AuthorKind: actorKind,
			Tags:       agentIDs,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		h.store.RecordOp(ctx, space.ID, node.ID, actor, actorID, "convene", nil)

		// Trigger Mind to call each tagged agent persona asynchronously.
		if h.mind != nil {
			go h.mind.OnCouncilConvened(space.ID, space.Slug, node)
		}

		if wantsJSON(r) {
			writeJSON(w, http.StatusCreated, map[string]any{"node": node, "op": "convene"})
			return
		}
		http.Redirect(w, r, fmt.Sprintf("/app/%s/council/%s", space.Slug, node.ID), http.StatusSeeOther)

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
	statePayload, _ := json.Marshal(map[string]string{
		"from_state": node.State,
		"to_state":   newState,
		"reason":     strings.TrimSpace(r.FormValue("reason")),
	})
	op, _ := h.store.RecordOp(r.Context(), space.ID, nodeID, h.userName(r), h.userID(r), opName, statePayload)

	// Notify assignee on state change.
	uid := h.userID(r)
	if op != nil && node != nil && node.AssigneeID != "" && node.AssigneeID != uid {
		h.notify(r.Context(), node.AssigneeID, h.userName(r), op.ID, space.ID, opName+" task: "+node.Title)
	}

	// Track first completion in this space.
	isFirstCompletion := false
	if newState == StateDone {
		isFirstCompletion, _ = h.store.MarkFirstCompletion(r.Context(), space.ID)
	}

	node, _ = h.store.GetNode(r.Context(), nodeID)
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"node": node, "op": opName, "first_completion": isFirstCompletion})
		return
	}
	if isHTMX(r) && node != nil {
		TaskCard(*node, space.Slug).Render(r.Context(), w)
		return
	}
	boardURL := "/app/" + space.Slug + "/board"
	if isFirstCompletion {
		boardURL += "?first_completion=1"
	}
	http.Redirect(w, r, boardURL, http.StatusSeeOther)
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
	populateFormFromJSON(r)
	_ = r.ParseForm()
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
	node, err := h.store.GetNode(r.Context(), nodeID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if node.SpaceID != space.ID {
		http.Error(w, "node does not belong to requested space", http.StatusBadRequest)
		return
	}
	if err := h.store.SoftDeleteNodePreservingContent(r.Context(), nodeID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reason := strings.TrimSpace(r.FormValue("reason"))
	if reason == "" {
		reason = strings.TrimSpace(r.URL.Query().Get("reason"))
	}
	payload, _ := json.Marshal(map[string]string{
		"from_state": node.State,
		"to_state":   "deleted",
		"reason":     reason,
		"mode":       "soft_delete_preserve_content",
	})
	op, _ := h.store.RecordOp(r.Context(), space.ID, nodeID, h.userName(r), h.userID(r), "delete", payload)

	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"deleted": nodeID, "soft_delete": true, "op": op})
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

type RefineryLineage struct {
	RootID                   string
	LatestVersion            string
	LatestVersionID          string
	LatestTitle              string
	LatestState              string
	LatestMissing            string
	PreviousVersion          string
	VersionCount             int
	ActiveInvestigationID    string
	ActiveInvestigationTitle string
	MovementSummary          string
}

func groupByState(nodes []Node) []BoardColumn {
	columns := []BoardColumn{
		{State: StateOpen, Label: "Open"},
		{State: StateActive, Label: "Active"},
		{State: StateBlocked, Label: "Blocked"},
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
	// Done column: newest completions first (updated_at DESC).
	if done, ok := byState[StateDone]; ok {
		sort.Slice(done.Nodes, func(i, j int) bool {
			return done.Nodes[i].UpdatedAt.After(done.Nodes[j].UpdatedAt)
		})
	}
	return columns
}

func groupSpecsByRefineryStage(nodes []Node) []BoardColumn {
	columns := []BoardColumn{
		{State: SpecStateIntakeRaw, Label: "Intake"},
		{State: SpecStateRequirementClarify, Label: "Clarify"},
		{State: SpecStateInvestigate, Label: "Investigate"},
		{State: SpecStateDraft, Label: "Draft Spec"},
		{State: SpecStateReview, Label: "Review"},
		{State: SpecStateNormative, Label: "Normative"},
		{State: SpecStateNeedsAttention, Label: "Needs Attention"},
		{State: SpecStateWorkReady, Label: "Build"},
		{State: SpecStateParked, Label: "Parked"},
		{State: SpecStateWorkShipped, Label: "Shipped"},
	}
	byState := map[string]*BoardColumn{}
	for i := range columns {
		byState[columns[i].State] = &columns[i]
	}
	for _, n := range nodes {
		key := refineryColumnState(n.State)
		if col, ok := byState[key]; ok {
			col.Nodes = append(col.Nodes, n)
		}
	}
	return columns
}

func (h *Handlers) refineryLineageByRoot(ctx context.Context, roots []Node) map[string]RefineryLineage {
	result := make(map[string]RefineryLineage, len(roots))
	for _, root := range roots {
		children, err := h.store.ListNodes(ctx, ListNodesParams{SpaceID: root.SpaceID, ParentID: root.ID, Limit: 500})
		if err != nil {
			continue
		}
		result[root.ID] = buildRefineryLineage(root, children)
	}
	return result
}

func buildRefineryLineage(root Node, children []Node) RefineryLineage {
	lineage := RefineryLineage{
		RootID:          root.ID,
		LatestMissing:   refineryMissingSections(root.Body),
		MovementSummary: "No SemVer refinement yet",
	}

	var latest *Node
	var latestVersion semverParts
	versionCount := 0
	for i := range children {
		if children[i].Kind != KindSpec {
			continue
		}
		version, ok := parseRefineryVersion(children[i])
		if !ok {
			continue
		}
		versionCount++
		if latest == nil || compareSemver(version, latestVersion) > 0 {
			latest = &children[i]
			latestVersion = version
		}
	}

	lineage.VersionCount = versionCount
	if latest != nil {
		lineage.LatestVersion = latestVersion.raw
		lineage.LatestVersionID = latest.ID
		lineage.LatestTitle = latest.Title
		lineage.LatestState = latest.State
		lineage.LatestMissing = refineryMissingSections(latest.Body)
		if previous := previousRefineryVersion(children, latestVersion); previous != nil {
			if parsed, ok := parseRefineryVersion(*previous); ok {
				lineage.PreviousVersion = parsed.raw
			}
		}
		lineage.MovementSummary = fmt.Sprintf("SemVer advanced to v%s (%s)", latestVersion.raw, refineryStateLabel(latest.State))
		if lineage.LatestMissing != "None detected." {
			lineage.MovementSummary += "; gaps remain: " + lineage.LatestMissing
		}
	}

	for i := range children {
		child := children[i]
		if child.Kind != KindTask || child.State == StateDone || child.State == "deleted" {
			continue
		}
		if latest != nil && containsString(child.Causes, latest.ID) {
			lineage.ActiveInvestigationID = child.ID
			lineage.ActiveInvestigationTitle = child.Title
			break
		}
		if latest == nil && strings.HasPrefix(child.Title, "Investigate refinery gaps:") {
			lineage.ActiveInvestigationID = child.ID
			lineage.ActiveInvestigationTitle = child.Title
			break
		}
	}
	if lineage.ActiveInvestigationTitle != "" {
		lineage.MovementSummary += "; active investigation: " + lineage.ActiveInvestigationTitle
	}
	return lineage
}

func previousRefineryVersion(children []Node, latest semverParts) *Node {
	var previous *Node
	var previousVersion semverParts
	for i := range children {
		if children[i].Kind != KindSpec {
			continue
		}
		version, ok := parseRefineryVersion(children[i])
		if !ok || compareSemver(version, latest) >= 0 {
			continue
		}
		if previous == nil || compareSemver(version, previousVersion) > 0 {
			previous = &children[i]
			previousVersion = version
		}
	}
	return previous
}

func refineryColumnState(state string) string {
	switch state {
	case SpecStateIntakeRaw, SpecStateIntakeTriaged, StateOpen, "":
		return SpecStateIntakeRaw
	case SpecStateRequirementDraft, SpecStateRequirementClarify:
		return SpecStateRequirementClarify
	case SpecStateInvestigate:
		return SpecStateInvestigate
	case SpecStateDraft:
		return SpecStateDraft
	case SpecStateReview, StateReview:
		return SpecStateReview
	case SpecStateNormative:
		return SpecStateNormative
	case SpecStateNeedsAttention:
		return SpecStateNeedsAttention
	case SpecStateParked:
		return SpecStateParked
	case SpecStateWorkReady, SpecStateWorkImplementing, SpecStateWorkVerifying, StateActive, StateBlocked:
		return SpecStateWorkReady
	case SpecStateWorkShipped, SpecStateWorkLearned, StateDone, StateClosed:
		return SpecStateWorkShipped
	default:
		return SpecStateIntakeRaw
	}
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

// maxHivePosts is the upper bound on posts fetched and processed by the /hive
// dashboard. Invariant 13 (BOUNDED): every operation has defined scope.
const maxHivePosts = 20

// activeRoleThreshold is the window within which a pipeline role is considered
// active. A post within this window sets the role's Active = true.
const activeRoleThreshold = 30 * time.Minute

// HiveStats holds aggregated metrics parsed from hive agent post bodies.
type HiveStats struct {
	Features  int
	TotalCost float64
	AvgCost   float64
}

var (
	reCost     = regexp.MustCompile(`\$(\d+\.\d+)`)
	reDuration = regexp.MustCompile(`Duration:\s*(\d+m(?:\d+s)?)`)
)

// parseCostDollars extracts the first dollar amount from a post body (e.g. "Cost: $0.53").
func parseCostDollars(body string) float64 {
	m := reCost.FindStringSubmatch(body)
	if m == nil {
		return 0
	}
	v, _ := strconv.ParseFloat(m[1], 64)
	return v
}

// parseDurationStr extracts the duration string from a post body (e.g. "Duration: 3m28s" → "3m28s").
func parseDurationStr(body string) string {
	m := reDuration.FindStringSubmatch(body)
	if m == nil {
		return ""
	}
	return m[1]
}

// hiveCostStr returns a formatted cost string for a node's body, e.g. "$0.42".
// Returns empty string if no cost is found or cost is zero.
func hiveCostStr(n Node) string {
	c := parseCostDollars(n.Body)
	if c <= 0 {
		return ""
	}
	return fmt.Sprintf("$%.2f", c)
}

// hiveDurationStr returns the duration string from a node's body, e.g. "3m28s".
// Returns empty string if no duration is found.
func hiveDurationStr(n Node) string {
	return parseDurationStr(n.Body)
}

// computeHiveStats aggregates cost metrics across hive agent posts.
// posts must be pre-bounded (callers must pass at most maxHivePosts entries).
// Features counts only posts that include a cost line (cost > 0); posts without
// a cost entry are excluded from all three metrics. This is intentional: a post
// without a cost line is not a verified build iteration.
func computeHiveStats(posts []Node) HiveStats {
	var total float64
	var features int
	for _, p := range posts {
		c := parseCostDollars(p.Body)
		if c > 0 {
			total += c
			features++
		}
	}
	avg := 0.0
	if features > 0 {
		avg = total / float64(features)
	}
	return HiveStats{Features: features, TotalCost: total, AvgCost: avg}
}

// PipelineRole holds display state for a pipeline role on the /hive dashboard.
type PipelineRole struct {
	Name       string    // "Scout", "Builder", "Critic", "Reflector"
	LastActive time.Time // zero = never seen in fetched posts
	Active     bool      // true = post within last 30 minutes
}

// pipelineRoleDefs maps display names to the title prefix used by each role.
var pipelineRoleDefs = []struct {
	display string
	prefix  string
}{
	{"Scout", "[hive:scout]"},
	{"Architect", "[hive:architect]"},
	{"Builder", "[hive:builder]"},
	{"Critic", "[hive:critic]"},
	{"Reflector", "[hive:reflector]"},
}

// computePipelineRoles extracts last-active timestamps for Scout, Builder, Critic, and Reflector
// by scanning post titles for the standard [hive:role] prefix.
func computePipelineRoles(posts []Node) []PipelineRole {
	last := make(map[string]time.Time, len(pipelineRoleDefs))
	for _, p := range posts {
		lower := strings.ToLower(p.Title)
		for _, rd := range pipelineRoleDefs {
			if strings.HasPrefix(lower, rd.prefix) {
				if t, ok := last[rd.display]; !ok || p.CreatedAt.After(t) {
					last[rd.display] = p.CreatedAt
				}
			}
		}
	}
	now := time.Now()
	roles := make([]PipelineRole, len(pipelineRoleDefs))
	for i, rd := range pipelineRoleDefs {
		t := last[rd.display]
		roles[i] = PipelineRole{
			Name:       rd.display,
			LastActive: t,
			Active:     !t.IsZero() && now.Sub(t) < activeRoleThreshold,
		}
	}
	return roles
}

// maxHiveAgentTasks is the upper bound on open tasks shown on the /hive dashboard.
// Invariant 13 (BOUNDED).
const maxHiveAgentTasks = 10

// parseIterFromPosts returns the highest iteration number found in post titles,
// or 0 if none are found. Post titles use the format "[hive:role] iter N: ...".
func parseIterFromPosts(posts []Node) int {
	re := regexp.MustCompile(`\biter\s+(\d+)\b`)
	best := 0
	for _, p := range posts {
		m := re.FindStringSubmatch(strings.ToLower(p.Title))
		if len(m) < 2 {
			continue
		}
		n := 0
		for _, c := range m[1] {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		if n > best {
			best = n
		}
	}
	return best
}

// handleHive renders the public /hive dashboard.
// Reads from the DATABASE first (works on Fly), falls back to local files (dev).
func (h *Handlers) handleHive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Diagnostics: DB first (works on Fly), fall back to local files (dev).
	entries, _ := h.store.ListHiveDiagnostics(ctx, maxHiveDiagEntries)
	if len(entries) == 0 {
		entries = readDiagnostics(h.loopDir, maxHiveDiagEntries)
	}

	// Local file data (dev only — empty on Fly, and that's fine).
	ls := readLoopState(h.loopDir)
	repoDir := ""
	if h.loopDir != "" {
		repoDir = filepath.Dir(h.loopDir)
	}
	commits := readRecentCommits(repoDir, 10)

	// Supplement with database data so deployed instances show real numbers.
	agentID := h.store.GetHiveAgentID(ctx)
	if agentID != "" {
		totalOps, lastActive, _ := h.store.GetHiveTotals(ctx, agentID)
		posts, _ := h.store.ListHiveActivity(ctx, agentID, maxHivePosts)

		// Iteration count: try "iter N" in post titles, fall back to done task count.
		if ic := parseIterFromPosts(posts); ic > ls.Iteration {
			ls.Iteration = ic
		}
		if ls.Iteration == 0 {
			// Count done tasks as iteration proxy.
			tasks, _ := h.store.ListHiveAgentTasks(ctx, agentID, 1000)
			doneCount := 0
			for _, t := range tasks {
				if t.State == "done" {
					doneCount++
				}
			}
			if doneCount > ls.Iteration {
				ls.Iteration = doneCount
			}
		}

		if totalOps > 0 && ls.Phase == "" {
			ls.Phase = "idle"
		}
		if !lastActive.IsZero() && ls.BuildTitle == "" {
			ls.BuildTitle = "Last active: " + lastActive.Format("2006-01-02 15:04")
		}
	}

	HivePage(ls, entries, commits, h.viewUser(r), profile.FromContext(ctx)).Render(ctx, w)
}

// handleHiveStatus renders the main content partial for HTMX polling (every 5s).
// It re-fetches posts and tasks and returns the #hive-content element only,
// with no surrounding HTML shell.
func (h *Handlers) handleHiveStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	agentID := h.store.GetHiveAgentID(ctx)
	posts, err := h.store.ListHiveActivity(ctx, agentID, maxHivePosts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stats := computeHiveStats(posts)
	roles := computePipelineRoles(posts)
	tasks, err := h.store.ListHiveAgentTasks(ctx, agentID, maxHiveAgentTasks)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	totalOps, lastActive, err := h.store.GetHiveTotals(ctx, agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	iterCount := parseIterFromPosts(posts)
	ls := readLoopState(h.loopDir)
	if ls.Iteration > iterCount {
		iterCount = ls.Iteration
	}
	p := profile.FromContext(ctx)
	if p == nil {
		p = profile.Default()
	}
	HiveStatusPartial(posts, stats, roles, tasks, totalOps, lastActive, iterCount, ls, p).Render(ctx, w)
}

// handleHiveDiagnostic accepts POST /api/hive/diagnostic from the hive runner.
// Stores the phase event so /hive/feed works in production where loop files are absent.
func (h *Handlers) handleHiveDiagnostic(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	var ev struct {
		Phase   string  `json:"phase"`
		Outcome string  `json:"outcome"`
		CostUSD float64 `json:"cost_usd"`
	}
	if err := json.Unmarshal(body, &ev); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.store.AppendHiveDiagnostic(r.Context(), ev.Phase, ev.Outcome, ev.CostUSD, body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

// handleHiveEscalation accepts POST /api/hive/escalation from the hive pipeline.
// Sets the task state to "escalated", creates a notification for the space owner,
// and records an escalate op for auditability.
func (h *Handlers) handleHiveEscalation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SpaceSlug  string `json:"space_slug"`
		TaskID     string `json:"task_id"`
		Reason     string `json:"reason"`
		AssigneeID string `json:"assignee_id"` // optional: specific user to notify
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.TaskID == "" || req.SpaceSlug == "" {
		http.Error(w, "task_id and space_slug required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Set task state to escalated.
	if err := h.store.UpdateNodeState(ctx, req.TaskID, "escalated"); err != nil {
		http.Error(w, "update task state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Look up the space to find the owner for notification.
	space, err := h.store.GetSpaceBySlug(ctx, req.SpaceSlug)
	if err != nil {
		http.Error(w, "space not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Record an escalate op for the audit trail.
	payload, _ := json.Marshal(map[string]string{"reason": req.Reason})
	op, _ := h.store.RecordOp(ctx, space.ID, req.TaskID, "hive", h.userID(r), "escalate", payload)

	// Notify the assignee, or fall back to the space owner.
	notifyTarget := req.AssigneeID
	if notifyTarget == "" {
		notifyTarget = space.OwnerID
	}
	opID := ""
	if op != nil {
		opID = op.ID
	}
	h.store.CreateNotification(ctx, notifyTarget, opID, space.ID,
		"Hive ESCALATION: "+req.Reason)

	writeJSON(w, http.StatusOK, map[string]string{"status": "escalated", "task_id": req.TaskID})
}

// handleHiveMirror accepts POST /api/hive/mirror from the Civilization runtime.
// It mirrors canonical EventGraph task progress back onto the Site node that
// originated the work, so the operator console does not invent parallel work.
type hiveMirrorRequest struct {
	NodeID       string `json:"node_id"`
	HiveTaskID   string `json:"hive_task_id"`
	HiveChainRef string `json:"hive_chain_ref"`
	EventType    string `json:"event_type"`
	State        string `json:"state"`
	Summary      string `json:"summary"`
}

func (h *Handlers) handleHiveMirror(w http.ResponseWriter, r *http.Request) {
	var req hiveMirrorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.NodeID) == "" {
		http.Error(w, "node_id required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	node, err := h.store.GetNode(ctx, req.NodeID)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	nextState := strings.TrimSpace(req.State)
	if nextState == "" && req.EventType == "work.task.completed" {
		nextState = StateDone
	}
	if nextState != "" && node.State != "deleted" {
		if err := h.store.UpdateNodeState(ctx, node.ID, nextState); err != nil {
			http.Error(w, "update node state: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	payload, _ := json.Marshal(map[string]string{
		"source":         "hive.eventgraph.mirror",
		"event_type":     req.EventType,
		"hive_task_id":   req.HiveTaskID,
		"hive_chain_ref": req.HiveChainRef,
		"state":          nextState,
	})
	_, _ = h.store.RecordOp(ctx, node.SpaceID, node.ID, "Hive", "hive", "hive_mirror", payload)

	if strings.TrimSpace(req.Summary) != "" {
		commentTitle := "Hive EventGraph progress"
		if req.EventType == "work.task.completed" {
			commentTitle = "Hive EventGraph completion"
		}
		if !h.hasDuplicateHiveMirrorComment(ctx, *node, commentTitle, req.Summary) {
			comment, err := h.store.CreateNode(ctx, CreateNodeParams{
				SpaceID:    node.SpaceID,
				ParentID:   node.ID,
				Kind:       KindComment,
				Title:      commentTitle,
				Body:       req.Summary,
				State:      StateOpen,
				Priority:   PriorityMedium,
				Author:     "Hive",
				AuthorID:   "hive",
				AuthorKind: "agent",
			})
			if err != nil {
				http.Error(w, "create mirror comment: "+err.Error(), http.StatusInternalServerError)
				return
			}
			commentPayload, _ := json.Marshal(map[string]string{
				"source":         "hive.eventgraph.mirror",
				"event_type":     req.EventType,
				"hive_task_id":   req.HiveTaskID,
				"hive_chain_ref": req.HiveChainRef,
			})
			_, _ = h.store.RecordOp(ctx, node.SpaceID, comment.ID, "Hive", "hive", "hive_mirror", commentPayload)
		}
	}

	versionID := ""
	if req.EventType == "work.task.completed" && strings.TrimSpace(req.Summary) != "" {
		versionedSpec, err := h.createRefineryVersionFromHiveMirror(ctx, *node, req)
		if err != nil {
			http.Error(w, "create versioned refinement: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if versionedSpec != nil {
			versionID = versionedSpec.ID
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":         "mirrored",
		"node_id":        node.ID,
		"hive_chain_ref": req.HiveChainRef,
		"version_id":     versionID,
	})
}

// handleHiveWebhookDeliveries exposes Site-to-Hive delivery exceptions and retry state.
func (h *Handlers) handleHiveWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	deliveries, err := h.store.ListWebhookDeliveries(r.Context(), status, limit)
	if err != nil {
		http.Error(w, "list webhook deliveries: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deliveries": deliveries})
}

func (h *Handlers) hasDuplicateHiveMirrorComment(ctx context.Context, node Node, title, summary string) bool {
	children, err := h.store.ListNodes(ctx, ListNodesParams{SpaceID: node.SpaceID, ParentID: node.ID, Limit: 500})
	if err != nil {
		return false
	}
	trimmedSummary := strings.TrimSpace(summary)
	for _, child := range children {
		if child.Kind == KindComment && child.Title == title && strings.TrimSpace(child.Body) == trimmedSummary {
			return true
		}
	}
	return false
}

func (h *Handlers) createRefineryVersionFromHiveMirror(ctx context.Context, source Node, req hiveMirrorRequest) (*Node, error) {
	root, err := h.findRefineryRootSpec(ctx, source)
	if err != nil {
		return nil, err
	}
	if root == nil {
		return nil, nil
	}

	children, err := h.store.ListNodes(ctx, ListNodesParams{SpaceID: root.SpaceID, ParentID: root.ID, Limit: 500})
	if err != nil {
		return nil, err
	}
	missing := refineryMissingSections(req.Summary)
	for _, child := range children {
		if child.Kind != KindSpec {
			continue
		}
		if req.HiveChainRef != "" && strings.Contains(child.Body, "hive_chain_ref: "+req.HiveChainRef) {
			if missing != "None detected." {
				if _, err := h.createFollowupVersionGapTask(ctx, *root, child, missing); err != nil {
					return nil, err
				}
			}
			return &child, nil
		}
		if req.HiveTaskID != "" && strings.Contains(child.Body, "hive_task_id: "+req.HiveTaskID) {
			if missing != "None detected." {
				if _, err := h.createFollowupVersionGapTask(ctx, *root, child, missing); err != nil {
					return nil, err
				}
			}
			return &child, nil
		}
	}

	version, previous := nextRefineryVersion(children)
	body := buildRefineryVersionBody(*root, previous, source, req, version)
	state := SpecStateRequirementClarify
	if missing == "None detected." {
		state = SpecStateDraft
	}

	versionedSpec, err := h.store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    root.SpaceID,
		ParentID:   root.ID,
		Kind:       KindSpec,
		Title:      fmt.Sprintf("%s v%s", root.Title, version),
		Body:       body,
		State:      state,
		Priority:   PriorityHigh,
		Author:     "Hive",
		AuthorID:   "hive",
		AuthorKind: "agent",
		Causes:     []string{root.ID, source.ID},
		Tags:       []string{"refinery-version", "hive-mirror"},
	})
	if err != nil {
		return nil, err
	}

	previousID := ""
	if previous != nil {
		previousID = previous.ID
	}
	payload, _ := json.Marshal(map[string]string{
		"source":              "hive.eventgraph.versioned_refinement",
		"event_type":          req.EventType,
		"hive_task_id":        req.HiveTaskID,
		"hive_chain_ref":      req.HiveChainRef,
		"root_spec_id":        root.ID,
		"source_node_id":      source.ID,
		"previous_version_id": previousID,
		"version":             version,
		"state":               state,
	})
	_, _ = h.store.RecordOp(ctx, root.SpaceID, versionedSpec.ID, "Hive", "hive", "hive_mirror", payload)

	if missing != "None detected." {
		if _, err := h.createFollowupVersionGapTask(ctx, *root, *versionedSpec, missing); err != nil {
			return nil, err
		}
	}

	return versionedSpec, nil
}

func (h *Handlers) createFollowupVersionGapTask(ctx context.Context, root Node, version Node, missing string) (*Node, error) {
	children, err := h.store.ListNodes(ctx, ListNodesParams{SpaceID: root.SpaceID, ParentID: root.ID, Limit: 500})
	if err != nil {
		return nil, err
	}
	for _, child := range children {
		if child.Kind == KindTask && child.State != "deleted" && containsString(child.Causes, version.ID) && strings.HasPrefix(child.Title, "Investigate refinery gaps:") {
			return nil, nil
		}
	}

	taskBody := fmt.Sprintf(`## Investigation Trigger
Version %s was created from Hive/EventGraph output, but it is still missing or insufficient for these normative gates: %s.

## Version Under Investigation
Title: %s
Node ID: %s
State at spawn: %s

## Required Investigation Output
1. Inspect the versioned spec artifact and its root intake/spec.
2. Fill the missing gates only when the evidence supports them.
3. If a gate cannot be completed, name the exact ambiguity, missing source, or violated invariant.
4. Produce the next candidate version content as Hive completion output, not by mutating the parent artifact.
5. Preserve the Version Lineage and recommend the next FSM state.

## Guardrails
Do not rewrite the root intake/spec or any previous version. Each refinement is a new SemVer artifact and a matter of record.`, version.Title, missing, version.Title, version.ID, version.State)

	task, err := h.store.CreateNode(ctx, CreateNodeParams{
		SpaceID:    root.SpaceID,
		ParentID:   root.ID,
		Kind:       KindTask,
		Title:      "Investigate refinery gaps: " + version.Title,
		Body:       taskBody,
		State:      StateOpen,
		Priority:   PriorityHigh,
		Assignee:   "Agents",
		Author:     "Hive",
		AuthorID:   "hive",
		AuthorKind: "agent",
		Causes:     []string{root.ID, version.ID},
		Tags:       []string{"refinery-gap", "version-followup"},
	})
	if err != nil {
		return nil, err
	}

	payload, _ := json.Marshal(map[string]string{
		"source":          "refinery.version_gap_investigation",
		"kind":            KindTask,
		"priority":        PriorityHigh,
		"description":     taskBody,
		"parent_id":       root.ID,
		"version_id":      version.ID,
		"missing_gates":   missing,
		"recommended_fsm": SpecStateInvestigate,
	})
	_, _ = h.store.RecordOp(ctx, root.SpaceID, task.ID, "Hive", "hive", "intend", payload)
	return task, nil
}

func (h *Handlers) findRefineryRootSpec(ctx context.Context, node Node) (*Node, error) {
	current := node
	var nearestSpec *Node
	for {
		if current.Kind == KindSpec {
			copied := current
			nearestSpec = &copied
		}
		if current.ParentID == "" {
			return nearestSpec, nil
		}
		parent, err := h.store.GetNode(ctx, current.ParentID)
		if errors.Is(err, ErrNotFound) {
			return nearestSpec, nil
		}
		if err != nil {
			return nil, err
		}
		current = *parent
	}
}

type semverParts struct {
	major int
	minor int
	patch int
	raw   string
}

func nextRefineryVersion(children []Node) (string, *Node) {
	var latest *Node
	var latestVersion semverParts
	for i := range children {
		if children[i].Kind != KindSpec {
			continue
		}
		version, ok := parseRefineryVersion(children[i])
		if !ok {
			continue
		}
		if latest == nil || compareSemver(version, latestVersion) > 0 {
			latest = &children[i]
			latestVersion = version
		}
	}
	if latest == nil {
		return "0.1.0", nil
	}
	return fmt.Sprintf("%d.%d.0", latestVersion.major, latestVersion.minor+1), latest
}

func parseRefineryVersion(node Node) (semverParts, bool) {
	bodyRE := regexp.MustCompile(`(?m)^refinery_version:\s*([0-9]+)\.([0-9]+)\.([0-9]+)\s*$`)
	if match := bodyRE.FindStringSubmatch(node.Body); len(match) == 4 {
		return semverFromMatch(match)
	}
	titleRE := regexp.MustCompile(`\bv([0-9]+)\.([0-9]+)\.([0-9]+)\b`)
	if match := titleRE.FindStringSubmatch(node.Title); len(match) == 4 {
		return semverFromMatch(match)
	}
	return semverParts{}, false
}

func semverFromMatch(match []string) (semverParts, bool) {
	major, err1 := strconv.Atoi(match[1])
	minor, err2 := strconv.Atoi(match[2])
	patch, err3 := strconv.Atoi(match[3])
	if err1 != nil || err2 != nil || err3 != nil {
		return semverParts{}, false
	}
	return semverParts{
		major: major,
		minor: minor,
		patch: patch,
		raw:   fmt.Sprintf("%d.%d.%d", major, minor, patch),
	}, true
}

func compareSemver(a, b semverParts) int {
	if a.major != b.major {
		return a.major - b.major
	}
	if a.minor != b.minor {
		return a.minor - b.minor
	}
	return a.patch - b.patch
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func buildRefineryVersionBody(root Node, previous *Node, source Node, req hiveMirrorRequest, version string) string {
	previousID := "none"
	previousVersion := "none"
	if previous != nil {
		previousID = previous.ID
		if parsed, ok := parseRefineryVersion(*previous); ok {
			previousVersion = parsed.raw
		}
	}

	missing := refineryMissingSections(req.Summary)
	gateAssessment := "This version contains all currently required normative gate headings in the Hive investigation output and is ready to enter Draft Spec review."
	if missing != "None detected." {
		gateAssessment = "This version is not draft-ready. The next refinement must investigate or clarify these missing gates: " + missing + "."
	}

	return fmt.Sprintf(`---
refinery_version: %s
root_spec_id: %s
previous_version_id: %s
previous_version: %s
source_node_id: %s
hive_task_id: %s
hive_chain_ref: %s
event_type: %s
created_by: hive
created_at: %s
---

# %s v%s

## Version Lineage
- Root intake/spec: %s
- Previous version: %s (%s)
- Source Site node: %s
- Hive task: %s
- EventGraph ref: %s

## Investigation Result
%s

## Gate Assessment
%s

## Provenance Rule
This artifact is a versioned refinement. It does not rewrite or replace the parent intake/spec. The parent remains a matter of record; this version advances the lineage from rough input toward a vetted, ship-ready normative spec.
`, version, root.ID, previousID, previousVersion, source.ID, req.HiveTaskID, req.HiveChainRef, req.EventType, time.Now().UTC().Format(time.RFC3339), root.Title, version, root.ID, previousID, previousVersion, source.ID, req.HiveTaskID, req.HiveChainRef, strings.TrimSpace(req.Summary), gateAssessment)
}

// handleHiveFeed renders the phase timeline partial for HTMX polling — public, no auth required.
// Reads from the database first (works on Fly); falls back to local files (dev without DB).
func (h *Handlers) handleHiveFeed(w http.ResponseWriter, r *http.Request) {
	entries, _ := h.store.ListHiveDiagnostics(r.Context(), maxHiveDiagEntries)
	if len(entries) == 0 {
		entries = readDiagnostics(h.loopDir, maxHiveDiagEntries)
	}
	HiveDiagFeed(entries).Render(r.Context(), w)
}

// handleHiveStats renders the live stats bar partial for HTMX polling.
func (h *Handlers) handleHiveStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	agentID := h.store.GetHiveAgentID(ctx)
	totalOps, lastActive, err := h.store.GetHiveTotals(ctx, agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	HiveStatsBar(totalOps, lastActive).Render(ctx, w)
}

// groundedLabel returns "grounded in N doc(s)" for agent messages that used document context,
// or "" if no documents were used. Tags use the format "grounded:N".
func groundedLabel(tags []string) string {
	for _, t := range tags {
		if strings.HasPrefix(t, "grounded:") {
			n, err := strconv.Atoi(strings.TrimPrefix(t, "grounded:"))
			if err == nil && n > 0 {
				if n == 1 {
					return "grounded in 1 doc"
				}
				return fmt.Sprintf("grounded in %d docs", n)
			}
		}
	}
	return ""
}
