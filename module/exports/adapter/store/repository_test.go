package store

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"mannaiah/module/exports/domain"
	"mannaiah/module/exports/port"
)

// TestRepositoryCreateGetAndList verifies registry persistence and type filtering.
func TestRepositoryCreateGetAndList(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&reportModel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)

	contactsReport := newReport("contacts-1", domain.ReportTypeContacts, now)
	ordersReport := newReport("orders-1", domain.ReportTypeOrders, now.Add(time.Minute))
	if err := repository.Create(context.Background(), contactsReport); err != nil {
		t.Fatalf("Create(contacts) error = %v", err)
	}
	if err := repository.Create(context.Background(), ordersReport); err != nil {
		t.Fatalf("Create(orders) error = %v", err)
	}

	got, err := repository.GetByID(context.Background(), "orders-1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Type != domain.ReportTypeOrders || got.StorageKey != ordersReport.StorageKey {
		t.Fatalf("GetByID() = %#v", got)
	}

	rows, total, err := repository.List(context.Background(), port.ListQuery{Type: domain.ReportTypeContacts, Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != "contacts-1" {
		t.Fatalf("List() rows=%#v total=%d", rows, total)
	}
}

func newReport(id string, reportType domain.ReportType, generatedAt time.Time) *domain.Report {
	return &domain.Report{
		ID:          id,
		Type:        reportType,
		Status:      domain.ReportStatusCompleted,
		Stamp:       generatedAt.Format("20060102T150405Z"),
		FileName:    string(reportType) + ".csv",
		StorageKey:  "exports/" + string(reportType) + "/" + id + ".csv",
		SHA256:      "hash-" + id,
		ContentType: "text/csv",
		RowCount:    1,
		ByteSize:    10,
		GeneratedAt: generatedAt,
		CreatedAt:   generatedAt,
		UpdatedAt:   generatedAt,
	}
}
