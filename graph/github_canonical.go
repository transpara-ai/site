package graph

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type OpsGitHubCanonicalData struct {
	GeneratedAt     string
	Status          string
	ProjectionState string
	Parent          OpsGitHubCanonicalIssue
	RepoSummaries   []OpsGitHubCanonicalRepoSummary
	Lanes           []OpsGitHubCanonicalLane
	EvidenceRecords []OpsGitHubCanonicalEvidenceRecord
	CutoverChecks   []OpsGitHubCanonicalCutoverCheck
	Boundaries      []string
	LegacyEvidence  []OpsGitHubCanonicalLegacyEvidence
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
)

func buildOpsGitHubCanonicalData(now time.Time) *OpsGitHubCanonicalData {
	lanes := []OpsGitHubCanonicalLane{
		githubCanonicalLane("transpara-ai/docs", 197, "Development Arc issue-source migration parent tracker", "https://github.com/transpara-ai/docs/issues/197", "parent", "cross-repo governance coordination and scanner evidence", githubCanonicalStateDeferred, "docs PR deferred until EventGraph projection/write governance and scanner evidence are complete", "protected-action", "docs closeout PR cannot mark markdown superseded until EventGraph projection-store/write governance lands", []string{"cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"scanner:2026-06-26T03:12:27Z", "site#133", "operation#34", "hive#220", "hive#221"}, []string{"dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md"}),
		githubCanonicalLane("transpara-ai/work", 61, "Requirements and task-draft derivation from issue records", "https://github.com/transpara-ai/work/issues/61", "docs#197", "Work proposal evidence", githubCanonicalStateCompleted, "merged by PR #71", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#61", "work PR #71"}, nil),
		githubCanonicalLane("transpara-ai/work", 62, "Proof-of-work packet linked to issue source records", "https://github.com/transpara-ai/work/issues/62", "docs#197", "Work proof-of-work packet contract", githubCanonicalStateCompleted, "merged by PR #72", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#62", "work PR #72"}, nil),
		githubCanonicalLane("transpara-ai/work", 63, "AuditReport closeout linked to GitHub issue source records", "https://github.com/transpara-ai/work/issues/63", "docs#197", "Work AuditReport closeout evidence", githubCanonicalStateCompleted, "merged by PR #73", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#63", "work PR #73"}, nil),
		githubCanonicalLane("transpara-ai/site", 127, "GitHub-canonical issue migration progress surface", "https://github.com/transpara-ai/site/issues/127", "docs#197", "Site operator UI and read-only migration progress projection", githubCanonicalStateCompleted, "merged by PR #128", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"site#127", "https://github.com/transpara-ai/site/pull/128", "merge:07cd69f730faf93ed7e9e03ed74c2836db9dc62c", "docs#197"}, nil),
		githubCanonicalLane("transpara-ai/site", 129, "Typed projection-backed GitHub-canonical monitor", "https://github.com/transpara-ai/site/issues/129", "docs#197", "Site ops monitor and typed projection-shaped migration evidence view", githubCanonicalStateCompleted, "merged by PR #130; refreshed by site#131 and site#133", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "eventgraph#63", "https://github.com/transpara-ai/site/pull/130", "https://github.com/transpara-ai/site/pull/132", "merge:cf3dcddbb06d47199dd7c94662e329422d27d10c", "merge:c3dc3a63eb16eafed490b7e6be28affe3469f7ea", "site#131", "site#133"}, nil),
		githubCanonicalLane("transpara-ai/platform", 5, "Arc issue duplicate and stale-source detection", "https://github.com/transpara-ai/platform/issues/5", "docs#197", "platform scanner/read-only validation rule", githubCanonicalStateCompleted, "merged by PR #8; latest duplicate-anchor scan returned Findings: 0", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"platform#5", "https://github.com/transpara-ai/platform/pull/8", "merge:566c518893ba152f93339082b4b94b4a6140aed2", "arc_issue_scan:Findings=0"}, []string{"dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md"}),
		githubCanonicalLane("transpara-ai/.github", 3, "Change-control issue form arc-anchor field upgrade", "https://github.com/transpara-ai/.github/issues/3", "docs#197", "organization issue template and issue-first intake form", githubCanonicalStateCompleted, "merged by PR #4", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{".github#3", "https://github.com/transpara-ai/.github/pull/4", "merge:30cd2f25f6e0008c8d4b9fb412e66ce1e6c7bc8e"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 63, "Native TestRun GateResult and AuditReport persistence contract", "https://github.com/transpara-ai/eventgraph/issues/63", "docs#197", "EventGraph native evidence content contract", githubCanonicalStateCompleted, "merged by PR #67", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"eventgraph#63", "https://github.com/transpara-ai/eventgraph/pull/67", "merge:c6f261a27a193a470a9e287d15580a05d1b0fafc", "evidence.testrun.recorded", "evidence.gateresult.recorded", "evidence.auditreport.recorded"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 62, "Authority evidence schema and store governance", "https://github.com/transpara-ai/eventgraph/issues/62", "docs#197", "EventGraph authority/evidence schema and migration governance", githubCanonicalStateCompleted, "merged by PR #68; authority schema/store-governance substrate only", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "eventgraph#63", "https://github.com/transpara-ai/eventgraph/pull/68", "merge:fd6f80253791e7500b2d43cae421d0a9701ae221"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 59, "Persistent EventGraph projection-store event for Civilization Assembly truth", "https://github.com/transpara-ai/eventgraph/issues/59", "docs#197", "EventGraph durable projection truth and Civilization Assembly provenance", githubCanonicalStateDeferred, "requires durable projection authority and write-path boundary", "protected-action", "projection store event is not authorized for production writes", []string{"cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 61, "Production EventGraph write path for runtime and issue evidence", "https://github.com/transpara-ai/eventgraph/issues/61", "docs#197", "EventGraph persistent write path and evidence truth", githubCanonicalStateNeedsHumanScope, "requires governed authority packet before PR work", "protected-action", "production write path still human-scope blocked", []string{"cc:intake", "cc:needs-human-scope", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#200"}, nil),
		githubCanonicalLane("transpara-ai/hive", 220, "AuthorityDecision evaluation from issue scope evidence", "https://github.com/transpara-ai/hive/issues/220", "docs#197", "Hive authority recommendation semantics", githubCanonicalStateCompleted, "merged by PR #231; recommendation-only policy does not create AuthorityDecision records", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/hive/pull/231", "merge:523181b83ad8540fba747a64a12975996db170a4", "authority_recommendation_policy"}, nil),
		githubCanonicalLane("transpara-ai/hive", 221, "Human-required protected-action classification surface", "https://github.com/transpara-ai/hive/issues/221", "docs#197", "Hive authority and human-required classification", githubCanonicalStateCompleted, "merged by PR #230; human-required classification policy persisted/projected", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/hive/pull/230", "merge:0c0fdb5f9c116cef99ed87ed9f31bfc5cbd9e10e", "human_required_classification_policy"}, nil),
		githubCanonicalLane("transpara-ai/hive", 222, "Scanner recommender tackler role separation policy", "https://github.com/transpara-ai/hive/issues/222", "docs#197", "Hive model policy and role authority semantics", githubCanonicalStateCompleted, "merged by PR #228; role-separation policy persisted/projected", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/hive/pull/228", "merge:567af4a15b51f77468597680914f067e5c357705", "role_separation_policy"}, nil),
		githubCanonicalLane("transpara-ai/hive", 223, "Autonomy-increase guard for issue-driven recommendations", "https://github.com/transpara-ai/hive/issues/223", "docs#197", "Hive autonomy boundary enforcement", githubCanonicalStateCompleted, "merged by PR #229; autonomy guard policy persisted/projected", "protected-action", "", []string{"cc:intake", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/hive/pull/229", "merge:b114f9862545ca6152474d9a06155c0c5fc58c34", "autonomy_guard_policy"}, nil),
		githubCanonicalLane("transpara-ai/operation", 34, "Clean suspend and bus-factor runbook for issue-source workflow", "https://github.com/transpara-ai/operation/issues/34", "docs#197", "operation runbook and continuity procedure", githubCanonicalStateCompleted, "merged by PR #37; issue-source workflow continuity runbook complete", "normal", "", []string{"cc:intake", "cc:pr-ready", "cc:civilization-presence"}, []string{"docs#197", "https://github.com/transpara-ai/operation/pull/37", "merge:326f90a49d986e66d171e0eb0b5be23b8e64324c", "operation#34", "operation runbook"}, nil),
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
		GeneratedAt:     now.UTC().Format(time.RFC3339),
		Status:          "partial",
		ProjectionState: "typed projection-shaped Site contract; static until EventGraph store governance is authorized",
		Parent:          OpsGitHubCanonicalIssue{Repo: "transpara-ai/docs", Number: 197, Title: "Development Arc issue-source migration parent tracker", URL: "https://github.com/transpara-ai/docs/issues/197"},
		RepoSummaries:   githubCanonicalRepoSummaries(lanes),
		Lanes:           lanes,
		EvidenceRecords: githubCanonicalEvidenceRecords(),
		CutoverChecks: []OpsGitHubCanonicalCutoverCheck{
			{Label: "Issue coverage", State: "partial", Evidence: "docs#197 and child issue lanes exist; scanner:2026-06-26T03:12:27Z found no multi-issue bundles", Blocker: "remaining EventGraph/docs protected lanes open"},
			{Label: "Work traceability", State: "completed", Evidence: "work#61, work#62, work#63 merged", Blocker: ""},
			{Label: "Issue form schema", State: "completed", Evidence: ".github#3 merged by PR #4", Blocker: ""},
			{Label: "Duplicate stale-source scanner", State: "completed", Evidence: "platform#5 merged by PR #8; latest arc_issue_scan returned Findings: 0", Blocker: ""},
			{Label: "Native evidence content", State: "completed", Evidence: "eventgraph#63 merged by PR #67; TestRun, GateResult, and AuditReport content types registered", Blocker: ""},
			{Label: "EventGraph authority projection", State: "deferred", Evidence: "eventgraph#62 merged; eventgraph#59 and eventgraph#61 remain open", Blocker: "projection-store truth and production write path still not authorized"},
			{Label: "Hive issue intake", State: "completed", Evidence: "hive#220, hive#221, hive#222, hive#223 merged with policy/projection-only CFAR evidence", Blocker: ""},
			{Label: "Operation continuity", State: "completed", Evidence: "operation#34 closed by PR #37; issue-source workflow runbook merged", Blocker: ""},
			{Label: "Site typed monitor", State: "completed", Evidence: "site#129 merged by PR #130; site#131 merged by PR #132; site#133 refresh in progress", Blocker: ""},
			{Label: "Markdown retirement", State: githubCanonicalStateLegacyEvidenceOnly, Evidence: "docs#197 closeout pending", Blocker: "legacy arc remains background evidence only"},
		},
		Boundaries: []string{
			"Read-only typed projection-shaped Site data; no live GitHub fetch or mutation.",
			"No Hive wake, runtime start, queue launch, EventGraph write, deploy, merge, approval, Test 001 GREEN claim, autonomy increase, value allocation, or residual-risk closure.",
			"Markdown is displayed only as archived/background evidence, never as the live work queue.",
			"source_issue_refs, validation_refs, cfar_refs, authority_boundary_refs, residual_risk_refs, and trace_score_basis_points are display contracts only until EventGraph write-path governance lands.",
		},
		LegacyEvidence: legacy,
	}
}

func githubCanonicalEvidenceRecords() []OpsGitHubCanonicalEvidenceRecord {
	return []OpsGitHubCanonicalEvidenceRecord{
		{
			Name:                    "GitHub canonical validation TestRun",
			EventType:               "evidence.testrun.recorded",
			Outcome:                 "tests.pass",
			Schema:                  "TestRun",
			SourceIssueRefs:         []string{"docs#197", "site#129", "site#131", "site#133", "eventgraph#63"},
			PRRefs:                  []string{"eventgraph PR #67", "site PR #130", "site PR #132"},
			ValidationRefs:          []string{"go test ./pkg/event -run TestNativeEvidence -count=1", "make verify-go", "make verify"},
			CFARRefs:                []string{"eventgraph PR #67 CFAR PASS", "site PR #132 CFAR PASS"},
			AuthorityBoundaryRefs:   []string{"docs#197", "eventgraph#61", "docs#200"},
			ResidualRiskRefs:        []string{"docs#201", "docs#202", "docs#203"},
			ProvenanceRefs:          []string{"https://github.com/transpara-ai/eventgraph/pull/67", "https://github.com/transpara-ai/site/pull/132", "merge:c6f261a27a193a470a9e287d15580a05d1b0fafc", "merge:c3dc3a63eb16eafed490b7e6be28affe3469f7ea", "https://github.com/transpara-ai/eventgraph/pull/67#issuecomment-4803740786"},
			TraceScoreBasisPoints:   10000,
			ProjectionReadinessNote: "10000 bp covers the eventgraph#63 content-contract validation only; projection store write path still deferred",
		},
		{
			Name:                  "GitHub canonical gate result",
			EventType:             "evidence.gateresult.recorded",
			Outcome:               "gate.partial",
			Schema:                "GateResult",
			SourceIssueRefs:       []string{"docs#197", "work#61", "work#62", "work#63", "site#127", "site#129", "site#133", "platform#5", ".github#3", "eventgraph#63", "eventgraph#62", "hive#220", "hive#221", "hive#222", "hive#223", "operation#34"},
			PRRefs:                []string{"https://github.com/transpara-ai/work/pull/71", "https://github.com/transpara-ai/work/pull/72", "https://github.com/transpara-ai/work/pull/73", "https://github.com/transpara-ai/site/pull/128", "https://github.com/transpara-ai/site/pull/130", "https://github.com/transpara-ai/site/pull/132", "https://github.com/transpara-ai/platform/pull/8", "https://github.com/transpara-ai/.github/pull/4", "https://github.com/transpara-ai/eventgraph/pull/67", "https://github.com/transpara-ai/eventgraph/pull/68", "https://github.com/transpara-ai/hive/pull/228", "https://github.com/transpara-ai/hive/pull/229", "https://github.com/transpara-ai/hive/pull/230", "https://github.com/transpara-ai/hive/pull/231", "https://github.com/transpara-ai/operation/pull/37"},
			ValidationRefs:        []string{"change-control scan: no multi-issue bundles", "arc_issue_scan: Findings=0"},
			CFARRefs:              []string{"work PR #71/#72/#73 CFAR PASS", "site PR #128/#130/#132 CFAR PASS", "platform PR #8 CFAR PASS", ".github PR #4 CFAR PASS", "eventgraph PR #67/#68 CFAR PASS", "hive PR #228/#229/#230/#231 CFAR PASS", "operation PR #37 CFAR PASS"},
			AuthorityBoundaryRefs: []string{"docs#197", "docs#199", "docs#200"},
			ResidualRiskRefs:      []string{"docs#172", "operation#26", "operation#35"},
			ProvenanceRefs: []string{
				"work#71 merge:f118276665c0bbbea282be7803070948b8d8e297",
				"work#72 merge:b4fecec3003056cd20ffefa5992e7c5833a8b8eb",
				"work#73 merge:731aa1c367b7627ad69c531a07a8a0e302c3900f",
				"site#128 merge:07cd69f730faf93ed7e9e03ed74c2836db9dc62c",
				"site#130 merge:cf3dcddbb06d47199dd7c94662e329422d27d10c",
				"site#132 merge:c3dc3a63eb16eafed490b7e6be28affe3469f7ea",
				"platform#8 merge:566c518893ba152f93339082b4b94b4a6140aed2",
				".github#4 merge:30cd2f25f6e0008c8d4b9fb412e66ce1e6c7bc8e",
				"eventgraph#67 merge:c6f261a27a193a470a9e287d15580a05d1b0fafc",
				"eventgraph#68 merge:fd6f80253791e7500b2d43cae421d0a9701ae221",
				"hive#228 merge:567af4a15b51f77468597680914f067e5c357705",
				"hive#229 merge:b114f9862545ca6152474d9a06155c0c5fc58c34",
				"hive#230 merge:0c0fdb5f9c116cef99ed87ed9f31bfc5cbd9e10e",
				"hive#231 merge:523181b83ad8540fba747a64a12975996db170a4",
				"operation#37 merge:326f90a49d986e66d171e0eb0b5be23b8e64324c",
				"https://github.com/transpara-ai/docs/issues/197#issuecomment-4803515276",
				"https://github.com/transpara-ai/docs/issues/197#issuecomment-4803782217",
			},
			TraceScoreBasisPoints:   8300,
			ProjectionReadinessNote: "8300 bp is a display confidence score for the cited source-of-intent substrate; EventGraph projection-store/write and docs authority lanes remain open",
		},
		{
			Name:                    "GitHub canonical closeout AuditReport",
			EventType:               "evidence.auditreport.recorded",
			Outcome:                 "closeout.blocked",
			Schema:                  "AuditReport",
			SourceIssueRefs:         []string{"docs#197", "site#129", "site#131", "site#133"},
			PRRefs:                  []string{"pending"},
			ValidationRefs:          []string{"site#130 validation complete", "site#131 validation complete", "pending site#133 validation", "pending final docs closeout scan"},
			CFARRefs:                []string{"site PR #130/#132 CFAR PASS", "pending site#133 CFAR", "pending final docs closeout CFAR"},
			AuthorityBoundaryRefs:   []string{"eventgraph#59", "eventgraph#61", "docs#200"},
			ResidualRiskRefs:        []string{"docs#201", "docs#202", "docs#203"},
			ProvenanceRefs:          []string{"https://github.com/transpara-ai/site/issues/129", "https://github.com/transpara-ai/site/issues/131", "https://github.com/transpara-ai/site/issues/133", "https://github.com/transpara-ai/docs/issues/197", "pending final docs closeout PR"},
			TraceScoreBasisPoints:   7000,
			ProjectionReadinessNote: "7000 bp is a blocked-closeout display score; canonical cutover stays blocked until EventGraph projection-store/write authority lands",
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
