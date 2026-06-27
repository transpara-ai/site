package graph

import (
	"fmt"
	"sort"
	"strings"
)

type OpsCivilizationIssueScanProjection struct {
	Runs     []OpsCivilizationIssueScanRunProjected     `json:"runs,omitempty"`
	Stages   []OpsCivilizationIssueScanStageProjected   `json:"stages,omitempty"`
	Blockers []OpsCivilizationIssueScanBlockerProjected `json:"blockers,omitempty"`
	Lineage  []OpsCivilizationIssueScanLineageProjected `json:"lineage,omitempty"`
}

type OpsCivilizationIssueRef struct {
	Repo        string   `json:"repo"`
	Number      int      `json:"number"`
	URL         string   `json:"url,omitempty"`
	Title       string   `json:"title,omitempty"`
	State       string   `json:"state,omitempty"`
	StateReason string   `json:"state_reason,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

type OpsCivilizationIssueScanRunProjected struct {
	RunID            string                    `json:"run_id"`
	FactoryOrderID   string                    `json:"factory_order_id,omitempty"`
	LifecycleVersion string                    `json:"lifecycle_version"`
	State            string                    `json:"state"`
	TargetIssue      OpsCivilizationIssueRef   `json:"target_issue"`
	SelectedIssue    OpsCivilizationIssueRef   `json:"selected_issue"`
	CandidateIssues  []OpsCivilizationIssueRef `json:"candidate_issues,omitempty"`
	SourceRefs       []string                  `json:"source_refs,omitempty"`
	EvidenceRefs     []string                  `json:"evidence_refs,omitempty"`
}

type OpsCivilizationIssueScanStageProjected struct {
	RunID             string   `json:"run_id"`
	FactoryOrderID    string   `json:"factory_order_id,omitempty"`
	StageID           string   `json:"stage_id"`
	StageNumber       int      `json:"stage_number"`
	StageCount        int      `json:"stage_count,omitempty"`
	CanonicalTaskID   string   `json:"canonical_task_id"`
	TaskID            string   `json:"task_id,omitempty"`
	CurrentState      string   `json:"current_state"`
	CompletionGate    string   `json:"completion_gate"`
	AuthorityBoundary string   `json:"authority_boundary"`
	AssignedAgentIDs  []string `json:"assigned_agent_ids,omitempty"`
	TouchingAgentIDs  []string `json:"touching_agent_ids,omitempty"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
	SourceRefs        []string `json:"source_refs,omitempty"`
}

type OpsCivilizationIssueScanBlockerProjected struct {
	RunID          string   `json:"run_id"`
	FactoryOrderID string   `json:"factory_order_id,omitempty"`
	StageID        string   `json:"stage_id,omitempty"`
	BlockerType    string   `json:"blocker_type"`
	Reason         string   `json:"reason,omitempty"`
	RequiredAction string   `json:"required_action"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
	SourceRefs     []string `json:"source_refs,omitempty"`
}

type OpsCivilizationIssueScanLineageProjected struct {
	RunID             string   `json:"run_id"`
	FactoryOrderID    string   `json:"factory_order_id,omitempty"`
	StageID           string   `json:"stage_id,omitempty"`
	CanonicalTaskID   string   `json:"canonical_task_id"`
	PrimaryTaskID     string   `json:"primary_task_id,omitempty"`
	TaskIDs           []string `json:"task_ids"`
	DuplicateTaskIDs  []string `json:"duplicate_task_ids,omitempty"`
	DuplicateOf       string   `json:"duplicate_of,omitempty"`
	SupersededTaskIDs []string `json:"superseded_task_ids,omitempty"`
	SourceRefs        []string `json:"source_refs,omitempty"`
}

type OpsCivilizationIssueScanKanban struct {
	Status  string
	Summary string
	Columns []OpsCivilizationIssueScanKanbanColumn
}

type OpsCivilizationIssueScanKanbanColumn struct {
	State string
	Label string
	Cards []OpsCivilizationIssueScanKanbanCard
}

type OpsCivilizationIssueScanKanbanCard struct {
	RunID             string
	FactoryOrderID    string
	LifecycleVersion  string
	StageID           string
	StageNumber       int
	StageCount        int
	CanonicalTaskID   string
	TaskID            string
	CurrentState      string
	ProjectionSource  string
	Readiness         string
	PRReadyWhen       string
	Labels            []string
	CompletionGate    string
	AuthorityBoundary string
	TargetIssue       OpsCivilizationIssueRef
	SelectedIssue     OpsCivilizationIssueRef
	CandidateIssues   []OpsCivilizationIssueRef
	AssignedAgentIDs  []string
	TouchingAgentIDs  []string
	Blockers          []OpsCivilizationIssueScanBlockerProjected
	Lineage           OpsCivilizationIssueScanLineageProjected
	HasLineage        bool
	SourceRefs        []string
	EvidenceRefs      []string
}

func opsCivilizationIssueScanKanban(projection *OpsCivilizationAssemblyProjection) OpsCivilizationIssueScanKanban {
	if projection == nil {
		return OpsCivilizationIssueScanKanban{
			Status:  opsCivilizationProjectionStatusUnavailable,
			Summary: "Issue-scan projection unavailable to Site.",
		}
	}
	input := projection.IssueScanProjection
	if !issueScanProjectionHasRecords(input) {
		return issueScanKanbanFromIssueIntakeFallback(projection)
	}

	runsByID := map[string]OpsCivilizationIssueScanRunProjected{}
	for _, run := range input.Runs {
		run.RunID = strings.TrimSpace(run.RunID)
		if run.RunID == "" {
			continue
		}
		run.SourceRefs = sortedNonEmpty(run.SourceRefs)
		run.EvidenceRefs = sortedNonEmpty(run.EvidenceRefs)
		run.CandidateIssues = sortedIssueRefs(run.CandidateIssues)
		runsByID[run.RunID] = run
	}

	blockersByRunStage := map[string][]OpsCivilizationIssueScanBlockerProjected{}
	for _, blocker := range input.Blockers {
		blocker.RunID = strings.TrimSpace(blocker.RunID)
		blocker.StageID = strings.TrimSpace(blocker.StageID)
		if blocker.RunID == "" && blocker.StageID == "" {
			continue
		}
		blocker.EvidenceRefs = sortedNonEmpty(blocker.EvidenceRefs)
		blocker.SourceRefs = sortedNonEmpty(blocker.SourceRefs)
		key := runStageKey(blocker.RunID, blocker.StageID)
		blockersByRunStage[key] = append(blockersByRunStage[key], blocker)
	}
	for key := range blockersByRunStage {
		sort.Slice(blockersByRunStage[key], func(i, j int) bool {
			if blockersByRunStage[key][i].BlockerType == blockersByRunStage[key][j].BlockerType {
				return blockersByRunStage[key][i].RequiredAction < blockersByRunStage[key][j].RequiredAction
			}
			return blockersByRunStage[key][i].BlockerType < blockersByRunStage[key][j].BlockerType
		})
	}

	lineageByRunStage := map[string]OpsCivilizationIssueScanLineageProjected{}
	for _, lineage := range input.Lineage {
		lineage.RunID = strings.TrimSpace(lineage.RunID)
		lineage.StageID = strings.TrimSpace(lineage.StageID)
		if lineage.RunID == "" && lineage.StageID == "" && strings.TrimSpace(lineage.CanonicalTaskID) == "" {
			continue
		}
		lineage.TaskIDs = sortedNonEmpty(lineage.TaskIDs)
		lineage.DuplicateTaskIDs = sortedNonEmpty(lineage.DuplicateTaskIDs)
		lineage.SupersededTaskIDs = sortedNonEmpty(lineage.SupersededTaskIDs)
		lineage.SourceRefs = sortedNonEmpty(lineage.SourceRefs)
		lineageByRunStage[runStageKey(lineage.RunID, lineage.StageID)] = lineage
		if lineage.CanonicalTaskID != "" {
			lineageByRunStage[runCanonicalKey(lineage.RunID, lineage.CanonicalTaskID)] = lineage
		}
	}

	cards := make([]OpsCivilizationIssueScanKanbanCard, 0, len(input.Stages)+len(input.Runs))
	seenRunStage := map[string]bool{}
	for _, stage := range input.Stages {
		stage.RunID = strings.TrimSpace(stage.RunID)
		stage.StageID = strings.TrimSpace(stage.StageID)
		if stage.RunID == "" && stage.StageID == "" {
			continue
		}
		run := runsByID[stage.RunID]
		card := issueScanCardFromStage(run, stage, blockersByRunStage, lineageByRunStage)
		cards = append(cards, card)
		seenRunStage[runStageKey(card.RunID, card.StageID)] = true
	}
	for _, run := range runsByID {
		if run.RunID == "" || hasStageForRun(input.Stages, run.RunID) {
			continue
		}
		card := issueScanCardFromRun(run, blockersByRunStage, lineageByRunStage)
		cards = append(cards, card)
		seenRunStage[runStageKey(card.RunID, card.StageID)] = true
	}
	for key, blockers := range blockersByRunStage {
		if seenRunStage[key] {
			continue
		}
		runID, stageID := splitRunStageKey(key)
		if stageID == "" && hasStageForRun(input.Stages, runID) {
			continue
		}
		run := runsByID[runID]
		card := issueScanCardFromRun(run, blockersByRunStage, lineageByRunStage)
		card.RunID = opsCivilizationValue(runID, card.RunID)
		card.StageID = stageID
		card.CurrentState = issueScanColumnState("", blockers)
		card.Blockers = blockers
		cards = append(cards, card)
	}

	return OpsCivilizationIssueScanKanban{
		Status:  opsCivilizationFieldAvailable,
		Summary: fmt.Sprintf("%d run(s), %d stage(s), %d blocker(s), %d lineage record(s) projected.", len(input.Runs), len(input.Stages), len(input.Blockers), len(input.Lineage)),
		Columns: issueScanKanbanColumns(cards),
	}
}

func issueScanProjectionHasRecords(input OpsCivilizationIssueScanProjection) bool {
	return len(input.Runs) > 0 || len(input.Stages) > 0 || len(input.Blockers) > 0 || len(input.Lineage) > 0
}

func issueScanKanbanFromIssueIntakeFallback(projection *OpsCivilizationAssemblyProjection) OpsCivilizationIssueScanKanban {
	issues := normalizedIssueIntakeIssues(projection.IssueIntakeProjection.Issues)
	if len(issues) == 0 {
		return OpsCivilizationIssueScanKanban{
			Status:  "not projected",
			Summary: "No typed issue-scan projection records are present.",
		}
	}
	cards := make([]OpsCivilizationIssueScanKanbanCard, 0, len(issues))
	for _, issue := range issues {
		cards = append(cards, issueScanCardFromIssueIntakeFallback(issue))
	}
	return OpsCivilizationIssueScanKanban{
		Status:  "intake fallback",
		Summary: fmt.Sprintf("No typed issue-scan projection records are present; rendering %d scanner issue-intake fallback card(s). Fallback cards are not runtime execution or agent-touch evidence.", len(cards)),
		Columns: issueScanKanbanColumns(cards),
	}
}

func issueScanKanbanColumns(cards []OpsCivilizationIssueScanKanbanCard) []OpsCivilizationIssueScanKanbanColumn {
	columnsByState := map[string][]OpsCivilizationIssueScanKanbanCard{}
	for _, card := range cards {
		state := issueScanColumnState(card.CurrentState, card.Blockers)
		card.CurrentState = state
		columnsByState[state] = append(columnsByState[state], card)
	}

	columns := make([]OpsCivilizationIssueScanKanbanColumn, 0, len(columnsByState))
	for _, state := range issueScanColumnOrder() {
		columnCards := columnsByState[state]
		if len(columnCards) == 0 {
			continue
		}
		sortIssueScanCards(columnCards)
		columns = append(columns, OpsCivilizationIssueScanKanbanColumn{
			State: state,
			Label: issueScanStateLabel(state),
			Cards: columnCards,
		})
		delete(columnsByState, state)
	}
	extraStates := make([]string, 0, len(columnsByState))
	for state := range columnsByState {
		extraStates = append(extraStates, state)
	}
	sort.Strings(extraStates)
	for _, state := range extraStates {
		columnCards := columnsByState[state]
		sortIssueScanCards(columnCards)
		columns = append(columns, OpsCivilizationIssueScanKanbanColumn{
			State: state,
			Label: issueScanStateLabel(state),
			Cards: columnCards,
		})
	}
	return columns
}

func issueScanCardFromIssueIntakeFallback(issue OpsCivilizationIssueIntakeProjected) OpsCivilizationIssueScanKanbanCard {
	labels := sortedUniqueNonEmpty(issue.Labels)
	readiness := issueScanFallbackReadiness(issue)
	sourceRefs := sortedUniqueNonEmpty(append(append([]string{}, issue.SourceRefs...), issue.URL))
	selectedIssue := OpsCivilizationIssueRef{
		Repo:        issue.Repo,
		Number:      issue.Number,
		URL:         issue.URL,
		Title:       issue.Title,
		State:       issue.State,
		StateReason: issue.StateReason,
		Labels:      labels,
	}
	return OpsCivilizationIssueScanKanbanCard{
		RunID:             issueScanFallbackRunID(issue),
		StageID:           "issue_intake_fallback",
		CurrentState:      issueScanFallbackState(issue),
		ProjectionSource:  "scanner issue-intake fallback; not runtime execution or agent-touch evidence",
		Readiness:         readiness,
		PRReadyWhen:       issue.PRReadyWhen,
		Labels:            labels,
		CompletionGate:    opsCivilizationValue(issue.PRReadyWhen, opsCivilizationValue(readiness, "read-only scanner issue-intake fallback")),
		AuthorityBoundary: opsCivilizationValue(issue.AuthorityBoundary, "read-only scanner issue record; no runtime, protected-action, or merge authority"),
		TargetIssue:       selectedIssue,
		SelectedIssue:     selectedIssue,
		Blockers:          issueScanFallbackBlockers(issue, sourceRefs),
		SourceRefs:        sourceRefs,
	}
}

func issueScanFallbackRunID(issue OpsCivilizationIssueIntakeProjected) string {
	id := issueIntakeGroupID("intake", issue.Repo, fmt.Sprintf("%d", issue.Number), issue.Title)
	if id == "" {
		return "intake-issue"
	}
	return id
}

func issueScanFallbackReadiness(issue OpsCivilizationIssueIntakeProjected) string {
	if strings.TrimSpace(issue.Readiness) != "" {
		return strings.TrimSpace(issue.Readiness)
	}
	return strings.Join(sortedUniqueNonEmpty(issue.ReadinessStates), ", ")
}

func issueScanFallbackState(issue OpsCivilizationIssueIntakeProjected) string {
	if issueScanFallbackNeedsHumanScope(issue) || issueIntakeHasProtectedActionRisk(issue) {
		return "human_action"
	}
	if issueScanFallbackDeferredOrStale(issue) {
		return "parked"
	}
	if issueScanFallbackPRReady(issue) {
		return "ready_for_human"
	}
	return "projection_only"
}

func issueScanFallbackBlockers(issue OpsCivilizationIssueIntakeProjected, sourceRefs []string) []OpsCivilizationIssueScanBlockerProjected {
	blockers := []OpsCivilizationIssueScanBlockerProjected{}
	reason := issueScanFallbackReason(issue)
	if issueIntakeHasProtectedActionRisk(issue) {
		blockers = append(blockers, OpsCivilizationIssueScanBlockerProjected{
			RunID:          issueScanFallbackRunID(issue),
			StageID:        "issue_intake_fallback",
			BlockerType:    "protected_action",
			Reason:         reason,
			RequiredAction: "separate authority scope is required before protected work can proceed",
			SourceRefs:     sourceRefs,
		})
	}
	if issueScanFallbackNeedsHumanScope(issue) {
		blockers = append(blockers, OpsCivilizationIssueScanBlockerProjected{
			RunID:          issueScanFallbackRunID(issue),
			StageID:        "issue_intake_fallback",
			BlockerType:    "needs_human_scope",
			Reason:         reason,
			RequiredAction: "human scope decision is required before runtime or PR work continues",
			SourceRefs:     sourceRefs,
		})
	}
	if issueScanFallbackDeferredOrStale(issue) {
		blockers = append(blockers, OpsCivilizationIssueScanBlockerProjected{
			RunID:          issueScanFallbackRunID(issue),
			StageID:        "issue_intake_fallback",
			BlockerType:    "parked_issue_intake",
			Reason:         reason,
			RequiredAction: "issue remains parked in scanner intake; do not queue runtime work from this fallback card",
			SourceRefs:     sourceRefs,
		})
	}
	return blockers
}

func issueScanFallbackReason(issue OpsCivilizationIssueIntakeProjected) string {
	return opsCivilizationValue(issueScanFallbackReadiness(issue), opsCivilizationValue(issue.StateReason, issue.AuthorityBoundary))
}

func issueScanFallbackNeedsHumanScope(issue OpsCivilizationIssueIntakeProjected) bool {
	if issueIntakeHasLabel(issue.Labels, "cc:needs-human-scope") {
		return true
	}
	text := issueScanFallbackText(issue)
	return strings.Contains(text, "needs-human-scope") || strings.Contains(text, "human scope")
}

func issueScanFallbackDeferredOrStale(issue OpsCivilizationIssueIntakeProjected) bool {
	if issueIntakeHasLabel(issue.Labels, "cc:pr-deferred") {
		return true
	}
	for _, signal := range issueScanFallbackStateSignals(issue) {
		text := strings.ToLower(strings.TrimSpace(signal))
		if strings.Contains(text, "pr-deferred") ||
			strings.Contains(text, "deferred") ||
			strings.Contains(text, "stale") {
			return true
		}
		if strings.Contains(text, "blocked") &&
			!strings.Contains(text, "not blocked") &&
			!strings.Contains(text, "unblocked") {
			return true
		}
	}
	return false
}

func issueScanFallbackPRReady(issue OpsCivilizationIssueIntakeProjected) bool {
	if issueIntakeHasLabel(issue.Labels, "cc:pr-ready") {
		return true
	}
	readiness := strings.ToLower(strings.TrimSpace(issue.Readiness))
	return strings.HasPrefix(readiness, "ready:") || strings.Contains(readiness, "pr-ready now")
}

func issueScanFallbackText(issue OpsCivilizationIssueIntakeProjected) string {
	parts := []string{
		issue.Readiness,
		issue.PRReadyWhen,
		issue.State,
		issue.StateReason,
		issue.RiskClass,
		issue.AuthorityBoundary,
	}
	parts = append(parts, issue.ReadinessStates...)
	parts = append(parts, issue.RiskClasses...)
	parts = append(parts, issue.Labels...)
	return strings.ToLower(strings.Join(parts, " "))
}

func issueScanFallbackStateSignals(issue OpsCivilizationIssueIntakeProjected) []string {
	parts := []string{
		issue.Readiness,
		issue.State,
		issue.StateReason,
	}
	parts = append(parts, issue.ReadinessStates...)
	parts = append(parts, issue.Labels...)
	return parts
}

func issueScanCardFromStage(run OpsCivilizationIssueScanRunProjected, stage OpsCivilizationIssueScanStageProjected, blockersByRunStage map[string][]OpsCivilizationIssueScanBlockerProjected, lineageByRunStage map[string]OpsCivilizationIssueScanLineageProjected) OpsCivilizationIssueScanKanbanCard {
	runBlockers := blockersByRunStage[runStageKey(stage.RunID, "")]
	stageBlockers := blockersByRunStage[runStageKey(stage.RunID, stage.StageID)]
	blockers := append([]OpsCivilizationIssueScanBlockerProjected{}, stageBlockers...)
	blockers = append(blockers, runBlockers...)
	lineage, hasLineage := lineageByRunStage[runStageKey(stage.RunID, stage.StageID)]
	if !hasLineage && stage.CanonicalTaskID != "" {
		lineage, hasLineage = lineageByRunStage[runCanonicalKey(stage.RunID, stage.CanonicalTaskID)]
	}
	sourceRefs := sortedNonEmpty(append(append([]string{}, stage.SourceRefs...), run.SourceRefs...))
	evidenceRefs := sortedNonEmpty(append(append([]string{}, stage.EvidenceRefs...), run.EvidenceRefs...))
	return OpsCivilizationIssueScanKanbanCard{
		RunID:             opsCivilizationValue(stage.RunID, run.RunID),
		FactoryOrderID:    opsCivilizationValue(stage.FactoryOrderID, run.FactoryOrderID),
		LifecycleVersion:  run.LifecycleVersion,
		StageID:           stage.StageID,
		StageNumber:       stage.StageNumber,
		StageCount:        stage.StageCount,
		CanonicalTaskID:   stage.CanonicalTaskID,
		TaskID:            stage.TaskID,
		CurrentState:      issueScanColumnState(stage.CurrentState, blockers),
		ProjectionSource:  "typed issue-scan projection",
		Labels:            sortedNonEmpty(run.SelectedIssue.Labels),
		CompletionGate:    stage.CompletionGate,
		AuthorityBoundary: stage.AuthorityBoundary,
		TargetIssue:       run.TargetIssue,
		SelectedIssue:     run.SelectedIssue,
		CandidateIssues:   sortedIssueRefs(run.CandidateIssues),
		AssignedAgentIDs:  sortedNonEmpty(stage.AssignedAgentIDs),
		TouchingAgentIDs:  sortedNonEmpty(stage.TouchingAgentIDs),
		Blockers:          blockers,
		Lineage:           lineage,
		HasLineage:        hasLineage,
		SourceRefs:        sourceRefs,
		EvidenceRefs:      evidenceRefs,
	}
}

func issueScanCardFromRun(run OpsCivilizationIssueScanRunProjected, blockersByRunStage map[string][]OpsCivilizationIssueScanBlockerProjected, lineageByRunStage map[string]OpsCivilizationIssueScanLineageProjected) OpsCivilizationIssueScanKanbanCard {
	blockers := blockersByRunStage[runStageKey(run.RunID, "")]
	lineage, hasLineage := lineageByRunStage[runStageKey(run.RunID, "")]
	return OpsCivilizationIssueScanKanbanCard{
		RunID:            run.RunID,
		FactoryOrderID:   run.FactoryOrderID,
		LifecycleVersion: run.LifecycleVersion,
		CurrentState:     issueScanColumnState(run.State, blockers),
		ProjectionSource: "typed issue-scan projection",
		Labels:           sortedNonEmpty(run.SelectedIssue.Labels),
		TargetIssue:      run.TargetIssue,
		SelectedIssue:    run.SelectedIssue,
		CandidateIssues:  sortedIssueRefs(run.CandidateIssues),
		Blockers:         blockers,
		Lineage:          lineage,
		HasLineage:       hasLineage,
		SourceRefs:       sortedNonEmpty(run.SourceRefs),
		EvidenceRefs:     sortedNonEmpty(run.EvidenceRefs),
	}
}

func issueScanColumnState(state string, blockers []OpsCivilizationIssueScanBlockerProjected) string {
	normalized := strings.ToLower(strings.TrimSpace(state))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, " ", "_")
	if normalized == "" && len(blockers) > 0 {
		return "blocked"
	}
	if normalized == "" {
		return "projection_only"
	}
	return normalized
}

func issueScanStateLabel(state string) string {
	state = strings.TrimSpace(state)
	if state == "" {
		return "Projection Only"
	}
	parts := strings.Split(strings.ReplaceAll(state, "-", "_"), "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func issueScanColumnOrder() []string {
	return []string{
		"queued",
		"dispatched",
		"running",
		"blocked",
		"parked",
		"human_action",
		"ready_for_human",
		"superseded",
		"completed",
		"projection_only",
	}
}

func sortIssueScanCards(cards []OpsCivilizationIssueScanKanbanCard) {
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].RunID != cards[j].RunID {
			return cards[i].RunID < cards[j].RunID
		}
		if cards[i].StageNumber != cards[j].StageNumber {
			return cards[i].StageNumber < cards[j].StageNumber
		}
		if cards[i].StageID != cards[j].StageID {
			return cards[i].StageID < cards[j].StageID
		}
		if cards[i].CanonicalTaskID != cards[j].CanonicalTaskID {
			return cards[i].CanonicalTaskID < cards[j].CanonicalTaskID
		}
		if cards[i].FactoryOrderID != cards[j].FactoryOrderID {
			return cards[i].FactoryOrderID < cards[j].FactoryOrderID
		}
		if cards[i].TaskID != cards[j].TaskID {
			return cards[i].TaskID < cards[j].TaskID
		}
		return cards[i].CompletionGate < cards[j].CompletionGate
	})
}

func hasStageForRun(stages []OpsCivilizationIssueScanStageProjected, runID string) bool {
	for _, stage := range stages {
		if strings.TrimSpace(stage.RunID) == runID {
			return true
		}
	}
	return false
}

func runStageKey(runID string, stageID string) string {
	return strings.TrimSpace(runID) + "\x00" + strings.TrimSpace(stageID)
}

func splitRunStageKey(key string) (string, string) {
	before, after, ok := strings.Cut(key, "\x00")
	if !ok {
		return key, ""
	}
	return before, after
}

func runCanonicalKey(runID string, canonicalTaskID string) string {
	return strings.TrimSpace(runID) + "\x00canonical\x00" + strings.TrimSpace(canonicalTaskID)
}

func sortedNonEmpty(items []string) []string {
	out := opsCivilizationNonEmpty(items)
	sort.Strings(out)
	return out
}

func sortedIssueRefs(items []OpsCivilizationIssueRef) []OpsCivilizationIssueRef {
	out := make([]OpsCivilizationIssueRef, 0, len(items))
	for _, item := range items {
		item.Repo = strings.TrimSpace(item.Repo)
		item.URL = strings.TrimSpace(item.URL)
		item.Title = strings.TrimSpace(item.Title)
		item.State = strings.TrimSpace(item.State)
		item.StateReason = strings.TrimSpace(item.StateReason)
		item.Labels = sortedNonEmpty(item.Labels)
		if item.Repo == "" && item.Number == 0 && item.URL == "" && item.Title == "" {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Repo == out[j].Repo {
			return out[i].Number < out[j].Number
		}
		return out[i].Repo < out[j].Repo
	})
	return out
}

func opsCivilizationIssueRefLabel(ref OpsCivilizationIssueRef) string {
	if ref.Repo != "" && ref.Number > 0 {
		return fmt.Sprintf("%s#%d", ref.Repo, ref.Number)
	}
	if ref.URL != "" {
		return ref.URL
	}
	return opsCivilizationValue(ref.Title, "issue not projected")
}
