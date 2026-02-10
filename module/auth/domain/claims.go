package domain

import "strings"

// Claims defines normalized JWT claims used by auth application services.
type Claims struct {
	// Subject defines principal identifier.
	Subject string
	// Issuer defines token issuer claim.
	Issuer string
	// Audience defines token audience claim values.
	Audience []string
	// Scope defines space-delimited permission scopes.
	Scope string
	// Raw stores original claim key-value pairs.
	Raw map[string]any
}

// Scopes parses scope values into a normalized list.
func (c *Claims) Scopes() []string {
	if c == nil {
		return nil
	}

	parts := strings.Fields(strings.TrimSpace(c.Scope))
	if len(parts) == 0 {
		return nil
	}

	result := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			result = append(result, value)
		}
	}

	return result
}
