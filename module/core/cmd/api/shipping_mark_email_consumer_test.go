package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	contactdomain "mannaiah/module/contacts/domain"
	coremsgbus "mannaiah/module/core/messaging/bus"
	coremsgplatform "mannaiah/module/core/messaging/platform"
	emailapplication "mannaiah/module/email/application"
	emaildomain "mannaiah/module/email/domain"
	ordersdomain "mannaiah/module/orders/domain"
	productdomain "mannaiah/module/products/domain/product"
	variationdomain "mannaiah/module/products/domain/variation"
	shippingdomain "mannaiah/module/shipping/domain"
	shippingport "mannaiah/module/shipping/port"
)

// shippingEmailRegistrarMock defines topic-handler registration behavior for shipping email consumer tests.
type shippingEmailRegistrarMock struct {
	// handlers defines handlers keyed by topic values.
	handlers map[string]coremsgbus.Handler
}

// AddHandler stores handlers by topic.
func (m *shippingEmailRegistrarMock) AddHandler(topic string, handler coremsgbus.Handler) error {
	if m.handlers == nil {
		m.handlers = map[string]coremsgbus.Handler{}
	}
	m.handlers[topic] = handler
	return nil
}

// shippingEmailMarkServiceMock defines mark lookup behavior for shipping email consumer tests.
type shippingEmailMarkServiceMock struct {
	// mark defines returned mark values.
	mark *shippingdomain.ShippingMark
}

// Get resolves one shipping mark by id.
func (m shippingEmailMarkServiceMock) Get(ctx context.Context, id string) (*shippingdomain.ShippingMark, error) {
	_ = ctx
	_ = id
	return m.mark, nil
}

// shippingEmailCarrierServiceMock defines carrier lookup behavior for shipping email consumer tests.
type shippingEmailCarrierServiceMock struct {
	// carrier defines returned carrier values.
	carrier *shippingdomain.Carrier
}

// Get resolves one carrier by id.
func (m shippingEmailCarrierServiceMock) Get(ctx context.Context, id string) (*shippingdomain.Carrier, error) {
	_ = ctx
	_ = id
	return m.carrier, nil
}

// shippingEmailOrderServiceMock defines order lookup behavior for shipping email consumer tests.
type shippingEmailOrderServiceMock struct {
	// order defines returned order values.
	order *ordersdomain.Order
}

// Get resolves one order by id.
func (m shippingEmailOrderServiceMock) Get(ctx context.Context, id string) (*ordersdomain.Order, error) {
	_ = ctx
	_ = id
	return m.order, nil
}

// shippingEmailContactServiceMock defines contact lookup behavior for shipping email consumer tests.
type shippingEmailContactServiceMock struct {
	// contact defines returned contact values.
	contact *contactdomain.Contact
}

// Get resolves one contact by id.
func (m shippingEmailContactServiceMock) Get(ctx context.Context, id string) (*contactdomain.Contact, error) {
	_ = ctx
	_ = id
	return m.contact, nil
}

// shippingEmailProductServiceMock defines product lookup behavior for shipping email consumer tests.
type shippingEmailProductServiceMock struct {
	// product defines returned product values.
	product *productdomain.Product
}

// GetBySKU resolves one product by sku.
func (m shippingEmailProductServiceMock) GetBySKU(ctx context.Context, sku string) (*productdomain.Product, error) {
	_ = ctx
	_ = sku
	return m.product, nil
}

// shippingEmailVariationServiceMock defines variation lookup behavior for shipping email consumer tests.
type shippingEmailVariationServiceMock struct {
	// variation defines returned variation values.
	variation *variationdomain.Variation
}

// Get resolves one variation by id.
func (m shippingEmailVariationServiceMock) Get(ctx context.Context, id string) (*variationdomain.Variation, error) {
	_ = ctx
	_ = id
	return m.variation, nil
}

// shippingEmailSenderMock defines email send behavior for shipping email consumer tests.
type shippingEmailSenderMock struct {
	// command defines captured send command values.
	command emailapplication.SendCommand
	// sendErr defines send errors.
	sendErr error
}

// Send captures send command values.
func (m *shippingEmailSenderMock) Send(ctx context.Context, command emailapplication.SendCommand) (*emaildomain.Delivery, error) {
	_ = ctx
	m.command = command
	if m.sendErr != nil {
		return nil, m.sendErr
	}
	return &emaildomain.Delivery{ID: "delivery-1"}, nil
}

// TestRegisterShippingMarkTransactionalEmailConsumer verifies transactional shipping email dispatch behavior.
func TestRegisterShippingMarkTransactionalEmailConsumer(t *testing.T) {
	registrar := &shippingEmailRegistrarMock{}
	renderer, err := newShippingTemplateRenderer()
	if err != nil {
		t.Fatalf("newShippingTemplateRenderer() error = %v", err)
	}
	sender := &shippingEmailSenderMock{}
	deps := shippingEmailConsumerDependencies{
		marks:    shippingEmailMarkServiceMock{mark: &shippingdomain.ShippingMark{ID: "mark-1", OrderID: "order-1", CarrierID: "tcc", TrackingNumber: "6039", DocumentRef: "DOC-1", Recipient: shippingdomain.Address{Email: "fallback@example.com"}}},
		carriers: shippingEmailCarrierServiceMock{carrier: &shippingdomain.Carrier{ID: "tcc", Name: "TCC"}},
		orders: shippingEmailOrderServiceMock{order: &ordersdomain.Order{
			ID:            "order-1",
			Identifier:    "601205",
			ContactID:     "contact-1",
			PaymentMethod: "contraentrega",
			ShippingAddress: ordersdomain.ShippingAddress{
				Address:  "Calle 1",
				Address2: "Apto 2",
				CityCode: "11001",
			},
			Items: []ordersdomain.Item{{SKU: "SKU-1", AlternateName: "Producto fallback", Quantity: 2}},
		}},
		contacts: shippingEmailContactServiceMock{contact: &contactdomain.Contact{
			ID:           "contact-1",
			FirstName:    "Juliana",
			LastName:     "Villegas",
			Email:        "juliana@example.com",
			Address:      "Calle 9",
			AddressExtra: "Casa",
			CityCode:     "11001",
		}},
		products: shippingEmailProductServiceMock{product: &productdomain.Product{
			ID:  "product-1",
			SKU: "SKU-1",
			Datasheets: []productdomain.Datasheet{
				{Realm: "default", Name: "Morral Dream Nubuk"},
			},
			Gallery: []productdomain.GalleryItem{
				{AssetID: "https://cdn.example.com/product-1.jpg", IsMain: true},
			},
			Variants: []productdomain.Variant{{SKU: "SKU-1", VariationIDs: []string{"var-1"}}},
		}},
		variations:       variationServiceAdapter{service: shippingEmailVariationServiceMock{variation: &variationdomain.Variation{ID: "var-1", Name: "Oliva"}}},
		emails:           sender,
		assetResolver:    analyticsAssetURLResolver{},
		templateRenderer: renderer,
	}
	if err := registerShippingMarkTransactionalEmailConsumer(registrar, deps, nil); err != nil {
		t.Fatalf("registerShippingMarkTransactionalEmailConsumer() error = %v", err)
	}
	handler := registrar.handlers[shippingport.TopicMarkGenerated]
	if handler == nil {
		t.Fatalf("missing handler for %q", shippingport.TopicMarkGenerated)
	}
	if registrar.handlers[shippingport.TopicMarkVoided] == nil {
		t.Fatalf("missing handler for %q", shippingport.TopicMarkVoided)
	}
	if err := handler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkGenerated,
		Payload: []byte(`{"markId":"mark-1","orderId":"order-1"}`),
	}); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if sender.command.Email != "juliana@example.com" {
		t.Fatalf("sender.command.Email = %q, want %q", sender.command.Email, "juliana@example.com")
	}
	if sender.command.IdempotencyKey != "shipping_mark_dispatched:mark-1" {
		t.Fatalf("sender.command.IdempotencyKey = %q", sender.command.IdempotencyKey)
	}
	if !strings.Contains(sender.command.HTMLBody, "https://rastreo.flockstore.co") {
		t.Fatalf("HTMLBody should contain tracking URL")
	}
	if !strings.Contains(sender.command.HTMLBody, "https://wa.me/573104314990") {
		t.Fatalf("HTMLBody should contain help URL")
	}
}

// TestRegisterShippingMarkTransactionalEmailConsumerVoided verifies transactional shipping mark-voided email dispatch behavior.
func TestRegisterShippingMarkTransactionalEmailConsumerVoided(t *testing.T) {
	registrar := &shippingEmailRegistrarMock{}
	renderer, err := newShippingTemplateRenderer()
	if err != nil {
		t.Fatalf("newShippingTemplateRenderer() error = %v", err)
	}
	sender := &shippingEmailSenderMock{}
	deps := shippingEmailConsumerDependencies{
		marks:    shippingEmailMarkServiceMock{mark: &shippingdomain.ShippingMark{ID: "mark-void-1", OrderID: "order-1", CarrierID: "tcc", TrackingNumber: "6039", DocumentRef: "DOC-1", Recipient: shippingdomain.Address{Email: "fallback@example.com"}}},
		carriers: shippingEmailCarrierServiceMock{carrier: &shippingdomain.Carrier{ID: "tcc", Name: "TCC"}},
		orders: shippingEmailOrderServiceMock{order: &ordersdomain.Order{
			ID:            "order-1",
			Identifier:    "601205",
			ContactID:     "contact-1",
			PaymentMethod: "contraentrega",
			Items:         []ordersdomain.Item{{SKU: "SKU-1", Quantity: 1}},
		}},
		contacts:         shippingEmailContactServiceMock{contact: &contactdomain.Contact{ID: "contact-1", FirstName: "Juliana", LastName: "Villegas", Email: "juliana@example.com"}},
		emails:           sender,
		templateRenderer: renderer,
	}
	if err := registerShippingMarkTransactionalEmailConsumer(registrar, deps, nil); err != nil {
		t.Fatalf("registerShippingMarkTransactionalEmailConsumer() error = %v", err)
	}
	handler := registrar.handlers[shippingport.TopicMarkVoided]
	if handler == nil {
		t.Fatalf("missing handler for %q", shippingport.TopicMarkVoided)
	}
	if err := handler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkVoided,
		Payload: []byte(`{"markId":"mark-void-1","orderId":"order-1","trackingNumber":"6039"}`),
	}); err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	if sender.command.Subject != "Actualizacion de envio de tu pedido #601205" {
		t.Fatalf("sender.command.Subject = %q", sender.command.Subject)
	}
	if sender.command.IdempotencyKey != "shipping_mark_voided:mark-void-1" {
		t.Fatalf("sender.command.IdempotencyKey = %q", sender.command.IdempotencyKey)
	}
	if strings.Contains(sender.command.HTMLBody, "RASTREAR PEDIDO") {
		t.Fatalf("HTMLBody should not contain tracking button")
	}
	if !strings.Contains(sender.command.HTMLBody, "se generó una guía de envío para tu pedido") {
		t.Fatalf("HTMLBody should include voided-guide content")
	}
	if !strings.Contains(sender.command.TextBody, "ha tenido que ser anulada") {
		t.Fatalf("TextBody should include voided-guide content")
	}
}

// TestRegisterShippingMarkTransactionalEmailConsumerErrorPaths verifies non-retriable decode and duplicate idempotency paths.
func TestRegisterShippingMarkTransactionalEmailConsumerErrorPaths(t *testing.T) {
	registrar := &shippingEmailRegistrarMock{}
	renderer, err := newShippingTemplateRenderer()
	if err != nil {
		t.Fatalf("newShippingTemplateRenderer() error = %v", err)
	}
	sender := &shippingEmailSenderMock{sendErr: errors.New("duplicate key value violates unique constraint uq_email_deliveries_idempotency_key")}
	deps := shippingEmailConsumerDependencies{
		marks:            shippingEmailMarkServiceMock{mark: &shippingdomain.ShippingMark{ID: "mark-1", OrderID: "order-1", CarrierID: "tcc", TrackingNumber: "6039", Recipient: shippingdomain.Address{Email: "fallback@example.com"}}},
		orders:           shippingEmailOrderServiceMock{order: &ordersdomain.Order{ID: "order-1", Identifier: "601205", ContactID: "contact-1", Items: []ordersdomain.Item{{SKU: "SKU-1", Quantity: 1}}}},
		contacts:         shippingEmailContactServiceMock{contact: &contactdomain.Contact{ID: "contact-1", Email: "juliana@example.com", FirstName: "Juliana", LastName: "Villegas"}},
		emails:           sender,
		templateRenderer: renderer,
	}
	if err := registerShippingMarkTransactionalEmailConsumer(registrar, deps, nil); err != nil {
		t.Fatalf("registerShippingMarkTransactionalEmailConsumer() error = %v", err)
	}
	handler := registrar.handlers[shippingport.TopicMarkGenerated]
	voidedHandler := registrar.handlers[shippingport.TopicMarkVoided]
	if voidedHandler == nil {
		t.Fatalf("missing handler for %q", shippingport.TopicMarkVoided)
	}
	if err := handler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkGenerated,
		Payload: []byte(`invalid`),
	}); !coremsgplatform.IsNonRetriable(err) {
		t.Fatalf("handler(invalid payload) error = %v, want non-retriable", err)
	}
	if err := handler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkGenerated,
		Payload: []byte(`{"markId":"mark-1","orderId":"order-1"}`),
	}); err != nil {
		t.Fatalf("handler(duplicate idempotency) error = %v, want nil", err)
	}
	if err := voidedHandler(context.Background(), coremsgbus.Message{
		Topic:   shippingport.TopicMarkVoided,
		Payload: []byte(`{"markId":"mark-1","orderId":"order-1"}`),
	}); err != nil {
		t.Fatalf("voidedHandler(duplicate idempotency) error = %v, want nil", err)
	}
}

// variationServiceAdapter wraps variation mock values to satisfy interface assignment in tests.
type variationServiceAdapter struct {
	// service defines variation lookup dependencies.
	service shippingEmailVariationService
}

// Get resolves one variation by identifier.
func (a variationServiceAdapter) Get(ctx context.Context, id string) (*variationdomain.Variation, error) {
	return a.service.Get(ctx, id)
}
