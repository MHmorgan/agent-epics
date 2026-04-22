# agent-epics v1 — Full Implementation

## Problem
The `agent-epics` project has a detailed DESIGN.md and a minimal skeleton (config,
stub models, empty schema), but no working implementation. Agents currently have no
tool for structured task planning, progress tracking, and handoff within epics.

## Desired behavior
A fully functional `ae` CLI that implements the complete DESIGN.md spec: hierarchical
task management with split/unsplit, status transitions, composed context, append-only
records, within-epic sibling dependencies with cycle detection, and the `next` command
for agent orchestration. Human-facing commands (`epics`, `rm`, `purge`) pretty-print;
all task commands output JSON with an `{"ok": bool, "data": ..., "error": ...}` envelope.

## Acceptance criteria
- [ ] `ae task:new-epic <id>` creates an epic (SQLite DB in `~/.agent-epics/epics/`) with a root task (status=pending, empty body)
- [ ] `ae task:set <id> <markdown>` replaces the body of a leaf task; rejects branches
- [ ] `ae task:get <id>` returns the task as JSON
- [ ] `ae task:split <id>` splits a pending/blocked leaf on literal `---` line separators into numbered children (`:1`, `:2`, ...), extracting `# Heading` as titles
- [ ] `ae task:unsplit <id>` reverses a split when all children are clean pending leaves
- [ ] `ae task:add-child <parent>` creates an empty pending leaf under an existing branch; ID suffix and position = max existing + 1; rejects leaf parents
- [ ] Status commands (`task:start`, `task:block`, `task:unblock`, `task:done`, `task:abandon`) enforce the transition graph; `block` and `abandon` require a reason argument
- [ ] `ae task:list` returns all tasks (branches with derived status, leaves with stored status) as JSON; `all` flag includes terminal; `parent=<id>` flag filters to immediate children
- [ ] `ae task:context:set <id> <markdown>` writes/overwrites context on any task
- [ ] `ae task:context:get <id>` composes ancestor + terminal-sibling + self contexts, ancestors-first, omitting empty entries
- [ ] `ae task:record <id> <text>` appends an agent record
- [ ] `ae task:records <id>` returns subtree records; `ae task:records <id> self` returns exact match — each record has `source`, `ts` (ISO 8601), `task`, `text`
- [ ] `ae task:after <id> <pred>` creates a dependency edge; both must be siblings (same parent); rejects cycles via DFS
- [ ] `ae task:unafter <id> <pred>` removes a dependency edge
- [ ] `ae task:next <epic>` returns the first pending leaf whose sibling dependencies are satisfied
- [ ] `ae attr:set <epic> <attr> <value>` and `ae attr:get <epic> <attr>` manage epic-level attributes
- [ ] `ae epics` lists all epics with derived status (pretty-printed, simple text)
- [ ] `ae rm <epic>` deletes an epic's DB file
- [ ] `ae purge` removes all terminal (done/abandoned) epics
- [ ] System records auto-written for all structural events per DESIGN.md
- [ ] All invariants enforced (status↔leaf, body frozen on branch, append-only records, split/unsplit preconditions, sibling-only deps, no cycles)
- [ ] Sequential migration system using `PRAGMA user_version`
- [ ] Unit tests for pure logic (ID parsing/validation, markdown splitting, status transitions)
- [ ] Integration tests against real in-memory SQLite (split/unsplit, dependency cycles, context composition, `next` query)
- [ ] Installable via `go install`

## Scope
- No cross-epic references or dependencies
- No multi-agent concurrency beyond SQLite WAL defaults
- No non-ASCII slugs
- No reopening from terminal states
- No bulk operations
- No build tooling beyond `go install`
- Splitting is naive literal `---` line split (no code-block awareness)

## Constraints
- CLI: `github.com/Minimal-Viable-Software/cli-go` — colon-separated flat subcommands (`task:list`, `task:context:get`), `name=value` flags, positional args (all required, no optionals)
- Config: `github.com/Minimal-Viable-Software/config-go` with `AE_` prefix
- Logging: `github.com/Minimal-Viable-Software/log-go`
- Database: `modernc.org/sqlite` (pure Go) + `sqlc` for query generation
- No additional dependencies
- App directory: `~/.agent-epics` (update DESIGN.md to match code)
- DB per epic — each epic is a separate SQLite file
- `epic/` defines its own domain types; maps to/from sqlc-generated `db/` types
- Dependencies are strictly sibling-only (same parent_id)
- `add-child` only works on branches — new epics must be split first to get children
- `task list` default excludes terminal tasks; `all` flag includes them; branches show derived status

## Context
- **DESIGN.md** (471 lines) is the authoritative spec
- **Existing skeleton**: `main.go` (app setup), `common/config.go` (env config), `epic/epic.go` and `epic/task.go` (stubs), `db/database.go` (SQLite open), `db/sqlc.yaml` (codegen config)
- **cli-go API**: `app.SubCommand("task:list", usage)`, `cmd.StringFlag(&v, "name", "usage")`, `cmd.StringArg(&v, "name", "usage")`, `cmd.Run(func() error)`
- **cli-go limitation**: no optional/variadic arguments — `all` and `self` modifiers must be flags
- `db/schema.sql` has a dummy table; `db/queries/` is empty; `cli/` is empty

## Implementation notes

### Software considerations
- Continue the context-based DI pattern from `common/config.go` and `db/database.go` (`WithConfig`, `GetConfig`, `WithQ`, `GetQ`)
- `is_leaf` and `all_leaves_terminal` can be expressed as SQL subqueries — no custom SQLite functions needed
- ID validation fits a single `regexp.MustCompile` pattern
- Context composition can derive parent IDs by string-splitting hierarchical IDs (no parent_id walk needed)
- Transactions for compound operations (split, unsplit) via `db.Begin()` + `queries.WithTx(tx)` + `tx.Commit()`
- Per-epic DB lifecycle: each command opens a single epic DB by name, does work, and closes it — no persistent connections
- JSON envelope output should use a shared helper to keep formatting consistent across all task commands
- Some complex queries (notably `next` with `all_leaves_terminal`) may exceed sqlc capabilities — fall back to hand-written SQL if needed
- `blocked → active` transition (unblock command) needs system record text `status blocked → active`

### Architecture considerations
- **Layer dependency**: `cli/` → `epic/` → `db/` (cli imports epic, epic imports db, no reverse)
- **Hierarchical ID type** in `epic/` is foundational — needed by every layer; implement early
- **sqlc types live in `db/`**; `epic/` defines domain types and maps between them
- **Migration runner** replaces current `Open()` — check `user_version`, apply migrations sequentially, set new version, all in a transaction
- **Command registration** in `cli/` will be a flat list of ~25 `app.SubCommand()` calls — organize into files by command group (human commands, task read commands, task write commands, structure commands, status commands)
- Risk: sqlc may not handle the `next` query's complexity — validate early and have a raw-SQL fallback

### Agent workflow recommendation
- **Phase 1** (sequential): Schema + sqlc queries + codegen — must complete before Go code references generated types
- **Phase 2** (single agent): Core types in `epic/` — ID parsing, status transitions, domain types. Unit-testable immediately
- **Phase 3** (2-3 parallel): Business logic in `epic/` — lifecycle ops, split/unsplit, deps/cycle detection, context composition, `next` query
- **Phase 4** (2-3 parallel): CLI commands in `cli/` — group by human/read/write commands to avoid file conflicts
- **Phase 5** (single agent): Wire into `main.go`, integration tests, final verification
- **High-risk areas**: `next` query correctness, cycle detection (test transitive cycles), context composition ordering, branch derived status computation
