package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type missionSiteTestClock struct {
	mu    sync.Mutex
	value time.Time
}

func (c *missionSiteTestClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.value }
func (c *missionSiteTestClock) Add(delta time.Duration) {
	c.mu.Lock()
	c.value = c.value.Add(delta)
	c.mu.Unlock()
}

func missionTestMark(now time.Time, basis string) MissionEvidenceMark {
	state := map[string]string{"exact": "current", "inferred": "inferred", "projected_only": "projected_only"}[basis]
	return MissionEvidenceMark{State: state, Freshness: "current", Basis: basis, SourceID: "test", ObservedAt: now.Add(-time.Second), GeneratedAt: now, EvidenceRefs: []string{"evidence:test"}}
}

func missionTestUnavailable(now time.Time) MissionEvidenceMark {
	return MissionEvidenceMark{State: "unavailable", Freshness: "unavailable", Basis: "unavailable", SourceID: "test", GeneratedAt: now, Reason: "not observed"}
}

func missionTestProjection(now time.Time) MissionControlProjection {
	exact, projected := missionTestMark(now, "exact"), missionTestMark(now, "projected_only")
	unavailable := missionTestUnavailable(now)
	evidenceFieldMarks := map[string]MissionEvidenceMark{}
	for _, field := range missionEvidenceFieldNames {
		evidenceFieldMarks[field] = exact
	}
	value := func(v any, mark MissionEvidenceMark) MissionMarkedValue {
		return MissionMarkedValue{Value: v, Mark: mark}
	}
	return MissionControlProjection{
		SchemaVersion: missionControlSchemaVersion, GeneratedAt: now, DerivationState: projected, OperationalStatus: "healthy",
		Completeness: MissionCompleteness{Complete: true, SourceEventGraphHead: "head-1", StartHead: "head-1", EndHead: "head-1", DomainCounts: map[string]int{"work.task.created": 2}, PageCounts: map[string]int{"work.task.created": 2}},
		Sources: []MissionSourceEnvelope{
			{SourceID: "eventgraph_wip_evidence", Required: true, Completeness: MissionCompleteness{Complete: true, StartHead: "head-1", EndHead: "head-1"}, Mark: exact},
			{SourceID: "roster_routing", Required: true, Completeness: MissionCompleteness{Complete: true, StartHead: "head-1", EndHead: "head-1"}, Mark: exact},
			{SourceID: "authority_actions", Required: true, Completeness: MissionCompleteness{Complete: true, StartHead: "head-1", EndHead: "head-1"}, Mark: exact},
			{SourceID: "factory_runtime", Required: true, Completeness: MissionCompleteness{Complete: true, StartHead: "boot-1", EndHead: "boot-1"}, Mark: projected},
		},
		Services: []MissionServiceHealth{
			{ServiceID: "civilization", Label: "Civilization", OperationalStatus: "healthy", Detail: "complete", Mark: projected},
			{ServiceID: "eventgraph", Label: "EventGraph evidence", OperationalStatus: "healthy", Detail: "complete", Mark: exact},
			{ServiceID: "work_projection", Label: "Work projection", OperationalStatus: "healthy", Detail: "complete", Mark: exact},
			{ServiceID: "hive_ops_api", Label: "Hive ops API", OperationalStatus: "healthy", Detail: "current", Mark: projected},
			{ServiceID: "factory_runtime", Label: "Factory worker runtime", OperationalStatus: "healthy", Detail: "polling", Mark: projected},
		},
		WIP: []MissionWIPItem{{
			Kind: "factory_order", StableID: "factory:FO-MC-1@1.0.0", FactoryOrderID: "FO-MC-1", FactoryOrderVersion: "1.0.0", DocumentSHA256: strings.Repeat("d", 64), WorkTaskID: "task-1", Title: "Mission Control",
			TargetRepository: value("transpara-ai/site", exact), Assignment: value("attempt-1", exact), LifecycleStatus: value("human_review", exact), EngineProtocol: value("tlc-v1", exact), TLCStage: value("human_review", exact), TLCStageIndex: value(11, exact),
			ItemStartedAt: value(now.Add(-time.Hour), exact), LastEffectAt: value(now.Add(-time.Minute), exact), ElapsedMS: value(3600000, exact), NextHandoff: value("Tier 3 Human Review", exact), Completeness: value(true, exact),
			Classification: MissionClassification{EngineProtocol: "tlc-v1", EffectiveGovernanceProtocol: "4.5.0", EffectivePacketProfile: "P-ENVELOPE", EffectiveHumanReviewTier: 3, Mark: missionTestMark(now, "inferred")},
			BlockerRefs:    []string{"blocker:test"}, InterventionRefs: []string{"intervention:test"}, EvidenceRollup: MissionEvidenceRollup{FactoryOrderRef: "0586cdc8", DesignBlobSHA: "abb89405", HumanDesignReviewRef: "6d409d", PRRepository: "transpara-ai/site", PRNumber: 42, PRState: "ready", PRHeadSHA: strings.Repeat("a", 40), ReviewedHeadSHA: strings.Repeat("a", 40), ReadyHeadMatches: true, PendingTier3HumanReview: true, Items: []MissionEvidenceItem{{Kind: "cfar", Stage: "cfar", State: "passed", Reference: "cfar:evidence", PRHeadSHA: strings.Repeat("a", 40), ReviewedHeadSHA: strings.Repeat("a", 40), AuthorFamily: "OpenAI/Codex", ReviewerFamily: "Anthropic/Claude", ProviderID: "claude", Mark: exact}}, FieldMarks: evidenceFieldMarks, Mark: exact}, Mark: exact,
		}, {Kind: "independent_work_task", StableID: "work:task-2", WorkTaskID: "task-2", Title: "Independent", TargetRepository: value(nil, unavailable), Assignment: value(nil, unavailable), LifecycleStatus: value("created", exact), EngineProtocol: value("work-v3.9", missionTestMark(now, "inferred")), TLCStage: value(nil, unavailable), TLCStageIndex: value(nil, unavailable), ItemStartedAt: value(now.Add(-time.Hour), exact), LastEffectAt: value(now.Add(-time.Minute), exact), ElapsedMS: value(3600000, exact), NextHandoff: value(nil, unavailable), Completeness: value(true, exact), Classification: MissionClassification{EngineProtocol: "work-v3.9", EffectiveGovernanceProtocol: "4.5.0", EffectivePacketProfile: "P-ENVELOPE", EffectiveHumanReviewTier: 3, Mark: missionTestMark(now, "inferred")}, EvidenceRollup: MissionEvidenceRollup{Mark: unavailable}, Mark: exact}},
		Roles:         []MissionRoleAgentRow{{StableID: "role:guardian", Role: "guardian", Configured: value(true, projected), Instantiated: value(1, exact), EventActive: value(1, projected), Running: value(nil, unavailable), Provider: value("anthropic", projected), Model: value("claude-opus", projected), Authority: value(map[string]any{"can_operate": false}, exact), Capacity: value(32768, projected), Status: value("configured", projected), Assignment: value(nil, unavailable), Mark: projected}},
		WorkerPool:    MissionWorkerPool{ConfiguredWorkers: value(3, projected), ActiveWorkers: value(1, projected), AvailableWorkers: value(2, projected), QueuedOrders: value(2, projected), SchedulableOrders: value(1, projected), UtilizationPercent: value(33.3, projected), Assignments: []MissionRuntimeAssignment{{OrderID: "FO-MC-1", OrderVersion: "1.0.0", Stage: "human_review", AttemptID: "attempt-1", ProviderID: "codex", ModelID: "gpt-5.6-sol", AssignedAt: now.Add(-time.Minute)}}, Mark: projected},
		HumanActions:  []MissionHumanAction{{ActionID: "human-review:FO-MC-1", Kind: "human_review", Severity: "high", OwningStage: "human_review", SubjectID: "FO-MC-1", Summary: "Merge-ready PR waits for Human review.", RequiredAction: "Approve, reject, or request changes.", SourceTime: now.Add(-time.Minute), EvidenceRefs: []string{"head:" + strings.Repeat("a", 40)}, Mark: exact}},
		Interventions: []MissionIntervention{{InterventionID: "intervention:test", OrderID: "FO-MC-1", Kind: "review", Status: "open", Prompt: "Human review required", RequestedAt: now.Add(-time.Minute), Mark: exact}},
		Handoffs:      []MissionHandoff{{HandoffID: "handoff:FO-MC-1", SubjectID: "FO-MC-1", FromStage: "mark_pr_ready", ToStage: "human_review", ExpectedRoles: []string{"human"}, CompletionPredicate: "exact Human review receipt", EvidenceRefs: []string{"head:test"}, Mark: exact}},
		ResidualRisks: []string{"runtime can become stale"}, NonAuthorizations: []string{"No merge or deployment authority."},
	}
}

func withMissionTestAcquirer(t *testing.T, acquirer *missionControlAcquirer) {
	t.Helper()
	prior := defaultMissionControlAcquirer
	defaultMissionControlAcquirer = acquirer
	t.Cleanup(func() { defaultMissionControlAcquirer = prior })
}

func TestSITEMCT1OneScreenContractFullPageAndFragment(t *testing.T) {
	now := time.Date(2026, 8, 8, 13, 0, 0, 0, time.UTC)
	projection := missionTestProjection(now)
	hiveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != missionControlPath {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(projection)
	}))
	defer hiveServer.Close()
	workServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer workServer.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveServer.URL)
	t.Setenv("HIVE_OPS_API_KEY", "secret")
	t.Setenv("WORK_API_BASE_URL", workServer.URL)
	clock := &missionSiteTestClock{value: now}
	withMissionTestAcquirer(t, &missionControlAcquirer{client: hiveServer.Client(), now: clock.Now})
	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)
	for _, path := range []string{"/console", "/console/mission-control/fragment"} {
		req := httptest.NewRequest(http.MethodGet, "http://site.test"+path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
		body := rec.Body.String()
		for _, want := range []string{`data-mission-landmark="aggregate"`, `data-mission-landmark="health"`, `data-mission-landmark="capacity"`, `data-mission-landmark="human-actions"`, `data-mission-landmark="wip"`, `data-mission-landmark="roles-agents"`, `data-mission-landmark="workflow"`, `data-mission-landmark="exact-evidence"`, `data-mission-landmark="sources"`, `data-mission-landmark="epistemic-legend"`, `data-mission-landmark="non-authorization"`, `hx-trigger="every 5s"`, "P-ENVELOPE", "Tier 3", "tlc-v1", "FO-MC-1", "transpara-ai/site", "claude-opus", "exact-head", "No merge or deployment authority"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s missing %q", path, want)
			}
		}
		if path == "/console/mission-control/fragment" && strings.Contains(strings.ToLower(body), "<!doctype") {
			t.Fatal("fragment contains full page shell")
		}
	}
}

func TestSITEMCT2HealthTruthTable(t *testing.T) {
	now := time.Now().UTC()
	projection := missionTestProjection(now)
	exact := missionTestMark(now, "exact")
	projected := missionTestMark(now, "projected_only")
	base := MissionControlView{Projection: &projection, HiveAcquisition: exact, WorkHealth: MissionObservedService{OperationalStatus: "healthy", Mark: projected}, SiteHealth: MissionObservedService{OperationalStatus: "healthy", Mark: projected}}
	if got := missionOverallStatus(base); got != "healthy" {
		t.Fatalf("all-current status=%q", got)
	}
	stale := base
	stale.HiveAcquisition = missionSiteMark("stale", "exact", "test", now.Add(-time.Minute), now, nil, "stale")
	if got := missionOverallStatus(stale); got != "degraded" {
		t.Fatalf("stale status=%q", got)
	}
	incomplete := base
	copyProjection := projection
	copyProjection.Completeness.Complete = false
	incomplete.Projection = &copyProjection
	if got := missionOverallStatus(incomplete); got != "degraded" {
		t.Fatalf("incomplete status=%q", got)
	}
	unknown := base
	unknownProjection := projection
	unknownProjection.Services[0].OperationalStatus = "future"
	unknown.Projection = &unknownProjection
	if got := missionOverallStatus(unknown); got != "unavailable" {
		t.Fatalf("unknown service status=%q", got)
	}
	unavailable := base
	unavailable.Projection = nil
	unavailable.HiveAcquisition = missionTestUnavailable(now)
	if got := missionOverallStatus(unavailable); got != "unavailable" {
		t.Fatalf("unavailable status=%q", got)
	}
	for i := range projection.Services {
		serviceFailure := base
		failedProjection := projection
		failedProjection.Services = append([]MissionServiceHealth(nil), projection.Services...)
		failedProjection.Services[i].OperationalStatus = "unavailable"
		serviceFailure.Projection = &failedProjection
		if got := missionOverallStatus(serviceFailure); got != "unavailable" {
			t.Errorf("service %s unavailable aggregate=%q", failedProjection.Services[i].ServiceID, got)
		}
	}
	siteUnavailable := base
	siteUnavailable.SiteHealth.OperationalStatus = "unavailable"
	if got := missionOverallStatus(siteUnavailable); got != "unavailable" {
		t.Fatalf("Site unavailable aggregate=%q", got)
	}
}

func TestSITEMCT3T4MaterialAndUnknownStatesRenderUnavailable(t *testing.T) {
	now := time.Now().UTC()
	projection := missionTestProjection(now)
	projection.Roles[0].Running.Mark = MissionEvidenceMark{State: "future", Freshness: "current", Basis: "quantum", SourceID: "future"}
	projection.WIP[0].Assignment.Mark = missionSiteMark("stale", "exact", "test", now.Add(-time.Minute), now, []string{"assignment:old"}, "retained assignment")
	missionNormalizeProjection(&projection, now)
	if got := missionMarkState(projection.Roles[0].Running.Mark); got != "unavailable" {
		t.Fatalf("unknown mark normalized to %q", got)
	}
	view := MissionControlView{Projection: &projection, HiveAcquisition: missionTestMark(now, "exact"), WorkHealth: MissionObservedService{Label: "Work", OperationalStatus: "healthy", Mark: missionTestMark(now, "projected_only")}, SiteHealth: MissionObservedService{Label: "Site", OperationalStatus: "healthy", Mark: missionTestMark(now, "projected_only")}, OverallStatus: "healthy", GeneratedAt: now}
	var body bytes.Buffer
	if err := missionControlFragment(view).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"current", "stale", "inferred", "projected_only", "unavailable"} {
		if !strings.Contains(body.String(), `data-evidence-state="`+state+`"`) {
			t.Errorf("missing state %s", state)
		}
	}
	if strings.Contains(body.String(), "quantum") || !strings.Contains(body.String(), "unavailable") {
		t.Fatal("future evidence vocabulary did not fail closed")
	}
}

func TestSITEMCT4UnknownSemanticValuesAndIncompleteSourcesFailClosed(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name   string
		mutate func(*MissionControlProjection)
	}{
		{name: "unknown profile", mutate: func(p *MissionControlProjection) { p.WIP[0].Classification.EffectivePacketProfile = "P-FUTURE" }},
		{name: "invalid tier", mutate: func(p *MissionControlProjection) { p.WIP[0].Classification.EffectiveHumanReviewTier = 9 }},
		{name: "unknown stage", mutate: func(p *MissionControlProjection) { p.WIP[0].TLCStage.Value = "future_stage" }},
		{name: "mismatched stage index", mutate: func(p *MissionControlProjection) { p.WIP[0].TLCStageIndex.Value = 4 }},
		{name: "unknown lifecycle status", mutate: func(p *MissionControlProjection) { p.WIP[0].LifecycleStatus.Value = "teleported" }},
		{name: "unknown WIP kind", mutate: func(p *MissionControlProjection) { p.WIP[0].Kind = "future_work" }},
		{name: "truncated required sources", mutate: func(p *MissionControlProjection) { p.Sources = p.Sources[:3] }},
		{name: "unknown evidence field", mutate: func(p *MissionControlProjection) {
			p.WIP[0].EvidenceRollup.FieldMarks["future_field"] = missionTestMark(now, "exact")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projection := missionTestProjection(now)
			tc.mutate(&projection)
			if err := missionValidateProjection(projection, now); err == nil {
				t.Fatalf("invalid projection accepted: %+v", projection.WIP[0])
			}
		})
	}

	view := MissionControlView{
		Projection: nil, HiveAcquisition: missionTestUnavailable(now),
		WorkHealth:    MissionObservedService{OperationalStatus: "healthy", Mark: missionTestMark(now, "projected_only")},
		SiteHealth:    MissionObservedService{OperationalStatus: "healthy", Mark: missionTestMark(now, "projected_only")},
		OverallStatus: "unavailable", GeneratedAt: now, Notices: []string{"Hive semantic validation failed closed"},
	}
	var body bytes.Buffer
	if err := missionControlFragment(view).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	rendered := strings.ToLower(body.String())
	for _, forbidden := range []string{"p-future", "future_stage", "teleported", "no wip rows were projected", "hx-post=", "<button", "<form", "method=\"post\"", "→"} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("unavailable view contains forbidden normal/write surface %q", forbidden)
		}
	}
	for _, required := range []string{"unavailable", "wip — incomplete source set", "no normal handoff is inferred"} {
		if !strings.Contains(rendered, required) {
			t.Errorf("unavailable view missing %q", required)
		}
	}
}

func TestSITEMCT5AtomicStaleRetentionExpiryAndRecovery(t *testing.T) {
	clock := &missionSiteTestClock{value: time.Date(2026, 8, 8, 14, 0, 0, 0, time.UTC)}
	var mu sync.Mutex
	hiveFail, workFail := false, false
	hiveServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		fail := hiveFail
		mu.Unlock()
		if fail {
			http.Error(w, "fail", http.StatusServiceUnavailable)
			return
		}
		projection := missionTestProjection(clock.Now())
		_ = json.NewEncoder(w).Encode(projection)
	}))
	defer hiveServer.Close()
	workServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		fail := workFail
		mu.Unlock()
		if fail {
			http.Error(w, "fail", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer workServer.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveServer.URL)
	t.Setenv("WORK_API_BASE_URL", workServer.URL)
	client := &http.Client{Timeout: time.Second}
	acquirer := &missionControlAcquirer{client: client, now: clock.Now}
	current := acquirer.acquire(context.Background())
	observed := current.HiveAcquisition.ObservedAt
	if current.Projection == nil || current.OverallStatus != "healthy" {
		t.Fatalf("current=%+v", current)
	}
	mu.Lock()
	workFail = true
	mu.Unlock()
	clock.Add(time.Minute)
	stale := acquirer.acquire(context.Background())
	if stale.Projection == nil || missionMarkState(stale.HiveAcquisition) != "current" || missionMarkState(stale.WorkHealth.Mark) != "stale" || stale.HiveAcquisition.ObservedAt == observed || stale.SourceSkew <= 5*time.Second || stale.OverallStatus != "degraded" {
		t.Fatalf("stale=%+v", stale)
	}
	if age := missionEvidenceAge(stale.WorkHealth.Mark); age != "1m0s old" {
		t.Fatalf("visible stale age = %q", age)
	}
	workObserved := stale.WorkHealth.Mark.ObservedAt
	clock.Add(14 * time.Minute)
	stillStale := acquirer.acquire(context.Background())
	if missionMarkState(stillStale.WorkHealth.Mark) != "stale" || stillStale.WorkHealth.Mark.ObservedAt != workObserved {
		t.Fatalf("15-minute boundary did not retain original Work receipt: %+v", stillStale.WorkHealth)
	}
	clock.Add(time.Second)
	expired := acquirer.acquire(context.Background())
	if expired.Projection == nil || missionMarkState(expired.HiveAcquisition) != "current" || missionMarkState(expired.WorkHealth.Mark) != "unavailable" || expired.OverallStatus != "unavailable" {
		t.Fatalf("expired=%+v", expired)
	}
	mu.Lock()
	workFail = false
	mu.Unlock()
	recovered := acquirer.acquire(context.Background())
	if recovered.Projection == nil || missionMarkState(recovered.HiveAcquisition) != "current" || recovered.OverallStatus != "healthy" {
		t.Fatalf("recovered=%+v", recovered)
	}

	mu.Lock()
	hiveFail = true
	mu.Unlock()
	clock.Add(time.Minute)
	hiveStale := acquirer.acquire(context.Background())
	if hiveStale.Projection == nil || missionMarkState(hiveStale.HiveAcquisition) != "stale" || hiveStale.HiveAcquisition.ObservedAt != recovered.Projection.GeneratedAt {
		t.Fatalf("whole-Hive transport fallback = %+v", hiveStale)
	}
}

func TestSITEMCT6AdditiveLegacyDecodeAndFutureSchema(t *testing.T) {
	now := time.Now().UTC()
	clock := &missionSiteTestClock{value: now}
	var legacy MissionControlProjection
	if err := json.Unmarshal([]byte(`{"schema_version":"civilization-mission-control/v1","generated_at":"`+now.Format(time.RFC3339Nano)+`","derivation_state":{"state":"current","freshness":"current","basis":"exact","source_id":"legacy","observed_at":"`+now.Format(time.RFC3339Nano)+`","generated_at":"`+now.Format(time.RFC3339Nano)+`","evidence_refs":[]},"operational_status":"degraded","completeness":{"complete":false}}`), &legacy); err != nil {
		t.Fatalf("additive legacy decode: %v", err)
	}
	missionNormalizeProjection(&legacy, now)
	if legacy.WIP == nil || legacy.Roles == nil || legacy.Sources == nil || legacy.WorkerPool.Assignments == nil {
		t.Fatalf("legacy omitted slices were not normalized: %+v", legacy)
	}
	futureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"schema_version":"future","generated_at":"` + now.Format(time.RFC3339Nano) + `"}`))
	}))
	defer futureServer.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", futureServer.URL)
	acquirer := &missionControlAcquirer{client: futureServer.Client(), now: clock.Now}
	projection, mark, err := acquirer.acquireHive(context.Background(), now)
	if err == nil || projection != nil || missionMarkState(mark) != "unavailable" {
		t.Fatalf("future schema accepted: %v %+v", err, projection)
	}
}
