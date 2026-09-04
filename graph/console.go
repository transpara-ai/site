package graph

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/transpara-ai/site/profile"
)

type ConsolePageData struct {
	Title          string
	Active         string // mission-control | health | kanban | intake | config
	MissionControl *MissionControlView
	Health         *ConsoleHealthWall
	Kanban         *ConsoleKanban
	IssueScan      *ConsoleIssueScan
	Config         *ConsoleConfig
	Workbench      *CivilizationWorkbench
}

func (h *Handlers) renderConsole(w http.ResponseWriter, r *http.Request, data ConsolePageData) {
	ConsolePage(data, h.viewUser(r), profile.FromContext(r.Context())).Render(r.Context(), w)
}

type ConsoleFreshness string

const (
	FreshnessCurrent     ConsoleFreshness = "current"
	FreshnessStale       ConsoleFreshness = "stale"
	FreshnessPartial     ConsoleFreshness = "partial"
	FreshnessUnavailable ConsoleFreshness = "unavailable"
)

const consoleStaleWindow = 30 * time.Second
const consoleFutureSkew = 5 * time.Second

type ConsoleHealthWall struct {
	Freshness        ConsoleFreshness
	GeneratedAt      string
	ActiveAgents     int
	Agents           []ConsoleAgentRow
	PendingApprovals int
	Approvals        []ConsoleApproval
	Notices          []string
}

type ConsoleAgentRow struct {
	Name  string
	Role  string
	Model string
}

type ConsoleApproval struct {
	RequestID string
	Action    string
	Target    string
	Risk      string
	CreatedAt string
}

// buildConsoleHealthWall maps the hive operator projection (or a fetch error)
// into the Health Wall view-model. On any fetch failure it returns an
// unavailable wall with a human-readable notice and no invented agents.
func buildConsoleHealthWall(proj *OpsHiveProjection, fetchErr error, now time.Time) ConsoleHealthWall {
	if fetchErr != nil || proj == nil {
		reason := "operator projection unavailable"
		if fetchErr != nil {
			reason = fetchErr.Error()
		}
		return ConsoleHealthWall{Freshness: FreshnessUnavailable, Notices: []string{reason}}
	}

	wall := ConsoleHealthWall{
		Freshness:    deriveFreshness(proj.GeneratedAt, nil, len(proj.Errors) > 0, now, consoleStaleWindow),
		GeneratedAt:  proj.GeneratedAt,
		ActiveAgents: proj.RuntimeEvidence.AgentEvents.ObservedActive,
		Notices:      append([]string(nil), proj.Errors...),
	}
	for _, a := range proj.RuntimeEvidence.AgentEvents.ActiveAgents {
		wall.Agents = append(wall.Agents, ConsoleAgentRow{Name: a.Name, Role: a.Role, Model: a.Model})
	}
	for _, ap := range proj.PendingApprovals {
		wall.Approvals = append(wall.Approvals, ConsoleApproval{
			RequestID: ap.RequestID,
			Action:    ap.ActionName,
			Target:    ap.Target,
			Risk:      ap.RiskSummary,
			CreatedAt: ap.CreatedAt,
		})
	}
	wall.PendingApprovals = len(wall.Approvals)
	return wall
}

// ConsoleIssueScan is the Intake-tab view-model: the reused issue-scan kanban
// board plus an explicit freshness state derived from the civilization
// projection. It fails closed — a nil, failed, or timestamp-less projection is
// rendered as unavailable with a human-readable notice and no invented cards.
type ConsoleIssueScan struct {
	Freshness     ConsoleFreshness
	GeneratedAt   string
	Status        string
	Summary       string
	Board         OpsCivilizationIssueScanKanban
	RecentRuns    ConsoleRecentRuns
	SourceMarkers ConsoleSourceMarkers
	Notices       []string
}

// ConsoleRecentRuns is the recent-intakes rail view-model derived from the
// hive recent_issue_scan_runs projection section (schema 1.6.0). Available is
// false — never populated with entries — unless BOTH the surface freshness is
// usable (current/stale/partial, the board's exact set) AND the section's own
// status is "available"; this keeps a 1.5.0 payload (no field at all) and any
// unknown/unavailable section status byte-identical to "no rail".
type ConsoleRecentRuns struct {
	Available bool
	Truncated bool
	Entries   []ConsoleRecentRunEntry
}

// ConsoleRecentRunEntry is one rail chip. StyleKind is allowlist-derived
// ("amber" only for parked/human_action; "neutral" for every other value,
// known or not — never a default that yields amber). DrawerURL is only ever
// set when Linked is true (the (RunID, StageID) pair exists on the rendered
// board), via consoleIssueScanCardURL mechanics, so the rail never fabricates
// a drawer target.
type ConsoleRecentRunEntry struct {
	RunID      string
	StageID    string
	IssueLabel string
	IssueURL   string
	State      string
	StyleKind  string
	Age        string
	DrawerURL  string
	Linked     bool
}

// consoleRecentRunAmberStates is the allowlist of states that render amber.
// Every other state value (queued, in_flight, recorded, and any unknown/
// future value including a hypothetical ready state) renders neutral — no
// denylist, no default that yields a healthy or alarming color by accident.
var consoleRecentRunAmberStates = map[string]bool{
	"parked":       true,
	"human_action": true,
}

// consoleRecentRunUsableFreshness is the EXACT surface-freshness set that
// renders the board (buildConsoleIssueScan's own success path): current,
// stale, partial. This is intentionally the same decision as the board's, so
// the rail and board can never diverge on when they render.
var consoleRecentRunUsableFreshness = map[ConsoleFreshness]bool{
	FreshnessCurrent: true,
	FreshnessStale:   true,
	FreshnessPartial: true,
}

// buildConsoleRecentRuns derives the recent-intakes rail view-model. It is a
// pure function: no I/O, no time.Now() (now is passed in), fail-closed at
// every gate.
//
// Gates, both required:
//  1. freshness must be in consoleRecentRunUsableFreshness (the board's own
//     usable set) — an unavailable/unknown surface never renders a rail even
//     if the section itself claims to be available.
//  2. proj.RecentIssueScanRuns.Status must be exactly "available" — absent
//     field (1.5.0 payload, decodes to the zero value ""), "unavailable", or
//     any unrecognized value all deny.
//
// Linked is true only when the (RunID, StageID) pair is present on the
// rendered board (site-side index built from the same board passed in), so a
// rail entry can never link to a drawer the board itself doesn't expose.
func buildConsoleRecentRuns(proj *OpsCivilizationAssemblyProjection, board OpsCivilizationIssueScanKanban, freshness ConsoleFreshness, now time.Time) ConsoleRecentRuns {
	if proj == nil {
		return ConsoleRecentRuns{}
	}
	if !consoleRecentRunUsableFreshness[freshness] {
		return ConsoleRecentRuns{}
	}
	section := proj.RecentIssueScanRuns
	if section.Status != opsCivilizationFieldAvailable {
		return ConsoleRecentRuns{}
	}

	onBoard := map[string]bool{}
	for _, col := range board.Columns {
		for _, card := range col.Cards {
			onBoard[runStageKey(card.RunID, card.StageID)] = true
		}
	}

	entries := make([]ConsoleRecentRunEntry, 0, len(section.Runs))
	for _, run := range section.Runs {
		linked := onBoard[runStageKey(run.RunID, run.StageID)]
		entry := ConsoleRecentRunEntry{
			RunID:      run.RunID,
			StageID:    run.StageID,
			IssueLabel: opsCivilizationIssueRefLabel(OpsCivilizationIssueRef{Repo: run.Repo, Number: run.IssueNumber, URL: run.IssueURL, Title: run.IssueTitle}),
			IssueURL:   run.IssueURL,
			State:      run.State,
			StyleKind:  consoleRecentRunStyleKind(run.State),
			Age:        consoleRecentRunAge(now, run.LastEventAt),
			Linked:     linked,
		}
		if linked {
			entry.DrawerURL = consoleIssueScanCardURL(OpsCivilizationIssueScanKanbanCard{RunID: run.RunID, StageID: run.StageID})
		}
		entries = append(entries, entry)
	}

	return ConsoleRecentRuns{
		Available: true,
		Truncated: section.Truncated,
		Entries:   entries,
	}
}

// consoleRecentRunStyleKind is a pure allowlist: amber ONLY for parked and
// human_action; every other value — including queued/in_flight/recorded and
// any state not in the v1 set — is neutral. No default branch produces amber.
func consoleRecentRunStyleKind(state string) string {
	if consoleRecentRunAmberStates[state] {
		return "amber"
	}
	return "neutral"
}

// consoleRecentRunAge reuses humanizeAge (graph/console_kanban.go) so the
// rail's relative-age formatting matches the rest of the console. Unparseable
// or absent timestamps yield "" (never "now") per humanizeAge's own contract.
func consoleRecentRunAge(now time.Time, lastEventAt string) string {
	return humanizeAge(now, lastEventAt)
}

// buildConsoleIssueScan maps the civilization assembly projection (or its
// absence/failure) into the Intake view-model. It reuses the existing
// opsCivilizationIssueScanKanban builder verbatim and derives honest staleness.
//
// Fail-closed by allowlist: only the explicitly-usable derivation statuses
// (complete, partial) render as data. Every other value — failed, unavailable,
// unknown, empty, or any status added to the enum later — resolves to
// FreshnessUnavailable rather than falling through to usable/current data. A
// denylist here would silently pass every status we forgot; the default denies.
func buildConsoleIssueScan(proj *OpsCivilizationAssemblyProjection, now time.Time) ConsoleIssueScan {
	board := opsCivilizationIssueScanKanban(proj) // nil-safe: returns an unavailable board
	unavailable := func(notices []string) ConsoleIssueScan {
		return ConsoleIssueScan{
			Freshness: FreshnessUnavailable,
			Status:    board.Status,
			Summary:   board.Summary,
			Board:     board,
			Notices:   notices,
		}
	}
	if proj == nil {
		return unavailable([]string{"civilization projection unavailable to Site"})
	}
	status := strings.ToLower(strings.TrimSpace(proj.DerivationStatus))
	switch status {
	case opsCivilizationProjectionStatusComplete, opsCivilizationProjectionStatusPartial:
		// Usable — fall through to timestamp + freshness derivation below.
	case opsCivilizationProjectionStatusFailed:
		return unavailable(append([]string(nil), proj.FailureReasons...))
	default:
		// unavailable, unknown, empty, or any future enum value: deny.
		notices := append([]string(nil), proj.FailureReasons...)
		if len(notices) == 0 {
			label := status
			if label == "" {
				label = "missing"
			}
			notices = []string{"projection derivation status not usable: " + label}
		}
		return unavailable(notices)
	}
	if proj.GeneratedAt.IsZero() {
		return unavailable([]string{"projection missing generated_at timestamp"})
	}
	generatedAt := proj.GeneratedAt.UTC().Format(time.RFC3339)
	hasPartial := status == opsCivilizationProjectionStatusPartial
	freshness := deriveFreshness(generatedAt, nil, hasPartial, now, consoleStaleWindow)
	// Unblock hints/commands are gated to a verified-current projection by
	// allowlist: only freshness == current stamps UnblockAvailable. Stale and
	// partial still render cards, blockers, and required actions honestly —
	// they just never offer commands derived from a label snapshot that may
	// no longer match reality (partial is a degraded source set, not a
	// verified-current one). The default (any other freshness) leaves every
	// card's UnblockAvailable at its zero value (false).
	if freshness == FreshnessCurrent {
		runBlockerTypes := consoleIssueScanRunBlockerTypes(board)
		for ci := range board.Columns {
			for i := range board.Columns[ci].Cards {
				_, ok := consoleIssueScanUnblockPlan(board.Columns[ci].Cards[i], runBlockerTypes)
				board.Columns[ci].Cards[i].UnblockAvailable = ok
			}
		}
	}
	return ConsoleIssueScan{
		Freshness:     freshness,
		GeneratedAt:   generatedAt,
		Status:        board.Status,
		Summary:       board.Summary,
		Board:         board,
		RecentRuns:    buildConsoleRecentRuns(proj, board, freshness, now),
		SourceMarkers: buildConsoleSourceMarkers(proj, freshness, now),
	}
}

func (h *Handlers) handleConsoleHealth(w http.ResponseWriter, r *http.Request) {
	proj, err := fetchHiveOperatorProjection(r)
	wall := buildConsoleHealthWall(proj, err, time.Now().UTC())
	h.renderConsole(w, r, ConsolePageData{Title: "Health wall", Active: "health", Health: &wall})
}

func (h *Handlers) handleConsoleHealthFragment(w http.ResponseWriter, r *http.Request) {
	proj, err := fetchHiveOperatorProjection(r)
	wall := buildConsoleHealthWall(proj, err, time.Now().UTC())
	consoleHealthWallFragment(wall).Render(r.Context(), w)
}

func (h *Handlers) handleConsoleKanban(w http.ResponseWriter, r *http.Request) {
	lens := parseLens(r.URL.Query().Get("lens"))
	res := fetchConsoleWork(r)
	k := buildConsoleKanban(res.Tasks, res.Err, lens, time.Now().UTC())
	h.renderConsole(w, r, ConsolePageData{Title: "Kanban", Active: "kanban", Kanban: &k})
}

func (h *Handlers) handleConsoleKanbanFragment(w http.ResponseWriter, r *http.Request) {
	lens := parseLens(r.URL.Query().Get("lens"))
	res := fetchConsoleWork(r)
	k := buildConsoleKanban(res.Tasks, res.Err, lens, time.Now().UTC())
	consoleKanbanFragment(k).Render(r.Context(), w)
}

func (h *Handlers) handleConsoleIntake(w http.ResponseWriter, r *http.Request) {
	proj := fetchOpsCivilizationProjection(r)
	scan := buildConsoleIssueScan(proj, time.Now().UTC())
	h.renderConsole(w, r, ConsolePageData{Title: "Intake", Active: "intake", IssueScan: &scan})
}

func (h *Handlers) handleConsoleIntakeFragment(w http.ResponseWriter, r *http.Request) {
	proj := fetchOpsCivilizationProjection(r)
	scan := buildConsoleIssueScan(proj, time.Now().UTC())
	consoleIssueScanFragment(scan).Render(r.Context(), w)
}

func (h *Handlers) handleConsoleIntakeCard(w http.ResponseWriter, r *http.Request) {
	run := strings.TrimSpace(r.URL.Query().Get("run"))
	stage := strings.TrimSpace(r.URL.Query().Get("stage"))
	// Two-tier honest-staleness gate, both fail-closed against the SAME
	// freshness computation as the board (buildConsoleIssueScan):
	//   1. Finding the card at all: gated to != FreshnessUnavailable. A failed
	//      / timestamp-less projection can still carry records; without this
	//      gate a direct card request would expose run details the board
	//      intentionally hides. Stale/partial still show the run honestly —
	//      hiding a real run because its timestamp is old would itself be
	//      dishonest.
	//   2. Deriving the unblock plan: gated to == FreshnessCurrent only. A
	//      stale or partial projection's label snapshot may no longer match
	//      reality, so the exact gh issue edit commands it would produce are
	//      not trustworthy — only a verified-current projection may offer
	//      commands. Stale/partial drawers still render the projected
	//      required action, just never a command.
	// When the surface is unavailable, the loop is skipped and the honest
	// not-found drawer renders.
	scan := buildConsoleIssueScan(fetchOpsCivilizationProjection(r), time.Now().UTC())
	if scan.Freshness != FreshnessUnavailable {
		var runBlockerTypes map[string][]string
		if scan.Freshness == FreshnessCurrent {
			runBlockerTypes = consoleIssueScanRunBlockerTypes(scan.Board)
		}
		for _, col := range scan.Board.Columns {
			for _, card := range col.Cards {
				if card.RunID == run && card.StageID == stage {
					plan, planOK := consoleUnblockPlan{}, false
					if scan.Freshness == FreshnessCurrent {
						plan, planOK = consoleIssueScanUnblockPlan(card, runBlockerTypes)
					}
					consoleIssueScanDrawer(card, true, plan, planOK).Render(r.Context(), w)
					return
				}
			}
		}
	}
	// Not found, upstream error, or unavailable surface: honest empty drawer, never a fabricated run.
	consoleIssueScanDrawer(OpsCivilizationIssueScanKanbanCard{RunID: run, StageID: stage}, false, consoleUnblockPlan{}, false).Render(r.Context(), w)
}

// consoleIssueScanCardIssue renders the leading issue reference (repo#number,
// falling back to URL/title) for an issue-scan card, preferring the target
// issue and falling back to the selected candidate.
func consoleIssueScanCardIssue(card OpsCivilizationIssueScanKanbanCard) string {
	ref := card.TargetIssue
	if ref.Repo == "" && ref.Number == 0 {
		ref = card.SelectedIssue
	}
	return opsCivilizationIssueRefLabel(ref)
}

// consoleIssueScanCardIssueURL returns the projected issue URL for the same ref
// consoleIssueScanCardIssue labels (target preferred, then selected), or "" when
// none is projected. The template sanitizes it via templ.URL before use.
func consoleIssueScanCardIssueURL(card OpsCivilizationIssueScanKanbanCard) string {
	ref := card.TargetIssue
	if ref.Repo == "" && ref.Number == 0 {
		ref = card.SelectedIssue
	}
	return strings.TrimSpace(ref.URL)
}

func consoleIssueScanCardTitle(card OpsCivilizationIssueScanKanbanCard) string {
	if card.TargetIssue.Title != "" {
		return card.TargetIssue.Title
	}
	return card.SelectedIssue.Title
}

// consoleIssueScanCardAgents lists the possessing agents — assigned first, then
// any touching-only agents, deduplicated — or "unassigned" when the projection
// names none. Both fields are surfaced so a stage worked by a touching-only
// agent (e.g. blocker repair) is not hidden behind its assignee. Never invented.
func consoleIssueScanCardAgents(card OpsCivilizationIssueScanKanbanCard) string {
	seen := map[string]bool{}
	agents := make([]string, 0, len(card.AssignedAgentIDs)+len(card.TouchingAgentIDs))
	for _, id := range card.AssignedAgentIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		agents = append(agents, id)
	}
	for _, id := range card.TouchingAgentIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		agents = append(agents, id)
	}
	if len(agents) == 0 {
		return "unassigned"
	}
	return strings.Join(agents, ", ")
}

func consoleIssueScanCardBlocker(card OpsCivilizationIssueScanKanbanCard) string {
	if len(card.Blockers) == 0 {
		return ""
	}
	b := card.Blockers[0]
	if b.RequiredAction != "" {
		return b.BlockerType + " — " + b.RequiredAction
	}
	return b.BlockerType
}

// consoleIssueScanCardReady reports whether the card is in the terminal
// ready-for-human state, so the board can state the no-merge boundary honestly.
func consoleIssueScanCardReady(card OpsCivilizationIssueScanKanbanCard) bool {
	return strings.EqualFold(strings.TrimSpace(card.CurrentState), "ready_for_human")
}

// consoleIssueScanCardURL builds the drawer hx-get URL for a card. RunID and
// StageID are query-escaped so a projected id containing query metacharacters
// (&, #, +, =) round-trips exactly through r.URL.Query().Get in the handler —
// attribute escaping alone would prevent HTML injection but not this parameter
// corruption, which would open the wrong (or not-found) drawer for a real card.
func consoleIssueScanCardURL(card OpsCivilizationIssueScanKanbanCard) string {
	return "/console/intake/card?run=" + url.QueryEscape(card.RunID) + "&stage=" + url.QueryEscape(card.StageID)
}

func (h *Handlers) handleConsoleKanbanOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res := fetchConsoleWork(r)
	now := time.Now().UTC()
	if res.Err == nil {
		for _, t := range res.Tasks {
			if t.ID == id {
				consoleOrderDrawer(cardForTask(t, now), true).Render(r.Context(), w)
				return
			}
		}
	}
	// Not found or upstream error: render an honest empty drawer, never a fabricated order.
	consoleOrderDrawer(ConsoleOrderCard{ID: id}, false).Render(r.Context(), w)
}

// deriveFreshness maps upstream signals onto an explicit freshness state.
// It fails closed: a fetch error, an empty or unparseable timestamp, or any
// other ambiguity resolves to FreshnessUnavailable. Only a parseable,
// within-window, error-free projection earns FreshnessCurrent.
func deriveFreshness(generatedAt string, fetchErr error, hasPartialErrors bool, now time.Time, staleWindow time.Duration) ConsoleFreshness {
	if fetchErr != nil {
		return FreshnessUnavailable
	}
	ts, err := time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		return FreshnessUnavailable
	}
	age := now.Sub(ts)
	if age < -consoleFutureSkew {
		return FreshnessUnavailable
	}
	if age > staleWindow {
		return FreshnessStale
	}
	if hasPartialErrors {
		return FreshnessPartial
	}
	return FreshnessCurrent
}
