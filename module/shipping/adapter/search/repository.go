package search

import (
	"context"
	"errors"
	"fmt"

	"mannaiah/module/shipping/domain"
	coresearch "mannaiah/module/core/search"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("shipping search db must not be nil")
)

// markSearchRecord mirrors the shipping_marks table for search reads.
type markSearchRecord struct {
	ID              string  `gorm:"primaryKey;size:64"`
	OrderID         string  `gorm:"size:64"`
	CarrierID       string  `gorm:"size:64"`
	TrackingNumber  string  `gorm:"size:255"`
	Status          string  `gorm:"size:32"`
	DispatchBatchID *string `gorm:"size:64"`
	ShipmentMode    string  `gorm:"size:32"`
	DeclaredValue   float64
	Observations    string  `gorm:"type:text"`
	CreatedAt       string
	UpdatedAt       string
}

func (markSearchRecord) TableName() string { return "shipping_marks" }

// Repository implements search.Repository for shipping marks.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a shipping marks search repository.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

// Descriptor returns the shipping marks search descriptor.
func Descriptor() coresearch.Descriptor {
	return coresearch.Descriptor{
		TextFields: []string{"tracking_number", "order_id", "observations"},
		FilterableFields: map[string][]coresearch.Operator{
			"carrier_id":        {coresearch.OpEQ},
			"status":            {coresearch.OpEQ, coresearch.OpIn},
			"dispatch_batch_id": {coresearch.OpEQ},
			"shipment_mode":     {coresearch.OpEQ},
			"created_at":        {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
			"declared_value":    {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
		},
		SortableFields: []string{"tracking_number", "created_at", "declared_value", "status"},
		DefaultSort:    coresearch.SortField{Field: "created_at", Direction: coresearch.Desc},
	}
}

// Search executes a search query against the shipping_marks table.
func (r *Repository) Search(ctx context.Context, query coresearch.Query) (*coresearch.Result[domain.ShippingMark], error) {
	desc := Descriptor()
	base, paginated := coresearch.BuildGORMQuery(
		r.db.WithContext(ctx).Model(&markSearchRecord{}),
		query,
		desc,
	)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count shipping marks: %w", err)
	}

	var records []markSearchRecord
	if err := paginated.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("search shipping marks: %w", err)
	}

	marks := make([]domain.ShippingMark, 0, len(records))
	for _, rec := range records {
		marks = append(marks, toDomain(rec))
	}

	return coresearch.NewResult(marks, total, query.Page, query.PageSize), nil
}

// SpotlightSearch returns scored spotlight hits for shipping marks.
func (r *Repository) SpotlightSearch(ctx context.Context, term string, limit int) ([]coresearch.SpotlightHit, error) {
	q := coresearch.Query{Term: term, Page: 1, PageSize: limit}
	result, err := r.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	scored := coresearch.ScoreResults(result.Data, term,
		[]string{"tracking_number"},
		[]string{"order_id"},
		markFieldExtractor,
	)

	hits := make([]coresearch.SpotlightHit, 0, len(scored))
	for _, s := range scored {
		m := s.Entity
		hits = append(hits, coresearch.SpotlightHit{
			Type:         "shipping_mark",
			ID:           m.ID,
			Title:        fmt.Sprintf("Track: %s", m.TrackingNumber),
			Subtitle:     fmt.Sprintf("%s \u2014 Order %s", m.Status, m.OrderID),
			MatchedField: s.MatchedField,
			Score:        s.Score,
		})
	}
	return hits, nil
}

// SpotlightType returns the resource type identifier.
func (r *Repository) SpotlightType() string { return "shipping_mark" }

// toDomain maps a search record to a minimal domain shipping mark.
func toDomain(rec markSearchRecord) domain.ShippingMark {
	return domain.ShippingMark{
		ID:              rec.ID,
		OrderID:         rec.OrderID,
		CarrierID:       rec.CarrierID,
		TrackingNumber:  rec.TrackingNumber,
		Status:          domain.MarkStatus(rec.Status),
		DispatchBatchID: rec.DispatchBatchID,
		ShipmentMode:    domain.ShipmentMode(rec.ShipmentMode),
		DeclaredValue:   rec.DeclaredValue,
		Observations:    rec.Observations,
	}
}

// markFieldExtractor extracts shipping mark field values for scoring.
func markFieldExtractor(m domain.ShippingMark, field string) string {
	switch field {
	case "tracking_number":
		return m.TrackingNumber
	case "order_id":
		return m.OrderID
	case "observations":
		return m.Observations
	default:
		return ""
	}
}
