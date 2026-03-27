package service

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jung-kurt/gofpdf"

	corecache "mannaiah/module/core/cache"
	"mannaiah/module/shipping/domain"
)

// batchManifestOrderSummaryResolverStub defines order summary resolver behavior for manifest-document tests.
type batchManifestOrderSummaryResolverStub struct {
	// summaries defines order summary fixtures by order id.
	summaries map[string]BatchManifestOrderSummary
}

// batchManifestCacheStoreStub defines in-memory cache-store behavior for redis-cache integration tests.
type batchManifestCacheStoreStub struct {
	// mu guards map/counter access.
	mu sync.Mutex
	// values defines cached payload values.
	values map[string]string
	// getCalls counts cache get calls.
	getCalls int
	// setCalls counts cache set calls.
	setCalls int
}

// Ping is a no-op for tests.
func (s *batchManifestCacheStoreStub) Ping(ctx context.Context) error { return nil }

// Get resolves one cache value by key.
func (s *batchManifestCacheStoreStub) Get(ctx context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCalls++
	value, exists := s.values[key]
	if !exists {
		return "", errors.New("not found")
	}

	return value, nil
}

// Set stores one cache value by key.
func (s *batchManifestCacheStoreStub) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[key] = value
	s.setCalls++

	return nil
}

// Delete removes one cache key.
func (s *batchManifestCacheStoreStub) Delete(ctx context.Context, key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.values[key]; !exists {
		return 0, nil
	}
	delete(s.values, key)

	return 1, nil
}

// Keys resolves cache keys matching one suffix-wildcard pattern.
func (s *batchManifestCacheStoreStub) Keys(ctx context.Context, pattern string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.values))
	prefix := strings.TrimSuffix(pattern, "*")
	for key := range s.values {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	return keys, nil
}

// GetByPattern resolves key-value entries matching one suffix-wildcard pattern.
func (s *batchManifestCacheStoreStub) GetByPattern(ctx context.Context, pattern string) (map[string]string, error) {
	keys, err := s.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make(map[string]string, len(keys))
	for _, key := range keys {
		rows[key] = s.values[key]
	}

	return rows, nil
}

// Close is a no-op for tests.
func (s *batchManifestCacheStoreStub) Close() error { return nil }

var _ corecache.Store = (*batchManifestCacheStoreStub)(nil)

// ResolveBatchManifestOrderSummary resolves one order summary fixture by order id.
func (s batchManifestOrderSummaryResolverStub) ResolveBatchManifestOrderSummary(ctx context.Context, orderID string) (*BatchManifestOrderSummary, error) {
	row, exists := s.summaries[strings.TrimSpace(orderID)]
	if !exists {
		return nil, errors.New("order summary not found")
	}
	copy := row
	return &copy, nil
}

// TestManifestDocumentBuildsMergedPDFAndCaches verifies merged-PDF generation and 5-minute cache hit behavior.
func TestManifestDocumentBuildsMergedPDFAndCaches(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	batchRepository.markStore = markRepository
	service := NewService(batchRepository, markRepository, nil)
	service.SetBatchManifestDocumentCacheTTL(5 * time.Minute)
	cacheStore := &batchManifestCacheStoreStub{values: map[string]string{}}
	service.SetBatchManifestDocumentCacheStore(cacheStore)
	service.SetBatchManifestOrderSummaryResolver(batchManifestOrderSummaryResolverStub{summaries: map[string]BatchManifestOrderSummary{
		"order-1": {OrderNumber: "601205", Items: []string{"MORRAL AXEL", "NECESER"}},
		"order-2": {OrderNumber: "601206", Items: []string{"MORRAL NEO"}},
	}})

	manifestOne := buildTestPDF(t, "manifest-one")
	manifestTwo := buildTestPDF(t, "manifest-two")
	logoBytes := decodeBase64OrFail(t, "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGNgYAAAAAMAASsJTYQAAAAASUVORK5CYII=")

	var mu sync.Mutex
	manifestHits := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mu.Lock()
		manifestHits[request.URL.Path]++
		mu.Unlock()
		switch request.URL.Path {
		case "/logo.png":
			writer.Header().Set("Content-Type", "image/png")
			_, _ = writer.Write(logoBytes)
		case "/manifest-1.pdf":
			writer.Header().Set("Content-Type", "application/pdf")
			_, _ = writer.Write(manifestOne)
		case "/manifest-2.pdf":
			writer.Header().Set("Content-Type", "application/pdf")
			_, _ = writer.Write(manifestTwo)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	service.SetBatchManifestDocumentHTTPClient(server.Client())
	service.SetBatchManifestDocumentLogoURL(server.URL + "/logo.png")

	batchID := "batch-1"
	now := time.Now().UTC()
	batchRepository.batches[batchID] = domain.DispatchBatch{
		ID:        batchID,
		CarrierID: "tcc",
		Status:    domain.BatchStatusClosed,
		CreatedBy: "user-1",
		CreatedAt: now,
		ClosedAt:  &now,
	}
	markRepository.marks["mark-1"] = domain.ShippingMark{
		ID:              "mark-1",
		OrderID:         "order-1",
		CarrierID:       "tcc",
		Status:          domain.MarkStatusCreated,
		TrackingNumber:  "615099019",
		ManifestType:    domain.MarkDocumentLink,
		ManifestRef:     server.URL + "/manifest-1.pdf",
		Recipient:       domain.Address{Name: "John Harold", CityCode: "TULUA"},
		Units:           []domain.PackageUnit{{Description: "MORRAL AXEL"}},
		DispatchBatchID: &batchID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	markRepository.marks["mark-2"] = domain.ShippingMark{
		ID:              "mark-2",
		OrderID:         "order-2",
		CarrierID:       "tcc",
		Status:          domain.MarkStatusCreated,
		TrackingNumber:  "615099020",
		ManifestType:    domain.MarkDocumentLink,
		ManifestRef:     server.URL + "/manifest-2.pdf",
		Recipient:       domain.Address{Name: "Ana Ruiz", CityCode: "CALI"},
		Units:           []domain.PackageUnit{{Description: "MORRAL NEO"}},
		DispatchBatchID: &batchID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	firstPayload, err := service.ManifestDocument(context.Background(), batchID)
	if err != nil {
		t.Fatalf("ManifestDocument(first) error = %v", err)
	}
	if len(firstPayload) == 0 || !strings.HasPrefix(string(firstPayload), "%PDF") {
		t.Fatalf("ManifestDocument(first) returned non-pdf payload")
	}
	if cacheStore.setCalls == 0 {
		t.Fatalf("expected external cache set calls after first manifest generation")
	}
	service.manifestDocuments.cacheMutex.Lock()
	service.manifestDocuments.cache = map[string]batchManifestDocumentCacheEntry{}
	service.manifestDocuments.cacheMutex.Unlock()

	secondPayload, err := service.ManifestDocument(context.Background(), batchID)
	if err != nil {
		t.Fatalf("ManifestDocument(second) error = %v", err)
	}
	if string(firstPayload) != string(secondPayload) {
		t.Fatalf("ManifestDocument(second) payload differs from first cached payload")
	}

	mu.Lock()
	defer mu.Unlock()
	if got := manifestHits["/manifest-1.pdf"]; got != 1 {
		t.Fatalf("manifest-1 hits = %d, want 1", got)
	}
	if got := manifestHits["/manifest-2.pdf"]; got != 1 {
		t.Fatalf("manifest-2 hits = %d, want 1", got)
	}
	if got := manifestHits["/logo.png"]; got != 1 {
		t.Fatalf("logo hits = %d, want 1", got)
	}
	if cacheStore.getCalls == 0 {
		t.Fatalf("expected external cache get calls on second manifest request")
	}
}

// TestManifestDocumentRejectsOpenBatch verifies manifest-document generation rejects non-closed batches.
func TestManifestDocumentRejectsOpenBatch(t *testing.T) {
	batchRepository := newDispatchBatchRepositoryStub()
	markRepository := newDispatchMarkRepositoryStub()
	service := NewService(batchRepository, markRepository, nil)

	batchRepository.batches["batch-open"] = domain.DispatchBatch{
		ID:        "batch-open",
		CarrierID: "tcc",
		Status:    domain.BatchStatusOpen,
		CreatedBy: "user-1",
		CreatedAt: time.Now().UTC(),
	}

	_, err := service.ManifestDocument(context.Background(), "batch-open")
	if !errors.Is(err, domain.ErrInvalidBatchStatus) {
		t.Fatalf("ManifestDocument() error = %v, want ErrInvalidBatchStatus", err)
	}
}

// buildTestPDF creates one deterministic in-memory PDF payload for merge tests.
func buildTestPDF(t *testing.T, value string) []byte {
	t.Helper()
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(10, 10, 10)
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(100, 10, value, "", 1, "L", false, 0, "")
	var payload strings.Builder
	if err := pdf.Output(&payload); err != nil {
		t.Fatalf("buildTestPDF() output error = %v", err)
	}
	return []byte(payload.String())
}

// decodeBase64OrFail decodes one base64 payload and fails the test on decode errors.
func decodeBase64OrFail(t *testing.T, value string) []byte {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}
	return decoded
}
