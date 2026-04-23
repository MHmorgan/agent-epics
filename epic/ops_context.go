package epic

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/MHmorgan/agent-epics/db"
)

// ComposeContext returns the composed context string for a task.
// Includes ancestor contexts (root to parent), terminal sibling contexts,
// and self context. Omits entries with empty context. Each entry is formatted as:
//
//	# <task-id> — context
//	<context text>
//
// Sections are separated by blank lines.
func ComposeContext(ctx context.Context, q *db.Queries, id TaskID) (string, error) {
	var sections []string

	// 1. Ancestors — from root to immediate parent.
	for _, anc := range id.Ancestors() {
		text, err := fetchContext(ctx, q, anc)
		if err != nil {
			return "", fmt.Errorf("ancestor context %s: %w", anc, err)
		}
		if text != "" {
			sections = append(sections, formatSection(anc, text))
		}
	}

	// 2. Terminal siblings — siblings whose effective status is terminal.
	if parent := id.Parent(); parent != "" {
		nullParent := sql.NullString{String: parent.String(), Valid: true}
		siblings, err := q.ListAllTasksByParent(ctx, nullParent)
		if err != nil {
			return "", fmt.Errorf("list siblings of %s: %w", id, err)
		}
		for _, sib := range siblings {
			if sib.ID == id.String() {
				continue
			}
			terminal, err := isTaskTerminal(ctx, q, sib)
			if err != nil {
				return "", fmt.Errorf("check terminal %s: %w", sib.ID, err)
			}
			if !terminal {
				continue
			}
			if sib.Context.Valid && sib.Context.String != "" {
				sections = append(sections, formatSection(TaskID(sib.ID), sib.Context.String))
			}
		}
	}

	// 3. Self context.
	selfText, err := fetchContext(ctx, q, id)
	if err != nil {
		return "", fmt.Errorf("self context %s: %w", id, err)
	}
	if selfText != "" {
		sections = append(sections, formatSection(id, selfText))
	}

	return strings.Join(sections, "\n\n"), nil
}

// fetchContext retrieves the context string for a single task. Returns "" if empty/null.
func fetchContext(ctx context.Context, q *db.Queries, id TaskID) (string, error) {
	ns, err := q.GetTaskContext(ctx, id.String())
	if err != nil {
		return "", err
	}
	if !ns.Valid {
		return "", nil
	}
	return ns.String, nil
}

// isTaskTerminal determines whether a task is in a terminal state.
// Leaf tasks use their status directly; branch tasks use derived status.
func isTaskTerminal(ctx context.Context, q *db.Queries, t db.Task) (bool, error) {
	count, err := q.CountChildren(ctx, sql.NullString{String: t.ID, Valid: true})
	if err != nil {
		return false, err
	}
	if count == 0 {
		// Leaf: check status directly.
		return IsTerminal(Status(t.Status.String)), nil
	}
	// Branch: compute derived status.
	derived, err := GetDerivedStatus(ctx, q, TaskID(t.ID))
	if err != nil {
		return false, err
	}
	return IsTerminal(derived), nil
}

// formatSection formats a single context entry with the standard heading.
func formatSection(id TaskID, text string) string {
	return fmt.Sprintf("# %s — context\n%s", id, text)
}
