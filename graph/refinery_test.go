package graph

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/transpara-ai/site/profile"
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
	if got := projection.StateOrder; len(got) != 5 || got[0] != "inbox" || got[4] != "done" {
		t.Fatalf("StateOrder = %#v, want five-state FSM", got)
	}
	if projection.OpenCount != 2 {
		t.Fatalf("OpenCount = %d, want 2", projection.OpenCount)
	}
	if projection.Counts["ready"] != 1 || projection.Counts["refining"] != 1 {
		t.Fatalf("counts = %#v, want ready=1 refining=1", projection.Counts)
	}
	if projection.ExecCounts["building"] != 1 || projection.ExecCounts["refining"] != 1 {
		t.Fatalf("execution counts = %#v, want building=1 refining=1", projection.ExecCounts)
	}
	if projection.HumanStatus == "" {
		t.Fatal("HumanStatus is empty")
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

func TestRefineryFSMKeepsExecutionOutOfColumns(t *testing.T) {
	cases := []struct {
		name   string
		task   Node
		state  string
		status string
	}{
		{
			name:   "unassigned open intake",
			task:   Node{ID: "task-1", State: StateOpen},
			state:  "inbox",
			status: "unassigned",
		},
		{
			name:   "design work is refining",
			task:   Node{ID: "task-2", State: StateOpen, Tags: []string{"design"}},
			state:  "refining",
			status: "refining",
		},
		{
			name:   "review remains review",
			task:   Node{ID: "task-3", State: StateReview},
			state:  "review",
			status: "reviewing",
		},
		{
			name:   "assigned work is ready",
			task:   Node{ID: "task-4", State: StateOpen, Assignee: "builder"},
			state:  "ready",
			status: "assigned",
		},
		{
			name:   "active build is ready with building status",
			task:   Node{ID: "task-5", State: StateActive, Assignee: "builder"},
			state:  "ready",
			status: "building",
		},
		{
			name:   "blocked build is ready with blocked status",
			task:   Node{ID: "task-6", State: StateBlocked, Assignee: "builder", BlockerCount: 1},
			state:  "ready",
			status: "blocked",
		},
		{
			name:   "done work is done",
			task:   Node{ID: "task-7", State: StateDone},
			state:  "done",
			status: "complete",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := refineryState(tc.task); got != tc.state {
				t.Fatalf("refineryState = %q, want %q", got, tc.state)
			}
			if got := refineryExecutionStatus(tc.task); got != tc.status {
				t.Fatalf("refineryExecutionStatus = %q, want %q", got, tc.status)
			}
		})
	}
}

func TestRefineryViewRendersExecutionFilters(t *testing.T) {
	projectedAt := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	space := Space{ID: "space-1", Slug: "journey-test", Name: "Journey Test"}
	projection := buildRefineryProjection(space, []Node{
		{ID: "task-building", Title: "Build it", State: StateActive, UpdatedAt: projectedAt},
		{ID: "task-blocked", Title: "Unblock it", State: StateBlocked, BlockerCount: 1, UpdatedAt: projectedAt},
		{ID: "task-assigned", Title: "Assigned item", State: StateOpen, Assignee: "builder", UpdatedAt: projectedAt},
	}, projectedAt)

	var buf bytes.Buffer
	if err := RefineryView(space, []Space{space}, projection, ViewUser{}, profile.Default()).Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	body := buf.String()
	for _, want := range []string{
		`data-refinery-filter="building"`,
		`data-refinery-filter="blocked"`,
		`data-refinery-filter="assigned"`,
		`data-refinery-exec-status="building"`,
		`data-refinery-exec-status="blocked"`,
		`data-refinery-exec-status="assigned"`,
		`id="refinery-markdown-file"`,
		`id="refinery-markdown-body"`,
		`Paste Markdown idea or spec`,
		"Filter Ready items by execution status",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("rendered refinery missing %q", want)
		}
	}
}
