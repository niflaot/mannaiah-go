package service

import (
	"context"
	"errors"
	"testing"

	contactsdomain "mannaiah/module/contacts/domain"
	shopifyport "mannaiah/module/shopify/port"
)

type contactServiceCustomerSourceStub struct{}

func (contactServiceCustomerSourceStub) Validate(ctx context.Context) error {
	_ = ctx
	return nil
}

func (contactServiceCustomerSourceStub) GetCustomer(ctx context.Context, id string) (shopifyport.ShopifyCustomer, error) {
	_ = ctx
	_ = id
	return shopifyport.ShopifyCustomer{}, shopifyport.ErrCustomerNotFound
}

func (contactServiceCustomerSourceStub) ListCustomers(ctx context.Context, sinceID string, limit int) ([]shopifyport.ShopifyCustomer, bool, error) {
	_ = ctx
	_ = sinceID
	_ = limit
	return []shopifyport.ShopifyCustomer{}, false, nil
}

type contactServiceTargetStub struct{}

func (contactServiceTargetStub) UpsertContact(ctx context.Context, command shopifyport.ContactSyncCommand) (*contactsdomain.Contact, error) {
	_ = ctx
	_ = command
	return &contactsdomain.Contact{ID: "contact-1"}, nil
}

type contactServiceRecorderStub struct {
	startErr      error
	completeCalls int
	failCalls     int
}

func (r *contactServiceRecorderStub) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	_ = ctx
	_ = kind
	_ = trigger
	return "", r.startErr
}

func (r *contactServiceRecorderStub) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	_ = ctx
	_ = runID
	_ = processed
	_ = succeeded
	_ = failed
	_ = skipped
	r.completeCalls++
	return nil
}

func (r *contactServiceRecorderStub) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []shopifyport.SyncError) error {
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

// TestResolveTriggerMapsShopifyWebhookToSyncRecordEvent verifies Shopify webhook triggers match syncrecord enums.
func TestResolveTriggerMapsShopifyWebhookToSyncRecordEvent(t *testing.T) {
	tests := []struct {
		name    string
		trigger string
		want    string
	}{
		{name: "blank defaults manual", trigger: " ", want: "manual"},
		{name: "manual remains manual", trigger: "manual", want: "manual"},
		{name: "webhook maps event", trigger: "webhook", want: "event"},
		{name: "webhook is case insensitive", trigger: "WEBHOOK", want: "event"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveTrigger(tc.trigger); got != tc.want {
				t.Fatalf("resolveTrigger(%q) = %q, want %q", tc.trigger, got, tc.want)
			}
		})
	}
}

// TestSyncContactsSkipsCompletionWhenRunStartFails verifies recorder failures do not emit empty-run completions.
func TestSyncContactsSkipsCompletionWhenRunStartFails(t *testing.T) {
	recorder := &contactServiceRecorderStub{startErr: errors.New("recorder unavailable")}
	service, err := NewService(SyncConfig{Enabled: true}, contactServiceCustomerSourceStub{}, contactServiceTargetStub{}, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetSyncRecorder(recorder)

	summary, err := service.SyncContacts(context.Background(), "webhook")
	if err != nil {
		t.Fatalf("SyncContacts() error = %v", err)
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
