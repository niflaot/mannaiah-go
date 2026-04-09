package domain

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// NormalizeCarrierSlug normalizes carrier-like labels into lowercase, space-free slugs.
func NormalizeCarrierSlug(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}

	builder := strings.Builder{}
	for _, current := range norm.NFD.String(trimmed) {
		if unicode.Is(unicode.Mn, current) {
			continue
		}
		if (current >= 'a' && current <= 'z') || (current >= '0' && current <= '9') || current == '-' || current == '_' {
			builder.WriteRune(current)
		}
	}

	return builder.String()
}

// IsManualCarrierID reports whether the provided carrier identifier belongs to the manual carrier.
func IsManualCarrierID(carrierID string) bool {
	return strings.EqualFold(strings.TrimSpace(carrierID), "manual")
}

// ResolveTrackingCarrierSlug resolves the tracking carrier slug used in public tracking URLs.
func ResolveTrackingCarrierSlug(carrierID string, manualCarrierLabel string) string {
	if IsManualCarrierID(carrierID) {
		if slug := NormalizeCarrierSlug(manualCarrierLabel); slug != "" {
			return slug
		}
	}

	return NormalizeCarrierSlug(carrierID)
}
