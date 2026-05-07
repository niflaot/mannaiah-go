package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"mannaiah/module/exports/domain"
	"mannaiah/module/exports/port"
)

var (
	// ErrNilDB is returned when nil database dependencies are provided.
	ErrNilDB = errors.New("exports db must not be nil")
)

// Repository defines GORM-backed export report registry behavior.
type Repository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

var (
	// _ ensures Repository satisfies export repository ports.
	_ port.Repository = (*Repository)(nil)
)

// NewRepository creates GORM-backed export report repositories.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// Create persists a generated export report.
func (r *Repository) Create(ctx context.Context, report *domain.Report) error {
	if report == nil {
		return domain.ErrInvalidReportID
	}
	report.Normalize()
	if err := report.Validate(); err != nil {
		return err
	}

	model := mapReportToModel(report)
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("insert export report: %w", err)
	}

	return nil
}

// GetByID retrieves a generated export report by id.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Report, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, domain.ErrInvalidReportID
	}

	model := reportModel{}
	err := r.db.WithContext(ctx).Where("id = ?", trimmedID).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrReportNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select export report: %w", err)
	}

	report := mapModelToReport(model)
	return &report, nil
}

// List returns paginated generated export reports.
func (r *Repository) List(ctx context.Context, query port.ListQuery) ([]domain.Report, int64, error) {
	db := r.db.WithContext(ctx).Model(&reportModel{})
	if query.Type != "" {
		db = db.Where("report_type = ?", string(query.Type.Normalize()))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count export reports: %w", err)
	}
	if total == 0 {
		return []domain.Report{}, 0, nil
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	models := make([]reportModel, 0, limit)
	if err := db.Order("generated_at DESC, id DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list export reports: %w", err)
	}

	reports := make([]domain.Report, 0, len(models))
	for _, model := range models {
		reports = append(reports, mapModelToReport(model))
	}

	return reports, total, nil
}
