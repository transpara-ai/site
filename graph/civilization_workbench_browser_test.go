package graph

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/transpara-ai/site/auth"
)

// Opt-in browser fixture exercises actual Site routes/templates and HTMX with
// deterministic API states. Real Hive persistence is covered by Platform's journey.
func TestWorkbenchBrowserFixture(t *testing.T) {
	urlFile := os.Getenv("WORKBENCH_BROWSER_URL_FILE")
	if urlFile == "" {
		t.Skip("opt-in browser fixture")
	}
	var items []CivilizationWork
	if err := json.Unmarshal([]byte(`[
	{"work_id":"brief","state":"awaiting_confirmation","source":{"repository":"transpara-ai/site"},"intake_text":"Make the workbench easier to navigate","selection":{"provider":"codex"},"bound":{"idempotency_key":"exact-brief","envelope":{"route":"Designed","workflow":{"name":"transpara-tlc","version":"0.1.2"},"brief":{"outcome":"Make the workbench easier to navigate","scope":["Group current work by the attention it needs","Focus on one work item with a clear next action","Keep drafts while progress refreshes"],"tests":["Exercise confirmation, retry and diff inspection in the browser","Check keyboard navigation and mobile layouts"],"non_goals":["Change execution or publication authority"]}}},"updated_at":"2026-09-08T15:20:00Z"},
	{"work_id":"blocked","state":"human_required","resume_state":"implementing","source":{"repository":"transpara-ai/hive"},"intake_text":"Clarify retry guidance for operators","blocker":"The implementation needs your choice of terminology.","interventions":[{"id":"answer","status":"open","prompt":"Which term should the operator guide use?"}],"updated_at":"2026-09-08T15:10:00Z"},
	{"work_id":"active","state":"implementing","source":{"repository":"transpara-ai/hive"},"intake_text":"Improve execution progress messages","selection":{"provider":"codex","model":"configured-model"},"next_action":"Implementation is running in an isolated worktree.","updated_at":"2026-09-08T15:00:00Z"},
	{"work_id":"prepared","state":"prepared","source":{"repository":"transpara-ai/site"},"intake_text":"Add verification guidance to the operator guide","provider_runs":[{"operation":"implement","result":{"summary":"Added a concise verification checklist and recovery guidance.","checks":[{"name":"make verify","status":"passed","summary":"Native checks passed."}]}},{"operation":"review","result":{"review":{"status":"passed","summary":"No unresolved findings."}}}],"updated_at":"2026-09-08T14:50:00Z"}
	]`), &items); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/artifact"):
			json.NewEncoder(w).Encode(civilizationArtifact{Repository: "transpara-ai/site", Branch: "codex/operator-guide", BaseSHA: "fixture-base", WorkspaceDigest: "fixture-reviewed-digest", Patch: "diff --git a/README.md b/README.md\n--- a/README.md\n+++ b/README.md\n@@ -4,1 +4,4 @@\n ## Verification\n+Run make verify.\n+Review the result before publishing.\n+<script>window.diffExecuted = true</script>\n"})
		case r.Method == "GET":
			json.NewEncoder(w).Encode(map[string]any{"items": items})
		case strings.HasSuffix(r.URL.Path, "/intake"):
			w.WriteHeader(400)
			w.Write([]byte(`{"error":"Selected model is unavailable"}`))
		case strings.HasSuffix(r.URL.Path, "/confirm"):
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["brief_id"] != "exact-brief" {
				w.WriteHeader(409)
				w.Write([]byte(`{"error":"Brief changed"}`))
				return
			}
			items[0].State = "implementing"
			json.NewEncoder(w).Encode(items[0])
		case strings.HasSuffix(r.URL.Path, "/resolve"):
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["resolution"] == "fail" {
				w.WriteHeader(409)
				w.Write([]byte(`{"error":"Retry is temporarily unavailable"}`))
				return
			}
			items[1].Interventions = nil
			items[1].Blocker = ""
			items[1].State = "implementing"
			json.NewEncoder(w).Encode(items[1])
		default:
			w.WriteHeader(404)
		}
	}))
	defer upstream.Close()
	configureCivilizationTestClient(t, upstream.URL)
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("../static"))))
	done := make(chan struct{}, 1)
	mux.HandleFunc("POST /__fixture/stop", func(w http.ResponseWriter, r *http.Request) {
		select {
		case done <- struct{}{}:
		default:
		}
		w.WriteHeader(204)
	})
	identity := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next(w, r.WithContext(auth.ContextWithUser(r.Context(), &auth.User{ID: "browser-fixture", Name: "Fixture"})))
		})
	}
	NewHandlers(nil, identity, identity).Register(mux)
	site := httptest.NewServer(mux)
	defer site.Close()
	if err := os.WriteFile(urlFile, []byte(site.URL), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(10 * time.Minute):
		t.Fatal("browser fixture timed out")
	}
}
