package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newConsoleTestHandlers builds Handlers with a nil store (no DB) and
// pass-through auth wraps, matching the codebase's no-DB test pattern
// (graph/observatory_test.go:458). The console read-only handlers never touch
// the store; viewUser guards h.store == nil.
func newConsoleTestHandlers() *Handlers {
	passthrough := func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) })
	}
	return NewHandlers(nil, passthrough, passthrough)
}

func TestHandleConsoleHealth(t *testing.T) {
	t.Run("unavailable when upstream unset renders explicit state, not green", func(t *testing.T) {
		h := newConsoleTestHandlers()
		t.Setenv("HIVE_OPS_API_BASE_URL", "")

		mux := http.NewServeMux()
		h.Register(mux)

		req := httptest.NewRequest(http.MethodGet, "http://site.test/console/health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "unavailable") {
			t.Fatal("expected explicit unavailable state in body")
		}
	})

	t.Run("renders agents from a live upstream", func(t *testing.T) {
		h := newConsoleTestHandlers()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/hive/operator-projection" {
				http.NotFound(w, r)
				return
			}
			proj := OpsHiveProjection{GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
			proj.RuntimeEvidence.AgentEvents.ObservedActive = 1
			proj.RuntimeEvidence.AgentEvents.ActiveAgents = []OpsHiveRuntimeAgent{{Name: "Guardian", Role: "guardian", Model: "sonnet-4-6"}}
			if err := json.NewEncoder(w).Encode(proj); err != nil {
				t.Errorf("encode projection: %v", err)
			}
		}))
		defer srv.Close()
		t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)

		mux := http.NewServeMux()
		h.Register(mux)
		req := httptest.NewRequest(http.MethodGet, "http://site.test/console/health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Guardian") {
			t.Fatal("expected agent name in rendered wall")
		}
	})
}

func TestBuildConsoleHealthWall(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)

	t.Run("fetch error renders unavailable with notice", func(t *testing.T) {
		wall := buildConsoleHealthWall(nil, errors.New("HIVE_OPS_API_BASE_URL is not configured"), now)
		if wall.Freshness != FreshnessUnavailable {
			t.Fatalf("freshness = %q, want unavailable", wall.Freshness)
		}
		if len(wall.Notices) == 0 {
			t.Fatal("expected a notice explaining the unavailable state")
		}
		if wall.ActiveAgents != 0 || len(wall.Agents) != 0 {
			t.Fatal("unavailable wall must not invent agents")
		}
		if wall.PendingApprovals != 0 || len(wall.Approvals) != 0 {
			t.Fatal("unavailable wall must not invent approvals")
		}
	})

	t.Run("populated projection maps agents and approvals", func(t *testing.T) {
		proj := &OpsHiveProjection{
			GeneratedAt: now.Add(-2 * time.Second).Format(time.RFC3339),
			PendingApprovals: []OpsHiveApproval{
				{RequestID: "req_1", ActionName: "pull_request.create", Target: "transpara-ai/site", RiskSummary: "medium", CreatedAt: now.Format(time.RFC3339)},
			},
		}
		proj.RuntimeEvidence.AgentEvents.ObservedActive = 2
		proj.RuntimeEvidence.AgentEvents.ActiveAgents = []OpsHiveRuntimeAgent{
			{Name: "Strategist", Role: "strategist", Model: "opus-4-6"},
			{Name: "Implementer", Role: "implementer", Model: "gpt5.5"},
		}
		wall := buildConsoleHealthWall(proj, nil, now)
		if wall.Freshness != FreshnessCurrent {
			t.Fatalf("freshness = %q, want current", wall.Freshness)
		}
		if wall.ActiveAgents != 2 || len(wall.Agents) != 2 {
			t.Fatalf("agents = %d (active %d), want 2/2", len(wall.Agents), wall.ActiveAgents)
		}
		if wall.PendingApprovals != 1 || wall.Approvals[0].RequestID != "req_1" {
			t.Fatalf("approvals not mapped: %+v", wall.Approvals)
		}
		if wall.Agents[0].Model != "opus-4-6" {
			t.Fatalf("agent model = %q, want opus-4-6", wall.Agents[0].Model)
		}
	})

	t.Run("projection errors downgrade fresh data to partial", func(t *testing.T) {
		proj := &OpsHiveProjection{
			GeneratedAt: now.Add(-1 * time.Second).Format(time.RFC3339),
			Errors:      []string{"telemetry source degraded"},
		}
		wall := buildConsoleHealthWall(proj, nil, now)
		if wall.Freshness != FreshnessPartial {
			t.Fatalf("freshness = %q, want partial", wall.Freshness)
		}
	})
}

func TestConsoleHealthEmptyRosterIsPurposeful(t *testing.T) {
	// Fresh projection, zero agents — a genuinely usable-but-empty roster
	// (between civilization runs). The empty state must say why, not just
	// shrug with "reported."
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	proj := &OpsHiveProjection{GeneratedAt: now.Add(-2 * time.Second).Format(time.RFC3339)}
	wall := buildConsoleHealthWall(proj, nil, now)
	if wall.Freshness != FreshnessCurrent {
		t.Fatalf("freshness = %q, want current for a fresh, empty roster", wall.Freshness)
	}
	var buf bytes.Buffer
	if err := consoleHealthWall(wall).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"No active agents.", "civilization run"} {
		if !strings.Contains(out, want) {
			t.Errorf("empty health body missing %q; body: %s", want, out)
		}
	}
}

func TestHandleConsoleHealthFragment(t *testing.T) {
	h := newConsoleTestHandlers()
	t.Setenv("HIVE_OPS_API_BASE_URL", "")

	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/health/fragment", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "<html") {
		t.Fatal("fragment must not include the full page shell")
	}
	if !strings.Contains(body, "unavailable") {
		t.Fatal("fragment must render honest staleness")
	}
}

func TestDeriveFreshness(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	rfc := func(d time.Duration) string { return now.Add(d).Format(time.RFC3339) }

	tests := []struct {
		name             string
		generatedAt      string
		fetchErr         error
		hasPartialErrors bool
		want             ConsoleFreshness
	}{
		{"fetch error is unavailable", rfc(-1 * time.Second), errors.New("down"), false, FreshnessUnavailable},
		{"empty timestamp is unavailable", "", nil, false, FreshnessUnavailable},
		{"unparseable timestamp is unavailable", "not-a-time", nil, false, FreshnessUnavailable},
		{"older than window is stale", rfc(-90 * time.Second), nil, false, FreshnessStale},
		{"fresh with partial errors is partial", rfc(-2 * time.Second), nil, true, FreshnessPartial},
		{"fresh and clean is current", rfc(-2 * time.Second), nil, false, FreshnessCurrent},
		{"far-future timestamp is unavailable", rfc(90 * time.Second), nil, false, FreshnessUnavailable},
		{"slightly-future within skew is current", rfc(2 * time.Second), nil, false, FreshnessCurrent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveFreshness(tt.generatedAt, tt.fetchErr, tt.hasPartialErrors, now, consoleStaleWindow)
			if got != tt.want {
				t.Fatalf("deriveFreshness = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConsoleFreshnessBadgeCarriesStateDot(t *testing.T) {
	tests := []struct {
		name  string
		state ConsoleFreshness
		want  string
	}{
		{"current", FreshnessCurrent, `data-freshness="current"`},
		{"stale", FreshnessStale, `data-freshness="stale"`},
		{"partial", FreshnessPartial, `data-freshness="partial"`},
		{"unavailable", FreshnessUnavailable, `data-freshness="unavailable"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := consoleFreshnessBadge(tt.state, "2026-07-02T12:00:00Z").Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			out := buf.String()
			if !strings.Contains(out, tt.want) {
				t.Errorf("badge for %q missing %q; body: %s", tt.state, tt.want, out)
			}
			if tt.state == FreshnessCurrent && !strings.Contains(out, "live") {
				t.Errorf("current badge missing %q; body: %s", "live", out)
			}
		})
	}
}

func TestConsoleActiveTabIsAriaCurrent(t *testing.T) {
	t.Setenv("HIVE_OPS_API_BASE_URL", "")
	h := NewHandlers(nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterReadOnlyConsole(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()

	count := strings.Count(body, `aria-current="page"`)
	if count != 1 {
		t.Fatalf(`aria-current="page" appears %d times, want 1; body: %s`, count, body)
	}

	idx := strings.Index(body, `aria-current="page"`)
	if idx == -1 {
		t.Fatal(`aria-current="page" not found`)
	}
	// The nearest anchor start before aria-current must be the kanban tab's href.
	start := strings.LastIndex(body[:idx], "<a href=")
	if start == -1 {
		t.Fatal("no enclosing <a href= found before aria-current")
	}
	surrounding := body[start:idx]
	if !strings.Contains(surrounding, "/console/kanban") {
		t.Fatalf("aria-current is not within the kanban tab anchor; surrounding: %s", surrounding)
	}
}

func TestConsoleReadOnlyRoutesNoDB(t *testing.T) {
	t.Setenv("HIVE_OPS_API_BASE_URL", "")
	h := NewHandlers(nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterReadOnlyConsole(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unavailable") {
		t.Fatal("no-DB console must render explicit unavailable state")
	}
	if strings.Contains(w.Body.String(), `data-console-surface="civilization-mission-control"`) {
		t.Fatal("no-DB console exposed the authenticated Mission Control projection")
	}
	req = httptest.NewRequest(http.MethodGet, "http://site.test/console/mission-control/fragment", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("no-DB Mission Control fragment status = %d, want 404", w.Code)
	}
}
