package graph

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestConsoleIssueScanCardURLRoundTripsMetacharacters(t *testing.T) {
	// A projected run/stage id with query metacharacters must round-trip
	// through the drawer URL exactly, or clicking the card opens the wrong
	// (or not-found) drawer. This guards the query-escaping in the builder
	// against the handler's r.URL.Query().Get decode.
	card := OpsCivilizationIssueScanKanbanCard{
		RunID:   "run+a&b#c=d",
		StageID: "stage a/b&x",
	}
	u, err := url.Parse(consoleIssueScanCardURL(card))
	if err != nil {
		t.Fatalf("parse drawer url: %v", err)
	}
	q := u.Query()
	if got := q.Get("run"); got != card.RunID {
		t.Errorf("run round-trip = %q, want %q", got, card.RunID)
	}
	if got := q.Get("stage"); got != card.StageID {
		t.Errorf("stage round-trip = %q, want %q", got, card.StageID)
	}
}

func TestConsoleIntakeCardDrawerHiddenWhenSurfaceUnavailable(t *testing.T) {
	// A projection that passes validation and carries issue-scan records but has
	// no generated_at is FreshnessUnavailable — the board hides its cards. The
	// drawer endpoint must honor the same gate: a direct card request must NOT
	// leak run details, or honest-staleness is one HTMX call away from bypass.
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"projection_schema_version":"1.0.0","projection_subject":"civilization_assembly","derivation_status":"complete","issue_scan_projection":{"runs":[{"run_id":"run_x","target_issue":{"repo":"transpara-ai/site","number":1}}],"stages":[{"run_id":"run_x","stage_id":"stg_x","current_state":"parked","assigned_agent_ids":["secret_agent"]}]}}`)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake/card?run=run_x&stage=stg_x", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "not found") {
		t.Error("drawer must render honest not-found when the surface is unavailable")
	}
	if strings.Contains(body, "secret_agent") {
		t.Error("drawer leaked run details for an unavailable (timestamp-less) projection")
	}
}

func TestBuildConsoleIssueScanNilProjectionIsUnavailable(t *testing.T) {
	scan := buildConsoleIssueScan(nil, time.Now().UTC())
	if scan.Freshness != FreshnessUnavailable {
		t.Fatalf("freshness = %q, want unavailable", scan.Freshness)
	}
	if len(scan.Board.Columns) != 0 {
		t.Fatalf("nil projection must yield zero columns, got %d", len(scan.Board.Columns))
	}
	if len(scan.Notices) == 0 {
		t.Fatal("nil projection must carry an explicit notice")
	}
}

func TestBuildConsoleIssueScanFailedProjectionIsUnavailable(t *testing.T) {
	proj := &OpsCivilizationAssemblyProjection{
		DerivationStatus: opsCivilizationProjectionStatusFailed,
		GeneratedAt:      time.Now().UTC(), // failed sentinel carries a NON-zero timestamp
		FailureReasons:   []string{"hive civilization projection returned 503 Service Unavailable"},
	}
	scan := buildConsoleIssueScan(proj, time.Now().UTC())
	if scan.Freshness != FreshnessUnavailable {
		t.Fatalf("freshness = %q, want unavailable for failed status", scan.Freshness)
	}
	if len(scan.Notices) == 0 || !strings.Contains(scan.Notices[0], "503") {
		t.Fatalf("failed projection must surface its failure reason, got %v", scan.Notices)
	}
}

func TestBuildConsoleIssueScanZeroTimestampIsUnavailable(t *testing.T) {
	proj := &OpsCivilizationAssemblyProjection{DerivationStatus: "complete"} // GeneratedAt zero
	scan := buildConsoleIssueScan(proj, time.Now().UTC())
	if scan.Freshness != FreshnessUnavailable {
		t.Fatalf("freshness = %q, want unavailable for zero generated_at", scan.Freshness)
	}
}

func TestBuildConsoleIssueScanStaleTimestamp(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	proj := &OpsCivilizationAssemblyProjection{
		DerivationStatus: "complete",
		GeneratedAt:      now.Add(-2 * time.Minute), // older than consoleStaleWindow (30s)
	}
	scan := buildConsoleIssueScan(proj, now)
	if scan.Freshness != FreshnessStale {
		t.Fatalf("freshness = %q, want stale", scan.Freshness)
	}
}

func TestBuildConsoleIssueScanCurrentPassesBoardThrough(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	proj := &OpsCivilizationAssemblyProjection{
		DerivationStatus: "complete",
		GeneratedAt:      now.Add(-5 * time.Second),
		IssueScanProjection: OpsCivilizationIssueScanProjection{
			Runs: []OpsCivilizationIssueScanRunProjected{{
				RunID:       "run_x",
				TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 200, Title: "Do the thing"},
			}},
		},
	}
	scan := buildConsoleIssueScan(proj, now)
	if scan.Freshness != FreshnessCurrent {
		t.Fatalf("freshness = %q, want current", scan.Freshness)
	}
	if len(scan.Board.Columns) == 0 {
		t.Fatal("a projected run must produce at least one board column")
	}
}

func TestConsoleIntakeRendersIssueScanBoard(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hive/civilization/assembly-projection" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, hiveCivilizationAssemblyProjectionFixture)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	// Issue ref, a working agent, a stage, and a blocker from the fixture.
	for _, want := range []string{"transpara-ai/docs#172", "agent_reviewer", "run_adversarial_review", "duplicate_chain"} {
		if !strings.Contains(body, want) {
			t.Errorf("intake board missing %q", want)
		}
	}
	// This shared fixture has NO ready_for_human card (states are parked /
	// human_action only), so the "not merged" affordance is asserted in the
	// dedicated card-render tests below, not here.
}

func TestConsoleIssueScanCardStatesNoMergeWhenReady(t *testing.T) {
	card := OpsCivilizationIssueScanKanbanCard{
		RunID:        "run_ready",
		StageID:      "surface_ready_for_human_result_pr",
		CurrentState: "ready_for_human",
		TargetIssue:  OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 400},
	}
	var buf bytes.Buffer
	if err := consoleIssueScanCard(card).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), "not merged") {
		t.Error("a ready_for_human card must state the no-merge boundary")
	}
}

func TestConsoleIssueScanCardOmitsNoMergeWhenNotReady(t *testing.T) {
	card := OpsCivilizationIssueScanKanbanCard{RunID: "r", StageID: "s", CurrentState: "parked"}
	var buf bytes.Buffer
	if err := consoleIssueScanCard(card).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(buf.String(), "not merged") {
		t.Error("a non-ready card must not claim a no-merge boundary")
	}
}

func TestConsoleIntakeUnavailableWhenProjectionAbsent(t *testing.T) {
	t.Setenv("HIVE_OPS_API_BASE_URL", "") // no upstream configured -> nil projection

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "unavailable") {
		t.Error("absent projection must render an explicit unavailable state")
	}
	// No fabricated cards from the fixture may appear.
	if strings.Contains(body, "transpara-ai/docs#172") {
		t.Error("unavailable board must not fabricate issue-scan cards")
	}
}

func TestConsoleIntakeTabEnabled(t *testing.T) {
	t.Setenv("HIVE_OPS_API_BASE_URL", "")
	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()
	// The enabled Intake tab is an anchor to /console/intake, not a disabled span.
	if !strings.Contains(body, `href="/console/intake"`) {
		t.Error("Intake tab must be enabled (anchor to /console/intake)")
	}
}

func TestConsoleIntakeCardDrawerRendersPossessionAndLineage(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hive/civilization/assembly-projection" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, hiveCivilizationAssemblyProjectionFixture)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	// run_docs_172 / run_adversarial_review is a parked stage with agent_reviewer in the fixture.
	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake/card?run=run_docs_172&stage=run_adversarial_review", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"agent_reviewer", "run_adversarial_review", "Run details"} {
		if !strings.Contains(body, want) {
			t.Errorf("drawer missing %q", want)
		}
	}
}

func TestConsoleIntakeCardDrawerUnknownIsHonestNotFound(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, hiveCivilizationAssemblyProjectionFixture)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake/card?run=nope&stage=nope", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not found") {
		t.Error("unknown card must render an honest not-found drawer")
	}
}

// TestConsoleIntakeSurfaceEscapesHostileProjectionData is a characterization/
// regression guard for the Intake surface (Build 1 relocation of the
// issue-scan board to /console/intake). The old surface it replaced had
// exhaustive XSS/escaping tests; this closes the gap for the new one. templ's
// default `{ }` interpolation HTML-escapes everything, and this surface uses
// no templ.Raw/SafeHTML, so hostile operator-visible projection strings must
// come out escaped on both the board card and the run-details drawer.
func TestConsoleIntakeSurfaceEscapesHostileProjectionData(t *testing.T) {
	const runID = "run_hostile"
	const stageID = "stage_hostile"

	proj := &OpsCivilizationAssemblyProjection{
		DerivationStatus: "complete",
		GeneratedAt:      time.Now().UTC().Add(-5 * time.Second),
		IssueScanProjection: OpsCivilizationIssueScanProjection{
			Runs: []OpsCivilizationIssueScanRunProjected{{
				RunID: runID,
				TargetIssue: OpsCivilizationIssueRef{
					Repo:   "transpara-ai/x",
					Number: 900,
					Title:  "<script>alert('title')</script>",
				},
			}},
			Stages: []OpsCivilizationIssueScanStageProjected{{
				RunID:             runID,
				StageID:           stageID,
				CurrentState:      "parked",
				AuthorityBoundary: `<button onclick="x">auth</button>`,
				AssignedAgentIDs:  []string{"<img src=x onerror=y>agent"},
			}},
			Blockers: []OpsCivilizationIssueScanBlockerProjected{{
				RunID:          runID,
				StageID:        stageID,
				BlockerType:    `<form action="/hive">block</form>`,
				RequiredAction: `<form action="/hive">block</form>`,
			}},
			Lineage: []OpsCivilizationIssueScanLineageProjected{{
				RunID:         runID,
				StageID:       stageID,
				PrimaryTaskID: "<script>lineage()</script>",
			}},
		},
	}

	// Render the board.
	scan := buildConsoleIssueScan(proj, time.Now().UTC())
	var buf bytes.Buffer
	if err := consoleIssueScan(scan).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render board: %v", err)
	}

	// Render the drawer for the first card produced by the same projection.
	board := opsCivilizationIssueScanKanban(proj)
	if len(board.Columns) == 0 || len(board.Columns[0].Cards) == 0 {
		t.Fatal("hostile fixture produced zero cards; fixture is wrong, not the surface")
	}
	card := board.Columns[0].Cards[0]
	var buf2 bytes.Buffer
	if err := consoleIssueScanDrawer(card, true).Render(context.Background(), &buf2); err != nil {
		t.Fatalf("render drawer: %v", err)
	}

	boardOut := buf.String()
	drawerOut := buf2.String()
	combined := boardOut + drawerOut

	// Match the raw hostile payloads verbatim, not bare tag prefixes: the
	// surface's OWN chrome legitimately emits a real `<button ...>` element
	// (the clickable card) and this must not be confused with a leaked
	// hostile `<button onclick="x">`. Matching the exact injected string is
	// unambiguous — it only appears if the surface failed to escape it.
	rawHostile := []string{
		`<script>alert('title')</script>`,
		`<button onclick="x">auth</button>`,
		`<form action="/hive">block</form>`,
		"<img src=x onerror=y>agent",
		"<script>lineage()</script>",
	}
	for _, raw := range rawHostile {
		if strings.Contains(combined, raw) {
			t.Errorf("hostile raw markup %q survived escaping in the intake surface; board+drawer output leaked unescaped projection data", raw)
		}
	}

	if !strings.Contains(boardOut, "&lt;script") {
		t.Error("expected escaped form \"&lt;script\" in board output; escaping did not occur (data may have vanished instead of being escaped)")
	}
}
