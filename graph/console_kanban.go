package graph

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// consoleWorkResult is the focused work-tasks fetch the Kanban consumes. Unlike
// fetchOpsWork (which caps to a 10-task /ops summary), this returns the full
// task set so the Kanban can group every order. /tasks is a live query, so a
// successful fetch IS current as of GeneratedAt; an error yields zero tasks.
type consoleWorkResult struct {
	GeneratedAt string
	Tasks       []OpsWorkTask
	Err         error
}

func fetchConsoleWork(r *http.Request) consoleWorkResult {
	base := serverWorkAPIBaseURL()
	url := legacyWorkURL(base, "/tasks")
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
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
}

type ConsoleKanbanColumn struct {
	Key   string
	Label string
	Cards []ConsoleOrderCard
}

type ConsoleKanban struct {
	Freshness   ConsoleFreshness
	GeneratedAt string
	Lens        ConsoleKanbanLens
	Columns     []ConsoleKanbanColumn
	TotalCards  int
	Notices     []string
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

func buildConsoleKanban(tasks []OpsWorkTask, fetchErr error, lens ConsoleKanbanLens, now time.Time) ConsoleKanban {
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
	for _, t := range tasks {
		card := cardForTask(t, now)
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
			return col.Cards[i].CreatedAt < col.Cards[j].CreatedAt
		})
		k.Columns = append(k.Columns, *col)
	}
	k.TotalCards = len(tasks)
	return k
}
