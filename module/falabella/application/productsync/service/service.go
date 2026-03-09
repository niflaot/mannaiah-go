package service

import (
	"context"
	"errors"
	"fmt"
	syncdomain "mannaiah/module/falabella/domain/sync"
	"mannaiah/module/falabella/port"
	"strings"
	"sync"
	"time"

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

const (
	defaultSyncWorkers = 4
	maxSyncWorkers     = 16
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
	// SyncWorkers defines max concurrent workers for batch product sync.
	SyncWorkers int
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
	// Feeds defines all feed submissions generated during one SKU sync (for example, product and image).
	Feeds []ResultFeed `json:"feeds,omitempty"`
	// Warnings defines Falabella WarningDetail messages from the sync response.
	Warnings []string `json:"warnings,omitempty"`
}

// ResultFeed defines one feed submission linked to a product sync result.
type ResultFeed struct {
	// Step defines the logical sync step that emitted this feed (for example, product or image).
	Step string `json:"step"`
	// Action defines Falabella request action values (for example, ProductCreate, ProductUpdate, Image).
	Action string `json:"action,omitempty"`
	// FeedID defines Falabella feed identifier values.
	FeedID string `json:"feedId"`
}

// Summary defines aggregate sync result values.
type Summary struct {
	// ExecutionID defines one unique identifier for this sync execution.
	ExecutionID string `json:"executionId"`
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
	if resolved.SyncWorkers <= 0 {
		resolved.SyncWorkers = defaultSyncWorkers
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
	summary.ExecutionID = newExecutionID()
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
	summary.ExecutionID = newExecutionID()
	workers := resolveSyncWorkerCount(s.cfg.SyncWorkers, len(products))
	if workers == 1 {
		for _, product := range products {
			s.syncOne(ctx, summary, product)
		}
		return summary, nil
	}

	type syncJob struct {
		index   int
		product port.CatalogProduct
	}
	type syncOutcome struct {
		index   int
		summary Summary
	}

	jobs := make(chan syncJob, len(products))
	outcomes := make(chan syncOutcome, len(products))

	var waitGroup sync.WaitGroup
	for workerIndex := 0; workerIndex < workers; workerIndex++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for job := range jobs {
				localSummary := Summary{ExecutionID: summary.ExecutionID, Results: make([]Result, 0, 1)}
				s.syncOne(ctx, &localSummary, job.product)
				outcomes <- syncOutcome{index: job.index, summary: localSummary}
			}
		}()
	}

	for index, product := range products {
		jobs <- syncJob{index: index, product: product}
	}
	close(jobs)

	waitGroup.Wait()
	close(outcomes)

	orderedOutcomes := make([]Summary, len(products))
	for outcome := range outcomes {
		orderedOutcomes[outcome.index] = outcome.summary
	}

	for _, outcome := range orderedOutcomes {
		summary.Requested += outcome.Requested
		summary.Synced += outcome.Synced
		summary.Skipped += outcome.Skipped
		summary.Failed += outcome.Failed
		summary.Results = append(summary.Results, outcome.Results...)
	}

	return summary, nil
}

// resolveSyncWorkerCount resolves bounded worker counts for batch sync execution.
func resolveSyncWorkerCount(configuredWorkers int, totalProducts int) int {
	if totalProducts <= 1 {
		return 1
	}

	resolved := configuredWorkers
	if resolved <= 0 {
		resolved = defaultSyncWorkers
	}
	if resolved > maxSyncWorkers {
		resolved = maxSyncWorkers
	}
	if resolved > totalProducts {
		resolved = totalProducts
	}
	if resolved <= 0 {
		return 1
	}

	return resolved
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
		productAction := syncActionFromResponse(actionResp)
		feedID := ""
		var warnings []string
		if actionResp != nil {
			feedID = actionResp.RequestID
			warnings = actionResp.WarningMessages()
		}

		if actionResp != nil && actionResp.HasRequiredFieldViolations() {
			result := Result{
				ProductID: product.ID,
				SKU:       request.SKU,
				Status:    "failed",
				Reason:    "required_field_warnings",
				FeedID:    feedID,
				Warnings:  warnings,
			}
			appendFeedResult(&result, "product", actionResp)
			summary.Failed++
			summary.Results = append(summary.Results, result)
			continue
		}

		variantImageURLs := resolveImageURLs(product.Images, s.cfg.Realm, variant.VariationIDs)
		imageActionResp, imageErr := s.syncImages(ctx, request.SKU, variantImageURLs)
		if imageErr != nil {
			result := Result{
				ProductID: product.ID,
				SKU:       request.SKU,
				Status:    "failed",
				Reason:    imageErr.Error(),
				FeedID:    feedID,
				Warnings:  warnings,
			}
			appendFeedResult(&result, "product", actionResp)
			summary.Failed++
			summary.Results = append(summary.Results, result)
			continue
		}

		s.recordSyncEntry(ctx, summary.ExecutionID, product.ID, request.SKU, feedID, variant.VariationIDs, syncdomain.SyncStepProduct, productAction)
		imageFeedID := ""
		imageAction := syncdomain.SyncActionCreate
		if imageActionResp != nil {
			imageFeedID = strings.TrimSpace(imageActionResp.RequestID)
			imageAction = syncActionFromResponse(imageActionResp)
		}
		s.recordSyncEntry(ctx, summary.ExecutionID, product.ID, request.SKU, imageFeedID, variant.VariationIDs, syncdomain.SyncStepImage, imageAction)

		result := Result{
			ProductID: product.ID,
			SKU:       request.SKU,
			Status:    "synced",
			FeedID:    feedID,
			Warnings:  warnings,
		}
		appendFeedResult(&result, "product", actionResp)
		appendFeedResult(&result, "image", imageActionResp)

		summary.Synced++
		summary.Results = append(summary.Results, result)
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
	productAction := syncActionFromResponse(actionResp)
	feedID := ""
	var warnings []string
	if actionResp != nil {
		feedID = actionResp.RequestID
		warnings = actionResp.WarningMessages()
	}

	if actionResp != nil && actionResp.HasRequiredFieldViolations() {
		result := Result{
			ProductID: productID,
			SKU:       request.SKU,
			Status:    "failed",
			Reason:    "required_field_warnings",
			FeedID:    feedID,
			Warnings:  warnings,
		}
		appendFeedResult(&result, "product", actionResp)
		summary.Failed++
		summary.Results = append(summary.Results, result)
		return
	}

	imageActionResp, imageErr := s.syncImages(ctx, request.SKU, imageURLs)
	if imageErr != nil {
		result := Result{
			ProductID: productID,
			SKU:       request.SKU,
			Status:    "failed",
			Reason:    imageErr.Error(),
			FeedID:    feedID,
			Warnings:  warnings,
		}
		appendFeedResult(&result, "product", actionResp)
		summary.Failed++
		summary.Results = append(summary.Results, result)
		return
	}

	s.recordSyncEntry(ctx, summary.ExecutionID, productID, request.SKU, feedID, nil, syncdomain.SyncStepProduct, productAction)
	imageFeedID := ""
	imageAction := syncdomain.SyncActionCreate
	if imageActionResp != nil {
		imageFeedID = strings.TrimSpace(imageActionResp.RequestID)
		imageAction = syncActionFromResponse(imageActionResp)
	}
	s.recordSyncEntry(ctx, summary.ExecutionID, productID, request.SKU, imageFeedID, nil, syncdomain.SyncStepImage, imageAction)

	result := Result{
		ProductID: productID,
		SKU:       request.SKU,
		Status:    "synced",
		FeedID:    feedID,
		Warnings:  warnings,
	}
	appendFeedResult(&result, "product", actionResp)
	appendFeedResult(&result, "image", imageActionResp)

	summary.Synced++
	summary.Results = append(summary.Results, result)
}

// syncImages syncs image URLs for a single SKU when URLs are provided.
func (s *ProductSyncService) syncImages(ctx context.Context, sku string, urls []string) (*syncdomain.ActionResponse, error) {
	normalized := uniqueTrimmedValues(urls)
	if len(normalized) == 0 {
		return nil, nil
	}

	response, err := s.source.SyncProductImages(ctx, port.SyncProductImagesRequest{SKU: strings.TrimSpace(sku), URLs: normalized})
	if err != nil {
		return nil, fmt.Errorf("sync falabella product images: %w", err)
	}

	return parseSyncResponse(response), nil
}

// appendFeedResult appends feed metadata into one sync result.
func appendFeedResult(result *Result, step string, actionResp *syncdomain.ActionResponse) {
	if result == nil || actionResp == nil {
		return
	}

	feedID := strings.TrimSpace(actionResp.RequestID)
	if feedID == "" {
		return
	}
	result.Feeds = append(result.Feeds, ResultFeed{
		Step:   strings.TrimSpace(step),
		Action: strings.TrimSpace(actionResp.RequestAction),
		FeedID: feedID,
	})
}

// syncActionFromResponse resolves persisted sync action values from Falabella action responses.
func syncActionFromResponse(actionResp *syncdomain.ActionResponse) syncdomain.SyncAction {
	if actionResp == nil {
		return syncdomain.SyncActionCreate
	}

	return actionResp.SyncAction()
}

// newExecutionID returns one best-effort unique identifier per sync execution.
func newExecutionID() string {
	return fmt.Sprintf("falabella-sync-%d", time.Now().UTC().UnixNano())
}
