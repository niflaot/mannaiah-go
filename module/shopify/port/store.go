package port

import (
	"context"
	"time"
)

// SyncKind defines supported Shopify linkage aggregate kinds.
type SyncKind string

const (
	// SyncKindContact defines customer/contact link rows.
	SyncKindContact SyncKind = "contact"
	// SyncKindOrder defines order link rows.
	SyncKindOrder SyncKind = "order"
	// SyncKindProduct defines product link rows.
	SyncKindProduct SyncKind = "product"
	// SyncKindVariant defines product-variant link rows.
	SyncKindVariant SyncKind = "variant"
	// SyncKindFulfillment defines shipping-mark to Shopify fulfillment link rows.
	SyncKindFulfillment SyncKind = "fulfillment"
)

// SyncLink defines one persisted Shopify-to-Mannaiah link row.
type SyncLink struct {
	// ID defines link identifiers.
	ID string
	// Kind defines linked aggregate kinds.
	Kind SyncKind
	// ShopDomain defines the Shopify store domain associated with the link.
	ShopDomain string
	// ShopifyID defines Shopify aggregate identifiers.
	ShopifyID string
	// MannaiahID defines Mannaiah aggregate identifiers.
	MannaiahID string
	// LastKnownStatus defines the latest imported status for linked aggregates.
	LastKnownStatus string
	// LastSyncedAt defines the latest successful synchronization timestamp.
	LastSyncedAt *time.Time
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
}

// UpsertSyncLinkInput defines link persistence payload values.
type UpsertSyncLinkInput struct {
	// Kind defines linked aggregate kinds.
	Kind SyncKind
	// ShopDomain defines the Shopify store domain associated with the link.
	ShopDomain string
	// ShopifyID defines Shopify aggregate identifiers.
	ShopifyID string
	// MannaiahID defines Mannaiah aggregate identifiers.
	MannaiahID string
	// LastKnownStatus defines the latest imported status for linked aggregates.
	LastKnownStatus string
	// LastSyncedAt defines the latest successful synchronization timestamp.
	LastSyncedAt *time.Time
}

// SyncLinkRepository defines persisted Shopify linkage behavior.
type SyncLinkRepository interface {
	// GetLinkByShopifyID resolves one link row by Shopify identifier.
	GetLinkByShopifyID(ctx context.Context, kind SyncKind, shopDomain string, shopifyID string) (*SyncLink, error)
	// GetLinkByMannaiahID resolves one link row by Mannaiah identifier.
	GetLinkByMannaiahID(ctx context.Context, kind SyncKind, mannaiahID string) (*SyncLink, error)
	// UpsertLink creates or updates one link row.
	UpsertLink(ctx context.Context, input UpsertSyncLinkInput) (*SyncLink, error)
	// UpdateLastKnownStatus persists the latest imported status for one linked aggregate.
	UpdateLastKnownStatus(ctx context.Context, kind SyncKind, mannaiahID string, status string) error
}

// WebhookDeliveryRepository defines idempotency persistence for Shopify webhooks.
type WebhookDeliveryRepository interface {
	// CreateDeliveryIfAbsent stores one webhook delivery id and reports whether it was new.
	CreateDeliveryIfAbsent(ctx context.Context, deliveryID string, topic string) (bool, error)
}

// Installation defines one persisted Shopify app installation.
type Installation struct {
	// ID defines installation identifiers.
	ID string
	// ShopDomain defines Shopify store domains.
	ShopDomain string
	// AccessToken defines permanent Shopify offline access tokens.
	AccessToken string
	// Scopes defines granted installation scopes.
	Scopes string
	// InstalledAt defines installation timestamps.
	InstalledAt time.Time
	// UninstalledAt defines uninstall timestamps.
	UninstalledAt *time.Time
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
}

// UpsertInstallationInput defines persisted Shopify installation payload values.
type UpsertInstallationInput struct {
	// ShopDomain defines Shopify store domains.
	ShopDomain string
	// AccessToken defines permanent Shopify offline access tokens.
	AccessToken string
	// Scopes defines granted installation scopes.
	Scopes string
	// InstalledAt defines installation timestamps.
	InstalledAt time.Time
	// UninstalledAt defines uninstall timestamps.
	UninstalledAt *time.Time
}

// InstallationRepository defines persisted Shopify installation behavior.
type InstallationRepository interface {
	// UpsertInstallation creates or updates one installation row.
	UpsertInstallation(ctx context.Context, input UpsertInstallationInput) (*Installation, error)
	// FindByShopDomain resolves one installation row by Shopify store domain.
	FindByShopDomain(ctx context.Context, shopDomain string) (*Installation, error)
	// ListActive lists non-uninstalled Shopify installations.
	ListActive(ctx context.Context) ([]Installation, error)
	// MarkUninstalled sets uninstall timestamps for one Shopify installation.
	MarkUninstalled(ctx context.Context, shopDomain string, uninstalledAt time.Time) error
}
