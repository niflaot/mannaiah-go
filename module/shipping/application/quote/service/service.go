package service

import (
	"context"
	"errors"
	"fmt"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

var (
	// ErrNilGatewayMap is returned when gateway dependency maps are empty.
	ErrNilGatewayMap = errors.New("shipping quote gateways must not be empty")
)

// Service defines shipping quote use-case behavior.
type Service interface {
	// Quote retrieves one shipping quote from the requested carrier.
	Quote(ctx context.Context, request domain.QuoteRequest) (*domain.QuoteResult, error)
}

// RateQuoteService defines shipping quote use-case dependencies.
type RateQuoteService struct {
	// gateways defines carrier-specific quote adapter dependencies.
	gateways map[domain.Carrier]port.RateQuoteGateway
}

var (
	// _ ensures RateQuoteService satisfies service contracts.
	_ Service = (*RateQuoteService)(nil)
)

// NewService creates shipping quote services.
func NewService(gateways map[domain.Carrier]port.RateQuoteGateway) (*RateQuoteService, error) {
	if len(gateways) == 0 {
		return nil, ErrNilGatewayMap
	}

	resolved := make(map[domain.Carrier]port.RateQuoteGateway, len(gateways))
	for carrier, gateway := range gateways {
		if gateway == nil {
			continue
		}
		resolved[domain.NormalizeCarrier(string(carrier))] = gateway
	}
	if len(resolved) == 0 {
		return nil, ErrNilGatewayMap
	}

	return &RateQuoteService{gateways: resolved}, nil
}

// Quote retrieves one shipping quote from the requested carrier.
func (s *RateQuoteService) Quote(ctx context.Context, request domain.QuoteRequest) (*domain.QuoteResult, error) {
	request.Carrier = domain.NormalizeCarrier(string(request.Carrier))
	request.BusinessUnit = domain.NormalizeBusinessUnit(string(request.BusinessUnit))

	if err := domain.ValidateQuoteRequest(request); err != nil {
		return nil, err
	}

	gateway, ok := s.gateways[request.Carrier]
	if !ok || gateway == nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrUnsupportedCarrier, request.Carrier)
	}

	result, err := gateway.Quote(ctx, request)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteRejected) || errors.Is(err, domain.ErrIntegrationUnavailable) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrIntegrationUnavailable, err)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: empty quote result", domain.ErrIntegrationUnavailable)
	}

	result.BusinessUnit = request.BusinessUnit
	return result, nil
}
