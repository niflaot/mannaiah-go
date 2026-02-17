package service

import (
	"context"
	errorspkg "errors"
	"testing"

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
}

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

	return []byte(`{"ok":true}`), nil
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
					{Realm: "falabella", Name: "Backpack 1", Attributes: map[string]any{"brand": "GENERIC"}},
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
				{Realm: "falabella", Name: "Backpack 1"},
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

