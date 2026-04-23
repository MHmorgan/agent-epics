package epic

import (
	"context"
	"testing"
)

func TestTransitionStatus_PendingToActive(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "proj")
	insertLeaf(t, ctx, q, "proj:1", "pending", "proj")

	err := TransitionStatus(ctx, conn, q, "proj:1", StatusActive, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify status updated.
	task, err := q.GetTask(ctx, "proj:1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.Status.String != "active" {
		t.Errorf("status = %q, want %q", task.Status.String, "active")
	}

	// Verify system record.
	records, err := q.ListRecordsByTask(ctx, "proj:1")
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Source != "system" {
		t.Errorf("source = %q, want %q", records[0].Source, "system")
	}
	want := "status pending \u2192 active"
	if records[0].Text != want {
		t.Errorf("text = %q, want %q", records[0].Text, want)
	}
}

func TestTransitionStatus_ActiveToDone(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "proj")
	insertLeaf(t, ctx, q, "proj:1", "active", "proj")

	err := TransitionStatus(ctx, conn, q, "proj:1", StatusDone, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task, err := q.GetTask(ctx, "proj:1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.Status.String != "done" {
		t.Errorf("status = %q, want %q", task.Status.String, "done")
	}

	records, err := q.ListRecordsByTask(ctx, "proj:1")
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	want := "status active \u2192 done"
	if records[0].Text != want {
		t.Errorf("text = %q, want %q", records[0].Text, want)
	}
}

func TestTransitionStatus_ActiveToBlocked_WithReason(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "proj")
	insertLeaf(t, ctx, q, "proj:1", "active", "proj")

	err := TransitionStatus(ctx, conn, q, "proj:1", StatusBlocked, "waiting for API key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	task, err := q.GetTask(ctx, "proj:1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.Status.String != "blocked" {
		t.Errorf("status = %q, want %q", task.Status.String, "blocked")
	}

	records, err := q.ListRecordsByTask(ctx, "proj:1")
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	want := "status active \u2192 blocked: waiting for API key"
	if records[0].Text != want {
		t.Errorf("text = %q, want %q", records[0].Text, want)
	}
}

func TestTransitionStatus_InvalidTransition_PendingToDone(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "proj")
	insertLeaf(t, ctx, q, "proj:1", "pending", "proj")

	err := TransitionStatus(ctx, conn, q, "proj:1", StatusDone, "")
	if err == nil {
		t.Fatal("expected error for invalid transition pending -> done, got nil")
	}

	// Verify status unchanged.
	task, err := q.GetTask(ctx, "proj:1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.Status.String != "pending" {
		t.Errorf("status = %q, want %q (should be unchanged)", task.Status.String, "pending")
	}
}

func TestTransitionStatus_MissingReasonForBlock(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "proj")
	insertLeaf(t, ctx, q, "proj:1", "active", "proj")

	err := TransitionStatus(ctx, conn, q, "proj:1", StatusBlocked, "")
	if err == nil {
		t.Fatal("expected error for missing reason on block, got nil")
	}

	// Verify status unchanged.
	task, err := q.GetTask(ctx, "proj:1")
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.Status.String != "active" {
		t.Errorf("status = %q, want %q (should be unchanged)", task.Status.String, "active")
	}
}

func TestTransitionStatus_BranchFails(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	// Create a branch with a child.
	insertBranch(t, ctx, q, "proj")
	insertLeaf(t, ctx, q, "proj:1", "pending", "proj")

	err := TransitionStatus(ctx, conn, q, "proj", StatusActive, "")
	if err == nil {
		t.Fatal("expected error for branch transition, got nil")
	}
}
