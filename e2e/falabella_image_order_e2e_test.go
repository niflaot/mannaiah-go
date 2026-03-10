package e2e_test

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	corehttp "mannaiah/module/core/http"
	"mannaiah/module/falabella"
	"mannaiah/module/falabella/port"
)

type falabellaImageRequestPayload struct {
	XMLName xml.Name `xml:"Request"`
	Images  []string `xml:"ProductImage>Images>Image"`
}

// newFalabellaImageOrderServer creates a mock Falabella API server that captures image payload order.
func newFalabellaImageOrderServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()

	var (
		mutex          sync.Mutex
		capturedImages []string
	)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		action := strings.TrimSpace(request.URL.Query().Get("Action"))
		writer.Header().Set("Content-Type", "text/xml; charset=utf-8")

		switch action {
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
    <RequestId>feed-product-order-e2e</RequestId>
    <RequestAction>ProductCreate</RequestAction>
  </Head>
  <Body></Body>
</SuccessResponse>`))
		case "FeedStatus":
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head><RequestAction>FeedStatus</RequestAction></Head>
  <Body>
    <FeedDetail>
      <Feed>feed-product-order-e2e</Feed>
      <Status>Finished</Status>
      <Action>ProductCreate</Action>
      <TotalRecords>1</TotalRecords>
      <ProcessedRecords>1</ProcessedRecords>
      <FailedRecords>0</FailedRecords>
    </FeedDetail>
  </Body>
</SuccessResponse>`))
		case "Image":
			payloadBytes, readErr := io.ReadAll(request.Body)
			if readErr != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}

			var payload falabellaImageRequestPayload
			if decodeErr := xml.Unmarshal(payloadBytes, &payload); decodeErr != nil {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}

			mutex.Lock()
			capturedImages = append([]string(nil), payload.Images...)
			mutex.Unlock()

			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId>feed-image-order-e2e</RequestId>
    <RequestAction>Image</RequestAction>
  </Head>
  <Body></Body>
</SuccessResponse>`))
		default:
			writer.WriteHeader(http.StatusBadRequest)
		}
	}))

	getCaptured := func() []string {
		mutex.Lock()
		defer mutex.Unlock()
		return append([]string(nil), capturedImages...)
	}

	return server, getCaptured
}

// TestFalabellaImageSyncRespectsGalleryAndVariationPositionsE2E verifies Falabella image payload order honors position values.
func TestFalabellaImageSyncRespectsGalleryAndVariationPositionsE2E(t *testing.T) {
	tracer := newStepTracer(t)

	tracer.Step("start falabella mock server")
	mockServer, getCapturedImages := newFalabellaImageOrderServer(t)
	defer mockServer.Close()

	tracer.Step("initialize falabella module with position-aware catalog payload")
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
			ID:  "prod-image-order",
			SKU: "SKU-ORDER",
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
			Variants: []port.CatalogVariant{
				{SKU: "SKU-ORDER-V1", VariationIDs: []string{"v-color"}},
			},
			Images: []port.CatalogImage{
				{URL: "https://cdn.example.com/generic-low.jpg", Position: intPointer(10)},
				{URL: "https://cdn.example.com/variation-2.jpg", Position: intPointer(12), VariationPosition: intPointer(2), VariationIDs: []string{"v-color"}},
				{URL: "https://cdn.example.com/variation-1.jpg", Position: intPointer(11), VariationPosition: intPointer(1), VariationIDs: []string{"v-color"}},
				{URL: "https://cdn.example.com/generic-high.jpg", Position: intPointer(0)},
			},
		},
	})
	if err != nil {
		t.Fatalf("falabella.New() error = %v", err)
	}

	tracer.Step("initialize http server")
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8507}, tracer.logger)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(module.RegisterRoutes)

	tracer.Step("run product sync endpoint")
	req, _ := http.NewRequest(http.MethodPost, "/falabella/sync/products/prod-image-order", nil)
	resp, testErr := server.App().Test(req)
	if testErr != nil {
		t.Fatalf("App().Test() error = %v", testErr)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /falabella/sync/products/prod-image-order status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body struct {
		Synced int `json:"synced"`
		Failed int `json:"failed"`
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&body); decodeErr != nil {
		t.Fatalf("decode sync response error = %v", decodeErr)
	}

	tracer.Step("assert falabella image payload order")
	if body.Synced != 1 || body.Failed != 0 {
		t.Fatalf("sync summary = %#v, want synced=1 failed=0", body)
	}

	captured := getCapturedImages()
	expected := []string{
		"https://cdn.example.com/variation-1.jpg",
		"https://cdn.example.com/variation-2.jpg",
		"https://cdn.example.com/generic-high.jpg",
		"https://cdn.example.com/generic-low.jpg",
	}
	if len(captured) != len(expected) {
		t.Fatalf("captured image payload = %#v, want %d images", captured, len(expected))
	}
	for index, expectedURL := range expected {
		if captured[index] != expectedURL {
			t.Fatalf("captured[%d] = %q, want %q", index, captured[index], expectedURL)
		}
	}

	tracer.AssertStepCount(5)
}

func intPointer(value int) *int {
	resolved := value
	return &resolved
}
