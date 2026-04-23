package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func userVersion(t *testing.T, conn *sql.DB) int {
	t.Helper()
	var v int
	if err := conn.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	return v
}

func tableExists(t *testing.T, conn *sql.DB, name string) bool {
	t.Helper()
	var n int
	err := conn.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name,
	).Scan(&n)
	if err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	return n > 0
}

func TestOpen_FreshDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fresh.db")

	conn, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	if v := userVersion(t, conn); v != len(migrations) {
		t.Errorf("user_version = %d, want %d", v, len(migrations))
	}

	for _, table := range []string{"task", "attribute", "record", "dep"} {
		if !tableExists(t, conn, table) {
			t.Errorf("table %q not found after migration", table)
		}
	}
}

func TestOpen_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idem.db")

	// First open: applies migrations.
	conn, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}

	// Insert a row so we can verify the DB is not wiped on re-open.
	_, err = conn.Exec(`INSERT INTO attribute (attribute, value) VALUES ('test_key', 'test_val')`)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	conn.Close()

	// Second open: should not re-run migrations.
	conn, err = Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer conn.Close()

	if v := userVersion(t, conn); v != len(migrations) {
		t.Errorf("user_version = %d, want %d", v, len(migrations))
	}

	var val string
	err = conn.QueryRow("SELECT value FROM attribute WHERE attribute = 'test_key'").Scan(&val)
	if err != nil {
		t.Fatalf("select after re-open: %v", err)
	}
	if val != "test_val" {
		t.Errorf("value = %q, want %q", val, "test_val")
	}
}

func TestOpen_VersionZeroGetsMigrated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v0.db")

	// Create a bare SQLite database at version 0 (the default).
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("create raw db: %v", err)
	}
	// Ensure the file is actually written (empty DB).
	if _, err := raw.Exec("SELECT 1"); err != nil {
		t.Fatalf("ping: %v", err)
	}
	raw.Close()

	// Verify the file exists and version is 0.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("db file missing: %v", err)
	}

	conn, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	if v := userVersion(t, conn); v != len(migrations) {
		t.Errorf("user_version = %d, want %d", v, len(migrations))
	}

	if !tableExists(t, conn, "task") {
		t.Error("task table not created from version 0")
	}
}

func TestOpen_ForeignKeysEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fk.db")

	conn, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer conn.Close()

	var fk int
	if err := conn.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("read foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}
