package http

import (
	"context"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/exports/application"
	"mannaiah/module/exports/domain"
	"mannaiah/module/exports/port"
)

// TestHandlerGenerateContacts verifies contact export routes return generated report metadata.
func TestHandlerGenerateContacts(t *testing.T) {
	service := &fakeService{report: sampleReport(domain.ReportTypeContacts)}
	handler, err := NewHandler(service)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	server := newHTTPServerForHandler(t, handler)
	response, err := server.App().Test(httptest.NewRequest(stdhttp.MethodPost, "/exports/contacts", nil))
	if err != nil {
		t.Fatalf("request error = %v", err)
	}

	if response.StatusCode != 201 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if !service.contactsGenerated {
		t.Fatal("contacts export was not generated")
	}
}

// TestHandlerSearchRejectsInvalidPagination verifies query validation is controlled.
func TestHandlerSearchRejectsInvalidPagination(t *testing.T) {
	handler, err := NewHandler(&fakeService{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	server := newHTTPServerForHandler(t, handler)
	response, err := server.App().Test(httptest.NewRequest(stdhttp.MethodGet, "/exports/search?page=0", nil))
	if err != nil {
		t.Fatalf("request error = %v", err)
	}

	if response.StatusCode != 400 {
		t.Fatalf("status = %d", response.StatusCode)
	}
}

type fakeService struct {
	report            *domain.Report
	contactsGenerated bool
	ordersGenerated   bool
}

// GenerateContacts creates a contact CSV report.
func (s *fakeService) GenerateContacts(context.Context) (*domain.Report, error) {
	s.contactsGenerated = true
	return s.report, nil
}

// GenerateOrders creates an order CSV report.
func (s *fakeService) GenerateOrders(context.Context) (*domain.Report, error) {
	s.ordersGenerated = true
	return s.report, nil
}

// GetReport retrieves one report by id.
func (s *fakeService) GetReport(_ context.Context, id string) (*domain.Report, error) {
	if id == "" {
		return nil, domain.ErrInvalidReportID
	}
	return s.report, nil
}

// ListReports returns paginated reports.
func (s *fakeService) ListReports(context.Context, port.ListQuery) (*application.ListResult, error) {
	if s.report == nil {
		return nil, errors.New("missing report")
	}
	return &application.ListResult{Data: []domain.Report{*s.report}, Page: 1, Limit: 50, Total: 1, TotalPages: 1}, nil
}

// SearchReports returns paginated reports using filter criteria.
func (s *fakeService) SearchReports(context.Context, port.ListQuery) (*application.ListResult, error) {
	return s.ListReports(context.Background(), port.ListQuery{})
}

func sampleReport(reportType domain.ReportType) *domain.Report {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	return &domain.Report{
		ID:          "report-1",
		Type:        reportType,
		Status:      domain.ReportStatusCompleted,
		Stamp:       "20260506T120000Z",
		FileName:    string(reportType) + ".csv",
		StorageKey:  "exports/" + string(reportType) + "/report.csv",
		SHA256:      "hash",
		ContentType: "text/csv",
		RowCount:    1,
		ByteSize:    10,
		GeneratedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func newHTTPServerForHandler(t *testing.T, handler *Handler) *corehttp.Server {
	t.Helper()

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8100}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(handler.RegisterRoutes)

	return server
}
