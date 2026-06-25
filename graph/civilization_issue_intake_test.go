package graph

import (
	"strings"
	"testing"
)

func TestOpsCivilizationIssueIntakeEmptyProjectionIsNotProjected(t *testing.T) {
	got := opsCivilizationIssueIntake(&OpsCivilizationAssemblyProjection{})

	if got.Status != "not projected" {
		t.Fatalf("status = %q, want not projected", got.Status)
	}
	if got.Summary != "No scanner-visible issue intake records are projected." {
		t.Fatalf("summary = %q", got.Summary)
	}
	if len(got.Issues) != 0 || len(got.Groups) != 0 {
		t.Fatalf("empty projection produced records: %+v", got)
	}
}

func TestOpsCivilizationIssueIntakeDerivesAggregateCandidateGroup(t *testing.T) {
	projection := &OpsCivilizationAssemblyProjection{
		IssueIntakeProjection: OpsCivilizationIssueIntakeProjection{
			Issues: []OpsCivilizationIssueIntakeProjected{
				{
					Repo:             "transpara-ai/site",
					Number:           114,
					Title:            "Read-only issue intake aggregation projection UI",
					PrimaryRepo:      "transpara-ai/site",
					TouchedSubstrate: "site operator ui projection",
					RiskClass:        "normal",
					PRReadyWhen:      "same readiness",
					Labels:           []string{"cc:intake", "cc:aggregate-candidate"},
				},
				{
					Repo:             "transpara-ai/site",
					Number:           120,
					Title:            "Promote issue-scan Kanban into a durable read-only Civilization operator surface",
					PrimaryRepo:      "transpara-ai/site",
					TouchedSubstrate: "site operator ui projection",
					RiskClass:        "normal",
					PRReadyWhen:      "same readiness",
					Labels:           []string{"cc:intake", "cc:aggregate-candidate"},
				},
			},
		},
	}

	got := opsCivilizationIssueIntake(projection)
	if got.Status != opsCivilizationFieldAvailable {
		t.Fatalf("status = %q, want available", got.Status)
	}
	if len(got.Groups) != 1 {
		t.Fatalf("groups = %+v, want one aggregate group", got.Groups)
	}
	group := got.Groups[0]
	if len(group.IssueRefs) != 2 {
		t.Fatalf("issue refs = %+v, want two refs", group.IssueRefs)
	}
	if len(group.Blockers) != 0 {
		t.Fatalf("blockers = %+v, want none", group.Blockers)
	}
	if !strings.Contains(group.Recommendation, "aggregate candidate") {
		t.Fatalf("recommendation = %q, want aggregate candidate", group.Recommendation)
	}
}

func TestOpsCivilizationIssueIntakeProtectedActionBlockerUsesExactLabelAndDedupes(t *testing.T) {
	projection := &OpsCivilizationAssemblyProjection{
		IssueIntakeProjection: OpsCivilizationIssueIntakeProjection{
			Issues: []OpsCivilizationIssueIntakeProjected{
				{
					Repo:             "transpara-ai/site",
					Number:           116,
					PrimaryRepo:      "transpara-ai/site",
					TouchedSubstrate: "site civilization display data source",
					RiskClass:        "protected-action",
					PRReadyWhen:      "same readiness",
					Labels:           []string{"cc:intake", "cc:protected-action"},
				},
				{
					Repo:             "transpara-ai/site",
					Number:           117,
					PrimaryRepo:      "transpara-ai/site",
					TouchedSubstrate: "site civilization display data source",
					RiskClass:        "protected-action",
					PRReadyWhen:      "same readiness",
					Labels:           []string{"cc:intake", "cc:protected-action"},
				},
			},
		},
	}

	got := opsCivilizationIssueIntake(projection)
	if len(got.Groups) != 1 {
		t.Fatalf("groups = %+v, want one protected group", got.Groups)
	}
	group := got.Groups[0]
	if len(group.Blockers) != 1 {
		t.Fatalf("blockers = %+v, want one deduped protected-action blocker", group.Blockers)
	}
	if !strings.Contains(group.Recommendation, "do not group") {
		t.Fatalf("recommendation = %q, want do not group", group.Recommendation)
	}

	if issueIntakeHasLabel([]string{"not-cc:protected-action", "cc:protected-action-waived"}, "cc:protected-action") {
		t.Fatal("issueIntakeHasLabel matched a substring instead of an exact label")
	}
}
