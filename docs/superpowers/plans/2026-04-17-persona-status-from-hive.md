# Persona Status from Hive — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Populate site's `agent_personas.status` column from hive's `agents/*.md` files at build time via a git submodule + `go generate` pipeline, replacing the hand-maintained metadata propagation of PR #14.

**Architecture:** Hive becomes a git submodule at `third_party/hive`. A small Go generator (`cmd/gen-persona-status`) reads `agents/*.md`, extracts each `<!-- Status: X -->` comment, and writes `graph/personas/status_gen.go` containing `var HiveStatus = map[string]string{...}`. At boot, `SeedAgentPersonas` looks up each persona's status in `HiveStatus` and forces `Active=false` for `absorbed`/`retired`. The generated Go file is committed, so `go build` has no submodule dependency; only `go generate` does.

**Tech Stack:** Go 1.25, git submodules, `go generate`, PostgreSQL (`lib/pq`), GitHub Actions.

**Starting branch:** `feat/persona-status-from-hive` (already created from `transpara/main`, spec committed as `1cd993b`).

**Spec:** [`docs/superpowers/specs/2026-04-17-persona-submodule-generate-design.md`](../specs/2026-04-17-persona-submodule-generate-design.md).

---

## Task 1: Close PR #14 with superseding comment

**Files:** none (external GitHub action)

- [ ] **Step 1: Post closing comment and close the PR**

```bash
gh pr comment 14 --repo transpara-ai/site --body "Superseded by the forthcoming PR on \`feat/persona-status-from-hive\`. That branch replaces this PR's 54-file metadata propagation with a generated \`HiveStatus\` map sourced from hive at build time (git submodule + \`go generate\`). See the design spec committed on the new branch: \`docs/superpowers/specs/2026-04-17-persona-submodule-generate-design.md\`."
gh pr close 14 --repo transpara-ai/site
```

Expected output: a confirmation that PR #14 is closed.

- [ ] **Step 2: Verify the PR is closed and the branch still exists (we don't delete it — keeps history)**

```bash
gh pr view 14 --repo transpara-ai/site --json state,headRefName
```

Expected: `{"state":"CLOSED","headRefName":"feat/persona-status-sync"}`.

---

## Task 2: Add hive as a git submodule

**Files:**
- Create: `.gitmodules`
- Create: `third_party/hive/` (submodule checkout)
- Create: `.dockerignore`

- [ ] **Step 1: Add the submodule**

Run from repo root (`/Transpara/transpara-ai/data/repos/lovyou-ai-site`):

```bash
git submodule add https://github.com/transpara-ai/hive.git third_party/hive
```

Expected output: `Cloning into '.../third_party/hive'... done.` Creates `.gitmodules` and populates `third_party/hive/`.

- [ ] **Step 2: Verify `.gitmodules` content**

```bash
cat .gitmodules
```

Expected:
```
[submodule "third_party/hive"]
	path = third_party/hive
	url = https://github.com/transpara-ai/hive.git
```

- [ ] **Step 3: Confirm pinned commit and the presence of `agents/*.md`**

```bash
git -C third_party/hive rev-parse HEAD
ls third_party/hive/agents/*.md | head -5
```

Expected: a commit SHA and a listing of at least 5 `.md` files (treasurer.md, advocate.md, etc.).

- [ ] **Step 4: Create `.dockerignore` excluding `third_party/`**

```bash
cat > .dockerignore <<'EOF'
# Git submodule — the binary uses the generated status_gen.go, not the submodule.
third_party/
# Local dev state
.git
.vscode
*.log
EOF
```

- [ ] **Step 5: Commit submodule + .dockerignore**

```bash
git add .gitmodules third_party/hive .dockerignore
git commit -m "$(cat <<'EOF'
feat: add hive as git submodule at third_party/hive

Pins lovyou-ai-site to a specific transpara-ai/hive commit. The site
binary does not depend on this submodule at runtime; only `go generate`
reads it. .dockerignore excludes it from build images.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds. `git log --oneline -1` shows the new commit.

---

## Task 3: Write the generator using TDD

**Files:**
- Create: `cmd/gen-persona-status/main.go`
- Create: `cmd/gen-persona-status/main_test.go`

- [ ] **Step 1: Create the test file first (TDD)**

Write `cmd/gen-persona-status/main_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanExtractsStatusFromLeadingComment(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"treasurer.md": "<!-- Status: designed -->\n<!-- Council-2026-04-16: created from merger -->\n# Treasurer\n",
		"ceo.md":       "<!-- Status: retired -->\n# CEO\n",
		"builder.md":   "<!-- Status: running -->\n# Builder\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := scan(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}
	// Sorted alphabetically by name.
	if entries[0].Name != "builder" || entries[0].Status != "running" {
		t.Errorf("entries[0] = %+v, want {builder, running}", entries[0])
	}
	if entries[1].Name != "ceo" || entries[1].Status != "retired" {
		t.Errorf("entries[1] = %+v, want {ceo, retired}", entries[1])
	}
	if entries[2].Name != "treasurer" || entries[2].Status != "designed" {
		t.Errorf("entries[2] = %+v, want {treasurer, designed}", entries[2])
	}
}

func TestScanSkipsFilesWithoutStatusComment(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"treasurer.md": "<!-- Status: designed -->\n# Treasurer\n",
		"CONTEXT.md":   "# Shared Context\n\nNo status comment here.\n",
		"README.md":    "# Readme\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := scan(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1 (CONTEXT.md and README.md should be skipped)", len(entries))
	}
	if entries[0].Name != "treasurer" {
		t.Errorf("entries[0].Name = %q, want treasurer", entries[0].Name)
	}
}

func TestScanMissingDirHintsAtSubmoduleInit(t *testing.T) {
	_, err := scan("/nonexistent/hive/agents")
	if err == nil {
		t.Fatal("want error for missing dir")
	}
	if !strings.Contains(err.Error(), "git submodule update") {
		t.Errorf("error should hint at submodule init; got: %v", err)
	}
}

func TestRenderProducesDeterministicOutput(t *testing.T) {
	entries := []entry{
		{Name: "advocate", Status: "running"},
		{Name: "treasurer", Status: "designed"},
	}
	got, err := render(entries)
	if err != nil {
		t.Fatal(err)
	}
	want := `// Code generated by gen-persona-status; DO NOT EDIT.

package personas

// HiveStatus maps persona name → hive-declared lifecycle status.
// Source: third_party/hive/agents/<name>.md <!-- Status: X -->
var HiveStatus = map[string]string{
	"advocate":  "running",
	"treasurer": "designed",
}
`
	if string(got) != want {
		t.Errorf("render mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
```

- [ ] **Step 2: Run tests; verify they fail because `main` package has no `scan` / `render` / `entry`**

```bash
cd /Transpara/transpara-ai/data/repos/lovyou-ai-site
go test ./cmd/gen-persona-status/...
```

Expected: build error — `undefined: scan`, `undefined: render`, `undefined: entry`, etc.

- [ ] **Step 3: Implement `cmd/gen-persona-status/main.go`**

```go
// Command gen-persona-status reads hive agents/*.md files and emits
// graph/personas/status_gen.go containing HiveStatus = map[string]string.
//
// Invoked by //go:generate in graph/personas/generate.go. Not used at runtime.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var statusRegex = regexp.MustCompile(`<!--\s*Status:\s*(\w+)\s*-->`)

type entry struct {
	Name   string
	Status string
}

func main() {
	src := flag.String("src", "../../third_party/hive/agents", "directory of hive agents/*.md files")
	out := flag.String("out", "status_gen.go", "output Go file path")
	flag.Parse()

	entries, err := scan(*src)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	body, err := render(entries)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := os.WriteFile(*out, body, 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "gen-persona-status: wrote %s (%d personas)\n", *out, len(entries))
}

// scan walks dir, reads leading 20 lines of each *.md, extracts Status comment.
// Files with no Status comment are skipped (logged to stderr).
func scan(dir string) ([]entry, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w (try: git submodule update --init --recursive)", dir, err)
	}
	var out []entry
	for _, it := range items {
		if it.IsDir() || !strings.HasSuffix(it.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, it.Name())
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		head := leadingLines(body, 20)
		m := statusRegex.FindSubmatch(head)
		if m == nil {
			fmt.Fprintf(os.Stderr, "gen-persona-status: skip %s (no Status comment)\n", it.Name())
			continue
		}
		name := strings.TrimSuffix(it.Name(), ".md")
		out = append(out, entry{Name: name, Status: string(m[1])})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// leadingLines returns the first n lines of body.
func leadingLines(body []byte, n int) []byte {
	lines := bytes.SplitN(body, []byte("\n"), n+1)
	if len(lines) > n {
		return bytes.Join(lines[:n], []byte("\n"))
	}
	return body
}

var outputTemplate = template.Must(template.New("out").Parse(`// Code generated by gen-persona-status; DO NOT EDIT.

package personas

// HiveStatus maps persona name → hive-declared lifecycle status.
// Source: third_party/hive/agents/<name>.md <!-- Status: X -->
var HiveStatus = map[string]string{
{{- range .Entries }}
	{{ printf "%-*q" $.MaxName .Name }}: {{ printf "%q" .Status }},
{{- end }}
}
`))

// render emits the Go source for status_gen.go, passed through gofmt.
func render(entries []entry) ([]byte, error) {
	maxName := 0
	for _, e := range entries {
		q := fmt.Sprintf("%q", e.Name) // includes quotes
		if len(q) > maxName {
			maxName = len(q)
		}
	}
	var buf bytes.Buffer
	if err := outputTemplate.Execute(&buf, struct {
		Entries []entry
		MaxName int
	}{entries, maxName}); err != nil {
		return nil, err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Fall back to unformatted on gofmt failure so the error is visible.
		return buf.Bytes(), fmt.Errorf("gofmt: %w", err)
	}
	return formatted, nil
}
```

- [ ] **Step 4: Run tests; verify they pass**

```bash
go test -v ./cmd/gen-persona-status/...
```

Expected: all 4 tests PASS. If `TestRenderProducesDeterministicOutput` fails due to alignment spacing (the `%-*q` formatting makes columns line up), accept the actual output as the new `want` and update the test — gofmt may rewrite the exact spacing.

- [ ] **Step 5: Run `go vet` on the new package**

```bash
go vet ./cmd/gen-persona-status/...
```

Expected: no output (clean).

- [ ] **Step 6: Commit**

```bash
git add cmd/gen-persona-status/
git commit -m "$(cat <<'EOF'
feat: add gen-persona-status generator

Reads third_party/hive/agents/*.md, extracts leading
<!-- Status: X --> comments, emits graph/personas/status_gen.go
containing HiveStatus = map[string]string. Files without a Status
comment are skipped (e.g. CONTEXT.md). Invoked via go:generate;
not used at runtime.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Add `//go:generate` directive

**Files:**
- Create: `graph/personas/generate.go`

- [ ] **Step 1: Create directive file**

```go
// Package personas holds the embedded persona markdown files and the
// generated HiveStatus map (status_gen.go). Run `go generate ./graph/personas/...`
// after bumping the third_party/hive submodule.
package personas

//go:generate go run ../../cmd/gen-persona-status
```

Write to `graph/personas/generate.go`.

- [ ] **Step 2: Verify the package compiles (no `status_gen.go` yet means `HiveStatus` is not yet defined — that's fine, nothing references it yet)**

```bash
go build ./graph/personas/...
```

Expected: success (package has no other code, just the doc comment + directive).

- [ ] **Step 3: Commit**

```bash
git add graph/personas/generate.go
git commit -m "$(cat <<'EOF'
feat: add //go:generate directive for persona status

Invokes cmd/gen-persona-status to (re)generate status_gen.go from
the pinned third_party/hive submodule.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Run the generator and commit `status_gen.go`

**Files:**
- Create: `graph/personas/status_gen.go` (generated)

- [ ] **Step 1: Run `go generate`**

```bash
go generate ./graph/personas/...
```

Expected stderr output:
```
gen-persona-status: skip CONTEXT.md (no Status comment)
gen-persona-status: wrote status_gen.go (N personas)
```
where N is between 60 and 70 (the hive has ~69 agent files; CONTEXT.md is skipped).

- [ ] **Step 2: Inspect the generated file**

```bash
head -20 graph/personas/status_gen.go
```

Expected:
```go
// Code generated by gen-persona-status; DO NOT EDIT.

package personas

// HiveStatus maps persona name → hive-declared lifecycle status.
// Source: third_party/hive/agents/<name>.md <!-- Status: X -->
var HiveStatus = map[string]string{
	"advocate":  "running",
	"allocator": "designed",
	...
}
```

- [ ] **Step 3: Verify it builds**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 4: Commit the generated file**

```bash
git add graph/personas/status_gen.go
git commit -m "$(cat <<'EOF'
chore: add generated graph/personas/status_gen.go

Committed so `go build` has no submodule dependency. Regenerate
with `go generate ./graph/personas/...` after bumping the hive pin.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Re-land PR #14's `graph/store.go` changes

**Files:**
- Modify: `graph/store.go` (lines ~479, ~3836–3876, ~3935, ~3951, ~3974, ~3986, ~3996)

The store.go diff from `05a4fd3` is pure re-use. We `git checkout` that file version from the old branch, review, then commit.

- [ ] **Step 1: Pull store.go from the old branch**

```bash
git checkout feat/persona-status-sync -- graph/store.go
```

- [ ] **Step 2: Review the diff — confirm it matches the expected shape**

```bash
git diff --cached graph/store.go | head -100
```

Expected to see these changes (and no others):
1. Migration SQL block: `ALTER TABLE agent_personas ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'ready';`
2. `AgentPersona` struct gains `Status string \`json:"status"\`` and a doc comment explaining the seven status values.
3. `UpsertAgentPersona`: defaults blank status to `"ready"`, adds `status` column to INSERT and `EXCLUDED.status` to ON CONFLICT clause.
4. Three SELECT statements (`GetAgentPersonasForConversations`, `GetAgentPersona`, `ListAgentPersonas`) gain `ap.status` / `status` in the projection and the corresponding `&p.Status` in the Scan.

If anything else changed, abort and investigate — do not commit.

- [ ] **Step 3: Verify it builds (consumer code in `graph/personas.go` still uses old signature; build should pass because new `Status` field defaults to zero value)**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 4: Commit**

```bash
git add graph/store.go
git commit -m "$(cat <<'EOF'
feat(store): add Status field and column to agent_personas

Re-lands the store-side plumbing from closed PR #14:
- AgentPersona.Status string field
- ALTER TABLE agent_personas ADD COLUMN IF NOT EXISTS status
  TEXT NOT NULL DEFAULT 'ready' (idempotent)
- status threaded through all three SELECT statements and the UPSERT

Consumer changes (populating from HiveStatus) in a follow-up commit.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Wire `SeedAgentPersonas` to use `HiveStatus`

**Files:**
- Modify: `graph/personas.go` (the `SeedAgentPersonas` function)

The `parsePersonaFile` signature stays `(display, description string)` — do NOT add a status return. Status comes from `HiveStatus`, not from the `.md`.

- [ ] **Step 1: Edit `SeedAgentPersonas`**

In `graph/personas.go`, locate this block (currently around lines 140–150):

```go
		active := personaActive[name]

		if err := s.UpsertAgentPersona(ctx, AgentPersona{
			Name:        name,
			Display:     display,
			Description: description,
			Category:    category,
			Prompt:      string(data),
			Model:       model,
			Active:      active,
		}); err != nil {
```

Replace with:

```go
		status := HiveStatus[name]
		if status == "" {
			status = "ready"
		}

		active := personaActive[name]
		if status == "absorbed" || status == "retired" {
			active = false
		}

		if err := s.UpsertAgentPersona(ctx, AgentPersona{
			Name:        name,
			Display:     display,
			Description: description,
			Category:    category,
			Prompt:      string(data),
			Model:       model,
			Active:      active,
			Status:      status,
		}); err != nil {
```

- [ ] **Step 2: Verify the build**

```bash
go build ./...
```

Expected: success. `graph/personas.go` now references `HiveStatus` (defined in `graph/personas/status_gen.go`, but wait — that's a different package). Check the import path.

**Important:** `graph/personas.go` is in package `graph`; `status_gen.go` is in package `personas` (subdirectory `graph/personas/`). So the reference must be `personas.HiveStatus`, not bare `HiveStatus`. Add the import:

```go
import (
	"context"
	"embed"
	"log"
	"strings"

	"github.com/lovyou-ai/site/graph/personas"
)
```

And change the reference to `personas.HiveStatus[name]`.

Re-run:

```bash
go build ./...
```

Expected: success.

- [ ] **Step 3: Run existing tests (non-DB ones) to confirm no regressions**

```bash
go test -count=1 -short ./...
```

Expected: passes or `skip` on DB-requiring tests (`DATABASE_URL not set`).

- [ ] **Step 4: Commit**

```bash
git add graph/personas.go
git commit -m "$(cat <<'EOF'
feat(personas): populate Status from generated HiveStatus map

SeedAgentPersonas now looks up each persona's lifecycle status in
personas.HiveStatus (generated from third_party/hive at go generate
time). Defaults to "ready" for personas with no hive counterpart.
Forces Active=false when status ∈ {absorbed, retired}, matching the
behaviour of closed PR #14 but with hive as the single source of truth.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Add integration test for Status → Active=false

**Files:**
- Modify: `graph/store_test.go` (new test function at end of file)

- [ ] **Step 1: Add the test at the end of `graph/store_test.go`**

```go
// TestSeedAgentPersonas_StatusForcesInactive verifies that a persona whose
// HiveStatus is "absorbed" or "retired" is upserted with Active=false,
// regardless of what personaActive[name] says.
func TestSeedAgentPersonas_StatusForcesInactive(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	// Find any persona whose HiveStatus is absorbed or retired. If none exist
	// in the current pinned hive, the test is a no-op — log and skip.
	var targetName, targetStatus string
	for name, status := range personas.HiveStatus {
		if status == "absorbed" || status == "retired" {
			targetName, targetStatus = name, status
			break
		}
	}
	if targetName == "" {
		t.Skip("no absorbed/retired personas in current hive pin")
	}

	// Seeding runs against the embedded personas; ensure target is among them.
	if _, err := personaFiles.ReadFile("personas/" + targetName + ".md"); err != nil {
		t.Skipf("target persona %q has no embedded .md (skip): %v", targetName, err)
	}

	store.SeedAgentPersonas(ctx)

	got := store.GetAgentPersona(ctx, targetName)
	if got == nil {
		t.Fatalf("GetAgentPersona(%q) returned nil", targetName)
	}
	if got.Status != targetStatus {
		t.Errorf("Status = %q, want %q", got.Status, targetStatus)
	}
	if got.Active {
		t.Errorf("Active = true for %s persona %q, want false", targetStatus, targetName)
	}
}
```

Add the import at the top of `graph/store_test.go` if not already present:

```go
import (
	// ...existing imports...
	"github.com/lovyou-ai/site/graph/personas"
)
```

- [ ] **Step 2: Start the local test DB and run the test**

```bash
docker compose up -d
DATABASE_URL=postgres://site:site@localhost:5433/site?sslmode=disable go test -v -count=1 -run TestSeedAgentPersonas_StatusForcesInactive ./graph/
```

Expected: PASS, or SKIP with message "no absorbed/retired personas in current hive pin" (acceptable — the test is conditional on hive content).

If PASS, note which persona was tested — e.g. `treasurer` (designed, so wouldn't match) vs one actually `retired`. If no personas currently have retired/absorbed status, the test skips; bump the hive pin when one appears.

- [ ] **Step 3: Commit**

```bash
git add graph/store_test.go
git commit -m "$(cat <<'EOF'
test(personas): assert HiveStatus absorbed/retired forces Active=false

Scans the generated HiveStatus map for a target persona with status
absorbed or retired, seeds, then verifies the seeded row is inactive.
Skips gracefully if the current hive pin has no such personas.

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Update CI workflow

**Files:**
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Edit the workflow**

Change the `actions/checkout` block from:

```yaml
      - uses: actions/checkout@v4
```

to:

```yaml
      - uses: actions/checkout@v4
        with:
          submodules: recursive
```

Insert a new step between "Install templ" and "Generate templates":

```yaml
      - name: Generate persona status
        run: go generate ./graph/personas/...
```

Change the diff-check step from:

```yaml
      - name: Check for uncommitted generated file changes
        run: git diff --exit-code -- '*_templ.go'
```

to:

```yaml
      - name: Check for uncommitted generated file changes
        run: git diff --exit-code -- '*_templ.go' '*_gen.go'
```

- [ ] **Step 2: Sanity-check the YAML**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"
```

Expected: no output (YAML valid).

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "$(cat <<'EOF'
ci: fetch submodules, run go generate, check *_gen.go diff

- checkout: submodules: recursive so third_party/hive populates
- new step: go generate ./graph/personas/... (regen status_gen.go)
- broaden diff check to cover *_gen.go in addition to *_templ.go

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Local verification, push, open PR

**Files:** none (verification + remote actions)

- [ ] **Step 1: Full local build and test**

```bash
make build
DATABASE_URL=postgres://site:site@localhost:5433/site?sslmode=disable go test -count=1 ./...
go vet ./...
```

Expected: all commands succeed.

- [ ] **Step 2: Confirm the generated file has no uncommitted drift**

```bash
go generate ./graph/personas/...
git diff --exit-code -- '*_templ.go' '*_gen.go'
```

Expected: no diff (exit code 0).

- [ ] **Step 3: Push the branch to the transpara remote**

```bash
git push -u transpara feat/persona-status-from-hive
```

- [ ] **Step 4: Open PR on transpara-ai/site**

```bash
gh pr create --repo transpara-ai/site --base main --head feat/persona-status-from-hive \
  --title "feat(personas): populate Status from hive via submodule + go generate" \
  --body "$(cat <<'EOF'
## Summary

Implements Option B from the role/persona decomposition debt doc ([transpara-ai/hive/docs/debt/role-persona-decomposition.md](https://github.com/transpara-ai/hive/blob/main/docs/debt/role-persona-decomposition.md)). Supersedes closed PR #14.

- Adds `third_party/hive` as a git submodule pinned to a specific hive commit.
- Adds `cmd/gen-persona-status` — reads hive `agents/*.md`, emits `graph/personas/status_gen.go` containing `var HiveStatus = map[string]string{persona: status}`.
- `SeedAgentPersonas` looks up `HiveStatus[name]` and forces `Active=false` for `absorbed`/`retired`.
- Re-lands PR #14's store-side plumbing (Status field, DDL, SELECTs, UPSERT) unchanged.
- CI fetches submodules, runs `go generate`, and fails on generated-file drift.

Design spec: `docs/superpowers/specs/2026-04-17-persona-submodule-generate-design.md`.
Plan: `docs/superpowers/plans/2026-04-17-persona-status-from-hive.md`.

## Why not just re-propagate PR #14's 54-file metadata?

PR #14 inlined the hive metadata into each site `.md` file — two copies of the same fact, hand-maintained. The debt doc explicitly flags this as the drift risk. Option B removes the second copy entirely: hive is authoritative, site's `status_gen.go` is generated output, site's `.md` bodies carry no metadata.

## Bumping the hive pin

```bash
cd third_party/hive && git fetch && git checkout origin/main && cd -
go generate ./graph/personas/...
git add third_party/hive graph/personas/status_gen.go
git commit -m "chore: bump hive pin"
```

## Test plan

- [ ] CI passes (submodule checkout, `go generate`, diff check, build, test).
- [ ] Locally: `go test -count=1 ./...` passes.
- [ ] After deploy: `SELECT name, active, status FROM agent_personas WHERE status IN ('absorbed','retired');` returns rows with `active = false`.
- [ ] Agents page no longer lists absorbed/retired personas.

## Non-goals

- Runtime enforcement of Status (blocking agent spawns) — deferred, tracked in the debt doc.
- Role/Persona decomposition itself — this is the stopgap the debt doc describes as "already shipping".

Generated with [Claude Code](https://claude.com/claude-code).
EOF
)"
```

Expected: a PR URL. Return it to the user.

---

## Self-Review

**Spec coverage:** Every section of the spec is implemented:
- Hive submodule at `third_party/hive` → Task 2.
- Generator `cmd/gen-persona-status` → Task 3.
- `//go:generate` directive → Task 4.
- Committed `status_gen.go` → Task 5.
- Store plumbing (`Status` field, DDL, SELECTs, UPSERT) → Task 6.
- Consumer (SeedAgentPersonas with HiveStatus lookup + active forcing) → Task 7.
- Integration test → Task 8.
- CI wiring (submodules, go generate, diff glob) → Task 9.
- `.dockerignore` update → Task 2 step 4.
- Transition (close #14, new branch) → Task 1 + already on branch.

**Placeholder scan:** No TBDs, TODOs, or "implement later" phrases. All code blocks contain real code.

**Type consistency:**
- `scan` returns `[]entry` (used in Task 3 test and Task 3 main).
- `render` takes `[]entry`, returns `([]byte, error)` (used in Task 3 test and Task 3 main).
- `HiveStatus` is `map[string]string` in `graph/personas/status_gen.go`, referenced as `personas.HiveStatus` in `graph/personas.go`.
- `AgentPersona.Status` is `string`, JSON tag `"status"` (Task 6 re-land).

**Known conditional:** Task 8's integration test SKIPS when no personas have status `absorbed` or `retired`. That's intentional — correctness on real data depends on the pinned hive commit.
