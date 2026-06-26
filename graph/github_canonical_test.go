package graph

import (
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
		{repo: "transpara-ai/site", completed: 2, protected: 1},
		{repo: "transpara-ai/.github", completed: 1, protected: 1},
		{repo: "transpara-ai/eventgraph", completed: 2, deferred: 1, humanScope: 1, protected: 4},
		{repo: "transpara-ai/hive", completed: 4, protected: 4},
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
	if !githubCanonicalContainsString(testRun.SourceIssueRefs, "site#131") || !githubCanonicalContainsString(testRun.SourceIssueRefs, "site#133") || !githubCanonicalContainsString(testRun.ValidationRefs, "make verify") || !githubCanonicalContainsString(testRun.AuthorityBoundaryRefs, "eventgraph#61") {
		t.Fatalf("TestRun refs are incomplete: %+v", testRun)
	}
	if !githubCanonicalContainsString(testRun.ProvenanceRefs, "merge:c6f261a27a193a470a9e287d15580a05d1b0fafc") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "merge:c3dc3a63eb16eafed490b7e6be28affe3469f7ea") || !githubCanonicalContainsString(testRun.ProvenanceRefs, "https://github.com/transpara-ai/eventgraph/pull/67#issuecomment-4803740786") {
		t.Fatalf("TestRun provenance refs are incomplete: %+v", testRun.ProvenanceRefs)
	}

	gateResult := records["evidence.gateresult.recorded"]
	if gateResult.Schema != "GateResult" || gateResult.Outcome != "gate.partial" || gateResult.TraceScoreBasisPoints != 8300 {
		t.Fatalf("GateResult record = %+v", gateResult)
	}
	if !githubCanonicalContainsString(gateResult.SourceIssueRefs, "hive#220") || !githubCanonicalContainsString(gateResult.SourceIssueRefs, "operation#34") || !githubCanonicalContainsString(gateResult.CFARRefs, "hive PR #228/#229/#230/#231 CFAR PASS") || !githubCanonicalContainsString(gateResult.CFARRefs, "operation PR #37 CFAR PASS") {
		t.Fatalf("GateResult refs are incomplete: %+v", gateResult)
	}
	if !githubCanonicalContainsString(gateResult.ProvenanceRefs, "hive#231 merge:523181b83ad8540fba747a64a12975996db170a4") || !githubCanonicalContainsString(gateResult.ProvenanceRefs, "operation#37 merge:326f90a49d986e66d171e0eb0b5be23b8e64324c") || !githubCanonicalContainsString(gateResult.ProvenanceRefs, "https://github.com/transpara-ai/docs/issues/197#issuecomment-4803515276") {
		t.Fatalf("GateResult provenance refs are incomplete: %+v", gateResult.ProvenanceRefs)
	}

	auditReport := records["evidence.auditreport.recorded"]
	if auditReport.Schema != "AuditReport" || auditReport.Outcome != "closeout.blocked" || auditReport.TraceScoreBasisPoints != 7000 {
		t.Fatalf("AuditReport record = %+v", auditReport)
	}
	if !githubCanonicalContainsString(auditReport.AuthorityBoundaryRefs, "eventgraph#59") || !githubCanonicalContainsString(auditReport.ResidualRiskRefs, "docs#203") {
		t.Fatalf("AuditReport refs are incomplete: %+v", auditReport)
	}
	if !githubCanonicalContainsString(auditReport.ProvenanceRefs, "https://github.com/transpara-ai/site/issues/131") || !githubCanonicalContainsString(auditReport.ProvenanceRefs, "https://github.com/transpara-ai/site/issues/133") {
		t.Fatalf("AuditReport provenance refs are incomplete: %+v", auditReport.ProvenanceRefs)
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
