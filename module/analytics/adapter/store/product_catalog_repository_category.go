package store

import (
	"context"
	"fmt"
	"strings"
)

// categoryLookupRow defines minimal category fields used in recommendation category resolution.
type categoryLookupRow struct {
	// ID is the category identifier.
	ID string `gorm:"column:id"`
	// IncludeChildren reports whether this category includes descendants in product resolution.
	IncludeChildren bool `gorm:"column:include_children"`
}

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

	resolvedRows := make([]categoryLookupRow, 0, 1)
	if err := r.db.WithContext(ctx).
		Table("categories").
		Select("id, include_children").
		Where("deleted_at IS NULL AND (id = ? OR slug = ? OR LOWER(name) = LOWER(?))", trimmedRef, trimmedRef, trimmedRef).
		Scan(&resolvedRows).Error; err != nil {
		return nil, fmt.Errorf("resolve category identifiers: %w", err)
	}
	for _, row := range resolvedRows {
		appendCategoryID(row.ID)
	}

	descendantRootIDs := make([]string, 0, len(resolvedRows))
	for _, row := range resolvedRows {
		if row.IncludeChildren {
			descendantRootIDs = append(descendantRootIDs, row.ID)
		}
	}
	if len(descendantRootIDs) > 0 {
		descendantIDs, err := r.resolveDescendantCategoryIDs(ctx, descendantRootIDs)
		if err != nil {
			return nil, err
		}
		for _, descendantID := range descendantIDs {
			appendCategoryID(descendantID)
		}
	}

	return result, nil
}

// resolveDescendantCategoryIDs resolves all recursive descendant category IDs for root category IDs.
func (r *ProductCatalogRepository) resolveDescendantCategoryIDs(ctx context.Context, rootCategoryIDs []string) ([]string, error) {
	if len(rootCategoryIDs) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(rootCategoryIDs))
	queue := make([]string, 0, len(rootCategoryIDs))
	for _, rootCategoryID := range rootCategoryIDs {
		resolved := strings.TrimSpace(rootCategoryID)
		if resolved == "" {
			continue
		}
		if _, exists := seen[resolved]; exists {
			continue
		}
		seen[resolved] = struct{}{}
		queue = append(queue, resolved)
	}
	if len(queue) == 0 {
		return nil, nil
	}

	descendants := make([]string, 0)
	for len(queue) > 0 {
		currentIDs := queue
		queue = nil

		childIDs := make([]string, 0)
		if err := r.db.WithContext(ctx).
			Table("categories").
			Select("id").
			Where("deleted_at IS NULL AND parent_id IN ?", currentIDs).
			Pluck("id", &childIDs).Error; err != nil {
			return nil, fmt.Errorf("resolve descendant category identifiers: %w", err)
		}
		for _, childID := range childIDs {
			resolvedChildID := strings.TrimSpace(childID)
			if resolvedChildID == "" {
				continue
			}
			if _, exists := seen[resolvedChildID]; exists {
				continue
			}
			seen[resolvedChildID] = struct{}{}
			descendants = append(descendants, resolvedChildID)
			queue = append(queue, resolvedChildID)
		}
	}

	return descendants, nil
}
