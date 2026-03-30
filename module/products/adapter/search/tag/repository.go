package tag

import (
	"context"
	"errors"
	"fmt"

	"mannaiah/module/products/domain/tag"
	coresearch "mannaiah/module/core/search"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("tag search db must not be nil")
)

// tagSearchRecord mirrors the tags table for search reads.
type tagSearchRecord struct {
	ID        uint           `gorm:"primaryKey;autoIncrement"`
	Name      string         `gorm:"size:255"`
	CreatedAt string
	UpdatedAt string
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (tagSearchRecord) TableName() string { return "tags" }

// Repository implements search.Repository for product tags.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a tags search repository.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

// Descriptor returns the tags search descriptor.
func Descriptor() coresearch.Descriptor {
	return coresearch.Descriptor{
		TextFields: []string{"name"},
		FilterableFields: map[string][]coresearch.Operator{
			"created_at": {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
		},
		SortableFields: []string{"name", "created_at"},
		DefaultSort:    coresearch.SortField{Field: "name", Direction: coresearch.Asc},
	}
}

// Search executes a search query against the tags table.
func (r *Repository) Search(ctx context.Context, query coresearch.Query) (*coresearch.Result[tag.Tag], error) {
	desc := Descriptor()
	base, paginated := coresearch.BuildGORMQuery(
		r.db.WithContext(ctx).Model(&tagSearchRecord{}),
		query,
		desc,
	)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count tags: %w", err)
	}

	var records []tagSearchRecord
	if err := paginated.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("search tags: %w", err)
	}

	tags := make([]tag.Tag, 0, len(records))
	for _, rec := range records {
		tags = append(tags, toDomain(rec))
	}

	return coresearch.NewResult(tags, total, query.Page, query.PageSize), nil
}

// SpotlightSearch returns scored spotlight hits for tags.
func (r *Repository) SpotlightSearch(ctx context.Context, term string, limit int) ([]coresearch.SpotlightHit, error) {
	q := coresearch.Query{Term: term, Page: 1, PageSize: limit}
	result, err := r.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	scored := coresearch.ScoreResults(result.Data, term,
		[]string{"name"},
		nil,
		tagFieldExtractor,
	)

	hits := make([]coresearch.SpotlightHit, 0, len(scored))
	for _, s := range scored {
		t := s.Entity
		hits = append(hits, coresearch.SpotlightHit{
			Type:         "tag",
			ID:           fmt.Sprintf("%d", t.ID),
			Title:        t.Name,
			Subtitle:     "Product Tag",
			MatchedField: s.MatchedField,
			Score:        s.Score,
		})
	}
	return hits, nil
}

// SpotlightType returns the resource type identifier.
func (r *Repository) SpotlightType() string { return "tag" }

// toDomain maps a search record to a minimal domain tag.
func toDomain(rec tagSearchRecord) tag.Tag {
	return tag.Tag{
		ID:   rec.ID,
		Name: rec.Name,
	}
}

// tagFieldExtractor extracts tag field values for scoring.
func tagFieldExtractor(t tag.Tag, field string) string {
	switch field {
	case "name":
		return t.Name
	default:
		return ""
	}
}
