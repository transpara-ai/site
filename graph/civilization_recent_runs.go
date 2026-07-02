package graph

// OpsCivilizationRecentIssueScanRuns mirrors hive's
// CivilizationRecentIssueScanRuns JSON shape exactly (verified against the
// hive worktree file pkg/hive/civilization_recent_issue_scan.go). It is
// additive/omitempty on the projection (schema 1.6.0) so a 1.5.0 payload
// decodes with a zero value here and the section is simply absent.
type OpsCivilizationRecentIssueScanRuns struct {
	Status    string                              `json:"status"` // "available" | "unavailable"
	Summary   string                              `json:"summary,omitempty"`
	Truncated bool                                `json:"truncated,omitempty"`
	Runs      []OpsCivilizationRecentIssueScanRun `json:"runs,omitempty"`
}

// OpsCivilizationRecentIssueScanRun mirrors hive's
// CivilizationRecentIssueScanRun JSON shape exactly, field-for-field and
// tag-for-tag.
type OpsCivilizationRecentIssueScanRun struct {
	RunID          string   `json:"run_id"`
	FactoryOrderID string   `json:"factory_order_id,omitempty"`
	Repo           string   `json:"repo"`
	IssueNumber    int      `json:"issue_number"`
	IssueURL       string   `json:"issue_url,omitempty"`
	IssueTitle     string   `json:"issue_title,omitempty"`
	State          string   `json:"state"` // parked|human_action|queued|in_flight|recorded
	FirstEventAt   string   `json:"first_event_at,omitempty"`
	LastEventAt    string   `json:"last_event_at,omitempty"`
	BlockerType    string   `json:"blocker_type,omitempty"`
	RequiredAction string   `json:"required_action,omitempty"`
	StageID        string   `json:"stage_id,omitempty"`
	SourceRefs     []string `json:"source_refs,omitempty"`
}
