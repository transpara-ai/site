package graph

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConsoleKanbanCivilizationRoutesAndLiveUpdates(t *testing.T) {
	state := "human_required"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/civilization/v1/work" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+strings.Repeat("k", 32) {
			t.Error("missing Civilization credential")
		}
		json.NewEncoder(w).Encode(map[string]any{"items": []CivilizationWork{{
			WorkID: "work-real", State: state, IntakeText: "Fix <script>alert(1)</script>",
			Source:       CivilizationSource{Kind: "human", Identity: "human:alice:intake-1", Repository: "transpara-ai/site"},
			HumanOwnerID: "bob", Blocker: "Clarification needed", UpdatedAt: time.Now().UTC().Add(-time.Hour),
			Selection: CivilizationSelection{Provider: "codex", Model: "gpt-5.6-sol"},
		}}})
	}))
	defer upstream.Close()
	configureCivilizationTestClient(t, upstream.URL)
	t.Setenv("WORK_API_BASE_URL", "")
	t.Setenv("WORK_UI_BASE_URL", "")
	t.Setenv("WORK_API_KEY", "legacy-key-must-not-be-forwarded")
	h := NewHandlers(nil, nil, nil)
	for _, lens := range []string{"", "status", "risk", "agent", "source"} {
		w := httptest.NewRecorder()
		h.handleConsoleKanban(w, httptest.NewRequest("GET", "/console/kanban?lens="+lens, nil))
		body := w.Body.String()
		for _, want := range []string{`data-work-id="work-real"`, `data-freshness="current"`, "alice", "bob", "Clarification needed", "codex / gpt-5.6-sol", "Updated 1h ago", `hx-trigger="every 5s"`} {
			if !strings.Contains(body, want) {
				t.Errorf("lens %q missing %q", lens, want)
			}
		}
		if strings.Contains(body, "<script>alert(1)</script>") || strings.Contains(body, "404 Not Found") || strings.Contains(body, strings.Repeat("k", 32)) {
			t.Error("unsafe or broken board")
		}
		if lens == "" && !strings.Contains(body, `href="/console/kanban?lens=status" aria-current="true"`) {
			t.Error("Civilization should default to Status")
		}
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/console/kanban/order/work-real", nil)
	r.SetPathValue("id", "work-real")
	h.handleConsoleKanbanOrder(w, r)
	for _, want := range []string{"Responsible person", "bob", "Not linked by this work feed", `href="/console/workbench?work=work-real"`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("drawer missing %q", want)
		}
	}
	state = "completed"
	w = httptest.NewRecorder()
	h.handleConsoleKanbanFragment(w, httptest.NewRequest("GET", "/console/kanban/fragment?lens=status", nil))
	if !strings.Contains(w.Body.String(), "Completed") || strings.Contains(w.Body.String(), `id="console-kanban-drawer"`) {
		t.Error("poll did not update state or would erase drawer")
	}
}

func TestConsoleKanbanCivilizationFailureAndEmpty(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
		available  bool
	}{
		{"empty", `{"items":[]}`, 200, true},
		{"wrong envelope", `{"tasks":[]}`, 200, false},
		{"null envelope", `null`, 200, false},
		{"invalid array", `{"items":{}}`, 200, false},
		{"missing identity", `{"items":[{"state":"prepared"}]}`, 200, false},
		{"missing state", `{"items":[{"work_id":"x"}]}`, 200, false},
		{"duplicate identity", `{"items":[{"work_id":"x","state":"prepared"},{"work_id":"x","state":"completed"}]}`, 200, false},
		{"upstream failure", `{"error":"Work store unavailable"}`, 503, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.code); io.WriteString(w, tc.body) }))
			defer upstream.Close()
			configureCivilizationTestClient(t, upstream.URL)
			t.Setenv("WORK_API_BASE_URL", "")
			t.Setenv("WORK_UI_BASE_URL", "")
			result := fetchConsoleWork(httptest.NewRequest("GET", "/console/kanban", nil))
			if (result.Err == nil) != tc.available {
				t.Fatalf("available=%v error=%v", tc.available, result.Err)
			}
			w := httptest.NewRecorder()
			NewHandlers(nil, nil, nil).handleConsoleKanban(w, httptest.NewRequest("GET", "/console/kanban", nil))
			body := w.Body.String()
			if tc.available {
				if !strings.Contains(body, "No work yet.") || !strings.Contains(body, "Create work") || strings.Contains(body, "No factory orders") {
					t.Error("incorrect Civilization empty state")
				}
			} else if !strings.Contains(body, `data-freshness="unavailable"`) || strings.Contains(body, "No work yet.") || strings.Contains(body, "0 items") {
				t.Error("failed feed looked empty")
			}
		})
	}
}

func TestConsoleKanbanExplicitLegacyConfigurationWins(t *testing.T) {
	for _, key := range []string{"WORK_API_BASE_URL", "WORK_UI_BASE_URL"} {
		t.Run(key, func(t *testing.T) {
			legacy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/tasks" {
					t.Errorf("legacy path=%s", r.URL.Path)
				}
				io.WriteString(w, `{"tasks":[{"id":"legacy-task","status":"running"}]}`)
			}))
			defer legacy.Close()
			t.Setenv("WORK_API_BASE_URL", "")
			t.Setenv("WORK_UI_BASE_URL", "")
			t.Setenv(key, legacy.URL)
			t.Setenv("CIVILIZATION_API_BASE_URL", "http://127.0.0.1:1")
			r := httptest.NewRequest("GET", "/console/kanban", nil)
			result := fetchConsoleWork(r)
			if result.Err != nil || result.Civilization || len(result.Tasks) != 1 || result.Tasks[0].ID != "legacy-task" {
				t.Fatalf("legacy fetch: %+v", result)
			}
			if consoleRequestLens(r) != LensRisk {
				t.Error("legacy default changed")
			}
		})
	}
}

func TestCivilizationKanbanDoesNotInventTaskEvidence(t *testing.T) {
	now := time.Now().UTC()
	work := CivilizationWork{WorkID: "work-1", State: "prepared", UpdatedAt: now.Add(-time.Hour), Selection: CivilizationSelection{Provider: "codex", Model: "selected"}}
	card := cardForCivilizationWork(work, now)
	if card.CreatedAt != "" || card.AgeLabel != "" || card.Agent != "" || card.Risk != "" || card.FactoryOrderID != "" {
		t.Fatalf("fabricated task evidence: %+v", card)
	}
	if card.UpdatedLabel != "1h" || !strings.Contains(card.Provider, "selected") {
		t.Fatalf("update/provider evidence missing: %+v", card)
	}
	result := consoleWorkResult{Civilization: true, Cards: []ConsoleOrderCard{card, {ID: "blocked", Status: "blocked", Civilization: true}, {ID: "completed", Status: "completed", Civilization: true}, {ID: "future", Status: "new_state", Civilization: true}}}
	board := result.board(LensStatus, now)
	if got := strings.Join(columnKeys(board), ","); got != "blocked,prepared,completed,new_state" {
		t.Fatalf("status order=%s", got)
	}
	for _, lens := range []ConsoleKanbanLens{LensRisk, LensAgent, LensSource} {
		if board := result.board(lens, now); board.TotalCards != 4 {
			t.Fatalf("lens %s dropped cards", lens)
		}
	}
}
