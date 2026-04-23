package epic

import (
	"context"
	"testing"

	"github.com/MHmorgan/agent-epics/db"
)

func insertTestRecord(t *testing.T, q *db.Queries, taskID, source, text string) {
	t.Helper()
	err := q.InsertRecord(context.Background(), db.InsertRecordParams{
		Task:   taskID,
		Source: source,
		Text:   text,
	})
	if err != nil {
		t.Fatalf("insert record for %s: %v", taskID, err)
	}
}

func TestGetRecords_SubtreeIncludesParentAndChild(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTask(t, ctx, q, "proj", "", "Project", "")
	insertTask(t, ctx, q, "proj:1", "proj", "First task", "pending")

	insertTestRecord(t, q, "proj", "agent", "parent note")
	insertTestRecord(t, q, "proj:1", "agent", "child note")

	records, err := GetRecords(ctx, q, "proj", false)
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	texts := map[string]bool{}
	for _, r := range records {
		texts[r.Text] = true
	}
	if !texts["parent note"] {
		t.Error("missing parent record")
	}
	if !texts["child note"] {
		t.Error("missing child record")
	}
}

func TestGetRecords_SelfOnlyExcludesChildren(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTask(t, ctx, q, "proj", "", "Project", "")
	insertTask(t, ctx, q, "proj:1", "proj", "First task", "pending")

	insertTestRecord(t, q, "proj", "agent", "parent note")
	insertTestRecord(t, q, "proj:1", "agent", "child note")

	records, err := GetRecords(ctx, q, "proj", true)
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Text != "parent note" {
		t.Errorf("expected %q, got %q", "parent note", records[0].Text)
	}
	if records[0].Task != "proj" {
		t.Errorf("expected task ID %q, got %q", "proj", records[0].Task)
	}
}

func TestGetRecords_EmptyResult(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTask(t, ctx, q, "proj", "", "Project", "")

	records, err := GetRecords(ctx, q, "proj", false)
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestGetRecords_SubtreeDoesNotMatchSiblings(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTask(t, ctx, q, "proj", "", "Project", "")
	insertTask(t, ctx, q, "proj:1", "proj", "Task one", "pending")
	insertTask(t, ctx, q, "proj:2", "proj", "Task two", "pending")

	insertTestRecord(t, q, "proj:1", "agent", "task 1 note")
	insertTestRecord(t, q, "proj:2", "agent", "task 2 note")

	// Subtree query for proj:1 should not include proj:2's records.
	records, err := GetRecords(ctx, q, "proj:1", false)
	if err != nil {
		t.Fatalf("GetRecords: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Text != "task 1 note" {
		t.Errorf("expected %q, got %q", "task 1 note", records[0].Text)
	}
}
