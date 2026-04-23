package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// readStdin reads all of stdin and returns the content as a trimmed string.
func readStdin() (string, error) {
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}

	// TrimRight (not TrimSpace) so intentional leading whitespace is preserved;
	// only trailing newlines, carriage returns, tabs, and spaces are removed.
	s := strings.TrimRight(string(b), "\n\r\t ")

	if s == "" {
		return "", errors.New("empty input on stdin")
	}

	return s, nil
}
