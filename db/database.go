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

// Open opens the SQLite database at the given path and initializes
// the schema with CREATE TABLE IF NOT EXISTS + PRAGMA user_version = 1.
func Open(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	q := fmt.Sprintf("%s\nPRAGMA user_version = 1;", schema)
	if _, err := conn.Exec(q); err != nil {
		conn.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return conn, nil
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
