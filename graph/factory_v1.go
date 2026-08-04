package graph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	factoryV1SchemaVersion       = "factory-v1"
	factoryV1MaxTitleBytes       = 300
	factoryV1MaxRepositoryBytes  = 300
	factoryV1MaxIdeaBytes        = 12_000
	factoryV1MaxInstructionBytes = 12_000
	factoryV1MaxOrderBytes       = 100_000
	factoryV1MaxResolutionBytes  = 12_000
)

var factoryV1IDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$`)

// FactoryV1Projection is the Site-owned transport view of Hive's canonical
// factory-v1 projection. Optional fields deliberately retain their zero values:
// the view-model builder turns absent canonical evidence into an explicit
// blocked state instead of manufacturing healthy-looking defaults.
type FactoryV1Projection struct {
	SchemaVersion string                  `json:"schema_version"`
	GeneratedAt   string                  `json:"generated_at"`
	Service       FactoryV1Service        `json:"service"`
	Orders        []FactoryV1Order        `json:"orders"`
	Ideas         []FactoryV1Idea         `json:"ideas"`
	Interventions []FactoryV1Intervention `json:"interventions"`
}

type FactoryV1Service struct {
	ServiceID          string `json:"service_id"`
	InstanceID         string `json:"instance_id"`
	StartedAt          string `json:"started_at"`
	RecoveryEpoch      int64  `json:"recovery_epoch"`
	RecoveryGeneration int64  `json:"recovery_generation"`
	Status             string `json:"status"`
	Healthy            bool   `json:"healthy"`
	Detail             string `json:"detail"`
}

type FactoryV1Order struct {
	OrderID              string              `json:"order_id"`
	Version              string              `json:"version"`
	Title                string              `json:"title"`
	Channel              string              `json:"channel"`
	SourceRef            json.RawMessage     `json:"source_ref"`
	DocumentSHA256       string              `json:"document_sha256"`
	Status               string              `json:"status"`
	TLCStage             string              `json:"tlc_stage"`
	TLCIndex             int                 `json:"tlc_index"`
	ElapsedMS            int64               `json:"elapsed_ms"`
	ActiveAttemptID      string              `json:"active_attempt_id"`
	ActivelyExecuting    bool                `json:"actively_executing"`
	Peers                []string            `json:"peers"`
	GateState            string              `json:"gate_state"`
	Evidence             []FactoryV1Evidence `json:"evidence"`
	Blocker              string              `json:"blocker"`
	NextAction           string              `json:"next_action"`
	Budget               FactoryV1Budget     `json:"budget"`
	PR                   FactoryV1PR         `json:"pr"`
	HumanApprovalBasis   string              `json:"human_approval_basis"`
	HumanApprovalReceipt json.RawMessage     `json:"human_approval_receipt"`
	Stages               []FactoryV1Stage    `json:"stages"`
}

type FactoryV1Evidence struct {
	Kind            string            `json:"kind"`
	Ref             string            `json:"ref"`
	Reference       string            `json:"reference"`
	SHA256          string            `json:"sha256"`
	EventID         string            `json:"event_id"`
	DesignBlobSHA   string            `json:"design_blob_sha"`
	PRHeadSHA       string            `json:"pr_head_sha"`
	ReviewedHeadSHA string            `json:"reviewed_head_sha"`
	AuthorFamily    string            `json:"author_family"`
	ReviewerFamily  string            `json:"reviewer_family"`
	BlockerCount    *int              `json:"blocker_count"`
	Metadata        map[string]string `json:"metadata"`
}

type FactoryV1Budget struct {
	AttemptLimit        int   `json:"attempt_limit"`
	AttemptsUsed        int   `json:"attempts_used"`
	Remaining           int   `json:"remaining"`
	MaxAttempts         int   `json:"max_attempts"`
	ConsumedAttempts    int   `json:"consumed_attempts"`
	RemainingAttempts   int   `json:"remaining_attempts"`
	MaxTokens           int64 `json:"max_tokens"`
	ConsumedTokens      int64 `json:"consumed_tokens"`
	RemainingTokens     int64 `json:"remaining_tokens"`
	MaxCostMicros       int64 `json:"max_cost_micros"`
	ConsumedCostMicros  int64 `json:"consumed_cost_micros"`
	RemainingCostMicros int64 `json:"remaining_cost_micros"`
	Exhausted           bool  `json:"exhausted"`
}

type FactoryV1PR struct {
	Repository      string `json:"repository"`
	Number          int    `json:"number"`
	URL             string `json:"url"`
	HeadSHA         string `json:"head_sha"`
	ReviewedHeadSHA string `json:"reviewed_head_sha"`
	ChecksState     string `json:"checks_state"`
	Open            bool   `json:"open"`
	Draft           bool   `json:"draft"`
	ChecksPassing   bool   `json:"checks_passing"`
}

type FactoryV1Stage struct {
	Stage          string              `json:"stage"`
	Index          int                 `json:"index"`
	Status         string              `json:"status"`
	State          string              `json:"state"`
	AttemptID      string              `json:"attempt_id"`
	Ordinal        int                 `json:"ordinal"`
	EventID        string              `json:"event_id"`
	StartedAt      string              `json:"started_at"`
	CompletedAt    string              `json:"completed_at"`
	OccurredAt     string              `json:"occurred_at"`
	Peers          []string            `json:"peers"`
	Evidence       []FactoryV1Evidence `json:"evidence"`
	WorkArtifactID string              `json:"work_artifact_id"`
	Recovered      bool                `json:"recovered"`
}

type FactoryV1Idea struct {
	IdeaID           string                  `json:"idea_id"`
	Title            string                  `json:"title"`
	TargetRepository string                  `json:"target_repository"`
	Status           string                  `json:"status"`
	CurrentRevision  int                     `json:"current_revision"`
	Revisions        []FactoryV1IdeaRevision `json:"revisions"`
}

type FactoryV1IdeaRevision struct {
	Revision         int             `json:"revision"`
	Instruction      string          `json:"instruction"`
	Note             string          `json:"note"`
	Candidate        json.RawMessage `json:"candidate"`
	ValidationErrors []string        `json:"validation_errors"`
	CreatedAt        string          `json:"created_at"`
	RecordedAt       string          `json:"recorded_at"`
	EventID          string          `json:"event_id"`
}

type FactoryV1Intervention struct {
	InterventionID string `json:"intervention_id"`
	OrderID        string `json:"order_id"`
	Kind           string `json:"kind"`
	Prompt         string `json:"prompt"`
	Status         string `json:"status"`
	RequestedAt    string `json:"requested_at"`
	EventID        string `json:"event_id"`
}

type FactoryV1OrderView struct {
	Order           FactoryV1Order
	EffectiveStatus string
	Missing         []string
}

// FactoryV1MissionControl is the single, live Human surface. Writable is true
// only for a current projection from a running/healthy Hive service. All other
// states remain fail-closed even if an old response still contains orders.
type FactoryV1MissionControl struct {
	Freshness     ConsoleFreshness
	GeneratedAt   string
	Service       FactoryV1Service
	Orders        []FactoryV1OrderView
	Ideas         []FactoryV1Idea
	Interventions []FactoryV1Intervention
	Writable      bool
	Notices       []string
}

func buildFactoryV1MissionControl(proj *FactoryV1Projection, fetchErr error, now time.Time) FactoryV1MissionControl {
	if fetchErr != nil || proj == nil {
		reason := "factory-v1 projection unavailable"
		if fetchErr != nil {
			reason = fetchErr.Error()
		}
		return FactoryV1MissionControl{Freshness: FreshnessUnavailable, Notices: []string{reason}}
	}
	if proj.SchemaVersion != factoryV1SchemaVersion {
		return FactoryV1MissionControl{
			Freshness: FreshnessUnavailable,
			Notices:   []string{fmt.Sprintf("unsupported factory projection schema %q", proj.SchemaVersion)},
		}
	}

	freshness := deriveFreshness(proj.GeneratedAt, nil, false, now, consoleStaleWindow)
	view := FactoryV1MissionControl{
		Freshness:     freshness,
		GeneratedAt:   proj.GeneratedAt,
		Service:       proj.Service,
		Ideas:         append([]FactoryV1Idea(nil), proj.Ideas...),
		Interventions: append([]FactoryV1Intervention(nil), proj.Interventions...),
	}
	if freshness == FreshnessUnavailable {
		view.Notices = append(view.Notices, "projection generated_at is missing, invalid, or in the future")
	}
	if (strings.TrimSpace(proj.Service.InstanceID) == "" && strings.TrimSpace(proj.Service.ServiceID) == "") || strings.TrimSpace(proj.Service.StartedAt) == "" {
		view.Notices = append(view.Notices, "service identity or start receipt is not projected")
		if view.Freshness == FreshnessCurrent {
			view.Freshness = FreshnessPartial
		}
	}
	serviceStatus := factoryV1ServiceStatus(proj.Service)
	if serviceStatus != "running" && serviceStatus != "healthy" {
		view.Notices = append(view.Notices, "factory service is not confirmed running")
		if view.Freshness == FreshnessCurrent {
			view.Freshness = FreshnessPartial
		}
	}

	for _, order := range proj.Orders {
		missing := factoryV1OrderMissingEvidence(order)
		status := strings.ToLower(strings.TrimSpace(order.Status))
		switch status {
		case "accepted", "queued":
			status = "progressing"
		case "progressing", "blocked", "human_required", "human_review":
		default:
			missing = append(missing, "recognized status")
		}
		if len(missing) > 0 {
			status = "blocked"
		}
		view.Orders = append(view.Orders, FactoryV1OrderView{Order: order, EffectiveStatus: status, Missing: missing})
	}
	view.Writable = view.Freshness == FreshnessCurrent && (serviceStatus == "running" || serviceStatus == "healthy")
	return view
}

func factoryV1OrderMissingEvidence(order FactoryV1Order) []string {
	missing := make([]string, 0, 12)
	require := func(ok bool, label string) {
		if !ok {
			missing = append(missing, label)
		}
	}
	require(strings.TrimSpace(order.OrderID) != "", "order_id")
	require(strings.TrimSpace(order.Version) != "", "version")
	require(strings.TrimSpace(order.Title) != "", "title")
	require(strings.TrimSpace(order.Channel) != "", "channel")
	require(factoryV1SourceRefValid(order.SourceRef), "source_ref")
	require(len(strings.TrimSpace(order.DocumentSHA256)) == 64, "document_sha256")
	require(strings.TrimSpace(order.TLCStage) != "" && order.TLCIndex > 0, "TLC stage/index")
	require(strings.TrimSpace(order.GateState) != "", "gate_state")
	require(strings.TrimSpace(order.NextAction) != "", "next_action")
	limit, consumed, remaining := factoryV1BudgetAttempts(order.Budget)
	require(limit > 0 && consumed >= 0 && remaining >= 0, "budget")
	require(factoryV1OrderCanonicalEvidencePresent(order), "canonical evidence")
	require(len(order.Stages) > 0, "stage ledger")
	if order.TLCIndex >= 6 {
		basis := strings.TrimSpace(order.HumanApprovalBasis)
		require(basis == "standing_scoped" || basis == "fresh_scoped", "human_approval_basis")
		require(factoryV1RawPresent(order.HumanApprovalReceipt), "human_approval_receipt")
	}
	if order.TLCIndex >= 8 {
		require(strings.TrimSpace(order.PR.Repository) != "" && order.PR.Number > 0 && strings.TrimSpace(order.PR.HeadSHA) != "", "PR identity/head")
	}
	return missing
}

func factoryV1OrderCanonicalEvidencePresent(order FactoryV1Order) bool {
	if len(order.Evidence) > 0 {
		return true
	}
	// Hive deliberately emits a running transition before the external stage
	// effect exists. That durable ledger event plus its exact attempt identity
	// is the canonical progress evidence; demanding terminal stage evidence here
	// would falsely render all actively executing work as blocked.
	if order.ActivelyExecuting && strings.TrimSpace(order.ActiveAttemptID) != "" {
		for _, stage := range order.Stages {
			if stage.AttemptID == order.ActiveAttemptID && factoryV1StageState(stage) == "running" && strings.TrimSpace(stage.EventID) != "" {
				return true
			}
		}
	}
	return false
}

func factoryV1RawPresent(raw json.RawMessage) bool {
	v := strings.TrimSpace(string(raw))
	return v != "" && v != "null" && v != `""` && v != "{}" && v != "[]"
}

func factoryV1SourceRefValid(raw json.RawMessage) bool {
	if !factoryV1RawPresent(raw) {
		return false
	}
	var legacy string
	if err := json.Unmarshal(raw, &legacy); err == nil {
		return strings.TrimSpace(legacy) != ""
	}
	var source struct {
		Identity string `json:"identity"`
		SHA256   string `json:"sha256"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return false
	}
	return strings.TrimSpace(source.Identity) != "" && len(strings.TrimSpace(source.SHA256)) == 64
}

func fetchFactoryV1Projection(r *http.Request) (*FactoryV1Projection, error) {
	base := strings.TrimSpace(os.Getenv("HIVE_OPS_API_BASE_URL"))
	if base == "" {
		return nil, errors.New("HIVE_OPS_API_BASE_URL is not configured")
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, strings.TrimRight(base, "/")+"/api/hive/factory/v1/projection", nil)
	if err != nil {
		return nil, err
	}
	if key := strings.TrimSpace(os.Getenv("HIVE_OPS_API_KEY")); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := hiveOpsProjectionClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("factory-v1 projection returned %s", resp.Status)
	}
	var projection FactoryV1Projection
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 4<<20))
	if err := decoder.Decode(&projection); err != nil {
		return nil, fmt.Errorf("decode factory-v1 projection: %w", err)
	}
	return &projection, nil
}

func (h *Handlers) handleFactoryV1MissionControl(w http.ResponseWriter, r *http.Request) {
	projection, err := fetchFactoryV1Projection(r)
	view := buildFactoryV1MissionControl(projection, err, time.Now().UTC())
	h.renderConsole(w, r, ConsolePageData{Title: "Factory v1", Active: "factory-v1", FactoryV1: &view})
}

func (h *Handlers) handleFactoryV1MissionControlFragment(w http.ResponseWriter, r *http.Request) {
	projection, err := fetchFactoryV1Projection(r)
	view := buildFactoryV1MissionControl(projection, err, time.Now().UTC())
	factoryV1MissionControlSurface(view).Render(r.Context(), w)
}

func (h *Handlers) handleFactoryV1IdeaCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	title, err := requiredFactoryV1FormValue(r, "title", factoryV1MaxTitleBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	idea, err := requiredFactoryV1FormValue(r, "idea", factoryV1MaxIdeaBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	repo, err := requiredFactoryV1FormValue(r, "target_repository", factoryV1MaxRepositoryBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.factoryV1Mutation(w, r, "/api/hive/factory/v1/ideas", map[string]any{"title": title, "idea": idea, "target_repository": repo})
}

func (h *Handlers) handleFactoryV1IdeaRefine(w http.ResponseWriter, r *http.Request) {
	id, ok := validFactoryV1ID(r.PathValue("id"))
	if !ok {
		http.Error(w, "invalid idea id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	instruction, err := requiredFactoryV1FormValue(r, "instruction", factoryV1MaxInstructionBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.factoryV1Mutation(w, r, "/api/hive/factory/v1/ideas/"+id+"/refine", map[string]any{"instruction": instruction})
}

func (h *Handlers) handleFactoryV1IdeaSubmit(w http.ResponseWriter, r *http.Request) {
	id, ok := validFactoryV1ID(r.PathValue("id"))
	if !ok {
		http.Error(w, "invalid idea id", http.StatusBadRequest)
		return
	}
	h.factoryV1Mutation(w, r, "/api/hive/factory/v1/ideas/"+id+"/submit", map[string]any{"approved": true})
}

func (h *Handlers) handleFactoryV1OrderCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	raw, err := requiredFactoryV1FormValue(r, "factory_order", factoryV1MaxOrderBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var order map[string]any
	if err := json.Unmarshal([]byte(raw), &order); err != nil || order == nil {
		http.Error(w, "factory_order must be one JSON object", http.StatusBadRequest)
		return
	}
	h.factoryV1Mutation(w, r, "/api/hive/factory/v1/orders", map[string]any{"factory_order": order})
}

func (h *Handlers) handleFactoryV1InterventionResolve(w http.ResponseWriter, r *http.Request) {
	id, ok := validFactoryV1ID(r.PathValue("id"))
	if !ok {
		http.Error(w, "invalid intervention id", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	resolution, err := requiredFactoryV1FormValue(r, "resolution", factoryV1MaxResolutionBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	actorID, ok := h.factoryV1ActorID(r)
	if !ok {
		http.Error(w, "factory operator identity is not configured", http.StatusUnauthorized)
		return
	}
	h.factoryV1Mutation(w, r, "/api/hive/factory/v1/interventions/"+id+"/resolve", map[string]any{"resolution": resolution, "actor_id": actorID})
}

// factoryV1ActorID binds intervention receipts to a configured EventGraph
// Human actor in the local acceptance stack. An authenticated Site identity is
// the fallback for normal application use; the anonymous sentinel is never
// forwarded as Human authority.
func (h *Handlers) factoryV1ActorID(r *http.Request) (string, bool) {
	if configured := strings.TrimSpace(os.Getenv("HIVE_FACTORY_V1_ACTOR_ID")); configured != "" {
		return validFactoryV1ID(configured)
	}
	actorID := h.userID(r)
	if actorID == anonUserID {
		return "", false
	}
	return validFactoryV1ID(actorID)
}

func (h *Handlers) factoryV1Mutation(w http.ResponseWriter, r *http.Request, endpoint string, payload any) {
	if err := postFactoryV1JSON(r, endpoint, payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/console/factory-v1", http.StatusSeeOther)
}

func postFactoryV1JSON(r *http.Request, endpoint string, payload any) error {
	base := strings.TrimSpace(os.Getenv("HIVE_OPS_API_BASE_URL"))
	key := strings.TrimSpace(os.Getenv("HIVE_OPS_API_KEY"))
	if base == "" {
		return errors.New("HIVE_OPS_API_BASE_URL is not configured")
	}
	if key == "" {
		return errors.New("HIVE_OPS_API_KEY is not configured; factory writes fail closed")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(base, "/")+endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := hiveOpsProjectionClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if message := strings.TrimSpace(string(body)); message != "" {
			return fmt.Errorf("factory command returned %s: %s", resp.Status, message)
		}
		return fmt.Errorf("factory command returned %s", resp.Status)
	}
	return nil
}

func requiredFactoryV1FormValue(r *http.Request, key string, maxBytes int) (string, error) {
	value := strings.TrimSpace(r.FormValue(key))
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	if len(value) > maxBytes {
		return "", fmt.Errorf("%s exceeds %s bytes", key, strconv.Itoa(maxBytes))
	}
	return value, nil
}

func validFactoryV1ID(value string) (string, bool) {
	value = strings.TrimSpace(value)
	return value, factoryV1IDPattern.MatchString(value)
}

func factoryV1Elapsed(ms int64) string {
	if ms <= 0 {
		return "not projected"
	}
	return (time.Duration(ms) * time.Millisecond).Round(time.Second).String()
}

func factoryV1Join(values []string, fallback string) string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			trimmed = append(trimmed, value)
		}
	}
	if len(trimmed) == 0 {
		return fallback
	}
	return strings.Join(trimmed, ", ")
}

func factoryV1RawSummary(raw json.RawMessage, fallback string) string {
	if !factoryV1RawPresent(raw) {
		return fallback
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err == nil {
		return compact.String()
	}
	return strings.TrimSpace(string(raw))
}

func factoryV1InterventionOpen(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "open", "requested", "human_required":
		return true
	default:
		return false
	}
}

func factoryV1Value(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}

func factoryV1OpenInterventionCount(items []FactoryV1Intervention) int {
	count := 0
	for _, item := range items {
		if factoryV1InterventionOpen(item.Status) {
			count++
		}
	}
	return count
}

func factoryV1IdeaApprovable(idea FactoryV1Idea) bool {
	if idea.CurrentRevision <= 0 || len(idea.Revisions) == 0 {
		return false
	}
	for _, revision := range idea.Revisions {
		if revision.Revision == idea.CurrentRevision {
			return factoryV1RawPresent(revision.Candidate) && len(revision.ValidationErrors) == 0
		}
	}
	return false
}

func factoryV1PRLabel(pr FactoryV1PR) string {
	if strings.TrimSpace(pr.Repository) == "" || pr.Number <= 0 {
		return "not projected"
	}
	return fmt.Sprintf("%s#%d", pr.Repository, pr.Number)
}

func factoryV1ServiceStatus(service FactoryV1Service) string {
	if status := strings.ToLower(strings.TrimSpace(service.Status)); status != "" {
		return status
	}
	if service.Healthy {
		return "healthy"
	}
	return "unhealthy"
}

func factoryV1ServiceIdentity(service FactoryV1Service) string {
	if id := strings.TrimSpace(service.InstanceID); id != "" {
		return id
	}
	return factoryV1Value(service.ServiceID, "not projected")
}

func factoryV1RecoveryGeneration(service FactoryV1Service) int64 {
	if service.RecoveryGeneration != 0 {
		return service.RecoveryGeneration
	}
	return service.RecoveryEpoch
}

func factoryV1BudgetAttempts(budget FactoryV1Budget) (limit, consumed, remaining int) {
	if budget.MaxAttempts > 0 {
		return budget.MaxAttempts, budget.ConsumedAttempts, budget.RemainingAttempts
	}
	return budget.AttemptLimit, budget.AttemptsUsed, budget.Remaining
}

func factoryV1BudgetLabel(budget FactoryV1Budget) string {
	limit, consumed, remaining := factoryV1BudgetAttempts(budget)
	label := fmt.Sprintf("%d / %d attempts · %d left", consumed, limit, remaining)
	if budget.MaxTokens > 0 {
		label += fmt.Sprintf(" · %d / %d tokens", budget.ConsumedTokens, budget.MaxTokens)
	}
	if budget.MaxCostMicros > 0 {
		label += fmt.Sprintf(" · $%.2f / $%.2f", float64(budget.ConsumedCostMicros)/1_000_000, float64(budget.MaxCostMicros)/1_000_000)
	}
	if budget.Exhausted {
		label += " · exhausted"
	}
	return label
}

func factoryV1PRChecks(pr FactoryV1PR) string {
	if checks := strings.TrimSpace(pr.ChecksState); checks != "" {
		return checks
	}
	if pr.ChecksPassing {
		return "passing"
	}
	return "not passing or not projected"
}

func factoryV1PRState(pr FactoryV1PR) string {
	if pr.Open {
		if pr.Draft {
			return "open draft"
		}
		return "open ready"
	}
	if pr.Number > 0 {
		return "not open"
	}
	return "not projected"
}

func factoryV1StageState(stage FactoryV1Stage) string {
	if state := strings.TrimSpace(stage.State); state != "" {
		return state
	}
	return factoryV1Value(stage.Status, "status missing")
}

func factoryV1StageTime(stage FactoryV1Stage) string {
	if stage.StartedAt != "" || stage.CompletedAt != "" {
		return factoryV1Value(stage.StartedAt, "not started") + " → " + factoryV1Value(stage.CompletedAt, "not complete")
	}
	return factoryV1Value(stage.OccurredAt, "time not projected")
}

func factoryV1IdeaRevisionInstruction(revision FactoryV1IdeaRevision) string {
	if instruction := strings.TrimSpace(revision.Instruction); instruction != "" {
		return instruction
	}
	return factoryV1Value(revision.Note, "initial idea")
}

func factoryV1IdeaRevisionTime(revision FactoryV1IdeaRevision) string {
	if createdAt := strings.TrimSpace(revision.CreatedAt); createdAt != "" {
		return createdAt
	}
	return factoryV1Value(revision.RecordedAt, "time missing")
}

func factoryV1EvidenceReference(evidence FactoryV1Evidence) string {
	if reference := strings.TrimSpace(evidence.Reference); reference != "" {
		return reference
	}
	return factoryV1Value(evidence.Ref, "reference missing")
}

func factoryV1EvidenceExactness(evidence FactoryV1Evidence) string {
	parts := make([]string, 0, 6)
	for _, item := range []struct{ label, value string }{
		{"design", evidence.DesignBlobSHA},
		{"PR head", evidence.PRHeadSHA},
		{"reviewed head", evidence.ReviewedHeadSHA},
		{"author", evidence.AuthorFamily},
		{"reviewer", evidence.ReviewerFamily},
	} {
		if value := strings.TrimSpace(item.value); value != "" {
			parts = append(parts, item.label+" "+value)
		}
	}
	if evidence.BlockerCount != nil {
		parts = append(parts, fmt.Sprintf("blockers %d", *evidence.BlockerCount))
	}
	return factoryV1Join(parts, "no additional exactness fields")
}

func factoryV1OrderPlaceholder() string {
	return `{"doc_id":"FO-...","version":"1.0.0","status":"approved","title":"...","channel":"completed_factory_order","target_repository":"transpara-ai/repository","requirements":[],"acceptance_criteria":[],"test_plan":[],"constraints":[],"expected_outputs":[],"authority_scope":{},"budget":{}}`
}
