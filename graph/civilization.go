package graph

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	opsCivilizationProjectionStatusComplete    = "complete"
	opsCivilizationProjectionStatusPartial     = "partial"
	opsCivilizationProjectionStatusUnavailable = "unavailable"
	opsCivilizationProjectionStatusFailed      = "failed"

	opsCivilizationFieldAvailable   = "available"
	opsCivilizationFieldUnavailable = "unavailable"

	opsCivilizationProjectionStaleAfter = 24 * time.Hour
	opsCivilizationProjectionFutureSkew = 5 * time.Minute

	opsCivilizationIssueScanStageArtifactPrefix = "issue_scan_lifecycle_stage_"
)

type OpsCivilizationAssemblyData struct {
	GeneratedAt            string
	AuthoritySource        string
	ProjectionSource       string
	ProjectionTarget       string
	ProjectionStatus       string
	ProjectionFreshness    string
	Civilization           ObsCivilization
	Boundary               []OpsCivilizationBoundary
	StatusRows             []OpsCivilizationStatusRow
	ReferenceGroups        []OpsCivilizationReferenceGroup
	IssueIntake            OpsCivilizationIssueIntake
	IssueReadiness         OpsCivilizationIssueReadiness
	FactoryOrders          []OpsCivilizationAssemblyFactoryOrder
	WorkEvidence           OpsCivilizationAssemblyWorkEvidence
	QueuedRunRequest       *OpsHiveQueuedRunRequest
	IssueScanStageEvidence []OpsCivilizationIssueScanStageEvidence
	IssueScanKanban        OpsCivilizationIssueScanKanban
}

type OpsCivilizationBoundary struct {
	Label  string
	State  string
	Detail string
}

type OpsCivilizationStatusRow struct {
	Label string
	Value string
}

type OpsCivilizationReferenceGroup struct {
	Label string
	Refs  []string
}

type OpsCivilizationIssueReadiness struct {
	Status              string
	PRReadyWhen         string
	FirstPendingStage   string
	RecommendationState string
	GroupingSummary     string
	GroupingInputs      []string
	SourceRefs          []string
	Guardrails          []OpsCivilizationIssueGuardrail
}

type OpsCivilizationIssueGuardrail struct {
	Label  string
	State  string
	Detail string
}

type OpsCivilizationAssemblyProjection struct {
	ProjectionID                       string                                    `json:"projection_id"`
	ProjectionSchemaVersion            string                                    `json:"projection_schema_version"`
	ProjectionSubject                  string                                    `json:"projection_subject"`
	GeneratedAt                        time.Time                                 `json:"generated_at"`
	SourceEventGraphHeadOrStateVersion string                                    `json:"source_eventgraph_head_or_state_version"`
	SourceEventIDsOrQueryWindow        []string                                  `json:"source_event_ids_or_query_window"`
	DerivationStatus                   string                                    `json:"derivation_status"`
	AuthorityState                     OpsCivilizationAssemblyAuthorityState     `json:"authority_state"`
	ExternalCommitteeState             OpsCivilizationAssemblyCommitteeState     `json:"external_committee_state"`
	ActorRoster                        []OpsCivilizationAssemblyActorSummary     `json:"actor_roster"`
	RoleBindings                       []OpsCivilizationAssemblyRoleBinding      `json:"role_bindings"`
	AgentLifecycleSummary              []OpsCivilizationAssemblyLifecycleSummary `json:"agent_lifecycle_summary"`
	FactoryOrderSummary                []OpsCivilizationAssemblyFactoryOrder     `json:"factory_order_summary"`
	WorkEvidenceSummary                OpsCivilizationAssemblyWorkEvidence       `json:"work_evidence_summary"`
	QueuedRunRequest                   *OpsHiveQueuedRunRequest                  `json:"queued_run_request,omitempty"`
	IssueIntakeProjection              OpsCivilizationIssueIntakeProjection      `json:"issue_intake_projection,omitempty"`
	IssueScanProjection                OpsCivilizationIssueScanProjection        `json:"issue_scan_projection,omitempty"`
	SiteConsumerStatus                 OpsCivilizationAssemblyFieldStatus        `json:"site_consumer_status"`
	OpenGateSummary                    []OpsCivilizationAssemblyGateSummary      `json:"open_gate_summary"`
	ResidualRiskSummary                []OpsCivilizationAssemblyResidualRisk     `json:"residual_risk_summary"`
	WithheldOrUnavailableFields        []OpsCivilizationAssemblyUnavailableField `json:"withheld_or_unavailable_fields"`
	BoundaryFlags                      []string                                  `json:"boundary_flags"`
	ProvenanceRefs                     []string                                  `json:"provenance_refs"`
	ValidationRefs                     []string                                  `json:"validation_refs"`
	FailureReasons                     []string                                  `json:"failure_reasons,omitempty"`
}

type OpsCivilizationAssemblyFieldStatus struct {
	Status     string   `json:"status"`
	Summary    string   `json:"summary"`
	SourceRefs []string `json:"source_refs,omitempty"`
}

type OpsCivilizationAssemblyUnavailableField struct {
	Field  string `json:"field"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type OpsCivilizationAssemblyAuthorityState struct {
	Status             string                                     `json:"status"`
	Summary            string                                     `json:"summary"`
	AuthorityRequests  []OpsCivilizationAssemblyAuthorityRequest  `json:"authority_requests,omitempty"`
	AuthorityDecisions []OpsCivilizationAssemblyAuthorityDecision `json:"authority_decisions,omitempty"`
	ExecutionReceipts  []OpsCivilizationAssemblyExecutionReceipt  `json:"execution_receipts,omitempty"`
	SourceRefs         []string                                   `json:"source_refs,omitempty"`
}

type OpsCivilizationAssemblyAuthorityRequest struct {
	ID         string `json:"id"`
	ActorID    string `json:"actor_id"`
	ActorRole  string `json:"actor_role"`
	Action     string `json:"action"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	RiskClass  string `json:"risk_class"`
	Status     string `json:"status,omitempty"`
}

type OpsCivilizationAssemblyAuthorityDecision struct {
	ID                 string   `json:"id"`
	AuthorityRequestID string   `json:"authority_request_id"`
	DeciderActorID     string   `json:"decider_actor_id"`
	DeciderRole        string   `json:"decider_role"`
	Decision           string   `json:"decision"`
	Status             string   `json:"status,omitempty"`
	Scope              []string `json:"scope,omitempty"`
}

type OpsCivilizationAssemblyExecutionReceipt struct {
	ID                  string `json:"id"`
	AuthorityDecisionID string `json:"authority_decision_id"`
	Action              string `json:"action"`
	TargetID            string `json:"target_id"`
	Result              string `json:"result"`
	Status              string `json:"status,omitempty"`
}

type OpsCivilizationAssemblyCommitteeState struct {
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	DecisionRefs   []string `json:"decision_refs,omitempty"`
	ApprovalRefs   []string `json:"approval_refs,omitempty"`
	CommitteeRoles []string `json:"committee_roles,omitempty"`
}

type OpsCivilizationAssemblyActorSummary struct {
	ID           string `json:"id"`
	ActorID      string `json:"actor_id"`
	ActorType    string `json:"actor_type"`
	IdentityMode string `json:"identity_mode"`
	Status       string `json:"status,omitempty"`
}

type OpsCivilizationAssemblyRoleBinding struct {
	ActorID    string `json:"actor_id"`
	Role       string `json:"role"`
	SourceRef  string `json:"source_ref"`
	SourceType string `json:"source_type"`
}

type OpsCivilizationAssemblyLifecycleSummary struct {
	ID                  string  `json:"id"`
	ActorID             string  `json:"actor_id"`
	FromState           string  `json:"from_state,omitempty"`
	ToState             string  `json:"to_state,omitempty"`
	TrustLevel          string  `json:"trust_level,omitempty"`
	AuthorityDecisionID *string `json:"authority_decision_id,omitempty"`
	Status              string  `json:"status,omitempty"`
}

type OpsCivilizationAssemblyFactoryOrder struct {
	ID                      string   `json:"id"`
	Status                  string   `json:"status,omitempty"`
	RiskClass               string   `json:"risk_class"`
	ReleasePolicy           string   `json:"release_policy"`
	RequirementRefs         []string `json:"requirement_refs,omitempty"`
	AcceptanceCriterionRefs []string `json:"acceptance_criterion_refs,omitempty"`
	TaskRefs                []string `json:"task_refs,omitempty"`
	ReleaseCandidateRefs    []string `json:"release_candidate_refs,omitempty"`
}

type OpsCivilizationAssemblyWorkEvidence struct {
	Status          string                                    `json:"status"`
	Summary         string                                    `json:"summary"`
	TaskRefs        []string                                  `json:"task_refs,omitempty"`
	Tasks           []OpsCivilizationAssemblyTaskEvidence     `json:"tasks,omitempty"`
	ArtifactRefs    []string                                  `json:"artifact_refs,omitempty"`
	Artifacts       []OpsCivilizationAssemblyArtifactEvidence `json:"artifacts,omitempty"`
	TestRunRefs     []string                                  `json:"test_run_refs,omitempty"`
	GateResultRefs  []string                                  `json:"gate_result_refs,omitempty"`
	AuditReportRefs []string                                  `json:"audit_report_refs,omitempty"`
	SourceRefs      []string                                  `json:"source_refs,omitempty"`
}

type OpsCivilizationAssemblyTaskEvidence struct {
	ID                      string                              `json:"id"`
	CanonicalTaskID         string                              `json:"canonical_task_id,omitempty"`
	FactoryOrderID          string                              `json:"factory_order_id,omitempty"`
	LifecycleStageID        string                              `json:"lifecycle_stage_id,omitempty"`
	Title                   string                              `json:"title"`
	Cell                    string                              `json:"cell,omitempty"`
	RiskClass               string                              `json:"risk_class,omitempty"`
	Status                  string                              `json:"status"`
	Ready                   bool                                `json:"ready"`
	Blocked                 bool                                `json:"blocked"`
	RequirementRefs         []string                            `json:"requirement_refs,omitempty"`
	AcceptanceCriterionRefs []string                            `json:"acceptance_criterion_refs,omitempty"`
	ExpectedOutputs         []string                            `json:"expected_outputs,omitempty"`
	DependsOnRefs           []string                            `json:"depends_on_refs,omitempty"`
	SourceRefs              []string                            `json:"source_refs,omitempty"`
	RequiredRoles           []string                            `json:"required_roles,omitempty"`
	RoleContractRefs        []string                            `json:"role_contract_refs,omitempty"`
	AgentExecutionPlan      []OpsHiveQueuedRunAgentPlanStep     `json:"agent_execution_plan,omitempty"`
	RequiredEvidence        []string                            `json:"required_evidence,omitempty"`
	OutputContractRefs      []string                            `json:"output_contract_refs,omitempty"`
	RuntimeEvidenceRefs     []string                            `json:"runtime_evidence_refs,omitempty"`
	RuntimeEvidenceStatus   string                              `json:"runtime_evidence_status,omitempty"`
	RoleOutputContracts     []OpsCivilizationRoleOutputContract `json:"role_output_contracts,omitempty"`
}

type OpsCivilizationRoleOutputContract struct {
	Role              string   `json:"role"`
	CanOperate        bool     `json:"can_operate"`
	RequiredOutputs   []string `json:"required_outputs,omitempty"`
	AuthorityBoundary string   `json:"authority_boundary,omitempty"`
	CompletionGate    string   `json:"completion_gate,omitempty"`
	EvidenceStatus    string   `json:"evidence_status,omitempty"`
}

type OpsCivilizationAssemblyArtifactEvidence struct {
	ID         string   `json:"id"`
	TaskRef    string   `json:"task_ref"`
	Label      string   `json:"label"`
	MediaType  string   `json:"media_type,omitempty"`
	SourceRefs []string `json:"source_refs,omitempty"`
}

type OpsCivilizationIssueScanStageEvidence struct {
	StageID        string
	StageName      string
	ArtifactID     string
	Label          string
	MediaType      string
	TaskRef        string
	SourceRefs     []string
	EvidenceStatus string
}

type OpsCivilizationAssemblyGateSummary struct {
	ID                 string   `json:"id"`
	GateName           string   `json:"gate_name"`
	Status             string   `json:"status,omitempty"`
	FactoryOrderID     string   `json:"factory_order_id,omitempty"`
	ReleaseCandidateID *string  `json:"release_candidate_id,omitempty"`
	EvidenceRefs       []string `json:"evidence_refs,omitempty"`
}

type OpsCivilizationAssemblyResidualRisk struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Severity string `json:"severity,omitempty"`
	Status   string `json:"status,omitempty"`
	Summary  string `json:"summary"`
}

func buildOpsCivilizationAssemblyData() *OpsCivilizationAssemblyData {
	return buildOpsCivilizationAssemblyDataFromProjection(nil, time.Now().UTC())
}

func buildOpsCivilizationAssemblyDataFromProjection(projection *OpsCivilizationAssemblyProjection, now time.Time) *OpsCivilizationAssemblyData {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	status := opsCivilizationProjectionStatus(projection)
	freshness := opsCivilizationProjectionFreshness(projection, now)
	civ := opsCivilizationFromProjection(projection, status, freshness)

	return &OpsCivilizationAssemblyData{
		GeneratedAt:            now.UTC().Format("2006-01-02 15:04:05 UTC"),
		AuthoritySource:        "docs v4.0 Event 10 Site Civilization projection-consumer AuthorityDecision",
		ProjectionSource:       opsCivilizationProjectionSource(projection),
		ProjectionTarget:       "EventGraph Civilization Assembly projection",
		ProjectionStatus:       status,
		ProjectionFreshness:    freshness,
		Civilization:           civ,
		Boundary:               opsCivilizationBoundary(projection, status, freshness),
		StatusRows:             opsCivilizationStatusRows(projection, status, freshness),
		ReferenceGroups:        opsCivilizationReferenceGroups(projection),
		IssueIntake:            opsCivilizationIssueIntake(projection),
		IssueReadiness:         opsCivilizationIssueReadiness(projection),
		FactoryOrders:          opsCivilizationFactoryOrders(projection),
		WorkEvidence:           opsCivilizationWorkEvidence(projection),
		QueuedRunRequest:       opsCivilizationQueuedRunRequest(projection),
		IssueScanStageEvidence: opsCivilizationIssueScanStageEvidence(projection),
		IssueScanKanban:        opsCivilizationIssueScanKanban(projection),
	}
}

func opsCivilizationProjectionStatus(projection *OpsCivilizationAssemblyProjection) string {
	if projection == nil {
		return opsCivilizationProjectionStatusUnavailable
	}
	switch strings.ToLower(strings.TrimSpace(projection.DerivationStatus)) {
	case opsCivilizationProjectionStatusComplete:
		return opsCivilizationProjectionStatusComplete
	case opsCivilizationProjectionStatusPartial:
		return opsCivilizationProjectionStatusPartial
	case opsCivilizationProjectionStatusFailed:
		return opsCivilizationProjectionStatusFailed
	case opsCivilizationProjectionStatusUnavailable, "":
		return opsCivilizationProjectionStatusUnavailable
	default:
		return strings.TrimSpace(projection.DerivationStatus)
	}
}

func opsCivilizationProjectionFreshness(projection *OpsCivilizationAssemblyProjection, now time.Time) string {
	if projection == nil || projection.GeneratedAt.IsZero() {
		return "unknown"
	}
	generatedAt := projection.GeneratedAt.UTC()
	now = now.UTC()
	if generatedAt.After(now.Add(opsCivilizationProjectionFutureSkew)) {
		return "skewed"
	}
	if now.Sub(generatedAt) > opsCivilizationProjectionStaleAfter {
		return "stale"
	}
	return "current"
}

func opsCivilizationProjectionSource(projection *OpsCivilizationAssemblyProjection) string {
	if projection == nil {
		return "EventGraph Civilization Assembly projection unavailable to Site"
	}
	return "EventGraph Civilization Assembly projection " + opsCivilizationValue(projection.ProjectionID, "projection id unavailable")
}

func opsCivilizationFromProjection(projection *OpsCivilizationAssemblyProjection, status string, freshness string) ObsCivilization {
	if projection == nil {
		return opsCivilizationUnavailable("EventGraph projection-shaped input was not available; Site rendered no inferred runtime truth.")
	}

	civ := ObsCivilization{
		OrgLevels:            opsCivilizationProjectionOrgLevels(false),
		Roster:               opsCivilizationRoster(projection),
		Emergence:            opsCivilizationEmergence(projection),
		GlobalModelMode:      "unknown",
		GlobalModeProvenance: "not projected",
		GlobalModeReason:     "Model Selection Mode is not a field in the Civilization Assembly projection.",
		ModelSource:          "not projected by Civilization Assembly projection",
	}
	if len(civ.Roster) > 0 {
		civ.OrgLevels = opsCivilizationProjectionOrgLevels(true)
	}
	civ.Findings = opsCivilizationFindings(projection, status, freshness)
	return civ
}

func opsCivilizationUnavailable(reason string) ObsCivilization {
	return ObsCivilization{
		OrgLevels:            opsCivilizationProjectionOrgLevels(false),
		GlobalModelMode:      "unknown",
		GlobalModeProvenance: "not projected",
		GlobalModeReason:     "EventGraph Civilization Assembly projection unavailable",
		ModelSource:          "EventGraph Civilization Assembly projection unavailable",
		Emergence: []ObsEmergenceStep{
			{
				Subject:  "Civilization Assembly projection",
				State:    opsCivilizationProjectionStatusUnavailable,
				Why:      "Site has no projection input to consume for this render.",
				Evidence: "EventGraph projection-shaped input missing",
			},
		},
		Findings: []string{
			reason,
			"Missing projection data does not imply authority, readiness, Gate T closeout, runtime execution, deployment, autonomy increase, or value allocation.",
			"This route is display-only and carries no authority to execute, deploy, mutate protected settings, or allocate value.",
		},
	}
}

func opsCivilizationProjectionOrgLevels(hasRoster bool) []ObsOrgLevel {
	state := "not projected"
	detail := "tier evidence is unavailable in the current Civilization Assembly projection input"
	if hasRoster {
		state = "role bindings projected"
		detail = "role bindings are projected; explicit tier evidence remains unavailable unless supplied by a later projection field"
	}
	return []ObsOrgLevel{
		{Tier: "A", Label: "Bootstrap / foundation", Used: state, Detail: detail},
		{Tier: "B", Label: "Organic emergence", Used: "not projected", Detail: detail},
		{Tier: "C", Label: "Business operations", Used: "not projected", Detail: detail},
		{Tier: "D", Label: "Self-governance", Used: "not projected", Detail: detail},
	}
}

func opsCivilizationRoster(projection *OpsCivilizationAssemblyProjection) []ObsCivilizationRole {
	actors := map[string]OpsCivilizationAssemblyActorSummary{}
	for _, actor := range projection.ActorRoster {
		actorID := strings.TrimSpace(actor.ActorID)
		if actorID != "" {
			actors[actorID] = actor
		}
	}

	rows := make([]ObsCivilizationRole, 0, len(projection.RoleBindings)+len(projection.ActorRoster))
	seenActors := map[string]bool{}
	seenRows := map[string]bool{}
	for _, binding := range projection.RoleBindings {
		actorID := strings.TrimSpace(binding.ActorID)
		role := strings.TrimSpace(binding.Role)
		if actorID == "" && role == "" {
			continue
		}
		key := actorID + "\x00" + role + "\x00" + binding.SourceRef
		if seenRows[key] {
			continue
		}
		seenRows[key] = true
		seenActors[actorID] = true
		actor := actors[actorID]
		rows = append(rows, ObsCivilizationRole{
			Role:                opsCivilizationValue(role, "role not projected"),
			Agent:               opsCivilizationValue(actorID, "actor not projected"),
			Tier:                "not projected",
			Category:            opsCivilizationValue(actor.ActorType, "EventGraph role binding"),
			Origin:              opsCivilizationValue(binding.SourceType, "EventGraph role binding"),
			Status:              opsCivilizationValue(actor.Status, "projected"),
			CanOperate:          "not projected",
			Model:               "not projected",
			ModelMode:           "unknown",
			ModelModeProvenance: "not projected",
			ReportsTo:           "not projected",
			EscalationPath:      "not projected",
			Why:                 "role binding supplied by the EventGraph Civilization Assembly projection",
			Evidence:            opsCivilizationValue(binding.SourceRef, "source ref unavailable"),
		})
	}
	for _, actor := range projection.ActorRoster {
		actorID := strings.TrimSpace(actor.ActorID)
		if actorID == "" || seenActors[actorID] {
			continue
		}
		rows = append(rows, ObsCivilizationRole{
			Role:                "role not projected",
			Agent:               actorID,
			Tier:                "not projected",
			Category:            opsCivilizationValue(actor.ActorType, "actor"),
			Origin:              "EventGraph actor roster",
			Status:              opsCivilizationValue(actor.Status, "projected"),
			CanOperate:          "not projected",
			Model:               "not projected",
			ModelMode:           "unknown",
			ModelModeProvenance: "not projected",
			ReportsTo:           "not projected",
			EscalationPath:      "not projected",
			Why:                 "actor was present in the projection without an accompanying role binding",
			Evidence:            opsCivilizationValue(actor.ID, "source ref unavailable"),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Role == rows[j].Role {
			return rows[i].Agent < rows[j].Agent
		}
		return rows[i].Role < rows[j].Role
	})
	return rows
}

func opsCivilizationEmergence(projection *OpsCivilizationAssemblyProjection) []ObsEmergenceStep {
	steps := make([]ObsEmergenceStep, 0, len(projection.AgentLifecycleSummary)+len(projection.OpenGateSummary)+len(projection.ResidualRiskSummary))
	for _, item := range projection.AgentLifecycleSummary {
		stateParts := []string{}
		if item.FromState != "" || item.ToState != "" {
			stateParts = append(stateParts, opsCivilizationValue(item.FromState, "unknown")+" -> "+opsCivilizationValue(item.ToState, "unknown"))
		}
		if item.TrustLevel != "" {
			stateParts = append(stateParts, "trust "+item.TrustLevel)
		}
		steps = append(steps, ObsEmergenceStep{
			Subject:  opsCivilizationValue(item.ActorID, item.ID),
			State:    opsCivilizationValue(strings.Join(stateParts, "; "), opsCivilizationValue(item.Status, "projected")),
			Why:      "agent lifecycle state supplied by the EventGraph Civilization Assembly projection",
			Evidence: opsCivilizationValue(item.ID, "source ref unavailable"),
		})
	}
	for _, gate := range projection.OpenGateSummary {
		steps = append(steps, ObsEmergenceStep{
			Subject:  opsCivilizationValue(gate.GateName, gate.ID),
			State:    opsCivilizationValue(gate.Status, "open gate projected"),
			Why:      "open gate is projected and is not closed by Site rendering",
			Evidence: opsCivilizationValue(gate.ID, "gate ref unavailable"),
		})
	}
	for _, risk := range projection.ResidualRiskSummary {
		steps = append(steps, ObsEmergenceStep{
			Subject:  opsCivilizationValue(risk.ID, risk.Kind),
			State:    opsCivilizationValue(risk.Status, "residual risk projected"),
			Why:      opsCivilizationValue(risk.Summary, "residual risk remains projected"),
			Evidence: opsCivilizationValue(risk.Severity, "severity not projected"),
		})
	}
	sort.Slice(steps, func(i, j int) bool {
		if steps[i].State == steps[j].State {
			return steps[i].Subject < steps[j].Subject
		}
		return steps[i].State < steps[j].State
	})
	return steps
}

func opsCivilizationFindings(projection *OpsCivilizationAssemblyProjection, status string, freshness string) []string {
	findings := []string{
		"EventGraph Civilization Assembly projection derivation status: " + status + ".",
		"Projection freshness: " + freshness + ".",
		"Even a complete projection derivation does not close Gate T, approve production readiness, execute runtime work, deploy, mutate protected settings, increase autonomy, or allocate value.",
		"This route is display-only and carries no authority to execute, deploy, mutate protected settings, or allocate value.",
	}
	if projection.AuthorityState.Summary != "" {
		findings = append(findings, "Authority state: "+projection.AuthorityState.Summary)
	}
	if projection.ExternalCommitteeState.Summary != "" {
		findings = append(findings, "External Committee state: "+projection.ExternalCommitteeState.Summary)
	}
	if projection.WorkEvidenceSummary.Summary != "" {
		findings = append(findings, "Work evidence: "+projection.WorkEvidenceSummary.Summary)
	}
	if projection.SiteConsumerStatus.Summary != "" {
		findings = append(findings, "Site consumer: "+projection.SiteConsumerStatus.Summary)
	}
	for _, field := range projection.WithheldOrUnavailableFields {
		findings = append(findings, fmt.Sprintf("Unavailable projection field %s: %s", opsCivilizationValue(field.Field, "unknown"), opsCivilizationValue(field.Reason, "reason not projected")))
	}
	for _, reason := range projection.FailureReasons {
		if reason != "" {
			findings = append(findings, "Projection failure reason: "+reason)
		}
	}
	if len(projection.OpenGateSummary) > 0 {
		findings = append(findings, fmt.Sprintf("%d open gate(s) remain projected; Site rendering does not close them.", len(projection.OpenGateSummary)))
	}
	if len(projection.ResidualRiskSummary) > 0 {
		findings = append(findings, fmt.Sprintf("%d residual risk item(s) remain projected; Site rendering does not dispose them.", len(projection.ResidualRiskSummary)))
	}
	return findings
}

func opsCivilizationBoundary(projection *OpsCivilizationAssemblyProjection, status string, freshness string) []OpsCivilizationBoundary {
	boundary := []OpsCivilizationBoundary{
		{
			Label:  "Route authority",
			State:  "bounded",
			Detail: "One read-only Site consumer surface authorized by the merged v4.0 Event 10 packet.",
		},
		{
			Label:  "Registered method",
			State:  "GET only",
			Detail: "No mutation handler is registered for this page.",
		},
		{
			Label:  "Projection derivation",
			State:  status,
			Detail: "Site renders the projection status as supplied; it does not infer missing approval or readiness.",
		},
		{
			Label:  "Projection freshness",
			State:  freshness,
			Detail: "Stale or absent projection input remains advisory display data, not authority.",
		},
		{
			Label:  "Runtime control",
			State:  "withheld",
			Detail: "No executor, deploy, protected-setting, Hive-write, Work-mutation, autonomy, or value path is exposed.",
		},
	}
	if projection != nil {
		boundary = append(boundary,
			OpsCivilizationBoundary{
				Label:  "Authority state",
				State:  opsCivilizationStatusValue(projection.AuthorityState.Status),
				Detail: opsCivilizationValue(projection.AuthorityState.Summary, "authority summary unavailable"),
			},
			OpsCivilizationBoundary{
				Label:  "External Committee",
				State:  opsCivilizationStatusValue(projection.ExternalCommitteeState.Status),
				Detail: opsCivilizationValue(projection.ExternalCommitteeState.Summary, "committee summary unavailable"),
			},
			OpsCivilizationBoundary{
				Label:  "Site consumer",
				State:  opsCivilizationStatusValue(projection.SiteConsumerStatus.Status),
				Detail: opsCivilizationValue(projection.SiteConsumerStatus.Summary, "site consumer evidence unavailable"),
			},
		)
	}
	return boundary
}

func opsCivilizationStatusRows(projection *OpsCivilizationAssemblyProjection, status string, freshness string) []OpsCivilizationStatusRow {
	rows := make([]OpsCivilizationStatusRow, 0, 12)
	add := func(label string, value string) {
		rows = append(rows, OpsCivilizationStatusRow{Label: label, Value: value})
	}
	if projection == nil {
		add("projection id", "not projected")
		add("schema version", "not projected")
		add("subject", "civilization_assembly")
		add("source state", "not projected")
		add("source events/window", "not projected")
		add("projection generated", "not projected")
		add("freshness", freshness)
		add("derivation status", status)
		add("authority state", "unavailable")
		add("external committee", "unavailable")
		add("work evidence", "unavailable")
		add("site consumer", "unavailable")
		return rows
	}
	add("projection id", opsCivilizationValue(projection.ProjectionID, "not projected"))
	add("schema version", opsCivilizationValue(projection.ProjectionSchemaVersion, "not projected"))
	add("subject", opsCivilizationValue(projection.ProjectionSubject, "not projected"))
	add("source state", opsCivilizationValue(projection.SourceEventGraphHeadOrStateVersion, "not projected"))
	add("source events/window", opsCivilizationJoin(projection.SourceEventIDsOrQueryWindow, "not projected"))
	add("projection generated", opsCivilizationTime(projection.GeneratedAt))
	add("freshness", freshness)
	add("derivation status", status)
	add("authority state", opsCivilizationStatusSummary(projection.AuthorityState.Status, projection.AuthorityState.Summary))
	add("external committee", opsCivilizationStatusSummary(projection.ExternalCommitteeState.Status, projection.ExternalCommitteeState.Summary))
	add("work evidence", opsCivilizationStatusSummary(projection.WorkEvidenceSummary.Status, projection.WorkEvidenceSummary.Summary))
	add("site consumer", opsCivilizationStatusSummary(projection.SiteConsumerStatus.Status, projection.SiteConsumerStatus.Summary))
	return rows
}

func opsCivilizationReferenceGroups(projection *OpsCivilizationAssemblyProjection) []OpsCivilizationReferenceGroup {
	if projection == nil {
		return nil
	}
	groups := []OpsCivilizationReferenceGroup{
		{Label: "source events/window", Refs: append([]string(nil), projection.SourceEventIDsOrQueryWindow...)},
		{Label: "provenance refs", Refs: append([]string(nil), projection.ProvenanceRefs...)},
		{Label: "validation refs", Refs: append([]string(nil), projection.ValidationRefs...)},
		{Label: "boundary flags", Refs: append([]string(nil), projection.BoundaryFlags...)},
		{Label: "failure reasons", Refs: append([]string(nil), projection.FailureReasons...)},
	}
	withheld := make([]string, 0, len(projection.WithheldOrUnavailableFields))
	for _, field := range projection.WithheldOrUnavailableFields {
		withheld = append(withheld, fmt.Sprintf("%s: %s", opsCivilizationValue(field.Field, "unknown field"), opsCivilizationValue(field.Reason, "reason not projected")))
	}
	groups = append(groups, OpsCivilizationReferenceGroup{Label: "withheld/unavailable fields", Refs: withheld})
	out := groups[:0]
	for _, group := range groups {
		refs := opsCivilizationNonEmpty(group.Refs)
		if len(refs) == 0 {
			continue
		}
		sort.Strings(refs)
		group.Refs = refs
		out = append(out, group)
	}
	return out
}

func opsCivilizationIssueReadiness(projection *OpsCivilizationAssemblyProjection) OpsCivilizationIssueReadiness {
	readiness := OpsCivilizationIssueReadiness{
		Status:              opsCivilizationProjectionStatusUnavailable,
		PRReadyWhen:         "PR-ready when recommendation states and guardrail labels are defined, validated, and supported by implementation, validation, exact-head CFAR, and ready-for-Human PR evidence.",
		FirstPendingStage:   "not projected",
		RecommendationState: "unavailable",
		GroupingSummary:     "No grouping recommendation is available without queued issue-scan projection input.",
		Guardrails:          opsCivilizationIssueGuardrails(),
	}
	if projection == nil {
		return readiness
	}

	sourceRefs := []string{}
	if projection.QueuedRunRequest != nil {
		q := projection.QueuedRunRequest
		sourceRefs = append(sourceRefs, q.EventID, q.SourceEventID, q.BriefEventID, q.RunID)
		if q.SelectionPolicy != nil {
			policy := q.SelectionPolicy
			readiness.RecommendationState = fmt.Sprintf("recommendation-only rank %s of %s by %s",
				opsHiveIntValue(policy.SelectedRank, "?"),
				opsHiveIntValue(policy.CandidateCount, "?"),
				opsCivilizationValue(policy.PolicyID, "unprojected policy"),
			)
			readiness.GroupingSummary = opsCivilizationValue(policy.Rationale, "Selection policy did not project a grouping rationale.")
			readiness.GroupingInputs = opsCivilizationNonEmpty(policy.RankingInputs)
			sort.Strings(readiness.GroupingInputs)
		} else {
			readiness.RecommendationState = "queued issue-scan projected without a selection policy"
			readiness.GroupingSummary = "Candidate grouping remains undefined until selection policy evidence is projected."
		}
		readiness.Status, readiness.FirstPendingStage = opsCivilizationIssueReadinessStatus(projection)
	} else {
		readiness.Status = "not projected"
		readiness.RecommendationState = "not projected"
		readiness.GroupingSummary = "Queued issue-scan selection policy is not projected."
	}
	sourceRefs = append(sourceRefs, projection.ValidationRefs...)
	readiness.SourceRefs = opsCivilizationNonEmpty(sourceRefs)
	sort.Strings(readiness.SourceRefs)
	return readiness
}

func opsCivilizationIssueReadinessStatus(projection *OpsCivilizationAssemblyProjection) (string, string) {
	if projection == nil || projection.QueuedRunRequest == nil {
		return "not projected", "not projected"
	}
	lifecycle := projection.QueuedRunRequest.DevelopmentLifecycle
	if len(lifecycle) == 0 {
		return "not projected", "lifecycle stages not projected"
	}

	completedStageEvidence := map[string]bool{}
	for _, task := range projection.WorkEvidenceSummary.Tasks {
		stageID := strings.TrimSpace(task.LifecycleStageID)
		if stageID == "" {
			continue
		}
		if task.Ready || opsCivilizationEvidenceObserved(task.RuntimeEvidenceStatus) {
			completedStageEvidence[stageID] = true
		}
	}

	for _, stage := range lifecycle {
		stageID := strings.TrimSpace(stage.ID)
		stageName := opsCivilizationValue(stage.Name, stageID)
		if stageID != "" && completedStageEvidence[stageID] {
			continue
		}
		if !opsCivilizationEvidenceObserved(stage.EvidenceStatus) {
			return "pending: " + stageName, stageName
		}
	}
	return "ready-for-Human PR evidence projected", "none"
}

func opsCivilizationEvidenceObserved(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case "available",
		"complete",
		"completed",
		"green",
		"passed",
		"recorded",
		"stage completed runtime evidence recorded",
		"stage step completed runtime evidence recorded",
		"stage_completed_runtime_evidence_recorded",
		"stage_step_completed_runtime_evidence_recorded",
		"zero blocker",
		"zero blockers",
		"zero_blocker",
		"zero_blockers":
		return true
	default:
		return false
	}
}

func opsCivilizationIssueGuardrails() []OpsCivilizationIssueGuardrail {
	return []OpsCivilizationIssueGuardrail{
		{
			Label:  "cc:intake",
			State:  "scanner-visible",
			Detail: "Durable source-of-intent and scope evidence; not implementation, PR, merge, deploy, or authority approval.",
		},
		{
			Label:  "cc:pr-deferred",
			State:  "hold",
			Detail: "PR work remains deferred until PR-Ready-When evidence is satisfied and visibly validated.",
		},
		{
			Label:  "cc:aggregate-candidate",
			State:  "recommendation-only",
			Detail: "Grouping is advisory and requires matching repo, substrate, risk, acceptance path, and readiness condition.",
		},
		{
			Label:  "cc:civilization-presence",
			State:  "visible",
			Detail: "Issue should remain visible to future Civilization intake and aggregation scans.",
		},
		{
			Label:  "cc:protected-action",
			State:  "authority gate",
			Detail: "If present or if scope becomes protected-action sensitive, implementation requires a separate human-scoped AuthorityDecision.",
		},
	}
}

func opsCivilizationFactoryOrders(projection *OpsCivilizationAssemblyProjection) []OpsCivilizationAssemblyFactoryOrder {
	if projection == nil {
		return nil
	}
	out := append([]OpsCivilizationAssemblyFactoryOrder(nil), projection.FactoryOrderSummary...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func opsCivilizationWorkEvidence(projection *OpsCivilizationAssemblyProjection) OpsCivilizationAssemblyWorkEvidence {
	if projection == nil {
		return OpsCivilizationAssemblyWorkEvidence{
			Status:  opsCivilizationFieldUnavailable,
			Summary: "EventGraph Civilization Assembly projection unavailable to Site",
		}
	}
	return projection.WorkEvidenceSummary
}

func opsCivilizationQueuedRunRequest(projection *OpsCivilizationAssemblyProjection) *OpsHiveQueuedRunRequest {
	if projection == nil {
		return nil
	}
	return projection.QueuedRunRequest
}

func opsCivilizationIssueScanStageEvidence(projection *OpsCivilizationAssemblyProjection) []OpsCivilizationIssueScanStageEvidence {
	if projection == nil {
		return nil
	}
	stageNames := map[string]string{}
	stageStatuses := map[string]string{}
	stageOrder := map[string]int{}
	if projection.QueuedRunRequest != nil {
		for i, stage := range projection.QueuedRunRequest.DevelopmentLifecycle {
			stageID := strings.TrimSpace(stage.ID)
			if stageID == "" {
				continue
			}
			stageNames[stageID] = strings.TrimSpace(stage.Name)
			stageStatuses[stageID] = strings.TrimSpace(stage.EvidenceStatus)
			stageOrder[stageID] = i
		}
	}
	out := make([]OpsCivilizationIssueScanStageEvidence, 0)
	for _, artifact := range projection.WorkEvidenceSummary.Artifacts {
		label := strings.TrimSpace(artifact.Label)
		if !strings.HasPrefix(label, opsCivilizationIssueScanStageArtifactPrefix) {
			continue
		}
		stageID := strings.TrimSpace(strings.TrimPrefix(label, opsCivilizationIssueScanStageArtifactPrefix))
		if stageID == "" {
			continue
		}
		sourceRefs := opsCivilizationNonEmpty(artifact.SourceRefs)
		sort.Strings(sourceRefs)
		out = append(out, OpsCivilizationIssueScanStageEvidence{
			StageID:        stageID,
			StageName:      stageNames[stageID],
			ArtifactID:     strings.TrimSpace(artifact.ID),
			Label:          label,
			MediaType:      strings.TrimSpace(artifact.MediaType),
			TaskRef:        strings.TrimSpace(artifact.TaskRef),
			SourceRefs:     sourceRefs,
			EvidenceStatus: opsCivilizationEvidenceStatusValue(stageStatuses[stageID], "declared pending runtime evidence"),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		oi, iKnown := stageOrder[out[i].StageID]
		oj, jKnown := stageOrder[out[j].StageID]
		if iKnown && jKnown && oi != oj {
			return oi < oj
		}
		if iKnown != jKnown {
			return iKnown
		}
		if out[i].StageID == out[j].StageID {
			return out[i].ArtifactID < out[j].ArtifactID
		}
		return out[i].StageID < out[j].StageID
	})
	return out
}

func opsCivilizationStatusSummary(status string, summary string) string {
	status = opsCivilizationStatusValue(status)
	if summary == "" {
		return status
	}
	return status + " - " + summary
}

func opsCivilizationStatusValue(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case opsCivilizationFieldAvailable:
		return opsCivilizationFieldAvailable
	case opsCivilizationFieldUnavailable, "":
		return opsCivilizationFieldUnavailable
	default:
		return strings.TrimSpace(status)
	}
}

func opsCivilizationEvidenceStatusValue(status string, fallback string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return fallback
	}
	return strings.ReplaceAll(status, "_", " ")
}

func opsCivilizationBoolValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func opsCivilizationTime(t time.Time) string {
	if t.IsZero() {
		return "not projected"
	}
	return t.UTC().Format("2006-01-02 15:04:05 UTC")
}

func opsCivilizationJoin(items []string, fallback string) string {
	items = opsCivilizationNonEmpty(items)
	if len(items) == 0 {
		return fallback
	}
	return strings.Join(items, ", ")
}

func opsCivilizationNonEmpty(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func opsCivilizationValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
