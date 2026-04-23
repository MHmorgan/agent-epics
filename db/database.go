package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"reflect"

	_ "modernc.org/sqlite"
)

//go:generate sqlc generate

//go:embed schema.sql
var schema string

// migration is a function that applies a schema change within a transaction.
type migration func(tx *sql.Tx) error

// migrations is the ordered list of migrations. Index 0 is migration 1, etc.
var migrations = []migration{
	func(tx *sql.Tx) error {
		_, err := tx.Exec(schema)
		return err
	},
}

// Open opens the SQLite database at the given path, enables foreign keys,
// and applies any pending migrations sequentially using PRAGMA user_version.
func Open(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := migrate(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return conn, nil
}

// migrate reads the current user_version and applies all pending migrations.
func migrate(conn *sql.DB) error {
	var current int
	if err := conn.QueryRow("PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	latest := len(migrations)
	if current >= latest {
		return nil
	}

	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for i := current; i < latest; i++ {
		if err := migrations[i](tx); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	// PRAGMA cannot run inside a transaction, so set it after commit.
	if _, err := conn.Exec(fmt.Sprintf("PRAGMA user_version = %d", latest)); err != nil {
		return fmt.Errorf("set user_version: %w", err)
	}

	return nil
}

// Q returns a Queries instance wrapping the given db connection.
func Q(conn *sql.DB) *Queries {
	return New(conn)
}

// GetQ returns the [Queries] object from the "db" context value, or panics.
func GetQ(ctx context.Context) *Queries {
	value := ctx.Value("db")
	if value == nil {
		panic("db not found in context")
	}
	db, ok := value.(*Queries)
	if !ok {
		t := reflect.TypeOf(value)
		s := fmt.Sprintf("failed to cast db (of type %v) to *Queries", t)
		panic(s)
	}
	return db
}
