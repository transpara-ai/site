# Mission Control Kanban (Site Consumer) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the read-only Mission Control **Kanban** view at `/console/kanban` on the Plan 1 console foundation — four lenses (status, agent, source, risk+aging) over live work tasks, human-titled order cards, and a shared details drawer.

**Architecture:** The Site console is a read-only projection consumer (Plan 1 foundation: `graph/console.go` freshness state machine + `ConsolePage` shell, `graph/console.templ`). This plan adds a Kanban view that fetches the work-server `GET /tasks` (now enriched by Plan 2a with `risk_class`/`cell`/`factory_order_id`/`created_at`/`created_by`), maps it to a column view-model grouped by a selectable lens, and renders order cards + a details drawer. It does NOT modify the existing `/ops/*` dashboards or `fetchOpsWork` (the Kanban gets its own focused fetcher reusing the same work auth/client helpers).

**Tech Stack:** Go, Templ (HTML components, regenerated via `templ generate`), HTMX (fragment polling), `net/http`.

## Global Constraints

- **Honest staleness / no fabrication (the console's core contract).**
  - A work `/tasks` fetch error → freshness `unavailable`, **zero cards**, a human-readable notice. Never invent cards.
  - `effort-to-date` / `predicted-effort` and a clean `linked-PR` URL are **genuinely not in any projection yet** — the details drawer renders each as an explicit literal **"unavailable"** (muted, "not yet in any projection"). Never fabricate a value or a PR link.
  - Card **aging** comes from the real `created_at` only. An empty, unparseable, or future `created_at` → **no age label** (never a fabricated age).
  - Every card must land in a **visible** column. An empty group key renders as an explicit `unclassified` / `unassigned` / `unknown` column — fail-closed, never silently dropped.
- **Additive / non-perturbing.** Do not change `fetchOpsWork`, the `/ops/*` routes, or existing `OpsWorkTask` JSON tags. Only ADD fields (with `omitempty`) and ADD new console code.
- **Reuse the Plan 1 foundation verbatim:** `ConsoleFreshness` + `deriveFreshness` (fail-closed), `consoleStaleWindow`, the `ConsolePage`/`consoleTab`/`consoleFreshnessBadge` templ components, the `renderConsole` helper, the `Register` + `RegisterReadOnlyConsole` route pattern, and the `httptest`-upstream test idiom.
- **Canonical orderings (Site defines these; it only sees JSON strings):**
  - `risk_class` severity (column order): `critical`, `high`, `medium`, `low`, then `unclassified` last. Valid values upstream are exactly `{low, medium, high, critical}`.
  - v3.9 `status` lifecycle (column order): `created`, `ready`, `running`, `blocked`, `failed`, `repair_required`, `repair_running`, `repaired`, `verification_running`, `verified`, `certified`, `rejected`, `superseded`, `policy_blocked`; any unrecognized status after these; `unknown` (empty) last.
  - `agent` (assignee) and `source` (`created_by`) lenses: known keys sorted alphabetically, then the empty-key column (`unassigned` / `unknown`) last.
  - **Default lens = `risk`** (per the design). An empty/unrecognized `?lens=` value resolves to `risk`.
- **Go:** handle every error explicitly; table-driven tests; `*_test.go` in package `graph`.
- **Templ:** after editing any `.templ`, run `make generate` (runs `templ generate`); the generated `*_templ.go` files are committed. `make verify` depends on `generate`.
- **Commits:** conventional, lowercase imperative subject, no trailing period. Branch `feat/mission-control-kanban` (never `main`).
- **Verify gate:** `make generate` (clean tree) · `go build ./...` · `go vet ./...` · `go test ./graph/...`.

---

### Task 1: Add the Kanban fields to `OpsWorkTask`

**Files:**
- Modify: `graph/ops.go` (the `OpsWorkTask` struct, ~line 505-518)
- Test: `graph/console_kanban_test.go` (new; package `graph`)

**Interfaces:**
- Produces: `OpsWorkTask` gains `CreatedBy`, `RiskClass`, `Cell`, `FactoryOrderID`, `CreatedAt` (all `string`, `omitempty`) — consumed by Tasks 2-5.

- [ ] **Step 1: Write the failing test**

Create `graph/console_kanban_test.go`:

```go
package graph

import (
	"encoding/json"
	"testing"
)

func TestOpsWorkTaskDecodesKanbanFields(t *testing.T) {
	const body = `{
		"id": "task_1", "title": "Build civic-roles doc",
		"status": "running", "assignee": "implementer",
		"created_by": "michael", "risk_class": "high",
		"cell": "cell_a", "factory_order_id": "fo_42",
		"created_at": "2026-06-30T12:00:00Z"
	}`
	var task OpsWorkTask
	if err := json.Unmarshal([]byte(body), &task); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if task.CreatedBy != "michael" {
		t.Errorf("CreatedBy = %q, want michael", task.CreatedBy)
	}
	if task.RiskClass != "high" {
		t.Errorf("RiskClass = %q, want high", task.RiskClass)
	}
	if task.Cell != "cell_a" {
		t.Errorf("Cell = %q, want cell_a", task.Cell)
	}
	if task.FactoryOrderID != "fo_42" {
		t.Errorf("FactoryOrderID = %q, want fo_42", task.FactoryOrderID)
	}
	if task.CreatedAt != "2026-06-30T12:00:00Z" {
		t.Errorf("CreatedAt = %q, want the RFC3339 string", task.CreatedAt)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./graph/ -run TestOpsWorkTaskDecodesKanbanFields -v`
Expected: FAIL — compile error, `task.CreatedBy` undefined.

- [ ] **Step 3: Implement**

In `graph/ops.go`, add to the `OpsWorkTask` struct (after `MissingGates`):

```go
	MissingGates  []string `json:"missing_gates"`
	CreatedBy      string  `json:"created_by,omitempty"`
	RiskClass      string  `json:"risk_class,omitempty"`
	Cell           string  `json:"cell,omitempty"`
	FactoryOrderID string  `json:"factory_order_id,omitempty"`
	CreatedAt      string  `json:"created_at,omitempty"`
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./graph/ -run TestOpsWorkTaskDecodesKanbanFields -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add graph/ops.go graph/console_kanban_test.go
git commit -m "feat: add kanban fields to opsworktask"
```

---

### Task 2: Console-scoped work fetcher `fetchConsoleWork`

**Files:**
- Create: `graph/console_kanban.go` (new; package `graph`)
- Test: `graph/console_kanban_test.go`

**Interfaces:**
- Consumes (existing package-internal helpers in `graph/ops.go`): `serverWorkAPIBaseURL() string`, `legacyWorkURL(base, path string) string`, `setWorkAuth(req *http.Request)`, the `obsWorkClient` HTTP client, and the `opsWorkTasksResponse` type (which has a `Tasks []OpsWorkTask` field). Confirm each name by reading `fetchOpsWork` in `graph/ops.go` (~line 4534) before use; if any differs, use the real name.
- Produces: `type consoleWorkResult struct { GeneratedAt string; Tasks []OpsWorkTask; Err error }` and `func fetchConsoleWork(r *http.Request) consoleWorkResult` — consumed by Tasks 4-5.

- [ ] **Step 1: Write the failing test**

Add to `graph/console_kanban_test.go`:

```go
func TestFetchConsoleWorkDecodesTasks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tasks":[
			{"id":"task_1","title":"Build civic-roles doc","status":"running",
			 "assignee":"implementer","created_by":"michael","risk_class":"high",
			 "cell":"cell_a","factory_order_id":"fo_42","created_at":"2026-06-30T12:00:00Z"}
		]}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban", nil)
	res := fetchConsoleWork(req)
	if res.Err != nil {
		t.Fatalf("unexpected error: %v", res.Err)
	}
	if len(res.Tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(res.Tasks))
	}
	if res.Tasks[0].RiskClass != "high" || res.Tasks[0].CreatedBy != "michael" {
		t.Errorf("enriched fields not decoded: %+v", res.Tasks[0])
	}
}

func TestFetchConsoleWorkReportsUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban", nil)
	res := fetchConsoleWork(req)
	if res.Err == nil {
		t.Fatal("want an error on non-2xx upstream, got nil")
	}
	if len(res.Tasks) != 0 {
		t.Fatalf("want zero tasks on error, got %d", len(res.Tasks))
	}
}
```

Add imports to the test file as needed: `net/http`, `net/http/httptest`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./graph/ -run TestFetchConsoleWork -v`
Expected: FAIL — `fetchConsoleWork` undefined.

- [ ] **Step 3: Implement**

Create `graph/console_kanban.go`. First confirm the helper/type names by reading `fetchOpsWork` (`graph/ops.go` ~4534); the names below match what that function uses.

```go
package graph

import (
	"fmt"
	"net/http"
	"time"

	"encoding/json"
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
```

(If `graph/ops.go` already imports `encoding/json`/`time` package-wide, keep this file's imports limited to what it uses; group them per gofmt.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./graph/ -run TestFetchConsoleWork -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add graph/console_kanban.go graph/console_kanban_test.go
git commit -m "feat: add console-scoped work tasks fetcher"
```

---

### Task 3: Kanban view-model builder (lenses, aging, fail-closed)

**Files:**
- Modify: `graph/console_kanban.go`
- Test: `graph/console_kanban_test.go`

**Interfaces:**
- Consumes: `consoleWorkResult.Tasks []OpsWorkTask` (Task 2), `ConsoleFreshness`/`deriveFreshness`/`consoleStaleWindow` (Plan 1, `graph/console.go`).
- Produces (consumed by Tasks 4-5):
  - `type ConsoleKanbanLens string` with consts `LensRisk="risk"`, `LensStatus="status"`, `LensAgent="agent"`, `LensSource="source"`.
  - `func parseLens(raw string) ConsoleKanbanLens` (unknown/empty → `LensRisk`).
  - `type ConsoleOrderCard struct { ID, Title, FactoryOrderID, Submitter, Status, Agent, Risk, Cell, CreatedAt, AgeLabel string }`.
  - `type ConsoleKanbanColumn struct { Key, Label string; Cards []ConsoleOrderCard }`.
  - `type ConsoleKanban struct { Freshness ConsoleFreshness; GeneratedAt string; Lens ConsoleKanbanLens; Columns []ConsoleKanbanColumn; TotalCards int; Notices []string }`.
  - `func buildConsoleKanban(tasks []OpsWorkTask, fetchErr error, lens ConsoleKanbanLens, now time.Time) ConsoleKanban`.
  - `func cardForTask(t OpsWorkTask, now time.Time) ConsoleOrderCard` (also reused by Task 5).
  - `func humanizeAge(now time.Time, createdAt string) string`.

- [ ] **Step 1: Write the failing test**

Add to `graph/console_kanban_test.go`:

```go
func sampleKanbanTasks() []OpsWorkTask {
	return []OpsWorkTask{
		{ID: "t1", Title: "Alpha", Status: "running", Assignee: "implementer",
			CreatedBy: "michael", RiskClass: "high", CreatedAt: "2026-06-29T12:00:00Z"},
		{ID: "t2", Title: "Bravo", Status: "blocked", Assignee: "",
			CreatedBy: "codex", RiskClass: "critical", CreatedAt: "2026-06-28T12:00:00Z"},
		{ID: "t3", Title: "Charlie", Status: "running", Assignee: "implementer",
			CreatedBy: "michael", RiskClass: "", CreatedAt: ""},
	}
}

func columnKeys(k ConsoleKanban) []string {
	keys := make([]string, 0, len(k.Columns))
	for _, c := range k.Columns {
		keys = append(keys, c.Key)
	}
	return keys
}

func TestBuildConsoleKanbanFetchErrorIsUnavailable(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	k := buildConsoleKanban(nil, fmt.Errorf("boom"), LensRisk, now)
	if k.Freshness != FreshnessUnavailable {
		t.Errorf("freshness = %q, want unavailable", k.Freshness)
	}
	if k.TotalCards != 0 || len(k.Columns) != 0 {
		t.Errorf("want zero cards/columns on error, got %d cards %d cols", k.TotalCards, len(k.Columns))
	}
	if len(k.Notices) == 0 {
		t.Error("want a notice explaining the unavailable state")
	}
}

func TestBuildConsoleKanbanRiskLensOrderAndUnclassified(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	k := buildConsoleKanban(sampleKanbanTasks(), nil, LensRisk, now)
	if k.Freshness != FreshnessCurrent {
		t.Errorf("freshness = %q, want current on a clean fetch", k.Freshness)
	}
	// critical, high present; low/medium absent (omitted); unclassified (empty risk) last.
	if got := columnKeys(k); !reflect.DeepEqual(got, []string{"critical", "high", "unclassified"}) {
		t.Errorf("risk columns = %v, want [critical high unclassified]", got)
	}
	if k.TotalCards != 3 {
		t.Errorf("TotalCards = %d, want 3", k.TotalCards)
	}
}

func TestBuildConsoleKanbanStatusLensOrder(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	k := buildConsoleKanban(sampleKanbanTasks(), nil, LensStatus, now)
	// lifecycle order: running before blocked.
	if got := columnKeys(k); !reflect.DeepEqual(got, []string{"running", "blocked"}) {
		t.Errorf("status columns = %v, want [running blocked]", got)
	}
}

func TestBuildConsoleKanbanAgentLensUnassignedLast(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	k := buildConsoleKanban(sampleKanbanTasks(), nil, LensAgent, now)
	if got := columnKeys(k); !reflect.DeepEqual(got, []string{"implementer", "unassigned"}) {
		t.Errorf("agent columns = %v, want [implementer unassigned]", got)
	}
}

func TestBuildConsoleKanbanSourceLens(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	k := buildConsoleKanban(sampleKanbanTasks(), nil, LensSource, now)
	if got := columnKeys(k); !reflect.DeepEqual(got, []string{"codex", "michael"}) {
		t.Errorf("source columns = %v, want [codex michael]", got)
	}
}

func TestHumanizeAge(t *testing.T) {
	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name, in, want string
	}{
		{"days", "2026-06-28T12:00:00Z", "2d"},
		{"hours", "2026-06-30T09:00:00Z", "3h"},
		{"minutes", "2026-06-30T11:30:00Z", "30m"},
		{"empty", "", ""},
		{"unparseable", "not-a-time", ""},
		{"future", "2026-07-01T12:00:00Z", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := humanizeAge(now, c.in); got != c.want {
				t.Errorf("humanizeAge(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestParseLensDefaultsToRisk(t *testing.T) {
	for _, raw := range []string{"", "bogus", "RISK?"} {
		if got := parseLens(raw); got != LensRisk {
			t.Errorf("parseLens(%q) = %q, want risk", raw, got)
		}
	}
	if parseLens("status") != LensStatus {
		t.Error("parseLens(status) should be status")
	}
}
```

Add `reflect` to the test imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./graph/ -run "TestBuildConsoleKanban|TestHumanizeAge|TestParseLens" -v`
Expected: FAIL — builder + helpers undefined.

- [ ] **Step 3: Implement**

Add to `graph/console_kanban.go` (add `sort`, `strings` to imports):

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./graph/ -run "TestBuildConsoleKanban|TestHumanizeAge|TestParseLens" -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add graph/console_kanban.go graph/console_kanban_test.go
git commit -m "feat: add kanban view-model builder with lenses and aging"
```

---

### Task 4: Kanban templ view + handler + routes + enable the nav tab

**Files:**
- Modify: `graph/console.go` (add `Kanban *ConsoleKanban` to `ConsolePageData`; add two handlers)
- Modify: `graph/console.templ` (enable the Kanban tab; add Kanban components; render Kanban in `ConsolePage`)
- Modify: `graph/handlers.go` (register routes in `Register` ~line 390-393 and `RegisterReadOnlyConsole` ~line 424-428)
- Regenerate: `graph/console_templ.go` (via `make generate`)
- Test: `graph/console_kanban_test.go`

**Interfaces:**
- Consumes: `buildConsoleKanban`, `fetchConsoleWork`, `parseLens` (Tasks 2-3); `renderConsole`, `ConsolePageData`, `consoleTab`, `consoleFreshnessBadge` (Plan 1).
- Produces: routes `GET /console/kanban` and `GET /console/kanban/fragment`; handlers `handleConsoleKanban`, `handleConsoleKanbanFragment`; templ components `consoleKanbanFragment`, `consoleKanban`, `consoleKanbanLensNav`, `consoleOrderCard`. (`consoleOrderCard` is reused by Task 5.)

- [ ] **Step 1: Write the failing test**

Add to `graph/console_kanban_test.go` (reuse the console test handler constructor — read `console_test.go` for the exact `newConsoleTestHandlers()` / `testHandlers(t)` idiom and mirror it; the work upstream env is `WORK_API_BASE_URL`):

```go
func TestConsoleKanbanRendersOrderCards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tasks":[
			{"id":"task_1","title":"Build civic-roles doc","status":"running",
			 "assignee":"implementer","created_by":"michael","risk_class":"high",
			 "cell":"cell_a","factory_order_id":"fo_42","created_at":"2026-06-29T12:00:00Z"}
		]}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	h := newConsoleTestHandlers() // mirror console_test.go
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{"Build civic-roles doc", "fo_42", "michael", "implementer", "high"} {
		if !strings.Contains(body, want) {
			t.Errorf("kanban body missing %q", want)
		}
	}
}

func TestConsoleKanbanUpstreamErrorIsHonest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unavailable") {
		t.Error("expected an explicit unavailable state on upstream error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./graph/ -run TestConsoleKanban -v`
Expected: FAIL — `/console/kanban` route not registered (404) / components undefined.

- [ ] **Step 3: Implement**

(a) In `graph/console.go`, add the field to `ConsolePageData`:

```go
type ConsolePageData struct {
	Title  string
	Active string // health | kanban | intake | config
	Health *ConsoleHealthWall
	Kanban *ConsoleKanban
}
```

(b) In `graph/console.go`, add the handlers (mirror `handleConsoleHealth`):

```go
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
```

(c) In `graph/console.templ`: enable the Kanban tab (flip the 5th arg `false`→`true`):

```templ
@consoleTab("kanban", "Kanban", "/console/kanban", data.Active, true)
```

In `ConsolePage`, render the Kanban view-model when present (add after the `data.Health` block):

```templ
				if data.Health != nil {
					@consoleHealthWallFragment(*data.Health)
				}
				if data.Kanban != nil {
					@consoleKanbanFragment(*data.Kanban)
				}
```

Add the Kanban components to `console.templ`:

```templ
templ consoleKanbanFragment(k ConsoleKanban) {
	<div id="console-kanban" hx-get={ "/console/kanban/fragment?lens=" + string(k.Lens) } hx-trigger="every 10s" hx-swap="outerHTML">
		@consoleKanban(k)
	</div>
}

templ consoleKanban(k ConsoleKanban) {
	<div class="space-y-4">
		<div class="flex items-center justify-between gap-3">
			@consoleKanbanLensNav(k.Lens)
			@consoleFreshnessBadge(k.Freshness, k.GeneratedAt)
		</div>
		if k.Freshness == FreshnessUnavailable {
			<p class="text-sm text-warm-muted">unavailable — { noticeText(k.Notices) }</p>
		} else if k.TotalCards == 0 {
			<p class="text-sm text-warm-muted">No orders.</p>
		} else {
			<div class="flex gap-4 overflow-x-auto pb-2">
				for _, col := range k.Columns {
					<section class="min-w-[16rem] flex-shrink-0 space-y-2">
						<h2 class="text-xs uppercase tracking-wide text-warm-muted">{ col.Label } · { columnCount(col) }</h2>
						for _, card := range col.Cards {
							@consoleOrderCard(card)
						}
					</section>
				}
			</div>
		}
		<div id="console-kanban-drawer"></div>
	</div>
}

templ consoleKanbanLensNav(active ConsoleKanbanLens) {
	<div class="flex gap-1 text-xs">
		@consoleLensLink("risk", "Risk + aging", active)
		@consoleLensLink("status", "Status", active)
		@consoleLensLink("agent", "Agent", active)
		@consoleLensLink("source", "Source", active)
	</div>
}

templ consoleLensLink(lens, label string, active ConsoleKanbanLens) {
	if ConsoleKanbanLens(lens) == active {
		<a href={ templ.SafeURL("/console/kanban?lens=" + lens) } class="px-2 py-1 border-b-2 border-brand text-warm">{ label }</a>
	} else {
		<a href={ templ.SafeURL("/console/kanban?lens=" + lens) } class="px-2 py-1 text-warm-muted hover:text-warm transition-colors">{ label }</a>
	}
}

templ consoleOrderCard(card ConsoleOrderCard) {
	<button
		type="button"
		class="w-full text-left rounded border border-edge bg-surface/40 p-3 space-y-1 hover:border-brand transition-colors"
		hx-get={ templ.SafeURL("/console/kanban/order/" + card.ID) }
		hx-target="#console-kanban-drawer"
		hx-swap="innerHTML"
	>
		<div class="flex items-center justify-between gap-2">
			<span class="text-sm text-warm font-medium">{ cardTitle(card) }</span>
			<span class="text-[10px] text-warm-muted font-mono">{ cardTag(card) }</span>
		</div>
		<div class="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-warm-muted">
			<span>by { orFallback(card.Submitter, "unknown") }</span>
			<span>{ orFallback(card.Status, "unknown") }</span>
			<span>{ orFallback(card.Agent, "unassigned") }</span>
			if card.Risk != "" {
				<span>risk: { card.Risk }</span>
			}
			if card.AgeLabel != "" {
				<span>{ card.AgeLabel }</span>
			}
		</div>
	</button>
}
```

(d) Add the small templ string helpers. Templ cannot call arbitrary expressions inline cleanly, so add these to `graph/console_kanban.go`:

```go
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
```

(e) In `graph/handlers.go`, register the routes. In `Register` (after the `/console/health/fragment` line ~393):

```go
	mux.Handle("GET /console/kanban", h.writeWrap(h.handleConsoleKanban))
	mux.Handle("GET /console/kanban/fragment", h.writeWrap(h.handleConsoleKanbanFragment))
```

In `RegisterReadOnlyConsole` (after its `/console/health/fragment` line ~428):

```go
	mux.HandleFunc("GET /console/kanban", h.handleConsoleKanban)
	mux.HandleFunc("GET /console/kanban/fragment", h.handleConsoleKanbanFragment)
```

(f) Regenerate templ:

```bash
make generate
```

- [ ] **Step 4: Run test to verify it passes**

Run: `make generate && go test ./graph/ -run TestConsoleKanban -v`
Expected: PASS. Then `go build ./... && go vet ./...` clean.

- [ ] **Step 5: Commit**

```bash
git add graph/console.go graph/console.templ graph/console_templ.go graph/handlers.go graph/console_kanban.go graph/console_kanban_test.go
git commit -m "feat: render mission control kanban view with lenses"
```

---

### Task 5: Order details drawer fragment (effort + linked-PR "unavailable")

**Files:**
- Modify: `graph/console.go` (add `handleConsoleKanbanOrder`)
- Modify: `graph/console.templ` (add `consoleOrderDrawer` component)
- Modify: `graph/handlers.go` (register `GET /console/kanban/order/{id}` in both `Register` and `RegisterReadOnlyConsole`)
- Regenerate: `graph/console_templ.go`
- Test: `graph/console_kanban_test.go`

**Interfaces:**
- Consumes: `fetchConsoleWork`, `cardForTask` (Tasks 2-3); `renderConsole` is NOT used (this returns a bare fragment).
- Produces: route `GET /console/kanban/order/{id}`; handler `handleConsoleKanbanOrder`; templ component `consoleOrderDrawer(card ConsoleOrderCard, found bool)`.

- [ ] **Step 1: Write the failing test**

Add to `graph/console_kanban_test.go`:

```go
func TestConsoleOrderDrawerShowsUnavailableForEffortAndPR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"tasks":[
			{"id":"task_1","title":"Build civic-roles doc","status":"running",
			 "assignee":"implementer","created_by":"michael","risk_class":"high",
			 "factory_order_id":"fo_42","created_at":"2026-06-29T12:00:00Z"}
		]}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban/order/task_1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Build civic-roles doc") {
		t.Error("drawer missing order title")
	}
	// effort + linked PR are genuinely not in any projection — must read "unavailable", never fabricated.
	if strings.Count(body, "unavailable") < 2 {
		t.Errorf("drawer must mark effort and linked-PR unavailable; body: %s", body)
	}
}

func TestConsoleOrderDrawerUnknownIDIsHonest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tasks":[]}`))
	}))
	defer srv.Close()
	t.Setenv("WORK_API_BASE_URL", srv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/kanban/order/nope", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "unavailable") && !strings.Contains(w.Body.String(), "not found") {
		t.Error("unknown order must render an honest not-found/unavailable drawer, not a fabricated order")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./graph/ -run TestConsoleOrderDrawer -v`
Expected: FAIL — route/handler/component undefined.

- [ ] **Step 3: Implement**

(a) In `graph/console.go`, add the handler:

```go
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
```

(b) In `graph/console.templ`, add the drawer component:

```templ
templ consoleOrderDrawer(card ConsoleOrderCard, found bool) {
	<aside class="rounded border border-edge bg-surface/60 p-4 space-y-3" role="dialog" aria-label="Order details">
		<div class="flex items-center justify-between gap-2">
			<h3 class="text-sm font-medium text-warm">Order details</h3>
			<button type="button" class="text-xs text-warm-muted hover:text-warm" hx-get="/console/kanban/order/none/close" hx-target="#console-kanban-drawer" hx-swap="innerHTML" onclick="document.getElementById('console-kanban-drawer').innerHTML='';return false;">close</button>
		</div>
		if !found {
			<p class="text-sm text-warm-muted">Order { card.ID } is unavailable (not found).</p>
		} else {
			<dl class="grid grid-cols-[8rem,1fr] gap-y-1 text-xs">
				<dt class="text-warm-muted">Title</dt><dd class="text-warm">{ cardTitle(card) }</dd>
				<dt class="text-warm-muted">Order</dt><dd class="font-mono text-warm">{ cardTag(card) }</dd>
				<dt class="text-warm-muted">Submitter</dt><dd>{ orFallback(card.Submitter, "unknown") }</dd>
				<dt class="text-warm-muted">Status</dt><dd>{ orFallback(card.Status, "unknown") }</dd>
				<dt class="text-warm-muted">Agent</dt><dd>{ orFallback(card.Agent, "unassigned") }</dd>
				<dt class="text-warm-muted">Risk</dt><dd>{ orFallback(card.Risk, "unclassified") }</dd>
				<dt class="text-warm-muted">Cell</dt><dd>{ orFallback(card.Cell, "—") }</dd>
				<dt class="text-warm-muted">Age</dt><dd>{ orFallback(card.AgeLabel, "—") }</dd>
				<dt class="text-warm-muted">Effort to date</dt><dd class="text-warm-muted italic">unavailable — not yet in any projection</dd>
				<dt class="text-warm-muted">Predicted effort</dt><dd class="text-warm-muted italic">unavailable — not yet in any projection</dd>
				<dt class="text-warm-muted">Linked PR</dt><dd class="text-warm-muted italic">unavailable — not yet in any projection</dd>
			</dl>
		}
	</aside>
}
```

(Note: the close control clears the drawer client-side via the inline `onclick`; the `hx-get` is a harmless fallback. If the reviewer prefers no JS, the `onclick` may be simplified — keep the "close" affordance.)

(c) In `graph/handlers.go`, register the route in `Register` (after the kanban fragment route):

```go
	mux.Handle("GET /console/kanban/order/{id}", h.writeWrap(h.handleConsoleKanbanOrder))
```

and in `RegisterReadOnlyConsole`:

```go
	mux.HandleFunc("GET /console/kanban/order/{id}", h.handleConsoleKanbanOrder)
```

(d) Regenerate templ:

```bash
make generate
```

- [ ] **Step 4: Run test to verify it passes**

Run: `make generate && go test ./graph/ -run TestConsoleOrderDrawer -v`
Expected: PASS. Then the full gate: `make generate && go build ./... && go vet ./... && go test ./graph/...`.

- [ ] **Step 5: Commit**

```bash
git add graph/console.go graph/console.templ graph/console_templ.go graph/handlers.go
git commit -m "feat: add kanban order details drawer with unavailable effort and pr"
```

---

## Self-Review

- **Spec coverage:** four lenses (status/agent/source/risk+aging) ✓ (Task 3 builder + Task 4 nav); human-titled cards with fo/ID tag + submitter + status + agent + risk + aging ✓ (Task 4 `consoleOrderCard`); shared order-card + details-drawer ✓ (Tasks 4-5); drawer renders effort-to-date/predicted + linked-PR as explicit "unavailable" ✓ (Task 5); default lens risk ✓ (`parseLens`); enable the Kanban nav tab ✓ (Task 4); built on the Plan 1 foundation (freshness machine, `ConsolePage`, fragment pattern, route registration incl. no-DB) ✓.
- **Honest staleness / no fabrication:** fetch error → unavailable + zero cards (Task 3 test); empty group keys → explicit columns, never dropped (Task 3 tests); aging only from real `created_at`, future/unparseable → no label (Task 3 `TestHumanizeAge`); effort + linked-PR → literal "unavailable" (Task 5 test); unknown order id → honest not-found (Task 5 test). ✓
- **Additive:** `OpsWorkTask` gains `omitempty` fields only; `fetchOpsWork` and `/ops` untouched; new Kanban code is isolated in `console_kanban.go` + new console handlers. ✓
- **Type consistency:** `ConsoleKanban`/`ConsoleKanbanColumn`/`ConsoleOrderCard`/`ConsoleKanbanLens` defined in Task 3, consumed unchanged in Tasks 4-5; `cardForTask` defined Task 3, reused Task 5; helper names (`cardTitle`/`cardTag`/`orFallback`/`columnCount`/`noticeText`) defined Task 4, used by Task 4-5 templ. ✓
- **Deferred (out of scope, consistent with Plan 2a):** effort/predicted-effort and linked-PR are not in any projection — rendered "unavailable" (a later upstream enrichment). The `/ops/*` retirement is a future plan. Per-agent live FSM state is a telemetry enrichment, not in this view.
