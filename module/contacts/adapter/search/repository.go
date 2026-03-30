package search

import (
	"context"
	"errors"
	"strings"

	"mannaiah/module/contacts/domain"
	coresearch "mannaiah/module/core/search"

	"gorm.io/gorm"
)

var (
	// ErrNilDB is returned when a nil DB dependency is provided.
	ErrNilDB = errors.New("contacts search db must not be nil")
)

// contactRecord mirrors the contacts table for search-only reads.
type contactRecord struct {
	ID             string         `gorm:"primaryKey;size:64"`
	DocumentType   string         `gorm:"size:16"`
	DocumentNumber string         `gorm:"size:128"`
	LegalName      string         `gorm:"size:255"`
	FirstName      string         `gorm:"size:255"`
	LastName       string         `gorm:"size:255"`
	Email          string         `gorm:"size:255"`
	Phone          string         `gorm:"size:64"`
	Address        string         `gorm:"size:512"`
	AddressExtra   string         `gorm:"size:512"`
	CityCode       string         `gorm:"size:64"`
	CreatedAt      string
	UpdatedAt      string
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (contactRecord) TableName() string { return "contacts" }

// Repository implements search.Repository for contacts.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a contacts search repository.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}
	return &Repository{db: db}, nil
}

// Descriptor returns the contacts search descriptor.
func Descriptor() coresearch.Descriptor {
	return coresearch.Descriptor{
		TextFields: []string{"first_name", "last_name", "legal_name", "email", "document_number", "phone"},
		FilterableFields: map[string][]coresearch.Operator{
			"document_type": {coresearch.OpEQ},
			"city_code":     {coresearch.OpEQ},
			"created_at":    {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
			"updated_at":    {coresearch.OpGTE, coresearch.OpLTE, coresearch.OpGT, coresearch.OpLT, coresearch.OpBetween},
		},
		SortableFields: []string{"first_name", "last_name", "email", "document_number", "created_at", "updated_at"},
		DefaultSort:    coresearch.SortField{Field: "created_at", Direction: coresearch.Desc},
	}
}

// Search executes a search query against the contacts table.
func (r *Repository) Search(ctx context.Context, query coresearch.Query) (*coresearch.Result[domain.Contact], error) {
	desc := Descriptor()
	base, paginated := coresearch.BuildGORMQuery(
		r.db.WithContext(ctx).Model(&contactRecord{}),
		query,
		desc,
	)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []contactRecord
	if err := paginated.Find(&records).Error; err != nil {
		return nil, err
	}

	contacts := make([]domain.Contact, 0, len(records))
	for _, rec := range records {
		contacts = append(contacts, toDomain(rec))
	}

	return coresearch.NewResult(contacts, total, query.Page, query.PageSize), nil
}

// SpotlightSearch returns scored spotlight hits for the given term.
func (r *Repository) SpotlightSearch(ctx context.Context, term string, limit int) ([]coresearch.SpotlightHit, error) {
	q := coresearch.Query{Term: term, Page: 1, PageSize: limit}
	result, err := r.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	scored := coresearch.ScoreResults(result.Data, term,
		[]string{"first_name", "last_name", "email"},
		[]string{"document_number", "phone"},
		contactFieldExtractor,
	)

	hits := make([]coresearch.SpotlightHit, 0, len(scored))
	for _, s := range scored {
		c := s.Entity
		title := strings.TrimSpace(c.FirstName + " " + c.LastName)
		if title == "" {
			title = c.LegalName
		}
		hits = append(hits, coresearch.SpotlightHit{
			Type:         "contact",
			ID:           c.ID,
			Title:        title,
			Subtitle:     c.Email,
			MatchedField: s.MatchedField,
			Score:        s.Score,
		})
	}
	return hits, nil
}

// SpotlightType returns the resource type identifier.
func (r *Repository) SpotlightType() string { return "contact" }

// toDomain maps a search record to a domain contact.
func toDomain(rec contactRecord) domain.Contact {
	return domain.Contact{
		ID:             rec.ID,
		DocumentType:   domain.DocumentType(rec.DocumentType),
		DocumentNumber: rec.DocumentNumber,
		LegalName:      rec.LegalName,
		FirstName:      rec.FirstName,
		LastName:       rec.LastName,
		Email:          rec.Email,
		Phone:          rec.Phone,
		Address:        rec.Address,
		AddressExtra:   rec.AddressExtra,
		CityCode:       rec.CityCode,
	}
}

// contactFieldExtractor returns the string value of a contact field by name.
func contactFieldExtractor(c domain.Contact, field string) string {
	switch field {
	case "first_name":
		return c.FirstName
	case "last_name":
		return c.LastName
	case "email":
		return c.Email
	case "legal_name":
		return c.LegalName
	case "document_number":
		return c.DocumentNumber
	case "phone":
		return c.Phone
	default:
		return ""
	}
}
