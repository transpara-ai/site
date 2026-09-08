package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/transpara-ai/site/auth"
)

func TestPreparedResultShowsReviewControlsAndRetainsFeedback(t *testing.T) {
	work := CivilizationWork{WorkID: "smoke", State: "prepared", PreparedResultID: "result-exact", PreparedResultDigest: "digest-exact"}
	var body bytes.Buffer
	civilizationWorkDetail(work, CivilizationWorkbench{ResultFeedback: "Please improve <script>unsafe</script>"}).Render(context.Background(), &body)
	for _, want := range []string{"Approve result", "Reject result", "Request changes / enhance", "result-exact", "digest-exact", "Inspect prepared diff", "Review delivered work"} {
		if !strings.Contains(body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body.String(), "<script>unsafe</script>") {
		t.Fatal("unescaped feedback")
	}
	for _, state := range []string{"approved", "rejected", "changes_requested"} {
		work.State = state
		body.Reset()
		civilizationWorkDetail(work, CivilizationWorkbench{}).Render(context.Background(), &body)
		if strings.Contains(body.String(), `name="decision"`) || !strings.Contains(body.String(), "Inspect prepared diff") {
			t.Errorf("incorrect actions for %s", state)
		}
	}
	if civilizationHumanStep(CivilizationWork{State: "prepared"}) != "Review delivered result" {
		t.Fatal("prepared result not routed to human")
	}
}

func TestResultReviewForwardsExactResultAndAuthenticatedReviewer(t *testing.T) {
	var payload map[string]string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if r.URL.Path != "/api/civilization/v1/work/smoke/result-review" || r.Header.Get("Authorization") != "Bearer "+strings.Repeat("k", 32) {
				t.Error("wrong endpoint or auth")
			}
			json.NewDecoder(r.Body).Decode(&payload)
			w.Write([]byte(`{"work_id":"smoke","state":"approved"}`))
			return
		}
		w.Write([]byte(`{"items":[]}`))
	}))
	defer upstream.Close()
	configureCivilizationTestClient(t, upstream.URL)
	form := url.Values{"decision": {"approve"}, "result_id": {"result-exact"}, "workspace_digest": {"digest-exact"}, "reviewed_by": {"forged-person"}, "feedback": {" Looks good "}}
	r := httptest.NewRequest("POST", "/console/workbench/work/smoke/result-review", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("workID", "smoke")
	r = r.WithContext(auth.ContextWithUser(r.Context(), &auth.User{ID: "alice", Name: "Alice"}))
	w := httptest.NewRecorder()
	NewHandlers(nil, nil, nil).handleCivilizationResultReview(w, r)
	if payload["reviewed_by"] != "alice" || payload["result_id"] != "result-exact" || payload["workspace_digest"] != "digest-exact" || payload["feedback"] != "Looks good" {
		t.Fatalf("payload=%v", payload)
	}
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/console/workbench?work=" {
		t.Fatalf("response=%d %v", w.Code, w.Header())
	}
}

func TestResultReviewConflictKeepsFeedback(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(409)
			w.Write([]byte(`{"error":"Delivered result changed"}`))
			return
		}
		w.Write([]byte(`{"items":[{"work_id":"smoke","state":"prepared","prepared_result_id":"new-result","prepared_result_digest":"new-digest"}]}`))
	}))
	defer upstream.Close()
	configureCivilizationTestClient(t, upstream.URL)
	r := httptest.NewRequest("POST", "/console/workbench/work/smoke/result-review", strings.NewReader("decision=request_changes&feedback=Keep+my+enhancement&result_id=old-result&workspace_digest=old-digest"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("HX-Request", "true")
	r.Header.Set("HX-Target", "civilization-work-list")
	r.SetPathValue("workID", "smoke")
	r = r.WithContext(auth.ContextWithUser(r.Context(), &auth.User{ID: "alice", Name: "Alice"}))
	w := httptest.NewRecorder()
	NewHandlers(nil, nil, nil).handleCivilizationResultReview(w, r)
	for _, want := range []string{"Delivered result changed", "Keep my enhancement", "new-result"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("missing %q", want)
		}
	}
}
