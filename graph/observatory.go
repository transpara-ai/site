package graph

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/transpara-ai/site/graph/svgviz"
)

// The observatory is the read-only civilization transparency surface
// (dark-factory transparency contract T1–T7, observatory phase-3 plan).
// It consumes egress APIs only — work /telemetry/status, /telemetry/agents/history,
// /tasks/{id}/events and the hive operator projection — and performs no writes.
// Every panel renders either honest data with provenance or an explicit
// unavailable block; svgviz renderers fail closed and the templates show the
// reason instead.

type OpsObservatoryData struct {
	GeneratedAt string

	// Vitals (work /telemetry/status)
	StatusURL     string
	Vitals        *ObsVitals
	VitalsError   string
	SpendGaugeSVG string // empty when cost or cap unknown — template states why

	// Agent roster + 24h state timelines (status + /telemetry/agents/history)
	HistoryURL   string
	Agents       []ObsAgentView
	HistoryError string

	// Pipeline phase costs (reuses the telemetry fetch: /telemetry/overview + pipeline report)
	Telemetry        *OpsTelemetryData
	PhaseCostBarsSVG string
	PhaseCostTotal   string
	PhaseCostLabels  []string

	// Authority, lifecycle, audit traces (hive operator projection via fetchOpsHive)
	Hive *OpsHiveData

	// Causal trace explorer (work /tasks/{id}/events, on demand via ?task=)
	TraceTaskID   string
	TraceURL      string
	TraceSteps    []ObsTraceStep
	TraceSVG      string
	TraceError    string
}

// ObsVitals mirrors the hive snapshot served by work /telemetry/status.
// Pointer fields are nullable in the feeder; nil renders as explicit unknown.
type ObsVitals struct {
	ActiveAgents int      `json:"active_agents"`
	TotalActors  int      `json:"total_actors"`
	ChainLength  int64    `json:"chain_length"`
	ChainOK      bool     `json:"chain_ok"`
	EventRate    *float64 `json:"event_rate"`
	DailyCost    *float64 `json:"daily_cost"`
	DailyCap     *float64 `json:"daily_cap"`
	Severity     string   `json:"severity"`
}

type obsStatusAgent struct {
	Role          string    `json:"role"`
	ActorID       string    `json:"actor_id"`
	State         string    `json:"state"`
	Model         string    `json:"model"`
	Iteration     int       `json:"iteration"`
	MaxIterations int       `json:"max_iterations"`
	TokensUsed    int64     `json:"tokens_used"`
	CostUSD       float64   `json:"cost_usd"`
	TrustScore    *float64  `json:"trust_score"`
	LastEventType *string   `json:"last_event_type"`
	LastEventAt   time.Time `json:"last_event_at"`
	Errors        int       `json:"errors"`
}

type obsStatusResponse struct {
	Agents    []obsStatusAgent `json:"agents"`
	Hive      *ObsVitals       `json:"hive"`
	Timestamp time.Time        `json:"timestamp"`
}

type obsStateSpan struct {
	State     string    `json:"state"`
	EnteredAt time.Time `json:"entered_at"`
	Duration  float64   `json:"duration_seconds"`
}

type obsAgentHistory struct {
	Role         string         `json:"role"`
	ActorID      string         `json:"actor_id"`
	CurrentState string         `json:"current_state"`
	States       []obsStateSpan `json:"states"`
}

type obsHistoryResponse struct {
	Agents []obsAgentHistory `json:"agents"`
	Window string            `json:"window"`
}

type obsTraceEvent struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
}

type obsTraceResponse struct {
	TaskID string          `json:"task_id"`
	Events []obsTraceEvent `json:"events"`
}

// ObsAgentView is one roster row: latest snapshot numbers plus the 24h state
// timeline strip (empty SVG when history is unavailable for this actor).
type ObsAgentView struct {
	Role         string
	ActorID      string
	State        string
	Model        string
	Iteration    int
	MaxIter      int
	TokensUsed   int64
	CostUSD      string
	Trust        string // formatted 0.00–1.00, or "unknown"
	Errors       int
	LastEventAt  string
	TimelineSVG  string
}

type ObsTraceStep struct {
	Label string
	Sub   string
	At    string
}

func (h *Handlers) handleOpsObservatory(w http.ResponseWriter, r *http.Request) {
	data := &OpsObservatoryData{GeneratedAt: time.Now().UTC().Format("2006-01-02 15:04:05 UTC")}

	status, statusURL, err := fetchObservatoryStatus(r)
	data.StatusURL = statusURL
	if err != nil {
		data.VitalsError = err.Error()
	} else {
		data.Vitals = status.Hive
		if status.Hive != nil && status.Hive.DailyCost != nil && status.Hive.DailyCap != nil {
			data.SpendGaugeSVG = svgviz.Gauge(*status.Hive.DailyCost, *status.Hive.DailyCap, 320, 22)
		}
	}

	histories, historyURL, histErr := fetchObservatoryHistory(r)
	data.HistoryURL = historyURL
	if histErr != nil {
		data.HistoryError = histErr.Error()
	}
	if err == nil {
		data.Agents = buildObsAgents(status.Agents, histories)
	}

	data.Telemetry = fetchOpsTelemetry(r)
	if data.Telemetry != nil && data.Telemetry.Pipeline != nil {
		svg, labels, total := phaseCostBars(data.Telemetry.Pipeline)
		data.PhaseCostBarsSVG = svg
		data.PhaseCostLabels = labels
		if total > 0 {
			data.PhaseCostTotal = fmt.Sprintf("%.2f", total)
		}
	}

	data.Hive = h.fetchOpsHive(r)

	if taskID := strings.TrimSpace(r.URL.Query().Get("task")); taskID != "" {
		data.TraceTaskID = taskID
		steps, traceURL, traceErr := fetchObservatoryTrace(r, taskID)
		data.TraceURL = traceURL
		if traceErr != nil {
			data.TraceError = traceErr.Error()
		} else {
			data.TraceSteps = steps
			data.TraceSVG = svgviz.Staircase(obsStaircaseSteps(steps), 720, 30+34*len(steps))
		}
	}

	h.renderOps(w, r, OpsPageData{
		Title:       "Observatory",
		Description: "Read-only civilization transparency: vitals, spend vs cap, agent lifecycle, authority decisions, and causal traces. Projection only — EventGraph remains truth; decisions happen on governed surfaces.",
		Active:      "observatory",
		Observatory: data,
	})
}

func fetchObservatoryStatus(r *http.Request) (*obsStatusResponse, string, error) {
	statusURL := legacyWorkURL(serverWorkAPIBaseURL(), "/telemetry/status")
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, statusURL, err
	}
	setWorkAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, statusURL, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, statusURL, fmt.Errorf("work telemetry status returned %s", resp.Status)
	}
	var payload obsStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, statusURL, err
	}
	return &payload, statusURL, nil
}

func fetchObservatoryHistory(r *http.Request) (map[string]obsAgentHistory, string, error) {
	historyURL := legacyWorkURL(serverWorkAPIBaseURL(), "/telemetry/agents/history") + "?window=24h"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, historyURL, nil)
	if err != nil {
		return nil, historyURL, err
	}
	setWorkAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, historyURL, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, historyURL, fmt.Errorf("work agent history returned %s", resp.Status)
	}
	var payload obsHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, historyURL, err
	}
	byActor := make(map[string]obsAgentHistory, len(payload.Agents))
	for _, a := range payload.Agents {
		byActor[a.ActorID] = a
	}
	return byActor, historyURL, nil
}

func fetchObservatoryTrace(r *http.Request, taskID string) ([]ObsTraceStep, string, error) {
	traceURL := legacyWorkURL(serverWorkAPIBaseURL(), "/tasks/"+taskID+"/events")
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, traceURL, nil)
	if err != nil {
		return nil, traceURL, err
	}
	setWorkAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, traceURL, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, traceURL, fmt.Errorf("work task events returned %s", resp.Status)
	}
	var payload obsTraceResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, traceURL, err
	}
	steps := make([]ObsTraceStep, 0, len(payload.Events))
	for _, ev := range payload.Events {
		steps = append(steps, ObsTraceStep{
			Label: ev.Type,
			Sub:   ev.Source,
			At:    formatOpsTime(ev.Timestamp),
		})
	}
	return steps, traceURL, nil
}

// buildObsAgents joins latest snapshots with per-actor 24h state timelines.
// Roster order is deterministic: by role, then actor id.
func buildObsAgents(agents []obsStatusAgent, histories map[string]obsAgentHistory) []ObsAgentView {
	views := make([]ObsAgentView, 0, len(agents))
	for _, a := range agents {
		v := ObsAgentView{
			Role:       a.Role,
			ActorID:    a.ActorID,
			State:      a.State,
			Model:      a.Model,
			Iteration:  a.Iteration,
			MaxIter:    a.MaxIterations,
			TokensUsed: a.TokensUsed,
			CostUSD:    fmt.Sprintf("%.2f", a.CostUSD),
			Trust:      "unknown",
			Errors:     a.Errors,
		}
		if a.TrustScore != nil {
			v.Trust = fmt.Sprintf("%.2f", *a.TrustScore)
		}
		if !a.LastEventAt.IsZero() {
			v.LastEventAt = a.LastEventAt.UTC().Format("2006-01-02 15:04:05")
		}
		if hist, ok := histories[a.ActorID]; ok {
			v.TimelineSVG = svgviz.SpanStrip(obsSpans(hist.States), 320, 12)
		}
		views = append(views, v)
	}
	sort.Slice(views, func(i, j int) bool {
		if views[i].Role != views[j].Role {
			return views[i].Role < views[j].Role
		}
		return views[i].ActorID < views[j].ActorID
	})
	return views
}

func obsSpans(states []obsStateSpan) []svgviz.Span {
	spans := make([]svgviz.Span, 0, len(states))
	for _, s := range states {
		spans = append(spans, svgviz.Span{
			Label:   s.State + " from " + s.EnteredAt.UTC().Format("15:04:05"),
			Seconds: s.Duration,
			Kind:    s.State,
		})
	}
	return spans
}

func obsStaircaseSteps(steps []ObsTraceStep) []svgviz.Step {
	out := make([]svgviz.Step, 0, len(steps))
	for _, s := range steps {
		out = append(out, svgviz.Step{Label: s.Label, Sub: s.Sub + " @ " + s.At})
	}
	return out
}

// phaseCostBars renders per-phase cost from the pipeline report. Phases with
// non-positive cost are kept in the labels (a free phase is information) but
// the chart fails closed if no phase carries cost.
func phaseCostBars(report *OpsPipelineReport) (string, []string, float64) {
	if report == nil || len(report.Phases) == 0 {
		return "", nil, 0
	}
	values := make([]float64, 0, len(report.Phases))
	labels := make([]string, 0, len(report.Phases))
	total := 0.0
	for _, p := range report.Phases {
		cost := p.CostUSD
		if cost < 0 {
			return "", nil, 0 // a negative cost is undrawable; render the unavailable block
		}
		values = append(values, cost)
		labels = append(labels, fmt.Sprintf("%s/%s $%.2f (%s)", p.WorkflowStage, p.Phase, cost, p.Outcome))
		total += cost
	}
	return svgviz.Bars(values, labels, 720, 80), labels, total
}
