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
	// ErrNilAssetLookup is returned when asset lookup dependencies are nil.
	ErrNilAssetLookup = errors.New("products asset lookup must not be nil")
	// ErrInvalidID is returned when ids are empty.
	ErrInvalidID = errors.New("product id is required")
	// ErrInvalidSKU is returned when sku values are empty.
	ErrInvalidSKU = errors.New("product sku is required")
	// ErrAssetNotFound is returned when referenced gallery assets do not exist.
	ErrAssetNotFound = errors.New("product gallery asset not found")
)

// AssetLookup defines asset-reference lookup behavior required by products.
type AssetLookup interface {
	// Exists verifies whether an asset exists by identifier.
	Exists(ctx context.Context, id string) (bool, error)
}

// TagRegistrar defines tag registry behavior required by products.
// It ensures product tags exist in the canonical registry before persistence.
type TagRegistrar interface {
	// EnsureAll creates missing tags and reintegrates soft-deleted ones.
	EnsureAll(ctx context.Context, names []string) error
}

// CreateCommand defines create-product command payloads.
type CreateCommand struct {
	// SKU defines product stock-keeping values.
	SKU string
	// Price defines optional product price values.
	Price *float64
	// Tags defines product taxonomy tag values.
	Tags []string
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
	// Price defines optional price updates.
	Price *float64
	// Tags defines optional tag replacement values.
	Tags []string
	// Gallery defines optional gallery replacement values.
	Gallery []productdomain.GalleryItem
	// Datasheets defines optional datasheet upsert values.
	Datasheets []productdomain.Datasheet
	// Variations defines optional variation replacement values.
	Variations []string
	// Variants defines optional variant replacement values.
	Variants []productdomain.Variant
	// HasPrice reports whether Price was provided in the payload.
	HasPrice bool
	// HasTags reports whether Tags values were provided in payload.
	HasTags bool
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
	// GetBySKU retrieves a product by product-level or variant-level SKU.
	GetBySKU(ctx context.Context, sku string) (*productdomain.Product, error)
	// List lists all products.
	List(ctx context.Context) ([]productdomain.Product, error)
	// ListByTags lists products filtered by one or more tags with optional pagination.
	// When tags is empty it returns all products. page/pageSize default to 1/20.
	ListByTags(ctx context.Context, tags []string, page, pageSize int) ([]*productdomain.Product, int64, error)
	// Update updates products by ID.
	Update(ctx context.Context, id string, command UpdateCommand) (*productdomain.Product, error)
	// Delete deletes products by ID.
	Delete(ctx context.Context, id string) error
}

// ProductService implements product use cases.
type ProductService struct {
	// repository defines persistence dependencies.
	repository productport.Repository
	// assetLookup defines gallery-asset lookup dependencies.
	assetLookup AssetLookup
	// tagRegistrar defines optional tag registry dependencies.
	tagRegistrar TagRegistrar
	// storefrontNavigationRefresher defines optional storefront navigation refresh dependencies.
	storefrontNavigationRefresher StorefrontNavigationRefresher
}

var (
	// _ ensures ProductService satisfies Service contracts.
	_ Service = (*ProductService)(nil)
)

// NewService creates product services.
func NewService(repository productport.Repository, assetLookup AssetLookup, tagRegistrar ...TagRegistrar) (*ProductService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}
	if assetLookup == nil {
		return nil, ErrNilAssetLookup
	}

	svc := &ProductService{repository: repository, assetLookup: assetLookup}
	if len(tagRegistrar) > 0 {
		svc.tagRegistrar = tagRegistrar[0]
	}

	return svc, nil
}

// SetTagRegistrar configures tag registry dependencies.
func (s *ProductService) SetTagRegistrar(tagRegistrar TagRegistrar) {
	if s == nil {
		return
	}

	s.tagRegistrar = tagRegistrar
}

// Create creates products.
func (s *ProductService) Create(ctx context.Context, command CreateCommand) (*productdomain.Product, error) {
	entity := &productdomain.Product{
		SKU:        strings.TrimSpace(command.SKU),
		Price:      command.Price,
		Tags:       command.Tags,
		Gallery:    command.Gallery,
		Datasheets: command.Datasheets,
		Variations: command.Variations,
		Variants:   command.Variants,
	}
	entity.Normalize()
	if err := entity.Validate(); err != nil {
		return nil, err
	}
	if err := validateGalleryAssets(ctx, s.assetLookup, entity.Gallery); err != nil {
		return nil, err
	}
	if s.tagRegistrar != nil && len(entity.Tags) > 0 {
		if err := s.tagRegistrar.EnsureAll(ctx, entity.Tags); err != nil {
			return nil, fmt.Errorf("ensure product tags: %w", err)
		}
	}

	if err := s.repository.Create(ctx, entity); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	s.triggerStorefrontNavigationRefresh(ctx)

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

// GetBySKU retrieves products by product-level or variant-level SKU.
func (s *ProductService) GetBySKU(ctx context.Context, sku string) (*productdomain.Product, error) {
	trimmed := strings.TrimSpace(sku)
	if trimmed == "" {
		return nil, ErrInvalidSKU
	}

	entity, err := s.repository.GetBySKU(ctx, trimmed)
	if err != nil {
		return nil, fmt.Errorf("get product by sku: %w", err)
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

// ListByTags lists products filtered by tags with optional pagination.
func (s *ProductService) ListByTags(ctx context.Context, tags []string, page, pageSize int) ([]*productdomain.Product, int64, error) {
	entities, total, err := s.repository.ListByTagsAndPrice(ctx, tags, nil, nil, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list products by tags: %w", err)
	}

	return entities, total, nil
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
	if command.HasPrice {
		entity.Price = command.Price
	}
	if command.HasTags {
		entity.Tags = command.Tags
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
	if err := validateGalleryAssets(ctx, s.assetLookup, entity.Gallery); err != nil {
		return nil, err
	}
	if s.tagRegistrar != nil && len(entity.Tags) > 0 {
		if err := s.tagRegistrar.EnsureAll(ctx, entity.Tags); err != nil {
			return nil, fmt.Errorf("ensure product tags: %w", err)
		}
	}

	if err := s.repository.Update(ctx, entity); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}
	s.triggerStorefrontNavigationRefresh(ctx)

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
	s.triggerStorefrontNavigationRefresh(ctx)

	return nil
}
