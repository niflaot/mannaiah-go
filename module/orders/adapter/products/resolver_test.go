package products

import (
	"context"
	"errors"
	"testing"

	coredatabase "mannaiah/module/core/database"
)

// TestNewResolverValidation verifies constructor validation behavior.
func TestNewResolverValidation(t *testing.T) {
	_, err := NewResolver(nil)
	if !errors.Is(err, ErrNilDB) {
		t.Fatalf("NewResolver(nil) error = %v, want ErrNilDB", err)
	}
}

// TestResolve verifies SKU and alternate-name resolution behavior.
func TestResolve(t *testing.T) {
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("coredatabase.Open() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	if execErr := db.Exec(`
		CREATE TABLE products (
			id TEXT PRIMARY KEY,
			sku TEXT NOT NULL UNIQUE
		);
	`).Error; execErr != nil {
		t.Fatalf("create products table error = %v", execErr)
	}
	if execErr := db.Exec(`
		CREATE TABLE product_datasheets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id TEXT NOT NULL,
			name TEXT NOT NULL
		);
	`).Error; execErr != nil {
		t.Fatalf("create product_datasheets table error = %v", execErr)
	}

	if execErr := db.Exec(`INSERT INTO products (id, sku) VALUES ('p-1', 'SKU-1'), ('p-2', 'SKU-2')`).Error; execErr != nil {
		t.Fatalf("seed products error = %v", execErr)
	}
	if execErr := db.Exec(`INSERT INTO product_datasheets (product_id, name) VALUES ('p-2', 'Fallback Name')`).Error; execErr != nil {
		t.Fatalf("seed product_datasheets error = %v", execErr)
	}

	resolver, err := NewResolver(db)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	skuMatch, err := resolver.Resolve(context.Background(), "SKU-1", "")
	if err != nil {
		t.Fatalf("Resolve(sku) error = %v", err)
	}
	if skuMatch == nil || skuMatch.ProductID != "p-1" || skuMatch.MatchedBy != "sku" {
		t.Fatalf("Resolve(sku) = %#v, want p-1/sku", skuMatch)
	}

	altMatch, err := resolver.Resolve(context.Background(), "MISSING", "fallback name")
	if err != nil {
		t.Fatalf("Resolve(alt) error = %v", err)
	}
	if altMatch == nil || altMatch.ProductID != "p-2" || altMatch.MatchedBy != "alternate_name" {
		t.Fatalf("Resolve(alt) = %#v, want p-2/alternate_name", altMatch)
	}

	none, err := resolver.Resolve(context.Background(), "MISSING", "UNKNOWN")
	if err != nil {
		t.Fatalf("Resolve(none) error = %v", err)
	}
	if none != nil {
		t.Fatalf("Resolve(none) = %#v, want nil", none)
	}
}
