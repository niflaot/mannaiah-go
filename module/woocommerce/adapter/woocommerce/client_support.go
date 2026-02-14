package woocommerce

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// shouldUseRawOrderFallback reports whether strict SDK decode failures should use tolerant raw decoding.
func shouldUseRawOrderFallback(err error) bool {
	if err == nil {
		return false
	}

	value := strings.ToLower(err.Error())
	markers := [...]string{
		"fuzzystringdecoder",
		"entity.order.meta",
		"entity.meta.value",
		"meta_data",
		"not number or string",
		"cannot unmarshal",
		"json:",
	}
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}

	return false
}

// normalizeMetadataValue converts dynamic metadata values to stable string representations.
func normalizeMetadataValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	case []any:
		if len(typed) == 1 {
			return normalizeMetadataValue(typed[0])
		}
		payload, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(payload)
	case map[string]any:
		payload, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(payload)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

// compactError normalizes and truncates error text for concise diagnostics.
func compactError(err error, limit int) string {
	if err == nil {
		return ""
	}

	value := strings.Join(strings.Fields(strings.TrimSpace(err.Error())), " ")
	if limit <= 0 || len(value) <= limit {
		return value
	}

	return value[:limit] + "..."
}

// parseWooOrderTime parses WooCommerce order date values.
func parseWooOrderTime(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}

	layouts := [...]string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.UTC()
		}
	}

	return time.Time{}
}

// validateConfig validates WooCommerce client configuration values.
func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.URL) == "" {
		return ErrInvalidURL
	}
	if strings.TrimSpace(cfg.ConsumerKey) == "" {
		return ErrInvalidConsumerKey
	}
	if strings.TrimSpace(cfg.ConsumerSecret) == "" {
		return ErrInvalidConsumerSecret
	}

	return nil
}

// resolveHasNextPage resolves pagination continuation behavior from header and payload signals.
func resolveHasNextPage(page int, pageSize int, itemCount int, totalPages int, isLastPage bool) bool {
	if totalPages > 0 && page < totalPages {
		return true
	}

	if pageSize > 0 && itemCount >= pageSize {
		return true
	}

	return !isLastPage
}
