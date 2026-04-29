package graph

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/transpara-ai/site/profile"
)

type OpsSurface struct {
	ID          string
	Label       string
	Description string
	Href        string
	Target      string
	Owner       string
	Status      string
}

type OpsPageData struct {
	Title       string
	Description string
	Active      string
	Surfaces    []OpsSurface
	EmbedURL    string
	EmbedLabel  string
	Telemetry   *OpsTelemetryData
	Work        *OpsWorkData
	LegacyURL   string
}

type OpsTelemetryData struct {
	WorkURL              string
	GeneratedAt          string
	ActorCount           int
	AgentCount           int
	ActiveAgentCount     int
	ProcessingAgentCount int
	PhaseLabel           string
	PhaseStatus          string
	RecentAgents         []OpsTelemetryAgent
	RecentEvents         []OpsTelemetryEvent
	Error                string
}

type OpsTelemetryAgent struct {
	Role        string `json:"role"`
	State       string `json:"state"`
	Model       string `json:"model"`
	LastEventAt string `json:"last_event_at"`
	LastMessage string `json:"last_message"`
	Errors      int    `json:"errors"`
}

type OpsTelemetryEvent struct {
	EventType string `json:"event_type"`
	ActorRole string `json:"actor_role"`
	Summary   string `json:"summary"`
	At        string `json:"at"`
}

type OpsWorkData struct {
	WorkURL        string
	GeneratedAt    string
	Total          int
	Open           int
	Active         int
	Blocked        int
	Completed      int
	HighPriority   int
	Unassigned     int
	EvidenceCount  int
	WaivedCount    int
	RecentTasks    []OpsWorkTask
	BlockedTasks   []OpsWorkTask
	Error          string
}

type OpsWorkTask struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Priority      string `json:"priority"`
	Workspace     string `json:"workspace"`
	Status        string `json:"status"`
	Assignee      string `json:"assignee"`
	Blocked       bool   `json:"blocked"`
	ArtifactCount int    `json:"artifact_count"`
	Waived        bool   `json:"waived"`
}

type opsWorkTasksResponse struct {
	Tasks []OpsWorkTask `json:"tasks"`
}

type opsTelemetryOverview struct {
	Timestamp string `json:"timestamp"`
	Actors    []struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		ActorType   string `json:"actor_type"`
		Status      string `json:"status"`
	} `json:"actors"`
	Agents       []OpsTelemetryAgent `json:"agents"`
	RecentEvents []OpsTelemetryEvent `json:"recent_events"`
	Phases       []struct {
		Phase  int    `json:"phase"`
		Label  string `json:"label"`
		Status string `json:"status"`
	} `json:"phases"`
}

func (h *Handlers) handleOps(w http.ResponseWriter, r *http.Request) {
	h.renderOps(w, r, OpsPageData{
		Title:       "Operations",
		Description: "Site-owned operator shell for work, telemetry, hive status, and refinery review.",
		Active:      "overview",
	})
}

func (h *Handlers) handleOpsWork(w http.ResponseWriter, r *http.Request) {
	legacyURL := legacyWorkURL(workUIBaseURL(r), "/")
	h.renderOps(w, r, OpsPageData{
		Title:       "Work",
		Description: "Native Site task summary sourced from the Work API.",
		Active:      "work",
		Work:        fetchOpsWork(r),
		LegacyURL:   legacyURL,
	})
}

func (h *Handlers) handleOpsTelemetry(w http.ResponseWriter, r *http.Request) {
	legacyURL := legacyWorkURL(workUIBaseURL(r), "/telemetry/")
	h.renderOps(w, r, OpsPageData{
		Title:       "Telemetry",
		Description: "Native Site telemetry summary sourced from Work APIs.",
		Active:      "telemetry",
		Telemetry:   fetchOpsTelemetry(r),
		LegacyURL:   legacyURL,
	})
}

func (h *Handlers) handleOpsHive(w http.ResponseWriter, r *http.Request) {
	h.renderOps(w, r, OpsPageData{
		Title:       "Hive",
		Description: "Hive runtime status rendered through the site shell.",
		Active:      "hive",
		EmbedURL:    "/hive",
		EmbedLabel:  "Hive status",
	})
}

func (h *Handlers) handleOpsRefinery(w http.ResponseWriter, r *http.Request) {
	target := "/app/journey-test/refinery?profile=transpara"
	if slug := strings.TrimSpace(r.URL.Query().Get("space")); slug != "" {
		q := url.Values{}
		q.Set("profile", "transpara")
		target = "/app/" + url.PathEscape(slug) + "/refinery?" + q.Encode()
	}
	h.renderOps(w, r, OpsPageData{
		Title:       "Refinery",
		Description: "Intake and design review surface backed by the site refinery projection and simplified FSM.",
		Active:      "refinery",
		EmbedURL:    target,
		EmbedLabel:  "Refinery",
	})
}

func (h *Handlers) renderOps(w http.ResponseWriter, r *http.Request, data OpsPageData) {
	data.Surfaces = opsSurfaces(r)
	OpsPage(data, h.viewUser(r), profile.FromContext(r.Context())).Render(r.Context(), w)
}

func opsSurfaces(r *http.Request) []OpsSurface {
	workBase := workUIBaseURL(r)
	return []OpsSurface{
		{
			ID:          "work",
			Label:       "Work",
			Description: "Task queue, assignment, blockers, artifacts, and completion evidence.",
			Href:        "/ops/work",
			Target:      legacyWorkURL(workBase, "/"),
			Owner:       "site shell, work API",
			Status:      "legacy UI framed",
		},
		{
			ID:          "telemetry",
			Label:       "Telemetry",
			Description: "Agent status, phase activity, pipeline report, and event stream health.",
			Href:        "/ops/telemetry",
			Target:      legacyWorkURL(workBase, "/telemetry/"),
			Owner:       "site native UI, work telemetry",
			Status:      "native summary",
		},
		{
			ID:          "hive",
			Label:       "Hive",
			Description: "Runtime iteration, phase timeline, diagnostics, and recent build signal.",
			Href:        "/ops/hive",
			Target:      "/hive",
			Owner:       "site shell, hive diagnostics",
			Status:      "native site page framed",
		},
		{
			ID:          "refinery",
			Label:       "Refinery",
			Description: "Intake/design review backed by the simplified FSM projection.",
			Href:        "/ops/refinery",
			Target:      "/app/journey-test/refinery?profile=transpara",
			Owner:       "site shell, eventgraph/work projection",
			Status:      "native FSM framed",
		},
	}
}

func fetchOpsTelemetry(r *http.Request) *OpsTelemetryData {
	workBase := serverWorkAPIBaseURL()
	data := &OpsTelemetryData{
		WorkURL: legacyWorkURL(workBase, "/telemetry/overview"),
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, data.WorkURL, nil)
	if err != nil {
		data.Error = err.Error()
		return data
	}
	if key := strings.TrimSpace(os.Getenv("WORK_API_KEY")); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	} else {
		req.Header.Set("Authorization", "Bearer dev")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		data.Error = err.Error()
		return data
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data.Error = fmt.Sprintf("work telemetry returned %s", resp.Status)
		return data
	}
	var overview opsTelemetryOverview
	if err := json.NewDecoder(resp.Body).Decode(&overview); err != nil {
		data.Error = err.Error()
		return data
	}
	data.ActorCount = len(overview.Actors)
	data.AgentCount = len(overview.Agents)
	for _, agent := range overview.Agents {
		switch strings.ToLower(agent.State) {
		case "processing":
			data.ProcessingAgentCount++
			data.ActiveAgentCount++
		case "idle", "":
		default:
			data.ActiveAgentCount++
		}
	}
	for _, phase := range overview.Phases {
		if phase.Status == "in_progress" || phase.Status == "blocked" {
			data.PhaseLabel = phase.Label
			data.PhaseStatus = phase.Status
			break
		}
	}
	if data.PhaseLabel == "" && len(overview.Phases) > 0 {
		last := overview.Phases[len(overview.Phases)-1]
		data.PhaseLabel = last.Label
		data.PhaseStatus = last.Status
	}
	data.GeneratedAt = formatOpsTime(overview.Timestamp)
	data.RecentAgents = takeTelemetryAgents(overview.Agents, 8)
	data.RecentEvents = takeTelemetryEvents(overview.RecentEvents, 8)
	return data
}

func fetchOpsWork(r *http.Request) *OpsWorkData {
	workBase := serverWorkAPIBaseURL()
	data := &OpsWorkData{
		WorkURL:     legacyWorkURL(workBase, "/tasks"),
		GeneratedAt: time.Now().UTC().Format("2006-01-02 15:04:05"),
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, data.WorkURL, nil)
	if err != nil {
		data.Error = err.Error()
		return data
	}
	if key := strings.TrimSpace(os.Getenv("WORK_API_KEY")); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	} else {
		req.Header.Set("Authorization", "Bearer dev")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		data.Error = err.Error()
		return data
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data.Error = fmt.Sprintf("work tasks returned %s", resp.Status)
		return data
	}
	var tasks opsWorkTasksResponse
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		data.Error = err.Error()
		return data
	}
	data.Total = len(tasks.Tasks)
	for _, task := range tasks.Tasks {
		switch strings.ToLower(task.Status) {
		case "completed", "done", "closed":
			data.Completed++
		case "in_progress", "active", "assigned":
			data.Active++
			data.Open++
		default:
			data.Open++
		}
		if task.Blocked {
			data.Blocked++
			data.BlockedTasks = append(data.BlockedTasks, task)
		}
		if strings.EqualFold(task.Priority, "high") {
			data.HighPriority++
		}
		if strings.TrimSpace(task.Assignee) == "" && !isCompletedWorkTask(task) {
			data.Unassigned++
		}
		data.EvidenceCount += task.ArtifactCount
		if task.Waived {
			data.WaivedCount++
		}
	}
	data.RecentTasks = takeWorkTasks(tasks.Tasks, 10)
	data.BlockedTasks = takeWorkTasks(data.BlockedTasks, 6)
	return data
}

func serverWorkAPIBaseURL() string {
	if base := strings.TrimSpace(os.Getenv("WORK_API_BASE_URL")); base != "" {
		return strings.TrimRight(base, "/")
	}
	if base := strings.TrimSpace(os.Getenv("WORK_UI_BASE_URL")); base != "" {
		return strings.TrimRight(base, "/")
	}
	return "http://localhost:8080"
}

func takeWorkTasks(tasks []OpsWorkTask, limit int) []OpsWorkTask {
	if len(tasks) <= limit {
		return tasks
	}
	return tasks[:limit]
}

func isCompletedWorkTask(task OpsWorkTask) bool {
	switch strings.ToLower(task.Status) {
	case "completed", "done", "closed":
		return true
	default:
		return false
	}
}

func takeTelemetryAgents(agents []OpsTelemetryAgent, limit int) []OpsTelemetryAgent {
	if len(agents) <= limit {
		return agents
	}
	return agents[:limit]
}

func takeTelemetryEvents(events []OpsTelemetryEvent, limit int) []OpsTelemetryEvent {
	if len(events) <= limit {
		return events
	}
	return events[:limit]
}

func formatOpsTime(raw string) string {
	if raw == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.Format("2006-01-02 15:04:05")
	}
	return raw
}

func opsAgentStateClass(state string) string {
	switch strings.ToLower(state) {
	case "processing":
		return "border-sky-400/30 text-sky-300 bg-sky-400/10"
	case "idle":
		return "border-edge text-warm-faint bg-void/30"
	case "error", "failed":
		return "border-red-400/30 text-red-300 bg-red-400/10"
	default:
		return "border-brand/30 text-brand bg-brand/10"
	}
}

func workUIBaseURL(r *http.Request) string {
	if base := strings.TrimSpace(os.Getenv("WORK_UI_BASE_URL")); base != "" {
		return strings.TrimRight(base, "/")
	}

	host := r.Host
	if host == "" {
		return "http://localhost:8080"
	}

	name, _, err := net.SplitHostPort(host)
	if err != nil {
		name = host
	}
	if name == "" {
		name = "localhost"
	}

	scheme := "http"
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.Split(forwarded, ",")[0]
	} else if r.TLS != nil {
		scheme = "https"
	}

	return scheme + "://" + name + ":8080"
}

func legacyWorkURL(base, path string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	u.Path = path
	u.RawQuery = ""
	return u.String()
}
