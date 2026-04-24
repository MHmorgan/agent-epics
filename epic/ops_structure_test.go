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

	if err := SplitTask(ctx, conn, q, "proj", false); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Verify 2 children exist (first section skipped).
	children, err := q.ListAllTasksByParent(ctx, sql.NullString{String: "proj", Valid: true})
	if err != nil {
		t.Fatalf("list children: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("got %d children, want 2", len(children))
	}

	// Verify IDs, titles, bodies, and positions.
	wantIDs := []string{"proj:1", "proj:2"}
	wantTitles := []string{"Build", "Deploy"}
	wantBodies := []string{"# Build\nRun make", "# Deploy\nShip it"}

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

	if err := SplitTask(ctx, conn, q, "proj", true); err != nil {
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

	err := SplitTask(ctx, conn, q, "proj", true)
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

	err = SplitTask(ctx, conn, q, "proj", true)
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

	err = SplitTask(ctx, conn, q, "proj", true)
	if err == nil {
		t.Fatal("expected error splitting body with no separators, got nil")
	}
}

func TestSplitTask_SkipFirst(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	// Insert a leaf with a body containing two separators (three sections).
	body := "# Desc\nIntro\n---\n# Build\nRun make\n---\n# Deploy\nShip it"
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

	if err := SplitTask(ctx, conn, q, "proj", false); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Verify 2 children exist (first section skipped).
	children, err := q.ListAllTasksByParent(ctx, sql.NullString{String: "proj", Valid: true})
	if err != nil {
		t.Fatalf("list children: %v", err)
	}
	if len(children) != 2 {
		t.Fatalf("got %d children, want 2", len(children))
	}

	// Verify IDs, positions, and titles.
	wantIDs := []string{"proj:1", "proj:2"}
	wantTitles := []string{"Build", "Deploy"}

	for i, child := range children {
		if child.ID != wantIDs[i] {
			t.Errorf("child[%d].ID = %q, want %q", i, child.ID, wantIDs[i])
		}
		wantPos := int64(i + 1)
		if child.Position.Int64 != wantPos {
			t.Errorf("child[%d].Position = %d, want %d", i, child.Position.Int64, wantPos)
		}
		if child.Title.String != wantTitles[i] {
			t.Errorf("child[%d].Title = %q, want %q", i, child.Title.String, wantTitles[i])
		}
	}

	// Parent body unchanged.
	parent, err := q.GetTask(ctx, "proj")
	if err != nil {
		t.Fatalf("get parent: %v", err)
	}
	if parent.Body.String != body {
		t.Errorf("parent body = %q, want %q", parent.Body.String, body)
	}

	// Parent status cleared.
	if parent.Status.Valid {
		t.Errorf("parent status = %q, want NULL", parent.Status.String)
	}
}

func TestSplitTask_RejectTwoSectionsWithSkipFirst(t *testing.T) {
	conn, q := newTestDB(t)
	ctx := context.Background()

	// Insert a leaf with a body containing one separator (two sections).
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

	err = SplitTask(ctx, conn, q, "proj", false)
	if err == nil {
		t.Fatal("expected error splitting 2-section body with keepFirst=false, got nil")
	}
}
