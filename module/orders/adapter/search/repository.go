package search

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"mannaiah/module/orders/domain"
	coresearch "mannaiah/module/core/search"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("orders search db must not be nil")
)

// orderSearchRecord mirrors a simplified order row for search reads.
type orderSearchRecord struct {
	ID            string         `gorm:"primaryKey;size:64"`
	Identifier    string         `gorm:"size:255"`
	Realm         string         `gorm:"size:128"`
	ContactID     string         `gorm:"size:64"`
	PaymentMethod string         `gorm:"size:128"`
	CreatedAt     string
	UpdatedAt     string
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (orderSearchRecord) TableName() string { return "orders" }

// Repository implements search.Repository for orders.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates an orders search repository.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

// Descriptor returns the orders search descriptor.
func Descriptor() coresearch.Descriptor {
	return coresearch.Descriptor{
		TextFields: []string{"identifier", "payment_method"},
		FilterableFields: map[string][]coresearch.Operator{
			"realm":          {coresearch.OpEQ},
			"contact_id":     {coresearch.OpEQ},
			"payment_method": {coresearch.OpEQ},
			"created_at":     {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
			"updated_at":     {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
		},
		SortableFields: []string{"identifier", "created_at", "updated_at"},
		DefaultSort:    coresearch.SortField{Field: "created_at", Direction: coresearch.Desc},
	}
}

// Search executes a search query against the orders table.
func (r *Repository) Search(ctx context.Context, query coresearch.Query) (*coresearch.Result[domain.Order], error) {
	desc := Descriptor()
	base, paginated := coresearch.BuildGORMQuery(
		r.db.WithContext(ctx).Model(&orderSearchRecord{}),
		query,
		desc,
	)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count orders: %w", err)
	}

	var records []orderSearchRecord
	if err := paginated.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("search orders: %w", err)
	}

	orders := make([]domain.Order, 0, len(records))
	for _, rec := range records {
		orders = append(orders, toDomain(rec))
	}

	return coresearch.NewResult(orders, total, query.Page, query.PageSize), nil
}

// SpotlightSearch returns scored spotlight hits for orders.
func (r *Repository) SpotlightSearch(ctx context.Context, term string, limit int) ([]coresearch.SpotlightHit, error) {
	q := coresearch.Query{Term: term, Page: 1, PageSize: limit}
	result, err := r.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	scored := coresearch.ScoreResults(result.Data, term,
		[]string{"identifier"},
		[]string{"payment_method", "realm"},
		orderFieldExtractor,
	)

	hits := make([]coresearch.SpotlightHit, 0, len(scored))
	for _, s := range scored {
		o := s.Entity
		hits = append(hits, coresearch.SpotlightHit{
			Type:         "order",
			ID:           o.ID,
			Title:        fmt.Sprintf("Order #%s", o.Identifier),
			Subtitle:     strings.ToUpper(string(o.CurrentStatus)) + " \u2014 " + o.Realm,
			MatchedField: s.MatchedField,
			Score:        s.Score,
		})
	}
	return hits, nil
}

// SpotlightType returns the resource type identifier.
func (r *Repository) SpotlightType() string { return "order" }

// toDomain maps a search record to a minimal domain order.
func toDomain(rec orderSearchRecord) domain.Order {
	return domain.Order{
		ID:            rec.ID,
		Identifier:    rec.Identifier,
		Realm:         rec.Realm,
		ContactID:     rec.ContactID,
		PaymentMethod: rec.PaymentMethod,
	}
}

// orderFieldExtractor extracts order field values for scoring.
func orderFieldExtractor(o domain.Order, field string) string {
	switch field {
	case "identifier":
		return o.Identifier
	case "payment_method":
		return o.PaymentMethod
	case "realm":
		return o.Realm
	default:
		return ""
	}
}
