# Learnings

Preferences and patterns observed from the codebase and planning discussions.

## Project structure

- Flat package layout: `main.go`, `common/`, `cli/`, `db/`, `epic/`
- `epic/` owns domain types and business logic; `db/` is the data access layer; `cli/` is presentation
- Layer dependency flows one way: `cli/` -> `epic/` -> `db/`
- `epic/` defines its own domain types, separate from sqlc-generated types in `db/`

## Go coding

- Context-based dependency injection with string keys and typed getter functions that panic on misuse (see `common.GetConfig`, `db.GetQ`)
- Config via environment variables only (`config-go` with `AE_` prefix), no config files
- Embedded SQL schema via `//go:embed`
- `//go:generate sqlc generate` for query codegen
- Minimal dependencies -- only the libraries listed in CLAUDE.md

## Software architecture

- One SQLite file per epic -- no shared database, no persistent connections
- CLI commands use colon-separated naming: `task:list`, `task:set-body`, `attr:set`
- JSON envelope `{"ok": bool, "data": ..., "error": ...}` for machine-readable task commands; simple text for human commands
- Two command tiers: colon-separated for machine-readable JSON (`task:list`, `task:set-body`), top-level for human plain text (`show`, `context`, `epics`)
- Errors in task commands go in the JSON envelope to stdout (with non-zero exit code), not stderr
- Sequential DB migration via `PRAGMA user_version`
- Naive literal `---` line splitting for task decomposition (no markdown-aware parsing)
- Dependencies restricted to siblings only (same parent)
- `add-child` requires the parent to already be a branch (split first)
- `task:list` shows both branches (derived status) and leaves
- Context composition: ancestors + terminal siblings + self

## Corrections

(None yet -- will be updated as corrections arise.)
