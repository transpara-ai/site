package graph

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/transpara-ai/site/auth"
)

func (h *Handlers) workbenchForRequest(r *http.Request) CivilizationWorkbench {
	data := loadCivilizationWorkbench(r.Context())
	if r.Header.Get("HX-Target") != "civilization-work-list" {
		if projection, err := fetchHiveOperatorProjection(r); err == nil && projection != nil {
			for _, model := range projection.ModelSelection.Models {
				if !model.Deprecated && (model.Provider == "codex-cli" || model.Provider == "claude-cli") {
					data.ModelOptions = append(data.ModelOptions, model)
				}
			}
		}
	}
	data.SelectedWorkID = r.FormValue("work")
	data.View, data.Repository, data.Operator = r.FormValue("view"), r.FormValue("filter_repository"), r.FormValue("operator")
	data.Responsible = r.FormValue("responsible")
	if data.View != "team" && data.View != "history" {
		data.View = "focus"
	}
	viewer := h.viewUser(r)
	data.ViewerID = viewer.ID
	data.ViewerRole, data.ViewerName = viewer.Role, viewer.Name
	data.OperatorNames = map[string]string{}
	var ids []string
	for _, work := range data.Items {
		if id := civilizationRequesterID(work); id != "" {
			ids = append(ids, id)
		}
		if work.HumanOwnerID != "" {
			ids = append(ids, work.HumanOwnerID)
		}
	}
	if h.store != nil {
		data.OperatorNames = h.store.ResolveUserNames(r.Context(), ids)
	}
	if viewer.ID != "" {
		data.OperatorNames[viewer.ID] = viewer.Name
	}
	for _, operator := range auth.PrivateOperators(r.Context()) {
		data.OperatorNames[operator.ID] = operator.Name
		if operator.Role == "operator" || operator.Role == "reviewer" {
			data.AssignableOperators = append(data.AssignableOperators, operator.ID)
		}
	}
	// A result reviewed in another session should recede on the next poll too.
	if data.View != "history" {
		for _, work := range data.HistoryItems() {
			if work.WorkID == data.SelectedWorkID {
				data.SelectedWorkID = ""
				break
			}
		}
	}
	return data
}

func civilizationRequesterID(work CivilizationWork) string {
	if work.Source.Kind != "human" {
		return ""
	}
	identity, ok := strings.CutPrefix(work.Source.Identity, "human:")
	if !ok {
		return ""
	}
	// Site creates human:<authenticated user ID>:<intake identity>.
	index := strings.LastIndex(identity, ":")
	if index <= 0 {
		return ""
	}
	return identity[:index]
}

func (data CivilizationWorkbench) Requester(work CivilizationWork) string {
	id := civilizationRequesterID(work)
	if id == "" {
		if work.Source.Kind == "issue" {
			return "Repository issue"
		}
		return "Not recorded"
	}
	if name := data.OperatorNames[id]; name != "" {
		return name
	}
	return id
}

type civilizationOperatorOption struct{ ID, Name string }

func (data CivilizationWorkbench) Operators() []civilizationOperatorOption {
	seen := map[string]bool{}
	var result []civilizationOperatorOption
	for _, work := range data.Items {
		id := civilizationRequesterID(work)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, civilizationOperatorOption{ID: id, Name: data.Requester(work)})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (data CivilizationWorkbench) OwnerCandidates() []civilizationOperatorOption {
	ids := map[string]bool{}
	for _, id := range data.AssignableOperators {
		ids[id] = true
	}
	for _, operator := range data.Operators() {
		ids[operator.ID] = true
	}
	for _, work := range data.Items {
		if work.HumanOwnerID != "" {
			ids[work.HumanOwnerID] = true
		}
	}
	if data.ViewerID != "" {
		ids[data.ViewerID] = true
	}
	var result []civilizationOperatorOption
	for id := range ids {
		if data.ViewerRole != "" && !containsString(data.AssignableOperators, id) {
			continue
		}
		name := data.OperatorNames[id]
		if name == "" {
			name = id
		}
		result = append(result, civilizationOperatorOption{ID: id, Name: name})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (data CivilizationWorkbench) HumanOwner(work CivilizationWork) string {
	if work.HumanOwnerID == "" {
		return "Unassigned"
	}
	if name := data.OperatorNames[work.HumanOwnerID]; name != "" {
		return name
	}
	return work.HumanOwnerID
}

func (data CivilizationWorkbench) WorkOwner(work CivilizationWork) string {
	if civilizationHumanStep(work) != "" {
		return data.HumanOwner(work)
	}
	return civilizationWorkOwner(work)
}

func civilizationHostName(work CivilizationWork) string {
	provider := work.Selection.Provider
	for index := len(work.ProviderRuns) - 1; index >= 0; index-- {
		if execution := work.ProviderRuns[index].Result.Execution; execution != nil {
			provider = execution.Effective.Provider
			break
		}
	}
	switch provider {
	case "codex":
		return "Codex"
	case "claude":
		return "Claude"
	case "":
		return "Configured host"
	default:
		return provider
	}
}

func (h *Handlers) handleCivilizationHumanOwner(w http.ResponseWriter, r *http.Request) {
	if err := requireOpsHiveSameOrigin(r); err != nil {
		http.Error(w, "Assignment must come from this workbench.", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, civilizationMaxIntakeBytes)
	if err := r.ParseForm(); err != nil {
		h.renderCivilizationMutationError(w, r, "Choose a responsible person before saving.")
		return
	}
	viewer := h.viewUser(r)
	if viewer.ID == "" {
		http.Error(w, "Sign in to assign human responsibility.", http.StatusUnauthorized)
		return
	}
	owner := r.FormValue("owner_id")
	data := h.workbenchForRequest(r)
	known := owner == ""
	for _, candidate := range data.OwnerCandidates() {
		if candidate.ID == owner {
			known = true
		}
	}
	if !known {
		h.renderCivilizationMutationError(w, r, "Choose an operator from the workbench.")
		return
	}
	client, err := newCivilizationClient(15 * time.Second)
	if err == nil {
		err = client.request(r.Context(), http.MethodPost, "/api/civilization/v1/work/"+url.PathEscape(r.PathValue("workID"))+"/human-owner", map[string]string{
			"owner_id": owner, "assigned_by": viewer.ID, "previous_assignment_id": r.FormValue("previous_assignment_id"),
		}, nil)
	}
	if err != nil {
		h.renderCivilizationMutationError(w, r, err.Error())
		return
	}
	h.renderCivilizationAfterMutation(w, r)
}

func (data CivilizationWorkbench) filteredItems() []CivilizationWork {
	var result []CivilizationWork
	for _, work := range data.Items {
		if data.Repository != "" && work.Source.Repository != data.Repository {
			continue
		}
		if data.Operator != "" && civilizationRequesterID(work) != data.Operator {
			continue
		}
		if data.Responsible == "unassigned" && work.HumanOwnerID != "" {
			continue
		}
		if data.Responsible != "" && data.Responsible != "unassigned" && data.Responsible != "person:"+work.HumanOwnerID {
			continue
		}
		result = append(result, work)
	}
	return result
}

func civilizationWorkInHistory(work CivilizationWork) bool {
	switch work.State {
	case "approved", "rejected", "changes_requested", "completed":
		return true
	}
	return false
}

func (data CivilizationWorkbench) VisibleItems() []CivilizationWork {
	var result []CivilizationWork
	for _, work := range data.filteredItems() {
		if civilizationWorkInHistory(work) == (data.View == "history") {
			result = append(result, work)
		}
	}
	return result
}

func (data CivilizationWorkbench) HistoryItems() []CivilizationWork {
	var result []CivilizationWork
	for _, work := range data.filteredItems() {
		if civilizationWorkInHistory(work) {
			result = append(result, work)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return result
}

func (data CivilizationWorkbench) PageURL(view, workID string) string {
	query := url.Values{}
	query.Set("work", workID)
	if view == "team" || view == "history" {
		query.Set("view", view)
	}
	if data.Repository != "" {
		query.Set("filter_repository", data.Repository)
	}
	if data.Operator != "" {
		query.Set("operator", data.Operator)
	}
	if data.Responsible != "" {
		query.Set("responsible", data.Responsible)
	}
	return "/console/workbench?" + query.Encode()
}

func (data CivilizationWorkbench) FragmentURL(view, workID string) string {
	return strings.Replace(data.PageURL(view, workID), "/console/workbench?", "/console/workbench/work-fragment?", 1)
}

func civilizationHumanStep(work CivilizationWork) string {
	if work.State == "prepared" {
		return "Review delivered result"
	}
	if work.State == "awaiting_confirmation" {
		return "Confirm brief"
	}
	for _, intervention := range work.Interventions {
		if intervention.Status == "open" {
			return "Answer and retry"
		}
	}
	if work.State == "ready" {
		return "Review pull request"
	}
	if work.State == "blocked" || work.State == "human_required" {
		return "Resolve blocker"
	}
	return ""
}

func civilizationWorkRole(work CivilizationWork) string {
	switch work.State {
	case "routing":
		return "Briefing"
	case "queued":
		return "Waiting for a worker"
	case "implementing":
		return "Implementation"
	case "validating":
		return "Verification"
	case "reviewing":
		return "Review"
	case "publishing", "merge_queued":
		return "Publication"
	default:
		return "No execution in progress"
	}
}

func (data CivilizationWorkbench) TeamCount(kind string) int {
	count := 0
	for _, item := range data.VisibleItems() {
		switch kind {
		case "progress":
			if civilizationWorkGroupIndex(item.State) == 1 {
				count++
			}
		case "human":
			if civilizationHumanStep(item) != "" {
				count++
			}
		case "errors":
			if item.Blocker != "" {
				count++
			}
		case "people": // Computed below from the filtered items.
		}
	}
	if kind == "people" {
		ids := map[string]bool{}
		for _, work := range data.VisibleItems() {
			if id := civilizationRequesterID(work); id != "" {
				ids[id] = true
			}
		}
		return len(ids)
	}
	return count
}

func (h *Handlers) handleCivilizationRuntime(response http.ResponseWriter, request *http.Request) {
	projection, err := fetchHiveOperatorProjection(request)
	wall := buildConsoleHealthWall(projection, err, time.Now().UTC())
	response.Header().Set("Cache-Control", "no-store")
	civilizationRuntimeRoster(wall).Render(request.Context(), response)
}
