package event_test

import (
	"context"
	"testing"
	"time"

	couponevent "mannaiah/module/coupons/application/coupon/event"
	"mannaiah/module/coupons/domain"
	"mannaiah/module/coupons/port"
)

// TestResolvePublisher_nilReturnsNoop verifies that nil resolves to a no-op publisher.
func TestResolvePublisher_nilReturnsNoop(t *testing.T) {
	pub := couponevent.ResolvePublisher(nil)
	if pub == nil {
		t.Fatal("expected non-nil noop publisher")
	}
}

// TestResolvePublisher_nonNilPassthrough verifies that a real publisher is returned as-is.
func TestResolvePublisher_nonNilPassthrough(t *testing.T) {
	stub := &stubPublisher{}
	pub := couponevent.ResolvePublisher(stub)
	if pub == nil {
		t.Fatal("expected non-nil publisher")
	}
}

// TestNewCouponCreatedEvent_topic verifies that coupon-created events use the correct topic.
func TestNewCouponCreatedEvent_topic(t *testing.T) {
	ev := couponevent.NewCouponCreatedEvent(makeCoupon())
	if ev.Topic != port.TopicCouponCreated {
		t.Fatalf("expected topic %q, got %q", port.TopicCouponCreated, ev.Topic)
	}
	if ev.ID == "" {
		t.Fatal("expected non-empty event ID")
	}
	if ev.OccurredAt.IsZero() {
		t.Fatal("expected non-zero OccurredAt")
	}
}

// TestNewCouponUpdatedEvent_topic verifies that coupon-updated events use the correct topic.
func TestNewCouponUpdatedEvent_topic(t *testing.T) {
	ev := couponevent.NewCouponUpdatedEvent(makeCoupon())
	if ev.Topic != port.TopicCouponUpdated {
		t.Fatalf("expected topic %q, got %q", port.TopicCouponUpdated, ev.Topic)
	}
}

// TestNewCouponDeletedEvent_topic verifies that coupon-deleted events use the correct topic.
func TestNewCouponDeletedEvent_topic(t *testing.T) {
	ev := couponevent.NewCouponDeletedEvent(makeCoupon())
	if ev.Topic != port.TopicCouponDeleted {
		t.Fatalf("expected topic %q, got %q", port.TopicCouponDeleted, ev.Topic)
	}
}

// TestNewCouponUsedEvent_payload verifies that coupon-used events carry correct payload values.
func TestNewCouponUsedEvent_payload(t *testing.T) {
	usedAt := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	ev := couponevent.NewCouponUsedEvent("coupon-1", "SAVE10", "order-1", "user@example.com", usedAt)
	if ev.Topic != port.TopicCouponUsed {
		t.Fatalf("expected topic %q, got %q", port.TopicCouponUsed, ev.Topic)
	}
	payload, ok := ev.Payload.(couponevent.CouponUsedPayload)
	if !ok {
		t.Fatalf("expected CouponUsedPayload, got %T", ev.Payload)
	}
	if payload.CouponID != "coupon-1" {
		t.Errorf("expected coupon id %q, got %q", "coupon-1", payload.CouponID)
	}
	if payload.Code != "SAVE10" {
		t.Errorf("expected code %q, got %q", "SAVE10", payload.Code)
	}
	if payload.OrderID != "order-1" {
		t.Errorf("expected order id %q, got %q", "order-1", payload.OrderID)
	}
}

// makeCoupon returns a minimal valid coupon for testing.
func makeCoupon() domain.Coupon {
	return domain.Coupon{
		ID:             "c1",
		Code:           "TEST10",
		Origin:         "manual",
		DiscountType:   domain.DiscountTypeFixed,
		DiscountAmount: 10,
		Active:         true,
	}
}

// stubPublisher defines a test stub for IntegrationEventPublisher.
type stubPublisher struct{}

// Publish records events without acting on them.
func (s *stubPublisher) Publish(_ context.Context, _ port.IntegrationEvent) error { return nil }
