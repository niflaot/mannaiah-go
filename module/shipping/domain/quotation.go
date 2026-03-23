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
	// CollectOnDeliveryAmount defines requested COD collection amounts.
	CollectOnDeliveryAmount float64 `json:"collectOnDeliveryAmount,omitempty"`
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
	collectOnDeliveryAmount := r.CollectOnDeliveryAmount
	if collectOnDeliveryAmount < 0 {
		collectOnDeliveryAmount = 0
	}

	return QuotationRequest{
		OrderID:                 strings.TrimSpace(r.OrderID),
		CarrierID:               strings.TrimSpace(r.CarrierID),
		OriginCityCode:          strings.TrimSpace(r.OriginCityCode),
		DestCityCode:            strings.TrimSpace(r.DestCityCode),
		Units:                   units,
		DeclaredValue:           round2(value),
		CollectOnDeliveryAmount: round2(collectOnDeliveryAmount),
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
	// FullFreightCost defines carrier-reported freight-cost amounts before discount.
	FullFreightCost float64 `json:"fullFreightCost"`
	// DiscountPercent defines configured freight discount percentage values.
	DiscountPercent float64 `json:"discountPercent"`
	// DiscountedFreightCost defines freight-cost amounts after discount.
	DiscountedFreightCost float64 `json:"discountedFreightCost"`
	// FreightCost defines discounted freight-cost amounts for backward compatibility.
	FreightCost float64 `json:"freightCost"`
	// CollectOnDeliveryAmount defines requested COD collection amounts.
	CollectOnDeliveryAmount float64 `json:"collectOnDeliveryAmount,omitempty"`
	// CollectOnDeliveryFeePercent defines applied COD fee percentage values.
	CollectOnDeliveryFeePercent float64 `json:"collectOnDeliveryFeePercent,omitempty"`
	// CollectOnDeliveryFeeAmount defines applied COD fee amount values.
	CollectOnDeliveryFeeAmount float64 `json:"collectOnDeliveryFeeAmount,omitempty"`
	// CollectOnDeliveryChargedAmount defines final COD amount sent to carrier.
	CollectOnDeliveryChargedAmount float64 `json:"collectOnDeliveryChargedAmount,omitempty"`
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
