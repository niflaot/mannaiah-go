package category

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrSlugRequired is returned when category slug is missing.
	ErrSlugRequired = errors.New("category slug is required")
	// ErrNameRequired is returned when category name is missing.
	ErrNameRequired = errors.New("category name is required")
	// ErrCircularParent is returned when a category references itself as parent.
	ErrCircularParent = errors.New("category cannot be its own parent")
)

// PriceRange defines optional price range filter values.
type PriceRange struct {
	// Min defines optional minimum price bound.
	Min *float64 `json:"min,omitempty"`
	// Max defines optional maximum price bound.
	Max *float64 `json:"max,omitempty"`
}

// Filter defines category membership criteria.
type Filter struct {
	// Tags defines tag values products must have to match this category.
	Tags []string `json:"tags,omitempty"`
	// PriceRange defines optional price range filter values.
	PriceRange *PriceRange `json:"priceRange,omitempty"`
	// CategoryRefs defines category IDs whose products are also included.
	CategoryRefs []string `json:"categoryRefs,omitempty"`
}

// Category defines the category aggregate root.
type Category struct {
	// ID defines unique category identifiers.
	ID string `json:"id"`
	// Slug defines URL-friendly unique category slugs.
	Slug string `json:"slug"`
	// Name defines human-readable category names.
	Name string `json:"name"`
	// Description defines optional category description values.
	Description string `json:"description,omitempty"`
	// ParentID defines optional parent category identifiers.
	ParentID *string `json:"parentId,omitempty"`
	// IncludeChildren reports whether descendant categories are included in product resolution.
	IncludeChildren bool `json:"includeChildren"`
	// Filter defines product membership filter criteria.
	Filter Filter `json:"filter"`
	// ProductIDs defines manually pinned product identifiers.
	ProductIDs []string `json:"productIds,omitempty"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}

// Normalize canonicalizes category values before validation/persistence.
func (c *Category) Normalize() {
	if c == nil {
		return
	}

	c.Slug = strings.TrimSpace(c.Slug)
	c.Name = strings.TrimSpace(c.Name)
	c.Description = strings.TrimSpace(c.Description)

	for i := range c.Filter.Tags {
		c.Filter.Tags[i] = strings.TrimSpace(c.Filter.Tags[i])
	}
	for i := range c.Filter.CategoryRefs {
		c.Filter.CategoryRefs[i] = strings.TrimSpace(c.Filter.CategoryRefs[i])
	}
	for i := range c.ProductIDs {
		c.ProductIDs[i] = strings.TrimSpace(c.ProductIDs[i])
	}
}

// Validate verifies category invariants.
func (c Category) Validate() error {
	if strings.TrimSpace(c.Slug) == "" {
		return ErrSlugRequired
	}
	if strings.TrimSpace(c.Name) == "" {
		return ErrNameRequired
	}
	if c.ParentID != nil && strings.TrimSpace(*c.ParentID) == c.ID {
		return ErrCircularParent
	}

	return nil
}
