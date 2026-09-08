package graph

import (
	"net/url"
	"sort"
	"strings"
)

type civilizationWorkGroup struct {
	Label string
	Items []CivilizationWork
}

func civilizationWorkGroupIndex(state string) int {
	switch state {
	case "awaiting_confirmation", "blocked", "human_required":
		return 0
	case "routing", "queued", "implementing", "validating", "reviewing", "publishing", "merge_queued":
		return 1
	case "prepared":
		return 2
	case "ready":
		return 3
	case "completed":
		return 4
	default:
		return 5
	}
}

func (data CivilizationWorkbench) Groups() []civilizationWorkGroup {
	groups := []civilizationWorkGroup{{Label: "Needs you"}, {Label: "In progress"}, {Label: "Prepared"}, {Label: "Ready for review"}, {Label: "Completed"}, {Label: "Other work"}}
	for _, item := range data.Items {
		index := civilizationWorkGroupIndex(item.State)
		groups[index].Items = append(groups[index].Items, item)
	}
	for i := range groups {
		sort.SliceStable(groups[i].Items, func(a, b int) bool { return groups[i].Items[a].UpdatedAt.After(groups[i].Items[b].UpdatedAt) })
	}
	return groups
}

func (data CivilizationWorkbench) Selected() *CivilizationWork {
	if data.SelectedWorkID != "" {
		for i := range data.Items {
			if data.Items[i].WorkID == data.SelectedWorkID {
				return &data.Items[i]
			}
		}
		return nil
	}
	for _, group := range data.Groups() {
		if len(group.Items) > 0 {
			return &group.Items[0]
		}
	}
	return nil
}

func civilizationWorkURL(workID string) string {
	return "/console/workbench?work=" + url.QueryEscape(workID)
}

func (data CivilizationWorkbench) PollURL() string {
	selected := data.SelectedWorkID
	if item := data.Selected(); item != nil {
		selected = item.WorkID
	}
	return "/console/workbench/work-fragment?work=" + url.QueryEscape(selected)
}

func civilizationRepositoryLabel(repository string) string {
	return strings.TrimPrefix(repository, "transpara-ai/")
}

func civilizationStateLabel(state string) string {
	labels := map[string]string{"awaiting_confirmation": "Brief ready", "routing": "Preparing brief", "queued": "Queued", "implementing": "Implementing", "validating": "Checking implementation", "reviewing": "In review", "publishing": "Publishing", "merge_queued": "Merge queued", "blocked": "Blocked", "human_required": "Needs your answer", "prepared": "Prepared", "ready": "Ready for review", "completed": "Completed"}
	if label := labels[state]; label != "" {
		return label
	}
	if state == "" {
		return "Status unavailable"
	}
	return strings.ReplaceAll(state, "_", " ")
}

// Verification and review share a stage: the caller verifies independently
// after provider review, so the UI must not infer that verification passed early.
func civilizationWorkStage(work CivilizationWork) int {
	state := work.State
	if state == "blocked" || state == "human_required" {
		state = work.ResumeState
	}
	switch state {
	case "routing", "awaiting_confirmation":
		return 0
	case "queued", "implementing":
		return 1
	case "validating", "reviewing":
		return 2
	case "publishing", "prepared", "ready", "merge_queued", "completed":
		return 3
	default:
		return -1
	}
}

func civilizationSelectedCurrent(data CivilizationWorkbench, workID string) string {
	if selected := data.Selected(); selected != nil && selected.WorkID == workID {
		return "true"
	}
	return "false"
}

func civilizationStageStatus(work CivilizationWork, index int) string {
	stage := civilizationWorkStage(work)
	if index < stage {
		return "done"
	}
	if index == stage {
		if work.State == "blocked" || work.State == "human_required" {
			return "blocked"
		}
		if work.State == "completed" {
			return "done"
		}
		return "current"
	}
	return "pending"
}

func civilizationOwnerInitial(work CivilizationWork) string {
	owner := []rune(civilizationWorkOwner(work))
	if len(owner) == 0 {
		return "?"
	}
	return strings.ToUpper(string(owner[0]))
}

func civilizationResolutionText(data CivilizationWorkbench, interventionID string) string {
	if data.ResolutionID == interventionID {
		return data.ResolutionText
	}
	return ""
}

type civilizationArtifact struct {
	Repository      string `json:"repository"`
	Branch          string `json:"branch"`
	BaseSHA         string `json:"base_sha"`
	WorkspaceDigest string `json:"workspace_digest"`
	Patch           string `json:"patch"`
}

type civilizationDiffLine struct {
	Text string
	Kind string
}

func civilizationDiffPreview(patch string) ([]civilizationDiffLine, bool) {
	parts := strings.SplitN(patch, "\n", 401)
	truncated := len(parts) > 400
	if truncated {
		parts = parts[:400]
	}
	lines := make([]civilizationDiffLine, 0, len(parts))
	for _, line := range parts {
		runes := []rune(line)
		if len(runes) > 2000 {
			line = string(runes[:2000]) + "…"
			truncated = true
		}
		kind := "context"
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			kind = "added"
		}
		if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			kind = "removed"
		}
		lines = append(lines, civilizationDiffLine{Text: line, Kind: kind})
	}
	return lines, truncated
}
