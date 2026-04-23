package epic

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/MHmorgan/agent-epics/db"
)

// openTestDB creates a fresh in-memory-like SQLite DB in a temp dir,
// applies migrations, and returns the connection and queries handle.
func openTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn, db.Q(conn)
}

// insertTestTask is a helper that inserts a task with the given id, parent, and position.
func insertTestTask(t *testing.T, ctx context.Context, q *db.Queries, id string, parentID string, position int64) {
	t.Helper()
	var parent sql.NullString
	if parentID != "" {
		parent = sql.NullString{String: parentID, Valid: true}
	}
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       id,
		ParentID: parent,
		Status:   sql.NullString{String: "pending", Valid: true},
		Position: sql.NullInt64{Int64: position, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task %s: %v", id, err)
	}
}

func TestAddDependency_ValidSiblings(t *testing.T) {
	_, q := openTestDB(t)
	ctx := context.Background()

	// Create parent and two sibling tasks.
	insertTestTask(t, ctx, q, "epic", "", 0)
	insertTestTask(t, ctx, q, "epic:1", "epic", 1)
	insertTestTask(t, ctx, q, "epic:2", "epic", 2)

	err := AddDependency(ctx, q, TaskID("epic:2"), TaskID("epic:1"))
	if err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	// Verify the dep was inserted.
	deps, err := q.ListDepsForTask(ctx, "epic:2")
	if err != nil {
		t.Fatalf("ListDepsForTask: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(deps))
	}
	if deps[0].AfterID != "epic:1" {
		t.Errorf("expected after_id=epic:1, got %s", deps[0].AfterID)
	}
}

func TestAddDependency_NonSiblings(t *testing.T) {
	_, q := openTestDB(t)
	ctx := context.Background()

	// Create two tasks with different parents.
	insertTestTask(t, ctx, q, "epic", "", 0)
	insertTestTask(t, ctx, q, "epic:1", "epic", 1)
	insertTestTask(t, ctx, q, "epic:2", "epic", 2)
	// Split epic:1 into children.
	insertTestTask(t, ctx, q, "epic:1:1", "epic:1", 1)

	// epic:1:1 and epic:2 have different parents.
	err := AddDependency(ctx, q, TaskID("epic:1:1"), TaskID("epic:2"))
	if err == nil {
		t.Fatal("expected error for non-siblings, got nil")
	}
}

func TestAddDependency_DirectCycle(t *testing.T) {
	_, q := openTestDB(t)
	ctx := context.Background()

	insertTestTask(t, ctx, q, "epic", "", 0)
	insertTestTask(t, ctx, q, "epic:1", "epic", 1)
	insertTestTask(t, ctx, q, "epic:2", "epic", 2)

	// A after B.
	err := AddDependency(ctx, q, TaskID("epic:2"), TaskID("epic:1"))
	if err != nil {
		t.Fatalf("first AddDependency: %v", err)
	}

	// B after A -- should fail (direct cycle).
	err = AddDependency(ctx, q, TaskID("epic:1"), TaskID("epic:2"))
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
}

func TestAddDependency_TransitiveCycle(t *testing.T) {
	_, q := openTestDB(t)
	ctx := context.Background()

	insertTestTask(t, ctx, q, "epic", "", 0)
	insertTestTask(t, ctx, q, "epic:1", "epic", 1)
	insertTestTask(t, ctx, q, "epic:2", "epic", 2)
	insertTestTask(t, ctx, q, "epic:3", "epic", 3)

	// A after B.
	if err := AddDependency(ctx, q, TaskID("epic:2"), TaskID("epic:1")); err != nil {
		t.Fatalf("dep 1: %v", err)
	}
	// B after C.
	if err := AddDependency(ctx, q, TaskID("epic:3"), TaskID("epic:2")); err != nil {
		t.Fatalf("dep 2: %v", err)
	}
	// C after A -- should fail (transitive cycle: A -> B -> C -> A).
	err := AddDependency(ctx, q, TaskID("epic:1"), TaskID("epic:3"))
	if err == nil {
		t.Fatal("expected transitive cycle error, got nil")
	}
}

func TestRemoveDependency(t *testing.T) {
	_, q := openTestDB(t)
	ctx := context.Background()

	insertTestTask(t, ctx, q, "epic", "", 0)
	insertTestTask(t, ctx, q, "epic:1", "epic", 1)
	insertTestTask(t, ctx, q, "epic:2", "epic", 2)

	if err := AddDependency(ctx, q, TaskID("epic:2"), TaskID("epic:1")); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	if err := RemoveDependency(ctx, q, TaskID("epic:2"), TaskID("epic:1")); err != nil {
		t.Fatalf("RemoveDependency: %v", err)
	}

	// Verify dep is gone.
	deps, err := q.ListDepsForTask(ctx, "epic:2")
	if err != nil {
		t.Fatalf("ListDepsForTask: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 deps after removal, got %d", len(deps))
	}
}

func TestAddDependency_SystemRecords(t *testing.T) {
	_, q := openTestDB(t)
	ctx := context.Background()

	insertTestTask(t, ctx, q, "epic", "", 0)
	insertTestTask(t, ctx, q, "epic:1", "epic", 1)
	insertTestTask(t, ctx, q, "epic:2", "epic", 2)

	if err := AddDependency(ctx, q, TaskID("epic:2"), TaskID("epic:1")); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	records, err := q.ListRecordsByTask(ctx, "epic:2")
	if err != nil {
		t.Fatalf("ListRecordsByTask: %v", err)
	}

	found := false
	for _, r := range records {
		if r.Source == "system" && r.Text == "after epic:1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected system record 'after epic:1' not found")
	}

	// Now remove and check unafter record.
	if err := RemoveDependency(ctx, q, TaskID("epic:2"), TaskID("epic:1")); err != nil {
		t.Fatalf("RemoveDependency: %v", err)
	}

	records, err = q.ListRecordsByTask(ctx, "epic:2")
	if err != nil {
		t.Fatalf("ListRecordsByTask: %v", err)
	}

	found = false
	for _, r := range records {
		if r.Source == "system" && r.Text == "unafter epic:1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected system record 'unafter epic:1' not found")
	}
}
