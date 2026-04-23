package epic

import (
	"fmt"
	"regexp"
	"strings"
)

// TaskID represents a validated hierarchical task identifier.
//
// IDs use colon-separated segments: the root segment is an ASCII slug
// matching [a-z][a-z0-9-]* (no leading/trailing hyphens), and subsequent
// segments are either the same slug pattern or positive integers.
type TaskID string

var (
	slugRe    = regexp.MustCompile(`^[a-z]([a-z0-9-]*[a-z0-9])?$`)
	numericRe = regexp.MustCompile(`^[1-9][0-9]*$`)
)

// ParseTaskID validates and returns a TaskID. Returns error if invalid.
func ParseTaskID(s string) (TaskID, error) {
	if s == "" {
		return "", fmt.Errorf("task ID cannot be empty")
	}
	if s != strings.TrimSpace(s) {
		return "", fmt.Errorf("task ID has leading/trailing whitespace: %q", s)
	}

	segments := strings.Split(s, ":")
	for i, seg := range segments {
		if seg == "" {
			return "", fmt.Errorf("task ID %q has empty segment", s)
		}
		if i == 0 {
			if !slugRe.MatchString(seg) {
				return "", fmt.Errorf("task ID root segment %q is not a valid slug", seg)
			}
		} else {
			if !slugRe.MatchString(seg) && !numericRe.MatchString(seg) {
				return "", fmt.Errorf("task ID segment %q is neither a valid slug nor a positive integer", seg)
			}
		}
	}

	return TaskID(s), nil
}

// Root returns the epic root slug (first segment).
func (id TaskID) Root() string {
	s := string(id)
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return s[:i]
	}
	return s
}

// Parent returns the parent ID, or empty string if this is the root.
func (id TaskID) Parent() TaskID {
	s := string(id)
	if i := strings.LastIndexByte(s, ':'); i >= 0 {
		return TaskID(s[:i])
	}
	return ""
}

// Depth returns the nesting depth (root = 0, root:1 = 1, etc.).
func (id TaskID) Depth() int {
	return strings.Count(string(id), ":")
}

// IsRoot returns true if this is a root-level epic ID.
func (id TaskID) IsRoot() bool {
	return !strings.Contains(string(id), ":")
}

// Ancestors returns all ancestor IDs from root to immediate parent (excludes self).
func (id TaskID) Ancestors() []TaskID {
	if id.IsRoot() {
		return nil
	}

	segments := strings.Split(string(id), ":")
	ancestors := make([]TaskID, 0, len(segments)-1)
	for i := 1; i < len(segments); i++ {
		ancestors = append(ancestors, TaskID(strings.Join(segments[:i], ":")))
	}
	return ancestors
}

// ChildID returns a child ID with the given numeric suffix.
func (id TaskID) ChildID(n int) TaskID {
	return TaskID(fmt.Sprintf("%s:%d", id, n))
}

// String returns the string representation.
func (id TaskID) String() string {
	return string(id)
}

// SiblingPrefix returns the prefix for matching siblings of this task.
// For non-root tasks, returns Parent() + ":" (e.g., "epic:" for "epic:1").
// For root tasks, returns the root + ":" for matching direct children.
func (id TaskID) SiblingPrefix() string {
	if id.IsRoot() {
		return string(id) + ":"
	}
	return string(id.Parent()) + ":"
}