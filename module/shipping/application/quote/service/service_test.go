package service

import (
	"context"
	"errors"
	"testing"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// gatewayMock defines quote-gateway behavior for service tests.
type gatewayMock struct {
	// result defines successful quote result values.
	result *domain.QuoteResult
	// err defines quote execution errors.
	err error
	// request captures quote request values.
	request domain.QuoteRequest
}

// Quote returns configured quote result values.
func (m *gatewayMock) Quote(ctx context.Context, request domain.QuoteRequest) (*domain.QuoteResult, error) {
	m.request = request
	if m.err != nil {
		return nil, m.err
	}

	return m.result, nil
}

// TestNewServiceValidation verifies constructor validation behavior.
func TestNewServiceValidation(t *testing.T) {
	if _, err := NewService(nil); !errors.Is(err, ErrNilGatewayMap) {
		t.Fatalf("NewService(nil) error = %v, want %v", err, ErrNilGatewayMap)
	}
	if _, err := NewService(map[domain.Carrier]port.RateQuoteGateway{domain.CarrierTCC: nil}); !errors.Is(err, ErrNilGatewayMap) {
		t.Fatalf("NewService(nil gateway) error = %v, want %v", err, ErrNilGatewayMap)
	}
}

// TestQuoteSuccess verifies quote orchestration behavior.
func TestQuoteSuccess(t *testing.T) {
	mock := &gatewayMock{result: &domain.QuoteResult{CarrierMessage: "ok", QuoteValue: 42}}
	service, err := NewService(map[domain.Carrier]port.RateQuoteGateway{domain.CarrierTCC: mock})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, err := service.Quote(context.Background(), domain.QuoteRequest{
		Carrier:             "TCC",
		BusinessUnit:        "COURIER",
		OriginCityCode:      "05001",
		DestinationCityCode: "11001",
		DeclaredValue:       100,
		Units:               []domain.QuoteUnit{{Number: 1, RealWeight: 1, Height: 1, Width: 1, Length: 1}},
	})
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if result == nil {
		t.Fatalf("expected quote result")
	}
	if result.BusinessUnit != domain.BusinessUnitCourier {
		t.Fatalf("result.BusinessUnit = %q, want %q", result.BusinessUnit, domain.BusinessUnitCourier)
	}
	if mock.request.Carrier != domain.CarrierTCC {
		t.Fatalf("mock.request.Carrier = %q, want %q", mock.request.Carrier, domain.CarrierTCC)
	}
}

// TestQuoteErrors verifies quote error mapping behavior.
func TestQuoteErrors(t *testing.T) {
	service, err := NewService(map[domain.Carrier]port.RateQuoteGateway{domain.CarrierTCC: &gatewayMock{err: errors.New("boom")}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Quote(context.Background(), domain.QuoteRequest{
		Carrier:             "tcc",
		BusinessUnit:        "courier",
		OriginCityCode:      "05001",
		DestinationCityCode: "11001",
		DeclaredValue:       10,
		Units:               []domain.QuoteUnit{{Number: 1, RealWeight: 1, Height: 1, Width: 1, Length: 1}},
	})
	if !errors.Is(err, domain.ErrIntegrationUnavailable) {
		t.Fatalf("Quote() error = %v, want %v", err, domain.ErrIntegrationUnavailable)
	}

	_, err = service.Quote(context.Background(), domain.QuoteRequest{
		Carrier:             "other",
		BusinessUnit:        "courier",
		OriginCityCode:      "05001",
		DestinationCityCode: "11001",
		DeclaredValue:       10,
		Units:               []domain.QuoteUnit{{Number: 1, RealWeight: 1, Height: 1, Width: 1, Length: 1}},
	})
	if !errors.Is(err, domain.ErrUnsupportedCarrier) {
		t.Fatalf("Quote() error = %v, want %v", err, domain.ErrUnsupportedCarrier)
	}
}
