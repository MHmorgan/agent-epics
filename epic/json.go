package epic

import (
	"encoding/json"
	"fmt"
)

// envelope is the standard JSON response wrapper for task commands.
type envelope struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// JSONSuccess returns a JSON string wrapping data in {"ok": true, "data": ...}.
func JSONSuccess(data any) string {
	b, err := json.Marshal(envelope{OK: true, Data: data})
	if err != nil {
		return fmt.Sprintf(`{"ok":false,"error":"marshal error: %s"}`, escapeJSON(err.Error()))
	}
	return string(b)
}

// JSONError returns a JSON string wrapping an error in {"ok": false, "error": "..."}.
func JSONError(err error) string {
	b, merr := json.Marshal(envelope{OK: false, Error: err.Error()})
	if merr != nil {
		return fmt.Sprintf(`{"ok":false,"error":"marshal error: %s"}`, escapeJSON(err.Error()))
	}
	return string(b)
}

// escapeJSON escapes a string for safe embedding in a JSON string literal.
func escapeJSON(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return "unknown error"
	}
	// Strip surrounding quotes from the marshaled string.
	return string(b[1 : len(b)-1])
}
