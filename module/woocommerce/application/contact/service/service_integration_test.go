package service

import (
	"context"
	errorspkg "errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	woocontactevent "mannaiah/module/woocommerce/application/contact/event"
	"mannaiah/module/woocommerce/port"
)

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

// TestValidateIntegrationCircuitOpen verifies source breaker open-state mapping behavior.
func TestValidateIntegrationCircuitOpen(t *testing.T) {
	source := &sourceMock{}
	target := &targetMock{outcomes: map[string]port.UpsertOutcome{}, errors: map[string]error{}}
	breaker := &circuitBreakerMock{
		executeErr: errorspkg.New("source breaker open"),
		openError:  true,
	}

	service, err := NewService(
		SyncConfig{Enabled: true},
		source,
		target,
		nil,
		nil,
		CircuitBreakers{Source: breaker},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	validationErr := service.ValidateIntegration(context.Background())
	if !errorspkg.Is(validationErr, ErrIntegrationUnavailable) {
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
					CreatedAt:        mustTimeFromRFC3339(t, "2024-03-04T10:00:00Z"),
					ID:               1002,
					Metadata:         map[string]string{billingDocumentMetaKey: "1234"},
				},
				{
					BillingEmail:     "john@example.com",
					BillingFirstName: "John",
					BillingLastName:  "Doe",
					CreatedAt:        mustTimeFromRFC3339(t, "2024-03-01T08:00:00Z"),
					ID:               1001,
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
	if publisher.events[0].Topic != woocontactevent.TopicContactsSyncStarted {
		t.Fatalf("events[0].Topic = %q, want %q", publisher.events[0].Topic, woocontactevent.TopicContactsSyncStarted)
	}
	if publisher.events[1].Topic != woocontactevent.TopicContactsSyncCompleted {
		t.Fatalf("events[1].Topic = %q, want %q", publisher.events[1].Topic, woocontactevent.TopicContactsSyncCompleted)
	}

	if len(target.commands) != 2 {
		t.Fatalf("len(commands) = %d, want %d", len(target.commands), 2)
	}

	var johnCommand *port.ContactSyncCommand
	for index := range target.commands {
		if target.commands[index].Email == "john@example.com" {
			johnCommand = &target.commands[index]
			break
		}
	}
	if johnCommand == nil {
		t.Fatalf("expected john@example.com upsert command")
	}
	if johnCommand.Phone != "+573124567890" {
		t.Fatalf("john phone = %q, want %q", johnCommand.Phone, "+573124567890")
	}
	if johnCommand.DocumentNumber != "1234" {
		t.Fatalf("john document number = %q, want %q", johnCommand.DocumentNumber, "1234")
	}
	if johnCommand.DocumentType != "CC" {
		t.Fatalf("john document type = %q, want %q", johnCommand.DocumentType, "CC")
	}
	if johnCommand.CreatedAt == nil || johnCommand.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00") != "2024-03-01T08:00:00Z" {
		t.Fatalf("john createdAt = %v, want %q", johnCommand.CreatedAt, "2024-03-01T08:00:00Z")
	}
	if johnCommand.Metadata[syncMetadataOldestOrderIDKey] != "1001" {
		t.Fatalf("john metadata oldest_order_id = %q, want %q", johnCommand.Metadata[syncMetadataOldestOrderIDKey], "1001")
	}
	if johnCommand.Metadata[syncMetadataOldestOrderAtKey] != "2024-03-01T08:00:00Z" {
		t.Fatalf("john metadata oldest_order_created_at = %q, want %q", johnCommand.Metadata[syncMetadataOldestOrderAtKey], "2024-03-01T08:00:00Z")
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

// TestSyncContactsUpsertCircuitOpen verifies degraded upsert behavior when breaker opens.
func TestSyncContactsUpsertCircuitOpen(t *testing.T) {
	source := &sourceMock{
		pages: [][]port.WooOrder{
			{{BillingEmail: "broken@example.com", BillingFirstName: "Broken", BillingLastName: "Case"}},
		},
	}
	target := &targetMock{
		outcomes: map[string]port.UpsertOutcome{},
		errors:   map[string]error{},
	}
	breaker := &circuitBreakerMock{
		executeErr: errorspkg.New("upsert breaker open"),
		openError:  true,
	}

	service, err := NewService(
		SyncConfig{Enabled: true},
		source,
		target,
		&publisherMock{},
		zap.NewNop(),
		CircuitBreakers{Upsert: breaker},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncContacts(context.Background(), "manual")
	if syncErr != nil {
		t.Fatalf("SyncContacts() error = %v", syncErr)
	}
	if summary.Processed != 1 {
		t.Fatalf("summary.Processed = %d, want %d", summary.Processed, 1)
	}
	if summary.Failed != 1 {
		t.Fatalf("summary.Failed = %d, want %d", summary.Failed, 1)
	}
	if len(target.commands) != 0 {
		t.Fatalf("len(commands) = %d, want %d when upsert breaker is open", len(target.commands), 0)
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
	if publisher.events[1].Topic != woocontactevent.TopicContactsSyncFailed {
		t.Fatalf("events[1].Topic = %q, want %q", publisher.events[1].Topic, woocontactevent.TopicContactsSyncFailed)
	}
}

// TestSyncContactsListErrorDoesNotApplyPartialWrites verifies that source-page failures do not upsert partial state.
func TestSyncContactsListErrorDoesNotApplyPartialWrites(t *testing.T) {
	source := &sourceMock{
		pages: [][]port.WooOrder{
			{{BillingEmail: "first@example.com", BillingFirstName: "First", BillingLastName: "User"}},
			{},
		},
		listErrAtPage: map[int]error{2: errorspkg.New("upstream page failure")},
	}
	target := &targetMock{outcomes: map[string]port.UpsertOutcome{}, errors: map[string]error{}}

	service, err := NewService(SyncConfig{Enabled: true}, source, target, &publisherMock{}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, syncErr := service.SyncContacts(context.Background(), "manual"); syncErr == nil {
		t.Fatalf("expected SyncContacts() error")
	} else if !strings.Contains(syncErr.Error(), "processed=0") {
		t.Fatalf("SyncContacts() error = %q, expected progress diagnostics", syncErr.Error())
	}

	if len(target.commands) != 0 {
		t.Fatalf("len(commands) = %d, want %d when page listing fails before apply phase", len(target.commands), 0)
	}
}

// searchableSourceMock defines source behavior with search support for targeted contact sync tests.
type searchableSourceMock struct {
	// sourceMock defines baseline source behavior.
	sourceMock
	// searchPages defines paginated search responses.
	searchPages [][]port.WooOrder
}

// SearchOrders resolves paginated search results.
func (m *searchableSourceMock) SearchOrders(ctx context.Context, search string, page int, pageSize int) (orders []port.WooOrder, hasNext bool, err error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if page <= 0 || page > len(m.searchPages) {
		return nil, false, nil
	}

	items := m.searchPages[page-1]
	return items, page < len(m.searchPages), nil
}

// TestSyncContactByEmailSuccess verifies targeted contact-sync behavior.
func TestSyncContactByEmailSuccess(t *testing.T) {
	source := &sourceMock{
		pages: [][]port.WooOrder{
			{
				{
					BillingEmail:     "one@example.com",
					BillingFirstName: "One",
					BillingLastName:  "Person",
					BillingAddress1:  "A",
					BillingCity:      "11001",
					Items:            []port.WooOrderItem{{SKU: "SKU-1", Quantity: 1}},
				},
				{
					BillingEmail:     "target@example.com",
					BillingFirstName: "Target",
					BillingLastName:  "Person",
					BillingAddress1:  "B",
					BillingCity:      "05001",
					CreatedAt:        mustTimeFromRFC3339(t, "2024-04-01T08:00:00Z"),
					ID:               2001,
				},
			},
		},
	}
	target := &targetMock{
		outcomes: map[string]port.UpsertOutcome{
			"target@example.com": port.UpsertOutcomeCreated,
		},
		errors: map[string]error{},
	}
	service, err := NewService(SyncConfig{Enabled: true, WorkerCount: 2}, source, target, &publisherMock{}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncContactByEmail(context.Background(), "manual", "target@example.com")
	if syncErr != nil {
		t.Fatalf("SyncContactByEmail() error = %v", syncErr)
	}
	if summary.Processed != 1 || summary.Created != 1 {
		t.Fatalf("summary = %+v, want processed=1 created=1", summary)
	}
	if len(target.commands) != 1 || target.commands[0].Email != "target@example.com" {
		t.Fatalf("target.commands = %+v, want one target@example.com command", target.commands)
	}
}

// TestSyncContactByEmailSearchSource verifies targeted contact-sync behavior using source search capabilities.
func TestSyncContactByEmailSearchSource(t *testing.T) {
	source := &searchableSourceMock{
		sourceMock: sourceMock{},
		searchPages: [][]port.WooOrder{
			{
				{
					BillingEmail:     "target@example.com",
					BillingFirstName: "Target",
					BillingLastName:  "Person",
					BillingAddress1:  "B",
					BillingCity:      "05001",
					CreatedAt:        mustTimeFromRFC3339(t, "2024-04-01T08:00:00Z"),
					ID:               2001,
				},
			},
		},
	}
	target := &targetMock{
		outcomes: map[string]port.UpsertOutcome{
			"target@example.com": port.UpsertOutcomeUpdated,
		},
		errors: map[string]error{},
	}
	service, err := NewService(SyncConfig{Enabled: true, WorkerCount: 1}, source, target, &publisherMock{}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncContactByEmail(context.Background(), "manual", "target@example.com")
	if syncErr != nil {
		t.Fatalf("SyncContactByEmail() error = %v", syncErr)
	}
	if summary.Processed != 1 || summary.Updated != 1 {
		t.Fatalf("summary = %+v, want processed=1 updated=1", summary)
	}
}

// TestSyncContactByEmailValidation verifies targeted contact-sync validation and not-found behavior.
func TestSyncContactByEmailValidation(t *testing.T) {
	source := &sourceMock{pages: [][]port.WooOrder{}}
	target := &targetMock{outcomes: map[string]port.UpsertOutcome{}, errors: map[string]error{}}
	service, err := NewService(SyncConfig{Enabled: true}, source, target, &publisherMock{}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, syncErr := service.SyncContactByEmail(context.Background(), "manual", ""); !errorspkg.Is(syncErr, ErrInvalidEmail) {
		t.Fatalf("SyncContactByEmail(empty) error = %v, want ErrInvalidEmail", syncErr)
	}
	if _, syncErr := service.SyncContactByEmail(context.Background(), "manual", "missing@example.com"); !errorspkg.Is(syncErr, ErrContactNotFound) {
		t.Fatalf("SyncContactByEmail(missing) error = %v, want ErrContactNotFound", syncErr)
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

// mustTimeFromRFC3339 parses RFC3339 time values for test fixtures.
func mustTimeFromRFC3339(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("time.Parse(%q) error = %v", value, err)
	}

	return parsed
}
