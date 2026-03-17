package clickhouse

import (
	"testing"
	"time"

	"mannaiah/module/analytics/port"
)

// TestUpsertProductTaxonomy_NilSafe verifies nil-safety of UpsertProductTaxonomy.
func TestUpsertProductTaxonomy_NilSafe(t *testing.T) {
	s := &StoreAdapter{client: nil}
	if err := s.UpsertProductTaxonomy(t.Context(), []port.ProductTaxonomyRow{
		{ProductID: "p-1", Tag: "coffee", UpdatedAt: time.Now()},
	}); err != nil {
		t.Errorf("UpsertProductTaxonomy(nil client) error = %v", err)
	}
}

// TestUpsertVariationTaxonomy_NilSafe verifies nil-safety of UpsertVariationTaxonomy.
func TestUpsertVariationTaxonomy_NilSafe(t *testing.T) {
	s := &StoreAdapter{client: nil}
	if err := s.UpsertVariationTaxonomy(t.Context(), []port.VariationTaxonomyRow{
		{ProductID: "p-1", SKU: "SKU-1", VariationID: "v-1", VariationName: "color", VariationValue: "black", UpdatedAt: time.Now()},
	}); err != nil {
		t.Errorf("UpsertVariationTaxonomy(nil client) error = %v", err)
	}
}

// TestUpsertProductTaxonomy_Empty verifies no-op on empty rows.
func TestUpsertProductTaxonomy_Empty(t *testing.T) {
	s := &StoreAdapter{client: nil}
	if err := s.UpsertProductTaxonomy(t.Context(), nil); err != nil {
		t.Errorf("UpsertProductTaxonomy(empty) error = %v", err)
	}
}
