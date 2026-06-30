package graph

import (
	"encoding/json"
	"fmt"
	"net/http"
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
