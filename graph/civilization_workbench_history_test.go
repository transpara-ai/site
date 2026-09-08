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
	"time"

	"github.com/transpara-ai/site/auth"
)

func TestWorkbenchHistoryKeepsReviewedWorkOutOfCurrentFocusAndTotals(t *testing.T) {
	data := CivilizationWorkbench{Available: true, Items: []CivilizationWork{{WorkID: "next", State: "prepared"}}}
	for i, state := range []string{"approved", "rejected", "changes_requested", "completed"} {
		data.Items = append(data.Items, CivilizationWork{WorkID: state, State: state, UpdatedAt: time.Unix(int64(i), 0), Blocker: "Historical error", Source: CivilizationSource{Repository: "transpara-ai/site"}})
	}
	data.SelectedWorkID = "approved" // An already-open page or old bookmark.
	if data.Selected().WorkID != "next" || len(data.VisibleItems()) != 1 || len(data.HistoryItems()) != 4 || data.TeamCount("errors") != 0 || data.TeamCount("human") != 1 {
		t.Fatal("reviewed results still occupy current work")
	}
	var body bytes.Buffer
	CivilizationWorkList(data).Render(context.Background(), &body)
	if strings.Contains(body.String(), `id="work-select-approved"`) || !strings.Contains(body.String(), `id="history-work-approved"`) || strings.Contains(body.String(), `id="workbench-history" class="wb-history" data-workbench-details open`) {
		t.Fatal("history must be accessible separately and collapsed by default")
	}
	data.View = "history"
	if data.Selected().WorkID != "approved" || data.Groups()[0].Items[0].WorkID != "completed" {
		t.Fatal("explicit history selection or newest-first history was lost")
	}
	body.Reset()
	CivilizationWorkList(data).Render(context.Background(), &body)
	if !strings.Contains(body.String(), `data-work-state="approved"`) || !strings.Contains(body.String(), "Inspect prepared diff") {
		t.Fatal("history no longer exposes the original artifact")
	}
	data.Repository = "transpara-ai/hive"
	if data.Selected() != nil || len(data.HistoryItems()) != 0 {
		t.Fatal("history escaped repository filters")
	}
	data = CivilizationWorkbench{Available: true, Items: data.Items[1:]}
	body.Reset()
	CivilizationWorkList(data).Render(context.Background(), &body)
	if data.Selected() != nil || strings.Contains(body.String(), `id="workbench-detail"`) || !strings.Contains(body.String(), "all caught up") {
		t.Fatal("completed queue should have a quiet empty state")
	}
}

func TestResultReviewAdvancesFocusAndPreservesFilters(t *testing.T) {
	for _, decision := range []string{"approve", "reject", "request_changes"} {
		for _, htmx := range []bool{false, true} {
			t.Run(decision+"/htmx="+map[bool]string{true: "true", false: "false"}[htmx], func(t *testing.T) {
				state := map[string]string{"approve": "approved", "reject": "rejected", "request_changes": "changes_requested"}[decision]
				reviewed := CivilizationWork{WorkID: "smoke", State: state, Source: CivilizationSource{Repository: "transpara-ai/site"}, ResultReview: &CivilizationResultReview{Decision: decision}}
				items := []CivilizationWork{reviewed, {WorkID: "next", State: "prepared", Source: reviewed.Source}}
				want := "next"
				if decision == "request_changes" {
					reviewed.ResultReview.RevisionWorkID = "revision"
					items = append(items, CivilizationWork{WorkID: "revision", State: "routing", Source: reviewed.Source})
					want = "revision"
				}
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "POST" {
						json.NewEncoder(w).Encode(reviewed)
					} else {
						json.NewEncoder(w).Encode(map[string]any{"items": items})
					}
				}))
				defer upstream.Close()
				configureCivilizationTestClient(t, upstream.URL)
				form := url.Values{"decision": {decision}, "feedback": {"Review feedback"}, "filter_repository": {"transpara-ai/site"}}
				r := httptest.NewRequest("POST", "/console/workbench/work/smoke/result-review", strings.NewReader(form.Encode()))
				r.SetPathValue("workID", "smoke")
				r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				if htmx {
					r.Header.Set("HX-Request", "true")
					r.Header.Set("HX-Target", "civilization-work-list")
				}
				r = r.WithContext(auth.ContextWithUser(r.Context(), &auth.User{ID: "alice", Name: "Alice"}))
				w := httptest.NewRecorder()
				NewHandlers(nil, nil, nil).handleCivilizationResultReview(w, r)
				header := "Location"
				if htmx {
					header = "HX-Push-Url"
					if !strings.Contains(w.Body.String(), `data-work-id="`+want+`"`) || !strings.Contains(w.Body.String(), "Saved in History") || strings.Contains(w.Body.String(), `data-work-id="smoke"`) {
						t.Fatal("review did not advance the visible detail")
					}
				}
				location, _ := url.Parse(w.Header().Get(header))
				if location.Query().Get("work") != want || location.Query().Get("filter_repository") != "transpara-ai/site" {
					t.Fatalf("unexpected navigation: %s", location)
				}
			})
		}
	}
}

func TestWorkbenchPollRetiresSelectionReviewedInAnotherSession(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[{"work_id":"smoke","state":"approved"}]}`))
	}))
	defer upstream.Close()
	configureCivilizationTestClient(t, upstream.URL)
	for _, view := range []string{"focus", "history"} {
		r := httptest.NewRequest("GET", "/console/workbench/work-fragment?work=smoke&view="+view, nil)
		r.Header.Set("HX-Request", "true")
		r.Header.Set("HX-Trigger", "civilization-work-list")
		r.Header.Set("HX-Target", "civilization-work-list")
		w := httptest.NewRecorder()
		NewHandlers(nil, nil, nil).handleCivilizationWorkList(w, r)
		if view == "focus" {
			if w.Header().Get("HX-Replace-Url") != "/console/workbench?work=" || strings.Contains(w.Body.String(), `id="workbench-detail"`) {
				t.Fatal("remote review remained prominent or pinned in the URL")
			}
		} else if w.Header().Get("HX-Replace-Url") != "" || !strings.Contains(w.Body.String(), `data-work-state="approved"`) {
			t.Fatal("poll discarded an explicitly opened history item")
		}
	}
}
