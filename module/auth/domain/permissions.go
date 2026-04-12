package domain

// Permission constants define all scope strings used across Mannaiah modules.
const (
	// Contacts
	PermContactManage = "contact:manage" // full contact administration
	PermContactView   = "contact:view"   // read-only contact access
	PermContactSync   = "contact:sync"   // exclusive sync operations

	// Orders
	PermOrderManage = "order:manage" // full order administration (archive, edit, items)
	PermOrderView   = "order:view"   // read-only order access
	PermOrderTriage = "order:triage" // change status and add/edit comments
	PermOrderSync   = "order:sync"   // exclusive sync operations

	// Products
	PermProductManage = "product:manage" // full product administration (variations, categories, falabella/jpg sync)
	PermProductEdit   = "product:edit"   // create/modify products, realms, assign photos (requires assets:view)
	PermProductView   = "product:view"   // list products
	PermProductTags   = "product:tags"   // view and edit product tags

	// Assets
	PermAssetsManage = "assets:manage" // create, delete, modify assets
	PermAssetsView   = "assets:view"   // view and download assets
	PermAssetsTags   = "assets:tags"   // tag assets

	// Marketing
	PermMarketingManage = "marketing:manage" // full marketing access (RFM, segments, campaigns, affinity)

	// Shipping
	PermShippingManage     = "shipping:manage"     // full shipping management including void
	PermShippingGenerate   = "shipping:generate"   // create batches and close them for guides
	PermShippingQuotations = "shipping:quotations" // generate quotations

	// Storefront
	PermStorefrontManage = "storefront:manage" // manage storefront-facing navigation and rewritten URL schemas
)

// PermissionCovers defines intermediate permission hierarchy.
// A key permission implies all permissions in its value slice, in addition to the manage wildcard.
// The manage wildcard (resource:manage covers resource:*) is handled separately.
var PermissionCovers = map[string][]string{
	PermProductEdit:      {PermProductView},
	PermShippingGenerate: {PermShippingQuotations},
}

// DependencyRule defines a cross-domain permission dependency.
// If a user holds any permission matching Subject, they must also satisfy all Required permissions.
type DependencyRule struct {
	// Subject is the permission that triggers the dependency check.
	Subject string
	// Required lists permissions that must be satisfied when Subject is held.
	Required []string
	// Description explains the dependency in human-readable form.
	Description string
}

// DependencyRules defines all cross-domain permission dependencies.
// These are used by the /users/malformation endpoint to detect misconfigured permission sets.
var DependencyRules = []DependencyRule{
	{
		Subject:     PermOrderView,
		Required:    []string{PermContactView},
		Description: "Order access requires contact:view (or contact:manage)",
	},
	{
		Subject:     PermOrderView,
		Required:    []string{PermProductView},
		Description: "Order access requires product:view (or product:edit/manage)",
	},
	{
		Subject:     PermOrderTriage,
		Required:    []string{PermContactView},
		Description: "Order triage requires contact:view (or contact:manage)",
	},
	{
		Subject:     PermOrderTriage,
		Required:    []string{PermProductView},
		Description: "Order triage requires product:view (or product:edit/manage)",
	},
	{
		Subject:     PermOrderManage,
		Required:    []string{PermContactView},
		Description: "Order management requires contact:view (or contact:manage)",
	},
	{
		Subject:     PermOrderManage,
		Required:    []string{PermProductView},
		Description: "Order management requires product:view (or product:edit/manage)",
	},
	{
		Subject:     PermOrderSync,
		Required:    []string{PermContactView},
		Description: "Order sync requires contact:view (or contact:manage)",
	},
	{
		Subject:     PermOrderSync,
		Required:    []string{PermProductView},
		Description: "Order sync requires product:view (or product:edit/manage)",
	},
	{
		Subject:     PermProductEdit,
		Required:    []string{PermAssetsView},
		Description: "Product edit requires assets:view to assign photos to products",
	},
	{
		Subject:     PermProductManage,
		Required:    []string{PermAssetsView},
		Description: "Product manage requires assets:view to assign images and run asset syncs",
	},
}
