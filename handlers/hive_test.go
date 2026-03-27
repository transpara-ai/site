package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadHiveState verifies Iteration and Phase extraction from state.md.
func TestReadHiveState(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "state.md"), []byte("Iteration: 42\nPhase: Critic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	iter, phase := readHiveState(dir)
	if iter != 42 {
		t.Errorf("iter = %d, want 42", iter)
	}
	if phase != "Critic" {
		t.Errorf("phase = %q, want Critic", phase)
	}
}

// TestReadHiveState_MissingFile returns zero values when state.md is absent.
func TestReadHiveState_MissingFile(t *testing.T) {
	iter, phase := readHiveState(t.TempDir())
	if iter != 0 || phase != "" {
		t.Errorf("got (%d, %q), want (0, '')", iter, phase)
	}
}

// TestReadHiveBuild verifies title and cost extraction from build.md.
func TestReadHiveBuild(t *testing.T) {
	dir := t.TempDir()
	content := "# Add knowledge layer\n\nBuilder shipped. Cost: $0.53. Duration: 3m.\n"
	if err := os.WriteFile(filepath.Join(dir, "build.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	title, cost := readHiveBuild(dir)
	if title != "Add knowledge layer" {
		t.Errorf("title = %q, want 'Add knowledge layer'", title)
	}
	if cost != "$0.53" {
		t.Errorf("cost = %q, want '$0.53'", cost)
	}
}

// TestReadHiveBuild_MissingFile returns empty strings when build.md is absent.
func TestReadHiveBuild_MissingFile(t *testing.T) {
	title, cost := readHiveBuild(t.TempDir())
	if title != "" || cost != "" {
		t.Errorf("got (%q, %q), want ('', '')", title, cost)
	}
}

// TestReadHiveDiagnostics_EmptyFile returns nil for an empty diagnostics file.
func TestReadHiveDiagnostics_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "diagnostics.jsonl"), []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	got := readHiveDiagnostics(dir, 10)
	if got != nil {
		t.Errorf("expected nil for empty file, got %v", got)
	}
}

// TestReadHiveDiagnostics_MalformedLine skips malformed JSON lines without error.
func TestReadHiveDiagnostics_MalformedLine(t *testing.T) {
	dir := t.TempDir()
	content := `{"phase":"scout","outcome":"done","cost_usd":0.1,"timestamp":"2026-01-01T00:00:00Z"}
not-json
{"phase":"builder","outcome":"pass","cost_usd":0.2,"timestamp":"2026-01-02T00:00:00Z"}
`
	if err := os.WriteFile(filepath.Join(dir, "diagnostics.jsonl"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got := readHiveDiagnostics(dir, 10)
	if len(got) != 2 {
		t.Errorf("len = %d, want 2 (malformed line should be skipped)", len(got))
	}
	// Newest first: builder is index 0.
	if got[0].Phase != "builder" {
		t.Errorf("got[0].Phase = %q, want 'builder'", got[0].Phase)
	}
}

// TestReadHiveDiagnostics_Limit verifies that only the last `limit` entries are returned.
func TestReadHiveDiagnostics_Limit(t *testing.T) {
	dir := t.TempDir()
	var lines []string
	for i := range 20 {
		lines = append(lines, `{"phase":"scout","outcome":"done","cost_usd":0.01,"timestamp":"2026-01-01T00:00:00Z"}`)
		_ = i
	}
	// Add a distinctive last entry.
	lines = append(lines, `{"phase":"critic","outcome":"pass","cost_usd":0.99,"timestamp":"2026-01-02T00:00:00Z"}`)
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "diagnostics.jsonl"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got := readHiveDiagnostics(dir, 5)
	if len(got) != 5 {
		t.Errorf("len = %d, want 5", len(got))
	}
	// Newest first: distinctive "critic" entry should be first.
	if got[0].Phase != "critic" {
		t.Errorf("got[0].Phase = %q, want 'critic'", got[0].Phase)
	}
}

// TestReadHiveDiagnostics_CostFormat verifies cost is formatted as "$X.XXX".
func TestReadHiveDiagnostics_CostFormat(t *testing.T) {
	dir := t.TempDir()
	content := `{"phase":"builder","outcome":"done","cost_usd":0.123456,"timestamp":"2026-01-01T00:00:00Z"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "diagnostics.jsonl"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got := readHiveDiagnostics(dir, 10)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Cost != "$0.123" {
		t.Errorf("Cost = %q, want '$0.123'", got[0].Cost)
	}
}

// TestReadHiveDiagnostics_ZeroCost verifies zero cost renders as empty string.
func TestReadHiveDiagnostics_ZeroCost(t *testing.T) {
	dir := t.TempDir()
	content := `{"phase":"reflector","outcome":"idle","cost_usd":0,"timestamp":"2026-01-01T00:00:00Z"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "diagnostics.jsonl"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got := readHiveDiagnostics(dir, 10)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Cost != "" {
		t.Errorf("Cost = %q, want '' for zero cost", got[0].Cost)
	}
}

// TestReadHiveDiagnostics_MissingDir returns nil when loopDir is empty.
func TestReadHiveDiagnostics_MissingDir(t *testing.T) {
	got := readHiveDiagnostics("", 10)
	if got != nil {
		t.Errorf("expected nil for empty loopDir, got %v", got)
	}
}

// TestHiveFeed_JSON verifies GET /hive/feed returns JSON with Content-Type application/json.
func TestHiveFeed_JSON(t *testing.T) {
	dir := t.TempDir()
	content := `{"phase":"scout","outcome":"done","cost_usd":0.05,"timestamp":"2026-03-01T12:00:00Z"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "diagnostics.jsonl"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HIVE_REPO_PATH", filepath.Dir(dir)) // parent of loop/

	req := httptest.NewRequest(http.MethodGet, "/hive/feed", nil)
	w := httptest.NewRecorder()
	HiveFeed(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var got []DiagEntry
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
}

// TestHiveFeed_EmptyDir returns JSON null when no diagnostics file exists.
func TestHiveFeed_EmptyDir(t *testing.T) {
	t.Setenv("HIVE_REPO_PATH", t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/hive/feed", nil)
	w := httptest.NewRecorder()
	HiveFeed(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

// TestHiveFeed_MaxEntries verifies at most maxFeedEntries entries are returned.
func TestHiveFeed_MaxEntries(t *testing.T) {
	dir := t.TempDir()
	var lines []string
	for range maxFeedEntries + 5 {
		lines = append(lines, `{"phase":"scout","outcome":"done","cost_usd":0.01,"timestamp":"2026-01-01T00:00:00Z"}`)
	}
	if err := os.WriteFile(filepath.Join(dir, "diagnostics.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HIVE_REPO_PATH", dir)

	req := httptest.NewRequest(http.MethodGet, "/hive/feed", nil)
	w := httptest.NewRecorder()
	HiveFeed(w, req)

	var got []DiagEntry
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(got) > maxFeedEntries {
		t.Errorf("len = %d, want <= %d", len(got), maxFeedEntries)
	}
}

// TestHiveDashboard_Returns200 verifies HiveDashboard returns 200 with loop files present.
func TestHiveDashboard_Returns200(t *testing.T) {
	dir := t.TempDir()
	loopDir := filepath.Join(dir, "loop")
	if err := os.Mkdir(loopDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(loopDir, "state.md"), []byte("Iteration: 5\nPhase: Builder\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(loopDir, "build.md"), []byte("# Ship the feature\n\nCost: $0.30\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HIVE_REPO_PATH", dir)

	req := httptest.NewRequest(http.MethodGet, "/hive", nil)
	w := httptest.NewRecorder()
	HiveDashboard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

// TestHiveDashboard_NoFiles returns 200 with zero-value data when loop files are absent.
func TestHiveDashboard_NoFiles(t *testing.T) {
	t.Setenv("HIVE_REPO_PATH", t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/hive", nil)
	w := httptest.NewRecorder()
	HiveDashboard(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 even without loop files", w.Code)
	}
}
