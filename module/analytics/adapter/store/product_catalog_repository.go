package store

import (
	"context"
	"fmt"

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
	// Price is the optional product price.
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
	// ProductID is the owning product identifier.
	ProductID string `gorm:"column:product_id"`
	// Realm is the datasheet realm identifier.
	Realm string `gorm:"column:realm"`
	// Name is the realm-specific product display name.
	Name string `gorm:"column:name"`
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

// NewProductCatalogRepository creates GORM-backed product catalog repositories.
func NewProductCatalogRepository(db *gorm.DB) (*ProductCatalogRepository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &ProductCatalogRepository{db: db}, nil
}

// GetProductsByBaseTag returns active products filtered by base tag, optional expanded tags,
// optional category, and optional excluded product IDs.
func (r *ProductCatalogRepository) GetProductsByBaseTag(ctx context.Context, baseTag string, expandedTags []string, categoryID string, excludeIDs []string, limit int) ([]port.ProductCatalogEntry, error) {
	if limit <= 0 {
		limit = 3
	}

	// Step 1: resolve candidate product IDs (excluding pinned + excluded).
	candidateIDs, err := r.resolveProductIDs(ctx, baseTag, expandedTags, categoryID, excludeIDs, limit*5)
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
func (r *ProductCatalogRepository) GetProductsByIDs(ctx context.Context, ids []string) ([]port.ProductCatalogEntry, error) {
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

// loadProductEntries loads tags, datasheets, and gallery for the given product IDs
// and assembles ProductCatalogEntry values preserving the input id order.
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

	// Index base records by ID for fast lookup.
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

	// Load datasheets.
	datasheetRows := make([]productDatasheetRecord, 0)
	if err := r.db.WithContext(ctx).
		Table("product_datasheets").
		Select("product_id, realm, name").
		Where("product_id IN ?", ids).
		Order("product_id ASC, position ASC").
		Scan(&datasheetRows).Error; err != nil {
		return nil, fmt.Errorf("load product datasheets for catalog: %w", err)
	}

	// Load gallery items + included realms.
	galleryRows := make([]productGalleryFlatRecord, 0)
	if err := r.db.WithContext(ctx).
		Table("product_gallery AS g").
		Select("g.product_id, g.id AS gallery_item_id, g.asset_id, g.is_main, COALESCE(r.realm, '') AS realm").
		Joins("LEFT JOIN product_gallery_included_realms r ON r.gallery_item_id = g.id").
		Where("g.product_id IN ?", ids).
		Order("g.product_id ASC, g.position ASC, r.position ASC").
		Scan(&galleryRows).Error; err != nil {
		return nil, fmt.Errorf("load product gallery for catalog: %w", err)
	}

	// Assemble lookup maps.
	tagsByProduct := make(map[string][]string, len(ids))
	for _, row := range tagRows {
		tagsByProduct[row.ProductID] = append(tagsByProduct[row.ProductID], row.TagName)
	}

	datasheetsByProduct := make(map[string][]port.ProductDatasheetEntry, len(ids))
	for _, row := range datasheetRows {
		datasheetsByProduct[row.ProductID] = append(datasheetsByProduct[row.ProductID], port.ProductDatasheetEntry{
			Realm: row.Realm,
			Name:  row.Name,
		})
	}

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
				AssetID: row.AssetID,
				IsMain:  row.IsMain,
			}
			if row.Realm != "" {
				entry.IncludedRealms = []string{row.Realm}
			}
			itemSeen[key] = len(galleryByProduct[row.ProductID])
			galleryByProduct[row.ProductID] = append(galleryByProduct[row.ProductID], entry)
		}
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
			ID:         rec.ID,
			Price:      price,
			Tags:       tagsByProduct[rec.ID],
			Datasheets: datasheetsByProduct[rec.ID],
			Gallery:    galleryByProduct[rec.ID],
		})
	}

	return entries, nil
}

// resolveProductIDs returns product IDs matching the base tag filter, optional expanded tags,
// optional category, and excluding the given product IDs.
func (r *ProductCatalogRepository) resolveProductIDs(ctx context.Context, baseTag string, expandedTags []string, categoryID string, excludeIDs []string, limit int) ([]string, error) {
	// Resolve tag ID for baseTag.
	var baseTagID int64
	if err := r.db.WithContext(ctx).
		Table("tags").
		Select("id").
		Where("name = ? AND deleted_at IS NULL", baseTag).
		Limit(1).
		Scan(&baseTagID).Error; err != nil {
		return nil, fmt.Errorf("resolve base tag id: %w", err)
	}
	if baseTagID == 0 {
		return nil, nil
	}

	// Get IDs of products with baseTag.
	var baseTagProductIDs []string
	if err := r.db.WithContext(ctx).
		Table("product_tags").
		Select("product_id").
		Where("tag_id = ?", baseTagID).
		Pluck("product_id", &baseTagProductIDs).Error; err != nil {
		return nil, fmt.Errorf("resolve products with base tag: %w", err)
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
