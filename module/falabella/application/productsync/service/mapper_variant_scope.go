package service

import (
	"strings"
	"unicode"
)

// applyVariantScopedAttributes applies variant-SKU scoped attributes (for example, "0013.color") into mapped attributes.
func applyVariantScopedAttributes(target map[string]string, source map[string]string, variantSKU string, knownVariantSKUs map[string]struct{}) {
	if target == nil || len(source) == 0 {
		return
	}

	normalizedVariantSKU := normalizeScopedVariantToken(variantSKU)
	if normalizedVariantSKU == "" {
		return
	}

	resolvedKnownVariantSKUs := normalizeKnownVariantSKUs(knownVariantSKUs, normalizedVariantSKU)
	for rawKey, rawValue := range source {
		scope, key, scoped := splitScopedAttributeKey(rawKey)
		if !scoped {
			continue
		}

		normalizedScope := normalizeScopedVariantToken(scope)
		if normalizedScope == "" {
			continue
		}
		if _, ok := resolvedKnownVariantSKUs[normalizedScope]; !ok {
			continue
		}

		delete(target, rawKey)
		if normalizedScope != normalizedVariantSKU {
			continue
		}

		value := strings.TrimSpace(rawValue)
		if value == "" {
			continue
		}

		normalizedKey := normalizeScopedAttributeKey(key)
		if normalizedKey == "" {
			continue
		}
		target[normalizedKey] = value
	}
}

// splitScopedAttributeKey resolves "<scope>.<key>" segments from attribute keys.
func splitScopedAttributeKey(key string) (string, string, bool) {
	scope, field, ok := strings.Cut(strings.TrimSpace(key), ".")
	if !ok {
		return "", "", false
	}
	if strings.TrimSpace(scope) == "" || strings.TrimSpace(field) == "" {
		return "", "", false
	}

	return scope, field, true
}

// normalizeKnownVariantSKUs resolves normalized variant SKU sets with fallback to the current variant SKU.
func normalizeKnownVariantSKUs(known map[string]struct{}, fallbackVariantSKU string) map[string]struct{} {
	if len(known) == 0 {
		return map[string]struct{}{fallbackVariantSKU: {}}
	}

	normalized := make(map[string]struct{}, len(known))
	for sku := range known {
		normalizedSKU := normalizeScopedVariantToken(sku)
		if normalizedSKU == "" {
			continue
		}
		normalized[normalizedSKU] = struct{}{}
	}
	if len(normalized) == 0 {
		normalized[fallbackVariantSKU] = struct{}{}
	}

	return normalized
}

// normalizeScopedVariantToken resolves normalized variant-SKU tokens for case-insensitive comparisons.
func normalizeScopedVariantToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// normalizeScopedAttributeKey resolves canonical Falabella attribute names from scoped attribute keys.
func normalizeScopedAttributeKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	switch normalizeAttributeToken(trimmed) {
	case "color":
		return "Color"
	case "colorbase", "colorbasico", "colorbasic", "basiccolor", "basecolor":
		return "ColorBasico"
	case "size", "talla":
		return "Talla"
	case "businessunits", "businessunit":
		return "BusinessUnits"
	default:
		return trimmed
	}
}

// normalizeAttributeToken resolves case-insensitive alphanumeric tokens from attribute names.
func normalizeAttributeToken(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	buffer := make([]rune, 0, len(trimmed))
	for _, runeValue := range trimmed {
		if unicode.IsLetter(runeValue) || unicode.IsDigit(runeValue) {
			buffer = append(buffer, unicode.ToLower(runeValue))
		}
	}

	return string(buffer)
}

// normalizeFalabellaAttributeKeys canonicalizes known Falabella attribute aliases into required key names.
func normalizeFalabellaAttributeKeys(attributes map[string]string) {
	if len(attributes) == 0 {
		return
	}

	for key, value := range copyAttributes(attributes) {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}

		canonical := normalizeScopedAttributeKey(trimmedKey)
		if canonical == trimmedKey {
			continue
		}

		existing := strings.TrimSpace(attributes[canonical])
		if existing == "" {
			attributes[canonical] = value
		}
		delete(attributes, key)
	}
}
