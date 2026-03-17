package runtime

import (
	"context"

	"mannaiah/module/analytics/domain"
	"mannaiah/module/analytics/port"
)

// noopRFMStore is a no-op RFM store used when ClickHouse is not configured.
type noopRFMStore struct{}

var _ port.RFMStore = (*noopRFMStore)(nil)

func (n *noopRFMStore) ScoreContact(_ context.Context, _ string, _ []domain.RFMBandConfig) (*domain.RFMScore, error) {
	return nil, nil
}

func (n *noopRFMStore) ScoreBatch(_ context.Context, _ []string, _ []domain.RFMBandConfig) ([]domain.RFMScore, error) {
	return nil, nil
}

func (n *noopRFMStore) RefreshMV(_ context.Context) error {
	return nil
}

func (n *noopRFMStore) ComputeMonetaryPercentiles(_ context.Context) ([4]float64, error) {
	return [4]float64{}, nil
}

// noopAffinityStore is a no-op affinity store used when ClickHouse is not configured.
type noopAffinityStore struct{}

var _ port.AffinityStore = (*noopAffinityStore)(nil)

func (n *noopAffinityStore) GetTagAffinity(_ context.Context, _ string, _ int, _ float64) ([]domain.TagAffinity, error) {
	return nil, nil
}

func (n *noopAffinityStore) GetCategoryAffinity(_ context.Context, _ string, _ int, _ float64) ([]domain.CategoryAffinity, error) {
	return nil, nil
}

func (n *noopAffinityStore) GetVariationAffinity(_ context.Context, _ string, _ int, _ float64) ([]domain.VariationAffinity, error) {
	return nil, nil
}

func (n *noopAffinityStore) GetProfile(_ context.Context, contactID string, _ int, _ float64) (*domain.AffinityProfile, error) {
	return &domain.AffinityProfile{ContactID: contactID}, nil
}

func (n *noopAffinityStore) RefreshTagMV(_ context.Context) error      { return nil }
func (n *noopAffinityStore) RefreshCategoryMV(_ context.Context) error { return nil }
func (n *noopAffinityStore) RefreshVariationMV(_ context.Context) error { return nil }
