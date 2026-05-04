package port

import (
	"context"
	"time"
)

const (
	// TopicCouponCreated defines coupon-created integration event topics.
	TopicCouponCreated = "coupons.v1.coupon.created"
	// TopicCouponUpdated defines coupon-updated integration event topics.
	TopicCouponUpdated = "coupons.v1.coupon.updated"
	// TopicCouponDeleted defines coupon-deleted integration event topics.
	TopicCouponDeleted = "coupons.v1.coupon.deleted"
	// TopicCouponUsed defines coupon-used integration event topics.
	TopicCouponUsed = "coupons.v1.coupon.used"
)

// IntegrationEvent defines transport-level coupon integration event envelopes.
type IntegrationEvent struct {
	// ID defines unique event identifiers.
	ID string
	// Topic defines event routing topics.
	Topic string
	// SchemaVersion defines payload schema versions.
	SchemaVersion string
	// OccurredAt defines event timestamps.
	OccurredAt time.Time
	// Payload defines serialized event payload values.
	Payload any
	// Metadata defines optional transport metadata values.
	Metadata map[string]string
}

// IntegrationEventPublisher defines event publication behavior for coupon integration events.
type IntegrationEventPublisher interface {
	// Publish emits integration events to the configured transport.
	Publish(ctx context.Context, event IntegrationEvent) error
}
