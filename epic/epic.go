package epic

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MHmorgan/agent-epics/db"
)

var epicIDRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidateEpicID checks that the epic ID matches ^[a-z][a-z0-9-]*$
// (the root slug charset — no slashes, dots, colons, or path separators).
// Returns an error if invalid.
func ValidateEpicID(s string) error {
	if !epicIDRe.MatchString(s) {
		return fmt.Errorf("invalid epic ID %q: must match [a-z][a-z0-9-]*", s)
	}
	return nil
}

// EpicInfo summarizes an epic for the listing.
type EpicInfo struct {
	ID     string
	Status Status // derived from leaf descendants
}

// epicPath constructs and validates the DB file path for an epic.
// It validates the epic ID first, then verifies the resolved path
// stays within epicsDir as defense-in-depth.
func epicPath(epicID, epicsDir string) (string, error) {
	if err := ValidateEpicID(epicID); err != nil {
		return "", err
	}
	p := filepath.Join(epicsDir, epicID+".db")
	rel, err := filepath.Rel(epicsDir, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path traversal rejected for epic ID %q", epicID)
	}
	return p, nil
}

// NewEpic creates a new epic DB file in epicsDir, inserts the root task
// (status=pending, empty body), and writes a "created" system record.
// Returns error if the file already exists or the ID is invalid.
func NewEpic(epicID string, epicsDir string) error {
	p, err := epicPath(epicID, epicsDir)
	if err != nil {
		return err
	}

	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("epic %q already exists", epicID)
	}

	conn, err := db.Open(p)
	if err != nil {
		return fmt.Errorf("create epic db: %w", err)
	}
	defer conn.Close()

	q := db.Q(conn)
	ctx := context.Background()

	err = q.InsertTask(ctx, db.InsertTaskParams{
		ID:     epicID,
		Status: sql.NullString{String: string(StatusPending), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("insert root task: %w", err)
	}

	if err := addSystemRecord(ctx, q, TaskID(epicID), "created"); err != nil {
		return fmt.Errorf("write system record: %w", err)
	}

	return nil
}

// OpenEpic opens an existing epic DB file by name. Returns the DB connection
// and Queries instance. Caller must close the DB.
func OpenEpic(epicID string, epicsDir string) (*sql.DB, *db.Queries, error) {
	p, err := epicPath(epicID, epicsDir)
	if err != nil {
		return nil, nil, err
	}

	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("epic %q not found", epicID)
	}

	conn, err := db.Open(p)
	if err != nil {
		return nil, nil, fmt.Errorf("open epic db: %w", err)
	}

	return conn, db.Q(conn), nil
}

// ListEpics scans the epics directory and returns info for each epic.
// Filters to only .db files. Validates each filename before constructing paths.
func ListEpics(epicsDir string) ([]EpicInfo, error) {
	entries, err := os.ReadDir(epicsDir)
	if err != nil {
		return nil, fmt.Errorf("read epics dir: %w", err)
	}

	var infos []EpicInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".db")
		if ValidateEpicID(id) != nil {
			continue
		}

		status, err := deriveEpicStatus(id, epicsDir)
		if err != nil {
			return nil, fmt.Errorf("derive status for %q: %w", id, err)
		}
		infos = append(infos, EpicInfo{ID: id, Status: status})
	}
	return infos, nil
}

// RemoveEpic deletes the DB file for an epic.
func RemoveEpic(epicID string, epicsDir string) error {
	p, err := epicPath(epicID, epicsDir)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		return fmt.Errorf("remove epic %q: %w", epicID, err)
	}
	return nil
}

// PurgeTerminalEpics removes all epics whose derived status is done or abandoned.
// Returns the list of purged epic IDs.
func PurgeTerminalEpics(epicsDir string) ([]string, error) {
	epics, err := ListEpics(epicsDir)
	if err != nil {
		return nil, err
	}

	var purged []string
	for _, e := range epics {
		if !IsTerminal(e.Status) {
			continue
		}
		if err := RemoveEpic(e.ID, epicsDir); err != nil {
			return purged, fmt.Errorf("purge %q: %w", e.ID, err)
		}
		purged = append(purged, e.ID)
	}
	return purged, nil
}

// deriveEpicStatus opens the epic DB, finds all leaf tasks, and computes
// the derived status from their statuses.
func deriveEpicStatus(epicID, epicsDir string) (Status, error) {
	conn, q, err := OpenEpic(epicID, epicsDir)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	ctx := context.Background()
	tasks, err := q.ListAllTasks(ctx)
	if err != nil {
		return "", fmt.Errorf("list tasks: %w", err)
	}

	// Build a set of IDs that are parents.
	parents := make(map[string]bool)
	for _, t := range tasks {
		if t.ParentID.Valid {
			parents[t.ParentID.String] = true
		}
	}

	// Collect statuses of leaf tasks (those not in the parents set).
	var leafStatuses []Status
	for _, t := range tasks {
		if !parents[t.ID] {
			leafStatuses = append(leafStatuses, Status(t.Status.String))
		}
	}

	return DerivedStatus(leafStatuses), nil
}
