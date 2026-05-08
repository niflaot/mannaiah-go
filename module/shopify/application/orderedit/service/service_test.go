package service

import (
	"context"
	"testing"

	ordersport "mannaiah/module/orders/port"
	shopifyport "mannaiah/module/shopify/port"
)

// TestHandleOrderEventSkipsShopifySyncSource verifies inbound Shopify sync events are not written back to Shopify.
func TestHandleOrderEventSkipsShopifySyncSource(t *testing.T) {
	destination := &orderEditDestinationMock{}
	service, err := NewService(destination, &orderEditLinksMock{}, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	err = service.HandleOrderEvent(context.Background(), ordersport.OrderEventPayload{
		ID:     "order-1",
		Realm:  "shopify",
		Source: syncAuthor,
	})
	if err != nil {
		t.Fatalf("HandleOrderEvent() error = %v", err)
	}
	if destination.applyCalls != 0 || destination.cancelCalls != 0 {
		t.Fatalf("destination calls apply/cancel = %d/%d, want 0/0", destination.applyCalls, destination.cancelCalls)
	}
}

type orderEditDestinationMock struct {
	applyCalls  int
	cancelCalls int
}

func (m *orderEditDestinationMock) ApplyOrderUpdate(context.Context, string, ordersport.OrderEventPayload, shopifyport.ShopifyVariantResolver) error {
	m.applyCalls++
	return nil
}

func (m *orderEditDestinationMock) CancelOrder(context.Context, string, string) error {
	m.cancelCalls++
	return nil
}

type orderEditLinksMock struct{}

func (m *orderEditLinksMock) GetLinkByShopifyID(context.Context, shopifyport.SyncKind, string, string) (*shopifyport.SyncLink, error) {
	return nil, nil
}

func (m *orderEditLinksMock) GetLinkByMannaiahID(context.Context, shopifyport.SyncKind, string) (*shopifyport.SyncLink, error) {
	return nil, nil
}

func (m *orderEditLinksMock) UpsertLink(context.Context, shopifyport.UpsertSyncLinkInput) (*shopifyport.SyncLink, error) {
	return nil, nil
}

func (m *orderEditLinksMock) UpdateLastKnownStatus(context.Context, shopifyport.SyncKind, string, string) error {
	return nil
}
