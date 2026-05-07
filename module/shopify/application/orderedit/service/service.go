package service

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"
	ordersport "mannaiah/module/orders/port"
	shopifyport "mannaiah/module/shopify/port"
)

var (
	// ErrNilDestination is returned when Shopify destination dependencies are nil.
	ErrNilDestination = errors.New("shopify order-edit destination must not be nil")
	// ErrNilSyncLinks is returned when sync-link repositories are nil.
	ErrNilSyncLinks = errors.New("shopify order-edit sync links must not be nil")
)

const (
	shopifyRealm = "shopify"
	syncAuthor   = "shopify_sync"
)

// Service applies Mannaiah-origin operational order updates to Shopify orders.
type Service struct {
	// destination defines Shopify mutation dependencies.
	destination shopifyport.ShopifyOrderDestination
	// links defines Shopify sync-link persistence dependencies.
	links shopifyport.SyncLinkRepository
	// logger defines structured logging dependencies.
	logger *zap.Logger
}

var (
	// _ ensures Service satisfies messaging handler contracts.
	_ OrderEventHandler = (*Service)(nil)
)

// OrderEventHandler defines order event handling behavior.
type OrderEventHandler interface {
	// HandleOrderEvent handles one Mannaiah order event.
	HandleOrderEvent(ctx context.Context, payload ordersport.OrderEventPayload) error
}

// NewService creates Shopify order edit services.
func NewService(destination shopifyport.ShopifyOrderDestination, links shopifyport.SyncLinkRepository, providedLogger *zap.Logger) (*Service, error) {
	if destination == nil {
		return nil, ErrNilDestination
	}
	if links == nil {
		return nil, ErrNilSyncLinks
	}
	if providedLogger == nil {
		providedLogger = zap.NewNop()
	}

	return &Service{destination: destination, links: links, logger: providedLogger}, nil
}

// HandleOrderEvent handles one Mannaiah order event.
func (s *Service) HandleOrderEvent(ctx context.Context, payload ordersport.OrderEventPayload) error {
	if !isShopifyRealm(payload.Realm) || isShopifySyncSource(payload.Source, payload.LatestStatus.Author) {
		return nil
	}
	link, err := s.links.GetLinkByMannaiahID(ctx, shopifyport.SyncKindOrder, payload.ID)
	if err != nil || link == nil {
		return err
	}
	requestCtx := shopifyport.WithShopDomain(ctx, link.ShopDomain)
	if strings.EqualFold(payload.CurrentStatus, "CANCELLED") {
		return s.destination.CancelOrder(requestCtx, link.ShopifyID, "Mannaiah cancellation")
	}

	return s.destination.ApplyOrderUpdate(requestCtx, link.ShopifyID, payload, s)
}

// ResolveVariantID resolves one Shopify variant ID for a Mannaiah product ID.
func (s *Service) ResolveVariantID(ctx context.Context, productID string) (string, error) {
	trimmedProductID := strings.TrimSpace(productID)
	if trimmedProductID == "" {
		return "", nil
	}
	link, err := s.links.GetLinkByMannaiahID(ctx, shopifyport.SyncKindVariant, trimmedProductID)
	if err != nil || link == nil {
		return "", err
	}
	return strings.TrimSpace(link.ShopifyID), nil
}

func isShopifyRealm(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), shopifyRealm)
}

func isShopifySyncSource(source string, author string) bool {
	return strings.EqualFold(strings.TrimSpace(source), syncAuthor) || strings.EqualFold(strings.TrimSpace(author), syncAuthor)
}
