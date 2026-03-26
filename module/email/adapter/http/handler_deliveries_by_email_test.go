package http

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/email/application"
	"mannaiah/module/email/domain"
)

type deliveriesByEmailServiceStub struct {
	listByEmailFn func(ctx context.Context, email string) ([]*domain.Delivery, error)
}

// Send returns a no-op send result for deliveries-by-email handler tests.
func (s deliveriesByEmailServiceStub) Send(ctx context.Context, command application.SendCommand) (*domain.Delivery, error) {
	return &domain.Delivery{}, nil
}

// HandleWebhook returns nil for deliveries-by-email handler tests.
func (s deliveriesByEmailServiceStub) HandleWebhook(ctx context.Context, command application.WebhookCommand) error {
	return nil
}

// Get returns a no-op delivery for deliveries-by-email handler tests.
func (s deliveriesByEmailServiceStub) Get(ctx context.Context, deliveryID string) (*domain.Delivery, error) {
	return &domain.Delivery{}, nil
}

// ListByEmail executes configured list behavior for deliveries-by-email handler tests.
func (s deliveriesByEmailServiceStub) ListByEmail(ctx context.Context, email string) ([]*domain.Delivery, error) {
	return s.listByEmailFn(ctx, email)
}

// TrackOpen returns nil for deliveries-by-email handler tests.
func (s deliveriesByEmailServiceStub) TrackOpen(ctx context.Context, deliveryID string) error {
	return nil
}

// TestDeliveriesByEmailEndpoint verifies recipient email list endpoint behavior.
func TestDeliveriesByEmailEndpoint(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(deliveriesByEmailServiceStub{
		listByEmailFn: func(ctx context.Context, email string) ([]*domain.Delivery, error) {
			if email != "user@example.com" {
				t.Fatalf("email = %q, want user@example.com", email)
			}
			return []*domain.Delivery{
				{ID: "delivery-2", Email: "user@example.com"},
				{ID: "delivery-1", Email: "user@example.com"},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := newHTTPServerForEmailHandler(t, handler)
	request, _ := http.NewRequest(http.MethodGet, "/email/deliveries?email=user@example.com", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var rows []domain.Delivery
	if decodeErr := json.NewDecoder(response.Body).Decode(&rows); decodeErr != nil {
		t.Fatalf("decode response error = %v", decodeErr)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
}

// TestDeliveriesByEmailEndpointRejectsEmptyEmail verifies validation mapping behavior for empty email query values.
func TestDeliveriesByEmailEndpointRejectsEmptyEmail(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(deliveriesByEmailServiceStub{
		listByEmailFn: func(ctx context.Context, email string) ([]*domain.Delivery, error) {
			if email != "" {
				t.Fatalf("email = %q, want empty string", email)
			}
			return nil, domain.ErrInvalidEmail
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	server := newHTTPServerForEmailHandler(t, handler)
	request, _ := http.NewRequest(http.MethodGet, "/email/deliveries", nil)
	response, testErr := server.App().Test(request)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
}

// newHTTPServerForEmailHandler creates a test HTTP server for email handlers.
func newHTTPServerForEmailHandler(t *testing.T, handler *Handler) *corehttp.Server {
	t.Helper()

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8188}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	return server
}
