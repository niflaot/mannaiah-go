package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
	shopifyport "mannaiah/module/shopify/port"
)

// TestUpsertOrderReusesExistingIdentifierAcrossRealms verifies Shopify imports update previous-realm orders instead of duplicating them.
func TestUpsertOrderReusesExistingIdentifierAcrossRealms(t *testing.T) {
	existing := ordersdomain.Order{
		ID:            "order-existing",
		Identifier:    "1025395",
		Realm:         "woocommerce",
		ContactID:     "contact-1",
		CurrentStatus: ordersdomain.StatusPending,
	}
	orders := &upsertOrderServiceMock{orders: map[string]ordersdomain.Order{existing.ID: existing}}
	links := &upsertSyncLinksMock{}
	upserter, err := NewUpserter(orders, links)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	updated, err := upserter.UpsertOrder(context.Background(), shopifyport.OrderSyncCommand{
		ShopDomain:    "store.myshopify.com",
		ShopifyID:     "gid://shopify/Order/1025395",
		Identifier:    "1025395",
		Realm:         "shopify",
		ContactID:     "contact-2",
		Status:        ordersdomain.StatusCreated,
		Source:        "manual",
		Items:         []shopifyport.OrderSyncItemCommand{{SKU: "SKU-1", Quantity: 1, Value: 157000}},
		PaymentMethod: "wompi",
	})
	if err != nil {
		t.Fatalf("UpsertOrder() error = %v", err)
	}
	if updated.ID != existing.ID {
		t.Fatalf("updated.ID = %q, want existing order id", updated.ID)
	}
	if len(orders.creates) != 0 {
		t.Fatalf("creates len = %d, want 0", len(orders.creates))
	}
	if len(orders.updates) != 1 {
		t.Fatalf("updates len = %d, want 1", len(orders.updates))
	}
	if orders.updates[0].Source != syncMutationSource {
		t.Fatalf("update source = %q, want %q", orders.updates[0].Source, syncMutationSource)
	}
	if len(orders.statuses) != 1 {
		t.Fatalf("statuses len = %d, want 1", len(orders.statuses))
	}
	if orders.statuses[0].Author != syncMutationSource || orders.statuses[0].Source != syncMutationSource {
		t.Fatalf("status author/source = %q/%q, want %q", orders.statuses[0].Author, orders.statuses[0].Source, syncMutationSource)
	}
	if len(links.upserts) != 1 {
		t.Fatalf("link upserts len = %d, want 1", len(links.upserts))
	}
	if links.upserts[0].MannaiahID != existing.ID {
		t.Fatalf("link MannaiahID = %q, want existing order id", links.upserts[0].MannaiahID)
	}
}

// TestUpsertOrderPrefersExistingShopifyLink verifies repeated imports follow the persisted Shopify link.
func TestUpsertOrderPrefersExistingShopifyLink(t *testing.T) {
	existing := ordersdomain.Order{
		ID:            "order-linked",
		Identifier:    "old-name",
		Realm:         "woocommerce",
		ContactID:     "contact-1",
		CurrentStatus: ordersdomain.StatusCreated,
	}
	orders := &upsertOrderServiceMock{orders: map[string]ordersdomain.Order{existing.ID: existing}}
	links := &upsertSyncLinksMock{
		byShopifyID: map[string]shopifyport.SyncLink{
			syncLinkShopifyKey(shopifyport.SyncKindOrder, "store.myshopify.com", "gid://shopify/Order/1025395"): {
				Kind:       shopifyport.SyncKindOrder,
				ShopDomain: "store.myshopify.com",
				ShopifyID:  "gid://shopify/Order/1025395",
				MannaiahID: existing.ID,
			},
		},
	}
	upserter, err := NewUpserter(orders, links)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	updated, err := upserter.UpsertOrder(context.Background(), shopifyport.OrderSyncCommand{
		ShopDomain: "store.myshopify.com",
		ShopifyID:  "gid://shopify/Order/1025395",
		Identifier: "1025395",
		Realm:      "shopify",
		ContactID:  "contact-2",
		Status:     ordersdomain.StatusCreated,
		Items:      []shopifyport.OrderSyncItemCommand{{SKU: "SKU-1", Quantity: 1, Value: 157000}},
	})
	if err != nil {
		t.Fatalf("UpsertOrder() error = %v", err)
	}
	if updated.ID != existing.ID {
		t.Fatalf("updated.ID = %q, want linked order id", updated.ID)
	}
	if len(orders.creates) != 0 {
		t.Fatalf("creates len = %d, want 0", len(orders.creates))
	}
}

type upsertOrderServiceMock struct {
	orders   map[string]ordersdomain.Order
	creates  []ordersapplication.CreateCommand
	updates  []ordersapplication.UpdateCommand
	statuses []ordersapplication.UpdateStatusCommand
}

func (m *upsertOrderServiceMock) Create(_ context.Context, command ordersapplication.CreateCommand) (*ordersdomain.Order, error) {
	m.creates = append(m.creates, command)
	created := ordersdomain.Order{ID: "created-order", Identifier: command.Identifier, Realm: command.Realm, ContactID: command.ContactID}
	if command.InitialStatus != nil {
		created.CurrentStatus = *command.InitialStatus
	}
	if m.orders == nil {
		m.orders = map[string]ordersdomain.Order{}
	}
	m.orders[created.ID] = created
	return &created, nil
}

func (m *upsertOrderServiceMock) Update(_ context.Context, id string, command ordersapplication.UpdateCommand) (*ordersdomain.Order, error) {
	m.updates = append(m.updates, command)
	entity, ok := m.orders[strings.TrimSpace(id)]
	if !ok {
		return nil, ordersport.ErrNotFound
	}
	return &entity, nil
}

func (m *upsertOrderServiceMock) Get(_ context.Context, id string) (*ordersdomain.Order, error) {
	entity, ok := m.orders[strings.TrimSpace(id)]
	if !ok {
		return nil, ordersport.ErrNotFound
	}
	return &entity, nil
}

func (m *upsertOrderServiceMock) List(_ context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error) {
	rows := make([]ordersdomain.Order, 0, len(m.orders))
	for _, order := range m.orders {
		if strings.TrimSpace(query.Identifier) != "" && order.Identifier != strings.TrimSpace(query.Identifier) {
			continue
		}
		if strings.TrimSpace(query.Realm) != "" && order.Realm != strings.TrimSpace(query.Realm) {
			continue
		}
		rows = append(rows, order)
	}
	if len(rows) > 1 && query.Limit > 0 {
		rows = rows[:query.Limit]
	}
	return &ordersapplication.ListResult{Data: rows, Total: int64(len(rows)), Page: 1, Limit: query.Limit}, nil
}

func (m *upsertOrderServiceMock) UpdateStatus(_ context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error) {
	m.statuses = append(m.statuses, command)
	entity, ok := m.orders[strings.TrimSpace(id)]
	if !ok {
		return nil, ordersport.ErrNotFound
	}
	entity.CurrentStatus = command.Status
	m.orders[entity.ID] = entity
	return &entity, nil
}

func (m *upsertOrderServiceMock) AddComment(context.Context, string, ordersapplication.AddCommentCommand) (*ordersdomain.Order, error) {
	return nil, errors.New("not implemented")
}

func (m *upsertOrderServiceMock) UpdateComment(context.Context, string, string, ordersapplication.UpdateCommentCommand) (*ordersdomain.Order, error) {
	return nil, errors.New("not implemented")
}

func (m *upsertOrderServiceMock) DeleteComment(context.Context, string, string, ordersapplication.DeleteCommentCommand) (*ordersdomain.Order, error) {
	return nil, errors.New("not implemented")
}

type upsertSyncLinksMock struct {
	byShopifyID  map[string]shopifyport.SyncLink
	byMannaiahID map[string]shopifyport.SyncLink
	upserts      []shopifyport.UpsertSyncLinkInput
}

func (m *upsertSyncLinksMock) GetLinkByShopifyID(_ context.Context, kind shopifyport.SyncKind, shopDomain string, shopifyID string) (*shopifyport.SyncLink, error) {
	if m.byShopifyID == nil {
		return nil, nil
	}
	link, ok := m.byShopifyID[syncLinkShopifyKey(kind, shopDomain, shopifyID)]
	if !ok {
		return nil, nil
	}
	return &link, nil
}

func (m *upsertSyncLinksMock) GetLinkByMannaiahID(_ context.Context, kind shopifyport.SyncKind, mannaiahID string) (*shopifyport.SyncLink, error) {
	if m.byMannaiahID == nil {
		return nil, nil
	}
	link, ok := m.byMannaiahID[string(kind)+"|"+strings.TrimSpace(mannaiahID)]
	if !ok {
		return nil, nil
	}
	return &link, nil
}

func (m *upsertSyncLinksMock) UpsertLink(_ context.Context, input shopifyport.UpsertSyncLinkInput) (*shopifyport.SyncLink, error) {
	m.upserts = append(m.upserts, input)
	link := shopifyport.SyncLink{
		Kind:            input.Kind,
		ShopDomain:      input.ShopDomain,
		ShopifyID:       input.ShopifyID,
		MannaiahID:      input.MannaiahID,
		LastKnownStatus: input.LastKnownStatus,
		LastSyncedAt:    input.LastSyncedAt,
	}
	return &link, nil
}

func (m *upsertSyncLinksMock) UpdateLastKnownStatus(context.Context, shopifyport.SyncKind, string, string) error {
	return nil
}

func syncLinkShopifyKey(kind shopifyport.SyncKind, shopDomain string, shopifyID string) string {
	return string(kind) + "|" + strings.TrimSpace(shopDomain) + "|" + strings.TrimSpace(shopifyID)
}
