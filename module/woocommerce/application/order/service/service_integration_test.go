package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"mannaiah/module/woocommerce/port"
)

// sourceMock defines source behavior for service tests.
type sourceMock struct {
	// validateFn defines validate behavior.
	validateFn func(ctx context.Context) error
	// listFn defines list behavior.
	listFn func(ctx context.Context, page int, pageSize int) ([]port.WooOrder, bool, error)
}

// Validate executes mocked validate behavior.
func (m sourceMock) Validate(ctx context.Context) error {
	return m.validateFn(ctx)
}

// ListOrders executes mocked list behavior.
func (m sourceMock) ListOrders(ctx context.Context, page int, pageSize int) ([]port.WooOrder, bool, error) {
	return m.listFn(ctx, page, pageSize)
}

// sourceByIDMock defines source behavior with direct order-by-id lookup support.
type sourceByIDMock struct {
	// sourceMock defines baseline source behavior.
	sourceMock
	// getByIDFn defines direct order lookup behavior.
	getByIDFn func(ctx context.Context, orderID int) (port.WooOrder, error)
}

// GetOrderByID executes direct order lookup behavior.
func (m sourceByIDMock) GetOrderByID(ctx context.Context, orderID int) (port.WooOrder, error) {
	return m.getByIDFn(ctx, orderID)
}

// targetMock defines target behavior for service tests.
type targetMock struct {
	// upsertFn defines upsert behavior.
	upsertFn func(ctx context.Context, command port.OrderSyncCommand) (port.UpsertOutcome, error)
}

// UpsertByIdentifier executes mocked upsert behavior.
func (m targetMock) UpsertByIdentifier(ctx context.Context, command port.OrderSyncCommand) (port.UpsertOutcome, error) {
	return m.upsertFn(ctx, command)
}

// publisherProbe defines event publication behavior for service tests.
type publisherProbe struct {
	// topics stores published topics.
	topics []string
}

// Publish records published topics.
func (p *publisherProbe) Publish(ctx context.Context, event port.IntegrationEvent) error {
	p.topics = append(p.topics, event.Topic)
	return nil
}

// breakerMock defines circuit-breaker behavior for service tests.
type breakerMock struct {
	// executeFn defines execution behavior.
	executeFn func(operation func() error) error
	// isOpenFn defines open-state behavior.
	isOpenFn func(err error) bool
}

// Execute executes wrapped operations.
func (m breakerMock) Execute(operation func() error) error {
	return m.executeFn(operation)
}

// IsOpenError reports open-circuit behavior.
func (m breakerMock) IsOpenError(err error) bool {
	return m.isOpenFn(err)
}

// TestNewServiceValidation verifies constructor validation behavior.
func TestNewServiceValidation(t *testing.T) {
	_, err := NewService(SyncConfig{}, nil, targetMock{}, nil, nil)
	if !errors.Is(err, ErrNilSource) {
		t.Fatalf("NewService(nil source) error = %v, want ErrNilSource", err)
	}

	_, err = NewService(SyncConfig{}, sourceMock{}, nil, nil, nil)
	if !errors.Is(err, ErrNilTarget) {
		t.Fatalf("NewService(nil target) error = %v, want ErrNilTarget", err)
	}
}

// TestValidateIntegration verifies integration validation behavior.
func TestValidateIntegration(t *testing.T) {
	service, err := NewService(SyncConfig{Enabled: false}, sourceMock{}, targetMock{}, nil, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if validateErr := service.ValidateIntegration(context.Background()); !errors.Is(validateErr, ErrSyncDisabled) {
		t.Fatalf("ValidateIntegration(disabled) error = %v, want ErrSyncDisabled", validateErr)
	}

	service, err = NewService(SyncConfig{Enabled: true}, sourceMock{
		validateFn: func(ctx context.Context) error { return errors.New("boom") },
	}, targetMock{}, nil, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if validateErr := service.ValidateIntegration(context.Background()); !errors.Is(validateErr, ErrIntegrationUnavailable) {
		t.Fatalf("ValidateIntegration(error) error = %v, want ErrIntegrationUnavailable", validateErr)
	}
}

// TestSyncOrdersSuccess verifies full sync flow behavior.
func TestSyncOrdersSuccess(t *testing.T) {
	publisher := &publisherProbe{}
	var upsertCalls atomic.Int32
	service, err := NewService(
		SyncConfig{Enabled: true, PageSize: 2, WorkerCount: 2},
		sourceMock{
			validateFn: func(ctx context.Context) error { return nil },
			listFn: func(ctx context.Context, page int, pageSize int) ([]port.WooOrder, bool, error) {
				if page == 1 {
					return []port.WooOrder{
						{
							ID:                   1001,
							Status:               "completed",
							BillingEmail:         "woo.one@example.com",
							BillingFirstName:     "Woo",
							BillingLastName:      "One",
							BillingAddress1:      "A",
							BillingAddress2:      "B",
							BillingCity:          "11001",
							ShippingAddressLine1: "A",
							ShippingAddressLine2: "B",
							ShippingCityCode:     "11001",
							Items: []port.WooOrderItem{
								{SKU: "SKU-1", Name: "Item 1", Quantity: 1, Value: 12000},
							},
							CreatedAt: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
						},
						{
							ID:                   1002,
							Status:               "pending",
							BillingEmail:         "woo.two@example.com",
							BillingFirstName:     "Woo",
							BillingLastName:      "Two",
							BillingAddress1:      "C",
							BillingCity:          "05001",
							ShippingAddressLine1: "Ship",
							ShippingCityCode:     "05002",
							Items: []port.WooOrderItem{
								{SKU: "SKU-2", Name: "Item 2", Quantity: 2},
							},
							CreatedAt: time.Date(2026, 2, 2, 10, 0, 0, 0, time.UTC),
						},
					}, false, nil
				}

				return nil, false, nil
			},
		},
		targetMock{
			upsertFn: func(ctx context.Context, command port.OrderSyncCommand) (port.UpsertOutcome, error) {
				upsertCalls.Add(1)
				if command.Identifier == "1001" && command.Status != "completed" {
					t.Fatalf("command.Status = %q, want %q", command.Status, "completed")
				}
				if command.Identifier == "1002" && command.ShippingAddress == nil {
					t.Fatalf("expected custom shipping address for identifier 1002")
				}
				return port.UpsertOutcomeCreated, nil
			},
		},
		publisher,
		nil,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, err := service.SyncOrders(context.Background(), "manual")
	if err != nil {
		t.Fatalf("SyncOrders() error = %v", err)
	}
	if upsertCalls.Load() != 2 {
		t.Fatalf("upsertCalls = %d, want %d", upsertCalls.Load(), 2)
	}
	if summary.Processed != 2 || summary.Created != 2 || summary.Failed != 0 {
		t.Fatalf("summary = %+v, want processed=2 created=2 failed=0", summary)
	}
	if len(publisher.topics) != 2 {
		t.Fatalf("published topics = %d, want %d", len(publisher.topics), 2)
	}
}

// TestSyncOrdersFailure verifies sync failure mapping behavior.
func TestSyncOrdersFailure(t *testing.T) {
	publisher := &publisherProbe{}
	service, err := NewService(
		SyncConfig{Enabled: true, PageSize: 1, WorkerCount: 1},
		sourceMock{
			validateFn: func(ctx context.Context) error { return nil },
			listFn: func(ctx context.Context, page int, pageSize int) ([]port.WooOrder, bool, error) {
				return nil, false, errors.New("list failed")
			},
		},
		targetMock{
			upsertFn: func(ctx context.Context, command port.OrderSyncCommand) (port.UpsertOutcome, error) {
				return port.UpsertOutcomeCreated, nil
			},
		},
		publisher,
		nil,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, syncErr := service.SyncOrders(context.Background(), "manual"); syncErr == nil {
		t.Fatalf("expected SyncOrders() error")
	}
	if len(publisher.topics) != 2 {
		t.Fatalf("published topics = %d, want %d", len(publisher.topics), 2)
	}
}

// TestSyncOrderByIDSuccess verifies targeted order-sync behavior by Woo order identifier.
func TestSyncOrderByIDSuccess(t *testing.T) {
	service, err := NewService(
		SyncConfig{Enabled: true, WorkerCount: 1},
		sourceByIDMock{
			sourceMock: sourceMock{
				validateFn: func(ctx context.Context) error { return nil },
				listFn: func(ctx context.Context, page int, pageSize int) ([]port.WooOrder, bool, error) {
					return nil, false, nil
				},
			},
			getByIDFn: func(ctx context.Context, orderID int) (port.WooOrder, error) {
				return port.WooOrder{
					ID:                   orderID,
					Status:               "processing",
					BillingEmail:         "single@example.com",
					BillingFirstName:     "Single",
					BillingLastName:      "Order",
					BillingAddress1:      "A",
					BillingCity:          "11001",
					ShippingAddressLine1: "A",
					ShippingCityCode:     "11001",
					Items: []port.WooOrderItem{
						{SKU: "SKU-1", Name: "Item", Quantity: 1, Value: 1000},
					},
					CreatedAt: time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC),
				}, nil
			},
		},
		targetMock{
			upsertFn: func(ctx context.Context, command port.OrderSyncCommand) (port.UpsertOutcome, error) {
				if command.Identifier != "1001" {
					t.Fatalf("command.Identifier = %q, want %q", command.Identifier, "1001")
				}
				return port.UpsertOutcomeCreated, nil
			},
		},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncOrderByID(context.Background(), "manual", 1001)
	if syncErr != nil {
		t.Fatalf("SyncOrderByID() error = %v", syncErr)
	}
	if summary.Processed != 1 || summary.Created != 1 {
		t.Fatalf("summary = %+v, want processed=1 created=1", summary)
	}
}

// TestSyncOrderByIDValidation verifies targeted order-sync validation behavior.
func TestSyncOrderByIDValidation(t *testing.T) {
	service, err := NewService(
		SyncConfig{Enabled: true},
		sourceMock{
			validateFn: func(ctx context.Context) error { return nil },
			listFn: func(ctx context.Context, page int, pageSize int) ([]port.WooOrder, bool, error) {
				return nil, false, nil
			},
		},
		targetMock{
			upsertFn: func(ctx context.Context, command port.OrderSyncCommand) (port.UpsertOutcome, error) {
				return port.UpsertOutcomeCreated, nil
			},
		},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, syncErr := service.SyncOrderByID(context.Background(), "manual", 0); !errors.Is(syncErr, ErrInvalidOrderID) {
		t.Fatalf("SyncOrderByID(invalid) error = %v, want ErrInvalidOrderID", syncErr)
	}
}

// TestSyncOrderByIDNotFound verifies targeted order-sync not-found behavior.
func TestSyncOrderByIDNotFound(t *testing.T) {
	service, err := NewService(
		SyncConfig{Enabled: true},
		sourceByIDMock{
			sourceMock: sourceMock{
				validateFn: func(ctx context.Context) error { return nil },
				listFn: func(ctx context.Context, page int, pageSize int) ([]port.WooOrder, bool, error) {
					return nil, false, nil
				},
			},
			getByIDFn: func(ctx context.Context, orderID int) (port.WooOrder, error) {
				return port.WooOrder{}, errors.New("404 not found")
			},
		},
		targetMock{
			upsertFn: func(ctx context.Context, command port.OrderSyncCommand) (port.UpsertOutcome, error) {
				return port.UpsertOutcomeCreated, nil
			},
		},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, syncErr := service.SyncOrderByID(context.Background(), "manual", 1001); !errors.Is(syncErr, ErrOrderNotFound) {
		t.Fatalf("SyncOrderByID(not found) error = %v, want ErrOrderNotFound", syncErr)
	}
}

// TestSyncOrdersBreakerOpen verifies breaker-open mapping behavior.
func TestSyncOrdersBreakerOpen(t *testing.T) {
	openErr := errors.New("open")
	service, err := NewService(
		SyncConfig{Enabled: true},
		sourceMock{
			validateFn: func(ctx context.Context) error { return errors.New("source fail") },
			listFn: func(ctx context.Context, page int, pageSize int) ([]port.WooOrder, bool, error) {
				return nil, false, nil
			},
		},
		targetMock{
			upsertFn: func(ctx context.Context, command port.OrderSyncCommand) (port.UpsertOutcome, error) {
				return port.UpsertOutcomeCreated, nil
			},
		},
		nil,
		nil,
		CircuitBreakers{
			Source: breakerMock{
				executeFn: func(operation func() error) error { return openErr },
				isOpenFn:  func(err error) bool { return errors.Is(err, openErr) },
			},
		},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if validateErr := service.ValidateIntegration(context.Background()); !errors.Is(validateErr, ErrIntegrationUnavailable) {
		t.Fatalf("ValidateIntegration() error = %v, want ErrIntegrationUnavailable", validateErr)
	}
}

// TestMapOrderToCommand verifies order mapping behavior.
func TestMapOrderToCommand(t *testing.T) {
	command, ok, reason := mapOrderToCommand(port.WooOrder{
		ID:                   9,
		Status:               "processing",
		BillingEmail:         "test@example.com",
		BillingFirstName:     "Test",
		BillingLastName:      "User",
		BillingAddress1:      "A",
		BillingAddress2:      "B",
		BillingCity:          "11001",
		ShippingAddressLine1: "A",
		ShippingAddressLine2: "B",
		ShippingCityCode:     "11001",
		Items: []port.WooOrderItem{
			{SKU: "SKU-1", Name: "Item", Quantity: 1, Value: 20000},
			{SKU: "", Name: "Ignored", Quantity: 1, Value: 10000},
		},
		ShippingCharges: []port.WooOrderShippingCharge{
			{MethodID: "flat_rate", MethodTitle: "Flat Rate", Price: 9000},
		},
		Comments: []port.WooOrderComment{
			{Author: "system", Description: "comment", OccurredAt: time.Now().UTC()},
		},
		Metadata: map[string]string{"raw_key": "raw_value"},
	})
	if !ok {
		t.Fatalf("expected mapped command")
	}
	if reason != "" {
		t.Fatalf("reason = %q, want empty", reason)
	}
	if command.Identifier != "9" {
		t.Fatalf("command.Identifier = %q, want %q", command.Identifier, "9")
	}
	if len(command.Items) != 2 {
		t.Fatalf("len(command.Items) = %d, want 2", len(command.Items))
	}
	if command.Items[0].SKU != "SKU-1" || command.Items[1].Name != "Ignored" {
		t.Fatalf("command.Items = %+v, want sku and quota/name rows", command.Items)
	}
	if command.Items[0].Value != 20000 || command.Items[1].Value != 10000 {
		t.Fatalf("command.Items values = %+v, want mapped value rows", command.Items)
	}
	if command.ShippingAddress == nil || command.ShippingAddress.Address != "A" {
		t.Fatalf("expected billing snapshot address, got %+v", command.ShippingAddress)
	}
	if command.Metadata != nil {
		t.Fatalf("command.Metadata = %+v, want nil", command.Metadata)
	}
	if len(command.ShippingCharges) != 1 || command.ShippingCharges[0].MethodID != "flat_rate" {
		t.Fatalf("command.ShippingCharges = %+v, want one flat_rate row", command.ShippingCharges)
	}
}
