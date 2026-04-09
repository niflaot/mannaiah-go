package service

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

type rotulusOrderSourceStub struct {
	row *port.OrderQuotationData
}

func (s rotulusOrderSourceStub) GetByIDOrIdentifier(ctx context.Context, identifier string) (*port.OrderQuotationData, error) {
	return s.row, nil
}

// TestRotulusDocumentBuildsPDFAndCaches verifies rotulus PDFs render and reuse cache for unchanged marks.
func TestRotulusDocumentBuildsPDFAndCaches(t *testing.T) {
	repository := newMarkRepositoryStub()
	now := time.Now().UTC()
	repository.rows["mark-1"] = domain.ShippingMark{
		ID:             "mark-1",
		OrderID:        "order-1",
		CarrierID:      "manual",
		Observations:   "interrapidisimo",
		TrackingNumber: "11515151",
		Recipient:      domain.Address{Name: "Ian Castano"},
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	service := NewService(repository, markRegistryStub{}, &publisherStub{})
	service.SetOrderSource(rotulusOrderSourceStub{row: &port.OrderQuotationData{OrderID: "order-1", OrderIdentifier: "1024751"}})
	service.SetRotulusDocumentSigningSecret("secret-123")

	firstPayload, err := service.RotulusDocument(context.Background(), "mark-1")
	if err != nil {
		t.Fatalf("RotulusDocument(first) error = %v", err)
	}
	if !bytes.HasPrefix(firstPayload, []byte("%PDF")) {
		t.Fatalf("RotulusDocument(first) returned non-pdf payload")
	}
	if len(service.rotulusDocuments.cache) != 1 {
		t.Fatalf("expected one cached rotulus payload, got %d", len(service.rotulusDocuments.cache))
	}

	secondPayload, err := service.RotulusDocument(context.Background(), "mark-1")
	if err != nil {
		t.Fatalf("RotulusDocument(second) error = %v", err)
	}
	if !bytes.Equal(firstPayload, secondPayload) {
		t.Fatalf("RotulusDocument(second) payload differs from cached payload")
	}
}

// TestBuildSignedRotulusQRToken verifies QR payload tokens include the signed version prefix.
func TestBuildSignedRotulusQRToken(t *testing.T) {
	service := NewService(newMarkRepositoryStub(), markRegistryStub{}, &publisherStub{})
	service.SetRotulusDocumentSigningSecret("secret-123")

	token, err := service.buildSignedRotulusQRToken(markRotulusMeta{
		MarkID:      "mark-1",
		OrderID:     "order-1",
		OrderNumber: "1024751",
		GeneratedAt: time.Unix(1712617200, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("buildSignedRotulusQRToken() error = %v", err)
	}
	if !strings.HasPrefix(token, "flkrotulus.v1.") {
		t.Fatalf("token prefix = %q", token)
	}
}
