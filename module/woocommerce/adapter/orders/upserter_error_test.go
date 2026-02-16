package orders

import (
	"context"
	errorspkg "errors"
	"testing"
	"time"

	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
	"mannaiah/module/woocommerce/port"
)

// TestUpsertByIdentifierErrorPaths verifies error propagation behavior.
func TestUpsertByIdentifierErrorPaths(t *testing.T) {
	contactService := newContactServiceMock()
	orderService := newOrdersServiceMock()
	upserter, err := NewUpserter(orderService, contactService)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	_, err = upserter.UpsertByIdentifier(context.Background(), port.OrderSyncCommand{
		Identifier: "1001",
		Status:     "processing",
		Contact: port.ContactSyncCommand{
			Email: "",
		},
		Items: []port.OrderSyncItem{{SKU: "SKU-1", Quantity: 1}},
	})
	if err == nil {
		t.Fatalf("expected contact sync error")
	}

	contactService = newContactServiceMock()
	contactService.listErr = errorspkg.New("list failed")
	upserter, err = NewUpserter(newOrdersServiceMock(), contactService)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	_, err = upserter.UpsertByIdentifier(context.Background(), port.OrderSyncCommand{
		Identifier: "1001",
		Status:     "processing",
		Contact: port.ContactSyncCommand{
			Email:     "woo.one@example.com",
			FirstName: "Woo",
			LastName:  "One",
		},
		Items: []port.OrderSyncItem{{SKU: "SKU-1", Quantity: 1}},
	})
	if err == nil {
		t.Fatalf("expected contact lookup error")
	}
}

// TestMapOrderStatus verifies WooCommerce status mapping behavior.
func TestMapOrderStatus(t *testing.T) {
	cases := map[string]ordersdomain.Status{
		"cancelled":       ordersdomain.StatusCancelled,
		"canceled":        ordersdomain.StatusCancelled,
		"processing":      ordersdomain.StatusCreated,
		"on-hold":         ordersdomain.StatusHold,
		"pending-payment": ordersdomain.StatusPending,
		"completed":       ordersdomain.StatusCompleted,
		"unknown":         ordersdomain.StatusCreated,
	}

	for value, expected := range cases {
		if got := mapOrderStatus(value); got != expected {
			t.Fatalf("mapOrderStatus(%q) = %q, want %q", value, got, expected)
		}
	}
}

// raceOrderServiceMock defines race-condition service behavior for duplicate-create branch tests.
type raceOrderServiceMock struct {
	// listResults defines list results returned across calls.
	listResults []*ordersapplication.ListResult
	// listCallCount defines list call counters.
	listCallCount int
}

// Create returns duplicate identifier errors.
func (m *raceOrderServiceMock) Create(ctx context.Context, command ordersapplication.CreateCommand) (*ordersdomain.Order, error) {
	return nil, ordersport.ErrDuplicateIdentifier
}

// Get returns empty order rows.
func (m *raceOrderServiceMock) Get(ctx context.Context, id string) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{}, nil
}

// List returns configured list sequences.
func (m *raceOrderServiceMock) List(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error) {
	if m.listCallCount >= len(m.listResults) {
		return &ordersapplication.ListResult{}, nil
	}

	result := m.listResults[m.listCallCount]
	m.listCallCount++
	if result == nil {
		return &ordersapplication.ListResult{}, nil
	}

	return result, nil
}

// Update updates mutable order rows.
func (m *raceOrderServiceMock) Update(ctx context.Context, id string, command ordersapplication.UpdateCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{ID: id}, nil
}

// UpdateStatus returns updated order rows.
func (m *raceOrderServiceMock) UpdateStatus(ctx context.Context, id string, command ordersapplication.UpdateStatusCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{
		ID:            id,
		CurrentStatus: command.Status,
		StatusHistory: []ordersdomain.StatusEntry{{Status: command.Status, Author: command.Author, Description: command.Description, OccurredAt: time.Now().UTC()}},
	}, nil
}

// AddComment appends comments to order rows.
func (m *raceOrderServiceMock) AddComment(ctx context.Context, id string, command ordersapplication.AddCommentCommand) (*ordersdomain.Order, error) {
	occurredAt := time.Now().UTC()
	if command.OccurredAt != nil && !command.OccurredAt.IsZero() {
		occurredAt = command.OccurredAt.UTC()
	}

	return &ordersdomain.Order{
		ID: id,
		Comments: []ordersdomain.Comment{
			{
				Author:     command.Author,
				Comment:    command.Comment,
				Internal:   command.Internal,
				OccurredAt: occurredAt,
			},
		},
	}, nil
}

// UpdateComment updates order comment rows.
func (m *raceOrderServiceMock) UpdateComment(ctx context.Context, id string, commentID string, command ordersapplication.UpdateCommentCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{ID: id}, nil
}

// DeleteComment deletes order comment rows.
func (m *raceOrderServiceMock) DeleteComment(ctx context.Context, id string, commentID string, command ordersapplication.DeleteCommentCommand) (*ordersdomain.Order, error) {
	return &ordersdomain.Order{ID: id}, nil
}

// TestCreateOrderDuplicateFallback verifies duplicate-create fallback behavior.
func TestCreateOrderDuplicateFallback(t *testing.T) {
	mock := &raceOrderServiceMock{
		listResults: []*ordersapplication.ListResult{
			{
				Data: []ordersdomain.Order{{
					ID:            "order-1",
					Identifier:    "1001",
					Realm:         defaultRealm,
					CurrentStatus: ordersdomain.StatusCreated,
				}},
			},
		},
	}
	upserter := &Upserter{orderService: mock}

	outcome, err := upserter.createOrder(context.Background(), port.OrderSyncCommand{
		Identifier: "1001",
		Status:     "processing",
	}, "contact-1", defaultRealm, ordersdomain.StatusCreated)
	if err != nil {
		t.Fatalf("createOrder() error = %v", err)
	}
	if outcome != port.UpsertOutcomeUnchanged {
		t.Fatalf("outcome = %q, want %q", outcome, port.UpsertOutcomeUnchanged)
	}
}

// TestCreateOrderDuplicateMissingLatest verifies duplicate-create error behavior when latest rows remain unavailable.
func TestCreateOrderDuplicateMissingLatest(t *testing.T) {
	upserter := &Upserter{
		orderService: &raceOrderServiceMock{
			listResults: []*ordersapplication.ListResult{
				{Data: []ordersdomain.Order{}},
			},
		},
	}

	if _, err := upserter.createOrder(context.Background(), port.OrderSyncCommand{
		Identifier: "1001",
		Status:     "processing",
	}, "contact-1", defaultRealm, ordersdomain.StatusCreated); err == nil {
		t.Fatalf("expected createOrder() error")
	}
}

// TestSupportHelpers verifies support helper behavior.
func TestSupportHelpers(t *testing.T) {
	if normalizeRealm("") != defaultRealm {
		t.Fatalf("normalizeRealm(empty) = %q, want %q", normalizeRealm(""), defaultRealm)
	}
	metadata := normalizeMetadata(map[string]string{" key ": " value ", "": "skip"})
	if metadata["key"] != "value" {
		t.Fatalf("normalizeMetadata() = %+v, want normalized key/value", metadata)
	}
	if len(sortedComments(nil)) != 0 {
		t.Fatalf("sortedComments(nil) should return nil/empty slice")
	}
	if toShippingAddress(nil) != nil {
		t.Fatalf("toShippingAddress(nil) should return nil")
	}
	if toShippingAddress(&port.OrderSyncShippingAddress{}) != nil {
		t.Fatalf("toShippingAddress(empty) should return nil")
	}
}

// TestResolveContactIDByEmailNotFound verifies not-found lookup behavior.
func TestResolveContactIDByEmailNotFound(t *testing.T) {
	upserter := &Upserter{
		contactService: newContactServiceMock(),
	}

	if _, err := upserter.resolveContactIDByEmail(context.Background(), "missing@example.com"); !errorspkg.Is(err, ErrContactNotFound) {
		t.Fatalf("resolveContactIDByEmail() error = %v, want ErrContactNotFound", err)
	}
}
