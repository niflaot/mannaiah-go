package store

import (
	"context"
	"fmt"
	"strings"
)

// resolveCategoryIDs resolves one category reference to category IDs usable in category_products lookups.
func (r *ProductCatalogRepository) resolveCategoryIDs(ctx context.Context, categoryRef string) ([]string, error) {
	trimmedRef := strings.TrimSpace(categoryRef)
	if trimmedRef == "" {
		return nil, nil
	}

	result := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	appendCategoryID := func(raw string) {
		value := strings.TrimSpace(raw)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	appendCategoryID(trimmedRef)

	var resolvedIDs []string
	if err := r.db.WithContext(ctx).
		Table("categories").
		Select("id").
		Where("deleted_at IS NULL AND (id = ? OR slug = ? OR LOWER(name) = LOWER(?))", trimmedRef, trimmedRef, trimmedRef).
		Pluck("id", &resolvedIDs).Error; err != nil {
		return nil, fmt.Errorf("resolve category identifiers: %w", err)
	}
	for _, id := range resolvedIDs {
		appendCategoryID(id)
	}

	return result, nil
}
