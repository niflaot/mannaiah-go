package domain

import (
	"strings"
	"time"
)

// PackageUnit defines one package for quotation/mark workflows.
type PackageUnit struct {
	// Description defines package-content descriptions.
	Description string `json:"description"`
	// Dimensions defines package dimensions.
	Dimensions Dimensions `json:"dimensions"`
	// PackageType defines package-type values.
	PackageType string `json:"packageType"`
}

// Normalize normalizes package-unit fields.
func (u PackageUnit) Normalize() PackageUnit {
	return PackageUnit{
		Description: strings.TrimSpace(u.Description),
		Dimensions:  u.Dimensions.Normalize(),
		PackageType: strings.TrimSpace(u.PackageType),
	}
}

// QuotationRequest defines quotation input values.
type QuotationRequest struct {
	// OrderID defines optional order identifier values.
	OrderID string `json:"orderId,omitempty"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// OriginCityCode defines origin city-code values.
	OriginCityCode string `json:"originCityCode"`
	// DestCityCode defines destination city-code values.
	DestCityCode string `json:"destCityCode"`
	// Units defines package units.
	Units []PackageUnit `json:"units"`
	// DeclaredValue defines total declared-value amounts.
	DeclaredValue float64 `json:"declaredValue"`
}

// Normalize normalizes quotation request fields.
func (r QuotationRequest) Normalize() QuotationRequest {
	units := make([]PackageUnit, 0, len(r.Units))
	for _, unit := range r.Units {
		units = append(units, unit.Normalize())
	}

	value := r.DeclaredValue
	if value < 0 {
		value = 0
	}

	return QuotationRequest{
		OrderID:        strings.TrimSpace(r.OrderID),
		CarrierID:      strings.TrimSpace(r.CarrierID),
		OriginCityCode: strings.TrimSpace(r.OriginCityCode),
		DestCityCode:   strings.TrimSpace(r.DestCityCode),
		Units:          units,
		DeclaredValue:  round2(value),
	}
}

// Validate validates quotation request fields.
func (r QuotationRequest) Validate() error {
	normalized := r.Normalize()
	if normalized.CarrierID == "" {
		return ErrInvalidCarrierID
	}
	if normalized.OriginCityCode == "" || normalized.DestCityCode == "" {
		return ErrInvalidID
	}
	if len(normalized.Units) == 0 {
		return ErrInvalidID
	}

	return nil
}

// QuotationResult defines normalized quotation response values.
type QuotationResult struct {
	// ID defines quotation identifier values.
	ID string `json:"id"`
	// OrderID defines optional order identifier values.
	OrderID string `json:"orderId,omitempty"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// OriginCityCode defines origin city-code values.
	OriginCityCode string `json:"originCityCode"`
	// DestCityCode defines destination city-code values.
	DestCityCode string `json:"destCityCode"`
	// FreightCost defines total freight-cost amounts.
	FreightCost float64 `json:"freightCost"`
	// EstimatedDays defines estimated delivery-day values.
	EstimatedDays int `json:"estimatedDays"`
	// CurrencyCode defines currency-code values.
	CurrencyCode string `json:"currencyCode"`
	// ExpiresAt defines quotation expiration timestamps.
	ExpiresAt time.Time `json:"expiresAt"`
	// RawResponse defines raw provider-response payloads.
	RawResponse string `json:"rawResponse,omitempty"`
	// CreatedAt defines row creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
}
