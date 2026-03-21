package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"mannaiah/module/campaign/domain"
	"mannaiah/module/campaign/port"
)

var (
	// ErrNilDB is returned when nil db dependencies are provided.
	ErrNilDB = errors.New("campaign db must not be nil")
)

// model defines campaign persistence row values.
type model struct {
	// ID defines campaign identifier values.
	ID string `gorm:"column:id;primaryKey"`
	// Name defines campaign names.
	Name string `gorm:"column:name"`
	// Slug defines campaign slugs.
	Slug string `gorm:"column:slug"`
	// Channel defines target channel values.
	Channel string `gorm:"column:channel"`
	// SegmentID defines target segment identifier values.
	SegmentID string `gorm:"column:segment_id"`
	// Subject defines email subject values.
	Subject string `gorm:"column:subject"`
	// HTMLBody defines html content values.
	HTMLBody string `gorm:"column:html_body"`
	// TextBody defines text content values.
	TextBody string `gorm:"column:text_body"`
	// Status defines campaign status values.
	Status string `gorm:"column:status"`
	// TotalRecipients defines total resolved recipients values.
	TotalRecipients int `gorm:"column:total_recipients"`
	// SentCount defines delivered send count values.
	SentCount int `gorm:"column:sent_count"`
	// FailedCount defines failed send count values.
	FailedCount int `gorm:"column:failed_count"`
	// TemplateVars stores campaign-level custom variable values as a JSON object.
	TemplateVars string `gorm:"column:template_vars"`
	// ProductBlocks stores product recommendation block configurations as a JSON array.
	ProductBlocks string `gorm:"column:product_blocks"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `gorm:"column:created_at"`
	// UpdatedAt defines row update timestamp values.
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName resolves campaign table names.
func (model) TableName() string {
	return "campaigns"
}

// Repository defines GORM-backed campaign persistence behavior.
type Repository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

var (
	// _ ensures Repository satisfies campaign repository contracts.
	_ port.Repository = (*Repository)(nil)
)

// NewRepository creates GORM-backed campaign repositories.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// Create persists campaign rows.
func (r *Repository) Create(ctx context.Context, campaign *domain.Campaign) error {
	if campaign.ID == "" {
		campaign.ID = uuid.NewString()
	}

	templateVars, productBlocks := marshalTemplateFields(campaign)

	row := model{
		ID:              campaign.ID,
		Name:            campaign.Name,
		Slug:            campaign.Slug,
		Channel:         campaign.Channel,
		SegmentID:       campaign.SegmentID,
		Subject:         campaign.Subject,
		HTMLBody:        campaign.HTMLBody,
		TextBody:        campaign.TextBody,
		Status:          string(campaign.Status),
		TotalRecipients: campaign.TotalRecipients,
		SentCount:       campaign.SentCount,
		FailedCount:     campaign.FailedCount,
		TemplateVars:    templateVars,
		ProductBlocks:   productBlocks,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("create campaign: %w", err)
	}
	campaign.CreatedAt = row.CreatedAt.UTC()
	campaign.UpdatedAt = row.UpdatedAt.UTC()
	return nil
}

// GetByID retrieves campaign rows by id.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Campaign, error) {
	row := model{}
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select campaign by id: %w", err)
	}

	return mapModel(row), nil
}

// List retrieves campaign rows.
func (r *Repository) List(ctx context.Context, page int, limit int) ([]domain.Campaign, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	if err := r.db.WithContext(ctx).Model(&model{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count campaigns: %w", err)
	}

	rows := make([]model, 0, limit)
	if err := r.db.WithContext(ctx).Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list campaigns: %w", err)
	}

	campaigns := make([]domain.Campaign, 0, len(rows))
	for _, row := range rows {
		campaigns = append(campaigns, *mapModel(row))
	}

	return campaigns, total, nil
}

// Update persists campaign row updates.
func (r *Repository) Update(ctx context.Context, campaign *domain.Campaign) error {
	templateVars, productBlocks := marshalTemplateFields(campaign)

	updates := map[string]any{
		"name":             campaign.Name,
		"slug":             campaign.Slug,
		"channel":          campaign.Channel,
		"segment_id":       campaign.SegmentID,
		"subject":          campaign.Subject,
		"html_body":        campaign.HTMLBody,
		"text_body":        campaign.TextBody,
		"status":           string(campaign.Status),
		"total_recipients": campaign.TotalRecipients,
		"sent_count":       campaign.SentCount,
		"failed_count":     campaign.FailedCount,
		"template_vars":    templateVars,
		"product_blocks":   productBlocks,
	}
	result := r.db.WithContext(ctx).Model(&model{}).Where("id = ?", campaign.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update campaign: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	updated, err := r.GetByID(ctx, campaign.ID)
	if err != nil {
		return err
	}
	campaign.CreatedAt = updated.CreatedAt
	campaign.UpdatedAt = updated.UpdatedAt
	return nil
}

// Delete removes one campaign by id.
func (r *Repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model{})
	if result.Error != nil {
		return fmt.Errorf("delete campaign: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// mapModel maps persistence rows into domain values.
func mapModel(row model) *domain.Campaign {
	templateVars, productBlocks := unmarshalTemplateFields(row)
	return &domain.Campaign{
		ID:              row.ID,
		Name:            row.Name,
		Slug:            row.Slug,
		Channel:         row.Channel,
		SegmentID:       row.SegmentID,
		Subject:         row.Subject,
		HTMLBody:        row.HTMLBody,
		TextBody:        row.TextBody,
		Status:          domain.Status(row.Status),
		TotalRecipients: row.TotalRecipients,
		SentCount:       row.SentCount,
		FailedCount:     row.FailedCount,
		TemplateVars:    templateVars,
		ProductBlocks:   productBlocks,
		CreatedAt:       row.CreatedAt.UTC(),
		UpdatedAt:       row.UpdatedAt.UTC(),
	}
}

// marshalTemplateFields encodes TemplateVars and ProductBlocks as JSON strings for storage.
func marshalTemplateFields(campaign *domain.Campaign) (templateVars string, productBlocks string) {
	if len(campaign.TemplateVars) > 0 {
		if b, err := json.Marshal(campaign.TemplateVars); err == nil {
			templateVars = string(b)
		}
	}
	if len(campaign.ProductBlocks) > 0 {
		if b, err := json.Marshal(campaign.ProductBlocks); err == nil {
			productBlocks = string(b)
		}
	}
	return templateVars, productBlocks
}

// unmarshalTemplateFields decodes TemplateVars and ProductBlocks from JSON strings.
func unmarshalTemplateFields(row model) (templateVars map[string]string, productBlocks []domain.ProductBlock) {
	if row.TemplateVars != "" {
		_ = json.Unmarshal([]byte(row.TemplateVars), &templateVars)
	}
	if row.ProductBlocks != "" {
		_ = json.Unmarshal([]byte(row.ProductBlocks), &productBlocks)
	}
	return templateVars, productBlocks
}
