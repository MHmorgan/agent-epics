Agent Epics
===========

A CLI tool for planning and tracking agent-driven work. Each *epic* is a
self-contained SQLite file holding a hierarchy of tasks — a single portable
artifact you can copy, share, or archive.

The tool is designed to be paired with two Claude Code skills, `epic-plan`
and `epic-work`, which walk an agent through interviewing, planning,
splitting, executing, and summarizing an epic.

The binary is named `ae`. See [DESIGN.md](DESIGN.md) for the full design
spec — this README is the human-facing guide.


Installation
------------

Install with `go install`:

```
go install github.com/MHmorgan/agent-epics@latest
```

This drops a binary named `agent-epics` into `$GOBIN` (or `$GOPATH/bin`).
The rest of this document — and the skills — assume the binary is named
`ae`, so add an alias or either rename it:

```
mv "$(go env GOPATH)/bin/agent-epics" "$(go env GOPATH)/bin/ae"
```

Make sure `$GOBIN` (or `$GOPATH/bin`) is on your `PATH`.

Epics live in `~/.agent-epics/epics/` by default, one `.db` file per epic.
Override with the `AE_APPDIR` environment variable.


Skills
------

Agent Epics is a tool — the workflow it enables lives in two skills.

### `epic-plan`

Invoked as `/epic-plan <description>`. Takes an idea, creates an epic, and
interviews you to flesh out the work. Recursively splits large epics into
sub-tasks; finalizes each leaf with acceptance criteria, constraints,
implementation notes from three parallel reviews (software, architecture,
agent workflow), and a handoff context. Planning ends when every leaf has
a finalized body and is ready to be executed in its own session.

### `epic-work`

Invoked as `/epic-work <id>`. Walks a planned epic, branch, or leaf. Picks
the next ready leaf via `ae task:next`, dispatches a fresh subagent as a
*leaf orchestrator* to implement it, and applies the resulting status
transition (`done`, `blocked`, or `abandoned`). Blocks halt the run;
abandons don't. On completion, writes an epic summary or a branch-context
rollup and reports back.

Both skills are checked into this repo under `skills/`. Symlink or copy
them into your Claude Code skills directory to make them user-invocable.


Example workflows
-----------------

### Planning a new epic

```
you> /epic-plan Add OIDC authentication to the API
```

The agent proposes a slug (`oidc-auth`), confirms, and runs
`ae task:new-epic oidc-auth`. It then interviews you — one question at a
time — rewriting the task body after each answer. If the work is large,
it drafts a `---`-separated outline, confirms the shape with you, and
runs `ae task:split oidc-auth`. Each child is planned the same way,
depth-first.

You can exit and resume at any point — state is on disk. Check what's
been planned so far:

```
ae epics                       # see all epics
ae show oidc-auth              # view the root body
ae task:list oidc-auth         # list tasks still in play
ae show oidc-auth:2            # a specific child
ae context oidc-auth:2:1       # composed context for a leaf
```

### Executing a planned epic

```
you> /epic-work oidc-auth
```

The orchestrator picks the first ready leaf, dispatches a subagent to
implement it, marks it `done`, and moves on. If a leaf gets blocked, the
run stops and the orchestrator reports to you. When the whole epic is
terminal it writes a summary:

```
ae attr:get oidc-auth summary
```

### Running a single leaf

`/epic-work` also takes a leaf or branch id when you only want to execute
part of the tree:

```
you> /epic-work oidc-auth:2:1
```

### Inspecting an epic by hand

```
ae epics                       # list all epics with their derived status
ae show oidc-auth:2            # body as plain text
ae context oidc-auth:2:1       # composed context: epic → parent → siblings → self
ae task:records oidc-auth      # full subtree journal (agent + system records)
ae task:records oidc-auth:2 --self  # only records attached to this exact task
ae task:next oidc-auth         # what would `epic-work` pick up next?
```

### Cleaning up

```
ae rm oidc-auth                # remove a specific epic
ae purge                       # remove all terminal epics (all leaves done/abandoned)
```


Human command reference
-----------------------

These are the commands intended for interactive human use. They
pretty-print their output. The `task:*` family used by the skills returns
JSON and is documented in [DESIGN.md](DESIGN.md).

### `ae epics`

List all epics with their derived status (computed from the status of
their leaves).

### `ae show <id>`

Print a task's body as plain text. Works for any task — epic root,
branch, or leaf. Branch bodies are frozen snapshots from before the split.

### `ae context <id>`

Print the composed context for a task: epic context, then each ancestor's
context, then terminal siblings' contexts, then the task's own context —
each under its own `# <id> — context` header. Empty pieces are omitted.

### `ae rm <epic>`

Delete an epic. Takes the epic root id. The `.db` file is removed.

### `ae purge`

Remove every epic whose derived status is terminal (all leaves `done` or
`abandoned`). Use this to clean up after finished work.

### Getting help

```
ae help
ae <command> --help
```
