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

func TestOpsCivilizationIssueIntakePreservesUnavailableProjectionState(t *testing.T) {
	got := opsCivilizationIssueIntake(&OpsCivilizationAssemblyProjection{
		IssueIntakeProjection: OpsCivilizationIssueIntakeProjection{
			Status:            "unavailable",
			Summary:           "no GitHub issue source-intent refs are present in the EventGraph source snapshot",
			SourceRefs:        []string{"github:transpara-ai/docs#172", " github:transpara-ai/site#115 "},
			ScannerBoundaries: []string{"scanner_read_only", "no_eventgraph_writes", "scanner_read_only"},
		},
	})

	if got.Status != opsCivilizationFieldUnavailable {
		t.Fatalf("status = %q, want unavailable", got.Status)
	}
	if got.Summary != "no GitHub issue source-intent refs are present in the EventGraph source snapshot" {
		t.Fatalf("summary = %q", got.Summary)
	}
	if strings.Join(got.SourceRefs, ",") != "github:transpara-ai/docs#172,github:transpara-ai/site#115" {
		t.Fatalf("source refs = %+v", got.SourceRefs)
	}
	if strings.Join(got.Boundaries, ",") != "no_eventgraph_writes,scanner_read_only" {
		t.Fatalf("boundaries = %+v", got.Boundaries)
	}
	if len(got.Issues) != 0 || len(got.Groups) != 0 {
		t.Fatalf("unavailable projection produced records: %+v", got)
	}
}

func TestOpsCivilizationIssueIntakeConsumesTypedEventGraphIssueFields(t *testing.T) {
	projection := &OpsCivilizationAssemblyProjection{
		IssueIntakeProjection: OpsCivilizationIssueIntakeProjection{
			Status:            opsCivilizationFieldAvailable,
			Summary:           "typed issue-intake projection available",
			ScannerBoundaries: []string{"scanner_read_only"},
			Issues: []OpsCivilizationIssueIntakeProjected{
				{
					Repo:              " transpara-ai/site ",
					Number:            116,
					URL:               " https://github.com/transpara-ai/site/issues/116 ",
					PrimaryRepo:       " transpara-ai/site ",
					TouchedSubstrate:  "site civilization display data source",
					TouchedSubstrates: []string{" eventgraph read model ", "site ops surface", "eventgraph read model"},
					RiskClass:         "protected-action",
					RiskClasses:       []string{"normal", " protected-action ", "normal"},
					UnrecognizedRisk:  []string{"experimental", " experimental "},
					Readiness:         "dependency merged",
					ReadinessStates:   []string{"cc:pr-ready", "dependency merged", "cc:pr-ready"},
					AuthorityBoundary: "read-only display only; no issue mutation",
					SourceRefs:        []string{" github:transpara-ai/eventgraph#60 ", "github:transpara-ai/eventgraph#66"},
				},
			},
		},
	}

	got := opsCivilizationIssueIntake(projection)
	if got.Status != opsCivilizationFieldAvailable {
		t.Fatalf("status = %q, want available", got.Status)
	}
	if got.Summary != "typed issue-intake projection available" {
		t.Fatalf("summary = %q", got.Summary)
	}
	if len(got.Issues) != 1 {
		t.Fatalf("issues = %+v, want one issue", got.Issues)
	}
	issue := got.Issues[0]
	if issue.Repo != "transpara-ai/site" || issue.Number != 116 || issue.PrimaryRepo != "transpara-ai/site" {
		t.Fatalf("issue identity not normalized: %+v", issue)
	}
	if strings.Join(issue.TouchedSubstrates, ",") != "eventgraph read model,site ops surface" {
		t.Fatalf("touched substrates = %+v", issue.TouchedSubstrates)
	}
	if strings.Join(issue.RiskClasses, ",") != "normal,protected-action" {
		t.Fatalf("risk classes = %+v", issue.RiskClasses)
	}
	if strings.Join(issue.UnrecognizedRisk, ",") != "experimental" {
		t.Fatalf("unrecognized risk terms = %+v", issue.UnrecognizedRisk)
	}
	if strings.Join(issue.ReadinessStates, ",") != "cc:pr-ready,dependency merged" {
		t.Fatalf("readiness states = %+v", issue.ReadinessStates)
	}
	if strings.Join(issue.SourceRefs, ",") != "github:transpara-ai/eventgraph#60,github:transpara-ai/eventgraph#66" {
		t.Fatalf("source refs = %+v", issue.SourceRefs)
	}
	if len(got.Groups) != 1 {
		t.Fatalf("groups = %+v, want derived singleton group", got.Groups)
	}
	group := got.Groups[0]
	if group.GroupID != "repo-transpara-ai-site-substrate-site-civilization-display-data-source-risk-protected-action-readiness-dependency-merged" {
		t.Fatalf("group id = %q", group.GroupID)
	}
	if group.Readiness != "dependency merged" {
		t.Fatalf("group readiness = %q", group.Readiness)
	}
	if len(group.Blockers) != 1 {
		t.Fatalf("typed protected-action risk should synthesize one blocker: %+v", group.Blockers)
	}
	if !strings.Contains(group.Recommendation, "do not group") {
		t.Fatalf("recommendation = %q, want do not group", group.Recommendation)
	}
}

func TestOpsCivilizationIssueIntakeDropsEmptyProjectedGroupsBeforeSynthesizingID(t *testing.T) {
	got := opsCivilizationIssueIntake(&OpsCivilizationAssemblyProjection{
		IssueIntakeProjection: OpsCivilizationIssueIntakeProjection{
			Groups: []OpsCivilizationIssueIntakeGroupProjected{
				{},
				{SourceRefs: []string{"github:transpara-ai/site#116"}},
			},
		},
	})

	if len(got.Groups) != 1 {
		t.Fatalf("groups = %+v, want one non-empty projected group", got.Groups)
	}
	group := got.Groups[0]
	if group.GroupID != "repo-substrate-risk-readiness" {
		t.Fatalf("group id = %q", group.GroupID)
	}
	if strings.Join(group.SourceRefs, ",") != "github:transpara-ai/site#116" {
		t.Fatalf("source refs = %+v", group.SourceRefs)
	}
}

func TestOpsCivilizationIssueIntakePreservesUnavailableStatusWhenRecordsArePresent(t *testing.T) {
	got := opsCivilizationIssueIntake(&OpsCivilizationAssemblyProjection{
		IssueIntakeProjection: OpsCivilizationIssueIntakeProjection{
			Status: opsCivilizationFieldUnavailable,
			Issues: []OpsCivilizationIssueIntakeProjected{
				{Repo: "transpara-ai/site", Number: 116},
			},
		},
	})

	if got.Status != opsCivilizationFieldUnavailable {
		t.Fatalf("status = %q, want unavailable", got.Status)
	}
	if len(got.Issues) != 1 {
		t.Fatalf("issues = %+v, want present records to remain visible", got.Issues)
	}
}

func TestOpsCivilizationIssueIntakeUsesPostNormalizationRecordsForStatus(t *testing.T) {
	got := opsCivilizationIssueIntake(&OpsCivilizationAssemblyProjection{
		IssueIntakeProjection: OpsCivilizationIssueIntakeProjection{
			Status: "available",
			Issues: []OpsCivilizationIssueIntakeProjected{
				{},
			},
		},
	})

	if got.Status != "not projected" {
		t.Fatalf("status = %q, want not projected", got.Status)
	}
	if got.Summary != "No scanner-visible issue intake records are projected." {
		t.Fatalf("summary = %q", got.Summary)
	}
	if len(got.Issues) != 0 || len(got.Groups) != 0 {
		t.Fatalf("empty normalized projection produced records: %+v", got)
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

func TestOpsCivilizationIssueIntakeProtectedActionBlockerUsesTypedRiskClasses(t *testing.T) {
	got := opsCivilizationIssueIntake(&OpsCivilizationAssemblyProjection{
		IssueIntakeProjection: OpsCivilizationIssueIntakeProjection{
			Issues: []OpsCivilizationIssueIntakeProjected{
				{
					Repo:              "transpara-ai/site",
					Number:            116,
					PrimaryRepo:       "transpara-ai/site",
					TouchedSubstrate:  "site civilization display data source",
					RiskClass:         "normal",
					RiskClasses:       []string{"Protected-Action"},
					Readiness:         "dependency merged",
					AuthorityBoundary: "read-only display only; no issue mutation",
				},
			},
		},
	})

	if len(got.Groups) != 1 {
		t.Fatalf("groups = %+v, want one group", got.Groups)
	}
	group := got.Groups[0]
	if len(group.Blockers) != 1 {
		t.Fatalf("blockers = %+v, want one typed risk-class blocker", group.Blockers)
	}
	if !strings.Contains(group.Recommendation, "do not group") {
		t.Fatalf("recommendation = %q, want do not group", group.Recommendation)
	}
}
