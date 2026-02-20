package service

import (
	"context"
	errorspkg "errors"
	"sync"
	"testing"
	"time"

	syncdomain "mannaiah/module/falabella/domain/sync"
	"mannaiah/module/falabella/port"
)

// sourceMock defines Falabella source behavior for sync-service tests.
type sourceMock struct {
	// validateErr defines Validate() errors.
	validateErr error
	// syncErr defines SyncProduct() errors.
	syncErr error
	// synced defines captured sync payload values.
	synced []port.SyncProductRequest
	// syncImagesErr defines SyncProductImages() errors.
	syncImagesErr error
	// syncedImages defines captured sync-image payload values.
	syncedImages []port.SyncProductImagesRequest
	// syncImagesResponse defines SyncProductImages() payload values.
	syncImagesResponse []byte
}

const testSyncResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId>feed-abc-123</RequestId>
    <RequestAction>ProductCreate</RequestAction>
  </Head>
  <Body/>
</SuccessResponse>`

const testSyncResponseWithWarningsXML = `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId>feed-warn-456</RequestId>
    <RequestAction>ProductCreate</RequestAction>
  </Head>
  <Body>
    <WarningDetail>
      <Field>Color</Field>
      <Message>Field 'Color' cannot be empty</Message>
      <Value>Empty</Value>
    </WarningDetail>
  </Body>
</SuccessResponse>`

const testImageSyncResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
	<Head>
		<RequestId>feed-img-999</RequestId>
		<RequestAction>Image</RequestAction>
	</Head>
	<Body/>
</SuccessResponse>`

// Validate returns configured integration-validation errors.
func (m *sourceMock) Validate(ctx context.Context) error {
	return m.validateErr
}

// SyncProduct captures sync payload values and returns configured errors.
func (m *sourceMock) SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error) {
	m.synced = append(m.synced, request)
	if m.syncErr != nil {
		return nil, m.syncErr
	}

	return []byte(testSyncResponseXML), nil
}

// SyncProductImages captures sync-image payload values and returns configured errors.
func (m *sourceMock) SyncProductImages(ctx context.Context, request port.SyncProductImagesRequest) ([]byte, error) {
	m.syncedImages = append(m.syncedImages, request)
	if m.syncImagesErr != nil {
		return nil, m.syncImagesErr
	}
	if len(m.syncImagesResponse) > 0 {
		return m.syncImagesResponse, nil
	}

	return []byte(`<SuccessResponse/>`), nil
}

// catalogMock defines product-catalog behavior for sync-service tests.
type catalogMock struct {
	// product defines GetProduct() values.
	product *port.CatalogProduct
	// products defines ListProducts() values.
	products []port.CatalogProduct
	// err defines GetProduct()/ListProducts() errors.
	err error
}

// GetProduct returns configured product values.
func (m catalogMock) GetProduct(ctx context.Context, id string) (*port.CatalogProduct, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.product, nil
}

// ListProducts returns configured product collections.
func (m catalogMock) ListProducts(ctx context.Context) ([]port.CatalogProduct, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.products, nil
}

// TestNewServiceValidation verifies constructor validation behavior.
func TestNewServiceValidation(t *testing.T) {
	if _, err := NewService(nil, catalogMock{}, Config{}); !errorspkg.Is(err, ErrNilSource) {
		t.Fatalf("NewService(nil source) error = %v, want ErrNilSource", err)
	}
	if _, err := NewService(&sourceMock{}, nil, Config{}); !errorspkg.Is(err, ErrNilCatalog) {
		t.Fatalf("NewService(nil catalog) error = %v, want ErrNilCatalog", err)
	}
}

// TestValidateIntegration verifies integration validation behavior.
func TestValidateIntegration(t *testing.T) {
	service, err := NewService(&sourceMock{}, catalogMock{}, Config{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if validationErr := service.ValidateIntegration(context.Background()); validationErr != nil {
		t.Fatalf("ValidateIntegration() error = %v", validationErr)
	}

	serviceUnavailable, err := NewService(&sourceMock{validateErr: errorspkg.New("down")}, catalogMock{}, Config{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if validationErr := serviceUnavailable.ValidateIntegration(context.Background()); !errorspkg.Is(validationErr, ErrIntegrationUnavailable) {
		t.Fatalf("ValidateIntegration() error = %v, want ErrIntegrationUnavailable", validationErr)
	}
}

// TestSyncProduct verifies single-product sync behavior.
func TestSyncProduct(t *testing.T) {
	source := &sourceMock{}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{
					Realm:       "falabella",
					Name:        "Backpack Name",
					Description: "Backpack description",
					Attributes: map[string]any{
						"brand": "GENERIC",
					},
				},
			},
		},
	}, Config{
		Realm:            "falabella",
		CategoryID:       "1638",
		GlobalIdentifier: "G08010305",
		AttributeSetID:   "5",
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Synced != 1 || summary.Failed != 0 || summary.Skipped != 0 {
		t.Fatalf("summary = %#v, want synced=1 failed=0 skipped=0", summary)
	}
	if len(source.synced) != 1 {
		t.Fatalf("len(source.synced) = %d, want %d", len(source.synced), 1)
	}
	if source.synced[0].PrimaryCategory != "1638" {
		t.Fatalf("PrimaryCategory = %q, want %q", source.synced[0].PrimaryCategory, "1638")
	}
	if summary.Results[0].FeedID != "feed-abc-123" {
		t.Fatalf("FeedID = %q, want %q", summary.Results[0].FeedID, "feed-abc-123")
	}
	if summary.ExecutionID == "" {
		t.Fatalf("ExecutionID should not be empty")
	}
}

// TestSyncProductWithImageFeedAggregation verifies one result can expose both product and image feed IDs.
func TestSyncProductWithImageFeedAggregation(t *testing.T) {
	source := &sourceMock{syncImagesResponse: []byte(testImageSyncResponseXML)}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack", Description: "Backpack description"},
			},
			Images: []port.CatalogImage{
				{URL: "https://cdn.example.com/1.jpg"},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Synced != 1 {
		t.Fatalf("summary.Synced = %d, want %d", summary.Synced, 1)
	}
	if summary.ExecutionID == "" {
		t.Fatalf("ExecutionID should not be empty")
	}
	if len(summary.Results) != 1 {
		t.Fatalf("len(summary.Results) = %d, want %d", len(summary.Results), 1)
	}
	if len(summary.Results[0].Feeds) != 2 {
		t.Fatalf("len(result.Feeds) = %d, want %d", len(summary.Results[0].Feeds), 2)
	}
	if summary.Results[0].Feeds[0].Step != "product" {
		t.Fatalf("first feed step = %q, want %q", summary.Results[0].Feeds[0].Step, "product")
	}
	if summary.Results[0].Feeds[0].FeedID != "feed-abc-123" {
		t.Fatalf("first feed id = %q, want %q", summary.Results[0].Feeds[0].FeedID, "feed-abc-123")
	}
	if summary.Results[0].Feeds[1].Step != "image" {
		t.Fatalf("second feed step = %q, want %q", summary.Results[0].Feeds[1].Step, "image")
	}
	if summary.Results[0].Feeds[1].FeedID != "feed-img-999" {
		t.Fatalf("second feed id = %q, want %q", summary.Results[0].Feeds[1].FeedID, "feed-img-999")
	}
}

// TestSyncProductWithVariants verifies parent/child variant sync behavior.
func TestSyncProductWithVariants(t *testing.T) {
	source := &sourceMock{}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-PARENT",
			Datasheets: []port.CatalogDatasheet{
				{
					Realm:       "falabella",
					Name:        "Backpack",
					Description: "Backpack description",
					Attributes: map[string]any{
						"brand": "GENERIC",
					},
				},
			},
			Images: []port.CatalogImage{
				{URL: "https://cdn.example.com/parent.jpg"},
				{URL: "https://cdn.example.com/variant.jpg", VariationIDs: []string{"v-color"}},
			},
			Variants: []port.CatalogVariant{
				{
					SKU:          "SKU-RED-M",
					VariationIDs: []string{"v-color", "v-size"},
					Variations: []port.CatalogVariation{
						{Definition: "COLOR", Value: "Red"},
						{Definition: "SIZE", Value: "M"},
					},
				},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Requested != 1 || summary.Synced != 1 || summary.Failed != 0 || summary.Skipped != 0 {
		t.Fatalf("summary = %#v, want requested=1 synced=1 failed=0 skipped=0", summary)
	}
	if len(source.synced) != 1 {
		t.Fatalf("len(source.synced) = %d, want %d", len(source.synced), 1)
	}
	if source.synced[0].SKU != "SKU-RED-M" || source.synced[0].ParentSKU != "SKU-PARENT" {
		t.Fatalf("child request = %#v, want sku=SKU-RED-M parentSKU=SKU-PARENT", source.synced[0])
	}
	if source.synced[0].Attributes["Color"] != "Red" {
		t.Fatalf("Color = %q, want %q", source.synced[0].Attributes["Color"], "Red")
	}
	if source.synced[0].Attributes["Talla"] != "M" {
		t.Fatalf("Talla = %q, want %q", source.synced[0].Attributes["Talla"], "M")
	}
	if len(source.syncedImages) != 1 {
		t.Fatalf("len(source.syncedImages) = %d, want %d", len(source.syncedImages), 1)
	}
	if len(source.syncedImages[0].URLs) != 2 {
		t.Fatalf("variant image urls = %#v, want 2 urls", source.syncedImages[0].URLs)
	}
}

// TestSyncProducts verifies multi-product sync behavior.
func TestSyncProducts(t *testing.T) {
	source := &sourceMock{}
	service, err := NewService(source, catalogMock{
		products: []port.CatalogProduct{
			{
				ID:  "p-1",
				SKU: "SKU-1",
				Datasheets: []port.CatalogDatasheet{
					{Realm: "falabella", Name: "Backpack 1", Description: "Backpack 1 description", Attributes: map[string]any{"brand": "GENERIC"}},
				},
			},
			{
				ID:         "p-2",
				SKU:        "SKU-2",
				Datasheets: []port.CatalogDatasheet{{Realm: "default", Name: "No sync"}},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProducts(context.Background(), nil)
	if syncErr != nil {
		t.Fatalf("SyncProducts() error = %v", syncErr)
	}
	if summary.Requested != 2 || summary.Synced != 1 || summary.Skipped != 1 || summary.Failed != 0 {
		t.Fatalf("summary = %#v, want requested=2 synced=1 skipped=1 failed=0", summary)
	}
}

// concurrentSourceMock defines source behavior for sync concurrency tests.
type concurrentSourceMock struct {
	// mutex guards active counters.
	mutex sync.Mutex
	// active defines currently running sync calls.
	active int
	// maxActive defines observed max concurrent sync calls.
	maxActive int
}

// Validate returns nil.
func (m *concurrentSourceMock) Validate(ctx context.Context) error { return nil }

// SyncProduct tracks concurrent execution and returns success responses.
func (m *concurrentSourceMock) SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error) {
	m.mutex.Lock()
	m.active++
	if m.active > m.maxActive {
		m.maxActive = m.active
	}
	m.mutex.Unlock()

	time.Sleep(20 * time.Millisecond)

	m.mutex.Lock()
	m.active--
	m.mutex.Unlock()

	return []byte(testSyncResponseXML), nil
}

// SyncProductImages returns no-op responses.
func (m *concurrentSourceMock) SyncProductImages(ctx context.Context, request port.SyncProductImagesRequest) ([]byte, error) {
	return []byte(`<SuccessResponse/>`), nil
}

// TestSyncProductsUsesConcurrentWorkers verifies batch sync uses bounded parallelism.
func TestSyncProductsUsesConcurrentWorkers(t *testing.T) {
	source := &concurrentSourceMock{}
	products := []port.CatalogProduct{
		{ID: "p-1", SKU: "SKU-1", Datasheets: []port.CatalogDatasheet{{Realm: "falabella", Name: "N1", Description: "D1"}}},
		{ID: "p-2", SKU: "SKU-2", Datasheets: []port.CatalogDatasheet{{Realm: "falabella", Name: "N2", Description: "D2"}}},
		{ID: "p-3", SKU: "SKU-3", Datasheets: []port.CatalogDatasheet{{Realm: "falabella", Name: "N3", Description: "D3"}}},
		{ID: "p-4", SKU: "SKU-4", Datasheets: []port.CatalogDatasheet{{Realm: "falabella", Name: "N4", Description: "D4"}}},
	}

	service, err := NewService(source, catalogMock{products: products}, Config{Realm: "falabella", SyncWorkers: 4})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProducts(context.Background(), nil)
	if syncErr != nil {
		t.Fatalf("SyncProducts() error = %v", syncErr)
	}
	if summary.Synced != 4 {
		t.Fatalf("summary.Synced = %d, want %d", summary.Synced, 4)
	}
	if source.maxActive < 2 {
		t.Fatalf("maxActive = %d, want >= 2", source.maxActive)
	}
}

// TestResolveSyncWorkerCount verifies worker-count normalization behavior.
func TestResolveSyncWorkerCount(t *testing.T) {
	if got := resolveSyncWorkerCount(0, 0); got != 1 {
		t.Fatalf("resolveSyncWorkerCount(0,0) = %d, want %d", got, 1)
	}
	if got := resolveSyncWorkerCount(0, 1); got != 1 {
		t.Fatalf("resolveSyncWorkerCount(0,1) = %d, want %d", got, 1)
	}
	if got := resolveSyncWorkerCount(0, 10); got != defaultSyncWorkers {
		t.Fatalf("resolveSyncWorkerCount(0,10) = %d, want %d", got, defaultSyncWorkers)
	}
	if got := resolveSyncWorkerCount(maxSyncWorkers+5, 100); got != maxSyncWorkers {
		t.Fatalf("resolveSyncWorkerCount(max+5,100) = %d, want %d", got, maxSyncWorkers)
	}
	if got := resolveSyncWorkerCount(10, 3); got != 3 {
		t.Fatalf("resolveSyncWorkerCount(10,3) = %d, want %d", got, 3)
	}
}

// TestSyncProductsErrors verifies invalid input and downstream-error behavior.
func TestSyncProductsErrors(t *testing.T) {
	service, err := NewService(&sourceMock{}, catalogMock{}, Config{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, syncErr := service.SyncProduct(context.Background(), " "); !errorspkg.Is(syncErr, ErrInvalidProductID) {
		t.Fatalf("SyncProduct(empty) error = %v, want ErrInvalidProductID", syncErr)
	}
	if _, syncErr := service.SyncProducts(context.Background(), []string{" "}); !errorspkg.Is(syncErr, ErrInvalidProductID) {
		t.Fatalf("SyncProducts(empty-id) error = %v, want ErrInvalidProductID", syncErr)
	}

	downstreamErr := errorspkg.New("boom")
	serviceWithErrors, err := NewService(&sourceMock{}, catalogMock{err: downstreamErr}, Config{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, syncErr := serviceWithErrors.SyncProduct(context.Background(), "p-1"); syncErr == nil {
		t.Fatalf("SyncProduct() expected catalog error")
	}
	if _, syncErr := serviceWithErrors.SyncProducts(context.Background(), nil); syncErr == nil {
		t.Fatalf("SyncProducts() expected catalog error")
	}
}

// TestSyncOneFailure verifies sync failure aggregation behavior.
func TestSyncOneFailure(t *testing.T) {
	source := &sourceMock{syncErr: errorspkg.New("upstream failed")}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack 1", Description: "Backpack description"},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Failed != 1 || summary.Synced != 0 {
		t.Fatalf("summary = %#v, want failed=1 synced=0", summary)
	}
}

// TestSyncProductVariantValidationFailure verifies invalid variant SKU handling.
func TestSyncProductVariantValidationFailure(t *testing.T) {
	source := &sourceMock{}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-PARENT",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack 1", Description: "Backpack description"},
			},
			Variants: []port.CatalogVariant{
				{SKU: " "},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Requested != 1 || summary.Synced != 0 || summary.Failed != 1 {
		t.Fatalf("summary = %#v, want requested=1 synced=0 failed=1", summary)
	}
	if len(source.synced) != 0 {
		t.Fatalf("len(source.synced) = %d, want %d", len(source.synced), 0)
	}
}

// TestSyncProductDuplicateVariantSKU verifies duplicate variant sku skip behavior.
func TestSyncProductDuplicateVariantSKU(t *testing.T) {
	source := &sourceMock{}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-PARENT",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack 1", Description: "Backpack description"},
			},
			Variants: []port.CatalogVariant{
				{SKU: "SKU-RED-M"},
				{SKU: "SKU-RED-M"},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Requested != 2 || summary.Synced != 1 || summary.Skipped != 1 || summary.Failed != 0 {
		t.Fatalf("summary = %#v, want requested=2 synced=1 skipped=1 failed=0", summary)
	}
	if len(source.synced) != 1 {
		t.Fatalf("len(source.synced) = %d, want %d", len(source.synced), 1)
	}
}

// TestSyncProductImageFailure verifies image-sync failure aggregation behavior.
func TestSyncProductImageFailure(t *testing.T) {
	source := &sourceMock{syncImagesErr: errorspkg.New("image failed")}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack 1", Description: "Backpack description"},
			},
			Images: []port.CatalogImage{
				{URL: "https://cdn.example.com/1.jpg"},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Requested != 1 || summary.Synced != 0 || summary.Failed != 1 {
		t.Fatalf("summary = %#v, want requested=1 synced=0 failed=1", summary)
	}
}

// warningSourceMock defines Falabella source behavior that returns warning responses.
type warningSourceMock struct {
	// synced defines captured sync payload values.
	synced []port.SyncProductRequest
}

// Validate returns nil.
func (m *warningSourceMock) Validate(ctx context.Context) error { return nil }

// SyncProduct returns a response with warnings.
func (m *warningSourceMock) SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error) {
	m.synced = append(m.synced, request)
	return []byte(testSyncResponseWithWarningsXML), nil
}

// SyncProductImages returns a no-op response.
func (m *warningSourceMock) SyncProductImages(ctx context.Context, request port.SyncProductImagesRequest) ([]byte, error) {
	return []byte(`<SuccessResponse/>`), nil
}

// TestSyncProductWithRequiredFieldWarnings verifies required-field violation warnings cause sync failure.
func TestSyncProductWithRequiredFieldWarnings(t *testing.T) {
	source := &warningSourceMock{}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack", Description: "Backpack description"},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Synced != 0 {
		t.Fatalf("summary.Synced = %d, want %d", summary.Synced, 0)
	}
	if summary.Failed != 1 {
		t.Fatalf("summary.Failed = %d, want %d", summary.Failed, 1)
	}
	if summary.Results[0].Status != "failed" {
		t.Fatalf("Status = %q, want %q", summary.Results[0].Status, "failed")
	}
	if summary.Results[0].Reason != "required_field_warnings" {
		t.Fatalf("Reason = %q, want %q", summary.Results[0].Reason, "required_field_warnings")
	}
	if summary.Results[0].FeedID != "feed-warn-456" {
		t.Fatalf("FeedID = %q, want %q", summary.Results[0].FeedID, "feed-warn-456")
	}
	if len(summary.Results[0].Warnings) != 1 {
		t.Fatalf("len(Warnings) = %d, want %d", len(summary.Results[0].Warnings), 1)
	}
}

const testSyncResponseBenignWarningXML = `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId>feed-benign-789</RequestId>
    <RequestAction>ProductCreate</RequestAction>
  </Head>
  <Body>
    <WarningDetail>
      <Field>SomeField</Field>
      <Message>This is an informational note</Message>
      <Value>Info</Value>
    </WarningDetail>
  </Body>
</SuccessResponse>`

// benignWarningSourceMock defines Falabella source behavior that returns benign warning responses.
type benignWarningSourceMock struct {
	// synced defines captured sync payload values.
	synced []port.SyncProductRequest
}

// Validate returns nil.
func (m *benignWarningSourceMock) Validate(ctx context.Context) error { return nil }

// SyncProduct returns a response with benign warnings.
func (m *benignWarningSourceMock) SyncProduct(ctx context.Context, request port.SyncProductRequest) ([]byte, error) {
	m.synced = append(m.synced, request)
	return []byte(testSyncResponseBenignWarningXML), nil
}

// SyncProductImages returns a no-op response.
func (m *benignWarningSourceMock) SyncProductImages(ctx context.Context, request port.SyncProductImagesRequest) ([]byte, error) {
	return []byte(`<SuccessResponse/>`), nil
}

// TestSyncProductWithBenignWarnings verifies benign warnings do not cause sync failure.
func TestSyncProductWithBenignWarnings(t *testing.T) {
	source := &benignWarningSourceMock{}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack", Description: "Backpack description"},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Synced != 1 {
		t.Fatalf("summary.Synced = %d, want %d", summary.Synced, 1)
	}
	if summary.Failed != 0 {
		t.Fatalf("summary.Failed = %d, want %d", summary.Failed, 0)
	}
	if summary.Results[0].FeedID != "feed-benign-789" {
		t.Fatalf("FeedID = %q, want %q", summary.Results[0].FeedID, "feed-benign-789")
	}
	if len(summary.Results[0].Warnings) != 1 {
		t.Fatalf("len(Warnings) = %d, want %d", len(summary.Results[0].Warnings), 1)
	}
}

// recorderMock defines sync status recording behavior for productsync service tests.
type recorderMock struct {
	// entries defines recorded sync entries.
	entries []*syncdomain.SyncEntry
	// err defines RecordEntry() errors.
	err error
}

// RecordEntry captures entries or returns configured errors.
func (m *recorderMock) RecordEntry(ctx context.Context, entry *syncdomain.SyncEntry) error {
	if m.err != nil {
		return m.err
	}
	m.entries = append(m.entries, entry)
	return nil
}

// TestSyncProductRecordsEntry verifies sync status entry recording on successful sync.
func TestSyncProductRecordsEntry(t *testing.T) {
	source := &sourceMock{}
	recorder := &recorderMock{}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack", Description: "Backpack description"},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetRecorder(recorder)

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Synced != 1 {
		t.Fatalf("summary.Synced = %d, want %d", summary.Synced, 1)
	}
	if len(recorder.entries) != 1 {
		t.Fatalf("len(recorder.entries) = %d, want %d", len(recorder.entries), 1)
	}
	if recorder.entries[0].ProductID != "p-1" {
		t.Fatalf("ProductID = %q, want %q", recorder.entries[0].ProductID, "p-1")
	}
	if recorder.entries[0].SKU != "SKU-1" {
		t.Fatalf("SKU = %q, want %q", recorder.entries[0].SKU, "SKU-1")
	}
	if recorder.entries[0].FeedID != "feed-abc-123" {
		t.Fatalf("FeedID = %q, want %q", recorder.entries[0].FeedID, "feed-abc-123")
	}
	if recorder.entries[0].Action != syncdomain.SyncActionCreate {
		t.Fatalf("Action = %q, want %q", recorder.entries[0].Action, syncdomain.SyncActionCreate)
	}
	if recorder.entries[0].Status != syncdomain.SyncStatusPending {
		t.Fatalf("Status = %q, want %q", recorder.entries[0].Status, syncdomain.SyncStatusPending)
	}
}

// TestSyncProductRecorderErrorDoesNotFailSync verifies recorder errors do not fail the sync.
func TestSyncProductRecorderErrorDoesNotFailSync(t *testing.T) {
	source := &sourceMock{}
	recorder := &recorderMock{err: errorspkg.New("db down")}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack", Description: "Backpack description"},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetRecorder(recorder)

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Synced != 1 {
		t.Fatalf("summary.Synced = %d, want %d", summary.Synced, 1)
	}
}

// TestSyncProductNoRecorderDoesNotPanic verifies sync works without recorder configured.
func TestSyncProductNoRecorderDoesNotPanic(t *testing.T) {
	source := &sourceMock{}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack", Description: "Backpack description"},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Synced != 1 {
		t.Fatalf("summary.Synced = %d, want %d", summary.Synced, 1)
	}
}

// TestSyncProductVariantRecordsEntry verifies sync status entry recording for variant syncs.
func TestSyncProductVariantRecordsEntry(t *testing.T) {
	source := &sourceMock{}
	recorder := &recorderMock{}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-PARENT",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack", Description: "Backpack description"},
			},
			Variants: []port.CatalogVariant{
				{SKU: "SKU-RED-M", Variations: []port.CatalogVariation{
					{Definition: "COLOR", Value: "Red"},
				}},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetRecorder(recorder)

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Synced != 1 {
		t.Fatalf("summary.Synced = %d, want %d", summary.Synced, 1)
	}
	if len(recorder.entries) != 1 {
		t.Fatalf("len(recorder.entries) = %d, want %d", len(recorder.entries), 1)
	}
	if recorder.entries[0].SKU != "SKU-RED-M" {
		t.Fatalf("SKU = %q, want %q", recorder.entries[0].SKU, "SKU-RED-M")
	}
}

// TestSyncProductRequiredFieldWarningsVariant verifies required-field warnings fail variant syncs.
func TestSyncProductRequiredFieldWarningsVariant(t *testing.T) {
	source := &warningSourceMock{}
	service, err := NewService(source, catalogMock{
		product: &port.CatalogProduct{
			ID:  "p-1",
			SKU: "SKU-PARENT",
			Datasheets: []port.CatalogDatasheet{
				{Realm: "falabella", Name: "Backpack", Description: "Backpack description"},
			},
			Variants: []port.CatalogVariant{
				{SKU: "SKU-RED-M"},
			},
		},
	}, Config{Realm: "falabella"})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	summary, syncErr := service.SyncProduct(context.Background(), "p-1")
	if syncErr != nil {
		t.Fatalf("SyncProduct() error = %v", syncErr)
	}
	if summary.Failed != 1 {
		t.Fatalf("summary.Failed = %d, want %d", summary.Failed, 1)
	}
	if summary.Synced != 0 {
		t.Fatalf("summary.Synced = %d, want %d", summary.Synced, 0)
	}
	if summary.Results[0].Reason != "required_field_warnings" {
		t.Fatalf("Reason = %q, want %q", summary.Results[0].Reason, "required_field_warnings")
	}
}
