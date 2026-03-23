package port

import (
	"context"

	"mannaiah/module/shipping/domain"
)

// TrackingProvider defines carrier-tracking query behavior.
type TrackingProvider interface {
	// SupportsCourier reports whether this provider supports one carrier identifier.
	SupportsCourier(carrierID string) bool
	// GetTrackingHistory retrieves normalized tracking history values.
	GetTrackingHistory(ctx context.Context, trackingNumber string) (*domain.TrackingHistory, error)
}
