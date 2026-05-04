package service

import (
	"context"
	errorspkg "errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"mannaiah/module/woocommerce/port"
)

// TestCollectCommandsFromOrders verifies order-command collection and deduplication behavior.
func TestCollectCommandsFromOrders(t *testing.T) {
	summary := &SyncSummary{}
	commands := collectCommandsFromOrders([]port.WooOrder{
		{
			ID:               10,
			Status:           "processing",
			BillingEmail:     "dup@example.com",
			BillingFirstName: "First",
			BillingLastName:  "Last",
			BillingAddress1:  "A",
			BillingCity:      "11001",
			Items:            []port.WooOrderItem{{SKU: "SKU-1", Quantity: 1}},
			CreatedAt:        time.Date(2026, time.February, 10, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:               10,
			Status:           "completed",
			BillingEmail:     "dup@example.com",
			BillingFirstName: "First",
			BillingLastName:  "Last",
			BillingAddress1:  "A",
			BillingCity:      "11001",
			Items:            []port.WooOrderItem{{SKU: "SKU-1", Quantity: 1}},
			CreatedAt:        time.Date(2026, time.February, 10, 11, 0, 0, 0, time.UTC),
		},
		{
			ID:               12,
			Status:           "completed",
			BillingEmail:     "no-item@example.com",
			BillingFirstName: "No",
			BillingLastName:  "Item",
		},
	}, map[string]int{}, nil, summary, zap.NewNop())

	if len(commands) != 1 {
		t.Fatalf("len(commands) = %d, want 1", len(commands))
	}
	if summary.Skipped != 2 {
		t.Fatalf("summary.Skipped = %d, want 2", summary.Skipped)
	}
	if commands[0].Status != "completed" {
		t.Fatalf("merged command status = %q, want %q", commands[0].Status, "completed")
	}
}

// TestCollectCommandsFromOrdersLogsSkipReasons verifies skip warning log behavior with non-sensitive context.
func TestCollectCommandsFromOrdersLogsSkipReasons(t *testing.T) {
	core, entries := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	summary := &SyncSummary{}
	commands := collectCommandsFromOrders(
		[]port.WooOrder{
			{
				ID:               1001,
				Status:           "processing",
				BillingEmail:     "dup@example.com",
				BillingFirstName: "First",
				BillingLastName:  "Last",
				Items:            []port.WooOrderItem{{SKU: "SKU-1", Quantity: 1}},
			},
			{
				ID:               1001,
				Status:           "completed",
				BillingEmail:     "dup@example.com",
				BillingFirstName: "First",
				BillingLastName:  "Last",
				Items:            []port.WooOrderItem{{SKU: "SKU-1", Quantity: 1}},
			},
			{
				ID:               1002,
				Status:           "processing",
				BillingEmail:     "",
				BillingFirstName: "No",
				BillingLastName:  "Email",
				Items:            []port.WooOrderItem{{SKU: "SKU-2", Quantity: 1}},
			},
		},
		map[string]int{},
		nil,
		summary,
		logger,
	)

	if len(commands) != 1 {
		t.Fatalf("len(commands) = %d, want 1", len(commands))
	}
	if summary.Skipped != 2 {
		t.Fatalf("summary.Skipped = %d, want 2", summary.Skipped)
	}

	logs := entries.All()
	if len(logs) != 2 {
		t.Fatalf("len(logs) = %d, want 2", len(logs))
	}
	if logs[0].Message != "woocommerce order skipped" || logs[1].Message != "woocommerce order skipped" {
		t.Fatalf("unexpected log messages: %+v", logs)
	}
	if logs[0].ContextMap()["reason"] != "duplicate_identifier_merged" {
		t.Fatalf("logs[0].reason = %v, want duplicate_identifier_merged", logs[0].ContextMap()["reason"])
	}
	if logs[1].ContextMap()["reason"] != string(skipReasonMissingContactEmail) {
		t.Fatalf("logs[1].reason = %v, want %s", logs[1].ContextMap()["reason"], skipReasonMissingContactEmail)
	}
	if logs[0].ContextMap()["order_ref"] != "1001" || logs[1].ContextMap()["order_ref"] != "1002" {
		t.Fatalf("unexpected order_ref values: %v / %v", logs[0].ContextMap()["order_ref"], logs[1].ContextMap()["order_ref"])
	}
	if _, exists := logs[0].ContextMap()["billing_email"]; exists {
		t.Fatalf("expected no sensitive billing_email field in logs")
	}
}

// TestMergeOrderSyncCommand verifies command-merge behavior for duplicate identifiers.
func TestMergeOrderSyncCommand(t *testing.T) {
	existingCreatedAt := time.Date(2026, time.February, 12, 10, 0, 0, 0, time.UTC)
	existing := port.OrderSyncCommand{
		Identifier: "1001",
		Realm:      "woocommerce",
		Status:     "processing",
		CreatedAt:  &existingCreatedAt,
		Metadata:   map[string]string{"first": "value"},
		Comments: []port.OrderSyncComment{
			{Author: "system", Comment: "a"},
		},
	}

	candidateCreatedAt := time.Date(2026, time.February, 11, 10, 0, 0, 0, time.UTC)
	candidate := port.OrderSyncCommand{
		Identifier: "1001",
		Realm:      "woocommerce",
		Status:     "completed",
		CreatedAt:  &candidateCreatedAt,
		Items:      []port.OrderSyncItem{{SKU: "SKU-2", Quantity: 1}},
		Metadata:   map[string]string{"second": "value"},
		Comments: []port.OrderSyncComment{
			{Author: "agent", Comment: "b"},
		},
	}

	mergeOrderSyncCommand(&existing, candidate)
	if existing.Status != "completed" {
		t.Fatalf("existing.Status = %q, want %q", existing.Status, "completed")
	}
	if existing.CreatedAt == nil || !existing.CreatedAt.UTC().Equal(candidateCreatedAt) {
		t.Fatalf("existing.CreatedAt = %v, want %v", existing.CreatedAt, candidateCreatedAt)
	}
	if len(existing.Items) != 1 || existing.Items[0].SKU != "SKU-2" {
		t.Fatalf("existing.Items = %+v, want candidate items", existing.Items)
	}
	if len(existing.Comments) != 2 {
		t.Fatalf("len(existing.Comments) = %d, want 2", len(existing.Comments))
	}
	if existing.Metadata["first"] != "value" || existing.Metadata["second"] != "value" {
		t.Fatalf("existing.Metadata = %+v, want merged metadata", existing.Metadata)
	}
}

// TestProcessCommands verifies command-processing outcome and failure behavior.
func TestProcessCommands(t *testing.T) {
	service, err := NewService(
		SyncConfig{Enabled: true, WorkerCount: 2},
		sourceMock{
			validateFn: func(ctx context.Context) error { return nil },
			listFn: func(ctx context.Context, page int, pageSize int) ([]port.WooOrder, bool, error) {
				return nil, false, nil
			},
		},
		targetMock{
			upsertFn: func(ctx context.Context, command port.OrderSyncCommand) (port.UpsertOutcome, error) {
				if command.Identifier == "fail" {
					return "", errorspkg.New("upsert failed")
				}
				return port.UpsertOutcomeUpdated, nil
			},
		},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary := &SyncSummary{}
	processErr := service.processCommands(context.Background(), []port.OrderSyncCommand{
		{Identifier: "ok-1"},
		{Identifier: "fail"},
	}, summary)
	if processErr != nil {
		t.Fatalf("processCommands() error = %v", processErr)
	}
	if summary.Processed != 2 || summary.Updated != 1 || summary.Failed != 1 {
		t.Fatalf("summary = %+v, want processed=2 updated=1 failed=1", summary)
	}
}

// TestProcessCommandsCanceled verifies cancellation behavior.
func TestProcessCommandsCanceled(t *testing.T) {
	service, err := NewService(
		SyncConfig{Enabled: true, WorkerCount: 1},
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

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	processErr := service.processCommands(ctx, []port.OrderSyncCommand{{Identifier: "ok-1"}}, &SyncSummary{})
	if !errorspkg.Is(processErr, context.Canceled) {
		t.Fatalf("processCommands() error = %v, want context.Canceled", processErr)
	}
}

// TestApplyOutcome verifies summary counter updates by outcome.
func TestApplyOutcome(t *testing.T) {
	summary := &SyncSummary{}
	applyOutcome(summary, port.UpsertOutcomeCreated)
	applyOutcome(summary, port.UpsertOutcomeUnchanged)
	applyOutcome(summary, port.UpsertOutcomeUpdated)

	if summary.Created != 1 || summary.Unchanged != 1 || summary.Updated != 1 {
		t.Fatalf("summary = %+v, want created=1 unchanged=1 updated=1", summary)
	}
}
