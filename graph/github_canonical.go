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
		githubCanonicalLane("transpara-ai/docs", 197, "Development Arc issue-source migration parent tracker", "https://github.com/transpara-ai/docs/issues/197", "parent", "cross-repo governance coordination and scanner evidence", githubCanonicalStateDeferred, "docs PR deferred until child lanes and scanner evidence are complete", "protected-action", "docs closeout PR cannot mark markdown superseded until typed projection and Hive fail-closed evidence land", []string{"cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"scanner:2026-06-25T20:15:00Z", "site#129"}, []string{"dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md"}),
		githubCanonicalLane("transpara-ai/work", 61, "Requirements and task-draft derivation from issue records", "https://github.com/transpara-ai/work/issues/61", "docs#197", "Work proposal evidence", githubCanonicalStateCompleted, "merged by PR #71", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#61", "work PR #71"}, nil),
		githubCanonicalLane("transpara-ai/work", 62, "Proof-of-work packet linked to issue source records", "https://github.com/transpara-ai/work/issues/62", "docs#197", "Work proof-of-work packet contract", githubCanonicalStateCompleted, "merged by PR #72", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#62", "work PR #72"}, nil),
		githubCanonicalLane("transpara-ai/work", 63, "AuditReport closeout linked to GitHub issue source records", "https://github.com/transpara-ai/work/issues/63", "docs#197", "Work AuditReport closeout evidence", githubCanonicalStateCompleted, "merged by PR #73", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#63", "work PR #73"}, nil),
		githubCanonicalLane("transpara-ai/site", 127, "GitHub-canonical issue migration progress surface", "https://github.com/transpara-ai/site/issues/127", "docs#197", "Site operator UI and read-only migration progress projection", githubCanonicalStateCompleted, "merged by PR #128", "normal", "", []string{"cc:intake", "cc:pr-ready", "cc:civilization-presence"}, []string{"site#127", "site PR #128", "docs#197"}, nil),
		githubCanonicalLane("transpara-ai/site", 129, "Typed projection-backed GitHub-canonical monitor", "https://github.com/transpara-ai/site/issues/129", "docs#197", "Site ops monitor and typed projection-shaped migration evidence view", githubCanonicalStateReady, "read-only Site monitor PR-ready", "protected-action", "implementation must not add runtime wakeups, writes, or live mutation APIs", []string{"cc:intake", "cc:pr-ready", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "eventgraph#63"}, nil),
		githubCanonicalLane("transpara-ai/platform", 5, "Arc issue duplicate and stale-source detection", "https://github.com/transpara-ai/platform/issues/5", "docs#197", "platform scanner/read-only validation rule", githubCanonicalStateCompleted, "merged by PR #8; latest duplicate-anchor scan returned Findings: 0", "normal", "", []string{"cc:intake", "cc:pr-ready", "cc:civilization-presence"}, []string{"platform#5", "platform PR #8", "arc_issue_scan:Findings=0"}, []string{"dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md"}),
		githubCanonicalLane("transpara-ai/.github", 3, "Change-control issue form arc-anchor field upgrade", "https://github.com/transpara-ai/.github/issues/3", "docs#197", "organization issue template and issue-first intake form", githubCanonicalStateCompleted, "merged by PR #4", "protected-action", "", []string{"cc:intake", "cc:pr-ready", "cc:protected-action", "cc:civilization-presence"}, []string{".github#3", ".github PR #4"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 63, "Native TestRun GateResult and AuditReport persistence contract", "https://github.com/transpara-ai/eventgraph/issues/63", "docs#197", "EventGraph native evidence content contract", githubCanonicalStateCompleted, "merged by PR #67", "protected-action", "", []string{"cc:intake", "cc:pr-ready", "cc:protected-action", "cc:civilization-presence"}, []string{"eventgraph#63", "eventgraph PR #67", "evidence.testrun.recorded", "evidence.gateresult.recorded", "evidence.auditreport.recorded"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 62, "Authority evidence schema and store governance", "https://github.com/transpara-ai/eventgraph/issues/62", "docs#197", "EventGraph authority/evidence schema and migration governance", githubCanonicalStateDeferred, "schema migration and fail-closed governance still required", "protected-action", "production authority/projection-store governance not complete", []string{"cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197", "eventgraph#63"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 59, "Persistent EventGraph projection-store event for Civilization Assembly truth", "https://github.com/transpara-ai/eventgraph/issues/59", "docs#197", "EventGraph durable projection truth and Civilization Assembly provenance", githubCanonicalStateDeferred, "requires durable projection authority and write-path boundary", "protected-action", "projection store event is not authorized for production writes", []string{"cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 61, "Production EventGraph write path for runtime and issue evidence", "https://github.com/transpara-ai/eventgraph/issues/61", "docs#197", "EventGraph persistent write path and evidence truth", githubCanonicalStateNeedsHumanScope, "requires governed authority packet before PR work", "protected-action", "production write path still human-scope blocked", []string{"cc:intake", "cc:needs-human-scope", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#200"}, nil),
		githubCanonicalLane("transpara-ai/hive", 220, "AuthorityDecision evaluation from issue scope evidence", "https://github.com/transpara-ai/hive/issues/220", "docs#197", "Hive authority recommendation semantics", githubCanonicalStateNeedsHumanScope, "requires human-scoped authority before PR work", "protected-action", "Hive cannot treat issues as authority until fail-closed policy is explicit", []string{"cc:intake", "cc:needs-human-scope", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197"}, nil),
		githubCanonicalLane("transpara-ai/hive", 221, "Human-required protected-action classification surface", "https://github.com/transpara-ai/hive/issues/221", "docs#197", "Hive authority and human-required classification", githubCanonicalStateNeedsHumanScope, "classification vocabulary and human handoff evidence required", "protected-action", "protected-action work must park without token burn", []string{"cc:intake", "cc:needs-human-scope", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197"}, nil),
		githubCanonicalLane("transpara-ai/hive", 222, "Scanner recommender tackler role separation policy", "https://github.com/transpara-ai/hive/issues/222", "docs#197", "Hive model policy and role authority semantics", githubCanonicalStateDeferred, "role-boundary mapping still required", "protected-action", "recommender output must not become automatic implementation authority", []string{"cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197"}, nil),
		githubCanonicalLane("transpara-ai/hive", 223, "Autonomy-increase guard for issue-driven recommendations", "https://github.com/transpara-ai/hive/issues/223", "docs#197", "Hive autonomy boundary enforcement", githubCanonicalStateDeferred, "guard behavior evidence still required", "protected-action", "issue-driven recommendation must not silently increase autonomy", []string{"cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197"}, nil),
		githubCanonicalLane("transpara-ai/operation", 34, "Clean suspend and bus-factor runbook for issue-source workflow", "https://github.com/transpara-ai/operation/issues/34", "docs#197", "operation runbook and continuity procedure", githubCanonicalStateDeferred, "runbook scope handoff evidence pending", "normal", "operator continuity procedure not updated for GitHub-canonical cutover", []string{"cc:intake", "cc:pr-deferred", "cc:civilization-presence"}, []string{"docs#197"}, nil),
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
			{Label: "Issue coverage", State: "partial", Evidence: "docs#197 and child issue lanes exist; scanner:2026-06-25T20:15:00Z found no multi-issue bundles", Blocker: "remaining protected/deferred/human-scope lanes open"},
			{Label: "Work traceability", State: "completed", Evidence: "work#61, work#62, work#63 merged", Blocker: ""},
			{Label: "Issue form schema", State: "completed", Evidence: ".github#3 merged by PR #4", Blocker: ""},
			{Label: "Duplicate stale-source scanner", State: "completed", Evidence: "platform#5 merged by PR #8; latest arc_issue_scan returned Findings: 0", Blocker: ""},
			{Label: "Native evidence content", State: "completed", Evidence: "eventgraph#63 merged by PR #67; TestRun, GateResult, and AuditReport content types registered", Blocker: ""},
			{Label: "EventGraph authority projection", State: "deferred", Evidence: "eventgraph#62, eventgraph#59, eventgraph#61", Blocker: "authority/projection-store governance and production write path still not authorized"},
			{Label: "Hive issue intake", State: "needs-human-scope", Evidence: "hive#220, hive#221, hive#222, hive#223", Blocker: "authority semantics, protected-action parking, role separation, and autonomy guards still need governed scope"},
			{Label: "Site typed monitor", State: "pr-ready", Evidence: "site#129", Blocker: "bounded read-only Site monitor PR pending"},
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
			SourceIssueRefs:         []string{"docs#197", "site#129", "eventgraph#63"},
			PRRefs:                  []string{"eventgraph PR #67"},
			ValidationRefs:          []string{"go test ./pkg/event -run TestNativeEvidence -count=1", "make verify-go", "make verify"},
			CFARRefs:                []string{"eventgraph PR #67 CFAR PASS"},
			AuthorityBoundaryRefs:   []string{"docs#197", "eventgraph#62", "eventgraph#61"},
			ResidualRiskRefs:        []string{"docs#201", "docs#202", "docs#203"},
			TraceScoreBasisPoints:   10000,
			ProjectionReadinessNote: "content contract merged; projection store write path still deferred",
		},
		{
			Name:                    "GitHub canonical gate result",
			EventType:               "evidence.gateresult.recorded",
			Outcome:                 "gate.partial",
			Schema:                  "GateResult",
			SourceIssueRefs:         []string{"docs#197", "work#61", "work#62", "work#63", "site#127", "platform#5", ".github#3", "eventgraph#63"},
			PRRefs:                  []string{"work PR #71", "work PR #72", "work PR #73", "site PR #128", "platform PR #8", ".github PR #4", "eventgraph PR #67"},
			ValidationRefs:          []string{"change-control scan: no multi-issue bundles", "arc_issue_scan: Findings=0"},
			CFARRefs:                []string{"work PR #71/#72/#73 CFAR PASS", "site PR #128 CFAR PASS", "platform PR #8 CFAR PASS", ".github PR #4 CFAR PASS", "eventgraph PR #67 CFAR PASS"},
			AuthorityBoundaryRefs:   []string{"docs#197", "docs#199", "docs#200"},
			ResidualRiskRefs:        []string{"docs#172", "operation#26", "operation#35"},
			TraceScoreBasisPoints:   7300,
			ProjectionReadinessNote: "GitHub source-of-intent substrate is present; Hive and EventGraph protected lanes remain open",
		},
		{
			Name:                    "GitHub canonical closeout AuditReport",
			EventType:               "evidence.auditreport.recorded",
			Outcome:                 "closeout.blocked",
			Schema:                  "AuditReport",
			SourceIssueRefs:         []string{"docs#197", "site#129"},
			PRRefs:                  []string{"pending"},
			ValidationRefs:          []string{"pending Site PR validation", "pending final docs closeout scan"},
			CFARRefs:                []string{"pending Site PR CFAR", "pending final docs closeout CFAR"},
			AuthorityBoundaryRefs:   []string{"eventgraph#62", "eventgraph#61", "hive#220", "hive#221", "hive#222", "hive#223"},
			ResidualRiskRefs:        []string{"docs#201", "docs#202", "docs#203"},
			TraceScoreBasisPoints:   6200,
			ProjectionReadinessNote: "AuditReport shape is visible; canonical cutover stays blocked until protected lanes close",
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
