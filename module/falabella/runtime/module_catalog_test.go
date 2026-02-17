package runtime

import (
	"context"
	errorspkg "errors"
	"testing"
)

// TestResolveCatalog verifies catalog resolution behavior.
func TestResolveCatalog(t *testing.T) {
	resolved := resolveCatalog()
	if resolved == nil {
		t.Fatalf("resolveCatalog() should not return nil")
	}

	if _, err := resolved.GetProduct(context.Background(), "p-1"); !errorspkg.Is(err, ErrProductCatalogNotConfigured) {
		t.Fatalf("GetProduct() error = %v, want ErrProductCatalogNotConfigured", err)
	}
	if _, err := resolved.ListProducts(context.Background()); !errorspkg.Is(err, ErrProductCatalogNotConfigured) {
		t.Fatalf("ListProducts() error = %v, want ErrProductCatalogNotConfigured", err)
	}
}

