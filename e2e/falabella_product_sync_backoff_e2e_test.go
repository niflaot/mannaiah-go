package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/falabella"
	"mannaiah/module/falabella/port"
)

// falabellaProductCatalogMock defines product-catalog behavior for Falabella product sync e2e tests.
type falabellaProductCatalogMock struct {
	// product defines fixed catalog product values returned by GetProduct.
	product port.CatalogProduct
}

// GetProduct returns configured catalog product values.
func (m falabellaProductCatalogMock) GetProduct(ctx context.Context, id string) (*port.CatalogProduct, error) {
	_ = ctx
	_ = id
	product := m.product
	return &product, nil
}

// ListProducts returns configured catalog products.
func (m falabellaProductCatalogMock) ListProducts(ctx context.Context) ([]port.CatalogProduct, error) {
	_ = ctx
	return []port.CatalogProduct{m.product}, nil
}

// newFalabellaProductSyncServer creates a mock Falabella API server for product/image/feed-status flows.
func newFalabellaProductSyncServer(t *testing.T, feedStatusPayloads []string) (*httptest.Server, *int32, *int32) {
	t.Helper()

	var (
		feedStatusCalls int32
		imageCalls      int32
		indexMutex      sync.Mutex
		payloadIndex    int
	)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		action := request.URL.Query().Get("Action")
		writer.Header().Set("Content-Type", "text/xml; charset=utf-8")

		switch strings.TrimSpace(action) {
		case "GetBrands":
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head><RequestAction>GetBrands</RequestAction></Head>
  <Body><Brands></Brands></Body>
</SuccessResponse>`))
		case "GetProducts":
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head><RequestAction>GetProducts</RequestAction></Head>
  <Body><Products></Products></Body>
</SuccessResponse>`))
		case "ProductCreate", "ProductUpdate":
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId>feed-product-e2e-1</RequestId>
    <RequestAction>ProductCreate</RequestAction>
  </Head>
  <Body></Body>
</SuccessResponse>`))
		case "FeedStatus":
			atomic.AddInt32(&feedStatusCalls, 1)

			indexMutex.Lock()
			idx := payloadIndex
			if idx >= len(feedStatusPayloads) {
				idx = len(feedStatusPayloads) - 1
			}
			payloadIndex++
			payload := feedStatusPayloads[idx]
			indexMutex.Unlock()

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(payload))
		case "Image":
			atomic.AddInt32(&imageCalls, 1)
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId>feed-image-e2e-1</RequestId>
    <RequestAction>Image</RequestAction>
  </Head>
  <Body></Body>
</SuccessResponse>`))
		default:
			writer.WriteHeader(http.StatusBadRequest)
		}
	}))

	return server, &feedStatusCalls, &imageCalls
}

// TestFalabellaProductSyncWaitsForFeedResolutionE2E verifies image feed starts only after product feed resolves successfully.
func TestFalabellaProductSyncWaitsForFeedResolutionE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("start falabella mock server")
	mockServer, feedStatusCalls, imageCalls := newFalabellaProductSyncServer(t, []string{
		`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head><RequestAction>FeedStatus</RequestAction></Head>
  <Body>
    <FeedDetail>
      <Feed>feed-product-e2e-1</Feed>
      <Status>Queued</Status>
      <Action>ProductCreate</Action>
      <TotalRecords>1</TotalRecords>
      <ProcessedRecords>0</ProcessedRecords>
      <FailedRecords>0</FailedRecords>
    </FeedDetail>
  </Body>
</SuccessResponse>`,
		`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head><RequestAction>FeedStatus</RequestAction></Head>
  <Body>
    <FeedDetail>
      <Feed>feed-product-e2e-1</Feed>
      <Status>Finished</Status>
      <Action>ProductCreate</Action>
      <TotalRecords>1</TotalRecords>
      <ProcessedRecords>1</ProcessedRecords>
      <FailedRecords>0</FailedRecords>
    </FeedDetail>
  </Body>
</SuccessResponse>`,
	})
	defer mockServer.Close()

	tracer.Step("initialize falabella module with product catalog and feed backoff")
	module, err := falabella.New(falabella.Config{
		URL:                                   mockServer.URL,
		UserID:                                "e2e@test.com",
		APIKey:                                "e2e-key",
		RequestTimeoutMS:                      1000,
		ValidationTimeoutMS:                   500,
		ProductFeedResolutionAttempts:         3,
		ProductFeedResolutionBackoffMS:        10,
		ProductFeedResolutionRequestTimeoutMS: 500,
	}, tracer.logger, falabellaProductCatalogMock{
		product: port.CatalogProduct{
			ID:  "prod-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{
					Realm:       "falabella",
					Name:        "Backpack",
					Description: "Backpack description",
					Attributes: map[string]any{
						"PriceFalabella": "100000",
						"Stock":          "10",
					},
				},
			},
			Images: []port.CatalogImage{
				{URL: "https://cdn.example.com/image.jpg"},
			},
		},
	})
	if err != nil {
		t.Fatalf("falabella.New() error = %v", err)
	}

	tracer.Step("initialize http server")
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8505}, tracer.logger)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	tracer.Step("run product sync endpoint")
	req, _ := http.NewRequest(http.MethodPost, "/falabella/sync/products/prod-1", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /falabella/sync/products/prod-1 status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body struct {
		Synced  int `json:"synced"`
		Failed  int `json:"failed"`
		Results []struct {
			Feeds []struct {
				Step string `json:"step"`
				Task string `json:"task"`
			} `json:"feeds"`
		} `json:"results"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&body); decodeErr != nil {
		t.Fatalf("decode sync response error = %v", decodeErr)
	}

	tracer.Step("assert feed-resolution gate behavior")
	if body.Synced != 1 || body.Failed != 0 {
		t.Fatalf("sync summary = %#v, want synced=1 failed=0", body)
	}
	if len(body.Results) != 1 || len(body.Results[0].Feeds) != 2 {
		t.Fatalf("sync feeds = %#v, want product+image feeds", body.Results)
	}
	if atomic.LoadInt32(feedStatusCalls) < 2 {
		t.Fatalf("feed status calls = %d, want >= 2", atomic.LoadInt32(feedStatusCalls))
	}
	if atomic.LoadInt32(imageCalls) != 1 {
		t.Fatalf("image calls = %d, want %d", atomic.LoadInt32(imageCalls), 1)
	}

	tracer.AssertStepCount(5)
}

// TestFalabellaProductSyncBlocksImageWhenFeedUnresolvedE2E verifies unresolved product feeds block image sync dispatch.
func TestFalabellaProductSyncBlocksImageWhenFeedUnresolvedE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("start falabella mock server")
	mockServer, _, imageCalls := newFalabellaProductSyncServer(t, []string{
		`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head><RequestAction>FeedStatus</RequestAction></Head>
  <Body>
    <FeedDetail>
      <Feed>feed-product-e2e-1</Feed>
      <Status>Queued</Status>
      <Action>ProductCreate</Action>
      <TotalRecords>1</TotalRecords>
      <ProcessedRecords>0</ProcessedRecords>
      <FailedRecords>0</FailedRecords>
    </FeedDetail>
  </Body>
</SuccessResponse>`,
	})
	defer mockServer.Close()

	tracer.Step("initialize falabella module with short feed-resolution retries")
	module, err := falabella.New(falabella.Config{
		URL:                                   mockServer.URL,
		UserID:                                "e2e@test.com",
		APIKey:                                "e2e-key",
		RequestTimeoutMS:                      1000,
		ValidationTimeoutMS:                   500,
		ProductFeedResolutionAttempts:         2,
		ProductFeedResolutionBackoffMS:        10,
		ProductFeedResolutionRequestTimeoutMS: 500,
	}, tracer.logger, falabellaProductCatalogMock{
		product: port.CatalogProduct{
			ID:  "prod-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{
					Realm:       "falabella",
					Name:        "Backpack",
					Description: "Backpack description",
					Attributes: map[string]any{
						"PriceFalabella": "100000",
						"Stock":          "10",
					},
				},
			},
			Images: []port.CatalogImage{
				{URL: "https://cdn.example.com/image.jpg"},
			},
		},
	})
	if err != nil {
		t.Fatalf("falabella.New() error = %v", err)
	}

	tracer.Step("initialize http server")
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8506}, tracer.logger)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	tracer.Step("run product sync endpoint")
	req, _ := http.NewRequest(http.MethodPost, "/falabella/sync/products/prod-1", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /falabella/sync/products/prod-1 status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body struct {
		Synced  int `json:"synced"`
		Failed  int `json:"failed"`
		Results []struct {
			Reason string `json:"reason"`
		} `json:"results"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&body); decodeErr != nil {
		t.Fatalf("decode sync response error = %v", decodeErr)
	}

	tracer.Step("assert unresolved feed blocked image sync")
	if body.Synced != 0 || body.Failed != 1 {
		t.Fatalf("sync summary = %#v, want synced=0 failed=1", body)
	}
	if len(body.Results) != 1 || !strings.Contains(body.Results[0].Reason, "was not finished") {
		t.Fatalf("sync reason = %#v, want unresolved feed reason", body.Results)
	}
	if atomic.LoadInt32(imageCalls) != 0 {
		t.Fatalf("image calls = %d, want %d", atomic.LoadInt32(imageCalls), 0)
	}

	tracer.AssertStepCount(5)
}
