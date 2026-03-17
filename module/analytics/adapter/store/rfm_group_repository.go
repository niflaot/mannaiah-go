package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"mannaiah/module/analytics/domain"
	"mannaiah/module/analytics/port"
)

var (
	// ErrNilDB is returned when nil db dependencies are provided.
	ErrNilDB = errors.New("analytics db must not be nil")
	// ErrRFMGroupNotFound is returned when an RFM group row is not found.
	ErrRFMGroupNotFound = errors.New("rfm group not found")
)

// rfmGroupRecord defines the GORM persistence model for rfm_groups.
type rfmGroupRecord struct {
	// ID defines persistence identifier values.
	ID string `gorm:"column:id;primaryKey"`
	// Name defines human-readable group names.
	Name string `gorm:"column:name"`
	// Slug defines URL-safe group slug values.
	Slug string `gorm:"column:slug"`
	// Description defines optional group description values.
	Description string `gorm:"column:description"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `gorm:"column:created_at"`
	// UpdatedAt defines row update timestamp values.
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName returns the rfm_groups table name.
func (rfmGroupRecord) TableName() string { return "rfm_groups" }

// rfmGroupConditionRecord defines the GORM persistence model for rfm_group_conditions.
type rfmGroupConditionRecord struct {
	// ID defines persistence identifier values.
	ID      int64  `gorm:"column:id;primaryKey;autoIncrement"`
	GroupID string `gorm:"column:group_id"`
	RMin    *int   `gorm:"column:r_min"`
	RMax    *int   `gorm:"column:r_max"`
	FMin    *int   `gorm:"column:f_min"`
	FMax    *int   `gorm:"column:f_max"`
	MMin    *int   `gorm:"column:m_min"`
	MMax    *int   `gorm:"column:m_max"`
}

// TableName returns the rfm_group_conditions table name.
func (rfmGroupConditionRecord) TableName() string { return "rfm_group_conditions" }

// rfmBandConfigRecord defines the GORM persistence model for rfm_band_configs.
type rfmBandConfigRecord struct {
	// ID defines persistence identifier values.
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Dimension string    `gorm:"column:dimension"`
	Ascending bool      `gorm:"column:ascending"`
	Band5Min  float64   `gorm:"column:band5_min"`
	Band4Min  float64   `gorm:"column:band4_min"`
	Band3Min  float64   `gorm:"column:band3_min"`
	Band2Min  float64   `gorm:"column:band2_min"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName returns the rfm_band_configs table name.
func (rfmBandConfigRecord) TableName() string { return "rfm_band_configs" }

// RFMGroupRepository implements GORM-backed RFM group persistence.
type RFMGroupRepository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

var _ port.RFMGroupRepository = (*RFMGroupRepository)(nil)

// NewRFMGroupRepository creates GORM-backed RFM group repositories.
func NewRFMGroupRepository(db *gorm.DB) (*RFMGroupRepository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &RFMGroupRepository{db: db}, nil
}

// Create persists a new RFM group.
func (r *RFMGroupRepository) Create(ctx context.Context, group *domain.RFMGroup) error {
	if group.ID == "" {
		group.ID = uuid.NewString()
	}
	rec := rfmGroupRecord{
		ID:          group.ID,
		Name:        group.Name,
		Slug:        group.Slug,
		Description: group.Description,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return fmt.Errorf("create rfm group: %w", err)
	}

	group.CreatedAt = rec.CreatedAt.UTC()
	group.UpdatedAt = rec.UpdatedAt.UTC()

	return r.upsertConditions(ctx, group)
}

// GetByID retrieves one RFM group by identifier.
func (r *RFMGroupRepository) GetByID(ctx context.Context, id string) (*domain.RFMGroup, error) {
	rec := rfmGroupRecord{}
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRFMGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get rfm group by id: %w", err)
	}

	return r.mapWithConditions(ctx, rec)
}

// GetBySlug retrieves one RFM group by slug.
func (r *RFMGroupRepository) GetBySlug(ctx context.Context, slug string) (*domain.RFMGroup, error) {
	rec := rfmGroupRecord{}
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRFMGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get rfm group by slug: %w", err)
	}

	return r.mapWithConditions(ctx, rec)
}

// List retrieves all RFM groups.
func (r *RFMGroupRepository) List(ctx context.Context) ([]domain.RFMGroup, error) {
	recs := make([]rfmGroupRecord, 0)
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&recs).Error; err != nil {
		return nil, fmt.Errorf("list rfm groups: %w", err)
	}

	groups := make([]domain.RFMGroup, 0, len(recs))
	for _, rec := range recs {
		g, err := r.mapWithConditions(ctx, rec)
		if err != nil {
			return nil, err
		}
		groups = append(groups, *g)
	}

	return groups, nil
}

// Update persists RFM group updates.
func (r *RFMGroupRepository) Update(ctx context.Context, group *domain.RFMGroup) error {
	updates := map[string]any{
		"name":        group.Name,
		"slug":        group.Slug,
		"description": group.Description,
	}
	result := r.db.WithContext(ctx).Model(&rfmGroupRecord{}).Where("id = ?", group.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update rfm group: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrRFMGroupNotFound
	}

	return r.upsertConditions(ctx, group)
}

// Delete removes one RFM group by identifier.
func (r *RFMGroupRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("group_id = ?", id).Delete(&rfmGroupConditionRecord{}).Error; err != nil {
		return fmt.Errorf("delete rfm group conditions: %w", err)
	}
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&rfmGroupRecord{})
	if result.Error != nil {
		return fmt.Errorf("delete rfm group: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrRFMGroupNotFound
	}

	return nil
}

// GetBandConfigs retrieves all RFM band threshold configurations.
func (r *RFMGroupRepository) GetBandConfigs(ctx context.Context) ([]domain.RFMBandConfig, error) {
	recs := make([]rfmBandConfigRecord, 0)
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&recs).Error; err != nil {
		return nil, fmt.Errorf("get rfm band configs: %w", err)
	}

	result := make([]domain.RFMBandConfig, 0, len(recs))
	for _, rec := range recs {
		result = append(result, domain.RFMBandConfig{
			ID:        rec.ID,
			Dimension: domain.RFMDimension(rec.Dimension),
			Ascending: rec.Ascending,
			Band5Min:  rec.Band5Min,
			Band4Min:  rec.Band4Min,
			Band3Min:  rec.Band3Min,
			Band2Min:  rec.Band2Min,
			UpdatedAt: rec.UpdatedAt.UTC(),
		})
	}

	return result, nil
}

// UpdateBandConfig persists a single RFM band configuration.
func (r *RFMGroupRepository) UpdateBandConfig(ctx context.Context, cfg domain.RFMBandConfig) error {
	updates := map[string]any{
		"ascending":  cfg.Ascending,
		"band5_min":  cfg.Band5Min,
		"band4_min":  cfg.Band4Min,
		"band3_min":  cfg.Band3Min,
		"band2_min":  cfg.Band2Min,
		"updated_at": time.Now().UTC(),
	}
	result := r.db.WithContext(ctx).Model(&rfmBandConfigRecord{}).Where("dimension = ?", string(cfg.Dimension)).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update rfm band config: %w", result.Error)
	}

	return nil
}

// SeedDefaultBands creates default R/F/M band configs when none exist.
func (r *RFMGroupRepository) SeedDefaultBands(ctx context.Context) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&rfmBandConfigRecord{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count rfm band configs: %w", err)
	}
	if count > 0 {
		return nil
	}

	now := time.Now().UTC()
	defaults := []rfmBandConfigRecord{
		{Dimension: "recency", Ascending: false, Band5Min: 7, Band4Min: 30, Band3Min: 90, Band2Min: 180, UpdatedAt: now},
		{Dimension: "frequency", Ascending: true, Band5Min: 10, Band4Min: 6, Band3Min: 3, Band2Min: 2, UpdatedAt: now},
		{Dimension: "monetary", Ascending: true, Band5Min: 0, Band4Min: 0, Band3Min: 0, Band2Min: 0, UpdatedAt: now},
	}
	if err := r.db.WithContext(ctx).Create(&defaults).Error; err != nil {
		return fmt.Errorf("seed default rfm band configs: %w", err)
	}

	return nil
}

// upsertConditions replaces RFM group conditions for the given group.
func (r *RFMGroupRepository) upsertConditions(ctx context.Context, group *domain.RFMGroup) error {
	if err := r.db.WithContext(ctx).Where("group_id = ?", group.ID).Delete(&rfmGroupConditionRecord{}).Error; err != nil {
		return fmt.Errorf("delete rfm group conditions: %w", err)
	}

	cond := group.Conditions
	hasAny := cond.RMin != nil || cond.RMax != nil || cond.FMin != nil ||
		cond.FMax != nil || cond.MMin != nil || cond.MMax != nil
	if !hasAny {
		return nil
	}

	mMin, mMax := condIntToIntPtr(cond.MMin), condIntToIntPtr(cond.MMax)
	rec := rfmGroupConditionRecord{
		GroupID: group.ID,
		RMin:    cond.RMin,
		RMax:    cond.RMax,
		FMin:    cond.FMin,
		FMax:    cond.FMax,
		MMin:    mMin,
		MMax:    mMax,
	}
	if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
		return fmt.Errorf("create rfm group conditions: %w", err)
	}

	return nil
}

// mapWithConditions maps a record row into a domain group, loading conditions.
func (r *RFMGroupRepository) mapWithConditions(ctx context.Context, rec rfmGroupRecord) (*domain.RFMGroup, error) {
	condRec := rfmGroupConditionRecord{}
	err := r.db.WithContext(ctx).Where("group_id = ?", rec.ID).First(&condRec).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("get rfm group conditions: %w", err)
	}

	var cond domain.RFMGroupConditions
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		cond = domain.RFMGroupConditions{
			RMin: condRec.RMin,
			RMax: condRec.RMax,
			FMin: condRec.FMin,
			FMax: condRec.FMax,
			MMin: condRec.MMin,
			MMax: condRec.MMax,
		}
	}

	return &domain.RFMGroup{
		ID:          rec.ID,
		Name:        rec.Name,
		Slug:        rec.Slug,
		Description: rec.Description,
		Conditions:  cond,
		CreatedAt:   rec.CreatedAt.UTC(),
		UpdatedAt:   rec.UpdatedAt.UTC(),
	}, nil
}

// condIntToIntPtr converts a *int RFM condition to an *int (pass-through helper for M-score).
func condIntToIntPtr(v *int) *int { return v }
