package epic

import (
	"context"
	"fmt"

	"github.com/MHmorgan/agent-epics/db"
)

// AddDependency creates a dependency edge (id after pred). Both must be siblings
// (same parent). Rejects cycles via DFS traversal of the dependency graph.
// Writes system record "after <pred>" on the task.
func AddDependency(ctx context.Context, q *db.Queries, id TaskID, pred TaskID) error {
	// Verify both tasks exist.
	taskRow, err := q.GetTask(ctx, id.String())
	if err != nil {
		return fmt.Errorf("get task %s: %w", id, err)
	}
	predRow, err := q.GetTask(ctx, pred.String())
	if err != nil {
		return fmt.Errorf("get task %s: %w", pred, err)
	}

	// Verify they are siblings (same parent_id).
	if taskRow.ParentID != predRow.ParentID {
		return fmt.Errorf("tasks %s and %s are not siblings", id, pred)
	}

	// Check for cycles.
	cycle, err := hasCycle(ctx, q, id, pred)
	if err != nil {
		return fmt.Errorf("cycle check: %w", err)
	}
	if cycle {
		return fmt.Errorf("adding dependency %s after %s would create a cycle", id, pred)
	}

	// Insert the dep.
	if err := q.InsertDep(ctx, db.InsertDepParams{
		TaskID:  id.String(),
		AfterID: pred.String(),
	}); err != nil {
		return fmt.Errorf("insert dep: %w", err)
	}

	// Write system record.
	if err := addSystemRecord(ctx, q, id, "after "+pred.String()); err != nil {
		return fmt.Errorf("write system record: %w", err)
	}

	return nil
}

// RemoveDependency removes a dependency edge.
// Writes system record "unafter <pred>" on the task.
func RemoveDependency(ctx context.Context, q *db.Queries, id TaskID, pred TaskID) error {
	if err := q.DeleteDep(ctx, db.DeleteDepParams{
		TaskID:  id.String(),
		AfterID: pred.String(),
	}); err != nil {
		return fmt.Errorf("delete dep: %w", err)
	}

	if err := addSystemRecord(ctx, q, id, "unafter "+pred.String()); err != nil {
		return fmt.Errorf("write system record: %w", err)
	}

	return nil
}

// hasCycle checks if adding an edge from id (dependent) to pred (predecessor)
// would create a cycle. It does this by checking if id is reachable from pred
// by following the predecessor chain: from pred, follow its "after" edges
// (predecessors). If we reach id, adding (id after pred) would close a cycle.
func hasCycle(ctx context.Context, q *db.Queries, id TaskID, pred TaskID) (bool, error) {
	visited := make(map[string]bool)
	return dfs(ctx, q, pred, id, visited)
}

// dfs walks the predecessor chain starting from current, looking for target.
// Returns true if target is reachable.
func dfs(ctx context.Context, q *db.Queries, current TaskID, target TaskID, visited map[string]bool) (bool, error) {
	if current == target {
		return true, nil
	}
	if visited[current.String()] {
		return false, nil
	}
	visited[current.String()] = true

	deps, err := q.ListDepsForTask(ctx, current.String())
	if err != nil {
		return false, fmt.Errorf("list deps for %s: %w", current, err)
	}

	for _, dep := range deps {
		found, err := dfs(ctx, q, TaskID(dep.AfterID), target, visited)
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}

	return false, nil
}
