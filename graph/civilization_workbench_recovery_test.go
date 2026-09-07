package graph

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkbenchNeverLabelsFailedOrUnknownChecksPassed(t *testing.T) {
	for _, status := range []string{"failed", "blocked", "skipped", "invented"} {
		var item CivilizationWork
		run := CivilizationProviderRun{Operation: "implement"}
		run.Result.Status = "blocked"
		run.Result.Checks = append(run.Result.Checks, struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		}{Name: "native verification", Status: status, Summary: "needs attention"})
		item.ProviderRuns = []CivilizationProviderRun{run}
		var html bytes.Buffer
		if err := civilizationWorkCard(item).Render(context.Background(), &html); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(html.String(), ">Passed</span>") {
			t.Fatalf("%s was labeled passed", status)
		}
	}
}

func TestWorkbenchHTMXErrorPreservesIntake(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"Selected model is unavailable"}`))
			return
		}
		w.Write([]byte(`{"items":[]}`))
	}))
	defer upstream.Close()
	configureCivilizationTestClient(t, upstream.URL)
	req := httptest.NewRequest("POST", "/console/workbench/intake", strings.NewReader("repository=transpara-ai%2Fhive&source_identity=unchanged&text=Preserve+my+request&provider=claude&model=unavailable"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	response := httptest.NewRecorder()
	NewHandlers(nil, nil, nil).handleCivilizationIntake(response, req)
	if response.Code != 200 {
		t.Fatalf("HTMX will not swap status %d", response.Code)
	}
	for _, text := range []string{"Selected model is unavailable", "Preserve my request", `value="unchanged"`, `value="unavailable"`} {
		if !strings.Contains(response.Body.String(), text) {
			t.Fatalf("lost %q", text)
		}
	}
}
