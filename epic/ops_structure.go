package epic

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MHmorgan/agent-epics/db"
)

// SplitTask splits a leaf task into children based on --- separators.
// Preconditions: task is leaf, status is pending or blocked, body has >= 1 separator.
// Uses a transaction.
func SplitTask(ctx context.Context, conn *sql.DB, q *db.Queries, id TaskID, keepFirst bool) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := q.WithTx(tx)

	// Fetch task and verify it exists.
	task, err := qtx.GetTask(ctx, id.String())
	if err != nil {
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

	// Verify status is pending or blocked.
	status := Status(task.Status.String)
	if status != StatusPending && status != StatusBlocked {
		return fmt.Errorf("task %s has status %q; split requires pending or blocked", id, status)
	}

	// Split the body into sections.
	sections, err := SplitMarkdown(task.Body.String)
	if err != nil {
		return fmt.Errorf("split body of %s: %w", id, err)
	}

	// Determine child sections based on keepFirst flag.
	if !keepFirst && len(sections) < 3 {
		return fmt.Errorf("task %s: need at least 3 sections to split (got %d); use keepfirst to include all sections", id, len(sections))
	}
	childSections := sections
	if !keepFirst {
		childSections = sections[1:]
	}

	// Create children.
	for i, sec := range childSections {
		pos := i + 1
		childID := id.ChildID(pos)

		if err := qtx.InsertTask(ctx, db.InsertTaskParams{
			ID:       childID.String(),
			ParentID: sql.NullString{String: id.String(), Valid: true},
			Title:    sql.NullString{String: sec.Title, Valid: sec.Title != ""},
			Body:     sql.NullString{String: sec.Body, Valid: sec.Body != ""},
			Status:   sql.NullString{String: string(StatusPending), Valid: true},
			Position: sql.NullInt64{Int64: int64(pos), Valid: true},
		}); err != nil {
			return fmt.Errorf("insert child %s: %w", childID, err)
		}

		if err := addSystemRecord(ctx, qtx, childID, "created"); err != nil {
			return err
		}
	}

	// Clear parent status (it is now a branch).
	if err := qtx.ClearTaskStatus(ctx, id.String()); err != nil {
		return fmt.Errorf("clear status of %s: %w", id, err)
	}

	// Write system record on parent.
	msg := fmt.Sprintf("split into %d children", len(childSections))
	if err := addSystemRecord(ctx, qtx, id, msg); err != nil {
		return err
	}

	return tx.Commit()
}

