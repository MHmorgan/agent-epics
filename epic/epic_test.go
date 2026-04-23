package epic

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/MHmorgan/agent-epics/db"
)

func TestValidateEpicID(t *testing.T) {
	valid := []string{
		"myepic",
		"a",
		"abc-123",
		"a1",
		"my-long-epic-name",
	}
	for _, id := range valid {
		if err := ValidateEpicID(id); err != nil {
			t.Errorf("ValidateEpicID(%q) = %v, want nil", id, err)
		}
	}

	invalid := []string{
		"",
		"1abc",          // starts with digit
		"ABC",           // uppercase
		"my.epic",       // dot
		"my/epic",       // slash
		"my\\epic",      // backslash
		"my:epic",       // colon
		"../malicious",  // path traversal
		"-leading-dash", // starts with hyphen
		"MY-EPIC",       // uppercase
		"my epic",       // space
	}
	for _, id := range invalid {
		if err := ValidateEpicID(id); err == nil {
			t.Errorf("ValidateEpicID(%q) = nil, want error", id)
		}
	}
}

func TestNewEpic(t *testing.T) {
	dir := t.TempDir()

	// Create a new epic.
	if err := NewEpic("test-epic", dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}

	// DB file should exist.
	p := filepath.Join(dir, "test-epic.db")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("DB file not created: %v", err)
	}

	// Open and verify root task exists with status=pending.
	conn, q, err := OpenEpic("test-epic", dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	task, err := q.GetTask(ctx, "test-epic")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.ID != "test-epic" {
		t.Errorf("root task ID = %q, want %q", task.ID, "test-epic")
	}
	if !task.Status.Valid || task.Status.String != "pending" {
		t.Errorf("root task status = %v, want pending", task.Status)
	}

	// Verify system record was written.
	records, err := q.ListRecordsByTask(ctx, "test-epic")
	if err != nil {
		t.Fatalf("ListRecordsByTask: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Source != "system" {
		t.Errorf("record source = %q, want %q", records[0].Source, "system")
	}
	if records[0].Text != "created" {
		t.Errorf("record text = %q, want %q", records[0].Text, "created")
	}
}

func TestNewEpicRejectsDuplicate(t *testing.T) {
	dir := t.TempDir()

	if err := NewEpic("dup", dir); err != nil {
		t.Fatalf("first NewEpic: %v", err)
	}
	if err := NewEpic("dup", dir); err == nil {
		t.Error("second NewEpic should have returned error for duplicate")
	}
}

func TestOpenEpic(t *testing.T) {
	dir := t.TempDir()

	if err := NewEpic("open-test", dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}

	conn, q, err := OpenEpic("open-test", dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Verify we can query with the returned connection.
	ctx := context.Background()
	tasks, err := q.ListAllTasks(ctx)
	if err != nil {
		t.Fatalf("ListAllTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
}

func TestOpenEpicNotFound(t *testing.T) {
	dir := t.TempDir()

	_, _, err := OpenEpic("nonexistent", dir)
	if err == nil {
		t.Error("OpenEpic should have returned error for nonexistent epic")
	}
}

func TestRemoveEpic(t *testing.T) {
	dir := t.TempDir()

	if err := NewEpic("rm-test", dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}

	if err := RemoveEpic("rm-test", dir); err != nil {
		t.Fatalf("RemoveEpic: %v", err)
	}

	p := filepath.Join(dir, "rm-test.db")
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("DB file should have been deleted")
	}
}

func TestPathTraversalRejected(t *testing.T) {
	dir := t.TempDir()

	malicious := []string{
		"../malicious",
		"../../etc",
		"a/b",
		"a.b",
		"a:b",
	}
	for _, id := range malicious {
		if err := NewEpic(id, dir); err == nil {
			t.Errorf("NewEpic(%q) should have been rejected", id)
		}
		if _, _, err := OpenEpic(id, dir); err == nil {
			t.Errorf("OpenEpic(%q) should have been rejected", id)
		}
		if err := RemoveEpic(id, dir); err == nil {
			t.Errorf("RemoveEpic(%q) should have been rejected", id)
		}
	}
}

func TestListEpics(t *testing.T) {
	dir := t.TempDir()

	// Empty directory should return empty list.
	epics, err := ListEpics(dir)
	if err != nil {
		t.Fatalf("ListEpics on empty dir: %v", err)
	}
	if len(epics) != 0 {
		t.Fatalf("expected 0 epics, got %d", len(epics))
	}

	// Create two epics.
	if err := NewEpic("alpha", dir); err != nil {
		t.Fatalf("NewEpic alpha: %v", err)
	}
	if err := NewEpic("beta", dir); err != nil {
		t.Fatalf("NewEpic beta: %v", err)
	}

	epics, err = ListEpics(dir)
	if err != nil {
		t.Fatalf("ListEpics: %v", err)
	}
	if len(epics) != 2 {
		t.Fatalf("expected 2 epics, got %d", len(epics))
	}

	// Both should be pending (single root leaf in pending state).
	for _, e := range epics {
		if e.Status != StatusPending {
			t.Errorf("epic %q status = %q, want %q", e.ID, e.Status, StatusPending)
		}
	}
}

func TestPurgeTerminalEpics(t *testing.T) {
	dir := t.TempDir()

	// Create three epics.
	for _, id := range []string{"active-epic", "done-epic", "abandoned-epic"} {
		if err := NewEpic(id, dir); err != nil {
			t.Fatalf("NewEpic %q: %v", id, err)
		}
	}

	// Move done-epic's root to done (pending -> active -> done).
	ctx := context.Background()

	setRootStatus := func(epicID string, status Status) {
		conn, q, err := OpenEpic(epicID, dir)
		if err != nil {
			t.Fatalf("OpenEpic %q: %v", epicID, err)
		}
		defer conn.Close()
		err = q.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
			Status: sql.NullString{String: string(status), Valid: true},
			ID:     epicID,
		})
		if err != nil {
			t.Fatalf("UpdateTaskStatus %q: %v", epicID, err)
		}
	}

	setRootStatus("active-epic", StatusActive)
	setRootStatus("done-epic", StatusDone)
	setRootStatus("abandoned-epic", StatusAbandoned)

	purged, err := PurgeTerminalEpics(dir)
	if err != nil {
		t.Fatalf("PurgeTerminalEpics: %v", err)
	}

	if len(purged) != 2 {
		t.Fatalf("expected 2 purged, got %d: %v", len(purged), purged)
	}

	// active-epic should still exist.
	remaining, err := ListEpics(dir)
	if err != nil {
		t.Fatalf("ListEpics after purge: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining, got %d", len(remaining))
	}
	if remaining[0].ID != "active-epic" {
		t.Errorf("remaining epic = %q, want %q", remaining[0].ID, "active-epic")
	}
}
