package service

import (
	"context"
	"errors"
	"testing"

	contactsdomain "mannaiah/module/contacts/domain"
	ordersdomain "mannaiah/module/orders/domain"
	shopifyport "mannaiah/module/shopify/port"
)

type orderServiceSourceStub struct{}

func (orderServiceSourceStub) Validate(ctx context.Context) error {
	_ = ctx
	return nil
}

func (orderServiceSourceStub) GetOrder(ctx context.Context, id string) (shopifyport.ShopifyOrder, error) {
	_ = ctx
	_ = id
	return shopifyport.ShopifyOrder{}, shopifyport.ErrOrderNotFound
}

func (orderServiceSourceStub) ListOrders(ctx context.Context, sinceID string, limit int) ([]shopifyport.ShopifyOrder, bool, error) {
	_ = ctx
	_ = sinceID
	_ = limit
	return []shopifyport.ShopifyOrder{}, false, nil
}

type orderServiceContactTargetStub struct{}

func (orderServiceContactTargetStub) UpsertContact(ctx context.Context, command shopifyport.ContactSyncCommand) (*contactsdomain.Contact, error) {
	_ = ctx
	_ = command
	return &contactsdomain.Contact{ID: "contact-1"}, nil
}

type orderServiceTargetStub struct{}

func (orderServiceTargetStub) UpsertOrder(ctx context.Context, command shopifyport.OrderSyncCommand) (*ordersdomain.Order, error) {
	_ = ctx
	_ = command
	return &ordersdomain.Order{ID: "order-1"}, nil
}

type orderServiceRecorderStub struct {
	startErr      error
	completeCalls int
	failCalls     int
}

func (r *orderServiceRecorderStub) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	_ = ctx
	_ = kind
	_ = trigger
	return "", r.startErr
}

func (r *orderServiceRecorderStub) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	_ = ctx
	_ = runID
	_ = processed
	_ = succeeded
	_ = failed
	_ = skipped
	r.completeCalls++
	return nil
}

func (r *orderServiceRecorderStub) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []shopifyport.SyncError) error {
	_ = ctx
	_ = runID
	_ = processed
	_ = succeeded
	_ = failed
	_ = skipped
	_ = syncErrors
	r.failCalls++
	return nil
}

// TestResolveTriggerMapsOrderTriggersToSyncRecordValues verifies order sync triggers match syncrecord enums.
func TestResolveTriggerMapsOrderTriggersToSyncRecordValues(t *testing.T) {
	tests := []struct {
		name    string
		trigger string
		want    string
	}{
		{name: "blank defaults manual", trigger: " ", want: "manual"},
		{name: "manual remains manual", trigger: "manual", want: "manual"},
		{name: "webhook maps event", trigger: "webhook", want: "event"},
		{name: "does not prefix manual", trigger: " manual ", want: "manual"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveTrigger(tc.trigger); got != tc.want {
				t.Fatalf("resolveTrigger(%q) = %q, want %q", tc.trigger, got, tc.want)
			}
		})
	}
}

// TestSyncOrdersSkipsCompletionWhenRunStartFails verifies recorder failures do not emit empty-run completions.
func TestSyncOrdersSkipsCompletionWhenRunStartFails(t *testing.T) {
	recorder := &orderServiceRecorderStub{startErr: errors.New("recorder unavailable")}
	service, err := NewService(
		SyncConfig{Enabled: true},
		orderServiceSourceStub{},
		orderServiceContactTargetStub{},
		orderServiceTargetStub{},
		nil,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetSyncRecorder(recorder)

	summary, err := service.SyncOrders(context.Background(), "webhook")
	if err != nil {
		t.Fatalf("SyncOrders() error = %v", err)
	}
	if summary.Trigger != "event" {
		t.Fatalf("summary trigger = %q, want event", summary.Trigger)
	}
	if recorder.completeCalls != 0 {
		t.Fatalf("CompleteRun calls = %d, want 0", recorder.completeCalls)
	}
	if recorder.failCalls != 0 {
		t.Fatalf("FailRun calls = %d, want 0", recorder.failCalls)
	}
}
