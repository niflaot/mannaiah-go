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

func (s *quotationRepositoryStub) DeleteExpired(ctx context.Context) (int64, error) {
	now := time.Now()
	kept := s.rows[:0]
	var deleted int64
	for _, r := range s.rows {
		if !r.ExpiresAt.IsZero() && r.ExpiresAt.Before(now) {
			deleted++
		} else {
			kept = append(kept, r)
		}
	}
	s.rows = kept

	return deleted, nil
}

// quotationProviderNoExpiry returns a quotation result with a zero ExpiresAt to test TTL fallback.
type quotationProviderNoExpiry struct{}

func (quotationProviderNoExpiry) CarrierID() string { return "manual" }
func (quotationProviderNoExpiry) Carrier() domain.Carrier {
	return domain.Carrier{ID: "manual", Name: "Manual", Type: domain.CarrierTypeManual, Active: true}
}
func (quotationProviderNoExpiry) Quote(ctx context.Context, request domain.QuotationRequest) (*domain.QuotationResult, error) {
	return &domain.QuotationResult{CarrierID: request.CarrierID, OrderID: request.OrderID, FreightCost: 5000, EstimatedDays: 1, CurrencyCode: "COP"}, nil
}
func (quotationProviderNoExpiry) GenerateMark(ctx context.Context, mark *domain.ShippingMark) error {
	return nil
}
func (quotationProviderNoExpiry) VoidMark(ctx context.Context, trackingNumber string) error { return nil }
func (quotationProviderNoExpiry) CheckBalance(ctx context.Context) error                    { return nil }
func (quotationProviderNoExpiry) SupportsQuotation() bool                                   { return true }

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
	service := NewService(repository, quotationRegistryStub{provider: quotationProviderStub{}}, Config{})

	result, err := service.Quote(context.Background(), QuoteCommand{
		OrderID:                 "order-1",
		CarrierID:               "manual",
		OriginCityCode:          "11001000",
		DestCityCode:            "76001000",
		DeclaredValue:           50000,
		CollectOnDeliveryAmount: 100000,
		ShipmentMode:            domain.ShipmentModeParcel,
		Units:                   []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
	})
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if result == nil || result.FreightCost != 12000 {
		t.Fatalf("unexpected result = %#v", result)
	}
	if result.CollectOnDeliveryAmount != 100000 {
		t.Fatalf("result.CollectOnDeliveryAmount = %v, want %v", result.CollectOnDeliveryAmount, 100000.0)
	}
	if result.CollectOnDeliveryChargedAmount != 100000 {
		t.Fatalf("result.CollectOnDeliveryChargedAmount = %v, want %v", result.CollectOnDeliveryChargedAmount, 100000.0)
	}
	if result.CollectOnDeliveryFeePercent != 0 {
		t.Fatalf("result.CollectOnDeliveryFeePercent = %v, want 0", result.CollectOnDeliveryFeePercent)
	}
	if len(repository.rows) != 1 {
		t.Fatalf("stored rows = %d, want 1", len(repository.rows))
	}
	if repository.rows[0].FreightCost != 12000 {
		t.Fatalf("stored freight cost = %v, want 12000", repository.rows[0].FreightCost)
	}
}

// TestQuoteDefaultsCODFeeAmountWhenProviderOmitsFields verifies COD fee fallback behavior when providers omit COD fee fields.
func TestQuoteDefaultsCODFeeAmountWhenProviderOmitsFields(t *testing.T) {
	repository := &quotationRepositoryStub{}
	service := NewService(repository, quotationRegistryStub{provider: quotationProviderStub{}}, Config{})

	result, err := service.Quote(context.Background(), QuoteCommand{
		OrderID:                 "order-2",
		CarrierID:               "manual",
		OriginCityCode:          "11001000",
		DestCityCode:            "76001000",
		DeclaredValue:           50000,
		CollectOnDeliveryAmount: 100000,
		ShipmentMode:            domain.ShipmentModeParcel,
		Units:                   []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
	})
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if result.CollectOnDeliveryFeePercent != 0 {
		t.Fatalf("result.CollectOnDeliveryFeePercent = %v, want 0", result.CollectOnDeliveryFeePercent)
	}
	if result.CollectOnDeliveryFeeAmount != 0 {
		t.Fatalf("result.CollectOnDeliveryFeeAmount = %v, want 0", result.CollectOnDeliveryFeeAmount)
	}
	if result.CollectOnDeliveryChargedAmount != 100000 {
		t.Fatalf("result.CollectOnDeliveryChargedAmount = %v, want 100000", result.CollectOnDeliveryChargedAmount)
	}
}

// TestQuoteExpiresAtSetFromTTL verifies ExpiresAt is set from TTL when the provider returns zero.
func TestQuoteExpiresAtSetFromTTL(t *testing.T) {
	repository := &quotationRepositoryStub{}
	service := NewService(repository, quotationRegistryStub{provider: quotationProviderNoExpiry{}}, Config{ExpirationTTLHours: 2})

	before := time.Now()
	result, err := service.Quote(context.Background(), QuoteCommand{
		OrderID: "order-ttl", CarrierID: "manual",
		OriginCityCode: "11001000", DestCityCode: "76001000",
		ShipmentMode: domain.ShipmentModeParcel,
		Units:        []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}}},
	})
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	minExpiry := before.Add(time.Hour)
	if result.ExpiresAt.Before(minExpiry) {
		t.Fatalf("ExpiresAt = %v, want >= %v (2h from now)", result.ExpiresAt, minExpiry)
	}
}

// TestPurgeExpired verifies expired quotations are removed by PurgeExpired.
func TestPurgeExpired(t *testing.T) {
	repository := &quotationRepositoryStub{rows: []port.QuotationRecord{
		{ID: "q-expired", ExpiresAt: time.Now().Add(-time.Hour)},
		{ID: "q-valid", ExpiresAt: time.Now().Add(time.Hour)},
		{ID: "q-no-expiry"},
	}}
	service := NewService(repository, quotationRegistryStub{}, Config{})

	deleted, err := service.PurgeExpired(context.Background())
	if err != nil {
		t.Fatalf("PurgeExpired() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("PurgeExpired() deleted = %d, want 1", deleted)
	}
	if len(repository.rows) != 2 {
		t.Fatalf("remaining rows = %d, want 2", len(repository.rows))
	}
}

// TestApplySurcharge verifies surcharge application behavior.
func TestApplySurcharge(t *testing.T) {
	if got := applySurcharge(100000, 4); got != 104000 {
		t.Fatalf("applySurcharge(100000, 4) = %v", got)
	}
	if got := applySurcharge(100000, 0); got != 100000 {
		t.Fatalf("applySurcharge(100000, 0) = %v", got)
	}
	if got := applySurcharge(100000, -4); got != 100000 {
		t.Fatalf("applySurcharge(100000, -4) = %v", got)
	}
}
