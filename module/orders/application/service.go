package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	ordersevent "mannaiah/module/orders/application/event"
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
	// ErrEmptyOrderUpdate is returned when update commands do not include mutable fields.
	ErrEmptyOrderUpdate = errors.New("order update command requires at least one mutable field")
	// ErrInvalidCommentID is returned when order comment identifiers are empty.
	ErrInvalidCommentID = errors.New("order comment id is required")
	// ErrEmptyCommentUpdate is returned when comment update commands do not include mutable fields.
	ErrEmptyCommentUpdate = errors.New("order comment update command requires at least one mutable field")
)

// CreateItemCommand defines order-item creation payload values.
type CreateItemCommand struct {
	// SKU defines product SKU values.
	SKU string
	// AlternateName defines alternate product-name values.
	AlternateName string
	// Quantity defines ordered quantity values.
	Quantity int
	// Value defines item monetary value values.
	Value float64
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

// ShippingChargeCommand defines shipping-charge command values.
type ShippingChargeCommand struct {
	// MethodID defines shipping method identifier values.
	MethodID string
	// MethodTitle defines shipping method display title values.
	MethodTitle string
	// Price defines shipping price values.
	Price float64
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
	// ShippingCharges defines shipping charge values.
	ShippingCharges []ShippingChargeCommand
	// Metadata defines order metadata values.
	Metadata map[string]string
	// CreatedAt defines optional source creation timestamps.
	CreatedAt *time.Time
	// Source defines mutation source values.
	Source string
}

// UpdateCommand defines mutable update payload values.
type UpdateCommand struct {
	// Items defines optional order item values.
	Items *[]CreateItemCommand
	// ShippingAddress defines optional explicit shipping-address values.
	ShippingAddress *ShippingAddressCommand
	// ShippingCharges defines optional shipping charge values.
	ShippingCharges *[]ShippingChargeCommand
	// Source defines mutation source values.
	Source string
}

// UpdateStatusCommand defines status-update payload values.
type UpdateStatusCommand struct {
	// Status defines next status values.
	Status ordersdomain.Status
	// Author defines status author values.
	Author string
	// Description defines status description values.
	Description string
	// NoteOwner defines optional note owner values associated with this status transition.
	NoteOwner string
	// Note defines optional note text values associated with this status transition.
	Note string
	// OccurredAt defines optional status timestamp values.
	OccurredAt *time.Time
	// Source defines mutation source values.
	Source string
}

// AddCommentCommand defines comment-append payload values.
type AddCommentCommand struct {
	// Author defines comment author values.
	Author string
	// Comment defines comment text values.
	Comment string
	// Internal reports whether comments are internal-only.
	Internal bool
	// OccurredAt defines optional comment timestamp values.
	OccurredAt *time.Time
	// Source defines mutation source values.
	Source string
}

// UpdateCommentCommand defines comment-update payload values.
type UpdateCommentCommand struct {
	// Author defines optional comment author values.
	Author *string
	// Comment defines optional comment text values.
	Comment *string
	// Internal reports optional internal-visibility values.
	Internal *bool
	// Source defines mutation source values.
	Source string
}

// DeleteCommentCommand defines comment-delete payload values.
type DeleteCommentCommand struct {
	// Source defines mutation source values.
	Source string
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
	// Update updates mutable order aggregate values.
	Update(ctx context.Context, id string, command UpdateCommand) (*ordersdomain.Order, error)
	// Get resolves order aggregate values by identifiers.
	Get(ctx context.Context, id string) (*ordersdomain.Order, error)
	// List lists paginated order aggregate values.
	List(ctx context.Context, query ListQuery) (*ListResult, error)
	// UpdateStatus appends status values for order identifiers.
	UpdateStatus(ctx context.Context, id string, command UpdateStatusCommand) (*ordersdomain.Order, error)
	// AddComment appends comment values for order identifiers.
	AddComment(ctx context.Context, id string, command AddCommentCommand) (*ordersdomain.Order, error)
	// UpdateComment updates comment values for order identifiers.
	UpdateComment(ctx context.Context, id string, commentID string, command UpdateCommentCommand) (*ordersdomain.Order, error)
	// DeleteComment deletes comment values for order identifiers.
	DeleteComment(ctx context.Context, id string, commentID string, command DeleteCommentCommand) (*ordersdomain.Order, error)
}

// OrderService defines orders application dependencies.
type OrderService struct {
	// repository defines repository dependencies.
	repository ordersport.Repository
	// customerSource defines customer lookup dependencies.
	customerSource ordersport.CustomerSource
	// productResolver defines product lookup dependencies.
	productResolver ordersport.ProductResolver
	// publisher defines integration event publication dependencies.
	publisher ordersport.IntegrationEventPublisher
}

var (
	// _ ensures OrderService satisfies service contracts.
	_ Service = (*OrderService)(nil)
)

// NewService creates order application services.
func NewService(repository ordersport.Repository, customerSource ordersport.CustomerSource, resolvers ...ordersport.ProductResolver) (*OrderService, error) {
	return NewServiceWithPublisher(repository, customerSource, nil, resolvers...)
}

// NewServiceWithPublisher creates order application services with integration event publishing support.
func NewServiceWithPublisher(
	repository ordersport.Repository,
	customerSource ordersport.CustomerSource,
	publisher ordersport.IntegrationEventPublisher,
	resolvers ...ordersport.ProductResolver,
) (*OrderService, error) {
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
		publisher:       ordersevent.ResolvePublisher(publisher),
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
		Identifier:      strings.TrimSpace(command.Identifier),
		Realm:           strings.TrimSpace(command.Realm),
		ContactID:       strings.TrimSpace(command.ContactID),
		Items:           items,
		CurrentStatus:   initialStatus,
		StatusHistory:   []ordersdomain.StatusEntry{entry},
		ShippingCharges: normalizeShippingCharges(command.ShippingCharges),
		Metadata:        command.Metadata,
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
	if err := s.publisher.Publish(
		ctx,
		ordersevent.BuildOrderCreatedIntegrationEvent(*order, command.Source),
	); err != nil {
		return nil, fmt.Errorf("publish order created event: %w", err)
	}

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
