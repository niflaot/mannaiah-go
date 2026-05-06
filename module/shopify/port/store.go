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
)

// SyncLink defines one persisted Shopify-to-Mannaiah link row.
type SyncLink struct {
	// ID defines link identifiers.
	ID string
	// Kind defines linked aggregate kinds.
	Kind SyncKind
	// ShopifyID defines Shopify aggregate identifiers.
	ShopifyID string
	// MannaiahID defines Mannaiah aggregate identifiers.
	MannaiahID string
	// LastKnownStatus defines the last pushed outbound status.
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
	// ShopifyID defines Shopify aggregate identifiers.
	ShopifyID string
	// MannaiahID defines Mannaiah aggregate identifiers.
	MannaiahID string
	// LastKnownStatus defines the last pushed outbound status.
	LastKnownStatus string
	// LastSyncedAt defines the latest successful synchronization timestamp.
	LastSyncedAt *time.Time
}

// SyncLinkRepository defines persisted Shopify linkage behavior.
type SyncLinkRepository interface {
	// GetLinkByShopifyID resolves one link row by Shopify identifier.
	GetLinkByShopifyID(ctx context.Context, kind SyncKind, shopifyID string) (*SyncLink, error)
	// GetLinkByMannaiahID resolves one link row by Mannaiah identifier.
	GetLinkByMannaiahID(ctx context.Context, kind SyncKind, mannaiahID string) (*SyncLink, error)
	// UpsertLink creates or updates one link row.
	UpsertLink(ctx context.Context, input UpsertSyncLinkInput) (*SyncLink, error)
	// UpdateLastKnownStatus persists the latest outbound status for one linked aggregate.
	UpdateLastKnownStatus(ctx context.Context, kind SyncKind, mannaiahID string, status string) error
}

// WebhookDeliveryRepository defines idempotency persistence for Shopify webhooks.
type WebhookDeliveryRepository interface {
	// CreateDeliveryIfAbsent stores one webhook delivery id and reports whether it was new.
	CreateDeliveryIfAbsent(ctx context.Context, deliveryID string, topic string) (bool, error)
}
