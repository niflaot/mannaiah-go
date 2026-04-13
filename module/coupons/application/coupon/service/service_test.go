package service_test

import (
	"context"
	"testing"
	"time"

	couponservice "mannaiah/module/coupons/application/coupon/service"
	"mannaiah/module/coupons/domain"
	"mannaiah/module/coupons/port"
)

// TestNewService_nilRepositoryReturnsError verifies that nil repository is rejected.
func TestNewService_nilRepositoryReturnsError(t *testing.T) {
	_, err := couponservice.NewService(nil, newMockUsageRepository(), nil)
	if err == nil {
		t.Fatal("expected error for nil repository")
	}
}

// TestNewService_nilUsageRepositoryReturnsError verifies that nil usage repository is rejected.
func TestNewService_nilUsageRepositoryReturnsError(t *testing.T) {
	_, err := couponservice.NewService(newMockRepository(), nil, nil)
	if err == nil {
		t.Fatal("expected error for nil usage repository")
	}
}

// TestCreate_randomCode verifies that a coupon is created with a generated code when none is provided.
func TestCreate_randomCode(t *testing.T) {
	svc := mustNewService(t)

	coupon, err := svc.Create(context.Background(), couponservice.CreateCommand{
		Origin:         "manual",
		DiscountType:   domain.DiscountTypeFixed,
		DiscountAmount: 10,
		Active:         true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if coupon.Code == "" {
		t.Fatal("expected generated code")
	}
	if len(coupon.Code) < 10 {
		t.Errorf("generated code seems too short: %q", coupon.Code)
	}
}

// TestCreate_manualCode verifies that a provided code is persisted as-is (uppercased).
func TestCreate_manualCode(t *testing.T) {
	svc := mustNewService(t)

	coupon, err := svc.Create(context.Background(), couponservice.CreateCommand{
		Code:           "summer25",
		Origin:         "campaign",
		DiscountType:   domain.DiscountTypePercentage,
		DiscountAmount: 25,
		Active:         true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if coupon.Code != "SUMMER25" {
		t.Errorf("expected code SUMMER25, got %q", coupon.Code)
	}
}

// TestCreate_duplicateCodeReturnsConflict verifies that duplicate codes are rejected.
func TestCreate_duplicateCodeReturnsConflict(t *testing.T) {
	svc := mustNewService(t)
	cmd := couponservice.CreateCommand{
		Code:           "UNIQUE10",
		DiscountType:   domain.DiscountTypeFixed,
		DiscountAmount: 10,
		Active:         true,
	}
	if _, err := svc.Create(context.Background(), cmd); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err := svc.Create(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected conflict error on duplicate code")
	}
}

// TestCreate_invalidDiscountTypeReturnsValidationError verifies that domain validation is applied.
func TestCreate_invalidDiscountTypeReturnsValidationError(t *testing.T) {
	svc := mustNewService(t)
	_, err := svc.Create(context.Background(), couponservice.CreateCommand{
		Code:           "BAD",
		DiscountType:   "unknown",
		DiscountAmount: 5,
		Active:         true,
	})
	if err == nil {
		t.Fatal("expected validation error for invalid discount type")
	}
}

// TestCreate_emitsCreatedEvent verifies that a coupon-created event is published.
func TestCreate_emitsCreatedEvent(t *testing.T) {
	repo := newMockRepository()
	usageRepo := newMockUsageRepository()
	pub := &mockPublisher{}
	svc, _ := couponservice.NewService(repo, usageRepo, pub)

	if _, err := svc.Create(context.Background(), couponservice.CreateCommand{
		DiscountType:   domain.DiscountTypeFixed,
		DiscountAmount: 5,
		Active:         true,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := pub.published()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Topic != port.TopicCouponCreated {
		t.Errorf("expected topic %q, got %q", port.TopicCouponCreated, events[0].Topic)
	}
}

// TestUpdate_appliesMutations verifies that coupon fields are updated correctly.
func TestUpdate_appliesMutations(t *testing.T) {
	svc := mustNewService(t)
	coupon, err := svc.Create(context.Background(), couponservice.CreateCommand{
		DiscountType:   domain.DiscountTypeFixed,
		DiscountAmount: 10,
		Active:         true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := svc.Update(context.Background(), couponservice.UpdateCommand{
		ID:             coupon.ID,
		Origin:         "campaign",
		DiscountType:   domain.DiscountTypePercentage,
		DiscountAmount: 20,
		Active:         false,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.DiscountAmount != 20 {
		t.Errorf("expected amount 20, got %v", updated.DiscountAmount)
	}
	if updated.Active {
		t.Error("expected inactive coupon")
	}
}

// TestDelete_notFoundReturnsError verifies that deleting a missing coupon returns an error.
func TestDelete_notFoundReturnsError(t *testing.T) {
	svc := mustNewService(t)
	err := svc.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected not found error")
	}
}

// TestRecordUsage_happyPath verifies that usage is recorded and event emitted.
func TestRecordUsage_happyPath(t *testing.T) {
	repo := newMockRepository()
	usageRepo := newMockUsageRepository()
	pub := &mockPublisher{}
	svc, _ := couponservice.NewService(repo, usageRepo, pub)

	coupon, _ := svc.Create(context.Background(), couponservice.CreateCommand{
		DiscountType:   domain.DiscountTypeFixed,
		DiscountAmount: 5,
		Active:         true,
		ExpiresAt:      futureTime(),
	})

	err := svc.RecordUsage(context.Background(), couponservice.RecordUsageCommand{
		CouponID: coupon.ID,
		OrderID:  "order-1",
		Email:    "user@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := pub.published()
	usedEvents := filterByTopic(events, port.TopicCouponUsed)
	if len(usedEvents) != 1 {
		t.Fatalf("expected 1 used event, got %d", len(usedEvents))
	}
}

// TestSyncUsageByCode_historicalBackfillBypassesLiveRedemptionRules verifies historical sync behavior.
func TestSyncUsageByCode_historicalBackfillBypassesLiveRedemptionRules(t *testing.T) {
	repo := newMockRepository()
	usageRepo := newMockUsageRepository()
	pub := &mockPublisher{}
	svc, _ := couponservice.NewService(repo, usageRepo, pub)

	coupon, _ := svc.Create(context.Background(), couponservice.CreateCommand{
		Code:            "woo-sync",
		DiscountType:    domain.DiscountTypeFixed,
		DiscountAmount:  15,
		Active:          false,
		ExpiresAt:       pastTime(),
		MaxUsagesGlobal: ptr(1),
	})

	_ = usageRepo.RecordUsage(context.Background(), port.UsageRecord{
		CouponID: coupon.ID,
		OrderID:  "previous-order",
		Email:    "existing@example.com",
		UsedAt:   time.Now().UTC(),
	})

	usedAt := time.Date(2026, time.April, 13, 14, 30, 0, 0, time.UTC)
	err := svc.SyncUsageByCode(context.Background(), couponservice.SyncUsageByCodeCommand{
		Code:    "woo-sync",
		OrderID: "order-42",
		Email:   "SYNCED@EXAMPLE.COM",
		UsedAt:  &usedAt,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(usageRepo.usages) != 2 {
		t.Fatalf("len(usages) = %d, want 2", len(usageRepo.usages))
	}
	if usageRepo.usages[1].CouponID != coupon.ID {
		t.Fatalf("usage coupon id = %q, want %q", usageRepo.usages[1].CouponID, coupon.ID)
	}
	if usageRepo.usages[1].OrderID != "order-42" {
		t.Fatalf("usage order id = %q, want %q", usageRepo.usages[1].OrderID, "order-42")
	}
	if usageRepo.usages[1].Email != "synced@example.com" {
		t.Fatalf("usage email = %q, want %q", usageRepo.usages[1].Email, "synced@example.com")
	}
	if !usageRepo.usages[1].UsedAt.Equal(usedAt) {
		t.Fatalf("usage usedAt = %v, want %v", usageRepo.usages[1].UsedAt, usedAt)
	}

	usedEvents := filterByTopic(pub.published(), port.TopicCouponUsed)
	if len(usedEvents) != 1 {
		t.Fatalf("expected 1 used event, got %d", len(usedEvents))
	}
}

// TestSyncUsageByCode_duplicateOrderIsNoOp verifies sync idempotency for repeated order imports.
func TestSyncUsageByCode_duplicateOrderIsNoOp(t *testing.T) {
	repo := newMockRepository()
	usageRepo := newMockUsageRepository()
	svc, _ := couponservice.NewService(repo, usageRepo, nil)
	coupon, _ := svc.Create(context.Background(), couponservice.CreateCommand{
		Code:           "DUPLICATE10",
		DiscountType:   domain.DiscountTypeFixed,
		DiscountAmount: 10,
		Active:         true,
	})

	if err := svc.SyncUsageByCode(context.Background(), couponservice.SyncUsageByCodeCommand{
		Code:    coupon.Code,
		OrderID: "order-1",
		Email:   "one@example.com",
	}); err != nil {
		t.Fatalf("first sync usage error = %v", err)
	}
	if err := svc.SyncUsageByCode(context.Background(), couponservice.SyncUsageByCodeCommand{
		Code:    coupon.Code,
		OrderID: "order-1",
		Email:   "one@example.com",
	}); err != nil {
		t.Fatalf("second sync usage error = %v", err)
	}

	if len(usageRepo.usages) != 1 {
		t.Fatalf("len(usages) = %d, want 1", len(usageRepo.usages))
	}
}

// TestRecordUsage_globalLimitReached verifies that global usage limits are enforced.
func TestRecordUsage_globalLimitReached(t *testing.T) {
	repo := newMockRepository()
	usageRepo := newMockUsageRepository()
	svc, _ := couponservice.NewService(repo, usageRepo, nil)

	coupon, _ := svc.Create(context.Background(), couponservice.CreateCommand{
		DiscountType:    domain.DiscountTypeFixed,
		DiscountAmount:  5,
		Active:          true,
		MaxUsagesGlobal: ptr(1),
	})

	_ = svc.RecordUsage(context.Background(), couponservice.RecordUsageCommand{
		CouponID: coupon.ID,
		OrderID:  "order-1",
		Email:    "a@b.com",
	})

	err := svc.RecordUsage(context.Background(), couponservice.RecordUsageCommand{
		CouponID: coupon.ID,
		OrderID:  "order-2",
		Email:    "c@d.com",
	})
	if err == nil {
		t.Fatal("expected exhausted error")
	}
}

// TestRecordUsage_perEmailLimitReached verifies that per-email usage limits are enforced.
func TestRecordUsage_perEmailLimitReached(t *testing.T) {
	repo := newMockRepository()
	usageRepo := newMockUsageRepository()
	svc, _ := couponservice.NewService(repo, usageRepo, nil)

	coupon, _ := svc.Create(context.Background(), couponservice.CreateCommand{
		DiscountType:      domain.DiscountTypeFixed,
		DiscountAmount:    5,
		Active:            true,
		MaxUsagesPerEmail: ptr(1),
	})

	_ = svc.RecordUsage(context.Background(), couponservice.RecordUsageCommand{
		CouponID: coupon.ID,
		OrderID:  "order-1",
		Email:    "same@example.com",
	})

	err := svc.RecordUsage(context.Background(), couponservice.RecordUsageCommand{
		CouponID: coupon.ID,
		OrderID:  "order-2",
		Email:    "same@example.com",
	})
	if err == nil {
		t.Fatal("expected per-email exhausted error")
	}
}

// TestRecordUsage_expiredCoupon verifies that expired coupons are rejected.
func TestRecordUsage_expiredCoupon(t *testing.T) {
	repo := newMockRepository()
	usageRepo := newMockUsageRepository()
	svc, _ := couponservice.NewService(repo, usageRepo, nil)

	coupon, _ := svc.Create(context.Background(), couponservice.CreateCommand{
		DiscountType:   domain.DiscountTypeFixed,
		DiscountAmount: 5,
		Active:         true,
		ExpiresAt:      pastTime(),
	})

	err := svc.RecordUsage(context.Background(), couponservice.RecordUsageCommand{
		CouponID: coupon.ID,
		OrderID:  "order-1",
		Email:    "user@example.com",
	})
	if err == nil {
		t.Fatal("expected expired error")
	}
}

// TestRecordUsage_duplicateOrderRejected verifies that applying the same coupon twice to one order is rejected.
func TestRecordUsage_duplicateOrderRejected(t *testing.T) {
	repo := newMockRepository()
	usageRepo := newMockUsageRepository()
	svc, _ := couponservice.NewService(repo, usageRepo, nil)

	coupon, _ := svc.Create(context.Background(), couponservice.CreateCommand{
		DiscountType:   domain.DiscountTypeFixed,
		DiscountAmount: 5,
		Active:         true,
	})

	_ = svc.RecordUsage(context.Background(), couponservice.RecordUsageCommand{
		CouponID: coupon.ID,
		OrderID:  "order-1",
		Email:    "user@example.com",
	})
	err := svc.RecordUsage(context.Background(), couponservice.RecordUsageCommand{
		CouponID: coupon.ID,
		OrderID:  "order-1",
		Email:    "user@example.com",
	})
	if err == nil {
		t.Fatal("expected already-used-on-order error")
	}
}

// mustNewService creates a Service with in-memory dependencies for testing.
func mustNewService(t *testing.T) *couponservice.Service {
	t.Helper()
	svc, err := couponservice.NewService(newMockRepository(), newMockUsageRepository(), nil)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

// filterByTopic filters integration events by topic.
func filterByTopic(events []port.IntegrationEvent, topic string) []port.IntegrationEvent {
	var result []port.IntegrationEvent
	for _, ev := range events {
		if ev.Topic == topic {
			result = append(result, ev)
		}
	}
	return result
}
