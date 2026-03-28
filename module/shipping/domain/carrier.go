package domain

import "strings"

// CarrierType classifies carrier integration modes.
type CarrierType string

const (
	// CarrierTypeAPI defines API-integrated carrier types.
	CarrierTypeAPI CarrierType = "API"
	// CarrierTypeManual defines manually-managed carrier types.
	CarrierTypeManual CarrierType = "MANUAL"
)

// Carrier defines one shipping carrier configuration.
type Carrier struct {
	// ID defines carrier identifier values.
	ID string `json:"id"`
	// Name defines display-name values.
	Name string `json:"name"`
	// Type defines carrier implementation-mode values.
	Type CarrierType `json:"type"`
	// Active defines whether the carrier is currently selectable.
	Active bool `json:"active"`
	// RequiresBalanceCheck defines whether mark generation requires pre-balance validation.
	RequiresBalanceCheck bool `json:"requiresBalanceCheck"`
	// HasQuotation defines whether the carrier supports freight quotation.
	HasQuotation bool `json:"hasQuotation"`
	// HasManifestDocument defines whether the carrier generates a carrier-specific manifest document.
	HasManifestDocument bool `json:"hasManifestDocument"`
	// HasTracking defines whether the carrier provides a tracking URL.
	HasTracking bool `json:"hasTracking"`
}

// Validate validates carrier fields.
func (c Carrier) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return ErrInvalidCarrierID
	}

	return nil
}
