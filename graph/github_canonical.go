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
		githubCanonicalLane("transpara-ai/docs", 197, "Development Arc issue-source migration parent tracker", "https://github.com/transpara-ai/docs/issues/197", "parent", "cross-repo governance coordination and scanner evidence", githubCanonicalStateDeferred, "docs PR deferred until child lanes and scanner evidence are complete", "protected-action", "docs closeout PR cannot mark markdown superseded until implementation evidence lands", []string{"cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"scanner:2026-06-25T18:37:31Z"}, []string{"dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md"}),
		githubCanonicalLane("transpara-ai/work", 61, "Requirements and task-draft derivation from issue records", "https://github.com/transpara-ai/work/issues/61", "docs#197", "Work proposal evidence", githubCanonicalStateCompleted, "merged by PR #71", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#61", "work PR #71"}, nil),
		githubCanonicalLane("transpara-ai/work", 62, "Proof-of-work packet linked to issue source records", "https://github.com/transpara-ai/work/issues/62", "docs#197", "Work proof-of-work packet contract", githubCanonicalStateCompleted, "merged by PR #72", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#62", "work PR #72"}, nil),
		githubCanonicalLane("transpara-ai/work", 63, "AuditReport closeout linked to GitHub issue source records", "https://github.com/transpara-ai/work/issues/63", "docs#197", "Work AuditReport closeout evidence", githubCanonicalStateCompleted, "merged by PR #73", "normal", "", []string{"cc:intake", "cc:civilization-presence"}, []string{"work#63", "work PR #73"}, nil),
		githubCanonicalLane("transpara-ai/site", 127, "GitHub-canonical issue migration progress surface", "https://github.com/transpara-ai/site/issues/127", "docs#197", "Site operator UI and read-only migration progress projection", githubCanonicalStateReady, "static typed fixture contract identified", "normal", "", []string{"cc:intake", "cc:pr-ready", "cc:civilization-presence"}, []string{"docs#197"}, nil),
		githubCanonicalLane("transpara-ai/platform", 5, "Arc issue duplicate and stale-source detection", "https://github.com/transpara-ai/platform/issues/5", "docs#197", "platform scanner/read-only validation rule", githubCanonicalStateDeferred, "deferred until duplicate/stale fixtures are specified", "normal", "duplicate/stale-source detector not implemented", []string{"cc:intake", "cc:pr-deferred", "cc:civilization-presence"}, []string{"docs#197"}, []string{"dark-factory/v4.0/implementation/epics/00-integration-arc-v4.0.md"}),
		githubCanonicalLane("transpara-ai/.github", 3, "Change-control issue form arc-anchor field upgrade", "https://github.com/transpara-ai/.github/issues/3", "docs#197", "organization issue template and issue-first intake form", githubCanonicalStateProtectedAction, "deferred until issue-form field changes are authorized", "protected-action", "organization template changes need protected-action review", []string{"cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197"}, nil),
		githubCanonicalLane("transpara-ai/eventgraph", 63, "Native TestRun GateResult and AuditReport persistence contract", "https://github.com/transpara-ai/eventgraph/issues/63", "docs#197", "EventGraph native evidence persistence contract", githubCanonicalStateProtectedAction, "deferred until Work/EventGraph boundaries are specified", "protected-action", "typed EventGraph projection contract not merged", []string{"cc:intake", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"work#61", "work#62", "work#63"}, nil),
		githubCanonicalLane("transpara-ai/hive", 220, "AuthorityDecision evaluation from issue scope evidence", "https://github.com/transpara-ai/hive/issues/220", "docs#197", "Hive authority recommendation semantics", githubCanonicalStateNeedsHumanScope, "requires human-scoped authority before PR work", "protected-action", "Hive cannot treat issues as authority until fail-closed policy is explicit", []string{"cc:intake", "cc:needs-human-scope", "cc:pr-deferred", "cc:protected-action", "cc:civilization-presence"}, []string{"docs#197"}, nil),
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
		ProjectionState: "static typed Site fixture",
		Parent:          OpsGitHubCanonicalIssue{Repo: "transpara-ai/docs", Number: 197, Title: "Development Arc issue-source migration parent tracker", URL: "https://github.com/transpara-ai/docs/issues/197"},
		RepoSummaries:   githubCanonicalRepoSummaries(lanes),
		Lanes:           lanes,
		CutoverChecks: []OpsGitHubCanonicalCutoverCheck{
			{Label: "Issue coverage", State: "partial", Evidence: "docs#197 and child issue lanes exist", Blocker: "remaining protected/deferred lanes open"},
			{Label: "Work traceability", State: "completed", Evidence: "work#61, work#62, work#63 merged", Blocker: ""},
			{Label: "EventGraph typed projection", State: "deferred", Evidence: "eventgraph#63", Blocker: "typed issue/evidence projection not merged"},
			{Label: "Hive issue intake", State: "needs-human-scope", Evidence: "hive#220", Blocker: "authority semantics and fail-closed policy still need human-scoped design"},
			{Label: "Markdown retirement", State: githubCanonicalStateLegacyEvidenceOnly, Evidence: "docs#197 closeout pending", Blocker: "legacy arc remains background evidence only"},
		},
		Boundaries: []string{
			"Read-only Site fixture; no live GitHub fetch or mutation.",
			"No Hive wake, runtime start, queue launch, EventGraph write, deploy, merge, approval, Test 001 GREEN claim, autonomy increase, value allocation, or residual-risk closure.",
			"Markdown is displayed only as archived/background evidence, never as the live work queue.",
		},
		LegacyEvidence: legacy,
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
			summary.Protected++
		case githubCanonicalStateLegacyEvidenceOnly:
			summary.LegacyOnly++
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
	case githubCanonicalStateCompleted:
		return "border-emerald-400/40 text-emerald-300 bg-emerald-400/10"
	case githubCanonicalStateReady:
		return "border-brand/40 text-brand bg-brand/10"
	case githubCanonicalStateNeedsHumanScope:
		return "border-amber-300/40 text-amber-300 bg-amber-300/10"
	case githubCanonicalStateProtectedAction:
		return "border-red-300/40 text-red-300 bg-red-300/10"
	case githubCanonicalStateLegacyEvidenceOnly:
		return "border-edge text-warm-faint bg-void/30"
	default:
		return "border-edge text-warm-muted bg-void/30"
	}
}
