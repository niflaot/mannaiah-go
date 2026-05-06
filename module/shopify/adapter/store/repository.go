package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	shopifyport "mannaiah/module/shopify/port"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// ErrNilDB is returned when a nil database dependency is provided.
	ErrNilDB = errors.New("shopify store db must not be nil")
)

// Repository defines the GORM-backed Shopify persistence adapter.
type Repository struct {
	// db defines database dependencies.
	db *gorm.DB
}

type syncLinkModel struct {
	ID              string     `gorm:"column:id;primaryKey;size:32"`
	Kind            string     `gorm:"column:kind;size:32;not null;index:ux_shopify_sync_links_kind_shopify,unique;index:ux_shopify_sync_links_kind_mannaiah,unique"`
	ShopifyID       string     `gorm:"column:shopify_id;size:128;not null;index:ux_shopify_sync_links_kind_shopify,unique"`
	MannaiahID      string     `gorm:"column:mannaiah_id;size:64;not null;index:ux_shopify_sync_links_kind_mannaiah,unique"`
	LastKnownStatus string     `gorm:"column:last_known_status;size:32"`
	LastSyncedAt    *time.Time `gorm:"column:last_synced_at"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;not null"`
}

type webhookDeliveryModel struct {
	DeliveryID  string    `gorm:"column:delivery_id;primaryKey;size:255"`
	Topic       string    `gorm:"column:topic;size:255;not null"`
	ProcessedAt time.Time `gorm:"column:processed_at;not null"`
}

// TableName defines the Shopify link table name.
func (syncLinkModel) TableName() string {
	return "shopify_sync_links"
}

// TableName defines the Shopify webhook delivery table name.
func (webhookDeliveryModel) TableName() string {
	return "shopify_webhook_deliveries"
}

// NewRepository creates GORM-backed Shopify persistence adapters.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// GetLinkByShopifyID resolves one link row by Shopify identifier.
func (r *Repository) GetLinkByShopifyID(ctx context.Context, kind shopifyport.SyncKind, shopifyID string) (*shopifyport.SyncLink, error) {
	var model syncLinkModel
	err := r.db.WithContext(ctx).Where("kind = ? AND shopify_id = ?", normalizeKind(kind), strings.TrimSpace(shopifyID)).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	link := toPortLink(model)
	return &link, nil
}

// GetLinkByMannaiahID resolves one link row by Mannaiah identifier.
func (r *Repository) GetLinkByMannaiahID(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string) (*shopifyport.SyncLink, error) {
	var model syncLinkModel
	err := r.db.WithContext(ctx).Where("kind = ? AND mannaiah_id = ?", normalizeKind(kind), strings.TrimSpace(mannaiahID)).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	link := toPortLink(model)
	return &link, nil
}

// UpsertLink creates or updates one link row.
func (r *Repository) UpsertLink(ctx context.Context, input shopifyport.UpsertSyncLinkInput) (*shopifyport.SyncLink, error) {
	kind := normalizeKind(input.Kind)
	shopifyID := strings.TrimSpace(input.ShopifyID)
	mannaiahID := strings.TrimSpace(input.MannaiahID)
	status := strings.TrimSpace(input.LastKnownStatus)
	lastSyncedAt := input.LastSyncedAt
	now := time.Now().UTC()

	var model syncLinkModel
	err := r.db.WithContext(ctx).Where(
		"kind = ? AND (shopify_id = ? OR mannaiah_id = ?)",
		kind,
		shopifyID,
		mannaiahID,
	).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		model = syncLinkModel{
			ID:              newID(),
			Kind:            kind,
			ShopifyID:       shopifyID,
			MannaiahID:      mannaiahID,
			LastKnownStatus: status,
			LastSyncedAt:    normalizeTime(lastSyncedAt),
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if createErr := r.db.WithContext(ctx).Create(&model).Error; createErr != nil {
			return nil, createErr
		}

		link := toPortLink(model)
		return &link, nil
	}
	if err != nil {
		return nil, err
	}

	model.ShopifyID = shopifyID
	model.MannaiahID = mannaiahID
	model.LastKnownStatus = status
	model.LastSyncedAt = normalizeTime(lastSyncedAt)
	model.UpdatedAt = now
	if saveErr := r.db.WithContext(ctx).Save(&model).Error; saveErr != nil {
		return nil, saveErr
	}

	link := toPortLink(model)
	return &link, nil
}

// UpdateLastKnownStatus persists the last pushed outbound status for one linked aggregate.
func (r *Repository) UpdateLastKnownStatus(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string, status string) error {
	updates := map[string]any{
		"last_known_status": strings.TrimSpace(status),
		"updated_at":        time.Now().UTC(),
	}

	result := r.db.WithContext(ctx).Model(&syncLinkModel{}).Where(
		"kind = ? AND mannaiah_id = ?",
		normalizeKind(kind),
		strings.TrimSpace(mannaiahID),
	).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}

	return nil
}

// CreateDeliveryIfAbsent stores one webhook delivery id and reports whether it was new.
func (r *Repository) CreateDeliveryIfAbsent(ctx context.Context, deliveryID string, topic string) (bool, error) {
	model := webhookDeliveryModel{
		DeliveryID:  strings.TrimSpace(deliveryID),
		Topic:       strings.TrimSpace(topic),
		ProcessedAt: time.Now().UTC(),
	}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&model)
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

func normalizeKind(kind shopifyport.SyncKind) string {
	return strings.TrimSpace(string(kind))
}

func normalizeTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	resolved := value.UTC()
	return &resolved
}

func toPortLink(model syncLinkModel) shopifyport.SyncLink {
	return shopifyport.SyncLink{
		ID:              strings.TrimSpace(model.ID),
		Kind:            shopifyport.SyncKind(strings.TrimSpace(model.Kind)),
		ShopifyID:       strings.TrimSpace(model.ShopifyID),
		MannaiahID:      strings.TrimSpace(model.MannaiahID),
		LastKnownStatus: strings.TrimSpace(model.LastKnownStatus),
		LastSyncedAt:    normalizeTime(model.LastSyncedAt),
		CreatedAt:       model.CreatedAt.UTC(),
		UpdatedAt:       model.UpdatedAt.UTC(),
	}
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	}

	return hex.EncodeToString(bytes)
}
