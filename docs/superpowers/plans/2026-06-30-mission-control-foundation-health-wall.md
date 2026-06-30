# Mission Control — Console Foundation + Health Wall Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the new `/console` operator shell and a live, read-only Health Wall that renders the running Civilization with honest staleness.

**Architecture:** A new, self-contained console shell in `site` (`graph/console.go` + `graph/console.templ`) mounted at `/console`, decoupled from the 4800-LOC `ops.go`/3200-LOC `ops.templ` surfaces it will eventually retire. It reuses the existing upstream fetcher `fetchHiveOperatorProjection` (already calls hive's `hive-ops-api`) and maps its result into console view-models through an explicit freshness state machine. Phase 1 is read-only: no governed writes.

**Tech Stack:** Go 1.25 + Templ + HTMX v2 (CDN) + Tailwind v4 + `net/http`. No new dependencies.

**Plan scope:** This is **Plan 1 of 4** for Phase 1 (the full vertical slice). Plans 2–4 (Kanban, Intake-draft, Config read-only) build on the shell and freshness machine delivered here and will be written when reached. Design source: `site/docs/designs/mission-control-console-design-v0.1.0.md` (`SITE-MISSION-CONTROL-DESIGN-001`).

## Global Constraints

- **Read-only in Phase 1.** No governed writes (no approve/deny, submit, or model change). Routes use `h.writeWrap` (auth required) like other `/ops/*` routes, but perform no mutation.
- **Honest staleness is mandatory.** Missing/stale/partial/unavailable upstream data must render as an explicit state, never a healthy default. The permissive ("current") outcome is the explicitly-proven branch; the fall-through is `unavailable` (fail closed).
- **No new charting/JS dependency** unless a visualization cannot be expressed with existing Templ/Tailwind/SVG (per MFOF-001).
- **Run `templ generate` after every `.templ` edit.** Never edit `*_templ.go` by hand.
- **Tests run without a database** for these read-only routes. Construct handlers with a nil store via `NewHandlers(nil, nil, passthroughWrap)` (the codebase's established no-DB pattern — see `graph/observatory_test.go:458` and `graph/ops_test.go:3380`; `viewUser` guards `h.store == nil`). Use a `newConsoleTestHandlers()` local helper (defined in Task 5) + `httptest.Server` to mock upstream. Do NOT use `testHandlers(t)` — it spins up Postgres via `testDB`.
- **Go:** handle every error explicitly; table-driven tests; `*_test.go` in package `graph`.
- **Commits:** conventional, lowercase imperative subject, no trailing period. Already on branch `feat/mission-control-console-design` (never commit to `main`).
- **Build/verify commands:** `templ generate` · `go test -count=1 ./graph/...` · `go build ./cmd/site/` · `make verify`.

---

### Task 1: Freshness state machine

The shared honest-staleness primitive every console view will use. Fail-closed: anything ambiguous resolves to `unavailable`.

**Files:**
- Create: `graph/console.go`
- Test: `graph/console_test.go`

**Interfaces:**
- Produces: `type ConsoleFreshness string`; constants `FreshnessCurrent`, `FreshnessStale`, `FreshnessPartial`, `FreshnessUnavailable`; `func deriveFreshness(generatedAt string, fetchErr error, hasPartialErrors bool, now time.Time, staleWindow time.Duration) ConsoleFreshness`; const `consoleStaleWindow = 30 * time.Second`.

- [ ] **Step 1: Write the failing test**

```go
package graph

import (
	"errors"
	"testing"
	"time"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./graph/ -run TestDeriveFreshness -v`
Expected: FAIL — `undefined: deriveFreshness` (and the constants).

- [ ] **Step 3: Write minimal implementation**

```go
package graph

import "time"

type ConsoleFreshness string

const (
	FreshnessCurrent     ConsoleFreshness = "current"
	FreshnessStale       ConsoleFreshness = "stale"
	FreshnessPartial     ConsoleFreshness = "partial"
	FreshnessUnavailable ConsoleFreshness = "unavailable"
)

const consoleStaleWindow = 30 * time.Second

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
	if now.Sub(ts) > staleWindow {
		return FreshnessStale
	}
	if hasPartialErrors {
		return FreshnessPartial
	}
	return FreshnessCurrent
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./graph/ -run TestDeriveFreshness -v`
Expected: PASS (all six subtests).

- [ ] **Step 5: Commit**

```bash
git add graph/console.go graph/console_test.go
git commit -m "feat: add console freshness state machine"
```

---

### Task 2: Health Wall view-model + mapping

Map the existing `*OpsHiveProjection` (or a fetch error) into a `ConsoleHealthWall` the templ can render. This is the only place upstream shape meets the view.

**Files:**
- Modify: `graph/console.go`
- Test: `graph/console_test.go`

**Interfaces:**
- Consumes: `fetchHiveOperatorProjection(r) (*OpsHiveProjection, error)` and types `OpsHiveProjection`, `OpsHiveApproval`, `OpsHiveRuntimeAgent` (defined in `graph/ops.go`); `deriveFreshness` (Task 1).
- Produces: `type ConsoleHealthWall struct`, `type ConsoleAgentRow struct`, `type ConsoleApproval struct`; `func buildConsoleHealthWall(proj *OpsHiveProjection, fetchErr error, now time.Time) ConsoleHealthWall`.

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./graph/ -run TestBuildConsoleHealthWall -v`
Expected: FAIL — `undefined: buildConsoleHealthWall` / `ConsoleHealthWall`.

- [ ] **Step 3: Write minimal implementation** (append to `graph/console.go`)

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./graph/ -run TestBuildConsoleHealthWall -v`
Expected: PASS (all three subtests).

- [ ] **Step 5: Commit**

```bash
git add graph/console.go graph/console_test.go
git commit -m "feat: map hive projection into health wall view-model"
```

---

### Task 3: Console shell + nav + freshness badge (templ)

The reusable shell every console view renders inside, plus the freshness badge component that makes staleness visible.

**Files:**
- Create: `graph/console.templ`
- Modify: `graph/console.go` (add `ConsolePageData` + `renderConsole`)
- Reference (do not edit): `graph/ops.templ` `OpsPage` head/body boilerplate (CSS + htmx includes) — mirror its `<html>`/`<head>` block so assets load identically.

**Interfaces:**
- Consumes: `h.viewUser(r) ViewUser`, `profile.FromContext(ctx) *profile.Profile` (used by `renderOps` in `graph/ops.go:1840`).
- Produces: `type ConsolePageData struct { Title string; Active string; Health *ConsoleHealthWall }`; templ `ConsolePage(data ConsolePageData, user ViewUser, p *profile.Profile)`; templ `consoleFreshnessBadge(f ConsoleFreshness, generatedAt string)`; method `func (h *Handlers) renderConsole(w http.ResponseWriter, r *http.Request, data ConsolePageData)`.

- [ ] **Step 1: Add `ConsolePageData` + `renderConsole` to `graph/console.go`**

```go
import (
	"net/http"

	"<module>/profile" // match the existing import path used in graph/ops.go for profile
)

type ConsolePageData struct {
	Title  string
	Active string // health | kanban | intake | config
	Health *ConsoleHealthWall
}

func (h *Handlers) renderConsole(w http.ResponseWriter, r *http.Request, data ConsolePageData) {
	ConsolePage(data, h.viewUser(r), profile.FromContext(r.Context())).Render(r.Context(), w)
}
```

Note: copy the exact `profile` import path from the top of `graph/ops.go` (same package, so `ViewUser` and `h.viewUser` need no import).

- [ ] **Step 2: Write the shell + badge templ**

```go
package graph

import "<module>/profile" // match graph/ops.templ's profile import

templ ConsolePage(data ConsolePageData, user ViewUser, p *profile.Profile) {
	<!DOCTYPE html>
	<html lang="en">
		@consoleHead(data.Title)
		<body class="bg-void text-warm-secondary min-h-screen flex flex-col">
			<main class="flex-1 max-w-6xl mx-auto px-4 md:px-6 py-8 md:py-10 w-full space-y-6">
				<header class="flex items-center justify-between gap-3">
					<div>
						<h1 class="text-2xl md:text-3xl font-display font-normal text-warm">Mission control</h1>
						<p class="text-sm text-warm-muted">transpara-ai civilization</p>
					</div>
				</header>
				<nav class="flex gap-1 border-b border-edge pb-1 text-sm">
					@consoleTab("health", "Health wall", "/console", data.Active)
					@consoleTab("kanban", "Kanban", "/console/kanban", data.Active)
					@consoleTab("intake", "Intake", "/console/intake", data.Active)
					@consoleTab("config", "Config", "/console/config", data.Active)
				</nav>
				if data.Health != nil {
					<div id="console-health" data-console-surface="health"></div>
				}
			</main>
		</body>
	</html>
}

templ consoleHead(title string) {
	<head>
		<meta charset="utf-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1"/>
		<title>{ title } — Mission control</title>
		<link rel="stylesheet" href={ consoleCSSHref() }/>
		<script src="/static/js/htmx.min.js"></script>
	</head>
}

templ consoleTab(id, label, href, active string) {
	if id == active {
		<a href={ templ.SafeURL(href) } class="px-3 py-2 border-b-2 border-brand text-warm font-medium">{ label }</a>
	} else {
		<a href={ templ.SafeURL(href) } class="px-3 py-2 text-warm-muted hover:text-warm transition-colors">{ label }</a>
	}
}

templ consoleFreshnessBadge(f ConsoleFreshness, generatedAt string) {
	switch f {
		case FreshnessCurrent:
			<span class="inline-flex items-center gap-1 text-xs text-emerald-300">live · { generatedAt }</span>
		case FreshnessStale:
			<span class="inline-flex items-center gap-1 text-xs text-amber-300">stale · last update { generatedAt }</span>
		case FreshnessPartial:
			<span class="inline-flex items-center gap-1 text-xs text-amber-300">partial · some sources degraded</span>
		default:
			<span class="inline-flex items-center gap-1 text-xs text-warm-muted">unavailable</span>
	}
}
```

Notes for the implementer:
- `consoleCSSHref()` is a tiny Go helper (add to `console.go`) that returns the versioned `site.css` URL. Reuse the existing helper that `views/layout.templ`/`ops.templ` use for the hashed CSS path (search `assets.` in `graph/` / `views/`); if it is `assets.SiteCSSHref()` (or similar), call that and delete the local helper. The plan provides a fallback in Task 5 if no helper is found.
- Tailwind class tokens (`bg-void`, `text-warm`, `border-edge`, `text-brand`) are the repo's existing semantic colors used throughout `ops.templ`; reuse them so the console matches. `text-emerald-300`/`text-amber-300` are Tailwind built-ins for the freshness states; if the repo defines semantic success/warning tokens, prefer those.

- [ ] **Step 3: Generate templ + build**

Run: `templ generate && go build ./cmd/site/`
Expected: builds clean. If `consoleCSSHref` is undefined, add the fallback from Task 5 Step 3 now.

- [ ] **Step 4: Commit**

```bash
git add graph/console.go graph/console.templ graph/console_templ.go
git commit -m "feat: add mission control console shell and freshness badge"
```

---

### Task 4: Health Wall component (templ)

Render the wall: status strip with freshness, agent roster (honest staleness), pending-approvals count, and notices.

**Files:**
- Modify: `graph/console.templ`

**Interfaces:**
- Consumes: `ConsoleHealthWall`, `consoleFreshnessBadge` (Task 3).
- Produces: templ `consoleHealthWall(w ConsoleHealthWall)`.

- [ ] **Step 1: Write the component**

```go
templ consoleHealthWall(w ConsoleHealthWall) {
	<section class="space-y-4" data-console-surface="health">
		<div class="flex items-center justify-between">
			<h2 class="text-lg font-medium text-warm">Health wall</h2>
			@consoleFreshnessBadge(w.Freshness, w.GeneratedAt)
		</div>
		if w.Freshness == FreshnessUnavailable {
			<div class="border border-edge bg-surface rounded-lg p-4 text-sm text-warm-muted" data-state="unavailable">
				<p class="font-medium text-warm">Live status unavailable</p>
				for _, n := range w.Notices {
					<p class="mt-1">{ n }</p>
				}
			</div>
		} else {
			<div class="grid grid-cols-2 md:grid-cols-3 gap-3">
				<div class="border border-edge bg-surface rounded-lg p-4">
					<p class="text-xs text-warm-muted">Active agents</p>
					<p class="text-2xl text-warm">{ strconv.Itoa(w.ActiveAgents) }</p>
				</div>
				<div class="border border-edge bg-surface rounded-lg p-4">
					<p class="text-xs text-warm-muted">Needs you</p>
					<p class="text-2xl text-warm">{ strconv.Itoa(w.PendingApprovals) }</p>
					<p class="text-xs text-warm-muted">pending approvals</p>
				</div>
			</div>
			<div class="border border-edge bg-surface rounded-lg overflow-hidden">
				<div class="px-4 py-2 border-b border-edge text-sm font-medium text-warm">Agents</div>
				if len(w.Agents) == 0 {
					<div class="px-4 py-3 text-sm text-warm-muted">No active agents reported.</div>
				} else {
					for _, a := range w.Agents {
						<div class="flex items-center gap-3 px-4 py-2 border-b border-edge last:border-0">
							<span class="text-sm font-medium text-warm w-28 truncate">{ a.Name }</span>
							<span class="text-xs text-warm-muted">{ a.Role }</span>
							<span class="text-xs text-warm-muted ml-auto">{ a.Model }</span>
						</div>
					}
				}
			</div>
			if len(w.Notices) > 0 {
				<div class="border border-amber-500/40 bg-surface rounded-lg p-3 text-xs text-amber-300">
					for _, n := range w.Notices {
						<p>{ n }</p>
					}
				</div>
			}
		}
	</section>
}
```

Add `"strconv"` to the `console.templ` import block.

- [ ] **Step 2: Wire the component into `ConsolePage`**

Task 3 left a placeholder in `ConsolePage`'s health region. Replace it so the real component renders:

```go
				if data.Health != nil {
					@consoleHealthWall(*data.Health)
				}
```

(Replaces the `<div id="console-health" data-console-surface="health"></div>` placeholder from Task 3. Task 6 will later wrap this call in a polling fragment.)

- [ ] **Step 3: Generate templ + build**

Run: `templ generate && go build ./cmd/site/`
Expected: builds clean.

- [ ] **Step 4: Commit**

```bash
git add graph/console.templ graph/console_templ.go
git commit -m "feat: render health wall with honest staleness"
```

---

### Task 5: Handler + route registration (the vertical thread closes)

Wire the handler, register `/console` and `/console/health`, and prove the whole thread with no-DB tests including the unavailable path.

**Files:**
- Modify: `graph/console.go` (handlers + optional `consoleCSSHref` fallback)
- Modify: `graph/handlers.go` (register routes in `Register`)
- Test: `graph/console_test.go`

**Interfaces:**
- Consumes: `fetchHiveOperatorProjection`, `buildConsoleHealthWall`, `renderConsole`.
- Produces: `func (h *Handlers) handleConsoleHealth(w http.ResponseWriter, r *http.Request)`; routes `GET /console`, `GET /console/health`.

- [ ] **Step 1: Write the failing test**

```go
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

		req := httptest.NewRequest(http.MethodGet, "http://site.test/console", nil)
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
			json.NewEncoder(w).Encode(proj)
		}))
		defer srv.Close()
		t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)

		mux := http.NewServeMux()
		h.Register(mux)
		req := httptest.NewRequest(http.MethodGet, "http://site.test/console", nil)
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
```

Add imports to `console_test.go`: `encoding/json`, `net/http`, `net/http/httptest`, `strings`, `time`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./graph/ -run TestHandleConsoleHealth -v`
Expected: FAIL — route not registered / `handleConsoleHealth` undefined.

- [ ] **Step 3: Implement the handler** (append to `graph/console.go`)

```go
func (h *Handlers) handleConsoleHealth(w http.ResponseWriter, r *http.Request) {
	proj, err := fetchHiveOperatorProjection(r)
	wall := buildConsoleHealthWall(proj, err, time.Now().UTC())
	h.renderConsole(w, r, ConsolePageData{Title: "Health wall", Active: "health", Health: &wall})
}
```

If `consoleCSSHref` was undefined in Task 3, add this fallback (and remove once the shared assets helper is wired):

```go
func consoleCSSHref() string { return "/static/css/site.css" }
```

Add `"time"` to `console.go` imports if not present.

- [ ] **Step 4: Register the routes** in `graph/handlers.go` `Register`, beside the other `/ops` routes (pattern matches `graph/handlers.go:278+`)

```go
	mux.Handle("GET /console", h.writeWrap(h.handleConsoleHealth))
	mux.Handle("GET /console/health", h.writeWrap(h.handleConsoleHealth))
```

- [ ] **Step 5: Run test to verify it passes**

Run: `templ generate && go test ./graph/ -run TestHandleConsoleHealth -v`
Expected: PASS (both subtests).

- [ ] **Step 6: Full package test + build**

Run: `go test -count=1 ./graph/... && go build ./cmd/site/`
Expected: all pass, clean build.

- [ ] **Step 7: Commit**

```bash
git add graph/console.go graph/handlers.go graph/console_test.go
git commit -m "feat: serve mission control health wall at /console"
```

---

### Task 6: Live refresh (HTMX polling fragment)

Make the wall self-refresh without a full page load, degrading honestly. Phase 1 uses HTMX polling (SSE is a later optimization per the spec's open question).

**Files:**
- Modify: `graph/console.go` (fragment handler)
- Modify: `graph/console.templ` (wrap roster region with `hx-get` polling; add a fragment-only render)
- Modify: `graph/handlers.go` (already registered `GET /console/health` in Task 5 — reuse it as the fragment source)
- Test: `graph/console_test.go`

**Interfaces:**
- Consumes: `buildConsoleHealthWall`, `consoleHealthWall`.
- Produces: templ `consoleHealthWallFragment(w ConsoleHealthWall)`; `func (h *Handlers) handleConsoleHealthFragment(w http.ResponseWriter, r *http.Request)`.

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./graph/ -run TestHandleConsoleHealthFragment -v`
Expected: FAIL — route/handler missing.

- [ ] **Step 3: Add the fragment templ** (in `graph/console.templ`)

```go
templ consoleHealthWallFragment(w ConsoleHealthWall) {
	<div id="console-health" hx-get="/console/health/fragment" hx-trigger="every 5s" hx-swap="outerHTML">
		@consoleHealthWall(w)
	</div>
}
```

Then change the page body in `ConsolePage` to use the polling wrapper instead of calling `consoleHealthWall` directly:

```go
				if data.Health != nil {
					@consoleHealthWallFragment(*data.Health)
				}
```

- [ ] **Step 4: Add the fragment handler** (in `graph/console.go`) and register it

```go
func (h *Handlers) handleConsoleHealthFragment(w http.ResponseWriter, r *http.Request) {
	proj, err := fetchHiveOperatorProjection(r)
	wall := buildConsoleHealthWall(proj, err, time.Now().UTC())
	consoleHealthWallFragment(wall).Render(r.Context(), w)
}
```

In `graph/handlers.go` `Register`:

```go
	mux.Handle("GET /console/health/fragment", h.writeWrap(h.handleConsoleHealthFragment))
```

- [ ] **Step 5: Generate, test, build**

Run: `templ generate && go test -count=1 ./graph/... && go build ./cmd/site/`
Expected: all pass, clean build.

- [ ] **Step 6: Commit**

```bash
git add graph/console.go graph/console.templ graph/console_templ.go graph/handlers.go graph/console_test.go
git commit -m "feat: poll-refresh the health wall fragment"
```

---

## Self-Review

**Spec coverage (Plan 1 scope = console shell + Health Wall, read-only):**
- Console shell + tab nav (Health/Kanban/Intake/Config) → Task 3. ✓
- Health Wall: status/freshness, agent roster, needs-you/approvals → Tasks 2, 4. ✓ (Per-agent live FSM state/cost/current-action is a telemetry enrichment deferred to a follow-on task within Phase 1 — the projection roster carries name/role/model today; this is scoped-out, not a placeholder.)
- Honest-staleness contract (current/stale/partial/unavailable, fail-closed) → Task 1 + rendered in Tasks 3–4. ✓
- Read-only boundary (no governed writes) → enforced; only reads. ✓
- Live refresh → Task 6 (HTMX polling; SSE deferred per spec open question). ✓
- BFF reuse of existing `fetchHiveOperatorProjection` → Tasks 2, 5. ✓

**Not in this plan (subsequent Phase 1 plans):** Kanban + shared order card/details drawer (Plan 2), Intake draft flow (Plan 3), Config read-only matrix (Plan 4). Each depends only on the shell + freshness machine delivered here.

**Placeholder scan:** Two grounded "reuse existing" notes (the `profile` import path and the hashed-CSS helper) cite where to copy from and provide a working fallback — not open-ended TODOs. No other placeholders.

**Type consistency:** `ConsoleFreshness`, `ConsoleHealthWall`, `ConsoleAgentRow`, `ConsoleApproval`, `ConsolePageData` are defined once (Tasks 1–3) and consumed with identical names/fields in Tasks 4–6. Upstream types (`OpsHiveProjection`, `OpsHiveApproval`, `OpsHiveRuntimeAgent`) match `graph/ops.go` exactly (verified field names: `GeneratedAt`, `PendingApprovals`, `RuntimeEvidence.AgentEvents.{ObservedActive,ActiveAgents}`, agent `{Name,Role,Model}`, approval `{RequestID,ActionName,Target,RiskSummary,CreatedAt}`).
