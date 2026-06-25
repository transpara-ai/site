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
		{repo: "transpara-ai/site", completed: 1, ready: 1, protected: 1},
		{repo: "transpara-ai/.github", completed: 1, protected: 1},
		{repo: "transpara-ai/eventgraph", completed: 1, deferred: 2, humanScope: 1, protected: 4},
		{repo: "transpara-ai/hive", deferred: 2, humanScope: 2, protected: 4},
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
	if !githubCanonicalContainsString(testRun.SourceIssueRefs, "site#129") || !githubCanonicalContainsString(testRun.ValidationRefs, "make verify") || !githubCanonicalContainsString(testRun.AuthorityBoundaryRefs, "eventgraph#61") {
		t.Fatalf("TestRun refs are incomplete: %+v", testRun)
	}

	gateResult := records["evidence.gateresult.recorded"]
	if gateResult.Schema != "GateResult" || gateResult.Outcome != "gate.partial" || gateResult.TraceScoreBasisPoints != 7300 {
		t.Fatalf("GateResult record = %+v", gateResult)
	}
	if !githubCanonicalContainsString(gateResult.SourceIssueRefs, ".github#3") || !githubCanonicalContainsString(gateResult.CFARRefs, "eventgraph PR #67 CFAR PASS") {
		t.Fatalf("GateResult refs are incomplete: %+v", gateResult)
	}

	auditReport := records["evidence.auditreport.recorded"]
	if auditReport.Schema != "AuditReport" || auditReport.Outcome != "closeout.blocked" || auditReport.TraceScoreBasisPoints != 6200 {
		t.Fatalf("AuditReport record = %+v", auditReport)
	}
	if !githubCanonicalContainsString(auditReport.AuthorityBoundaryRefs, "hive#223") || !githubCanonicalContainsString(auditReport.ResidualRiskRefs, "docs#203") {
		t.Fatalf("AuditReport refs are incomplete: %+v", auditReport)
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
