package epic

import (
	"context"
	"testing"
)

// TestDependency_CycleRejection verifies that both direct and transitive cycles
// are rejected when adding dependency edges.
func TestDependency_CycleRejection(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "cycle-epic"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Create 3 children via body + split.
	body := "# Task 1\n---\n# Task 2\n---\n# Task 3"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	child1 := TaskID(epicID + ":1")
	child2 := TaskID(epicID + ":2")
	child3 := TaskID(epicID + ":3")

	// child 2 after child 1 — should succeed.
	if err := AddDependency(ctx, q, child2, child1); err != nil {
		t.Fatalf("AddDependency(2 after 1): %v", err)
	}

	// child 3 after child 2 — should succeed.
	if err := AddDependency(ctx, q, child3, child2); err != nil {
		t.Fatalf("AddDependency(3 after 2): %v", err)
	}

	// child 1 after child 3 — should fail (transitive cycle: 1→2→3→1).
	if err := AddDependency(ctx, q, child1, child3); err == nil {
		t.Fatal("AddDependency(1 after 3) should have failed due to transitive cycle")
	}

	// Direct cycle: separate setup.
	t.Run("direct_cycle", func(t *testing.T) {
		dir := t.TempDir()
		const epicID = "direct-cycle"

		if err := NewEpic(epicID, dir); err != nil {
			t.Fatalf("NewEpic: %v", err)
		}
		conn, q, err := OpenEpic(epicID, dir)
		if err != nil {
			t.Fatalf("OpenEpic: %v", err)
		}
		defer conn.Close()

		body := "# A\n---\n# B"
		if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
			t.Fatalf("SetTaskBody: %v", err)
		}
		if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
			t.Fatalf("SplitTask: %v", err)
		}

		a := TaskID(epicID + ":1")
		b := TaskID(epicID + ":2")

		// A after B — should succeed.
		if err := AddDependency(ctx, q, a, b); err != nil {
			t.Fatalf("AddDependency(A after B): %v", err)
		}

		// B after A — should fail (direct cycle).
		if err := AddDependency(ctx, q, b, a); err == nil {
			t.Fatal("AddDependency(B after A) should have failed due to direct cycle")
		}
	})
}

// TestNext_RespectsDepOrder verifies that NextTask returns tasks in dependency
// order: only the task with no unsatisfied predecessors is returned.
func TestNext_RespectsDepOrder(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "deporder-epic"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	body := "# Task 1\n---\n# Task 2\n---\n# Task 3"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	child1 := TaskID(epicID + ":1")
	child2 := TaskID(epicID + ":2")
	child3 := TaskID(epicID + ":3")

	// Chain: child2 after child1, child3 after child2.
	if err := AddDependency(ctx, q, child2, child1); err != nil {
		t.Fatalf("AddDependency(2 after 1): %v", err)
	}
	if err := AddDependency(ctx, q, child3, child2); err != nil {
		t.Fatalf("AddDependency(3 after 2): %v", err)
	}

	// NextTask should return child1 (only one with no unsatisfied deps).
	next, err := NextTask(ctx, conn, q, epicID)
	if err != nil {
		t.Fatalf("NextTask: %v", err)
	}
	if next == nil {
		t.Fatal("NextTask returned nil, expected child1")
	}
	if next.ID != child1 {
		t.Fatalf("NextTask = %s, want %s", next.ID, child1)
	}

	// Start and complete child1.
	if err := TransitionStatus(ctx, conn, q, child1, StatusActive, ""); err != nil {
		t.Fatalf("start child1: %v", err)
	}
	if err := TransitionStatus(ctx, conn, q, child1, StatusDone, ""); err != nil {
		t.Fatalf("complete child1: %v", err)
	}

	// NextTask should return child2.
	next, err = NextTask(ctx, conn, q, epicID)
	if err != nil {
		t.Fatalf("NextTask after child1 done: %v", err)
	}
	if next == nil {
		t.Fatal("NextTask returned nil, expected child2")
	}
	if next.ID != child2 {
		t.Fatalf("NextTask = %s, want %s", next.ID, child2)
	}

	// Start and complete child2.
	if err := TransitionStatus(ctx, conn, q, child2, StatusActive, ""); err != nil {
		t.Fatalf("start child2: %v", err)
	}
	if err := TransitionStatus(ctx, conn, q, child2, StatusDone, ""); err != nil {
		t.Fatalf("complete child2: %v", err)
	}

	// NextTask should return child3.
	next, err = NextTask(ctx, conn, q, epicID)
	if err != nil {
		t.Fatalf("NextTask after child2 done: %v", err)
	}
	if next == nil {
		t.Fatal("NextTask returned nil, expected child3")
	}
	if next.ID != child3 {
		t.Fatalf("NextTask = %s, want %s", next.ID, child3)
	}
}

// TestNext_SkipsBlockedByDeps verifies that NextTask returns nil when
// the only pending task is blocked by an unsatisfied dependency and
// the predecessor is active (not done).
func TestNext_SkipsBlockedByDeps(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "skipblocked-epic"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	body := "# Task 1\n---\n# Task 2"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	child1 := TaskID(epicID + ":1")
	child2 := TaskID(epicID + ":2")

	// child2 after child1.
	if err := AddDependency(ctx, q, child2, child1); err != nil {
		t.Fatalf("AddDependency(2 after 1): %v", err)
	}

	// Start child1 (now active, not done).
	if err := TransitionStatus(ctx, conn, q, child1, StatusActive, ""); err != nil {
		t.Fatalf("start child1: %v", err)
	}

	// NextTask should return nil: child1 is active (not pending), child2 is blocked by dep.
	next, err := NextTask(ctx, conn, q, epicID)
	if err != nil {
		t.Fatalf("NextTask: %v", err)
	}
	if next != nil {
		t.Fatalf("NextTask = %s, want nil (child1 active, child2 dep-blocked)", next.ID)
	}
}

// TestNext_AbandonmentSatisfiesDeps verifies that abandoning a predecessor
// satisfies the dependency, making the dependent task available via NextTask.
func TestNext_AbandonmentSatisfiesDeps(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "abandon-dep-epic"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	body := "# Task 1\n---\n# Task 2"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	child1 := TaskID(epicID + ":1")
	child2 := TaskID(epicID + ":2")

	// child2 after child1.
	if err := AddDependency(ctx, q, child2, child1); err != nil {
		t.Fatalf("AddDependency(2 after 1): %v", err)
	}

	// Abandon child1 (pending → abandoned).
	if err := TransitionStatus(ctx, conn, q, child1, StatusAbandoned, "no longer needed"); err != nil {
		t.Fatalf("abandon child1: %v", err)
	}

	// NextTask should return child2 (dep satisfied by abandonment).
	next, err := NextTask(ctx, conn, q, epicID)
	if err != nil {
		t.Fatalf("NextTask: %v", err)
	}
	if next == nil {
		t.Fatal("NextTask returned nil, expected child2 (dep satisfied by abandonment)")
	}
	if next.ID != child2 {
		t.Fatalf("NextTask = %s, want %s", next.ID, child2)
	}
}

// TestNext_NoPendingTasks verifies that NextTask returns nil when all
// leaf tasks have been completed.
func TestNext_NoPendingTasks(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "nopending-epic"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	body := "# Task 1\n---\n# Task 2"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID), true); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	child1 := TaskID(epicID + ":1")
	child2 := TaskID(epicID + ":2")

	// Complete both children.
	if err := TransitionStatus(ctx, conn, q, child1, StatusActive, ""); err != nil {
		t.Fatalf("start child1: %v", err)
	}
	if err := TransitionStatus(ctx, conn, q, child1, StatusDone, ""); err != nil {
		t.Fatalf("complete child1: %v", err)
	}
	if err := TransitionStatus(ctx, conn, q, child2, StatusActive, ""); err != nil {
		t.Fatalf("start child2: %v", err)
	}
	if err := TransitionStatus(ctx, conn, q, child2, StatusDone, ""); err != nil {
		t.Fatalf("complete child2: %v", err)
	}

	// NextTask should return nil.
	next, err := NextTask(ctx, conn, q, epicID)
	if err != nil {
		t.Fatalf("NextTask: %v", err)
	}
	if next != nil {
		t.Fatalf("NextTask = %s, want nil (all tasks completed)", next.ID)
	}
}
