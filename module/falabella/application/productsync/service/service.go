package service

import (
	"context"
	"errors"
	"fmt"
	"mannaiah/module/falabella/port"
	"strings"

	"go.uber.org/zap"
)

var (
	// ErrNilSource is returned when Falabella source dependencies are nil.
	ErrNilSource = errors.New("falabella source must not be nil")
	// ErrNilCatalog is returned when product-catalog dependencies are nil.
	ErrNilCatalog = errors.New("falabella product catalog must not be nil")
	// ErrInvalidProductID is returned when product IDs are empty.
	ErrInvalidProductID = errors.New("falabella sync product id is required")
	// ErrIntegrationUnavailable is returned when Falabella integration is unavailable.
	ErrIntegrationUnavailable = errors.New("falabella integration is unavailable")
)

// Source defines Falabella product-sync source behavior required by this service.
type Source interface {
	// Validate verifies Falabella integration availability.
	Validate(ctx context.Context) error
	// SyncProduct upserts product payloads into Falabella.
	SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error)
	// SyncProductImages configures image URLs for a product SKU.
	SyncProductImages(ctx context.Context, request port.SyncProductImagesRequest) ([]byte, error)
}

// ProductCatalog defines product lookup behavior required by Falabella sync services.
type ProductCatalog interface {
	// GetProduct retrieves products by identifier.
	GetProduct(ctx context.Context, id string) (*port.CatalogProduct, error)
	// ListProducts retrieves all products.
	ListProducts(ctx context.Context) ([]port.CatalogProduct, error)
}

// Config defines product-sync mapping configuration values.
type Config struct {
	// Realm defines source datasheet realms to sync.
	Realm string
	// CategoryID defines Falabella primary-category identifier values.
	CategoryID string
	// GlobalIdentifier defines Falabella global-identifier values.
	GlobalIdentifier string
	// AttributeSetID defines Falabella attribute-set identifier values.
	AttributeSetID string
	// OperatorCode defines Falabella business-unit operator-code values.
	OperatorCode string
}

// Result defines per-product sync result values.
type Result struct {
	// ProductID defines source product identifier values.
	ProductID string `json:"productId"`
	// SKU defines source product SKU values.
	SKU string `json:"sku"`
	// Status defines sync result status values.
	Status string `json:"status"`
	// Reason defines sync result reason values.
	Reason string `json:"reason,omitempty"`
	// FeedID defines Falabella feed identifier returned on async submission.
	FeedID string `json:"feedId,omitempty"`
	// Warnings defines Falabella WarningDetail messages from the sync response.
	Warnings []string `json:"warnings,omitempty"`
}

// Summary defines aggregate sync result values.
type Summary struct {
	// Requested defines requested product count values.
	Requested int `json:"requested"`
	// Synced defines successful sync count values.
	Synced int `json:"synced"`
	// Skipped defines skipped sync count values.
	Skipped int `json:"skipped"`
	// Failed defines failed sync count values.
	Failed int `json:"failed"`
	// Results defines per-product sync results.
	Results []Result `json:"results"`
}

// Service defines product-sync use-case behavior.
type Service interface {
	// ValidateIntegration verifies Falabella integration availability.
	ValidateIntegration(ctx context.Context) error
	// SyncProduct syncs one product by identifier.
	SyncProduct(ctx context.Context, id string) (*Summary, error)
	// SyncProducts syncs a list of product IDs or all products when ids are empty.
	SyncProducts(ctx context.Context, ids []string) (*Summary, error)
}

// ProductSyncService implements product-sync use cases.
type ProductSyncService struct {
	// source defines Falabella source dependencies.
	source Source
	// catalog defines product-catalog lookup dependencies.
	catalog ProductCatalog
	// cfg defines mapping configuration values.
	cfg Config
	// recorder defines optional sync status recording dependencies.
	recorder SyncStatusRecorder
	// logger defines structured logging dependencies.
	logger *zap.Logger
}

var (
	// _ ensures ProductSyncService satisfies sync-service contracts.
	_ Service = (*ProductSyncService)(nil)
)

// NewService creates Falabella product-sync services.
func NewService(source Source, catalog ProductCatalog, cfg Config) (*ProductSyncService, error) {
	if source == nil {
		return nil, ErrNilSource
	}
	if catalog == nil {
		return nil, ErrNilCatalog
	}

	resolved := cfg
	if strings.TrimSpace(resolved.Realm) == "" {
		resolved.Realm = "falabella"
	}
	if strings.TrimSpace(resolved.OperatorCode) == "" {
		resolved.OperatorCode = "FACO"
	}

	return &ProductSyncService{source: source, catalog: catalog, cfg: resolved, logger: zap.NewNop()}, nil
}

// ValidateIntegration verifies Falabella integration availability.
func (s *ProductSyncService) ValidateIntegration(ctx context.Context) error {
	if err := s.source.Validate(ctx); err != nil {
		return fmt.Errorf("%w: %w", ErrIntegrationUnavailable, err)
	}

	return nil
}

// SyncProduct syncs one product by identifier.
func (s *ProductSyncService) SyncProduct(ctx context.Context, id string) (*Summary, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, ErrInvalidProductID
	}

	product, err := s.catalog.GetProduct(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("load product for falabella sync: %w", err)
	}

	summary := &Summary{Results: make([]Result, 0, 1)}
	s.syncOne(ctx, summary, *product)
	return summary, nil
}

// SyncProducts syncs provided products or all products when ids are empty.
func (s *ProductSyncService) SyncProducts(ctx context.Context, ids []string) (*Summary, error) {
	var (
		products []port.CatalogProduct
		err      error
	)
	if len(ids) == 0 {
		products, err = s.catalog.ListProducts(ctx)
		if err != nil {
			return nil, fmt.Errorf("list products for falabella sync: %w", err)
		}
	} else {
		products = make([]port.CatalogProduct, 0, len(ids))
		for _, id := range ids {
			trimmedID := strings.TrimSpace(id)
			if trimmedID == "" {
				return nil, ErrInvalidProductID
			}
			product, getErr := s.catalog.GetProduct(ctx, trimmedID)
			if getErr != nil {
				return nil, fmt.Errorf("load product %q for falabella sync: %w", trimmedID, getErr)
			}
			products = append(products, *product)
		}
	}

	summary := &Summary{Results: make([]Result, 0, len(products))}
	for _, product := range products {
		s.syncOne(ctx, summary, product)
	}

	return summary, nil
}

// syncOne synchronizes a single product and appends the result to the aggregate summary.
func (s *ProductSyncService) syncOne(ctx context.Context, summary *Summary, product port.CatalogProduct) {
	baseRequest, skipReason, err := mapProduct(product, s.cfg)
	if err != nil {
		summary.Requested++
		summary.Failed++
		summary.Results = append(summary.Results, Result{
			ProductID: product.ID,
			SKU:       product.SKU,
			Status:    "failed",
			Reason:    err.Error(),
		})
		return
	}
	if skipReason != "" {
		summary.Requested++
		summary.Skipped++
		summary.Results = append(summary.Results, Result{
			ProductID: product.ID,
			SKU:       product.SKU,
			Status:    "skipped",
			Reason:    skipReason,
		})
		return
	}

	if len(product.Variants) == 0 {
		s.syncBaseProduct(ctx, summary, product.ID, baseRequest, resolveImageURLs(product.Images, s.cfg.Realm, nil))
		return
	}

	knownVariantSKUs := normalizeVariantSKUs(product.Variants)
	seenVariantSKUs := map[string]struct{}{}
	for _, variant := range product.Variants {
		request, mapErr := mapVariantProduct(baseRequest, variant, knownVariantSKUs)
		summary.Requested++
		if mapErr != nil {
			summary.Failed++
			summary.Results = append(summary.Results, Result{
				ProductID: product.ID,
				SKU:       strings.TrimSpace(variant.SKU),
				Status:    "failed",
				Reason:    mapErr.Error(),
			})
			continue
		}
		if _, seen := seenVariantSKUs[request.SKU]; seen {
			summary.Skipped++
			summary.Results = append(summary.Results, Result{
				ProductID: product.ID,
				SKU:       request.SKU,
				Status:    "skipped",
				Reason:    "duplicate_variant_sku",
			})
			continue
		}
		seenVariantSKUs[request.SKU] = struct{}{}

		syncResponse, syncErr := s.source.SyncProduct(ctx, request)
		if syncErr != nil {
			summary.Failed++
			summary.Results = append(summary.Results, Result{
				ProductID: product.ID,
				SKU:       request.SKU,
				Status:    "failed",
				Reason:    syncErr.Error(),
			})
			continue
		}

		actionResp := parseSyncResponse(syncResponse)
		feedID := ""
		var warnings []string
		if actionResp != nil {
			feedID = actionResp.RequestID
			warnings = actionResp.WarningMessages()
		}

		if actionResp != nil && actionResp.HasRequiredFieldViolations() {
			summary.Failed++
			summary.Results = append(summary.Results, Result{
				ProductID: product.ID,
				SKU:       request.SKU,
				Status:    "failed",
				Reason:    "required_field_warnings",
				FeedID:    feedID,
				Warnings:  warnings,
			})
			continue
		}

		variantImageURLs := resolveImageURLs(product.Images, s.cfg.Realm, variant.VariationIDs)
		if imageErr := s.syncImages(ctx, request.SKU, variantImageURLs); imageErr != nil {
			summary.Failed++
			summary.Results = append(summary.Results, Result{
				ProductID: product.ID,
				SKU:       request.SKU,
				Status:    "failed",
				Reason:    imageErr.Error(),
				FeedID:    feedID,
				Warnings:  warnings,
			})
			continue
		}

		s.recordSyncEntry(ctx, product.ID, request.SKU, feedID, actionResp)

		summary.Synced++
		summary.Results = append(summary.Results, Result{
			ProductID: product.ID,
			SKU:       request.SKU,
			Status:    "synced",
			FeedID:    feedID,
			Warnings:  warnings,
		})
	}
}

// normalizeVariantSKUs resolves normalized variant-SKU sets for scoped-attribute matching.
func normalizeVariantSKUs(variants []port.CatalogVariant) map[string]struct{} {
	if len(variants) == 0 {
		return map[string]struct{}{}
	}

	normalized := make(map[string]struct{}, len(variants))
	for _, variant := range variants {
		token := normalizeScopedVariantToken(variant.SKU)
		if token == "" {
			continue
		}
		normalized[token] = struct{}{}
	}

	return normalized
}

// syncBaseProduct synchronizes one non-variant product SKU and optional image URLs.
func (s *ProductSyncService) syncBaseProduct(ctx context.Context, summary *Summary, productID string, request port.SyncProductRequest, imageURLs []string) {
	summary.Requested++
	syncResponse, syncErr := s.source.SyncProduct(ctx, request)
	if syncErr != nil {
		summary.Failed++
		summary.Results = append(summary.Results, Result{
			ProductID: productID,
			SKU:       request.SKU,
			Status:    "failed",
			Reason:    syncErr.Error(),
		})
		return
	}

	actionResp := parseSyncResponse(syncResponse)
	feedID := ""
	var warnings []string
	if actionResp != nil {
		feedID = actionResp.RequestID
		warnings = actionResp.WarningMessages()
	}

	if actionResp != nil && actionResp.HasRequiredFieldViolations() {
		summary.Failed++
		summary.Results = append(summary.Results, Result{
			ProductID: productID,
			SKU:       request.SKU,
			Status:    "failed",
			Reason:    "required_field_warnings",
			FeedID:    feedID,
			Warnings:  warnings,
		})
		return
	}

	if imageErr := s.syncImages(ctx, request.SKU, imageURLs); imageErr != nil {
		summary.Failed++
		summary.Results = append(summary.Results, Result{
			ProductID: productID,
			SKU:       request.SKU,
			Status:    "failed",
			Reason:    imageErr.Error(),
			FeedID:    feedID,
			Warnings:  warnings,
		})
		return
	}

	s.recordSyncEntry(ctx, productID, request.SKU, feedID, actionResp)

	summary.Synced++
	summary.Results = append(summary.Results, Result{
		ProductID: productID,
		SKU:       request.SKU,
		Status:    "synced",
		FeedID:    feedID,
		Warnings:  warnings,
	})
}

// syncImages syncs image URLs for a single SKU when URLs are provided.
func (s *ProductSyncService) syncImages(ctx context.Context, sku string, urls []string) error {
	normalized := uniqueTrimmedValues(urls)
	if len(normalized) == 0 {
		return nil
	}

	if _, err := s.source.SyncProductImages(ctx, port.SyncProductImagesRequest{SKU: strings.TrimSpace(sku), URLs: normalized}); err != nil {
		return fmt.Errorf("sync falabella product images: %w", err)
	}

	return nil
}
