package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	shopifyport "mannaiah/module/shopify/port"
)

var (
	// ErrNilFulfillmentDestination is returned when fulfillment destinations are nil.
	ErrNilFulfillmentDestination = errors.New("shopify fulfillment destination must not be nil")
	// ErrNilSyncLinks is returned when sync-link repositories are nil.
	ErrNilSyncLinks = errors.New("shopify fulfillment sync links must not be nil")
)

// MarkGeneratedPayload defines shipping mark-generated payload values.
type MarkGeneratedPayload struct {
	// MarkID defines mark identifiers.
	MarkID string `json:"markId"`
	// OrderID defines order identifiers.
	OrderID string `json:"orderId"`
	// CarrierID defines carrier identifiers.
	CarrierID string `json:"carrierId"`
	// TrackingCompany defines the carrier name to expose on Shopify fulfillments.
	TrackingCompany string `json:"trackingCompany"`
	// TrackingNumber defines tracking numbers.
	TrackingNumber string `json:"trackingNumber"`
}

// MarkVoidedPayload defines shipping mark-voided payload values.
type MarkVoidedPayload struct {
	// MarkID defines mark identifiers.
	MarkID string `json:"markId"`
	// OrderID defines order identifiers.
	OrderID string `json:"orderId"`
	// CarrierID defines carrier identifiers.
	CarrierID string `json:"carrierId"`
	// TrackingNumber defines tracking numbers.
	TrackingNumber string `json:"trackingNumber"`
}

// Service applies shipping mark events to Shopify fulfillments.
type Service struct {
	// destination defines Shopify fulfillment mutation behavior.
	destination shopifyport.ShopifyFulfillmentDestination
	// links defines Shopify sync-link persistence dependencies.
	links shopifyport.SyncLinkRepository
	// logger defines structured logging dependencies.
	logger *zap.Logger
}

// NewService creates Shopify fulfillment write-back services.
func NewService(destination shopifyport.ShopifyFulfillmentDestination, links shopifyport.SyncLinkRepository, providedLogger *zap.Logger) (*Service, error) {
	if destination == nil {
		return nil, ErrNilFulfillmentDestination
	}
	if links == nil {
		return nil, ErrNilSyncLinks
	}
	if providedLogger == nil {
		providedLogger = zap.NewNop()
	}
	return &Service{destination: destination, links: links, logger: providedLogger}, nil
}

// HandleMarkGenerated creates one Shopify fulfillment for a generated Mannaiah mark.
func (s *Service) HandleMarkGenerated(ctx context.Context, payload MarkGeneratedPayload) error {
	markID := strings.TrimSpace(payload.MarkID)
	orderID := strings.TrimSpace(payload.OrderID)
	if markID == "" || orderID == "" {
		return nil
	}
	existing, err := s.links.GetLinkByMannaiahID(ctx, shopifyport.SyncKindFulfillment, markID)
	if err != nil || existing != nil {
		return err
	}
	orderLink, err := s.links.GetLinkByMannaiahID(ctx, shopifyport.SyncKindOrder, orderID)
	if err != nil || orderLink == nil {
		return err
	}
	requestCtx := shopifyport.WithShopDomain(ctx, orderLink.ShopDomain)
	fulfillmentID, err := s.destination.FulfillOrder(requestCtx, shopifyport.ShopifyFulfillOrderInput{
		ShopifyOrderID:  strings.TrimSpace(orderLink.ShopifyID),
		TrackingNumber:  strings.TrimSpace(payload.TrackingNumber),
		TrackingCompany: resolveTrackingCompany(payload),
		NotifyCustomer:  false,
	})
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err = s.links.UpsertLink(requestCtx, shopifyport.UpsertSyncLinkInput{
		Kind:            shopifyport.SyncKindFulfillment,
		ShopDomain:      strings.TrimSpace(orderLink.ShopDomain),
		ShopifyID:       fulfillmentID,
		MannaiahID:      markID,
		LastKnownStatus: "CREATED",
		LastSyncedAt:    &now,
	})
	return err
}

// resolveTrackingCompany resolves the fulfillment carrier name persisted in Shopify.
func resolveTrackingCompany(payload MarkGeneratedPayload) string {
	if trackingCompany := strings.TrimSpace(payload.TrackingCompany); trackingCompany != "" {
		return trackingCompany
	}

	return strings.TrimSpace(payload.CarrierID)
}

// HandleMarkVoided cancels one Shopify fulfillment for a voided Mannaiah mark.
func (s *Service) HandleMarkVoided(ctx context.Context, payload MarkVoidedPayload) error {
	markID := strings.TrimSpace(payload.MarkID)
	if markID == "" {
		return nil
	}
	link, err := s.links.GetLinkByMannaiahID(ctx, shopifyport.SyncKindFulfillment, markID)
	if err != nil || link == nil {
		return err
	}
	requestCtx := shopifyport.WithShopDomain(ctx, link.ShopDomain)
	if err := s.destination.CancelFulfillment(requestCtx, link.ShopifyID); err != nil {
		return err
	}
	return s.links.UpdateLastKnownStatus(requestCtx, shopifyport.SyncKindFulfillment, markID, "CANCELLED")
}
