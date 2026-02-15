package port

import (
	"context"
	"time"
)

const (
	// TopicOrderCreated defines order-created integration event topics.
	TopicOrderCreated = "orders.v1.created"
	// TopicOrderUpdated defines order-updated integration event topics.
	TopicOrderUpdated = "orders.v1.updated"
	// TopicOrderStatusUpdated defines order-status-updated integration event topics.
	TopicOrderStatusUpdated = "orders.v1.status.updated"
	// EventSourceAPI defines default API-originated event source values.
	EventSourceAPI = "api"
	// EventSourceWooCommerceSync defines WooCommerce sync-originated event source values.
	EventSourceWooCommerceSync = "woocommerce_sync"
)

// OrderEventItem defines order-item integration event payload values.
type OrderEventItem struct {
	// SKU defines product SKU values.
	SKU string `json:"sku"`
	// AlternateName defines fallback item name values.
	AlternateName string `json:"alternateName,omitempty"`
	// Quantity defines ordered quantity values.
	Quantity int `json:"quantity"`
	// Value defines monetary item values.
	Value float64 `json:"value"`
	// ProductID defines resolved product identifiers.
	ProductID string `json:"productId,omitempty"`
	// ResolutionSource defines resolution-source values.
	ResolutionSource string `json:"resolutionSource,omitempty"`
}

// OrderEventShippingAddress defines order shipping-address integration payload values.
type OrderEventShippingAddress struct {
	// Address defines address line 1 values.
	Address string `json:"address"`
	// Address2 defines address line 2 values.
	Address2 string `json:"address2,omitempty"`
	// Phone defines shipping phone values.
	Phone string `json:"phone,omitempty"`
	// CityCode defines shipping city-code values.
	CityCode string `json:"cityCode"`
}

// OrderEventShippingCharge defines order shipping-charge integration payload values.
type OrderEventShippingCharge struct {
	// MethodID defines shipping method identifier values.
	MethodID string `json:"methodId,omitempty"`
	// MethodTitle defines shipping method title values.
	MethodTitle string `json:"methodTitle,omitempty"`
	// Price defines shipping price values.
	Price float64 `json:"price"`
}

// OrderEventStatus defines order status-entry integration payload values.
type OrderEventStatus struct {
	// Status defines status values.
	Status string `json:"status"`
	// Author defines status author values.
	Author string `json:"author"`
	// Description defines status description values.
	Description string `json:"description,omitempty"`
	// NoteOwner defines note owner values.
	NoteOwner string `json:"noteOwner,omitempty"`
	// Note defines note values.
	Note string `json:"note,omitempty"`
	// OccurredAt defines status timestamps.
	OccurredAt time.Time `json:"occurredAt"`
}

// OrderEventPayload defines order integration event payload values.
type OrderEventPayload struct {
	// ID defines order identifiers.
	ID string `json:"id"`
	// Identifier defines external order identifiers.
	Identifier string `json:"identifier"`
	// Realm defines order realm values.
	Realm string `json:"realm"`
	// ContactID defines contact identifiers.
	ContactID string `json:"contactId"`
	// Source defines mutation source values.
	Source string `json:"source"`
	// CurrentStatus defines current order status values.
	CurrentStatus string `json:"currentStatus"`
	// LatestStatus defines latest status-entry values.
	LatestStatus OrderEventStatus `json:"latestStatus"`
	// Items defines order item values.
	Items []OrderEventItem `json:"items"`
	// ShippingAddress defines shipping-address values.
	ShippingAddress OrderEventShippingAddress `json:"shippingAddress"`
	// HasCustomShippingAddress reports explicit shipping-address rows.
	HasCustomShippingAddress bool `json:"hasCustomShippingAddress"`
	// ShippingCharges defines shipping-charge rows.
	ShippingCharges []OrderEventShippingCharge `json:"shippingCharges,omitempty"`
	// Metadata defines order metadata values.
	Metadata map[string]string `json:"metadata,omitempty"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}

// IntegrationEvent defines transport-level order integration event envelopes.
type IntegrationEvent struct {
	// ID defines unique event identifiers.
	ID string
	// Topic defines routing topics.
	Topic string
	// SchemaVersion defines payload schema versions.
	SchemaVersion string
	// OccurredAt defines event timestamps.
	OccurredAt time.Time
	// CorrelationID defines end-to-end flow correlation values.
	CorrelationID string
	// CausationID defines causal event references.
	CausationID string
	// Payload defines serialized event payload values.
	Payload any
	// Metadata defines optional transport metadata values.
	Metadata map[string]string
}

// IntegrationEventPublisher defines transport publication behavior for order integration events.
type IntegrationEventPublisher interface {
	// Publish emits integration events to the configured transport.
	Publish(ctx context.Context, event IntegrationEvent) error
}
