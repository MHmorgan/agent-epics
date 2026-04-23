package epic

import (
	"context"
	"strings"
	"testing"
)

// TestContextComposition_AncestorChain verifies that ComposeContext returns
// ancestor contexts (root → parent → self) in order with proper headers,
// exercising a 3-level deep hierarchy built via real operations.
func TestContextComposition_AncestorChain(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "ancestor-chain"

	// Create and open the epic.
	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set context on the root task.
	if err := SetTaskContext(ctx, conn, q, TaskID(epicID), "Epic context"); err != nil {
		t.Fatalf("SetTaskContext root: %v", err)
	}

	// Set body with 2 sections, then split root into 2 children.
	body := "Child one\n---\nChild two"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody root: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID)); err != nil {
		t.Fatalf("SplitTask root: %v", err)
	}

	// Set context on child 1.
	child1 := TaskID(epicID + ":1")
	if err := SetTaskContext(ctx, conn, q, child1, "Child 1 context"); err != nil {
		t.Fatalf("SetTaskContext child1: %v", err)
	}

	// Split child 1 into 2 grandchildren.
	child1Body := "Grandchild one\n---\nGrandchild two"
	if err := SetTaskBody(ctx, conn, q, child1, child1Body); err != nil {
		t.Fatalf("SetTaskBody child1: %v", err)
	}
	if err := SplitTask(ctx, conn, q, child1); err != nil {
		t.Fatalf("SplitTask child1: %v", err)
	}

	// Set context on grandchild 1.
	grandchild1 := TaskID(epicID + ":1:1")
	if err := SetTaskContext(ctx, conn, q, grandchild1, "Grandchild context"); err != nil {
		t.Fatalf("SetTaskContext grandchild1: %v", err)
	}

	// ComposeContext for grandchild 1.
	got, err := ComposeContext(ctx, q, grandchild1)
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	// Verify ancestor chain: root, child 1, grandchild 1 — all in order.
	if !strings.Contains(got, "# "+epicID+" — context") {
		t.Error("missing root context header")
	}
	if !strings.Contains(got, "Epic context") {
		t.Error("missing root context body")
	}
	if !strings.Contains(got, "# "+string(child1)+" — context") {
		t.Error("missing child 1 context header")
	}
	if !strings.Contains(got, "Child 1 context") {
		t.Error("missing child 1 context body")
	}
	if !strings.Contains(got, "# "+string(grandchild1)+" — context") {
		t.Error("missing grandchild 1 context header")
	}
	if !strings.Contains(got, "Grandchild context") {
		t.Error("missing grandchild 1 context body")
	}

	// Verify ordering: root before child 1 before grandchild 1.
	rootIdx := strings.Index(got, "# "+epicID+" — context")
	child1Idx := strings.Index(got, "# "+string(child1)+" — context")
	gc1Idx := strings.Index(got, "# "+string(grandchild1)+" — context")
	if rootIdx >= child1Idx || child1Idx >= gc1Idx {
		t.Errorf("sections out of order: root@%d, child1@%d, grandchild1@%d", rootIdx, child1Idx, gc1Idx)
	}
}

// TestContextComposition_TerminalSiblingIncluded verifies that terminal
// sibling contexts are included while non-terminal sibling contexts are excluded.
func TestContextComposition_TerminalSiblingIncluded(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "sibling-ctx"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Split root into 3 children.
	body := "Section one\n---\nSection two\n---\nSection three"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID)); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	child1 := TaskID(epicID + ":1")
	child2 := TaskID(epicID + ":2")
	child3 := TaskID(epicID + ":3")

	// Set context on child 1, start it, and complete it (terminal).
	if err := SetTaskContext(ctx, conn, q, child1, "Sibling 1 done notes"); err != nil {
		t.Fatalf("SetTaskContext child1: %v", err)
	}
	if err := TransitionStatus(ctx, conn, q, child1, StatusActive, ""); err != nil {
		t.Fatalf("start child1: %v", err)
	}
	if err := TransitionStatus(ctx, conn, q, child1, StatusDone, ""); err != nil {
		t.Fatalf("complete child1: %v", err)
	}

	// Set context on child 2, start it (active, non-terminal).
	if err := SetTaskContext(ctx, conn, q, child2, "Sibling 2 notes"); err != nil {
		t.Fatalf("SetTaskContext child2: %v", err)
	}
	if err := TransitionStatus(ctx, conn, q, child2, StatusActive, ""); err != nil {
		t.Fatalf("start child2: %v", err)
	}

	// ComposeContext for child 3 (no context set on root or child 3).
	got, err := ComposeContext(ctx, q, child3)
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	// Child 1 is terminal — its context should be included.
	if !strings.Contains(got, "# "+string(child1)+" — context") {
		t.Error("missing terminal sibling (child1) context header")
	}
	if !strings.Contains(got, "Sibling 1 done notes") {
		t.Error("missing terminal sibling (child1) context body")
	}

	// Child 2 is non-terminal (active) — its context should NOT be included.
	if strings.Contains(got, "Sibling 2 notes") {
		t.Error("non-terminal sibling (child2) context should not appear")
	}

	// Root has no context set — should be omitted.
	if strings.Contains(got, "# "+epicID+" — context") {
		t.Error("root context header should not appear (root has no context)")
	}

	// Child 3 has no context set — should be omitted (no self section).
	if strings.Contains(got, "# "+string(child3)+" — context") {
		t.Error("child3 context header should not appear (no context set)")
	}
}

// TestContextComposition_EmptyContextsOmitted verifies that tasks with no
// context are omitted from the composed output.
func TestContextComposition_EmptyContextsOmitted(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "empty-ctx"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Do NOT set context on root.

	// Split root into 2 children.
	body := "First task\n---\nSecond task"
	if err := SetTaskBody(ctx, conn, q, TaskID(epicID), body); err != nil {
		t.Fatalf("SetTaskBody: %v", err)
	}
	if err := SplitTask(ctx, conn, q, TaskID(epicID)); err != nil {
		t.Fatalf("SplitTask: %v", err)
	}

	// Set context on child 2 only.
	child2 := TaskID(epicID + ":2")
	if err := SetTaskContext(ctx, conn, q, child2, "Only child 2 has context"); err != nil {
		t.Fatalf("SetTaskContext child2: %v", err)
	}

	got, err := ComposeContext(ctx, q, child2)
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	// Only child 2's context should be present.
	if !strings.Contains(got, "# "+string(child2)+" — context") {
		t.Error("missing child2 context header")
	}
	if !strings.Contains(got, "Only child 2 has context") {
		t.Error("missing child2 context body")
	}

	// Root has no context — should be omitted.
	if strings.Contains(got, "# "+epicID+" — context") {
		t.Error("root context header should not appear")
	}

	// Child 1 has no context — should not appear at all.
	child1 := TaskID(epicID + ":1")
	if strings.Contains(got, string(child1)) {
		t.Error("child1 should not appear in output")
	}
}

// TestContextComposition_RootOnly verifies that ComposeContext for the root
// task returns only the root's own context.
func TestContextComposition_RootOnly(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	const epicID = "root-only"

	if err := NewEpic(epicID, dir); err != nil {
		t.Fatalf("NewEpic: %v", err)
	}
	conn, q, err := OpenEpic(epicID, dir)
	if err != nil {
		t.Fatalf("OpenEpic: %v", err)
	}
	defer conn.Close()

	// Set context on root.
	if err := SetTaskContext(ctx, conn, q, TaskID(epicID), "Root only context"); err != nil {
		t.Fatalf("SetTaskContext root: %v", err)
	}

	got, err := ComposeContext(ctx, q, TaskID(epicID))
	if err != nil {
		t.Fatalf("ComposeContext: %v", err)
	}

	// Should contain only the root's context.
	if !strings.Contains(got, "# "+epicID+" — context") {
		t.Error("missing root context header")
	}
	if !strings.Contains(got, "Root only context") {
		t.Error("missing root context body")
	}

	// Should contain exactly one section (no double newlines indicating multiple sections).
	if strings.Contains(got, "\n\n") {
		t.Error("expected single section, but found section separator")
	}
}
