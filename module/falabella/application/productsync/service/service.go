package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"mannaiah/module/falabella/port"
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

	return &ProductSyncService{source: source, catalog: catalog, cfg: resolved}, nil
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

	if err := s.ValidateIntegration(ctx); err != nil {
		return nil, err
	}

	product, err := s.catalog.GetProduct(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("load product for falabella sync: %w", err)
	}

	summary := &Summary{Requested: 1, Results: make([]Result, 0, 1)}
	s.syncOne(ctx, summary, *product)
	return summary, nil
}

// SyncProducts syncs provided products or all products when ids are empty.
func (s *ProductSyncService) SyncProducts(ctx context.Context, ids []string) (*Summary, error) {
	if err := s.ValidateIntegration(ctx); err != nil {
		return nil, err
	}

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

	summary := &Summary{Requested: len(products), Results: make([]Result, 0, len(products))}
	for _, product := range products {
		s.syncOne(ctx, summary, product)
	}

	return summary, nil
}

// syncOne synchronizes a single product and appends the result to the aggregate summary.
func (s *ProductSyncService) syncOne(ctx context.Context, summary *Summary, product port.CatalogProduct) {
	request, skipReason, err := mapProduct(product, s.cfg)
	if err != nil {
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
		summary.Skipped++
		summary.Results = append(summary.Results, Result{
			ProductID: product.ID,
			SKU:       product.SKU,
			Status:    "skipped",
			Reason:    skipReason,
		})
		return
	}

	if _, syncErr := s.source.SyncProduct(ctx, request); syncErr != nil {
		summary.Failed++
		summary.Results = append(summary.Results, Result{
			ProductID: product.ID,
			SKU:       product.SKU,
			Status:    "failed",
			Reason:    syncErr.Error(),
		})
		return
	}

	summary.Synced++
	summary.Results = append(summary.Results, Result{
		ProductID: product.ID,
		SKU:       product.SKU,
		Status:    "synced",
	})
}

