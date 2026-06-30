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
