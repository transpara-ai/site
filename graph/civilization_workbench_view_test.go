package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWorkbenchSelectionPrioritizesAttentionAndKeepsExplicitMissingSelection(t *testing.T) {
	data := CivilizationWorkbench{Available: true, Items: []CivilizationWork{
		{WorkID: "prepared", State: "prepared", IntakeText: "Prepared result"},
		{WorkID: "older", State: "blocked", UpdatedAt: time.Unix(1, 0)},
		{WorkID: "newer", State: "awaiting_confirmation", UpdatedAt: time.Unix(2, 0)},
	}}
	if data.Selected().WorkID != "newer" || data.Items[0].WorkID != "prepared" {
		t.Fatal("attention order or original source order changed")
	}
	data.SelectedWorkID = "prepared"
	var html bytes.Buffer
	if err := CivilizationWorkList(data).Render(context.Background(), &html); err != nil {
		t.Fatal(err)
	}
	if strings.Count(html.String(), `id="workbench-detail"`) != 1 || !strings.Contains(html.String(), `data-work-id="prepared"`) {
		t.Fatal("selection must render exactly one matching detail")
	}
	data.SelectedWorkID = "missing&work=newer"
	if data.Selected() != nil || !strings.Contains(data.PollURL(), "missing%26work%3Dnewer") {
		t.Fatal("missing or unsafe selection silently changed to another item")
	}
	for _, work := range []CivilizationWork{{State: "unknown"}, {State: "blocked"}, {State: "human_required"}} {
		for i := 0; i < 4; i++ {
			if civilizationStageStatus(work, i) != "pending" {
				t.Fatal("unknown progress inferred as complete")
			}
		}
	}
	if civilizationStageStatus(CivilizationWork{State: "prepared"}, 3) != "current" {
		t.Fatal("prepared result must not imply completed publication")
	}
}

func TestWorkbenchArtifactPreviewEscapesAndBoundsDiffWhileDownloadStaysComplete(t *testing.T) {
	patch := "+<script>alert(1)</script>\n" + strings.Repeat("+line\n", 450)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(civilizationArtifact{Repository: "transpara-ai/site", Patch: patch})
	}))
	defer upstream.Close()
	configureCivilizationTestClient(t, upstream.URL)
	for _, htmx := range []bool{true, false} {
		req := httptest.NewRequest("GET", "/console/workbench/work/example/artifact", nil)
		req.SetPathValue("workID", "example")
		if htmx {
			req.Header.Set("HX-Request", "true")
		}
		response := httptest.NewRecorder()
		NewHandlers(nil, nil, nil).handleCivilizationArtifact(response, req)
		body := response.Body.String()
		if response.Code != 200 || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("artifact response unavailable or cacheable")
		}
		if htmx {
			if strings.Contains(body, "<script>") || !strings.Contains(body, "&lt;script&gt;") || !strings.Contains(body, "Preview shortened") || strings.Count(body, `data-diff=`) != 400 {
				t.Fatal("preview must escape content and bound line count")
			}
		} else if !strings.HasSuffix(body, patch) || response.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
			t.Fatal("full artifact download changed")
		}
	}
	lines, shortened := civilizationDiffPreview(strings.Repeat("界", 3000))
	if !shortened || len([]rune(lines[0].Text)) != 2001 {
		t.Fatal("long unicode line not bounded")
	}
}

func TestWorkbenchConfirmationTargetsBoardAndKeepsExactBrief(t *testing.T) {
	var confirmed string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			confirmed = payload["brief_id"]
			w.Write([]byte(`{}`))
			return
		}
		w.Write([]byte(`{"items":[{"work_id":"selected","state":"implementing","intake_text":"Implement"}]}`))
	}))
	defer upstream.Close()
	configureCivilizationTestClient(t, upstream.URL)
	req := httptest.NewRequest("POST", "/console/workbench/work/selected/confirm", strings.NewReader("brief_id=exact-bound-brief"))
	req.SetPathValue("workID", "selected")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "civilization-work-list")
	response := httptest.NewRecorder()
	NewHandlers(nil, nil, nil).handleCivilizationConfirm(response, req)
	if confirmed != "exact-bound-brief" || strings.Contains(response.Body.String(), `id="workbench-composer"`) || !strings.Contains(response.Body.String(), `data-work-id="selected"`) {
		t.Fatal("confirmation lost its binding, selection, or replaced unrelated composer")
	}
	if response.Header().Get("HX-Push-Url") != "/console/workbench?work=selected" {
		t.Fatal("confirmed work is not reloadable by URL")
	}
}
