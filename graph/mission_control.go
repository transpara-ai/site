package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	missionControlSchemaVersion = "civilization-mission-control/v1"
	missionControlPath          = "/api/hive/civilization/mission-control-projection"
	missionControlRetention     = 15 * time.Minute
	missionControlMaxBytes      = 4 * 1024 * 1024
)

type MissionEvidenceMark struct {
	State        string    `json:"state"`
	Freshness    string    `json:"freshness"`
	Basis        string    `json:"basis"`
	SourceID     string    `json:"source_id"`
	ObservedAt   time.Time `json:"observed_at"`
	GeneratedAt  time.Time `json:"generated_at"`
	EvidenceRefs []string  `json:"evidence_refs"`
	Reason       string    `json:"reason,omitempty"`
}

type MissionMarkedValue struct {
	Value any                 `json:"value"`
	Mark  MissionEvidenceMark `json:"mark"`
}

type MissionCompleteness struct {
	Complete             bool           `json:"complete"`
	Reasons              []string       `json:"reasons"`
	SourceEventGraphHead string         `json:"source_eventgraph_head"`
	StartHead            string         `json:"start_head"`
	EndHead              string         `json:"end_head"`
	DomainCounts         map[string]int `json:"domain_counts"`
	PageCounts           map[string]int `json:"page_counts"`
}

type MissionSourceEnvelope struct {
	SourceID     string              `json:"source_id"`
	Required     bool                `json:"required"`
	Completeness MissionCompleteness `json:"completeness"`
	Mark         MissionEvidenceMark `json:"mark"`
}

type MissionServiceHealth struct {
	ServiceID         string              `json:"service_id"`
	Label             string              `json:"label"`
	OperationalStatus string              `json:"operational_status"`
	Detail            string              `json:"detail"`
	Mark              MissionEvidenceMark `json:"mark"`
}

type MissionClassification struct {
	EngineProtocol              string              `json:"engine_protocol"`
	DeclaredGovernanceProtocol  string              `json:"declared_governance_protocol"`
	DeclaredPacketProfile       string              `json:"declared_packet_profile"`
	DeclaredHumanReviewTier     *int                `json:"declared_human_review_tier"`
	EffectiveGovernanceProtocol string              `json:"effective_governance_protocol"`
	EffectivePacketProfile      string              `json:"effective_packet_profile"`
	EffectiveHumanReviewTier    int                 `json:"effective_human_review_tier"`
	Mark                        MissionEvidenceMark `json:"mark"`
	EvidenceRefs                []string            `json:"evidence_refs"`
}

type MissionEvidenceItem struct {
	Kind            string              `json:"kind"`
	Stage           string              `json:"stage"`
	State           string              `json:"state"`
	Reference       string              `json:"reference"`
	BlobSHA         string              `json:"blob_sha"`
	PRHeadSHA       string              `json:"pr_head_sha"`
	ReviewedHeadSHA string              `json:"reviewed_head_sha"`
	BlockerCount    *int                `json:"blocker_count"`
	AuthorFamily    string              `json:"author_family"`
	ReviewerFamily  string              `json:"reviewer_family"`
	ProviderID      string              `json:"provider_id"`
	Mark            MissionEvidenceMark `json:"mark"`
}

type MissionEvidenceRollup struct {
	FactoryOrderRef         string                         `json:"factory_order_ref"`
	DesignBlobSHA           string                         `json:"design_blob_sha"`
	HumanDesignReviewRef    string                         `json:"human_design_review_ref"`
	PRRepository            string                         `json:"pr_repository"`
	PRNumber                int                            `json:"pr_number"`
	PRState                 string                         `json:"pr_state"`
	PRHeadSHA               string                         `json:"pr_head_sha"`
	ReviewedHeadSHA         string                         `json:"reviewed_head_sha"`
	ReadyHeadMatches        bool                           `json:"ready_head_matches"`
	PendingTier3HumanReview bool                           `json:"pending_tier_3_human_review"`
	Items                   []MissionEvidenceItem          `json:"items"`
	FieldMarks              map[string]MissionEvidenceMark `json:"field_marks"`
	Mark                    MissionEvidenceMark            `json:"mark"`
}

type MissionWIPItem struct {
	Kind                string                `json:"kind"`
	StableID            string                `json:"stable_id"`
	FactoryOrderID      string                `json:"factory_order_id"`
	FactoryOrderVersion string                `json:"factory_order_version"`
	DocumentSHA256      string                `json:"document_sha256"`
	WorkTaskID          string                `json:"work_task_id"`
	Title               string                `json:"title"`
	TargetRepository    MissionMarkedValue    `json:"target_repository"`
	Assignment          MissionMarkedValue    `json:"assignment"`
	LifecycleStatus     MissionMarkedValue    `json:"lifecycle_status"`
	EngineProtocol      MissionMarkedValue    `json:"engine_protocol"`
	TLCStage            MissionMarkedValue    `json:"tlc_stage"`
	TLCStageIndex       MissionMarkedValue    `json:"tlc_stage_index"`
	ItemStartedAt       MissionMarkedValue    `json:"item_started_at"`
	LastEffectAt        MissionMarkedValue    `json:"last_effect_at"`
	ElapsedMS           MissionMarkedValue    `json:"elapsed_ms"`
	NextHandoff         MissionMarkedValue    `json:"next_handoff"`
	Completeness        MissionMarkedValue    `json:"completeness"`
	Classification      MissionClassification `json:"classification"`
	BlockerRefs         []string              `json:"blocker_refs"`
	InterventionRefs    []string              `json:"intervention_refs"`
	EvidenceRollup      MissionEvidenceRollup `json:"evidence_rollup"`
	Mark                MissionEvidenceMark   `json:"mark"`
}

type MissionRoleAgentRow struct {
	StableID     string              `json:"stable_id"`
	Role         string              `json:"role"`
	ActorID      string              `json:"actor_id"`
	Configured   MissionMarkedValue  `json:"configured"`
	Instantiated MissionMarkedValue  `json:"instantiated"`
	EventActive  MissionMarkedValue  `json:"event_active"`
	Running      MissionMarkedValue  `json:"running"`
	Provider     MissionMarkedValue  `json:"provider"`
	Model        MissionMarkedValue  `json:"model"`
	Authority    MissionMarkedValue  `json:"authority"`
	Capacity     MissionMarkedValue  `json:"capacity"`
	Status       MissionMarkedValue  `json:"status"`
	Assignment   MissionMarkedValue  `json:"assignment"`
	Mark         MissionEvidenceMark `json:"mark"`
}

type MissionRuntimeAssignment struct {
	OrderID        string    `json:"order_id"`
	OrderVersion   string    `json:"order_version"`
	DocumentSHA256 string    `json:"document_sha256"`
	Stage          string    `json:"stage"`
	AttemptID      string    `json:"attempt_id"`
	ProviderID     string    `json:"provider_id"`
	ModelID        string    `json:"model_id"`
	AssignedAt     time.Time `json:"assigned_at"`
}

type MissionWorkerPool struct {
	ConfiguredWorkers  MissionMarkedValue         `json:"configured_workers"`
	ActiveWorkers      MissionMarkedValue         `json:"active_workers"`
	AvailableWorkers   MissionMarkedValue         `json:"available_workers"`
	QueuedOrders       MissionMarkedValue         `json:"queued_orders"`
	SchedulableOrders  MissionMarkedValue         `json:"schedulable_orders"`
	Assignments        []MissionRuntimeAssignment `json:"assignments"`
	UtilizationPercent MissionMarkedValue         `json:"utilization_percent"`
	Mark               MissionEvidenceMark        `json:"mark"`
}

type MissionHumanAction struct {
	ActionID       string              `json:"action_id"`
	Kind           string              `json:"kind"`
	Severity       string              `json:"severity"`
	OwningStage    string              `json:"owning_stage"`
	SubjectID      string              `json:"subject_id"`
	Summary        string              `json:"summary"`
	RequiredAction string              `json:"required_action"`
	SourceTime     time.Time           `json:"source_time"`
	EvidenceRefs   []string            `json:"evidence_refs"`
	Link           string              `json:"link"`
	Mark           MissionEvidenceMark `json:"mark"`
}

type MissionIntervention struct {
	InterventionID string              `json:"intervention_id"`
	OrderID        string              `json:"order_id"`
	Kind           string              `json:"kind"`
	Status         string              `json:"status"`
	Prompt         string              `json:"prompt"`
	RequestedAt    time.Time           `json:"requested_at"`
	EvidenceRefs   []string            `json:"evidence_refs"`
	Mark           MissionEvidenceMark `json:"mark"`
}

type MissionHandoff struct {
	HandoffID           string              `json:"handoff_id"`
	SubjectID           string              `json:"subject_id"`
	FromStage           string              `json:"from_stage"`
	ToStage             string              `json:"to_stage"`
	ExpectedRoles       []string            `json:"expected_roles"`
	CompletionPredicate string              `json:"completion_predicate"`
	Blocked             bool                `json:"blocked"`
	EvidenceRefs        []string            `json:"evidence_refs"`
	Mark                MissionEvidenceMark `json:"mark"`
}

type MissionControlProjection struct {
	SchemaVersion     string                  `json:"schema_version"`
	GeneratedAt       time.Time               `json:"generated_at"`
	DerivationState   MissionEvidenceMark     `json:"derivation_state"`
	OperationalStatus string                  `json:"operational_status"`
	Completeness      MissionCompleteness     `json:"completeness"`
	Sources           []MissionSourceEnvelope `json:"sources"`
	Services          []MissionServiceHealth  `json:"services"`
	WIP               []MissionWIPItem        `json:"wip"`
	Roles             []MissionRoleAgentRow   `json:"roles"`
	WorkerPool        MissionWorkerPool       `json:"worker_pool"`
	HumanActions      []MissionHumanAction    `json:"human_actions"`
	Interventions     []MissionIntervention   `json:"interventions"`
	Handoffs          []MissionHandoff        `json:"handoffs"`
	ResidualRisks     []string                `json:"residual_risks"`
	NonAuthorizations []string                `json:"non_authorizations"`
}

type MissionObservedService struct {
	ServiceID         string
	Label             string
	OperationalStatus string
	Detail            string
	Mark              MissionEvidenceMark
}

type MissionControlView struct {
	Projection      *MissionControlProjection
	HiveAcquisition MissionEvidenceMark
	WorkHealth      MissionObservedService
	SiteHealth      MissionObservedService
	OverallStatus   string
	GeneratedAt     time.Time
	SourceSkew      time.Duration
	Notices         []string
}

type missionProjectionCache struct {
	projection MissionControlProjection
	observedAt time.Time
	endpoint   string
	valid      bool
}

type missionWorkHealthCache struct {
	observedAt time.Time
	endpoint   string
	valid      bool
}

type missionControlAcquirer struct {
	mu     sync.Mutex
	hive   missionProjectionCache
	work   missionWorkHealthCache
	client *http.Client
	now    func() time.Time
}

var defaultMissionControlAcquirer = &missionControlAcquirer{client: &http.Client{Timeout: 5 * time.Second}, now: func() time.Time { return time.Now().UTC() }}

func (a *missionControlAcquirer) acquire(ctx context.Context) MissionControlView {
	now := a.now().UTC()
	view := MissionControlView{GeneratedAt: now}
	type hiveResult struct {
		projection *MissionControlProjection
		mark       MissionEvidenceMark
		err        error
	}
	type workResult struct {
		service MissionObservedService
		err     error
	}
	hiveCh, workCh := make(chan hiveResult, 1), make(chan workResult, 1)
	go func() {
		projection, mark, err := a.acquireHive(ctx, now)
		hiveCh <- hiveResult{projection: projection, mark: mark, err: err}
	}()
	go func() { service, err := a.acquireWork(ctx, now); workCh <- workResult{service: service, err: err} }()
	hive, work := <-hiveCh, <-workCh
	view.Projection, view.HiveAcquisition, view.WorkHealth = hive.projection, hive.mark, work.service
	if view.Projection != nil {
		view.SourceSkew = missionJoinedSourceSkew(*view.Projection, view.WorkHealth.Mark)
		if view.SourceSkew > 5*time.Second {
			view.Notices = append(view.Notices, "required source skew exceeds five seconds; coherent-current status is denied")
		}
	}
	view.SiteHealth = MissionObservedService{
		ServiceID: "site", Label: "Site renderer", OperationalStatus: "healthy", Detail: "Request rendered by Site; this is projected-only self-observation.",
		Mark: missionSiteMark("current", "projected_only", "site_process", now, now, nil, "Site self-health does not prove Hive, Work, or durable evidence health"),
	}
	if hive.err != nil {
		view.Notices = append(view.Notices, hive.err.Error())
	}
	if work.err != nil {
		view.Notices = append(view.Notices, work.err.Error())
	}
	view.OverallStatus = missionOverallStatus(view)
	return view
}

func (a *missionControlAcquirer) acquireHive(ctx context.Context, now time.Time) (*MissionControlProjection, MissionEvidenceMark, error) {
	base := strings.TrimSpace(os.Getenv("HIVE_OPS_API_BASE_URL"))
	endpoint := ""
	if base != "" {
		endpoint = strings.TrimRight(base, "/") + missionControlPath
	}
	projection, err := a.fetchHive(ctx, endpoint, now)
	if err == nil {
		a.mu.Lock()
		a.hive = missionProjectionCache{projection: projection, observedAt: projection.GeneratedAt, endpoint: endpoint, valid: true}
		a.mu.Unlock()
		mark := missionSiteMark("current", "exact", "site_hive_transport", projection.GeneratedAt, now, []string{endpoint}, "")
		return &projection, mark, nil
	}
	a.mu.Lock()
	cached := a.hive
	a.mu.Unlock()
	reason := "Hive Mission Control unavailable: " + err.Error()
	if cached.valid && cached.endpoint == endpoint && missionHiveFallbackEligible(cached.projection, now) {
		copy := cached.projection
		mark := missionSiteMark("stale", "exact", "site_hive_transport", cached.observedAt, now, []string{endpoint}, reason)
		return &copy, mark, errors.New(reason + "; retaining last complete Hive response")
	}
	return nil, missionSiteMark("unavailable", "unavailable", "site_hive_transport", time.Time{}, now, nil, reason), errors.New(reason)
}

func (a *missionControlAcquirer) fetchHive(ctx context.Context, endpoint string, now time.Time) (MissionControlProjection, error) {
	if endpoint == "" {
		return MissionControlProjection{}, errors.New("HIVE_OPS_API_BASE_URL is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return MissionControlProjection{}, err
	}
	if key := strings.TrimSpace(os.Getenv("HIVE_OPS_API_KEY")); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return MissionControlProjection{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return MissionControlProjection{}, fmt.Errorf("Hive returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, missionControlMaxBytes+1))
	if err != nil {
		return MissionControlProjection{}, err
	}
	if len(body) > missionControlMaxBytes {
		return MissionControlProjection{}, errors.New("Hive response exceeds limit")
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	var projection MissionControlProjection
	if err := decoder.Decode(&projection); err != nil {
		return MissionControlProjection{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return MissionControlProjection{}, errors.New("Hive response has trailing data")
	}
	if projection.SchemaVersion != missionControlSchemaVersion {
		return MissionControlProjection{}, fmt.Errorf("unsupported Mission Control schema %q", projection.SchemaVersion)
	}
	if projection.GeneratedAt.IsZero() || projection.GeneratedAt.After(now.Add(5*time.Second)) {
		return MissionControlProjection{}, errors.New("Hive generated_at is missing or in the future")
	}
	if err := missionValidateProjection(projection, now); err != nil {
		return MissionControlProjection{}, err
	}
	missionNormalizeProjection(&projection, now)
	return projection, nil
}

func (a *missionControlAcquirer) acquireWork(ctx context.Context, now time.Time) (MissionObservedService, error) {
	endpoint := strings.TrimRight(serverWorkAPIBaseURL(), "/") + "/health"
	err := a.fetchWork(ctx, endpoint)
	if err == nil {
		a.mu.Lock()
		a.work = missionWorkHealthCache{observedAt: now, endpoint: endpoint, valid: true}
		a.mu.Unlock()
		return MissionObservedService{ServiceID: "work_http", Label: "Work HTTP", OperationalStatus: "healthy", Detail: "GET /health returned the exact supported ok payload; this proves HTTP liveness only.", Mark: missionSiteMark("current", "projected_only", "site_work_health", now, now, []string{endpoint}, "HTTP liveness does not prove EventGraph completeness")}, nil
	}
	a.mu.Lock()
	cached := a.work
	a.mu.Unlock()
	reason := "Work HTTP health unavailable: " + err.Error()
	if cached.valid && cached.endpoint == endpoint && !now.Before(cached.observedAt) && now.Sub(cached.observedAt) <= missionControlRetention {
		return MissionObservedService{ServiceID: "work_http", Label: "Work HTTP", OperationalStatus: "degraded", Detail: reason + "; retaining last healthy observation", Mark: missionSiteMark("stale", "projected_only", "site_work_health", cached.observedAt, now, []string{endpoint}, reason)}, errors.New(reason)
	}
	return MissionObservedService{ServiceID: "work_http", Label: "Work HTTP", OperationalStatus: "unavailable", Detail: reason, Mark: missionSiteMark("unavailable", "unavailable", "site_work_health", time.Time{}, now, nil, reason)}, errors.New(reason)
}

func (a *missionControlAcquirer) fetchWork(ctx context.Context, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	setWorkAuth(req)
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Work returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024+1))
	if err != nil {
		return err
	}
	if len(body) > 64*1024 {
		return errors.New("Work health response exceeds limit")
	}
	var payload struct {
		Status string `json:"status"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("Work health response has trailing data")
	}
	if payload.Status != "ok" {
		return fmt.Errorf("unsupported Work health status %q", payload.Status)
	}
	return nil
}

func missionOverallStatus(view MissionControlView) string {
	if view.Projection == nil || missionMarkState(view.HiveAcquisition) == "unavailable" {
		return "unavailable"
	}
	if view.Projection.OperationalStatus == "unavailable" || view.WorkHealth.OperationalStatus == "unavailable" || view.SiteHealth.OperationalStatus == "unavailable" {
		return "unavailable"
	}
	if view.Projection.OperationalStatus != "healthy" || !view.Projection.Completeness.Complete || missionMarkState(view.HiveAcquisition) != "current" || view.WorkHealth.OperationalStatus != "healthy" || missionMarkState(view.WorkHealth.Mark) != "projected_only" || view.SiteHealth.OperationalStatus != "healthy" || missionMarkState(view.SiteHealth.Mark) != "projected_only" || view.SourceSkew > 5*time.Second {
		return "degraded"
	}
	for _, service := range view.Projection.Services {
		if missionServiceStatus(service.OperationalStatus) == "unavailable" {
			return "unavailable"
		}
		if missionServiceStatus(service.OperationalStatus) != "healthy" {
			return "degraded"
		}
	}
	return "healthy"
}

var missionRequiredSourceIDs = map[string]bool{
	"eventgraph_wip_evidence": true,
	"roster_routing":          true,
	"authority_actions":       true,
	"factory_runtime":         true,
}

func missionValidateProjection(projection MissionControlProjection, now time.Time) error {
	if missionServiceStatus(projection.OperationalStatus) != projection.OperationalStatus {
		return fmt.Errorf("unknown aggregate operational status %q", projection.OperationalStatus)
	}
	seen := map[string]bool{}
	for _, source := range projection.Sources {
		if !missionRequiredSourceIDs[source.SourceID] {
			return fmt.Errorf("unknown Mission Control source identity %q", source.SourceID)
		}
		if seen[source.SourceID] {
			return fmt.Errorf("duplicate Mission Control source identity %q", source.SourceID)
		}
		seen[source.SourceID] = true
		if !source.Required {
			return fmt.Errorf("required Mission Control source %q is not marked required", source.SourceID)
		}
		if err := missionValidateRawMark(source.Mark, now); err != nil {
			return fmt.Errorf("source %s: %w", source.SourceID, err)
		}
	}
	for sourceID := range missionRequiredSourceIDs {
		if !seen[sourceID] {
			return fmt.Errorf("required Mission Control source %q is missing", sourceID)
		}
	}
	marks := []MissionEvidenceMark{projection.DerivationState, projection.WorkerPool.Mark,
		projection.WorkerPool.ConfiguredWorkers.Mark, projection.WorkerPool.ActiveWorkers.Mark,
		projection.WorkerPool.AvailableWorkers.Mark, projection.WorkerPool.QueuedOrders.Mark,
		projection.WorkerPool.SchedulableOrders.Mark, projection.WorkerPool.UtilizationPercent.Mark}
	for _, service := range projection.Services {
		if missionServiceStatus(service.OperationalStatus) != service.OperationalStatus {
			return fmt.Errorf("service %s has unknown operational status %q", service.ServiceID, service.OperationalStatus)
		}
		marks = append(marks, service.Mark)
	}
	for _, row := range projection.WIP {
		if err := missionValidateWIPSemantics(row); err != nil {
			return err
		}
		marks = append(marks, row.Mark, row.Classification.Mark, row.EvidenceRollup.Mark,
			row.TargetRepository.Mark, row.Assignment.Mark, row.LifecycleStatus.Mark, row.EngineProtocol.Mark,
			row.TLCStage.Mark, row.TLCStageIndex.Mark, row.ItemStartedAt.Mark, row.LastEffectAt.Mark,
			row.ElapsedMS.Mark, row.NextHandoff.Mark, row.Completeness.Mark)
		for _, evidence := range row.EvidenceRollup.Items {
			marks = append(marks, evidence.Mark)
		}
		for field, mark := range row.EvidenceRollup.FieldMarks {
			if !missionEvidenceFieldNameAllowed(field) {
				return fmt.Errorf("unknown evidence rollup field mark %q", field)
			}
			marks = append(marks, mark)
		}
	}
	for _, row := range projection.Roles {
		marks = append(marks, row.Mark, row.Configured.Mark, row.Instantiated.Mark, row.EventActive.Mark, row.Running.Mark, row.Provider.Mark, row.Model.Mark, row.Authority.Mark, row.Capacity.Mark, row.Status.Mark, row.Assignment.Mark)
	}
	for _, action := range projection.HumanActions {
		marks = append(marks, action.Mark)
	}
	for _, intervention := range projection.Interventions {
		marks = append(marks, intervention.Mark)
	}
	for _, handoff := range projection.Handoffs {
		marks = append(marks, handoff.Mark)
	}
	for _, mark := range marks {
		if err := missionValidateRawMark(mark, now); err != nil {
			return err
		}
	}
	return nil
}

var missionTLCStageIndexes = map[string]int{
	"ingest_work": 0, "craft_factory_order": 1, "design": 2, "iada": 3, "cfada": 4,
	"human_design_review": 5, "write_code": 6, "create_draft_pr": 7, "iar": 8,
	"cfar": 9, "mark_pr_ready": 10, "human_review": 11,
}

var missionKnownWorkStatuses = map[string]bool{
	"created": true, "ready": true, "running": true, "blocked": true, "failed": true,
	"repair_required": true, "repair_running": true, "repaired": true, "verification_running": true,
	"verified": true, "certified": true, "rejected": true, "superseded": true, "policy_blocked": true,
	"assigned": true, "completed": true, "pending": true,
}

func missionValidateWIPSemantics(row MissionWIPItem) error {
	if row.Kind != "factory_order" && row.Kind != "independent_work_task" {
		return fmt.Errorf("unknown WIP kind %q", row.Kind)
	}
	profileRank := map[string]int{"P-MECHANICAL": 0, "P-IMPLEMENTATION": 1, "P-DESIGN-DELTA": 2, "P-ENVELOPE": 3}
	if row.Classification.EffectiveGovernanceProtocol != "4.5.0" {
		return fmt.Errorf("WIP %s has unknown effective governance protocol %q", row.StableID, row.Classification.EffectiveGovernanceProtocol)
	}
	if _, ok := profileRank[row.Classification.EffectivePacketProfile]; !ok {
		return fmt.Errorf("WIP %s has unknown effective packet profile %q", row.StableID, row.Classification.EffectivePacketProfile)
	}
	if row.Classification.EffectiveHumanReviewTier < 0 || row.Classification.EffectiveHumanReviewTier > 3 {
		return fmt.Errorf("WIP %s has invalid effective Human Review tier %d", row.StableID, row.Classification.EffectiveHumanReviewTier)
	}
	if row.Classification.DeclaredPacketProfile != "" {
		if _, ok := profileRank[row.Classification.DeclaredPacketProfile]; !ok {
			return fmt.Errorf("WIP %s has unknown declared packet profile %q", row.StableID, row.Classification.DeclaredPacketProfile)
		}
	}
	if row.Classification.DeclaredHumanReviewTier != nil && (*row.Classification.DeclaredHumanReviewTier < 0 || *row.Classification.DeclaredHumanReviewTier > 3) {
		return fmt.Errorf("WIP %s has invalid declared Human Review tier %d", row.StableID, *row.Classification.DeclaredHumanReviewTier)
	}
	if missionMarkState(row.TLCStage.Mark) != "unavailable" {
		stage, ok := row.TLCStage.Value.(string)
		index, known := missionTLCStageIndexes[stage]
		if !ok || !known {
			return fmt.Errorf("WIP %s has unknown TLC stage %v", row.StableID, row.TLCStage.Value)
		}
		if missionMarkState(row.TLCStageIndex.Mark) != "unavailable" {
			stageIndex, ok := missionIntegralValue(row.TLCStageIndex.Value)
			if !ok || stageIndex != index {
				return fmt.Errorf("WIP %s has mismatched TLC stage index %v for %q", row.StableID, row.TLCStageIndex.Value, stage)
			}
		}
	}
	if missionMarkState(row.LifecycleStatus.Mark) != "unavailable" {
		status, ok := row.LifecycleStatus.Value.(string)
		if !ok || !missionLifecycleStatusAllowed(row.Kind, status) {
			return fmt.Errorf("WIP %s has unknown lifecycle status %v", row.StableID, row.LifecycleStatus.Value)
		}
	}
	return nil
}

func missionLifecycleStatusAllowed(kind, status string) bool {
	if kind == "independent_work_task" {
		return missionKnownWorkStatuses[status]
	}
	factoryStatus, workStatus, joined := strings.Cut(status, " / work:")
	switch factoryStatus {
	case "accepted", "progressing", "blocked", "human_required", "human_review":
	default:
		return false
	}
	return !joined || missionKnownWorkStatuses[workStatus] || workStatus == "unavailable"
}

func missionIntegralValue(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case float64:
		if typed == float64(int(typed)) {
			return int(typed), true
		}
	}
	return 0, false
}

func missionValidateRawMark(mark MissionEvidenceMark, now time.Time) error {
	if missionMarkState(mark) == "unavailable" && !(mark.State == "unavailable" && mark.Freshness == "unavailable" && mark.Basis == "unavailable") {
		return fmt.Errorf("unknown or contradictory epistemic value state=%q freshness=%q basis=%q", mark.State, mark.Freshness, mark.Basis)
	}
	if mark.GeneratedAt.IsZero() || mark.GeneratedAt.After(now.Add(5*time.Second)) {
		return errors.New("evidence mark generated_at is missing or in the future")
	}
	if mark.Freshness != "unavailable" && (mark.ObservedAt.IsZero() || mark.ObservedAt.After(now.Add(5*time.Second))) {
		return errors.New("evidence mark observed_at is missing or in the future")
	}
	return nil
}

func missionHiveFallbackEligible(projection MissionControlProjection, now time.Time) bool {
	if projection.GeneratedAt.IsZero() || now.Before(projection.GeneratedAt) || now.Sub(projection.GeneratedAt) > missionControlRetention {
		return false
	}
	seen := map[string]bool{}
	for _, source := range projection.Sources {
		if !source.Required || !missionRequiredSourceIDs[source.SourceID] {
			continue
		}
		seen[source.SourceID] = true
		for _, observed := range []time.Time{source.Mark.ObservedAt, source.Mark.GeneratedAt} {
			if observed.IsZero() || now.Before(observed) || now.Sub(observed) > missionControlRetention {
				return false
			}
		}
	}
	return len(seen) == len(missionRequiredSourceIDs)
}

func missionRequiredSourceSkew(projection MissionControlProjection) time.Duration {
	return missionJoinedSourceSkew(projection, MissionEvidenceMark{})
}

func missionJoinedSourceSkew(projection MissionControlProjection, additional ...MissionEvidenceMark) time.Duration {
	var earliest, latest time.Time
	for _, source := range projection.Sources {
		if !source.Required || !missionRequiredSourceIDs[source.SourceID] || source.Mark.ObservedAt.IsZero() {
			continue
		}
		if earliest.IsZero() || source.Mark.ObservedAt.Before(earliest) {
			earliest = source.Mark.ObservedAt
		}
		if latest.IsZero() || source.Mark.ObservedAt.After(latest) {
			latest = source.Mark.ObservedAt
		}
	}
	for _, mark := range additional {
		if mark.ObservedAt.IsZero() {
			continue
		}
		if earliest.IsZero() || mark.ObservedAt.Before(earliest) {
			earliest = mark.ObservedAt
		}
		if latest.IsZero() || mark.ObservedAt.After(latest) {
			latest = mark.ObservedAt
		}
	}
	if earliest.IsZero() || latest.IsZero() {
		return 0
	}
	return latest.Sub(earliest)
}

func missionHiveTransportService(mark MissionEvidenceMark) MissionObservedService {
	status := "healthy"
	if missionMarkState(mark) == "unavailable" {
		status = "unavailable"
	} else if missionMarkState(mark) == "stale" {
		status = "degraded"
	}
	detail := mark.Reason
	if detail == "" {
		detail = "Authenticated Hive Mission Control response decoded with the supported schema."
	}
	return MissionObservedService{ServiceID: "hive_transport", Label: "Hive projection transport", OperationalStatus: status, Detail: detail, Mark: mark}
}

func missionNormalizeProjection(projection *MissionControlProjection, now time.Time) {
	if projection.Sources == nil {
		projection.Sources = []MissionSourceEnvelope{}
	}
	if projection.Services == nil {
		projection.Services = []MissionServiceHealth{}
	}
	if projection.WIP == nil {
		projection.WIP = []MissionWIPItem{}
	}
	if projection.Roles == nil {
		projection.Roles = []MissionRoleAgentRow{}
	}
	if projection.WorkerPool.Assignments == nil {
		projection.WorkerPool.Assignments = []MissionRuntimeAssignment{}
	}
	if projection.HumanActions == nil {
		projection.HumanActions = []MissionHumanAction{}
	}
	if projection.Interventions == nil {
		projection.Interventions = []MissionIntervention{}
	}
	if projection.Handoffs == nil {
		projection.Handoffs = []MissionHandoff{}
	}
	if projection.ResidualRisks == nil {
		projection.ResidualRisks = []string{}
	}
	if projection.NonAuthorizations == nil {
		projection.NonAuthorizations = []string{}
	}
	projection.DerivationState = missionNormalizeMark(projection.DerivationState, now)
	for i := range projection.Sources {
		projection.Sources[i].Mark = missionNormalizeMark(projection.Sources[i].Mark, now)
	}
	for i := range projection.Services {
		projection.Services[i].Mark = missionNormalizeMark(projection.Services[i].Mark, now)
		projection.Services[i].OperationalStatus = missionServiceStatus(projection.Services[i].OperationalStatus)
	}
	for i := range projection.WIP {
		missionNormalizeWIP(&projection.WIP[i], now)
	}
	for i := range projection.Roles {
		missionNormalizeRole(&projection.Roles[i], now)
	}
	for i := range projection.HumanActions {
		projection.HumanActions[i].Mark = missionNormalizeMark(projection.HumanActions[i].Mark, now)
	}
	for i := range projection.Interventions {
		projection.Interventions[i].Mark = missionNormalizeMark(projection.Interventions[i].Mark, now)
	}
	for i := range projection.Handoffs {
		projection.Handoffs[i].Mark = missionNormalizeMark(projection.Handoffs[i].Mark, now)
	}
	projection.WorkerPool.Mark = missionNormalizeMark(projection.WorkerPool.Mark, now)
	for _, value := range []*MissionMarkedValue{&projection.WorkerPool.ConfiguredWorkers, &projection.WorkerPool.ActiveWorkers, &projection.WorkerPool.AvailableWorkers, &projection.WorkerPool.QueuedOrders, &projection.WorkerPool.SchedulableOrders, &projection.WorkerPool.UtilizationPercent} {
		value.Mark = missionNormalizeMark(value.Mark, now)
	}
	sort.Slice(projection.WIP, func(i, j int) bool { return projection.WIP[i].StableID < projection.WIP[j].StableID })
	sort.Slice(projection.Roles, func(i, j int) bool { return projection.Roles[i].StableID < projection.Roles[j].StableID })
}

func missionNormalizeWIP(row *MissionWIPItem, now time.Time) {
	for _, value := range []*MissionMarkedValue{&row.TargetRepository, &row.Assignment, &row.LifecycleStatus, &row.EngineProtocol, &row.TLCStage, &row.TLCStageIndex, &row.ItemStartedAt, &row.LastEffectAt, &row.ElapsedMS, &row.NextHandoff, &row.Completeness} {
		value.Mark = missionNormalizeMark(value.Mark, now)
	}
	row.Mark = missionNormalizeMark(row.Mark, now)
	row.Classification.Mark = missionNormalizeMark(row.Classification.Mark, now)
	row.EvidenceRollup.Mark = missionNormalizeMark(row.EvidenceRollup.Mark, now)
	if row.EvidenceRollup.FieldMarks == nil {
		row.EvidenceRollup.FieldMarks = map[string]MissionEvidenceMark{}
	}
	for _, field := range missionEvidenceFieldNames {
		mark, exists := row.EvidenceRollup.FieldMarks[field]
		if !exists {
			mark = missionSiteMark("unavailable", "unavailable", "hive_legacy_projection", time.Time{}, now, nil, field+": field-level evidence mark unavailable in legacy payload")
		}
		row.EvidenceRollup.FieldMarks[field] = missionNormalizeMark(mark, now)
	}
	if row.BlockerRefs == nil {
		row.BlockerRefs = []string{}
	}
	if row.InterventionRefs == nil {
		row.InterventionRefs = []string{}
	}
	if row.EvidenceRollup.Items == nil {
		row.EvidenceRollup.Items = []MissionEvidenceItem{}
	}
	for i := range row.EvidenceRollup.Items {
		row.EvidenceRollup.Items[i].Mark = missionNormalizeMark(row.EvidenceRollup.Items[i].Mark, now)
	}
}

var missionEvidenceFieldNames = []string{
	"factory_order_ref", "design_blob_sha", "human_design_review_ref", "pr_repository", "pr_number",
	"pr_state", "pr_head_sha", "reviewed_head_sha", "ready_head_matches", "pending_tier_3_human_review",
}

func missionEvidenceFieldNameAllowed(candidate string) bool {
	for _, field := range missionEvidenceFieldNames {
		if candidate == field {
			return true
		}
	}
	return false
}

func missionNormalizeRole(row *MissionRoleAgentRow, now time.Time) {
	for _, value := range []*MissionMarkedValue{&row.Configured, &row.Instantiated, &row.EventActive, &row.Running, &row.Provider, &row.Model, &row.Authority, &row.Capacity, &row.Status, &row.Assignment} {
		value.Mark = missionNormalizeMark(value.Mark, now)
	}
	row.Mark = missionNormalizeMark(row.Mark, now)
}

func missionNormalizeMark(mark MissionEvidenceMark, now time.Time) MissionEvidenceMark {
	state := missionMarkState(mark)
	if state != "unavailable" {
		return mark
	}
	if mark.State == "unavailable" && mark.Freshness == "unavailable" && mark.Basis == "unavailable" {
		return mark
	}
	return missionSiteMark("unavailable", "unavailable", mark.SourceID, mark.ObservedAt, now, mark.EvidenceRefs, "unknown or contradictory evidence mark")
}

func missionMarkState(mark MissionEvidenceMark) string {
	if mark.Freshness == "unavailable" && mark.Basis == "unavailable" && mark.State == "unavailable" {
		return "unavailable"
	}
	if mark.Freshness == "stale" && (mark.Basis == "exact" || mark.Basis == "inferred" || mark.Basis == "projected_only") && mark.State == "stale" {
		return "stale"
	}
	if mark.Freshness != "current" {
		return "unavailable"
	}
	switch mark.Basis {
	case "exact":
		if mark.State == "current" {
			return "current"
		}
	case "inferred":
		if mark.State == "inferred" {
			return "inferred"
		}
	case "projected_only":
		if mark.State == "projected_only" {
			return "projected_only"
		}
	}
	return "unavailable"
}

func missionServiceStatus(status string) string {
	switch status {
	case "healthy", "degraded", "unavailable":
		return status
	default:
		return "unavailable"
	}
}

func missionSiteMark(freshness, basis, source string, observed, generated time.Time, refs []string, reason string) MissionEvidenceMark {
	state := "unavailable"
	if freshness == "stale" {
		state = "stale"
	} else if freshness == "current" {
		switch basis {
		case "exact":
			state = "current"
		case "inferred":
			state = "inferred"
		case "projected_only":
			state = "projected_only"
		}
	}
	return MissionEvidenceMark{State: state, Freshness: freshness, Basis: basis, SourceID: source, ObservedAt: observed.UTC(), GeneratedAt: generated.UTC(), EvidenceRefs: append([]string(nil), refs...), Reason: reason}
}

func missionValue(value MissionMarkedValue) string {
	if missionMarkState(value.Mark) == "unavailable" || value.Value == nil {
		return "unavailable"
	}
	switch typed := value.Value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return "unavailable"
		}
		return typed
	case float64:
		return fmt.Sprintf("%g", typed)
	case bool:
		if typed {
			return "yes"
		}
		return "no"
	}
	encoded, err := json.Marshal(value.Value)
	if err != nil {
		return "unavailable"
	}
	return string(encoded)
}

func missionTime(value time.Time) string {
	if value.IsZero() {
		return "unavailable"
	}
	return value.UTC().Format(time.RFC3339)
}

func missionEvidenceAge(mark MissionEvidenceMark) string {
	if mark.ObservedAt.IsZero() || mark.GeneratedAt.IsZero() || mark.GeneratedAt.Before(mark.ObservedAt) {
		return "age unavailable"
	}
	return mark.GeneratedAt.Sub(mark.ObservedAt).Round(time.Second).String() + " old"
}

func missionRefs(refs []string) string {
	if len(refs) == 0 {
		return "none"
	}
	return strings.Join(refs, " · ")
}

func (h *Handlers) handleMissionControl(w http.ResponseWriter, r *http.Request) {
	view := defaultMissionControlAcquirer.acquire(r.Context())
	h.renderConsole(w, r, ConsolePageData{Title: "Civilization Mission Control", Active: "mission-control", MissionControl: &view})
}

func (h *Handlers) handleMissionControlFragment(w http.ResponseWriter, r *http.Request) {
	view := defaultMissionControlAcquirer.acquire(r.Context())
	missionControlFragment(view).Render(r.Context(), w)
}
