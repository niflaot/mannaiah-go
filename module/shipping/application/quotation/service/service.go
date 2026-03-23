package service

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// Config defines quotation service behavior configuration values.
type Config struct {
	// DiscountPercent defines the freight discount percentage applied to quotation results.
	DiscountPercent float64
}

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
	// CollectOnDeliveryAmount defines requested cash-on-delivery collection amounts.
	CollectOnDeliveryAmount float64
	// CollectOnDeliveryFeePercent defines requested cash-on-delivery fee percentages.
	CollectOnDeliveryFeePercent float64
}

// Service defines quotation orchestration behavior.
type Service struct {
	// repository defines quotation persistence dependencies.
	repository port.QuotationRepository
	// registry defines carrier registry dependencies.
	registry port.ProviderRegistry
	// cfg defines quotation behavior configuration values.
	cfg Config
}

// NewService creates quotation services.
func NewService(repository port.QuotationRepository, registry port.ProviderRegistry, cfg Config) *Service {
	return &Service{
		repository: repository,
		registry:   registry,
		cfg: Config{
			DiscountPercent: normalizeDiscountPercent(cfg.DiscountPercent),
		},
	}
}

// Quote requests one carrier quotation and stores the audit record.
func (s *Service) Quote(ctx context.Context, command QuoteCommand) (*domain.QuotationResult, error) {
	request := domain.QuotationRequest{
		OrderID:                     strings.TrimSpace(command.OrderID),
		CarrierID:                   strings.TrimSpace(command.CarrierID),
		OriginCityCode:              strings.TrimSpace(command.OriginCityCode),
		DestCityCode:                strings.TrimSpace(command.DestCityCode),
		Units:                       command.Units,
		DeclaredValue:               command.DeclaredValue,
		CollectOnDeliveryAmount:     command.CollectOnDeliveryAmount,
		CollectOnDeliveryFeePercent: command.CollectOnDeliveryFeePercent,
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
	fullFreightCost := normalizeMoney(result.FullFreightCost)
	if fullFreightCost <= 0 {
		fullFreightCost = normalizeMoney(result.FreightCost)
	}
	discountPercent := normalizeDiscountPercent(s.cfg.DiscountPercent)
	discountedFreightCost := applyDiscount(fullFreightCost, discountPercent)
	result.FullFreightCost = fullFreightCost
	result.DiscountPercent = discountPercent
	result.DiscountedFreightCost = discountedFreightCost
	result.FreightCost = discountedFreightCost
	result.CollectOnDeliveryAmount = normalizeMoney(request.CollectOnDeliveryAmount)
	result.CollectOnDeliveryFeePercent = normalizeMoney(result.CollectOnDeliveryFeePercent)
	if result.CollectOnDeliveryAmount <= 0 {
		result.CollectOnDeliveryFeePercent = 0
		result.CollectOnDeliveryChargedAmount = 0
	} else {
		requestedCODFeePercent := normalizeDiscountPercent(request.CollectOnDeliveryFeePercent)
		if result.CollectOnDeliveryFeePercent <= 0 && requestedCODFeePercent > 0 {
			result.CollectOnDeliveryFeePercent = requestedCODFeePercent
		}
		result.CollectOnDeliveryChargedAmount = normalizeMoney(result.CollectOnDeliveryChargedAmount)
		if result.CollectOnDeliveryChargedAmount <= 0 {
			result.CollectOnDeliveryChargedAmount = applySurcharge(result.CollectOnDeliveryAmount, result.CollectOnDeliveryFeePercent)
		}
	}

	if s.repository != nil {
		snapshot, _ := json.Marshal(request)
		record := port.QuotationRecord{
			ID:                    result.ID,
			OrderID:               result.OrderID,
			CarrierID:             result.CarrierID,
			OriginCityCode:        result.OriginCityCode,
			DestCityCode:          result.DestCityCode,
			FullFreightCost:       result.FullFreightCost,
			DiscountPercent:       result.DiscountPercent,
			DiscountedFreightCost: result.DiscountedFreightCost,
			FreightCost:           result.FreightCost,
			EstimatedDays:         result.EstimatedDays,
			CurrencyCode:          result.CurrencyCode,
			ExpiresAt:             result.ExpiresAt,
			RequestSnapshot:       string(snapshot),
			RawResponse:           result.RawResponse,
			CreatedAt:             result.CreatedAt,
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

func normalizeDiscountPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}

	return normalizeMoney(value)
}

func normalizeMoney(value float64) float64 {
	if value < 0 {
		value = 0
	}

	return math.Round(value*100) / 100
}

func applyDiscount(value float64, discountPercent float64) float64 {
	normalizedValue := normalizeMoney(value)
	normalizedDiscount := normalizeDiscountPercent(discountPercent)
	if normalizedDiscount <= 0 {
		return normalizedValue
	}

	factor := 1 - (normalizedDiscount / 100)

	return normalizeMoney(normalizedValue * factor)
}

func applySurcharge(value float64, percent float64) float64 {
	normalizedValue := normalizeMoney(value)
	normalizedPercent := normalizeDiscountPercent(percent)
	if normalizedPercent <= 0 {
		return normalizedValue
	}

	factor := 1 + (normalizedPercent / 100)

	return normalizeMoney(normalizedValue * factor)
}
