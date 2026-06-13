package graph

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
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

// obsSSEClient bounds the upstream SSE handshake but leaves the body stream
// open for the browser's EventSource. A normal http.Client Timeout would cut
// off healthy long-lived streams.
var obsSSEClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       30 * time.Second,
	},
}

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

	// Civilization assembly: role/agent topology derived from Work runtime
	// status, Hive lifecycle/model projection, and a fallback snapshot of the
	// Hive bootstrap role registry shape. It is display-only, never authority.
	Civilization ObsCivilization

	// Causal trace explorer (work /tasks/{id}/events, on demand via ?task=)
	TraceTaskID string
	TraceURL    string
	TraceSteps  []ObsTraceStep
	TraceSVG    string
	TraceError  string

	// Live pulse (Site proxy -> work /telemetry/sse)
	EventPulseURL    string
	EventPulseSource string
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

type ObsCivilization struct {
	OrgLevels        []ObsOrgLevel
	Roster           []ObsCivilizationRole
	Emergence        []ObsEmergenceStep
	Findings         []string
	GlobalModelMode  string
	GlobalModeReason string
	ModelSource      string
}

type ObsOrgLevel struct {
	Tier   string
	Label  string
	Used   string
	Detail string
}

type ObsCivilizationRole struct {
	Role           string
	Agent          string
	Tier           string
	Category       string
	Origin         string
	Status         string
	CanOperate     bool
	Model          string
	ModelMode      string
	ReportsTo      string
	EscalationPath string
	Why            string
	Evidence       string
}

type ObsEmergenceStep struct {
	Subject  string
	State    string
	Why      string
	Evidence string
}

type obsStarterRole struct {
	Role           string
	Tier           string
	Category       string
	CanOperate     bool
	ReportsTo      string
	EscalationPath string
	Why            string
}

// obsStarterRoles is a Site fallback snapshot of Hive's StarterAgents /
// StarterRoleDefinitions bootstrap shape. Live runtime status, model policy,
// and emergence evidence must still come from Hive/Work projections.
var obsStarterRoles = []obsStarterRole{
	{Role: "guardian", Tier: "A", Category: "process", CanOperate: false, ReportsTo: "human", EscalationPath: "human", Why: "integrity monitor for soul violations, authority overreach, and policy breaches"},
	{Role: "sysmon", Tier: "A", Category: "process", CanOperate: false, ReportsTo: "guardian", EscalationPath: "guardian", Why: "operational health monitor"},
	{Role: "allocator", Tier: "A", Category: "process", CanOperate: false, ReportsTo: "guardian", EscalationPath: "guardian", Why: "resource and token-budget manager"},
	{Role: "cto", Tier: "A", Category: "leadership", CanOperate: false, ReportsTo: "human", EscalationPath: "human", Why: "architecture leader and structural-gap detector"},
	{Role: "spawner", Tier: "A", Category: "staffing", CanOperate: false, ReportsTo: "cto", EscalationPath: "guardian", Why: "growth engine that proposes new roles when gaps are detected"},
	{Role: "reviewer", Tier: "A", Category: "technical", CanOperate: false, ReportsTo: "cto", EscalationPath: "cto", Why: "quality gate for implementer output"},
	{Role: "strategist", Tier: "A", Category: "leadership", CanOperate: false, ReportsTo: "cto", EscalationPath: "human", Why: "decomposes seed ideas into high-level tasks"},
	{Role: "planner", Tier: "A", Category: "technical", CanOperate: false, ReportsTo: "strategist", EscalationPath: "cto", Why: "turns high-level tasks into implementable subtasks"},
	{Role: "implementer", Tier: "A", Category: "technical", CanOperate: true, ReportsTo: "strategist", EscalationPath: "cto", Why: "writes code, runs tests, and completes filesystem-backed work"},
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
	data := &OpsObservatoryData{
		GeneratedAt:      time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		EventPulseURL:    "/ops/observatory/events",
		EventPulseSource: legacyWorkURL(serverWorkAPIBaseURL(), "/telemetry/sse"),
	}

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
	data.Civilization = buildObsCivilization(data.Agents, data.Hive)

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

func (h *Handlers) handleOpsObservatoryEvents(w http.ResponseWriter, r *http.Request) {
	streamObservatoryEvents(w, r)
}

func streamObservatoryEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "event pulse streaming is not supported by this response writer", http.StatusInternalServerError)
		return
	}

	upstreamURL := legacyWorkURL(serverWorkAPIBaseURL(), "/telemetry/sse")
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstreamURL, nil)
	if err != nil {
		http.Error(w, "event pulse request could not be built: "+err.Error(), http.StatusInternalServerError)
		return
	}
	setWorkAuth(req)

	resp, err := obsSSEClient.Do(req)
	if err != nil {
		http.Error(w, "event pulse feeder unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		http.Error(w, "event pulse feeder returned "+resp.Status, http.StatusBadGateway)
		return
	}
	if ct := strings.ToLower(resp.Header.Get("Content-Type")); !strings.HasPrefix(ct, "text/event-stream") {
		http.Error(w, "event pulse feeder did not return text/event-stream", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	fmt.Fprint(w, ": site observatory event proxy connected\n\n")
	flusher.Flush()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !obsForwardableSSELine(line) {
			continue
		}
		fmt.Fprint(w, line+"\n")
		if line == "" {
			flusher.Flush()
		}
	}
	if err := scanner.Err(); err != nil && r.Context().Err() == nil {
		obsWriteSSEError(w, flusher, "event pulse upstream read failed")
	}
}

func obsForwardableSSELine(line string) bool {
	if line == "" {
		return true
	}
	for _, prefix := range []string{"data:", "id:", "retry:", ":"} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func obsWriteSSEError(w http.ResponseWriter, flusher http.Flusher, message string) {
	payload, err := json.Marshal(map[string]string{"error": message})
	if err != nil {
		payload = []byte(`{"error":"event pulse failed"}`)
	}
	fmt.Fprintf(w, "event: site-error\ndata: %s\n\n", payload)
	flusher.Flush()
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

func buildObsCivilization(agents []ObsAgentView, hive *OpsHiveData) ObsCivilization {
	civ := ObsCivilization{
		OrgLevels: []ObsOrgLevel{
			{Tier: "A", Label: "Bootstrap / foundation", Used: "defined", Detail: "StarterAgents registry; runtime activity requires Work/Hive projection evidence"},
			{Tier: "B", Label: "Organic emergence", Used: "not projected", Detail: "used when approved dynamic roles appear as spawned lifecycle records"},
			{Tier: "C", Label: "Business operations", Used: "not projected", Detail: "taxonomy level exists; no current projection evidence in this Site slice"},
			{Tier: "D", Label: "Self-governance", Used: "not projected", Detail: "taxonomy level exists; no current projection evidence in this Site slice"},
		},
		GlobalModelMode:  "Auto",
		GlobalModeReason: "inferred from Hive selector policy metadata until Hive projects an explicit global Model Selection Mode",
		ModelSource:      "Hive operator projection",
	}
	if hive == nil {
		civ.GlobalModelMode = "unknown"
		civ.GlobalModeReason = "Hive projection not fetched"
		civ.ModelSource = "hive operator projection unavailable"
		civ.Roster = obsBootstrapRoster(nil, nil)
		civ.Emergence = append(civ.Emergence, ObsEmergenceStep{
			Subject:  "role emergence",
			State:    "not projected",
			Why:      "Hive lifecycle and authority projection were not available to Site",
			Evidence: "HIVE_OPS_API_BASE_URL / operator projection",
		})
		civ.Findings = append(civ.Findings, "Only bootstrap definitions are visible. Runtime activity is withheld until Hive/Work projections are available.")
		return civ
	}
	if hive.ProjectionError != "" {
		civ.Findings = append(civ.Findings, "Hive projection reported an error: "+hive.ProjectionError)
	}
	if hive.ModelSelection.Source == "" && len(hive.ModelSelection.Assignments) == 0 {
		civ.GlobalModelMode = "unknown"
		civ.GlobalModeReason = "Hive did not return model-selection projection metadata"
	} else if mode := obsHiveProjectionModelMode(hive.ModelSelection); mode != "" {
		civ.GlobalModelMode = mode
		civ.GlobalModeReason = "projected by Hive model-selection metadata"
	}

	lifecycleByRole := obsLifecycleByRole(hive.Lifecycle)
	agentsByRole := obsAgentsByRole(agents)
	assignmentsByRole := obsModelAssignmentsByRole(hive.ModelSelection.Assignments)
	civ.Roster = obsBootstrapRoster(lifecycleByRole, agentsByRole)
	for i := range civ.Roster {
		role := strings.ToLower(civ.Roster[i].Role)
		if assignment, ok := assignmentsByRole[role]; ok {
			civ.Roster[i].Model = obsFirstNonEmpty(assignment.Model, assignment.PolicyModel, civ.Roster[i].Model)
			civ.Roster[i].ModelMode = obsAssignmentModelMode(hive.ModelSelection, assignment)
		}
		if civ.Roster[i].Model == "" {
			civ.Roster[i].Model = "not projected"
		}
		if civ.Roster[i].ModelMode == "" {
			civ.Roster[i].ModelMode = civ.GlobalModelMode
		}
	}

	bootstrap := map[string]bool{}
	for _, role := range obsStarterRoles {
		bootstrap[role.Role] = true
	}
	for _, l := range hive.Lifecycle {
		role := strings.ToLower(strings.TrimSpace(l.Role))
		if role == "" || bootstrap[role] {
			continue
		}
		civ.Roster = append(civ.Roster, ObsCivilizationRole{
			Role:           role,
			Agent:          obsFirstNonEmpty(l.DisplayName, l.ActorID, "projected actor"),
			Tier:           "B?",
			Category:       "emergent/runtime",
			Origin:         "runtime-projected",
			Status:         obsFirstNonEmpty(l.LifecycleStatus, "projected"),
			CanOperate:     false,
			Model:          "not projected",
			ModelMode:      civ.GlobalModelMode,
			ReportsTo:      "not projected",
			EscalationPath: "not projected",
			Why:            "role is not in the bootstrap registry and appears through Hive lifecycle projection",
			Evidence:       "Hive lifecycle projection",
		})
	}
	sort.Slice(civ.Roster, func(i, j int) bool {
		if civ.Roster[i].Tier != civ.Roster[j].Tier {
			return civ.Roster[i].Tier < civ.Roster[j].Tier
		}
		return civ.Roster[i].Role < civ.Roster[j].Role
	})

	civ.Emergence = buildObsEmergence(hive)
	civ.OrgLevels = obsOrgLevels(civ.Roster)
	civ.Findings = append(civ.Findings, obsCivilizationFindings(civ, hive)...)
	return civ
}

func obsBootstrapRoster(lifecycleByRole map[string][]OpsHiveLifecycle, agentsByRole map[string][]ObsAgentView) []ObsCivilizationRole {
	rows := make([]ObsCivilizationRole, 0, len(obsStarterRoles))
	for _, role := range obsStarterRoles {
		key := strings.ToLower(role.Role)
		row := ObsCivilizationRole{
			Role:           role.Role,
			Agent:          "not runtime-projected",
			Tier:           role.Tier,
			Category:       role.Category,
			Origin:         "bootstrap registry",
			Status:         "defined, not runtime-projected",
			CanOperate:     role.CanOperate,
			Model:          "not projected",
			ModelMode:      "unknown",
			ReportsTo:      role.ReportsTo,
			EscalationPath: role.EscalationPath,
			Why:            role.Why,
			Evidence:       "Hive StarterAgents / StarterRoleDefinitions",
		}
		if lifecycleByRole != nil {
			if lifecycle := lifecycleByRole[key]; len(lifecycle) > 0 {
				row.Agent = obsLifecycleActors(lifecycle)
				row.Status = obsLifecycleStatuses(lifecycle)
				row.Evidence = "Hive lifecycle projection + bootstrap registry"
			}
		}
		if agentsByRole != nil {
			if runtimeAgents := agentsByRole[key]; len(runtimeAgents) > 0 {
				row.Agent = obsRuntimeActors(runtimeAgents, row.Agent)
				row.Status = obsRuntimeStatuses(runtimeAgents, row.Status)
				if model := obsRuntimeModel(runtimeAgents); model != "" {
					row.Model = model
				}
				row.Evidence = "Work runtime telemetry + " + row.Evidence
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func buildObsEmergence(hive *OpsHiveData) []ObsEmergenceStep {
	if hive == nil {
		return nil
	}
	steps := make([]ObsEmergenceStep, 0)
	for _, a := range hive.PendingApprovals {
		if strings.Contains(strings.ToLower(a.ActionName), "agent.spawn") || strings.Contains(strings.ToLower(a.ActionName), "role") {
			steps = append(steps, ObsEmergenceStep{
				Subject:  obsFirstNonEmpty(a.Target, a.ActionName),
				State:    "awaiting approval",
				Why:      obsFirstNonEmpty(a.Justification, a.RiskSummary, "pending protected action"),
				Evidence: "authority request " + a.RequestID,
			})
		}
	}
	for _, d := range hive.AuthorityDecisions {
		action := strings.ToLower(obsFirstNonEmpty(d.ApprovedAction, d.RequestedAction))
		if strings.Contains(action, "agent.spawn") || strings.Contains(action, "role") {
			steps = append(steps, ObsEmergenceStep{
				Subject:  obsFirstNonEmpty(d.ApprovedTarget, d.RequestedTarget, d.RequestID),
				State:    obsFirstNonEmpty(d.Outcome, "recorded"),
				Why:      obsFirstNonEmpty(d.Rationale, "authority decision recorded"),
				Evidence: "authority decision " + obsFirstNonEmpty(d.DecisionID, d.EventID),
			})
		}
	}
	bootstrap := map[string]bool{}
	for _, role := range obsStarterRoles {
		bootstrap[role.Role] = true
	}
	for _, l := range hive.Lifecycle {
		role := strings.ToLower(strings.TrimSpace(l.Role))
		if role == "" || bootstrap[role] {
			continue
		}
		steps = append(steps, ObsEmergenceStep{
			Subject:  obsFirstNonEmpty(l.DisplayName, role),
			State:    obsFirstNonEmpty(l.LifecycleStatus, "projected"),
			Why:      "non-bootstrap role visible in lifecycle projection",
			Evidence: "lifecycle actor " + obsFirstNonEmpty(l.ActorID, role),
		})
	}
	sort.Slice(steps, func(i, j int) bool {
		if steps[i].State != steps[j].State {
			return steps[i].State < steps[j].State
		}
		return steps[i].Subject < steps[j].Subject
	})
	return steps
}

func obsOrgLevels(roster []ObsCivilizationRole) []ObsOrgLevel {
	levels := []ObsOrgLevel{
		{Tier: "A", Label: "Bootstrap / foundation"},
		{Tier: "B", Label: "Organic emergence"},
		{Tier: "C", Label: "Business operations"},
		{Tier: "D", Label: "Self-governance"},
	}
	counts := map[string]int{}
	projected := map[string]int{}
	for _, row := range roster {
		tier := strings.TrimSuffix(row.Tier, "?")
		if tier == "" {
			tier = "unknown"
		}
		counts[tier]++
		if row.Status != "defined, not runtime-projected" {
			projected[tier]++
		}
	}
	for i := range levels {
		n := counts[levels[i].Tier]
		p := projected[levels[i].Tier]
		switch {
		case n == 0:
			levels[i].Used = "not projected"
			levels[i].Detail = "no roles visible at this level in the current projection"
		case p == 0:
			levels[i].Used = "defined"
			levels[i].Detail = fmt.Sprintf("%d role(s) defined; none runtime-projected", n)
		default:
			levels[i].Used = "projected"
			levels[i].Detail = fmt.Sprintf("%d role(s) visible; %d runtime/lifecycle-projected", n, p)
		}
	}
	return levels
}

func obsCivilizationFindings(civ ObsCivilization, hive *OpsHiveData) []string {
	findings := []string{
		"Starter role rows use a Site fallback snapshot of Hive bootstrap definitions; runtime status, model policy, and emergence evidence require live Hive/Work projection.",
		"CanOperate is rendered as a capability flag only; it is not bootstrap membership or authority.",
		"Auto model-selection mode expands routing flexibility only; it does not expand agent authority.",
	}
	if hive == nil || hive.ModelSelection.Source == "" {
		findings = append(findings, "Model Selection Mode is not yet first-class in Hive projection; Site shows the intended control plane without persisting it.")
	}
	if len(civ.Emergence) == 0 {
		findings = append(findings, "No emergent role queue is visible in the current Hive projection.")
	}
	return findings
}

func obsLifecycleByRole(items []OpsHiveLifecycle) map[string][]OpsHiveLifecycle {
	out := make(map[string][]OpsHiveLifecycle)
	for _, item := range items {
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role == "" {
			continue
		}
		out[role] = append(out[role], item)
	}
	return out
}

func obsAgentsByRole(items []ObsAgentView) map[string][]ObsAgentView {
	out := make(map[string][]ObsAgentView)
	for _, item := range items {
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role == "" || role == "unknown" {
			continue
		}
		out[role] = append(out[role], item)
	}
	return out
}

func obsModelAssignmentsByRole(items []OpsHiveModelRoleAssignment) map[string]OpsHiveModelRoleAssignment {
	out := make(map[string]OpsHiveModelRoleAssignment)
	for _, item := range items {
		role := strings.ToLower(strings.TrimSpace(item.Role))
		if role != "" {
			out[role] = item
		}
	}
	return out
}

func obsAssignmentModelMode(selection OpsHiveModelSelection, item OpsHiveModelRoleAssignment) string {
	if mode := obsCanonicalModelMode(obsFirstNonEmpty(item.EffectiveMode, item.OverrideMode, item.SelectionMode)); mode != "" {
		return mode
	}
	if item.PolicyEventID != "" || item.Source == "hive-model-policy-event" {
		return "Manual"
	}
	if item.SelectionStrategy != "" || item.PreferredTier != "" || len(item.RequiredCapabilities) > 0 {
		return "Auto"
	}
	if mode := obsCanonicalModelMode(obsFirstNonEmpty(selection.GlobalMode, selection.SelectionMode)); mode != "" {
		return mode
	}
	if item.Model != "" || item.PolicyModel != "" {
		return "Manual"
	}
	return "unknown"
}

func obsHiveProjectionModelMode(selection OpsHiveModelSelection) string {
	if mode := obsCanonicalModelMode(obsFirstNonEmpty(selection.GlobalMode, selection.SelectionMode)); mode != "" {
		return mode
	}
	if selection.Source != "" || len(selection.Assignments) > 0 {
		return "Auto"
	}
	return "unknown"
}

func obsCanonicalModelMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "auto", "automatic":
		return "Auto"
	case "manual", "pinned":
		return "Manual"
	default:
		return ""
	}
}

func obsFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func obsLifecycleActors(items []OpsHiveLifecycle) string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, obsFirstNonEmpty(item.DisplayName, item.ActorID))
	}
	return strings.Join(obsUniqueNonEmpty(values), ", ")
}

func obsLifecycleStatuses(items []OpsHiveLifecycle) string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, obsFirstNonEmpty(item.LifecycleStatus, "projected"))
	}
	return strings.Join(obsUniqueNonEmpty(values), ", ")
}

func obsRuntimeActors(items []ObsAgentView, fallback string) string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, obsFirstNonEmpty(item.ActorID, item.Role))
	}
	if out := strings.Join(obsUniqueNonEmpty(values), ", "); out != "" {
		return out
	}
	return fallback
}

func obsRuntimeStatuses(items []ObsAgentView, fallback string) string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, obsFirstNonEmpty(item.State, "projected"))
	}
	if out := strings.Join(obsUniqueNonEmpty(values), ", "); out != "" {
		return out
	}
	return fallback
}

func obsRuntimeModel(items []ObsAgentView) string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		if item.Model != "unknown" {
			values = append(values, item.Model)
		}
	}
	return strings.Join(obsUniqueNonEmpty(values), ", ")
}

func obsUniqueNonEmpty(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
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
