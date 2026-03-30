package variation

import (
	"context"
	"errors"
	"fmt"

	"mannaiah/module/products/domain/variation"
	coresearch "mannaiah/module/core/search"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("variation search db must not be nil")
)

// variationSearchRecord mirrors the variations table for search reads.
type variationSearchRecord struct {
	ID         string         `gorm:"primaryKey;size:64"`
	Name       string         `gorm:"size:255"`
	Definition string         `gorm:"size:32"`
	Value      string         `gorm:"size:255"`
	CreatedAt  string
	UpdatedAt  string
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (variationSearchRecord) TableName() string { return "variations" }

// Repository implements search.Repository for product variations.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a variations search repository.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

// Descriptor returns the variations search descriptor.
func Descriptor() coresearch.Descriptor {
	return coresearch.Descriptor{
		TextFields: []string{"name", "value"},
		FilterableFields: map[string][]coresearch.Operator{
			"definition": {coresearch.OpEQ, coresearch.OpIn},
			"created_at": {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
		},
		SortableFields: []string{"name", "definition", "created_at"},
		DefaultSort:    coresearch.SortField{Field: "name", Direction: coresearch.Asc},
	}
}

// Search executes a search query against the variations table.
func (r *Repository) Search(ctx context.Context, query coresearch.Query) (*coresearch.Result[variation.Variation], error) {
	desc := Descriptor()
	base, paginated := coresearch.BuildGORMQuery(
		r.db.WithContext(ctx).Model(&variationSearchRecord{}),
		query,
		desc,
	)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count variations: %w", err)
	}

	var records []variationSearchRecord
	if err := paginated.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("search variations: %w", err)
	}

	variations := make([]variation.Variation, 0, len(records))
	for _, rec := range records {
		variations = append(variations, toDomain(rec))
	}

	return coresearch.NewResult(variations, total, query.Page, query.PageSize), nil
}

// SpotlightSearch returns scored spotlight hits for variations.
func (r *Repository) SpotlightSearch(ctx context.Context, term string, limit int) ([]coresearch.SpotlightHit, error) {
	q := coresearch.Query{Term: term, Page: 1, PageSize: limit}
	result, err := r.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	scored := coresearch.ScoreResults(result.Data, term,
		[]string{"name"},
		[]string{"value"},
		variationFieldExtractor,
	)

	hits := make([]coresearch.SpotlightHit, 0, len(scored))
	for _, s := range scored {
		v := s.Entity
		hits = append(hits, coresearch.SpotlightHit{
			Type:         "variation",
			ID:           v.ID,
			Title:        v.Name,
			Subtitle:     fmt.Sprintf("%s: %s", v.Definition, v.Value),
			MatchedField: s.MatchedField,
			Score:        s.Score,
		})
	}
	return hits, nil
}

// SpotlightType returns the resource type identifier.
func (r *Repository) SpotlightType() string { return "variation" }

// toDomain maps a search record to a minimal domain variation.
func toDomain(rec variationSearchRecord) variation.Variation {
	return variation.Variation{
		ID:         rec.ID,
		Name:       rec.Name,
		Definition: variation.Definition(rec.Definition),
		Value:      rec.Value,
	}
}

// variationFieldExtractor extracts variation field values for scoring.
func variationFieldExtractor(v variation.Variation, field string) string {
	switch field {
	case "name":
		return v.Name
	case "value":
		return v.Value
	default:
		return ""
	}
}
