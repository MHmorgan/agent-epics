package epic

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MHmorgan/agent-epics/db"
)

// GetTask retrieves a single task by ID. Returns error if not found.
// Computes isLeaf by checking CountChildren.
func GetTask(ctx context.Context, q *db.Queries, id TaskID) (Task, error) {
	row, err := q.GetTask(ctx, string(id))
	if err != nil {
		return Task{}, fmt.Errorf("get task %s: %w", id, err)
	}

	count, err := q.CountChildren(ctx, sql.NullString{String: string(id), Valid: true})
	if err != nil {
		return Task{}, fmt.Errorf("count children of %s: %w", id, err)
	}

	return TaskFromDB(row, count == 0), nil
}

// ListTasks returns tasks for an epic. If parentID is non-empty, filters to
// immediate children. If includeTerminal is false, excludes terminal leaf tasks.
// Branches always get derived status computed.
func ListTasks(ctx context.Context, q *db.Queries, parentID TaskID, includeTerminal bool) ([]Task, error) {
	var rows []db.Task
	var err error

	if parentID != "" {
		nullParent := sql.NullString{String: string(parentID), Valid: true}
		if includeTerminal {
			rows, err = q.ListAllTasksByParent(ctx, nullParent)
		} else {
			rows, err = q.ListTasksByParent(ctx, nullParent)
		}
	} else {
		if includeTerminal {
			rows, err = q.ListAllTasks(ctx)
		} else {
			rows, err = q.ListTasks(ctx)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	tasks := make([]Task, 0, len(rows))
	for _, row := range rows {
		count, err := q.CountChildren(ctx, sql.NullString{String: row.ID, Valid: true})
		if err != nil {
			return nil, fmt.Errorf("count children of %s: %w", row.ID, err)
		}

		isLeaf := count == 0
		t := TaskFromDB(row, isLeaf)

		if !isLeaf {
			derived, err := GetDerivedStatus(ctx, q, TaskID(row.ID))
			if err != nil {
				return nil, fmt.Errorf("derived status of %s: %w", row.ID, err)
			}
			t.Status = derived
		}

		tasks = append(tasks, t)
	}

	return tasks, nil
}

// GetDerivedStatus computes the derived status for a branch task by querying
// all its leaf descendants recursively.
func GetDerivedStatus(ctx context.Context, q *db.Queries, branchID TaskID) (Status, error) {
	leafStatuses, err := collectLeafStatuses(ctx, q, branchID)
	if err != nil {
		return "", fmt.Errorf("collect leaf statuses of %s: %w", branchID, err)
	}
	return DerivedStatus(leafStatuses), nil
}

// collectLeafStatuses recursively finds all leaf descendants of the given task
// and returns their statuses.
func collectLeafStatuses(ctx context.Context, q *db.Queries, parentID TaskID) ([]Status, error) {
	nullParent := sql.NullString{String: string(parentID), Valid: true}
	children, err := q.ListAllTasksByParent(ctx, nullParent)
	if err != nil {
		return nil, err
	}

	var statuses []Status
	for _, child := range children {
		count, err := q.CountChildren(ctx, sql.NullString{String: child.ID, Valid: true})
		if err != nil {
			return nil, err
		}

		if count == 0 {
			// Leaf node: collect its status.
			statuses = append(statuses, Status(child.Status.String))
		} else {
			// Branch node: recurse.
			sub, err := collectLeafStatuses(ctx, q, TaskID(child.ID))
			if err != nil {
				return nil, err
			}
			statuses = append(statuses, sub...)
		}
	}

	return statuses, nil
}
