package epic

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/MHmorgan/agent-epics/db"
)

// setupTestDB creates a temporary SQLite database and returns a Queries handle.
func setupTestDB(t *testing.T) *db.Queries {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	conn, err := db.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return db.Q(conn)
}

// insertTask is a test helper that inserts a task with the given fields.
func insertTask(t *testing.T, ctx context.Context, q *db.Queries, id, parentID, title, status string) {
	t.Helper()
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       id,
		ParentID: sql.NullString{String: parentID, Valid: parentID != ""},
		Title:    sql.NullString{String: title, Valid: title != ""},
		Status:   sql.NullString{String: status, Valid: status != ""},
	})
	if err != nil {
		t.Fatalf("insert task %s: %v", id, err)
	}
}

func TestGetTask_Existing(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTask(t, ctx, q, "proj", "", "Project", "")
	insertTask(t, ctx, q, "proj:1", "proj", "First task", "pending")

	// proj is a branch (has child proj:1).
	task, err := GetTask(ctx, q, "proj")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.ID != "proj" {
		t.Errorf("ID = %q, want %q", task.ID, "proj")
	}
	if task.IsLeaf {
		t.Error("proj should not be a leaf (it has children)")
	}

	// proj:1 is a leaf (no children).
	task, err = GetTask(ctx, q, "proj:1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.ID != "proj:1" {
		t.Errorf("ID = %q, want %q", task.ID, "proj:1")
	}
	if !task.IsLeaf {
		t.Error("proj:1 should be a leaf")
	}
	if task.Status != StatusPending {
		t.Errorf("Status = %q, want %q", task.Status, StatusPending)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	_, err := GetTask(ctx, q, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent task, got nil")
	}
}

func TestListTasks_ByParent(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTask(t, ctx, q, "proj", "", "Project", "")
	insertTask(t, ctx, q, "proj:1", "proj", "Task one", "pending")
	insertTask(t, ctx, q, "proj:2", "proj", "Task two", "done")
	insertTask(t, ctx, q, "proj:3", "proj", "Task three", "active")

	// Without terminal: should exclude done tasks (proj:2).
	tasks, err := ListTasks(ctx, q, "proj", false)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	ids := taskIDs(tasks)
	if contains(ids, "proj:2") {
		t.Error("expected proj:2 (done) to be excluded when includeTerminal=false")
	}
	if !contains(ids, "proj:1") || !contains(ids, "proj:3") {
		t.Errorf("expected proj:1 and proj:3 in results, got %v", ids)
	}

	// With terminal: should include all children.
	tasks, err = ListTasks(ctx, q, "proj", true)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	ids = taskIDs(tasks)
	if len(ids) != 3 {
		t.Errorf("expected 3 tasks, got %d: %v", len(ids), ids)
	}
}

func TestListTasks_BranchGetsDerivedStatus(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	// Structure: proj -> proj:1 (branch) -> proj:1:1 (active), proj:1:2 (pending)
	insertTask(t, ctx, q, "proj", "", "Project", "")
	insertTask(t, ctx, q, "proj:1", "proj", "Sub-epic", "")    // branch, no explicit status
	insertTask(t, ctx, q, "proj:1:1", "proj:1", "Leaf A", "active")
	insertTask(t, ctx, q, "proj:1:2", "proj:1", "Leaf B", "pending")

	tasks, err := ListTasks(ctx, q, "proj", true)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 child of proj, got %d", len(tasks))
	}
	// proj:1 is a branch; derived status should be active (active > pending).
	if tasks[0].Status != StatusActive {
		t.Errorf("derived status = %q, want %q", tasks[0].Status, StatusActive)
	}
}

func TestGetDerivedStatus(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	// Tree: root -> A (branch) -> A:1 (done), A:2 (pending)
	//             -> B (leaf, active)
	insertTask(t, ctx, q, "root", "", "Root", "")
	insertTask(t, ctx, q, "root:a", "root", "Branch A", "")
	insertTask(t, ctx, q, "root:a:1", "root:a", "Leaf A1", "done")
	insertTask(t, ctx, q, "root:a:2", "root:a", "Leaf A2", "pending")
	insertTask(t, ctx, q, "root:b", "root", "Leaf B", "active")

	// Derived status of root:a should be pending (mix of done + pending).
	s, err := GetDerivedStatus(ctx, q, "root:a")
	if err != nil {
		t.Fatalf("GetDerivedStatus(root:a): %v", err)
	}
	if s != StatusPending {
		t.Errorf("root:a derived = %q, want %q", s, StatusPending)
	}

	// Derived status of root should be active (leaves: done, pending, active -> active wins).
	s, err = GetDerivedStatus(ctx, q, "root")
	if err != nil {
		t.Fatalf("GetDerivedStatus(root): %v", err)
	}
	if s != StatusActive {
		t.Errorf("root derived = %q, want %q", s, StatusActive)
	}
}

func TestGetDerivedStatus_AllDone(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTask(t, ctx, q, "proj", "", "Project", "")
	insertTask(t, ctx, q, "proj:1", "proj", "Task 1", "done")
	insertTask(t, ctx, q, "proj:2", "proj", "Task 2", "done")

	s, err := GetDerivedStatus(ctx, q, "proj")
	if err != nil {
		t.Fatalf("GetDerivedStatus: %v", err)
	}
	if s != StatusDone {
		t.Errorf("derived = %q, want %q", s, StatusDone)
	}
}

func TestGetDerivedStatus_AllAbandoned(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTask(t, ctx, q, "proj", "", "Project", "")
	insertTask(t, ctx, q, "proj:1", "proj", "Task 1", "abandoned")
	insertTask(t, ctx, q, "proj:2", "proj", "Task 2", "abandoned")

	s, err := GetDerivedStatus(ctx, q, "proj")
	if err != nil {
		t.Fatalf("GetDerivedStatus: %v", err)
	}
	if s != StatusAbandoned {
		t.Errorf("derived = %q, want %q", s, StatusAbandoned)
	}
}

func TestGetDerivedStatus_NoLeaves(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTask(t, ctx, q, "proj", "", "Project", "")

	s, err := GetDerivedStatus(ctx, q, "proj")
	if err != nil {
		t.Fatalf("GetDerivedStatus: %v", err)
	}
	if s != StatusPending {
		t.Errorf("derived = %q, want %q (no leaves)", s, StatusPending)
	}
}

// --- helpers ---

func taskIDs(tasks []Task) []string {
	ids := make([]string, len(tasks))
	for i, t := range tasks {
		ids[i] = string(t.ID)
	}
	return ids
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
