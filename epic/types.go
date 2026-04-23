package epic

import (
	"time"

	"github.com/MHmorgan/agent-epics/db"
)

// Task is the domain representation of a task.
type Task struct {
	ID        TaskID    `json:"id"`
	ParentID  TaskID    `json:"parent_id,omitempty"`
	Title     string    `json:"title,omitempty"`
	Body      string    `json:"body,omitempty"`
	Context   string    `json:"context,omitempty"`
	Status    Status    `json:"status,omitempty"`
	Position  int       `json:"position"`
	IsLeaf    bool      `json:"is_leaf"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Record is the domain representation of a record entry.
type Record struct {
	ID     int64  `json:"id"`
	Task   TaskID `json:"task"`
	Ts     time.Time `json:"ts"`
	Source string `json:"source"`
	Text   string `json:"text"`
}

// TaskFromDB converts a db.Task to a domain Task.
// isLeaf should be determined by the caller (e.g., CountChildren == 0).
func TaskFromDB(t db.Task, isLeaf bool) Task {
	return Task{
		ID:        TaskID(t.ID),
		ParentID:  TaskID(t.ParentID.String),
		Title:     t.Title.String,
		Body:      t.Body.String,
		Context:   t.Context.String,
		Status:    Status(t.Status.String),
		Position:  int(t.Position.Int64),
		IsLeaf:    isLeaf,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// RecordFromDB converts a db.Record to a domain Record.
func RecordFromDB(r db.Record) Record {
	return Record{
		ID:     r.ID,
		Task:   TaskID(r.Task),
		Ts:     r.Ts,
		Source: r.Source,
		Text:   r.Text,
	}
}

// DerivedStatus computes branch status from a list of leaf descendant statuses.
//
// Rules:
//   - done: every leaf is done or abandoned, and at least one is done
//   - abandoned: every leaf is abandoned
//   - active: any leaf is active
//   - blocked: any leaf is blocked (and none are active)
//   - pending: otherwise
func DerivedStatus(leafStatuses []Status) Status {
	if len(leafStatuses) == 0 {
		return StatusPending
	}

	var hasActive, hasBlocked, hasDone, hasPending bool

	for _, s := range leafStatuses {
		switch s {
		case StatusActive:
			hasActive = true
		case StatusBlocked:
			hasBlocked = true
		case StatusDone:
			hasDone = true
		case StatusAbandoned:
			// counted implicitly: if nothing else is set, all are abandoned
		case StatusPending:
			hasPending = true
		}
	}

	switch {
	case hasActive:
		return StatusActive
	case hasBlocked:
		return StatusBlocked
	case hasPending:
		return StatusPending
	case hasDone:
		return StatusDone
	default:
		return StatusAbandoned
	}
}
