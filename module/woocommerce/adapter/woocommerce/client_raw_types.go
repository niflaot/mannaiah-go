package woocommerce

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// flexibleInt defines tolerant integer decoding values for WooCommerce payload fields.
type flexibleInt int

// UnmarshalJSON decodes number and string JSON values into integer values.
func (value *flexibleInt) UnmarshalJSON(payload []byte) error {
	parsed, err := parseFlexibleNumber(strings.TrimSpace(string(payload)))
	if err != nil {
		return err
	}

	*value = flexibleInt(int(parsed))
	return nil
}

// flexibleFloat64 defines tolerant float decoding values for WooCommerce payload fields.
type flexibleFloat64 float64

// UnmarshalJSON decodes number and string JSON values into float values.
func (value *flexibleFloat64) UnmarshalJSON(payload []byte) error {
	parsed, err := parseFlexibleNumber(strings.TrimSpace(string(payload)))
	if err != nil {
		return err
	}

	*value = flexibleFloat64(parsed)
	return nil
}

// parseFlexibleNumber parses number-literal and quoted-number JSON values.
func parseFlexibleNumber(value string) (float64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "null" {
		return 0, nil
	}

	if strings.HasPrefix(trimmed, "\"") {
		var decoded string
		if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
			return 0, fmt.Errorf("decode quoted number: %w", err)
		}

		trimmed = strings.TrimSpace(decoded)
		if trimmed == "" {
			return 0, nil
		}
	}

	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, fmt.Errorf("parse number value %q: %w", trimmed, err)
	}

	return parsed, nil
}
