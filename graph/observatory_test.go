package graph

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

func fptr(f float64) *float64 { return &f }
func iptr(i int) *int         { return &i }

func TestBuildObsAgentsJoinsHistoryAndSorts(t *testing.T) {
	now := time.Date(2026, 6, 13, 1, 2, 3, 0, time.UTC)
	i1, i5 := 1, 5
	tok := int64(1234)
	e0 := 0
	agents := []obsStatusAgent{
		{Role: "scout", ActorID: "b-2", State: "processing", CostUSD: fptr(1.5), TrustScore: fptr(0.42), Iteration: &i1, MaxIterations: &i5, TokensUsed: &tok, Errors: &e0, LastEventAt: &now},
		{Role: "builder", ActorID: "a-1"}, // feeder omitted every scalar
	}
	histories := map[string]obsAgentHistory{
		"b-2": {ActorID: "b-2", States: []obsStateSpan{{State: "processing", Duration: 60, EnteredAt: now}}},
	}
	views := buildObsAgents(agents, histories)
	if len(views) != 2 {
		t.Fatalf("want 2 views, got %d", len(views))
	}
	if views[0].Role != "builder" || views[1].Role != "scout" {
		t.Errorf("views must sort by role: got %s, %s", views[0].Role, views[1].Role)
	}
	// The omitted-everything agent must render unknowns, never zeros.
	b := views[0]
	if b.CostUSD != "—" || b.Tokens != "—" || b.Errors != "—" || b.Iterations != "—" {
		t.Errorf("omitted scalars must render as —, got cost=%q tokens=%q errors=%q iter=%q", b.CostUSD, b.Tokens, b.Errors, b.Iterations)
	}
	if b.Trust != "unknown" || b.LastEventAt != "unknown" || b.State != "unknown" {
		t.Errorf("omitted trust/lastEvent/state must render unknown, got %q %q %q", b.Trust, b.LastEventAt, b.State)
	}
	if b.TimelineSVG != "" {
		t.Error("agent without history must have empty timeline (template renders the reason)")
	}
	s := views[1]
	if s.Trust != "0.42" || s.CostUSD != "1.50" || s.Iterations != "1/5" {
		t.Errorf("present scalars must format, got trust=%q cost=%q iter=%q", s.Trust, s.CostUSD, s.Iterations)
	}
	if s.TimelineSVG == "" {
		t.Error("agent with history must have a timeline strip")
	}
}

func TestBuildObsCivilizationSeparatesBootstrapRuntimeAndEmergentRoles(t *testing.T) {
	hive := &OpsHiveData{
		Lifecycle: []OpsHiveLifecycle{
			{ActorID: "act-guardian", DisplayName: "guardian", Role: "guardian", LifecycleStatus: "active"},
			{ActorID: "act-cartographer", DisplayName: "cartographer", Role: "cartographer", LifecycleStatus: "active"},
		},
		PendingApprovals: []OpsHiveApproval{{
			RequestID:     "req-spawn-cartographer",
			ActionName:    "agent.spawn.persistent",
			Target:        "cartographer",
			Justification: "map system topology",
		}},
		ModelSelection: OpsHiveModelSelection{
			Source: "hive",
			Assignments: []OpsHiveModelRoleAssignment{
				{Role: "guardian", Model: "claude-sonnet", SelectionStrategy: "balanced", Source: "starter-role-definition"},
				{Role: "implementer", Model: "claude-opus", Source: "hive-model-policy-event", PolicyEventID: "ev-model"},
			},
		},
	}
	civ := buildObsCivilization([]ObsAgentView{{Role: "guardian", ActorID: "run-guardian", State: "processing", Model: "claude-sonnet"}}, hive)
	if len(civ.Roster) < len(obsStarterRoles)+1 {
		t.Fatalf("roster missing roles: got %d", len(civ.Roster))
	}
	byRole := map[string]ObsCivilizationRole{}
	for _, row := range civ.Roster {
		byRole[row.Role] = row
	}
	guardian := byRole["guardian"]
	if guardian.Status != "processing" || guardian.Agent == "not runtime-projected" {
		t.Fatalf("guardian should reflect runtime projection, got %+v", guardian)
	}
	if guardian.ModelMode != "Auto" {
		t.Fatalf("guardian mode = %q, want Auto", guardian.ModelMode)
	}
	if guardian.ModelModeProvenance != "inferred" {
		t.Fatalf("guardian mode provenance = %q, want inferred", guardian.ModelModeProvenance)
	}
	if !strings.Contains(civ.GlobalModeReason, "inferred") {
		t.Fatalf("global mode reason must disclose inferred mode, got %q", civ.GlobalModeReason)
	}
	if civ.GlobalModeProvenance != "inferred" {
		t.Fatalf("global mode provenance = %q, want inferred", civ.GlobalModeProvenance)
	}
	implementer := byRole["implementer"]
	if implementer.ModelMode != "Manual" {
		t.Fatalf("implementer mode = %q, want Manual for policy-event override", implementer.ModelMode)
	}
	if implementer.ModelModeProvenance != "override" {
		t.Fatalf("implementer mode provenance = %q, want override", implementer.ModelModeProvenance)
	}
	planner := byRole["planner"]
	if planner.ModelMode != "Auto" || planner.ModelModeProvenance != "default" {
		t.Fatalf("unassigned bootstrap role should inherit global mode default, got mode=%q provenance=%q", planner.ModelMode, planner.ModelModeProvenance)
	}
	emergent := byRole["cartographer"]
	if emergent.Origin != "runtime-projected" || emergent.Category != "emergent/runtime" {
		t.Fatalf("non-bootstrap lifecycle role must be marked emergent/runtime-projected, got %+v", emergent)
	}
	if emergent.CanOperate != "not projected" {
		t.Fatalf("emergent CanOperate = %q, want not projected", emergent.CanOperate)
	}
	if len(civ.Emergence) == 0 {
		t.Fatal("spawn approval should appear in emergence queue")
	}
}

func TestBuildObsCivilizationLabelsExplicitGlobalModelMode(t *testing.T) {
	civ := buildObsCivilization(nil, &OpsHiveData{
		ModelSelection: OpsHiveModelSelection{
			GlobalMode: "manual",
			Source:     "hive",
		},
	})
	if civ.GlobalModelMode != "Manual" {
		t.Fatalf("global mode = %q, want Manual", civ.GlobalModelMode)
	}
	if civ.GlobalModeProvenance != "explicit" {
		t.Fatalf("global mode provenance = %q, want explicit", civ.GlobalModeProvenance)
	}
	if !strings.Contains(civ.GlobalModeReason, "explicitly projected") {
		t.Fatalf("global mode reason must disclose explicit projection, got %q", civ.GlobalModeReason)
	}
}

func TestBuildObsSpendStatesTrueReasons(t *testing.T) {
	cases := []struct {
		name       string
		v          *ObsVitals
		wantGauge  bool
		wantReason string // substring
	}{
		{"nil vitals", nil, false, "no hive snapshot"},
		{"both absent", &ObsVitals{}, false, "both absent"},
		{"cost absent", &ObsVitals{DailyCap: fptr(25)}, false, "spend absent"},
		{"cap absent", &ObsVitals{DailyCost: fptr(11.5)}, false, "cap absent"},
		{"cap zero is invalid, not unknown", &ObsVitals{DailyCost: fptr(11.5), DailyCap: fptr(0)}, false, "non-positive"},
		{"negative cost is invalid", &ObsVitals{DailyCost: fptr(-2), DailyCap: fptr(25)}, false, "negative"},
		{"valid", &ObsVitals{DailyCost: fptr(11.5), DailyCap: fptr(25)}, true, ""},
		{"over cap still draws", &ObsVitals{DailyCost: fptr(30), DailyCap: fptr(25)}, true, ""},
	}
	for _, c := range cases {
		got := buildObsSpend(c.v)
		if c.wantGauge && (got.GaugeSVG == "" || got.Reason != "") {
			t.Errorf("%s: want gauge, got reason=%q", c.name, got.Reason)
		}
		if !c.wantGauge {
			if got.GaugeSVG != "" {
				t.Errorf("%s: gauge must be withheld", c.name)
			}
			if !strings.Contains(got.Reason, c.wantReason) {
				t.Errorf("%s: reason %q must contain %q", c.name, got.Reason, c.wantReason)
			}
		}
		// A withheld gauge must never claim "unknown" for present-but-invalid values.
		if c.name == "cap zero is invalid, not unknown" && strings.Contains(got.Reason, "unknown") {
			t.Errorf("%s: invalid cap must not be called unknown: %q", c.name, got.Reason)
		}
	}
}

func TestPhaseCostBarsStatesReasons(t *testing.T) {
	if _, _, _, reason := phaseCostBars(nil); !strings.Contains(reason, "no phases") {
		t.Errorf("nil report must state a reason, got %q", reason)
	}
	neg := &OpsPipelineReport{Phases: []OpsPipelinePhase{{Phase: "scout", WorkflowStage: "intake", CostUSD: -1}}}
	svg, labels, _, reason := phaseCostBars(neg)
	if svg != "" || labels != nil || !strings.Contains(reason, "negative") {
		t.Errorf("negative cost must withhold chart with a negative-cost reason, got svg=%d labels=%v reason=%q", len(svg), labels, reason)
	}
	zero := &OpsPipelineReport{Phases: []OpsPipelinePhase{{Phase: "scout", CostUSD: 0}}}
	svg, labels, total, reason := phaseCostBars(zero)
	if svg != "" || len(labels) != 1 || total != "" || reason != "" {
		t.Errorf("all-zero costs: no chart, labels preserved, no reason (zeros are recorded facts): svg=%d labels=%d total=%q reason=%q", len(svg), len(labels), total, reason)
	}
	ok := &OpsPipelineReport{Phases: []OpsPipelinePhase{{Phase: "scout", CostUSD: 1.5}, {Phase: "build", CostUSD: 0.5}}}
	svg, labels, total, reason = phaseCostBars(ok)
	if svg == "" || len(labels) != 2 || total != "2.00" || reason != "" {
		t.Errorf("valid costs must chart: svg=%d labels=%d total=%q reason=%q", len(svg), len(labels), total, reason)
	}

	// Renderer refusal on a NON-ZERO series must state a reason, never fall
	// through to the all-zero story (round-2 finding): 400 phases cannot fit
	// 720px, so Bars fails closed while sum > 0.
	many := &OpsPipelineReport{Phases: make([]OpsPipelinePhase, 400)}
	for i := range many.Phases {
		many.Phases[i] = OpsPipelinePhase{Phase: "p", CostUSD: 0.01}
	}
	svg, labels, _, reason = phaseCostBars(many)
	if svg != "" {
		t.Fatal("expected renderer refusal for 400 phases at 720px")
	}
	if !strings.Contains(reason, "renderer declined") || len(labels) != 400 {
		t.Errorf("non-zero renderer refusal must carry an explicit reason and keep labels, got reason=%q labels=%d", reason, len(labels))
	}
}

func TestFetchObservatoryStatusPresenceAwareDecode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// hive present but with chain_ok and counts OMITTED — must decode as nil, not false/0
		w.Write([]byte(`{"agents":[{"role":"scout","actor_id":"a1"}],"hive":{"daily_cost":11.5},"timestamp":"2026-06-13T01:00:00Z"}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)
	t.Setenv("WORK_API_KEY", "")

	r := httptest.NewRequest(http.MethodGet, "http://site/ops/observatory", nil)
	payload, _, err := fetchObservatoryStatus(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v := payload.Hive
	if v == nil {
		t.Fatal("hive must decode")
	}
	if v.ChainOK != nil || v.ActiveAgents != nil || v.TotalActors != nil || v.ChainLength != nil {
		t.Error("omitted fields must decode as nil — rendering them as 0/false fabricates facts")
	}
	if v.DailyCost == nil || *v.DailyCost != 11.5 {
		t.Error("present fields must decode")
	}
	if a := payload.Agents[0]; a.CostUSD != nil || a.Errors != nil || a.Iteration != nil {
		t.Error("omitted agent scalars must decode as nil")
	}
}

func TestFetchObservatoryStatusSendsAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"agents":[],"timestamp":"2026-06-13T01:00:00Z"}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)
	t.Setenv("WORK_API_KEY", "")
	r := httptest.NewRequest(http.MethodGet, "http://site/ops/observatory", nil)
	if _, _, err := fetchObservatoryStatus(r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer dev" {
		t.Errorf("must send work auth header, got %q", gotAuth)
	}
}

func TestFetchObservatoryHistoryErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	r := httptest.NewRequest(http.MethodGet, "http://site/ops/observatory", nil)
	if _, _, err := fetchObservatoryHistory(r); err == nil {
		t.Fatal("a 500 from the feeder must surface as an error, never an empty success")
	}
}

func TestFetchObservatoryTraceEscapesPathAndVerifiesTask(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"task_id":"t-1","events":[{"id":"e1","type":"work.task.created","source":"actor-a","timestamp":"2026-06-13T00:00:01Z"},{"id":"e2","type":"work.task.completed","source":"actor-b","timestamp":"2026-06-13T00:05:00Z"}]}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	r := httptest.NewRequest(http.MethodGet, "http://site/ops/observatory", nil)

	// A hostile task ID must be rejected before any request is built.
	if _, _, err := fetchObservatoryTrace(r, "../phase-gates?x=1#f"); err == nil {
		t.Error("hostile task ID must be rejected by the allowlist")
	}
	if gotPath != "" {
		t.Errorf("rejected task ID must produce no feeder request; feeder saw %q", gotPath)
	}

	// Normal flow: order preserved.
	steps, _, err := fetchObservatoryTrace(r, "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(steps) != 2 || steps[0].Label != "work.task.created" || steps[1].Label != "work.task.completed" {
		t.Errorf("steps must preserve feeder order, got %+v", steps)
	}

	// Feeder answering for a different task must be an error, not data.
	if _, _, err := fetchObservatoryTrace(r, "t-OTHER"); err == nil {
		t.Error("task_id mismatch must surface as an error")
	}

	// A feeder that omits the task_id echo is unverifiable — withheld.
	srvNoEcho := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"events":[{"id":"e1","type":"work.task.created","source":"a","timestamp":"2026-06-13T00:00:01Z"}]}`))
	}))
	defer srvNoEcho.Close()
	t.Setenv("WORK_API_BASE_URL", srvNoEcho.URL)
	if _, _, err := fetchObservatoryTrace(r, "t-1"); err == nil || !strings.Contains(err.Error(), "did not echo") {
		t.Errorf("missing task_id echo must fail closed, got err=%v", err)
	}
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	// Oversized IDs are rejected before any request.
	if _, _, err := fetchObservatoryTrace(r, strings.Repeat("x", 200)); err == nil {
		t.Error("oversized task ID must be rejected")
	}
	// UUID-shaped IDs (the real case) pass the allowlist.
	if !obsTaskIDPattern.MatchString("0197524e-9b7c-7cc3-b8c3-1a2b3c4d5e6f") {
		t.Error("UUID task IDs must pass the allowlist")
	}
}

func TestHandleOpsObservatoryRendersReadOnlyProjection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/telemetry/status":
			w.Write([]byte(`{"agents":[],"hive":{},"timestamp":"2026-06-18T08:00:00Z"}`))
		case "/telemetry/agents/history":
			w.Write([]byte(`{"agents":[]}`))
		case "/telemetry/overview":
			w.Write([]byte(`{"actors":[],"agents":[],"phases":[],"recent_events":[],"timestamp":"2026-06-18T08:00:00Z"}`))
		case "/telemetry/pipeline/report":
			w.Write([]byte(`{"report":null}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)
	t.Setenv("WORK_API_KEY", "")
	t.Setenv("HIVE_OPS_API_BASE_URL", "")

	h, _, _ := testHandlers(t)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/observatory", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ops/observatory: status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"Observatory",
		"Civilization vitals",
		"Live event pulse",
		"Authority projection",
		"read-only projection",
		"EventGraph remains truth",
		"decisions happen on governed surfaces only",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /ops/observatory: body does not contain %q", want)
		}
	}
	surface := body
	if start := strings.Index(surface, "Civilization vitals"); start >= 0 {
		surface = surface[start:]
	}
	if end := strings.Index(surface, "</main>"); end >= 0 {
		surface = surface[:end]
	}
	for _, forbidden := range []*regexp.Regexp{
		regexp.MustCompile(`(?i)method\s*=\s*['"]?post`),
		regexp.MustCompile(`(?i)hx-(post|put|patch|delete)\s*=`),
		regexp.MustCompile(`(?i)data-authority-action\s*=`),
	} {
		if forbidden.MatchString(surface) {
			t.Fatalf("GET /ops/observatory: body contains mutation control marker %q", forbidden)
		}
	}
	for _, forbidden := range []string{"Certify release", "Approve authority", "Execute work"} {
		if strings.Contains(surface, forbidden) {
			t.Fatalf("GET /ops/observatory: body contains mutation control text %q", forbidden)
		}
	}
}

func TestHandleOpsObservatoryEventsProxiesWorkSSEWithServerSideAuth(t *testing.T) {
	var gotPath, gotQuery, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(": keepalive\n\n"))
		w.Write([]byte("event: site-error\n"))
		w.Write([]byte(`data: {"type":"agent.state.changed","source":"sysmon","summary":"state changed","recorded_at":"2026-06-13T01:00:00Z"}` + "\n\n"))
	}))
	defer srv.Close()

	t.Setenv("WORK_API_BASE_URL", srv.URL)
	t.Setenv("WORK_API_KEY", "test-key")

	h := &Handlers{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://site/ops/observatory/events", nil)
	h.handleOpsObservatoryEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if gotPath != "/telemetry/sse" {
		t.Fatalf("upstream path = %q, want /telemetry/sse", gotPath)
	}
	if gotQuery != "" {
		t.Fatalf("upstream query = %q, want empty query so API keys never enter URLs", gotQuery)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("Authorization = %q, want bearer key sent server-side", gotAuth)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	body := w.Body.String()
	for _, want := range []string{": keepalive", `"type":"agent.state.changed"`, `"source":"sysmon"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("stream body missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "event: site-error") {
		t.Fatalf("upstream event names must not be forwarded into the proxy-owned site-error channel: %s", body)
	}
}

func TestOpsObservatoryEventsRouteThroughWriteWrapStreams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`data: {"type":"mock","source":"wrapper-test"}` + "\n\n"))
	}))
	defer srv.Close()

	t.Setenv("WORK_API_BASE_URL", srv.URL)
	t.Setenv("WORK_API_KEY", "")

	var wrapped bool
	h := NewHandlers(nil, nil, func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped = true
			next.ServeHTTP(w, r)
		})
	})
	mux := http.NewServeMux()
	h.Register(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://site/ops/observatory/events", nil)
	mux.ServeHTTP(w, req)

	if !wrapped {
		t.Fatal("observatory events route must be registered through writeWrap")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	if body := w.Body.String(); !strings.Contains(body, `"source":"wrapper-test"`) {
		t.Fatalf("stream body missing forwarded data frame: %s", body)
	}
}

func TestObsForwardableSSELineDropsUpstreamEventNames(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"", true},
		{": keepalive", true},
		{`data: {"type":"agent.state.changed"}`, true},
		{"id: 123", true},
		{"retry: 3000", true},
		{"event: site-error", false},
		{"event: message", false},
		{"junk: nope", false},
	}
	for _, c := range cases {
		if got := obsForwardableSSELine(c.line); got != c.want {
			t.Errorf("obsForwardableSSELine(%q) = %v, want %v", c.line, got, c.want)
		}
	}
}

func TestHandleOpsObservatoryEventsRejectsNonSSEUpstream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"events":[]}`))
	}))
	defer srv.Close()

	t.Setenv("WORK_API_BASE_URL", srv.URL)

	h := &Handlers{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://site/ops/observatory/events", nil)
	h.handleOpsObservatoryEvents(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusBadGateway, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "did not return text/event-stream") {
		t.Fatalf("error body should name the fail-closed reason, got: %s", w.Body.String())
	}
}

func TestObsCanonicalModelModeAllowlist(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		// Existing legacy vocabulary stays recognized, byte-identical.
		{"auto", "Auto"},
		{"automatic", "Auto"},
		{"manual", "Manual"},
		{"pinned", "Manual"},
		{" Manual ", "Manual"},
		// Projected resolver vocabulary (hive selection_mode contract).
		{"manual-explicit", "Manual"},
		{"auto-tier", "Auto"},
		{"system-default", "Auto"}, // a default IS automatic selection; the chip stays binary
		{" MANUAL-EXPLICIT ", "Manual"},
		{"Auto-Tier", "Auto"},
		// Anything else non-empty stays unrecognized — never coerced to a mode.
		{"", ""},
		{"   ", ""},
		{"explicit", ""},
		{"default", ""},
		{"manual_explicit", ""},
		{"manual-explicit-v2", ""},
		{"auto-tier-2", ""},
		{"system-defaults", ""},
		{"resolver-mode-v2", ""},
		{"tier", ""},
		{"unknown", ""},
	}
	for _, c := range cases {
		if got := obsCanonicalModelMode(c.raw); got != c.want {
			t.Errorf("obsCanonicalModelMode(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

// TestObsAssignmentModelModeStateProjectedModeTable is the D4/D5 mode-state
// table: the hive-projected selection_mode is evaluated first (valid →
// explicit; present-but-invalid → fail closed with the sticky sentinel, no
// legacy fallback, no inference); an absent field leaves every legacy
// derivation byte-identical.
func TestObsAssignmentModelModeStateProjectedModeTable(t *testing.T) {
	cases := []struct {
		name           string
		selection      OpsHiveModelSelection
		item           OpsHiveModelRoleAssignment
		wantMode       string
		wantProvenance string
	}{
		// --- selection_mode ABSENT: legacy chain byte-identical (old hive → new site pairing) ---
		{"absent: override mode wins", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{OverrideMode: "manual"}, "Manual", "override"},
		{"absent: effective mode is explicit", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{EffectiveMode: "auto"}, "Auto", "explicit"},
		{"absent: policy event id is manual override", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{PolicyEventID: "ev-1"}, "Manual", "override"},
		{"absent: policy event source is manual override", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{Source: "hive-model-policy-event"}, "Manual", "override"},
		{"absent: selection strategy infers auto", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionStrategy: "balanced"}, "Auto", "inferred"},
		{"absent: preferred tier infers auto", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{PreferredTier: "execution"}, "Auto", "inferred"},
		{"absent: required capabilities infer auto", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{RequiredCapabilities: []string{"tools"}}, "Auto", "inferred"},
		{"absent: global mode is default", OpsHiveModelSelection{GlobalMode: "manual"}, OpsHiveModelRoleAssignment{}, "Manual", "default"},
		{"absent: global selection mode is default", OpsHiveModelSelection{SelectionMode: "auto"}, OpsHiveModelRoleAssignment{}, "Auto", "default"},
		{"absent: model presence infers manual", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{Model: "claude-sonnet"}, "Manual", "inferred"},
		{"absent: policy model presence infers manual", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{PolicyModel: "claude-opus"}, "Manual", "inferred"},
		{"absent: nothing projected is unknown", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{}, "unknown", "not projected"},
		// Whitespace-only is not a vocabulary claim: the whole derivation file
		// treats whitespace as empty (obsFirstNonEmpty), so it degrades to the
		// legacy chain exactly like an absent field.
		{"whitespace-only selection mode degrades to legacy inference", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "   ", Model: "claude-sonnet"}, "Manual", "inferred"},

		// --- legacy vocabulary arriving in selection_mode keeps rendering as today ---
		{"legacy auto in selection_mode stays explicit", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "auto"}, "Auto", "explicit"},
		{"legacy pinned in selection_mode stays explicit", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "pinned"}, "Manual", "explicit"},

		// --- each projected vocabulary value → explicit provenance + the right chip ---
		{"projected manual-explicit", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "manual-explicit"}, "Manual", "explicit"},
		{"projected auto-tier", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "auto-tier"}, "Auto", "explicit"},
		{"projected system-default", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "system-default"}, "Auto", "explicit"},

		// --- conflicts: the projected resolver mode always precedes legacy site-side fields ---
		{"projected mode wins over override mode", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "auto-tier", OverrideMode: "manual"}, "Auto", "explicit"},
		{"projected mode wins over effective mode", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "manual-explicit", EffectiveMode: "auto", OverrideMode: "auto"}, "Manual", "explicit"},

		// --- present-but-invalid fails closed BEFORE inference (CFADA1-2) ---
		{"invalid with model present", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "resolver-mode-v2", Model: "claude-sonnet"}, "unknown", obsModeProvenanceInvalidProjection},
		{"invalid with policy event present", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "resolver-mode-v2", PolicyEventID: "ev-1", Source: "hive-model-policy-event"}, "unknown", obsModeProvenanceInvalidProjection},
		{"invalid cannot be masked by override mode", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "resolver-mode-v2", OverrideMode: "manual"}, "unknown", obsModeProvenanceInvalidProjection},
		{"invalid ignores strategy and tier inference", OpsHiveModelSelection{}, OpsHiveModelRoleAssignment{SelectionMode: "made-up", SelectionStrategy: "balanced", PreferredTier: "judgment"}, "unknown", obsModeProvenanceInvalidProjection},
		{"invalid ignores the projected global default", OpsHiveModelSelection{GlobalMode: "auto", Source: "hive"}, OpsHiveModelRoleAssignment{SelectionMode: "bogus"}, "unknown", obsModeProvenanceInvalidProjection},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mode, provenance := obsAssignmentModelModeState(c.selection, c.item)
			if mode != c.wantMode || provenance != c.wantProvenance {
				t.Fatalf("state = (%q, %q), want (%q, %q)", mode, provenance, c.wantMode, c.wantProvenance)
			}
		})
	}
}

func TestObsModeProvenanceDisplayMapsSentinelOnly(t *testing.T) {
	if got := obsModeProvenanceDisplay(obsModeProvenanceInvalidProjection); got != "not projected" {
		t.Fatalf("sentinel must display as %q, got %q", "not projected", got)
	}
	for _, passthrough := range []string{"explicit", "override", "inferred", "default", "not projected", ""} {
		if got := obsModeProvenanceDisplay(passthrough); got != passthrough {
			t.Errorf("obsModeProvenanceDisplay(%q) = %q, want passthrough", passthrough, got)
		}
	}
}

// TestObsAssignmentModelModeRenderAccessorsMapSentinel guards the ops.templ
// render boundary: the template prints these accessors directly, so the
// internal invalid-projection sentinel must already be mapped to the
// operator-visible "not projected" here.
func TestObsAssignmentModelModeRenderAccessorsMapSentinel(t *testing.T) {
	sel := OpsHiveModelSelection{Source: "hive", GlobalMode: "auto"}
	item := OpsHiveModelRoleAssignment{Role: "guardian", Model: "claude-sonnet", SelectionMode: "resolver-mode-v2"}
	if got := obsAssignmentModelMode(sel, item); got != "unknown" {
		t.Fatalf("mode accessor = %q, want unknown", got)
	}
	if got := obsAssignmentModelModeProvenance(sel, item); got != "not projected" {
		t.Fatalf("provenance accessor = %q, want %q (render boundary maps the sentinel)", got, "not projected")
	}
}

// TestBuildObsCivilizationInvalidProjectedModeIsStickyThroughInheritance is
// the CFADA2-1 end-to-end case: an assignment with an INVALID projected
// selection_mode PLUS model/policy metadata, while a global mode IS projected,
// must still render unknown · not projected through the full roster assembly.
// Without the inheritance gate the roster loop would launder the invalid mode
// into an inherited "Auto · default".
func TestBuildObsCivilizationInvalidProjectedModeIsStickyThroughInheritance(t *testing.T) {
	hive := &OpsHiveData{
		ModelSelection: OpsHiveModelSelection{
			Source:     "hive",
			GlobalMode: "auto", // global mode IS projected → the inheritance branch is armed
			Assignments: []OpsHiveModelRoleAssignment{
				{
					Role:              "guardian",
					Model:             "claude-sonnet", // model metadata present
					PolicyModel:       "claude-opus",
					PolicyEventID:     "ev-model", // policy metadata present
					Source:            "hive-model-policy-event",
					SelectionStrategy: "balanced",
					SelectionMode:     "resolver-mode-v2", // present-but-invalid projected mode
				},
				{Role: "implementer", Model: "gpt-5.5", SelectionMode: "auto-tier"},
			},
		},
	}
	civ := buildObsCivilization(nil, hive)
	if civ.GlobalModelMode != "Auto" || civ.GlobalModeProvenance != "explicit" {
		t.Fatalf("precondition: global mode must be projected, got mode=%q provenance=%q", civ.GlobalModelMode, civ.GlobalModeProvenance)
	}
	byRole := map[string]ObsCivilizationRole{}
	for _, row := range civ.Roster {
		byRole[row.Role] = row
	}
	guardian, ok := byRole["guardian"]
	if !ok {
		t.Fatal("guardian row missing from roster")
	}
	if guardian.ModelMode != "unknown" {
		t.Fatalf("invalid projected mode must stay unknown through roster assembly, got mode=%q provenance=%q", guardian.ModelMode, guardian.ModelModeProvenance)
	}
	if guardian.ModelModeProvenance != obsModeProvenanceInvalidProjection {
		t.Fatalf("invalid projected mode must carry the sticky sentinel, got provenance=%q", guardian.ModelModeProvenance)
	}
	if got := obsModeProvenanceDisplay(guardian.ModelModeProvenance); got != "not projected" {
		t.Fatalf("sentinel must display as not projected, got %q", got)
	}
	// A VALID projected vocabulary value renders explicit end-to-end.
	implementer := byRole["implementer"]
	if implementer.ModelMode != "Auto" || implementer.ModelModeProvenance != "explicit" {
		t.Fatalf("valid projected mode must render explicit, got mode=%q provenance=%q", implementer.ModelMode, implementer.ModelModeProvenance)
	}
	// A genuinely-absent row keeps today's inherit behavior byte-identical.
	planner := byRole["planner"]
	if planner.ModelMode != "Auto" || planner.ModelModeProvenance != "default" {
		t.Fatalf("unassigned bootstrap role must still inherit the projected global mode, got mode=%q provenance=%q", planner.ModelMode, planner.ModelModeProvenance)
	}
}

// TestObsCivilizationPanelNeverPrintsInvalidProjectionSentinel guards the
// observatory.templ render boundary: the internal sentinel stays internal;
// the operator sees "not projected".
func TestObsCivilizationPanelNeverPrintsInvalidProjectionSentinel(t *testing.T) {
	hive := &OpsHiveData{
		ModelSelection: OpsHiveModelSelection{
			Source:     "hive",
			GlobalMode: "auto",
			Assignments: []OpsHiveModelRoleAssignment{
				{Role: "guardian", Model: "claude-sonnet", SelectionMode: "resolver-mode-v2"},
			},
		},
	}
	data := &OpsObservatoryData{Civilization: buildObsCivilization(nil, hive)}
	var buf bytes.Buffer
	if err := obsCivilizationPanel(data).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, obsModeProvenanceInvalidProjection) {
		t.Fatalf("internal sentinel %q leaked to the rendered surface", obsModeProvenanceInvalidProjection)
	}
	if !strings.Contains(out, "not projected") {
		t.Fatal("rendered surface must state not projected for the invalid mode")
	}
}

func TestObsCanonicalOutcomeAllowlist(t *testing.T) {
	for _, ok := range []string{"Autonomous", "Notify", "ApprovalRequired", "Forbidden", " ApprovalRequired "} {
		if !obsCanonicalOutcome(ok) {
			t.Errorf("%q must be canonical", ok)
		}
	}
	for _, bad := range []string{"", "approved", "APPROVALREQUIRED", "Denied", "approvalrequired", "Unknown"} {
		if obsCanonicalOutcome(bad) {
			t.Errorf("%q must NOT be canonical — unknown vocabulary renders as unknown", bad)
		}
	}
}
