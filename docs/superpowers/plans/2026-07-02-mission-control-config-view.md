# Mission Control Config View Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the read-only `/console/config` tab to Mission Control, rendering the hive model-routing projection (catalog + per-role→model assignments + default-vs-override provenance) with honest staleness and zero write affordances.

**Architecture:** A Build-1 clone on the routing projection. `buildConsoleConfig(proj, err, now)` maps the already-fetched `OpsHiveProjection.ModelSelection` (graph/ops.go:729, fetched via `fetchHiveOperatorProjection`, graph/ops.go:4307) into a fail-closed view-model using `deriveFreshness` (graph/console.go:327). Two GET handlers mirror the intake pair (graph/console.go:188), registered in BOTH `Register` and `RegisterReadOnlyConsole`. Provenance reuses the /ops observatory helpers (`obsAssignmentModelModeState` graph/observatory.go:823, `obsHiveProjectionModelModeState` graph/observatory.go:855) — never reimplemented. The governed write `POST /ops/hive/model-policy` (graph/handlers.go:375 → `handleOpsHiveModelPolicySubmit`) is EXCLUDED: no POST route, no form, no write control renders.

**Tech Stack:** Go 1.26 net/http, templ (run `templ generate` after every `.templ` edit; generated `*_templ.go` is committed), HTMX 10s polling fragments, Tailwind classes matching the existing console surfaces. Tests: stdlib `testing` + `httptest`, no DB (`newConsoleTestHandlers`, graph/console_test.go:17).

## Global Constraints

- Repo: `/Transpara/transpara-ai/repos/site` (transpara-ai org only; never touch lovyou-ai/upstream).
- Branch: `feat/mission-control-config-view` off `main` @ `e6b4a1a`. NEVER commit to main. Never push unless asked.
- Read-only boundary: NO POST/PUT/DELETE route, no `<form>`, no `hx-post`, no `<input>`, no `<select>`, no `<button>` anywhere in the Config surface. Editing stays on the governed /ops surface (deferred here) — say so honestly in copy.
- Honest staleness / no fabrication: nil/failed/empty/timestamp-less projections render an explicit `unavailable` state with a human-readable notice; never invented routing data. Fail closed by allowlist: only a selection carrying actual data (`Source != "" || len(Models) > 0 || len(Assignments) > 0`) renders as data.
- NO retirement of the /ops model surface tonight (it hosts the governed write).
- Conventional commits, lowercase imperative subject. Commit generated `console_templ.go` alongside its `.templ` source.
- Verify suite (run from repo root): `templ generate && go build ./... && go vet ./... && go test ./graph/...`

---

### Task 1: Branch + Config view-model and builder

**Files:**
- Create: `graph/console_config.go`
- Test: `graph/console_config_test.go`

**Interfaces:**
- Consumes: `OpsHiveProjection`, `OpsHiveModelSelection`, `OpsHiveModelCatalogEntry`, `OpsHiveModelRoleAssignment` (graph/ops.go:562, 729, 743, 755); `deriveFreshness`, `ConsoleFreshness`, `consoleStaleWindow` (graph/console.go).
- Produces: `type ConsoleConfig struct { Freshness ConsoleFreshness; GeneratedAt string; Selection OpsHiveModelSelection; Notices []string }` and `func buildConsoleConfig(proj *OpsHiveProjection, fetchErr error, now time.Time) ConsoleConfig`. Task 2 renders `ConsoleConfig` and calls the builder from handlers; Task 2's helpers `consoleConfigAssignmentModel`, `consoleConfigAssignmentProvider`, `consoleConfigAssignmentMode`, `consoleConfigGlobalMode` also live in `graph/console_config.go`.

- [ ] **Step 1: Create the feature branch**

```bash
cd /Transpara/transpara-ai/repos/site
git switch main && git pull --ff-only origin main
git log -1 --oneline   # expect: e6b4a1a Merge pull request #200 ...
git switch -c feat/mission-control-config-view
```

- [ ] **Step 2: Write the failing builder tests**

Create `graph/console_config_test.go`:

```go
package graph

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// testConfigModelSelection returns a populated model-selection projection:
// two catalog models and two role assignments with distinct, deterministic
// provenance paths (policy-event override vs plain-model inferred manual).
func testConfigModelSelection() OpsHiveModelSelection {
	return OpsHiveModelSelection{
		Source:        "hive-operator-projection",
		CatalogSource: "catalog-mixed.yaml",
		Models: []OpsHiveModelCatalogEntry{
			{ID: "claude-opus-4-6", Provider: "claude-cli", AuthMode: "subscription", Tier: "judgment"},
			{ID: "gpt-5.5", Provider: "codex-cli", AuthMode: "subscription", Tier: "execution"},
		},
		Assignments: []OpsHiveModelRoleAssignment{
			// Policy-event assignment → obsAssignmentModelModeState = ("Manual", "override").
			{Role: "strategist", Model: "claude-opus-4-6", Provider: "claude-cli", Source: "hive-model-policy-event", PolicyEventID: "evt_123"},
			// Plain resolved model, no mode metadata → ("Manual", "inferred").
			{Role: "implementer", Model: "gpt-5.5", Provider: "codex-cli"},
		},
	}
}

func TestBuildConsoleConfig(t *testing.T) {
	now := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)

	t.Run("fetch error is unavailable with notice and no invented routing", func(t *testing.T) {
		cfg := buildConsoleConfig(nil, errors.New("HIVE_OPS_API_BASE_URL is not configured"), now)
		if cfg.Freshness != FreshnessUnavailable {
			t.Fatalf("freshness = %q, want unavailable", cfg.Freshness)
		}
		if len(cfg.Notices) == 0 {
			t.Fatal("expected a notice explaining the unavailable state")
		}
		if len(cfg.Selection.Models) != 0 || len(cfg.Selection.Assignments) != 0 {
			t.Fatal("unavailable config must not invent catalog or assignments")
		}
	})

	t.Run("nil projection is unavailable", func(t *testing.T) {
		cfg := buildConsoleConfig(nil, nil, now)
		if cfg.Freshness != FreshnessUnavailable || len(cfg.Notices) == 0 {
			t.Fatalf("nil projection: freshness = %q, notices = %v; want unavailable with notice", cfg.Freshness, cfg.Notices)
		}
	})

	t.Run("fresh projection with EMPTY model selection fails closed", func(t *testing.T) {
		// A fresh timestamp with no model-selection data must NOT render as a
		// current-but-empty surface; only a selection carrying data earns rendering.
		proj := &OpsHiveProjection{GeneratedAt: now.Add(-2 * time.Second).Format(time.RFC3339)}
		cfg := buildConsoleConfig(proj, nil, now)
		if cfg.Freshness != FreshnessUnavailable {
			t.Fatalf("freshness = %q, want unavailable for empty selection", cfg.Freshness)
		}
		if len(cfg.Notices) == 0 || !strings.Contains(strings.Join(cfg.Notices, "; "), "no model-selection") {
			t.Fatalf("empty selection must carry an explicit notice, got %v", cfg.Notices)
		}
	})

	t.Run("populated fresh projection is current and passes selection through", func(t *testing.T) {
		proj := &OpsHiveProjection{
			GeneratedAt:    now.Add(-2 * time.Second).Format(time.RFC3339),
			ModelSelection: testConfigModelSelection(),
		}
		cfg := buildConsoleConfig(proj, nil, now)
		if cfg.Freshness != FreshnessCurrent {
			t.Fatalf("freshness = %q, want current", cfg.Freshness)
		}
		if len(cfg.Selection.Models) != 2 || len(cfg.Selection.Assignments) != 2 {
			t.Fatalf("selection not passed through: %d models, %d assignments", len(cfg.Selection.Models), len(cfg.Selection.Assignments))
		}
	})

	t.Run("stale timestamp is stale", func(t *testing.T) {
		proj := &OpsHiveProjection{
			GeneratedAt:    now.Add(-2 * time.Minute).Format(time.RFC3339), // older than consoleStaleWindow (30s)
			ModelSelection: testConfigModelSelection(),
		}
		if cfg := buildConsoleConfig(proj, nil, now); cfg.Freshness != FreshnessStale {
			t.Fatalf("freshness = %q, want stale", cfg.Freshness)
		}
	})

	t.Run("projection errors downgrade fresh data to partial", func(t *testing.T) {
		proj := &OpsHiveProjection{
			GeneratedAt:    now.Add(-2 * time.Second).Format(time.RFC3339),
			ModelSelection: testConfigModelSelection(),
			Errors:         []string{"telemetry source degraded"},
		}
		cfg := buildConsoleConfig(proj, nil, now)
		if cfg.Freshness != FreshnessPartial {
			t.Fatalf("freshness = %q, want partial", cfg.Freshness)
		}
		if len(cfg.Notices) == 0 || cfg.Notices[0] != "telemetry source degraded" {
			t.Fatalf("projection errors must surface as notices, got %v", cfg.Notices)
		}
	})

	t.Run("model-selection errors also downgrade to partial", func(t *testing.T) {
		sel := testConfigModelSelection()
		sel.Errors = []string{"catalog reload failed"}
		proj := &OpsHiveProjection{
			GeneratedAt:    now.Add(-2 * time.Second).Format(time.RFC3339),
			ModelSelection: sel,
		}
		cfg := buildConsoleConfig(proj, nil, now)
		if cfg.Freshness != FreshnessPartial {
			t.Fatalf("freshness = %q, want partial for selection errors", cfg.Freshness)
		}
	})

	t.Run("populated selection with missing timestamp is unavailable with notice", func(t *testing.T) {
		proj := &OpsHiveProjection{ModelSelection: testConfigModelSelection()} // GeneratedAt empty
		cfg := buildConsoleConfig(proj, nil, now)
		if cfg.Freshness != FreshnessUnavailable {
			t.Fatalf("freshness = %q, want unavailable for missing timestamp", cfg.Freshness)
		}
		if len(cfg.Notices) == 0 {
			t.Fatal("timestamp-less unavailable state must carry an explicit notice")
		}
	})
}
```

- [ ] **Step 3: Run the tests to verify they fail**

```bash
go test ./graph/ -run TestBuildConsoleConfig -v
```

Expected: FAIL (compile error: `undefined: buildConsoleConfig` / `undefined: ConsoleConfig`).

- [ ] **Step 4: Write the builder**

Create `graph/console_config.go`:

```go
package graph

import (
	"net/http"
	"strings"
	"time"
)

// ConsoleConfig is the Config-tab view-model: the hive model-routing
// projection (catalog + per-role assignments + provenance) plus an explicit
// freshness state. It fails closed — a nil, failed, empty, or timestamp-less
// projection renders as unavailable with a human-readable notice and no
// invented routing data.
type ConsoleConfig struct {
	Freshness   ConsoleFreshness
	GeneratedAt string
	Selection   OpsHiveModelSelection
	Notices     []string
}

// buildConsoleConfig maps the hive operator projection (or a fetch error)
// into the Config view-model. Fail-closed by allowlist: only a model
// selection carrying actual data (source, models, or assignments) renders as
// data; an empty selection — even on a fresh projection — resolves to
// unavailable rather than a current-but-empty surface that could read as
// "hive routes nothing".
func buildConsoleConfig(proj *OpsHiveProjection, fetchErr error, now time.Time) ConsoleConfig {
	if fetchErr != nil || proj == nil {
		reason := "operator projection unavailable"
		if fetchErr != nil {
			reason = fetchErr.Error()
		}
		return ConsoleConfig{Freshness: FreshnessUnavailable, Notices: []string{reason}}
	}
	sel := proj.ModelSelection
	notices := append([]string(nil), proj.Errors...)
	notices = append(notices, sel.Errors...)
	if sel.Source == "" && len(sel.Models) == 0 && len(sel.Assignments) == 0 {
		return ConsoleConfig{
			Freshness: FreshnessUnavailable,
			Notices:   append(notices, "hive operator projection carries no model-selection data"),
		}
	}
	freshness := deriveFreshness(proj.GeneratedAt, nil, len(notices) > 0, now, consoleStaleWindow)
	if freshness == FreshnessUnavailable && len(notices) == 0 {
		notices = []string{"operator projection timestamp missing or unparseable"}
	}
	return ConsoleConfig{
		Freshness:   freshness,
		GeneratedAt: proj.GeneratedAt,
		Selection:   sel,
		Notices:     notices,
	}
}

func (h *Handlers) handleConsoleConfig(w http.ResponseWriter, r *http.Request) {
	proj, err := fetchHiveOperatorProjection(r)
	cfg := buildConsoleConfig(proj, err, time.Now().UTC())
	h.renderConsole(w, r, ConsolePageData{Title: "Config", Active: "config", Config: &cfg})
}

func (h *Handlers) handleConsoleConfigFragment(w http.ResponseWriter, r *http.Request) {
	proj, err := fetchHiveOperatorProjection(r)
	cfg := buildConsoleConfig(proj, err, time.Now().UTC())
	consoleConfigFragment(cfg).Render(r.Context(), w)
}

// consoleConfigAssignmentModel prefers the resolved model, then the policy
// model, then an honest placeholder — never an empty cell that reads as data.
func consoleConfigAssignmentModel(item OpsHiveModelRoleAssignment) string {
	if m := strings.TrimSpace(item.Model); m != "" {
		return m
	}
	if m := strings.TrimSpace(item.PolicyModel); m != "" {
		return m
	}
	return "not projected"
}

func consoleConfigAssignmentProvider(item OpsHiveModelRoleAssignment) string {
	if p := strings.TrimSpace(item.Provider); p != "" {
		return p
	}
	if p := strings.TrimSpace(item.PolicyProvider); p != "" {
		return p
	}
	return "not projected"
}

// consoleConfigAssignmentMode renders "Mode · provenance" for a role card,
// reusing the /ops observatory derivation verbatim so /console and /ops can
// never disagree about why a role routes where it does.
func consoleConfigAssignmentMode(sel OpsHiveModelSelection, item OpsHiveModelRoleAssignment) string {
	mode, provenance := obsAssignmentModelModeState(sel, item)
	return mode + " · " + provenance
}

func consoleConfigGlobalMode(sel OpsHiveModelSelection) string {
	mode, provenance, _ := obsHiveProjectionModelModeState(sel)
	return mode + " · " + provenance
}
```

NOTE: this file will not compile until Task 2 adds `ConsolePageData.Config` and the `consoleConfigFragment` templ component. For Task 1's commit, keep ONLY the type, the builder, and the four `consoleConfig*` string helpers in the file — add the two handlers in Task 2. (The helpers compile standalone; the handlers do not.)

- [ ] **Step 5: Run the tests to verify they pass**

```bash
go test ./graph/ -run TestBuildConsoleConfig -v
```

Expected: PASS (all 8 subtests).

- [ ] **Step 6: Commit**

```bash
git add graph/console_config.go graph/console_config_test.go
git commit -m "feat: add fail-closed config view-model for mission control"
```

---

### Task 2: Config templates, handlers, routes, and tab enable

**Files:**
- Modify: `graph/console.go:12-18` (add `Config` field to `ConsolePageData`)
- Modify: `graph/console.templ:27` (enable tab) and `graph/console.templ:36-39` (render branch), plus new templ components at end of file
- Modify: `graph/console_config.go` (add the two handlers from Task 1's listing)
- Modify: `graph/handlers.go:399` and `graph/handlers.go:439` (register routes in BOTH registrars)
- Generated: `graph/console_templ.go` (via `templ generate` — committed, never hand-edited)
- Test: `graph/console_config_test.go` (append)

**Interfaces:**
- Consumes: `ConsoleConfig`, `buildConsoleConfig`, `consoleConfigAssignmentModel`, `consoleConfigAssignmentProvider`, `consoleConfigAssignmentMode`, `consoleConfigGlobalMode` (Task 1); `consoleFreshnessBadge`, `noticeText`, `orFallback`, `newConsoleTestHandlers` (existing).
- Produces: `templ consoleConfigFragment(c ConsoleConfig)` (10s-poll wrapper, id `console-config`), `templ consoleConfig(c ConsoleConfig)` (the section); routes `GET /console/config` and `GET /console/config/fragment` in both `Register` and `RegisterReadOnlyConsole`. Task 3's guard tests render these components.

- [ ] **Step 1: Write the failing handler/route/tab tests**

Append to `graph/console_config_test.go` (add imports `"encoding/json"`, `"net/http"`, `"net/http/httptest"` to the existing import block):

```go
// newConfigHiveServer serves the operator projection with a populated model
// selection, mirroring the health-wall live-upstream test pattern.
func newConfigHiveServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hive/operator-projection" {
			http.NotFound(w, r)
			return
		}
		proj := OpsHiveProjection{
			GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
			ModelSelection: testConfigModelSelection(),
		}
		if err := json.NewEncoder(w).Encode(proj); err != nil {
			t.Errorf("encode projection: %v", err)
		}
	}))
}

func TestConsoleConfigRendersModelRouting(t *testing.T) {
	srv := newConfigHiveServer(t)
	defer srv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/config", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	// Catalog entries, role assignments, provenance, and catalog source all render.
	for _, want := range []string{
		"claude-opus-4-6", "gpt-5.5", // catalog + assignment models
		"strategist", "implementer", // roles
		"Manual · override",  // policy-event provenance (strategist)
		"Manual · inferred",  // plain-model provenance (implementer)
		"catalog-mixed.yaml", // catalog source
	} {
		if !strings.Contains(body, want) {
			t.Errorf("config surface missing %q", want)
		}
	}
}

func TestConsoleConfigTabEnabled(t *testing.T) {
	t.Setenv("HIVE_OPS_API_BASE_URL", "")
	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/config", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()
	// The enabled Config tab is an anchor to /console/config, not a disabled span.
	if !strings.Contains(body, `href="/console/config"`) {
		t.Error("Config tab must be enabled (anchor to /console/config)")
	}
	if strings.Contains(body, `title="coming soon"`) {
		t.Error("no console tab may remain in the disabled coming-soon state")
	}
}

func TestConsoleConfigUnavailableWhenProjectionAbsent(t *testing.T) {
	t.Setenv("HIVE_OPS_API_BASE_URL", "") // no upstream configured -> fetch error

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/config", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "unavailable") {
		t.Error("absent projection must render an explicit unavailable state")
	}
	// No fabricated routing data may appear.
	if strings.Contains(body, "claude-opus-4-6") || strings.Contains(body, "strategist") {
		t.Error("unavailable config must not fabricate routing rows")
	}
}

func TestConsoleConfigFragmentIsShellFree(t *testing.T) {
	t.Setenv("HIVE_OPS_API_BASE_URL", "")
	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/config/fragment", nil)
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

func TestConsoleConfigRoutesInReadOnlyRegistrar(t *testing.T) {
	// Both registrars must serve the config routes: Register (full site) and
	// RegisterReadOnlyConsole (no-DB console deployment).
	t.Setenv("HIVE_OPS_API_BASE_URL", "")
	h := NewHandlers(nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterReadOnlyConsole(mux)

	for _, path := range []string{"/console/config", "/console/config/fragment"} {
		req := httptest.NewRequest(http.MethodGet, "http://site.test"+path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200 in read-only registrar", path, w.Code)
		}
		if !strings.Contains(w.Body.String(), "unavailable") {
			t.Errorf("%s: no-DB console must render explicit unavailable state", path)
		}
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test ./graph/ -run 'TestConsoleConfig' -v
```

Expected: FAIL — first as a 404 on `/console/config` (route not registered), or a compile error once handlers reference the not-yet-generated `consoleConfigFragment`.

- [ ] **Step 3: Add the `Config` field to `ConsolePageData`**

In `graph/console.go`, change:

```go
type ConsolePageData struct {
	Title     string
	Active    string // health | kanban | intake | config
	Health    *ConsoleHealthWall
	Kanban    *ConsoleKanban
	IssueScan *ConsoleIssueScan
}
```

to:

```go
type ConsolePageData struct {
	Title     string
	Active    string // health | kanban | intake | config
	Health    *ConsoleHealthWall
	Kanban    *ConsoleKanban
	IssueScan *ConsoleIssueScan
	Config    *ConsoleConfig
}
```

- [ ] **Step 4: Enable the tab and add the render branch in `graph/console.templ`**

Change line 27 from:

```templ
					@consoleTab("config", "Config", "/console/config", data.Active, false)
```

to:

```templ
					@consoleTab("config", "Config", "/console/config", data.Active, true)
```

After the `data.IssueScan` block (lines 36–39), add:

```templ
					if data.Config != nil {
						@consoleConfigFragment(*data.Config)
					}
```

- [ ] **Step 5: Add the Config templ components at the end of `graph/console.templ`**

```templ
templ consoleConfigFragment(c ConsoleConfig) {
	<div id="console-config" hx-get="/console/config/fragment" hx-trigger="every 10s" hx-swap="outerHTML">
		@consoleConfig(c)
	</div>
}

// consoleConfig renders the read-only model-routing surface: catalog,
// role→model assignments, and default-vs-override provenance. It carries NO
// write affordances — no form, no input, no button. The governed write
// (POST /ops/hive/model-policy) intentionally stays on the /ops surface.
templ consoleConfig(c ConsoleConfig) {
	<section class="space-y-4" data-console-surface="config">
		<div class="flex items-center justify-between gap-3">
			<div>
				<h2 class="text-lg font-medium text-warm">Config — model routing</h2>
				<p class="text-xs text-warm-muted mt-1">read-only projection · policy editing is governed and lives on the /ops surface</p>
			</div>
			@consoleFreshnessBadge(c.Freshness, c.GeneratedAt)
		</div>
		if c.Freshness == FreshnessUnavailable {
			<p class="text-sm text-warm-muted" data-state="unavailable">unavailable — { noticeText(c.Notices) }</p>
		} else {
			if len(c.Notices) > 0 {
				<div class="border border-amber-500/40 bg-surface rounded-lg p-3 text-xs text-amber-300">
					for _, n := range c.Notices {
						<p>{ n }</p>
					}
				</div>
			}
			<div class="grid grid-cols-2 md:grid-cols-4 gap-3">
				<div class="border border-edge bg-surface rounded-lg p-4">
					<p class="text-xs text-warm-muted">Catalog models</p>
					<p class="text-2xl text-warm">{ strconv.Itoa(len(c.Selection.Models)) }</p>
				</div>
				<div class="border border-edge bg-surface rounded-lg p-4">
					<p class="text-xs text-warm-muted">Role assignments</p>
					<p class="text-2xl text-warm">{ strconv.Itoa(len(c.Selection.Assignments)) }</p>
				</div>
				<div class="border border-edge bg-surface rounded-lg p-4">
					<p class="text-xs text-warm-muted">Global mode</p>
					<p class="text-sm text-warm mt-2">{ consoleConfigGlobalMode(c.Selection) }</p>
				</div>
				<div class="border border-edge bg-surface rounded-lg p-4">
					<p class="text-xs text-warm-muted">Catalog source</p>
					<p class="text-xs text-warm mt-2 break-all">{ orFallback(c.Selection.CatalogSource, "not projected") }</p>
				</div>
			</div>
			<div class="border border-edge bg-surface rounded-lg overflow-hidden">
				<div class="px-4 py-2 border-b border-edge text-sm font-medium text-warm">Role → model routing</div>
				if len(c.Selection.Assignments) == 0 {
					<div class="px-4 py-3 text-sm text-warm-muted">No role assignments projected.</div>
				} else {
					for _, item := range c.Selection.Assignments {
						<div class="px-4 py-2 border-b border-edge last:border-0">
							<div class="flex flex-wrap items-center gap-x-4 gap-y-1">
								<span class="text-sm font-medium text-warm w-32 truncate">{ orFallback(item.Role, "role not projected") }</span>
								<span class="text-xs text-warm font-mono break-all">{ consoleConfigAssignmentModel(item) }</span>
								<span class="text-xs text-warm-muted">{ consoleConfigAssignmentProvider(item) }</span>
								<span class="text-xs text-warm-muted ml-auto">{ consoleConfigAssignmentMode(c.Selection, item) }</span>
							</div>
							if item.Error != "" {
								<p class="text-xs text-amber-300 mt-1 break-words">{ item.Error }</p>
							}
						</div>
					}
				}
			</div>
			<div class="border border-edge bg-surface rounded-lg overflow-hidden">
				<div class="px-4 py-2 border-b border-edge text-sm font-medium text-warm">Model catalog</div>
				if len(c.Selection.Models) == 0 {
					<div class="px-4 py-3 text-sm text-warm-muted">No catalog models projected.</div>
				} else {
					for _, m := range c.Selection.Models {
						<div class="flex flex-wrap items-center gap-x-4 gap-y-1 px-4 py-2 border-b border-edge last:border-0">
							<span class="text-sm text-warm font-mono break-all">{ orFallback(m.ID, "id not projected") }</span>
							<span class="text-xs text-warm-muted">{ orFallback(m.Provider, "provider not projected") }</span>
							<span class="text-xs text-warm-muted">{ orFallback(m.AuthMode, "auth not projected") }</span>
							<span class="text-xs text-warm-muted">{ orFallback(m.Tier, "tier not projected") }</span>
							if m.Deprecated {
								<span class="text-xs text-amber-300 ml-auto">deprecated</span>
							}
						</div>
					}
				}
			</div>
			<p class="text-xs text-warm-muted">Model policy is hive-owned. This console renders the routing projection only; it never persists or forwards changes.</p>
		}
	</section>
}
```

- [ ] **Step 6: Add the two handlers to `graph/console_config.go`**

Append (they were listed in Task 1 Step 4 but deferred to compile):

```go
func (h *Handlers) handleConsoleConfig(w http.ResponseWriter, r *http.Request) {
	proj, err := fetchHiveOperatorProjection(r)
	cfg := buildConsoleConfig(proj, err, time.Now().UTC())
	h.renderConsole(w, r, ConsolePageData{Title: "Config", Active: "config", Config: &cfg})
}

func (h *Handlers) handleConsoleConfigFragment(w http.ResponseWriter, r *http.Request) {
	proj, err := fetchHiveOperatorProjection(r)
	cfg := buildConsoleConfig(proj, err, time.Now().UTC())
	consoleConfigFragment(cfg).Render(r.Context(), w)
}
```

Add `"net/http"` to the file's imports.

- [ ] **Step 7: Register the routes in BOTH registrars in `graph/handlers.go`**

In `Register`, directly after line 399 (`GET /console/intake/card`):

```go
	mux.Handle("GET /console/config", h.writeWrap(h.handleConsoleConfig))
	mux.Handle("GET /console/config/fragment", h.writeWrap(h.handleConsoleConfigFragment))
```

In `RegisterReadOnlyConsole`, directly after line 439 (`GET /console/intake/card`):

```go
	mux.HandleFunc("GET /console/config", h.handleConsoleConfig)
	mux.HandleFunc("GET /console/config/fragment", h.handleConsoleConfigFragment)
```

Do NOT add any POST route. The governed write `POST /ops/hive/model-policy` (graph/handlers.go:375) stays exactly where it is, untouched.

- [ ] **Step 8: Generate templates and run the tests**

```bash
templ generate
go test ./graph/ -run 'TestConsoleConfig|TestBuildConsoleConfig' -v
```

Expected: PASS (all Task 1 + Task 2 tests).

- [ ] **Step 9: Build and vet the whole module**

```bash
go build ./... && go vet ./...
```

Expected: clean exit, no output.

- [ ] **Step 10: Commit (including generated code)**

```bash
git add graph/console.go graph/console.templ graph/console_templ.go graph/console_config.go graph/console_config_test.go graph/handlers.go
git diff --cached --stat   # verify ONLY these six files changed
git commit -m "feat: enable read-only mission control config tab on routing projection"
```

---

### Task 3: Read-only boundary and hostile-projection guard tests

These are characterization guards: if Task 2's template is correct they pass immediately; a failure means the template (not the test) must be fixed.

**Files:**
- Test: `graph/console_config_test.go` (append)
- Possibly modify: `graph/console.templ` + regenerate, only if a guard fails

**Interfaces:**
- Consumes: `consoleConfig`, `consoleConfigFragment` templ components (Task 2), `buildConsoleConfig`, `testConfigModelSelection`, `newConfigHiveServer` (Tasks 1–2).
- Produces: nothing new — regression guards only.

- [ ] **Step 1: Write the read-only boundary test**

Append to `graph/console_config_test.go` (add imports `"bytes"`, `"context"`):

```go
func TestConsoleConfigRendersNoWriteControls(t *testing.T) {
	// READ-ONLY BOUNDARY: the config surface must carry zero write affordances,
	// even fully populated. The governed write (POST /ops/hive/model-policy)
	// lives on /ops and must not leak here in any form.
	srv := newConfigHiveServer(t)
	defer srv.Close()
	t.Setenv("HIVE_OPS_API_BASE_URL", srv.URL)

	h := newConsoleTestHandlers()
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "http://site.test/console/config", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()
	for _, forbidden := range []string{
		"<form", "hx-post", "hx-put", "hx-delete", "hx-patch",
		"<input", "<select", "<textarea", "<button",
		"/ops/hive/model-policy",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("read-only config surface must not render %q", forbidden)
		}
	}
}

func TestConsoleConfigWriteRouteNotRegistered(t *testing.T) {
	// Belt and braces: no POST route may answer under /console/config in either
	// registrar. ServeMux returns 405 for a method-mismatched pattern — the
	// requirement is only that a POST can never succeed.
	t.Setenv("HIVE_OPS_API_BASE_URL", "")
	for name, register := range map[string]func(h *Handlers, mux *http.ServeMux){
		"Register":                func(h *Handlers, mux *http.ServeMux) { h.Register(mux) },
		"RegisterReadOnlyConsole": func(h *Handlers, mux *http.ServeMux) { h.RegisterReadOnlyConsole(mux) },
	} {
		h := NewHandlers(nil, nil, nil)
		if name == "Register" {
			h = newConsoleTestHandlers()
		}
		mux := http.NewServeMux()
		register(h, mux)
		req := httptest.NewRequest(http.MethodPost, "http://site.test/console/config", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code == http.StatusOK {
			t.Errorf("%s: POST /console/config answered 200; write path must not exist", name)
		}
	}
}
```

- [ ] **Step 2: Write the hostile-projection escaping test**

Append:

```go
func TestConsoleConfigSurfaceEscapesHostileProjectionData(t *testing.T) {
	// templ's default { } interpolation HTML-escapes everything and this surface
	// uses no templ.Raw/SafeHTML — hostile operator-visible projection strings
	// must come out escaped. Mirrors the intake surface guard.
	hostile := OpsHiveModelSelection{
		Source:        "hive-operator-projection",
		CatalogSource: `<script>catalog()</script>`,
		Models: []OpsHiveModelCatalogEntry{
			{ID: `<script>alert('model')</script>`, Provider: `<img src=x onerror=y>prov`, AuthMode: "subscription", Tier: "judgment"},
		},
		Assignments: []OpsHiveModelRoleAssignment{
			{Role: `<button onclick="x">role</button>`, Model: "m", Error: `<form action="/hive">err</form>`},
		},
		Errors: []string{`<script>selerr()</script>`},
	}
	cfg := buildConsoleConfig(&OpsHiveProjection{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		ModelSelection: hostile,
	}, nil, time.Now().UTC())

	var buf bytes.Buffer
	if err := consoleConfig(cfg).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()

	for _, raw := range []string{
		`<script>alert('model')</script>`,
		`<script>catalog()</script>`,
		`<script>selerr()</script>`,
		`<button onclick="x">role</button>`,
		`<form action="/hive">err</form>`,
		"<img src=x onerror=y>prov",
	} {
		if strings.Contains(out, raw) {
			t.Errorf("hostile raw markup %q survived escaping in the config surface", raw)
		}
	}
	if !strings.Contains(out, "&lt;script") {
		t.Error("expected escaped form \"&lt;script\" in output; escaping did not occur (data may have vanished instead of being escaped)")
	}
}
```

- [ ] **Step 3: Run the guard tests**

```bash
go test ./graph/ -run 'TestConsoleConfig' -v
```

Expected: PASS. If a guard fails, fix `graph/console.templ` (never weaken the test), re-run `templ generate`, and re-test.

- [ ] **Step 4: Commit**

```bash
git add graph/console_config_test.go
git commit -m "test: guard config surface read-only boundary and escaping"
```

(Include `graph/console.templ` + `graph/console_templ.go` in the add only if Step 3 forced a template fix.)

---

### Task 4: Full verification and demo smoke

**Files:** none created — verification only.

**Interfaces:**
- Consumes: everything above.
- Produces: a verified, demoable branch build.

- [ ] **Step 1: Run the full verify suite**

```bash
cd /Transpara/transpara-ai/repos/site
templ generate && git status --short   # expect NO modified *_templ.go (generated code committed)
go build ./... && go vet ./... && go test ./graph/...
```

Expected: `ok github.com/transpara-ai/site/graph`, zero failures, clean git status apart from pre-existing untracked `.adversarial-design/` and `.visual-evidence/`.

- [ ] **Step 2: Demo smoke — serve and hit the new tab**

```bash
make run &
sleep 3
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/console/config   # expect 200
curl -s http://localhost:8080/console/config | grep -o "Config — model routing"
kill %1
```

Expected: `200` and the section heading. Without a live hive projection the surface honestly renders `unavailable — ...` — that is correct fail-closed behavior, not a bug. For tonight's richer demo picture, run the hive with `--catalog catalog-mixed.yaml` (file lives at `../hive/catalog-mixed.yaml`) and point `HIVE_OPS_API_BASE_URL` at it so the Claude/Codex/Ollama/OpenRouter routing renders live. (`make run` needs the usual site env — `DATABASE_URL` etc. — per repo CLAUDE.md.)

- [ ] **Step 3: Final commit check**

```bash
git log --oneline main..HEAD   # expect the 3 task commits
git diff main --stat           # expect only: console.go, console.templ, console_templ.go, console_config.go, console_config_test.go, handlers.go (+ this plan file if committed)
```

---

## Self-Review (completed at plan time)

- **Spec coverage:** design points 1–7 from the approved checkpoint each map to a task — (1) projection reuse → Task 1 builder consumes `OpsHiveProjection.ModelSelection`; (2) honest staleness via `deriveFreshness` → Task 1; (3) handlers + templates mirroring intake → Task 2; (4) read-only boundary, no POST wired → Task 2 Step 7 + Task 3 guards; (5) tab enable + both registrars → Task 2 Steps 4/7; (6) no /ops retirement → explicitly out of scope (no task touches ops routes); (7) tests mirroring `console_intake_test.go` → Tasks 1–3.
- **Placeholder scan:** none — every code step carries complete code.
- **Type consistency:** `ConsoleConfig` / `buildConsoleConfig` / `consoleConfigFragment` / `consoleConfig` / `consoleConfigAssignmentModel` / `consoleConfigAssignmentProvider` / `consoleConfigAssignmentMode` / `consoleConfigGlobalMode` used identically across Tasks 1–3; provenance strings asserted in tests ("Manual · override", "Manual · inferred") match `obsAssignmentModelModeState` (graph/observatory.go:823-843) joined with `" · "` by `consoleConfigAssignmentMode`.
