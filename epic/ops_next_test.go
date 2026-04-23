package epic

import (
	"context"
	"testing"
)

func TestNextTask_SinglePendingLeafNoDeps(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "epic")
	insertLeaf(t, ctx, q, "epic:1", "pending", "epic")

	got, err := NextTask(ctx, conn, q, "epic")
	if err != nil {
		t.Fatalf("NextTask: %v", err)
	}
	if got == nil {
		t.Fatal("expected a task, got nil")
	}
	if got.ID != "epic:1" {
		t.Errorf("expected epic:1, got %s", got.ID)
	}
}

func TestNextTask_MultiplePendingLeaves_ReturnsByPosition(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "epic")
	insertLeafAt(t, ctx, q, "epic:1", "pending", "epic", 2)
	insertLeafAt(t, ctx, q, "epic:2", "pending", "epic", 1)

	got, err := NextTask(ctx, conn, q, "epic")
	if err != nil {
		t.Fatalf("NextTask: %v", err)
	}
	if got == nil {
		t.Fatal("expected a task, got nil")
	}
	if got.ID != "epic:2" {
		t.Errorf("expected epic:2 (lower position), got %s", got.ID)
	}
}

func TestNextTask_PendingLeafWithSatisfiedDep(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "epic")
	insertLeafAt(t, ctx, q, "epic:1", "done", "epic", 1)
	insertLeafAt(t, ctx, q, "epic:2", "pending", "epic", 2)

	// epic:2 depends on epic:1 (which is done).
	if err := AddDependency(ctx, q, TaskID("epic:2"), TaskID("epic:1")); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	got, err := NextTask(ctx, conn, q, "epic")
	if err != nil {
		t.Fatalf("NextTask: %v", err)
	}
	if got == nil {
		t.Fatal("expected a task, got nil")
	}
	if got.ID != "epic:2" {
		t.Errorf("expected epic:2, got %s", got.ID)
	}
}

func TestNextTask_PendingLeafWithUnsatisfiedDep_SkipsToNext(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "epic")
	insertLeafAt(t, ctx, q, "epic:1", "active", "epic", 1)
	insertLeafAt(t, ctx, q, "epic:2", "pending", "epic", 2)
	insertLeafAt(t, ctx, q, "epic:3", "pending", "epic", 3)

	// epic:2 depends on epic:1 (still active, not terminal).
	if err := AddDependency(ctx, q, TaskID("epic:2"), TaskID("epic:1")); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	got, err := NextTask(ctx, conn, q, "epic")
	if err != nil {
		t.Fatalf("NextTask: %v", err)
	}
	if got == nil {
		t.Fatal("expected a task, got nil")
	}
	// epic:2 is blocked by dep on epic:1, so epic:3 should be returned.
	if got.ID != "epic:3" {
		t.Errorf("expected epic:3, got %s", got.ID)
	}
}

func TestNextTask_AllTerminal_ReturnsNil(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "epic")
	insertLeaf(t, ctx, q, "epic:1", "done", "epic")
	insertLeaf(t, ctx, q, "epic:2", "abandoned", "epic")

	got, err := NextTask(ctx, conn, q, "epic")
	if err != nil {
		t.Fatalf("NextTask: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %s", got.ID)
	}
}

func TestNextTask_AllPendingBlocked_ReturnsNil(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "epic")
	insertLeafAt(t, ctx, q, "epic:1", "active", "epic", 1)
	insertLeafAt(t, ctx, q, "epic:2", "pending", "epic", 2)
	insertLeafAt(t, ctx, q, "epic:3", "pending", "epic", 3)

	// Both pending leaves depend on epic:1 which is still active.
	if err := AddDependency(ctx, q, TaskID("epic:2"), TaskID("epic:1")); err != nil {
		t.Fatalf("AddDependency epic:2: %v", err)
	}
	if err := AddDependency(ctx, q, TaskID("epic:3"), TaskID("epic:1")); err != nil {
		t.Fatalf("AddDependency epic:3: %v", err)
	}

	got, err := NextTask(ctx, conn, q, "epic")
	if err != nil {
		t.Fatalf("NextTask: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %s", got.ID)
	}
}

func TestNextTask_BranchPredecessorWithMixedLeaves(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	// epic (root branch) has two children: epic:1 (branch) and epic:2 (pending leaf).
	// epic:1 has two leaf children: epic:1:1 (done) and epic:1:2 (pending).
	// epic:2 depends on epic:1. Since epic:1:2 is still pending, dep is unsatisfied.
	insertBranch(t, ctx, q, "epic")
	insertBranchAt(t, ctx, q, "epic:1", "epic", 1)
	insertLeafAt(t, ctx, q, "epic:1:1", "done", "epic:1", 1)
	insertLeafAt(t, ctx, q, "epic:1:2", "pending", "epic:1", 2)
	insertLeafAt(t, ctx, q, "epic:2", "pending", "epic", 2)

	// epic:2 depends on epic:1 (branch predecessor, both are children of "epic").
	if err := AddDependency(ctx, q, TaskID("epic:2"), TaskID("epic:1")); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	got, err := NextTask(ctx, conn, q, "epic")
	if err != nil {
		t.Fatalf("NextTask: %v", err)
	}
	// epic:1:2 is pending with no deps, so it should be returned.
	// epic:2 is blocked by dep on epic:1 (which has non-terminal leaf epic:1:2).
	if got == nil {
		t.Fatal("expected a task, got nil")
	}
	if got.ID != "epic:1:2" {
		t.Errorf("expected epic:1:2, got %s", got.ID)
	}
}
