package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/transpara-ai/site/auth"
)

func TestWorkbenchTeamRollupDistinguishesRequesterResponsibilityAndExecution(t *testing.T) {
	data := CivilizationWorkbench{Available: true, View: "team", OperatorNames: map[string]string{"alice": "Alice", "bob": "Bob"}, Items: []CivilizationWork{
		{WorkID: "one", State: "awaiting_confirmation", Source: CivilizationSource{Kind: "human", Identity: "human:alice:intake-one", Repository: "transpara-ai/site"}, HumanOwnerID: "bob", IntakeText: "A clear brief"},
		{WorkID: "two", State: "implementing", Source: CivilizationSource{Kind: "human", Identity: "human:bob:intake-two", Repository: "transpara-ai/hive"}},
		{WorkID: "three", State: "blocked", Blocker: "Tests failed", Source: CivilizationSource{Kind: "issue", Identity: "issue:1", Repository: "transpara-ai/site"}},
		{WorkID: "unknown", State: "future", Source: CivilizationSource{Repository: "transpara-ai/hive"}},
	}}
	if data.TeamCount("progress") != 1 || data.TeamCount("human") != 2 || data.TeamCount("errors") != 1 || data.TeamCount("people") != 2 {
		t.Fatal("unknown or missing data was counted as real work/people")
	}
	if data.Requester(data.Items[0]) != "Alice" || data.HumanOwner(data.Items[0]) != "Bob" || data.HumanOwner(data.Items[2]) != "Unassigned" {
		t.Fatal("requester was conflated with the responsible person")
	}
	var html bytes.Buffer
	if err := civilizationTeamBoard(data).Render(context.Background(), &html); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Responsible person: Bob", "Responsible person: Unassigned", "Tests failed", "A clear brief"} {
		if !strings.Contains(html.String(), text) {
			t.Fatalf("missing %q", text)
		}
	}
	data.Operator = "alice"
	data.Repository = "transpara-ai/site"
	if len(data.VisibleItems()) != 1 || data.TeamCount("progress") != 0 {
		t.Fatal("filters did not apply to rollup")
	}
	data.SelectedWorkID = "two"
	if data.Selected() != nil {
		t.Fatal("focused view escaped the selected filters")
	}
}

func TestWorkbenchAssignmentRejectsCrossOriginSubmission(t *testing.T) {
	r := httptest.NewRequest("POST", "https://site.example/console/workbench/work/one/human-owner", nil)
	r.Header.Set("Origin", "https://outside.example")
	w := httptest.NewRecorder()
	NewHandlers(nil, nil, nil).handleCivilizationHumanOwner(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("cross-origin status=%d", w.Code)
	}
}

func TestWorkbenchAssignmentUsesSignedInOperatorAndExactAssignmentVersion(t *testing.T) {
	var received map[string]string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			json.NewDecoder(r.Body).Decode(&received)
			w.Write([]byte(`{}`))
			return
		}
		w.Write([]byte(`{"items":[{"work_id":"one","state":"awaiting_confirmation","source":{"kind":"human","identity":"human:bob:intake-one"}}]}`))
	}))
	defer upstream.Close()
	configureCivilizationTestClient(t, upstream.URL)
	r := httptest.NewRequest("POST", "/console/workbench/work/one/human-owner", strings.NewReader("owner_id=bob&assigned_by=forged&previous_assignment_id=exact-version"))
	r.SetPathValue("workID", "one")
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("HX-Request", "true")
	r.Header.Set("HX-Target", "civilization-work-list")
	r = r.WithContext(auth.ContextWithUser(r.Context(), &auth.User{ID: "alice", Name: "Alice"}))
	w := httptest.NewRecorder()
	NewHandlers(nil, nil, nil).handleCivilizationHumanOwner(w, r)
	if received["assigned_by"] != "alice" || received["owner_id"] != "bob" || received["previous_assignment_id"] != "exact-version" {
		t.Fatalf("assignment=%v", received)
	}
	if strings.Contains(w.Body.String(), `id="workbench-composer"`) {
		t.Fatal("assignment replaced unrelated intake draft")
	}
}
