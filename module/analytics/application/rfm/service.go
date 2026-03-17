package rfm

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"mannaiah/module/analytics/domain"
	"mannaiah/module/analytics/port"
)

const (
	// bandCacheTTL defines band configuration cache expiry duration.
	bandCacheTTL = 5 * time.Minute
)

var (
	// ErrNilRFMStore is returned when a nil RFM store dependency is provided.
	ErrNilRFMStore = errors.New("rfm store must not be nil")
	// ErrNilGroupRepo is returned when a nil group repository dependency is provided.
	ErrNilGroupRepo = errors.New("rfm group repository must not be nil")
	// ErrGroupNotFound is returned when an RFM group does not exist.
	ErrGroupNotFound = errors.New("rfm group not found")
)

// Service defines RFM use-case behavior.
type Service interface {
	// CreateGroup persists a new RFM group definition.
	CreateGroup(ctx context.Context, group domain.RFMGroup) (*domain.RFMGroup, error)
	// GetGroup retrieves one RFM group by identifier.
	GetGroup(ctx context.Context, id string) (*domain.RFMGroup, error)
	// ListGroups retrieves all RFM groups.
	ListGroups(ctx context.Context) ([]domain.RFMGroup, error)
	// UpdateGroup persists RFM group updates.
	UpdateGroup(ctx context.Context, group domain.RFMGroup) (*domain.RFMGroup, error)
	// DeleteGroup removes one RFM group by identifier.
	DeleteGroup(ctx context.Context, id string) error
	// GetBands retrieves all RFM band threshold configurations.
	GetBands(ctx context.Context) ([]domain.RFMBandConfig, error)
	// UpdateBand persists a single RFM band configuration and invalidates the cache.
	UpdateBand(ctx context.Context, cfg domain.RFMBandConfig) error
	// ScoreContact computes RFM scores for one contact.
	ScoreContact(ctx context.Context, contactID string) (*domain.RFMScore, error)
	// ScoreBatch computes RFM scores for up to 1000 contacts.
	ScoreBatch(ctx context.Context, contactIDs []string) ([]domain.RFMScore, error)
	// RefreshMV truncates and repopulates the rfm_scores_mv ClickHouse table.
	RefreshMV(ctx context.Context) error
}

// bandCache stores band configurations with TTL-based expiry.
type bandCache struct {
	mu        sync.RWMutex
	bands     []domain.RFMBandConfig
	expiresAt time.Time
}

// get returns cached bands and whether the cache is still valid.
func (c *bandCache) get() ([]domain.RFMBandConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Now().Before(c.expiresAt) && len(c.bands) > 0 {
		return c.bands, true
	}

	return nil, false
}

// set stores bands in the cache with a TTL.
func (c *bandCache) set(bands []domain.RFMBandConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bands = bands
	c.expiresAt = time.Now().Add(bandCacheTTL)
}

// invalidate clears the cache.
func (c *bandCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bands = nil
	c.expiresAt = time.Time{}
}

// RFMService implements RFM use-cases.
type RFMService struct {
	// rfmStore defines ClickHouse RFM scoring dependencies.
	rfmStore port.RFMStore
	// groupRepo defines RFM group persistence dependencies.
	groupRepo port.RFMGroupRepository
	// cache holds band configuration cache state.
	cache bandCache
}

var _ Service = (*RFMService)(nil)

// NewService creates RFM services with required dependencies.
func NewService(rfmStore port.RFMStore, groupRepo port.RFMGroupRepository) (*RFMService, error) {
	if rfmStore == nil {
		return nil, ErrNilRFMStore
	}
	if groupRepo == nil {
		return nil, ErrNilGroupRepo
	}

	return &RFMService{rfmStore: rfmStore, groupRepo: groupRepo}, nil
}

// CreateGroup persists a new RFM group definition.
func (s *RFMService) CreateGroup(ctx context.Context, group domain.RFMGroup) (*domain.RFMGroup, error) {
	if err := s.groupRepo.Create(ctx, &group); err != nil {
		return nil, fmt.Errorf("create rfm group: %w", err)
	}

	return &group, nil
}

// GetGroup retrieves one RFM group by identifier.
func (s *RFMService) GetGroup(ctx context.Context, id string) (*domain.RFMGroup, error) {
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get rfm group: %w", err)
	}

	return group, nil
}

// ListGroups retrieves all RFM groups.
func (s *RFMService) ListGroups(ctx context.Context) ([]domain.RFMGroup, error) {
	groups, err := s.groupRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list rfm groups: %w", err)
	}

	return groups, nil
}

// UpdateGroup persists RFM group updates.
func (s *RFMService) UpdateGroup(ctx context.Context, group domain.RFMGroup) (*domain.RFMGroup, error) {
	if err := s.groupRepo.Update(ctx, &group); err != nil {
		return nil, fmt.Errorf("update rfm group: %w", err)
	}

	return &group, nil
}

// DeleteGroup removes one RFM group by identifier.
func (s *RFMService) DeleteGroup(ctx context.Context, id string) error {
	if err := s.groupRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete rfm group: %w", err)
	}

	return nil
}

// GetBands retrieves all RFM band threshold configurations, using the in-process cache.
func (s *RFMService) GetBands(ctx context.Context) ([]domain.RFMBandConfig, error) {
	if cached, ok := s.cache.get(); ok {
		return cached, nil
	}

	bands, err := s.groupRepo.GetBandConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("get rfm band configs: %w", err)
	}
	s.cache.set(bands)

	return bands, nil
}

// UpdateBand persists a single RFM band configuration and invalidates the cache.
func (s *RFMService) UpdateBand(ctx context.Context, cfg domain.RFMBandConfig) error {
	if err := s.groupRepo.UpdateBandConfig(ctx, cfg); err != nil {
		return fmt.Errorf("update rfm band config: %w", err)
	}
	s.cache.invalidate()

	return nil
}

// ScoreContact computes RFM scores for one contact.
func (s *RFMService) ScoreContact(ctx context.Context, contactID string) (*domain.RFMScore, error) {
	bands, err := s.GetBands(ctx)
	if err != nil {
		return nil, err
	}

	score, err := s.rfmStore.ScoreContact(ctx, contactID, bands)
	if err != nil {
		return nil, fmt.Errorf("score rfm contact: %w", err)
	}

	return score, nil
}

// ScoreBatch computes RFM scores for up to 1000 contacts.
func (s *RFMService) ScoreBatch(ctx context.Context, contactIDs []string) ([]domain.RFMScore, error) {
	bands, err := s.GetBands(ctx)
	if err != nil {
		return nil, err
	}

	scores, err := s.rfmStore.ScoreBatch(ctx, contactIDs, bands)
	if err != nil {
		return nil, fmt.Errorf("score rfm batch: %w", err)
	}

	return scores, nil
}

// RefreshMV truncates and repopulates the rfm_scores_mv ClickHouse table.
func (s *RFMService) RefreshMV(ctx context.Context) error {
	if err := s.rfmStore.RefreshMV(ctx); err != nil {
		return fmt.Errorf("refresh rfm mv: %w", err)
	}

	return nil
}
