package epic

import (
	"testing"
)

func TestParseTaskID_Valid(t *testing.T) {
	valid := []string{
		"my-epic",
		"my-epic:1",
		"my-epic:1:2",
		"a",
		"abc:123:456",
		"my--epic",
		"a1",
		"ab2c",
		"foo:bar:1",
		"x:1:y:2",
	}
	for _, s := range valid {
		t.Run(s, func(t *testing.T) {
			id, err := ParseTaskID(s)
			if err != nil {
				t.Fatalf("ParseTaskID(%q) returned error: %v", s, err)
			}
			if id.String() != s {
				t.Fatalf("String() = %q, want %q", id.String(), s)
			}
		})
	}
}

func TestParseTaskID_Invalid(t *testing.T) {
	invalid := []struct {
		input string
		desc  string
	}{
		{"", "empty string"},
		{"123", "starts with number"},
		{"My-Epic", "uppercase"},
		{"-epic", "leading hyphen"},
		{"epic-", "trailing hyphen"},
		{"epic:", "trailing colon"},
		{":epic", "leading colon"},
		{"epic::1", "double colon"},
		{"epic/foo", "slash"},
		{"epic.foo", "dot"},
		{"epic:0", "zero child number"},
		{"epic:-1", "negative child number"},
		{"epic:01", "leading zero in number"},
		{"LOUD", "all uppercase"},
		{"a:b:", "trailing colon after slug"},
		{"a::b", "double colon between slugs"},
		{" epic", "leading space"},
		{"epic ", "trailing space"},
		{"epic:1: ", "trailing space after colon"},
	}
	for _, tc := range invalid {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := ParseTaskID(tc.input)
			if err == nil {
				t.Fatalf("ParseTaskID(%q) should have returned error (%s)", tc.input, tc.desc)
			}
		})
	}
}

func TestTaskID_Root(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-epic", "my-epic"},
		{"my-epic:1", "my-epic"},
		{"my-epic:1:2", "my-epic"},
		{"abc:123:456", "abc"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			id, _ := ParseTaskID(tc.input)
			if got := id.Root(); got != tc.want {
				t.Fatalf("Root() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTaskID_Parent(t *testing.T) {
	tests := []struct {
		input string
		want  TaskID
	}{
		{"my-epic", ""},
		{"my-epic:1", "my-epic"},
		{"my-epic:1:2", "my-epic:1"},
		{"abc:123:456", "abc:123"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			id, _ := ParseTaskID(tc.input)
			if got := id.Parent(); got != tc.want {
				t.Fatalf("Parent() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTaskID_Depth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"my-epic", 0},
		{"my-epic:1", 1},
		{"my-epic:1:2", 2},
		{"abc:123:456", 2},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			id, _ := ParseTaskID(tc.input)
			if got := id.Depth(); got != tc.want {
				t.Fatalf("Depth() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestTaskID_IsRoot(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"my-epic", true},
		{"my-epic:1", false},
		{"my-epic:1:2", false},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			id, _ := ParseTaskID(tc.input)
			if got := id.IsRoot(); got != tc.want {
				t.Fatalf("IsRoot() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTaskID_Ancestors(t *testing.T) {
	tests := []struct {
		input string
		want  []TaskID
	}{
		{"my-epic", nil},
		{"my-epic:1", []TaskID{"my-epic"}},
		{"my-epic:1:2", []TaskID{"my-epic", "my-epic:1"}},
		{"a:1:2:3", []TaskID{"a", "a:1", "a:1:2"}},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			id, _ := ParseTaskID(tc.input)
			got := id.Ancestors()
			if len(got) != len(tc.want) {
				t.Fatalf("Ancestors() len = %d, want %d", len(got), len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("Ancestors()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestTaskID_ChildID(t *testing.T) {
	tests := []struct {
		input string
		n     int
		want  TaskID
	}{
		{"my-epic", 1, "my-epic:1"},
		{"my-epic:1", 2, "my-epic:1:2"},
		{"abc", 42, "abc:42"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			id, _ := ParseTaskID(tc.input)
			if got := id.ChildID(tc.n); got != tc.want {
				t.Fatalf("ChildID(%d) = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}

func TestTaskID_SiblingPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-epic", "my-epic:"},
		{"my-epic:1", "my-epic:"},
		{"my-epic:1:2", "my-epic:1:"},
		{"a:1:2:3", "a:1:2:"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			id, _ := ParseTaskID(tc.input)
			if got := id.SiblingPrefix(); got != tc.want {
				t.Fatalf("SiblingPrefix() = %q, want %q", got, tc.want)
			}
		})
	}
}
