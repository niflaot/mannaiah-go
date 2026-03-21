package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"mannaiah/module/analytics/port"
)

// ProductCatalogRepository implements GORM-backed product catalog read behavior.
type ProductCatalogRepository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

// productBaseRecord defines minimal product root fields needed for recommendation queries.
type productBaseRecord struct {
	// ID is the product identifier.
	ID string `gorm:"column:id"`
	// Price is the optional product base price.
	Price *float64 `gorm:"column:price"`
}

// productTagNameRecord holds a product_id → tag name join result.
type productTagNameRecord struct {
	// ProductID is the owning product identifier.
	ProductID string `gorm:"column:product_id"`
	// TagName is the resolved tag name.
	TagName string `gorm:"column:tag_name"`
}

// productDatasheetRecord holds a product_id → realm/name join result.
type productDatasheetRecord struct {
	// ID is the datasheet row identifier, used to load price attributes.
	ID uint `gorm:"column:id"`
	// ProductID is the owning product identifier.
	ProductID string `gorm:"column:product_id"`
	// Realm is the datasheet realm identifier.
	Realm string `gorm:"column:realm"`
	// Name is the realm-specific product display name.
	Name string `gorm:"column:name"`
}

// productDatasheetPriceRecord holds a datasheet_id → price value_json join result.
type productDatasheetPriceRecord struct {
	// DatasheetID is the owning datasheet row identifier.
	DatasheetID uint `gorm:"column:datasheet_id"`
	// ValueJSON is the raw JSON-encoded attribute value for key='price'.
	ValueJSON string `gorm:"column:value_json"`
}

// productGalleryFlatRecord holds a gallery item row with included realms joined.
type productGalleryFlatRecord struct {
	// ProductID is the owning product identifier.
	ProductID string `gorm:"column:product_id"`
	// GalleryItemID is the gallery item surrogate identifier.
	GalleryItemID uint `gorm:"column:gallery_item_id"`
	// AssetID is the referenced asset identifier.
	AssetID string `gorm:"column:asset_id"`
	// IsMain reports whether this is the primary product image.
	IsMain bool `gorm:"column:is_main"`
	// Realm is an included realm for this gallery item (empty when all realms).
	Realm string `gorm:"column:realm"`
}

// productGalleryVariationRow holds a gallery_item_id → variation_id join result.
type productGalleryVariationRow struct {
	// GalleryItemID is the owning gallery item identifier.
	GalleryItemID uint `gorm:"column:gallery_item_id"`
	// VariationID is the linked variation identifier.
	VariationID string `gorm:"column:variation_id"`
}

// productVariationLinkRow holds a product_id → variation_id join result.
type productVariationLinkRow struct {
	// ProductID is the owning product identifier.
	ProductID string `gorm:"column:product_id"`
	// VariationID is the linked variation identifier.
	VariationID string `gorm:"column:variation_id"`
}

// NewProductCatalogRepository creates GORM-backed product catalog repositories.
func NewProductCatalogRepository(db *gorm.DB) (*ProductCatalogRepository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &ProductCatalogRepository{db: db}, nil
}

// GetProductsByBaseTags returns active products filtered by one or more base tags.
// baseTagMode "any" = union (product has at least one tag); "all" = intersection (product has every tag).
func (r *ProductCatalogRepository) GetProductsByBaseTags(ctx context.Context, baseTags []string, baseTagMode string, expandedTags []string, categoryID string, excludeIDs []string, filterVariationIDs []string, limit int) ([]port.ProductCatalogEntry, error) {
	if limit <= 0 {
		limit = 3
	}

	candidateIDs, err := r.resolveProductIDs(ctx, baseTags, baseTagMode, expandedTags, categoryID, excludeIDs, filterVariationIDs, limit*5)
	if err != nil {
		return nil, err
	}
	if len(candidateIDs) == 0 {
		return nil, nil
	}

	entries, err := r.loadProductEntries(ctx, candidateIDs)
	if err != nil {
		return nil, err
	}

	if len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}

// GetProductsByIDs returns active products for the given product IDs, preserving input order.
func (r *ProductCatalogRepository) GetProductsByIDs(ctx context.Context, ids []string, filterVariationIDs []string) ([]port.ProductCatalogEntry, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Confirm existence and soft-delete status.
	var activeIDs []string
	if err := r.db.WithContext(ctx).
		Table("products").
		Select("id").
		Where("id IN ? AND deleted_at IS NULL", ids).
		Pluck("id", &activeIDs).Error; err != nil {
		return nil, fmt.Errorf("filter active pinned products: %w", err)
	}
	if len(activeIDs) == 0 {
		return nil, nil
	}

	// Filter by variation if provided.
	if len(filterVariationIDs) > 0 {
		var varMatching []string
		if err := r.db.WithContext(ctx).
			Table("product_variation_links").
			Select("DISTINCT product_id").
			Where("product_id IN ? AND variation_id IN ?", activeIDs, filterVariationIDs).
			Pluck("product_id", &varMatching).Error; err != nil {
			return nil, fmt.Errorf("filter pinned products by variation: %w", err)
		}
		varSet := make(map[string]struct{}, len(varMatching))
		for _, id := range varMatching {
			varSet[id] = struct{}{}
		}
		filtered := make([]string, 0, len(varMatching))
		for _, id := range activeIDs {
			if _, ok := varSet[id]; ok {
				filtered = append(filtered, id)
			}
		}
		activeIDs = filtered
	}

	if len(activeIDs) == 0 {
		return nil, nil
	}

	// Build a set of confirmed IDs for fast lookup, then restore input order.
	activeSet := make(map[string]struct{}, len(activeIDs))
	for _, id := range activeIDs {
		activeSet[id] = struct{}{}
	}
	ordered := make([]string, 0, len(activeIDs))
	for _, id := range ids {
		if _, ok := activeSet[id]; ok {
			ordered = append(ordered, id)
		}
	}

	return r.loadProductEntries(ctx, ordered)
}

// loadProductEntries loads tags, datasheets (with realm price), gallery (with realm + variation links),
// and product variation links for the given product IDs, assembling ProductCatalogEntry values
// preserving the input id order.
func (r *ProductCatalogRepository) loadProductEntries(ctx context.Context, ids []string) ([]port.ProductCatalogEntry, error) {
	// Load base records.
	baseRecords := make([]productBaseRecord, 0, len(ids))
	if err := r.db.WithContext(ctx).
		Table("products").
		Select("id, price").
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&baseRecords).Error; err != nil {
		return nil, fmt.Errorf("load product base records: %w", err)
	}
	if len(baseRecords) == 0 {
		return nil, nil
	}

	baseByID := make(map[string]productBaseRecord, len(baseRecords))
	for _, rec := range baseRecords {
		baseByID[rec.ID] = rec
	}

	// Load tags.
	tagRows := make([]productTagNameRecord, 0)
	if err := r.db.WithContext(ctx).
		Table("product_tags").
		Select("product_tags.product_id, tags.name AS tag_name").
		Joins("JOIN tags ON tags.id = product_tags.tag_id AND tags.deleted_at IS NULL").
		Where("product_tags.product_id IN ?", ids).
		Scan(&tagRows).Error; err != nil {
		return nil, fmt.Errorf("load product tags for catalog: %w", err)
	}

	// Load datasheets (include id for price attribute lookup).
	datasheetRows := make([]productDatasheetRecord, 0)
	if err := r.db.WithContext(ctx).
		Table("product_datasheets").
		Select("id, product_id, realm, name").
		Where("product_id IN ?", ids).
		Order("product_id ASC, position ASC").
		Scan(&datasheetRows).Error; err != nil {
		return nil, fmt.Errorf("load product datasheets for catalog: %w", err)
	}

	// Load datasheet price attributes (key='price') for all loaded datasheets.
	datasheetPrices := make(map[uint]*float64)
	if len(datasheetRows) > 0 {
		datasheetIDs := make([]uint, 0, len(datasheetRows))
		for _, ds := range datasheetRows {
			datasheetIDs = append(datasheetIDs, ds.ID)
		}
		priceRows := make([]productDatasheetPriceRecord, 0)
		if err := r.db.WithContext(ctx).
			Table("product_datasheet_attributes").
			Select("datasheet_id, value_json").
			Where("datasheet_id IN ? AND `key` = 'price'", datasheetIDs).
			Scan(&priceRows).Error; err != nil {
			return nil, fmt.Errorf("load product datasheet prices for catalog: %w", err)
		}
		for _, pr := range priceRows {
			if p := parsePrice(pr.ValueJSON); p != nil {
				datasheetPrices[pr.DatasheetID] = p
			}
		}
	}

	// Load gallery items + included realms.
	galleryRows := make([]productGalleryFlatRecord, 0)
	if err := r.db.WithContext(ctx).
		Table("product_gallery_items AS g").
		Select("g.product_id, g.id AS gallery_item_id, g.asset_id, g.is_main, COALESCE(r.realm, '') AS realm").
		Joins("LEFT JOIN product_gallery_included_realms r ON r.gallery_item_id = g.id").
		Where("g.product_id IN ?", ids).
		Order("g.product_id ASC, g.position ASC, r.position ASC").
		Scan(&galleryRows).Error; err != nil {
		return nil, fmt.Errorf("load product gallery for catalog: %w", err)
	}

	// Collect distinct gallery item IDs for variation lookup.
	galleryItemIDs := make([]uint, 0)
	seenGalleryItems := make(map[uint]struct{})
	for _, row := range galleryRows {
		if _, ok := seenGalleryItems[row.GalleryItemID]; !ok {
			seenGalleryItems[row.GalleryItemID] = struct{}{}
			galleryItemIDs = append(galleryItemIDs, row.GalleryItemID)
		}
	}

	// Load gallery variation links.
	galleryVariationsByItem := make(map[uint][]string)
	if len(galleryItemIDs) > 0 {
		galleryVarRows := make([]productGalleryVariationRow, 0)
		if err := r.db.WithContext(ctx).
			Table("product_gallery_variations").
			Select("gallery_item_id, variation_id").
			Where("gallery_item_id IN ?", galleryItemIDs).
			Scan(&galleryVarRows).Error; err != nil {
			return nil, fmt.Errorf("load product gallery variations for catalog: %w", err)
		}
		for _, row := range galleryVarRows {
			galleryVariationsByItem[row.GalleryItemID] = append(galleryVariationsByItem[row.GalleryItemID], row.VariationID)
		}
	}

	// Load product-level variation links.
	variationLinkRows := make([]productVariationLinkRow, 0)
	if err := r.db.WithContext(ctx).
		Table("product_variation_links").
		Select("product_id, variation_id").
		Where("product_id IN ?", ids).
		Scan(&variationLinkRows).Error; err != nil {
		return nil, fmt.Errorf("load product variation links for catalog: %w", err)
	}

	// Assemble per-product tag map.
	tagsByProduct := make(map[string][]string, len(ids))
	for _, row := range tagRows {
		tagsByProduct[row.ProductID] = append(tagsByProduct[row.ProductID], row.TagName)
	}

	// Assemble per-product datasheet map (with parsed price).
	datasheetsByProduct := make(map[string][]port.ProductDatasheetEntry, len(ids))
	for _, row := range datasheetRows {
		price := datasheetPrices[row.ID]
		datasheetsByProduct[row.ProductID] = append(datasheetsByProduct[row.ProductID], port.ProductDatasheetEntry{
			Realm: row.Realm,
			Name:  row.Name,
			Price: price,
		})
	}

	// Assemble per-product gallery map, deduplicating gallery items and merging included realms.
	type galleryKey struct {
		productID     string
		galleryItemID uint
	}
	galleryByProduct := make(map[string][]port.ProductGalleryEntry)
	itemSeen := make(map[galleryKey]int)
	for _, row := range galleryRows {
		key := galleryKey{productID: row.ProductID, galleryItemID: row.GalleryItemID}
		if idx, exists := itemSeen[key]; exists {
			if row.Realm != "" {
				galleryByProduct[row.ProductID][idx].IncludedRealms = append(
					galleryByProduct[row.ProductID][idx].IncludedRealms, row.Realm,
				)
			}
		} else {
			entry := port.ProductGalleryEntry{
				AssetID:      row.AssetID,
				IsMain:       row.IsMain,
				VariationIDs: galleryVariationsByItem[row.GalleryItemID],
			}
			if row.Realm != "" {
				entry.IncludedRealms = []string{row.Realm}
			}
			itemSeen[key] = len(galleryByProduct[row.ProductID])
			galleryByProduct[row.ProductID] = append(galleryByProduct[row.ProductID], entry)
		}
	}

	// Assemble per-product variation map.
	variationsByProduct := make(map[string][]string)
	for _, row := range variationLinkRows {
		variationsByProduct[row.ProductID] = append(variationsByProduct[row.ProductID], row.VariationID)
	}

	// Build result preserving input order.
	entries := make([]port.ProductCatalogEntry, 0, len(ids))
	for _, id := range ids {
		rec, ok := baseByID[id]
		if !ok {
			continue
		}
		price := 0.0
		if rec.Price != nil {
			price = *rec.Price
		}
		entries = append(entries, port.ProductCatalogEntry{
			ID:           rec.ID,
			Price:        price,
			Tags:         tagsByProduct[rec.ID],
			VariationIDs: variationsByProduct[rec.ID],
			Datasheets:   datasheetsByProduct[rec.ID],
			Gallery:      galleryByProduct[rec.ID],
		})
	}

	return entries, nil
}

// resolveProductIDs returns product IDs matching the base tag filter, optional expanded tags,
// optional category, excluded IDs, and optional variation filter.
// baseTagMode "any" = union (product has at least one tag); "all" = intersection (product has every tag).
func (r *ProductCatalogRepository) resolveProductIDs(ctx context.Context, baseTags []string, baseTagMode string, expandedTags []string, categoryID string, excludeIDs []string, filterVariationIDs []string, limit int) ([]string, error) {
	// Resolve tag IDs for all requested base tags.
	var baseTagIDs []int64
	if err := r.db.WithContext(ctx).
		Table("tags").
		Select("id").
		Where("name IN ? AND deleted_at IS NULL", baseTags).
		Pluck("id", &baseTagIDs).Error; err != nil {
		return nil, fmt.Errorf("resolve base tag ids: %w", err)
	}
	if len(baseTagIDs) == 0 {
		return nil, nil
	}

	// Get candidate product IDs based on tag mode.
	var baseTagProductIDs []string
	if baseTagMode == "all" {
		// Intersection: products must carry every requested tag.
		// If fewer tag IDs were found than tags requested, some tags don't exist —
		// no product can satisfy the full intersection.
		if len(baseTagIDs) < len(baseTags) {
			return nil, nil
		}
		if err := r.db.WithContext(ctx).
			Table("product_tags").
			Select("product_id").
			Where("tag_id IN ?", baseTagIDs).
			Group("product_id").
			Having("COUNT(DISTINCT tag_id) = ?", len(baseTagIDs)).
			Pluck("product_id", &baseTagProductIDs).Error; err != nil {
			return nil, fmt.Errorf("resolve products with all base tags: %w", err)
		}
	} else {
		// Union: products with at least one of the requested tags.
		if err := r.db.WithContext(ctx).
			Table("product_tags").
			Select("DISTINCT product_id").
			Where("tag_id IN ?", baseTagIDs).
			Pluck("product_id", &baseTagProductIDs).Error; err != nil {
			return nil, fmt.Errorf("resolve products with any base tag: %w", err)
		}
	}

	if len(baseTagProductIDs) == 0 {
		return nil, nil
	}

	candidateIDs := baseTagProductIDs

	// Filter by expanded tags if provided.
	if len(expandedTags) > 0 {
		var expandedTagIDs []int64
		if err := r.db.WithContext(ctx).
			Table("tags").
			Select("id").
			Where("name IN ? AND deleted_at IS NULL", expandedTags).
			Pluck("id", &expandedTagIDs).Error; err != nil {
			return nil, fmt.Errorf("resolve expanded tag ids: %w", err)
		}
		if len(expandedTagIDs) > 0 {
			var affinityProductIDs []string
			if err := r.db.WithContext(ctx).
				Table("product_tags").
				Select("DISTINCT product_id").
				Where("tag_id IN ? AND product_id IN ?", expandedTagIDs, candidateIDs).
				Pluck("product_id", &affinityProductIDs).Error; err != nil {
				return nil, fmt.Errorf("filter products by expanded tags: %w", err)
			}
			candidateIDs = affinityProductIDs
		}
	}

	if len(candidateIDs) == 0 {
		return nil, nil
	}

	// Filter by category if provided.
	if categoryID != "" {
		var catProductIDs []string
		if err := r.db.WithContext(ctx).
			Table("category_products").
			Select("product_id").
			Where("category_id = ? AND product_id IN ?", categoryID, candidateIDs).
			Pluck("product_id", &catProductIDs).Error; err != nil {
			return nil, fmt.Errorf("filter products by category: %w", err)
		}
		candidateIDs = catProductIDs
	}

	if len(candidateIDs) == 0 {
		return nil, nil
	}

	// Filter by variation if provided.
	if len(filterVariationIDs) > 0 {
		var varFilteredIDs []string
		if err := r.db.WithContext(ctx).
			Table("product_variation_links").
			Select("DISTINCT product_id").
			Where("product_id IN ? AND variation_id IN ?", candidateIDs, filterVariationIDs).
			Pluck("product_id", &varFilteredIDs).Error; err != nil {
			return nil, fmt.Errorf("filter products by variation: %w", err)
		}
		candidateIDs = varFilteredIDs
	}

	if len(candidateIDs) == 0 {
		return nil, nil
	}

	// Apply soft-delete filter, exclusion list, and limit.
	q := r.db.WithContext(ctx).
		Table("products").
		Select("id").
		Where("id IN ? AND deleted_at IS NULL", candidateIDs)
	if len(excludeIDs) > 0 {
		q = q.Where("id NOT IN ?", excludeIDs)
	}

	var result []string
	if err := q.Limit(limit).Pluck("id", &result).Error; err != nil {
		return nil, fmt.Errorf("filter active products: %w", err)
	}

	return result, nil
}

// parsePrice attempts to parse a JSON-encoded attribute value as float64.
// Supports both JSON number (42.90) and JSON string ("42.90") representations.
func parsePrice(valueJSON string) *float64 {
	valueJSON = strings.TrimSpace(valueJSON)
	if valueJSON == "" {
		return nil
	}

	var f float64
	if err := json.Unmarshal([]byte(valueJSON), &f); err == nil {
		return &f
	}

	var s string
	if err := json.Unmarshal([]byte(valueJSON), &s); err == nil {
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return &parsed
		}
	}

	return nil
}
