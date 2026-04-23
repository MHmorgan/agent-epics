package epic

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MHmorgan/agent-epics/db"
)

// SetTaskBody replaces the body of a leaf task. Returns error if task is a branch.
// Writes system record "body set". Uses a transaction.
func SetTaskBody(ctx context.Context, conn *sql.DB, q *db.Queries, id TaskID, body string) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := q.WithTx(tx)

	// Verify task exists.
	if _, err := qtx.GetTask(ctx, id.String()); err != nil {
		return fmt.Errorf("get task %s: %w", id, err)
	}

	// Verify it is a leaf.
	count, err := qtx.CountChildren(ctx, sql.NullString{String: id.String(), Valid: true})
	if err != nil {
		return fmt.Errorf("count children of %s: %w", id, err)
	}
	if count > 0 {
		return fmt.Errorf("task %s is a branch, not a leaf", id)
	}

	if err := qtx.UpdateTaskBody(ctx, db.UpdateTaskBodyParams{
		Body: sql.NullString{String: body, Valid: true},
		ID:   id.String(),
	}); err != nil {
		return fmt.Errorf("update body of %s: %w", id, err)
	}

	if err := addSystemRecord(ctx, qtx, id, "body set"); err != nil {
		return err
	}

	return tx.Commit()
}

// SetTaskContext writes or overwrites the context of any task (leaf or branch, any status).
// Writes system record "context set".
func SetTaskContext(ctx context.Context, conn *sql.DB, q *db.Queries, id TaskID, markdown string) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := q.WithTx(tx)

	// Verify task exists.
	if _, err := qtx.GetTask(ctx, id.String()); err != nil {
		return fmt.Errorf("get task %s: %w", id, err)
	}

	if err := qtx.UpdateTaskContext(ctx, db.UpdateTaskContextParams{
		Context: sql.NullString{String: markdown, Valid: true},
		ID:      id.String(),
	}); err != nil {
		return fmt.Errorf("update context of %s: %w", id, err)
	}

	if err := addSystemRecord(ctx, qtx, id, "context set"); err != nil {
		return err
	}

	return tx.Commit()
}

// AddRecord appends an agent record to a task.
func AddRecord(ctx context.Context, q *db.Queries, id TaskID, text string) error {
	// Verify task exists.
	if _, err := q.GetTask(ctx, id.String()); err != nil {
		return fmt.Errorf("get task %s: %w", id, err)
	}

	return q.InsertRecord(ctx, db.InsertRecordParams{
		Task:   id.String(),
		Source: "agent",
		Text:   text,
	})
}

// AddChild creates a new empty pending leaf under an existing branch.
// Returns the new child's ID. Rejects if parent is a leaf.
// Uses a transaction.
func AddChild(ctx context.Context, conn *sql.DB, q *db.Queries, parentID TaskID) (TaskID, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := q.WithTx(tx)

	// Verify parent exists.
	if _, err := qtx.GetTask(ctx, parentID.String()); err != nil {
		return "", fmt.Errorf("get task %s: %w", parentID, err)
	}

	// Verify parent is a branch (has children).
	count, err := qtx.CountChildren(ctx, sql.NullString{String: parentID.String(), Valid: true})
	if err != nil {
		return "", fmt.Errorf("count children of %s: %w", parentID, err)
	}
	if count == 0 {
		return "", fmt.Errorf("task %s is a leaf, not a branch", parentID)
	}

	// Determine next position from max existing child position + 1.
	maxPosRaw, err := qtx.MaxChildPosition(ctx, sql.NullString{String: parentID.String(), Valid: true})
	if err != nil {
		return "", fmt.Errorf("max child position of %s: %w", parentID, err)
	}
	var maxPos int64
	switch v := maxPosRaw.(type) {
	case int64:
		maxPos = v
	case float64:
		maxPos = int64(v)
	default:
		maxPos = 0
	}
	nextPos := maxPos + 1

	childID := parentID.ChildID(int(nextPos))

	if err := qtx.InsertTask(ctx, db.InsertTaskParams{
		ID:       childID.String(),
		ParentID: sql.NullString{String: parentID.String(), Valid: true},
		Status:   sql.NullString{String: string(StatusPending), Valid: true},
		Position: sql.NullInt64{Int64: nextPos, Valid: true},
	}); err != nil {
		return "", fmt.Errorf("insert child %s: %w", childID, err)
	}

	if err := addSystemRecord(ctx, qtx, childID, "created"); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return childID, nil
}

// addSystemRecord appends a system record to a task.
func addSystemRecord(ctx context.Context, q *db.Queries, id TaskID, text string) error {
	return q.InsertRecord(ctx, db.InsertRecordParams{
		Task:   id.String(),
		Source: "system",
		Text:   text,
	})
}
