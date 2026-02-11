package contacts

import (
	"context"
	errorspkg "errors"
	"fmt"
	"testing"

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
}

// TestUpsertByEmailUpdate verifies update-path behavior.
func TestUpsertByEmailUpdate(t *testing.T) {
	mock := &serviceMock{
		listResult: &contactapplication.ListResult{
			Data: []contactdomain.Contact{{ID: "contact-1", Email: "existing@example.com", FirstName: "Old", LastName: "User"}},
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

	if !hasMeaningfulChange(contactdomain.Contact{Email: "old@example.com"}, port.ContactSyncCommand{Email: "new@example.com"}) {
		t.Fatalf("hasMeaningfulChange() should detect email changes")
	}
	if hasMeaningfulChange(
		contactdomain.Contact{Email: "same@example.com", FirstName: "A", LastName: "B", Phone: "1", Address: "a", AddressExtra: "b", CityCode: "c", DocumentType: contactdomain.DocumentTypeCC, DocumentNumber: "1"},
		port.ContactSyncCommand{Email: "same@example.com", FirstName: "A", LastName: "B", Phone: "1", Address: "a", AddressExtra: "b", CityCode: "c", DocumentType: "CE", DocumentNumber: "2"},
	) {
		t.Fatalf("hasMeaningfulChange() should ignore document-only changes")
	}
}
