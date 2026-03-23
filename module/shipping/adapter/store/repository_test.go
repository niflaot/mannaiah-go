package store

import (
	"context"
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
	if migrateErr := db.AutoMigrate(&shippingMarkModel{}, &shippingMarkUnitModel{}, &dispatchBatchModel{}, &quotationModel{}); migrateErr != nil {
		t.Fatalf("automigrate: %v", migrateErr)
	}

	markRepository, batchRepository, quotationRepository, repositoryErr := NewRepositories(db)
	if repositoryErr != nil {
		t.Fatalf("NewRepositories() error = %v", repositoryErr)
	}

	mark := domain.ShippingMark{
		ID:                               "mark-1",
		OrderID:                          "order-1",
		CarrierID:                        "manual",
		Status:                           domain.MarkStatusGenerated,
		Sender:                           domain.Address{Name: "Sender", ID: "900", IDType: "NIT", AddressLine: "street", CityCode: "11001000"},
		Recipient:                        domain.Address{Name: "Recipient", ID: "800", IDType: "CC", AddressLine: "street", CityCode: "76001000"},
		Units:                            []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
		CollectOnDeliveryAmount:          100000,
		CollectOnDeliveryDiscountPercent: 4,
		CollectOnDeliveryChargedAmount:   104000,
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
	if loadedMark.CollectOnDeliveryAmount != 100000 || loadedMark.CollectOnDeliveryChargedAmount != 104000 {
		t.Fatalf("loaded COD values = %#v", loadedMark)
	}

	batch := domain.DispatchBatch{ID: "batch-1", Name: "Batch", CarrierID: "manual", Status: domain.BatchStatusOpen, CreatedAt: time.Now().UTC()}
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
		ID:                    "quote-1",
		OrderID:               "order-1",
		CarrierID:             "manual",
		OriginCityCode:        "11001000",
		DestCityCode:          "76001000",
		FullFreightCost:       10000,
		DiscountPercent:       10,
		DiscountedFreightCost: 9000,
		FreightCost:           9000,
		EstimatedDays:         2,
		CurrencyCode:          "COP",
		ExpiresAt:             time.Now().UTC().Add(time.Hour),
		RequestSnapshot:       "{}",
		CreatedAt:             time.Now().UTC(),
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
	if quotations[0].FullFreightCost != 10000 || quotations[0].DiscountedFreightCost != 9000 || quotations[0].DiscountPercent != 10 {
		t.Fatalf("unexpected quotation values = %#v", quotations[0])
	}
}
