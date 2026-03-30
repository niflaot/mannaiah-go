package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	domain "mannaiah/module/products/domain/product"
	coresearch "mannaiah/module/core/search"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("products search db must not be nil")
)

// productSearchRecord mirrors the products table for search reads.
type productSearchRecord struct {
	ID        string         `gorm:"primaryKey;size:64"`
	SKU       string         `gorm:"size:255"`
	Price     *float64       `gorm:"type:double"`
	CreatedAt string
	UpdatedAt string
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (productSearchRecord) TableName() string { return "products" }

// Repository implements search.Repository for products.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a products search repository.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

// Descriptor returns the products search descriptor.
func Descriptor() coresearch.Descriptor {
	return coresearch.Descriptor{
		TextFields: []string{"sku"},
		FilterableFields: map[string][]coresearch.Operator{
			"sku":        {coresearch.OpEQ},
			"price":      {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
			"created_at": {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
		},
		SortableFields: []string{"sku", "price", "created_at", "updated_at"},
		DefaultSort:    coresearch.SortField{Field: "created_at", Direction: coresearch.Desc},
	}
}

// Search executes a search query against the products table.
func (r *Repository) Search(ctx context.Context, query coresearch.Query) (*coresearch.Result[domain.Product], error) {
	desc := Descriptor()
	base, paginated := coresearch.BuildGORMQuery(
		r.db.WithContext(ctx).Model(&productSearchRecord{}),
		query,
		desc,
	)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count products: %w", err)
	}

	var records []productSearchRecord
	if err := paginated.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("search products: %w", err)
	}

	products := make([]domain.Product, 0, len(records))
	for _, rec := range records {
		products = append(products, toDomain(rec))
	}

	return coresearch.NewResult(products, total, query.Page, query.PageSize), nil
}

// SpotlightSearch returns scored spotlight hits for products.
func (r *Repository) SpotlightSearch(ctx context.Context, term string, limit int) ([]coresearch.SpotlightHit, error) {
	q := coresearch.Query{Term: term, Page: 1, PageSize: limit}
	result, err := r.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	scored := coresearch.ScoreResults(result.Data, term,
		[]string{"sku"},
		nil,
		productFieldExtractor,
	)

	hits := make([]coresearch.SpotlightHit, 0, len(scored))
	for _, s := range scored {
		p := s.Entity
		subtitle := fmt.Sprintf("SKU: %s", p.SKU)
		if p.Price != nil {
			subtitle += fmt.Sprintf(" \u2014 $%.2f", *p.Price)
		}
		hits = append(hits, coresearch.SpotlightHit{
			Type:         "product",
			ID:           p.ID,
			Title:        p.SKU,
			Subtitle:     subtitle,
			MatchedField: s.MatchedField,
			Score:        s.Score,
		})
	}
	return hits, nil
}

// SpotlightType returns the resource type identifier.
func (r *Repository) SpotlightType() string { return "product" }

// SearchWithTags extends Search with tag filtering via product_tags join.
func (r *Repository) SearchWithTags(ctx context.Context, query coresearch.Query) (*coresearch.Result[domain.Product], error) {
	tags := extractTags(query)
	if len(tags) == 0 {
		return r.Search(ctx, query)
	}

	filteredQuery := coresearch.Query{
		Term:     query.Term,
		Sort:     query.Sort,
		Page:     query.Page,
		PageSize: query.PageSize,
	}
	for _, f := range query.Filters {
		if f.Field != "tags" {
			filteredQuery.Filters = append(filteredQuery.Filters, f)
		}
	}

	desc := Descriptor()
	tx := r.db.WithContext(ctx).Model(&productSearchRecord{}).
		Where("id IN (?)",
			r.db.Table("product_tags").
				Select("product_tags.product_id").
				Joins("JOIN tags ON tags.id = product_tags.tag_id AND tags.deleted_at IS NULL").
				Where("tags.name IN ?", tags),
		)
	base, paginated := coresearch.BuildGORMQuery(tx, filteredQuery, desc)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count products by tags: %w", err)
	}

	var records []productSearchRecord
	if err := paginated.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("search products by tags: %w", err)
	}

	products := make([]domain.Product, 0, len(records))
	for _, rec := range records {
		products = append(products, toDomain(rec))
	}
	_ = strings.TrimSpace
	return coresearch.NewResult(products, total, query.Page, query.PageSize), nil
}

// extractTags extracts tag filter from the query for use in sub-queries.
func extractTags(query coresearch.Query) []string {
	for _, f := range query.Filters {
		if f.Field == "tags" && f.Operator == coresearch.OpIn {
			if tags, ok := f.Value.([]string); ok {
				return tags
			}
		}
	}
	return nil
}

// toDomain maps a search record to a minimal domain product.
func toDomain(rec productSearchRecord) domain.Product {
	return domain.Product{
		ID:    rec.ID,
		SKU:   rec.SKU,
		Price: rec.Price,
	}
}

// productFieldExtractor extracts product field values for scoring.
func productFieldExtractor(p domain.Product, field string) string {
	switch field {
	case "sku":
		return p.SKU
	default:
		return ""
	}
}
