package epic

import (
	"context"
	"database/sql"
	"testing"

	"github.com/MHmorgan/agent-epics/db"
)

func TestSplitTask_ThreeSections(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	// Insert a leaf with a body containing two separators (three sections).
	body := "# Setup\nInstall deps\n---\n# Build\nRun make\n---\n# Deploy\nShip it"
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       "proj",
		Status:   sql.NullString{String: "pending", Valid: true},
		Title:    sql.NullString{String: "proj", Valid: true},
		Body:     sql.NullString{String: body, Valid: true},
		Position: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	if err := SplitTask(ctx, conn, q, "proj"); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Verify 3 children exist.
	children, err := q.ListAllTasksByParent(ctx, sql.NullString{String: "proj", Valid: true})
	if err != nil {
		t.Fatalf("list children: %v", err)
	}
	if len(children) != 3 {
		t.Fatalf("got %d children, want 3", len(children))
	}

	// Verify IDs, titles, bodies, and positions.
	wantIDs := []string{"proj:1", "proj:2", "proj:3"}
	wantTitles := []string{"Setup", "Build", "Deploy"}
	wantBodies := []string{"# Setup\nInstall deps", "# Build\nRun make", "# Deploy\nShip it"}

	for i, child := range children {
		if child.ID != wantIDs[i] {
			t.Errorf("child[%d].ID = %q, want %q", i, child.ID, wantIDs[i])
		}
		if child.Title.String != wantTitles[i] {
			t.Errorf("child[%d].Title = %q, want %q", i, child.Title.String, wantTitles[i])
		}
		if child.Body.String != wantBodies[i] {
			t.Errorf("child[%d].Body = %q, want %q", i, child.Body.String, wantBodies[i])
		}
		wantPos := int64(i + 1)
		if child.Position.Int64 != wantPos {
			t.Errorf("child[%d].Position = %d, want %d", i, child.Position.Int64, wantPos)
		}
		if child.Status.String != "pending" {
			t.Errorf("child[%d].Status = %q, want %q", i, child.Status.String, "pending")
		}

		// Each child should have a system record "created".
		recs, err := q.ListRecordsByTask(ctx, child.ID)
		if err != nil {
			t.Fatalf("list records for %s: %v", child.ID, err)
		}
		if len(recs) != 1 || recs[0].Source != "system" || recs[0].Text != "created" {
			t.Errorf("child %s records = %+v, want [system/created]", child.ID, recs)
		}
	}
}

func TestSplitTask_ParentStatusClearedBodyPreserved(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	body := "Part A\n---\nPart B"
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       "proj",
		Status:   sql.NullString{String: "pending", Valid: true},
		Title:    sql.NullString{String: "proj", Valid: true},
		Body:     sql.NullString{String: body, Valid: true},
		Position: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	if err := SplitTask(ctx, conn, q, "proj"); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Parent status should be cleared (NULL).
	parent, err := q.GetTask(ctx, "proj")
	if err != nil {
		t.Fatalf("get parent: %v", err)
	}
	if parent.Status.Valid {
		t.Errorf("parent status = %q, want NULL", parent.Status.String)
	}

	// Body should be preserved (not cleared).
	if parent.Body.String != body {
		t.Errorf("parent body = %q, want %q", parent.Body.String, body)
	}

	// Parent should have a system record about the split.
	recs, err := q.ListRecordsByTask(ctx, "proj")
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	found := false
	for _, r := range recs {
		if r.Source == "system" && r.Text == "split into 2 children" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no system record 'split into 2 children' found; got %+v", recs)
	}
}

func TestSplitTask_RejectNonLeaf(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	// Create a branch with a child — already split.
	insertBranch(t, ctx, q, "proj")
	insertLeaf(t, ctx, q, "proj:1", "pending", "proj")

	err := SplitTask(ctx, conn, q, "proj")
	if err == nil {
		t.Fatal("expected error splitting a branch, got nil")
	}
}

func TestSplitTask_RejectActiveStatus(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	body := "A\n---\nB"
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       "proj",
		Status:   sql.NullString{String: "active", Valid: true},
		Title:    sql.NullString{String: "proj", Valid: true},
		Body:     sql.NullString{String: body, Valid: true},
		Position: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	err = SplitTask(ctx, conn, q, "proj")
	if err == nil {
		t.Fatal("expected error splitting an active task, got nil")
	}
}

func TestSplitTask_RejectNoSeparators(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	// Body with no --- separator.
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       "proj",
		Status:   sql.NullString{String: "pending", Valid: true},
		Title:    sql.NullString{String: "proj", Valid: true},
		Body:     sql.NullString{String: "just a single block", Valid: true},
		Position: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	err = SplitTask(ctx, conn, q, "proj")
	if err == nil {
		t.Fatal("expected error splitting body with no separators, got nil")
	}
}

func TestUnsplitTask_ReversesCleanSplit(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	// Create a task, split it, then unsplit.
	body := "# Alpha\nFirst\n---\n# Beta\nSecond\n---\n# Gamma\nThird"
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       "proj",
		Status:   sql.NullString{String: "pending", Valid: true},
		Title:    sql.NullString{String: "proj", Valid: true},
		Body:     sql.NullString{String: body, Valid: true},
		Position: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	if err := SplitTask(ctx, conn, q, "proj"); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Verify it is a branch before unsplitting.
	children, err := q.ListAllTasksByParent(ctx, sql.NullString{String: "proj", Valid: true})
	if err != nil {
		t.Fatalf("list children: %v", err)
	}
	if len(children) != 3 {
		t.Fatalf("expected 3 children after split, got %d", len(children))
	}

	// Unsplit.
	if err := UnsplitTask(ctx, conn, q, "proj"); err != nil {
		t.Fatalf("UnsplitTask: %v", err)
	}

	// Verify children are gone.
	children, err = q.ListAllTasksByParent(ctx, sql.NullString{String: "proj", Valid: true})
	if err != nil {
		t.Fatalf("list children after unsplit: %v", err)
	}
	if len(children) != 0 {
		t.Errorf("expected 0 children after unsplit, got %d", len(children))
	}

	// Verify parent status restored to pending.
	parent, err := q.GetTask(ctx, "proj")
	if err != nil {
		t.Fatalf("get parent: %v", err)
	}
	if parent.Status.String != "pending" {
		t.Errorf("parent status = %q, want %q", parent.Status.String, "pending")
	}

	// Verify system record about unsplit.
	recs, err := q.ListRecordsByTask(ctx, "proj")
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	found := false
	for _, r := range recs {
		if r.Source == "system" && r.Text == "unsplit from 3 children" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no system record 'unsplit from 3 children'; got %+v", recs)
	}

	// Verify body is still preserved.
	if parent.Body.String != body {
		t.Errorf("parent body = %q, want %q", parent.Body.String, body)
	}
}

func TestUnsplitTask_RejectNonPendingChild(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	body := "A\n---\nB"
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       "proj",
		Status:   sql.NullString{String: "pending", Valid: true},
		Title:    sql.NullString{String: "proj", Valid: true},
		Body:     sql.NullString{String: body, Valid: true},
		Position: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	if err := SplitTask(ctx, conn, q, "proj"); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Move one child to active.
	err = q.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		Status: sql.NullString{String: "active", Valid: true},
		ID:     "proj:1",
	})
	if err != nil {
		t.Fatalf("update status: %v", err)
	}

	err = UnsplitTask(ctx, conn, q, "proj")
	if err == nil {
		t.Fatal("expected error unsplitting with non-pending child, got nil")
	}
}

func TestUnsplitTask_RejectChildWithContext(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	body := "A\n---\nB"
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       "proj",
		Status:   sql.NullString{String: "pending", Valid: true},
		Title:    sql.NullString{String: "proj", Valid: true},
		Body:     sql.NullString{String: body, Valid: true},
		Position: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	if err := SplitTask(ctx, conn, q, "proj"); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Set context on one child.
	err = q.UpdateTaskContext(ctx, db.UpdateTaskContextParams{
		Context: sql.NullString{String: "some context", Valid: true},
		ID:      "proj:1",
	})
	if err != nil {
		t.Fatalf("update context: %v", err)
	}

	err = UnsplitTask(ctx, conn, q, "proj")
	if err == nil {
		t.Fatal("expected error unsplitting with child that has context, got nil")
	}
}

func TestUnsplitTask_RejectChildWithAgentRecords(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	body := "A\n---\nB"
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       "proj",
		Status:   sql.NullString{String: "pending", Valid: true},
		Title:    sql.NullString{String: "proj", Valid: true},
		Body:     sql.NullString{String: body, Valid: true},
		Position: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	if err := SplitTask(ctx, conn, q, "proj"); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Add an agent record on one child.
	err = q.InsertRecord(ctx, db.InsertRecordParams{
		Task:   "proj:2",
		Source: "agent",
		Text:   "started working on this",
	})
	if err != nil {
		t.Fatalf("insert record: %v", err)
	}

	err = UnsplitTask(ctx, conn, q, "proj")
	if err == nil {
		t.Fatal("expected error unsplitting with child that has agent records, got nil")
	}
}
