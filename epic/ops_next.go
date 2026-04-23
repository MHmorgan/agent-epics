package epic

import (
	"context"
	"database/sql"

	"github.com/MHmorgan/agent-epics/db"
)

// NextTask returns the first pending leaf whose dependencies are all satisfied.
// A dependency is satisfied if all leaf descendants of the predecessor are terminal.
// Returns nil (not error) if no task is ready.
// Ordered by position then created_at.
func NextTask(ctx context.Context, conn *sql.DB, q *db.Queries, epicID string) (*Task, error) {
	allTasks, err := q.ListAllTasks(ctx)
	if err != nil {
		return nil, err
	}

	// Build lookup structures.
	taskByID := make(map[string]db.Task, len(allTasks))
	childrenOf := make(map[string][]db.Task)
	for _, t := range allTasks {
		taskByID[t.ID] = t
		if t.ParentID.Valid {
			childrenOf[t.ParentID.String] = append(childrenOf[t.ParentID.String], t)
		}
	}

	// Collect pending leaves in order (ListAllTasks is ordered by position, created_at).
	var pendingLeaves []db.Task
	for _, t := range allTasks {
		if len(childrenOf[t.ID]) > 0 {
			continue // branch
		}
		if Status(t.Status.String) != StatusPending {
			continue
		}
		pendingLeaves = append(pendingLeaves, t)
	}

	// For each pending leaf, check if all dependencies are satisfied.
	for _, leaf := range pendingLeaves {
		deps, err := q.ListDepsForTask(ctx, leaf.ID)
		if err != nil {
			return nil, err
		}

		satisfied := true
		for _, dep := range deps {
			if !allLeavesTerminal(dep.AfterID, childrenOf, taskByID) {
				satisfied = false
				break
			}
		}

		if satisfied {
			task := TaskFromDB(leaf, true)
			return &task, nil
		}
	}

	return nil, nil
}

// allLeavesTerminal returns true if every leaf descendant of the task with the
// given ID has a terminal status (done or abandoned). If the task itself is a
// leaf, its own status is checked.
func allLeavesTerminal(id string, childrenOf map[string][]db.Task, taskByID map[string]db.Task) bool {
	children := childrenOf[id]
	if len(children) == 0 {
		// Leaf node: check its own status.
		t, ok := taskByID[id]
		if !ok {
			return false
		}
		return IsTerminal(Status(t.Status.String))
	}

	for _, child := range children {
		if !allLeavesTerminal(child.ID, childrenOf, taskByID) {
			return false
		}
	}
	return true
}
