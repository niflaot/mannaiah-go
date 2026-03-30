package search

import (
	"context"
	"errors"
	"fmt"

	"mannaiah/module/campaign/domain"
	coresearch "mannaiah/module/core/search"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("campaign search db must not be nil")
)

// campaignSearchRecord mirrors the campaigns table for search reads.
type campaignSearchRecord struct {
	ID        string         `gorm:"primaryKey;size:64"`
	Name      string         `gorm:"size:255"`
	Slug      string         `gorm:"size:255"`
	Channel   string         `gorm:"size:64"`
	SegmentID string         `gorm:"size:64"`
	Subject   string         `gorm:"size:512"`
	Status    string         `gorm:"size:32"`
	SentCount int
	CreatedAt string
	UpdatedAt string
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (campaignSearchRecord) TableName() string { return "campaigns" }

// Repository implements search.Repository for campaigns.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a campaigns search repository.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

// Descriptor returns the campaigns search descriptor.
func Descriptor() coresearch.Descriptor {
	return coresearch.Descriptor{
		TextFields: []string{"name", "slug", "subject"},
		FilterableFields: map[string][]coresearch.Operator{
			"status":     {coresearch.OpEQ, coresearch.OpIn},
			"channel":    {coresearch.OpEQ},
			"segment_id": {coresearch.OpEQ},
			"created_at": {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
		},
		SortableFields: []string{"name", "status", "created_at", "sent_count"},
		DefaultSort:    coresearch.SortField{Field: "created_at", Direction: coresearch.Desc},
	}
}

// Search executes a search query against the campaigns table.
func (r *Repository) Search(ctx context.Context, query coresearch.Query) (*coresearch.Result[domain.Campaign], error) {
	desc := Descriptor()
	base, paginated := coresearch.BuildGORMQuery(
		r.db.WithContext(ctx).Model(&campaignSearchRecord{}),
		query,
		desc,
	)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count campaigns: %w", err)
	}

	var records []campaignSearchRecord
	if err := paginated.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("search campaigns: %w", err)
	}

	campaigns := make([]domain.Campaign, 0, len(records))
	for _, rec := range records {
		campaigns = append(campaigns, toDomain(rec))
	}

	return coresearch.NewResult(campaigns, total, query.Page, query.PageSize), nil
}

// SpotlightSearch returns scored spotlight hits for campaigns.
func (r *Repository) SpotlightSearch(ctx context.Context, term string, limit int) ([]coresearch.SpotlightHit, error) {
	q := coresearch.Query{Term: term, Page: 1, PageSize: limit}
	result, err := r.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	scored := coresearch.ScoreResults(result.Data, term,
		[]string{"name"},
		[]string{"slug", "subject"},
		campaignFieldExtractor,
	)

	hits := make([]coresearch.SpotlightHit, 0, len(scored))
	for _, s := range scored {
		c := s.Entity
		hits = append(hits, coresearch.SpotlightHit{
			Type:         "campaign",
			ID:           c.ID,
			Title:        c.Name,
			Subtitle:     fmt.Sprintf("%s \u2014 %s", c.Status, c.Channel),
			MatchedField: s.MatchedField,
			Score:        s.Score,
		})
	}
	return hits, nil
}

// SpotlightType returns the resource type identifier.
func (r *Repository) SpotlightType() string { return "campaign" }

// toDomain maps a search record to a minimal domain campaign.
func toDomain(rec campaignSearchRecord) domain.Campaign {
	return domain.Campaign{
		ID:        rec.ID,
		Name:      rec.Name,
		Slug:      rec.Slug,
		Channel:   rec.Channel,
		SegmentID: rec.SegmentID,
		Subject:   rec.Subject,
		Status:    domain.Status(rec.Status),
		SentCount: rec.SentCount,
	}
}

// campaignFieldExtractor extracts campaign field values for scoring.
func campaignFieldExtractor(c domain.Campaign, field string) string {
	switch field {
	case "name":
		return c.Name
	case "slug":
		return c.Slug
	case "subject":
		return c.Subject
	default:
		return ""
	}
}
