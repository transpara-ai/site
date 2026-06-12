package graph

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/transpara-ai/site/graph/svgviz"
)

// The observatory is the read-only civilization transparency surface
// (dark-factory transparency contract T1–T7, observatory phase-3 plan).
// It consumes egress APIs only — work /telemetry/status, /telemetry/agents/history,
// /tasks/{id}/events and the hive operator projection — and performs no writes.
//
// Fail-open is the enemy here: an omitted JSON field must never render as a
// fact (0, false, "no cost"). Feeder scalars decode as pointers; nil renders
// as an explicit unknown. Every withheld visual states its real reason.

// obsWorkClient bounds every observatory fetch to the Work API so a hung
// feeder renders an unavailable panel instead of hanging the page.
var obsWorkClient = &http.Client{Timeout: 5 * time.Second}

type OpsObservatoryData struct {
	GeneratedAt string

	// Vitals (work /telemetry/status)
	StatusURL   string
	Vitals      *ObsVitals
	VitalsError string

	// Spend vs cap, derived from vitals with explicit reasons (T3)
	Spend ObsSpend

	// Agent roster + 24h state timelines (status + /telemetry/agents/history)
	HistoryURL   string
	Agents       []ObsAgentView
	HistoryError string

	// Pipeline phase costs (reuses the telemetry fetch)
	Telemetry        *OpsTelemetryData
	PhaseCostBarsSVG string
	PhaseCostTotal   string
	PhaseCostLabels  []string
	PhaseCostReason  string // non-empty: chart withheld for this stated reason

	// Authority, lifecycle, audit traces (hive operator projection via fetchOpsHive)
	Hive *OpsHiveData

	// Causal trace explorer (work /tasks/{id}/events, on demand via ?task=)
	TraceTaskID string
	TraceURL    string
	TraceSteps  []ObsTraceStep
	TraceSVG    string
	TraceError  string
}

// ObsSpend carries either a drawable spend-vs-cap state or the explicit
// reason it is withheld. Exactly one of (GaugeSVG+Text) or Reason is set.
type ObsSpend struct {
	GaugeSVG string
	Text     string
	Reason   string
}

// ObsVitals mirrors the hive snapshot served by work /telemetry/status.
// Every scalar is a pointer: an omitted field is unknown, not zero/false.
type ObsVitals struct {
	ActiveAgents *int     `json:"active_agents"`
	TotalActors  *int     `json:"total_actors"`
	ChainLength  *int64   `json:"chain_length"`
	ChainOK      *bool    `json:"chain_ok"`
	EventRate    *float64 `json:"event_rate"`
	DailyCost    *float64 `json:"daily_cost"`
	DailyCap     *float64 `json:"daily_cap"`
	Severity     string   `json:"severity"`
}

type obsStatusAgent struct {
	Role          string     `json:"role"`
	ActorID       string     `json:"actor_id"`
	State         string     `json:"state"`
	Model         string     `json:"model"`
	Iteration     *int       `json:"iteration"`
	MaxIterations *int       `json:"max_iterations"`
	TokensUsed    *int64     `json:"tokens_used"`
	CostUSD       *float64   `json:"cost_usd"`
	TrustScore    *float64   `json:"trust_score"`
	LastEventAt   *time.Time `json:"last_event_at"`
	Errors        *int       `json:"errors"`
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

// ObsAgentView is one roster row. String fields carry "unknown" (or "—") when
// the feeder omitted the value — never a fabricated zero.
type ObsAgentView struct {
	Role        string
	ActorID     string
	State       string
	Model       string
	Iterations  string // "n/m", with unknown parts rendered as "?"
	Tokens      string
	CostUSD     string
	Trust       string
	Errors      string
	LastEventAt string
	TimelineSVG string
}

type ObsTraceStep struct {
	Label string
	Sub   string
	At    string
}

// authorityOutcomes is the canonical DF-SPEC-0004 vocabulary, allowlisted.
// Anything else renders as an explicit non-canonical value (T1/T6).
var authorityOutcomes = map[string]bool{
	"Autonomous":       true,
	"Notify":           true,
	"ApprovalRequired": true,
	"Forbidden":        true,
}

// obsCanonicalOutcome reports whether the feeder-supplied outcome is one of
// the four canonical authority outcomes.
func obsCanonicalOutcome(s string) bool { return authorityOutcomes[strings.TrimSpace(s)] }

func (h *Handlers) handleOpsObservatory(w http.ResponseWriter, r *http.Request) {
	data := &OpsObservatoryData{GeneratedAt: time.Now().UTC().Format("2006-01-02 15:04:05 UTC")}

	status, statusURL, err := fetchObservatoryStatus(r)
	data.StatusURL = statusURL
	if err != nil {
		data.VitalsError = err.Error()
		data.Spend = ObsSpend{Reason: "vitals unavailable: " + err.Error()}
	} else {
		data.Vitals = status.Hive
		data.Spend = buildObsSpend(status.Hive)
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
		data.PhaseCostBarsSVG, data.PhaseCostLabels, data.PhaseCostTotal, data.PhaseCostReason = phaseCostBars(data.Telemetry.Pipeline)
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

// buildObsSpend validates spend inputs and states the true reason whenever
// the gauge is withheld. "Absent" is reserved for actually-missing fields;
// present-but-invalid values are called invalid (review finding 2: a present
// cap <= 0 must not be reported as "cap unknown").
func buildObsSpend(v *ObsVitals) ObsSpend {
	if v == nil {
		return ObsSpend{Reason: "no hive snapshot recorded yet (telemetry writer has not run)"}
	}
	switch {
	case v.DailyCost == nil && v.DailyCap == nil:
		return ObsSpend{Reason: "recorded spend and daily cap both absent from the snapshot"}
	case v.DailyCost == nil:
		return ObsSpend{Reason: "recorded spend absent from the snapshot — gauge withheld"}
	case v.DailyCap == nil:
		return ObsSpend{Reason: fmt.Sprintf("recorded spend today: $%.2f; daily cap absent from the snapshot — gauge withheld rather than drawn against a guessed cap (T3)", *v.DailyCost)}
	case *v.DailyCost < 0:
		return ObsSpend{Reason: fmt.Sprintf("recorded spend is negative ($%.2f) — invalid feeder data, gauge withheld", *v.DailyCost)}
	case *v.DailyCap <= 0:
		return ObsSpend{Reason: fmt.Sprintf("daily cap is non-positive ($%.2f) — invalid feeder data, gauge withheld", *v.DailyCap)}
	}
	svg := svgviz.Gauge(*v.DailyCost, *v.DailyCap, 320, 22)
	if svg == "" {
		return ObsSpend{Reason: "gauge renderer declined validated input — report this as a bug"}
	}
	return ObsSpend{
		GaugeSVG: svg,
		Text:     fmt.Sprintf("$%.2f spent of $%.2f daily cap", *v.DailyCost, *v.DailyCap),
	}
}

func fetchObservatoryStatus(r *http.Request) (*obsStatusResponse, string, error) {
	statusURL := legacyWorkURL(serverWorkAPIBaseURL(), "/telemetry/status")
	payload := &obsStatusResponse{}
	if err := obsGetJSON(r, statusURL, payload); err != nil {
		return nil, statusURL, err
	}
	return payload, statusURL, nil
}

func fetchObservatoryHistory(r *http.Request) (map[string]obsAgentHistory, string, error) {
	historyURL := legacyWorkURL(serverWorkAPIBaseURL(), "/telemetry/agents/history") + "?window=24h"
	payload := &obsHistoryResponse{}
	if err := obsGetJSON(r, historyURL, payload); err != nil {
		return nil, historyURL, err
	}
	byActor := make(map[string]obsAgentHistory, len(payload.Agents))
	for _, a := range payload.Agents {
		byActor[a.ActorID] = a
	}
	return byActor, historyURL, nil
}

// obsTaskIDPattern allowlists task IDs (event IDs are UUIDs; allow common id
// charsets). Anything outside it is rejected before a request is built, so a
// hostile ID can never address an endpoint other than /tasks/{id}/events.
var obsTaskIDPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,128}$`)

// fetchObservatoryTrace validates the operator-supplied task ID against the
// allowlist, then verifies the feeder answered for the task that was asked.
func fetchObservatoryTrace(r *http.Request, taskID string) ([]ObsTraceStep, string, error) {
	if !obsTaskIDPattern.MatchString(taskID) {
		return nil, "", fmt.Errorf("task ID rejected: must match %s", obsTaskIDPattern.String())
	}
	traceURL := legacyWorkURL(serverWorkAPIBaseURL(), "/tasks/"+taskID+"/events")
	payload := &obsTraceResponse{}
	if err := obsGetJSON(r, traceURL, payload); err != nil {
		return nil, traceURL, err
	}
	// Fail closed: the feeder must echo exactly the requested task id. An
	// empty echo is unverifiable, not acceptable (round-2 review finding).
	if payload.TaskID != taskID {
		if payload.TaskID == "" {
			return nil, traceURL, fmt.Errorf("feeder response did not echo a task id — trail unverifiable, withheld")
		}
		return nil, traceURL, fmt.Errorf("feeder returned trail for task %q, not the requested %q", payload.TaskID, taskID)
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

// obsGetJSON is the one fetch path for the observatory: bounded client,
// work auth, JSON decode, explicit errors.
func obsGetJSON(r *http.Request, rawURL string, into any) error {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	setWorkAuth(req)
	resp, err := obsWorkClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("feeder returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(into)
}

// buildObsAgents joins latest snapshots with per-actor 24h state timelines.
// Absent scalars render as "—"/"unknown", never as zero. Roster order is
// deterministic: by role, then actor id.
func buildObsAgents(agents []obsStatusAgent, histories map[string]obsAgentHistory) []ObsAgentView {
	views := make([]ObsAgentView, 0, len(agents))
	for _, a := range agents {
		v := ObsAgentView{
			Role:        a.Role,
			ActorID:     a.ActorID,
			State:       orUnknown(a.State),
			Model:       orUnknown(a.Model),
			Iterations:  obsIntPair(a.Iteration, a.MaxIterations),
			Tokens:      obsInt64(a.TokensUsed),
			CostUSD:     obsMoney(a.CostUSD),
			Trust:       obsScore(a.TrustScore),
			Errors:      obsInt(a.Errors),
			LastEventAt: "unknown",
		}
		if a.LastEventAt != nil && !a.LastEventAt.IsZero() {
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

func orUnknown(s string) string {
	if strings.TrimSpace(s) == "" {
		return "unknown"
	}
	return s
}

func obsInt(p *int) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprint(*p)
}

func obsInt64(p *int64) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprint(*p)
}

func obsMoney(p *float64) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprintf("%.2f", *p)
}

func obsScore(p *float64) string {
	if p == nil {
		return "unknown"
	}
	return fmt.Sprintf("%.2f", *p)
}

func obsIntPair(a, b *int) string {
	left, right := "?", "?"
	if a != nil {
		left = fmt.Sprint(*a)
	}
	if b != nil {
		right = fmt.Sprint(*b)
	}
	if left == "?" && right == "?" {
		return "—"
	}
	return left + "/" + right
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
		out = append(out, svgviz.Step{Label: s.Label, Sub: s.Sub + " at " + s.At})
	}
	return out
}

// phaseCostBars renders per-phase cost from the pipeline report. It returns
// an explicit reason whenever the chart is withheld so the template never
// converts undrawable data into an all-zero story (review finding 3).
// Caveat: the pipeline report serializes cost as a plain number, so an
// upstream-omitted cost arrives as 0 — the template wording claims only what
// the report records.
func phaseCostBars(report *OpsPipelineReport) (svg string, labels []string, total string, reason string) {
	if report == nil || len(report.Phases) == 0 {
		return "", nil, "", "no phases recorded in the pipeline report"
	}
	values := make([]float64, 0, len(report.Phases))
	labels = make([]string, 0, len(report.Phases))
	sum := 0.0
	for _, p := range report.Phases {
		if p.CostUSD < 0 {
			return "", nil, "", fmt.Sprintf("pipeline report contains a negative phase cost ($%.2f for %s/%s) — invalid feeder data, chart withheld", p.CostUSD, p.WorkflowStage, p.Phase)
		}
		values = append(values, p.CostUSD)
		labels = append(labels, fmt.Sprintf("%s/%s $%.2f (%s)", p.WorkflowStage, p.Phase, p.CostUSD, p.Outcome))
		sum += p.CostUSD
	}
	if sum > 0 {
		total = fmt.Sprintf("%.2f", sum)
	}
	svg = svgviz.Bars(values, labels, 720, 80)
	// Fail closed: a renderer refusal on a non-zero series must never fall
	// through to the all-zero wording (round-2 review finding).
	if svg == "" && sum > 0 {
		return "", labels, total, fmt.Sprintf("chart renderer declined %d phases at this size — chart withheld; recorded costs listed below", len(values))
	}
	return svg, labels, total, ""
}
