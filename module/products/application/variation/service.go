package variation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	variationdomain "mannaiah/module/products/domain/variation"
	variationport "mannaiah/module/products/port/variation"
)

var (
	// ErrNilRepository is returned when repository dependencies are nil.
	ErrNilRepository = errors.New("variations repository must not be nil")
	// ErrInvalidID is returned when IDs are empty.
	ErrInvalidID = errors.New("variation id is required")
)

// CreateCommand defines command-side variation creation payload.
type CreateCommand struct {
	// Name defines variation labels.
	Name string
	// Definition defines variation type.
	Definition variationdomain.Definition
	// Value defines machine-readable variation value.
	Value string
}

// UpdateCommand defines command-side variation update payload.
type UpdateCommand struct {
	// Name defines optional variation label updates.
	Name *string
	// Definition defines optional variation definition updates.
	Definition *variationdomain.Definition
	// Value defines optional variation value updates.
	Value *string
}

// Service defines variation application use cases.
type Service interface {
	// Create handles variation creation.
	Create(ctx context.Context, command CreateCommand) (*variationdomain.Variation, error)
	// Get handles variation retrieval by id.
	Get(ctx context.Context, id string) (*variationdomain.Variation, error)
	// List handles variation listing.
	List(ctx context.Context) ([]variationdomain.Variation, error)
	// Update handles variation updates.
	Update(ctx context.Context, id string, command UpdateCommand) (*variationdomain.Variation, error)
	// Delete handles variation deletion.
	Delete(ctx context.Context, id string) error
}

// VariationService implements variation application use cases.
type VariationService struct {
	// repository defines persistence dependencies.
	repository variationport.Repository
}

var (
	// _ ensures VariationService satisfies service contracts.
	_ Service = (*VariationService)(nil)
)

// NewService creates variation services.
func NewService(repository variationport.Repository) (*VariationService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}

	return &VariationService{repository: repository}, nil
}

// Create handles variation creation.
func (s *VariationService) Create(ctx context.Context, command CreateCommand) (*variationdomain.Variation, error) {
	entity := &variationdomain.Variation{
		Name:       strings.TrimSpace(command.Name),
		Definition: command.Definition,
		Value:      strings.TrimSpace(command.Value),
	}
	entity.Normalize()
	if err := entity.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, entity); err != nil {
		return nil, fmt.Errorf("create variation: %w", err)
	}

	return entity, nil
}

// Get handles variation retrieval by ID.
func (s *VariationService) Get(ctx context.Context, id string) (*variationdomain.Variation, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("get variation: %w", err)
	}

	return entity, nil
}

// List handles variation listing.
func (s *VariationService) List(ctx context.Context) ([]variationdomain.Variation, error) {
	variations, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list variations: %w", err)
	}

	return variations, nil
}

// Update handles variation updates.
func (s *VariationService) Update(ctx context.Context, id string, command UpdateCommand) (*variationdomain.Variation, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidID
	}

	entity, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("load variation for update: %w", err)
	}

	if command.Name != nil {
		entity.Name = strings.TrimSpace(*command.Name)
	}
	if command.Value != nil {
		entity.Value = strings.TrimSpace(*command.Value)
	}
	entity.Normalize()
	if err := entity.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Update(ctx, entity); err != nil {
		return nil, fmt.Errorf("update variation: %w", err)
	}

	return entity, nil
}

// Delete handles variation deletion.
func (s *VariationService) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return ErrInvalidID
	}

	if err := s.repository.Delete(ctx, trimmedID); err != nil {
		return fmt.Errorf("delete variation: %w", err)
	}

	return nil
}
