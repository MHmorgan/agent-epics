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
	err = SplitTask(ctx, conn, q, TaskID(epicID), true)
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
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Attempt to split the root again — it's now a branch.
	err = SplitTask(ctx, conn, q, TaskID(epicID), true)
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
	err = SplitTask(ctx, conn, q, TaskID(epicID), true)
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
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
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

