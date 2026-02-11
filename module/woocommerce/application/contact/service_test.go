package contact

import (
	"context"
	errorspkg "errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"mannaiah/module/woocommerce/port"
)

// sourceMock defines order source behavior for sync tests.
type sourceMock struct {
	// validateErr defines validation errors.
	validateErr error
	// pages defines paginated order responses.
	pages [][]port.WooOrder
	// listErrAtPage defines page numbers that should return list errors.
	listErrAtPage map[int]error
}

// Validate verifies source connectivity.
func (m *sourceMock) Validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return m.validateErr
}

// ListOrders retrieves paginated order values.
func (m *sourceMock) ListOrders(ctx context.Context, page int, pageSize int) (orders []port.WooOrder, hasNext bool, err error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if listErr, hasError := m.listErrAtPage[page]; hasError {
		return nil, false, listErr
	}
	if page <= 0 || page > len(m.pages) {
		return nil, false, nil
	}

	items := m.pages[page-1]
	return items, page < len(m.pages), nil
}

// targetMock defines contact sync target behavior for sync tests.
type targetMock struct {
	// mu guards state mutation for concurrent workers.
	mu sync.Mutex
	// outcomes defines upsert outcomes keyed by email.
	outcomes map[string]port.UpsertOutcome
	// errors defines upsert errors keyed by email.
	errors map[string]error
	// commands stores received upsert commands.
	commands []port.ContactSyncCommand
}

// UpsertByEmail creates or updates contacts by email.
func (m *targetMock) UpsertByEmail(ctx context.Context, command port.ContactSyncCommand) (outcome port.UpsertOutcome, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.commands = append(m.commands, command)
	if err := m.errors[command.Email]; err != nil {
		return "", err
	}
	if outcome, ok := m.outcomes[command.Email]; ok {
		return outcome, nil
	}

	return port.UpsertOutcomeUpdated, nil
}

// publisherMock defines integration event publication behavior for sync tests.
type publisherMock struct {
	// events stores published integration events.
	events []port.IntegrationEvent
	// mu guards state mutation for concurrent event publication.
	mu sync.Mutex
}

// Publish captures integration events.
func (m *publisherMock) Publish(ctx context.Context, event port.IntegrationEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = append(m.events, event)
	return nil
}

// TestNewServiceValidation verifies constructor validation behavior.
func TestNewServiceValidation(t *testing.T) {
	target := &targetMock{outcomes: map[string]port.UpsertOutcome{}, errors: map[string]error{}}

	if _, err := NewService(SyncConfig{}, nil, target, nil, nil); !errorspkg.Is(err, ErrNilSource) {
		t.Fatalf("NewService(nil source) error = %v, want ErrNilSource", err)
	}
	if _, err := NewService(SyncConfig{}, &sourceMock{}, nil, nil, nil); !errorspkg.Is(err, ErrNilTarget) {
		t.Fatalf("NewService(nil target) error = %v, want ErrNilTarget", err)
	}
}

// TestValidateIntegration verifies integration validation behavior.
func TestValidateIntegration(t *testing.T) {
	source := &sourceMock{}
	target := &targetMock{outcomes: map[string]port.UpsertOutcome{}, errors: map[string]error{}}

	service, err := NewService(SyncConfig{Enabled: false}, source, target, nil, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if validationErr := service.ValidateIntegration(context.Background()); !errorspkg.Is(validationErr, ErrSyncDisabled) {
		t.Fatalf("ValidateIntegration() error = %v, want ErrSyncDisabled", validationErr)
	}

	source.validateErr = errorspkg.New("unreachable")
	service, err = NewService(SyncConfig{Enabled: true}, source, target, nil, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if validationErr := service.ValidateIntegration(context.Background()); !errorspkg.Is(validationErr, ErrIntegrationUnavailable) {
		t.Fatalf("ValidateIntegration() error = %v, want ErrIntegrationUnavailable", validationErr)
	}
}

// TestSyncContactsSuccess verifies successful sync behavior with summaries and events.
func TestSyncContactsSuccess(t *testing.T) {
	source := &sourceMock{
		pages: [][]port.WooOrder{
			{
				{
					BillingEmail:     "john@example.com",
					BillingFirstName: "John",
					BillingLastName:  "Doe",
					BillingPhone:     "+57 312 456 7890",
					BillingAddress1:  "Street 1",
					BillingAddress2:  "Suite 1",
					BillingCity:      "Bogota",
					Metadata:         map[string]string{billingDocumentMetaKey: "1234"},
				},
				{
					BillingEmail:     "john@example.com",
					BillingFirstName: "John",
					BillingLastName:  "Doe",
				},
				{
					BillingEmail:     "mary@example.com",
					BillingFirstName: "Mary",
					BillingLastName:  "Rose",
					BillingPhone:     "3001234567",
				},
				{BillingEmail: ""},
				{BillingEmail: "no-first@example.com", BillingFirstName: "", BillingLastName: "Last"},
			},
		},
	}
	target := &targetMock{
		outcomes: map[string]port.UpsertOutcome{
			"john@example.com": port.UpsertOutcomeCreated,
			"mary@example.com": port.UpsertOutcomeUnchanged,
		},
		errors: map[string]error{},
	}
	publisher := &publisherMock{}

	service, err := NewService(SyncConfig{Enabled: true, WorkerCount: 4, PageSize: 100}, source, target, publisher, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncContacts(context.Background(), "manual")
	if syncErr != nil {
		t.Fatalf("SyncContacts() error = %v", syncErr)
	}
	if summary.Processed != 2 {
		t.Fatalf("summary.Processed = %d, want %d", summary.Processed, 2)
	}
	if summary.Created != 1 {
		t.Fatalf("summary.Created = %d, want %d", summary.Created, 1)
	}
	if summary.Updated != 0 {
		t.Fatalf("summary.Updated = %d, want %d", summary.Updated, 0)
	}
	if summary.Unchanged != 1 {
		t.Fatalf("summary.Unchanged = %d, want %d", summary.Unchanged, 1)
	}
	if summary.Skipped != 3 {
		t.Fatalf("summary.Skipped = %d, want %d", summary.Skipped, 3)
	}
	if summary.Failed != 0 {
		t.Fatalf("summary.Failed = %d, want %d", summary.Failed, 0)
	}

	if len(publisher.events) != 2 {
		t.Fatalf("len(events) = %d, want %d", len(publisher.events), 2)
	}
	if publisher.events[0].Topic != TopicContactsSyncStarted {
		t.Fatalf("events[0].Topic = %q, want %q", publisher.events[0].Topic, TopicContactsSyncStarted)
	}
	if publisher.events[1].Topic != TopicContactsSyncCompleted {
		t.Fatalf("events[1].Topic = %q, want %q", publisher.events[1].Topic, TopicContactsSyncCompleted)
	}

	if len(target.commands) != 2 {
		t.Fatalf("len(commands) = %d, want %d", len(target.commands), 2)
	}
	if target.commands[0].Phone != "+573124567890" {
		t.Fatalf("commands[0].Phone = %q, want %q", target.commands[0].Phone, "+573124567890")
	}
	if target.commands[0].DocumentNumber != "1234" {
		t.Fatalf("commands[0].DocumentNumber = %q, want %q", target.commands[0].DocumentNumber, "1234")
	}
	if target.commands[0].DocumentType != "CC" {
		t.Fatalf("commands[0].DocumentType = %q, want %q", target.commands[0].DocumentType, "CC")
	}
}

// TestSyncContactsDeduplicatesAcrossPages verifies run-level dedupe behavior.
func TestSyncContactsDeduplicatesAcrossPages(t *testing.T) {
	source := &sourceMock{
		pages: [][]port.WooOrder{
			{{BillingEmail: "john@example.com", BillingFirstName: "John", BillingLastName: "Doe"}},
			{
				{BillingEmail: "john@example.com", BillingFirstName: "John", BillingLastName: "Doe"},
				{BillingEmail: "alice@example.com", BillingFirstName: "Alice", BillingLastName: "Doe"},
			},
		},
	}
	target := &targetMock{outcomes: map[string]port.UpsertOutcome{}, errors: map[string]error{}}
	service, err := NewService(SyncConfig{Enabled: true, WorkerCount: 2}, source, target, &publisherMock{}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncContacts(context.Background(), "manual")
	if syncErr != nil {
		t.Fatalf("SyncContacts() error = %v", syncErr)
	}
	if summary.Processed != 2 {
		t.Fatalf("summary.Processed = %d, want %d", summary.Processed, 2)
	}
	if summary.Skipped != 1 {
		t.Fatalf("summary.Skipped = %d, want %d", summary.Skipped, 1)
	}
}

// TestSyncContactsFailure verifies failing sync behavior and completed event emission.
func TestSyncContactsFailure(t *testing.T) {
	source := &sourceMock{
		pages: [][]port.WooOrder{
			{{BillingEmail: "broken@example.com", BillingFirstName: "Broken", BillingLastName: "Case"}},
		},
	}
	target := &targetMock{
		outcomes: map[string]port.UpsertOutcome{},
		errors: map[string]error{
			"broken@example.com": errorspkg.New("write failed"),
		},
	}
	publisher := &publisherMock{}

	service, err := NewService(SyncConfig{Enabled: true}, source, target, publisher, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncContacts(context.Background(), "manual")
	if syncErr != nil {
		t.Fatalf("SyncContacts() error = %v", syncErr)
	}
	if summary.Failed != 1 {
		t.Fatalf("summary.Failed = %d, want %d", summary.Failed, 1)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("len(events) = %d, want %d", len(publisher.events), 2)
	}
}

// TestSyncContactsListError verifies page fetch failures and failed event emission.
func TestSyncContactsListError(t *testing.T) {
	source := &sourceMock{
		listErrAtPage: map[int]error{1: errorspkg.New("upstream error")},
	}
	target := &targetMock{outcomes: map[string]port.UpsertOutcome{}, errors: map[string]error{}}
	publisher := &publisherMock{}

	service, err := NewService(SyncConfig{Enabled: true}, source, target, publisher, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, syncErr := service.SyncContacts(context.Background(), "manual"); syncErr == nil {
		t.Fatalf("expected SyncContacts() error")
	}
	if len(publisher.events) != 2 {
		t.Fatalf("len(events) = %d, want %d", len(publisher.events), 2)
	}
	if publisher.events[1].Topic != TopicContactsSyncFailed {
		t.Fatalf("events[1].Topic = %q, want %q", publisher.events[1].Topic, TopicContactsSyncFailed)
	}
}

// TestSyncContactsContextCancel verifies cancellation behavior.
func TestSyncContactsContextCancel(t *testing.T) {
	source := &sourceMock{
		pages: [][]port.WooOrder{
			{{BillingEmail: "john@example.com", BillingFirstName: "John", BillingLastName: "Doe"}},
		},
	}
	target := &targetMock{
		outcomes: map[string]port.UpsertOutcome{},
		errors: map[string]error{
			"john@example.com": context.Canceled,
		},
	}

	service, err := NewService(SyncConfig{Enabled: true}, source, target, &publisherMock{}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, syncErr := service.SyncContacts(ctx, "manual"); !errorspkg.Is(syncErr, ErrIntegrationUnavailable) {
		t.Fatalf("SyncContacts() error = %v, want ErrIntegrationUnavailable", syncErr)
	}
}

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
