package port

import (
	"context"

	"mannaiah/module/shipping/domain"
)

// CarrierProvider defines per-carrier quotation/mark operations.
type CarrierProvider interface {
	// CarrierID returns the carrier identifier served by this provider.
	CarrierID() string
	// Carrier returns static carrier metadata.
	Carrier() domain.Carrier
	// Quote requests one freight quotation.
	Quote(ctx context.Context, request domain.QuotationRequest) (*domain.QuotationResult, error)
	// GenerateMark creates one shipping mark in the upstream carrier.
	GenerateMark(ctx context.Context, mark *domain.ShippingMark) error
	// VoidMark voids one existing shipping mark.
	VoidMark(ctx context.Context, trackingNumber string) error
	// CheckBalance validates account balance before mark generation.
	CheckBalance(ctx context.Context) error
	// SupportsQuotation reports whether Quote is supported.
	SupportsQuotation() bool
}

// ProviderRegistry defines carrier and tracking provider lookup behavior.
type ProviderRegistry interface {
	// CarrierProvider resolves one carrier provider by identifier.
	CarrierProvider(carrierID string) (CarrierProvider, bool)
	// TrackingProvider resolves one tracking provider by carrier identifier.
	TrackingProvider(carrierID string) (TrackingProvider, bool)
	// Carriers lists all configured carriers.
	Carriers() []domain.Carrier
}
