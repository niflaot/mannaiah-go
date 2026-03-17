package port

import (
	"context"

	"mannaiah/module/analytics/domain"
)

// RFMGroupRepository defines persistence behavior for RFM group definitions.
type RFMGroupRepository interface {
	// Create persists a new RFM group.
	Create(ctx context.Context, group *domain.RFMGroup) error
	// GetByID retrieves one RFM group by identifier.
	GetByID(ctx context.Context, id string) (*domain.RFMGroup, error)
	// GetBySlug retrieves one RFM group by slug.
	GetBySlug(ctx context.Context, slug string) (*domain.RFMGroup, error)
	// List retrieves all RFM groups (expected ≤50 rows).
	List(ctx context.Context) ([]domain.RFMGroup, error)
	// Update persists RFM group updates.
	Update(ctx context.Context, group *domain.RFMGroup) error
	// Delete removes one RFM group by identifier.
	Delete(ctx context.Context, id string) error
	// GetBandConfigs retrieves all RFM band threshold configurations.
	GetBandConfigs(ctx context.Context) ([]domain.RFMBandConfig, error)
	// UpdateBandConfig persists a single RFM band configuration.
	UpdateBandConfig(ctx context.Context, cfg domain.RFMBandConfig) error
	// SeedDefaultBands creates default R/F/M band configs when none exist.
	SeedDefaultBands(ctx context.Context) error
}

// RFMStore defines ClickHouse-backed RFM scoring behavior.
type RFMStore interface {
	// ScoreContact computes RFM scores for one contact using the provided band configs.
	ScoreContact(ctx context.Context, contactID string, bands []domain.RFMBandConfig) (*domain.RFMScore, error)
	// ScoreBatch computes RFM scores for up to 1000 contacts.
	ScoreBatch(ctx context.Context, contactIDs []string, bands []domain.RFMBandConfig) ([]domain.RFMScore, error)
	// RefreshMV truncates and repopulates the rfm_scores_mv table.
	RefreshMV(ctx context.Context) error
	// ComputeMonetaryPercentiles returns [p20, p40, p60, p80] monetary percentile thresholds.
	ComputeMonetaryPercentiles(ctx context.Context) ([4]float64, error)
}
