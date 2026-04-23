package epic

import (
	"context"
	"database/sql"
	"testing"

	"github.com/MHmorgan/agent-epics/db"
)

// insertTaskWithContext inserts a task with an optional context string.
func insertTaskWithContext(t *testing.T, ctx context.Context, q *db.Queries, id, parentID, status, taskCtx string, pos int64) {
	t.Helper()
	err := q.InsertTask(ctx, db.InsertTaskParams{
		ID:       id,
		ParentID: sql.NullString{String: parentID, Valid: parentID != ""},
		Title:    sql.NullString{String: id, Valid: true},
		Status:   sql.NullString{String: status, Valid: status != ""},
		Context:  sql.NullString{String: taskCtx, Valid: taskCtx != ""},
		Position: sql.NullInt64{Int64: pos, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert task %s: %v", id, err)
	}
}

func TestComposeContext_SimpleRootAndLeaf(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTaskWithContext(t, ctx, q, "proj", "", "", "project-wide setup notes", 0)
	insertTaskWithContext(t, ctx, q, "proj:1", "proj", "pending", "my task context", 1)

	got, err := ComposeContext(ctx, q, "proj:1")
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	want := "# proj — context\nproject-wide setup notes\n\n# proj:1 — context\nmy task context"
	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestComposeContext_AncestorChain(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTaskWithContext(t, ctx, q, "proj", "", "", "root ctx", 0)
	insertTaskWithContext(t, ctx, q, "proj:1", "proj", "", "mid ctx", 1)
	insertTaskWithContext(t, ctx, q, "proj:1:1", "proj:1", "active", "leaf ctx", 1)

	got, err := ComposeContext(ctx, q, "proj:1:1")
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	want := "# proj — context\nroot ctx\n\n# proj:1 — context\nmid ctx\n\n# proj:1:1 — context\nleaf ctx"
	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestComposeContext_TerminalSiblingIncluded(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTaskWithContext(t, ctx, q, "proj", "", "", "", 0)
	insertTaskWithContext(t, ctx, q, "proj:1", "proj", "done", "done sibling ctx", 1)
	insertTaskWithContext(t, ctx, q, "proj:2", "proj", "active", "active sibling ctx", 2)
	insertTaskWithContext(t, ctx, q, "proj:3", "proj", "pending", "self ctx", 3)

	got, err := ComposeContext(ctx, q, "proj:3")
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	// proj has no context (empty) — omitted.
	// proj:1 is terminal (done) — included.
	// proj:2 is non-terminal (active) — excluded.
	// proj:3 is self — included.
	want := "# proj:1 — context\ndone sibling ctx\n\n# proj:3 — context\nself ctx"
	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestComposeContext_NonTerminalSiblingExcluded(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTaskWithContext(t, ctx, q, "proj", "", "", "root ctx", 0)
	insertTaskWithContext(t, ctx, q, "proj:1", "proj", "active", "should not appear", 1)
	insertTaskWithContext(t, ctx, q, "proj:2", "proj", "blocked", "also should not appear", 2)
	insertTaskWithContext(t, ctx, q, "proj:3", "proj", "pending", "self ctx", 3)

	got, err := ComposeContext(ctx, q, "proj:3")
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	// No terminal siblings, so only root + self.
	want := "# proj — context\nroot ctx\n\n# proj:3 — context\nself ctx"
	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestComposeContext_EmptyContextsOmitted(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTaskWithContext(t, ctx, q, "proj", "", "", "", 0)           // no context
	insertTaskWithContext(t, ctx, q, "proj:1", "proj", "done", "", 1) // terminal but no context
	insertTaskWithContext(t, ctx, q, "proj:2", "proj", "pending", "self ctx", 2)

	got, err := ComposeContext(ctx, q, "proj:2")
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	// Only self context since everything else is empty.
	want := "# proj:2 — context\nself ctx"
	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestComposeContext_SelfContextAtEnd(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTaskWithContext(t, ctx, q, "proj", "", "", "root ctx", 0)
	insertTaskWithContext(t, ctx, q, "proj:1", "proj", "abandoned", "abandoned ctx", 1)
	insertTaskWithContext(t, ctx, q, "proj:2", "proj", "done", "done ctx", 2)
	insertTaskWithContext(t, ctx, q, "proj:3", "proj", "active", "self ctx", 3)

	got, err := ComposeContext(ctx, q, "proj:3")
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	want := "# proj — context\nroot ctx\n\n# proj:1 — context\nabandoned ctx\n\n# proj:2 — context\ndone ctx\n\n# proj:3 — context\nself ctx"
	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestComposeContext_TerminalBranchSibling(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	// proj:1 is a branch with all-done leaves — derived status is "done" (terminal).
	insertTaskWithContext(t, ctx, q, "proj", "", "", "", 0)
	insertTaskWithContext(t, ctx, q, "proj:1", "proj", "", "branch sibling ctx", 1)
	insertTaskWithContext(t, ctx, q, "proj:1:1", "proj:1", "done", "", 1)
	insertTaskWithContext(t, ctx, q, "proj:1:2", "proj:1", "done", "", 2)
	insertTaskWithContext(t, ctx, q, "proj:2", "proj", "pending", "self ctx", 2)

	got, err := ComposeContext(ctx, q, "proj:2")
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	// proj:1 is a terminal branch sibling — included.
	want := "# proj:1 — context\nbranch sibling ctx\n\n# proj:2 — context\nself ctx"
	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestComposeContext_RootTask(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTaskWithContext(t, ctx, q, "proj", "", "pending", "root ctx", 0)

	got, err := ComposeContext(ctx, q, "proj")
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	// Root has no ancestors, no parent (so no siblings), only self.
	want := "# proj — context\nroot ctx"
	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestComposeContext_AllEmpty(t *testing.T) {
	q := setupTestDB(t)
	ctx := context.Background()

	insertTaskWithContext(t, ctx, q, "proj", "", "", "", 0)
	insertTaskWithContext(t, ctx, q, "proj:1", "proj", "pending", "", 1)

	got, err := ComposeContext(ctx, q, "proj:1")
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	if got != "" {
		t.Errorf("expected empty string, got:\n%s", got)
	}
}
