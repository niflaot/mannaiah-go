package service

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mannaiah/module/shipping/domain"
)

// TestBatchAllMarksDocumentRejectsEmptyBatchID verifies empty batch ID returns ErrInvalidID.
func TestBatchAllMarksDocumentRejectsEmptyBatchID(t *testing.T) {
	service := NewService(newMarkRepositoryStub(), markRegistryStub{}, &publisherStub{})

	_, err := service.BatchAllMarksDocument(context.Background(), "")
	if err != domain.ErrInvalidID {
		t.Fatalf("BatchAllMarksDocument() error = %v, want ErrInvalidID", err)
	}
}

// TestBatchAllMarksDocumentReturnsNotFoundWhenNoDocumentRefs verifies that a batch with marks but no DocumentRef returns ErrNotFound.
func TestBatchAllMarksDocumentReturnsNotFoundWhenNoDocumentRefs(t *testing.T) {
	repository := newMarkRepositoryStub()
	batchID := "batch-1"
	repository.rows["mark-1"] = domain.ShippingMark{
		ID:              "mark-1",
		DispatchBatchID: strPtr(batchID),
		Status:          domain.MarkStatusGenerated,
		DocumentType:    domain.MarkDocumentType(""),
		DocumentRef:     "",
	}

	service := NewService(repository, markRegistryStub{}, &publisherStub{})

	_, err := service.BatchAllMarksDocument(context.Background(), batchID)
	if err != domain.ErrNotFound {
		t.Fatalf("BatchAllMarksDocument() error = %v, want ErrNotFound", err)
	}
}

// TestBatchAllMarksDocumentSkipsFailedMarks verifies that failed marks are excluded from the merged PDF.
func TestBatchAllMarksDocumentSkipsFailedMarks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(minimalPDFBytes())
	}))
	defer server.Close()

	repository := newMarkRepositoryStub()
	batchID := "batch-2"
	repository.rows["mark-failed"] = domain.ShippingMark{
		ID:              "mark-failed",
		DispatchBatchID: strPtr(batchID),
		Status:          domain.MarkStatusFailed,
		DocumentType:    domain.MarkDocumentLink,
		DocumentRef:     server.URL + "/mark.pdf",
	}

	service := NewService(repository, markRegistryStub{}, &publisherStub{})

	_, err := service.BatchAllMarksDocument(context.Background(), batchID)
	if err != domain.ErrNotFound {
		t.Fatalf("BatchAllMarksDocument() with only failed marks error = %v, want ErrNotFound", err)
	}
}

// TestBatchAllMarksDocumentDownloadsAndMerges verifies successful PDF download and merge.
func TestBatchAllMarksDocumentDownloadsAndMerges(t *testing.T) {
	pdfBytes := minimalPDFBytes()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer server.Close()

	repository := newMarkRepositoryStub()
	batchID := "batch-3"
	for i, id := range []string{"mark-a", "mark-b"} {
		_ = i
		repository.rows[id] = domain.ShippingMark{
			ID:              id,
			DispatchBatchID: strPtr(batchID),
			Status:          domain.MarkStatusGenerated,
			DocumentType:    domain.MarkDocumentLink,
			DocumentRef:     server.URL + "/" + id + ".pdf",
		}
	}

	service := NewService(repository, markRegistryStub{}, &publisherStub{})

	payload, err := service.BatchAllMarksDocument(context.Background(), batchID)
	if err != nil {
		t.Fatalf("BatchAllMarksDocument() error = %v", err)
	}
	if !bytes.HasPrefix(payload, []byte("%PDF")) {
		t.Fatalf("BatchAllMarksDocument() result is not a valid PDF")
	}
}

// TestMarkDocumentDownloadsAndStampsContent verifies single carrier-label downloads are stamped.
func TestMarkDocumentDownloadsAndStampsContent(t *testing.T) {
	pdfBytes := minimalPDFBytes()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(pdfBytes)
	}))
	defer server.Close()

	repository := newMarkRepositoryStub()
	repository.rows["mark-1"] = domain.ShippingMark{
		ID:           "mark-1",
		OrderID:      "order-1",
		Status:       domain.MarkStatusGenerated,
		DocumentType: domain.MarkDocumentLink,
		DocumentRef:  server.URL + "/mark.pdf",
		Units:        []domain.PackageUnit{{Description: "X1 Totepack Kairos Classic NEGRO"}},
	}
	service := NewService(repository, markRegistryStub{}, &publisherStub{})

	payload, err := service.MarkDocument(context.Background(), "mark-1")
	if err != nil {
		t.Fatalf("MarkDocument() error = %v", err)
	}
	if !bytes.HasPrefix(payload, []byte("%PDF")) {
		t.Fatalf("MarkDocument() returned non-pdf payload")
	}
}

// TestBatchAllMarksDocumentCachesResult verifies that a second call returns the cached payload.
func TestBatchAllMarksDocumentCachesResult(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(minimalPDFBytes())
	}))
	defer server.Close()

	repository := newMarkRepositoryStub()
	batchID := "batch-4"
	repository.rows["mark-c"] = domain.ShippingMark{
		ID:              "mark-c",
		DispatchBatchID: strPtr(batchID),
		Status:          domain.MarkStatusGenerated,
		DocumentType:    domain.MarkDocumentLink,
		DocumentRef:     server.URL + "/mark-c.pdf",
	}

	service := NewService(repository, markRegistryStub{}, &publisherStub{})
	service.SetBatchAllMarksDocumentCacheTTL(time.Minute)

	if _, err := service.BatchAllMarksDocument(context.Background(), batchID); err != nil {
		t.Fatalf("first call error = %v", err)
	}
	callsAfterFirst := calls

	if _, err := service.BatchAllMarksDocument(context.Background(), batchID); err != nil {
		t.Fatalf("second call error = %v", err)
	}
	if calls != callsAfterFirst {
		t.Fatalf("expected cache hit on second call, but HTTP server was called again")
	}
}

// strPtr is a test helper for string pointer values.
func strPtr(s string) *string { return &s }

// minimalPDFBytes returns a minimal valid PDF byte slice for testing.
func minimalPDFBytes() []byte {
	return []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] >>\nendobj\n" +
		"xref\n0 4\n0000000000 65535 f \n0000000009 00000 n \n0000000068 00000 n \n" +
		"0000000125 00000 n \ntrailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n210\n%%EOF\n")
}
