package messaging

import (
	"context"
	"errors"
	"testing"

	"mannaiah/module/core/messaging/bus"
	ordersport "mannaiah/module/orders/port"
)

// handlerMock defines order event handling behavior for consumer tests.
type handlerMock struct {
	// payload defines captured payload values.
	payload ordersport.OrderEventPayload
	// err defines handling errors.
	err error
}

// HandleOrderEvent captures payload values.
func (m *handlerMock) HandleOrderEvent(ctx context.Context, payload ordersport.OrderEventPayload) error {
	m.payload = payload
	return m.err
}

// registrarMock defines registration behavior for consumer tests.
type registrarMock struct {
	// handlers defines handlers keyed by topic values.
	handlers map[string]bus.Handler
	// addErr defines registration errors.
	addErr error
}

// AddHandler stores handlers by topic.
func (m *registrarMock) AddHandler(topic string, handler bus.Handler) error {
	if m.addErr != nil {
		return m.addErr
	}
	if m.handlers == nil {
		m.handlers = map[string]bus.Handler{}
	}
	m.handlers[topic] = handler
	return nil
}

// TestNewOrderConsumerValidation verifies constructor validation behavior.
func TestNewOrderConsumerValidation(t *testing.T) {
	if _, err := NewOrderConsumer(nil, nil); !errors.Is(err, ErrNilOrderEventHandler) {
		t.Fatalf("NewOrderConsumer(nil) error = %v, want ErrNilOrderEventHandler", err)
	}
}

// TestRegisterValidation verifies registration validation behavior.
func TestRegisterValidation(t *testing.T) {
	consumer, err := NewOrderConsumer(&handlerMock{}, nil)
	if err != nil {
		t.Fatalf("NewOrderConsumer() error = %v", err)
	}
	if err := consumer.Register(nil); !errors.Is(err, ErrNilRegistrar) {
		t.Fatalf("Register(nil) error = %v, want ErrNilRegistrar", err)
	}
}

// TestRegisterAndHandle verifies registration and payload dispatch behavior.
func TestRegisterAndHandle(t *testing.T) {
	handler := &handlerMock{}
	consumer, err := NewOrderConsumer(handler, nil)
	if err != nil {
		t.Fatalf("NewOrderConsumer() error = %v", err)
	}
	registrar := &registrarMock{}
	if err := consumer.Register(registrar); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if len(registrar.handlers) != 3 {
		t.Fatalf("len(registrar.handlers) = %d, want %d", len(registrar.handlers), 3)
	}

	integrationPayload := []byte(`{"identifier":"1001","realm":"woocommerce"}`)
	if err := registrar.handlers[ordersport.TopicOrderUpdated](context.Background(), bus.Message{
		Topic:   ordersport.TopicOrderUpdated,
		Payload: integrationPayload,
	}); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if handler.payload.Identifier != "1001" {
		t.Fatalf("handler.payload.Identifier = %q, want %q", handler.payload.Identifier, "1001")
	}
}

// TestHandleErrors verifies decoding and handler error behavior.
func TestHandleErrors(t *testing.T) {
	handler := &handlerMock{err: errors.New("handle failed")}
	consumer, err := NewOrderConsumer(handler, nil)
	if err != nil {
		t.Fatalf("NewOrderConsumer() error = %v", err)
	}
	registrar := &registrarMock{}
	if err := consumer.Register(registrar); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if err := registrar.handlers[ordersport.TopicOrderCreated](context.Background(), bus.Message{
		Topic:   ordersport.TopicOrderCreated,
		Payload: []byte("invalid json"),
	}); err != nil {
		t.Fatalf("handler() error = %v, want nil", err)
	}
	if err := registrar.handlers[ordersport.TopicOrderCreated](context.Background(), bus.Message{
		Topic:   ordersport.TopicOrderCreated,
		Payload: []byte(`{"identifier":"1001","realm":"woocommerce"}`),
	}); err != nil {
		t.Fatalf("handler() error = %v, want nil", err)
	}
}
