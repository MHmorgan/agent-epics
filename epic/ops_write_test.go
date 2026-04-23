package epic

import (
	"context"
	"testing"
)

func TestSetTaskBody_Leaf(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertLeaf(t, ctx, q, "solo", "pending", "")

	if err := SetTaskBody(ctx, conn, q, "solo", "new body"); err != nil {
		t.Fatalf("SetTaskBody on leaf: %v", err)
	}

	task, err := q.GetTask(ctx, "solo")
	if err != nil {
		t.Fatal(err)
	}
	if task.Body.String != "new body" {
		t.Errorf("body = %q, want %q", task.Body.String, "new body")
	}

	// Verify system record was written.
	recs, err := q.ListRecordsByTask(ctx, "solo")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Source != "system" || recs[0].Text != "body set" {
		t.Errorf("expected system record 'body set', got %+v", recs)
	}
}

func TestSetTaskBody_Branch(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertBranch(t, ctx, q, "proj")
	insertLeaf(t, ctx, q, "proj:1", "pending", "proj")

	err := SetTaskBody(ctx, conn, q, "proj", "should fail")
	if err == nil {
		t.Fatal("expected error setting body on branch, got nil")
	}
}

func TestSetTaskContext(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	// Test on a leaf.
	insertLeaf(t, ctx, q, "solo", "pending", "")
	if err := SetTaskContext(ctx, conn, q, "solo", "# Leaf context"); err != nil {
		t.Fatalf("SetTaskContext on leaf: %v", err)
	}
	task, err := q.GetTask(ctx, "solo")
	if err != nil {
		t.Fatal(err)
	}
	if task.Context.String != "# Leaf context" {
		t.Errorf("context = %q, want %q", task.Context.String, "# Leaf context")
	}

	// Test on a branch.
	insertBranch(t, ctx, q, "proj")
	insertLeaf(t, ctx, q, "proj:1", "pending", "proj")
	if err := SetTaskContext(ctx, conn, q, "proj", "# Branch context"); err != nil {
		t.Fatalf("SetTaskContext on branch: %v", err)
	}
	task, err = q.GetTask(ctx, "proj")
	if err != nil {
		t.Fatal(err)
	}
	if task.Context.String != "# Branch context" {
		t.Errorf("context = %q, want %q", task.Context.String, "# Branch context")
	}
}

func TestAddRecord(t *testing.T) {
	_, q := newTestDB(t)
	ctx := context.Background()

	insertLeaf(t, ctx, q, "solo", "pending", "")

	if err := AddRecord(ctx, q, "solo", "first note"); err != nil {
		t.Fatalf("AddRecord: %v", err)
	}
	if err := AddRecord(ctx, q, "solo", "second note"); err != nil {
		t.Fatalf("AddRecord: %v", err)
	}

	recs, err := q.ListRecordsByTask(ctx, "solo")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}
	if recs[0].Source != "agent" || recs[0].Text != "first note" {
		t.Errorf("record[0] = %+v, want agent/first note", recs[0])
	}
	if recs[1].Source != "agent" || recs[1].Text != "second note" {
		t.Errorf("record[1] = %+v, want agent/second note", recs[1])
	}
}

func TestAddChild_Branch(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	// Seed branch with child at position 1 (matching its numeric ID suffix).
	insertBranch(t, ctx, q, "proj")
	insertLeafAt(t, ctx, q, "proj:1", "pending", "proj", 1)

	childID, err := AddChild(ctx, conn, q, "proj")
	if err != nil {
		t.Fatalf("AddChild on branch: %v", err)
	}

	want := TaskID("proj:2")
	if childID != want {
		t.Errorf("child ID = %q, want %q", childID, want)
	}

	// Verify the child was inserted correctly.
	task, err := q.GetTask(ctx, childID.String())
	if err != nil {
		t.Fatalf("get new child: %v", err)
	}
	if task.Status.String != string(StatusPending) {
		t.Errorf("child status = %q, want %q", task.Status.String, StatusPending)
	}
	if task.Position.Int64 != 2 {
		t.Errorf("child position = %d, want 2", task.Position.Int64)
	}

	// Verify system record on the new child.
	recs, err := q.ListRecordsByTask(ctx, childID.String())
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].Source != "system" || recs[0].Text != "created" {
		t.Errorf("expected system record 'created', got %+v", recs)
	}
}

func TestAddChild_Leaf(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	insertLeaf(t, ctx, q, "solo", "pending", "")

	_, err := AddChild(ctx, conn, q, "solo")
	if err == nil {
		t.Fatal("expected error adding child to leaf, got nil")
	}
}
