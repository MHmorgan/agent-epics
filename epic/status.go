package epic

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusBlocked   Status = "blocked"
	StatusDone      Status = "done"
	StatusAbandoned Status = "abandoned"
)

// transitions defines the set of valid from -> to status changes.
var transitions = map[Status][]Status{
	StatusPending: {StatusActive, StatusBlocked, StatusAbandoned},
	StatusActive:  {StatusBlocked, StatusDone, StatusAbandoned},
	StatusBlocked: {StatusActive, StatusAbandoned},
}

// ValidTransition returns true if from -> to is allowed per the transition graph.
func ValidTransition(from, to Status) bool {
	for _, s := range transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// IsTerminal returns true if the status is done or abandoned.
func IsTerminal(s Status) bool {
	return s == StatusDone || s == StatusAbandoned
}

// TransitionRequiresReason returns true if transitioning TO this status requires a reason.
// Block and abandon require reasons.
func TransitionRequiresReason(s Status) bool {
	return s == StatusBlocked || s == StatusAbandoned
}
