package epic

import "testing"

func TestDerivedStatus(t *testing.T) {
	tests := []struct {
		name   string
		input  []Status
		expect Status
	}{
		{
			name:   "all done",
			input:  []Status{StatusDone, StatusDone, StatusDone},
			expect: StatusDone,
		},
		{
			name:   "all abandoned",
			input:  []Status{StatusAbandoned, StatusAbandoned},
			expect: StatusAbandoned,
		},
		{
			name:   "mix of done and abandoned yields done",
			input:  []Status{StatusDone, StatusAbandoned, StatusDone},
			expect: StatusDone,
		},
		{
			name:   "any active yields active",
			input:  []Status{StatusPending, StatusActive, StatusBlocked},
			expect: StatusActive,
		},
		{
			name:   "any blocked with none active yields blocked",
			input:  []Status{StatusPending, StatusBlocked, StatusDone},
			expect: StatusBlocked,
		},
		{
			name:   "some pending some done yields pending",
			input:  []Status{StatusPending, StatusDone},
			expect: StatusPending,
		},
		{
			name:   "empty slice yields pending",
			input:  []Status{},
			expect: StatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DerivedStatus(tt.input)
			if got != tt.expect {
				t.Errorf("DerivedStatus(%v) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}
