package graph

import (
	"fmt"
	"sort"
	"strings"
)

type OpsCivilizationIssueIntakeProjection struct {
	Issues     []OpsCivilizationIssueIntakeProjected      `json:"issues,omitempty"`
	Groups     []OpsCivilizationIssueIntakeGroupProjected `json:"groups,omitempty"`
	SourceRefs []string                                   `json:"source_refs,omitempty"`
}

type OpsCivilizationIssueIntakeProjected struct {
	Repo              string   `json:"repo"`
	Number            int      `json:"number"`
	URL               string   `json:"url,omitempty"`
	Title             string   `json:"title,omitempty"`
	State             string   `json:"state,omitempty"`
	StateReason       string   `json:"state_reason,omitempty"`
	Labels            []string `json:"labels,omitempty"`
	PrimaryRepo       string   `json:"primary_repo,omitempty"`
	TouchedSubstrate  string   `json:"touched_substrate,omitempty"`
	RiskClass         string   `json:"risk_class,omitempty"`
	Readiness         string   `json:"readiness,omitempty"`
	PRReadyWhen       string   `json:"pr_ready_when,omitempty"`
	AuthorityBoundary string   `json:"authority_boundary,omitempty"`
	UpdatedAt         string   `json:"updated_at,omitempty"`
	SourceRefs        []string `json:"source_refs,omitempty"`
}

type OpsCivilizationIssueIntakeGroupProjected struct {
	GroupID          string                    `json:"group_id,omitempty"`
	Summary          string                    `json:"summary,omitempty"`
	PrimaryRepo      string                    `json:"primary_repo,omitempty"`
	TouchedSubstrate string                    `json:"touched_substrate,omitempty"`
	RiskClass        string                    `json:"risk_class,omitempty"`
	Readiness        string                    `json:"readiness,omitempty"`
	Recommendation   string                    `json:"recommendation,omitempty"`
	IssueRefs        []OpsCivilizationIssueRef `json:"issue_refs,omitempty"`
	Blockers         []string                  `json:"blockers,omitempty"`
	SourceRefs       []string                  `json:"source_refs,omitempty"`
}

type OpsCivilizationIssueIntake struct {
	Status     string
	Summary    string
	SourceRefs []string
	Issues     []OpsCivilizationIssueIntakeProjected
	Groups     []OpsCivilizationIssueIntakeGroupProjected
	Guardrails []OpsCivilizationIssueGuardrail
}

func opsCivilizationIssueIntake(projection *OpsCivilizationAssemblyProjection) OpsCivilizationIssueIntake {
	intake := OpsCivilizationIssueIntake{
		Status:     opsCivilizationProjectionStatusUnavailable,
		Summary:    "Issue intake projection unavailable to Site.",
		Guardrails: opsCivilizationIssueGuardrails(),
	}
	if projection == nil {
		return intake
	}

	input := projection.IssueIntakeProjection
	if len(input.Issues) == 0 && len(input.Groups) == 0 {
		intake.Status = "not projected"
		intake.Summary = "No scanner-visible issue intake records are projected."
		intake.SourceRefs = sortedNonEmpty(input.SourceRefs)
		return intake
	}

	intake.Status = opsCivilizationFieldAvailable
	intake.SourceRefs = sortedNonEmpty(input.SourceRefs)
	intake.Issues = normalizedIssueIntakeIssues(input.Issues)
	intake.Groups = normalizedIssueIntakeGroups(input.Groups)
	if len(intake.Groups) == 0 {
		intake.Groups = deriveIssueIntakeGroups(intake.Issues)
	}
	intake.Summary = fmt.Sprintf("%d issue(s), %d advisory group(s) projected.", len(intake.Issues), len(intake.Groups))
	return intake
}

func normalizedIssueIntakeIssues(items []OpsCivilizationIssueIntakeProjected) []OpsCivilizationIssueIntakeProjected {
	out := make([]OpsCivilizationIssueIntakeProjected, 0, len(items))
	for _, item := range items {
		item.Repo = strings.TrimSpace(item.Repo)
		item.URL = strings.TrimSpace(item.URL)
		item.Title = strings.TrimSpace(item.Title)
		item.State = strings.TrimSpace(item.State)
		item.StateReason = strings.TrimSpace(item.StateReason)
		item.PrimaryRepo = strings.TrimSpace(item.PrimaryRepo)
		item.TouchedSubstrate = strings.TrimSpace(item.TouchedSubstrate)
		item.RiskClass = strings.TrimSpace(item.RiskClass)
		item.Readiness = strings.TrimSpace(item.Readiness)
		item.PRReadyWhen = strings.TrimSpace(item.PRReadyWhen)
		item.AuthorityBoundary = strings.TrimSpace(item.AuthorityBoundary)
		item.UpdatedAt = strings.TrimSpace(item.UpdatedAt)
		item.Labels = sortedNonEmpty(item.Labels)
		item.SourceRefs = sortedNonEmpty(item.SourceRefs)
		if item.Repo == "" && item.Number == 0 && item.Title == "" {
			continue
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PrimaryRepo != out[j].PrimaryRepo {
			return out[i].PrimaryRepo < out[j].PrimaryRepo
		}
		if out[i].Repo != out[j].Repo {
			return out[i].Repo < out[j].Repo
		}
		if out[i].Number != out[j].Number {
			return out[i].Number < out[j].Number
		}
		return out[i].Title < out[j].Title
	})
	return out
}

func normalizedIssueIntakeGroups(items []OpsCivilizationIssueIntakeGroupProjected) []OpsCivilizationIssueIntakeGroupProjected {
	out := make([]OpsCivilizationIssueIntakeGroupProjected, 0, len(items))
	for _, item := range items {
		item.GroupID = strings.TrimSpace(item.GroupID)
		item.Summary = strings.TrimSpace(item.Summary)
		item.PrimaryRepo = strings.TrimSpace(item.PrimaryRepo)
		item.TouchedSubstrate = strings.TrimSpace(item.TouchedSubstrate)
		item.RiskClass = strings.TrimSpace(item.RiskClass)
		item.Readiness = strings.TrimSpace(item.Readiness)
		item.Recommendation = strings.TrimSpace(item.Recommendation)
		item.IssueRefs = sortedIssueRefs(item.IssueRefs)
		item.Blockers = sortedNonEmpty(item.Blockers)
		item.SourceRefs = sortedNonEmpty(item.SourceRefs)
		if item.GroupID == "" {
			item.GroupID = issueIntakeGroupID(item.PrimaryRepo, item.TouchedSubstrate, item.RiskClass, item.Readiness)
		}
		if item.GroupID == "" && len(item.IssueRefs) == 0 {
			continue
		}
		out = append(out, item)
	}
	sortIssueIntakeGroups(out)
	return out
}

func deriveIssueIntakeGroups(issues []OpsCivilizationIssueIntakeProjected) []OpsCivilizationIssueIntakeGroupProjected {
	groupsByKey := map[string]OpsCivilizationIssueIntakeGroupProjected{}
	for _, issue := range issues {
		primaryRepo := opsCivilizationValue(issue.PrimaryRepo, issue.Repo)
		readiness := opsCivilizationValue(issue.PRReadyWhen, issue.Readiness)
		key := issueIntakeGroupKey(primaryRepo, issue.TouchedSubstrate, issue.RiskClass, readiness)
		group := groupsByKey[key]
		group.PrimaryRepo = primaryRepo
		group.TouchedSubstrate = issue.TouchedSubstrate
		group.RiskClass = issue.RiskClass
		group.Readiness = readiness
		group.GroupID = issueIntakeGroupID(primaryRepo, issue.TouchedSubstrate, issue.RiskClass, readiness)
		group.IssueRefs = append(group.IssueRefs, OpsCivilizationIssueRef{
			Repo:        issue.Repo,
			Number:      issue.Number,
			URL:         issue.URL,
			Title:       issue.Title,
			State:       issue.State,
			StateReason: issue.StateReason,
			Labels:      issue.Labels,
		})
		if issueIntakeHasLabel(issue.Labels, "cc:protected-action") {
			group.Blockers = appendIssueIntakeUniqueString(group.Blockers, "protected-action issue requires separate authority scope before grouped implementation")
		}
		groupsByKey[key] = group
	}

	groups := make([]OpsCivilizationIssueIntakeGroupProjected, 0, len(groupsByKey))
	for _, group := range groupsByKey {
		group.IssueRefs = sortedIssueRefs(group.IssueRefs)
		group.Blockers = sortedNonEmpty(group.Blockers)
		if len(group.IssueRefs) > 1 && len(group.Blockers) == 0 {
			group.Recommendation = "aggregate candidate; verify same repo, substrate, risk, acceptance path, and readiness before PR grouping"
		} else if len(group.Blockers) > 0 {
			group.Recommendation = "do not group until protected-action scope is explicitly authorized"
		} else {
			group.Recommendation = "singleton; no aggregation candidate projected"
		}
		group.Summary = fmt.Sprintf("%d issue(s) share repo/substrate/risk/readiness key.", len(group.IssueRefs))
		groups = append(groups, group)
	}
	sortIssueIntakeGroups(groups)
	return groups
}

func sortIssueIntakeGroups(groups []OpsCivilizationIssueIntakeGroupProjected) {
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].PrimaryRepo != groups[j].PrimaryRepo {
			return groups[i].PrimaryRepo < groups[j].PrimaryRepo
		}
		if groups[i].RiskClass != groups[j].RiskClass {
			return groups[i].RiskClass < groups[j].RiskClass
		}
		if groups[i].TouchedSubstrate != groups[j].TouchedSubstrate {
			return groups[i].TouchedSubstrate < groups[j].TouchedSubstrate
		}
		return groups[i].GroupID < groups[j].GroupID
	})
}

func issueIntakeGroupKey(parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(part)))
	}
	return strings.Join(normalized, "\x00")
}

func issueIntakeGroupID(parts ...string) string {
	joined := strings.ToLower(strings.Join(parts, "-"))
	var b strings.Builder
	lastDash := false
	for _, r := range joined {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func issueIntakeHasLabel(labels []string, want string) bool {
	for _, label := range labels {
		if strings.TrimSpace(label) == want {
			return true
		}
	}
	return false
}

func appendIssueIntakeUniqueString(items []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return items
	}
	for _, item := range items {
		if strings.TrimSpace(item) == value {
			return items
		}
	}
	return append(items, value)
}

func issueIntakeProjectedRefLabel(issue OpsCivilizationIssueIntakeProjected) string {
	return opsCivilizationIssueRefLabel(OpsCivilizationIssueRef{
		Repo:        issue.Repo,
		Number:      issue.Number,
		URL:         issue.URL,
		Title:       issue.Title,
		State:       issue.State,
		StateReason: issue.StateReason,
		Labels:      issue.Labels,
	})
}
