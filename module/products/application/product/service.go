package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	productdomain "mannaiah/module/products/domain/product"
	productport "mannaiah/module/products/port/product"
)

var (
	// ErrNilRepository is returned when repository dependencies are nil.
	ErrNilRepository = errors.New("products repository must not be nil")
	// ErrInvalidID is returned when ids are empty.
	ErrInvalidID = errors.New("product id is required")
)

// CreateCommand defines create-product command payloads.
type CreateCommand struct {
	// SKU defines product stock-keeping values.
	SKU string
	// Gallery defines product gallery entries.
	Gallery []productdomain.GalleryItem
	// Datasheets defines product datasheet entries.
	Datasheets []productdomain.Datasheet
	// Variations defines product-linked variation IDs.
	Variations []string
	// Variants defines product variant entries.
	Variants []productdomain.Variant
}

// UpdateCommand defines update-product command payloads.
type UpdateCommand struct {
	// SKU defines optional SKU updates.
	SKU *string
	// Gallery defines optional gallery replacement values.
	Gallery []productdomain.GalleryItem
	// Datasheets defines optional datasheet upsert values.
	Datasheets []productdomain.Datasheet
	// Variations defines optional variation replacement values.
	Variations []string
	// Variants defines optional variant replacement values.
	Variants []productdomain.Variant
	// HasGallery reports whether Gallery values were provided in payload.
	HasGallery bool
	// HasDatasheets reports whether Datasheets values were provided in payload.
	HasDatasheets bool
	// HasVariations reports whether Variations values were provided in payload.
	HasVariations bool
	// HasVariants reports whether Variants values were provided in payload.
	HasVariants bool
}

// Service defines product application use cases.
type Service interface {
	// Create creates a product.
	Create(ctx context.Context, command CreateCommand) (*productdomain.Product, error)
	// Get retrieves a product by ID.
	Get(ctx context.Context, id string) (*productdomain.Product, error)
	// List lists all products.
	List(ctx context.Context) ([]productdomain.Product, error)
	// Update updates products by ID.
	Update(ctx context.Context, id string, command UpdateCommand) (*productdomain.Product, error)
	// Delete deletes products by ID.
	Delete(ctx context.Context, id string) error
}

// ProductService implements product use cases.
type ProductService struct {
	// repository defines persistence dependencies.
	repository productport.Repository
}

var (
	// _ ensures ProductService satisfies Service contracts.
	_ Service = (*ProductService)(nil)
)

// NewService creates product services.
func NewService(repository productport.Repository) (*ProductService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}

	return &ProductService{repository: repository}, nil
}

// Create creates products.
func (s *ProductService) Create(ctx context.Context, command CreateCommand) (*productdomain.Product, error) {
	entity := &productdomain.Product{
		SKU:        strings.TrimSpace(command.SKU),
		Gallery:    command.Gallery,
		Datasheets: command.Datasheets,
		Variations: command.Variations,
		Variants:   command.Variants,
	}
	entity.Normalize()
	if err := entity.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, entity); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	return entity, nil
}

// Get retrieves products by ID.
func (s *ProductService) Get(ctx context.Context, id string) (*productdomain.Product, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}

	return entity, nil
}

// List lists products.
func (s *ProductService) List(ctx context.Context) ([]productdomain.Product, error) {
	entities, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	return entities, nil
}

// Update updates products by ID.
func (s *ProductService) Update(ctx context.Context, id string, command UpdateCommand) (*productdomain.Product, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("load product for update: %w", err)
	}

	if command.SKU != nil {
		entity.SKU = strings.TrimSpace(*command.SKU)
	}
	if command.HasGallery {
		entity.Gallery = command.Gallery
	}
	if command.HasDatasheets {
		entity.Datasheets = productdomain.MergeDatasheets(entity.Datasheets, command.Datasheets)
	}
	if command.HasVariations {
		entity.Variations = command.Variations
	}
	if command.HasVariants {
		entity.Variants = command.Variants
	}

	entity.Normalize()
	if err := entity.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Update(ctx, entity); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	return entity, nil
}

// Delete deletes products by ID.
func (s *ProductService) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return ErrInvalidID
	}

	if err := s.repository.Delete(ctx, trimmedID); err != nil {
		return fmt.Errorf("delete product: %w", err)
	}

	return nil
}
