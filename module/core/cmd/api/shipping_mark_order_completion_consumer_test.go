package main

import (
	"context"
	"errors"
	"testing"

	coremsgbus "mannaiah/module/core/messaging/bus"
	coremsgplatform "mannaiah/module/core/messaging/platform"
	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
	shippingport "mannaiah/module/shipping/port"
)

// orderCompletionRegistrarMock defines topic-handler registration behavior for consumer tests.
type orderCompletionRegistrarMock struct {
	// handlers defines handlers keyed by topic values.
	handlers map[string]coremsgbus.Handler
}

// AddHandler stores handlers by topic.
func (m *orderCompletionRegistrarMock) AddHandler(topic string, handler coremsgbus.Handler) error {
	if m.handlers == nil {
		m.handlers = map[string]coremsgbus.Handler{}
	}
	m.handlers[topic] = handler
	return nil
}

// orderCompletionServiceMock defines order-status behavior for consumer tests.
type orderCompletionServiceMock struct {
	// order defines loaded order values.
	order *ordersdomain.Order
	// getErr defines load errors.
	getErr error
	// updateErr defines update errors.
	updateErr error
	// updated defines captured update command values.
	updated ordersapplication.UpdateStatusCommand
	// updatedID defines captured update order-id values.
	updatedID string
}

// Get resolves order values by id.
func (m *orderCompletionServiceMock) Get(ctx context.Context, id string) (*ordersdomain.Order, error) {
	_ = ctx
	_ = id
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.order, nil
}

// UpdateStatus captures status update command values.
func (m *orderCompletionServiceMock) UpdateStatus(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error) {
	_ = ctx
	m.updatedID = id
	m.updated = command
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.order == nil {
		return nil, nil
	}
	copy := *m.order
	copy.CurrentStatus = command.Status
	return &copy, nil
}

// TestRegisterShippingMarkOrderCompletionConsumer verifies registration and order completion behavior.
func TestRegisterShippingMarkOrderCompletionConsumer(t *testing.T) {
	registrar := &orderCompletionRegistrarMock{}
	service := &orderCompletionServiceMock{
		order: &ordersdomain.Order{ID: "order-1", CurrentStatus: ordersdomain.StatusCreated},
	}
	if err := registerShippingMarkOrderCompletionConsumer(registrar, service, nil); err != nil {
		t.Fatalf("registerShippingMarkOrderCompletionConsumer() error = %v", err)
	}

	handler := registrar.handlers[shippingport.TopicMarkGenerated]
	if handler == nil {
		t.Fatalf("missing handler for %q", shippingport.TopicMarkGenerated)
	}
	voidedHandler := registrar.handlers[shippingport.TopicMarkVoided]
	if voidedHandler == nil {
		t.Fatalf("missing handler for %q", shippingport.TopicMarkVoided)
	}

	if err := handler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkGenerated,
		Payload: []byte(`{"markId":"mark-1","orderId":"order-1"}`),
	}); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if service.updatedID != "order-1" {
		t.Fatalf("service.updatedID = %q, want %q", service.updatedID, "order-1")
	}
	if service.updated.Status != ordersdomain.StatusCompleted {
		t.Fatalf("service.updated.Status = %q, want %q", service.updated.Status, ordersdomain.StatusCompleted)
	}

	if err := voidedHandler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkVoided,
		Payload: []byte(`{"markId":"mark-1","orderId":"order-1","trackingNumber":"6039"}`),
	}); err != nil {
		t.Fatalf("voidedHandler() error = %v", err)
	}
	if service.updated.Status != ordersdomain.StatusCreated {
		t.Fatalf("service.updated.Status = %q, want %q", service.updated.Status, ordersdomain.StatusCreated)
	}
	if service.updated.Source != "shipping_mark_voided" {
		t.Fatalf("service.updated.Source = %q, want %q", service.updated.Source, "shipping_mark_voided")
	}
	if service.updated.Description != "order returned to created after shipping mark voided: 6039" {
		t.Fatalf("service.updated.Description = %q", service.updated.Description)
	}
}

// TestRegisterShippingMarkOrderCompletionConsumerNoopPaths verifies non-failing paths for already-completed and not-found orders.
func TestRegisterShippingMarkOrderCompletionConsumerNoopPaths(t *testing.T) {
	registrar := &orderCompletionRegistrarMock{}
	service := &orderCompletionServiceMock{
		order: &ordersdomain.Order{ID: "order-1", CurrentStatus: ordersdomain.StatusCompleted},
	}
	if err := registerShippingMarkOrderCompletionConsumer(registrar, service, nil); err != nil {
		t.Fatalf("registerShippingMarkOrderCompletionConsumer() error = %v", err)
	}
	handler := registrar.handlers[shippingport.TopicMarkGenerated]
	voidedHandler := registrar.handlers[shippingport.TopicMarkVoided]
	if voidedHandler == nil {
		t.Fatalf("missing handler for %q", shippingport.TopicMarkVoided)
	}
	if err := handler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkGenerated,
		Payload: []byte(`{"markId":"mark-1","orderId":"order-1"}`),
	}); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if service.updatedID != "" {
		t.Fatalf("service.updatedID = %q, want empty", service.updatedID)
	}

	service.getErr = ordersport.ErrNotFound
	if err := handler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkGenerated,
		Payload: []byte(`{"markId":"mark-1","orderId":"order-1"}`),
	}); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if err := voidedHandler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkVoided,
		Payload: []byte(`{"markId":"mark-1","orderId":"order-1"}`),
	}); err != nil {
		t.Fatalf("voidedHandler() error = %v", err)
	}
}

// TestRegisterShippingMarkOrderCompletionConsumerErrors verifies retriable and non-retriable error paths.
func TestRegisterShippingMarkOrderCompletionConsumerErrors(t *testing.T) {
	registrar := &orderCompletionRegistrarMock{}
	service := &orderCompletionServiceMock{
		order: &ordersdomain.Order{ID: "order-1", CurrentStatus: ordersdomain.StatusCreated},
	}
	if err := registerShippingMarkOrderCompletionConsumer(registrar, service, nil); err != nil {
		t.Fatalf("registerShippingMarkOrderCompletionConsumer() error = %v", err)
	}
	handler := registrar.handlers[shippingport.TopicMarkGenerated]
	voidedHandler := registrar.handlers[shippingport.TopicMarkVoided]
	if voidedHandler == nil {
		t.Fatalf("missing handler for %q", shippingport.TopicMarkVoided)
	}

	err := handler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkGenerated,
		Payload: []byte(`invalid`),
	})
	if !coremsgplatform.IsNonRetriable(err) {
		t.Fatalf("handler(invalid json) error = %v, want non-retriable", err)
	}

	service.updateErr = errors.New("db down")
	err = handler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkGenerated,
		Payload: []byte(`{"markId":"mark-1","orderId":"order-1"}`),
	})
	if err == nil {
		t.Fatalf("handler() error = nil, want non-nil")
	}

	service.updateErr = errors.New("db down")
	err = voidedHandler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkVoided,
		Payload: []byte(`{"markId":"mark-1","orderId":"order-1","trackingNumber":"6039"}`),
	})
	if err == nil {
		t.Fatalf("voidedHandler() error = nil, want non-nil")
	}
}
