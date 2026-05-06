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
	ShopDomain      string     `gorm:"column:shop_domain;size:255;not null;default:'';index:idx_shopify_sync_links_shop_domain;index:ux_shopify_sync_links_kind_shopify,unique"`
	ShopifyID       string     `gorm:"column:shopify_id;size:128;not null;index:ux_shopify_sync_links_kind_shopify,unique"`
	MannaiahID      string     `gorm:"column:mannaiah_id;size:64;not null;index:ux_shopify_sync_links_kind_mannaiah,unique"`
	LastKnownStatus string     `gorm:"column:last_known_status;size:32"`
	LastSyncedAt    *time.Time `gorm:"column:last_synced_at"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;not null"`
}

type installationModel struct {
	ID            string     `gorm:"column:id;primaryKey;size:32"`
	ShopDomain    string     `gorm:"column:shop_domain;size:255;not null;uniqueIndex:idx_shopify_installations_shop_domain"`
	AccessToken   string     `gorm:"column:access_token;size:255;not null"`
	Scopes        string     `gorm:"column:scopes;size:500;not null"`
	InstalledAt   time.Time  `gorm:"column:installed_at;not null"`
	UninstalledAt *time.Time `gorm:"column:uninstalled_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;not null"`
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

// TableName defines the Shopify installation table name.
func (installationModel) TableName() string {
	return "shopify_installations"
}

// NewRepository creates GORM-backed Shopify persistence adapters.
func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &Repository{db: db}, nil
}

// GetLinkByShopifyID resolves one link row by Shopify identifier.
func (r *Repository) GetLinkByShopifyID(ctx context.Context, kind shopifyport.SyncKind, shopDomain string, shopifyID string) (*shopifyport.SyncLink, error) {
	resolvedShopDomain := shopifyport.NormalizeShopDomain(shopDomain)
	resolvedShopifyID := strings.TrimSpace(shopifyID)

	var model syncLinkModel
	err := r.db.WithContext(ctx).Where(
		"kind = ? AND shop_domain = ? AND shopify_id = ?",
		normalizeKind(kind),
		resolvedShopDomain,
		resolvedShopifyID,
	).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) && resolvedShopDomain != "" {
		err = r.db.WithContext(ctx).Where(
			"kind = ? AND shop_domain = '' AND shopify_id = ?",
			normalizeKind(kind),
			resolvedShopifyID,
		).First(&model).Error
	}
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
	shopDomain := shopifyport.NormalizeShopDomain(input.ShopDomain)
	shopifyID := strings.TrimSpace(input.ShopifyID)
	mannaiahID := strings.TrimSpace(input.MannaiahID)
	status := strings.TrimSpace(input.LastKnownStatus)
	lastSyncedAt := input.LastSyncedAt
	now := time.Now().UTC()

	var model syncLinkModel
	err := r.db.WithContext(ctx).Where(
		"kind = ? AND ((shop_domain = ? AND shopify_id = ?) OR (shop_domain = '' AND shopify_id = ?) OR mannaiah_id = ?)",
		kind,
		shopDomain,
		shopifyID,
		shopifyID,
		mannaiahID,
	).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		model = syncLinkModel{
			ID:              newID(),
			Kind:            kind,
			ShopDomain:      shopDomain,
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

	model.ShopDomain = shopDomain
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

// UpsertInstallation creates or updates one Shopify installation row.
func (r *Repository) UpsertInstallation(ctx context.Context, input shopifyport.UpsertInstallationInput) (*shopifyport.Installation, error) {
	now := time.Now().UTC()
	shopDomain := shopifyport.NormalizeShopDomain(input.ShopDomain)
	installedAt := input.InstalledAt.UTC()
	if installedAt.IsZero() {
		installedAt = now
	}
	model := installationModel{
		ID:            newID(),
		ShopDomain:    shopDomain,
		AccessToken:   strings.TrimSpace(input.AccessToken),
		Scopes:        strings.TrimSpace(input.Scopes),
		InstalledAt:   installedAt,
		UninstalledAt: normalizeTime(input.UninstalledAt),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "shop_domain"}},
		DoUpdates: clause.Assignments(map[string]any{
			"access_token":   model.AccessToken,
			"scopes":         model.Scopes,
			"installed_at":   model.InstalledAt,
			"uninstalled_at": model.UninstalledAt,
			"updated_at":     now,
		}),
	}).Create(&model)
	if result.Error != nil {
		return nil, result.Error
	}

	return r.FindByShopDomain(ctx, shopDomain)
}

// FindByShopDomain resolves one Shopify installation row by store domain.
func (r *Repository) FindByShopDomain(ctx context.Context, shopDomain string) (*shopifyport.Installation, error) {
	var model installationModel
	err := r.db.WithContext(ctx).Where("shop_domain = ?", shopifyport.NormalizeShopDomain(shopDomain)).First(&model).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	installation := toPortInstallation(model)
	return &installation, nil
}

// ListActive lists Shopify installations that have not been uninstalled.
func (r *Repository) ListActive(ctx context.Context) ([]shopifyport.Installation, error) {
	var models []installationModel
	err := r.db.WithContext(ctx).Where("uninstalled_at IS NULL").Order("shop_domain ASC").Find(&models).Error
	if err != nil {
		return nil, err
	}

	installations := make([]shopifyport.Installation, 0, len(models))
	for _, model := range models {
		installations = append(installations, toPortInstallation(model))
	}

	return installations, nil
}

// MarkUninstalled sets uninstall timestamps for one Shopify installation row.
func (r *Repository) MarkUninstalled(ctx context.Context, shopDomain string, uninstalledAt time.Time) error {
	resolved := uninstalledAt.UTC()
	if resolved.IsZero() {
		resolved = time.Now().UTC()
	}

	return r.db.WithContext(ctx).Model(&installationModel{}).Where(
		"shop_domain = ?",
		shopifyport.NormalizeShopDomain(shopDomain),
	).Updates(map[string]any{
		"uninstalled_at": &resolved,
		"updated_at":     time.Now().UTC(),
	}).Error
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
		ShopDomain:      shopifyport.NormalizeShopDomain(model.ShopDomain),
		ShopifyID:       strings.TrimSpace(model.ShopifyID),
		MannaiahID:      strings.TrimSpace(model.MannaiahID),
		LastKnownStatus: strings.TrimSpace(model.LastKnownStatus),
		LastSyncedAt:    normalizeTime(model.LastSyncedAt),
		CreatedAt:       model.CreatedAt.UTC(),
		UpdatedAt:       model.UpdatedAt.UTC(),
	}
}

func toPortInstallation(model installationModel) shopifyport.Installation {
	return shopifyport.Installation{
		ID:            strings.TrimSpace(model.ID),
		ShopDomain:    shopifyport.NormalizeShopDomain(model.ShopDomain),
		AccessToken:   strings.TrimSpace(model.AccessToken),
		Scopes:        strings.TrimSpace(model.Scopes),
		InstalledAt:   model.InstalledAt.UTC(),
		UninstalledAt: normalizeTime(model.UninstalledAt),
		CreatedAt:     model.CreatedAt.UTC(),
		UpdatedAt:     model.UpdatedAt.UTC(),
	}
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	}

	return hex.EncodeToString(bytes)
}
