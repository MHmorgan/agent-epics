package epic

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MHmorgan/agent-epics/db"
)

// TransitionStatus changes a leaf task's status. Validates the transition.
// For block and abandon, reason must be non-empty.
// Writes appropriate system records:
//
//	"status pending -> active"
//	"status active -> blocked: <reason>"
//	"status blocked -> active"
//	"status active -> done"
//	"status active -> abandoned: <reason>"
//	etc.
//
// Uses a transaction for atomicity.
func TransitionStatus(ctx context.Context, conn *sql.DB, q *db.Queries, id TaskID, to Status, reason string) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	txq := q.WithTx(tx)

	// Fetch the task.
	task, err := txq.GetTask(ctx, string(id))
	if err != nil {
		return fmt.Errorf("get task %s: %w", id, err)
	}

	// Verify it's a leaf (no children).
	count, err := txq.CountChildren(ctx, sql.NullString{String: task.ID, Valid: true})
	if err != nil {
		return fmt.Errorf("count children of %s: %w", id, err)
	}
	if count != 0 {
		return fmt.Errorf("task %s is a branch, not a leaf", id)
	}

	// Get current status.
	from := Status(task.Status.String)

	// Validate transition.
	if !ValidTransition(from, to) {
		return fmt.Errorf("invalid transition: %s -> %s", from, to)
	}

	// Check reason requirement.
	if TransitionRequiresReason(to) && reason == "" {
		return fmt.Errorf("transition to %s requires a reason", to)
	}

	// Update status.
	err = txq.UpdateTaskStatus(ctx, db.UpdateTaskStatusParams{
		Status: sql.NullString{String: string(to), Valid: true},
		ID:     task.ID,
	})
	if err != nil {
		return fmt.Errorf("update status of %s: %w", id, err)
	}

	// Write system record.
	text := fmt.Sprintf("status %s \u2192 %s", from, to)
	if reason != "" {
		text = fmt.Sprintf("status %s \u2192 %s: %s", from, to, reason)
	}
	if err := addSystemRecord(ctx, txq, id, text); err != nil {
		return err
	}

	return tx.Commit()
}
