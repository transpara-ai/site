package graph

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestOpsGitHubCanonicalRepoSummariesCountProtectedRisk(t *testing.T) {
	data := buildOpsGitHubCanonicalData(time.Date(2026, 6, 25, 20, 30, 0, 0, time.UTC))

	summaries := map[string]OpsGitHubCanonicalRepoSummary{}
	for _, summary := range data.RepoSummaries {
		summaries[summary.Repo] = summary
	}

	tests := []struct {
		repo       string
		completed  int
		ready      int
		deferred   int
		humanScope int
		protected  int
	}{
		{repo: "transpara-ai/docs", completed: 2, deferred: 1, humanScope: 1, protected: 4},
		{repo: "transpara-ai/work", completed: 3, humanScope: 1, protected: 1},
		{repo: "transpara-ai/site", completed: 2, protected: 1},
		{repo: "transpara-ai/platform", completed: 4, protected: 1},
		{repo: "transpara-ai/.github", completed: 1, protected: 1},
		{repo: "transpara-ai/eventgraph", completed: 3, deferred: 0, humanScope: 2, protected: 5},
		{repo: "transpara-ai/hive", completed: 5, protected: 5},
		{repo: "transpara-ai/operation", completed: 1},
	}

	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			got, ok := summaries[tt.repo]
			if !ok {
				t.Fatalf("missing repo summary for %s", tt.repo)
			}
			if got.Completed != tt.completed || got.Ready != tt.ready || got.Deferred != tt.deferred || got.HumanScope != tt.humanScope || got.Protected != tt.protected {
				t.Fatalf("summary = %+v, want completed=%d ready=%d deferred=%d humanScope=%d protected=%d", got, tt.completed, tt.ready, tt.deferred, tt.humanScope, tt.protected)
			}
		})
	}
}

func TestOpsGitHubCanonicalRepoSummariesCountLiteralProtectedStateOnce(t *testing.T) {
	summaries := githubCanonicalRepoSummaries([]OpsGitHubCanonicalLane{
		{
			Issue: OpsGitHubCanonicalIssue{Repo: "transpara-ai/example", Number: 1},
			State: githubCanonicalStateProtectedAction,
			Risk:  "normal",
		},
		{
			Issue: OpsGitHubCanonicalIssue{Repo: "transpara-ai/example", Number: 2},
			State: githubCanonicalStateProtectedAction,
			Risk:  githubCanonicalStateProtectedAction,
		},
	})
	if len(summaries) != 1 {
		t.Fatalf("summaries len = %d, want 1", len(summaries))
	}
	got := summaries[0]
	if got.Protected != 2 {
		t.Fatalf("Protected = %d, want 2: %+v", got.Protected, got)
	}
	if got.Completed != 0 || got.Ready != 0 || got.Deferred != 0 || got.HumanScope != 0 || got.LegacyOnly != 0 {
		t.Fatalf("literal protected state should not increment other lifecycle buckets: %+v", got)
	}
}

func TestOpsGitHubCanonicalAutonomyFrontierReflectsParkedScannerState(t *testing.T) {
	data := buildOpsGitHubCanonicalData(time.Date(2026, 6, 26, 10, 40, 0, 0, time.UTC))

	if data.GeneratedAt != "2026-06-26T10:40:00Z" {
		t.Fatalf("GeneratedAt/rendered_at = %q", data.GeneratedAt)
	}
	if data.ScannerSnapshotAt != "2026-06-26T12:08:57Z" {
		t.Fatalf("ScannerSnapshotAt = %q", data.ScannerSnapshotAt)
	}
	if data.ProjectionSource != "static transcription of scanner evidence; request render is not a live GitHub scan" {
		t.Fatalf("ProjectionSource = %q", data.ProjectionSource)
	}
	if !githubCanonicalContainsString(data.Boundaries, "Rendered-at time is request freshness only; scanner_snapshot_at is the latest scan represented by autonomy frontier and issue-shape state; individual evidence rows may cite earlier confirming scans.") {
		t.Fatalf("freshness boundary missing: %+v", data.Boundaries)
	}
	if got := latestGitHubCanonicalScannerTimestamp(data); got != data.ScannerSnapshotAt {
		t.Fatalf("ScannerSnapshotAt = %q, want latest scanner timestamp %q", data.ScannerSnapshotAt, got)
	}

	frontier := data.AutonomyFrontier
	if frontier.Recommendation != "park-autonomy-no-pr-ready-work" {
		t.Fatalf("Recommendation = %q, want parked frontier", frontier.Recommendation)
	}
	if frontier.PRReadyIssueCount != 0 || frontier.AutonomousPRReadyIssueCount != 0 || frontier.CandidateBundleCount != 0 || frontier.IssueShapeWarningCount != 0 {
		t.Fatalf("frontier ready/candidate/warning counts should be zero: %+v", frontier)
	}
	if frontier.NeedsHumanScopeIssueCount != 13 || frontier.ProtectedActionIssueCount != 14 || frontier.DeferredIssueCount != 13 {
		t.Fatalf("frontier guarded counts = %+v, want human=13 protected=14 deferred=13", frontier)
	}
	for _, want := range []string{
		"platform#17",
		"docs#198",
		"https://github.com/transpara-ai/docs/pull/206",
		"platform#7",
		"https://github.com/transpara-ai/platform/pull/18",
		"https://github.com/transpara-ai/platform/pull/19",
		"merge:87b0337f380b7e6ec9beb3c5be6dc7c0c5ec8ee8",
		"merge:b4ba2f98254ff32360dfcb490eb86e4613d8999d",
		"merge:e6691b62c4fd98179441f0085f23ab1c7c9a2f52",
		"reviewed_head:c9b1274e70173c3b29c5ee4a03805852a9a65d30",
		"reviewed_head:7d4da36507fc62e979c6d3a46efd005126d33f53",
		"reviewed_head:488bf95db116c0555757c7781173fd41923599e2",
		"scanner:2026-06-26T12:08:57Z autonomy_frontier:park-autonomy-no-pr-ready-work",
		"scanner:2026-06-26T12:08:57Z total_issue_count:14 needs_human_scope_issue_count:13 protected_action_issue_count:14 deferred_issue_count:13",
		"scanner:2026-06-26T12:08:57Z blocker_refs:transpara-ai/docs#172,transpara-ai/docs#193,transpara-ai/docs#197,transpara-ai/docs#200,transpara-ai/docs#201,transpara-ai/docs#202,transpara-ai/docs#203,transpara-ai/eventgraph#59,transpara-ai/eventgraph#61,transpara-ai/hive#204,transpara-ai/operation#26,transpara-ai/operation#35,transpara-ai/work#59,transpara-ai/work#64",
		"arc_issue_scan:Findings=0",
	} {
		if !githubCanonicalContainsString(frontier.EvidenceRefs, want) {
			t.Fatalf("frontier evidence refs missing %q: %+v", want, frontier.EvidenceRefs)
		}
	}
	for _, want := range []string{
		"transpara-ai/docs#197",
		"transpara-ai/work#64",
	} {
		if !githubCanonicalContainsString(frontier.BlockerRefs, want) {
			t.Fatalf("frontier blocker refs missing %q: %+v", want, frontier.BlockerRefs)
		}
	}
	if frontier.Boundary == "" {
		t.Fatal("frontier boundary must be explicit")
	}
	for _, closed := range []string{"transpara-ai/docs#198", "transpara-ai/docs#199", "transpara-ai/platform#7", "transpara-ai/site#143", "transpara-ai/site#145", "transpara-ai/site#147", "transpara-ai/site#149", "transpara-ai/site#151", "transpara-ai/site#153"} {
		if githubCanonicalContainsString(frontier.BlockerRefs, closed) {
			t.Fatalf("frontier blocker refs include closed or self-refresh issue %q: %+v", closed, frontier.BlockerRefs)
		}
	}
}

func TestOpsGitHubCanonicalProgressShowsCloseoutsAndParkedReasons(t *testing.T) {
	data := buildOpsGitHubCanonicalData(time.Date(2026, 6, 26, 12, 45, 0, 0, time.UTC))
	progress := data.Progress

	if progress.RecentClosedIssueCount != 6 || progress.ParkedOpenIssueCount != 14 || progress.PRReadyIssueCount != 0 || progress.CandidateBundleCount != 0 {
		t.Fatalf("progress counts = %+v, want six recent closeouts, fourteen parked blockers, zero ready/candidate work", progress)
	}
	if progress.Recommendation != "park-autonomy-no-pr-ready-work" {
		t.Fatalf("progress recommendation = %q", progress.Recommendation)
	}
	if len(progress.RecentCloseouts) != progress.RecentClosedIssueCount {
		t.Fatalf("recent closeouts len = %d, want %d", len(progress.RecentCloseouts), progress.RecentClosedIssueCount)
	}

	for _, want := range []struct {
		repo         string
		number       int
		prRef        string
		mergeCommit  string
		reviewedHead string
	}{
		{repo: "transpara-ai/site", number: 153, prRef: "site PR #154", mergeCommit: "d177a8bbf019d0260862fab986474e6d8b8888b5", reviewedHead: "618e22f084396b5721aa618d81d8b1e98a9fe7ec"},
		{repo: "transpara-ai/platform", number: 7, prRef: "platform PR #19", mergeCommit: "e6691b62c4fd98179441f0085f23ab1c7c9a2f52", reviewedHead: "488bf95db116c0555757c7781173fd41923599e2"},
		{repo: "transpara-ai/docs", number: 198, prRef: "docs PR #206", mergeCommit: "87b0337f380b7e6ec9beb3c5be6dc7c0c5ec8ee8", reviewedHead: "c9b1274e70173c3b29c5ee4a03805852a9a65d30"},
		{repo: "transpara-ai/site", number: 151, prRef: "site PR #152", mergeCommit: "50428bd3a7b61c2b42634eab4040928eee99e051", reviewedHead: "99c277ade2e0af826e09faf3e87d9f880668cb5b"},
		{repo: "transpara-ai/site", number: 149, prRef: "site PR #150", mergeCommit: "08d5fc9d798fa60cefbb344666ebe2e59094b821", reviewedHead: "62ced862bb6775f0da71e965cb0de4aa5472859f"},
		{repo: "transpara-ai/site", number: 147, prRef: "site PR #148", mergeCommit: "56777d134cd2e1c9a0996162c6565ed88f01cb37", reviewedHead: "ce992679f415e0ec0bdaa7f05d29396445f7560f"},
	} {
		var found *OpsGitHubCanonicalCloseout
		for i := range progress.RecentCloseouts {
			item := &progress.RecentCloseouts[i]
			if item.Issue.Repo == want.repo && item.Issue.Number == want.number {
				found = item
				break
			}
		}
		if found == nil {
			t.Fatalf("missing recent closeout %s#%d: %+v", want.repo, want.number, progress.RecentCloseouts)
		}
		if found.PRRef != want.prRef || found.MergeCommit != want.mergeCommit || found.ReviewedHead != want.reviewedHead {
			t.Fatalf("recent closeout %s#%d = %+v", want.repo, want.number, *found)
		}
	}

	if len(progress.ParkedGroups) != 2 {
		t.Fatalf("parked groups len = %d, want 2", len(progress.ParkedGroups))
	}
	if progress.ParkedOpenIssueCount != data.AutonomyFrontier.TotalIssueCount {
		t.Fatalf("progress parked count = %d, frontier total = %d", progress.ParkedOpenIssueCount, data.AutonomyFrontier.TotalIssueCount)
	}
	groupRefs := map[string]bool{}
	groupCount := 0
	for _, group := range progress.ParkedGroups {
		groupCount += group.Count
		for _, ref := range group.Refs {
			groupRefs[ref] = true
		}
	}
	if groupCount != data.AutonomyFrontier.TotalIssueCount {
		t.Fatalf("parked group count sum = %d, frontier total = %d", groupCount, data.AutonomyFrontier.TotalIssueCount)
	}
	if len(groupRefs) != len(data.AutonomyFrontier.BlockerRefs) {
		t.Fatalf("parked group refs len = %d, frontier blockers len = %d: %+v", len(groupRefs), len(data.AutonomyFrontier.BlockerRefs), groupRefs)
	}
	for _, ref := range data.AutonomyFrontier.BlockerRefs {
		if !groupRefs[ref] {
			t.Fatalf("frontier blocker %q missing from progress parked groups: %+v", ref, progress.ParkedGroups)
		}
	}
	if progress.ParkedGroups[0].Count != 13 || !githubCanonicalContainsString(progress.ParkedGroups[0].Refs, "transpara-ai/work#64") {
		t.Fatalf("protected/human-scope parked group incomplete: %+v", progress.ParkedGroups[0])
	}
	if progress.ParkedGroups[1].Count != 1 || !githubCanonicalContainsString(progress.ParkedGroups[1].Refs, "transpara-ai/docs#197") {
		t.Fatalf("parent tracker parked group incomplete: %+v", progress.ParkedGroups[1])
	}
	for _, ref := range progress.EvidenceRefs {
		if strings.Contains(ref, "site#155") {
			t.Fatalf("progress evidence should not self-reference current refresh issue site#155: %+v", progress.EvidenceRefs)
		}
	}
}

func latestGitHubCanonicalScannerTimestamp(data *OpsGitHubCanonicalData) string {
	// Lexical ordering is chronological because scanner evidence uses fixed-width UTC Z stamps.
	scannerStamp := regexp.MustCompile(`scanner:(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)`)
	var latest string
	observe := func(text string) {
		for _, match := range scannerStamp.FindAllStringSubmatch(text, -1) {
			if len(match) == 2 && match[1] > latest {
				latest = match[1]
			}
		}
	}
	observeIssue := func(issue OpsGitHubCanonicalIssue) {
		observe(issue.Repo)
		observe(issue.Title)
		observe(issue.URL)
	}
	observe(data.GeneratedAt)
	observe(data.ScannerSnapshotAt)
	observe(data.ProjectionSource)
	observe(data.Status)
	observe(data.ProjectionState)
	observeIssue(data.Parent)
	for _, lane := range data.Lanes {
		observeIssue(lane.Issue)
		observe(lane.ParentRef)
		observe(lane.Substrate)
		observe(lane.State)
		observe(lane.Readiness)
		observe(lane.Risk)
		observe(lane.BlockedReason)
		for _, label := range lane.Labels {
			observe(label)
		}
		for _, ref := range lane.EvidenceRefs {
			observe(ref)
		}
		for _, ref := range lane.LegacyRefs {
			observe(ref)
		}
	}
	for _, summary := range data.RepoSummaries {
		observe(summary.Repo)
		observe(summary.BlockedReason)
	}
	for _, ref := range data.AutonomyFrontier.EvidenceRefs {
		observe(ref)
	}
	for _, ref := range data.AutonomyFrontier.BlockerRefs {
		observe(ref)
	}
	observe(data.AutonomyFrontier.Recommendation)
	observe(data.AutonomyFrontier.Boundary)
	for _, warning := range data.IssueWarnings {
		observeIssue(warning.Issue)
		for _, ref := range warning.EvidenceRefs {
			observe(ref)
		}
		observe(warning.Recommendation)
		observe(warning.Boundary)
	}
	for _, record := range data.EvidenceRecords {
		observe(record.Name)
		observe(record.EventType)
		observe(record.Outcome)
		observe(record.Schema)
		for _, refs := range [][]string{record.SourceIssueRefs, record.PRRefs, record.ValidationRefs, record.CFARRefs, record.AuthorityBoundaryRefs, record.ResidualRiskRefs, record.ProvenanceRefs} {
			for _, ref := range refs {
				observe(ref)
			}
		}
		observe(record.ProjectionReadinessNote)
	}
	for _, check := range data.CutoverChecks {
		observe(check.Label)
		observe(check.State)
		observe(check.Evidence)
		observe(check.Blocker)
	}
	for _, boundary := range data.Boundaries {
		observe(boundary)
	}
	for _, legacy := range data.LegacyEvidence {
		observe(legacy.Ref)
		observe(legacy.State)
		observe(legacy.Disposition)
	}
	return latest
}

func TestOpsGitHubCanonicalSiteMonitorLaneIncludesRefreshEvidence(t *testing.T) {
	data := buildOpsGitHubCanonicalData(time.Date(2026, 6, 26, 9, 50, 0, 0, time.UTC))

	var lane *OpsGitHubCanonicalLane
	for i := range data.Lanes {
		if data.Lanes[i].Issue.Repo == "transpara-ai/site" && data.Lanes[i].Issue.Number == 129 {
			lane = &data.Lanes[i]
			break
		}
	}
	if lane == nil {
		t.Fatal("missing site#129 monitor lane")
	}
	for _, want := range []string{
		"site#143",
		"site#145",
		"https://github.com/transpara-ai/site/pull/144",
		"https://github.com/transpara-ai/site/pull/146",
		"https://github.com/transpara-ai/platform/pull/19",
		"merge:885d8f14fbcf15c6d5ae1b67d88a3f40a7d9104d",
		"merge:fac357e0836adc54a65f1778c229a44bd3f0d364",
		"merge:e6691b62c4fd98179441f0085f23ab1c7c9a2f52",
		"reviewed_head:80c979a8c969e8c3f10511f4de10aadef783be9f",
		"reviewed_head:488bf95db116c0555757c7781173fd41923599e2",
		"https://github.com/transpara-ai/site/pull/146#issuecomment-4808512003",
		"https://github.com/transpara-ai/platform/pull/19#issuecomment-4809397170",
	} {
		if !githubCanonicalContainsString(lane.EvidenceRefs, want) {
			t.Fatalf("site#129 lane evidence refs missing %q: %+v", want, lane.EvidenceRefs)
		}
	}
	if lane.Readiness != "merged by PR #130; refreshed by site#131, site#133, site#135, site#139, site#143, site#145, and site#153" {
		t.Fatalf("site#129 readiness = %q", lane.Readiness)
	}

	var parent *OpsGitHubCanonicalLane
	for i := range data.Lanes {
		if data.Lanes[i].Issue.Repo == "transpara-ai/docs" && data.Lanes[i].Issue.Number == 197 {
			parent = &data.Lanes[i]
			break
		}
	}
	if parent == nil {
		t.Fatal("missing docs#197 parent lane")
	}
	if !githubCanonicalContainsString(parent.EvidenceRefs, "site#145") || !githubCanonicalContainsString(parent.EvidenceRefs, "platform#19") {
		t.Fatalf("docs#197 parent lane evidence refs missing refresh evidence: %+v", parent.EvidenceRefs)
	}
	if !githubCanonicalContainsString(parent.EvidenceRefs, "docs#198") || !githubCanonicalContainsString(parent.EvidenceRefs, "docs#206") {
		t.Fatalf("docs#197 parent lane evidence refs missing docs#198 closeout evidence: %+v", parent.EvidenceRefs)
	}
	if !githubCanonicalContainsString(parent.EvidenceRefs, "https://github.com/transpara-ai/docs/issues/197#issuecomment-4809241930") {
		t.Fatalf("docs#197 parent lane evidence refs missing docs#198 closeout comment: %+v", parent.EvidenceRefs)
	}
	if !githubCanonicalContainsString(parent.EvidenceRefs, "https://github.com/transpara-ai/docs/issues/197#issuecomment-4809411010") {
		t.Fatalf("docs#197 parent lane evidence refs missing latest closeout comment: %+v", parent.EvidenceRefs)
	}
}

func TestOpsGitHubCanonicalCompletedProtectedLanesCarryCloseoutEvidence(t *testing.T) {
	data := buildOpsGitHubCanonicalData(time.Date(2026, 6, 26, 12, 30, 0, 0, time.UTC))

	docs198 := mustFindGitHubCanonicalLane(t, data, "transpara-ai/docs", 198)
	if docs198.State != githubCanonicalStateCompleted {
		t.Fatalf("docs#198 state = %q, want completed", docs198.State)
	}
	for _, want := range []string{
		"https://github.com/transpara-ai/docs/pull/206",
		"merge:87b0337f380b7e6ec9beb3c5be6dc7c0c5ec8ee8",
		"reviewed_head:c9b1274e70173c3b29c5ee4a03805852a9a65d30",
		"https://github.com/transpara-ai/docs/pull/206#issuecomment-4809200144",
		"https://github.com/transpara-ai/docs/issues/197#issuecomment-4809241930",
	} {
		if !githubCanonicalContainsString(docs198.EvidenceRefs, want) {
			t.Fatalf("docs#198 evidence refs missing %q: %+v", want, docs198.EvidenceRefs)
		}
	}

	platform7 := mustFindGitHubCanonicalLane(t, data, "transpara-ai/platform", 7)
	if platform7.State != githubCanonicalStateCompleted {
		t.Fatalf("platform#7 state = %q, want completed", platform7.State)
	}
	for _, want := range []string{
		"https://github.com/transpara-ai/platform/pull/19",
		"merge:e6691b62c4fd98179441f0085f23ab1c7c9a2f52",
		"reviewed_head:488bf95db116c0555757c7781173fd41923599e2",
		"https://github.com/transpara-ai/platform/pull/19#issuecomment-4809397170",
		"https://github.com/transpara-ai/docs/issues/197#issuecomment-4809411010",
	} {
		if !githubCanonicalContainsString(platform7.EvidenceRefs, want) {
			t.Fatalf("platform#7 evidence refs missing %q: %+v", want, platform7.EvidenceRefs)
		}
	}
	for _, stale := range []string{"cc:pr-deferred", "cc:needs-human-scope"} {
		if githubCanonicalContainsString(platform7.Labels, stale) {
			t.Fatalf("platform#7 completed lane still carries stale label %q: %+v", stale, platform7.Labels)
		}
	}
}

func TestOpsGitHubCanonicalEvidenceRecordsExposeEventGraphContract(t *testing.T) {
	data := buildOpsGitHubCanonicalData(time.Date(2026, 6, 25, 20, 30, 0, 0, time.UTC))
	if len(data.EvidenceRecords) != 3 {
		t.Fatalf("EvidenceRecords len = %d, want 3", len(data.EvidenceRecords))
	}

	records := map[string]OpsGitHubCanonicalEvidenceRecord{}
	for _, record := range data.EvidenceRecords {
		records[record.EventType] = record
	}

	testRun := records["evidence.testrun.recorded"]
	if testRun.Schema != "TestRun" || testRun.Outcome != "tests.pass" || testRun.TraceScoreBasisPoints != 10000 {
		t.Fatalf("TestRun record = %+v", testRun)
	}
	if !githubCanonicalContainsString(testRun.SourceIssueRefs, "docs#199") || !githubCanonicalContainsString(testRun.SourceIssueRefs, "site#131") || !githubCanonicalContainsString(testRun.SourceIssueRefs, "site#133") || !githubCanonicalContainsString(testRun.SourceIssueRefs, "site#135") || !githubCanonicalContainsString(testRun.SourceIssueRefs, "site#143") || !githubCanonicalContainsString(testRun.SourceIssueRefs, "site#145") || !githubCanonicalContainsString(testRun.SourceIssueRefs, "eventgraph#69") || !githubCanonicalContainsString(testRun.SourceIssueRefs, "hive#232") || !githubCanonicalContainsString(testRun.SourceIssueRefs, "platform#17") || !githubCanonicalContainsString(testRun.ValidationRefs, "make verify") || !githubCanonicalContainsString(testRun.ValidationRefs, "site#143 validation complete") || !githubCanonicalContainsString(testRun.ValidationRefs, "site#145 validation complete by site PR #146") || !githubCanonicalContainsString(testRun.ValidationRefs, "platform scanner autonomy_frontier recommendation=park-autonomy-no-pr-ready-work") || !githubCanonicalContainsString(testRun.AuthorityBoundaryRefs, "eventgraph#61") {
		t.Fatalf("TestRun refs are incomplete: %+v", testRun)
	}
	if !githubCanonicalContainsString(testRun.PRRefs, "site PR #144") || !githubCanonicalContainsString(testRun.PRRefs, "site PR #146") || !githubCanonicalContainsString(testRun.CFARRefs, "site PR #132/#144/#146 CFAR PASS") {
		t.Fatalf("TestRun missing site PR/CFAR refs: %+v", testRun)
	}
	if !githubCanonicalContainsString(testRun.ProvenanceRefs, "merge:874980a7ab6d1b5c6ef3bacfc8c02f1401f00a13") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "reviewed_head:2c76e779e51b004db4004f81117cfcb6dd3e3638") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "merge:c6f261a27a193a470a9e287d15580a05d1b0fafc") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "merge:ec22be652d0f117c68393104ad911042fc5cc272") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "merge:89921d82d5019f2181e2b75435019c19e9ab92c9") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "merge:c3dc3a63eb16eafed490b7e6be28affe3469f7ea") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "merge:885d8f14fbcf15c6d5ae1b67d88a3f40a7d9104d") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "merge:fac357e0836adc54a65f1778c229a44bd3f0d364") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "reviewed_head:80c979a8c969e8c3f10511f4de10aadef783be9f") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "merge:b4ba2f98254ff32360dfcb490eb86e4613d8999d") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "https://github.com/transpara-ai/docs/pull/205#issuecomment-4808081439") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "https://github.com/transpara-ai/eventgraph/pull/67#issuecomment-4803740786") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "https://github.com/transpara-ai/hive/pull/233#issuecomment-4806413483") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "https://github.com/transpara-ai/site/pull/144#issuecomment-4808318161") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "https://github.com/transpara-ai/site/pull/146#issuecomment-4808512003") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "https://github.com/transpara-ai/platform/pull/18#issuecomment-4807330168") {
		t.Fatalf("TestRun provenance refs are incomplete: %+v", testRun.ProvenanceRefs)
	}
	if githubCanonicalContainsString(testRun.SourceIssueRefs, "site#147") {
		t.Fatalf("TestRun should not self-reference current refresh issue site#147: %+v", testRun.SourceIssueRefs)
	}
	if githubCanonicalContainsString(testRun.SourceIssueRefs, "site#149") {
		t.Fatalf("TestRun should not self-reference current repair issue site#149: %+v", testRun.SourceIssueRefs)
	}
	if githubCanonicalContainsString(testRun.SourceIssueRefs, "site#153") {
		t.Fatalf("TestRun should not self-reference current refresh issue site#153: %+v", testRun.SourceIssueRefs)
	}

	gateResult := records["evidence.gateresult.recorded"]
	if gateResult.Schema != "GateResult" || gateResult.Outcome != "gate.partial" || gateResult.TraceScoreBasisPoints != 8500 {
		t.Fatalf("GateResult record = %+v", gateResult)
	}
	if !githubCanonicalContainsString(gateResult.SourceIssueRefs, "docs#199") || !githubCanonicalContainsString(gateResult.SourceIssueRefs, "site#143") || !githubCanonicalContainsString(gateResult.SourceIssueRefs, "site#145") || !githubCanonicalContainsString(gateResult.SourceIssueRefs, "hive#220") || !githubCanonicalContainsString(gateResult.SourceIssueRefs, "hive#232") || !githubCanonicalContainsString(gateResult.SourceIssueRefs, "eventgraph#69") || !githubCanonicalContainsString(gateResult.SourceIssueRefs, "eventgraph#59") || !githubCanonicalContainsString(gateResult.SourceIssueRefs, "work#59") || !githubCanonicalContainsString(gateResult.SourceIssueRefs, "docs#193") || !githubCanonicalContainsString(gateResult.SourceIssueRefs, "operation#34") || !githubCanonicalContainsString(gateResult.SourceIssueRefs, "platform#17") || !githubCanonicalContainsString(gateResult.PRRefs, "https://github.com/transpara-ai/site/pull/144") || !githubCanonicalContainsString(gateResult.PRRefs, "https://github.com/transpara-ai/site/pull/146") || !githubCanonicalContainsString(gateResult.CFARRefs, "docs PR #205/#206 CFAR PASS") || !githubCanonicalContainsString(gateResult.CFARRefs, "site PR #128/#130/#132/#144/#146 CFAR PASS") || !githubCanonicalContainsString(gateResult.CFARRefs, "hive PR #228/#229/#230/#231/#233 CFAR PASS") || !githubCanonicalContainsString(gateResult.CFARRefs, "eventgraph PR #67/#68/#70 CFAR PASS") || !githubCanonicalContainsString(gateResult.CFARRefs, "operation PR #37 CFAR PASS") || !githubCanonicalContainsString(gateResult.CFARRefs, "platform PR #8/#16/#18/#19 CFAR PASS") {
		t.Fatalf("GateResult refs are incomplete: %+v", gateResult)
	}
	if !githubCanonicalContainsString(gateResult.ProvenanceRefs, "docs#205 merge:874980a7ab6d1b5c6ef3bacfc8c02f1401f00a13") || !githubCanonicalContainsString(gateResult.ProvenanceRefs, "site#144 merge:885d8f14fbcf15c6d5ae1b67d88a3f40a7d9104d") || !githubCanonicalContainsString(gateResult.ProvenanceRefs, "site#146 merge:fac357e0836adc54a65f1778c229a44bd3f0d364") || !githubCanonicalContainsString(gateResult.ProvenanceRefs, "hive#231 merge:523181b83ad8540fba747a64a12975996db170a4") || !githubCanonicalContainsString(gateResult.ProvenanceRefs, "hive#233 merge:89921d82d5019f2181e2b75435019c19e9ab92c9") || !githubCanonicalContainsString(gateResult.ProvenanceRefs, "eventgraph#70 merge:ec22be652d0f117c68393104ad911042fc5cc272") || !githubCanonicalContainsString(gateResult.ProvenanceRefs, "operation#37 merge:326f90a49d986e66d171e0eb0b5be23b8e64324c") || !githubCanonicalContainsString(gateResult.ProvenanceRefs, "platform#18 merge:b4ba2f98254ff32360dfcb490eb86e4613d8999d") || !githubCanonicalContainsString(gateResult.ProvenanceRefs, "https://github.com/transpara-ai/docs/issues/197#issuecomment-4808529248") {
		t.Fatalf("GateResult provenance refs are incomplete: %+v", gateResult.ProvenanceRefs)
	}
	if githubCanonicalContainsString(gateResult.SourceIssueRefs, "site#147") {
		t.Fatalf("GateResult should not self-reference current refresh issue site#147: %+v", gateResult.SourceIssueRefs)
	}
	if githubCanonicalContainsString(gateResult.SourceIssueRefs, "site#149") {
		t.Fatalf("GateResult should not self-reference current repair issue site#149: %+v", gateResult.SourceIssueRefs)
	}
	if githubCanonicalContainsString(gateResult.SourceIssueRefs, "site#153") {
		t.Fatalf("GateResult should not self-reference current refresh issue site#153: %+v", gateResult.SourceIssueRefs)
	}

	auditReport := records["evidence.auditreport.recorded"]
	if auditReport.Schema != "AuditReport" || auditReport.Outcome != "closeout.blocked" || auditReport.TraceScoreBasisPoints != 7000 {
		t.Fatalf("AuditReport record = %+v", auditReport)
	}
	if !githubCanonicalContainsString(auditReport.SourceIssueRefs, "docs#199") || !githubCanonicalContainsString(auditReport.SourceIssueRefs, "site#139") || !githubCanonicalContainsString(auditReport.SourceIssueRefs, "site#143") || !githubCanonicalContainsString(auditReport.SourceIssueRefs, "site#145") || !githubCanonicalContainsString(auditReport.AuthorityBoundaryRefs, "docs#199") || !githubCanonicalContainsString(auditReport.AuthorityBoundaryRefs, "eventgraph#59") || !githubCanonicalContainsString(auditReport.AuthorityBoundaryRefs, "work#59") || !githubCanonicalContainsString(auditReport.AuthorityBoundaryRefs, "docs#193") || !githubCanonicalContainsString(auditReport.ResidualRiskRefs, "docs#203") {
		t.Fatalf("AuditReport refs are incomplete: %+v", auditReport)
	}
	if !githubCanonicalContainsString(auditReport.PRRefs, "https://github.com/transpara-ai/site/pull/144") || !githubCanonicalContainsString(auditReport.PRRefs, "https://github.com/transpara-ai/site/pull/146") || !githubCanonicalContainsString(auditReport.ValidationRefs, "site#143 validation complete by site PR #144") || !githubCanonicalContainsString(auditReport.ValidationRefs, "site#145 validation complete by site PR #146") || !githubCanonicalContainsString(auditReport.CFARRefs, "site PR #130/#132/#134/#136/#138/#140/#142/#144/#146 CFAR PASS") {
		t.Fatalf("AuditReport missing completed site#143/site#144 refs: %+v", auditReport)
	}
	for _, stale := range []string{"pending site#143 validation", "pending site#143 CFAR"} {
		if githubCanonicalContainsString(auditReport.ValidationRefs, stale) || githubCanonicalContainsString(auditReport.CFARRefs, stale) {
			t.Fatalf("AuditReport still contains stale %q: %+v", stale, auditReport)
		}
	}
	if !githubCanonicalContainsString(auditReport.ProvenanceRefs, "merge:874980a7ab6d1b5c6ef3bacfc8c02f1401f00a13") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "reviewed_head:2c76e779e51b004db4004f81117cfcb6dd3e3638") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "site#144 merge:885d8f14fbcf15c6d5ae1b67d88a3f40a7d9104d") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "site#144 reviewed_head:5d62b4d83c942795f49fd423aac29da9d0b897ea") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "site#146 merge:fac357e0836adc54a65f1778c229a44bd3f0d364") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "site#146 reviewed_head:80c979a8c969e8c3f10511f4de10aadef783be9f") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "https://github.com/transpara-ai/site/pull/144#issuecomment-4808318161") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "https://github.com/transpara-ai/site/pull/146#issuecomment-4808512003") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "https://github.com/transpara-ai/site/issues/131") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "https://github.com/transpara-ai/site/issues/133") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "https://github.com/transpara-ai/site/issues/135") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "https://github.com/transpara-ai/site/issues/139") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "https://github.com/transpara-ai/site/issues/143") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "https://github.com/transpara-ai/site/issues/145") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "https://github.com/transpara-ai/docs/issues/197#issuecomment-4808529248") {
		t.Fatalf("AuditReport provenance refs are incomplete: %+v", auditReport.ProvenanceRefs)
	}
	if githubCanonicalContainsString(auditReport.SourceIssueRefs, "site#147") {
		t.Fatalf("AuditReport should not self-reference current refresh issue site#147: %+v", auditReport.SourceIssueRefs)
	}
	if githubCanonicalContainsString(auditReport.SourceIssueRefs, "site#149") {
		t.Fatalf("AuditReport should not self-reference current repair issue site#149: %+v", auditReport.SourceIssueRefs)
	}
	if githubCanonicalContainsString(auditReport.SourceIssueRefs, "site#153") {
		t.Fatalf("AuditReport should not self-reference current refresh issue site#153: %+v", auditReport.SourceIssueRefs)
	}
}

func githubCanonicalContainsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func mustFindGitHubCanonicalLane(t *testing.T, data *OpsGitHubCanonicalData, repo string, number int) OpsGitHubCanonicalLane {
	t.Helper()
	for _, lane := range data.Lanes {
		if lane.Issue.Repo == repo && lane.Issue.Number == number {
			return lane
		}
	}
	t.Fatalf("missing lane %s#%d", repo, number)
	return OpsGitHubCanonicalLane{}
}
