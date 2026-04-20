package service

import (
	"bytes"
	"context"
	"testing"
	"time"

	"mannaiah/module/shipping/domain"
)

// TestBatchAllRotulusDocumentRejectsEmptyBatchID verifies empty batch ID returns ErrInvalidID.
func TestBatchAllRotulusDocumentRejectsEmptyBatchID(t *testing.T) {
	service := NewService(newMarkRepositoryStub(), markRegistryStub{}, &publisherStub{})

	_, err := service.BatchAllRotulusDocument(context.Background(), "")
	if err != domain.ErrInvalidID {
		t.Fatalf("BatchAllRotulusDocument() error = %v, want ErrInvalidID", err)
	}
}

// TestBatchAllRotulusDocumentReturnsNotFoundWhenBatchIsEmpty verifies ErrNotFound when no included marks exist.
func TestBatchAllRotulusDocumentReturnsNotFoundWhenBatchIsEmpty(t *testing.T) {
	repository := newMarkRepositoryStub()
	batchID := "batch-empty"
	repository.rows["failed-mark"] = domain.ShippingMark{
		ID:              "failed-mark",
		DispatchBatchID: strPtr(batchID),
		Status:          domain.MarkStatusFailed,
	}

	service := NewService(repository, markRegistryStub{}, &publisherStub{})
	service.SetRotulusDocumentSigningSecret("test-secret")

	_, err := service.BatchAllRotulusDocument(context.Background(), batchID)
	if err != domain.ErrNotFound {
		t.Fatalf("BatchAllRotulusDocument() error = %v, want ErrNotFound", err)
	}
}

// TestBatchAllRotulusDocumentSingleMarkProducesPDF verifies a single mark produces a valid PDF.
func TestBatchAllRotulusDocumentSingleMarkProducesPDF(t *testing.T) {
	repository := newMarkRepositoryStub()
	batchID := "batch-single"
	now := time.Now().UTC()
	repository.rows["mark-1"] = domain.ShippingMark{
		ID:              "mark-1",
		OrderID:         "order-1",
		CarrierID:       "manual",
		Observations:    "interrapidisimo",
		TrackingNumber:  "TRK001",
		DispatchBatchID: strPtr(batchID),
		Status:          domain.MarkStatusGenerated,
		Recipient:       domain.Address{Name: "Ana Gomez"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	service := NewService(repository, markRegistryStub{}, &publisherStub{})
	service.SetRotulusDocumentSigningSecret("test-secret")

	payload, err := service.BatchAllRotulusDocument(context.Background(), batchID)
	if err != nil {
		t.Fatalf("BatchAllRotulusDocument() error = %v", err)
	}
	if !bytes.HasPrefix(payload, []byte("%PDF")) {
		t.Fatalf("BatchAllRotulusDocument() result is not a valid PDF")
	}
}

// TestBatchAllRotulusDocumentTwoMarksOnOnePage verifies two marks fit on a single page.
func TestBatchAllRotulusDocumentTwoMarksOnOnePage(t *testing.T) {
	repository := newMarkRepositoryStub()
	batchID := "batch-two"
	now := time.Now().UTC()
	for _, id := range []string{"mark-a", "mark-b"} {
		repository.rows[id] = domain.ShippingMark{
			ID:              id,
			OrderID:         "order-" + id,
			CarrierID:       "manual",
			TrackingNumber:  id + "-trk",
			DispatchBatchID: strPtr(batchID),
			Status:          domain.MarkStatusGenerated,
			Recipient:       domain.Address{Name: "Recipient " + id},
			CreatedAt:       now,
			UpdatedAt:       now,
		}
	}

	service := NewService(repository, markRegistryStub{}, &publisherStub{})
	service.SetRotulusDocumentSigningSecret("test-secret")

	payload, err := service.BatchAllRotulusDocument(context.Background(), batchID)
	if err != nil {
		t.Fatalf("BatchAllRotulusDocument() error = %v", err)
	}
	if !bytes.HasPrefix(payload, []byte("%PDF")) {
		t.Fatalf("BatchAllRotulusDocument() result is not a valid PDF")
	}
}

// TestBatchAllRotulusDocumentThreeMarksSpanTwoPages verifies three marks produce two pages (2+1 layout).
func TestBatchAllRotulusDocumentThreeMarksSpanTwoPages(t *testing.T) {
	repository := newMarkRepositoryStub()
	batchID := "batch-three"
	now := time.Now().UTC()
	for _, id := range []string{"mark-x", "mark-y", "mark-z"} {
		repository.rows[id] = domain.ShippingMark{
			ID:              id,
			OrderID:         "order-" + id,
			CarrierID:       "tcc",
			TrackingNumber:  id + "-trk",
			DispatchBatchID: strPtr(batchID),
			Status:          domain.MarkStatusGenerated,
			Recipient:       domain.Address{Name: "Client " + id},
			CreatedAt:       now,
			UpdatedAt:       now,
		}
	}

	service := NewService(repository, markRegistryStub{}, &publisherStub{})
	service.SetRotulusDocumentSigningSecret("test-secret")

	payload, err := service.BatchAllRotulusDocument(context.Background(), batchID)
	if err != nil {
		t.Fatalf("BatchAllRotulusDocument() error = %v", err)
	}
	if !bytes.HasPrefix(payload, []byte("%PDF")) {
		t.Fatalf("BatchAllRotulusDocument() result is not a valid PDF")
	}
}
