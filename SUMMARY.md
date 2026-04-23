# Summary

## What was done
Full v1 implementation of the `ae` CLI tool for hierarchical task epic management. 33 tasks across 10 waves implemented the complete DESIGN.md spec: SQLite schema with migration runner, domain types with ID validation, all business logic (task lifecycle, split/unsplit, status transitions, dependency management with cycle detection, context composition, next-task query), 25 CLI commands organized into 6 groups, and comprehensive test coverage (unit + integration).

## Acceptance criteria status
- [x] `ae task:new-epic <id>` creates epic with root task (status=pending)
- [x] `ae task:set <id> <markdown>` replaces leaf body; rejects branches
- [x] `ae task:get <id>` returns task as JSON
- [x] `ae task:split <id>` splits on `---` into numbered children with titles
- [x] `ae task:unsplit <id>` reverses clean splits
- [x] `ae task:add-child <parent>` creates empty leaf; rejects leaf parents
- [x] Status commands enforce transition graph; block/abandon require reason
- [x] `ae task:list <epic>` with `all` and `parent` flags; branches show derived status
- [x] `ae task:context:set <id> <markdown>` writes context on any task
- [x] `ae task:context:get <id>` composes ancestor + terminal sibling + self contexts
- [x] `ae task:record <id> <text>` appends agent record
- [x] `ae task:records <id>` returns subtree; `self` flag for exact match
- [x] `ae task:after <id> <pred>` creates sibling dep; rejects cycles via DFS
- [x] `ae task:unafter <id> <pred>` removes dep edge
- [x] `ae task:next <epic>` returns first ready pending leaf
- [x] `ae attr:set` and `ae attr:get` manage epic attributes
- [x] `ae epics` lists epics with derived status (plain text)
- [x] `ae rm <epic>` deletes epic DB
- [x] `ae purge` removes terminal epics
- [x] System records for all structural events
- [x] All invariants enforced
- [x] Sequential migration via PRAGMA user_version
- [x] Unit tests for ID parsing, markdown splitting, status transitions
- [x] Integration tests for lifecycle, deps, context composition, split/unsplit
- [x] Installable via `go install`

## Decisions & trade-offs
- `self` and `all` implemented as boolean flags (not positional args) due to cli-go's no-optional-args limitation
- `task:list` requires an `<epic>` positional arg since each epic is a separate DB
- `NextTask` uses in-memory graph traversal rather than complex SQL (avoids sqlc limitations)
- Path traversal defense on all epic ID → filepath operations (regex + containment check)
- Domain types in `epic/` separate from sqlc-generated `db/` types, with explicit mapping functions

## Deviations from plan
- Context-based DI pattern (`GetConfig`, `GetQ`) not used — explicit parameter passing throughout instead (simpler)
- `SetTaskContext` co-located in `ops_write.go` rather than a separate context write file
- `AddChild` in `ops_write.go` rather than `ops_structure.go`

## Architectural notes
- Clean three-layer architecture: `cli/` → `epic/` → `db/` with no reverse dependencies
- Consistent CLI registration pattern across all 6 command group files
- JSON envelope (`JSONSuccess`/`JSONError`) consistently applied to all task commands
- All compound writes wrapped in transactions (split, unsplit, status transitions, body/context set)

## Suggested follow-ups
- Extract CLI boilerplate into a shared `withEpic` helper to reduce repetition across ~15 handlers
- Add `isLeaf(ctx, q, id)` helper to consolidate repeated CountChildren + null-string checks
- Move `json.go` from `epic/` to `cli/` (presentation concern, not domain logic)
- Remove unused `common.GetConfig()` and `db.GetQ()` context accessors
- Add `nullStr()` helper for `sql.NullString` construction
- Wrap `AddDependency`/`RemoveDependency` in transactions for consistency
- Remove unused `conn *sql.DB` parameter from `NextTask`
