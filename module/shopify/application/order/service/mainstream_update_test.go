package service

import (
	"context"
	"testing"

	"go.uber.org/zap"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
	shopifyport "mannaiah/module/shopify/port"
)

type destinationStub struct {
	calls       int
	lastShopify string
	lastStatus  ordersdomain.Status
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

type linkRepositoryStub struct {
	link            *shopifyport.SyncLink
	updatedMannaiah string
	updatedStatus   string
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
	_ = kind
	_ = mannaiahID
	return s.link, nil
}

func (s *linkRepositoryStub) UpsertLink(ctx context.Context, input shopifyport.UpsertSyncLinkInput) (*shopifyport.SyncLink, error) {
	_ = ctx
	_ = input
	return nil, nil
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
		links := &linkRepositoryStub{link: &shopifyport.SyncLink{ShopifyID: "321", MannaiahID: "ord-1"}}
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
		links := &linkRepositoryStub{link: &shopifyport.SyncLink{ShopifyID: "321", MannaiahID: "ord-1", LastKnownStatus: "PENDING"}}
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
}
