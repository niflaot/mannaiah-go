package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

// Config defines quotation service behavior configuration values.
type Config struct {
	// ExpirationTTLMinutes defines how many minutes a stored quotation is valid before it expires.
	// Zero or negative values default to 10 minutes.
	ExpirationTTLMinutes int
}

// QuoteCommand defines quotation command input values.
type QuoteCommand struct {
	// OrderID defines optional order identifier values.
	OrderID string
	// OrderIdentifier defines optional external order identifier values (e.g. WooCommerce number).
	OrderIdentifier string
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
	// ShipmentMode defines the delivery mode for this quotation.
	ShipmentMode domain.ShipmentMode
}

// Service defines quotation orchestration behavior.
type Service struct {
	// repository defines quotation persistence dependencies.
	repository port.QuotationRepository
	// registry defines carrier registry dependencies.
	registry port.ProviderRegistry
	// cfg defines quotation behavior configuration values.
	cfg Config
	// orderSource defines optional order data source dependencies.
	orderSource port.OrderQuotationSource
	// productSource defines optional product shipping attribute source dependencies.
	productSource port.OrderProductSource
}

// NewService creates quotation services.
func NewService(repository port.QuotationRepository, registry port.ProviderRegistry, cfg Config) *Service {
	ttl := cfg.ExpirationTTLMinutes
	if ttl <= 0 {
		ttl = 10
	}

	return &Service{
		repository: repository,
		registry:   registry,
		cfg: Config{
			ExpirationTTLMinutes: ttl,
		},
	}
}

// SetOrderSource configures the order data source used for order-based quotation workflows.
func (s *Service) SetOrderSource(source port.OrderQuotationSource) {
	s.orderSource = source
}

// SetProductSource configures the product shipping attribute source used for box-packing.
func (s *Service) SetProductSource(source port.OrderProductSource) {
	s.productSource = source
}

// Quote requests one carrier quotation and stores the audit record.
func (s *Service) Quote(ctx context.Context, command QuoteCommand) (*domain.QuotationResult, error) {
	request := domain.QuotationRequest{
		OrderID:                 strings.TrimSpace(command.OrderID),
		CarrierID:               strings.TrimSpace(command.CarrierID),
		OriginCityCode:          strings.TrimSpace(command.OriginCityCode),
		DestCityCode:            strings.TrimSpace(command.DestCityCode),
		Units:                   command.Units,
		DeclaredValue:           command.DeclaredValue,
		CollectOnDeliveryAmount: command.CollectOnDeliveryAmount,
		ShipmentMode:            command.ShipmentMode,
	}.Normalize()
	request.ShipmentMode = resolveShipmentModeByUnits(request.Units)
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
		if isCityError(err) {
			return nil, fmt.Errorf("%w: %v", domain.ErrInvalidCityCode, err)
		}
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
	if strings.TrimSpace(result.OrderIdentifier) == "" {
		result.OrderIdentifier = strings.TrimSpace(command.OrderIdentifier)
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
	if result.ExpiresAt.IsZero() {
		result.ExpiresAt = result.CreatedAt.Add(time.Duration(s.cfg.ExpirationTTLMinutes) * time.Minute)
	}
	result.FreightCost = normalizeMoney(result.FreightCost)
	result.CollectOnDeliveryAmount = normalizeMoney(request.CollectOnDeliveryAmount)
	result.CollectOnDeliveryFeePercent = normalizeMoney(result.CollectOnDeliveryFeePercent)
	if result.CollectOnDeliveryAmount <= 0 {
		result.CollectOnDeliveryFeePercent = 0
		result.CollectOnDeliveryFeeAmount = 0
		result.CollectOnDeliveryChargedAmount = 0
	} else {
		result.CollectOnDeliveryChargedAmount = normalizeMoney(result.CollectOnDeliveryChargedAmount)
		if result.CollectOnDeliveryChargedAmount <= 0 {
			result.CollectOnDeliveryChargedAmount = applySurcharge(result.CollectOnDeliveryAmount, result.CollectOnDeliveryFeePercent)
		}
		result.CollectOnDeliveryFeeAmount = normalizeMoney(result.CollectOnDeliveryChargedAmount - result.CollectOnDeliveryAmount)
		if result.CollectOnDeliveryFeeAmount < 0 {
			result.CollectOnDeliveryFeeAmount = 0
		}
	}

	if s.repository != nil {
		snapshot, _ := json.Marshal(request)
		encodedRequestSnapshot := base64.StdEncoding.EncodeToString(snapshot)
		if trimmed := strings.TrimSpace(result.RequestSnapshot); trimmed != "" {
			encodedRequestSnapshot = base64.StdEncoding.EncodeToString([]byte(trimmed))
		}
		rawResponse := strings.TrimSpace(result.RawResponse)
		encodedRawResponse := ""
		if rawResponse != "" {
			encodedRawResponse = base64.StdEncoding.EncodeToString([]byte(rawResponse))
		}
		record := port.QuotationRecord{
			ID:              result.ID,
			OrderID:         result.OrderID,
			OrderIdentifier: result.OrderIdentifier,
			CarrierID:       result.CarrierID,
			OriginCityCode:  result.OriginCityCode,
			DestCityCode:    result.DestCityCode,
			FreightCost:     result.FreightCost,
			EstimatedDays:   result.EstimatedDays,
			CurrencyCode:    result.CurrencyCode,
			ExpiresAt:       result.ExpiresAt,
			Units:           request.Units,
			RequestSnapshot: encodedRequestSnapshot,
			RawResponse:     encodedRawResponse,
			CreatedAt:       result.CreatedAt,
		}
		if err := s.repository.Create(ctx, record); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// PurgeExpired deletes all expired quotation records and returns the number of deleted rows.
func (s *Service) PurgeExpired(ctx context.Context) (int64, error) {
	if s == nil || s.repository == nil {
		return 0, nil
	}

	return s.repository.DeleteExpired(ctx)
}

// ListByOrderID lists quotation history rows by order identifier.
func (s *Service) ListByOrderID(ctx context.Context, orderID string) ([]port.QuotationRecord, error) {
	if s == nil || s.repository == nil {
		return []port.QuotationRecord{}, nil
	}

	return s.repository.ListByOrderID(ctx, strings.TrimSpace(orderID))
}

// GetLatestByOrderAndCarrier returns the most recent non-expired quotation for the given order and carrier.
func (s *Service) GetLatestByOrderAndCarrier(ctx context.Context, orderID string, carrierID string) (*port.QuotationRecord, error) {
	if s == nil || s.repository == nil {
		return nil, nil
	}

	return s.repository.GetLatestByOrderAndCarrier(ctx, strings.TrimSpace(orderID), strings.TrimSpace(carrierID))
}

func resolveShipmentModeByUnits(units []domain.PackageUnit) domain.ShipmentMode {
	if len(units) <= 1 {
		return domain.ShipmentModeExpress
	}

	return domain.ShipmentModeParcel
}

func normalizeMoney(value float64) float64 {
	if value < 0 {
		value = 0
	}

	return math.Round(value*100) / 100
}

func applySurcharge(value float64, percent float64) float64 {
	normalizedValue := normalizeMoney(value)
	normalizedPercent := normalizeMoney(percent)
	if normalizedPercent <= 0 {
		return normalizedValue
	}

	factor := 1 + (normalizedPercent / 100)

	return normalizeMoney(normalizedValue * factor)
}
