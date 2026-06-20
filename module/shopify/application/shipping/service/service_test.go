package service

import (
	"context"
	"testing"

	shopifyport "mannaiah/module/shopify/port"
)

// TestHandleMarkGeneratedUsesTrackingCompany verifies manual carrier labels are written to Shopify fulfillments.
func TestHandleMarkGeneratedUsesTrackingCompany(t *testing.T) {
	destination := &shippingDestinationMock{}
	links := &shippingLinksMock{
		byMannaiah: map[shopifyport.SyncKind]map[string]*shopifyport.SyncLink{
			shopifyport.SyncKindOrder: {
				"order-1": {
					Kind:       shopifyport.SyncKindOrder,
					ShopDomain: "store.myshopify.com",
					ShopifyID:  "shopify-order-1",
					MannaiahID: "order-1",
				},
			},
		},
	}
	service, err := NewService(destination, links, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	err = service.HandleMarkGenerated(context.Background(), MarkGeneratedPayload{
		MarkID:          "mark-1",
		OrderID:         "order-1",
		CarrierID:       "manual",
		TrackingCompany: "mensajerosurbanos",
		TrackingNumber:  "TRACK-1",
	})
	if err != nil {
		t.Fatalf("HandleMarkGenerated() error = %v", err)
	}
	if destination.fulfillCalls != 1 {
		t.Fatalf("destination.fulfillCalls = %d, want 1", destination.fulfillCalls)
	}
	if destination.lastInput.TrackingCompany != "mensajerosurbanos" {
		t.Fatalf("destination.lastInput.TrackingCompany = %q, want %q", destination.lastInput.TrackingCompany, "mensajerosurbanos")
	}
}

// TestHandleMarkGeneratedFallsBackToCarrierID verifies older events still use the carrier id.
func TestHandleMarkGeneratedFallsBackToCarrierID(t *testing.T) {
	destination := &shippingDestinationMock{}
	links := &shippingLinksMock{
		byMannaiah: map[shopifyport.SyncKind]map[string]*shopifyport.SyncLink{
			shopifyport.SyncKindOrder: {
				"order-1": {
					Kind:       shopifyport.SyncKindOrder,
					ShopDomain: "store.myshopify.com",
					ShopifyID:  "shopify-order-1",
					MannaiahID: "order-1",
				},
			},
		},
	}
	service, err := NewService(destination, links, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	err = service.HandleMarkGenerated(context.Background(), MarkGeneratedPayload{
		MarkID:         "mark-1",
		OrderID:        "order-1",
		CarrierID:      "tcc",
		TrackingNumber: "TRACK-1",
	})
	if err != nil {
		t.Fatalf("HandleMarkGenerated() error = %v", err)
	}
	if destination.lastInput.TrackingCompany != "tcc" {
		t.Fatalf("destination.lastInput.TrackingCompany = %q, want %q", destination.lastInput.TrackingCompany, "tcc")
	}
}

// shippingDestinationMock defines Shopify fulfillment behavior for tests.
type shippingDestinationMock struct {
	// fulfillCalls defines how many times fulfillments were created.
	fulfillCalls int
	// lastInput defines the latest fulfillment payload values.
	lastInput shopifyport.ShopifyFulfillOrderInput
}

// FulfillOrder captures fulfillment inputs.
func (m *shippingDestinationMock) FulfillOrder(ctx context.Context, input shopifyport.ShopifyFulfillOrderInput) (string, error) {
	_ = ctx
	m.fulfillCalls++
	m.lastInput = input
	return "gid://shopify/Fulfillment/1", nil
}

// CancelFulfillment satisfies the fulfillment destination contract for tests.
func (m *shippingDestinationMock) CancelFulfillment(ctx context.Context, fulfillmentID string) error {
	_ = ctx
	_ = fulfillmentID
	return nil
}

// shippingLinksMock defines Shopify sync-link behavior for tests.
type shippingLinksMock struct {
	// byMannaiah defines links keyed by kind and Mannaiah id.
	byMannaiah map[shopifyport.SyncKind]map[string]*shopifyport.SyncLink
}

// GetLinkByShopifyID satisfies the repository contract for tests.
func (m *shippingLinksMock) GetLinkByShopifyID(ctx context.Context, kind shopifyport.SyncKind, shopDomain string, shopifyID string) (*shopifyport.SyncLink, error) {
	_ = ctx
	_ = kind
	_ = shopDomain
	_ = shopifyID
	return nil, nil
}

// GetLinkByMannaiahID resolves links by aggregate kind and Mannaiah id.
func (m *shippingLinksMock) GetLinkByMannaiahID(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string) (*shopifyport.SyncLink, error) {
	_ = ctx
	if m == nil || m.byMannaiah == nil {
		return nil, nil
	}
	row := m.byMannaiah[kind][mannaiahID]
	if row == nil {
		return nil, nil
	}
	copy := *row
	return &copy, nil
}

// UpsertLink stores fulfillment links created by the service.
func (m *shippingLinksMock) UpsertLink(ctx context.Context, input shopifyport.UpsertSyncLinkInput) (*shopifyport.SyncLink, error) {
	_ = ctx
	if m.byMannaiah == nil {
		m.byMannaiah = map[shopifyport.SyncKind]map[string]*shopifyport.SyncLink{}
	}
	if m.byMannaiah[input.Kind] == nil {
		m.byMannaiah[input.Kind] = map[string]*shopifyport.SyncLink{}
	}
	link := &shopifyport.SyncLink{
		Kind:            input.Kind,
		ShopDomain:      input.ShopDomain,
		ShopifyID:       input.ShopifyID,
		MannaiahID:      input.MannaiahID,
		LastKnownStatus: input.LastKnownStatus,
		LastSyncedAt:    input.LastSyncedAt,
	}
	m.byMannaiah[input.Kind][input.MannaiahID] = link
	copy := *link
	return &copy, nil
}

// UpdateLastKnownStatus satisfies the repository contract for tests.
func (m *shippingLinksMock) UpdateLastKnownStatus(ctx context.Context, kind shopifyport.SyncKind, mannaiahID string, status string) error {
	_ = ctx
	_ = kind
	_ = mannaiahID
	_ = status
	return nil
}
