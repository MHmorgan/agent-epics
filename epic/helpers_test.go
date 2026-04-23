package epic

import (
	"context"
	"database/sql"
	_ "embed"
	"testing"

	"github.com/MHmorgan/agent-epics/db"
	_ "modernc.org/sqlite"
)

//go:embed testdata/schema.sql
var testSchema string

// newTestDB creates an in-memory SQLite database with the schema applied.
func newTestDB(t *testing.T) (*sql.DB, *db.Queries) {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	if _, err := conn.Exec(testSchema); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	return conn, db.New(conn)
}

// insertLeaf inserts a leaf task with the given id, status, and optional parent.
func insertLeaf(t *testing.T, ctx context.Context, q *db.Queries, id string, status string, parentID string) {
	t.Helper()
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       id,
		ParentID: sql.NullString{String: parentID, Valid: parentID != ""},
		Title:    sql.NullString{String: id, Valid: true},
		Status:   sql.NullString{String: status, Valid: true},
		Position: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task %s: %v", id, err)
	}
}

// insertBranch inserts a branch task (no status) with the given id.
func insertBranch(t *testing.T, ctx context.Context, q *db.Queries, id string) {
	t.Helper()
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       id,
		Title:    sql.NullString{String: id, Valid: true},
		Position: sql.NullInt64{Int64: 0, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert branch %s: %v", id, err)
	}
}

// insertBranchAt inserts a branch task (no status) with the given id, parent, and position.
func insertBranchAt(t *testing.T, ctx context.Context, q *db.Queries, id string, parentID string, pos int64) {
	t.Helper()
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       id,
		ParentID: sql.NullString{String: parentID, Valid: parentID != ""},
		Title:    sql.NullString{String: id, Valid: true},
		Position: sql.NullInt64{Int64: pos, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert branch %s: %v", id, err)
	}
}

// insertLeafAt inserts a leaf task at a specific position.
func insertLeafAt(t *testing.T, ctx context.Context, q *db.Queries, id string, status string, parentID string, pos int64) {
	t.Helper()
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       id,
		ParentID: sql.NullString{String: parentID, Valid: parentID != ""},
		Title:    sql.NullString{String: id, Valid: true},
		Status:   sql.NullString{String: status, Valid: true},
		Position: sql.NullInt64{Int64: pos, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task %s: %v", id, err)
	}
}
