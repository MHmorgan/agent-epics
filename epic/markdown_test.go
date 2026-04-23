package epic

import (
	"testing"
)

func TestSplitMarkdown_BasicTwoSections(t *testing.T) {
	body := "First section\n---\nSecond section"
	sections, err := SplitMarkdown(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].Body != "First section" {
		t.Errorf("section 0 body = %q, want %q", sections[0].Body, "First section")
	}
	if sections[1].Body != "Second section" {
		t.Errorf("section 1 body = %q, want %q", sections[1].Body, "Second section")
	}
}

func TestSplitMarkdown_ThreeSections(t *testing.T) {
	body := "Alpha\n---\nBravo\n---\nCharlie"
	sections, err := SplitMarkdown(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(sections))
	}
	if sections[0].Body != "Alpha" {
		t.Errorf("section 0 body = %q, want %q", sections[0].Body, "Alpha")
	}
	if sections[1].Body != "Bravo" {
		t.Errorf("section 1 body = %q, want %q", sections[1].Body, "Bravo")
	}
	if sections[2].Body != "Charlie" {
		t.Errorf("section 2 body = %q, want %q", sections[2].Body, "Charlie")
	}
}

func TestSplitMarkdown_TitleExtraction(t *testing.T) {
	body := "# My Title\nSome content\n---\n# Other Title\nMore content"
	sections, err := SplitMarkdown(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].Title != "My Title" {
		t.Errorf("section 0 title = %q, want %q", sections[0].Title, "My Title")
	}
	if sections[1].Title != "Other Title" {
		t.Errorf("section 1 title = %q, want %q", sections[1].Title, "Other Title")
	}
}

func TestSplitMarkdown_NoHeading(t *testing.T) {
	body := "Just text\n---\nMore text"
	sections, err := SplitMarkdown(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sections[0].Title != "" {
		t.Errorf("section 0 title = %q, want empty", sections[0].Title)
	}
	if sections[1].Title != "" {
		t.Errorf("section 1 title = %q, want empty", sections[1].Title)
	}
}

func TestSplitMarkdown_SeparatorWithWhitespace(t *testing.T) {
	body := "First\n  ---  \nSecond"
	sections, err := SplitMarkdown(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].Body != "First" {
		t.Errorf("section 0 body = %q, want %q", sections[0].Body, "First")
	}
	if sections[1].Body != "Second" {
		t.Errorf("section 1 body = %q, want %q", sections[1].Body, "Second")
	}
}

func TestSplitMarkdown_NoSeparator(t *testing.T) {
	body := "Just a single section with no separator"
	_, err := SplitMarkdown(body)
	if err == nil {
		t.Fatal("expected error for input with no separator, got nil")
	}
}

func TestSplitMarkdown_EmptyBody(t *testing.T) {
	_, err := SplitMarkdown("")
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}

func TestSplitMarkdown_MultipleHeadingsFirstUsed(t *testing.T) {
	body := "# First Heading\nSome text\n# Second Heading\nMore text\n---\nOther section"
	sections, err := SplitMarkdown(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sections[0].Title != "First Heading" {
		t.Errorf("section 0 title = %q, want %q", sections[0].Title, "First Heading")
	}
	// Both heading lines should be in the body
	if sections[0].Body != "# First Heading\nSome text\n# Second Heading\nMore text" {
		t.Errorf("section 0 body = %q", sections[0].Body)
	}
}

func TestSplitMarkdown_LeadingTrailingWhitespaceTrimmed(t *testing.T) {
	body := "  \n  First section  \n  \n---\n\n  Second section\n  "
	sections, err := SplitMarkdown(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sections[0].Body != "First section" {
		t.Errorf("section 0 body = %q, want %q", sections[0].Body, "First section")
	}
	if sections[1].Body != "Second section" {
		t.Errorf("section 1 body = %q, want %q", sections[1].Body, "Second section")
	}
}

func TestSplitMarkdown_DoubleHashNotTitle(t *testing.T) {
	body := "## Not A Title\nContent\n---\n## Also Not A Title\nMore"
	sections, err := SplitMarkdown(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sections[0].Title != "" {
		t.Errorf("section 0 title = %q, want empty (## should not match)", sections[0].Title)
	}
	if sections[1].Title != "" {
		t.Errorf("section 1 title = %q, want empty (## should not match)", sections[1].Title)
	}
}

func TestSplitMarkdown_HeadingIncludedInBody(t *testing.T) {
	body := "# Title\nContent here\n---\nSecond part"
	sections, err := SplitMarkdown(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sections[0].Body != "# Title\nContent here" {
		t.Errorf("section 0 body = %q, want %q", sections[0].Body, "# Title\nContent here")
	}
}
