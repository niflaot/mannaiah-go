package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// QuoteCommand defines quotation command input values.
type QuoteCommand struct {
	// OrderID defines optional order identifier values.
	OrderID string
	// CarrierID defines carrier identifier values.
	CarrierID string
	// OriginCityCode defines origin city-code values.
	OriginCityCode string
	// DestCityCode defines destination city-code values.
	DestCityCode string
	// Units defines package units.
	Units []domain.PackageUnit
	// DeclaredValue defines declared shipment value amounts.
	DeclaredValue float64
}

// Service defines quotation orchestration behavior.
type Service struct {
	// repository defines quotation persistence dependencies.
	repository port.QuotationRepository
	// registry defines carrier registry dependencies.
	registry port.ProviderRegistry
}

// NewService creates quotation services.
func NewService(repository port.QuotationRepository, registry port.ProviderRegistry) *Service {
	return &Service{repository: repository, registry: registry}
}

// Quote requests one carrier quotation and stores the audit record.
func (s *Service) Quote(ctx context.Context, command QuoteCommand) (*domain.QuotationResult, error) {
	request := domain.QuotationRequest{
		OrderID:        strings.TrimSpace(command.OrderID),
		CarrierID:      strings.TrimSpace(command.CarrierID),
		OriginCityCode: strings.TrimSpace(command.OriginCityCode),
		DestCityCode:   strings.TrimSpace(command.DestCityCode),
		Units:          command.Units,
		DeclaredValue:  command.DeclaredValue,
	}.Normalize()
	if err := request.Validate(); err != nil {
		return nil, err
	}
	if s == nil || s.registry == nil {
		return nil, domain.ErrCarrierNotSupported
	}
	provider, exists := s.registry.CarrierProvider(request.CarrierID)
	if !exists || provider == nil {
		return nil, domain.ErrCarrierNotSupported
	}
	if !provider.SupportsQuotation() {
		return nil, domain.ErrQuotationNotSupported
	}

	result, err := provider.Quote(ctx, request)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, domain.ErrQuotationNotSupported
	}
	if strings.TrimSpace(result.ID) == "" {
		result.ID = uuid.NewString()
	}
	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now().UTC()
	}
	if strings.TrimSpace(result.CarrierID) == "" {
		result.CarrierID = request.CarrierID
	}
	if strings.TrimSpace(result.OrderID) == "" {
		result.OrderID = request.OrderID
	}
	if strings.TrimSpace(result.OriginCityCode) == "" {
		result.OriginCityCode = request.OriginCityCode
	}
	if strings.TrimSpace(result.DestCityCode) == "" {
		result.DestCityCode = request.DestCityCode
	}
	if strings.TrimSpace(result.CurrencyCode) == "" {
		result.CurrencyCode = "COP"
	}

	if s.repository != nil {
		snapshot, _ := json.Marshal(request)
		record := port.QuotationRecord{
			ID:              result.ID,
			OrderID:         result.OrderID,
			CarrierID:       result.CarrierID,
			OriginCityCode:  result.OriginCityCode,
			DestCityCode:    result.DestCityCode,
			FreightCost:     result.FreightCost,
			EstimatedDays:   result.EstimatedDays,
			CurrencyCode:    result.CurrencyCode,
			ExpiresAt:       result.ExpiresAt,
			RequestSnapshot: string(snapshot),
			RawResponse:     result.RawResponse,
			CreatedAt:       result.CreatedAt,
		}
		if err := s.repository.Create(ctx, record); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// ListByOrderID lists quotation history rows by order identifier.
func (s *Service) ListByOrderID(ctx context.Context, orderID string) ([]port.QuotationRecord, error) {
	if s == nil || s.repository == nil {
		return []port.QuotationRecord{}, nil
	}

	return s.repository.ListByOrderID(ctx, strings.TrimSpace(orderID))
}
