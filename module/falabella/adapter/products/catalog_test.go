package products

import (
	"context"
	errorspkg "errors"
	"testing"

	productdomain "mannaiah/module/products/domain/product"
)

// serviceMock defines product-service behavior for adapter tests.
type serviceMock struct {
	// entity defines Get() return values.
	entity *productdomain.Product
	// entities defines List() return values.
	entities []productdomain.Product
	// err defines Get()/List() error values.
	err error
}

// Get returns configured product values.
func (m serviceMock) Get(ctx context.Context, id string) (*productdomain.Product, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.entity, nil
}

// List returns configured product collections.
func (m serviceMock) List(ctx context.Context) ([]productdomain.Product, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.entities, nil
}

// TestNewCatalogValidation verifies constructor validation behavior.
func TestNewCatalogValidation(t *testing.T) {
	_, err := NewCatalog(nil)
	if !errorspkg.Is(err, ErrNilService) {
		t.Fatalf("NewCatalog() error = %v, want %v", err, ErrNilService)
	}
}

// TestCatalogGetProduct verifies GetProduct mapping behavior.
func TestCatalogGetProduct(t *testing.T) {
	catalog, err := NewCatalog(serviceMock{
		entity: &productdomain.Product{
			ID:  "p-1",
			SKU: "SKU-1",
			Datasheets: []productdomain.Datasheet{
				{
					Realm:       "falabella",
					Name:        "Backpack",
					Description: "Desc",
					Attributes:  map[string]any{"brand": "GENERIC"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	entity, getErr := catalog.GetProduct(context.Background(), "p-1")
	if getErr != nil {
		t.Fatalf("GetProduct() error = %v", getErr)
	}
	if entity.SKU != "SKU-1" {
		t.Fatalf("entity.SKU = %q, want %q", entity.SKU, "SKU-1")
	}
	if entity.Datasheets[0].Attributes["brand"] != "GENERIC" {
		t.Fatalf("entity.Datasheets[0].Attributes[brand] = %v, want %q", entity.Datasheets[0].Attributes["brand"], "GENERIC")
	}
}

// TestCatalogListProducts verifies ListProducts mapping behavior.
func TestCatalogListProducts(t *testing.T) {
	catalog, err := NewCatalog(serviceMock{
		entities: []productdomain.Product{
			{ID: "p-1", SKU: "SKU-1"},
			{ID: "p-2", SKU: "SKU-2"},
		},
	})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	entities, listErr := catalog.ListProducts(context.Background())
	if listErr != nil {
		t.Fatalf("ListProducts() error = %v", listErr)
	}
	if len(entities) != 2 {
		t.Fatalf("len(entities) = %d, want %d", len(entities), 2)
	}
}

// TestCatalogErrorPropagation verifies service error propagation behavior.
func TestCatalogErrorPropagation(t *testing.T) {
	expectedErr := errorspkg.New("boom")
	catalog, err := NewCatalog(serviceMock{err: expectedErr})
	if err != nil {
		t.Fatalf("NewCatalog() error = %v", err)
	}

	if _, getErr := catalog.GetProduct(context.Background(), "p-1"); getErr == nil {
		t.Fatalf("GetProduct() expected error")
	}
	if _, listErr := catalog.ListProducts(context.Background()); listErr == nil {
		t.Fatalf("ListProducts() expected error")
	}
}

// TestMapProductNil verifies nil-mapping behavior.
func TestMapProductNil(t *testing.T) {
	if entity := mapProduct(nil); entity != nil {
		t.Fatalf("mapProduct(nil) = %#v, want nil", entity)
	}
}

