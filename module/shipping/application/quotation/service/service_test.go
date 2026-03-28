package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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

func (s *quotationRepositoryStub) GetByID(ctx context.Context, id string) (*port.QuotationRecord, error) {
	for _, row := range s.rows {
		if row.ID == id {
			copy := row
			return &copy, nil
		}
	}

	return nil, domain.ErrNotFound
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

func (s *quotationRepositoryStub) GetLatestByOrderAndCarrier(ctx context.Context, orderID string, carrierID string) (*port.QuotationRecord, error) {
	return nil, nil
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
func (quotationProviderNoExpiry) VoidMark(ctx context.Context, trackingNumber string) error {
	return nil
}
func (quotationProviderNoExpiry) CheckBalance(ctx context.Context) error { return nil }
func (quotationProviderNoExpiry) SupportsQuotation() bool                { return true }

type quotationProviderStub struct{}

func (quotationProviderStub) CarrierID() string { return "manual" }
func (quotationProviderStub) Carrier() domain.Carrier {
	return domain.Carrier{ID: "manual", Name: "Manual", Type: domain.CarrierTypeManual, Active: true}
}
func (quotationProviderStub) Quote(ctx context.Context, request domain.QuotationRequest) (*domain.QuotationResult, error) {
	return &domain.QuotationResult{
		CarrierID:      request.CarrierID,
		OrderID:        request.OrderID,
		OriginCityCode: request.OriginCityCode,
		DestCityCode:   request.DestCityCode,
		FreightCost:    12000,
		EstimatedDays:  2,
		CurrencyCode:   "COP",
		ExpiresAt:      time.Now().Add(time.Hour),
		RawResponse:    `{"provider":"manual"}`,
	}, nil
}
func (quotationProviderStub) GenerateMark(ctx context.Context, mark *domain.ShippingMark) error {
	return nil
}
func (quotationProviderStub) VoidMark(ctx context.Context, trackingNumber string) error { return nil }
func (quotationProviderStub) CheckBalance(ctx context.Context) error                    { return nil }
func (quotationProviderStub) SupportsQuotation() bool                                   { return true }

type quotationProviderCaptureStub struct {
	lastRequest domain.QuotationRequest
}

func (s *quotationProviderCaptureStub) CarrierID() string { return "manual" }
func (s *quotationProviderCaptureStub) Carrier() domain.Carrier {
	return domain.Carrier{ID: "manual", Name: "Manual", Type: domain.CarrierTypeManual, Active: true}
}
func (s *quotationProviderCaptureStub) Quote(ctx context.Context, request domain.QuotationRequest) (*domain.QuotationResult, error) {
	s.lastRequest = request
	return &domain.QuotationResult{
		CarrierID:      request.CarrierID,
		OrderID:        request.OrderID,
		OriginCityCode: request.OriginCityCode,
		DestCityCode:   request.DestCityCode,
		FreightCost:    12000,
		EstimatedDays:  2,
		CurrencyCode:   "COP",
		ExpiresAt:      time.Now().Add(time.Hour),
		RawResponse:    `{"provider":"manual"}`,
	}, nil
}
func (s *quotationProviderCaptureStub) GenerateMark(ctx context.Context, mark *domain.ShippingMark) error {
	return nil
}
func (s *quotationProviderCaptureStub) VoidMark(ctx context.Context, trackingNumber string) error {
	return nil
}
func (s *quotationProviderCaptureStub) CheckBalance(ctx context.Context) error { return nil }
func (s *quotationProviderCaptureStub) SupportsQuotation() bool                { return true }

type quotationProviderCityErrorStub struct{}

func (quotationProviderCityErrorStub) CarrierID() string { return "manual" }
func (quotationProviderCityErrorStub) Carrier() domain.Carrier {
	return domain.Carrier{ID: "manual", Name: "Manual", Type: domain.CarrierTypeManual, Active: true}
}
func (quotationProviderCityErrorStub) Quote(ctx context.Context, request domain.QuotationRequest) (*domain.QuotationResult, error) {
	return nil, errors.New("tcc quotation rejected: Codigo de ciudad de origen esta incorrecto")
}
func (quotationProviderCityErrorStub) GenerateMark(ctx context.Context, mark *domain.ShippingMark) error {
	return nil
}
func (quotationProviderCityErrorStub) VoidMark(ctx context.Context, trackingNumber string) error {
	return nil
}
func (quotationProviderCityErrorStub) CheckBalance(ctx context.Context) error { return nil }
func (quotationProviderCityErrorStub) SupportsQuotation() bool                { return true }

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

type orderSourceStub struct {
	row *port.OrderQuotationData
}

func (s orderSourceStub) GetByIDOrIdentifier(ctx context.Context, identifier string) (*port.OrderQuotationData, error) {
	return s.row, nil
}

type productSourceStub struct {
	attrsBySKU map[string]port.ProductShippingAttributes
}

func (s productSourceStub) GetShippingAttributes(ctx context.Context, sku string) (*port.ProductShippingAttributes, error) {
	row, exists := s.attrsBySKU[sku]
	if !exists {
		return nil, nil
	}
	copy := row
	return &copy, nil
}

func (s productSourceStub) GetShippingAttributesByID(ctx context.Context, productID string) (*port.ProductShippingAttributes, error) {
	return nil, nil
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
	decodedSnapshot, decodeErr := base64.StdEncoding.DecodeString(repository.rows[0].RequestSnapshot)
	if decodeErr != nil {
		t.Fatalf("decode request snapshot: %v", decodeErr)
	}
	var snapshot map[string]any
	if jsonErr := json.Unmarshal(decodedSnapshot, &snapshot); jsonErr != nil {
		t.Fatalf("unmarshal request snapshot: %v", jsonErr)
	}
	if snapshot["shipmentMode"] != "express" {
		t.Fatalf("snapshot shipmentMode = %v, want express", snapshot["shipmentMode"])
	}
	if repository.rows[0].RawResponse == "" {
		t.Fatalf("stored raw response should be base64-encoded and non-empty")
	}
	if _, decodeErr := base64.StdEncoding.DecodeString(repository.rows[0].RawResponse); decodeErr != nil {
		t.Fatalf("decode raw response: %v", decodeErr)
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
	service := NewService(repository, quotationRegistryStub{provider: quotationProviderNoExpiry{}}, Config{ExpirationTTLMinutes: 120})

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

// TestQuoteForcesParcelWhenMultipleUnits verifies shipment mode normalization to parcel for two or more units.
func TestQuoteForcesParcelWhenMultipleUnits(t *testing.T) {
	repository := &quotationRepositoryStub{}
	provider := &quotationProviderCaptureStub{}
	service := NewService(repository, quotationRegistryStub{provider: provider}, Config{})

	_, err := service.Quote(context.Background(), QuoteCommand{
		OrderID:        "order-2-boxes",
		CarrierID:      "manual",
		OriginCityCode: "11001000",
		DestCityCode:   "76001000",
		ShipmentMode:   domain.ShipmentModeExpress,
		Units: []domain.PackageUnit{
			{Description: "box-1", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 2}},
			{Description: "box-2", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 12, WidthCM: 12, DepthCM: 12, RealWeightKG: 2}},
		},
	})
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if provider.lastRequest.ShipmentMode != domain.ShipmentModeParcel {
		t.Fatalf("provider.lastRequest.ShipmentMode = %q, want %q", provider.lastRequest.ShipmentMode, domain.ShipmentModeParcel)
	}
}

// TestQuoteMapsCityValidationErrors verifies provider city-code errors map to ErrInvalidCityCode.
func TestQuoteMapsCityValidationErrors(t *testing.T) {
	repository := &quotationRepositoryStub{}
	service := NewService(repository, quotationRegistryStub{provider: quotationProviderCityErrorStub{}}, Config{})

	_, err := service.Quote(context.Background(), QuoteCommand{
		OrderID:        "order-city",
		CarrierID:      "manual",
		OriginCityCode: "1024608",
		DestCityCode:   "11001",
		ShipmentMode:   domain.ShipmentModeExpress,
		Units:          []domain.PackageUnit{{Description: "box", PackageType: "CAJA", Dimensions: domain.Dimensions{HeightCM: 10, WidthCM: 10, DepthCM: 10, RealWeightKG: 1}}},
	})
	if !errors.Is(err, domain.ErrInvalidCityCode) {
		t.Fatalf("Quote() error = %v, want ErrInvalidCityCode", err)
	}
}

// TestQuoteFromOrderReturnsCityValidationErrors verifies order quotations surface invalid city as errors.
func TestQuoteFromOrderReturnsCityValidationErrors(t *testing.T) {
	repository := &quotationRepositoryStub{}
	service := NewService(repository, quotationRegistryStub{provider: quotationProviderCityErrorStub{}}, Config{})
	service.SetOrderSource(orderSourceStub{row: &port.OrderQuotationData{
		OrderID:         "order-city",
		OrderIdentifier: "1024554",
		DestCityCode:    "11001",
		TotalValue:      311000,
		Items:           []port.OrderQuotationItem{{SKU: "7709738583238", Quantity: 2}},
	}})
	service.SetProductSource(productSourceStub{attrsBySKU: map[string]port.ProductShippingAttributes{
		"7709738583238": {
			SKU:        "7709738583238",
			WeightKG:   1,
			HeightCM:   5,
			WidthCM:    40,
			LengthCM:   30,
			Price:      157000,
			Overlapped: false,
			Valid:      true,
		},
	}})

	_, err := service.QuoteFromOrder(context.Background(), QuoteFromOrderCommand{
		OrderIdentifier: "1024554",
		CarrierID:       "manual",
		OriginCityCode:  "1024608",
	})
	if !errors.Is(err, domain.ErrInvalidCityCode) {
		t.Fatalf("QuoteFromOrder() error = %v, want ErrInvalidCityCode", err)
	}
}

// TestOrderPackagingFromOrderPreviewsUnitsWithoutPersistence verifies packaging previews are computed without carrier calls or quotation persistence.
func TestOrderPackagingFromOrderPreviewsUnitsWithoutPersistence(t *testing.T) {
	repository := &quotationRepositoryStub{}
	service := NewService(repository, quotationRegistryStub{provider: quotationProviderStub{}}, Config{})
	service.SetOrderSource(orderSourceStub{row: &port.OrderQuotationData{
		OrderID:         "order-packaging",
		OrderIdentifier: "1024554",
		DestCityCode:    "11001",
		TotalValue:      311000,
		Items: []port.OrderQuotationItem{
			{SKU: "7709738583238", Quantity: 1},
			{SKU: "7709296832021", Quantity: 1},
		},
	}})
	service.SetProductSource(productSourceStub{attrsBySKU: map[string]port.ProductShippingAttributes{
		"7709738583238": {
			SKU:        "7709738583238",
			WeightKG:   1,
			HeightCM:   5,
			WidthCM:    40,
			LengthCM:   30,
			Price:      157000,
			Overlapped: false,
			Valid:      true,
		},
		"7709296832021": {
			SKU:        "7709296832021",
			WeightKG:   1,
			HeightCM:   5,
			WidthCM:    40,
			LengthCM:   30,
			Price:      154000,
			Overlapped: false,
			Valid:      true,
		},
	}})

	result, err := service.OrderPackagingFromOrder(context.Background(), QuoteFromOrderCommand{
		OrderIdentifier: "1024554",
		CarrierID:       "manual",
		OriginCityCode:  "11001",
	})
	if err != nil {
		t.Fatalf("OrderPackagingFromOrder() error = %v", err)
	}
	if result == nil {
		t.Fatal("OrderPackagingFromOrder() returned nil result")
	}
	if len(result.Units) != 2 {
		t.Fatalf("units = %d, want 2", len(result.Units))
	}
	if result.Units[0].Dimensions.VolumetricWeightKG != 2.4 {
		t.Fatalf("units[0].dimensions.volumetricWeightKg = %v, want 2.4", result.Units[0].Dimensions.VolumetricWeightKG)
	}
	if result.ShipmentMode != domain.ShipmentModeParcel {
		t.Fatalf("shipment mode = %q, want %q", result.ShipmentMode, domain.ShipmentModeParcel)
	}
	if result.CollectOnDeliveryAmount != 0 {
		t.Fatalf("collectOnDeliveryAmount = %v, want 0", result.CollectOnDeliveryAmount)
	}
	if len(repository.rows) != 0 {
		t.Fatalf("quotation repository rows = %d, want 0", len(repository.rows))
	}
}

// TestQuoteFromOrderUsesResolvedCollectOnDeliveryAmount verifies QuoteFromOrder forwards resolved COD amounts from order data.
func TestQuoteFromOrderUsesResolvedCollectOnDeliveryAmount(t *testing.T) {
	repository := &quotationRepositoryStub{}
	provider := &quotationProviderCaptureStub{}
	service := NewService(repository, quotationRegistryStub{provider: provider}, Config{})
	service.SetOrderSource(orderSourceStub{row: &port.OrderQuotationData{
		OrderID:                 "order-cod",
		OrderIdentifier:         "1024590",
		DestCityCode:            "11001",
		TotalValue:              311000,
		CollectOnDeliveryAmount: 0,
		Items: []port.OrderQuotationItem{
			{SKU: "7709738583238", Quantity: 1},
		},
	}})
	service.SetProductSource(productSourceStub{attrsBySKU: map[string]port.ProductShippingAttributes{
		"7709738583238": {
			SKU:        "7709738583238",
			WeightKG:   1,
			HeightCM:   5,
			WidthCM:    40,
			LengthCM:   30,
			Price:      157000,
			Overlapped: false,
			Valid:      true,
		},
	}})

	result, err := service.QuoteFromOrder(context.Background(), QuoteFromOrderCommand{
		OrderIdentifier: "1024590",
		CarrierID:       "manual",
		OriginCityCode:  "11001",
	})
	if err != nil {
		t.Fatalf("QuoteFromOrder() error = %v", err)
	}
	if result == nil {
		t.Fatal("QuoteFromOrder() returned nil result")
	}
	if result.CollectOnDeliveryAmount != 0 {
		t.Fatalf("result.CollectOnDeliveryAmount = %v, want 0", result.CollectOnDeliveryAmount)
	}
	if provider.lastRequest.CollectOnDeliveryAmount != 0 {
		t.Fatalf("provider.lastRequest.CollectOnDeliveryAmount = %v, want 0", provider.lastRequest.CollectOnDeliveryAmount)
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
