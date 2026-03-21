package store

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"mannaiah/module/analytics/port"
)

// TagCorrelationRepository implements GORM-backed tag correlation read behavior.
type TagCorrelationRepository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

// tagCorrelationRecord defines the GORM scan model for tag_correlations rows.
type tagCorrelationRecord struct {
	// TargetTag is the correlated product tag.
	TargetTag string `gorm:"column:target_tag"`
	// Probability is the configured cross-sell probability.
	Probability float64 `gorm:"column:probability"`
}

// NewTagCorrelationRepository creates GORM-backed tag correlation repositories.
func NewTagCorrelationRepository(db *gorm.DB) (*TagCorrelationRepository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &TagCorrelationRepository{db: db}, nil
}

// GetCorrelations returns all target tags correlated to any of the given source tags.
func (r *TagCorrelationRepository) GetCorrelations(ctx context.Context, sourceTags []string) ([]port.TagCorrelation, error) {
	if len(sourceTags) == 0 {
		return nil, nil
	}

	rows := make([]tagCorrelationRecord, 0, len(sourceTags)*3)
	if err := r.db.WithContext(ctx).
		Table("tag_correlations").
		Select("target_tag, probability").
		Where("source_tag IN ? AND probability > 0", sourceTags).
		Order("probability DESC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("get tag correlations: %w", err)
	}

	result := make([]port.TagCorrelation, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seen[row.TargetTag]; ok {
			continue
		}
		seen[row.TargetTag] = struct{}{}
		result = append(result, port.TagCorrelation{
			TargetTag:   row.TargetTag,
			Probability: row.Probability,
		})
	}

	return result, nil
}
