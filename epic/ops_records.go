package epic

import (
	"context"
	"fmt"

	"github.com/MHmorgan/agent-epics/db"
)

// GetRecords returns records for a task. If selfOnly is true, returns only
// records for the exact task ID. Otherwise returns subtree records
// (where task = id OR task LIKE id||':%').
func GetRecords(ctx context.Context, q *db.Queries, id TaskID, selfOnly bool) ([]Record, error) {
	var rows []db.Record
	var err error

	if selfOnly {
		rows, err = q.ListRecordsByTask(ctx, id.String())
	} else {
		rows, err = q.ListRecordsByPrefix(ctx, id.String())
	}
	if err != nil {
		return nil, fmt.Errorf("get records for %s: %w", id, err)
	}

	records := make([]Record, len(rows))
	for i, r := range rows {
		records[i] = RecordFromDB(r)
	}
	return records, nil
}
