package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
	shopifyport "mannaiah/module/shopify/port"
)

var (
	// ErrNilDestination is returned when a nil Shopify destination is provided.
	ErrNilDestination = errors.New("shopify order destination must not be nil")
	// ErrNilLinkRepository is returned when a nil sync-link repository is provided.
	ErrNilLinkRepository = errors.New("shopify link repository must not be nil")
)

// OrderEventHandler defines mainstream order integration event handling behavior.
type OrderEventHandler interface {
	// HandleOrderEvent pushes one mainstream order event back to Shopify when appropriate.
	HandleOrderEvent(ctx context.Context, payload ordersport.OrderEventPayload) error
}

// MainstreamUpdateService defines outbound Shopify order update behavior.
type MainstreamUpdateService struct {
	// destination defines Shopify destination dependencies.
	destination shopifyport.OrderDestination
	// links defines Shopify sync-link persistence dependencies.
	links shopifyport.SyncLinkRepository
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// destinationBreaker defines optional outbound breaker behavior.
	destinationBreaker CircuitBreaker
}

var (
	// _ ensures MainstreamUpdateService satisfies handler contracts.
	_ OrderEventHandler = (*MainstreamUpdateService)(nil)
)

// NewMainstreamUpdateService creates outbound Shopify order update services.
func NewMainstreamUpdateService(destination shopifyport.OrderDestination, links shopifyport.SyncLinkRepository, providedLogger *zap.Logger, breakers ...CircuitBreakers) (*MainstreamUpdateService, error) {
	if destination == nil {
		return nil, ErrNilDestination
	}
	if links == nil {
		return nil, ErrNilLinkRepository
	}

	resolvedBreaker := CircuitBreakers{}
	if len(breakers) > 0 {
		resolvedBreaker = breakers[0]
	}
	logger := providedLogger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &MainstreamUpdateService{
		destination:        destination,
		links:              links,
		logger:             logger,
		destinationBreaker: resolvedBreaker.Destination,
	}, nil
}

// HandleOrderEvent pushes one mainstream order event back to Shopify when appropriate.
func (s *MainstreamUpdateService) HandleOrderEvent(ctx context.Context, payload ordersport.OrderEventPayload) error {
	if !strings.EqualFold(strings.TrimSpace(payload.Realm), "shopify") {
		return nil
	}
	if strings.HasPrefix(strings.TrimSpace(payload.Source), "shopify_") {
		return nil
	}
	status := resolvePayloadStatus(payload)
	if strings.TrimSpace(string(status)) == "" || strings.TrimSpace(payload.ID) == "" {
		return nil
	}

	link, err := s.links.GetLinkByMannaiahID(ctx, shopifyport.SyncKindOrder, payload.ID)
	if err != nil || link == nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(link.LastKnownStatus), strings.TrimSpace(string(status))) {
		return nil
	}

	err = s.executeWithBreaker(s.destinationBreaker, ErrIntegrationUnavailable, func() error {
		return s.destination.UpdateOrderFromMainstream(ctx, link.ShopifyID, shopifyport.MainstreamOrderUpdateCommand{Status: status})
	})
	if err != nil {
		if !errors.Is(err, ErrIntegrationUnavailable) {
			return fmt.Errorf("%w: %v", ErrIntegrationUnavailable, err)
		}
		return err
	}

	if updateErr := s.links.UpdateLastKnownStatus(ctx, shopifyport.SyncKindOrder, payload.ID, string(status)); updateErr != nil {
		s.logger.Warn("update shopify last known status failed", zap.Error(updateErr))
		return updateErr
	}

	return nil
}

func (s *MainstreamUpdateService) executeWithBreaker(breaker CircuitBreaker, unavailableErr error, fn func() error) error {
	if breaker == nil {
		return fn()
	}

	var operationErr error
	err := breaker.Execute(func() error {
		operationErr = fn()
		return operationErr
	})
	if err == nil {
		return nil
	}
	if operationErr != nil {
		return operationErr
	}

	return unavailableErr
}

func resolvePayloadStatus(payload ordersport.OrderEventPayload) ordersdomain.Status {
	if strings.TrimSpace(payload.LatestStatus.Status) != "" {
		return ordersdomain.Status(strings.TrimSpace(payload.LatestStatus.Status))
	}
	if strings.TrimSpace(payload.CurrentStatus) != "" {
		return ordersdomain.Status(strings.TrimSpace(payload.CurrentStatus))
	}
	return ""
}
