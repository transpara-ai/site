package graph

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// consoleWorkResult is the focused work-tasks fetch the Kanban consumes. Unlike
// fetchOpsWork (which caps to a 10-task /ops summary), this returns the full
// task set from the configured legacy Work or Civilization API. A successful
// live query is current as of GeneratedAt; an error yields zero cards.
type consoleWorkResult struct {
	GeneratedAt  string
	Tasks        []OpsWorkTask
	Cards        []ConsoleOrderCard
	Civilization bool
	Err          error
}

func fetchConsoleWork(r *http.Request) consoleWorkResult {
	if consoleUsesCivilizationWork() {
		return fetchConsoleCivilizationWork(r)
	}
	base := serverWorkAPIBaseURL()
	tasksURL := legacyWorkURL(base, "/tasks")
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, tasksURL, nil)
	if err != nil {
		return consoleWorkResult{Err: err}
	}
	setWorkAuth(req)
	resp, err := obsWorkClient.Do(req)
	if err != nil {
		return consoleWorkResult{Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return consoleWorkResult{Err: fmt.Errorf("work tasks returned %s", resp.Status)}
	}
	var payload opsWorkTasksResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return consoleWorkResult{Err: err}
	}
	return consoleWorkResult{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Tasks:       payload.Tasks,
	}
}

func fetchConsoleCivilizationWork(r *http.Request) consoleWorkResult {
	result := consoleWorkResult{Civilization: true}
	client, err := newCivilizationClient(8 * time.Second)
	if err != nil {
		result.Err = err
		return result
	}
	// Require the work-list envelope: a successful response with the wrong
	// schema must not masquerade as an empty board.
	var payload struct {
		Items json.RawMessage `json:"items"`
	}
	if err := client.request(r.Context(), http.MethodGet, "/api/civilization/v1/work", nil, &payload); err != nil {
		result.Err = err
		return result
	}
	var items []CivilizationWork
	if json.Unmarshal(payload.Items, &items) != nil || items == nil {
		result.Err = fmt.Errorf("Civilization work list is invalid")
		return result
	}
	seen := map[string]bool{}
	for _, item := range items {
		if strings.TrimSpace(item.WorkID) == "" || strings.TrimSpace(item.State) == "" || seen[item.WorkID] {
			result.Err = fmt.Errorf("Civilization work list is invalid")
			return result
		}
		seen[item.WorkID] = true
	}
	now := time.Now().UTC()
	for _, item := range items {
		result.Cards = append(result.Cards, cardForCivilizationWork(item, now))
	}
	result.GeneratedAt = now.Format(time.RFC3339)
	return result
}

func (r consoleWorkResult) board(lens ConsoleKanbanLens, now time.Time) ConsoleKanban {
	if !r.Civilization {
		return buildConsoleKanban(r.Tasks, r.Err, lens, now)
	}
	k := buildConsoleKanbanCards(r.Cards, r.Err, lens, now)
	k.Civilization = true
	if lens == LensStatus {
		sort.SliceStable(k.Columns, func(i, j int) bool {
			return rankLess(k.Columns[i].Key, k.Columns[j].Key, civilizationStatusRank, "unknown")
		})
		for i := range k.Columns {
			k.Columns[i].Label = civilizationStateLabel(k.Columns[i].Key)
		}
	}
	return k
}

func consoleRequestLens(r *http.Request) ConsoleKanbanLens {
	if strings.TrimSpace(r.URL.Query().Get("lens")) == "" && consoleUsesCivilizationWork() {
		return LensStatus
	}
	return parseLens(r.URL.Query().Get("lens"))
}

type ConsoleKanbanLens string

const (
	LensRisk   ConsoleKanbanLens = "risk"
	LensStatus ConsoleKanbanLens = "status"
	LensAgent  ConsoleKanbanLens = "agent"
	LensSource ConsoleKanbanLens = "source"
)

// parseLens resolves the ?lens= query value, defaulting to risk (the design
// default) for empty or unrecognized input.
func parseLens(raw string) ConsoleKanbanLens {
	switch ConsoleKanbanLens(strings.ToLower(strings.TrimSpace(raw))) {
	case LensStatus:
		return LensStatus
	case LensAgent:
		return LensAgent
	case LensSource:
		return LensSource
	default:
		return LensRisk
	}
}

type ConsoleOrderCard struct {
	ID             string
	Title          string
	FactoryOrderID string
	Submitter      string
	Status         string
	Agent          string
	Risk           string
	Cell           string
	CreatedAt      string
	AgeLabel       string
	Civilization   bool
	Repository     string
	Provider       string
	HumanOwner     string
	HumanStep      string
	Blocker        string
	UpdatedAt      string
	UpdatedLabel   string
}

func (c ConsoleOrderCard) DrawerURL() string {
	return "/console/kanban/order/" + url.PathEscape(c.ID)
}

func (c ConsoleOrderCard) sortTime() string {
	if c.Civilization {
		return c.UpdatedAt
	}
	return c.CreatedAt
}

func cardForCivilizationWork(work CivilizationWork, now time.Time) ConsoleOrderCard {
	title := work.IntakeText
	if work.Bound != nil && strings.TrimSpace(work.Bound.Envelope.Brief.Outcome) != "" {
		title = work.Bound.Envelope.Brief.Outcome
	}
	data := CivilizationWorkbench{}
	card := ConsoleOrderCard{
		Civilization: true, ID: work.WorkID, Title: title, Status: work.State,
		Submitter: data.Requester(work), Repository: work.Source.Repository,
		Provider: civilizationExecutionLabel(work), HumanOwner: data.HumanOwner(work),
		HumanStep: civilizationHumanStep(work), Blocker: work.Blocker,
	}
	// This feed has no actor assignment, risk class, or creation time. Keep
	// those absent; a provider is not an agent and update age is not total age.
	if !work.UpdatedAt.IsZero() {
		card.UpdatedAt = work.UpdatedAt.Format(time.RFC3339)
		card.UpdatedLabel = humanizeAge(now, card.UpdatedAt)
	}
	return card
}

type ConsoleKanbanColumn struct {
	Key   string
	Label string
	Cards []ConsoleOrderCard
}

type ConsoleKanban struct {
	Civilization bool
	Freshness    ConsoleFreshness
	GeneratedAt  string
	Lens         ConsoleKanbanLens
	Columns      []ConsoleKanbanColumn
	TotalCards   int
	Notices      []string
}

// riskRank orders the known risk classes by severity (highest first). Unknown
// values sort after all known ones; the empty/unclassified key sorts last.
var riskRank = map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}

// statusRank orders the v3.9 lifecycle. Unknown statuses sort after known ones;
// the empty/unknown key sorts last.
var statusRank = map[string]int{
	"created": 0, "ready": 1, "running": 2, "blocked": 3, "failed": 4,
	"repair_required": 5, "repair_running": 6, "repaired": 7,
	"verification_running": 8, "verified": 9, "certified": 10,
	"rejected": 11, "superseded": 12, "policy_blocked": 13,
}

var civilizationStatusRank = map[string]int{
	"awaiting_confirmation": 0, "human_required": 1, "blocked": 2,
	"routing": 3, "queued": 4, "implementing": 5, "validating": 6,
	"reviewing": 7, "prepared": 8, "publishing": 9, "ready": 10,
	"merge_queued": 11, "completed": 12, "approved": 13, "rejected": 14, "changes_requested": 15,
}

func humanizeAge(now time.Time, createdAt string) string {
	if strings.TrimSpace(createdAt) == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return ""
	}
	d := now.Sub(t)
	if d < 0 {
		return "" // future timestamp — fail closed, no fabricated age
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func cardForTask(t OpsWorkTask, now time.Time) ConsoleOrderCard {
	return ConsoleOrderCard{
		ID:             t.ID,
		Title:          t.Title,
		FactoryOrderID: t.FactoryOrderID,
		Submitter:      t.CreatedBy,
		Status:         t.Status,
		Agent:          t.Assignee,
		Risk:           t.RiskClass,
		Cell:           t.Cell,
		CreatedAt:      t.CreatedAt,
		AgeLabel:       humanizeAge(now, t.CreatedAt),
	}
}

// lensKey returns the grouping key and the column label for a card under a lens.
// An empty raw key maps to an explicit fallback so the card stays visible.
func lensKey(card ConsoleOrderCard, lens ConsoleKanbanLens) (key, label string) {
	switch lens {
	case LensStatus:
		if card.Status == "" {
			return "unknown", "unknown"
		}
		return card.Status, card.Status
	case LensAgent:
		if card.Agent == "" {
			return "unassigned", "unassigned"
		}
		return card.Agent, card.Agent
	case LensSource:
		if card.Submitter == "" {
			return "unknown", "unknown"
		}
		return card.Submitter, card.Submitter
	default: // LensRisk
		if card.Risk == "" {
			return "unclassified", "unclassified"
		}
		return card.Risk, card.Risk
	}
}

// columnLess orders two column keys under a lens. Ranked vocabularies
// (risk, status) use their rank maps; unknown values sort after known ones;
// the empty-fallback key always sorts last. Agent/source sort alphabetically
// with the fallback key last.
func columnLess(lens ConsoleKanbanLens, a, b string) bool {
	switch lens {
	case LensRisk:
		return rankLess(a, b, riskRank, "unclassified")
	case LensStatus:
		return rankLess(a, b, statusRank, "unknown")
	case LensAgent:
		return fallbackLast(a, b, "unassigned")
	default: // LensSource
		return fallbackLast(a, b, "unknown")
	}
}

func rankLess(a, b string, rank map[string]int, fallback string) bool {
	if a == fallback || b == fallback {
		return b == fallback && a != fallback
	}
	ra, oka := rank[a]
	rb, okb := rank[b]
	if oka && okb {
		return ra < rb
	}
	if oka != okb {
		return oka // known sorts before unknown
	}
	return a < b
}

func fallbackLast(a, b, fallback string) bool {
	if a == fallback || b == fallback {
		return b == fallback && a != fallback
	}
	return a < b
}

// cardTitle returns the human title, or an explicit placeholder when absent —
// never a fabricated title.
func cardTitle(card ConsoleOrderCard) string {
	if strings.TrimSpace(card.Title) == "" {
		return "(untitled)"
	}
	return card.Title
}

// cardTag is the compact identity chip: the factory order id when linked,
// otherwise the task id.
func cardTag(card ConsoleOrderCard) string {
	if strings.TrimSpace(card.FactoryOrderID) != "" {
		return card.FactoryOrderID
	}
	return card.ID
}

func orFallback(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func columnCount(col ConsoleKanbanColumn) string {
	return fmt.Sprintf("%d", len(col.Cards))
}

func noticeText(notices []string) string {
	if len(notices) == 0 {
		return "no upstream data"
	}
	return strings.Join(notices, "; ")
}

func buildConsoleKanban(tasks []OpsWorkTask, fetchErr error, lens ConsoleKanbanLens, now time.Time) ConsoleKanban {
	cards := make([]ConsoleOrderCard, 0, len(tasks))
	for _, task := range tasks {
		cards = append(cards, cardForTask(task, now))
	}
	return buildConsoleKanbanCards(cards, fetchErr, lens, now)
}

func buildConsoleKanbanCards(cards []ConsoleOrderCard, fetchErr error, lens ConsoleKanbanLens, now time.Time) ConsoleKanban {
	freshness := deriveFreshness(now.Format(time.RFC3339), fetchErr, false, now, consoleStaleWindow)
	k := ConsoleKanban{
		Freshness:   freshness,
		GeneratedAt: now.Format(time.RFC3339),
		Lens:        lens,
	}
	if fetchErr != nil {
		k.Notices = []string{fetchErr.Error()}
		return k // unavailable: zero cards, never fabricated
	}

	byKey := map[string]*ConsoleKanbanColumn{}
	var order []string
	for _, card := range cards {
		key, label := lensKey(card, lens)
		col, ok := byKey[key]
		if !ok {
			col = &ConsoleKanbanColumn{Key: key, Label: label}
			byKey[key] = col
			order = append(order, key)
		}
		col.Cards = append(col.Cards, card)
	}
	sort.SliceStable(order, func(i, j int) bool { return columnLess(lens, order[i], order[j]) })
	for _, key := range order {
		col := byKey[key]
		// Within a column, oldest-first surfaces the most-aging order at the top.
		sort.SliceStable(col.Cards, func(i, j int) bool {
			ti, oki := parseCardTime(col.Cards[i].sortTime())
			tj, okj := parseCardTime(col.Cards[j].sortTime())
			if oki && okj {
				return ti.Before(tj) // oldest-first among dated cards
			}
			if oki != okj {
				return oki // a dated card sorts before an undated/invalid one
			}
			return false // both undated/invalid: stable order
		})
		k.Columns = append(k.Columns, *col)
	}
	k.TotalCards = len(cards)
	return k
}

// parseCardTime parses a card's CreatedAt; ok is false for empty/unparseable
// values (the same cases humanizeAge hides), so the sort can place them last.
func parseCardTime(createdAt string) (time.Time, bool) {
	if strings.TrimSpace(createdAt) == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
