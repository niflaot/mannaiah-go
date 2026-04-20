package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	pdfcpuapi "github.com/pdfcpu/pdfcpu/pkg/api"

	corecache "mannaiah/module/core/cache"
	"mannaiah/module/shipping/domain"
)

const (
	// defaultBatchAllMarksCacheTTL defines in-memory cache retention for merged mark PDFs.
	defaultBatchAllMarksCacheTTL = 5 * time.Minute
	// defaultBatchAllMarksCacheKeyPrefix defines Redis cache-key prefix for merged mark documents.
	defaultBatchAllMarksCacheKeyPrefix = "shipping:batch_all_marks_document:"
	// maxMarkDocumentDownloadBytes defines maximum download size for one mark document PDF.
	maxMarkDocumentDownloadBytes = 20 * 1024 * 1024
	// defaultBatchAllMarksHTTPTimeout defines HTTP timeout for mark document downloads.
	defaultBatchAllMarksHTTPTimeout = 20 * time.Second
)

// batchAllMarksCacheEntry defines one cached batch-marks merged document value.
type batchAllMarksCacheEntry struct {
	// Body defines merged PDF payload bytes.
	Body []byte
	// ExpiresAt defines cache expiration timestamps.
	ExpiresAt time.Time
}

// batchAllMarksDocumentBuilder defines dependencies used by batch all-marks document generation.
type batchAllMarksDocumentBuilder struct {
	// cacheMutex guards in-memory cache reads and writes.
	cacheMutex sync.Mutex
	// cache defines per-batch cached merged PDF payloads.
	cache map[string]batchAllMarksCacheEntry
	// cacheTTL defines cache expiration windows.
	cacheTTL time.Duration
	// cacheStore defines optional external cache dependencies (Redis).
	cacheStore corecache.Store
	// cacheKeyPrefix defines external cache-key prefixes.
	cacheKeyPrefix string
	// httpClient defines outbound HTTP client dependencies.
	httpClient *http.Client
}

// newBatchAllMarksDocumentBuilder creates default batch all-marks document builder dependencies.
func newBatchAllMarksDocumentBuilder() *batchAllMarksDocumentBuilder {
	return &batchAllMarksDocumentBuilder{
		cache:          map[string]batchAllMarksCacheEntry{},
		cacheTTL:       defaultBatchAllMarksCacheTTL,
		cacheKeyPrefix: defaultBatchAllMarksCacheKeyPrefix,
		httpClient:     &http.Client{Timeout: defaultBatchAllMarksHTTPTimeout},
	}
}

// BatchAllMarksDocument downloads and merges all shipping label PDFs for marks in a batch into one PDF.
func (s *Service) BatchAllMarksDocument(ctx context.Context, batchID string) ([]byte, error) {
	if s == nil || s.repository == nil || s.batchMarksDocuments == nil {
		return nil, domain.ErrInvalidID
	}
	trimmedBatchID := strings.TrimSpace(batchID)
	if trimmedBatchID == "" {
		return nil, domain.ErrInvalidID
	}

	if payload, ok := s.getCachedBatchAllMarksDocument(ctx, trimmedBatchID); ok {
		return payload, nil
	}

	marks, err := s.repository.ListByBatchID(ctx, trimmedBatchID)
	if err != nil {
		return nil, err
	}

	var readers []io.ReadSeeker
	for _, mark := range marks {
		if mark.Status == domain.MarkStatusFailed {
			continue
		}
		docRef := strings.TrimSpace(mark.DocumentRef)
		if mark.DocumentType != domain.MarkDocumentLink || docRef == "" {
			continue
		}
		pdfBytes, dlErr := s.downloadMarkDocumentPDF(ctx, docRef)
		if dlErr != nil {
			continue
		}
		readers = append(readers, bytes.NewReader(pdfBytes))
	}

	if len(readers) == 0 {
		return nil, domain.ErrNotFound
	}

	var out bytes.Buffer
	if mergeErr := pdfcpuapi.MergeRaw(readers, &out, false, nil); mergeErr != nil {
		return nil, mergeErr
	}

	payload := out.Bytes()
	s.cacheBatchAllMarksDocument(ctx, trimmedBatchID, payload)

	return append([]byte(nil), payload...), nil
}

// SetBatchAllMarksDocumentCacheStore configures external cache dependencies used by batch marks cache.
func (s *Service) SetBatchAllMarksDocumentCacheStore(store corecache.Store) {
	if s == nil || s.batchMarksDocuments == nil {
		return
	}
	s.batchMarksDocuments.cacheStore = store
}

// SetBatchAllMarksDocumentCacheTTL configures batch marks cache TTL values.
func (s *Service) SetBatchAllMarksDocumentCacheTTL(ttl time.Duration) {
	if s == nil || s.batchMarksDocuments == nil {
		return
	}
	if ttl <= 0 {
		s.batchMarksDocuments.cacheTTL = defaultBatchAllMarksCacheTTL
		return
	}
	s.batchMarksDocuments.cacheTTL = ttl
}

// downloadMarkDocumentPDF fetches one shipping mark document PDF from an external URL.
func (s *Service) downloadMarkDocumentPDF(ctx context.Context, rawURL string) ([]byte, error) {
	if s == nil || s.batchMarksDocuments == nil || s.batchMarksDocuments.httpClient == nil {
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
	response, err := s.batchMarksDocuments.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, domain.ErrNotFound
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxMarkDocumentDownloadBytes))
	if err != nil {
		return nil, err
	}
	if len(body) == 0 || !bytes.HasPrefix(body, []byte("%PDF")) {
		return nil, domain.ErrInvalidID
	}
	return body, nil
}

// getCachedBatchAllMarksDocument resolves one cached merged-marks payload when not expired.
func (s *Service) getCachedBatchAllMarksDocument(ctx context.Context, batchID string) ([]byte, bool) {
	if s == nil || s.batchMarksDocuments == nil {
		return nil, false
	}
	b := s.batchMarksDocuments
	if b.cacheStore != nil {
		cachedBase64, err := b.cacheStore.Get(ctx, b.batchAllMarksCacheKey(batchID))
		if err == nil {
			payload, decodeErr := base64.StdEncoding.DecodeString(strings.TrimSpace(cachedBase64))
			if decodeErr == nil && len(payload) > 0 {
				return payload, true
			}
		}
	}
	b.cacheMutex.Lock()
	defer b.cacheMutex.Unlock()
	entry, exists := b.cache[batchID]
	if !exists {
		return nil, false
	}
	if time.Now().UTC().After(entry.ExpiresAt) {
		delete(b.cache, batchID)
		return nil, false
	}
	return append([]byte(nil), entry.Body...), true
}

// cacheBatchAllMarksDocument stores one merged-marks payload with cache TTL.
func (s *Service) cacheBatchAllMarksDocument(ctx context.Context, batchID string, payload []byte) {
	if s == nil || s.batchMarksDocuments == nil || len(payload) == 0 {
		return
	}
	b := s.batchMarksDocuments
	b.cacheMutex.Lock()
	b.cache[batchID] = batchAllMarksCacheEntry{
		Body:      append([]byte(nil), payload...),
		ExpiresAt: time.Now().UTC().Add(b.cacheTTL),
	}
	b.cacheMutex.Unlock()
	if b.cacheStore != nil {
		_ = b.cacheStore.Set(ctx, b.batchAllMarksCacheKey(batchID), base64.StdEncoding.EncodeToString(payload), b.cacheTTL)
	}
}

// batchAllMarksCacheKey resolves the external-cache key for one batch identifier.
func (b *batchAllMarksDocumentBuilder) batchAllMarksCacheKey(batchID string) string {
	if b == nil {
		return defaultBatchAllMarksCacheKeyPrefix + batchID
	}
	return firstNonEmptyString(strings.TrimSpace(b.cacheKeyPrefix), defaultBatchAllMarksCacheKeyPrefix) + strings.TrimSpace(batchID)
}
