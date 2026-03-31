package search

import (
	"context"
	"errors"
	"fmt"

	"mannaiah/module/segment/domain"
	coresearch "mannaiah/module/core/search"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("segment search db must not be nil")
)

// segmentSearchRecord mirrors the segments table for search reads.
type segmentSearchRecord struct {
	ID              string  `gorm:"primaryKey;size:64"`
	Name            string  `gorm:"size:255"`
	Slug            string  `gorm:"size:255"`
	Channel         string  `gorm:"size:64"`
	ParentSegmentID *string `gorm:"size:64"`
	CreatedAt       string
	UpdatedAt       string
}

func (segmentSearchRecord) TableName() string { return "segments" }

// Repository implements search.Repository for segments.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a segments search repository.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

// Descriptor returns the segments search descriptor.
func Descriptor() coresearch.Descriptor {
	return coresearch.Descriptor{
		TextFields: []string{"name", "slug"},
		FilterableFields: map[string][]coresearch.Operator{
			"channel":           {coresearch.OpEQ},
			"parent_segment_id": {coresearch.OpEQ},
			"created_at":        {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
		},
		SortableFields: []string{"name", "created_at"},
		DefaultSort:    coresearch.SortField{Field: "name", Direction: coresearch.Asc},
	}
}

// Search executes a search query against the segments table.
func (r *Repository) Search(ctx context.Context, query coresearch.Query) (*coresearch.Result[domain.Segment], error) {
	desc := Descriptor()
	base, paginated := coresearch.BuildGORMQuery(
		r.db.WithContext(ctx).Model(&segmentSearchRecord{}),
		query,
		desc,
	)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count segments: %w", err)
	}

	var records []segmentSearchRecord
	if err := paginated.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("search segments: %w", err)
	}

	segments := make([]domain.Segment, 0, len(records))
	for _, rec := range records {
		segments = append(segments, toDomain(rec))
	}

	return coresearch.NewResult(segments, total, query.Page, query.PageSize), nil
}

// SpotlightSearch returns scored spotlight hits for segments.
func (r *Repository) SpotlightSearch(ctx context.Context, term string, limit int) ([]coresearch.SpotlightHit, error) {
	q := coresearch.Query{Term: term, Page: 1, PageSize: limit}
	result, err := r.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	scored := coresearch.ScoreResults(result.Data, term,
		[]string{"name"},
		[]string{"slug"},
		segmentFieldExtractor,
	)

	hits := make([]coresearch.SpotlightHit, 0, len(scored))
	for _, s := range scored {
		seg := s.Entity
		hits = append(hits, coresearch.SpotlightHit{
			Type:         "segment",
			ID:           seg.ID,
			Title:        seg.Name,
			Subtitle:     fmt.Sprintf("%s \u2014 %s", seg.Channel, seg.Slug),
			MatchedField: s.MatchedField,
			Score:        s.Score,
		})
	}
	return hits, nil
}

// SpotlightType returns the resource type identifier.
func (r *Repository) SpotlightType() string { return "segment" }

// toDomain maps a search record to a minimal domain segment.
func toDomain(rec segmentSearchRecord) domain.Segment {
	return domain.Segment{
		ID:              rec.ID,
		Name:            rec.Name,
		Slug:            rec.Slug,
		Channel:         rec.Channel,
		ParentSegmentID: rec.ParentSegmentID,
	}
}

// segmentFieldExtractor extracts segment field values for scoring.
func segmentFieldExtractor(s domain.Segment, field string) string {
	switch field {
	case "name":
		return s.Name
	case "slug":
		return s.Slug
	default:
		return ""
	}
}
