package graph

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchOpsWorkSummarizesWorkAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			t.Fatalf("path = %q, want /tasks", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tasks":[
			{"id":"task-1","title":"Blocked task","description":"Needs dependency","priority":"high","status":"open","assignee":"","blocked":true,"artifact_count":0,"waived":false},
			{"id":"task-2","title":"Active task","description":"Being built","priority":"medium","status":"in_progress","assignee":"agent-1","blocked":false,"artifact_count":2,"waived":false},
			{"id":"task-3","title":"Completed task","description":"Done","priority":"low","status":"completed","assignee":"agent-2","blocked":false,"artifact_count":1,"waived":true}
		]}`))
	}))
	defer srv.Close()

	t.Setenv("WORK_API_BASE_URL", srv.URL)
	t.Setenv("WORK_API_KEY", "test-key")
	req := httptest.NewRequest(http.MethodGet, "http://site.test/ops/work", nil)

	got := fetchOpsWork(req)

	if got.Error != "" {
		t.Fatalf("Error = %q, want empty", got.Error)
	}
	if got.Total != 3 || got.Open != 2 || got.Active != 1 || got.Blocked != 1 || got.Completed != 1 {
		t.Fatalf("counts = total:%d open:%d active:%d blocked:%d completed:%d", got.Total, got.Open, got.Active, got.Blocked, got.Completed)
	}
	if got.HighPriority != 1 || got.Unassigned != 1 || got.EvidenceCount != 3 || got.WaivedCount != 1 {
		t.Fatalf("summary = high:%d unassigned:%d evidence:%d waived:%d", got.HighPriority, got.Unassigned, got.EvidenceCount, got.WaivedCount)
	}
	if len(got.BlockedTasks) != 1 || got.BlockedTasks[0].ID != "task-1" {
		t.Fatalf("BlockedTasks = %#v, want task-1", got.BlockedTasks)
	}
}
