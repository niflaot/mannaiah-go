package domain

import "strings"

// SyncKind defines the type of synchronization operation.
type SyncKind string

const (
	// KindWooCommerceContacts defines WooCommerce contact sync runs.
	KindWooCommerceContacts SyncKind = "woocommerce.contacts"
	// KindWooCommerceCoupons defines WooCommerce coupon sync runs.
	KindWooCommerceCoupons SyncKind = "woocommerce.coupons"
	// KindWooCommerceOrders defines WooCommerce order sync runs.
	KindWooCommerceOrders SyncKind = "woocommerce.orders"
	// KindFalabellaProducts defines Falabella product sync runs.
	KindFalabellaProducts SyncKind = "falabella.products"
	// KindFalabellaStatusResolution defines Falabella status-resolution runs.
	KindFalabellaStatusResolution SyncKind = "falabella.status_resolution"
	// KindAssetsJPGConversion defines assets JPG worker runs.
	KindAssetsJPGConversion SyncKind = "assets.jpg_conversion"
	// KindAnalyticsSeed defines analytics seeding runs.
	KindAnalyticsSeed SyncKind = "analytics.seed"
	// KindAnalyticsFlush defines analytics flush runs.
	KindAnalyticsFlush SyncKind = "analytics.flush"
	// KindMembershipMigration defines membership migration runs.
	KindMembershipMigration SyncKind = "membership.migration"
	// KindCampaignSend defines campaign send runs.
	KindCampaignSend SyncKind = "campaign.send"
	// KindShopifyContacts defines Shopify contact sync runs.
	KindShopifyContacts SyncKind = "shopify.contacts"
	// KindShopifyOrders defines Shopify order sync runs.
	KindShopifyOrders SyncKind = "shopify.orders"
)

// IsValid reports whether a sync kind is recognized.
func (k SyncKind) IsValid() bool {
	switch SyncKind(strings.TrimSpace(string(k))) {
	case KindWooCommerceContacts,
		KindWooCommerceCoupons,
		KindWooCommerceOrders,
		KindFalabellaProducts,
		KindFalabellaStatusResolution,
		KindAssetsJPGConversion,
		KindAnalyticsSeed,
		KindAnalyticsFlush,
		KindMembershipMigration,
		KindCampaignSend,
		KindShopifyContacts,
		KindShopifyOrders:
		return true
	default:
		return false
	}
}

// Module returns the module segment of the sync kind.
func (k SyncKind) Module() string {
	parts := strings.SplitN(strings.TrimSpace(string(k)), ".", 2)
	if len(parts) == 0 {
		return ""
	}

	return strings.TrimSpace(parts[0])
}
