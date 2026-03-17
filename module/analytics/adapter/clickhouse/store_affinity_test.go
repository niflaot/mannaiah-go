package clickhouse

import "testing"

// TestStoreAdapterAffinity_NilSafe verifies that nil-safe guards on affinity methods do not panic.
func TestStoreAdapterAffinity_NilSafe(t *testing.T) {
	s := &StoreAdapter{client: nil}

	if _, err := s.GetTagAffinity(t.Context(), "c-1", 10, 0); err != nil {
		t.Errorf("GetTagAffinity(nil client) error = %v", err)
	}
	if _, err := s.GetCategoryAffinity(t.Context(), "c-1", 10, 0); err != nil {
		t.Errorf("GetCategoryAffinity(nil client) error = %v", err)
	}
	if _, err := s.GetVariationAffinity(t.Context(), "c-1", 10, 0); err != nil {
		t.Errorf("GetVariationAffinity(nil client) error = %v", err)
	}

	profile, err := s.GetProfile(t.Context(), "c-1", 10, 0)
	if err != nil {
		t.Errorf("GetProfile(nil client) error = %v", err)
	}
	if profile == nil {
		t.Errorf("GetProfile(nil client) returned nil")
	}
}

// TestStoreAdapterAffinity_RefreshNilSafe verifies that refresh methods are nil-safe.
func TestStoreAdapterAffinity_RefreshNilSafe(t *testing.T) {
	s := &StoreAdapter{client: nil}
	if err := s.RefreshTagMV(t.Context()); err != nil {
		t.Errorf("RefreshTagMV(nil) error = %v", err)
	}
	if err := s.RefreshCategoryMV(t.Context()); err != nil {
		t.Errorf("RefreshCategoryMV(nil) error = %v", err)
	}
	if err := s.RefreshVariationMV(t.Context()); err != nil {
		t.Errorf("RefreshVariationMV(nil) error = %v", err)
	}
}
