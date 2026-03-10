package product

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrSKURequired is returned when product SKU is missing.
	ErrSKURequired = errors.New("product sku is required")
	// ErrGalleryAssetIDRequired is returned when gallery items have missing asset IDs.
	ErrGalleryAssetIDRequired = errors.New("product gallery assetId is required")
	// ErrDatasheetRealmRequired is returned when datasheets have missing realms.
	ErrDatasheetRealmRequired = errors.New("product datasheet realm is required")
)

// GalleryItem defines a product gallery entry.
type GalleryItem struct {
	// AssetID is the referenced asset identifier.
	AssetID string `json:"assetId"`
	// Position defines explicit gallery ordering values for drag-and-drop sorting.
	Position *int `json:"position,omitempty"`
	// VariationPosition defines optional variation-scoped ordering values for variation-linked images.
	VariationPosition *int `json:"variationPosition,omitempty"`
	// IsMain reports whether this asset is the primary image.
	IsMain bool `json:"isMain"`
	// ExcludedRealms defines realms where this asset is hidden.
	ExcludedRealms []string `json:"excludedRealms,omitempty"`
	// VariationIDs defines linked variation IDs.
	VariationIDs []string `json:"variationIds,omitempty"`
}

// Datasheet defines product presentation values by realm.
type Datasheet struct {
	// Realm identifies where the datasheet applies.
	Realm string `json:"realm"`
	// Name defines product display names.
	Name string `json:"name"`
	// Description defines product descriptions.
	Description string `json:"description,omitempty"`
	// Attributes defines extra key-value attributes.
	Attributes map[string]any `json:"attributes,omitempty"`
}

// Variant defines product variant values.
type Variant struct {
	// VariationIDs defines variation combinations for this variant.
	VariationIDs []string `json:"variationIds"`
	// SKU defines optional variant SKU overrides.
	SKU string `json:"sku,omitempty"`
}

// Product defines the product aggregate root entity.
type Product struct {
	// ID defines unique product identifiers.
	ID string `json:"_id"`
	// SKU defines unique stock-keeping values.
	SKU string `json:"sku"`
	// Gallery defines media gallery entries.
	Gallery []GalleryItem `json:"gallery,omitempty"`
	// Datasheets defines localized/realm-specific display values.
	Datasheets []Datasheet `json:"datasheets,omitempty"`
	// Variations defines linked variation IDs.
	Variations []string `json:"variations,omitempty"`
	// Variants defines composed variant entries.
	Variants []Variant `json:"variants,omitempty"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
	// IsDeleted reports soft-delete state.
	IsDeleted bool `json:"isDeleted"`
	// DeletedAt defines soft-delete timestamps.
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// Normalize canonicalizes product values before validation/persistence.
func (p *Product) Normalize() {
	if p == nil {
		return
	}

	p.SKU = strings.TrimSpace(p.SKU)
	for index := range p.Gallery {
		p.Gallery[index].AssetID = strings.TrimSpace(p.Gallery[index].AssetID)
		p.Gallery[index].Position = normalizeOptionalPosition(p.Gallery[index].Position)
		p.Gallery[index].VariationPosition = normalizeOptionalPosition(p.Gallery[index].VariationPosition)
		for excludedIndex := range p.Gallery[index].ExcludedRealms {
			p.Gallery[index].ExcludedRealms[excludedIndex] = strings.TrimSpace(p.Gallery[index].ExcludedRealms[excludedIndex])
		}
		for variationIndex := range p.Gallery[index].VariationIDs {
			p.Gallery[index].VariationIDs[variationIndex] = strings.TrimSpace(p.Gallery[index].VariationIDs[variationIndex])
		}
	}
	for index := range p.Datasheets {
		p.Datasheets[index].Realm = strings.TrimSpace(p.Datasheets[index].Realm)
		p.Datasheets[index].Name = strings.TrimSpace(p.Datasheets[index].Name)
		p.Datasheets[index].Description = strings.TrimSpace(p.Datasheets[index].Description)
	}
	for index := range p.Variations {
		p.Variations[index] = strings.TrimSpace(p.Variations[index])
	}
	for index := range p.Variants {
		p.Variants[index].SKU = strings.TrimSpace(p.Variants[index].SKU)
		for variationIndex := range p.Variants[index].VariationIDs {
			p.Variants[index].VariationIDs[variationIndex] = strings.TrimSpace(p.Variants[index].VariationIDs[variationIndex])
		}
	}
}

// Validate verifies product invariants.
func (p Product) Validate() error {
	if strings.TrimSpace(p.SKU) == "" {
		return ErrSKURequired
	}
	for _, item := range p.Gallery {
		if strings.TrimSpace(item.AssetID) == "" {
			return ErrGalleryAssetIDRequired
		}
	}
	for _, datasheet := range p.Datasheets {
		if strings.TrimSpace(datasheet.Realm) == "" {
			return ErrDatasheetRealmRequired
		}
	}

	return nil
}

// MergeDatasheets merges incoming datasheets by realm onto existing datasheets.
func MergeDatasheets(existing []Datasheet, incoming []Datasheet) []Datasheet {
	if len(incoming) == 0 {
		return existing
	}

	merged := make([]Datasheet, 0, len(existing)+len(incoming))
	byRealm := make(map[string]int, len(existing)+len(incoming))
	for _, item := range existing {
		realm := strings.TrimSpace(item.Realm)
		if realm == "" {
			continue
		}
		byRealm[realm] = len(merged)
		merged = append(merged, item)
	}
	for _, item := range incoming {
		realm := strings.TrimSpace(item.Realm)
		if realm == "" {
			continue
		}
		if index, ok := byRealm[realm]; ok {
			current := merged[index]
			if strings.TrimSpace(item.Name) != "" {
				current.Name = item.Name
			}
			if item.Description != "" {
				current.Description = item.Description
			}
			if item.Attributes != nil {
				current.Attributes = item.Attributes
			}
			merged[index] = current
			continue
		}
		byRealm[realm] = len(merged)
		merged = append(merged, item)
	}

	return merged
}

// normalizeOptionalPosition normalizes optional position values to non-negative integers.
func normalizeOptionalPosition(value *int) *int {
	if value == nil {
		return nil
	}

	resolved := *value
	if resolved < 0 {
		resolved = 0
	}

	return &resolved
}
