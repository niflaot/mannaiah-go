package service

import (
	"context"
	errorspkg "errors"
	"testing"
	"time"

	"go.uber.org/zap"
	woocontactevent "mannaiah/module/woocommerce/application/contact/event"
	"mannaiah/module/woocommerce/port"
)

// TestMapOrderToCommand verifies order-to-command mapping behavior.
func TestMapOrderToCommand(t *testing.T) {
	orderCreatedAt := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	command, shouldProcess := mapOrderToCommand(port.WooOrder{
		BillingEmail:     " user@example.com ",
		BillingFirstName: "First",
		BillingLastName:  "Last",
		BillingPhone:     "+57 300 444 5566",
		BillingAddress1:  "Street 1",
		BillingAddress2:  "Suite 1",
		BillingCity:      "Bogota",
		CreatedAt:        orderCreatedAt,
		ID:               9001,
		Metadata: map[string]string{
			billingDocumentMetaKey:                          "  98765 ",
			"flock_checker_privacy_accept":                  "yes",
			"flock_checker_privacy_accept_accepted_at":      "2026-03-13 13:05:22",
			"flock_checker_privacy_accept_accepted_at_utc":  "2026-03-13T18:05:22Z",
			"flock_checker_circle_optin":                    "yes",
			"flock_checker_circle_optin_accepted_at":        "2026-03-13 13:05:22",
			"flock_checker_circle_optin_accepted_at_utc":    "2026-03-13T18:05:22Z",
			"flock_checker_terminos_extra":                  "no",
			"flock_checker_terminos_extra_accepted_at":      "2026-03-13 13:05:22",
			"flock_checker_terminos_extra_accepted_at_utc":  "2026-03-13T18:05:22Z",
			"integration.woocommerce.order_id":              "ignored-for-contact-sync",
			"flock_checker_unknown_dynamic_checker_enabled": "yes",
		},
	})
	if !shouldProcess {
		t.Fatalf("expected order to be processed")
	}
	if command.Email != "user@example.com" {
		t.Fatalf("command.Email = %q, want %q", command.Email, "user@example.com")
	}
	if command.DocumentNumber != "98765" {
		t.Fatalf("command.DocumentNumber = %q, want %q", command.DocumentNumber, "98765")
	}
	if command.DocumentType != "CC" {
		t.Fatalf("command.DocumentType = %q, want %q", command.DocumentType, "CC")
	}
	if command.CreatedAt == nil || !command.CreatedAt.Equal(orderCreatedAt) {
		t.Fatalf("command.CreatedAt = %v, want %v", command.CreatedAt, orderCreatedAt)
	}
	if command.Metadata[syncMetadataSourceKey] != syncMetadataSourceValue {
		t.Fatalf("command.Metadata[source] = %q, want %q", command.Metadata[syncMetadataSourceKey], syncMetadataSourceValue)
	}
	if command.Metadata[syncMetadataOldestOrderIDKey] != "9001" {
		t.Fatalf("command.Metadata[oldest_order_id] = %q, want %q", command.Metadata[syncMetadataOldestOrderIDKey], "9001")
	}
	if command.Metadata[syncMetadataOldestOrderAtKey] != "2024-01-02T03:04:05Z" {
		t.Fatalf("command.Metadata[oldest_order_created_at] = %q, want %q", command.Metadata[syncMetadataOldestOrderAtKey], "2024-01-02T03:04:05Z")
	}
	if command.Metadata["flock_checker_privacy_accept"] != "yes" {
		t.Fatalf("command.Metadata[flock_checker_privacy_accept] = %q, want %q", command.Metadata["flock_checker_privacy_accept"], "yes")
	}
	if command.Metadata["flock_checker_privacy_accept_accepted_at"] != "2026-03-13 13:05:22" {
		t.Fatalf("command.Metadata[flock_checker_privacy_accept_accepted_at] = %q, want %q", command.Metadata["flock_checker_privacy_accept_accepted_at"], "2026-03-13 13:05:22")
	}
	if command.Metadata["flock_checker_privacy_accept_accepted_at_utc"] != "2026-03-13T18:05:22Z" {
		t.Fatalf("command.Metadata[flock_checker_privacy_accept_accepted_at_utc] = %q, want %q", command.Metadata["flock_checker_privacy_accept_accepted_at_utc"], "2026-03-13T18:05:22Z")
	}
	if command.Metadata["flock_checker_circle_optin"] != "yes" {
		t.Fatalf("command.Metadata[flock_checker_circle_optin] = %q, want %q", command.Metadata["flock_checker_circle_optin"], "yes")
	}
	if command.Metadata["flock_checker_circle_optin_accepted_at"] != "2026-03-13 13:05:22" {
		t.Fatalf("command.Metadata[flock_checker_circle_optin_accepted_at] = %q, want %q", command.Metadata["flock_checker_circle_optin_accepted_at"], "2026-03-13 13:05:22")
	}
	if command.Metadata["flock_checker_circle_optin_accepted_at_utc"] != "2026-03-13T18:05:22Z" {
		t.Fatalf("command.Metadata[flock_checker_circle_optin_accepted_at_utc] = %q, want %q", command.Metadata["flock_checker_circle_optin_accepted_at_utc"], "2026-03-13T18:05:22Z")
	}
	if command.Metadata["flock_checker_terminos_extra"] != "no" {
		t.Fatalf("command.Metadata[flock_checker_terminos_extra] = %q, want %q", command.Metadata["flock_checker_terminos_extra"], "no")
	}
	if command.Metadata["flock_checker_terminos_extra_accepted_at"] != "2026-03-13 13:05:22" {
		t.Fatalf("command.Metadata[flock_checker_terminos_extra_accepted_at] = %q, want %q", command.Metadata["flock_checker_terminos_extra_accepted_at"], "2026-03-13 13:05:22")
	}
	if command.Metadata["flock_checker_terminos_extra_accepted_at_utc"] != "2026-03-13T18:05:22Z" {
		t.Fatalf("command.Metadata[flock_checker_terminos_extra_accepted_at_utc] = %q, want %q", command.Metadata["flock_checker_terminos_extra_accepted_at_utc"], "2026-03-13T18:05:22Z")
	}
	if command.Metadata["flock_checker_unknown_dynamic_checker_enabled"] != "yes" {
		t.Fatalf("command.Metadata[flock_checker_unknown_dynamic_checker_enabled] = %q, want %q", command.Metadata["flock_checker_unknown_dynamic_checker_enabled"], "yes")
	}

	_, shouldProcess = mapOrderToCommand(port.WooOrder{BillingEmail: "   "})
	if shouldProcess {
		t.Fatalf("expected order without email to be skipped")
	}
	command, shouldProcess = mapOrderToCommand(port.WooOrder{
		BillingEmail:   "user@example.com",
		BillingCompany: "Acme Corp",
	})
	if !shouldProcess {
		t.Fatalf("expected order with billing company to be processed as legal contact")
	}
	if command.LegalName != "Acme Corp" {
		t.Fatalf("command.LegalName = %q, want %q", command.LegalName, "Acme Corp")
	}

	_, shouldProcess = mapOrderToCommand(port.WooOrder{BillingEmail: "user@example.com", BillingFirstName: "", BillingLastName: "Doe", BillingCompany: ""})
	if shouldProcess {
		t.Fatalf("expected order without personal names and company to be skipped")
	}
}

// TestMapOrderToCommandBackfillsCircleOptInAcceptedAt verifies checker accepted-at fallback behavior.
func TestMapOrderToCommandBackfillsCircleOptInAcceptedAt(t *testing.T) {
	command, shouldProcess := mapOrderToCommand(port.WooOrder{
		BillingEmail:     "user@example.com",
		BillingFirstName: "First",
		BillingLastName:  "Last",
		CreatedAt:        time.Date(2026, time.March, 13, 18, 5, 22, 0, time.UTC),
		ID:               9100,
		Metadata: map[string]string{
			"flock_checker_circle_optin": "yes",
		},
	})
	if !shouldProcess {
		t.Fatalf("expected order to be processed")
	}
	if command.Metadata["flock_checker_circle_optin"] != "yes" {
		t.Fatalf("command.Metadata[flock_checker_circle_optin] = %q, want %q", command.Metadata["flock_checker_circle_optin"], "yes")
	}
	if command.Metadata["flock_checker_circle_optin_accepted_at"] != "2026-03-13 13:05:22" {
		t.Fatalf("command.Metadata[flock_checker_circle_optin_accepted_at] = %q, want %q", command.Metadata["flock_checker_circle_optin_accepted_at"], "2026-03-13 13:05:22")
	}
	if command.Metadata["flock_checker_circle_optin_accepted_at_utc"] != "2026-03-13T18:05:22Z" {
		t.Fatalf("command.Metadata[flock_checker_circle_optin_accepted_at_utc] = %q, want %q", command.Metadata["flock_checker_circle_optin_accepted_at_utc"], "2026-03-13T18:05:22Z")
	}
}

// TestMapOrderToCommandMapsCircleOptOutToRejectedAt verifies circle opt-out metadata mapping behavior.
func TestMapOrderToCommandMapsCircleOptOutToRejectedAt(t *testing.T) {
	command, shouldProcess := mapOrderToCommand(port.WooOrder{
		BillingEmail:     "user@example.com",
		BillingFirstName: "First",
		BillingLastName:  "Last",
		CreatedAt:        time.Date(2026, time.March, 13, 18, 5, 22, 0, time.UTC),
		ID:               9101,
		Metadata: map[string]string{
			"flock_checker_circle_optin":             "no",
			"flock_checker_circle_optin_accepted_at": "2026-03-13 13:05:22",
		},
	})
	if !shouldProcess {
		t.Fatalf("expected order to be processed")
	}
	if command.Metadata["flock_checker_circle_optin"] != "no" {
		t.Fatalf("command.Metadata[flock_checker_circle_optin] = %q, want %q", command.Metadata["flock_checker_circle_optin"], "no")
	}
	if command.Metadata["flock_checker_circle_optin_rejected_at"] != "2026-03-13 13:05:22" {
		t.Fatalf("command.Metadata[flock_checker_circle_optin_rejected_at] = %q, want %q", command.Metadata["flock_checker_circle_optin_rejected_at"], "2026-03-13 13:05:22")
	}
	if command.Metadata["flock_checker_circle_optin_rejected_at_utc"] != "2026-03-13T18:05:22Z" {
		t.Fatalf("command.Metadata[flock_checker_circle_optin_rejected_at_utc] = %q, want %q", command.Metadata["flock_checker_circle_optin_rejected_at_utc"], "2026-03-13T18:05:22Z")
	}
	if _, exists := command.Metadata["flock_checker_circle_optin_accepted_at"]; exists {
		t.Fatalf("expected flock_checker_circle_optin_accepted_at to be cleared when decision is no")
	}
}

// TestCollectCommandsFromOrdersKeepsOldestCreatedAt verifies duplicate-email command merge behavior.
func TestCollectCommandsFromOrdersKeepsOldestCreatedAt(t *testing.T) {
	summary := &SyncSummary{}
	commandIndexByEmail := map[string]int{}
	commands := collectCommandsFromOrders([]port.WooOrder{
		{
			ID:               1002,
			BillingEmail:     "same@example.com",
			BillingFirstName: "Same",
			BillingLastName:  "User",
			CreatedAt:        time.Date(2024, time.March, 10, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:               1001,
			BillingEmail:     "same@example.com",
			BillingFirstName: "Same",
			BillingLastName:  "User",
			CreatedAt:        time.Date(2024, time.March, 8, 9, 0, 0, 0, time.UTC),
		},
	}, commandIndexByEmail, nil, summary)
	if len(commands) != 1 {
		t.Fatalf("len(commands) = %d, want %d", len(commands), 1)
	}
	if commands[0].CreatedAt == nil || commands[0].CreatedAt.UTC().Format(time.RFC3339) != "2024-03-08T09:00:00Z" {
		t.Fatalf("commands[0].CreatedAt = %v, want %q", commands[0].CreatedAt, "2024-03-08T09:00:00Z")
	}
	if commands[0].Metadata[syncMetadataOldestOrderIDKey] != "1001" {
		t.Fatalf("commands[0].Metadata[oldest_order_id] = %q, want %q", commands[0].Metadata[syncMetadataOldestOrderIDKey], "1001")
	}
	if summary.Skipped != 1 {
		t.Fatalf("summary.Skipped = %d, want %d", summary.Skipped, 1)
	}
}

// TestCollectCommandsFromOrdersPrefersLatestCheckerMetadata verifies checker metadata values use latest duplicate-order values.
func TestCollectCommandsFromOrdersPrefersLatestCheckerMetadata(t *testing.T) {
	summary := &SyncSummary{}
	commands := collectCommandsFromOrders([]port.WooOrder{
		{
			ID:               1001,
			BillingEmail:     "same@example.com",
			BillingFirstName: "Same",
			BillingLastName:  "User",
			CreatedAt:        time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC),
			Metadata: map[string]string{
				"flock_checker_circle_optin":                 "no",
				"flock_checker_circle_optin_rejected_at":     "2026-03-10 07:00:00",
				"flock_checker_circle_optin_rejected_at_utc": "2026-03-10T12:00:00Z",
			},
		},
		{
			ID:               1002,
			BillingEmail:     "same@example.com",
			BillingFirstName: "Same",
			BillingLastName:  "User",
			CreatedAt:        time.Date(2026, time.March, 13, 18, 5, 22, 0, time.UTC),
			Metadata: map[string]string{
				"flock_checker_circle_optin":             "yes",
				"flock_checker_circle_optin_accepted_at": "2026-03-13 13:05:22",
			},
		},
	}, map[string]int{}, nil, summary)
	if len(commands) != 1 {
		t.Fatalf("len(commands) = %d, want 1", len(commands))
	}
	if commands[0].Metadata["flock_checker_circle_optin"] != "yes" {
		t.Fatalf("commands[0].Metadata[flock_checker_circle_optin] = %q, want yes", commands[0].Metadata["flock_checker_circle_optin"])
	}
	if commands[0].Metadata["flock_checker_circle_optin_accepted_at"] != "2026-03-13 13:05:22" {
		t.Fatalf("commands[0].Metadata[flock_checker_circle_optin_accepted_at] = %q, want %q", commands[0].Metadata["flock_checker_circle_optin_accepted_at"], "2026-03-13 13:05:22")
	}
	if _, exists := commands[0].Metadata["flock_checker_circle_optin_rejected_at"]; exists {
		t.Fatalf("expected rejected_at metadata to be cleared when latest checker decision is yes")
	}
}

// TestNormalizeHelpers verifies private normalization helper behavior.
func TestNormalizeHelpers(t *testing.T) {
	cfg := normalizeSyncConfig(SyncConfig{})
	if cfg.PageSize != 100 {
		t.Fatalf("cfg.PageSize = %d, want %d", cfg.PageSize, 100)
	}
	if cfg.WorkerCount != 8 {
		t.Fatalf("cfg.WorkerCount = %d, want %d", cfg.WorkerCount, 8)
	}
	if normalizeTrigger("  ") != "manual" {
		t.Fatalf("normalizeTrigger(\"  \") should fallback to manual")
	}
	if normalizePhone("+57 312 456 7890") != "+573124567890" {
		t.Fatalf("normalizePhone() should normalize +57 values")
	}
	if normalizePhone("  3112233445  ") != "+573112233445" {
		t.Fatalf("normalizePhone() should normalize local values")
	}
	if mapDocumentNumber(map[string]string{}) != "" {
		t.Fatalf("mapDocumentNumber(empty) should be empty")
	}
	if resolveCircuitBreakers(nil).Source != nil {
		t.Fatalf("resolveCircuitBreakers(nil).Source should be nil")
	}
	manualBreakers := CircuitBreakers{Source: &circuitBreakerMock{}}
	if resolveCircuitBreakers([]CircuitBreakers{manualBreakers}).Source == nil {
		t.Fatalf("resolveCircuitBreakers(values).Source should preserve values")
	}
	progress := formatSyncProgress(&SyncSummary{Trigger: "manual", Processed: 1, Created: 1})
	if progress == "" {
		t.Fatalf("formatSyncProgress(summary) should not be empty")
	}
	if formatSyncProgress(nil) == "" {
		t.Fatalf("formatSyncProgress(nil) should not be empty")
	}
}

// TestApplyOutcome verifies upsert outcome accounting behavior.
func TestApplyOutcome(t *testing.T) {
	summary := &SyncSummary{}
	applyOutcome(summary, port.UpsertOutcomeCreated)
	applyOutcome(summary, port.UpsertOutcomeUpdated)
	applyOutcome(summary, port.UpsertOutcomeUnchanged)

	if summary.Created != 1 {
		t.Fatalf("summary.Created = %d, want %d", summary.Created, 1)
	}
	if summary.Updated != 1 {
		t.Fatalf("summary.Updated = %d, want %d", summary.Updated, 1)
	}
	if summary.Unchanged != 1 {
		t.Fatalf("summary.Unchanged = %d, want %d", summary.Unchanged, 1)
	}
}

// TestPublishEventNoPanic verifies event publication fallback behavior.
func TestPublishEventNoPanic(t *testing.T) {
	source := &sourceMock{}
	target := &targetMock{outcomes: map[string]port.UpsertOutcome{}, errors: map[string]error{}}

	service, err := NewService(SyncConfig{Enabled: true}, source, target, nil, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	service.publishEvent(context.Background(), woocontactevent.NewSyncStartedEvent("manual"))
}

// TestProcessCommandsContextTimeout verifies context timeout behavior during processing.
func TestProcessCommandsContextTimeout(t *testing.T) {
	source := &sourceMock{}
	target := &targetMock{
		outcomes: map[string]port.UpsertOutcome{},
		errors: map[string]error{
			"timeout@example.com": context.DeadlineExceeded,
		},
	}

	service, err := NewService(SyncConfig{Enabled: true}, source, target, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond)

	summary := &SyncSummary{}
	commands := collectCommandsFromOrders(
		[]port.WooOrder{{BillingEmail: "timeout@example.com", BillingFirstName: "Time", BillingLastName: "Out"}},
		map[string]int{},
		nil,
		summary,
	)
	if processErr := service.processCommands(ctx, commands, summary); !errorspkg.Is(processErr, context.DeadlineExceeded) {
		t.Fatalf("processCommands() error = %v, want context.DeadlineExceeded", processErr)
	}
}

// TestExecuteWithBreaker verifies helper behavior for breaker-open and direct operation execution.
func TestExecuteWithBreaker(t *testing.T) {
	source := &sourceMock{}
	target := &targetMock{outcomes: map[string]port.UpsertOutcome{}, errors: map[string]error{}}
	service, err := NewService(SyncConfig{Enabled: true}, source, target, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	operationCalled := false
	if executeErr := service.executeWithBreaker(nil, ErrIntegrationUnavailable, func() error {
		operationCalled = true
		return nil
	}); executeErr != nil {
		t.Fatalf("executeWithBreaker(nil) error = %v", executeErr)
	}
	if !operationCalled {
		t.Fatalf("operation should execute when breaker is nil")
	}

	openBreaker := &circuitBreakerMock{
		executeErr: errorspkg.New("open"),
		openError:  true,
	}
	if executeErr := service.executeWithBreaker(openBreaker, ErrIntegrationUnavailable, func() error { return nil }); !errorspkg.Is(executeErr, ErrIntegrationUnavailable) {
		t.Fatalf("executeWithBreaker(open) error = %v, want ErrIntegrationUnavailable", executeErr)
	}
}
