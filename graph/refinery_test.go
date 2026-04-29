package graph

import (
	"testing"
	"time"
)

func TestBuildRefineryProjectionIncludesInstrumentation(t *testing.T) {
	projectedAt := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 4, 29, 9, 30, 0, 0, time.UTC)
	space := Space{ID: "space-1", Slug: "journey-test"}
	tasks := []Node{
		{
			ID:        "task-1",
			SpaceID:   "space-1",
			Kind:      KindTask,
			Title:     "Build dashboard",
			State:     StateActive,
			Assignee:  "implementer",
			ChildDone: 2,
			UpdatedAt: updatedAt,
		},
		{
			ID:        "task-2",
			SpaceID:   "space-1",
			Kind:      KindTask,
			Title:     "Clarify design",
			State:     StateOpen,
			Tags:      []string{"design"},
			UpdatedAt: updatedAt,
		},
	}

	projection := buildRefineryProjection(space, tasks, projectedAt)

	if projection.SourceSystem != "site" {
		t.Fatalf("SourceSystem = %q, want site", projection.SourceSystem)
	}
	if projection.SourceID != "space-1" || projection.SpaceSlug != "journey-test" {
		t.Fatalf("space identity = (%q, %q), want (space-1, journey-test)", projection.SourceID, projection.SpaceSlug)
	}
	if !projection.ProjectedAt.Equal(projectedAt) {
		t.Fatalf("ProjectedAt = %s, want %s", projection.ProjectedAt, projectedAt)
	}
	if projection.Counts["ready"] != 1 || projection.Counts["refining"] != 1 {
		t.Fatalf("counts = %#v, want ready=1 refining=1", projection.Counts)
	}
	if projection.ExecCounts["building"] != 1 || projection.ExecCounts["unassigned"] != 1 {
		t.Fatalf("execution counts = %#v, want building=1 unassigned=1", projection.ExecCounts)
	}

	var ready RefineryItem
	for _, col := range projection.Columns {
		if col.State != "ready" {
			continue
		}
		if len(col.Items) != 1 {
			t.Fatalf("ready items = %d, want 1", len(col.Items))
		}
		ready = col.Items[0]
	}
	if ready.SourceSystem != "eventgraph" || ready.SourceID != "task-1" {
		t.Fatalf("ready source = (%q, %q), want (eventgraph, task-1)", ready.SourceSystem, ready.SourceID)
	}
	if ready.ExecutionStatus != "building" {
		t.Fatalf("ExecutionStatus = %q, want building", ready.ExecutionStatus)
	}
	if ready.State != "ready" || ready.RawState != StateActive {
		t.Fatalf("states = (%q, %q), want (ready, %s)", ready.State, ready.RawState, StateActive)
	}
	if ready.Owner != "implementer" {
		t.Fatalf("Owner = %q, want implementer", ready.Owner)
	}
	if ready.EvidenceCount != 2 {
		t.Fatalf("EvidenceCount = %d, want 2", ready.EvidenceCount)
	}
	if !ready.ProjectedAt.Equal(projectedAt) || !ready.LastEventAt.Equal(updatedAt) {
		t.Fatalf("timestamps = (%s, %s), want (%s, %s)", ready.ProjectedAt, ready.LastEventAt, projectedAt, updatedAt)
	}
}
