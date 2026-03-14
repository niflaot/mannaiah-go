package domain

import "time"

// DeliveryStatus defines email delivery status values.
type DeliveryStatus string

const (
	// StatusPending defines pending dispatch statuses.
	StatusPending DeliveryStatus = "pending"
	// StatusSubmitted defines provider-submitted statuses.
	StatusSubmitted DeliveryStatus = "submitted_to_provider"
	// StatusFailedRetryable defines retryable failure statuses.
	StatusFailedRetryable DeliveryStatus = "failed_retryable"
	// StatusFailedPermanent defines permanent failure statuses.
	StatusFailedPermanent DeliveryStatus = "failed_permanent"
	// StatusDelivered defines delivered statuses.
	StatusDelivered DeliveryStatus = "delivered"
	// StatusBounced defines bounced statuses.
	StatusBounced DeliveryStatus = "bounced"
	// StatusComplained defines complaint statuses.
	StatusComplained DeliveryStatus = "complained"
)

// Delivery defines email delivery record values.
type Delivery struct {
	// ID defines delivery identifier values.
	ID string `json:"id"`
	// ContactID defines optional contact identifier values.
	ContactID string `json:"contactId,omitempty"`
	// Email defines recipient email values.
	Email string `json:"email"`
	// Subject defines delivery subject values.
	Subject string `json:"subject"`
	// HTMLBody defines html payload values.
	HTMLBody string `json:"htmlBody"`
	// TextBody defines text payload values.
	TextBody string `json:"textBody"`
	// IdempotencyKey defines idempotency values.
	IdempotencyKey string `json:"idempotencyKey"`
	// Provider defines provider labels.
	Provider string `json:"provider"`
	// ProviderMessageID defines provider message id values.
	ProviderMessageID string `json:"providerMessageId,omitempty"`
	// Status defines current delivery status values.
	Status DeliveryStatus `json:"status"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines row update timestamp values.
	UpdatedAt time.Time `json:"updatedAt"`
}

// StatusEntry defines immutable delivery status timeline rows.
type StatusEntry struct {
	// ID defines status entry identifier values.
	ID string `json:"id"`
	// DeliveryID defines parent delivery identifier values.
	DeliveryID string `json:"deliveryId"`
	// Status defines status values.
	Status DeliveryStatus `json:"status"`
	// Reason defines optional reason values.
	Reason string `json:"reason,omitempty"`
	// OccurredAt defines status timestamps.
	OccurredAt time.Time `json:"occurredAt"`
	// CreatedAt defines row creation timestamp values.
	CreatedAt time.Time `json:"createdAt"`
}
