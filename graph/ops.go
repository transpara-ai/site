package graph

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	View        string // optional sub-view selector; "forensic" shows the full evidence tiers
	Surfaces    []OpsSurface
	EmbedURL    string
	EmbedLabel  string
	Telemetry   *OpsTelemetryData
	Work        *OpsWorkData
	Hive        *OpsHiveData
	HiveShell   *OpsHiveShellData
	Evidence    *OpsEvidenceData
	Decision    *OpsDecisionData
	LegacyURL   string
}

type OpsHiveShellData struct {
	Active    string
	Intake    OpsHiveIntakeView
	Runs      []OpsHiveRunView
	Agents    []OpsHiveAgentView
	Resources []OpsHiveResourceView
}

type OpsHiveShellCard struct {
	ID          string
	Label       string
	Href        string
	Description string
	Status      string
}

type OpsHiveSourceView struct {
	Kind   string
	Title  string
	Detail string
	Status string
}

type OpsHiveMissingFieldView struct {
	Label  string
	Detail string
	Status string
}

type OpsHiveIntakeView struct {
	Status          string
	Confidence      string
	SuggestedMode   string
	AuthorityLevel  string
	EstimatedBudget string
	Sources         []OpsHiveSourceView
	MissingFields   []OpsHiveMissingFieldView
}

type OpsHiveRunView struct {
	ID        string
	Title     string
	Status    string
	Guardian  string
	Budget    string
	Phase     string
	Approvals int
	Artifacts int
	UpdatedAt string
}

type OpsHiveAgentView struct {
	Name      string
	Role      string
	State     string
	Budget    string
	LastEvent string
}

type OpsHiveResourceView struct {
	Label  string
	Used   string
	Limit  string
	Status string
	Detail string
}

type OpsTelemetryData struct {
	WorkURL              string
	PipelineURL          string
	GeneratedAt          string
	ActorCount           int
	AgentCount           int
	ActiveAgentCount     int
	ProcessingAgentCount int
	PhaseLabel           string
	PhaseStatus          string
	Pipeline             *OpsPipelineReport
	PipelineError        string
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

type OpsPipelineReport struct {
	CycleID           string             `json:"cycle_id"`
	Status            string             `json:"status"`
	CurrentStage      string             `json:"current_stage"`
	CurrentPhase      string             `json:"current_phase"`
	LastOutcome       string             `json:"last_outcome"`
	LastSummary       string             `json:"last_summary"`
	StartedAt         string             `json:"started_at"`
	UpdatedAt         string             `json:"updated_at"`
	DurationSecs      float64            `json:"duration_secs"`
	TotalCostUSD      float64            `json:"total_cost_usd"`
	TotalInputTokens  int                `json:"total_input_tokens"`
	TotalOutputTokens int                `json:"total_output_tokens"`
	TotalTokens       int                `json:"total_tokens"`
	OpenBoardItems    int                `json:"open_board_items"`
	ReviseCount       int                `json:"revise_count"`
	IntakeComplete    bool               `json:"intake_complete"`
	DesignComplete    bool               `json:"design_complete"`
	EmissionComplete  bool               `json:"emission_complete"`
	Phases            []OpsPipelinePhase `json:"phases"`
	HumanStatus       string             `json:"human_status"`
}

type OpsPipelinePhase struct {
	CycleID       string  `json:"cycle_id"`
	Phase         string  `json:"phase"`
	WorkflowStage string  `json:"workflow_stage"`
	Outcome       string  `json:"outcome"`
	Repo          string  `json:"repo"`
	TaskID        string  `json:"task_id"`
	TaskTitle     string  `json:"task_title"`
	DurationSecs  float64 `json:"duration_secs"`
	CostUSD       float64 `json:"cost_usd"`
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	BoardOpen     int     `json:"board_open"`
	ReviseCount   int     `json:"revise_count"`
	Summary       string  `json:"summary"`
	Error         string  `json:"error"`
	InputRef      string  `json:"input_ref"`
	OutputRef     string  `json:"output_ref"`
	RecordedAt    string  `json:"recorded_at"`
}

type OpsWorkData struct {
	WorkURL       string
	GeneratedAt   string
	Total         int
	Open          int
	Active        int
	Blocked       int
	Completed     int
	Ready         int
	HighPriority  int
	Unassigned    int
	EvidenceCount int
	WaivedCount   int
	RecentTasks   []OpsWorkTask
	BlockedTasks  []OpsWorkTask
	PhaseGates    []OpsPhaseGate
	Error         string
}

type OpsWorkTask struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Priority      string   `json:"priority"`
	Workspace     string   `json:"workspace"`
	Status        string   `json:"status"`
	Assignee      string   `json:"assignee"`
	Blocked       bool     `json:"blocked"`
	ArtifactCount int      `json:"artifact_count"`
	Waived        bool     `json:"waived"`
	Ready         bool     `json:"ready"`
	MissingGates  []string `json:"missing_gates"`
}

type OpsPhaseGate struct {
	ID         string   `json:"id"`
	Phase      string   `json:"phase"`
	Title      string   `json:"title"`
	Criteria   []string `json:"criteria"`
	Status     string   `json:"status"`
	Summary    string   `json:"summary"`
	Reason     string   `json:"reason"`
	DeclaredAt string   `json:"declared_at"`
	UpdatedAt  string   `json:"updated_at"`
}

type OpsHiveData struct {
	GeneratedAt        string
	Iteration          int
	Phase              string
	BuildTitle         string
	BuildCost          float64
	TotalOps           int
	LastActive         string
	VerifiedBuilds     int
	TotalCost          float64
	AverageCost        float64
	DiagnosticCount    int
	ProjectionSource   string
	ProjectionError    string
	PendingApprovals   []OpsHiveApproval
	AuthorityDecisions []OpsHiveDecision
	Lifecycle          []OpsHiveLifecycle
	KeyAuditTraces     []OpsHiveKeyAuditTrace
	RecentCommits      []RecentCommit
	RecentEvents       []DiagEntry
	Error              string
}

type OpsHiveProjection struct {
	GeneratedAt        string                 `json:"generated_at"`
	Source             string                 `json:"source"`
	PendingApprovals   []OpsHiveApproval      `json:"pending_approvals"`
	AuthorityDecisions []OpsHiveDecision      `json:"authority_decisions"`
	Lifecycle          []OpsHiveLifecycle     `json:"lifecycle"`
	KeyAuditTraces     []OpsHiveKeyAuditTrace `json:"key_audit_traces"`
	Errors             []string               `json:"errors"`
}

type OpsHiveApproval struct {
	EventID           string `json:"event_id"`
	RequestID         string `json:"request_id"`
	RequestingActor   string `json:"requesting_actor"`
	ActionName        string `json:"action_name"`
	Target            string `json:"target"`
	Environment       string `json:"environment"`
	RequestedOutcome  string `json:"requested_outcome"`
	Justification     string `json:"justification"`
	RiskSummary       string `json:"risk_summary"`
	ProposedOperation string `json:"proposed_operation"`
	CreatedAt         string `json:"created_at"`
}

type OpsHiveDecision struct {
	EventID         string `json:"event_id"`
	DecisionID      string `json:"decision_id"`
	RequestID       string `json:"request_id"`
	ApproverActor   string `json:"approver_actor"`
	Outcome         string `json:"outcome"`
	ApprovedAction  string `json:"approved_action"`
	ApprovedTarget  string `json:"approved_target"`
	Rationale       string `json:"rationale"`
	RequestedAction string `json:"requested_action"`
	RequestedTarget string `json:"requested_target"`
	CreatedAt       string `json:"created_at"`
}

type OpsHiveLifecycle struct {
	ActorID         string `json:"actor_id"`
	DisplayName     string `json:"display_name"`
	Role            string `json:"role"`
	LifecycleStatus string `json:"lifecycle_status"`
	AuthorityScope  string `json:"authority_scope"`
	KeyProvenance   string `json:"key_provenance"`
	Environment     string `json:"environment"`
	IdentityMode    string `json:"identity_mode"`
	LastEventType   string `json:"last_event_type"`
	UpdatedAt       string `json:"updated_at"`
}

type OpsHiveKeyAuditTrace struct {
	EventID          string `json:"event_id"`
	EventType        string `json:"event_type"`
	ActorID          string `json:"actor_id"`
	SubjectActorID   string `json:"subject_actor_id"`
	KeyProvenance    string `json:"key_provenance"`
	Environment      string `json:"environment"`
	IdentityMode     string `json:"identity_mode"`
	PublicKey        string `json:"public_key"`
	OldPublicKey     string `json:"old_public_key"`
	NewPublicKey     string `json:"new_public_key"`
	ExternalKeyRef   string `json:"external_key_ref"`
	Reason           string `json:"reason"`
	RecordKind       string `json:"record_kind"`
	Rationale        string `json:"rationale"`
	AuthorityRequest string `json:"authority_request"`
	DecisionEvent    string `json:"decision_event"`
	CreatedAt        string `json:"created_at"`
}

// OpsRoleTimelineRow is one row in the society-view role timeline.
// Roles are displayed in the canonical civic order:
//
//	strategist → planner → implementer → reviewer → guardian → human (Site approval) → draft PR
type OpsRoleTimelineRow struct {
	Role   string // canonical role label (e.g. "strategist", "human (Site approval)", "draft PR")
	Actors string // actor display names observed for this role, comma-separated; empty if none seen
	Status string // lifecycle_status of the actor(s), or derived status for synthetic rows
	Notes  string // additional context (e.g. approved action for the human row)
}

type OpsEvidenceData struct {
	GeneratedAt        string
	Source             string
	ProjectionURL      string
	ProjectionError    string
	FactoryOrderID     string
	ReleaseCandidateID string
	FactoryOrder       *OpsEvidenceFactoryOrder
	ReleaseCandidate   *OpsEvidenceReleaseCandidate
	Decision           *OpsEvidenceDecision
	AuditReport        *OpsEvidenceAuditReport
	Timeline           []OpsEvidenceTimelineEvent
	GateEvidence       []OpsEvidenceGate
	ReleaseEvidence    []OpsEvidenceReleaseEvidence
	FailuresRepairs    []OpsEvidenceFailureRepair
	MissingProvenance  []OpsEvidenceMissingProvenance
	ProofOfWorkPacket  *OpsProofOfWorkPacket
	Errors             []string
	// RoleTimeline is the society view: order built by the civic roles.
	// Built from the hive operator projection Lifecycle + AuthorityDecisions.
	// Empty when HIVE_OPS_API_BASE_URL is not configured.
	RoleTimeline []OpsRoleTimelineRow
}

type OpsDecisionData struct {
	AuthorizationSource string
	CorrelationID       string
	TraceID             string
	RequestID           string
	RequestedAction     string
	DecisionReason      string
	TargetType          string
	Repo                string
	TargetRef           string
	Status              string
	Effect              string
	OperatorSummary     string
	BlockedReasons      []string
	RequiredEvidence    []string
	Actions             []OpsDecisionAction
	GovernancePosture   []OpsDecisionPosture
	BoundaryChecks      []OpsDecisionPosture
}

// opsDecisionApprove, opsDecisionDeny, opsDecisionMoreEvidence are the canonical
// wire values emitted by the decision form buttons and tested by the submit handler.
// Using shared constants ensures the UI and handler can never drift independently.
//
//   - opsDecisionApprove / opsDecisionDeny are forwarded to hive normalised to the
//     hive-canonical past-tense form ("approved" / "denied").
//   - opsDecisionMoreEvidence is handled locally; hive is never called.
const (
	opsDecisionApprove      = "approve"
	opsDecisionDeny         = "deny"
	opsDecisionMoreEvidence = "request-more-evidence"
)

type OpsDecisionAction struct {
	Label       string
	WireValue   string
	Description string
}

type OpsDecisionPosture struct {
	Label  string
	Field  string
	Value  string
	Status string
}

type OpsEvidenceProjection struct {
	GeneratedAt       string                         `json:"generated_at"`
	Source            string                         `json:"source"`
	FactoryOrder      *OpsEvidenceFactoryOrder       `json:"factory_order"`
	ReleaseCandidate  *OpsEvidenceReleaseCandidate   `json:"release_candidate"`
	Decision          *OpsEvidenceDecision           `json:"decision"`
	AuditReport       *OpsEvidenceAuditReport        `json:"audit_report"`
	Timeline          []OpsEvidenceTimelineEvent     `json:"timeline"`
	GateEvidence      []OpsEvidenceGate              `json:"gate_evidence"`
	ReleaseEvidence   []OpsEvidenceReleaseEvidence   `json:"release_evidence"`
	FailuresRepairs   []OpsEvidenceFailureRepair     `json:"failures_repairs"`
	MissingProvenance []OpsEvidenceMissingProvenance `json:"missing_provenance"`
	ProofOfWorkPacket *OpsProofOfWorkPacket          `json:"proof_of_work_packet"`
	Errors            []string                       `json:"errors"`
}

type OpsEvidenceFactoryOrder struct {
	ID               string `json:"id"`
	Version          int    `json:"version"`
	Status           string `json:"status"`
	SourceIntentHash string `json:"source_intent_hash"`
	SourceIntentRef  string `json:"source_intent_ref"`
	RiskClass        string `json:"risk_class"`
	ReleasePolicy    string `json:"release_policy"`
}

type OpsEvidenceReleaseCandidate struct {
	ID                      string   `json:"id"`
	Status                  string   `json:"status"`
	FactoryOrderID          string   `json:"factory_order_id"`
	FactoryRuntimeVersionID string   `json:"factory_runtime_version_id"`
	ArtifactRefs            []string `json:"artifact_refs"`
}

type OpsEvidenceDecision struct {
	Kind         string   `json:"kind"`
	ID           string   `json:"id"`
	ActorID      string   `json:"actor_id"`
	Reason       string   `json:"reason"`
	EvidenceRefs []string `json:"evidence_refs"`
	Status       string   `json:"status"`
	CreatedAt    string   `json:"created_at"`
}

type OpsEvidenceAuditReport struct {
	ID           string   `json:"id"`
	TargetType   string   `json:"target_type"`
	TargetID     string   `json:"target_id"`
	Status       string   `json:"status"`
	TraceScore   float64  `json:"trace_score"`
	MissingLinks []string `json:"missing_links"`
}

type OpsEvidenceTimelineEvent struct {
	Label     string `json:"label"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	NodeID    string `json:"node_id"`
	CreatedAt string `json:"created_at"`
	Summary   string `json:"summary"`
}

type OpsEvidenceGate struct {
	GateName     string   `json:"gate_name"`
	Status       string   `json:"status"`
	GateResultID string   `json:"gate_result_id"`
	EvidenceRefs []string `json:"evidence_refs"`
	WaiverRef    string   `json:"waiver_ref"`
	MissingRefs  []string `json:"missing_refs"`
}

type OpsEvidenceReleaseEvidence struct {
	Label            string   `json:"label"`
	Status           string   `json:"status"`
	ArtifactRefs     []string `json:"artifact_refs"`
	RuntimeRefs      []string `json:"runtime_refs"`
	BOMRefs          []string `json:"bom_refs"`
	RequiredPathRefs []string `json:"required_path_refs"`
	MissingRefs      []string `json:"missing_refs"`
}

type OpsEvidenceFailureRepair struct {
	FailureID         string `json:"failure_id"`
	FailureClass      string `json:"failure_class"`
	Severity          string `json:"severity"`
	Summary           string `json:"summary"`
	TaskID            string `json:"task_id"`
	GateResultID      string `json:"gate_result_id"`
	TestRunID         string `json:"test_run_id"`
	RepairID          string `json:"repair_id"`
	RepairStatus      string `json:"repair_status"`
	ActorInvocationID string `json:"actor_invocation_id"`
}

type OpsEvidenceMissingProvenance struct {
	PathName  string   `json:"path_name"`
	NodeIDs   []string `json:"node_ids"`
	EdgeIDs   []string `json:"edge_ids"`
	Missing   []string `json:"missing"`
	Completed bool     `json:"completed"`
}

type OpsProofOfWorkPacket struct {
	ID                     string               `json:"id"`
	Status                 string               `json:"status"`
	Summary                string               `json:"summary"`
	WorkItem               *OpsProofOfWorkItem  `json:"work_item"`
	RuntimeInvocation      *OpsProofOfWorkItem  `json:"runtime_invocation"`
	ChangedFiles           []OpsProofOfWorkItem `json:"changed_files"`
	TestsRun               []OpsProofOfWorkItem `json:"tests_run"`
	CIStatus               *OpsProofOfWorkItem  `json:"ci_status"`
	ReviewFeedback         []OpsProofOfWorkItem `json:"review_feedback"`
	SecurityScanResults    []OpsProofOfWorkItem `json:"security_scan_results"`
	ScreenshotsWalkthrough []OpsProofOfWorkItem `json:"screenshots_walkthrough_artifacts"`
	KnownFailures          []OpsProofOfWorkItem `json:"known_failures"`
	OperatorDecision       *OpsProofOfWorkItem  `json:"operator_decision"`
	EventGraphRefs         []string             `json:"event_graph_refs"`
}

type OpsProofOfWorkItem struct {
	Label          string   `json:"label"`
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	ArtifactRef    string   `json:"artifact_ref"`
	EventGraphRefs []string `json:"event_graph_refs"`
}

type opsWorkTasksResponse struct {
	Tasks []OpsWorkTask `json:"tasks"`
}

type opsPhaseGatesResponse struct {
	Gates []OpsPhaseGate `json:"gates"`
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

type opsPipelineReportResponse struct {
	ComputedAt string             `json:"computed_at"`
	Report     *OpsPipelineReport `json:"report"`
	Error      string             `json:"error"`
	Detail     string             `json:"detail"`
}

var hiveOpsProjectionClient = &http.Client{Timeout: 3 * time.Second}
var evidenceOpsProjectionClient = &http.Client{Timeout: 3 * time.Second}

func (h *Handlers) handleOps(w http.ResponseWriter, r *http.Request) {
	h.renderOps(w, r, OpsPageData{
		Title:       "Operations",
		Description: "Site-owned operator shell for work, telemetry, hive status, evidence, and refinery review.",
		Active:      "overview",
	})
}

func (h *Handlers) handleOpsWork(w http.ResponseWriter, r *http.Request) {
	h.renderOps(w, r, OpsPageData{
		Title:       "Work",
		Description: "Native Site task summary sourced from the Work API.",
		Active:      "work",
		Work:        fetchOpsWork(r),
	})
}

func (h *Handlers) handleOpsTelemetry(w http.ResponseWriter, r *http.Request) {
	h.renderOps(w, r, OpsPageData{
		Title:       "Telemetry",
		Description: "Native Site telemetry summary sourced from Work APIs.",
		Active:      "telemetry",
		Telemetry:   fetchOpsTelemetry(r),
	})
}

func (h *Handlers) handleOpsHive(w http.ResponseWriter, r *http.Request) {
	h.renderOps(w, r, OpsPageData{
		Title:       "Hive",
		Description: "Native Site operator summary for Hive runtime status, diagnostics, and build signal.",
		Active:      "hive",
		Hive:        h.fetchOpsHive(r),
		LegacyURL:   "/hive",
	})
}

func (h *Handlers) handleOpsHiveIntake(w http.ResponseWriter, r *http.Request) {
	h.renderOps(w, r, OpsPageData{
		Title:       "Hive intake",
		Description: "Static Site-owned intake shell for source capture, interpretation, and launch readiness.",
		Active:      "hive",
		HiveShell:   buildOpsHiveShellData("intake"),
	})
}

func (h *Handlers) handleOpsHiveRuns(w http.ResponseWriter, r *http.Request) {
	h.renderOps(w, r, OpsPageData{
		Title:       "Hive runs",
		Description: "Static Site-owned run tower with sample pipeline, approval, artifact, and Guardian state.",
		Active:      "hive",
		HiveShell:   buildOpsHiveShellData("runs"),
	})
}

func (h *Handlers) handleOpsHiveAgents(w http.ResponseWriter, r *http.Request) {
	h.renderOps(w, r, OpsPageData{
		Title:       "Hive agents",
		Description: "Static Site-owned agent topology with sample roles, lifecycle state, and budget posture.",
		Active:      "hive",
		HiveShell:   buildOpsHiveShellData("agents"),
	})
}

func (h *Handlers) handleOpsHiveResources(w http.ResponseWriter, r *http.Request) {
	h.renderOps(w, r, OpsPageData{
		Title:       "Hive resources",
		Description: "Static Site-owned resource dashboard with sample budget, queue, and capacity signals.",
		Active:      "hive",
		HiveShell:   buildOpsHiveShellData("resources"),
	})
}

func (h *Handlers) handleOpsEvidence(w http.ResponseWriter, r *http.Request) {
	evidence := fetchOpsEvidence(r)
	// Build the society-view role timeline from the hive operator projection.
	// Failure is non-fatal: if the projection is unavailable, RoleTimeline holds
	// all canonical rows in empty state (no actors/statuses).
	if proj, err := fetchHiveOperatorProjection(r); err == nil {
		evidence.RoleTimeline = buildOpsRoleTimeline(proj)
		if proj.GeneratedAt != "" {
			evidence.GeneratedAt = formatOpsTime(proj.GeneratedAt)
		}
	} else {
		evidence.RoleTimeline = buildOpsRoleTimeline(nil)
	}
	h.renderOps(w, r, OpsPageData{
		Title:       "Evidence",
		Description: "Read-only operator projection for FactoryOrder timeline, gate, release, audit, failure, repair, and provenance evidence.",
		Active:      "evidence",
		View:        strings.ToLower(strings.TrimSpace(r.URL.Query().Get("view"))),
		Evidence:    evidence,
	})
}

func (h *Handlers) handleOpsDecision(w http.ResponseWriter, r *http.Request) {
	h.renderOps(w, r, OpsPageData{
		Title:       "Decision boundary",
		Description: "Non-executing Gate E decision surface for bounded approve, deny, and request-more-evidence envelopes.",
		Active:      "decision",
		Decision:    buildOpsDecisionData(r),
	})
}

// handleOpsDecisionSubmit forwards the operator's approve/deny to hive and
// re-renders the decision surface with a confirmation or error.
// Site emits a GOVERNANCE POST only — it never calls GitHub or writes the graph.
//
// Only "approved" and "denied" are forwarded to hive; hive's operator-decision
// endpoint accepts only those two values and returns 400 on anything else.
// "request-more-evidence" (and any unrecognised value) is handled LOCALLY:
// Site re-renders the decision page with a note that no decision was recorded.
// Requesting more evidence is not a recordable governance decision.
func (h *Handlers) handleOpsDecisionSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	requestID := strings.TrimSpace(r.FormValue("request_id"))
	decision := strings.TrimSpace(r.FormValue("decision"))
	reason := strings.TrimSpace(r.FormValue("reason"))
	approver := strings.TrimSpace(r.FormValue("approver"))

	// Normalise the UI wire value to the hive-canonical past-tense form.
	// opsDecisionApprove ("approve") → hive receives "approved"
	// opsDecisionDeny ("deny")       → hive receives "denied"
	// Anything else (including opsDecisionMoreEvidence) is handled locally;
	// hive only accepts "approved" / "denied" and would 400 on anything else.
	var hiveDecision string
	switch decision {
	case opsDecisionApprove:
		hiveDecision = "approved"
	case opsDecisionDeny:
		hiveDecision = "denied"
	default:
		h.renderOpsDecisionWithConfirmation(w, r, requestID, decision, "more-evidence-local")
		return
	}

	base := strings.TrimSpace(os.Getenv("HIVE_OPS_API_BASE_URL"))
	if base == "" {
		h.renderOpsDecisionWithConfirmation(w, r, requestID, "", "HIVE_OPS_API_BASE_URL is not configured")
		return
	}
	apiKey := strings.TrimSpace(os.Getenv("HIVE_OPS_API_KEY"))

	// Build the governance POST body (JSON — matches hive contract).
	// Use hiveDecision (past-tense canonical) rather than the UI wire value.
	payload := map[string]string{
		"request_id": requestID,
		"decision":   hiveDecision,
		"approver":   approver,
		"reason":     reason,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		h.renderOpsDecisionWithConfirmation(w, r, requestID, "", "internal error marshalling request: "+err.Error())
		return
	}

	endpoint := strings.TrimRight(base, "/") + "/api/hive/operator-decision"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		h.renderOpsDecisionWithConfirmation(w, r, requestID, "", "could not build hive request: "+err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := hiveOpsProjectionClient.Do(req)
	if err != nil {
		h.renderOpsDecisionWithConfirmation(w, r, requestID, "", "hive unreachable: "+err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		h.renderOpsDecisionWithConfirmation(w, r, requestID, "", fmt.Sprintf("hive returned %s", resp.Status))
		return
	}

	h.renderOpsDecisionWithConfirmation(w, r, requestID, hiveDecision, "")
}

// renderOpsDecisionWithConfirmation re-renders the decision surface with either
// a "decision recorded" confirmation, a local evidence-request note, or an error
// message. effect = none is preserved in all cases.
//
// errMsg sentinel values:
//
//	""                  → success: governance POST forwarded for approved/denied.
//	"more-evidence-local" → request-more-evidence handled locally; hive NOT called.
//	anything else       → governance POST failed; show error.
func (h *Handlers) renderOpsDecisionWithConfirmation(w http.ResponseWriter, r *http.Request, requestID, decision, errMsg string) {
	// Rebuild the display data from the real pending request (or fallback).
	var data *OpsDecisionData
	if requestID != "" {
		data = buildOpsDecisionDataFromProjection(r, requestID)
	} else {
		data = buildOpsDecisionData(r)
	}
	switch errMsg {
	case "":
		data.OperatorSummary = "Decision recorded on the graph by hive. Governance POST forwarded: decision=" + decision + ". Site remains display console; effect = none."
	case "more-evidence-local":
		// request-more-evidence is not a recordable governance decision; hive was NOT called.
		data.OperatorSummary = "More evidence requested — no decision recorded. Site handled this locally; hive was not called. effect = none."
	default:
		data.OperatorSummary = "Governance POST failed: " + errMsg + " (effect = none; no downstream action taken)"
		data.BlockedReasons = append(data.BlockedReasons, errMsg)
	}
	h.renderOps(w, r, OpsPageData{
		Title:       "Decision boundary",
		Description: "Non-executing Gate E decision surface for bounded approve, deny, and request-more-evidence envelopes.",
		Active:      "decision",
		Decision:    data,
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
	return []OpsSurface{
		{
			ID:          "work",
			Label:       "Work",
			Description: "Task queue, assignment, blockers, artifacts, and completion evidence.",
			Href:        "/ops/work",
			Target:      "work API /tasks",
			Owner:       "site shell, work API",
			Status:      "native summary",
		},
		{
			ID:          "telemetry",
			Label:       "Telemetry",
			Description: "Agent status, phase activity, pipeline report, and event stream health.",
			Href:        "/ops/telemetry",
			Target:      "work API /telemetry/*",
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
			Status:      "native summary",
		},
		{
			ID:          "evidence",
			Label:       "Evidence",
			Description: "Read-only FactoryOrder evidence projection: timeline, gates, release, audit, failures, repairs, and provenance gaps.",
			Href:        "/ops/evidence",
			Target:      "configured projection URL",
			Owner:       "site shell, eventgraph/work projection",
			Status:      "read only",
		},
		{
			ID:          "decision",
			Label:       "Decision",
			Description: "Gate E governed decision boundary with non-executing approve, deny, and request-more-evidence envelopes.",
			Href:        "/ops/decision",
			Target:      "local Site decision surface",
			Owner:       "site shell only",
			Status:      "effect none",
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

func opsHiveShellCards() []OpsHiveShellCard {
	return []OpsHiveShellCard{
		{
			ID:          "overview",
			Label:       "Overview",
			Href:        "/ops/hive",
			Description: "Runtime summary and public build link.",
			Status:      "native summary",
		},
		{
			ID:          "intake",
			Label:       "Intake",
			Href:        "/ops/hive/intake",
			Description: "Source capture, interpretation, and brief readiness.",
			Status:      "static sample",
		},
		{
			ID:          "runs",
			Label:       "Runs",
			Href:        "/ops/hive/runs",
			Description: "Run tower, pipeline phase, approvals, and artifacts.",
			Status:      "static sample",
		},
		{
			ID:          "agents",
			Label:       "Agents",
			Href:        "/ops/hive/agents",
			Description: "Agent topology, lifecycle state, and budget posture.",
			Status:      "static sample",
		},
		{
			ID:          "resources",
			Label:       "Resources",
			Href:        "/ops/hive/resources",
			Description: "Budget, queue, token, and capacity signals.",
			Status:      "static sample",
		},
	}
}

func buildOpsHiveShellData(active string) *OpsHiveShellData {
	return &OpsHiveShellData{
		Active: active,
		Intake: OpsHiveIntakeView{
			Status:          "draft ready",
			Confidence:      "0.91",
			SuggestedMode:   "full product pipeline",
			AuthorityLevel:  "human launch required",
			EstimatedBudget: "$18.00",
			Sources: []OpsHiveSourceView{
				{Kind: "PRD", Title: "checkout-redesign.md", Detail: "Product intent and acceptance criteria", Status: "parsed"},
				{Kind: "URL", Title: "customer-notes", Detail: "Reference source queued for review", Status: "classified"},
				{Kind: "Repo", Title: "transpara-ai/site", Detail: "UI boundary context selected", Status: "scoped"},
			},
			MissingFields: []OpsHiveMissingFieldView{
				{Label: "Rollback owner", Detail: "Name the human owner for launch rollback.", Status: "missing"},
				{Label: "Budget cap", Detail: "Confirm max spend before run launch.", Status: "warning"},
				{Label: "Target branch", Detail: "Choose the exact repo branch or draft-PR target.", Status: "ready"},
			},
		},
		Runs: []OpsHiveRunView{
			{ID: "run_static_001", Title: "Build onboarding control surface", Status: "active", Guardian: "clear", Budget: "18% used", Phase: "Design", Approvals: 2, Artifacts: 7, UpdatedAt: "sample now"},
			{ID: "run_static_002", Title: "Refine evidence inspection flow", Status: "waiting", Guardian: "watch", Budget: "4% used", Phase: "Research", Approvals: 0, Artifacts: 3, UpdatedAt: "sample 12m ago"},
			{ID: "run_static_003", Title: "Prepare integration checklist", Status: "paused", Guardian: "blocked", Budget: "31% used", Phase: "Review", Approvals: 1, Artifacts: 11, UpdatedAt: "sample 1h ago"},
		},
		Agents: []OpsHiveAgentView{
			{Name: "guardian", Role: "Risk and authority", State: "watching", Budget: "12%", LastEvent: "approval request classified"},
			{Name: "architect", Role: "System design", State: "active", Budget: "24%", LastEvent: "brief split into run graph"},
			{Name: "builder", Role: "Implementation", State: "waiting", Budget: "0%", LastEvent: "blocked until launch approval"},
			{Name: "tester", Role: "Verification", State: "idle", Budget: "0%", LastEvent: "waiting for artifact stream"},
		},
		Resources: []OpsHiveResourceView{
			{Label: "Run budget", Used: "$3.24", Limit: "$18.00", Status: "clear", Detail: "Human cap visible before launch."},
			{Label: "Approval queue", Used: "2", Limit: "unresolved", Status: "attention", Detail: "Operator decisions required before protected actions."},
			{Label: "Token budget", Used: "84k", Limit: "460k", Status: "clear", Detail: "Sample aggregate across active agents."},
			{Label: "Artifact store", Used: "7", Limit: "collected", Status: "clear", Detail: "Outputs stay inspectable and causally linked."},
		},
	}
}

// buildOpsDecisionDataFromProjection fetches the hive operator projection,
// finds the matching pending approval by request_id, and returns an
// OpsDecisionData populated from the real request. Effect stays "none".
func buildOpsDecisionDataFromProjection(r *http.Request, requestID string) *OpsDecisionData {
	projection, err := fetchHiveOperatorProjection(r)
	if err != nil {
		return &OpsDecisionData{
			AuthorizationSource: "transpara-ai/docs#75 / a50190ce470e8686a561e0e0b6e62ef0c5f5bb13",
			RequestID:           requestID,
			Status:              "blocked",
			Effect:              "none",
			OperatorSummary:     "Could not fetch hive operator projection: " + err.Error(),
			BlockedReasons:      []string{"hive projection unavailable: " + err.Error()},
			Actions:             opsDecisionActions(r),
		}
	}

	var found *OpsHiveApproval
	for i := range projection.PendingApprovals {
		if projection.PendingApprovals[i].RequestID == requestID {
			found = &projection.PendingApprovals[i]
			break
		}
	}
	if found == nil {
		return &OpsDecisionData{
			AuthorizationSource: "transpara-ai/docs#75 / a50190ce470e8686a561e0e0b6e62ef0c5f5bb13",
			RequestID:           requestID,
			Status:              "blocked",
			Effect:              "none",
			OperatorSummary:     "No pending approval found for request_id " + requestID,
			BlockedReasons:      []string{"request_id " + requestID + " not found in pending approvals"},
			Actions:             opsDecisionActions(r),
		}
	}

	// Target field encodes "repo ref" (space-separated); split into Repo and TargetRef.
	repo := ""
	targetRef := ""
	if parts := strings.SplitN(found.Target, " ", 2); len(parts) == 2 {
		repo = parts[0]
		targetRef = parts[1]
	} else {
		targetRef = found.Target
	}

	seed := strings.Join([]string{found.ActionName, found.Justification, repo, targetRef, requestID}, "|")
	return &OpsDecisionData{
		AuthorizationSource: "transpara-ai/docs#75 / a50190ce470e8686a561e0e0b6e62ef0c5f5bb13",
		CorrelationID:       "gate-e-correlation-" + opsDecisionHash(seed),
		TraceID:             "gate-e-trace-" + opsDecisionHash("trace|"+seed),
		RequestID:           found.RequestID,
		RequestedAction:     found.ActionName,
		TargetType:          "pull_request", // S1: OpsHiveApproval carries no target_type; Slice 1 handles only pull_request.create.
		Repo:                repo,
		TargetRef:           targetRef,
		Status:              "accepted_for_review",
		Effect:              "none",
		OperatorSummary:     found.Justification,
		BlockedReasons:      []string{},
		RequiredEvidence:    []string{},
		Actions:             opsDecisionActions(r),
		GovernancePosture:   []OpsDecisionPosture{},
		BoundaryChecks: []OpsDecisionPosture{
			{Label: "R-001", Field: "runner/worktree evidence gap", Value: "unresolved and excluded", Status: "blocked outside scope"},
			{Label: "R-002", Field: "real protected side effects", Value: "unresolved and excluded", Status: "blocked outside scope"},
			{Label: "R-003", Field: "policy adapter / policy bundle", Value: "unresolved and excluded", Status: "blocked outside scope"},
		},
	}
}

func buildOpsDecisionData(r *http.Request) *OpsDecisionData {
	q := r.URL.Query()

	// When request_id is present, load the real pending approval from the hive
	// operator projection and populate the decision surface from it.
	// Site remains a pure display console: effect = none is preserved.
	if requestID := strings.TrimSpace(q.Get("request_id")); requestID != "" {
		return buildOpsDecisionDataFromProjection(r, requestID)
	}

	actionRaw := strings.TrimSpace(q.Get("action"))
	action, actionOK := normalizeOpsDecisionAction(actionRaw)
	reason := strings.TrimSpace(q.Get("reason"))
	targetType := opsValueOr(strings.TrimSpace(q.Get("target_type")), "pull_request")
	repo := opsValueOr(strings.TrimSpace(q.Get("repo")), "transpara-ai/site")
	targetRef := strings.TrimSpace(q.Get("target_ref"))

	blocked := make([]string, 0)
	required := make([]string, 0)
	if actionRaw == "" {
		blocked = append(blocked, "missing requested_action")
		required = append(required, "requested_action must be approve, deny, or request-more-evidence")
	} else if !actionOK {
		blocked = append(blocked, "requested_action is outside the Gate E decision boundary")
		required = append(required, "requested_action must be approve, deny, or request-more-evidence")
	}
	if targetRef == "" {
		blocked = append(blocked, "missing target reference")
		required = append(required, "target_ref must point to an existing PR, issue, artifact, FactoryOrder, or release candidate")
	}
	if reason == "" {
		blocked = append(blocked, "missing decision_reason")
		required = append(required, "decision_reason must explain the operator-visible rationale")
	}
	if !opsDecisionTargetTypeAllowed(targetType) {
		blocked = append(blocked, "target_type is outside the governed decision boundary")
		required = append(required, "target_type must be pull_request, issue, artifact, factory_order, or release_candidate")
	}
	for _, guard := range []struct {
		keys   []string
		reason string
	}{
		{[]string{"direct_execution", "direct_execution_requested"}, "direct execution is forbidden for this Site slice"},
		{[]string{"protected_side_effect", "protected_side_effect_requested"}, "protected side effects are forbidden for this Site slice"},
		{[]string{"policy_adapter_reliance", "policy_adapter_reliance_requested"}, "policy-adapter reliance is forbidden for this Site slice"},
		{[]string{"runner_worktree_execution", "runner_worktree_execution_requested"}, "runner/worktree protected execution is forbidden for this Site slice"},
		{[]string{"production_autonomy", "production_autonomy_requested"}, "production autonomy is forbidden for this Site slice"},
		{[]string{"execution_receipt", "execution_receipt_requested"}, "ExecutionReceipt production behavior is forbidden for this Site slice"},
		{[]string{"policy_bundle", "policy_bundle_requested"}, "policy-bundle reliance is forbidden for this Site slice"},
	} {
		if opsDecisionQueryTrue(q, guard.keys...) {
			blocked = append(blocked, guard.reason)
		}
	}

	if action == "" {
		action = "none"
	}
	status := "accepted_for_review"
	summary := "Decision request stays inside the governed Gate E boundary. Site displays the envelope and trace only."
	if len(blocked) > 0 {
		status = "blocked"
		summary = "Request failed closed inside Site. No downstream action was executed."
	}

	seed := strings.Join([]string{action, reason, targetType, repo, targetRef, strings.Join(blocked, "|")}, "|")
	return &OpsDecisionData{
		AuthorizationSource: "transpara-ai/docs#75 / a50190ce470e8686a561e0e0b6e62ef0c5f5bb13",
		CorrelationID:       "gate-e-correlation-" + opsDecisionHash(seed),
		TraceID:             "gate-e-trace-" + opsDecisionHash("trace|"+seed),
		RequestedAction:     action,
		DecisionReason:      reason,
		TargetType:          targetType,
		Repo:                repo,
		TargetRef:           targetRef,
		Status:              status,
		Effect:              "none",
		OperatorSummary:     summary,
		BlockedReasons:      blocked,
		RequiredEvidence:    required,
		Actions:             opsDecisionActions(r),
		GovernancePosture: []OpsDecisionPosture{
			{Label: "direct execution", Field: "direct_execution_requested", Value: fmt.Sprintf("%t", opsDecisionQueryTrue(q, "direct_execution", "direct_execution_requested")), Status: opsDecisionGuardStatus(q, "direct_execution", "direct_execution_requested")},
			{Label: "protected side effects", Field: "protected_side_effect_requested", Value: fmt.Sprintf("%t", opsDecisionQueryTrue(q, "protected_side_effect", "protected_side_effect_requested")), Status: opsDecisionGuardStatus(q, "protected_side_effect", "protected_side_effect_requested")},
			{Label: "policy-adapter reliance", Field: "policy_adapter_reliance_requested", Value: fmt.Sprintf("%t", opsDecisionQueryTrue(q, "policy_adapter_reliance", "policy_adapter_reliance_requested")), Status: opsDecisionGuardStatus(q, "policy_adapter_reliance", "policy_adapter_reliance_requested")},
			{Label: "runner/worktree protected execution", Field: "runner_worktree_execution_requested", Value: fmt.Sprintf("%t", opsDecisionQueryTrue(q, "runner_worktree_execution", "runner_worktree_execution_requested")), Status: opsDecisionGuardStatus(q, "runner_worktree_execution", "runner_worktree_execution_requested")},
			{Label: "production autonomy", Field: "production_autonomy_requested", Value: fmt.Sprintf("%t", opsDecisionQueryTrue(q, "production_autonomy", "production_autonomy_requested")), Status: opsDecisionGuardStatus(q, "production_autonomy", "production_autonomy_requested")},
			{Label: "ExecutionReceipt production path", Field: "execution_receipt_requested", Value: fmt.Sprintf("%t", opsDecisionQueryTrue(q, "execution_receipt", "execution_receipt_requested")), Status: opsDecisionGuardStatus(q, "execution_receipt", "execution_receipt_requested")},
			{Label: "policy-bundle reliance", Field: "policy_bundle_requested", Value: fmt.Sprintf("%t", opsDecisionQueryTrue(q, "policy_bundle", "policy_bundle_requested")), Status: opsDecisionGuardStatus(q, "policy_bundle", "policy_bundle_requested")},
		},
		BoundaryChecks: []OpsDecisionPosture{
			{Label: "R-001", Field: "runner/worktree evidence gap", Value: "unresolved and excluded", Status: "blocked outside scope"},
			{Label: "R-002", Field: "real protected side effects", Value: "unresolved and excluded", Status: "blocked outside scope"},
			{Label: "R-003", Field: "policy adapter / policy bundle", Value: "unresolved and excluded", Status: "blocked outside scope"},
		},
	}
}

func normalizeOpsDecisionAction(action string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "approve":
		return "approve", true
	case "deny":
		return "deny", true
	case "request-more-evidence", "request_more_evidence":
		return "request-more-evidence", true
	default:
		return "", false
	}
}

func opsDecisionTargetTypeAllowed(targetType string) bool {
	switch strings.ToLower(strings.TrimSpace(targetType)) {
	case "pull_request", "issue", "artifact", "factory_order", "release_candidate":
		return true
	default:
		return false
	}
}

func opsDecisionQueryTrue(q url.Values, keys ...string) bool {
	for _, key := range keys {
		switch strings.ToLower(strings.TrimSpace(q.Get(key))) {
		case "1", "true", "yes", "on", "requested":
			return true
		}
	}
	return false
}

func opsDecisionGuardStatus(q url.Values, keys ...string) string {
	if opsDecisionQueryTrue(q, keys...) {
		return "blocked"
	}
	return "absent"
}

func opsDecisionActions(r *http.Request) []OpsDecisionAction {
	return []OpsDecisionAction{
		{
			Label:       "Approve",
			WireValue:   opsDecisionApprove,
			Description: "Displays an approval envelope for review with effect = none.",
		},
		{
			Label:       "Deny",
			WireValue:   opsDecisionDeny,
			Description: "Displays a denial envelope for review with effect = none.",
		},
		{
			Label:       "Request more evidence",
			WireValue:   opsDecisionMoreEvidence,
			Description: "Displays an evidence request envelope for review with effect = none.",
		},
	}
}

func opsDecisionHash(seed string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))
	return fmt.Sprintf("%012x", h.Sum64())[:12]
}

func fetchOpsEvidence(r *http.Request) *OpsEvidenceData {
	data := &OpsEvidenceData{
		GeneratedAt:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		FactoryOrderID:     strings.TrimSpace(r.URL.Query().Get("factory_order_id")),
		ReleaseCandidateID: strings.TrimSpace(r.URL.Query().Get("release_candidate_id")),
	}
	rawURL := strings.TrimSpace(os.Getenv("DARK_FACTORY_EVIDENCE_PROJECTION_URL"))
	if rawURL == "" {
		data.ProjectionError = "Dark Factory evidence projection URL is not configured."
		return data
	}
	endpoint, err := evidenceProjectionURL(rawURL, data.FactoryOrderID, data.ReleaseCandidateID)
	if err != nil {
		data.ProjectionError = err.Error()
		return data
	}
	data.ProjectionURL = endpoint
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		data.ProjectionError = err.Error()
		return data
	}
	resp, err := evidenceOpsProjectionClient.Do(req)
	if err != nil {
		data.ProjectionError = err.Error()
		return data
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data.ProjectionError = fmt.Sprintf("evidence projection returned %s", resp.Status)
		return data
	}
	var projection OpsEvidenceProjection
	if err := json.NewDecoder(resp.Body).Decode(&projection); err != nil {
		data.ProjectionError = err.Error()
		return data
	}
	data.Source = projection.Source
	if projection.GeneratedAt != "" {
		data.GeneratedAt = formatOpsTime(projection.GeneratedAt)
	}
	data.FactoryOrder = projection.FactoryOrder
	data.ReleaseCandidate = projection.ReleaseCandidate
	data.Decision = projection.Decision
	data.AuditReport = projection.AuditReport
	data.Timeline = projection.Timeline
	data.GateEvidence = projection.GateEvidence
	data.ReleaseEvidence = projection.ReleaseEvidence
	data.FailuresRepairs = projection.FailuresRepairs
	data.MissingProvenance = projection.MissingProvenance
	data.ProofOfWorkPacket = projection.ProofOfWorkPacket
	data.Errors = projection.Errors
	if data.FactoryOrderID == "" && data.FactoryOrder != nil {
		data.FactoryOrderID = data.FactoryOrder.ID
	}
	if data.ReleaseCandidateID == "" && data.ReleaseCandidate != nil {
		data.ReleaseCandidateID = data.ReleaseCandidate.ID
	}
	if len(data.Errors) > 0 {
		data.ProjectionError = strings.Join(data.Errors, "; ")
	}
	return data
}

func evidenceProjectionURL(rawURL, factoryOrderID, releaseCandidateID string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	if factoryOrderID != "" {
		q.Set("factory_order_id", factoryOrderID)
	}
	if releaseCandidateID != "" {
		q.Set("release_candidate_id", releaseCandidateID)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (h *Handlers) fetchOpsHive(r *http.Request) *OpsHiveData {
	ctx := r.Context()
	data := &OpsHiveData{
		GeneratedAt: time.Now().UTC().Format("2006-01-02 15:04:05"),
	}
	applyHiveOperatorProjection(r, data)

	entries, _ := h.store.ListHiveDiagnostics(ctx, maxHiveDiagEntries)
	if len(entries) == 0 {
		entries = readDiagnostics(h.loopDir, maxHiveDiagEntries)
	}
	data.RecentEvents = entries
	data.DiagnosticCount = len(entries)

	ls := readLoopState(h.loopDir)
	data.Iteration = ls.Iteration
	data.Phase = ls.Phase
	data.BuildTitle = ls.BuildTitle
	data.BuildCost = ls.BuildCost

	repoDir := ""
	if h.loopDir != "" {
		repoDir = filepath.Dir(h.loopDir)
	}
	data.RecentCommits = readRecentCommits(repoDir, 6)

	agentID := h.store.GetHiveAgentID(ctx)
	if agentID == "" {
		return data
	}

	totalOps, lastActive, err := h.store.GetHiveTotals(ctx, agentID)
	if err != nil {
		data.Error = err.Error()
		return data
	}
	data.TotalOps = totalOps
	if !lastActive.IsZero() {
		data.LastActive = lastActive.Format("2006-01-02 15:04")
		if data.BuildTitle == "" {
			data.BuildTitle = "Last active: " + data.LastActive
		}
	}

	posts, err := h.store.ListHiveActivity(ctx, agentID, maxHivePosts)
	if err != nil {
		data.Error = err.Error()
		return data
	}
	stats := computeHiveStats(posts)
	data.VerifiedBuilds = stats.Features
	data.TotalCost = stats.TotalCost
	data.AverageCost = stats.AvgCost

	if ic := parseIterFromPosts(posts); ic > data.Iteration {
		data.Iteration = ic
	}
	if data.Iteration == 0 {
		tasks, err := h.store.ListHiveAgentTasks(ctx, agentID, 1000)
		if err != nil {
			data.Error = err.Error()
			return data
		}
		doneCount := 0
		for _, task := range tasks {
			if task.State == "done" {
				doneCount++
			}
		}
		data.Iteration = doneCount
	}
	if data.TotalOps > 0 && data.Phase == "" {
		data.Phase = "idle"
	}
	return data
}

func applyHiveOperatorProjection(r *http.Request, data *OpsHiveData) {
	base := strings.TrimSpace(os.Getenv("HIVE_OPS_API_BASE_URL"))
	if base == "" {
		data.ProjectionError = "Hive operator projection source is not configured."
		return
	}
	endpoint := strings.TrimRight(base, "/") + "/api/hive/operator-projection"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		data.ProjectionError = err.Error()
		return
	}
	if key := strings.TrimSpace(os.Getenv("HIVE_OPS_API_KEY")); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := hiveOpsProjectionClient.Do(req)
	if err != nil {
		data.ProjectionError = err.Error()
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data.ProjectionError = fmt.Sprintf("hive operator projection returned %s", resp.Status)
		return
	}
	var projection OpsHiveProjection
	if err := json.NewDecoder(resp.Body).Decode(&projection); err != nil {
		data.ProjectionError = err.Error()
		return
	}
	data.ProjectionSource = projection.Source
	if projection.GeneratedAt != "" {
		data.GeneratedAt = formatOpsTime(projection.GeneratedAt)
	}
	if len(projection.Errors) > 0 {
		data.ProjectionError = strings.Join(projection.Errors, "; ")
	}
	data.PendingApprovals = projection.PendingApprovals
	data.AuthorityDecisions = projection.AuthorityDecisions
	data.Lifecycle = projection.Lifecycle
	data.KeyAuditTraces = projection.KeyAuditTraces
}

// fetchHiveOperatorProjection fetches and decodes the hive operator projection.
// Returns the projection or an error; it does not touch any OpsHiveData.
func fetchHiveOperatorProjection(r *http.Request) (*OpsHiveProjection, error) {
	base := strings.TrimSpace(os.Getenv("HIVE_OPS_API_BASE_URL"))
	if base == "" {
		return nil, errors.New("HIVE_OPS_API_BASE_URL is not configured")
	}
	endpoint := strings.TrimRight(base, "/") + "/api/hive/operator-projection"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
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
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("hive operator projection returned %s", resp.Status)
	}
	var projection OpsHiveProjection
	if err := json.NewDecoder(resp.Body).Decode(&projection); err != nil {
		return nil, err
	}
	return &projection, nil
}

// opsCanonicalRoles is the fixed civic-role order shown in the society view.
// The last two rows ("human (Site approval)" and "draft PR") are synthetic: they
// are not lifecycle actors but represent the Gate E approval and the resulting
// draft PR action that flows from an approved pull_request.create decision.
var opsCanonicalRoles = []string{
	"strategist",
	"planner",
	"implementer",
	"reviewer",
	"guardian",
	"human (Site approval)",
	"draft PR",
}

// buildOpsRoleTimeline maps the hive operator projection's Lifecycle entries and
// AuthorityDecisions into a role-grouped timeline in canonical civic order.
//
// Sources (read-only, no new external calls):
//   - projection.Lifecycle — one entry per actor; actor.Role normalised to lower-case.
//   - projection.AuthorityDecisions — for the "human (Site approval)" and "draft PR" rows.
//
// The "human (Site approval)" row is populated when any decision has
// outcome == "approved" and approved_action == "pull_request.create".
// The "draft PR" row immediately follows: its status is "pending" unless an
// approved pull_request.create decision exists, in which case it is "approved".
func buildOpsRoleTimeline(projection *OpsHiveProjection) []OpsRoleTimelineRow {
	if projection == nil {
		// Return all canonical rows in empty state.
		rows := make([]OpsRoleTimelineRow, len(opsCanonicalRoles))
		for i, role := range opsCanonicalRoles {
			rows[i] = OpsRoleTimelineRow{Role: role}
		}
		return rows
	}

	// Index lifecycle entries by normalised role name (lower-case).
	// Multiple actors can share a role; collect their display names.
	type actorSummary struct {
		names    []string
		statuses []string
	}
	byRole := make(map[string]*actorSummary)
	for _, lc := range projection.Lifecycle {
		role := strings.ToLower(strings.TrimSpace(lc.Role))
		if role == "" {
			continue
		}
		if _, ok := byRole[role]; !ok {
			byRole[role] = &actorSummary{}
		}
		name := lc.DisplayName
		if name == "" {
			name = lc.ActorID
		}
		byRole[role].names = append(byRole[role].names, name)
		byRole[role].statuses = append(byRole[role].statuses, lc.LifecycleStatus)
	}

	// Look for an approved pull_request.create decision.
	var prDecision *OpsHiveDecision
	for i := range projection.AuthorityDecisions {
		d := &projection.AuthorityDecisions[i]
		if strings.ToLower(d.Outcome) == "approved" &&
			strings.ToLower(d.ApprovedAction) == "pull_request.create" {
			prDecision = d
			break
		}
	}

	rows := make([]OpsRoleTimelineRow, 0, len(opsCanonicalRoles))
	for _, canonicalRole := range opsCanonicalRoles {
		row := OpsRoleTimelineRow{Role: canonicalRole}

		switch canonicalRole {
		case "human (Site approval)":
			if prDecision != nil {
				row.Status = prDecision.Outcome
				row.Actors = prDecision.ApproverActor
				row.Notes = "approved_action: " + prDecision.ApprovedAction
				if prDecision.ApprovedTarget != "" {
					row.Notes += " → " + prDecision.ApprovedTarget
				}
			}

		case "draft PR":
			if prDecision != nil {
				row.Status = "approved"
				row.Notes = "pull_request.create approved — draft PR queued"
			}

		default:
			// Match against lifecycle entries using the canonical role label.
			// The canonical label IS the normalised role key (all lower-case).
			if summary, ok := byRole[canonicalRole]; ok && len(summary.names) > 0 {
				row.Actors = strings.Join(summary.names, ", ")
				// Use the first (or only) lifecycle_status; if multiple,
				// prefer "active" > anything else.
				row.Status = summary.statuses[0]
				for _, s := range summary.statuses {
					if strings.ToLower(s) == "active" {
						row.Status = s
						break
					}
				}
			}
		}

		rows = append(rows, row)
	}
	return rows
}

func fetchOpsTelemetry(r *http.Request) *OpsTelemetryData {
	workBase := serverWorkAPIBaseURL()
	data := &OpsTelemetryData{
		WorkURL:     legacyWorkURL(workBase, "/telemetry/overview"),
		PipelineURL: legacyWorkURL(workBase, "/telemetry/pipeline/report"),
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, data.WorkURL, nil)
	if err != nil {
		data.Error = err.Error()
		return data
	}
	setWorkAuth(req)
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
	if report, err := fetchOpsPipelineReport(r, data.PipelineURL); err != nil {
		data.PipelineError = err.Error()
	} else {
		data.Pipeline = report
	}
	return data
}

func fetchOpsPipelineReport(r *http.Request, reportURL string) (*OpsPipelineReport, error) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, reportURL, nil)
	if err != nil {
		return nil, err
	}
	setWorkAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("work pipeline report returned %s", resp.Status)
	}
	var payload opsPipelineReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Report == nil {
		if payload.Detail != "" {
			return nil, errors.New(payload.Detail)
		}
		if payload.Error != "" {
			return nil, errors.New(payload.Error)
		}
		return nil, nil
	}
	return payload.Report, nil
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
	setWorkAuth(req)
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
		if task.Ready {
			data.Ready++
		}
	}
	data.RecentTasks = takeWorkTasks(tasks.Tasks, 10)
	data.BlockedTasks = takeWorkTasks(data.BlockedTasks, 6)
	data.PhaseGates = fetchOpsPhaseGates(r, workBase)
	return data
}

func fetchOpsPhaseGates(r *http.Request, workBase string) []OpsPhaseGate {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, legacyWorkURL(workBase, "/phase-gates"), nil)
	if err != nil {
		return nil
	}
	setWorkAuth(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil
	}
	var gates opsPhaseGatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&gates); err != nil {
		return nil
	}
	if len(gates.Gates) > 6 {
		return gates.Gates[:6]
	}
	return gates.Gates
}

func setWorkAuth(req *http.Request) {
	if key := strings.TrimSpace(os.Getenv("WORK_API_KEY")); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
		return
	}
	req.Header.Set("Authorization", "Bearer dev")
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

func formatOpsDuration(seconds float64) string {
	if seconds <= 0 {
		return "0s"
	}
	if seconds < 60 {
		return fmt.Sprintf("%.0fs", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%.1fm", seconds/60)
	}
	return fmt.Sprintf("%.1fh", seconds/3600)
}

func opsPipelineOutcomeClass(outcome string) string {
	switch {
	case strings.Contains(strings.ToLower(outcome), "pass"), strings.Contains(strings.ToLower(outcome), "done"):
		return "border-emerald-400/30 text-emerald-300 bg-emerald-400/10"
	case strings.Contains(strings.ToLower(outcome), "revise"), strings.Contains(strings.ToLower(outcome), "fail"), strings.Contains(strings.ToLower(outcome), "error"):
		return "border-red-400/30 text-red-300 bg-red-400/10"
	case strings.Contains(strings.ToLower(outcome), "progress"):
		return "border-sky-400/30 text-sky-300 bg-sky-400/10"
	default:
		return "border-brand/30 text-brand bg-brand/10"
	}
}

func opsPhaseGateStatusClass(status string) string {
	switch strings.ToLower(status) {
	case "approved":
		return "border-emerald-400/30 text-emerald-300 bg-emerald-400/10"
	case "rejected":
		return "border-red-400/30 text-red-300 bg-red-400/10"
	default:
		return "border-amber-400/30 text-amber-300 bg-amber-400/10"
	}
}

func opsDecisionStatusClass(status string) string {
	switch strings.ToLower(status) {
	case "accepted_for_review", "absent":
		return "border-emerald-400/30 text-emerald-300 bg-emerald-400/10"
	case "blocked", "blocked outside scope":
		return "border-amber-400/30 text-amber-300 bg-amber-400/10"
	default:
		return "border-brand/30 text-brand bg-brand/10"
	}
}

func opsHiveShellStatusClass(status string) string {
	switch strings.ToLower(status) {
	case "clear", "ready", "draft ready", "active", "watching":
		return "border-emerald-400/30 text-emerald-300 bg-emerald-400/10"
	case "attention", "warning", "watch", "waiting":
		return "border-amber-400/30 text-amber-300 bg-amber-400/10"
	case "blocked", "paused", "missing":
		return "border-red-400/30 text-red-300 bg-red-400/10"
	case "idle":
		return "border-edge text-warm-faint bg-void/30"
	default:
		return "border-brand/30 text-brand bg-brand/10"
	}
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

func legacyWorkURL(base, path string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	u.Path = path
	u.RawQuery = ""
	return u.String()
}
