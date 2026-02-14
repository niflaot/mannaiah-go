package orders

import "strings"

// normalizeMetadata normalizes metadata keys and values and removes empty keys.
func normalizeMetadata(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}

	normalized := make(map[string]string, len(values))
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		normalized[trimmedKey] = strings.TrimSpace(value)
	}
	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

// normalizeRealm resolves empty realm values to WooCommerce defaults.
func normalizeRealm(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultRealm
	}

	return trimmed
}
