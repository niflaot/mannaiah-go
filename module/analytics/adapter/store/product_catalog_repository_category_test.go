package store

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openProductCatalogTestDB opens an in-memory sqlite database with minimal catalog schema.
func openProductCatalogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	statements := []string{
		"CREATE TABLE tags (id INTEGER PRIMARY KEY, name TEXT NOT NULL, deleted_at DATETIME NULL);",
		"CREATE TABLE product_tags (product_id TEXT NOT NULL, tag_id INTEGER NOT NULL);",
		"CREATE TABLE products (id TEXT PRIMARY KEY, price REAL NOT NULL DEFAULT 0, deleted_at DATETIME NULL);",
		"CREATE TABLE categories (id TEXT PRIMARY KEY, slug TEXT NOT NULL, name TEXT NOT NULL, parent_id TEXT NULL, include_children BOOLEAN NOT NULL DEFAULT 0, deleted_at DATETIME NULL);",
		"CREATE TABLE category_products (category_id TEXT NOT NULL, product_id TEXT NOT NULL);",
		"CREATE TABLE product_variation_links (product_id TEXT NOT NULL, variation_id TEXT NOT NULL);",
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec schema statement %q: %v", stmt, err)
		}
	}

	return db
}

// seedProductCatalogCategoryFixtures inserts one tagged product and category mappings used in category filter tests.
func seedProductCatalogCategoryFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()

	statements := []string{
		"INSERT INTO tags (id, name, deleted_at) VALUES (1, 'tier-1', NULL);",
		"INSERT INTO tags (id, name, deleted_at) VALUES (2, 'travel', NULL);",
		"INSERT INTO tags (id, name, deleted_at) VALUES (3, 'excluded', NULL);",
		"INSERT INTO products (id, price, deleted_at) VALUES ('p-1', 100, NULL);",
		"INSERT INTO products (id, price, deleted_at) VALUES ('p-2', 220, NULL);",
		"INSERT INTO product_tags (product_id, tag_id) VALUES ('p-1', 1);",
		"INSERT INTO product_tags (product_id, tag_id) VALUES ('p-2', 1);",
		"INSERT INTO product_tags (product_id, tag_id) VALUES ('p-1', 2);",
		"INSERT INTO product_tags (product_id, tag_id) VALUES ('p-2', 3);",
		"INSERT INTO categories (id, slug, name, parent_id, include_children, deleted_at) VALUES ('cat-1', 'morrales', 'Morrales', NULL, 1, NULL);",
		"INSERT INTO categories (id, slug, name, parent_id, include_children, deleted_at) VALUES ('cat-2', 'morrales-mini', 'Morrales Mini', 'cat-1', 0, NULL);",
		"INSERT INTO categories (id, slug, name, parent_id, include_children, deleted_at) VALUES ('cat-3', 'accesorios', 'Accesorios', NULL, 0, NULL);",
		"INSERT INTO category_products (category_id, product_id) VALUES ('cat-1', 'p-1');",
		"INSERT INTO category_products (category_id, product_id) VALUES ('cat-2', 'p-2');",
		"INSERT INTO category_products (category_id, product_id) VALUES ('cat-3', 'p-1');",
		"INSERT INTO category_products (category_id, product_id) VALUES ('legacy-cat', 'p-1');",
	}
	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec fixture statement %q: %v", stmt, err)
		}
	}
}

// TestResolveProductIDsSupportsCategoryIDSlugAndName verifies category filtering supports category id, slug, and display name values.
func TestResolveProductIDsSupportsCategoryIDSlugAndName(t *testing.T) {
	t.Parallel()

	db := openProductCatalogTestDB(t)
	seedProductCatalogCategoryFixtures(t, db)

	repo, err := NewProductCatalogRepository(db)
	if err != nil {
		t.Fatalf("NewProductCatalogRepository() error = %v", err)
	}

	tests := []struct {
		name       string
		categoryID string
		want       []string
	}{
		{name: "by id", categoryID: "cat-1", want: []string{"p-1", "p-2"}},
		{name: "by slug", categoryID: "morrales", want: []string{"p-1", "p-2"}},
		{name: "by name", categoryID: "Morrales", want: []string{"p-1", "p-2"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.resolveProductIDs(
				context.Background(),
				[]string{"tier-1"},
				"any",
				nil,
				tt.categoryID,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				10,
			)
			if err != nil {
				t.Fatalf("resolveProductIDs() error = %v", err)
			}
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("resolveProductIDs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// TestResolveProductIDsCategoryFallbackToRawReference verifies category filtering still works when category rows are missing but category_products has a raw ID.
func TestResolveProductIDsCategoryFallbackToRawReference(t *testing.T) {
	t.Parallel()

	db := openProductCatalogTestDB(t)
	seedProductCatalogCategoryFixtures(t, db)

	repo, err := NewProductCatalogRepository(db)
	if err != nil {
		t.Fatalf("NewProductCatalogRepository() error = %v", err)
	}

	got, err := repo.resolveProductIDs(
		context.Background(),
		[]string{"tier-1"},
		"any",
		nil,
		"legacy-cat",
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		10,
	)
	if err != nil {
		t.Fatalf("resolveProductIDs() error = %v", err)
	}
	if !reflect.DeepEqual(got, []string{"p-1"}) {
		t.Fatalf("resolveProductIDs() = %#v, want %#v", got, []string{"p-1"})
	}
}

// TestResolveProductIDsSupportsExtendedFilters verifies include/exclude categories, include/exclude tags, and price filtering.
func TestResolveProductIDsSupportsExtendedFilters(t *testing.T) {
	t.Parallel()

	db := openProductCatalogTestDB(t)
	seedProductCatalogCategoryFixtures(t, db)

	repo, err := NewProductCatalogRepository(db)
	if err != nil {
		t.Fatalf("NewProductCatalogRepository() error = %v", err)
	}

	min150 := 150.0
	max150 := 150.0
	min90 := 90.0

	tests := []struct {
		name               string
		categoryIDs        []string
		excludeCategoryIDs []string
		includeTags        []string
		excludeTags        []string
		minPrice           *float64
		maxPrice           *float64
		want               []string
	}{
		{
			name:               "include and exclude categories",
			categoryIDs:        []string{"cat-1"},
			excludeCategoryIDs: []string{"cat-2"},
			want:               []string{"p-1"},
		},
		{
			name:        "include tags",
			includeTags: []string{"travel"},
			want:        []string{"p-1"},
		},
		{
			name:        "exclude tags",
			excludeTags: []string{"excluded"},
			want:        []string{"p-1"},
		},
		{
			name:     "min price",
			minPrice: &min150,
			want:     []string{"p-2"},
		},
		{
			name:     "max price",
			maxPrice: &max150,
			want:     []string{"p-1"},
		},
		{
			name:     "price range",
			minPrice: &min90,
			maxPrice: &max150,
			want:     []string{"p-1"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.resolveProductIDs(
				context.Background(),
				[]string{"tier-1"},
				"any",
				nil,
				"",
				tt.categoryIDs,
				tt.excludeCategoryIDs,
				tt.includeTags,
				tt.excludeTags,
				tt.minPrice,
				tt.maxPrice,
				nil,
				nil,
				10,
			)
			if err != nil {
				t.Fatalf("resolveProductIDs() error = %v", err)
			}
			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("resolveProductIDs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
