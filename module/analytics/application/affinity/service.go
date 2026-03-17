package affinity

import (
	"context"
	"errors"
	"fmt"

	"mannaiah/module/analytics/domain"
	"mannaiah/module/analytics/port"
)

var (
	// ErrNilAffinityStore is returned when a nil affinity store dependency is provided.
	ErrNilAffinityStore = errors.New("affinity store must not be nil")
)

// Service defines affinity use-case behavior.
type Service interface {
	// GetTagAffinity retrieves ranked tag affinity scores for one contact.
	GetTagAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.TagAffinity, error)
	// GetCategoryAffinity retrieves ranked category affinity scores for one contact.
	GetCategoryAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.CategoryAffinity, error)
	// GetVariationAffinity retrieves ranked variation affinity scores for one contact.
	GetVariationAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.VariationAffinity, error)
	// GetProfile assembles a full affinity profile for one contact.
	GetProfile(ctx context.Context, contactID string, limit int, minScore float64) (*domain.AffinityProfile, error)
	// RefreshAll truncates and repopulates all affinity materialized views.
	RefreshAll(ctx context.Context) error
}

// AffinityService implements affinity use-cases.
type AffinityService struct {
	// store defines ClickHouse affinity store dependencies.
	store port.AffinityStore
}

var _ Service = (*AffinityService)(nil)

// NewService creates affinity services with required dependencies.
func NewService(store port.AffinityStore) (*AffinityService, error) {
	if store == nil {
		return nil, ErrNilAffinityStore
	}

	return &AffinityService{store: store}, nil
}

// GetTagAffinity retrieves ranked tag affinity scores for one contact.
func (s *AffinityService) GetTagAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.TagAffinity, error) {
	rows, err := s.store.GetTagAffinity(ctx, contactID, limit, minScore)
	if err != nil {
		return nil, fmt.Errorf("get tag affinity: %w", err)
	}

	return rows, nil
}

// GetCategoryAffinity retrieves ranked category affinity scores for one contact.
func (s *AffinityService) GetCategoryAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.CategoryAffinity, error) {
	rows, err := s.store.GetCategoryAffinity(ctx, contactID, limit, minScore)
	if err != nil {
		return nil, fmt.Errorf("get category affinity: %w", err)
	}

	return rows, nil
}

// GetVariationAffinity retrieves ranked variation affinity scores for one contact.
func (s *AffinityService) GetVariationAffinity(ctx context.Context, contactID string, limit int, minScore float64) ([]domain.VariationAffinity, error) {
	rows, err := s.store.GetVariationAffinity(ctx, contactID, limit, minScore)
	if err != nil {
		return nil, fmt.Errorf("get variation affinity: %w", err)
	}

	return rows, nil
}

// GetProfile assembles a full affinity profile for one contact.
func (s *AffinityService) GetProfile(ctx context.Context, contactID string, limit int, minScore float64) (*domain.AffinityProfile, error) {
	profile, err := s.store.GetProfile(ctx, contactID, limit, minScore)
	if err != nil {
		return nil, fmt.Errorf("get affinity profile: %w", err)
	}

	return profile, nil
}

// RefreshAll truncates and repopulates all affinity materialized views.
func (s *AffinityService) RefreshAll(ctx context.Context) error {
	if err := s.store.RefreshTagMV(ctx); err != nil {
		return fmt.Errorf("refresh tag affinity mv: %w", err)
	}
	if err := s.store.RefreshCategoryMV(ctx); err != nil {
		return fmt.Errorf("refresh category affinity mv: %w", err)
	}
	if err := s.store.RefreshVariationMV(ctx); err != nil {
		return fmt.Errorf("refresh variation affinity mv: %w", err)
	}

	return nil
}
