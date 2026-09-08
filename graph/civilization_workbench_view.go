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
	case "approved", "rejected", "changes_requested":
		return 6
	case "ready":
		return 3
	case "completed":
		return 4
	default:
		return 5
	}
}

func (data CivilizationWorkbench) Groups() []civilizationWorkGroup {
	if data.View == "history" {
		return []civilizationWorkGroup{{Label: "Reviewed and completed", Items: data.HistoryItems()}}
	}
	groups := []civilizationWorkGroup{{Label: "Needs a human"}, {Label: "In progress"}, {Label: "Prepared"}, {Label: "Ready for review"}, {Label: "Completed"}, {Label: "Other work"}, {Label: "Reviewed results"}}
	for _, item := range data.VisibleItems() {
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
		items := data.VisibleItems()
		for i := range items {
			if items[i].WorkID == data.SelectedWorkID {
				return &items[i]
			}
		}
		// Old bookmarks and a live selection can refer to work just reviewed.
		// Only explicit History navigation keeps its full detail in focus.
		inHistory := false
		for _, work := range data.HistoryItems() {
			inHistory = inHistory || work.WorkID == data.SelectedWorkID
		}
		if data.View == "history" || !inHistory {
			return nil
		}
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

func civilizationWorkInspectURL(workID, state string) string {
	if civilizationWorkInHistory(CivilizationWork{State: state}) {
		return civilizationWorkURL(workID) + "&view=history"
	}
	return civilizationWorkURL(workID)
}

func (data CivilizationWorkbench) PollURL() string {
	selected := data.SelectedWorkID
	if item := data.Selected(); item != nil {
		selected = item.WorkID
	}
	return strings.Replace(data.PageURL(data.View, selected), "/console/workbench?", "/console/workbench/work-fragment?", 1)
}

func civilizationRepositoryLabel(repository string) string {
	return strings.TrimPrefix(repository, "transpara-ai/")
}

func civilizationStateLabel(state string) string {
	labels := map[string]string{"awaiting_confirmation": "Brief ready", "routing": "Preparing brief", "queued": "Queued", "implementing": "Implementing", "validating": "Checking implementation", "reviewing": "In review", "publishing": "Publishing", "merge_queued": "Merge queued", "blocked": "Blocked", "human_required": "Needs an answer", "prepared": "Awaiting your review", "approved": "Approved", "rejected": "Rejected", "changes_requested": "Changes requested", "ready": "Ready for review", "completed": "Completed"}
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
	case "publishing", "prepared", "ready", "merge_queued", "completed", "approved", "rejected", "changes_requested":
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
