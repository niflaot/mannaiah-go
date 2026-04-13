// Package event defines coupon integration event builders.
package event

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"mannaiah/module/coupons/domain"
	"mannaiah/module/coupons/port"
)

const schemaVersionV1 = "v1"

// CouponEventPayload defines the shared coupon event payload structure.
type CouponEventPayload struct {
	// CouponID defines the affected coupon identifier.
	CouponID string `json:"couponId"`
	// Code defines the coupon code.
	Code string `json:"code"`
	// Origin defines the coupon origin.
	Origin string `json:"origin"`
	// DiscountType defines the discount type.
	DiscountType string `json:"discountType"`
	// DiscountAmount defines the discount amount.
	DiscountAmount float64 `json:"discountAmount"`
	// Active reports whether the coupon is active.
	Active bool `json:"active"`
}

// CouponUsedPayload defines the coupon-used event payload structure.
type CouponUsedPayload struct {
	// CouponID defines the redeemed coupon identifier.
	CouponID string `json:"couponId"`
	// Code defines the coupon code.
	Code string `json:"code"`
	// OrderID defines the order where the coupon was applied.
	OrderID string `json:"orderId"`
	// Email defines the email that redeemed the coupon.
	Email string `json:"email"`
	// UsedAt defines the redemption timestamp.
	UsedAt time.Time `json:"usedAt"`
}

// noopPublisher defines no-op integration event publication behavior.
type noopPublisher struct{}

// Publish ignores integration events.
func (noopPublisher) Publish(_ context.Context, _ port.IntegrationEvent) error { return nil }

// ResolvePublisher resolves optional integration event publisher dependencies.
func ResolvePublisher(publisher port.IntegrationEventPublisher) port.IntegrationEventPublisher {
	if publisher != nil {
		return publisher
	}

	return noopPublisher{}
}

// NewCouponCreatedEvent builds a coupon-created integration event.
func NewCouponCreatedEvent(coupon domain.Coupon) port.IntegrationEvent {
	return buildCouponEvent(port.TopicCouponCreated, couponEventPayload(coupon))
}

// NewCouponUpdatedEvent builds a coupon-updated integration event.
func NewCouponUpdatedEvent(coupon domain.Coupon) port.IntegrationEvent {
	return buildCouponEvent(port.TopicCouponUpdated, couponEventPayload(coupon))
}

// NewCouponDeletedEvent builds a coupon-deleted integration event.
func NewCouponDeletedEvent(coupon domain.Coupon) port.IntegrationEvent {
	return buildCouponEvent(port.TopicCouponDeleted, couponEventPayload(coupon))
}

// NewCouponUsedEvent builds a coupon-used integration event.
func NewCouponUsedEvent(couponID, code, orderID, email string, usedAt time.Time) port.IntegrationEvent {
	return buildCouponEvent(port.TopicCouponUsed, CouponUsedPayload{
		CouponID: couponID,
		Code:     code,
		OrderID:  orderID,
		Email:    email,
		UsedAt:   usedAt,
	})
}

// buildCouponEvent creates a coupon integration event envelope from topic and payload values.
func buildCouponEvent(topic string, payload any) port.IntegrationEvent {
	return port.IntegrationEvent{
		ID:            generateEventID(),
		Topic:         topic,
		SchemaVersion: schemaVersionV1,
		OccurredAt:    time.Now().UTC(),
		Payload:       payload,
	}
}

// couponEventPayload maps a coupon domain value to the shared event payload.
func couponEventPayload(c domain.Coupon) CouponEventPayload {
	return CouponEventPayload{
		CouponID:       c.ID,
		Code:           c.Code,
		Origin:         c.Origin,
		DiscountType:   string(c.DiscountType),
		DiscountAmount: c.DiscountAmount,
		Active:         c.Active,
	}
}

// generateEventID creates random integration event identifiers.
func generateEventID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("event-%d", time.Now().UnixNano())
	}

	return hex.EncodeToString(bytes)
}
