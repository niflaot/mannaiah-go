package category

import (
	"context"
	"fmt"
	"strings"

	productstore "mannaiah/module/products/adapter/store/product"
	categorydomain "mannaiah/module/products/domain/category"
	categoryport "mannaiah/module/products/port/category"
)

// ListProducts resolves and returns paginated products for a category.
func (r *Repository) ListProducts(ctx context.Context, q categoryport.ListProductsQuery) (*categoryport.ListProductsResult, error) {
	cat, err := r.GetByID(ctx, q.CategoryID)
	if err != nil {
		return nil, err
	}

	categories := []*categorydomain.Category{cat}
	if cat.IncludeChildren {
		descendants, err := r.collectDescendants(ctx, cat.ID)
		if err != nil {
			return nil, err
		}
		for _, descendantID := range descendants {
			descendant, getErr := r.GetByID(ctx, descendantID)
			if getErr != nil {
				return nil, fmt.Errorf("load descendant category %s: %w", descendantID, getErr)
			}
			categories = append(categories, descendant)
		}
	}

	hasAnySource := false
	for _, category := range categories {
		if len(category.Filter.Tags) > 0 ||
			category.Filter.PriceRange != nil ||
			len(category.Filter.CategoryRefs) > 0 ||
			len(category.ProductIDs) > 0 {
			hasAnySource = true
			break
		}
	}
	if !hasAnySource {
		page, pageSize := normalizePagination(q.Page, q.PageSize)

		return &categoryport.ListProductsResult{
			Items:    nil,
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	orderedIDs := make([]string, 0)
	seenIDs := make(map[string]struct{})
	for _, category := range categories {
		if err := r.collectCategoryScopedProductIDs(ctx, category, &orderedIDs, seenIDs); err != nil {
			return nil, err
		}
	}

	if len(orderedIDs) == 0 {
		page, pageSize := normalizePagination(q.Page, q.PageSize)

		return &categoryport.ListProductsResult{
			Items:    nil,
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	page, pageSize := normalizePagination(q.Page, q.PageSize)
	total := int64(len(orderedIDs))
	offset := (page - 1) * pageSize
	if offset >= int(total) {
		return &categoryport.ListProductsResult{
			Items:    nil,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	pagedIDs := orderedIDs
	end := offset + pageSize
	if offset > 0 || end < len(orderedIDs) {
		if offset >= len(orderedIDs) {
			pagedIDs = nil
		} else {
			if end > len(orderedIDs) {
				end = len(orderedIDs)
			}
			pagedIDs = orderedIDs[offset:end]
		}
	}

	productRepo, err := productstore.NewRepository(r.db)
	if err != nil {
		return nil, fmt.Errorf("build product repository for category products: %w", err)
	}

	items, err := productRepo.GetByIDs(ctx, pagedIDs)
	if err != nil {
		return nil, fmt.Errorf("load category product entities: %w", err)
	}

	return &categoryport.ListProductsResult{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// collectCategoryScopedProductIDs resolves one category's product IDs from pinned and filter criteria.
func (r *Repository) collectCategoryScopedProductIDs(
	ctx context.Context,
	cat *categorydomain.Category,
	orderedIDs *[]string,
	seenIDs map[string]struct{},
) error {
	if cat == nil {
		return nil
	}

	appendUniqueProductIDs(orderedIDs, seenIDs, cat.ProductIDs)

	if len(cat.Filter.Tags) > 0 {
		var tagProductIDs []string
		err := r.db.WithContext(ctx).
			Table("product_tags").
			Select("product_tags.product_id").
			Joins("JOIN tags ON tags.id = product_tags.tag_id AND tags.deleted_at IS NULL").
			Where("tags.name IN ?", cat.Filter.Tags).
			Group("product_tags.product_id").
			Pluck("product_id", &tagProductIDs).Error
		if err != nil {
			return fmt.Errorf("filter products by tags for category %s: %w", cat.ID, err)
		}

		priceQuery := r.db.WithContext(ctx).
			Table("products").
			Select("id").
			Where("deleted_at IS NULL AND id IN ?", tagProductIDs).
			Order("created_at asc, id asc")
		if cat.Filter.PriceRange != nil {
			if cat.Filter.PriceRange.Min != nil {
				priceQuery = priceQuery.Where("price >= ?", *cat.Filter.PriceRange.Min)
			}
			if cat.Filter.PriceRange.Max != nil {
				priceQuery = priceQuery.Where("price <= ?", *cat.Filter.PriceRange.Max)
			}
		}

		var filteredIDs []string
		if err := priceQuery.Pluck("id", &filteredIDs).Error; err != nil {
			return fmt.Errorf("apply price filter on tag results for category %s: %w", cat.ID, err)
		}
		appendUniqueProductIDs(orderedIDs, seenIDs, filteredIDs)
	} else if cat.Filter.PriceRange != nil {
		priceQuery := r.db.WithContext(ctx).
			Table("products").
			Select("id").
			Where("deleted_at IS NULL").
			Order("created_at asc, id asc")
		if cat.Filter.PriceRange.Min != nil {
			priceQuery = priceQuery.Where("price >= ?", *cat.Filter.PriceRange.Min)
		}
		if cat.Filter.PriceRange.Max != nil {
			priceQuery = priceQuery.Where("price <= ?", *cat.Filter.PriceRange.Max)
		}

		var priceIDs []string
		if err := priceQuery.Pluck("id", &priceIDs).Error; err != nil {
			return fmt.Errorf("filter products by price for category %s: %w", cat.ID, err)
		}
		appendUniqueProductIDs(orderedIDs, seenIDs, priceIDs)
	}

	for _, refCatID := range cat.Filter.CategoryRefs {
		var refRows []categoryProductRecord
		if err := r.db.WithContext(ctx).Where("category_id = ?", refCatID).Order("position asc, id asc").Find(&refRows).Error; err != nil {
			return fmt.Errorf("load ref category products for category %s: %w", cat.ID, err)
		}
		appendCategoryProductRows(orderedIDs, seenIDs, refRows)
	}

	return nil
}

// collectDescendants collects all descendant category IDs recursively.
func (r *Repository) collectDescendants(ctx context.Context, parentID string) ([]string, error) {
	var result []string
	queue := []string{parentID}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		var childIDs []string
		if err := r.db.WithContext(ctx).Model(&categoryRecord{}).Select("id").Where("parent_id = ? AND deleted_at IS NULL", current).Order("created_at asc, id asc").Pluck("id", &childIDs).Error; err != nil {
			return nil, fmt.Errorf("collect descendant categories: %w", err)
		}
		result = append(result, childIDs...)
		queue = append(queue, childIDs...)
	}

	return result, nil
}

// normalizePagination returns safe page and pageSize values.
func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	return page, pageSize
}

func appendUniqueProductIDs(orderedIDs *[]string, seenIDs map[string]struct{}, ids []string) {
	for _, id := range ids {
		trimmedID := strings.TrimSpace(id)
		if trimmedID == "" {
			continue
		}
		if _, exists := seenIDs[trimmedID]; exists {
			continue
		}
		seenIDs[trimmedID] = struct{}{}
		*orderedIDs = append(*orderedIDs, trimmedID)
	}
}

func appendCategoryProductRows(orderedIDs *[]string, seenIDs map[string]struct{}, rows []categoryProductRecord) {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ProductID)
	}

	appendUniqueProductIDs(orderedIDs, seenIDs, ids)
}
