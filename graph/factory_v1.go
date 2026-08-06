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

var (
	factoryV1IDPattern      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$`)
	factoryV1SHA256Pattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	factoryV1GitHashPattern = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
)

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
	DesignBlobSHA   string            `json:"design_blob_sha"`
	PRHeadSHA       string            `json:"pr_head_sha"`
	ReviewedHeadSHA string            `json:"reviewed_head_sha"`
	AuthorFamily    string            `json:"author_family"`
	ReviewerFamily  string            `json:"reviewer_family"`
	BlockerCount    *int              `json:"blocker_count"`
	Approval        json.RawMessage   `json:"approval"`
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
	CandidateSHA256  string          `json:"candidate_sha256"`
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
	if (strings.TrimSpace(proj.Service.InstanceID) == "" && strings.TrimSpace(proj.Service.ServiceID) == "") || !factoryV1TimestampValid(proj.Service.StartedAt) {
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
		if basis, receipt := factoryV1OrderApproval(order); basis != "" {
			order.HumanApprovalBasis = basis
			order.HumanApprovalReceipt = receipt
		}
		missing := factoryV1OrderMissingEvidence(order)
		status := strings.ToLower(strings.TrimSpace(order.Status))
		switch status {
		case "accepted", "queued", "progressing", "blocked", "human_required", "human_review":
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
	if basis, receipt := factoryV1OrderApproval(order); basis != "" {
		order.HumanApprovalBasis = basis
		order.HumanApprovalReceipt = receipt
	}
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
	require(factoryV1SHA256Pattern.MatchString(strings.TrimSpace(order.DocumentSHA256)), "document_sha256")
	require(factoryV1TLCStageIndexValid(order.TLCStage, order.TLCIndex), "TLC stage/index")
	require(strings.TrimSpace(order.GateState) != "", "gate_state")
	require(strings.TrimSpace(order.NextAction) != "", "next_action")
	limit, consumed, remaining := factoryV1BudgetAttempts(order.Budget)
	require(limit > 0 && consumed >= 0 && remaining >= 0, "budget")
	status := strings.ToLower(strings.TrimSpace(order.Status))
	initiallyAccepted := order.TLCIndex == 0 && len(order.Stages) == 0 && (status == "accepted" || status == "queued" || status == "human_required")
	if !initiallyAccepted {
		require(factoryV1OrderCanonicalEvidencePresent(order), "canonical evidence")
		require(factoryV1StageLedgerValid(order.Stages), "stage ledger")
	}
	if order.TLCIndex >= 6 {
		basis := strings.TrimSpace(order.HumanApprovalBasis)
		require(basis == "standing_scoped" || basis == "fresh_scoped", "human_approval_basis")
		require(factoryV1ApprovalReceiptValid(order, order.HumanApprovalReceipt), "human_approval_receipt")
	}
	// Hive projects the next stage immediately after a pass. PR identity is
	// therefore due only after create_draft_pr (index 7) has passed and IAR is
	// current, not while create_draft_pr itself is accepted or running.
	if order.TLCIndex >= 8 {
		require(strings.TrimSpace(order.PR.Repository) != "" && order.PR.Number > 0 && factoryV1GitHashPattern.MatchString(strings.TrimSpace(order.PR.HeadSHA)), "PR identity/head")
	}
	// The cumulative PR projection becomes exact-head, green, and non-draft
	// only when mark_pr_ready (index 10) passes into Human Review (index 11).
	if order.TLCIndex >= 11 {
		require(factoryV1GitHashPattern.MatchString(strings.TrimSpace(order.PR.ReviewedHeadSHA)) && order.PR.ReviewedHeadSHA == order.PR.HeadSHA, "exact reviewed PR head")
		require(order.PR.Open && !order.PR.Draft && order.PR.ChecksPassing, "ready PR with passing checks")
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

func factoryV1StageLedgerValid(stages []FactoryV1Stage) bool {
	if len(stages) == 0 {
		return false
	}
	for _, stage := range stages {
		state := factoryV1StageState(stage)
		if !factoryV1TLCStageIndexValid(stage.Stage, stage.Index) || strings.TrimSpace(stage.AttemptID) == "" || strings.TrimSpace(stage.EventID) == "" || len(stage.Peers) == 0 {
			return false
		}
		switch state {
		case "running":
		case "passed", "blocked", "human_required":
			if len(stage.Evidence) == 0 || strings.TrimSpace(stage.WorkArtifactID) == "" {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func factoryV1RawPresent(raw json.RawMessage) bool {
	v := strings.TrimSpace(string(raw))
	return v != "" && v != "null" && v != `""` && v != "{}" && v != "[]"
}

func factoryV1ApprovalReceiptValid(order FactoryV1Order, raw json.RawMessage) bool {
	if !factoryV1RawPresent(raw) {
		return false
	}
	var receipt struct {
		Basis                 string `json:"basis"`
		ActorID               string `json:"actor_id"`
		CredentialKeyID       string `json:"credential_key_id"`
		SourceSHA256          string `json:"source_sha256"`
		FactoryOrderBlobSHA   string `json:"factory_order_blob_sha"`
		OrderID               string `json:"order_id"`
		OrderVersion          string `json:"order_version"`
		DocumentSHA256        string `json:"document_sha256"`
		ApprovalSentence      string `json:"approval_sentence"`
		ApprovalSourceEventID string `json:"approval_source_event_id"`
		IssuedAt              string `json:"issued_at"`
	}
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return false
	}
	basis := strings.TrimSpace(receipt.Basis)
	return (basis == "standing_scoped" || basis == "fresh_scoped") &&
		strings.TrimSpace(receipt.ActorID) != "" && strings.TrimSpace(receipt.CredentialKeyID) != "" &&
		factoryV1SHA256Pattern.MatchString(strings.TrimSpace(receipt.SourceSHA256)) &&
		factoryV1SHA256Pattern.MatchString(strings.TrimSpace(receipt.FactoryOrderBlobSHA)) &&
		receipt.OrderID == order.OrderID && receipt.OrderVersion == order.Version &&
		receipt.DocumentSHA256 == order.DocumentSHA256 &&
		strings.TrimSpace(receipt.ApprovalSentence) != "" && strings.TrimSpace(receipt.ApprovalSourceEventID) != "" &&
		factoryV1TimestampValid(receipt.IssuedAt)
}

func factoryV1OrderApproval(order FactoryV1Order) (string, json.RawMessage) {
	if factoryV1RawPresent(order.HumanApprovalReceipt) {
		return strings.TrimSpace(order.HumanApprovalBasis), order.HumanApprovalReceipt
	}
	for stageIndex := len(order.Stages) - 1; stageIndex >= 0; stageIndex-- {
		for evidenceIndex := len(order.Stages[stageIndex].Evidence) - 1; evidenceIndex >= 0; evidenceIndex-- {
			raw := order.Stages[stageIndex].Evidence[evidenceIndex].Approval
			if !factoryV1RawPresent(raw) {
				continue
			}
			var receipt struct {
				Basis string `json:"basis"`
			}
			if json.Unmarshal(raw, &receipt) == nil {
				return strings.TrimSpace(receipt.Basis), raw
			}
		}
	}
	return "", nil
}

func factoryV1TimestampValid(value string) bool {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	return err == nil && !parsed.IsZero()
}

var factoryV1TLCStages = [...]string{
	"ingest_work", "craft_factory_order", "design", "iada", "cfada", "human_design_review",
	"write_code", "create_draft_pr", "iar", "cfar", "mark_pr_ready", "human_review",
}

func factoryV1TLCStageIndexValid(stage string, index int) bool {
	return index >= 0 && index < len(factoryV1TLCStages) && strings.TrimSpace(stage) == factoryV1TLCStages[index]
}

func factoryV1SourceRefValid(raw json.RawMessage) bool {
	if !factoryV1RawPresent(raw) {
		return false
	}
	var source struct {
		Identity string `json:"identity"`
		SHA256   string `json:"sha256"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return false
	}
	return strings.TrimSpace(source.Identity) != "" && factoryV1SHA256Pattern.MatchString(strings.TrimSpace(source.SHA256))
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
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	revision, err := strconv.Atoi(strings.TrimSpace(r.FormValue("revision")))
	if err != nil || revision <= 0 {
		http.Error(w, "revision must be a positive integer", http.StatusBadRequest)
		return
	}
	candidateSHA256 := strings.TrimSpace(r.FormValue("candidate_sha256"))
	if !factoryV1SHA256Pattern.MatchString(candidateSHA256) {
		http.Error(w, "candidate_sha256 must be 64 lowercase hexadecimal characters", http.StatusBadRequest)
		return
	}
	h.factoryV1Mutation(w, r, "/api/hive/factory/v1/ideas/"+id+"/submit", map[string]any{"approved": true, "revision": revision, "candidate_sha256": candidateSHA256})
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
	actorID, operatorPrincipalID, ok := h.factoryV1ActorIdentity(r)
	if !ok {
		http.Error(w, "factory operator identity is not configured", http.StatusUnauthorized)
		return
	}
	h.factoryV1Mutation(w, r, "/api/hive/factory/v1/interventions/"+id+"/resolve", map[string]any{"resolution": resolution, "actor_id": actorID, "operator_principal_id": operatorPrincipalID})
}

// factoryV1ActorID binds intervention receipts to a configured EventGraph
// Human actor in the local acceptance stack. An authenticated Site identity is
// the fallback for normal application use; the anonymous sentinel is never
// forwarded as Human authority.
func (h *Handlers) factoryV1ActorIdentity(r *http.Request) (actorID, operatorPrincipalID string, ok bool) {
	principal := h.userID(r)
	if principal == anonUserID {
		return "", "", false
	}
	principal, principalOK := validFactoryV1ID(principal)
	if !principalOK {
		return "", "", false
	}
	configured := strings.TrimSpace(os.Getenv("HIVE_FACTORY_V1_ACTOR_ID"))
	if configured == "" {
		return principal, principal, true
	}
	configured, configuredOK := validFactoryV1ID(configured)
	if !configuredOK {
		return "", "", false
	}
	return configured, principal, true
}

func (h *Handlers) factoryV1Mutation(w http.ResponseWriter, r *http.Request, endpoint string, payload any) {
	projection, err := fetchFactoryV1Projection(r)
	view := buildFactoryV1MissionControl(projection, err, time.Now().UTC())
	if !view.Writable {
		reason := "factory writes are unavailable"
		if len(view.Notices) != 0 {
			reason += ": " + strings.Join(view.Notices, "; ")
		} else {
			reason += fmt.Sprintf(": projection freshness is %s and service status is %s", view.Freshness, factoryV1ServiceStatus(view.Service))
		}
		http.Error(w, reason, http.StatusServiceUnavailable)
		return
	}
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
			return factoryV1RawPresent(revision.Candidate) && factoryV1SHA256Pattern.MatchString(revision.CandidateSHA256) && len(revision.ValidationErrors) == 0
		}
	}
	return false
}

func factoryV1CurrentIdeaSHA256(idea FactoryV1Idea) string {
	for _, revision := range idea.Revisions {
		if revision.Revision == idea.CurrentRevision {
			return revision.CandidateSHA256
		}
	}
	return ""
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
		if !pr.ChecksPassing {
			return "open; checks not passing"
		}
		if !factoryV1GitHashPattern.MatchString(pr.HeadSHA) || pr.HeadSHA != pr.ReviewedHeadSHA {
			return "open; exact-head review missing"
		}
		return "open exact-head ready"
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
	return `{"doc_id":"FO-DEMO-001","version":"1.0.0","status":"approved","title":"Bounded demonstration","channel":"completed_factory_order","target_repository":"transpara-ai/docs","source_references":[{"kind":"human_submission","identity":"site:completed-order","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}],"requirements":[{"id":"R1","statement":"Make one bounded change.","rationale":"Human supplied completed order."}],"acceptance_criteria":[{"id":"AC1","statement":"Change is verified.","verification_method":"Run repository tests.","risk_class":"medium"}],"test_plan":["Run repository verification."],"constraints":["Non-production only"],"non_goals":["Unrelated refactors"],"expected_outputs":["Ready PR"],"authority_scope":{"actor_id":"actor_00000000000000000000000000000086","allowed_actions":["repo.branch.create","repo.commit.create","repo.pull_request.create","repo.pull_request.mark_ready"],"target_repositories":["transpara-ai/docs"],"non_production_only":true},"budget":{"max_attempts":24,"max_tokens":2000000,"max_cost_micros":100000000}}`
}
