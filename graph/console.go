package graph

import (
	"net/http"
	"time"

	"github.com/transpara-ai/site/profile"
)

type ConsolePageData struct {
	Title  string
	Active string // health | kanban | intake | config
	Health *ConsoleHealthWall
	Kanban *ConsoleKanban
}

func (h *Handlers) renderConsole(w http.ResponseWriter, r *http.Request, data ConsolePageData) {
	ConsolePage(data, h.viewUser(r), profile.FromContext(r.Context())).Render(r.Context(), w)
}

type ConsoleFreshness string

const (
	FreshnessCurrent     ConsoleFreshness = "current"
	FreshnessStale       ConsoleFreshness = "stale"
	FreshnessPartial     ConsoleFreshness = "partial"
	FreshnessUnavailable ConsoleFreshness = "unavailable"
)

const consoleStaleWindow = 30 * time.Second
const consoleFutureSkew = 5 * time.Second

type ConsoleHealthWall struct {
	Freshness        ConsoleFreshness
	GeneratedAt      string
	ActiveAgents     int
	Agents           []ConsoleAgentRow
	PendingApprovals int
	Approvals        []ConsoleApproval
	Notices          []string
}

type ConsoleAgentRow struct {
	Name  string
	Role  string
	Model string
}

type ConsoleApproval struct {
	RequestID string
	Action    string
	Target    string
	Risk      string
	CreatedAt string
}

// buildConsoleHealthWall maps the hive operator projection (or a fetch error)
// into the Health Wall view-model. On any fetch failure it returns an
// unavailable wall with a human-readable notice and no invented agents.
func buildConsoleHealthWall(proj *OpsHiveProjection, fetchErr error, now time.Time) ConsoleHealthWall {
	if fetchErr != nil || proj == nil {
		reason := "operator projection unavailable"
		if fetchErr != nil {
			reason = fetchErr.Error()
		}
		return ConsoleHealthWall{Freshness: FreshnessUnavailable, Notices: []string{reason}}
	}

	wall := ConsoleHealthWall{
		Freshness:    deriveFreshness(proj.GeneratedAt, nil, len(proj.Errors) > 0, now, consoleStaleWindow),
		GeneratedAt:  proj.GeneratedAt,
		ActiveAgents: proj.RuntimeEvidence.AgentEvents.ObservedActive,
		Notices:      append([]string(nil), proj.Errors...),
	}
	for _, a := range proj.RuntimeEvidence.AgentEvents.ActiveAgents {
		wall.Agents = append(wall.Agents, ConsoleAgentRow{Name: a.Name, Role: a.Role, Model: a.Model})
	}
	for _, ap := range proj.PendingApprovals {
		wall.Approvals = append(wall.Approvals, ConsoleApproval{
			RequestID: ap.RequestID,
			Action:    ap.ActionName,
			Target:    ap.Target,
			Risk:      ap.RiskSummary,
			CreatedAt: ap.CreatedAt,
		})
	}
	wall.PendingApprovals = len(wall.Approvals)
	return wall
}

func (h *Handlers) handleConsoleHealth(w http.ResponseWriter, r *http.Request) {
	proj, err := fetchHiveOperatorProjection(r)
	wall := buildConsoleHealthWall(proj, err, time.Now().UTC())
	h.renderConsole(w, r, ConsolePageData{Title: "Health wall", Active: "health", Health: &wall})
}

func (h *Handlers) handleConsoleHealthFragment(w http.ResponseWriter, r *http.Request) {
	proj, err := fetchHiveOperatorProjection(r)
	wall := buildConsoleHealthWall(proj, err, time.Now().UTC())
	consoleHealthWallFragment(wall).Render(r.Context(), w)
}

func (h *Handlers) handleConsoleKanban(w http.ResponseWriter, r *http.Request) {
	lens := parseLens(r.URL.Query().Get("lens"))
	res := fetchConsoleWork(r)
	k := buildConsoleKanban(res.Tasks, res.Err, lens, time.Now().UTC())
	h.renderConsole(w, r, ConsolePageData{Title: "Kanban", Active: "kanban", Kanban: &k})
}

func (h *Handlers) handleConsoleKanbanFragment(w http.ResponseWriter, r *http.Request) {
	lens := parseLens(r.URL.Query().Get("lens"))
	res := fetchConsoleWork(r)
	k := buildConsoleKanban(res.Tasks, res.Err, lens, time.Now().UTC())
	consoleKanbanFragment(k).Render(r.Context(), w)
}

// deriveFreshness maps upstream signals onto an explicit freshness state.
// It fails closed: a fetch error, an empty or unparseable timestamp, or any
// other ambiguity resolves to FreshnessUnavailable. Only a parseable,
// within-window, error-free projection earns FreshnessCurrent.
func deriveFreshness(generatedAt string, fetchErr error, hasPartialErrors bool, now time.Time, staleWindow time.Duration) ConsoleFreshness {
	if fetchErr != nil {
		return FreshnessUnavailable
	}
	ts, err := time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		return FreshnessUnavailable
	}
	age := now.Sub(ts)
	if age < -consoleFutureSkew {
		return FreshnessUnavailable
	}
	if age > staleWindow {
		return FreshnessStale
	}
	if hasPartialErrors {
		return FreshnessPartial
	}
	return FreshnessCurrent
}
