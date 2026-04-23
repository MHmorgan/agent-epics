package epic

import (
	"errors"
	"strings"
)

// Section represents one chunk of a split markdown body.
type Section struct {
	Title string // extracted from first # Heading, may be empty
	Body  string // the full chunk text (including the heading line)
}

// SplitMarkdown splits a markdown body on lines that are exactly "---"
// (with optional leading/trailing whitespace on the line).
// Returns the sections. Returns error if fewer than 2 sections result.
func SplitMarkdown(body string) ([]Section, error) {
	lines := strings.Split(body, "\n")

	// Split lines into chunks separated by "---" lines.
	chunks := [][]string{{}}
	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			chunks = append(chunks, []string{})
		} else {
			chunks[len(chunks)-1] = append(chunks[len(chunks)-1], line)
		}
	}

	if len(chunks) < 2 {
		return nil, errors.New("no --- separator found; need at least 2 sections")
	}

	sections := make([]Section, len(chunks))
	for i, chunk := range chunks {
		text := strings.TrimSpace(strings.Join(chunk, "\n"))
		sections[i] = Section{
			Title: extractTitle(text),
			Body:  text,
		}
	}

	return sections, nil
}

// extractTitle returns the text after "# " from the first H1 heading line,
// or empty string if none found. Only single-# headings match.
func extractTitle(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return ""
}
