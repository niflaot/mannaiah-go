package contacts

import (
	"context"
	errorspkg "errors"
	"fmt"
	"testing"
	"time"

	contactapplication "mannaiah/module/contacts/application"
	contactdomain "mannaiah/module/contacts/domain"
	contactport "mannaiah/module/contacts/port"
	"mannaiah/module/woocommerce/port"
)

// serviceMock defines contacts service behavior for upserter tests.
type serviceMock struct {
	// listResult defines list query result values.
	listResult *contactapplication.ListResult
	// listErr defines list query errors.
	listErr error
	// createErr defines create errors.
	createErr error
	// updateErr defines update errors.
	updateErr error
	// created stores create commands.
	created []contactapplication.CreateCommand
	// updates stores update commands.
	updates []contactapplication.UpdateCommand
	// updateIDs stores updated record ids.
	updateIDs []string
	// listSequence defines optional per-call list results.
	listSequence []*contactapplication.ListResult
	// listErrSequence defines optional per-call list errors.
	listErrSequence []error
	// listCalls defines list invocation counters.
	listCalls int
}

// Create creates contacts.
func (m *serviceMock) Create(ctx context.Context, command contactapplication.CreateCommand) (*contactdomain.Contact, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.created = append(m.created, command)
	return &contactdomain.Contact{ID: "created-id", Email: command.Email}, nil
}

// Get retrieves contacts by id.
func (m *serviceMock) Get(ctx context.Context, id string) (*contactdomain.Contact, error) {
	return nil, nil
}

// List retrieves contact pages.
func (m *serviceMock) List(ctx context.Context, query contactport.ListQuery) (*contactapplication.ListResult, error) {
	callIndex := m.listCalls
	m.listCalls++

	if callIndex < len(m.listErrSequence) && m.listErrSequence[callIndex] != nil {
		return nil, m.listErrSequence[callIndex]
	}
	if callIndex < len(m.listSequence) {
		if m.listSequence[callIndex] == nil {
			return &contactapplication.ListResult{}, nil
		}
		return m.listSequence[callIndex], nil
	}

	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.listResult == nil {
		return &contactapplication.ListResult{}, nil
	}

	return m.listResult, nil
}

// Update updates contacts.
func (m *serviceMock) Update(ctx context.Context, id string, command contactapplication.UpdateCommand) (*contactdomain.Contact, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	m.updateIDs = append(m.updateIDs, id)
	m.updates = append(m.updates, command)
	return &contactdomain.Contact{ID: id}, nil
}

// Delete deletes contacts.
func (m *serviceMock) Delete(ctx context.Context, id string) error {
	return nil
}

// TestNewUpserterValidation verifies constructor validation behavior.
func TestNewUpserterValidation(t *testing.T) {
	if _, err := NewUpserter(nil); !errorspkg.Is(err, ErrNilService) {
		t.Fatalf("NewUpserter(nil) error = %v, want ErrNilService", err)
	}
}

// TestUpsertByEmailCreate verifies create-path behavior.
func TestUpsertByEmailCreate(t *testing.T) {
	mock := &serviceMock{}
	upserter, err := NewUpserter(mock)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	outcome, upsertErr := upserter.UpsertByEmail(context.Background(), port.ContactSyncCommand{
		Email:          "new@example.com",
		FirstName:      "New",
		LastName:       "User",
		DocumentType:   "CC",
		DocumentNumber: "1234",
		CreatedAt:      timePointer(time.Date(2024, time.March, 10, 10, 0, 0, 0, time.UTC)),
		Metadata:       map[string]string{"integration.source": "woocommerce"},
	})
	if upsertErr != nil {
		t.Fatalf("UpsertByEmail() error = %v", upsertErr)
	}
	if outcome != port.UpsertOutcomeCreated {
		t.Fatalf("outcome = %q, want %q", outcome, port.UpsertOutcomeCreated)
	}
	if len(mock.created) != 1 {
		t.Fatalf("len(created) = %d, want %d", len(mock.created), 1)
	}
	if mock.created[0].DocumentType != contactdomain.DocumentTypeCC {
		t.Fatalf("created.DocumentType = %q, want %q", mock.created[0].DocumentType, contactdomain.DocumentTypeCC)
	}
	if mock.created[0].CreatedAt == nil || mock.created[0].CreatedAt.UTC().Format(time.RFC3339) != "2024-03-10T10:00:00Z" {
		t.Fatalf("created.CreatedAt = %v, want %q", mock.created[0].CreatedAt, "2024-03-10T10:00:00Z")
	}
	if mock.created[0].Metadata["integration.source"] != "woocommerce" {
		t.Fatalf("created.Metadata[integration.source] = %q, want %q", mock.created[0].Metadata["integration.source"], "woocommerce")
	}
}

// TestUpsertByEmailUpdate verifies update-path behavior.
func TestUpsertByEmailUpdate(t *testing.T) {
	mock := &serviceMock{
		listResult: &contactapplication.ListResult{
			Data: []contactdomain.Contact{{
				ID:        "contact-1",
				Email:     "existing@example.com",
				FirstName: "Old",
				LastName:  "User",
				CreatedAt: time.Date(2024, time.March, 12, 10, 0, 0, 0, time.UTC),
				Metadata:  map[string]string{"marketing.consent": "true"},
			}},
		},
	}
	upserter, err := NewUpserter(mock)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	outcome, upsertErr := upserter.UpsertByEmail(context.Background(), port.ContactSyncCommand{
		Email:        "existing@example.com",
		FirstName:    "New",
		LastName:     "User",
		Phone:        "123",
		Address:      "Street 1",
		AddressExtra: "Suite 1",
		CityCode:     "110111",
		CreatedAt:    timePointer(time.Date(2024, time.March, 10, 9, 0, 0, 0, time.UTC)),
		Metadata: map[string]string{
			"integration.source":                              "woocommerce",
			"integration.woocommerce.oldest_order_id":         "1001",
			"integration.woocommerce.oldest_order_created_at": "2024-03-10T09:00:00Z",
		},
	})
	if upsertErr != nil {
		t.Fatalf("UpsertByEmail() error = %v", upsertErr)
	}
	if outcome != port.UpsertOutcomeUpdated {
		t.Fatalf("outcome = %q, want %q", outcome, port.UpsertOutcomeUpdated)
	}
	if len(mock.updateIDs) != 1 || mock.updateIDs[0] != "contact-1" {
		t.Fatalf("updateIDs = %v, want [contact-1]", mock.updateIDs)
	}
	if len(mock.updates) != 1 {
		t.Fatalf("len(updates) = %d, want %d", len(mock.updates), 1)
	}
	if mock.updates[0].CreatedAt == nil || mock.updates[0].CreatedAt.UTC().Format(time.RFC3339) != "2024-03-10T09:00:00Z" {
		t.Fatalf("updates[0].CreatedAt = %v, want %q", mock.updates[0].CreatedAt, "2024-03-10T09:00:00Z")
	}
	if mock.updates[0].Metadata == nil || (*mock.updates[0].Metadata)["marketing.consent"] != "true" {
		t.Fatalf("updates[0].Metadata should preserve existing values")
	}
	if (*mock.updates[0].Metadata)["integration.woocommerce.oldest_order_id"] != "1001" {
		t.Fatalf("updates[0].Metadata[oldest_order_id] = %q, want %q", (*mock.updates[0].Metadata)["integration.woocommerce.oldest_order_id"], "1001")
	}
}

// TestUpsertByEmailUnchanged verifies unchanged-path behavior.
func TestUpsertByEmailUnchanged(t *testing.T) {
	mock := &serviceMock{
		listResult: &contactapplication.ListResult{
			Data: []contactdomain.Contact{{
				ID:           "contact-1",
				Email:        "existing@example.com",
				FirstName:    "Same",
				LastName:     "User",
				Phone:        "123",
				Address:      "Street 1",
				AddressExtra: "Suite 1",
				CityCode:     "110111",
			}},
		},
	}
	upserter, err := NewUpserter(mock)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	outcome, upsertErr := upserter.UpsertByEmail(context.Background(), port.ContactSyncCommand{
		Email:          "existing@example.com",
		FirstName:      "Same",
		LastName:       "User",
		Phone:          "123",
		Address:        "Street 1",
		AddressExtra:   "Suite 1",
		CityCode:       "110111",
		DocumentType:   "CC",
		DocumentNumber: "1234",
	})
	if upsertErr != nil {
		t.Fatalf("UpsertByEmail() error = %v", upsertErr)
	}
	if outcome != port.UpsertOutcomeUnchanged {
		t.Fatalf("outcome = %q, want %q", outcome, port.UpsertOutcomeUnchanged)
	}
	if len(mock.updateIDs) != 0 {
		t.Fatalf("expected no update for unchanged payload")
	}
}

// TestUpsertByEmailDuplicateCreate verifies duplicate create fallback behavior.
func TestUpsertByEmailDuplicateCreate(t *testing.T) {
	mock := &serviceMock{
		createErr: fmt.Errorf("create failed: %w", contactport.ErrDuplicateEmail),
		listSequence: []*contactapplication.ListResult{
			nil,
			{Data: []contactdomain.Contact{{ID: "contact-2", Email: "dup@example.com", FirstName: "Old", LastName: "Name"}}},
		},
	}
	upserter, err := NewUpserter(mock)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	outcome, upsertErr := upserter.UpsertByEmail(context.Background(), port.ContactSyncCommand{
		Email:     "dup@example.com",
		FirstName: "New",
		LastName:  "Name",
	})
	if upsertErr != nil {
		t.Fatalf("UpsertByEmail() error = %v", upsertErr)
	}
	if outcome != port.UpsertOutcomeUpdated {
		t.Fatalf("outcome = %q, want %q", outcome, port.UpsertOutcomeUpdated)
	}
	if len(mock.updateIDs) != 1 || mock.updateIDs[0] != "contact-2" {
		t.Fatalf("expected update fallback with contact-2")
	}
}

// TestFindByEmailError verifies list error handling behavior.
func TestFindByEmailError(t *testing.T) {
	mock := &serviceMock{listErr: errorspkg.New("list failed")}
	upserter, err := NewUpserter(mock)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	if _, findErr := upserter.findByEmail(context.Background(), "x@example.com"); findErr == nil {
		t.Fatalf("expected findByEmail() error")
	}
}

// TestPointer verifies pointer helper behavior.
func TestPointer(t *testing.T) {
	value := "demo"
	if pointer(value) == nil || *pointer(value) != value {
		t.Fatalf("expected pointer helper to preserve values")
	}
}

// TestUpsertByEmailCreateError verifies non-duplicate create error propagation.
func TestUpsertByEmailCreateError(t *testing.T) {
	mock := &serviceMock{
		createErr: errorspkg.New("create failed"),
	}
	upserter, err := NewUpserter(mock)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	if _, upsertErr := upserter.UpsertByEmail(context.Background(), port.ContactSyncCommand{
		Email: "create-error@example.com",
	}); upsertErr == nil {
		t.Fatalf("expected non-duplicate create error")
	}
}

// TestUpsertByEmailDuplicateCreateMissingLatest verifies duplicate fallback behavior when lookup misses rows.
func TestUpsertByEmailDuplicateCreateMissingLatest(t *testing.T) {
	mock := &serviceMock{
		createErr: fmt.Errorf("create failed: %w", contactport.ErrDuplicateContact),
		listSequence: []*contactapplication.ListResult{
			nil,
			{Data: []contactdomain.Contact{}},
		},
	}
	upserter, err := NewUpserter(mock)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	if _, upsertErr := upserter.UpsertByEmail(context.Background(), port.ContactSyncCommand{
		Email: "missing-latest@example.com",
	}); upsertErr == nil {
		t.Fatalf("expected duplicate-create fallback error when latest row is unavailable")
	}
}

// TestUpsertByEmailDuplicateCreateLookupError verifies duplicate fallback behavior when relookup fails.
func TestUpsertByEmailDuplicateCreateLookupError(t *testing.T) {
	mock := &serviceMock{
		createErr: fmt.Errorf("create failed: %w", contactport.ErrDuplicateEmail),
		listSequence: []*contactapplication.ListResult{
			nil,
		},
		listErrSequence: []error{
			nil,
			errorspkg.New("lookup failed"),
		},
	}
	upserter, err := NewUpserter(mock)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	if _, upsertErr := upserter.UpsertByEmail(context.Background(), port.ContactSyncCommand{
		Email: "lookup-error@example.com",
	}); upsertErr == nil {
		t.Fatalf("expected duplicate-create lookup error")
	}
}

// TestUpsertByEmailUpdateError verifies update-path error propagation.
func TestUpsertByEmailUpdateError(t *testing.T) {
	mock := &serviceMock{
		listResult: &contactapplication.ListResult{
			Data: []contactdomain.Contact{{ID: "contact-3", Email: "update-error@example.com", FirstName: "First", LastName: "Last"}},
		},
		updateErr: errorspkg.New("update failed"),
	}
	upserter, err := NewUpserter(mock)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	if _, upsertErr := upserter.UpsertByEmail(context.Background(), port.ContactSyncCommand{
		Email:     "update-error@example.com",
		FirstName: "Changed",
		LastName:  "Last",
	}); upsertErr == nil {
		t.Fatalf("expected update-path error")
	}
}

// TestPrivateHelpers verifies private helper behavior.
func TestPrivateHelpers(t *testing.T) {
	if !isDuplicateCreateError(contactport.ErrDuplicateDocument) {
		t.Fatalf("expected duplicate document to be retryable")
	}
	if isDuplicateCreateError(errorspkg.New("other")) {
		t.Fatalf("expected non-duplicate errors to be non-retryable")
	}

	documentType := documentTypePointer("CC")
	if documentType == nil || *documentType != contactdomain.DocumentTypeCC {
		t.Fatalf("documentTypePointer() should map CC values")
	}
	documentType = documentTypePointer("  ")
	if documentType == nil || *documentType != "" {
		t.Fatalf("documentTypePointer(empty) should map empty values")
	}

	if !hasMeaningfulChange(contactdomain.Contact{Email: "old@example.com"}, port.ContactSyncCommand{Email: "new@example.com"}, nil) {
		t.Fatalf("hasMeaningfulChange() should detect email changes")
	}
	if hasMeaningfulChange(
		contactdomain.Contact{Email: "same@example.com", FirstName: "A", LastName: "B", Phone: "1", Address: "a", AddressExtra: "b", CityCode: "c", DocumentType: contactdomain.DocumentTypeCC, DocumentNumber: "1"},
		port.ContactSyncCommand{Email: "same@example.com", FirstName: "A", LastName: "B", Phone: "1", Address: "a", AddressExtra: "b", CityCode: "c", DocumentType: "CE", DocumentNumber: "2"},
		nil,
	) {
		t.Fatalf("hasMeaningfulChange() should ignore document-only changes")
	}
	if !shouldUpdateCreatedAt(
		time.Date(2024, time.March, 12, 10, 0, 0, 0, time.UTC),
		timePointer(time.Date(2024, time.March, 10, 10, 0, 0, 0, time.UTC)),
	) {
		t.Fatalf("shouldUpdateCreatedAt() should update when candidate is older")
	}
	merged := mergeMetadata(map[string]string{"a": "1"}, map[string]string{"b": "2"})
	if len(merged) != 2 || merged["a"] != "1" || merged["b"] != "2" {
		t.Fatalf("mergeMetadata() should merge maps")
	}
	if !metadataEqual(map[string]string{"x": "1"}, map[string]string{"x": "1"}) {
		t.Fatalf("metadataEqual() should match equal maps")
	}
	if metadataEqual(map[string]string{"x": "1"}, map[string]string{"x": "2"}) {
		t.Fatalf("metadataEqual() should detect differences")
	}
	normalized := normalizeSyncMetadata(map[string]string{"  ": "x", " source ": " woo "})
	if len(normalized) != 1 || normalized["source"] != "woo" {
		t.Fatalf("normalizeSyncMetadata() = %#v, want source=woo", normalized)
	}
}

// timePointer returns time pointers for fixtures.
func timePointer(value time.Time) *time.Time {
	return &value
}
