package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"mannaiah/module/contacts/domain"
	"mannaiah/module/contacts/port"
)

var (
	// ErrNilRepository is returned when a nil repository dependency is provided.
	ErrNilRepository = errors.New("contacts repository must not be nil")
	// ErrInvalidID is returned when an empty id value is provided.
	ErrInvalidID = errors.New("contact id is required")
)

// CreateCommand defines command-side contact creation payload.
type CreateCommand struct {
	// DocumentType defines the document category.
	DocumentType domain.DocumentType
	// DocumentNumber defines the document number.
	DocumentNumber string
	// LegalName defines legal names.
	LegalName string
	// FirstName defines personal first names.
	FirstName string
	// LastName defines personal last names.
	LastName string
	// Email defines the contact email.
	Email string
	// Phone defines the contact phone.
	Phone string
	// Address defines physical address.
	Address string
	// AddressExtra defines optional address details.
	AddressExtra string
	// CityCode defines city code values.
	CityCode string
}

// UpdateCommand defines command-side contact update payload.
type UpdateCommand struct {
	// DocumentType defines optional document category updates.
	DocumentType *domain.DocumentType
	// DocumentNumber defines optional document number updates.
	DocumentNumber *string
	// LegalName defines optional legal name updates.
	LegalName *string
	// FirstName defines optional first name updates.
	FirstName *string
	// LastName defines optional last name updates.
	LastName *string
	// Email defines optional email updates.
	Email *string
	// Phone defines optional phone updates.
	Phone *string
	// Address defines optional address updates.
	Address *string
	// AddressExtra defines optional address extra updates.
	AddressExtra *string
	// CityCode defines optional city code updates.
	CityCode *string
}

// ListResult defines paginated query output for contact listings.
type ListResult struct {
	// Data defines contact rows for current page.
	Data []domain.Contact
	// Page defines current page number.
	Page int
	// Limit defines current page size.
	Limit int
	// Total defines total records after filtering/exclusions.
	Total int64
	// TotalPages defines total pages for current page size.
	TotalPages int
}

// Service defines application use cases for contact management.
type Service interface {
	// Create handles contact creation.
	Create(ctx context.Context, command CreateCommand) (*domain.Contact, error)
	// Get handles contact retrieval by id.
	Get(ctx context.Context, id string) (*domain.Contact, error)
	// List handles paginated contact querying.
	List(ctx context.Context, query port.ListQuery) (*ListResult, error)
	// Update handles contact updates.
	Update(ctx context.Context, id string, command UpdateCommand) (*domain.Contact, error)
	// Delete handles contact deletion.
	Delete(ctx context.Context, id string) error
}

// ContactService implements contact application use cases.
type ContactService struct {
	// repository defines persistence dependency.
	repository port.Repository
	// publisher defines integration event transport dependency.
	publisher port.IntegrationEventPublisher
}

var (
	// _ ensures ContactService satisfies Service contracts.
	_ Service = (*ContactService)(nil)
)

// NewService creates a new contact use-case service.
func NewService(repository port.Repository) (*ContactService, error) {
	return NewServiceWithPublisher(repository, nil)
}

// NewServiceWithPublisher creates a new contact use-case service with integration event publishing support.
func NewServiceWithPublisher(repository port.Repository, publisher port.IntegrationEventPublisher) (*ContactService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}

	return &ContactService{
		repository: repository,
		publisher:  resolvePublisher(publisher),
	}, nil
}

// Create handles contact creation.
func (s *ContactService) Create(ctx context.Context, command CreateCommand) (*domain.Contact, error) {
	entity := &domain.Contact{
		DocumentType:   command.DocumentType,
		DocumentNumber: strings.TrimSpace(command.DocumentNumber),
		LegalName:      strings.TrimSpace(command.LegalName),
		FirstName:      strings.TrimSpace(command.FirstName),
		LastName:       strings.TrimSpace(command.LastName),
		Email:          strings.TrimSpace(command.Email),
		Phone:          strings.TrimSpace(command.Phone),
		Address:        strings.TrimSpace(command.Address),
		AddressExtra:   strings.TrimSpace(command.AddressExtra),
		CityCode:       strings.TrimSpace(command.CityCode),
	}
	if err := entity.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, entity); err != nil {
		return nil, fmt.Errorf("create contact: %w", err)
	}
	if err := s.publisher.Publish(ctx, buildContactCreatedIntegrationEvent(domain.NewContactCreatedEvent(*entity))); err != nil {
		return nil, fmt.Errorf("publish contact created event: %w", err)
	}

	return entity, nil
}

// Get handles contact retrieval by id.
func (s *ContactService) Get(ctx context.Context, id string) (*domain.Contact, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("get contact: %w", err)
	}

	return entity, nil
}

// List handles paginated contact querying.
func (s *ContactService) List(ctx context.Context, query port.ListQuery) (*ListResult, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	normalized := query
	normalized.Page = page
	normalized.Limit = limit

	data, total, err := s.repository.List(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return &ListResult{
		Data:       data,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// Update handles contact updates.
func (s *ContactService) Update(ctx context.Context, id string, command UpdateCommand) (*domain.Contact, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("load contact for update: %w", err)
	}

	if command.DocumentType != nil {
		entity.DocumentType = *command.DocumentType
	}
	if command.DocumentNumber != nil {
		entity.DocumentNumber = strings.TrimSpace(*command.DocumentNumber)
	}
	if command.LegalName != nil {
		entity.LegalName = strings.TrimSpace(*command.LegalName)
	}
	if command.FirstName != nil {
		entity.FirstName = strings.TrimSpace(*command.FirstName)
	}
	if command.LastName != nil {
		entity.LastName = strings.TrimSpace(*command.LastName)
	}
	if command.Email != nil {
		entity.Email = strings.TrimSpace(*command.Email)
	}
	if command.Phone != nil {
		entity.Phone = strings.TrimSpace(*command.Phone)
	}
	if command.Address != nil {
		entity.Address = strings.TrimSpace(*command.Address)
	}
	if command.AddressExtra != nil {
		entity.AddressExtra = strings.TrimSpace(*command.AddressExtra)
	}
	if command.CityCode != nil {
		entity.CityCode = strings.TrimSpace(*command.CityCode)
	}

	if err := entity.Validate(); err != nil {
		return nil, err
	}
	if err := s.repository.Update(ctx, entity); err != nil {
		return nil, fmt.Errorf("update contact: %w", err)
	}
	if err := s.publisher.Publish(ctx, buildContactUpdatedIntegrationEvent(domain.NewContactUpdatedEvent(*entity))); err != nil {
		return nil, fmt.Errorf("publish contact updated event: %w", err)
	}

	return entity, nil
}

// Delete handles contact deletion.
func (s *ContactService) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return ErrInvalidID
	}

	if err := s.repository.Delete(ctx, trimmedID); err != nil {
		return fmt.Errorf("delete contact: %w", err)
	}

	return nil
}
