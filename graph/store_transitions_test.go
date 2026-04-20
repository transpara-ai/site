package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestUpdateNodeStateAndRecordTransition_TrackedKind verifies that a tracked
// kind (task) records a structured `transition` op on state change.
func TestUpdateNodeStateAndRecordTransition_TrackedKind(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	slug := fmt.Sprintf("trans-task-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(ctx, slug, "Transitions", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	task, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		Title:    "Trans task",
		Author:   "tester",
		AuthorID: "tester-1",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	op, err := store.UpdateNodeStateAndRecordTransition(ctx, space.ID, task.ID, StateDone, "tester", "tester-1")
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if op == nil {
		t.Fatalf("want transition op for tracked kind, got nil")
	}
	if op.Op != "transition" {
		t.Errorf("op = %q, want %q", op.Op, "transition")
	}

	var payload map[string]string
	if err := json.Unmarshal(op.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["from_state"] != StateOpen {
		t.Errorf("from_state = %q, want %q", payload["from_state"], StateOpen)
	}
	if payload["to_state"] != StateDone {
		t.Errorf("to_state = %q, want %q", payload["to_state"], StateDone)
	}
	if payload["node_kind"] != KindTask {
		t.Errorf("node_kind = %q, want %q", payload["node_kind"], KindTask)
	}

	// Verify the node state was actually updated.
	got, err := store.GetNode(ctx, task.ID)
	if err != nil {
		t.Fatalf("get node: %v", err)
	}
	if got.State != StateDone {
		t.Errorf("state = %q, want %q", got.State, StateDone)
	}
}

// TestUpdateNodeStateAndRecordTransition_UntrackedKind verifies that an
// untracked kind (proposal) updates state but does not record a transition op.
func TestUpdateNodeStateAndRecordTransition_UntrackedKind(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	slug := fmt.Sprintf("trans-prop-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(ctx, slug, "Transitions", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	prop, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindProposal,
		Title:    "Some proposal",
		Author:   "tester",
		AuthorID: "tester-1",
	})
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}

	op, err := store.UpdateNodeStateAndRecordTransition(ctx, space.ID, prop.ID, ProposalPassed, "tester", "tester-1")
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if op != nil {
		t.Fatalf("want no transition op for untracked kind %q, got %+v", KindProposal, op)
	}

	// State should still be updated.
	got, err := store.GetNode(ctx, prop.ID)
	if err != nil {
		t.Fatalf("get node: %v", err)
	}
	if got.State != ProposalPassed {
		t.Errorf("state = %q, want %q", got.State, ProposalPassed)
	}
}

// TestUpdateNodeStateAndRecordTransition_NoOpOnSameState ensures we don't
// emit a transition when from_state == to_state.
func TestUpdateNodeStateAndRecordTransition_NoOpOnSameState(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	slug := fmt.Sprintf("trans-same-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(ctx, slug, "Transitions", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	task, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID:  space.ID,
		Kind:     KindTask,
		Title:    "Stays open",
		Author:   "tester",
		AuthorID: "tester-1",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	op, err := store.UpdateNodeStateAndRecordTransition(ctx, space.ID, task.ID, task.State, "tester", "tester-1")
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if op != nil {
		t.Errorf("want no op for same-state transition, got %+v", op)
	}
}

// TestUpdateNodeStateAndRecordTransition_CascadeEmitsChildOps ensures that
// when a parent task is completed, each cascade-closed tracked-kind child
// also emits its own `transition` op — so the hive bridge sees a signal for
// every auto-closed subtask, not just the root.
func TestUpdateNodeStateAndRecordTransition_CascadeEmitsChildOps(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	slug := fmt.Sprintf("trans-cascade-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(ctx, slug, "Cascade", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	parent, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Parent", Author: "tester", AuthorID: "tester-1",
	})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	child, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Child", Author: "tester", AuthorID: "tester-1",
		ParentID: parent.ID,
	})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}

	if _, err := store.UpdateNodeStateAndRecordTransition(ctx, space.ID, parent.ID, StateDone, "tester", "tester-1"); err != nil {
		t.Fatalf("transition parent: %v", err)
	}

	gotChild, err := store.GetNode(ctx, child.ID)
	if err != nil {
		t.Fatalf("get child: %v", err)
	}
	if gotChild.State != StateDone {
		t.Errorf("child state = %q, want done", gotChild.State)
	}

	// Check that a transition op was recorded for the child (not just the parent).
	ops, err := store.ListOps(ctx, space.ID, 100)
	if err != nil {
		t.Fatalf("list ops: %v", err)
	}
	var sawChildTransition, sawParentTransition bool
	for _, op := range ops {
		if op.Op != "transition" {
			continue
		}
		switch op.NodeID {
		case child.ID:
			sawChildTransition = true
		case parent.ID:
			sawParentTransition = true
		}
	}
	if !sawChildTransition {
		t.Error("no transition op recorded for cascade-closed child")
	}
	if !sawParentTransition {
		t.Error("no transition op recorded for parent")
	}
}

// TestUpdateNodeStateAndRecordTransition_ConcurrentCallersEmitSingleOp guards
// the TOCTOU fix: two concurrent complete calls on the same node must result
// in exactly one transition op, not two. Pre-fix, both callers would read
// from_state=open, both would UPDATE to done (idempotent), and both would
// emit a duplicate `transition: open→done` op.
func TestUpdateNodeStateAndRecordTransition_ConcurrentCallersEmitSingleOp(t *testing.T) {
	_, store := testDB(t)
	ctx := context.Background()

	slug := fmt.Sprintf("trans-race-%d", time.Now().UnixNano())
	space, err := store.CreateSpace(ctx, slug, "Race", "", "owner-1", "project", "public")
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	t.Cleanup(func() { store.DeleteSpace(ctx, space.ID) })

	task, err := store.CreateNode(ctx, CreateNodeParams{
		SpaceID: space.ID, Kind: KindTask, Title: "Race", Author: "tester", AuthorID: "tester-1",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	const N = 8
	errs := make(chan error, N)
	ops := make(chan *Op, N)
	ready := make(chan struct{})
	for i := 0; i < N; i++ {
		go func() {
			<-ready
			op, err := store.UpdateNodeStateAndRecordTransition(ctx, space.ID, task.ID, StateDone, "tester", "tester-1")
			ops <- op
			errs <- err
		}()
	}
	close(ready)
	var nonNil int
	for i := 0; i < N; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent call %d: %v", i, err)
		}
		if op := <-ops; op != nil {
			nonNil++
		}
	}
	if nonNil != 1 {
		t.Errorf("got %d non-nil transition ops from %d concurrent callers, want exactly 1", nonNil, N)
	}
}
