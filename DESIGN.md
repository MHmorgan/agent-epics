# `agent-epic` — design spec

A CLI tool that guides agents through task planning, tracks progress, and records
final state. Designed to be paired with a SKILL document that formalizes the
workflow.

One epic = one SQLite database = one portable artifact.

Technical note: the CLI uses https://github.com/Minimal-Viable-Software/cli-go which
is an Obsidian-CLI like interface (not POSIX compatible).

---

## Philosophy

- **Markdown is the agent's native medium.** The plan for a task is a single
  markdown blob. Editing the plan is editing markdown. Splitting the plan is
  splitting markdown on `---`. No structured section CRUD.
- **Three text fields per task, orthogonal purposes:**
  *body* is the plan, *context* is the compressed handoff, *records* are the
  append-only journal.
- **The tool enforces invariants; the SKILL encodes habits.** Together they
  steer agents toward reliable task execution and token-efficient handoffs.
- **Leaves are implementation sessions.** One leaf = one chunk of work, typically
  one Claude Code session.
- **Only leaves have status.** Branches are structural; their state is
  transitive (derived from descendants), never commanded.
- **DB per epic.** Each epic is a portable file. Copy, share, archive.

---

## Storage

```
~/.agent-epic/
└── epics/
    ├── my-project.db
    ├── another-epic.db
    └── ...
```

One SQLite file per epic. Each DB uses the `user_version` PRAGMA and migrates on open.

---

## Data model

### `task`

One row per task.

| field       | type    | notes                                                   |
|-------------|---------|---------------------------------------------------------|
| id          | TEXT PK | hierarchical: `foo`, `foo:1`, `foo:1:2`                 |
| parent_id   | TEXT    | NULL for the epic root                                  |
| title       | TEXT    | derived from first `# Heading` in body at split time    |
| body        | TEXT    | markdown plan; frozen read-only once the task is split  |
| context     | TEXT    | agent-authored compressed handoff; editable only during planning |
| status      | TEXT    | see *Status*; NULL for branches                         |
| position    | INTEGER | for deterministic ordering among siblings               |
| created_at  | TIMESTAMP |                                                       |
| updated_at  | TIMESTAMP |                                                       |

**Derived:** a task is a *leaf* iff it has no children, else a *branch*. The
epic root is the task with `parent_id IS NULL`.

**Invariant:** `status IS NOT NULL ⇔ task is a leaf`. Enforced on split
(status dropped) and on child creation (branch's status cleared).

### `attribute`

Attributes that only exists at the epic boundary.

| field     | type    | notes               |
|-----------|---------|---------------------|
| attribute | TEXT PK | Epic attribute name |
| value     | TEXT    | Attribute value     |

Known attributes:
  - `summary` - the summary of the epic. Written by agent after completion. Markdown text.

### `record`

Flat, append-only. Filtered by hierarchical ID prefix — no separate
branch/leaf handling.

| field   | type      | notes                                             |
|---------|-----------|---------------------------------------------------|
| id      | INTEGER PK| autoincrement                                     |
| task    | TEXT      | full hierarchical task id                         |
| ts      | TIMESTAMP |                                                   |
| source  | TEXT      | `agent` \| `system`                               |
| text    | TEXT      | freeform (agent) or templated one-liner (system)  |

Strictly append-only — no UPDATE, no DELETE. Agent corrects by recording again.

### `dep`

Within-epic only.

| field    | type | notes            |
|----------|------|------------------|
| task_id  | TEXT | dependent        |
| after_id | TEXT | predecessor      |

`PRIMARY KEY (task_id, after_id)`. Cycle detection on INSERT. Both IDs must
exist in this epic.

---

## IDs

- Slug charset for the epic root and intermediate slugs: `[a-z][a-z0-9-]*`.
  ASCII only. No colons. No leading/trailing hyphens.
- Colons separate hierarchy levels: `epic:child:grandchild`.
- **Children created by split get auto-numeric suffixes**: `parent:1`, `parent:2`.
  Stable once assigned; not reused even if children are abandoned.
- **Titles come from the first `# Heading`** found at the top of each body
  chunk at split time. Pure UX — IDs stay numeric, listings render
  `my-epic:2  Set up auth`.
- Every command takes a full hierarchical ID. No "current epic" context.

---

## Status

Only leaves have status. Branches derive.

| state     | meaning                                  | terminal |
|-----------|------------------------------------------|----------|
| pending   | exists, not started                      | no       |
| active    | agent is working on it                   | no       |
| blocked   | agent has stopped on it with a reason    | no       |
| done      | agent has completed it                   | yes      |
| abandoned | agent has decided not to complete it     | yes      |

### Transitions (agent-driven only)

```
pending   → active | blocked | abandoned | (split)
active    → blocked | done | abandoned
blocked   → active | abandoned | (split)
done      → (terminal)
abandoned → (terminal)
```

`block` and `abandon` require a reason.

### Preconditions

- `done` on a leaf — none.
- `split` — task must be a leaf with `pending` or `blocked` status and at
  least one `---` separator in its body.
- Branches have no status. Branch status is derived - including the status of the epic itself.

### Branch state (derived view)

Useful for listings and `next` evaluation. A branch is:

- `done` if every leaf descendant is `done` or `abandoned` (and at least one
  is `done`);
- `abandoned` if every leaf descendant is `abandoned`;
- `active` if any leaf descendant is `active`;
- `blocked` if any leaf descendant is `blocked` (and none are `active`);
- `pending` otherwise.

Never stored. Computed on read.

---

## The three fields

### body — the plan

Markdown. On a leaf, fully editable via `ae task:set`. On a branch, frozen
read-only at split time; `ae task:set` on a branch fails. `ae task:get`
returns the body regardless.

When an agent starts working on a task it reads the body.

**Structure convention:** when the agent intends to split later, sections are
separated by `---` (markdown horizontal rule). Each section should start with
a `# Heading` whose text becomes the child's display title.

### context — the handoff

Agent-authored compressed representation for future sessions. The purpose is
to save tokens.

When reading the context for a leaf, all parent and sibling contexts are included.

Branch contexts are used to define common context useful for all agents
working on the child tasks.

Leaf contexts are used as handoff mechanism between agents. After a leaf task
is complete the agent updates the context of its own task with information
that affects other tasks (unresolved issues, blockers, etc.)

A context should be with <= 15 lines. Otherwise it might grow too big when 
appending parents and siblings.

- `ae task:context:set <id> <markdown>` — writes or overwrites.
- `ae task:context:get <id>` — **composes ancestors-first** with structured
  headers:

  ```
  # my-epic — context
  <context of epic my-epic>

  # my-epic:2 — context
  <context of parent my-epic:2>

  # my-epic:2:1 — context
  <context of sibling my-epic:2:1>

  # my-epic:2:2 — context
  <context of my-epic:2:2>
  ```

  Empty pieces are omitted. The composition includes the requested task itself
  at the bottom.

Editable at any task type, any status, any time.

### records — the journal

Flat append-only table. Queried by hierarchical ID prefix:

- `ae task:records my-epic` — subtree (default; all records where `task`
  begins with `my-epic:` or equals `my-epic`).
- `ae task:records my-epic self` — exact match only.
- `ae task:record <id> <text>` — append an agent entry.

System records are auto-written for structural events; see *System records*
below.

---

## Split and unsplit

### Split

`ae task:split <id>` — preconditions:
- Task is a leaf.
- Status is `pending` or `blocked`.
- Body contains at least one `---` separator.

Procedure:
1. Parse body: split on `---` into N chunks (sections).
2. For each chunk, in order (1-indexed):
   - Create child `<id>:<n>` with `body = <chunk>`, `status = pending`,
     `position = n`.
   - Extract `title` from the first `# Heading` in the chunk (if any).
3. Clear parent's `status` (becomes a branch).
4. Parent's body is now frozen — no further `set` allowed.
5. System record on parent: `split into N children`.

### Unsplit

`ae task:unsplit <id>` — preconditions:
- Task is a branch.
- Every child is a `pending` leaf with no context and no records other than
  the auto-record from creation.

Procedure:
1. Delete all children.
2. Restore parent's `status = pending`.
3. Parent's body becomes editable again (it was never deleted).
4. System record on parent: `unsplit from N children`.

Unsplit is a rare tool for undoing a misplanned split before any real work
has happened. Not a general restructuring mechanism.

---

## Dependencies

- Siblings run in parallel by default.
- `ae task:after <id> <pred>` creates an ordering edge.
- `ae task:unafter <id> <pred>` removes it.
- Both IDs within the same epic and siblings. Cycles rejected at insert.

### Target semantics

A dependency `X after Y` is satisfied iff every leaf descendant of `Y` has
status `done` or `abandoned`. Abandonment satisfies dependencies because
abandonment is a *decision to stop*, not a failure — forcing the agent to
route around it would just encourage fake completions.

### `ae task:next <epic>`

Returns the first `pending` leaf whose `after` dependencies are all satisfied,
ordered by `position` then `created_at`. Empty result means either all work
is terminal or all ready work is already `active`/`blocked`.

```
SELECT t FROM task
WHERE epic_root(t) = <epic>
  AND t.status = 'pending'
  AND is_leaf(t)
  AND NOT EXISTS (
    SELECT 1 FROM dep d
    WHERE d.task_id = t.id
      AND NOT all_leaves_terminal(d.after_id)
  )
ORDER BY t.position, t.created_at
LIMIT 1
```

---

## System records

System records are auto-written one-liners. Deterministic shape, grep-friendly.

```
created
body set
context set
split into N children
unsplit from N children
after <pred>
unafter <pred>
status pending → active
status active  → blocked: <reason>
status blocked → abandoned: <reason>
summary written
```

Written on the same `record` table with `source = 'system'`. They appear in
`ae task:records` output alongside agent records, visibly tagged.

---

## Command reference

All human interface commands pretty-prints its output.
All task commands uses json output for machine readability.

### Human interface

```
ae epics          # List all epics (root tasks)
ae rm <epic>      # Remove the epic
ae purge          # Remove all terminal epics (done or abandoned)
```

### Listing and reading

```
ae task:list                        # all tasks with a non-terminal status
ae task:list all                    # include done/abandoned tasks
ae task:list parent=<id>            # immediate children of a branch
ae task:get <id>                    # body (markdown)
ae task:context:get <id>            # composed context
ae task:records <id>                # subtree records (default)
ae task:records <id> self           # exact task only
ae task:next <epic>                 # first ready pending leaf
```

### Creation and planning

```
ae task:new-epic <epic>             # creates epic (top-level)
ae task:add-child <parent>          # create a new child for the given parent task (must be a branch)
ae task:set <id> <markdown>         # replaces body; leaf only
ae task:context:set <id> <markdown> # replaces context; any task
ae task:record  <id> <text>         # appends agent record
```

### Structure

```
ae task:split   <id>                # preconditions apply
ae task:unsplit <id>                # preconditions apply
ae task:after   <id> <pred>
ae task:unafter <id> <pred>
```

### Status (leaves only)

```
ae task:start    <id>               # → active
ae task:block    <id> <reason>      # → blocked
ae task:unblock  <id>               # → active
ae task:done     <id>               # → done
ae task:abandon  <id> <reason>      # → abandoned
```

### Epic attributes

```
ae attr:set <epic> <attr> <value>   # set an epic attribute
ae attr:get <epic> <attr>           # get the content of an epic attribute
```

---

## Invariants

Enforced by the tool on every write:

- A task's `status` is NULL iff the task has children.
- A task's body is immutable iff the task has children.
- Records are append-only; no UPDATE, no DELETE.
- `split` requires a leaf, status ∈ {pending, blocked}, and ≥ 1 `---` in
  the body.
- `unsplit` requires a branch with all children pending and clean (no
  context, no non-system records).
- Dependencies are within-epic only; no cycles.

---

## Example Planning Workflow

User starts with prompt: `/epic-planning Let's implement API authentication using OIDC for this project!`

The agent creates a new epic with `ae task:new-epic <id>`

### Small Task

The agent interviews the user, flushing out details about the project,
continuously updating the body of the task as more information is revealed.

### Large Task

The agent interviews the user, flushing out details about the project.
At some point the agent or user suggest the epic should be split into
sub-tasks.
They discuss if the task is ready to be split, based on the current body,
or if adjustments should be made.
When the user is happy, the agent splits the task.

Each sub-task is then interviewed and planned separately. If necessary,
a similar process can be performed to split sub-tasks into sub-tasks.

For each leaf task, a review process is performed before moving on
to planning the next leaf task.

When all leaf tasks of a branch has been planned, the branch task
is reviewed.

This continues until all tasks in the hierarchy has been planned.

During this entire process the context of branch tasks are updated
based on what information is deemed relevant for its sub tasks.


## Example Implementation Workflow

Users starts with prompt like one of this:
* `/epic-work <epic>`
* `/epic-work implement it!` (referring to the currently discussed epic)

The orchestrator agent does no work itself, but uses sub-agents.
The subagent starts working on a task using `ae task:next <epic>`

After all tasks is done, the orchestrator creates the epic
summary.

---

## Out of scope for v1

- Cross-epic references or dependencies.
- Multi-agent concurrency on one epic. SQLite WAL handles casual cases;
  explicit multi-writer workflows are not a goal.
- Non-ASCII slugs.
- Reopening from terminal states. If needed later: `ae task:reopen <id>`.
- Bulk operations like `ae task:sequence a b c d`. Sugar for later.
