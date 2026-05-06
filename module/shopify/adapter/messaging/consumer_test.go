package messaging

import (
	"context"
	"errors"
	"testing"

	contactsapplication "mannaiah/module/contacts/application"
	"mannaiah/module/core/messaging/bus"
	ordersport "mannaiah/module/orders/port"
)

type contactHandlerStub struct {
	err error
}

func (s contactHandlerStub) HandleContactEvent(ctx context.Context, payload contactsapplication.ContactEventPayload) error {
	_ = ctx
	_ = payload
	return s.err
}

type orderHandlerStub struct {
	err error
}

func (s orderHandlerStub) HandleOrderEvent(ctx context.Context, payload ordersport.OrderEventPayload) error {
	_ = ctx
	_ = payload
	return s.err
}

// TestContactConsumerDefersTemporaryShopifyUnavailable verifies breaker-open events are not retried immediately.
func TestContactConsumerDefersTemporaryShopifyUnavailable(t *testing.T) {
	consumer, err := NewContactConsumer(contactHandlerStub{err: errors.New("shopify integration is unavailable")}, nil)
	if err != nil {
		t.Fatalf("NewContactConsumer() error = %v", err)
	}

	handleErr := consumer.handleMessage(context.Background(), contactsapplication.TopicContactUpdated, bus.Message{
		ID:      "message-1",
		Payload: []byte(`{"id":"contact-1","email":"buyer@example.com"}`),
	})
	if handleErr != nil {
		t.Fatalf("handleMessage() error = %v", handleErr)
	}
}

// TestOrderConsumerDefersTemporaryShopifyUnavailable verifies breaker-open events are not retried immediately.
func TestOrderConsumerDefersTemporaryShopifyUnavailable(t *testing.T) {
	consumer, err := NewOrderConsumer(orderHandlerStub{err: errors.New("shopify integration is unavailable")}, nil)
	if err != nil {
		t.Fatalf("NewOrderConsumer() error = %v", err)
	}

	handleErr := consumer.handleMessage(context.Background(), ordersport.TopicOrderUpdated, bus.Message{
		ID:      "message-1",
		Payload: []byte(`{"id":"order-1","realm":"shopify","current_status":"pending"}`),
	})
	if handleErr != nil {
		t.Fatalf("handleMessage() error = %v", handleErr)
	}
}
