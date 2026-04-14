package storefront

import "time"

// Navigation defines the cached storefront rewrite tree snapshot.
type Navigation struct {
	// Realm defines the datasheet realm used to resolve names and URLs.
	Realm string `json:"realm"`
	// GeneratedAt defines when the navigation snapshot was regenerated.
	GeneratedAt time.Time `json:"generatedAt"`
	// Categories defines the ordered storefront category tree.
	Categories []CategoryNode `json:"categories"`
	// StaticPages defines the ordered static pages available in storefront navigation.
	StaticPages []StaticPageNode `json:"staticPages"`
}

// CategoryNode defines one category node in the storefront navigation tree.
type CategoryNode struct {
	// ID defines the source category identifier.
	ID string `json:"id"`
	// Name defines the source category display name.
	Name string `json:"name"`
	// Slug defines the mapped storefront slug.
	Slug string `json:"slug"`
	// Path defines the mapped storefront collection path.
	Path string `json:"path"`
	// CreatedAt defines the source category creation timestamp.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines the source category update timestamp.
	UpdatedAt time.Time `json:"updatedAt"`
	// Products defines the ordered products visible under this category path.
	Products []ProductNode `json:"products"`
	// Children defines nested child categories ordered from oldest to newest.
	Children []CategoryNode `json:"children"`
}

// ProductNode defines one product node in the storefront navigation tree.
type ProductNode struct {
	// ID defines the source product identifier.
	ID string `json:"id"`
	// SKU defines the source product SKU.
	SKU string `json:"sku"`
	// Name defines the default-realm product display name.
	Name string `json:"name"`
	// Slug defines the mapped storefront slug or configured storefronturl value.
	Slug string `json:"slug"`
	// Path defines the mapped storefront product path.
	Path string `json:"path"`
	// CreatedAt defines the source product creation timestamp.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines the source product update timestamp.
	UpdatedAt time.Time `json:"updatedAt"`
}

// StaticPageNode defines one static storefront page available in navigation.
type StaticPageNode struct {
	// ID defines the source static-page identifier.
	ID string `json:"id"`
	// RenderableID defines the bound renderable identifier.
	RenderableID string `json:"renderableId"`
	// Title defines the source static-page title.
	Title string `json:"title"`
	// URL defines the assigned storefront URL path.
	URL string `json:"url"`
	// CreatedAt defines the source static-page creation timestamp.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines the source static-page update timestamp.
	UpdatedAt time.Time `json:"updatedAt"`
}
