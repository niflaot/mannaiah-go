package application

import (
	"context"
	"errors"
	"testing"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

// repositoryMock defines repository behavior for service tests.
type repositoryMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, order *ordersdomain.Order) error
	// getByIDFn defines get behavior.
	getByIDFn func(ctx context.Context, id string) (*ordersdomain.Order, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error)
	// appendStatusFn defines append-status behavior.
	appendStatusFn func(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error)
}

// TestUpdateStatusCustomOccurredAt verifies update-status timestamp override behavior.
func TestUpdateStatusCustomOccurredAt(t *testing.T) {
	repository := repositoryMock{
		createFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id, ContactID: "c-1"}, nil
		},
		listFn: func(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error) {
			return nil, 0, nil
		},
		appendStatusFn: func(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{
				ID:            id,
				ContactID:     "c-1",
				CurrentStatus: entry.Status,
				StatusHistory: []ordersdomain.StatusEntry{entry},
			}, nil
		},
	}
	customers := customerSourceMock{
		getByIDFn: func(ctx context.Context, id string) (*ordersport.Customer, error) {
			return &ordersport.Customer{ID: id}, nil
		},
	}
	service, err := NewService(repository, customers)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	occurredAt := time.Date(2026, time.February, 14, 15, 0, 0, 0, time.UTC)
	updated, err := service.UpdateStatus(context.Background(), "o-1", UpdateStatusCommand{
		Status:      ordersdomain.StatusCompleted,
		Author:      "user:1",
		Description: "completed",
		OccurredAt:  &occurredAt,
	})
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if len(updated.StatusHistory) != 1 || !updated.StatusHistory[0].OccurredAt.UTC().Equal(occurredAt) {
		t.Fatalf("status occurredAt = %v, want %v", updated.StatusHistory, occurredAt)
	}
}

// Create executes mocked create behavior.
func (m repositoryMock) Create(ctx context.Context, order *ordersdomain.Order) error {
	return m.createFn(ctx, order)
}

// GetByID executes mocked get behavior.
func (m repositoryMock) GetByID(ctx context.Context, id string) (*ordersdomain.Order, error) {
	return m.getByIDFn(ctx, id)
}

// List executes mocked list behavior.
func (m repositoryMock) List(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error) {
	return m.listFn(ctx, query)
}

// AppendStatus executes mocked append-status behavior.
func (m repositoryMock) AppendStatus(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error) {
	return m.appendStatusFn(ctx, id, entry)
}

// customerSourceMock defines customer-source behavior for service tests.
type customerSourceMock struct {
	// getByIDFn defines lookup behavior.
	getByIDFn func(ctx context.Context, id string) (*ordersport.Customer, error)
}

// GetByID executes mocked customer lookup behavior.
func (m customerSourceMock) GetByID(ctx context.Context, id string) (*ordersport.Customer, error) {
	return m.getByIDFn(ctx, id)
}

// productResolverMock defines product resolver behavior for service tests.
type productResolverMock struct {
	// resolveFn defines resolve behavior.
	resolveFn func(ctx context.Context, sku string, alternateName string) (*ordersport.ProductResolution, error)
}

// Resolve executes mocked product resolution behavior.
func (m productResolverMock) Resolve(ctx context.Context, sku string, alternateName string) (*ordersport.ProductResolution, error) {
	return m.resolveFn(ctx, sku, alternateName)
}

// TestNewServiceValidation verifies constructor validation behavior.
func TestNewServiceValidation(t *testing.T) {
	_, err := NewService(nil, customerSourceMock{})
	if !errors.Is(err, ErrNilRepository) {
		t.Fatalf("NewService(nil repository) error = %v, want ErrNilRepository", err)
	}

	_, err = NewService(repositoryMock{}, nil)
	if !errors.Is(err, ErrNilCustomerSource) {
		t.Fatalf("NewService(nil customer source) error = %v, want ErrNilCustomerSource", err)
	}
}

// TestCreateResolvesItemsAndShipping verifies create behavior with SKU and alternate-name resolution and billing fallback shipping.
func TestCreateResolvesItemsAndShipping(t *testing.T) {
	var captured *ordersdomain.Order
	repository := repositoryMock{
		createFn: func(ctx context.Context, order *ordersdomain.Order) error {
			copied := *order
			captured = &copied
			order.ID = "o-1"
			return nil
		},
		getByIDFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) { return nil, nil },
		listFn: func(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error) {
			return nil, 0, nil
		},
		appendStatusFn: func(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error) {
			return nil, nil
		},
	}
	customers := customerSourceMock{
		getByIDFn: func(ctx context.Context, id string) (*ordersport.Customer, error) {
			return &ordersport.Customer{
				ID:           id,
				Address:      "Billing Address",
				AddressExtra: "Billing Address 2",
				Phone:        "+573001112233",
				CityCode:     "110111",
			}, nil
		},
	}
	resolver := productResolverMock{
		resolveFn: func(ctx context.Context, sku string, alternateName string) (*ordersport.ProductResolution, error) {
			if sku == "SKU-1" {
				return &ordersport.ProductResolution{ProductID: "p-1", MatchedBy: "sku"}, nil
			}
			if alternateName == "Fallback Name" {
				return &ordersport.ProductResolution{ProductID: "p-2", MatchedBy: "alternate_name"}, nil
			}

			return nil, nil
		},
	}
	service, err := NewService(repository, customers, resolver)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, err := service.Create(context.Background(), CreateCommand{
		Identifier: "ORD-1",
		Realm:      "woocommerce",
		ContactID:  "c-1",
		Items: []CreateItemCommand{
			{SKU: "SKU-1", Quantity: 1},
			{SKU: "MISSING", AlternateName: "Fallback Name", Quantity: 2},
		},
		Author: "system",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.ID != "o-1" {
		t.Fatalf("result.ID = %q, want %q", result.ID, "o-1")
	}
	if captured == nil {
		t.Fatalf("expected repository create capture")
	}
	if captured.Items[0].ProductID != "p-1" || captured.Items[0].ResolutionSource != ordersdomain.ItemResolutionSourceSKU {
		t.Fatalf("captured.Items[0] = %#v, want SKU resolution", captured.Items[0])
	}
	if captured.Items[1].ProductID != "p-2" || captured.Items[1].ResolutionSource != ordersdomain.ItemResolutionSourceAlternateName {
		t.Fatalf("captured.Items[1] = %#v, want alternate-name resolution", captured.Items[1])
	}
	if captured.HasCustomShippingAddress {
		t.Fatalf("captured.HasCustomShippingAddress = %v, want %v", captured.HasCustomShippingAddress, false)
	}
	if captured.ShippingAddress.Address != "Billing Address" {
		t.Fatalf("captured.ShippingAddress.Address = %q, want %q", captured.ShippingAddress.Address, "Billing Address")
	}
}

// TestCreateStoresExplicitShippingSnapshot verifies explicit shipping snapshot behavior.
func TestCreateStoresExplicitShippingSnapshot(t *testing.T) {
	repository := repositoryMock{
		createFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) {
			return nil, nil
		},
		listFn: func(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error) {
			return nil, 0, nil
		},
		appendStatusFn: func(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error) {
			return nil, nil
		},
	}
	customers := customerSourceMock{
		getByIDFn: func(ctx context.Context, id string) (*ordersport.Customer, error) {
			return &ordersport.Customer{ID: id, Address: "A", AddressExtra: "B", Phone: "C", CityCode: "D"}, nil
		},
	}
	service, err := NewService(repository, customers)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	order, err := service.Create(context.Background(), CreateCommand{
		Identifier: "ORD-2",
		Realm:      "woocommerce",
		ContactID:  "c-2",
		Items:      []CreateItemCommand{{SKU: "SKU-2", Quantity: 1}},
		ShippingAddress: &ShippingAddressCommand{
			Address:  "A",
			Address2: "B",
			Phone:    "C",
			CityCode: "D",
		},
		Author: "system",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !order.HasCustomShippingAddress {
		t.Fatalf("order.HasCustomShippingAddress = %v, want %v", order.HasCustomShippingAddress, true)
	}
}

// TestGetListAndUpdateStatus verifies get/list/status update behavior.
func TestGetListAndUpdateStatus(t *testing.T) {
	repository := repositoryMock{
		createFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id, ContactID: "c-1", HasCustomShippingAddress: false}, nil
		},
		listFn: func(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error) {
			return []ordersdomain.Order{{ID: "o-1", ContactID: "c-1", HasCustomShippingAddress: false}}, 1, nil
		},
		appendStatusFn: func(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{
				ID:            id,
				ContactID:     "c-1",
				CurrentStatus: entry.Status,
				StatusHistory: []ordersdomain.StatusEntry{entry},
			}, nil
		},
	}
	customers := customerSourceMock{
		getByIDFn: func(ctx context.Context, id string) (*ordersport.Customer, error) {
			return &ordersport.Customer{ID: id, Address: "Billing", CityCode: "110111"}, nil
		},
	}
	service, err := NewService(repository, customers)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	entity, err := service.Get(context.Background(), "o-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if entity.ShippingAddress.Address != "Billing" {
		t.Fatalf("entity.ShippingAddress.Address = %q, want %q", entity.ShippingAddress.Address, "Billing")
	}

	page, err := service.List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.Total != 1 || len(page.Data) != 1 {
		t.Fatalf("List() result = %#v, want one row", page)
	}

	updated, err := service.UpdateStatus(context.Background(), "o-1", UpdateStatusCommand{
		Status:      ordersdomain.StatusCompleted,
		Author:      "user:1",
		Description: "completed",
	})
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if updated.CurrentStatus != ordersdomain.StatusCompleted {
		t.Fatalf("updated.CurrentStatus = %q, want %q", updated.CurrentStatus, ordersdomain.StatusCompleted)
	}
}

// TestServiceErrorBranches verifies service error-mapping behavior.
func TestServiceErrorBranches(t *testing.T) {
	repository := repositoryMock{
		createFn: func(ctx context.Context, order *ordersdomain.Order) error {
			return ordersport.ErrDuplicateIdentifier
		},
		getByIDFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) {
			return nil, ordersport.ErrNotFound
		},
		listFn: func(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error) {
			return nil, 0, errors.New("list failed")
		},
		appendStatusFn: func(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error) {
			return nil, errors.New("update failed")
		},
	}
	customers := customerSourceMock{
		getByIDFn: func(ctx context.Context, id string) (*ordersport.Customer, error) {
			return nil, ordersport.ErrCustomerNotFound
		},
	}
	service, err := NewService(repository, customers, productResolverMock{
		resolveFn: func(ctx context.Context, sku string, alternateName string) (*ordersport.ProductResolution, error) {
			return nil, errors.New("resolve failed")
		},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, err := service.Create(context.Background(), CreateCommand{Identifier: "x", Realm: "y", ContactID: "c", Items: []CreateItemCommand{{SKU: "s", Quantity: 1}}}); !errors.Is(err, ordersport.ErrCustomerNotFound) {
		t.Fatalf("Create() error = %v, want ErrCustomerNotFound", err)
	}

	if _, err := service.Get(context.Background(), ""); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Get(empty) error = %v, want ErrInvalidID", err)
	}
	if _, err := service.List(context.Background(), ListQuery{}); err == nil {
		t.Fatalf("List() expected error")
	}
	if _, err := service.UpdateStatus(context.Background(), "", UpdateStatusCommand{}); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("UpdateStatus(empty id) error = %v, want ErrInvalidID", err)
	}
	if _, err := service.UpdateStatus(context.Background(), "o-1", UpdateStatusCommand{Status: ordersdomain.StatusCreated, Author: ""}); !errors.Is(err, ErrStatusAuthorRequired) {
		t.Fatalf("UpdateStatus(empty author) error = %v, want ErrStatusAuthorRequired", err)
	}
}
