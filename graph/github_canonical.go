package graph

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type OpsGitHubCanonicalData struct {
	GeneratedAt       string
	ScannerSnapshotAt string
	ProjectionSource  string
	Status            string
	ProjectionState   string
	Parent            OpsGitHubCanonicalIssue
	AutonomyFrontier  OpsGitHubCanonicalAutonomyFrontier
	RepoSummaries     []OpsGitHubCanonicalRepoSummary
	Lanes             []OpsGitHubCanonicalLane
	IssueWarnings     []OpsGitHubCanonicalIssueWarning
	EvidenceRecords   []OpsGitHubCanonicalEvidenceRecord
	CutoverChecks     []OpsGitHubCanonicalCutoverCheck
	Boundaries        []string
	LegacyEvidence    []OpsGitHubCanonicalLegacyEvidence
}

type OpsGitHubCanonicalAutonomyFrontier struct {
	Recommendation              string
	TotalIssueCount             int
	CandidateBundleCount        int
	ReviewGroupCount            int
	SingletonCount              int
	IssueShapeWarningCount      int
	PRReadyIssueCount           int
	AutonomousPRReadyIssueCount int
	NeedsHumanScopeIssueCount   int
	ProtectedActionIssueCount   int
	DeferredIssueCount          int
	BlockerRefs                 []string
	EvidenceRefs                []string
	Boundary                    string
}

type OpsGitHubCanonicalIssue struct {
	Repo   string
	Number int
	Title  string
	URL    string
}

type OpsGitHubCanonicalRepoSummary struct {
	Repo          string
	Total         int
	Completed     int
	Ready         int
	Deferred      int
	HumanScope    int
	Protected     int
	LegacyOnly    int
	BlockedReason string
}

type OpsGitHubCanonicalLane struct {
	Issue         OpsGitHubCanonicalIssue
	ParentRef     string
	Substrate     string
	State         string
	Readiness     string
	Risk          string
	BlockedReason string
	Labels        []string
	EvidenceRefs  []string
	LegacyRefs    []string
}

type OpsGitHubCanonicalIssueWarning struct {
	Issue          OpsGitHubCanonicalIssue
	MissingFields  []string
	Recommendation string
	EvidenceRefs   []string
	Boundary       string
}

type OpsGitHubCanonicalEvidenceRecord struct {
	Name                    string
	EventType               string
	Outcome                 string
	Schema                  string
	SourceIssueRefs         []string
	PRRefs                  []string
	ValidationRefs          []string
	CFARRefs                []string
	AuthorityBoundaryRefs   []string
	ResidualRiskRefs        []string
	ProvenanceRefs          []string
	TraceScoreBasisPoints   int
	ProjectionReadinessNote string
}

type OpsGitHubCanonicalCutoverCheck struct {
	Label    string
	State    string
	Evidence string
	Blocker  string
}

type OpsGitHubCanonicalLegacyEvidence struct {
	Ref         string
	State       string
	Disposition string
}

const (
	githubCanonicalStateCompleted          = "completed"
	githubCanonicalStateReady              = "pr-ready"
	githubCanonicalStateDeferred           = "deferred"
	githubCanonicalStateNeedsHumanScope    = "needs-human-scope"
	githubCanonicalStateProtectedAction    = "protected-action"
	githubCanonicalStateLegacyEvidenceOnly = "legacy-evidence-only"

	// Keep this equal to the latest scanner: timestamp represented in monitor evidence.
	githubCanonicalScannerSnapshotAt = "2026-06-26T12:08:57Z"
	githubCanonicalProjectionSource  = "static transcription of scanner evidence; request render is not a live GitHub scan"
)

func buildOpsGitHubCanonicalData(now time.Time) *OpsGitHubCanonicalData {
	lanes := []OpsGitHubCanonicalLane{
		githubCanonicalLane("transpara-ai/docs", 197, "Development Arc issue-source migration parent tracker", "https://github.com/transpara-ai/docs/issues/197", "parent", "cross-repo governance coordination and scanner evidence", githubCanonicalStateDeferred, "docs PR deferred until EventGraph projection/write governance and scanner evidence are complete", "protected-action", "docs closeout PR cannot mark markdown superseded until EventGraph projection-store/write governance lands", []string{"cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"scanner:2026-06-26T12:08:57Z autonomy_frontier:park-autonomy-no-pr-ready-work", "site#135", "site#143", "site#145", "operation#34", "hive#220", "hive#221", "hive#232", "eventgraph#69", "docs#198", "docs#199", "docs#206", "platform#7", "platform#19", "https://github.com/transpara-ai/docs/issues/197#issuecomment-4809241930", "https://github.com/transpara-ai/docs/issues/197#issuecomment-4809411010"}, []string{"dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md"}),
		githubCanonicalLane("transpara-ai/docs", 193, "Coordinate deployed public-reader and public-correction evidence event", "https://github.com/transpara-ai/docs/issues/193", "docs#197", "Public-reader freshness/correction evidence and user-facing proof boundary", githubCanonicalStateNeedsHumanScope, "requires docs AuthorityDecision selecting repo(s), routes, data sources, privacy boundary, and validation commands", "protected-action", "public-reader/public-correction implementation is still human-scope blocked", []string{"cc:intake", "cc:needs-human-scope", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/docs/issues/193#issuecomment-4806459443"}, nil),
		githubCanonicalLane("transpara-ai/docs", 199, "Protected execution authorization design packet", "https://github.com/transpara-ai/docs/issues/199", "docs#197", "Protected execution boundary and AuthorityDecision packet", githubCanonicalStateCompleted, "closed by docs PR #205", "protected-action", "", []string{"cc:intake", "cc:pr-ready", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#199", "https://github.com/transpara-ai/docs/pull/205", "merge:874980a7ab6d1b5c6ef3bacfc8c02f1401f00a13", "reviewed_head:2c76e779e51b004db4004f81117cfcb6dd3e3638", "https://github.com/transpara-ai/docs/pull/205#issuecomment-4808081439"}, nil),
		githubCanonicalLane("transpara-ai/docs", 198, "Gate K go-live revalidation source issue", "https://github.com/transpara-ai/docs/issues/198", "docs#197", "docs-only Gate K revalidation evidence record", githubCanonicalStateCompleted, "closed by docs PR #206; evidence capture only, no go-live authority", "protected-action", "", []string{"cc:intake", "cc:pr-ready", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#198", "https://github.com/transpara-ai/docs/pull/206", "merge:87b0337f380b7e6ec9beb3c5be6dc7c0c5ec8ee8", "reviewed_head:c9b1274e70173c3b29c5ee4a03805852a9a65d30", "https://github.com/transpara-ai/docs/pull/206#issuecomment-4809200144", "https://github.com/transpara-ai/docs/issues/197#issuecomment-4809241930"}, nil),
		githubCanonicalLane("transpara-ai/work", 61, "Requirements and task-draft derivation from issue records", "https://github.com/transpara-ai/work/issues/61", "docs#197", "Work proposal evidence", githubCanonicalStateCompleted, "merged by PR #71", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#61", "work PR #71"}, nil),
		githubCanonicalLane("transpara-ai/work", 62, "Proof-of-work packet linked to issue source records", "https://github.com/transpara-ai/work/issues/62", "docs#197", "Work proof-of-work packet contract", githubCanonicalStateCompleted, "merged by PR #72", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#62", "work PR #72"}, nil),
		githubCanonicalLane("transpara-ai/work", 63, "AuditReport closeout linked to GitHub issue source records", "https://github.com/transpara-ai/work/issues/63", "docs#197", "Work AuditReport closeout evidence", githubCanonicalStateCompleted, "merged by PR #73", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#63", "work PR #73"}, nil),
		githubCanonicalLane("transpara-ai/work", 59, "Runtime-envelope follow-on for governed runtime observation", "https://github.com/transpara-ai/work/issues/59", "docs#197", "Work runtime-envelope evidence and governed runtime observation path", githubCanonicalStateNeedsHumanScope, "requires docs AuthorityDecision granting precise Work lifecycle and allowed paths", "protected-action", "runtime-envelope implementation is still human-scope blocked", []string{"cc:intake", "cc:needs-human-scope", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/work/issues/59#issuecomment-4806453322"}, nil),
		githubCanonicalLane("transpara-ai/site", 127, "GitHub-canonical issue migration progress surface", "https://github.com/transpara-ai/site/issues/127", "docs#197", "Site operator UI and read-only migration progress projection", githubCanonicalStateCompleted, "merged by PR #128", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"site#127", "https://github.com/transpara-ai/site/pull/128", "merge:07cd69f730faf93ed7e9e03ed74c2836db9dc62c", "docs#197"}, nil),
		githubCanonicalLane("transpara-ai/site", 129, "Typed projection-backed GitHub-canonical monitor", "https://github.com/transpara-ai/site/issues/129", "docs#197", "Site ops monitor and typed projection-shaped migration evidence view", githubCanonicalStateCompleted, "merged by PR #130; refreshed by site#131, site#133, site#135, site#139, site#143, site#145, and site#153", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "docs#199", "eventgraph#63", "eventgraph#69", "hive#232", "platform#7", "platform#19", "https://github.com/transpara-ai/site/pull/130", "https://github.com/transpara-ai/site/pull/132", "https://github.com/transpara-ai/site/pull/144", "https://github.com/transpara-ai/site/pull/146", "https://github.com/transpara-ai/platform/pull/19", "merge:cf3dcddbb06d47199dd7c94662e329422d27d10c", "merge:c3dc3a63eb16eafed490b7e6be28affe3469f7ea", "merge:885d8f14fbcf15c6d5ae1b67d88a3f40a7d9104d", "merge:fac357e0836adc54a65f1778c229a44bd3f0d364", "merge:e6691b62c4fd98179441f0085f23ab1c7c9a2f52", "reviewed_head:80c979a8c969e8c3f10511f4de10aadef783be9f", "reviewed_head:488bf95db116c0555757c7781173fd41923599e2", "https://github.com/transpara-ai/site/pull/146#issuecomment-4808512003", "https://github.com/transpara-ai/platform/pull/19#issuecomment-4809397170", "site#131", "site#133", "site#135", "site#139", "site#143", "site#145", "site#153"}, nil),
		githubCanonicalLane("transpara-ai/platform", 5, "Arc issue duplicate and stale-source detection", "https://github.com/transpara-ai/platform/issues/5", "docs#197", "platform scanner/read-only validation rule", githubCanonicalStateCompleted, "merged by PR #8; latest duplicate-anchor scan returned Findings: 0", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"platform#5", "https://github.com/transpara-ai/platform/pull/8", "merge:566c518893ba152f93339082b4b94b4a6140aed2", "arc_issue_scan:Findings=0"}, []string{"dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md"}),
		githubCanonicalLane("transpara-ai/platform", 15, "Report change-control issue-shape warnings in aggregation scanner", "https://github.com/transpara-ai/platform/issues/15", "docs#197", "platform scanner issue-shape warning output", githubCanonicalStateCompleted, "merged by PR #16; scanner:2026-06-26T06:52:57Z reports issue_shape_warnings=[] while docs#172 and operation#26 remain open residual/protected trackers", "normal", "", []string{"cc:intake", "cc:pr-ready", "cc:civilization-presence"}, []string{"platform#15", "https://github.com/transpara-ai/platform/pull/16", "merge:c9b27259feceb4a7e4113afbbf36364cc84cde9d", "scanner:2026-06-26T06:52:57Z issue_shape_warnings:none", "docs#172 open deferred protected human-scope", "operation#26 open deferred protected human-scope"}, nil),
		githubCanonicalLane("transpara-ai/platform", 17, "Emit autonomy frontier summary from change-control aggregation scanner", "https://github.com/transpara-ai/platform/issues/17", "docs#197", "platform change-control aggregation scanner output contract and tests", githubCanonicalStateCompleted, "merged by PR #18; scanner:2026-06-26T07:34:14Z reports autonomy_frontier recommendation=park-autonomy-no-pr-ready-work with pr_ready_issue_count=0, autonomous_pr_ready_issue_count=0, candidate_bundle_count=0, issue_shape_warning_count=0", "normal", "", []string{"cc:intake", "cc:pr-ready", "cc:civilization-presence"}, []string{"platform#17", "https://github.com/transpara-ai/platform/pull/18", "merge:b4ba2f98254ff32360dfcb490eb86e4613d8999d", "reviewed_head:7d4da36507fc62e979c6d3a46efd005126d33f53", "scanner:2026-06-26T07:34:14Z autonomy_frontier:park-autonomy-no-pr-ready-work", "scanner:2026-06-26T07:34:14Z pr_ready_issue_count:0 autonomous_pr_ready_issue_count:0 candidate_bundle_count:0 issue_shape_warning_count:0"}, nil),
		githubCanonicalLane("transpara-ai/platform", 7, "Stage 2 and Stage 3 plumbing automation with model-family separation", "https://github.com/transpara-ai/platform/issues/7", "docs#197", "platform Stage 2/3 design boundary and model-family separation", githubCanonicalStateCompleted, "closed by platform PR #19; future implementation remains follow-on issue/AuthorityDecision scoped", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"platform#7", "https://github.com/transpara-ai/platform/pull/19", "merge:e6691b62c4fd98179441f0085f23ab1c7c9a2f52", "reviewed_head:488bf95db116c0555757c7781173fd41923599e2", "https://github.com/transpara-ai/platform/pull/19#issuecomment-4809397170", "https://github.com/transpara-ai/docs/issues/197#issuecomment-4809411010"}, nil),
		githubCanonicalLane("transpara-ai/.github", 3, "Change-control issue form arc-anchor field upgrade", "https://github.com/transpara-ai/.github/issues/3", "docs#197", "organization issue template and issue-first intake form", githubCanonicalStateCompleted, "merged by PR #4", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{".github#3", "https://github.com/transpara-ai/.github/pull/4", "merge:30cd2f25f6e0008c8d4b9fb412e66ce1e6c7bc8e"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 63, "Native TestRun GateResult and AuditReport persistence contract", "https://github.com/transpara-ai/eventgraph/issues/63", "docs#197", "EventGraph native evidence content contract", githubCanonicalStateCompleted, "merged by PR #67", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"eventgraph#63", "https://github.com/transpara-ai/eventgraph/pull/67", "merge:c6f261a27a193a470a9e287d15580a05d1b0fafc", "evidence.testrun.recorded", "evidence.gateresult.recorded", "evidence.auditreport.recorded"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 62, "Authority evidence schema and store governance", "https://github.com/transpara-ai/eventgraph/issues/62", "docs#197", "EventGraph authority/evidence schema and migration governance", githubCanonicalStateCompleted, "merged by PR #68; authority schema/store-governance substrate only", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "eventgraph#63", "https://github.com/transpara-ai/eventgraph/pull/68", "merge:fd6f80253791e7500b2d43cae421d0a9701ae221"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 69, "Codex CLI model-config aliases and resolver override support", "https://github.com/transpara-ai/eventgraph/issues/69", "docs#197", "EventGraph modelconfig catalog/resolver support for Codex CLI issue-scan routing", githubCanonicalStateCompleted, "merged by PR #70; modelconfig support only, no runtime execution", "protected-action", "", []string{"cc:intake", "cc:pr-ready", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/eventgraph/pull/70", "merge:ec22be652d0f117c68393104ad911042fc5cc272", "https://github.com/transpara-ai/eventgraph/pull/70#issuecomment-4806223812"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 59, "Persistent EventGraph projection-store event for Civilization Assembly truth", "https://github.com/transpara-ai/eventgraph/issues/59", "docs#197", "EventGraph durable projection truth and Civilization Assembly provenance", githubCanonicalStateNeedsHumanScope, "requires durable projection authority and write-path boundary", "protected-action", "projection store event is not authorized for production writes", []string{"cc:intake", "cc:needs-human-scope", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/eventgraph/issues/59#issuecomment-4806453286"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 61, "Production EventGraph write path for runtime and issue evidence", "https://github.com/transpara-ai/eventgraph/issues/61", "docs#197", "EventGraph persistent write path and evidence truth", githubCanonicalStateNeedsHumanScope, "requires governed authority packet before PR work", "protected-action", "production write path still human-scope blocked", []string{"cc:intake", "cc:needs-human-scope", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#200"}, nil),
		githubCanonicalLane("transpara-ai/hive", 220, "AuthorityDecision evaluation from issue scope evidence", "https://github.com/transpara-ai/hive/issues/220", "docs#197", "Hive authority recommendation semantics", githubCanonicalStateCompleted, "merged by PR #231; recommendation-only policy does not create AuthorityDecision records", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/hive/pull/231", "merge:523181b83ad8540fba747a64a12975996db170a4", "authority_recommendation_policy"}, nil),
		githubCanonicalLane("transpara-ai/hive", 221, "Human-required protected-action classification surface", "https://github.com/transpara-ai/hive/issues/221", "docs#197", "Hive authority and human-required classification", githubCanonicalStateCompleted, "merged by PR #230; human-required classification policy persisted/projected", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/hive/pull/230", "merge:0c0fdb5f9c116cef99ed87ed9f31bfc5cbd9e10e", "human_required_classification_policy"}, nil),
		githubCanonicalLane("transpara-ai/hive", 222, "Scanner recommender tackler role separation policy", "https://github.com/transpara-ai/hive/issues/222", "docs#197", "Hive model policy and role authority semantics", githubCanonicalStateCompleted, "merged by PR #228; role-separation policy persisted/projected", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/hive/pull/228", "merge:567af4a15b51f77468597680914f067e5c357705", "role_separation_policy"}, nil),
		githubCanonicalLane("transpara-ai/hive", 223, "Autonomy-increase guard for issue-driven recommendations", "https://github.com/transpara-ai/hive/issues/223", "docs#197", "Hive autonomy boundary enforcement", githubCanonicalStateCompleted, "merged by PR #229; autonomy guard policy persisted/projected", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/hive/pull/229", "merge:b114f9862545ca6152474d9a06155c0c5fc58c34", "autonomy_guard_policy"}, nil),
		githubCanonicalLane("transpara-ai/hive", 232, "Issue-scan model override flags for Codex-capable launches", "https://github.com/transpara-ai/hive/issues/232", "docs#197", "Hive issue-scan model routing and queued launch metadata", githubCanonicalStateCompleted, "merged by PR #233; model-routing metadata only, no runtime start or live dispatch", "protected-action", "", []string{"cc:intake", "cc:pr-ready", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/hive/pull/233", "merge:89921d82d5019f2181e2b75435019c19e9ab92c9", "https://github.com/transpara-ai/hive/pull/233#issuecomment-4806413483", "codex-cli/gpt-5.5 operate"}, nil),
		githubCanonicalLane("transpara-ai/operation", 34, "Clean suspend and bus-factor runbook for issue-source workflow", "https://github.com/transpara-ai/operation/issues/34", "docs#197", "operation runbook and continuity procedure", githubCanonicalStateCompleted, "merged by PR #37; issue-source workflow continuity runbook complete", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/operation/pull/37", "merge:326f90a49d986e66d171e0eb0b5be23b8e64324c", "operation#34", "operation_runbook"}, nil),
	}
	legacy := []OpsGitHubCanonicalLegacyEvidence{
		{Ref: "dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md", State: githubCanonicalStateLegacyEvidenceOnly, Disposition: "Historical source evidence until docs#197 cutover; not the live work queue."},
	}
	lanes = append(lanes, OpsGitHubCanonicalLane{
		Issue:         OpsGitHubCanonicalIssue{Repo: "legacy-markdown", Title: "Development/design arc markdown", URL: "dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md"},
		ParentRef:     "docs#197",
		Substrate:     "legacy source evidence",
		State:         githubCanonicalStateLegacyEvidenceOnly,
		Readiness:     "superseded only after docs#197 closeout criteria pass",
		Risk:          "historical",
		BlockedReason: "markdown cannot be retired until child issues and typed projections cover all live obligations",
		Labels:        []string{"legacy-evidence-only"},
		LegacyRefs:    []string{"docs#197"},
	})

	return &OpsGitHubCanonicalData{
		GeneratedAt:       now.UTC().Format(time.RFC3339),
		ScannerSnapshotAt: githubCanonicalScannerSnapshotAt,
		ProjectionSource:  githubCanonicalProjectionSource,
		Status:            "partial",
		ProjectionState:   "typed projection-shaped Site contract; static until EventGraph store governance is authorized",
		Parent:            OpsGitHubCanonicalIssue{Repo: "transpara-ai/docs", Number: 197, Title: "Development Arc issue-source migration parent tracker", URL: "https://github.com/transpara-ai/docs/issues/197"},
		RepoSummaries:     githubCanonicalRepoSummaries(lanes),
		Lanes:             lanes,
		IssueWarnings:     githubCanonicalIssueWarnings(),
		EvidenceRecords:   githubCanonicalEvidenceRecords(),
		CutoverChecks: []OpsGitHubCanonicalCutoverCheck{
			{Label: "Issue coverage", State: "partial", Evidence: "docs#197 and child issue lanes exist; scanner:2026-06-26T12:08:57Z found no PR-ready issues, no autonomous PR-ready issues, no candidate bundles, and no issue-shape warnings after platform#7 closed", Blocker: "remaining protected lanes are human-scope"},
			{Label: "Work traceability", State: "completed", Evidence: "work#61, work#62, work#63 merged", Blocker: ""},
			{Label: "Issue form schema", State: "completed", Evidence: ".github#3 merged by PR #4", Blocker: ""},
			{Label: "Duplicate stale-source scanner", State: "completed", Evidence: "platform#5 merged by PR #8; latest arc_issue_scan returned Findings: 0", Blocker: ""},
			{Label: "Issue-shape warning scanner", State: "completed", Evidence: "platform#15 merged by PR #16; scanner:2026-06-26T06:52:57Z reports issue_shape_warnings=[] while docs#172 and operation#26 remain open residual/protected trackers", Blocker: ""},
			{Label: "Autonomy frontier", State: githubCanonicalStateCompleted, Evidence: "platform#17 merged by PR #18; post-platform#7 autonomy_frontier remains park-autonomy-no-pr-ready-work with zero PR-ready, autonomous PR-ready, candidate-bundle, and issue-shape-warning counts", Blocker: ""},
			{Label: "Native evidence content", State: "completed", Evidence: "eventgraph#63 merged by PR #67; TestRun, GateResult, and AuditReport content types registered", Blocker: ""},
			{Label: "EventGraph authority projection", State: githubCanonicalStateNeedsHumanScope, Evidence: "eventgraph#62 and eventgraph#69 merged; eventgraph#59 and eventgraph#61 remain open", Blocker: "projection-store truth and production write path still not authorized"},
			{Label: "Hive issue intake", State: "completed", Evidence: "hive#220, hive#221, hive#222, hive#223, and hive#232 merged with policy/model-routing CFAR evidence", Blocker: ""},
			{Label: "Operation continuity", State: "completed", Evidence: "operation#34 merged by PR #37; issue-source workflow runbook merged", Blocker: ""},
			{Label: "Site typed monitor", State: "completed", Evidence: "site#129 merged by PR #130; site#131/site#133/site#135/site#139 merged; site#143 refresh records docs#199 completion; latest refresh records docs#198 and platform#7 closeout evidence", Blocker: ""},
			{Label: "Markdown retirement", State: githubCanonicalStateLegacyEvidenceOnly, Evidence: "docs#197 closeout pending", Blocker: "legacy arc remains background evidence only"},
		},
		AutonomyFrontier: githubCanonicalAutonomyFrontier(),
		Boundaries: []string{
			"Read-only typed projection-shaped Site data; no live GitHub fetch or mutation.",
			"Rendered-at time is request freshness only; scanner_snapshot_at is the latest scan represented by autonomy frontier and issue-shape state; individual evidence rows may cite earlier confirming scans.",
			"No Hive wake, runtime start, queue launch, EventGraph write, deploy, merge, approval, Test 001 GREEN claim, autonomy increase, value allocation, or residual-risk closure.",
			"Markdown is displayed only as archived/background evidence, never as the live work queue.",
			"Issue-shape warning state is scanner evidence only; warning absence does not authorize issue closure, PR-ready state, Hive wake, GitHub mutation, Test 001 GREEN, or residual-risk closure.",
			"Autonomy frontier state is parking evidence only; park-autonomy-no-pr-ready-work does not close docs#197, authorize protected lanes, or wake Hive.",
			"source_issue_refs, validation_refs, cfar_refs, authority_boundary_refs, residual_risk_refs, and trace_score_basis_points are display contracts only until EventGraph write-path governance lands.",
		},
		LegacyEvidence: legacy,
	}
}

func githubCanonicalAutonomyFrontier() OpsGitHubCanonicalAutonomyFrontier {
	return OpsGitHubCanonicalAutonomyFrontier{
		Recommendation:              "park-autonomy-no-pr-ready-work",
		TotalIssueCount:             14,
		CandidateBundleCount:        0,
		ReviewGroupCount:            0,
		SingletonCount:              14,
		IssueShapeWarningCount:      0,
		PRReadyIssueCount:           0,
		AutonomousPRReadyIssueCount: 0,
		NeedsHumanScopeIssueCount:   13,
		ProtectedActionIssueCount:   14,
		DeferredIssueCount:          13,
		BlockerRefs: []string{
			"transpara-ai/docs#172",
			"transpara-ai/docs#193",
			"transpara-ai/docs#197",
			"transpara-ai/docs#200",
			"transpara-ai/docs#201",
			"transpara-ai/docs#202",
			"transpara-ai/docs#203",
			"transpara-ai/eventgraph#59",
			"transpara-ai/eventgraph#61",
			"transpara-ai/hive#204",
			"transpara-ai/operation#26",
			"transpara-ai/operation#35",
			"transpara-ai/work#59",
			"transpara-ai/work#64",
		},
		EvidenceRefs: []string{
			"platform#17",
			"docs#198",
			"https://github.com/transpara-ai/docs/pull/206",
			"platform#7",
			"https://github.com/transpara-ai/platform/pull/19",
			"https://github.com/transpara-ai/platform/pull/18",
			"merge:87b0337f380b7e6ec9beb3c5be6dc7c0c5ec8ee8",
			"merge:b4ba2f98254ff32360dfcb490eb86e4613d8999d",
			"merge:e6691b62c4fd98179441f0085f23ab1c7c9a2f52",
			"reviewed_head:c9b1274e70173c3b29c5ee4a03805852a9a65d30",
			"reviewed_head:7d4da36507fc62e979c6d3a46efd005126d33f53",
			"reviewed_head:488bf95db116c0555757c7781173fd41923599e2",
			"scanner:2026-06-26T12:08:57Z autonomy_frontier:park-autonomy-no-pr-ready-work",
			"scanner:2026-06-26T12:08:57Z issue_shape_warning_count:0",
			"scanner:2026-06-26T12:08:57Z total_issue_count:14 needs_human_scope_issue_count:13 protected_action_issue_count:14 deferred_issue_count:13",
			"scanner:2026-06-26T12:08:57Z blocker_refs:transpara-ai/docs#172,transpara-ai/docs#193,transpara-ai/docs#197,transpara-ai/docs#200,transpara-ai/docs#201,transpara-ai/docs#202,transpara-ai/docs#203,transpara-ai/eventgraph#59,transpara-ai/eventgraph#61,transpara-ai/hive#204,transpara-ai/operation#26,transpara-ai/operation#35,transpara-ai/work#59,transpara-ai/work#64",
			"arc_issue_scan:Findings=0",
		},
		Boundary: "Read-only scanner projection. This parks autonomous work when no issue is PR-ready; it does not authorize protected actions, issue edits, Hive wake, EventGraph writes, Test 001 GREEN, residual-risk closure, value allocation, or markdown retirement.",
	}
}

func githubCanonicalIssueWarnings() []OpsGitHubCanonicalIssueWarning {
	return nil
}

func githubCanonicalEvidenceRecords() []OpsGitHubCanonicalEvidenceRecord {
	return []OpsGitHubCanonicalEvidenceRecord{
		{
			Name:                    "GitHub canonical validation TestRun",
			EventType:               "evidence.testrun.recorded",
			Outcome:                 "tests.pass",
			Schema:                  "TestRun",
			SourceIssueRefs:         []string{"docs#197", "docs#198", "docs#199", "site#129", "site#131", "site#133", "site#135", "site#139", "site#143", "site#145", "eventgraph#63", "eventgraph#69", "hive#232", "platform#7", "platform#15", "platform#17"},
			PRRefs:                  []string{"docs PR #205", "docs PR #206", "eventgraph PR #67", "eventgraph PR #70", "hive PR #233", "site PR #130", "site PR #132", "site PR #144", "site PR #146", "platform PR #16", "platform PR #18", "platform PR #19"},
			ValidationRefs:          []string{"go test ./pkg/event -run TestNativeEvidence -count=1", "make verify-go", "make verify", "docs#198 validation complete by docs PR #206", "site#143 validation complete", "site#145 validation complete by site PR #146", "platform#7 validation complete by platform PR #19", "platform scanner grouping comparison identical", "platform scanner issue_shape_warnings=[]", "platform scanner autonomy_frontier recommendation=park-autonomy-no-pr-ready-work"},
			CFARRefs:                []string{"docs PR #205/#206 CFAR PASS", "eventgraph PR #67/#70 CFAR PASS", "hive PR #233 CFAR PASS", "site PR #132/#144/#146 CFAR PASS", "platform PR #16/#18/#19 CFAR PASS"},
			AuthorityBoundaryRefs:   []string{"docs#197", "docs#199", "eventgraph#61", "docs#200"},
			ResidualRiskRefs:        []string{"docs#201", "docs#202", "docs#203"},
			ProvenanceRefs:          []string{"https://github.com/transpara-ai/docs/pull/205", "https://github.com/transpara-ai/docs/pull/206", "https://github.com/transpara-ai/eventgraph/pull/67", "https://github.com/transpara-ai/eventgraph/pull/70", "https://github.com/transpara-ai/hive/pull/233", "https://github.com/transpara-ai/site/pull/132", "https://github.com/transpara-ai/site/pull/144", "https://github.com/transpara-ai/site/pull/146", "https://github.com/transpara-ai/platform/pull/16", "https://github.com/transpara-ai/platform/pull/18", "https://github.com/transpara-ai/platform/pull/19", "merge:874980a7ab6d1b5c6ef3bacfc8c02f1401f00a13", "reviewed_head:2c76e779e51b004db4004f81117cfcb6dd3e3638", "merge:87b0337f380b7e6ec9beb3c5be6dc7c0c5ec8ee8", "reviewed_head:c9b1274e70173c3b29c5ee4a03805852a9a65d30", "merge:c6f261a27a193a470a9e287d15580a05d1b0fafc", "merge:ec22be652d0f117c68393104ad911042fc5cc272", "merge:89921d82d5019f2181e2b75435019c19e9ab92c9", "merge:c3dc3a63eb16eafed490b7e6be28affe3469f7ea", "merge:885d8f14fbcf15c6d5ae1b67d88a3f40a7d9104d", "merge:fac357e0836adc54a65f1778c229a44bd3f0d364", "reviewed_head:80c979a8c969e8c3f10511f4de10aadef783be9f", "merge:c9b27259feceb4a7e4113afbbf36364cc84cde9d", "merge:b4ba2f98254ff32360dfcb490eb86e4613d8999d", "merge:e6691b62c4fd98179441f0085f23ab1c7c9a2f52", "reviewed_head:488bf95db116c0555757c7781173fd41923599e2", "https://github.com/transpara-ai/docs/pull/205#issuecomment-4808081439", "https://github.com/transpara-ai/docs/pull/206#issuecomment-4809200144", "https://github.com/transpara-ai/eventgraph/pull/67#issuecomment-4803740786", "https://github.com/transpara-ai/eventgraph/pull/70#issuecomment-4806223812", "https://github.com/transpara-ai/hive/pull/233#issuecomment-4806413483", "https://github.com/transpara-ai/site/pull/144#issuecomment-4808318161", "https://github.com/transpara-ai/site/pull/146#issuecomment-4808512003", "https://github.com/transpara-ai/platform/pull/16#issuecomment-4806876411", "https://github.com/transpara-ai/platform/pull/18#issuecomment-4807330168", "https://github.com/transpara-ai/platform/pull/19#issuecomment-4809397170"},
			TraceScoreBasisPoints:   10000,
			ProjectionReadinessNote: "10000 bp covers content/model-routing validation only; projection store write path still human-scope blocked",
		},
		{
			Name:                  "GitHub canonical gate result",
			EventType:             "evidence.gateresult.recorded",
			Outcome:               "gate.partial",
			Schema:                "GateResult",
			SourceIssueRefs:       []string{"docs#197", "docs#193", "docs#198", "docs#199", "work#59", "work#61", "work#62", "work#63", "site#127", "site#129", "site#131", "site#133", "site#135", "site#139", "site#143", "site#145", "platform#5", "platform#7", "platform#15", "platform#17", ".github#3", "eventgraph#59", "eventgraph#63", "eventgraph#62", "eventgraph#69", "hive#220", "hive#221", "hive#222", "hive#223", "hive#232", "operation#34"},
			PRRefs:                []string{"https://github.com/transpara-ai/docs/pull/205", "https://github.com/transpara-ai/docs/pull/206", "https://github.com/transpara-ai/work/pull/71", "https://github.com/transpara-ai/work/pull/72", "https://github.com/transpara-ai/work/pull/73", "https://github.com/transpara-ai/site/pull/128", "https://github.com/transpara-ai/site/pull/130", "https://github.com/transpara-ai/site/pull/132", "https://github.com/transpara-ai/site/pull/144", "https://github.com/transpara-ai/site/pull/146", "https://github.com/transpara-ai/platform/pull/8", "https://github.com/transpara-ai/platform/pull/16", "https://github.com/transpara-ai/platform/pull/18", "https://github.com/transpara-ai/platform/pull/19", "https://github.com/transpara-ai/.github/pull/4", "https://github.com/transpara-ai/eventgraph/pull/67", "https://github.com/transpara-ai/eventgraph/pull/68", "https://github.com/transpara-ai/eventgraph/pull/70", "https://github.com/transpara-ai/hive/pull/228", "https://github.com/transpara-ai/hive/pull/229", "https://github.com/transpara-ai/hive/pull/230", "https://github.com/transpara-ai/hive/pull/231", "https://github.com/transpara-ai/hive/pull/233", "https://github.com/transpara-ai/operation/pull/37"},
			ValidationRefs:        []string{"change-control scan: no multi-issue bundles", "arc_issue_scan: Findings=0", "issue_shape_warnings:none", "autonomy_frontier:park-autonomy-no-pr-ready-work"},
			CFARRefs:              []string{"docs PR #205/#206 CFAR PASS", "work PR #71/#72/#73 CFAR PASS", "site PR #128/#130/#132/#144/#146 CFAR PASS", "platform PR #8/#16/#18/#19 CFAR PASS", ".github PR #4 CFAR PASS", "eventgraph PR #67/#68/#70 CFAR PASS", "hive PR #228/#229/#230/#231/#233 CFAR PASS", "operation PR #37 CFAR PASS"},
			AuthorityBoundaryRefs: []string{"docs#197", "docs#193", "docs#199", "docs#200", "eventgraph#59", "work#59"},
			ResidualRiskRefs:      []string{"docs#172", "operation#26", "operation#35"},
			ProvenanceRefs: []string{
				"work#71 merge:f118276665c0bbbea282be7803070948b8d8e297",
				"work#72 merge:b4fecec3003056cd20ffefa5992e7c5833a8b8eb",
				"work#73 merge:731aa1c367b7627ad69c531a07a8a0e302c3900f",
				"docs#205 merge:874980a7ab6d1b5c6ef3bacfc8c02f1401f00a13",
				"docs#206 merge:87b0337f380b7e6ec9beb3c5be6dc7c0c5ec8ee8",
				"site#128 merge:07cd69f730faf93ed7e9e03ed74c2836db9dc62c",
				"site#130 merge:cf3dcddbb06d47199dd7c94662e329422d27d10c",
				"site#132 merge:c3dc3a63eb16eafed490b7e6be28affe3469f7ea",
				"site#144 merge:885d8f14fbcf15c6d5ae1b67d88a3f40a7d9104d",
				"site#146 merge:fac357e0836adc54a65f1778c229a44bd3f0d364",
				"platform#8 merge:566c518893ba152f93339082b4b94b4a6140aed2",
				"platform#16 merge:c9b27259feceb4a7e4113afbbf36364cc84cde9d",
				"platform#18 merge:b4ba2f98254ff32360dfcb490eb86e4613d8999d",
				"platform#19 merge:e6691b62c4fd98179441f0085f23ab1c7c9a2f52",
				".github#4 merge:30cd2f25f6e0008c8d4b9fb412e66ce1e6c7bc8e",
				"eventgraph#67 merge:c6f261a27a193a470a9e287d15580a05d1b0fafc",
				"eventgraph#68 merge:fd6f80253791e7500b2d43cae421d0a9701ae221",
				"eventgraph#70 merge:ec22be652d0f117c68393104ad911042fc5cc272",
				"hive#228 merge:567af4a15b51f77468597680914f067e5c357705",
				"hive#229 merge:b114f9862545ca6152474d9a06155c0c5fc58c34",
				"hive#230 merge:0c0fdb5f9c116cef99ed87ed9f31bfc5cbd9e10e",
				"hive#231 merge:523181b83ad8540fba747a64a12975996db170a4",
				"hive#233 merge:89921d82d5019f2181e2b75435019c19e9ab92c9",
				"operation#37 merge:326f90a49d986e66d171e0eb0b5be23b8e64324c",
				"https://github.com/transpara-ai/docs/issues/197#issuecomment-4803515276",
				"https://github.com/transpara-ai/docs/issues/197#issuecomment-4808529248",
			},
			TraceScoreBasisPoints:   8500,
			ProjectionReadinessNote: "8500 bp is a display confidence score for the cited source-of-intent and model-routing substrate; protected human-scope lanes remain open",
		},
		{
			Name:                    "GitHub canonical closeout AuditReport",
			EventType:               "evidence.auditreport.recorded",
			Outcome:                 "closeout.blocked",
			Schema:                  "AuditReport",
			SourceIssueRefs:         []string{"docs#197", "docs#193", "docs#198", "docs#199", "work#59", "eventgraph#59", "eventgraph#61", "site#129", "site#131", "site#133", "site#135", "site#139", "site#143", "site#145", "platform#7"},
			PRRefs:                  []string{"https://github.com/transpara-ai/docs/pull/205", "https://github.com/transpara-ai/docs/pull/206", "https://github.com/transpara-ai/site/pull/144", "https://github.com/transpara-ai/site/pull/146", "https://github.com/transpara-ai/platform/pull/19", "pending final docs closeout PR"},
			ValidationRefs:          []string{"docs#198 validation complete by docs PR #206", "site#130 validation complete", "site#131 validation complete", "site#133 validation complete", "site#135 validation complete", "site#139 validation complete", "site#143 validation complete by site PR #144", "site#145 validation complete by site PR #146", "platform#7 validation complete by platform PR #19", "pending final docs closeout scan"},
			CFARRefs:                []string{"docs PR #205/#206 CFAR PASS", "site PR #130/#132/#134/#136/#138/#140/#142/#144/#146 CFAR PASS", "platform PR #19 CFAR PASS", "pending final docs closeout CFAR"},
			AuthorityBoundaryRefs:   []string{"docs#193", "docs#199", "eventgraph#59", "eventgraph#61", "docs#200", "work#59"},
			ResidualRiskRefs:        []string{"docs#201", "docs#202", "docs#203"},
			ProvenanceRefs:          []string{"https://github.com/transpara-ai/docs/pull/205", "https://github.com/transpara-ai/docs/pull/206", "https://github.com/transpara-ai/site/pull/144", "https://github.com/transpara-ai/site/pull/146", "https://github.com/transpara-ai/platform/pull/19", "merge:874980a7ab6d1b5c6ef3bacfc8c02f1401f00a13", "reviewed_head:2c76e779e51b004db4004f81117cfcb6dd3e3638", "merge:87b0337f380b7e6ec9beb3c5be6dc7c0c5ec8ee8", "reviewed_head:c9b1274e70173c3b29c5ee4a03805852a9a65d30", "site#144 merge:885d8f14fbcf15c6d5ae1b67d88a3f40a7d9104d", "site#144 reviewed_head:5d62b4d83c942795f49fd423aac29da9d0b897ea", "site#146 merge:fac357e0836adc54a65f1778c229a44bd3f0d364", "site#146 reviewed_head:80c979a8c969e8c3f10511f4de10aadef783be9f", "platform#19 merge:e6691b62c4fd98179441f0085f23ab1c7c9a2f52", "platform#19 reviewed_head:488bf95db116c0555757c7781173fd41923599e2", "https://github.com/transpara-ai/site/pull/144#issuecomment-4808318161", "https://github.com/transpara-ai/site/pull/146#issuecomment-4808512003", "https://github.com/transpara-ai/platform/pull/19#issuecomment-4809397170", "https://github.com/transpara-ai/site/issues/129", "https://github.com/transpara-ai/site/issues/131", "https://github.com/transpara-ai/site/issues/133", "https://github.com/transpara-ai/site/issues/135", "https://github.com/transpara-ai/site/issues/139", "https://github.com/transpara-ai/site/issues/143", "https://github.com/transpara-ai/site/issues/145", "https://github.com/transpara-ai/docs/issues/197", "https://github.com/transpara-ai/docs/issues/197#issuecomment-4808529248", "https://github.com/transpara-ai/docs/issues/197#issuecomment-4809241930", "https://github.com/transpara-ai/docs/issues/197#issuecomment-4809411010", "pending final docs closeout PR"},
			TraceScoreBasisPoints:   7000,
			ProjectionReadinessNote: "7000 bp is a blocked-closeout display score; canonical cutover stays blocked until human-scoped EventGraph/runtime/public-reader authority lands",
		},
	}
}

func githubCanonicalLane(repo string, number int, title string, url string, parentRef string, substrate string, state string, readiness string, risk string, blockedReason string, labels []string, evidenceRefs []string, legacyRefs []string) OpsGitHubCanonicalLane {
	return OpsGitHubCanonicalLane{
		Issue:         OpsGitHubCanonicalIssue{Repo: repo, Number: number, Title: title, URL: url},
		ParentRef:     parentRef,
		Substrate:     substrate,
		State:         state,
		Readiness:     readiness,
		Risk:          risk,
		BlockedReason: blockedReason,
		Labels:        sortedNonEmpty(labels),
		EvidenceRefs:  sortedNonEmpty(evidenceRefs),
		LegacyRefs:    sortedNonEmpty(legacyRefs),
	}
}

func githubCanonicalRepoSummaries(lanes []OpsGitHubCanonicalLane) []OpsGitHubCanonicalRepoSummary {
	byRepo := map[string]*OpsGitHubCanonicalRepoSummary{}
	for _, lane := range lanes {
		repo := strings.TrimSpace(lane.Issue.Repo)
		if repo == "" {
			repo = "unprojected"
		}
		summary := byRepo[repo]
		if summary == nil {
			summary = &OpsGitHubCanonicalRepoSummary{Repo: repo}
			byRepo[repo] = summary
		}
		summary.Total++
		switch lane.State {
		case githubCanonicalStateCompleted:
			summary.Completed++
		case githubCanonicalStateReady:
			summary.Ready++
		case githubCanonicalStateDeferred:
			summary.Deferred++
		case githubCanonicalStateNeedsHumanScope:
			summary.HumanScope++
		case githubCanonicalStateProtectedAction:
		case githubCanonicalStateLegacyEvidenceOnly:
			summary.LegacyOnly++
		}
		if strings.EqualFold(strings.TrimSpace(lane.Risk), githubCanonicalStateProtectedAction) || lane.State == githubCanonicalStateProtectedAction {
			summary.Protected++
		}
		if summary.BlockedReason == "" && strings.TrimSpace(lane.BlockedReason) != "" {
			summary.BlockedReason = lane.BlockedReason
		}
	}
	repos := make([]string, 0, len(byRepo))
	for repo := range byRepo {
		repos = append(repos, repo)
	}
	sort.Strings(repos)
	out := make([]OpsGitHubCanonicalRepoSummary, 0, len(repos))
	for _, repo := range repos {
		out = append(out, *byRepo[repo])
	}
	return out
}

func githubCanonicalIssueLabel(issue OpsGitHubCanonicalIssue) string {
	if issue.Repo != "" && issue.Number > 0 {
		return fmt.Sprintf("%s#%d", issue.Repo, issue.Number)
	}
	return opsCivilizationValue(issue.Title, "issue not projected")
}

func githubCanonicalStateClass(state string) string {
	switch strings.TrimSpace(state) {
	case githubCanonicalStateCompleted, "tests.pass":
		return "border-emerald-400/40 text-emerald-300 bg-emerald-400/10"
	case githubCanonicalStateReady:
		return "border-brand/40 text-brand bg-brand/10"
	case githubCanonicalStateNeedsHumanScope, "gate.partial":
		return "border-amber-300/40 text-amber-300 bg-amber-300/10"
	case githubCanonicalStateProtectedAction, "closeout.blocked":
		return "border-red-300/40 text-red-300 bg-red-300/10"
	case githubCanonicalStateLegacyEvidenceOnly:
		return "border-edge text-warm-faint bg-void/30"
	default:
		return "border-edge text-warm-muted bg-void/30"
	}
}
