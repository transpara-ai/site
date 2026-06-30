package graph

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
