package graph

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CivilizationResultReference struct {
	WorkID          string `json:"work_id"`
	ResultID        string `json:"result_id"`
	WorkspaceDigest string `json:"workspace_digest"`
}

type CivilizationResultReview struct {
	ResultID        string    `json:"result_id"`
	WorkspaceDigest string    `json:"workspace_digest"`
	Decision        string    `json:"decision"`
	Feedback        string    `json:"feedback"`
	ReviewedBy      string    `json:"reviewed_by"`
	RevisionWorkID  string    `json:"revision_work_id"`
	RecordedAt      time.Time `json:"recorded_at"`
}

func civilizationResultAvailable(work CivilizationWork) bool {
	switch work.State {
	case "prepared", "approved", "rejected", "changes_requested":
		return true
	}
	return false
}

func civilizationResultDecisionLabel(decision string) string {
	switch decision {
	case "approve":
		return "Result approved"
	case "reject":
		return "Result rejected"
	case "request_changes":
		return "Changes requested"
	}
	return decision
}

func (h *Handlers) handleCivilizationResultReview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, civilizationMaxIntakeBytes)
	if err := r.ParseForm(); err != nil {
		h.renderCivilizationMutationError(w, r, "The review was too large or malformed.")
		return
	}
	decision, feedback := r.FormValue("decision"), strings.TrimSpace(r.FormValue("feedback"))
	if decision != "approve" && decision != "reject" && decision != "request_changes" {
		h.renderCivilizationMutationError(w, r, "Choose how to review this result.")
		return
	}
	if decision != "approve" && feedback == "" {
		h.renderCivilizationMutationError(w, r, "Add a reason for rejection or describe the changes you want.")
		return
	}
	viewer := h.viewUser(r)
	if viewer.ID == "" {
		h.renderCivilizationMutationError(w, r, "Sign in before reviewing delivered work.")
		return
	}
	client, err := newCivilizationClient(15 * time.Second)
	var reviewed CivilizationWork
	if err == nil {
		err = client.request(r.Context(), http.MethodPost, "/api/civilization/v1/work/"+url.PathEscape(r.PathValue("workID"))+"/result-review", map[string]string{
			"result_id": r.FormValue("result_id"), "workspace_digest": r.FormValue("workspace_digest"), "decision": decision, "feedback": feedback, "reviewed_by": viewer.ID,
		}, &reviewed)
	}
	if err != nil {
		h.renderCivilizationMutationError(w, r, err.Error())
		return
	}
	h.renderCivilizationAfterMutation(w, r)
}
