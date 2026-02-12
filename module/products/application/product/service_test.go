package product

import (
	"context"
	errorspkg "errors"
	"sync/atomic"
	"testing"

	productdomain "mannaiah/module/products/domain/product"
	productport "mannaiah/module/products/port/product"
)

// repositoryMock defines persistence behavior for service tests.
type repositoryMock struct {
	// createFn defines create behavior.
	createFn func(ctx context.Context, product *productdomain.Product) error
	// getFn defines get behavior.
	getFn func(ctx context.Context, id string) (*productdomain.Product, error)
	// listFn defines list behavior.
	listFn func(ctx context.Context) ([]productdomain.Product, error)
	// updateFn defines update behavior.
	updateFn func(ctx context.Context, product *productdomain.Product) error
	// deleteFn defines delete behavior.
	deleteFn func(ctx context.Context, id string) error
}

// EnsureSchema ignores schema behavior for service tests.
func (m repositoryMock) EnsureSchema(ctx context.Context) error { return nil }

// Create executes configured create behavior.
func (m repositoryMock) Create(ctx context.Context, product *productdomain.Product) error {
	return m.createFn(ctx, product)
}

// GetByID executes configured get behavior.
func (m repositoryMock) GetByID(ctx context.Context, id string) (*productdomain.Product, error) {
	return m.getFn(ctx, id)
}

// List executes configured list behavior.
func (m repositoryMock) List(ctx context.Context) ([]productdomain.Product, error) {
	return m.listFn(ctx)
}

// Update executes configured update behavior.
func (m repositoryMock) Update(ctx context.Context, product *productdomain.Product) error {
	return m.updateFn(ctx, product)
}

// Delete executes configured delete behavior.
func (m repositoryMock) Delete(ctx context.Context, id string) error {
	return m.deleteFn(ctx, id)
}

// assetLookupMock defines asset lookup behavior for service tests.
type assetLookupMock struct {
	// existsFn defines exists behavior.
	existsFn func(ctx context.Context, id string) (bool, error)
}

// Exists executes configured lookup behavior.
func (m assetLookupMock) Exists(ctx context.Context, id string) (bool, error) {
	return m.existsFn(ctx, id)
}

// TestNewService validates constructor behavior.
func TestNewService(t *testing.T) {
	if _, err := NewService(nil, assetLookupMock{existsFn: func(ctx context.Context, id string) (bool, error) { return true, nil }}); !errorspkg.Is(err, ErrNilRepository) {
		t.Fatalf("NewService(nil, lookup) error = %v, want ErrNilRepository", err)
	}
	if _, err := NewService(repositoryMock{}, nil); !errorspkg.Is(err, ErrNilAssetLookup) {
		t.Fatalf("NewService(repo, nil) error = %v, want ErrNilAssetLookup", err)
	}
}

// TestCreate verifies create behavior.
func TestCreate(t *testing.T) {
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, product *productdomain.Product) error {
			product.ID = "p-1"
			return nil
		},
		getFn:    func(ctx context.Context, id string) (*productdomain.Product, error) { return nil, nil },
		listFn:   func(ctx context.Context) ([]productdomain.Product, error) { return nil, nil },
		updateFn: func(ctx context.Context, product *productdomain.Product) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	}, assetLookupMock{existsFn: func(ctx context.Context, id string) (bool, error) {
		return id != "missing", nil
	}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	entity, createErr := service.Create(context.Background(), CreateCommand{SKU: "SKU-1"})
	if createErr != nil {
		t.Fatalf("Create() error = %v", createErr)
	}
	if entity.ID != "p-1" {
		t.Fatalf("entity.ID = %q, want %q", entity.ID, "p-1")
	}

	_, missingAssetErr := service.Create(context.Background(), CreateCommand{
		SKU: "SKU-2",
		Gallery: []productdomain.GalleryItem{
			{AssetID: "missing"},
		},
	})
	if !errorspkg.Is(missingAssetErr, ErrAssetNotFound) {
		t.Fatalf("Create(missing asset) error = %v, want ErrAssetNotFound", missingAssetErr)
	}
}

// TestGetListDelete verifies get/list/delete behavior.
func TestGetListDelete(t *testing.T) {
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, product *productdomain.Product) error { return nil },
		getFn: func(ctx context.Context, id string) (*productdomain.Product, error) {
			return &productdomain.Product{ID: id, SKU: "SKU-1"}, nil
		},
		listFn: func(ctx context.Context) ([]productdomain.Product, error) {
			return []productdomain.Product{{ID: "p-1", SKU: "SKU-1"}}, nil
		},
		updateFn: func(ctx context.Context, product *productdomain.Product) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	}, assetLookupMock{existsFn: func(ctx context.Context, id string) (bool, error) { return true, nil }})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, getErr := service.Get(context.Background(), ""); !errorspkg.Is(getErr, ErrInvalidID) {
		t.Fatalf("Get(empty) error = %v, want ErrInvalidID", getErr)
	}
	if _, getErr := service.Get(context.Background(), "p-1"); getErr != nil {
		t.Fatalf("Get() error = %v", getErr)
	}
	if values, listErr := service.List(context.Background()); listErr != nil || len(values) != 1 {
		t.Fatalf("List() error = %v len = %d", listErr, len(values))
	}
	if deleteErr := service.Delete(context.Background(), ""); !errorspkg.Is(deleteErr, ErrInvalidID) {
		t.Fatalf("Delete(empty) error = %v, want ErrInvalidID", deleteErr)
	}
}

// TestUpdateMergesDatasheets verifies datasheet merge behavior.
func TestUpdateMergesDatasheets(t *testing.T) {
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, product *productdomain.Product) error { return nil },
		getFn: func(ctx context.Context, id string) (*productdomain.Product, error) {
			return &productdomain.Product{ID: id, SKU: "SKU-1", Datasheets: []productdomain.Datasheet{{Realm: "default", Name: "Old"}}}, nil
		},
		listFn: func(ctx context.Context) ([]productdomain.Product, error) { return nil, nil },
		updateFn: func(ctx context.Context, product *productdomain.Product) error {
			if len(product.Datasheets) != 2 {
				return errorspkg.New("datasheet merge failed")
			}
			return nil
		},
		deleteFn: func(ctx context.Context, id string) error { return nil },
	}, assetLookupMock{existsFn: func(ctx context.Context, id string) (bool, error) { return true, nil }})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	entity, updateErr := service.Update(context.Background(), "p-1", UpdateCommand{
		Datasheets:    []productdomain.Datasheet{{Realm: "default", Name: "New"}, {Realm: "b2b", Name: "Bulk"}},
		HasDatasheets: true,
	})
	if updateErr != nil {
		t.Fatalf("Update() error = %v", updateErr)
	}
	if len(entity.Datasheets) != 2 {
		t.Fatalf("len(entity.Datasheets) = %d, want %d", len(entity.Datasheets), 2)
	}
}

// TestUpdateValidatesGalleryAssets verifies gallery asset lookup behavior in updates.
func TestUpdateValidatesGalleryAssets(t *testing.T) {
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, product *productdomain.Product) error { return nil },
		getFn: func(ctx context.Context, id string) (*productdomain.Product, error) {
			return &productdomain.Product{ID: id, SKU: "SKU-1"}, nil
		},
		listFn:   func(ctx context.Context) ([]productdomain.Product, error) { return nil, nil },
		updateFn: func(ctx context.Context, product *productdomain.Product) error { return nil },
		deleteFn: func(ctx context.Context, id string) error { return nil },
	}, assetLookupMock{existsFn: func(ctx context.Context, id string) (bool, error) {
		if id == "missing" {
			return false, nil
		}
		return true, nil
	}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, updateErr := service.Update(context.Background(), "p-1", UpdateCommand{
		Gallery:    []productdomain.GalleryItem{{AssetID: "missing"}},
		HasGallery: true,
	})
	if !errorspkg.Is(updateErr, ErrAssetNotFound) {
		t.Fatalf("Update() error = %v, want ErrAssetNotFound", updateErr)
	}
}

// TestValidateGalleryAssetsConcurrency verifies deduplicated concurrent lookup behavior.
func TestValidateGalleryAssetsConcurrency(t *testing.T) {
	var calls int64
	lookup := assetLookupMock{existsFn: func(ctx context.Context, id string) (bool, error) {
		atomic.AddInt64(&calls, 1)
		return true, nil
	}}

	gallery := make([]productdomain.GalleryItem, 0, 20)
	for index := 0; index < 20; index++ {
		gallery = append(gallery, productdomain.GalleryItem{AssetID: "asset-1"})
	}

	if err := validateGalleryAssets(context.Background(), lookup, gallery); err != nil {
		t.Fatalf("validateGalleryAssets() error = %v", err)
	}
	if atomic.LoadInt64(&calls) != 1 {
		t.Fatalf("lookup calls = %d, want %d", calls, 1)
	}
}

// TestErrorWrapping verifies wrapped repository errors.
func TestErrorWrapping(t *testing.T) {
	repositoryErr := errorspkg.New("repository failed")
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, product *productdomain.Product) error { return repositoryErr },
		getFn:    func(ctx context.Context, id string) (*productdomain.Product, error) { return nil, repositoryErr },
		listFn:   func(ctx context.Context) ([]productdomain.Product, error) { return nil, repositoryErr },
		updateFn: func(ctx context.Context, product *productdomain.Product) error { return repositoryErr },
		deleteFn: func(ctx context.Context, id string) error { return repositoryErr },
	}, assetLookupMock{existsFn: func(ctx context.Context, id string) (bool, error) { return true, nil }})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, createErr := service.Create(context.Background(), CreateCommand{SKU: "SKU-1"}); createErr == nil {
		t.Fatalf("expected create wrapped error")
	}
	if _, getErr := service.Get(context.Background(), "id"); getErr == nil {
		t.Fatalf("expected get wrapped error")
	}
	if _, listErr := service.List(context.Background()); listErr == nil {
		t.Fatalf("expected list wrapped error")
	}
	if _, updateErr := service.Update(context.Background(), "id", UpdateCommand{}); updateErr == nil {
		t.Fatalf("expected update wrapped error")
	}
	if deleteErr := service.Delete(context.Background(), "id"); deleteErr == nil {
		t.Fatalf("expected delete wrapped error")
	}
}

// TestSentinelUnwrap verifies wrapped errors include repository sentinel values.
func TestSentinelUnwrap(t *testing.T) {
	service, err := NewService(repositoryMock{
		createFn: func(ctx context.Context, product *productdomain.Product) error { return productport.ErrDuplicateSKU },
		getFn: func(ctx context.Context, id string) (*productdomain.Product, error) {
			return nil, productport.ErrNotFound
		},
		listFn:   func(ctx context.Context) ([]productdomain.Product, error) { return nil, nil },
		updateFn: func(ctx context.Context, product *productdomain.Product) error { return productport.ErrDuplicateSKU },
		deleteFn: func(ctx context.Context, id string) error { return productport.ErrNotFound },
	}, assetLookupMock{existsFn: func(ctx context.Context, id string) (bool, error) { return true, nil }})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, createErr := service.Create(context.Background(), CreateCommand{SKU: "SKU-1"}); !errorspkg.Is(createErr, productport.ErrDuplicateSKU) {
		t.Fatalf("Create() error = %v, want productport.ErrDuplicateSKU", createErr)
	}
	if _, getErr := service.Get(context.Background(), "id"); !errorspkg.Is(getErr, productport.ErrNotFound) {
		t.Fatalf("Get() error = %v, want productport.ErrNotFound", getErr)
	}
	if deleteErr := service.Delete(context.Background(), "id"); !errorspkg.Is(deleteErr, productport.ErrNotFound) {
		t.Fatalf("Delete() error = %v, want productport.ErrNotFound", deleteErr)
	}
}

// TestCopyDatasheets verifies helper copy behavior.
func TestCopyDatasheets(t *testing.T) {
	values := []productdomain.Datasheet{{Realm: "default", Name: "Tee"}}
	copied := CopyDatasheets(values)
	if len(copied) != 1 {
		t.Fatalf("len(copied) = %d, want %d", len(copied), 1)
	}
	if &copied[0] == &values[0] {
		t.Fatalf("expected copied slice backing values")
	}
}

// TestUniqueGalleryAssetIDs verifies gallery-id normalization behavior.
func TestUniqueGalleryAssetIDs(t *testing.T) {
	ids := uniqueGalleryAssetIDs([]productdomain.GalleryItem{{AssetID: " a1 "}, {AssetID: "a1"}, {AssetID: "a2"}, {AssetID: " "}})
	if len(ids) != 2 {
		t.Fatalf("len(ids) = %d, want %d", len(ids), 2)
	}
}
