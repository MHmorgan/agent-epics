package epic

import "testing"

func TestValidTransition(t *testing.T) {
	valid := []struct {
		from, to Status
	}{
		{StatusPending, StatusActive},
		{StatusPending, StatusBlocked},
		{StatusPending, StatusAbandoned},
		{StatusActive, StatusBlocked},
		{StatusActive, StatusDone},
		{StatusActive, StatusAbandoned},
		{StatusBlocked, StatusActive},
		{StatusBlocked, StatusAbandoned},
	}
	for _, tc := range valid {
		if !ValidTransition(tc.from, tc.to) {
			t.Errorf("expected %s -> %s to be valid", tc.from, tc.to)
		}
	}

	invalid := []struct {
		from, to Status
	}{
		// Terminal states cannot transition out.
		{StatusDone, StatusPending},
		{StatusDone, StatusActive},
		{StatusDone, StatusBlocked},
		{StatusDone, StatusAbandoned},
		{StatusAbandoned, StatusPending},
		{StatusAbandoned, StatusActive},
		{StatusAbandoned, StatusBlocked},
		{StatusAbandoned, StatusDone},

		// Self-transitions are invalid.
		{StatusPending, StatusPending},
		{StatusActive, StatusActive},
		{StatusBlocked, StatusBlocked},

		// Specific disallowed transitions.
		{StatusPending, StatusDone},   // must go through active first
		{StatusBlocked, StatusDone},   // must unblock (go active) first
		{StatusBlocked, StatusBlocked}, // duplicate, but explicit
		{StatusActive, StatusPending},  // no going backwards
		{StatusBlocked, StatusPending}, // no going backwards
	}
	for _, tc := range invalid {
		if ValidTransition(tc.from, tc.to) {
			t.Errorf("expected %s -> %s to be invalid", tc.from, tc.to)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	terminal := []Status{StatusDone, StatusAbandoned}
	for _, s := range terminal {
		if !IsTerminal(s) {
			t.Errorf("expected %s to be terminal", s)
		}
	}

	nonTerminal := []Status{StatusPending, StatusActive, StatusBlocked}
	for _, s := range nonTerminal {
		if IsTerminal(s) {
			t.Errorf("expected %s to be non-terminal", s)
		}
	}
}

func TestTransitionRequiresReason(t *testing.T) {
	requiresReason := []Status{StatusBlocked, StatusAbandoned}
	for _, s := range requiresReason {
		if !TransitionRequiresReason(s) {
			t.Errorf("expected transition to %s to require a reason", s)
		}
	}

	noReason := []Status{StatusPending, StatusActive, StatusDone}
	for _, s := range noReason {
		if TransitionRequiresReason(s) {
			t.Errorf("expected transition to %s to not require a reason", s)
		}
	}
}
