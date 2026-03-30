package category

import (
	"context"
	"errors"
	"fmt"

	domain "mannaiah/module/products/domain/category"
	coresearch "mannaiah/module/core/search"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("categories search db must not be nil")
)

// categorySearchRecord mirrors the categories table for search reads.
type categorySearchRecord struct {
	ID          string         `gorm:"primaryKey;size:64"`
	Slug        string         `gorm:"size:255"`
	Name        string         `gorm:"size:255"`
	Description string         `gorm:"type:text"`
	ParentID    *string        `gorm:"size:64"`
	CreatedAt   string
	UpdatedAt   string
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (categorySearchRecord) TableName() string { return "categories" }

// Repository implements search.Repository for categories.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a categories search repository.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

// Descriptor returns the categories search descriptor.
func Descriptor() coresearch.Descriptor {
	return coresearch.Descriptor{
		TextFields: []string{"name", "slug", "description"},
		FilterableFields: map[string][]coresearch.Operator{
			"parent_id":  {coresearch.OpEQ},
			"slug":       {coresearch.OpEQ},
			"created_at": {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
		},
		SortableFields: []string{"name", "slug", "created_at"},
		DefaultSort:    coresearch.SortField{Field: "name", Direction: coresearch.Asc},
	}
}

// Search executes a search query against the categories table.
func (r *Repository) Search(ctx context.Context, query coresearch.Query) (*coresearch.Result[domain.Category], error) {
	desc := Descriptor()
	base, paginated := coresearch.BuildGORMQuery(
		r.db.WithContext(ctx).Model(&categorySearchRecord{}),
		query,
		desc,
	)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count categories: %w", err)
	}

	var records []categorySearchRecord
	if err := paginated.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("search categories: %w", err)
	}

	categories := make([]domain.Category, 0, len(records))
	for _, rec := range records {
		categories = append(categories, toDomain(rec))
	}

	return coresearch.NewResult(categories, total, query.Page, query.PageSize), nil
}

// SpotlightSearch returns scored spotlight hits for categories.
func (r *Repository) SpotlightSearch(ctx context.Context, term string, limit int) ([]coresearch.SpotlightHit, error) {
	q := coresearch.Query{Term: term, Page: 1, PageSize: limit}
	result, err := r.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	scored := coresearch.ScoreResults(result.Data, term,
		[]string{"name"},
		[]string{"slug", "description"},
		categoryFieldExtractor,
	)

	hits := make([]coresearch.SpotlightHit, 0, len(scored))
	for _, s := range scored {
		c := s.Entity
		hits = append(hits, coresearch.SpotlightHit{
			Type:         "category",
			ID:           c.ID,
			Title:        c.Name,
			Subtitle:     c.Slug,
			MatchedField: s.MatchedField,
			Score:        s.Score,
		})
	}
	return hits, nil
}

// SpotlightType returns the resource type identifier.
func (r *Repository) SpotlightType() string { return "category" }

// toDomain maps a search record to a minimal domain category.
func toDomain(rec categorySearchRecord) domain.Category {
	return domain.Category{
		ID:          rec.ID,
		Slug:        rec.Slug,
		Name:        rec.Name,
		Description: rec.Description,
		ParentID:    rec.ParentID,
	}
}

// categoryFieldExtractor extracts category field values for scoring.
func categoryFieldExtractor(c domain.Category, field string) string {
	switch field {
	case "name":
		return c.Name
	case "slug":
		return c.Slug
	case "description":
		return c.Description
	default:
		return ""
	}
}
