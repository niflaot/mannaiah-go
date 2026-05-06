package service

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	contactsapplication "mannaiah/module/contacts/application"
	contactsdomain "mannaiah/module/contacts/domain"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
	shopifyport "mannaiah/module/shopify/port"
)

type destinationStub struct {
	calls             int
	createCalls       int
	lastShopify       string
	lastStatus        ordersdomain.Status
	lastCreateCommand shopifyport.MainstreamOrderCreateCommand
}

func (d *destinationStub) Validate(ctx context.Context) error {
	_ = ctx
	return nil
}

func (d *destinationStub) UpdateOrderFromMainstream(ctx context.Context, shopifyID string, command shopifyport.MainstreamOrderUpdateCommand) error {
	_ = ctx
	d.calls++
	d.lastShopify = shopifyID
	d.lastStatus = command.Status
	return nil
}

func (d *destinationStub) CreateOrderFromMainstream(ctx context.Context, command shopifyport.MainstreamOrderCreateCommand) (shopifyport.ShopifyOrder, error) {
	_ = ctx
	d.createCalls++
	d.lastCreateCommand = command
	return shopifyport.ShopifyOrder{ShopDomain: "flock-6591.myshopify.com", ID: "shop-order-1"}, nil
}

type linkRepositoryStub struct {
	orderLink       *shopifyport.SyncLink
	contactLink     *shopifyport.SyncLink
	upserted        *shopifyport.UpsertSyncLinkInput
	updatedMannaiah string
	updatedStatus   string
}

type orderContactSourceStub struct {
	contact *contactsdomain.Contact
}

func (s orderContactSourceStub) Get(ctx context.Context, id string) (*contactsdomain.Contact, error) {
	_ = ctx
	_ = id
	return s.contact, nil
}

type orderContactHandlerStub struct {
	calls int
	links *linkRepositoryStub
}

func (s *orderContactHandlerStub) HandleContactEvent(ctx context.Context, payload contactsapplication.ContactEventPayload) error {
	_ = ctx
	s.calls++
	if s.links != nil {
		s.links.contactLink = &shopifyport.SyncLink{ShopDomain: "flock-6591.myshopify.com", ShopifyID: "created-customer-1", MannaiahID: payload.ID}
	}
	return nil
}

func (s *linkRepositoryStub) GetLinkByShopifyID(ctx context.Context, kind shopifyport.SyncKind, shopDomain string, shopifyID string) (*shopifyport.SyncLink, error) {
	_ = ctx
	_ = kind
	_ = shopDomain
	_ = shopifyID
	return nil, nil
}

func (s *linkRepositoryStub) GetLinkByMannaiahID(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string) (*shopifyport.SyncLink, error) {
	_ = ctx
	_ = mannaiahID
	switch kind {
	case shopifyport.SyncKindOrder:
		return s.orderLink, nil
	case shopifyport.SyncKindContact:
		return s.contactLink, nil
	default:
		return nil, nil
	}
}

func (s *linkRepositoryStub) UpsertLink(ctx context.Context, input shopifyport.UpsertSyncLinkInput) (*shopifyport.SyncLink, error) {
	_ = ctx
	copied := input
	s.upserted = &copied
	return &shopifyport.SyncLink{
		Kind:            input.Kind,
		ShopDomain:      input.ShopDomain,
		ShopifyID:       input.ShopifyID,
		MannaiahID:      input.MannaiahID,
		LastKnownStatus: input.LastKnownStatus,
		LastSyncedAt:    input.LastSyncedAt,
	}, nil
}

func (s *linkRepositoryStub) UpdateLastKnownStatus(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string, status string) error {
	_ = ctx
	_ = kind
	s.updatedMannaiah = mannaiahID
	s.updatedStatus = status
	return nil
}

// TestMainstreamUpdateServiceHandleOrderEvent verifies outbound status pushes and loop guards.
func TestMainstreamUpdateServiceHandleOrderEvent(t *testing.T) {
	t.Run("ignores shopify-originated events", func(t *testing.T) {
		destination := &destinationStub{}
		links := &linkRepositoryStub{orderLink: &shopifyport.SyncLink{ShopifyID: "321", MannaiahID: "ord-1"}}
		service, err := NewMainstreamUpdateService(destination, links, zap.NewNop())
		if err != nil {
			t.Fatalf("NewMainstreamUpdateService() error = %v", err)
		}

		err = service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{ID: "ord-1", Realm: "shopify", Source: "shopify_manual", CurrentStatus: "COMPLETED"})
		if err != nil {
			t.Fatalf("HandleOrderEvent() error = %v", err)
		}
		if destination.calls != 0 {
			t.Fatalf("destination calls = %d, want 0", destination.calls)
		}
	})

	t.Run("pushes new mainstream status", func(t *testing.T) {
		destination := &destinationStub{}
		links := &linkRepositoryStub{orderLink: &shopifyport.SyncLink{ShopDomain: "flock-6591.myshopify.com", ShopifyID: "321", MannaiahID: "ord-1", LastKnownStatus: "PENDING"}}
		service, err := NewMainstreamUpdateService(destination, links, zap.NewNop())
		if err != nil {
			t.Fatalf("NewMainstreamUpdateService() error = %v", err)
		}

		err = service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{ID: "ord-1", Realm: "shopify", Source: "api", CurrentStatus: "COMPLETED"})
		if err != nil {
			t.Fatalf("HandleOrderEvent() error = %v", err)
		}
		if destination.calls != 1 {
			t.Fatalf("destination calls = %d, want 1", destination.calls)
		}
		if destination.lastShopify != "321" {
			t.Fatalf("destination shopify id = %q, want 321", destination.lastShopify)
		}
		if destination.lastStatus != ordersdomain.StatusCompleted {
			t.Fatalf("destination status = %q, want COMPLETED", destination.lastStatus)
		}
		if links.updatedMannaiah != "ord-1" || links.updatedStatus != "COMPLETED" {
			t.Fatalf("updated link = (%q, %q), want (ord-1, COMPLETED)", links.updatedMannaiah, links.updatedStatus)
		}
	})

	t.Run("creates missing shopify order when contact link exists", func(t *testing.T) {
		destination := &destinationStub{}
		links := &linkRepositoryStub{contactLink: &shopifyport.SyncLink{ShopDomain: "flock-6591.myshopify.com", ShopifyID: "cust-1", MannaiahID: "contact-1"}}
		service, err := NewMainstreamUpdateService(destination, links, zap.NewNop())
		if err != nil {
			t.Fatalf("NewMainstreamUpdateService() error = %v", err)
		}

		createdAt := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
		err = service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{
			ID:            "ord-2",
			Identifier:    "M-1002",
			Realm:         "shopify",
			Source:        "api",
			ContactID:     "contact-1",
			CurrentStatus: "PENDING",
			Items: []ordersport.OrderEventItem{{
				SKU:           "sku-1",
				AlternateName: "Product 1",
				Quantity:      2,
				Value:         15.5,
			}},
			ShippingCharges: []ordersport.OrderEventShippingCharge{{
				MethodID:    "flat_rate",
				MethodTitle: "Flat Rate",
				Price:       5,
			}},
			CreatedAt: createdAt,
		})
		if err != nil {
			t.Fatalf("HandleOrderEvent() error = %v", err)
		}
		if destination.createCalls != 1 {
			t.Fatalf("create calls = %d, want 1", destination.createCalls)
		}
		if destination.lastCreateCommand.CustomerID != "cust-1" {
			t.Fatalf("create customer = %q, want cust-1", destination.lastCreateCommand.CustomerID)
		}
		if len(destination.lastCreateCommand.Items) != 1 || destination.lastCreateCommand.Items[0].SKU != "sku-1" {
			t.Fatalf("create items = %#v, want sku-1", destination.lastCreateCommand.Items)
		}
		if destination.lastCreateCommand.CreatedAt != createdAt {
			t.Fatalf("create CreatedAt = %v, want %v", destination.lastCreateCommand.CreatedAt, createdAt)
		}
		if links.upserted == nil {
			t.Fatalf("upserted link = nil, want order link")
		}
		if links.upserted.Kind != shopifyport.SyncKindOrder || links.upserted.ShopifyID != "shop-order-1" || links.upserted.MannaiahID != "ord-2" || links.upserted.LastKnownStatus != "PENDING" {
			t.Fatalf("upserted link = %#v, want created order link", links.upserted)
		}
	})

	t.Run("skips missing shopify order when contact link is missing", func(t *testing.T) {
		destination := &destinationStub{}
		links := &linkRepositoryStub{}
		service, err := NewMainstreamUpdateService(destination, links, zap.NewNop())
		if err != nil {
			t.Fatalf("NewMainstreamUpdateService() error = %v", err)
		}

		err = service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{ID: "ord-3", Realm: "shopify", Source: "api", ContactID: "contact-404", CurrentStatus: "PENDING"})
		if err != nil {
			t.Fatalf("HandleOrderEvent() error = %v", err)
		}
		if destination.createCalls != 0 {
			t.Fatalf("create calls = %d, want 0", destination.createCalls)
		}
		if links.upserted != nil {
			t.Fatalf("upserted link = %#v, want nil", links.upserted)
		}
	})

	t.Run("creates missing contact before creating missing shopify order", func(t *testing.T) {
		destination := &destinationStub{}
		links := &linkRepositoryStub{}
		contactHandler := &orderContactHandlerStub{links: links}
		service, err := NewMainstreamUpdateService(destination, links, zap.NewNop())
		if err != nil {
			t.Fatalf("NewMainstreamUpdateService() error = %v", err)
		}
		service.SetContactResolver(orderContactSourceStub{contact: &contactsdomain.Contact{
			ID:        "contact-10",
			FirstName: "Grace",
			LastName:  "Hopper",
			Email:     "grace@example.com",
		}}, contactHandler)

		err = service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{
			ID:            "ord-10",
			Identifier:    "M-1010",
			Realm:         "shopify",
			Source:        "api",
			ContactID:     "contact-10",
			CurrentStatus: "PENDING",
			Items:         []ordersport.OrderEventItem{{SKU: "sku-10", Quantity: 1, Value: 10}},
		})
		if err != nil {
			t.Fatalf("HandleOrderEvent() error = %v", err)
		}
		if contactHandler.calls != 1 {
			t.Fatalf("contact pre-sync calls = %d, want 1", contactHandler.calls)
		}
		if destination.createCalls != 1 {
			t.Fatalf("order create calls = %d, want 1", destination.createCalls)
		}
		if destination.lastCreateCommand.CustomerID != "created-customer-1" {
			t.Fatalf("create customer = %q, want created-customer-1", destination.lastCreateCommand.CustomerID)
		}
	})
}
