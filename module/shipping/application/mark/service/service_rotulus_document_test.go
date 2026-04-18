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
		ID:                      "mark-1",
		OrderID:                 "order-1",
		CarrierID:               "manual",
		Observations:            "interrapidisimo",
		TrackingNumber:          "11515151",
		CollectOnDeliveryAmount: 150000,
		Recipient:               domain.Address{Name: "Ian Castano"},
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	service := NewService(repository, markRegistryStub{}, &publisherStub{})
	service.SetOrderSource(rotulusOrderSourceStub{row: &port.OrderQuotationData{
		OrderID:               "order-1",
		OrderIdentifier:       "1024751",
		RecipientAddressLine:  "Calle 18 Sur # 24d - 46",
		RecipientAddressLine2: "Piso 2",
		RecipientPhone:        "3057901484",
		RecipientCity:         "76364",
	}})
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
	if !strings.Contains(string(firstPayload), "PEDIDO #1024751") {
		t.Fatalf("RotulusDocument(first) missing dynamic order title")
	}
	if strings.Contains(string(firstPayload), "11515151") {
		t.Fatalf("RotulusDocument(first) should not include tracking number text")
	}
	if !strings.Contains(string(firstPayload), "Calle 18 Sur # 24d - 46") {
		t.Fatalf("RotulusDocument(first) missing shipping address")
	}
	if !strings.Contains(string(firstPayload), "Piso 2") {
		t.Fatalf("RotulusDocument(first) missing shipping address 2")
	}
	if !strings.Contains(string(firstPayload), "3057901484") {
		t.Fatalf("RotulusDocument(first) missing shipping phone")
	}
	if !strings.Contains(string(firstPayload), "Jamund") {
		t.Fatalf("RotulusDocument(first) missing shipping city name")
	}
	if !strings.Contains(string(firstPayload), "TRANSPORTADORA:") {
		t.Fatalf("RotulusDocument(first) missing uppercase carrier label")
	}
	if !strings.Contains(string(firstPayload), "DESTINATARIO:") {
		t.Fatalf("RotulusDocument(first) missing uppercase recipient label")
	}
	if !strings.Contains(string(firstPayload), "RECAUDO: $150.000") {
		t.Fatalf("RotulusDocument(first) missing highlighted recaudo amount")
	}
	if strings.Contains(string(firstPayload), "despacho") {
		t.Fatalf("RotulusDocument(first) includes deprecated title")
	}
	if strings.Contains(string(firstPayload), "Generado: ") {
		t.Fatalf("RotulusDocument(first) includes deprecated generated label")
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

// TestResolveRotulusCityDisplayName verifies city-code values resolve to municipality names for rotulus labels.
func TestResolveRotulusCityDisplayName(t *testing.T) {
	if got := resolveRotulusCityDisplayName("76364"); got != "Jamundí" {
		t.Fatalf("resolveRotulusCityDisplayName(76364) = %q, want %q", got, "Jamundí")
	}
	if got := resolveRotulusCityDisplayName("76001000"); got != "Cali" {
		t.Fatalf("resolveRotulusCityDisplayName(76001000) = %q, want %q", got, "Cali")
	}
	if got := resolveRotulusCityDisplayName("Bogotá"); got != "Bogotá" {
		t.Fatalf("resolveRotulusCityDisplayName(Bogotá) = %q, want %q", got, "Bogotá")
	}
}
