package epic

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MHmorgan/agent-epics/db"
)

// SetAttribute sets an epic-level attribute. The caller is responsible for
// writing any associated system records (e.g. "summary written").
func SetAttribute(ctx context.Context, q *db.Queries, attr string, value string) error {
	err := q.SetAttribute(ctx, db.SetAttributeParams{
		Attribute: attr,
		Value:     sql.NullString{String: value, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("set attribute %s: %w", attr, err)
	}
	return nil
}

// GetAttribute gets an epic-level attribute value. Returns error if not found.
func GetAttribute(ctx context.Context, q *db.Queries, attr string) (string, error) {
	val, err := q.GetAttribute(ctx, attr)
	if err != nil {
		return "", fmt.Errorf("get attribute %s: %w", attr, err)
	}
	if !val.Valid {
		return "", fmt.Errorf("attribute %s has no value", attr)
	}
	return val.String, nil
}
