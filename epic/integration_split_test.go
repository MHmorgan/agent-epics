package epic

import (
	"context"
	"testing"
)

// TestSplit_NoSeparatorsError verifies that SplitTask fails when
// the body has no --- separators.
func TestSplit_NoSeparatorsError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "no-sep"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body with no separators.
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), "just a single section"); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}

	// SplitTask should fail.
	err = SplitTask(ctx, conn, q, TaskID(epicID))
	if err == nil {
		t.Fatal("SplitTask should have failed with no separators, but succeeded")
	}
}

// TestSplit_AlreadySplitError verifies that splitting an already-split
// (branch) task fails.
func TestSplit_AlreadySplitError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "already-split"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body and split.
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), "A\n---\nB"); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID)); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Attempt to split the root again — it's now a branch.
	err = SplitTask(ctx, conn, q, TaskID(epicID))
	if err == nil {
		t.Fatal("SplitTask on branch should have failed, but succeeded")
	}
}

// TestSplit_ActiveStatusError verifies that SplitTask rejects a task
// with active status.
func TestSplit_ActiveStatusError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "active-split"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body with separator.
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), "A\n---\nB"); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}

	// Transition root to active.
	if err := TransitionStatus(ctx, conn, q, TaskID(epicID), StatusActive, ""); err != nil {
		t.Fatalf("TransitionStatus to active: %v", err)
	}

	// SplitTask should fail because status is active.
	err = SplitTask(ctx, conn, q, TaskID(epicID))
	if err == nil {
		t.Fatal("SplitTask on active task should have failed, but succeeded")
	}
}

// TestSplit_BlockedStatusAllowed verifies that SplitTask succeeds on a
// blocked task.
func TestSplit_BlockedStatusAllowed(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "blocked-split"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body with separator.
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), "A\n---\nB"); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}

	// Transition root: pending → active → blocked.
	if err := TransitionStatus(ctx, conn, q, TaskID(epicID), StatusActive, ""); err != nil {
		t.Fatalf("TransitionStatus to active: %v", err)
	}
	if err := TransitionStatus(ctx, conn, q, TaskID(epicID), StatusBlocked, "waiting on something"); err != nil {
		t.Fatalf("TransitionStatus to blocked: %v", err)
	}

	// SplitTask should succeed on blocked status.
	if err := SplitTask(ctx, conn, q, TaskID(epicID)); err != nil {
		t.Fatalf("SplitTask on blocked task should have succeeded: %v", err)
	}

	// Verify children were created.
	children, err := ListTasks(ctx, q, TaskID(epicID), true)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
}

// TestUnsplit_DirtyChildrenError verifies that UnsplitTask fails when a
// child has context set.
func TestUnsplit_DirtyChildrenError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "dirty-unsplit"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body and split.
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), "A\n---\nB"); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID)); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Set context on child 1.
	child1 := TaskID(epicID + ":1")
	if err := SetTaskContext(ctx, conn, q, child1, "some context"); err != nil {
		t.Fatalf("SetTaskContext: %v", err)
	}

	// UnsplitTask should fail because child has context.
	err = UnsplitTask(ctx, conn, q, TaskID(epicID))
	if err == nil {
		t.Fatal("UnsplitTask should have failed with dirty child (context set), but succeeded")
	}
}

// TestUnsplit_ChildWithAgentRecordsError verifies that UnsplitTask fails
// when a child has non-system (agent) records.
func TestUnsplit_ChildWithAgentRecordsError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "record-unsplit"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body and split.
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), "A\n---\nB"); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID)); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Add agent record on child 1.
	child1 := TaskID(epicID + ":1")
	if err := AddRecord(ctx, q, child1, "agent did something"); err != nil {
		t.Fatalf("AddRecord: %v", err)
	}

	// UnsplitTask should fail because child has non-system records.
	err = UnsplitTask(ctx, conn, q, TaskID(epicID))
	if err == nil {
		t.Fatal("UnsplitTask should have failed with child having agent records, but succeeded")
	}
}

// TestUnsplit_NonPendingChildError verifies that UnsplitTask fails when a
// child is not in pending status.
func TestUnsplit_NonPendingChildError(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "nonpending-unsplit"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body and split.
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), "A\n---\nB"); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID)); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Start child 1 (pending → active).
	child1 := TaskID(epicID + ":1")
	if err := TransitionStatus(ctx, conn, q, child1, StatusActive, ""); err != nil {
		t.Fatalf("TransitionStatus: %v", err)
	}

	// UnsplitTask should fail because child is active, not pending.
	err = UnsplitTask(ctx, conn, q, TaskID(epicID))
	if err == nil {
		t.Fatal("UnsplitTask should have failed with non-pending child, but succeeded")
	}
}

// TestUnsplit_CleanRoundTrip verifies that split followed by unsplit
// restores the task to its original state.
func TestUnsplit_CleanRoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "roundtrip"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body with 2 sections.
	originalBody := "Section one\n---\nSection two"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), originalBody); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}

	// Split the root task.
	if err := SplitTask(ctx, conn, q, TaskID(epicID)); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Verify 2 children exist.
	children, err := ListTasks(ctx, q, TaskID(epicID), true)
	if err != nil {
		t.Fatalf("ListTasks after split: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children after split, got %d", len(children))
	}

	// Unsplit.
	if err := UnsplitTask(ctx, conn, q, TaskID(epicID)); err != nil {
		t.Fatalf("UnsplitTask: %v", err)
	}

	// Verify: root is back to pending and is a leaf.
	root, err := GetTask(ctx, q, TaskID(epicID))
	if err != nil {
		t.Fatalf("GetTask after unsplit: %v", err)
	}
	if root.Status != StatusPending {
		t.Fatalf("root status = %q, want %q", root.Status, StatusPending)
	}
	if !root.IsLeaf {
		t.Fatal("root should be a leaf after unsplit")
	}

	// Verify: no children remain.
	children, err = ListTasks(ctx, q, TaskID(epicID), true)
	if err != nil {
		t.Fatalf("ListTasks after unsplit: %v", err)
	}
	if len(children) != 0 {
		t.Fatalf("expected 0 children after unsplit, got %d", len(children))
	}

	// Verify: root body is preserved.
	if root.Body != originalBody {
		t.Fatalf("root body = %q, want %q", root.Body, originalBody)
	}
}
