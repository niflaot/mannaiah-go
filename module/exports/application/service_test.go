package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"mannaiah/module/exports/domain"
	"mannaiah/module/exports/port"
)

// TestGenerateContactsUploadsCSVAndStoresRegistry verifies contact exports create storage objects and registry rows.
func TestGenerateContactsUploadsCSVAndStoresRegistry(t *testing.T) {
	repository := &fakeRepository{}
	storage := &fakeStorage{}
	contacts := &fakeContactSource{rows: []port.ContactRow{{
		ID: "contact-1", LegalName: "Ian Castano", Email: "ian@example.com", Phone: "123",
		Address: "Street 1", AddressExtra: "Apt 2", CityCode: "BOG", Metadata: map[string]string{"source": "test"},
		CreatedAt: fixedTime(), UpdatedAt: fixedTime(),
	}}}
	service, err := NewService(repository, storage, contacts, &fakeOrderSource{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetClock(fixedTime)

	report, err := service.GenerateContacts(context.Background())
	if err != nil {
		t.Fatalf("GenerateContacts() error = %v", err)
	}

	if report.Type != domain.ReportTypeContacts {
		t.Fatalf("report type = %q", report.Type)
	}
	if !strings.HasPrefix(storage.request.Key, "exports/contacts/20260506T120000Z-") {
		t.Fatalf("storage key = %q", storage.request.Key)
	}
	if !strings.Contains(string(storage.request.Body), "Ian Castano") {
		t.Fatalf("csv body missing contact: %s", string(storage.request.Body))
	}
	sum := sha256.Sum256(storage.request.Body)
	if report.SHA256 != hex.EncodeToString(sum[:]) {
		t.Fatalf("report sha = %q", report.SHA256)
	}
	if repository.created == nil || repository.created.RowCount != 1 {
		t.Fatalf("registry row was not created: %#v", repository.created)
	}
}

// TestGenerateOrdersIncludesContactAndItems verifies order exports include customer and item details.
func TestGenerateOrdersIncludesContactAndItems(t *testing.T) {
	repository := &fakeRepository{}
	storage := &fakeStorage{}
	orders := &fakeOrderSource{rows: []port.OrderRow{{
		ID: "order-1", Identifier: "1001", ContactEmail: "buyer@example.com", Address: "Street 1",
		Address2: "Apt 2", Phone: "555", CityName: "Bogota", CityCode: "BOG", Status: "completed",
		Items:     []port.OrderItemRow{{SKU: "SKU-1", AlternateName: "Product", Quantity: 2, Value: 12.5}},
		CreatedAt: fixedTime(), UpdatedAt: fixedTime(),
	}}}
	service, err := NewService(repository, storage, &fakeContactSource{}, orders)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetClock(fixedTime)

	report, err := service.GenerateOrders(context.Background())
	if err != nil {
		t.Fatalf("GenerateOrders() error = %v", err)
	}

	body := string(storage.request.Body)
	for _, expected := range []string{"buyer@example.com", "Bogota", "SKU-1", "quantity=2"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("csv body missing %q: %s", expected, body)
		}
	}
	if report.Type != domain.ReportTypeOrders || report.RowCount != 1 {
		t.Fatalf("report = %#v", report)
	}
}

// TestSearchReportsRejectsInvalidType verifies invalid report types are controlled errors.
func TestSearchReportsRejectsInvalidType(t *testing.T) {
	service, err := NewService(&fakeRepository{}, &fakeStorage{}, &fakeContactSource{}, &fakeOrderSource{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.SearchReports(context.Background(), port.ListQuery{Type: "products"})
	if err != domain.ErrInvalidReportType {
		t.Fatalf("SearchReports() error = %v", err)
	}
}

func fixedTime() time.Time {
	return time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
}

type fakeRepository struct {
	created *domain.Report
	rows    []domain.Report
}

// Create persists a generated export report.
func (r *fakeRepository) Create(_ context.Context, report *domain.Report) error {
	copy := *report
	r.created = &copy
	return nil
}

// GetByID retrieves a generated export report by id.
func (r *fakeRepository) GetByID(_ context.Context, id string) (*domain.Report, error) {
	for _, row := range r.rows {
		if row.ID == id {
			copy := row
			return &copy, nil
		}
	}
	return nil, domain.ErrReportNotFound
}

// List returns paginated generated export reports.
func (r *fakeRepository) List(_ context.Context, _ port.ListQuery) ([]domain.Report, int64, error) {
	return r.rows, int64(len(r.rows)), nil
}

type fakeStorage struct {
	request port.UploadRequest
}

// Upload writes report object bytes to storage.
func (s *fakeStorage) Upload(_ context.Context, request port.UploadRequest) error {
	s.request = request
	return nil
}

// AvailabilityError reports storage availability failures.
func (s *fakeStorage) AvailabilityError() error {
	return nil
}

type fakeContactSource struct {
	rows []port.ContactRow
}

// ListContacts returns all contacts to export.
func (s *fakeContactSource) ListContacts(context.Context) ([]port.ContactRow, error) {
	return s.rows, nil
}

type fakeOrderSource struct {
	rows []port.OrderRow
}

// ListOrders returns all orders to export.
func (s *fakeOrderSource) ListOrders(context.Context) ([]port.OrderRow, error) {
	return s.rows, nil
}
