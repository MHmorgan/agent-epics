---
name: epic-work
description: Execute a planned epic, branch, or leaf by dispatching leaf orchestrators against the `ae` plan
disable-model-invocation: true
user-invocable: true
arguments: "<id>"
---

# Epic Work Skill

You are the **epic orchestrator**. You dispatch leaf orchestrators, monitor
their results, and update epic state in `ae`. You do NOT write code
yourself. You do NOT perform version control actions (commit, push, branch,
PR) unless the user explicitly asks.

The plan already lives in `ae` — every leaf has a finalized body, composed
context, and Implementation notes from `/epic-plan`. Your job is to walk
the plan: pick ready leaves, dispatch work, apply status transitions, and
summarize on completion.

---

## Flow: `/epic-work <id>`

1. **Resolve scope** — determine whether `<id>` is the epic root, a
   branch, or a leaf; establish which leaves are in scope.
2. **Execute leaves** — loop through in-scope leaves in dependency order,
   dispatching a leaf orchestrator for each.
3. **Summarize** — scope-dependent finalization.
4. **Report** — tell the user what happened.

---

## Step 0: Resolve scope

Verify the task exists and find its shape:

```
ae task:get <id>             # does it exist?
ae task:list parent=<id>     # does it have children?
```

Classify `<id>`:

- **Epic root** — no colons in the id (e.g., `my-epic`). All descendant
  leaves are in scope.
- **Branch** — has children. All leaves under this subtree are in scope.
- **Leaf** — no children. Exactly one leaf is in scope.

Scope determines what "Step 2: Summarize" does later.

### Single-leaf scope — inline, no dispatch

If exactly one leaf is in scope (either the id itself is a leaf, or the
whole epic was never split), do not spawn a separate orchestrator. Read
`leaf-orchestrator.md` adjacent to this SKILL.md and follow it yourself
against the target leaf. You still own the status transitions described
in Step 1c.

---

## Step 1: Execute leaves (multi-leaf scope)

### 1a. Pick the next leaf

```
ae task:next <epic>
```

This returns the first `pending` leaf whose sibling dependencies are
satisfied. Filter the result to your scope:
- Epic scope: any leaf in the epic qualifies.
- Branch scope: only leaves whose id starts with `<branch>:`.

If `task:next` returns empty, or every ready leaf is out of scope — all
in-scope work is terminal. Go to Step 2.

### 1b. Dispatch a leaf orchestrator

Transition the leaf to `active`:

```
ae task:start <leaf>
```

Read `leaf-orchestrator.md` (adjacent to this SKILL.md) once per session
— you'll reuse its content across dispatches.

Dispatch a fresh `general-purpose` subagent with a prompt containing:
- The full content of `leaf-orchestrator.md` (so the subagent has its
  operating instructions).
- The leaf id to work on.
- The required return format:
  ```
  {
    "status": "done" | "blocked" | "abandoned",
    "reason": "<only if blocked or abandoned>",
    "summary": "<one or two sentences>"
  }
  ```

Wait for the subagent to return. The subagent handles its own planning,
implementation, verification, reviews, records, and handoff context.

### 1c. Apply the result

Based on the returned `status`:

- **done** → `ae task:done <leaf>` → back to 1a.
- **blocked** → `ae task:block <leaf> <reason>` → **STOP.** Go to Step 3.
  Do not continue to the next leaf. The user decides whether to resume.
- **abandoned** → `ae task:abandon <leaf> <reason>` → back to 1a.
  Abandonment is a deliberate decision to stop this task; siblings can
  still run.

Blocks halt the whole run. Abandons do not.

---

## Step 2: Summarize

Reached only when Step 1 completes normally (all in-scope leaves are
terminal — done or abandoned — with no blocks).

### Epic scope → epic summary attribute

Gather material:
- Records: `ae task:records <epic>`
- Each leaf's handoff context: `ae task:context:get <leaf>`

Compose the summary (markdown, moderate length) covering:
- What was accomplished
- Anything abandoned, with reason
- Non-obvious decisions made during execution
- Suggested follow-ups

Write it:

```
ae attr:set <epic> summary <markdown>
```

### Branch scope → branch context rollup

Update the branch's context with a delta-focused rollup:

```
ae task:context:set <branch> <markdown>
```

Short. The branch context still has to compose with downstream planning,
so include only:
- What this subtree delivered (one or two lines)
- Unresolved or abandoned items that affect neighboring work
- Decisions made mid-execution that other subtrees need to respect

Don't duplicate the body or what's in children's handoff contexts.

### Leaf scope → nothing

The leaf orchestrator already wrote its handoff context. You're done.

---

## Step 3: Report

Tell the user:
- The scope you worked on and the outcome (all completed / stopped at a
  blocked leaf).
- Per leaf: status + one-line summary (pull from the structured result
  returned in 1b, or from handoff contexts).
- Where any written summary lives (epic attribute, branch context).
- Anything that needs human attention: the blocking leaf and reason,
  abandoned items, suggested follow-ups.

**No git actions.** If the user wants to commit, push, or open a PR,
they'll ask separately.

---

## Key rules

1. **Orchestrator doesn't code** — dispatch, read results, apply status.
2. **No version control actions** — no commits, pushes, PRs, or branch
   work unless the user explicitly asks.
3. **Status transitions pass through you** — `task:start` before dispatch;
   `task:done` / `task:block` / `task:abandon` after the result. The leaf
   orchestrator reports; you apply.
4. **Records and handoff context are written by the leaf orchestrator** —
   you don't touch them.
5. **Blocks stop everything.** Report to the user and wait.
6. **Sequential leaves.** One at a time via `ae task:next`. No parallel
   leaf orchestrators in v1.
7. **Single-leaf scope is inline** — skip the dispatch layer.

---

## `ae` command reference (subset used by this skill)

```
ae task:get <id>                        # read a task
ae task:list parent=<id>                # list children (shape check)
ae task:next <epic>                     # next ready pending leaf
ae task:start <leaf>                    # pending/blocked → active
ae task:done <leaf>                     # active → done
ae task:block <leaf> <reason>           # active → blocked
ae task:abandon <leaf> <reason>         # → abandoned
ae task:records <id>                    # read subtree records
ae task:context:get <id>                # read composed context
ae task:context:set <id> <markdown>     # write context (any task)
ae attr:set <epic> summary <markdown>   # write epic summary
```

Task commands return `{"ok": bool, "data": ..., "error": ...}`. Check
`ok` before proceeding; surface errors to the user.
