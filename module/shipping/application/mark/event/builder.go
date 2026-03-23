package event

import (
	"time"

	"github.com/google/uuid"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

const (
	// schemaVersion defines shipping event schema-version values.
	schemaVersion = "v1"
)

// MarkGeneratedPayload defines mark-generated event payload values.
type MarkGeneratedPayload struct {
	// MarkID defines mark identifier values.
	MarkID string `json:"markId"`
	// OrderID defines order identifier values.
	OrderID string `json:"orderId"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// TrackingNumber defines tracking number values.
	TrackingNumber string `json:"trackingNumber"`
	// DocumentRef defines mark document reference values.
	DocumentRef string `json:"documentRef,omitempty"`
}

// MarkFailedPayload defines mark-failed event payload values.
type MarkFailedPayload struct {
	// MarkID defines mark identifier values.
	MarkID string `json:"markId"`
	// OrderID defines order identifier values.
	OrderID string `json:"orderId"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// Reason defines failure reason values.
	Reason string `json:"reason"`
}

// MarkVoidedPayload defines mark-voided event payload values.
type MarkVoidedPayload struct {
	// MarkID defines mark identifier values.
	MarkID string `json:"markId"`
	// OrderID defines order identifier values.
	OrderID string `json:"orderId"`
	// CarrierID defines carrier identifier values.
	CarrierID string `json:"carrierId"`
	// TrackingNumber defines tracking number values.
	TrackingNumber string `json:"trackingNumber"`
	// Reason defines void reason values.
	Reason string `json:"reason,omitempty"`
}

// BuildMarkGenerated builds one mark-generated integration event.
func BuildMarkGenerated(mark domain.ShippingMark) port.IntegrationEvent {
	return port.IntegrationEvent{
		ID:            uuid.NewString(),
		Topic:         port.TopicMarkGenerated,
		SchemaVersion: schemaVersion,
		OccurredAt:    time.Now().UTC(),
		Payload: MarkGeneratedPayload{
			MarkID:         mark.ID,
			OrderID:        mark.OrderID,
			CarrierID:      mark.CarrierID,
			TrackingNumber: mark.TrackingNumber,
			DocumentRef:    mark.DocumentRef,
		},
	}
}

// BuildMarkFailed builds one mark-failed integration event.
func BuildMarkFailed(mark domain.ShippingMark, reason string) port.IntegrationEvent {
	return port.IntegrationEvent{
		ID:            uuid.NewString(),
		Topic:         port.TopicMarkFailed,
		SchemaVersion: schemaVersion,
		OccurredAt:    time.Now().UTC(),
		Payload: MarkFailedPayload{
			MarkID:    mark.ID,
			OrderID:   mark.OrderID,
			CarrierID: mark.CarrierID,
			Reason:    reason,
		},
	}
}

// BuildMarkVoided builds one mark-voided integration event.
func BuildMarkVoided(mark domain.ShippingMark, reason string) port.IntegrationEvent {
	return port.IntegrationEvent{
		ID:            uuid.NewString(),
		Topic:         port.TopicMarkVoided,
		SchemaVersion: schemaVersion,
		OccurredAt:    time.Now().UTC(),
		Payload: MarkVoidedPayload{
			MarkID:         mark.ID,
			OrderID:        mark.OrderID,
			CarrierID:      mark.CarrierID,
			TrackingNumber: mark.TrackingNumber,
			Reason:         reason,
		},
	}
}
