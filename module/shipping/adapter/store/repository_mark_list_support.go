package store

import (
	"strings"

	"gorm.io/gorm"
)

// applyMarkSearchTerm applies free-text search filters to shipping mark list queries.
func applyMarkSearchTerm(builder *gorm.DB, term string) *gorm.DB {
	if builder == nil {
		return builder
	}
	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(term)))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		likePattern := "%" + token + "%"
		builder = builder.Where(
			"(LOWER(tracking_number) LIKE ? OR LOWER(order_id) LIKE ? OR LOWER(recipient_name) LIKE ? OR LOWER(recipient_legal_name) LIKE ?)",
			likePattern,
			likePattern,
			likePattern,
			likePattern,
		)
	}

	return builder
}
