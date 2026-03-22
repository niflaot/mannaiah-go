package http

import (
	"context"
	"errors"
	"testing"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/email/application"
	"mannaiah/module/email/domain"
)

type handlerErrorServiceStub struct{}

// Send returns a no-op send result for handler error mapping tests.
func (handlerErrorServiceStub) Send(ctx context.Context, command application.SendCommand) (*domain.Delivery, error) {
	return &domain.Delivery{}, nil
}

// HandleWebhook returns nil for handler error mapping tests.
func (handlerErrorServiceStub) HandleWebhook(ctx context.Context, command application.WebhookCommand) error {
	return nil
}

// Get returns a no-op delivery for handler error mapping tests.
func (handlerErrorServiceStub) Get(ctx context.Context, deliveryID string) (*domain.Delivery, error) {
	return &domain.Delivery{}, nil
}

// TrackOpen returns nil for handler error mapping tests.
func (handlerErrorServiceStub) TrackOpen(ctx context.Context, deliveryID string) error {
	return nil
}

// TestMapErrorWebhookErrors verifies webhook-related domain errors are mapped to expected HTTP app errors.
func TestMapErrorWebhookErrors(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(handlerErrorServiceStub{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid webhook payload", err: domain.ErrInvalidWebhookPayload, wantStatus: 400, wantCode: "invalid_webhook_payload"},
		{name: "invalid webhook signature", err: domain.ErrInvalidWebhookSignature, wantStatus: 401, wantCode: "invalid_webhook_signature"},
		{name: "webhook topic mismatch", err: domain.ErrWebhookTopicMismatch, wantStatus: 403, wantCode: "webhook_topic_mismatch"},
		{name: "webhook subscription confirmation failed", err: domain.ErrWebhookSubscriptionConfirmationFailed, wantStatus: 503, wantCode: "webhook_subscription_confirmation_failed"},
		{name: "fallback", err: errors.New("boom"), wantStatus: 500, wantCode: "internal_server_error"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			mapped := handler.mapError(testCase.err)
			var appError *corehttp.AppError
			if !errors.As(mapped, &appError) || appError == nil {
				t.Fatalf("mapError() type = %T, want corehttp.AppError", mapped)
			}
			if appError.Status != testCase.wantStatus || appError.Message != testCase.wantCode {
				t.Fatalf("mapError() = (%d,%q), want (%d,%q)", appError.Status, appError.Message, testCase.wantStatus, testCase.wantCode)
			}
		})
	}
}
