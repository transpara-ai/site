package graph

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCivilizationWorkbenchIsNaturalLanguageFirstAndTechnicalDetailsCollapsed(t *testing.T) {
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		if request.Header.Get("Authorization") != "Bearer "+strings.Repeat("k", 32) {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"items":[{
  "work_id":"work-aaaaaaaaaaaaaaaaaaaaaaaa",
  "source":{"kind":"human","identity":"human:test","repository":"transpara-ai/hive"},
  "intake_text":"Improve the experience <script>alert(1)</script>",
  "bound":{"idempotency_key":"tlc-envelope-v1-test","envelope":{"schema_version":"tlc-envelope/v1","workflow":{"name":"transpara-tlc","version":"0.1.1"},"route":"Routine","brief":{"outcome":"A clear operator experience","scope":[],"non_goals":[],"assumptions":[],"constraints":[],"tests":[],"next_action":"Implement"}},"future_transport_field":true},
  "state":"reviewing","summary":"Review complete","next_action":"Continue","provider_runs":[
    {"operation":"implement","attempt_id":"attempt-1","result":{"status":"passed","summary":"Implemented the clearer experience","changed_files":["graph/console.templ"],"checks":[{"name":"go test ./...","status":"passed","summary":"all packages passed"}]}},
    {"operation":"review","attempt_id":"attempt-2","result":{"status":"passed","summary":"review complete","changed_files":[],"checks":[],"review":{"status":"passed","summary":"No unresolved findings","findings":[]}}}
  ],"interventions":[],"updated_at":"2026-09-03T12:00:00Z","latest_event_id":"future-event"
}]}`)
	}))
	defer upstream.Close()
	configureCivilizationTestClient(t, upstream.URL)

	handler := NewHandlers(nil, nil, nil)
	response := httptest.NewRecorder()
	handler.handleCivilizationWorkbench(response, httptest.NewRequest(http.MethodGet, "/console/workbench", nil))
	body := response.Body.String()
	for _, wanted := range []string{
		`data-console-surface="civilization-workbench"`, "What should Civilization accomplish?", "A clear operator experience",
		"Start work", "records the intake immediately", "Implementation result", "all packages passed", "No unresolved findings",
		"Technical details", "graph/console.templ", "transpara-tlc", "0.1.1", "Historical Factory evidence (read-only)",
	} {
		if !strings.Contains(body, wanted) {
			t.Fatalf("body missing %q:\n%s", wanted, body)
		}
	}
	if strings.Contains(body, "<script>alert(1)</script>") || strings.Contains(body, `"future_transport_field"`) {
		t.Fatalf("workbench exposed unsafe or raw transport data:\n%s", body)
	}
	if requests != 1 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestCivilizationWorkbenchIntakeUsesServerSideCredential(t *testing.T) {
	var got map[string]string
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost {
			if request.Header.Get("Authorization") != "Bearer "+strings.Repeat("k", 32) {
				t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
			}
			if err := json.NewDecoder(request.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			response.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(response, `{"work_id":"work-aaaaaaaaaaaaaaaaaaaaaaaa","state":"queued","source":{},"provider_runs":[],"interventions":[],"updated_at":"2026-09-03T12:00:00Z"}`)
			return
		}
		_, _ = io.WriteString(response, `{"items":[]}`)
	}))
	defer upstream.Close()
	configureCivilizationTestClient(t, upstream.URL)

	handler := NewHandlers(nil, nil, nil)
	request := httptest.NewRequest(http.MethodPost, "/console/workbench/intake", strings.NewReader("repository=transpara-ai%2Fhive&source_identity=intake-test&text=Make+it+clear"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	response := httptest.NewRecorder()
	handler.handleCivilizationIntake(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if got["text"] != "Make it clear" || got["repository"] != "transpara-ai/hive" || got["source_kind"] != "human" || !strings.HasPrefix(got["source_identity"], "human::intake-test") {
		t.Fatalf("intake = %#v", got)
	}
	if strings.Contains(response.Body.String(), strings.Repeat("k", 32)) {
		t.Fatal("server-side credential leaked to rendered response")
	}
}

func configureCivilizationTestClient(t *testing.T, base string) {
	t.Helper()
	keyPath := filepath.Join(t.TempDir(), "api-key")
	if err := os.WriteFile(keyPath, []byte(strings.Repeat("k", 32)), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CIVILIZATION_API_BASE_URL", base)
	t.Setenv("CIVILIZATION_API_KEY_FILE", keyPath)
	t.Setenv("CIVILIZATION_REPOSITORIES", "transpara-ai/hive,transpara-ai/site")
}
