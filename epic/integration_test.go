package epic

import (
	"context"
	"testing"
)

// TestFullEpicLifecycle exercises the complete happy-path lifecycle:
// create → open → set body → split → transition children → verify derived status.
func TestFullEpicLifecycle(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "test-epic"

	// 1. Create a new epic.
	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}

	// 2. Open it and verify root exists with status=pending.
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	root, err := GetTask(ctx, q, TaskID(epicID))
	if err != nil {
		t.Fatalf("GetTask root: %v", err)
	}
	if root.Status != StatusPending {
		t.Fatalf("root status = %q, want %q", root.Status, StatusPending)
	}
	if !root.IsLeaf {
		t.Fatal("root should be a leaf before split")
	}

	// 3. Set body with two sections separated by ---.
	body := "Section one\n---\nSection two"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}

	// 4. Split the root task.
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// 5. Verify: root is now a branch (no direct status), has 2 children.
	root, err = GetTask(ctx, q, TaskID(epicID))
	if err != nil {
		t.Fatalf("GetTask root after split: %v", err)
	}
	if root.IsLeaf {
		t.Fatal("root should be a branch after split")
	}

	children, err := ListTasks(ctx, q, TaskID(epicID), true)
	if err != nil {
		t.Fatalf("ListTasks children: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}

	child1 := children[0].ID
	child2 := children[1].ID

	// 6. Start child 1 (pending → active).
	if err := TransitionStatus(ctx, conn, q, child1, StatusActive, ""); err != nil {
		t.Fatalf("start child1: %v", err)
	}

	// 7. Complete child 1 (active → done).
	if err := TransitionStatus(ctx, conn, q, child1, StatusDone, ""); err != nil {
		t.Fatalf("complete child1: %v", err)
	}

	// 8. Start child 2, then complete it.
	if err := TransitionStatus(ctx, conn, q, child2, StatusActive, ""); err != nil {
		t.Fatalf("start child2: %v", err)
	}
	if err := TransitionStatus(ctx, conn, q, child2, StatusDone, ""); err != nil {
		t.Fatalf("complete child2: %v", err)
	}

	// 9. Verify: derived status of root is "done".
	derived, err := GetDerivedStatus(ctx, q, TaskID(epicID))
	if err != nil {
		t.Fatalf("GetDerivedStatus: %v", err)
	}
	if derived != StatusDone {
		t.Fatalf("derived status = %q, want %q", derived, StatusDone)
	}

	// 10. Verify: ListEpics shows the epic with derived status "done".
	infos, err := ListEpics(dir)
	if err != nil {
		t.Fatalf("ListEpics: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 epic, got %d", len(infos))
	}
	if infos[0].ID != epicID {
		t.Fatalf("epic ID = %q, want %q", infos[0].ID, epicID)
	}
	if infos[0].Status != StatusDone {
		t.Fatalf("epic status = %q, want %q", infos[0].Status, StatusDone)
	}
}

// TestEpicWithBlockedTask verifies the blocked/unblocked derived status flow.
func TestEpicWithBlockedTask(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "blocked-epic"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body with 2 sections and split.
	body := "Task A\n---\nTask B"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	children, err := ListTasks(ctx, q, TaskID(epicID), true)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	child1 := children[0].ID

	// Start child 1.
	if err := TransitionStatus(ctx, conn, q, child1, StatusActive, ""); err != nil {
		t.Fatalf("start child1: %v", err)
	}

	// Block child 1 with a reason.
	if err := TransitionStatus(ctx, conn, q, child1, StatusBlocked, "waiting on dependency"); err != nil {
		t.Fatalf("block child1: %v", err)
	}

	// Verify derived status is "blocked".
	derived, err := GetDerivedStatus(ctx, q, TaskID(epicID))
	if err != nil {
		t.Fatalf("GetDerivedStatus after block: %v", err)
	}
	if derived != StatusBlocked {
		t.Fatalf("derived status = %q, want %q", derived, StatusBlocked)
	}

	// Unblock child 1 (back to active).
	if err := TransitionStatus(ctx, conn, q, child1, StatusActive, ""); err != nil {
		t.Fatalf("unblock child1: %v", err)
	}

	// Verify derived status is "active".
	derived, err = GetDerivedStatus(ctx, q, TaskID(epicID))
	if err != nil {
		t.Fatalf("GetDerivedStatus after unblock: %v", err)
	}
	if derived != StatusActive {
		t.Fatalf("derived status = %q, want %q", derived, StatusActive)
	}
}

// TestEpicWithAbandonedTask verifies that done+abandoned yields "done".
func TestEpicWithAbandonedTask(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "abandon-epic"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body with 2 sections and split.
	body := "Task A\n---\nTask B"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	children, err := ListTasks(ctx, q, TaskID(epicID), true)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	child1 := children[0].ID
	child2 := children[1].ID

	// Abandon child 1 with reason (pending → abandoned).
	if err := TransitionStatus(ctx, conn, q, child1, StatusAbandoned, "no longer needed"); err != nil {
		t.Fatalf("abandon child1: %v", err)
	}

	// Complete child 2 (pending → active → done).
	if err := TransitionStatus(ctx, conn, q, child2, StatusActive, ""); err != nil {
		t.Fatalf("start child2: %v", err)
	}
	if err := TransitionStatus(ctx, conn, q, child2, StatusDone, ""); err != nil {
		t.Fatalf("complete child2: %v", err)
	}

	// Verify: derived status is "done" (at least one done, rest abandoned).
	derived, err := GetDerivedStatus(ctx, q, TaskID(epicID))
	if err != nil {
		t.Fatalf("GetDerivedStatus: %v", err)
	}
	if derived != StatusDone {
		t.Fatalf("derived status = %q, want %q", derived, StatusDone)
	}
}

// TestAddChildToSplitTask verifies that AddChild appends a third child correctly.
func TestAddChildToSplitTask(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "addchild-epic"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body with 2 sections and split.
	body := "Task A\n---\nTask B"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Add a third child via AddChild.
	newID, err := AddChild(ctx, conn, q, TaskID(epicID))
	if err != nil {
		t.Fatalf("AddChild: %v", err)
	}

	// Verify 3 children exist.
	children, err := ListTasks(ctx, q, TaskID(epicID), true)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}

	// Verify the new child has correct ID suffix (3) and position (3).
	expectedID := TaskID(epicID + ":3")
	if newID != expectedID {
		t.Fatalf("new child ID = %q, want %q", newID, expectedID)
	}

	// Find the new child in the list and verify its position.
	var found bool
	for _, c := range children {
		if c.ID == expectedID {
			found = true
			if c.Position != 3 {
				t.Fatalf("new child position = %d, want 3", c.Position)
			}
			if c.Status != StatusPending {
				t.Fatalf("new child status = %q, want %q", c.Status, StatusPending)
			}
			break
		}
	}
	if !found {
		t.Fatalf("new child %s not found in children list", expectedID)
	}
}

// TestBodyFrozenAfterSplit verifies that SetTaskBody fails on a branch task.
func TestBodyFrozenAfterSplit(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "frozen-epic"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set body and split.
	body := "Task A\n---\nTask B"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Attempt to SetTaskBody on the now-branch root — should fail.
	err = SetTaskBody(ctx, conn, q, TaskID(epicID), "new body")
	if err == nil {
		t.Fatal("SetTaskBody on branch should have failed, but succeeded")
	}
}
