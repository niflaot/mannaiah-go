package service

import (
	"context"
	"testing"
	"time"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

type quotationRepositoryStub struct {
	rows []port.QuotationRecord
}

func (s *quotationRepositoryStub) Create(ctx context.Context, record port.QuotationRecord) error {
	s.rows = append(s.rows, record)

	return nil
}

func (s *quotationRepositoryStub) ListByOrderID(ctx context.Context, orderID string) ([]port.QuotationRecord, error) {
	return s.rows, nil
}

type quotationProviderStub struct{}

func (quotationProviderStub) CarrierID() string { return "manual" }
func (quotationProviderStub) Carrier() domain.Carrier {
	return domain.Carrier{ID: "manual", Name: "Manual", Type: domain.CarrierTypeManual, Active: true}
}
func (quotationProviderStub) Quote(ctx context.Context, request domain.QuotationRequest) (*domain.QuotationResult, error) {
	return &domain.QuotationResult{CarrierID: request.CarrierID, OrderID: request.OrderID, OriginCityCode: request.OriginCityCode, DestCityCode: request.DestCityCode, FreightCost: 12000, EstimatedDays: 2, CurrencyCode: "COP", ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (quotationProviderStub) GenerateMark(ctx context.Context, mark *domain.ShippingMark) error {
	return nil
}
func (quotationProviderStub) VoidMark(ctx context.Context, trackingNumber string) error { return nil }
func (quotationProviderStub) CheckBalance(ctx context.Context) error                    { return nil }
func (quotationProviderStub) SupportsQuotation() bool                                   { return true }

type quotationRegistryStub struct {
	provider port.CarrierProvider
}

func (s quotationRegistryStub) CarrierProvider(carrierID string) (port.CarrierProvider, bool) {
	if s.provider == nil {
		return nil, false
	}

	return s.provider, true
}

func (quotationRegistryStub) TrackingProvider(carrierID string) (port.TrackingProvider, bool) {
	return nil, false
}

func (quotationRegistryStub) Carriers() []domain.Carrier {
	return nil
}

// TestQuote verifies quotation orchestration behavior.
func TestQuote(t *testing.T) {
	repository := &quotationRepositoryStub{}
	service := NewService(repository, quotationRegistryStub{provider: quotationProviderStub{}})

	result, err := service.Quote(context.Background(), QuoteCommand{
		OrderID:        "order-1",
		CarrierID:      "manual",
		OriginCityCode: "11001000",
		DestCityCode:   "76001000",
		DeclaredValue:  50000,
		Units:          []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
	})
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if result == nil || result.FreightCost <= 0 {
		t.Fatalf("unexpected result = %#v", result)
	}
	if len(repository.rows) != 1 {
		t.Fatalf("stored rows = %d, want 1", len(repository.rows))
	}
}
