package recommendation

import "strings"

// scopedProductSelections defines normalized product IDs and optional per-product variation selections.
type scopedProductSelections struct {
	// ProductIDs preserves first-seen product selection order across plain/scoped tokens.
	ProductIDs []string
	// PlainProductIDs includes only plain product tokens without scoped variation values.
	PlainProductIDs []string
	// ScopedVariationIDsByProduct maps product IDs to explicit variation IDs selected for those products.
	ScopedVariationIDsByProduct map[string][]string
}

// parseScopedProductSelections parses product tokens with optional scoped variation values.
func parseScopedProductSelections(values []string) scopedProductSelections {
	result := scopedProductSelections{
		ProductIDs:                  make([]string, 0, len(values)),
		PlainProductIDs:             make([]string, 0, len(values)),
		ScopedVariationIDsByProduct: make(map[string][]string),
	}
	if len(values) == 0 {
		return result
	}

	seenProducts := make(map[string]struct{}, len(values))
	seenPlainProducts := make(map[string]struct{}, len(values))
	seenScopedVariationsByProduct := make(map[string]map[string]struct{}, len(values))

	for _, raw := range values {
		productID, variationID, hasScopedVariation := parseScopedProductToken(raw)
		if productID == "" {
			continue
		}
		if _, ok := seenProducts[productID]; !ok {
			seenProducts[productID] = struct{}{}
			result.ProductIDs = append(result.ProductIDs, productID)
		}
		if !hasScopedVariation {
			if _, ok := seenPlainProducts[productID]; !ok {
				seenPlainProducts[productID] = struct{}{}
				result.PlainProductIDs = append(result.PlainProductIDs, productID)
			}
			continue
		}

		if seenScopedVariationsByProduct[productID] == nil {
			seenScopedVariationsByProduct[productID] = make(map[string]struct{}, 1)
		}
		if _, exists := seenScopedVariationsByProduct[productID][variationID]; exists {
			continue
		}
		seenScopedVariationsByProduct[productID][variationID] = struct{}{}
		result.ScopedVariationIDsByProduct[productID] = append(result.ScopedVariationIDsByProduct[productID], variationID)
	}

	return result
}

// parseScopedProductToken parses one token in the form "<product-id>[<sep><variation-id>]".
func parseScopedProductToken(value string) (productID string, variationID string, hasScopedVariation bool) {
	token := strings.TrimSpace(value)
	if token == "" {
		return "", "", false
	}
	for _, sep := range []string{"::", "|", "#", "@"} {
		left, right, ok := splitScopedProductToken(token, sep)
		if !ok {
			continue
		}
		return left, right, true
	}
	if !strings.Contains(token, "://") {
		if left, right, ok := splitScopedProductToken(token, ":"); ok {
			return left, right, true
		}
	}

	return token, "", false
}

// splitScopedProductToken splits one token by separator and normalizes both parts.
func splitScopedProductToken(token string, separator string) (productID string, variationID string, ok bool) {
	idx := strings.Index(token, separator)
	if idx <= 0 || idx >= len(token)-len(separator) {
		return "", "", false
	}
	left := strings.TrimSpace(token[:idx])
	right := normalizeVariationToken(token[idx+len(separator):])
	if left == "" || right == "" {
		return "", "", false
	}

	return left, right, true
}

// subtractVariationIDs removes blocked variation IDs from ordered variation candidates.
func subtractVariationIDs(values []string, blockedValues []string) []string {
	if len(values) == 0 || len(blockedValues) == 0 {
		return values
	}

	blockedSet := make(map[string]struct{}, len(blockedValues))
	for _, rawBlockedValue := range blockedValues {
		if blocked := normalizeVariationToken(rawBlockedValue); blocked != "" {
			blockedSet[blocked] = struct{}{}
		}
	}
	if len(blockedSet) == 0 {
		return values
	}

	filtered := make([]string, 0, len(values))
	for _, rawValue := range values {
		normalizedValue := normalizeVariationToken(rawValue)
		if normalizedValue == "" {
			continue
		}
		if _, blocked := blockedSet[normalizedValue]; blocked {
			continue
		}
		filtered = append(filtered, normalizedValue)
	}
	if len(filtered) == 0 {
		return nil
	}

	return filtered
}
