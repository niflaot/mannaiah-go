package contact

import (
	"context"
	errorspkg "errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"mannaiah/module/woocommerce/port"
)

// TestMapOrderToCommand verifies order-to-command mapping behavior.
func TestMapOrderToCommand(t *testing.T) {
	command, shouldProcess := mapOrderToCommand(port.WooOrder{
		BillingEmail:     " user@example.com ",
		BillingFirstName: "First",
		BillingLastName:  "Last",
		BillingPhone:     "+57 300 444 5566",
		BillingAddress1:  "Street 1",
		BillingAddress2:  "Suite 1",
		BillingCity:      "Bogota",
		Metadata:         map[string]string{billingDocumentMetaKey: "  98765 "},
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

	_, shouldProcess = mapOrderToCommand(port.WooOrder{BillingEmail: "   "})
	if shouldProcess {
		t.Fatalf("expected order without email to be skipped")
	}
	_, shouldProcess = mapOrderToCommand(port.WooOrder{BillingEmail: "user@example.com", BillingFirstName: "", BillingLastName: "Doe"})
	if shouldProcess {
		t.Fatalf("expected order without complete names to be skipped")
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

	service.publishEvent(context.Background(), buildSyncStartedEvent("manual"))
}

// TestProcessPageContextTimeout verifies context timeout behavior during processing.
func TestProcessPageContextTimeout(t *testing.T) {
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
	if processErr := service.processPage(ctx, []port.WooOrder{{BillingEmail: "timeout@example.com", BillingFirstName: "Time", BillingLastName: "Out"}}, map[string]struct{}{}, summary); !errorspkg.Is(processErr, context.DeadlineExceeded) {
		t.Fatalf("processPage() error = %v, want context.DeadlineExceeded", processErr)
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
