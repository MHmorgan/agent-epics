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
func SplitTask(ctx context.Context, conn *sql.DB, q *db.Queries, id TaskID) error {
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

	// Create children.
	for n, sec := range sections {
		pos := n + 1 // 1-indexed
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
	msg := fmt.Sprintf("split into %d children", len(sections))
	if err := addSystemRecord(ctx, qtx, id, msg); err != nil {
		return err
	}

	return tx.Commit()
}

// UnsplitTask reverses a split. Preconditions: task is branch, all children are
// pending leaves with no context and no non-system records.
// Uses a transaction.
func UnsplitTask(ctx context.Context, conn *sql.DB, q *db.Queries, id TaskID) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := q.WithTx(tx)

	// Fetch task and verify it exists.
	if _, err := qtx.GetTask(ctx, id.String()); err != nil {
		return fmt.Errorf("get task %s: %w", id, err)
	}

	// Verify it is a branch.
	parentNS := sql.NullString{String: id.String(), Valid: true}
	count, err := qtx.CountChildren(ctx, parentNS)
	if err != nil {
		return fmt.Errorf("count children of %s: %w", id, err)
	}
	if count == 0 {
		return fmt.Errorf("task %s is a leaf, not a branch", id)
	}

	// List all children.
	children, err := qtx.ListAllTasksByParent(ctx, parentNS)
	if err != nil {
		return fmt.Errorf("list children of %s: %w", id, err)
	}

	// Verify each child is a clean pending leaf.
	for _, child := range children {
		childID := TaskID(child.ID)

		// Must be a leaf.
		cc, err := qtx.CountChildren(ctx, sql.NullString{String: child.ID, Valid: true})
		if err != nil {
			return fmt.Errorf("count children of %s: %w", child.ID, err)
		}
		if cc > 0 {
			return fmt.Errorf("child %s is a branch, not a leaf", child.ID)
		}

		// Must be pending.
		if child.Status.String != string(StatusPending) {
			return fmt.Errorf("child %s has status %q, want pending", child.ID, child.Status.String)
		}

		// Must have no context.
		if child.Context.Valid && child.Context.String != "" {
			return fmt.Errorf("child %s has context set", child.ID)
		}

		// Must have no non-system records.
		nonSys, err := qtx.CountNonSystemRecordsByTask(ctx, childID.String())
		if err != nil {
			return fmt.Errorf("count non-system records of %s: %w", child.ID, err)
		}
		if nonSys > 0 {
			return fmt.Errorf("child %s has %d non-system records", child.ID, nonSys)
		}
	}

	// Clean up children: delete records, deps, then tasks.
	for _, child := range children {
		if err := qtx.DeleteRecordsByTask(ctx, child.ID); err != nil {
			return fmt.Errorf("delete records of %s: %w", child.ID, err)
		}
		if err := qtx.DeleteDepsByTask(ctx, db.DeleteDepsByTaskParams{
			TaskID:  child.ID,
			AfterID: child.ID,
		}); err != nil {
			return fmt.Errorf("delete deps of %s: %w", child.ID, err)
		}
	}

	if err := qtx.DeleteTasksByParent(ctx, parentNS); err != nil {
		return fmt.Errorf("delete children of %s: %w", id, err)
	}

	// Restore parent status to pending.
	if err := qtx.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		Status: sql.NullString{String: string(StatusPending), Valid: true},
		ID:     id.String(),
	}); err != nil {
		return fmt.Errorf("restore status of %s: %w", id, err)
	}

	// Write system record on parent.
	msg := fmt.Sprintf("unsplit from %d children", len(children))
	if err := addSystemRecord(ctx, qtx, id, msg); err != nil {
		return err
	}

	return tx.Commit()
}
