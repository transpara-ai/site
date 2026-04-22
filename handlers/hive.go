// Package handlers provides standalone HTTP handlers for the hive dashboard.
// These handlers read directly from hive loop files on disk and do not require
// a database connection. The hive repo path is resolved from the HIVE_REPO_PATH
// environment variable, falling back to the ../hive sibling directory.
package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lovyou-ai/site/graph"
	"github.com/lovyou-ai/site/profile"
)

// DiagEntry is a phase diagnostic event from loop/diagnostics.jsonl.
// All fields are strings for straightforward JSON serialisation.
type DiagEntry struct {
	Phase     string `json:"phase"`
	Outcome   string `json:"outcome"`
	Cost      string `json:"cost"`
	Timestamp string `json:"timestamp"`
}

// HiveDashboardData holds the collected state for the /hive dashboard.
type HiveDashboardData struct {
	Iteration      int
	Phase          string
	LastBuildTitle string
	BuildCost      string
	PhaseHistory   []DiagEntry
	RecentCommits  []string
}

const maxCommits = 10
const maxDiagEntries = 50
const maxFeedEntries = 10

var reDollar = regexp.MustCompile(`\$(\d+\.\d+)`)

// hiveRepoDir returns the hive repo root from HIVE_REPO_PATH env,
// or the ../hive sibling of the current working directory.
func hiveRepoDir() string {
	if p := os.Getenv("HIVE_REPO_PATH"); p != "" {
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(wd), "hive")
}

// hiveLoopDir returns the loop/ directory within the hive repo.
func hiveLoopDir() string {
	d := hiveRepoDir()
	if d == "" {
		return ""
	}
	return filepath.Join(d, "loop")
}

// readHiveSummary returns the current iteration count and phase for the
// dashboard header, sourced from loop/diagnostics.jsonl.
//
// Iteration = newline-count of the jsonl file. This matches the hive's
// own canonical counter (pkg/runner/pipeline_tree.go: countDiagnostics)
// and advances exactly once per phase event written by the daemon, so
// the number the dashboard shows is the number the hive itself uses.
//
// Phase = the `phase` field of the most recent well-formed entry. The
// scan walks backwards so a partially-written tail (mid-flush crash)
// falls through to the last complete event instead of blanking out.
//
// Returns zero values if the file is missing or contains no well-formed
// entries. state.md is no longer read here — drift between what the
// Reflector wrote there and what the site's parser expected was
// producing iteration=0 on a working hive.
func readHiveSummary(loopDir string) (iter int, phase string) {
	if loopDir == "" {
		return 0, ""
	}
	data, err := os.ReadFile(filepath.Join(loopDir, "diagnostics.jsonl"))
	if err != nil {
		return 0, ""
	}
	for _, b := range data {
		if b == '\n' {
			iter++
		}
	}
	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var entry struct {
			Phase string `json:"phase"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			phase = entry.Phase
			break
		}
	}
	return iter, phase
}

// readHiveBuild extracts the first H1 title and dollar cost from loop/build.md.
// Returns empty strings if the file is missing.
func readHiveBuild(loopDir string) (title, cost string) {
	if loopDir == "" {
		return "", ""
	}
	data, err := os.ReadFile(filepath.Join(loopDir, "build.md"))
	if err != nil {
		return "", ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		if after, ok := strings.CutPrefix(scanner.Text(), "# "); ok && title == "" {
			title = strings.TrimSpace(after)
		}
	}
	if m := reDollar.FindStringSubmatch(string(data)); m != nil {
		cost = "$" + m[1]
	}
	return title, cost
}

// readHiveDiagnostics reads the last limit entries from loop/diagnostics.jsonl,
// returning them newest-first. Malformed lines are silently skipped.
// Returns nil if the file is missing or loopDir is empty.
func readHiveDiagnostics(loopDir string, limit int) []DiagEntry {
	if loopDir == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(loopDir, "diagnostics.jsonl"))
	if err != nil {
		return nil
	}
	type raw struct {
		Phase     string  `json:"phase"`
		Outcome   string  `json:"outcome"`
		CostUSD   float64 `json:"cost_usd"`
		Timestamp string  `json:"timestamp"`
	}
	var all []DiagEntry
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		var r raw
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		cost := ""
		if r.CostUSD > 0 {
			cost = fmt.Sprintf("$%.3f", r.CostUSD)
		}
		all = append(all, DiagEntry{
			Phase:     r.Phase,
			Outcome:   r.Outcome,
			Cost:      cost,
			Timestamp: r.Timestamp,
		})
	}
	if len(all) == 0 {
		return nil
	}
	start := 0
	if len(all) > limit {
		start = len(all) - limit
	}
	tail := make([]DiagEntry, len(all)-start)
	copy(tail, all[start:])
	// Reverse so newest entry is first.
	for i, j := 0, len(tail)-1; i < j; i, j = i+1, j-1 {
		tail[i], tail[j] = tail[j], tail[i]
	}
	return tail
}

// readHiveCommits runs git log --oneline in repoDir and returns up to maxCommits lines.
// Returns nil if repoDir is empty or git is unavailable.
func readHiveCommits(repoDir string) []string {
	if repoDir == "" {
		return nil
	}
	out, err := exec.Command("git", "-C", repoDir, "log", "--oneline", fmt.Sprintf("-%d", maxCommits)).Output()
	if err != nil {
		return nil
	}
	var commits []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			commits = append(commits, line)
		}
	}
	return commits
}

// buildHiveDashboardData collects all loop state and git history for the hive dashboard.
func buildHiveDashboardData() HiveDashboardData {
	loopDir := hiveLoopDir()
	repoDir := hiveRepoDir()
	iter, phase := readHiveSummary(loopDir)
	title, cost := readHiveBuild(loopDir)
	history := readHiveDiagnostics(loopDir, maxDiagEntries)
	commits := readHiveCommits(repoDir)
	return HiveDashboardData{
		Iteration:      iter,
		Phase:          phase,
		LastBuildTitle: title,
		BuildCost:      cost,
		PhaseHistory:   history,
		RecentCommits:  commits,
	}
}

// HiveDashboard handles GET /hive — renders the hive dashboard page.
func HiveDashboard(w http.ResponseWriter, r *http.Request) {
	d := buildHiveDashboardData()

	ls := graph.LoopState{
		Iteration:  d.Iteration,
		Phase:      d.Phase,
		BuildTitle: d.LastBuildTitle,
	}
	if m := reDollar.FindStringSubmatch(d.BuildCost); m != nil {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			ls.BuildCost = v
		}
	}

	entries := make([]graph.DiagEntry, 0, len(d.PhaseHistory))
	for _, e := range d.PhaseHistory {
		ge := graph.DiagEntry{Phase: e.Phase, Outcome: e.Outcome}
		if m := reDollar.FindStringSubmatch(e.Cost); m != nil {
			if v, err := strconv.ParseFloat(m[1], 64); err == nil {
				ge.CostUSD = v
			}
		}
		if e.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, e.Timestamp); err == nil {
				ge.Timestamp = t
			}
		}
		entries = append(entries, ge)
	}

	commits := make([]graph.RecentCommit, 0, len(d.RecentCommits))
	for _, line := range d.RecentCommits {
		if parts := strings.SplitN(line, " ", 2); len(parts) == 2 {
			commits = append(commits, graph.RecentCommit{Hash: parts[0], Subject: parts[1]})
		}
	}

	graph.HivePage(ls, entries, commits, graph.ViewUser{}, profile.FromContext(r.Context())).Render(r.Context(), w)
}

// HiveFeed handles GET /hive/feed — returns JSON of the last maxFeedEntries phase history entries.
func HiveFeed(w http.ResponseWriter, r *http.Request) {
	d := buildHiveDashboardData()
	entries := d.PhaseHistory
	if len(entries) > maxFeedEntries {
		entries = entries[:maxFeedEntries]
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}
