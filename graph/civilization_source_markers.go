package graph

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const consoleSourceMarkerRenderLimit = 50

// OpsCivilizationIssueScanSourceMarkers mirrors the Site-facing read-model
// section for EventGraph issuescan.source.marker.projected events. It is
// display-only: Site must not derive lifecycle truth from GitHub marker
// comments or labels.
type OpsCivilizationIssueScanSourceMarkers struct {
	Status    string                                          `json:"status,omitempty"`
	Summary   string                                          `json:"summary,omitempty"`
	Markers   []OpsCivilizationIssueScanSourceMarkerProjected `json:"markers,omitempty"`
	Truncated bool                                            `json:"truncated,omitempty"`
}

type OpsCivilizationIssueScanSourceMarkerProjected struct {
	SchemaVersion       string                                     `json:"schema_version,omitempty"`
	ProjectionKind      string                                     `json:"projection_kind,omitempty"`
	Transition          string                                     `json:"transition,omitempty"`
	RunID               string                                     `json:"run_id,omitempty"`
	Target              OpsCivilizationIssueRef                    `json:"target,omitempty"`
	StageID             string                                     `json:"stage_id,omitempty"`
	StageNumber         int                                        `json:"stage_number,omitempty"`
	Gate                string                                     `json:"gate,omitempty"`
	WorkRef             OpsCivilizationIssueScanMarkerWorkRef      `json:"work_ref,omitempty"`
	ActorID             string                                     `json:"actor_id,omitempty"`
	ActorRole           string                                     `json:"actor_role,omitempty"`
	OccurredAt          string                                     `json:"occurred_at,omitempty"`
	IdempotencyKey      string                                     `json:"idempotency_key,omitempty"`
	AuthorityBoundary   string                                     `json:"authority_boundary,omitempty"`
	AuthorityExclusions []string                                   `json:"authority_exclusions,omitempty"`
	EvidenceRefs        OpsCivilizationIssueScanMarkerEvidenceRefs `json:"evidence_refs,omitempty"`
	SourceRefs          []string                                   `json:"source_refs,omitempty"`
	GitHubMarker        *OpsCivilizationIssueScanGitHubMarkerRef   `json:"github_marker,omitempty"`
	CanonicalSource     string                                     `json:"canonical_source,omitempty"`
	ProjectionOnly      bool                                       `json:"projection_only,omitempty"`
	SupersededBy        string                                     `json:"superseded_by,omitempty"`
	StaleTarget         bool                                       `json:"stale_target,omitempty"`
}

type OpsCivilizationIssueScanMarkerTargetRef struct {
	Repository  string `json:"repository,omitempty"`
	IssueNumber int    `json:"issue_number,omitempty"`
}

type OpsCivilizationIssueScanMarkerBlockerRef struct {
	Reason       string   `json:"reason,omitempty"`
	Detail       string   `json:"detail,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type OpsCivilizationIssueScanMarkerGateRef struct {
	Gate         string   `json:"gate,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type OpsCivilizationIssueScanMarkerEvidenceRefs struct {
	TestCaseIDs      []string `json:"test_case_ids,omitempty"`
	TestRunIDs       []string `json:"test_run_ids,omitempty"`
	GateResultIDs    []string `json:"gate_result_ids,omitempty"`
	FailureIDs       []string `json:"failure_ids,omitempty"`
	RepairAttemptIDs []string `json:"repair_attempt_ids,omitempty"`
	WaiverIDs        []string `json:"waiver_ids,omitempty"`
}

type OpsCivilizationIssueScanMarkerWorkRef struct {
	SchemaVersion          string                                     `json:"schema_version,omitempty"`
	ProjectionKind         string                                     `json:"projection_kind,omitempty"`
	CanonicalSource        string                                     `json:"canonical_source,omitempty"`
	ProjectionOnly         bool                                       `json:"projection_only,omitempty"`
	RunID                  string                                     `json:"run_id,omitempty"`
	Target                 OpsCivilizationIssueScanMarkerTargetRef    `json:"target,omitempty"`
	Stage                  string                                     `json:"stage,omitempty"`
	StageNumber            int                                        `json:"stage_number,omitempty"`
	Gate                   string                                     `json:"gate,omitempty"`
	TaskID                 string                                     `json:"task_id,omitempty"`
	CanonicalTaskID        string                                     `json:"canonical_task_id,omitempty"`
	FactoryOrderID         string                                     `json:"factory_order_id,omitempty"`
	RequirementIDs         []string                                   `json:"requirement_ids,omitempty"`
	AcceptanceCriterionIDs []string                                   `json:"acceptance_criterion_ids,omitempty"`
	LifecycleState         string                                     `json:"lifecycle_state,omitempty"`
	Ready                  bool                                       `json:"ready,omitempty"`
	Blocked                bool                                       `json:"blocked,omitempty"`
	MissingGates           []string                                   `json:"missing_gates,omitempty"`
	MissingFacts           []string                                   `json:"missing_facts,omitempty"`
	SupersededBy           string                                     `json:"superseded_by,omitempty"`
	LastTransitionEvent    string                                     `json:"last_transition_event,omitempty"`
	LatestBlocker          *OpsCivilizationIssueScanMarkerBlockerRef  `json:"latest_blocker,omitempty"`
	LatestGate             *OpsCivilizationIssueScanMarkerGateRef     `json:"latest_gate,omitempty"`
	VerificationRefs       OpsCivilizationIssueScanMarkerEvidenceRefs `json:"verification_refs,omitempty"`
	FailureRepairRefs      OpsCivilizationIssueScanMarkerEvidenceRefs `json:"failure_repair_refs,omitempty"`
	SourceIssueRefs        []string                                   `json:"source_issue_refs,omitempty"`
	AuthorityExclusions    []string                                   `json:"authority_exclusions,omitempty"`
}

type OpsCivilizationIssueScanGitHubMarkerRef struct {
	System         string   `json:"system,omitempty"`
	Repository     string   `json:"repository,omitempty"`
	IssueNumber    int      `json:"issue_number,omitempty"`
	CommentID      string   `json:"comment_id,omitempty"`
	CommentURL     string   `json:"comment_url,omitempty"`
	LabelNames     []string `json:"label_names,omitempty"`
	DerivedOutput  bool     `json:"derived_output,omitempty"`
	ProjectionSink bool     `json:"projection_sink,omitempty"`
}

type ConsoleSourceMarkers struct {
	Visible        bool
	Available      bool
	Status         string
	Summary        string
	Truncated      bool
	WithheldCount  int
	WithheldReason string
	Entries        []ConsoleSourceMarkerEntry
}

type ConsoleSourceMarkerEntry struct {
	Transition             string
	StyleKind              string
	IssueLabel             string
	IssueURL               string
	RunID                  string
	StageID                string
	StageNumber            int
	Gate                   string
	ActorID                string
	ActorRole              string
	OccurredAt             string
	Age                    string
	IdempotencyKey         string
	AuthorityBoundary      string
	AuthorityExclusions    []string
	CanonicalSource        string
	ProjectionKind         string
	ProjectionOnly         bool
	WorkProjectionKind     string
	WorkCanonicalSource    string
	WorkProjectionOnly     bool
	WorkLifecycleState     string
	WorkReady              bool
	WorkBlocked            bool
	WorkRefs               []string
	EventGraphRefs         []string
	EvidenceRefs           []string
	HasGitHubMarker        bool
	GitHubMarkerSystem     string
	GitHubMarkerIssueLabel string
	GitHubMarkerCommentID  string
	GitHubMarkerCommentURL string
	GitHubMarkerLabels     []string
	GitHubDerivedOutput    bool
	GitHubProjectionSink   bool
	StaleTarget            bool
	SupersededBy           string
}

var consoleSourceMarkerUsableFreshness = map[ConsoleFreshness]bool{
	FreshnessCurrent: true,
	FreshnessStale:   true,
	FreshnessPartial: true,
}

func buildConsoleSourceMarkers(proj *OpsCivilizationAssemblyProjection, freshness ConsoleFreshness, now time.Time) ConsoleSourceMarkers {
	if proj == nil {
		return ConsoleSourceMarkers{}
	}
	if !consoleSourceMarkerUsableFreshness[freshness] {
		return ConsoleSourceMarkers{}
	}
	section := proj.IssueScanSourceMarkers
	status := strings.TrimSpace(section.Status)
	if status == "" {
		return ConsoleSourceMarkers{}
	}
	out := ConsoleSourceMarkers{
		Visible:   true,
		Status:    status,
		Summary:   strings.TrimSpace(section.Summary),
		Truncated: section.Truncated,
	}
	if status != opsCivilizationFieldAvailable {
		return out
	}
	out.Available = true
	entries := make([]ConsoleSourceMarkerEntry, 0, len(section.Markers))
	invalidCount := 0
	for _, marker := range section.Markers {
		entry, ok := buildConsoleSourceMarkerEntry(marker, now)
		if ok {
			entries = append(entries, entry)
		} else {
			invalidCount++
		}
	}
	sortConsoleSourceMarkerEntries(entries)
	cappedCount := 0
	if len(entries) > consoleSourceMarkerRenderLimit {
		cappedCount = len(entries) - consoleSourceMarkerRenderLimit
		entries = entries[:consoleSourceMarkerRenderLimit]
	}
	out.Entries = entries
	out.WithheldCount = invalidCount + cappedCount
	out.WithheldReason = consoleSourceMarkerWithheldReason(invalidCount, cappedCount)
	return out
}

func buildConsoleSourceMarkerEntry(marker OpsCivilizationIssueScanSourceMarkerProjected, now time.Time) (ConsoleSourceMarkerEntry, bool) {
	marker.RunID = strings.TrimSpace(marker.RunID)
	marker.StageID = strings.TrimSpace(marker.StageID)
	marker.Transition = strings.TrimSpace(marker.Transition)
	marker.Target.Repo = strings.TrimSpace(marker.Target.Repo)
	marker.Target.URL = strings.TrimSpace(marker.Target.URL)
	marker.Target.Title = strings.TrimSpace(marker.Target.Title)
	marker.Target.State = strings.TrimSpace(marker.Target.State)
	marker.Target.StateReason = strings.TrimSpace(marker.Target.StateReason)
	marker.WorkRef.Target.Repository = strings.TrimSpace(marker.WorkRef.Target.Repository)
	if marker.RunID == "" {
		return ConsoleSourceMarkerEntry{}, false
	}
	entry := ConsoleSourceMarkerEntry{
		Transition:          marker.Transition,
		StyleKind:           consoleSourceMarkerStyleKind(marker.Transition),
		IssueLabel:          opsCivilizationIssueRefLabel(marker.Target),
		IssueURL:            strings.TrimSpace(marker.Target.URL),
		RunID:               marker.RunID,
		StageID:             marker.StageID,
		StageNumber:         marker.StageNumber,
		Gate:                strings.TrimSpace(marker.Gate),
		ActorID:             strings.TrimSpace(marker.ActorID),
		ActorRole:           strings.TrimSpace(marker.ActorRole),
		OccurredAt:          strings.TrimSpace(marker.OccurredAt),
		IdempotencyKey:      strings.TrimSpace(marker.IdempotencyKey),
		AuthorityBoundary:   strings.TrimSpace(marker.AuthorityBoundary),
		AuthorityExclusions: sortedNonEmpty(marker.AuthorityExclusions),
		CanonicalSource:     strings.TrimSpace(marker.CanonicalSource),
		ProjectionKind:      strings.TrimSpace(marker.ProjectionKind),
		ProjectionOnly:      marker.ProjectionOnly,
		WorkProjectionKind:  strings.TrimSpace(marker.WorkRef.ProjectionKind),
		WorkCanonicalSource: strings.TrimSpace(marker.WorkRef.CanonicalSource),
		WorkProjectionOnly:  marker.WorkRef.ProjectionOnly,
		WorkLifecycleState:  strings.TrimSpace(marker.WorkRef.LifecycleState),
		WorkReady:           marker.WorkRef.Ready,
		WorkBlocked:         marker.WorkRef.Blocked,
		WorkRefs:            consoleSourceMarkerWorkRefs(marker.WorkRef),
		EventGraphRefs:      consoleSourceMarkerEventGraphRefs(marker),
		EvidenceRefs:        consoleSourceMarkerEvidenceRefs(marker.EvidenceRefs),
		StaleTarget:         marker.StaleTarget,
		SupersededBy:        strings.TrimSpace(marker.SupersededBy),
	}
	entry.Age = humanizeAge(now, entry.OccurredAt)
	if marker.Target.Repo == "" && marker.Target.Number == 0 && marker.WorkRef.Target.Repository != "" && marker.WorkRef.Target.IssueNumber > 0 {
		entry.IssueLabel = fmt.Sprintf("%s#%d", marker.WorkRef.Target.Repository, marker.WorkRef.Target.IssueNumber)
		entry.IssueURL = ""
	}
	if marker.GitHubMarker != nil {
		entry.HasGitHubMarker = true
		entry.GitHubMarkerSystem = strings.TrimSpace(marker.GitHubMarker.System)
		entry.GitHubMarkerIssueLabel = consoleSourceMarkerGitHubIssueLabel(*marker.GitHubMarker)
		entry.GitHubMarkerCommentID = strings.TrimSpace(marker.GitHubMarker.CommentID)
		entry.GitHubMarkerCommentURL = strings.TrimSpace(marker.GitHubMarker.CommentURL)
		entry.GitHubMarkerLabels = sortedNonEmpty(marker.GitHubMarker.LabelNames)
		entry.GitHubDerivedOutput = marker.GitHubMarker.DerivedOutput
		entry.GitHubProjectionSink = marker.GitHubMarker.ProjectionSink
	}
	return entry, true
}

func sortConsoleSourceMarkerEntries(entries []ConsoleSourceMarkerEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		left, leftOK := consoleSourceMarkerOccurredAt(entries[i])
		right, rightOK := consoleSourceMarkerOccurredAt(entries[j])
		switch {
		case leftOK && rightOK:
			return left.After(right)
		case leftOK:
			return true
		case rightOK:
			return false
		default:
			return false
		}
	})
}

func consoleSourceMarkerOccurredAt(entry ConsoleSourceMarkerEntry) (time.Time, bool) {
	occurredAt := strings.TrimSpace(entry.OccurredAt)
	if occurredAt == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, occurredAt)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func consoleSourceMarkerWithheldReason(invalidCount, cappedCount int) string {
	switch {
	case invalidCount > 0 && cappedCount > 0:
		return "missing identifiers or local render cap"
	case invalidCount > 0:
		return "missing identifiers"
	case cappedCount > 0:
		return "local render cap"
	default:
		return ""
	}
}

func consoleSourceMarkerStyleKind(transition string) string {
	switch strings.TrimSpace(transition) {
	case "parked_human_action":
		return "amber"
	case "ready_for_human", "completed":
		return "ready"
	case "abandoned", "superseded":
		return "muted"
	default:
		return "neutral"
	}
}

func consoleSourceMarkerWorkRefs(workRef OpsCivilizationIssueScanMarkerWorkRef) []string {
	refs := []string{
		labeledNonEmpty("projection_kind", workRef.ProjectionKind),
		labeledNonEmpty("canonical_source", workRef.CanonicalSource),
		labeledNonEmpty("run_id", workRef.RunID),
		labeledNonEmpty("factory_order_id", workRef.FactoryOrderID),
		labeledNonEmpty("canonical_task_id", workRef.CanonicalTaskID),
		labeledNonEmpty("task_id", workRef.TaskID),
		labeledNonEmpty("stage", workRef.Stage),
		labeledNonEmpty("gate", workRef.Gate),
		labeledNonEmpty("lifecycle_state", workRef.LifecycleState),
		labeledNonEmpty("last_transition_event", workRef.LastTransitionEvent),
	}
	if workRef.Target.Repository != "" && workRef.Target.IssueNumber > 0 {
		refs = append(refs, fmt.Sprintf("target:%s#%d", workRef.Target.Repository, workRef.Target.IssueNumber))
	}
	if workRef.Ready {
		refs = append(refs, "ready:true")
	}
	if workRef.Blocked {
		refs = append(refs, "blocked:true")
	}
	if workRef.ProjectionOnly {
		refs = append(refs, "projection_only:true")
	}
	if workRef.LatestBlocker != nil {
		refs = append(refs, labeledNonEmpty("latest_blocker", workRef.LatestBlocker.Reason))
		refs = append(refs, labeledNonEmpty("latest_blocker_detail", workRef.LatestBlocker.Detail))
		refs = append(refs, prefixedStrings("latest_blocker_evidence", workRef.LatestBlocker.EvidenceRefs)...)
	}
	if workRef.LatestGate != nil {
		refs = append(refs, labeledNonEmpty("latest_gate", workRef.LatestGate.Gate))
		refs = append(refs, prefixedStrings("latest_gate_evidence", workRef.LatestGate.EvidenceRefs)...)
	}
	refs = append(refs, prefixedStrings("requirement", workRef.RequirementIDs)...)
	refs = append(refs, prefixedStrings("acceptance_criterion", workRef.AcceptanceCriterionIDs)...)
	refs = append(refs, prefixedStrings("missing_gate", workRef.MissingGates)...)
	refs = append(refs, prefixedStrings("missing_fact", workRef.MissingFacts)...)
	refs = append(refs, prefixedStrings("source_issue", workRef.SourceIssueRefs)...)
	refs = append(refs, consoleSourceMarkerEvidenceRefsWithPrefix("verification", workRef.VerificationRefs)...)
	refs = append(refs, consoleSourceMarkerEvidenceRefsWithPrefix("failure_repair", workRef.FailureRepairRefs)...)
	refs = append(refs, prefixedStrings("authority_exclusion", workRef.AuthorityExclusions)...)
	refs = sortedNonEmpty(refs)
	return refs
}

func consoleSourceMarkerEventGraphRefs(marker OpsCivilizationIssueScanSourceMarkerProjected) []string {
	refs := []string{
		labeledNonEmpty("projection_kind", marker.ProjectionKind),
		labeledNonEmpty("canonical_source", marker.CanonicalSource),
		labeledNonEmpty("schema_version", marker.SchemaVersion),
		labeledNonEmpty("run_id", marker.RunID),
		labeledNonEmpty("stage_id", marker.StageID),
		labeledNonEmpty("gate", marker.Gate),
	}
	if marker.ProjectionOnly {
		refs = append(refs, "projection_only:true")
	}
	refs = append(refs, marker.SourceRefs...)
	refs = append(refs, prefixedStrings("authority_exclusion", marker.AuthorityExclusions)...)
	return sortedNonEmpty(refs)
}

func consoleSourceMarkerEvidenceRefs(refs OpsCivilizationIssueScanMarkerEvidenceRefs) []string {
	return consoleSourceMarkerEvidenceRefsWithPrefix("", refs)
}

func consoleSourceMarkerEvidenceRefsWithPrefix(channel string, refs OpsCivilizationIssueScanMarkerEvidenceRefs) []string {
	prefix := func(kind string) string {
		channel = strings.TrimSpace(channel)
		if channel == "" {
			return kind
		}
		return channel + "_" + kind
	}
	out := []string{}
	out = append(out, prefixedStrings(prefix("test_case"), refs.TestCaseIDs)...)
	out = append(out, prefixedStrings(prefix("test_run"), refs.TestRunIDs)...)
	out = append(out, prefixedStrings(prefix("gate_result"), refs.GateResultIDs)...)
	out = append(out, prefixedStrings(prefix("failure"), refs.FailureIDs)...)
	out = append(out, prefixedStrings(prefix("repair_attempt"), refs.RepairAttemptIDs)...)
	out = append(out, prefixedStrings(prefix("waiver"), refs.WaiverIDs)...)
	return sortedNonEmpty(out)
}

func consoleSourceMarkerGitHubIssueLabel(marker OpsCivilizationIssueScanGitHubMarkerRef) string {
	repo := strings.TrimSpace(marker.Repository)
	if repo != "" && marker.IssueNumber > 0 {
		return fmt.Sprintf("%s#%d", repo, marker.IssueNumber)
	}
	return "not projected"
}

func labeledNonEmpty(label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.TrimSpace(label) + ":" + value
}

func prefixedStrings(prefix string, values []string) []string {
	prefix = strings.TrimSpace(prefix)
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if prefix == "" {
			out = append(out, value)
		} else {
			out = append(out, prefix+":"+value)
		}
	}
	sort.Strings(out)
	return out
}
