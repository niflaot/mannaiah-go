package store

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	coredatabase "mannaiah/module/core/database"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// TestRepositories verifies store repository CRUD behavior.
func TestRepositories(t *testing.T) {
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if migrateErr := db.AutoMigrate(&shippingMarkModel{}, &shippingMarkUnitModel{}, &dispatchBatchModel{}, &quotationModel{}, &quotationUnitModel{}); migrateErr != nil {
		t.Fatalf("automigrate: %v", migrateErr)
	}

	markRepository, batchRepository, quotationRepository, repositoryErr := NewRepositories(db)
	if repositoryErr != nil {
		t.Fatalf("NewRepositories() error = %v", repositoryErr)
	}

	mark := domain.ShippingMark{
		ID:                             "mark-1",
		OrderID:                        "order-1",
		CarrierID:                      "manual",
		Status:                         domain.MarkStatusGenerated,
		DocumentType:                   domain.MarkDocumentLink,
		DocumentRef:                    "https://carrier/labels/mark-1",
		ManifestType:                   domain.MarkDocumentLink,
		ManifestRef:                    "https://carrier/manifest/batch-1",
		Sender:                         domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"},
		Recipient:                      domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001000"},
		Units:                          []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		CollectOnDeliveryAmount:        100000,
		CollectOnDeliveryFeePercent:    4,
		CollectOnDeliveryChargedAmount: 104000,
	}
	if err := markRepository.Create(context.Background(), &mark); err != nil {
		t.Fatalf("Create(mark) error = %v", err)
	}
	loadedMark, err := markRepository.GetByID(context.Background(), mark.ID)
	if err != nil {
		t.Fatalf("GetByID(mark) error = %v", err)
	}
	if loadedMark.ID != mark.ID {
		t.Fatalf("loaded mark id = %q", loadedMark.ID)
	}
	if loadedMark.CollectOnDeliveryAmount != 100000 || loadedMark.CollectOnDeliveryChargedAmount != 104000 || loadedMark.CollectOnDeliveryFeePercent != 4 {
		t.Fatalf("loaded COD values = %#v", loadedMark)
	}
	if loadedMark.DocumentRef != "https://carrier/labels/mark-1" {
		t.Fatalf("loadedMark.DocumentRef = %q, want mark document URL", loadedMark.DocumentRef)
	}
	if loadedMark.ManifestRef != "https://carrier/manifest/batch-1" {
		t.Fatalf("loadedMark.ManifestRef = %q, want manifest URL", loadedMark.ManifestRef)
	}

	batch := domain.DispatchBatch{ID: "batch-1", CarrierID: "manual", Status: domain.BatchStatusOpen, CreatedBy: "user-123", CreatedAt: time.Now().UTC()}
	if err := batchRepository.Create(context.Background(), &batch); err != nil {
		t.Fatalf("Create(batch) error = %v", err)
	}
	if err := batchRepository.AddMark(context.Background(), batch.ID, mark.ID); err != nil {
		t.Fatalf("AddMark() error = %v", err)
	}
	loadedBatch, err := batchRepository.GetByID(context.Background(), batch.ID)
	if err != nil {
		t.Fatalf("GetByID(batch) error = %v", err)
	}
	if len(loadedBatch.MarkIDs) != 1 {
		t.Fatalf("batch mark count = %d", len(loadedBatch.MarkIDs))
	}
	if err := batchRepository.Close(context.Background(), batch.ID); err != nil {
		t.Fatalf("Close(batch) error = %v", err)
	}

	if err := quotationRepository.Create(context.Background(), port.QuotationRecord{
		ID:              "quote-1",
		OrderID:         "order-1",
		CarrierID:       "manual",
		OriginCityCode:  "11001000",
		DestCityCode:    "76001000",
		FreightCost:     9000,
		EstimatedDays:   2,
		CurrencyCode:    "COP",
		ExpiresAt:       time.Now().UTC().Add(time.Hour),
		Units:           []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2, VolumetricWeightKG: 2.4, DeclaredValueCOP: 10000}}},
		RequestSnapshot: base64.StdEncoding.EncodeToString([]byte(`{"units":[{"description":"box"}]}`)),
		RawResponse:     base64.StdEncoding.EncodeToString([]byte(`{"ok":true}`)),
		CreatedAt:       time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Create(quotation) error = %v", err)
	}
	quotations, err := quotationRepository.ListByOrderID(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("ListByOrderID(quotation) error = %v", err)
	}
	if len(quotations) != 1 {
		t.Fatalf("quotation count = %d", len(quotations))
	}
	if quotations[0].FreightCost != 9000 {
		t.Fatalf("unexpected quotation values = %#v", quotations[0])
	}
	if len(quotations[0].Units) != 1 {
		t.Fatalf("quotation units = %d, want 1", len(quotations[0].Units))
	}
	if quotations[0].Units[0].Description != "box" {
		t.Fatalf("quotation unit description = %q, want box", quotations[0].Units[0].Description)
	}
	loadedQuotation, err := quotationRepository.GetByID(context.Background(), "quote-1")
	if err != nil {
		t.Fatalf("GetByID(quotation) error = %v", err)
	}
	if loadedQuotation == nil || loadedQuotation.ID != "quote-1" {
		t.Fatalf("loaded quotation = %#v", loadedQuotation)
	}
}

// TestQuotationRepositoryPreventsDuplicateActiveRows verifies duplicate quotation inserts are ignored while non-expired.
func TestQuotationRepositoryPreventsDuplicateActiveRows(t *testing.T) {
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if migrateErr := db.AutoMigrate(&quotationModel{}, &quotationUnitModel{}); migrateErr != nil {
		t.Fatalf("automigrate: %v", migrateErr)
	}
	quotationRepository, repoErr := NewQuotationRepository(db)
	if repoErr != nil {
		t.Fatalf("NewQuotationRepository() error = %v", repoErr)
	}

	record := port.QuotationRecord{
		ID:              "quote-dedup-1",
		OrderID:         "order-dedup",
		CarrierID:       "manual",
		OriginCityCode:  "11001000",
		DestCityCode:    "76001000",
		FreightCost:     9000,
		EstimatedDays:   2,
		CurrencyCode:    "COP",
		ExpiresAt:       time.Now().UTC().Add(time.Hour),
		RequestSnapshot: base64.StdEncoding.EncodeToString([]byte(`{"orderId":"order-dedup","units":[{"description":"box"}]}`)),
		RawResponse:     base64.StdEncoding.EncodeToString([]byte(`{"ok":true}`)),
		Units:           []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		CreatedAt:       time.Now().UTC(),
	}
	if err := quotationRepository.Create(context.Background(), record); err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	record.ID = "quote-dedup-2"
	if err := quotationRepository.Create(context.Background(), record); err != nil {
		t.Fatalf("Create(duplicate) error = %v", err)
	}

	rows, err := quotationRepository.ListByOrderID(context.Background(), "order-dedup")
	if err != nil {
		t.Fatalf("ListByOrderID() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("quotation rows = %d, want 1", len(rows))
	}
}
