package service

import (
	"context"
	"errors"
	"testing"
	"time"

	corecache "mannaiah/module/core/cache"
	categorydomain "mannaiah/module/products/domain/category"
	productdomain "mannaiah/module/products/domain/product"
	storefrontdomain "mannaiah/module/products/domain/storefront"
)

// sourceMock defines storefront navigation source behavior for tests.
type sourceMock struct {
	// treeFn defines root-category lookup behavior.
	treeFn func(ctx context.Context) ([]*categorydomain.Category, error)
	// childrenFn defines child-category lookup behavior.
	childrenFn func(ctx context.Context, parentID string) ([]*categorydomain.Category, error)
	// productsFn defines category product lookup behavior.
	productsFn func(ctx context.Context, categoryID string) ([]*productdomain.Product, error)
	// pagesFn defines static-page lookup behavior.
	pagesFn func(ctx context.Context) ([]storefrontdomain.StaticPageNode, error)
}

// Tree executes configured root-category lookup behavior.
func (m sourceMock) Tree(ctx context.Context) ([]*categorydomain.Category, error) {
	return m.treeFn(ctx)
}

// Children executes configured child-category lookup behavior.
func (m sourceMock) Children(ctx context.Context, parentID string) ([]*categorydomain.Category, error) {
	return m.childrenFn(ctx, parentID)
}

// ListProducts executes configured category-product lookup behavior.
func (m sourceMock) ListProducts(ctx context.Context, categoryID string) ([]*productdomain.Product, error) {
	return m.productsFn(ctx, categoryID)
}

// ListStaticPages executes configured static-page lookup behavior.
func (m sourceMock) ListStaticPages(ctx context.Context) ([]storefrontdomain.StaticPageNode, error) {
	if m.pagesFn != nil {
		return m.pagesFn(ctx)
	}

	return []storefrontdomain.StaticPageNode{}, nil
}

// cacheStoreMock defines in-memory cache behavior for tests.
type cacheStoreMock struct {
	// values stores cache entries by key.
	values map[string]string
	// lastTTL stores the latest TTL written per key.
	lastTTL map[string]time.Duration
}

// Ping returns successful availability for tests.
func (m *cacheStoreMock) Ping(ctx context.Context) error { return nil }

// Get resolves one cache value by key.
func (m *cacheStoreMock) Get(ctx context.Context, key string) (string, error) {
	value, ok := m.values[key]
	if !ok {
		return "", errors.New("not found")
	}

	return value, nil
}

// Set writes one cache value by key.
func (m *cacheStoreMock) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if m.values == nil {
		m.values = map[string]string{}
	}
	if m.lastTTL == nil {
		m.lastTTL = map[string]time.Duration{}
	}
	m.values[key] = value
	m.lastTTL[key] = ttl
	return nil
}

// Delete removes one cache key.
func (m *cacheStoreMock) Delete(ctx context.Context, key string) (int64, error) { return 1, nil }

// Keys returns no-op key lists for tests.
func (m *cacheStoreMock) Keys(ctx context.Context, pattern string) ([]string, error) { return nil, nil }

// GetByPattern returns no-op key-value entries for tests.
func (m *cacheStoreMock) GetByPattern(ctx context.Context, pattern string) (map[string]string, error) {
	return nil, nil
}

// Close returns successful close behavior for tests.
func (m *cacheStoreMock) Close() error { return nil }

var _ corecache.Store = (*cacheStoreMock)(nil)

// TestNewServiceRejectsNilSource verifies storefront navigation source validation.
func TestNewServiceRejectsNilSource(t *testing.T) {
	if _, err := NewService(nil, nil, Config{}, nil); !errors.Is(err, ErrNilSource) {
		t.Fatalf("NewService() error = %v, want ErrNilSource", err)
	}
}

// TestGetUsesCache verifies cached storefront navigation hits are returned without regeneration.
func TestGetUsesCache(t *testing.T) {
	store := &cacheStoreMock{
		values: map[string]string{
			"products:storefront:navigation:default": `{"realm":"default","generatedAt":"2026-01-01T00:00:00Z","categories":[]}`,
		},
	}
	service, err := NewService(sourceMock{
		treeFn: func(ctx context.Context) ([]*categorydomain.Category, error) {
			t.Fatalf("Tree() should not be called on cache hit")
			return nil, nil
		},
		childrenFn: func(ctx context.Context, parentID string) ([]*categorydomain.Category, error) {
			return nil, nil
		},
		productsFn: func(ctx context.Context, categoryID string) ([]*productdomain.Product, error) {
			return nil, nil
		},
		pagesFn: func(ctx context.Context) ([]storefrontdomain.StaticPageNode, error) {
			return nil, nil
		},
	}, store, Config{}, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	snapshot, getErr := service.Get(context.Background())
	if getErr != nil {
		t.Fatalf("Get() error = %v", getErr)
	}
	if snapshot.Realm != "default" {
		t.Fatalf("snapshot.Realm = %q, want default", snapshot.Realm)
	}
}

// TestRegenerateBuildsAndCachesNavigation verifies live navigation regeneration behavior.
func TestRegenerateBuildsAndCachesNavigation(t *testing.T) {
	rootCreatedAt := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	productCreatedAt := rootCreatedAt.Add(time.Hour)
	store := &cacheStoreMock{}
	service, err := NewService(sourceMock{
		treeFn: func(ctx context.Context) ([]*categorydomain.Category, error) {
			return []*categorydomain.Category{{
				ID:        "cat-1",
				Name:      "Morrales Nuevos",
				CreatedAt: rootCreatedAt,
				UpdatedAt: rootCreatedAt,
			}}, nil
		},
		childrenFn: func(ctx context.Context, parentID string) ([]*categorydomain.Category, error) {
			return nil, nil
		},
		productsFn: func(ctx context.Context, categoryID string) ([]*productdomain.Product, error) {
			return []*productdomain.Product{{
				ID:        "prod-1",
				SKU:       "sku-1",
				CreatedAt: productCreatedAt,
				UpdatedAt: productCreatedAt,
				Datasheets: []productdomain.Datasheet{{
					Realm: "default",
					Name:  "Dream Nubuk Negro",
					Attributes: map[string]any{
						"storefronturl": "dream-nubuk-negro",
					},
				}},
			}}, nil
		},
		pagesFn: func(ctx context.Context) ([]storefrontdomain.StaticPageNode, error) {
			return []storefrontdomain.StaticPageNode{{
				ID:           "page-1",
				RenderableID: "renderable-1",
				Title:        "About",
				URL:          "/about",
				CreatedAt:    rootCreatedAt.Add(2 * time.Hour),
				UpdatedAt:    rootCreatedAt.Add(2 * time.Hour),
			}}, nil
		},
	}, store, Config{}, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	snapshot, regenerateErr := service.Regenerate(context.Background())
	if regenerateErr != nil {
		t.Fatalf("Regenerate() error = %v", regenerateErr)
	}
	if len(snapshot.Categories) != 1 {
		t.Fatalf("len(snapshot.Categories) = %d, want 1", len(snapshot.Categories))
	}
	if snapshot.Categories[0].Slug != "morrales-nuevos" {
		t.Fatalf("category slug = %q, want morrales-nuevos", snapshot.Categories[0].Slug)
	}
	if got := snapshot.Categories[0].Path; got != "/collections/morrales-nuevos" {
		t.Fatalf("category path = %q, want /collections/morrales-nuevos", got)
	}
	if got := snapshot.Categories[0].Products[0].Path; got != "/product/dream-nubuk-negro" {
		t.Fatalf("product path = %q, want /product/dream-nubuk-negro", got)
	}
	if len(snapshot.StaticPages) != 1 {
		t.Fatalf("len(snapshot.StaticPages) = %d, want 1", len(snapshot.StaticPages))
	}
	if got := snapshot.StaticPages[0].RenderableID; got != "renderable-1" {
		t.Fatalf("static page renderable id = %q, want renderable-1", got)
	}
	if got := snapshot.StaticPages[0].URL; got != "/about" {
		t.Fatalf("static page url = %q, want /about", got)
	}
	if ttl := store.lastTTL["products:storefront:navigation:default"]; ttl != 24*time.Hour {
		t.Fatalf("cache ttl = %s, want 24h", ttl)
	}
}

// TestRegenerateExtendsStaleSnapshotOnFailure verifies stale snapshot extension behavior.
func TestRegenerateExtendsStaleSnapshotOnFailure(t *testing.T) {
	store := &cacheStoreMock{
		values: map[string]string{
			"products:storefront:navigation:default": `{"realm":"default","generatedAt":"2026-01-01T00:00:00Z","categories":[]}`,
		},
	}
	service, err := NewService(sourceMock{
		treeFn: func(ctx context.Context) ([]*categorydomain.Category, error) {
			return nil, errors.New("boom")
		},
		childrenFn: func(ctx context.Context, parentID string) ([]*categorydomain.Category, error) {
			return nil, nil
		},
		productsFn: func(ctx context.Context, categoryID string) ([]*productdomain.Product, error) {
			return nil, nil
		},
		pagesFn: func(ctx context.Context) ([]storefrontdomain.StaticPageNode, error) {
			return nil, nil
		},
	}, store, Config{}, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, regenerateErr := service.Regenerate(context.Background()); regenerateErr == nil {
		t.Fatalf("Regenerate() error = nil, want non-nil")
	}
	if ttl := store.lastTTL["products:storefront:navigation:default"]; ttl != 12*time.Hour {
		t.Fatalf("failure-extension ttl = %s, want 12h", ttl)
	}
}
