package e2e_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"mannaiah/module/exports"
	exportsport "mannaiah/module/exports/port"
)

// TestExportsGenerationE2E verifies contact and order CSV exports through HTTP and storage wiring.
func TestExportsGenerationE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("initialize exports module")
	exportStorage := newInMemoryExportStorage()
	consentSource := &staticExportConsentSource{statuses: map[string][]exportsport.ContactConsentStatus{}}
	exportsModule, err := exports.New(harness.db, exportStorage, harness.contactsModule.Service(), harness.ordersModule.Service(), consentSource)
	if err != nil {
		t.Fatalf("exports.New() error = %v", err)
	}
	exportsModule.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(exportsModule.RegisterRoutes)

	manageToken := harness.SignToken(t, "contact:manage order:manage product:manage marketing:manage")

	harness.tracer.Step("create export contact")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/contacts", manageToken, []byte(`{"email":"exports@example.com","legalName":"Exports Buyer","address":"Street 1","addressExtra":"Apt 2","phone":"555","cityCode":"BOG","metadata":{"flock_checker_privacy_accept":"yes","flock_checker_privacy_accept_accepted_at_utc":"2026-05-06T14:30:00Z"}}`))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	contactID, _ := payload["id"].(string)
	if contactID == "" {
		t.Fatalf("expected contact id")
	}
	consentSource.statuses[contactID] = []exportsport.ContactConsentStatus{{
		Channel:    "all",
		Action:     "opt_in",
		OccurredAt: fixedE2EExportConsentTime(),
	}}

	harness.tracer.Step("create export order")
	orderPayload := `{"identifier":"EXP-1001","realm":"manual","contactId":"` + contactID + `","items":[{"sku":"SKU-EXP","alternateName":"Export Product","quantity":2,"value":19.5}],"shippingAddress":{"address":"Ship Street","address2":"Suite 4","phone":"555","cityCode":"BOG"},"author":"e2e","description":"exports"}`
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/orders", manageToken, []byte(orderPayload))
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}

	harness.tracer.Step("generate contacts export")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/exports/contacts", manageToken, nil)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	contactsKey, _ := payload["storageKey"].(string)
	if !strings.HasPrefix(contactsKey, "exports/contacts/") {
		t.Fatalf("contacts storageKey = %q", contactsKey)
	}
	contactsCSV := string(exportStorage.mustGet(t, contactsKey))
	for _, expected := range []string{"exports@example.com", "membershipOptIn", "true,2026-05-06T13:00:00Z,true,2026-05-06T14:30:00Z"} {
		if !strings.Contains(contactsCSV, expected) {
			t.Fatalf("contacts csv missing %q: %s", expected, contactsCSV)
		}
	}

	harness.tracer.Step("generate orders export through compatibility alias")
	status, payload = harness.DoJSONRequest(t, http.MethodPost, "/export/orders", manageToken, nil)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want %d", status, http.StatusCreated)
	}
	ordersKey, _ := payload["storageKey"].(string)
	if !strings.HasPrefix(ordersKey, "exports/orders/") {
		t.Fatalf("orders storageKey = %q", ordersKey)
	}
	ordersCSV := string(exportStorage.mustGet(t, ordersKey))
	for _, expected := range []string{"exports@example.com", "Ship Street", "BOG", "SKU-EXP", "quantity=2"} {
		if !strings.Contains(ordersCSV, expected) {
			t.Fatalf("orders csv missing %q: %s", expected, ordersCSV)
		}
	}

	harness.tracer.Step("search export report registry")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/exports/search?type=orders&page=1&limit=10", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["total"] != float64(1) {
		t.Fatalf("payload.total = %v, want %v", payload["total"], float64(1))
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(7)
}

// staticExportConsentSource defines deterministic membership statuses for export E2E flows.
type staticExportConsentSource struct {
	// statuses stores contact statuses by contact id.
	statuses map[string][]exportsport.ContactConsentStatus
}

// GetContactStatuses returns static contact consent statuses.
func (s *staticExportConsentSource) GetContactStatuses(_ context.Context, contactID string) ([]exportsport.ContactConsentStatus, error) {
	if s.statuses == nil {
		s.statuses = map[string][]exportsport.ContactConsentStatus{}
	}
	return s.statuses[contactID], nil
}

// fixedE2EExportConsentTime returns deterministic consent timestamps.
func fixedE2EExportConsentTime() time.Time {
	return time.Date(2026, 5, 6, 13, 0, 0, 0, time.UTC)
}

// inMemoryExportStorage defines E2E in-memory storage behavior for exports.
type inMemoryExportStorage struct {
	// mu protects concurrent object operations.
	mu sync.RWMutex
	// objects stores keyed object payload values.
	objects map[string][]byte
}

// newInMemoryExportStorage creates in-memory export storage.
func newInMemoryExportStorage() *inMemoryExportStorage {
	return &inMemoryExportStorage{objects: map[string][]byte{}}
}

// Upload stores payload bytes by key.
func (s *inMemoryExportStorage) Upload(_ context.Context, request exportsport.UploadRequest) error {
	if s == nil {
		return errors.New("export storage is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	copied := make([]byte, len(request.Body))
	copy(copied, request.Body)
	s.objects[request.Key] = copied

	return nil
}

// Download loads payload bytes by key.
func (s *inMemoryExportStorage) Download(_ context.Context, key string) ([]byte, error) {
	if s == nil {
		return nil, errors.New("export storage is nil")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	payload, exists := s.objects[key]
	if !exists {
		return nil, errors.New("export object does not exist")
	}
	copied := make([]byte, len(payload))
	copy(copied, payload)
	return copied, nil
}

// Delete removes payloads by key.
func (s *inMemoryExportStorage) Delete(_ context.Context, key string) error {
	if s == nil {
		return errors.New("export storage is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}

// Exists verifies whether payloads exist by key.
func (s *inMemoryExportStorage) Exists(_ context.Context, key string) (bool, error) {
	if s == nil {
		return false, errors.New("export storage is nil")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.objects[key]
	return exists, nil
}

// AvailabilityError reports storage availability behavior.
func (s *inMemoryExportStorage) AvailabilityError() error {
	return nil
}

// mustGet loads stored object bytes or fails the test.
func (s *inMemoryExportStorage) mustGet(t *testing.T, key string) []byte {
	t.Helper()

	payload, err := s.Download(context.Background(), key)
	if err != nil {
		t.Fatalf("Download(%q) error = %v", key, err)
	}
	return payload
}
