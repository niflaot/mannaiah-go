package application

import (
	"context"
	"errors"
	"testing"
	"time"

	ordersevent "mannaiah/module/orders/application/event"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

// repositoryMock defines repository behavior for service tests.
type repositoryMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, order *ordersdomain.Order) error
	// updateFn defines update behavior.
	updateFn func(ctx context.Context, order *ordersdomain.Order) error
	// getByIDFn defines get behavior.
	getByIDFn func(ctx context.Context, id string) (*ordersdomain.Order, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error)
	// appendStatusFn defines append-status behavior.
	appendStatusFn func(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error)
	// appendCommentFn defines append-comment behavior.
	appendCommentFn func(ctx context.Context, id string, comment ordersdomain.Comment) (*ordersdomain.Order, error)
}

// publisherMock defines integration event publication behavior for service tests.
type publisherMock struct {
	// events defines captured integration event values.
	events []ordersport.IntegrationEvent
	// err defines publication errors.
	err error
}

// Publish captures integration events.
func (m *publisherMock) Publish(ctx context.Context, integrationEvent ordersport.IntegrationEvent) error {
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, integrationEvent)
	return nil
}

// TestUpdateStatusCustomOccurredAt verifies update-status timestamp override behavior.
func TestUpdateStatusCustomOccurredAt(t *testing.T) {
	repository := repositoryMock{
		createFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id, ContactID: "c-1"}, nil
		},
		updateFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
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
		appendCommentFn: func(ctx context.Context, id string, comment ordersdomain.Comment) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id, ContactID: "c-1", Comments: []ordersdomain.Comment{comment}}, nil
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

// TestAddComment verifies order-comment append behavior.
func TestAddComment(t *testing.T) {
	repository := repositoryMock{
		createFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id, ContactID: "c-1"}, nil
		},
		updateFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
		listFn: func(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error) {
			return nil, 0, nil
		},
		appendStatusFn: func(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id, ContactID: "c-1", StatusHistory: []ordersdomain.StatusEntry{entry}, CurrentStatus: entry.Status}, nil
		},
		appendCommentFn: func(ctx context.Context, id string, comment ordersdomain.Comment) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id, ContactID: "c-1", Comments: []ordersdomain.Comment{comment}}, nil
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

	occurredAt := time.Date(2026, time.February, 14, 15, 30, 0, 0, time.UTC)
	updated, err := service.AddComment(context.Background(), "o-1", AddCommentCommand{
		Author:     "agent-1",
		Comment:    "call before dispatch",
		Internal:   true,
		OccurredAt: &occurredAt,
	})
	if err != nil {
		t.Fatalf("AddComment() error = %v", err)
	}
	if len(updated.Comments) != 1 {
		t.Fatalf("len(updated.Comments) = %d, want 1", len(updated.Comments))
	}
	if updated.Comments[0].Author != "agent-1" || updated.Comments[0].Comment != "call before dispatch" || !updated.Comments[0].Internal {
		t.Fatalf("updated.Comments[0] = %+v, want author/comment/internal values", updated.Comments[0])
	}
	if !updated.Comments[0].OccurredAt.UTC().Equal(occurredAt) {
		t.Fatalf("updated.Comments[0].OccurredAt = %v, want %v", updated.Comments[0].OccurredAt, occurredAt)
	}
}

// Create executes mocked create behavior.
func (m repositoryMock) Create(ctx context.Context, order *ordersdomain.Order) error {
	return m.createFn(ctx, order)
}

// Update executes mocked update behavior.
func (m repositoryMock) Update(ctx context.Context, order *ordersdomain.Order) error {
	return m.updateFn(ctx, order)
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

// AppendComment executes mocked append-comment behavior.
func (m repositoryMock) AppendComment(ctx context.Context, id string, comment ordersdomain.Comment) (*ordersdomain.Order, error) {
	return m.appendCommentFn(ctx, id, comment)
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
		updateFn:  func(ctx context.Context, order *ordersdomain.Order) error { return nil },
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
		updateFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
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
		updateFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
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
		updateFn: func(ctx context.Context, order *ordersdomain.Order) error {
			return errors.New("update failed")
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
	if _, err := service.Update(context.Background(), "", UpdateCommand{}); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Update(empty id) error = %v, want ErrInvalidID", err)
	}
	if _, err := service.Update(context.Background(), "o-1", UpdateCommand{}); !errors.Is(err, ErrEmptyOrderUpdate) {
		t.Fatalf("Update(empty command) error = %v, want ErrEmptyOrderUpdate", err)
	}
}

// TestUpdateAndEventPublishing verifies update behavior and integration event publication.
func TestUpdateAndEventPublishing(t *testing.T) {
	repository := repositoryMock{
		createFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
		updateFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{
				ID:            id,
				Identifier:    "1001",
				Realm:         "woocommerce",
				ContactID:     "c-1",
				CurrentStatus: ordersdomain.StatusCreated,
				StatusHistory: []ordersdomain.StatusEntry{{Status: ordersdomain.StatusCreated, Author: "system", OccurredAt: time.Now().UTC()}},
				Items:         []ordersdomain.Item{{SKU: "SKU-1", Quantity: 1}},
			}, nil
		},
		listFn: func(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error) {
			return nil, 0, nil
		},
		appendStatusFn: func(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{
				ID:            id,
				Identifier:    "1001",
				Realm:         "woocommerce",
				ContactID:     "c-1",
				CurrentStatus: entry.Status,
				StatusHistory: []ordersdomain.StatusEntry{entry},
				Items:         []ordersdomain.Item{{SKU: "SKU-1", Quantity: 1}},
			}, nil
		},
	}
	customers := customerSourceMock{
		getByIDFn: func(ctx context.Context, id string) (*ordersport.Customer, error) {
			return &ordersport.Customer{ID: id, Address: "A", CityCode: "11001"}, nil
		},
	}
	publisher := &publisherMock{}
	service, err := NewServiceWithPublisher(repository, customers, publisher)
	if err != nil {
		t.Fatalf("NewServiceWithPublisher() error = %v", err)
	}

	items := []CreateItemCommand{{SKU: "SKU-2", Quantity: 2, Value: 22}}
	charges := []ShippingChargeCommand{{MethodID: "flat_rate", MethodTitle: "Flat", Price: 9}}
	updated, err := service.Update(context.Background(), "o-1", UpdateCommand{
		Items:           &items,
		ShippingCharges: &charges,
		ShippingAddress: &ShippingAddressCommand{Address: "Street 1", CityCode: "11001"},
		Source:          "mainstream",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if len(updated.Items) != 1 || updated.Items[0].SKU != "SKU-2" {
		t.Fatalf("updated.Items = %+v, want SKU-2 row", updated.Items)
	}

	_, err = service.UpdateStatus(context.Background(), "o-1", UpdateStatusCommand{
		Status: ordersdomain.StatusCompleted,
		Author: "system",
		Source: "mainstream",
	})
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("len(publisher.events) = %d, want %d", len(publisher.events), 2)
	}
	if publisher.events[0].Topic != ordersport.TopicOrderUpdated {
		t.Fatalf("events[0].Topic = %q, want %q", publisher.events[0].Topic, ordersport.TopicOrderUpdated)
	}
	if publisher.events[1].Topic != ordersport.TopicOrderStatusUpdated {
		t.Fatalf("events[1].Topic = %q, want %q", publisher.events[1].Topic, ordersport.TopicOrderStatusUpdated)
	}
}

// TestCreatePublishError verifies create behavior when event publication fails.
func TestCreatePublishError(t *testing.T) {
	repository := repositoryMock{
		createFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
		updateFn: func(ctx context.Context, order *ordersdomain.Order) error { return nil },
		getByIDFn: func(ctx context.Context, id string) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id, ContactID: "c-1"}, nil
		},
		listFn: func(ctx context.Context, query ordersport.ListQuery) ([]ordersdomain.Order, int64, error) {
			return nil, 0, nil
		},
		appendStatusFn: func(ctx context.Context, id string, entry ordersdomain.StatusEntry) (*ordersdomain.Order, error) {
			return &ordersdomain.Order{ID: id, ContactID: "c-1"}, nil
		},
	}
	customers := customerSourceMock{
		getByIDFn: func(ctx context.Context, id string) (*ordersport.Customer, error) {
			return &ordersport.Customer{ID: id, Address: "A", CityCode: "11001"}, nil
		},
	}
	service, err := NewServiceWithPublisher(repository, customers, &publisherMock{err: errors.New("publish failed")})
	if err != nil {
		t.Fatalf("NewServiceWithPublisher() error = %v", err)
	}

	_, err = service.Create(context.Background(), CreateCommand{
		Identifier: "id-1",
		Realm:      "realm",
		ContactID:  "c-1",
		Items:      []CreateItemCommand{{SKU: "sku", Quantity: 1}},
		Author:     "system",
	})
	if err == nil {
		t.Fatalf("expected Create() error")
	}
}

// TestEventBuilderSourceDefaults verifies event source default behavior through service publishing.
func TestEventBuilderSourceDefaults(t *testing.T) {
	if resolved := ordersevent.ResolveSource(""); resolved != ordersport.EventSourceAPI {
		t.Fatalf("ResolveSource(empty) = %q, want %q", resolved, ordersport.EventSourceAPI)
	}
}
