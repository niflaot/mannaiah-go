package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	corecache "mannaiah/module/core/cache"
	"mannaiah/module/shipping/domain"
)

const (
	// defaultBatchManifestLogoURL defines the default logo URL rendered in batch manifest cover pages.
	defaultBatchManifestLogoURL = "https://storageapi.flockstore.co/fl-assets/assets/2308daa9-2a24-436c-bc92-ee00b8d19f35-flock.png"
	// defaultBatchManifestCacheTTL defines in-memory cache retention for generated merged PDFs.
	defaultBatchManifestCacheTTL = 5 * time.Minute
	// defaultBatchManifestHTTPTimeout defines outbound timeout values for logo/manifest downloads.
	defaultBatchManifestHTTPTimeout = 20 * time.Second
	// maxBatchManifestDownloadBytes defines the maximum download size for one manifest PDF payload.
	maxBatchManifestDownloadBytes = 20 * 1024 * 1024
	// defaultBatchManifestCacheKeyPrefix defines Redis cache-key prefixes for merged manifest documents.
	defaultBatchManifestCacheKeyPrefix = "shipping:batch_manifest_document:"
)

// BatchManifestOrderSummary defines order metadata rendered in one batch manifest cover row.
type BatchManifestOrderSummary struct {
	// OrderNumber defines the short/public order number displayed in cover rows.
	OrderNumber string
	// Items defines display item labels rendered in cover rows.
	Items []string
}

// BatchManifestOrderSummaryResolver defines order metadata lookup behavior for cover rows.
type BatchManifestOrderSummaryResolver interface {
	// ResolveBatchManifestOrderSummary resolves one order summary by order identifier.
	ResolveBatchManifestOrderSummary(ctx context.Context, orderID string) (*BatchManifestOrderSummary, error)
}

// batchManifestDocumentCacheEntry defines one cached merged manifest document value.
type batchManifestDocumentCacheEntry struct {
	// Body defines merged PDF payload bytes.
	Body []byte
	// ExpiresAt defines cache expiration timestamps.
	ExpiresAt time.Time
}

// batchManifestDocumentBuilder defines dependencies used by batch manifest document generation.
type batchManifestDocumentBuilder struct {
	// cacheMutex guards cache map mutations and reads.
	cacheMutex sync.Mutex
	// cache defines per-batch merged PDF cache values.
	cache map[string]batchManifestDocumentCacheEntry
	// cacheTTL defines cache expiration windows.
	cacheTTL time.Duration
	// cacheStore defines optional external cache dependencies (Redis).
	cacheStore corecache.Store
	// cacheKeyPrefix defines cache-key prefixes for external cache entries.
	cacheKeyPrefix string
	// logoURL defines cover-logo URL values.
	logoURL string
	// httpClient defines outbound HTTP client dependencies.
	httpClient *http.Client
	// orderSummaryResolver defines optional order summary lookup dependencies.
	orderSummaryResolver BatchManifestOrderSummaryResolver
	// coverTemplate defines visual strings and labels rendered in the summary cover.
	coverTemplate batchManifestCoverTemplate
}

// batchManifestCoverMeta defines batch metadata rendered in cover-page headers.
type batchManifestCoverMeta struct {
	// BatchID defines batch identifier values.
	BatchID string
	// CarrierID defines batch carrier identifier values.
	CarrierID string
	// GeneratedAt defines generation timestamps.
	GeneratedAt time.Time
	// Quantity defines mark-count values included in this document.
	Quantity int
}

// batchManifestCoverRow defines one row rendered in cover-page summary tables.
type batchManifestCoverRow struct {
	// TrackingNumber defines tracking/document fallback identifiers.
	TrackingNumber string
	// FreightCost defines the freight cost amount (excluding COD fees).
	FreightCost float64
	// RecipientName defines recipient display-name values.
	RecipientName string
	// OrderNumber defines short/public order-number values.
	OrderNumber string
	// City defines destination-city values.
	City string
	// Items defines row item-list values.
	Items []string
}

// newBatchManifestDocumentBuilder creates default batch manifest document builder dependencies.
func newBatchManifestDocumentBuilder() *batchManifestDocumentBuilder {
	return &batchManifestDocumentBuilder{
		cache:          map[string]batchManifestDocumentCacheEntry{},
		cacheTTL:       defaultBatchManifestCacheTTL,
		cacheKeyPrefix: defaultBatchManifestCacheKeyPrefix,
		logoURL:        defaultBatchManifestLogoURL,
		httpClient:     &http.Client{Timeout: defaultBatchManifestHTTPTimeout},
		coverTemplate:  loadDefaultBatchManifestCoverTemplate(),
	}
}

// ManifestDocument builds one merged batch manifest PDF (cover page + all manifest PDFs).
func (s *Service) ManifestDocument(ctx context.Context, batchID string) ([]byte, error) {
	if s == nil || s.batchRepository == nil || s.markRepository == nil {
		return nil, domain.ErrInvalidID
	}
	trimmedBatchID := strings.TrimSpace(batchID)
	if trimmedBatchID == "" {
		return nil, domain.ErrInvalidID
	}
	if payload, ok := s.getCachedBatchManifestDocument(ctx, trimmedBatchID); ok {
		return payload, nil
	}

	batch, err := s.batchRepository.GetByID(ctx, trimmedBatchID)
	if err != nil {
		return nil, err
	}
	if batch.Status != domain.BatchStatusClosed {
		return nil, domain.ErrInvalidBatchStatus
	}

	marks, err := s.markRepository.ListByBatchID(ctx, trimmedBatchID)
	if err != nil {
		return nil, err
	}
	rows, _ := s.resolveBatchManifestCoverRows(ctx, marks)
	cover, err := s.buildBatchManifestCoverPDF(ctx, batchManifestCoverMeta{
		BatchID:     batch.ID,
		CarrierID:   batch.CarrierID,
		GeneratedAt: time.Now().UTC(),
		Quantity:    len(rows),
	}, rows)
	if err != nil {
		return nil, err
	}

	s.cacheBatchManifestDocument(ctx, trimmedBatchID, cover)

	return append([]byte(nil), cover...), nil
}

// SetBatchManifestOrderSummaryResolver configures optional order summary lookup dependencies.
func (s *Service) SetBatchManifestOrderSummaryResolver(resolver BatchManifestOrderSummaryResolver) {
	if s == nil || s.manifestDocuments == nil {
		return
	}
	m := s.manifestDocuments
	m.orderSummaryResolver = resolver
}

// SetBatchManifestDocumentLogoURL configures optional cover-logo URL values.
func (s *Service) SetBatchManifestDocumentLogoURL(value string) {
	if s == nil || s.manifestDocuments == nil {
		return
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		s.manifestDocuments.logoURL = defaultBatchManifestLogoURL
		return
	}
	s.manifestDocuments.logoURL = trimmed
}

// SetBatchManifestDocumentCacheTTL configures merged-document cache TTL values.
func (s *Service) SetBatchManifestDocumentCacheTTL(ttl time.Duration) {
	if s == nil || s.manifestDocuments == nil {
		return
	}
	if ttl <= 0 {
		s.manifestDocuments.cacheTTL = defaultBatchManifestCacheTTL
		return
	}
	s.manifestDocuments.cacheTTL = ttl
}

// SetBatchManifestDocumentCacheStore configures external cache dependencies used by merged-document cache.
func (s *Service) SetBatchManifestDocumentCacheStore(store corecache.Store) {
	if s == nil || s.manifestDocuments == nil {
		return
	}
	s.manifestDocuments.cacheStore = store
}

// SetBatchManifestDocumentHTTPClient configures outbound HTTP client dependencies used by logo/manifest downloads.
func (s *Service) SetBatchManifestDocumentHTTPClient(client *http.Client) {
	if s == nil || s.manifestDocuments == nil || client == nil {
		return
	}
	s.manifestDocuments.httpClient = client
}

// invalidateBatchManifestDocumentCache removes one cached merged-document entry.
func (s *Service) invalidateBatchManifestDocumentCache(ctx context.Context, batchID string) {
	if s == nil || s.manifestDocuments == nil {
		return
	}
	trimmedBatchID := strings.TrimSpace(batchID)
	if trimmedBatchID == "" {
		return
	}
	m := s.manifestDocuments
	m.cacheMutex.Lock()
	delete(m.cache, trimmedBatchID)
	m.cacheMutex.Unlock()
	if m.cacheStore != nil {
		if _, err := m.cacheStore.Delete(ctx, m.batchManifestCacheKey(trimmedBatchID)); err != nil {
			zap.L().Warn("batch manifest cache delete failed", zap.String("batch_id", trimmedBatchID), zap.Error(err))
		}
	}
}

// getCachedBatchManifestDocument resolves one cached merged-document payload when not expired.
func (s *Service) getCachedBatchManifestDocument(ctx context.Context, batchID string) ([]byte, bool) {
	if s == nil || s.manifestDocuments == nil {
		return nil, false
	}
	m := s.manifestDocuments
	if m.cacheStore != nil {
		cachedBase64, err := m.cacheStore.Get(ctx, m.batchManifestCacheKey(batchID))
		if err == nil {
			payload, decodeErr := base64.StdEncoding.DecodeString(strings.TrimSpace(cachedBase64))
			if decodeErr == nil && len(payload) > 0 {
				return payload, true
			}
		}
	}
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	entry, exists := m.cache[batchID]
	if !exists {
		return nil, false
	}
	if time.Now().UTC().After(entry.ExpiresAt) {
		delete(m.cache, batchID)
		return nil, false
	}
	return append([]byte(nil), entry.Body...), true
}

// cacheBatchManifestDocument stores one merged-document payload with cache TTL.
func (s *Service) cacheBatchManifestDocument(ctx context.Context, batchID string, payload []byte) {
	if s == nil || s.manifestDocuments == nil {
		return
	}
	if len(payload) == 0 {
		return
	}
	m := s.manifestDocuments
	if m.cacheStore != nil {
		if err := m.cacheStore.Set(ctx, m.batchManifestCacheKey(batchID), base64.StdEncoding.EncodeToString(payload), m.cacheTTL); err != nil {
			zap.L().Warn("batch manifest cache set failed", zap.String("batch_id", batchID), zap.Error(err))
		}
	}
	m.cacheMutex.Lock()
	m.cache[batchID] = batchManifestDocumentCacheEntry{
		Body:      append([]byte(nil), payload...),
		ExpiresAt: time.Now().UTC().Add(m.cacheTTL),
	}
	m.cacheMutex.Unlock()
}

// resolveBatchManifestCoverRows resolves cover-table rows and unique manifest URLs from batch marks.
func (s *Service) resolveBatchManifestCoverRows(ctx context.Context, marks []domain.ShippingMark) ([]batchManifestCoverRow, []string) {
	rows := make([]batchManifestCoverRow, 0, len(marks))
	manifestURLSet := map[string]struct{}{}
	manifestURLs := make([]string, 0, len(marks))

	for _, mark := range marks {
		if !isBatchManifestMarkIncluded(mark) {
			continue
		}
		orderNumber, items := s.resolveBatchManifestOrderSummary(ctx, mark)
		rows = append(rows, batchManifestCoverRow{
			TrackingNumber: firstNonEmpty(strings.TrimSpace(mark.TrackingNumber), strings.TrimSpace(mark.DocumentRef), strings.TrimSpace(mark.ID)),
			FreightCost:    mark.QuotedFreightCost,
			RecipientName:  firstNonEmpty(strings.TrimSpace(mark.Recipient.Name), strings.TrimSpace(mark.Recipient.LegalName), "-"),
			OrderNumber:    orderNumber,
			City:           firstNonEmpty(strings.TrimSpace(mark.Recipient.CityCode), "-"),
			Items:          items,
		})
		manifestURL := strings.TrimSpace(mark.ManifestRef)
		if mark.ManifestType != domain.MarkDocumentLink || manifestURL == "" {
			continue
		}
		if _, exists := manifestURLSet[manifestURL]; exists {
			continue
		}
		manifestURLSet[manifestURL] = struct{}{}
		manifestURLs = append(manifestURLs, manifestURL)
	}

	sort.Strings(manifestURLs)

	return rows, manifestURLs
}

// isBatchManifestMarkIncluded reports whether one mark should be rendered in merged batch manifest documents.
func isBatchManifestMarkIncluded(mark domain.ShippingMark) bool {
	return mark.Status != domain.MarkStatusFailed
}

// resolveBatchManifestOrderSummary resolves one row order-number and item labels with resolver fallback behavior.
func (s *Service) resolveBatchManifestOrderSummary(ctx context.Context, mark domain.ShippingMark) (string, []string) {
	fallbackOrderNumber := fallbackBatchManifestOrderNumber(mark.OrderID)
	fallbackItems := fallbackBatchManifestItems(mark)
	if s == nil || s.manifestDocuments == nil || s.manifestDocuments.orderSummaryResolver == nil {
		return fallbackOrderNumber, fallbackItems
	}
	summary, err := s.manifestDocuments.orderSummaryResolver.ResolveBatchManifestOrderSummary(ctx, strings.TrimSpace(mark.OrderID))
	if err != nil || summary == nil {
		return fallbackOrderNumber, fallbackItems
	}
	orderNumber := firstNonEmpty(strings.TrimSpace(summary.OrderNumber), fallbackOrderNumber)
	items := normalizeBatchManifestItems(summary.Items)
	if len(items) == 0 {
		items = fallbackItems
	}
	return orderNumber, items
}

// fallbackBatchManifestOrderNumber resolves fallback order-number values from order identifiers.
func fallbackBatchManifestOrderNumber(orderID string) string {
	trimmed := strings.TrimSpace(orderID)
	if trimmed == "" {
		return "-"
	}
	if len(trimmed) <= 12 {
		return trimmed
	}
	return trimmed[:12]
}

// normalizeBatchManifestItems normalizes item labels and removes empty values.
func normalizeBatchManifestItems(items []string) []string {
	rows := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		rows = append(rows, trimmed)
	}
	if len(rows) == 0 {
		return []string{"-"}
	}
	return rows
}

// fallbackBatchManifestItems resolves fallback row-item labels from mark units.
func fallbackBatchManifestItems(mark domain.ShippingMark) []string {
	rows := make([]string, 0, len(mark.Units))
	for _, unit := range mark.Units {
		if trimmed := strings.TrimSpace(unit.Description); trimmed != "" {
			rows = append(rows, trimmed)
		}
	}
	return normalizeBatchManifestItems(rows)
}

// batchManifestCacheKey resolves normalized external cache keys for one batch identifier.
func (m *batchManifestDocumentBuilder) batchManifestCacheKey(batchID string) string {
	if m == nil {
		return strings.TrimSpace(batchID)
	}
	return strings.TrimSpace(m.cacheKeyPrefix) + strings.TrimSpace(batchID)
}

// downloadManifestPDF fetches one manifest PDF payload from an external URL.
func (s *Service) downloadManifestPDF(ctx context.Context, rawURL string) ([]byte, error) {
	if s == nil || s.manifestDocuments == nil || s.manifestDocuments.httpClient == nil {
		return nil, domain.ErrInvalidID
	}
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https") {
		return nil, domain.ErrInvalidID
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := s.manifestDocuments.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, domain.ErrNotFound
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBatchManifestDownloadBytes))
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, domain.ErrNotFound
	}
	if !bytes.HasPrefix(body, []byte("%PDF")) {
		return nil, domain.ErrInvalidID
	}
	return body, nil
}

// firstNonEmpty returns the first non-empty trimmed input value.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
