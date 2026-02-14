package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

var (
	// ErrNilRepository is returned when repository dependencies are nil.
	ErrNilRepository = errors.New("orders repository must not be nil")
	// ErrNilCustomerSource is returned when customer-source dependencies are nil.
	ErrNilCustomerSource = errors.New("orders customer source must not be nil")
	// ErrInvalidID is returned when order identifiers are empty.
	ErrInvalidID = errors.New("order id is required")
	// ErrStatusAuthorRequired is returned when update-status author values are empty.
	ErrStatusAuthorRequired = errors.New("order status author is required")
)

// CreateItemCommand defines order-item creation payload values.
type CreateItemCommand struct {
	// SKU defines product SKU values.
	SKU string
	// AlternateName defines alternate product-name values.
	AlternateName string
	// Quantity defines ordered quantity values.
	Quantity int
	// Metadata defines item metadata values.
	Metadata map[string]string
}

// ShippingAddressCommand defines shipping-address command values.
type ShippingAddressCommand struct {
	// Address defines shipping address line 1 values.
	Address string
	// Address2 defines shipping address line 2 values.
	Address2 string
	// Phone defines shipping phone values.
	Phone string
	// CityCode defines shipping city-code values.
	CityCode string
}

// CreateCommand defines order creation payload values.
type CreateCommand struct {
	// Identifier defines external order identifiers.
	Identifier string
	// Realm defines order realm values.
	Realm string
	// ContactID defines customer contact identifiers.
	ContactID string
	// Items defines order item values.
	Items []CreateItemCommand
	// InitialStatus defines optional initial-status values.
	InitialStatus *ordersdomain.Status
	// Author defines initial-status author values.
	Author string
	// Description defines initial-status description values.
	Description string
	// ShippingAddress defines optional explicit shipping-address values.
	ShippingAddress *ShippingAddressCommand
	// Metadata defines order metadata values.
	Metadata map[string]string
	// CreatedAt defines optional source creation timestamps.
	CreatedAt *time.Time
}

// UpdateStatusCommand defines status-update payload values.
type UpdateStatusCommand struct {
	// Status defines next status values.
	Status ordersdomain.Status
	// Author defines status author values.
	Author string
	// Description defines status description values.
	Description string
	// OccurredAt defines optional status timestamp values.
	OccurredAt *time.Time
}

// ListQuery defines list payload values.
type ListQuery struct {
	// Page defines requested page values.
	Page int
	// Limit defines requested page-size values.
	Limit int
	// Realm defines optional realm filter values.
	Realm string
	// ContactID defines optional contact-id filter values.
	ContactID string
	// Identifier defines optional identifier filter values.
	Identifier string
	// Status defines optional status filter values.
	Status ordersdomain.Status
}

// ListResult defines paginated order list result values.
type ListResult struct {
	// Data defines result rows.
	Data []ordersdomain.Order
	// Page defines current page values.
	Page int
	// Limit defines current page-size values.
	Limit int
	// Total defines filtered total values.
	Total int64
	// TotalPages defines total-page values.
	TotalPages int
}

// Service defines orders application behavior.
type Service interface {
	// Create creates order aggregate values.
	Create(ctx context.Context, command CreateCommand) (*ordersdomain.Order, error)
	// Get resolves order aggregate values by identifiers.
	Get(ctx context.Context, id string) (*ordersdomain.Order, error)
	// List lists paginated order aggregate values.
	List(ctx context.Context, query ListQuery) (*ListResult, error)
	// UpdateStatus appends status values for order identifiers.
	UpdateStatus(ctx context.Context, id string, command UpdateStatusCommand) (*ordersdomain.Order, error)
}

// OrderService defines orders application dependencies.
type OrderService struct {
	// repository defines repository dependencies.
	repository ordersport.Repository
	// customerSource defines customer lookup dependencies.
	customerSource ordersport.CustomerSource
	// productResolver defines product lookup dependencies.
	productResolver ordersport.ProductResolver
}

var (
	// _ ensures OrderService satisfies service contracts.
	_ Service = (*OrderService)(nil)
)

// NewService creates order application services.
func NewService(repository ordersport.Repository, customerSource ordersport.CustomerSource, resolvers ...ordersport.ProductResolver) (*OrderService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}
	if customerSource == nil {
		return nil, ErrNilCustomerSource
	}

	var productResolver ordersport.ProductResolver
	if len(resolvers) > 0 {
		productResolver = resolvers[0]
	}

	return &OrderService{
		repository:      repository,
		customerSource:  customerSource,
		productResolver: productResolver,
	}, nil
}

// Create creates order aggregate values.
func (s *OrderService) Create(ctx context.Context, command CreateCommand) (*ordersdomain.Order, error) {
	customer, err := s.customerSource.GetByID(ctx, strings.TrimSpace(command.ContactID))
	if err != nil {
		return nil, fmt.Errorf("resolve order customer: %w", err)
	}

	items, err := s.resolveItems(ctx, command.Items)
	if err != nil {
		return nil, err
	}

	initialStatus := ordersdomain.StatusCreated
	if command.InitialStatus != nil {
		initialStatus = *command.InitialStatus
	}
	author := strings.TrimSpace(command.Author)
	if author == "" {
		author = "system"
	}
	entry := ordersdomain.StatusEntry{
		Status:      initialStatus,
		Author:      author,
		Description: strings.TrimSpace(command.Description),
		OccurredAt:  time.Now().UTC(),
	}
	order := &ordersdomain.Order{
		Identifier:    strings.TrimSpace(command.Identifier),
		Realm:         strings.TrimSpace(command.Realm),
		ContactID:     strings.TrimSpace(command.ContactID),
		Items:         items,
		CurrentStatus: initialStatus,
		StatusHistory: []ordersdomain.StatusEntry{entry},
		Metadata:      command.Metadata,
	}
	if command.CreatedAt != nil && !command.CreatedAt.IsZero() {
		order.CreatedAt = command.CreatedAt.UTC()
	}
	applyShipping(order, customer, command.ShippingAddress)
	order.Normalize()
	if err := order.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	s.enrichShippingWithBilling(ctx, order)

	return order, nil
}

// Get resolves order aggregate values by identifiers.
func (s *OrderService) Get(ctx context.Context, id string) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	s.enrichShippingWithBilling(ctx, entity)

	return entity, nil
}

// List lists paginated order aggregate values.
func (s *OrderService) List(ctx context.Context, query ListQuery) (*ListResult, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	rows, total, err := s.repository.List(ctx, ordersport.ListQuery{
		Page:       page,
		Limit:      limit,
		Realm:      strings.TrimSpace(query.Realm),
		ContactID:  strings.TrimSpace(query.ContactID),
		Identifier: strings.TrimSpace(query.Identifier),
		Status:     query.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}

	for index := range rows {
		s.enrichShippingWithBilling(ctx, &rows[index])
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return &ListResult{
		Data:       rows,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// UpdateStatus appends status values for order identifiers.
func (s *OrderService) UpdateStatus(ctx context.Context, id string, command UpdateStatusCommand) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	entry := ordersdomain.StatusEntry{
		Status:      command.Status,
		Author:      strings.TrimSpace(command.Author),
		Description: strings.TrimSpace(command.Description),
		OccurredAt:  time.Now().UTC(),
	}
	if command.OccurredAt != nil && !command.OccurredAt.IsZero() {
		entry.OccurredAt = command.OccurredAt.UTC()
	}
	if strings.TrimSpace(entry.Author) == "" {
		return nil, ErrStatusAuthorRequired
	}
	if err := validateStatusEntry(entry); err != nil {
		return nil, err
	}

	entity, err := s.repository.AppendStatus(ctx, trimmedID, entry)
	if err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}
	s.enrichShippingWithBilling(ctx, entity)

	return entity, nil
}
