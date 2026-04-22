Claude Instructions
===================

## Libraries

* `github.com/Minimal-Viable-Software/cli-go` is used for CLI
* `github.com/Minimal-Viable-Software/config-go` is used for app config
* `github.com/Minimal-Viable-Software/log-go` is used for logging
* `modernc.org/sqlite` for SQLite
* `sqlc` for queries
* No other dependencies should be necessary


## Structure

`main.go` - application setup

`cli/` - implementation of CLI functionality and terminal UI stuff

`db/` - database interaction abstraction layer

`epic/` - implementation of the core `Epic` and `Task` models


## Learnings

You shall maintain a document - @LEARNINGS.md - where you document:

* My preferred way of structuring the project
* My preferences for Go coding
* My preferences in software architecture
* Any other stuff I correct you on, which impacts design decisions and produced code

You shall continuously update this document when working on this project.
