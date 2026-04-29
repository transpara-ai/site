package graph

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/transpara-ai/site/profile"
)

var refineryStateOrder = []string{"inbox", "refining", "review", "ready", "done"}
var refineryExecutionStatuses = []string{"unassigned", "refining", "reviewing", "assigned", "building", "blocked", "complete"}

type RefineryItem struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Body            string    `json:"body"`
	SourceSystem    string    `json:"source_system"`
	SourceID        string    `json:"source_id"`
	ProjectedAt     time.Time `json:"projected_at"`
	LastEventAt     time.Time `json:"last_event_at"`
	State           string    `json:"state"`
	RawState        string    `json:"raw_state"`
	ExecutionStatus string    `json:"execution_status"`
	Owner           string    `json:"owner"`
	NextAction      string    `json:"next_action"`
	BlockedReason   string    `json:"blocked_reason,omitempty"`
	EvidenceCount   int       `json:"evidence_count"`
	UpdatedLabel    string    `json:"updated_label"`
}

type RefineryColumn struct {
	State       string         `json:"state"`
	Label       string         `json:"label"`
	Description string         `json:"description"`
	Items       []RefineryItem `json:"items"`
}

type RefineryProjection struct {
	SourceSystem      string           `json:"source_system"`
	SourceID          string           `json:"source_id"`
	SpaceSlug         string           `json:"space_slug"`
	ProjectedAt       time.Time        `json:"projected_at"`
	StateOrder        []string         `json:"state_order"`
	ExecutionStatuses []string         `json:"execution_statuses"`
	HumanStatus       string           `json:"human_status"`
	OpenCount         int              `json:"open_count"`
	Counts            map[string]int   `json:"counts"`
	ExecCounts        map[string]int   `json:"execution_counts"`
	Columns           []RefineryColumn `json:"columns"`
}

func (h *Handlers) handleRefinery(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spaces, _ := h.store.ListSpaces(r.Context(), h.userID(r))
	tasks, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID: space.ID,
		Kind:    KindTask,
		Limit:   1000,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	projection := buildRefineryProjection(*space, tasks, time.Now().UTC())
	RefineryView(*space, spaces, projection, h.viewUser(r), profile.FromContext(r.Context())).Render(r.Context(), w)
}

func (h *Handlers) handleRefineryProjection(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tasks, err := h.store.ListNodes(r.Context(), ListNodesParams{
		SpaceID: space.ID,
		Kind:    KindTask,
		Limit:   1000,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, buildRefineryProjection(*space, tasks, time.Now().UTC()))
}

func (h *Handlers) handleRefineryIntake(w http.ResponseWriter, r *http.Request) {
	space, _, err := h.spaceForRead(r)
	if errors.Is(err, ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	node, err := h.store.CreateNode(r.Context(), CreateNodeParams{
		SpaceID:    space.ID,
		Kind:       KindTask,
		Title:      title,
		Body:       strings.TrimSpace(r.FormValue("body")),
		Priority:   PriorityMedium,
		Author:     h.userName(r),
		AuthorID:   h.userID(r),
		AuthorKind: h.userKind(r),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.store.RecordOp(r.Context(), space.ID, node.ID, h.userName(r), h.userID(r), "intend", nil)
	p := profile.FromContext(r.Context())
	if p == nil {
		p = profile.Default()
	}
	http.Redirect(w, r, "/app/"+space.Slug+"/refinery?profile="+p.GetSlug(), http.StatusSeeOther)
}

func buildRefineryProjection(space Space, tasks []Node, projectedAt time.Time) RefineryProjection {
	columns := []RefineryColumn{
		{State: "inbox", Label: "Inbox", Description: "Captured but not yet shaped."},
		{State: "refining", Label: "Refining", Description: "Requirements, design, gates, and dependencies are being clarified."},
		{State: "review", Label: "Review", Description: "Ready for human or agent review before execution."},
		{State: "ready", Label: "Ready", Description: "Accepted for execution; active build status is shown on each card."},
		{State: "done", Label: "Done", Description: "Completed or closed with evidence."},
	}
	index := map[string]int{}
	counts := map[string]int{}
	execCounts := map[string]int{}
	for i := range columns {
		index[columns[i].State] = i
		counts[columns[i].State] = 0
	}
	for _, status := range refineryExecutionStatuses {
		execCounts[status] = 0
	}
	openCount := 0
	for _, task := range tasks {
		state := refineryState(task)
		item := refineryItem(task, projectedAt)
		if i, ok := index[state]; ok {
			columns[i].Items = append(columns[i].Items, item)
			counts[state]++
			execCounts[item.ExecutionStatus]++
			if state != "done" {
				openCount++
			}
		}
	}
	return RefineryProjection{
		SourceSystem:      "site",
		SourceID:          space.ID,
		SpaceSlug:         space.Slug,
		ProjectedAt:       projectedAt,
		StateOrder:        append([]string(nil), refineryStateOrder...),
		ExecutionStatuses: append([]string(nil), refineryExecutionStatuses...),
		HumanStatus:       refineryHumanStatus(counts, execCounts, openCount),
		OpenCount:         openCount,
		Counts:            counts,
		ExecCounts:        execCounts,
		Columns:           columns,
	}
}

func refineryItem(task Node, projectedAt time.Time) RefineryItem {
	owner := task.Assignee
	if owner == "" {
		owner = "Unassigned"
	}
	blockedReason := ""
	if task.BlockerCount > 0 {
		blockedReason = "Has active blockers"
	}
	return RefineryItem{
		ID:              task.ID,
		Title:           task.Title,
		Body:            task.Body,
		SourceSystem:    "eventgraph",
		SourceID:        task.ID,
		ProjectedAt:     projectedAt,
		LastEventAt:     task.UpdatedAt,
		State:           refineryState(task),
		RawState:        task.State,
		ExecutionStatus: refineryExecutionStatus(task),
		Owner:           owner,
		NextAction:      refineryNextAction(task),
		BlockedReason:   blockedReason,
		EvidenceCount:   task.ChildDone,
		UpdatedLabel:    task.UpdatedAt.Format("2006-01-02 15:04"),
	}
}

func refineryState(task Node) string {
	switch task.State {
	case StateDone, StateClosed:
		return "done"
	case StateReview:
		return "review"
	case StateActive, StateBlocked:
		return "ready"
	case StateOpen:
		if task.AssigneeID != "" || task.Assignee != "" {
			return "ready"
		}
		if hasAnyTag(task, "design", "requirement", "refining", "investigating") || strings.Contains(strings.ToLower(task.Title), "design") {
			return "refining"
		}
		return "inbox"
	default:
		if strings.Contains(strings.ToLower(task.State), "review") {
			return "review"
		}
		if strings.Contains(strings.ToLower(task.State), "done") || strings.Contains(strings.ToLower(task.State), "closed") {
			return "done"
		}
		if strings.Contains(strings.ToLower(task.State), "investigating") || strings.Contains(strings.ToLower(task.State), "requirement") {
			return "refining"
		}
		return "inbox"
	}
}

func refineryExecutionStatus(task Node) string {
	if task.BlockerCount > 0 || task.State == StateBlocked {
		return "blocked"
	}
	switch task.State {
	case StateActive:
		return "building"
	case StateReview:
		return "reviewing"
	case StateDone, StateClosed:
		return "complete"
	case StateOpen:
		if task.AssigneeID != "" || task.Assignee != "" {
			return "assigned"
		}
		if hasAnyTag(task, "design", "requirement", "refining", "investigating") || strings.Contains(strings.ToLower(task.Title), "design") {
			return "refining"
		}
		return "unassigned"
	default:
		if strings.Contains(strings.ToLower(task.State), "review") {
			return "reviewing"
		}
		if strings.Contains(strings.ToLower(task.State), "done") || strings.Contains(strings.ToLower(task.State), "closed") {
			return "complete"
		}
		if strings.Contains(strings.ToLower(task.State), "investigating") || strings.Contains(strings.ToLower(task.State), "requirement") {
			return "refining"
		}
		if task.State == "" {
			return "unassigned"
		}
		return "unassigned"
	}
}

func refineryNextAction(task Node) string {
	if task.BlockerCount > 0 || task.State == StateBlocked {
		return "Clear blocker or update dependency evidence"
	}
	switch refineryState(task) {
	case "inbox":
		return "Classify intake and define owner"
	case "refining":
		return "Add gates, acceptance criteria, and test plan"
	case "review":
		return "Review design and approve or request changes"
	case "ready":
		if task.AssigneeID == "" && task.Assignee == "" {
			return "Assign implementer"
		}
		return "Build and attach evidence"
	case "done":
		return "No action"
	default:
		return "Clarify status"
	}
}

func hasAnyTag(task Node, tags ...string) bool {
	for _, tag := range task.Tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		for _, want := range tags {
			if tag == want {
				return true
			}
		}
	}
	return false
}

func refineryHumanStatus(counts, execCounts map[string]int, openCount int) string {
	if openCount == 0 {
		return "Refinery has no open items. Completed cards are in Done."
	}
	return fmt.Sprintf(
		"Refinery has %d open items: %d inbox, %d refining, %d in review, and %d ready. Execution status: %d building, %d blocked, %d assigned.",
		openCount,
		counts["inbox"],
		counts["refining"],
		counts["review"],
		counts["ready"],
		execCounts["building"],
		execCounts["blocked"],
		execCounts["assigned"],
	)
}
