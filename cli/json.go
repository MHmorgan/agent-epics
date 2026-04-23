package cli

import (
	"encoding/json"
	"fmt"
)

// AgentError is an error with JSON with the error message in
type AgentError string

func (e AgentError) Error() string {
	return string(e)
}

// envelope is the standard JSON response wrapper for task commands.
type envelope struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// jsonSuccess returns a JSON string wrapping data in {"ok": true, "data": ...}.
func jsonSuccess(data any) string {
	b, err := json.Marshal(envelope{OK: true, Data: data})
	if err != nil {
		return fmt.Sprintf(`{"ok":false,"error":"marshal error: %s"}`, escapeJSON(err.Error()))
	}
	return string(b)
}

// jsonError returns a JSON AgentError wrapping an error in {"ok": false, "error": "..."}.
func jsonError(err error) AgentError {
	b, merr := json.Marshal(envelope{OK: false, Error: err.Error()})
	if merr != nil {
		return AgentError(fmt.Sprintf(`{"ok":false,"error":"marshal error: %s"}`, escapeJSON(err.Error())))
	}
	return AgentError(string(b))
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
