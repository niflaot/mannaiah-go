package category

import (
	"context"
	"fmt"

	productstore "mannaiah/module/products/adapter/store/product"
	categoryport "mannaiah/module/products/port/category"
)

// ListProducts resolves and returns paginated products for a category.
func (r *Repository) ListProducts(ctx context.Context, q categoryport.ListProductsQuery) (*categoryport.ListProductsResult, error) {
	cat, err := r.GetByID(ctx, q.CategoryID)
	if err != nil {
		return nil, err
	}

	categoryIDs := []string{cat.ID}
	if cat.IncludeChildren {
		descendants, err := r.collectDescendants(ctx, cat.ID)
		if err != nil {
			return nil, err
		}
		categoryIDs = append(categoryIDs, descendants...)
	}

	hasFilters := len(cat.Filter.Tags) > 0 ||
		cat.Filter.PriceRange != nil ||
		len(cat.Filter.CategoryRefs) > 0
	hasPinned := len(cat.ProductIDs) > 0
	if !hasFilters && !hasPinned {
		page, pageSize := normalizePagination(q.Page, q.PageSize)

		return &categoryport.ListProductsResult{
			Items:    nil,
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	idSet := make(map[string]struct{})

	for _, catID := range categoryIDs {
		var rows []categoryProductRecord
		if err := r.db.WithContext(ctx).Where("category_id = ?", catID).Find(&rows).Error; err != nil {
			return nil, fmt.Errorf("load pinned products for category %s: %w", catID, err)
		}
		for _, row := range rows {
			idSet[row.ProductID] = struct{}{}
		}
	}

	if len(cat.Filter.Tags) > 0 {
		var tagProductIDs []string
		err := r.db.WithContext(ctx).
			Table("product_tags").
			Select("product_id").
			Where("tag IN ?", cat.Filter.Tags).
			Group("product_id").
			Pluck("product_id", &tagProductIDs).Error
		if err != nil {
			return nil, fmt.Errorf("filter products by tags: %w", err)
		}

		priceQuery := r.db.WithContext(ctx).Table("products").Select("id").Where("deleted_at IS NULL AND id IN ?", tagProductIDs)
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
			return nil, fmt.Errorf("apply price filter on tag results: %w", err)
		}
		for _, id := range filteredIDs {
			idSet[id] = struct{}{}
		}
	} else if cat.Filter.PriceRange != nil {
		priceQuery := r.db.WithContext(ctx).Table("products").Select("id").Where("deleted_at IS NULL")
		if cat.Filter.PriceRange.Min != nil {
			priceQuery = priceQuery.Where("price >= ?", *cat.Filter.PriceRange.Min)
		}
		if cat.Filter.PriceRange.Max != nil {
			priceQuery = priceQuery.Where("price <= ?", *cat.Filter.PriceRange.Max)
		}

		var priceIDs []string
		if err := priceQuery.Pluck("id", &priceIDs).Error; err != nil {
			return nil, fmt.Errorf("filter products by price: %w", err)
		}
		for _, id := range priceIDs {
			idSet[id] = struct{}{}
		}
	}

	for _, refCatID := range cat.Filter.CategoryRefs {
		var refRows []categoryProductRecord
		if err := r.db.WithContext(ctx).Where("category_id = ?", refCatID).Find(&refRows).Error; err != nil {
			return nil, fmt.Errorf("load ref category products: %w", err)
		}
		for _, row := range refRows {
			idSet[row.ProductID] = struct{}{}
		}
	}

	if len(idSet) == 0 {
		page, pageSize := normalizePagination(q.Page, q.PageSize)

		return &categoryport.ListProductsResult{
			Items:    nil,
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	allIDs := make([]string, 0, len(idSet))
	for id := range idSet {
		allIDs = append(allIDs, id)
	}

	page, pageSize := normalizePagination(q.Page, q.PageSize)
	total := int64(len(allIDs))
	offset := (page - 1) * pageSize
	if offset >= int(total) {
		return &categoryport.ListProductsResult{
			Items:    nil,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	pagedIDs := allIDs
	end := offset + pageSize
	if offset > 0 || end < len(allIDs) {
		if offset >= len(allIDs) {
			pagedIDs = nil
		} else {
			if end > len(allIDs) {
				end = len(allIDs)
			}
			pagedIDs = allIDs[offset:end]
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

// collectDescendants collects all descendant category IDs recursively.
func (r *Repository) collectDescendants(ctx context.Context, parentID string) ([]string, error) {
	var result []string
	queue := []string{parentID}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		var childIDs []string
		if err := r.db.WithContext(ctx).Model(&categoryRecord{}).Select("id").Where("parent_id = ? AND deleted_at IS NULL", current).Pluck("id", &childIDs).Error; err != nil {
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
