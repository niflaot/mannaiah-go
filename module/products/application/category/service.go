package category

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	categorydomain "mannaiah/module/products/domain/category"
	categoryport "mannaiah/module/products/port/category"
)

var (
	// ErrNilRepository is returned when repository dependencies are nil.
	ErrNilRepository = errors.New("category repository must not be nil")
	// ErrInvalidID is returned when category IDs are empty.
	ErrInvalidID = errors.New("category id is required")
	// ErrNotFound is returned when category records are missing.
	ErrNotFound = errors.New("category not found")
	// ErrDuplicateSlug is returned when a slug already exists.
	ErrDuplicateSlug = errors.New("category slug already exists")
	// ErrHasChildren is returned when a category with children is deleted.
	ErrHasChildren = errors.New("category has children and cannot be deleted")
)

// CreateCommand defines create-category command payloads.
type CreateCommand struct {
	// Slug defines URL-friendly category slug values.
	Slug string
	// Name defines human-readable category name values.
	Name string
	// Description defines optional category description values.
	Description string
	// ParentID defines optional parent category identifier.
	ParentID *string
	// IncludeChildren reports whether descendant categories are included in product resolution.
	IncludeChildren bool
	// FilterTags defines tag filter values for product membership.
	FilterTags []string
	// FilterMinPrice defines optional minimum price filter value.
	FilterMinPrice *float64
	// FilterMaxPrice defines optional maximum price filter value.
	FilterMaxPrice *float64
	// FilterCategoryRefs defines category ID references for product membership.
	FilterCategoryRefs []string
	// ProductIDs defines manually pinned product identifiers.
	ProductIDs []string
}

// UpdateCommand defines update-category command payloads.
type UpdateCommand struct {
	// Slug defines optional slug updates.
	Slug *string
	// Name defines optional name updates.
	Name *string
	// Description defines optional description updates.
	Description *string
	// ParentID defines optional parent category update.
	ParentID *string
	// HasParentID reports whether ParentID was explicitly provided.
	HasParentID bool
	// IncludeChildren defines optional include-children updates.
	IncludeChildren *bool
	// FilterTags defines optional tag filter replacement values.
	FilterTags []string
	// HasFilterTags reports whether FilterTags was provided.
	HasFilterTags bool
	// FilterMinPrice defines optional minimum price filter update.
	FilterMinPrice *float64
	// FilterMaxPrice defines optional maximum price filter update.
	FilterMaxPrice *float64
	// HasFilterPriceRange reports whether filter price range was provided.
	HasFilterPriceRange bool
	// FilterCategoryRefs defines optional category ref replacement values.
	FilterCategoryRefs []string
	// HasFilterCategoryRefs reports whether FilterCategoryRefs was provided.
	HasFilterCategoryRefs bool
	// ProductIDs defines optional pinned product ID replacement values.
	ProductIDs []string
	// HasProductIDs reports whether ProductIDs was provided.
	HasProductIDs bool
}

// ListProductsQuery defines parameters for product listing within a category.
type ListProductsQuery struct {
	// CategoryID defines the target category identifier.
	CategoryID string
	// Page defines the 1-based page number.
	Page int
	// PageSize defines the maximum number of results per page.
	PageSize int
}

// Service defines category application use cases.
type Service interface {
	// Create creates a category.
	Create(ctx context.Context, command CreateCommand) (*categorydomain.Category, error)
	// Get retrieves a category by ID.
	Get(ctx context.Context, id string) (*categorydomain.Category, error)
	// GetBySlug retrieves a category by slug.
	GetBySlug(ctx context.Context, slug string) (*categorydomain.Category, error)
	// Tree returns all root-level categories.
	Tree(ctx context.Context) ([]*categorydomain.Category, error)
	// Children returns direct children of a category.
	Children(ctx context.Context, parentID string) ([]*categorydomain.Category, error)
	// Update updates a category by ID.
	Update(ctx context.Context, id string, command UpdateCommand) (*categorydomain.Category, error)
	// Delete deletes a category by ID.
	Delete(ctx context.Context, id string) error
	// ListProducts returns paginated products for a category.
	ListProducts(ctx context.Context, q ListProductsQuery) (*categoryport.ListProductsResult, error)
}

// CategoryService implements category use cases.
type CategoryService struct {
	// repository defines persistence dependencies.
	repository categoryport.Repository
	// storefrontNavigationRefresher defines optional storefront navigation refresh dependencies.
	storefrontNavigationRefresher StorefrontNavigationRefresher
}

var (
	// _ ensures CategoryService satisfies Service contracts.
	_ Service = (*CategoryService)(nil)
)

// NewService creates category services.
func NewService(repo categoryport.Repository) (*CategoryService, error) {
	if repo == nil {
		return nil, ErrNilRepository
	}

	return &CategoryService{repository: repo}, nil
}

// Create creates a category.
func (s *CategoryService) Create(ctx context.Context, command CreateCommand) (*categorydomain.Category, error) {
	entity := &categorydomain.Category{
		ID:              generateCategoryID(),
		Slug:            strings.TrimSpace(command.Slug),
		Name:            strings.TrimSpace(command.Name),
		Description:     strings.TrimSpace(command.Description),
		ParentID:        command.ParentID,
		IncludeChildren: command.IncludeChildren,
		ProductIDs:      command.ProductIDs,
		Filter: categorydomain.Filter{
			Tags:         command.FilterTags,
			CategoryRefs: command.FilterCategoryRefs,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if command.FilterMinPrice != nil || command.FilterMaxPrice != nil {
		entity.Filter.PriceRange = &categorydomain.PriceRange{
			Min: command.FilterMinPrice,
			Max: command.FilterMaxPrice,
		}
	}

	entity.Normalize()
	if err := entity.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, entity); err != nil {
		return nil, mapPortError(err)
	}
	s.triggerStorefrontNavigationRefresh(ctx)

	return entity, nil
}

// Get retrieves a category by ID.
func (s *CategoryService) Get(ctx context.Context, id string) (*categorydomain.Category, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, mapPortError(err)
	}

	return entity, nil
}

// GetBySlug retrieves a category by slug.
func (s *CategoryService) GetBySlug(ctx context.Context, slug string) (*categorydomain.Category, error) {
	trimmedSlug := strings.TrimSpace(slug)
	if trimmedSlug == "" {
		return nil, categorydomain.ErrSlugRequired
	}

	entity, err := s.repository.GetBySlug(ctx, trimmedSlug)
	if err != nil {
		return nil, mapPortError(err)
	}

	return entity, nil
}

// Tree returns all root-level categories.
func (s *CategoryService) Tree(ctx context.Context) ([]*categorydomain.Category, error) {
	result, err := s.repository.Tree(ctx)
	if err != nil {
		return nil, fmt.Errorf("list category tree: %w", err)
	}

	return result, nil
}

// Children returns direct children of a category.
func (s *CategoryService) Children(ctx context.Context, parentID string) ([]*categorydomain.Category, error) {
	trimmedID := strings.TrimSpace(parentID)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	result, err := s.repository.ListChildren(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("list category children: %w", err)
	}

	return result, nil
}

// Update updates a category by ID.
func (s *CategoryService) Update(ctx context.Context, id string, command UpdateCommand) (*categorydomain.Category, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, mapPortError(err)
	}

	if command.Slug != nil {
		entity.Slug = strings.TrimSpace(*command.Slug)
	}
	if command.Name != nil {
		entity.Name = strings.TrimSpace(*command.Name)
	}
	if command.Description != nil {
		entity.Description = strings.TrimSpace(*command.Description)
	}
	if command.HasParentID {
		entity.ParentID = command.ParentID
	}
	if command.IncludeChildren != nil {
		entity.IncludeChildren = *command.IncludeChildren
	}
	if command.HasFilterTags {
		entity.Filter.Tags = command.FilterTags
	}
	if command.HasFilterPriceRange {
		if command.FilterMinPrice != nil || command.FilterMaxPrice != nil {
			entity.Filter.PriceRange = &categorydomain.PriceRange{
				Min: command.FilterMinPrice,
				Max: command.FilterMaxPrice,
			}
		} else {
			entity.Filter.PriceRange = nil
		}
	}
	if command.HasFilterCategoryRefs {
		entity.Filter.CategoryRefs = command.FilterCategoryRefs
	}
	if command.HasProductIDs {
		entity.ProductIDs = command.ProductIDs
	}

	entity.UpdatedAt = time.Now()
	entity.Normalize()
	if err := entity.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Update(ctx, entity); err != nil {
		return nil, mapPortError(err)
	}
	s.triggerStorefrontNavigationRefresh(ctx)

	return entity, nil
}

// Delete deletes a category by ID.
func (s *CategoryService) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return ErrInvalidID
	}

	if err := s.repository.Delete(ctx, trimmedID); err != nil {
		return mapPortError(err)
	}
	s.triggerStorefrontNavigationRefresh(ctx)

	return nil
}

// ListProducts returns paginated products for a category.
func (s *CategoryService) ListProducts(ctx context.Context, q ListProductsQuery) (*categoryport.ListProductsResult, error) {
	trimmedID := strings.TrimSpace(q.CategoryID)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	result, err := s.repository.ListProducts(ctx, categoryport.ListProductsQuery{
		CategoryID: trimmedID,
		Page:       q.Page,
		PageSize:   q.PageSize,
	})
	if err != nil {
		return nil, mapPortError(err)
	}

	return result, nil
}

// mapPortError translates port-layer errors to service-layer sentinel errors.
func mapPortError(err error) error {
	if errors.Is(err, categoryport.ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, categoryport.ErrDuplicateSlug) {
		return ErrDuplicateSlug
	}
	if errors.Is(err, categoryport.ErrHasChildren) {
		return ErrHasChildren
	}

	return err
}

// generateCategoryID creates random category identifiers.
func generateCategoryID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(b)
}
