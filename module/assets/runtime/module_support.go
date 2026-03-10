package runtime

import (
	"strings"
	"time"
)

const (
	defaultWorkerTimeout = 5 * time.Minute
)

// resolveWorkerTags parses comma-separated config tags into normalized values.
func resolveWorkerTags(raw string) []string {
	segments := strings.Split(strings.TrimSpace(raw), ",")
	seen := map[string]struct{}{}
	tags := make([]string, 0, len(segments))

	for _, segment := range segments {
		trimmed := strings.ToLower(strings.TrimSpace(segment))
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		tags = append(tags, trimmed)
	}

	return tags
}

// resolveWorkerTimeout normalizes configured worker timeout values.
func resolveWorkerTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return defaultWorkerTimeout
	}

	return time.Duration(timeoutMS) * time.Millisecond
}
