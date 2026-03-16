package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"mannaiah/module/email/domain"
	"mannaiah/module/email/port"
)

var (
	// ErrNilDB is returned when nil db dependencies are provided.
	ErrNilDB = errors.New("email db must not be nil")
)

// deliveryModel defines delivery persistence row values.
type deliveryModel struct {
	// ID defines delivery identifier values.
	ID string `gorm:"column:id;primaryKey"`
	// ContactID defines optional contact identifier values.
	ContactID string `gorm:"column:contact_id"`
	// Email defines recipient email values.
	Email string `gorm:"column:email"`
	// Subject defines subject values.
	Subject string `gorm:"column:subject"`
	// HTMLBody defines html payload values.
	HTMLBody string `gorm:"column:html_body"`
	// TextBody defines text payload values.
	TextBody string `gorm:"column:text_body"`
	// IdempotencyKey defines idempotency values.
	IdempotencyKey string `gorm:"column:idempotency_key"`
	// Provider defines provider labels.
	Provider string `gorm:"column:provider"`
	// ProviderMessageID defines provider message id values.
	ProviderMessageID string `gorm:"column:provider_message_id"`
	// Status defines current delivery status values.
	Status string `gorm:"column:status"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `gorm:"column:created_at"`
	// UpdatedAt defines row update timestamp values.
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName resolves delivery table names.
func (deliveryModel) TableName() string {
	return "email_deliveries"
}

// statusEntryModel defines delivery status entry persistence row values.
type statusEntryModel struct {
	// ID defines status entry identifier values.
	ID string `gorm:"column:id;primaryKey"`
	// DeliveryID defines parent delivery identifier values.
	DeliveryID string `gorm:"column:delivery_id"`
	// Status defines status values.
	Status string `gorm:"column:status"`
	// Reason defines optional reason values.
	Reason string `gorm:"column:reason"`
	// OccurredAt defines status timestamps.
	OccurredAt time.Time `gorm:"column:occurred_at"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `gorm:"column:created_at"`
}

// TableName resolves status history table names.
func (statusEntryModel) TableName() string {
	return "email_delivery_status_history"
}

// Repository defines GORM-backed email delivery persistence behavior.
type Repository struct {
	// db defines GORM database dependencies.
	db *gorm.DB
}

var (
	// _ ensures Repository satisfies email repository contracts.
	_ port.Repository = (*Repository)(nil)
)

// NewRepository creates GORM-backed email repositories.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// CreateDelivery persists delivery rows.
func (r *Repository) CreateDelivery(ctx context.Context, delivery *domain.Delivery) error {
	row := deliveryModel{
		ID:                delivery.ID,
		ContactID:         delivery.ContactID,
		Email:             delivery.Email,
		Subject:           delivery.Subject,
		HTMLBody:          delivery.HTMLBody,
		TextBody:          delivery.TextBody,
		IdempotencyKey:    delivery.IdempotencyKey,
		Provider:          delivery.Provider,
		ProviderMessageID: delivery.ProviderMessageID,
		Status:            string(delivery.Status),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("create email delivery: %w", err)
	}
	return nil
}

// UpdateDeliveryStatus updates current delivery status values.
func (r *Repository) UpdateDeliveryStatus(ctx context.Context, deliveryID string, status domain.DeliveryStatus, providerMessageID string) error {
	updates := map[string]any{
		"status": string(status),
	}
	if providerMessageID != "" {
		updates["provider_message_id"] = providerMessageID
	}
	result := r.db.WithContext(ctx).Model(&deliveryModel{}).Where("id = ?", deliveryID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update email delivery status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// AddStatusEntry persists immutable status timeline rows.
func (r *Repository) AddStatusEntry(ctx context.Context, entry *domain.StatusEntry) error {
	row := statusEntryModel{
		ID:         entry.ID,
		DeliveryID: entry.DeliveryID,
		Status:     string(entry.Status),
		Reason:     entry.Reason,
		OccurredAt: entry.OccurredAt.UTC(),
		CreatedAt:  entry.CreatedAt.UTC(),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("insert email status entry: %w", err)
	}

	return nil
}

// GetByID retrieves delivery rows by id.
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Delivery, error) {
	row := deliveryModel{}
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select email delivery by id: %w", err)
	}

	return mapDelivery(row), nil
}

// GetByProviderMessageID retrieves delivery rows by provider message id.
func (r *Repository) GetByProviderMessageID(ctx context.Context, providerMessageID string) (*domain.Delivery, error) {
	row := deliveryModel{}
	err := r.db.WithContext(ctx).Where("provider_message_id = ?", providerMessageID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select email delivery by provider id: %w", err)
	}

	return mapDelivery(row), nil
}

// ListByCampaignID retrieves paginated delivery rows for a campaign by idempotency key prefix.
func (r *Repository) ListByCampaignID(ctx context.Context, campaignID string, page int, limit int) ([]*domain.Delivery, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}

	prefix := strings.TrimSpace(campaignID) + ":%"

	var total int64
	if err := r.db.WithContext(ctx).Model(&deliveryModel{}).
		Where("idempotency_key LIKE ?", prefix).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count campaign deliveries: %w", err)
	}
	if total == 0 {
		return []*domain.Delivery{}, 0, nil
	}

	rows := make([]deliveryModel, 0, limit)
	if err := r.db.WithContext(ctx).
		Where("idempotency_key LIKE ?", prefix).
		Order("created_at ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list campaign deliveries: %w", err)
	}

	result := make([]*domain.Delivery, 0, len(rows))
	for _, row := range rows {
		result = append(result, mapDelivery(row))
	}

	return result, total, nil
}

// mapDelivery maps persistence rows into domain values.
func mapDelivery(row deliveryModel) *domain.Delivery {
	return &domain.Delivery{
		ID:                row.ID,
		ContactID:         row.ContactID,
		Email:             row.Email,
		Subject:           row.Subject,
		HTMLBody:          row.HTMLBody,
		TextBody:          row.TextBody,
		IdempotencyKey:    row.IdempotencyKey,
		Provider:          row.Provider,
		ProviderMessageID: row.ProviderMessageID,
		Status:            domain.DeliveryStatus(row.Status),
		CreatedAt:         row.CreatedAt.UTC(),
		UpdatedAt:         row.UpdatedAt.UTC(),
	}
}
