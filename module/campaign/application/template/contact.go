package template

import "strings"

// ExtractFirstName returns the first word of name before the first space.
// Returns the full name unchanged when no space is present.
func ExtractFirstName(name string) string {
	trimmed := strings.TrimSpace(name)
	if idx := strings.IndexByte(trimmed, ' '); idx > 0 {
		return trimmed[:idx]
	}

	return trimmed
}
