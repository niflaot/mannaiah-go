package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"mannaiah/module/shipping/domain"
)

// TestChecklistDocumentBuildsPDFForOpenBatch verifies checklist-document generation for open manual/tcc batches.
func TestChecklistDocumentBuildsPDFForOpenBatch(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	service := NewService(batchRepository, markRepository, nil)

	batchID := "batch-open"
	batchRepository.batches[batchID] = domain.DispatchBatch{
		ID:        batchID,
		CarrierID: "tcc",
		Status:    domain.BatchStatusOpen,
		CreatedBy: "user-1",
		CreatedAt: time.Now().UTC(),
	}
	markRepository.marks["mark-1"] = domain.ShippingMark{
		ID:              "mark-1",
		OrderID:         "1025080",
		CarrierID:       "tcc",
		Status:          domain.MarkStatusCreated,
		Recipient:       domain.Address{Name: "Laura Camila Segura Sandoval", CityCode: "25175"},
		Units:           []domain.PackageUnit{{Description: "MORRAL EXPLORER WAVE VINO"}, {Description: "NECESER CARRY ESSENTIAL GRIS"}},
		DispatchBatchID: &batchID,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	payload, err := service.ChecklistDocument(context.Background(), batchID)
	if err != nil {
		t.Fatalf("ChecklistDocument() error = %v, want nil", err)
	}
	if len(payload) == 0 || !strings.HasPrefix(string(payload), "%PDF") {
		t.Fatalf("ChecklistDocument() returned non-pdf payload")
	}
}

// TestChecklistDocumentRejectsClosedBatch verifies checklist-document generation rejects non-open batches.
func TestChecklistDocumentRejectsClosedBatch(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	service := NewService(batchRepository, markRepository, nil)

	batchRepository.batches["batch-closed"] = domain.DispatchBatch{
		ID:        "batch-closed",
		CarrierID: "tcc",
		Status:    domain.BatchStatusClosed,
		CreatedBy: "user-1",
		CreatedAt: time.Now().UTC(),
	}

	_, err := service.ChecklistDocument(context.Background(), "batch-closed")
	if !errors.Is(err, domain.ErrInvalidBatchStatus) {
		t.Fatalf("ChecklistDocument() error = %v, want ErrInvalidBatchStatus", err)
	}
}

// TestResolveBatchChecklistRowsResolvesCityNames verifies checklist rows map city codes to city names.
func TestResolveBatchChecklistRowsResolvesCityNames(t *testing.T) {
	service := NewService(newDispatchBatchRepositoryStub(), newDispatchMarkRepositoryStub(), nil)

	rows := service.resolveBatchChecklistRows(context.Background(), []domain.ShippingMark{
		{
			ID:        "mark-1",
			OrderID:   "1025080",
			Status:    domain.MarkStatusCreated,
			Recipient: domain.Address{Name: "Laura Camila", CityCode: "25175"},
			Units:     []domain.PackageUnit{{Description: "MORRAL"}},
		},
		{
			ID:        "mark-2",
			OrderID:   "1025081",
			Status:    domain.MarkStatusRemoved,
			Recipient: domain.Address{Name: "Removed", CityCode: "11001"},
			Units:     []domain.PackageUnit{{Description: "MORRAL"}},
		},
	})

	if len(rows) != 1 {
		t.Fatalf("rows len = %d, want 1", len(rows))
	}
	if strings.TrimSpace(rows[0].City) == "" || rows[0].City == "25175" {
		t.Fatalf("rows[0].City = %q, expected resolved city name", rows[0].City)
	}
}
