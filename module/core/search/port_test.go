package search

import (
	"testing"
)

// TestNormalizePagination verifies page/pageSize defaults and clamping.
func TestNormalizePagination(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
	}{
		{"defaults for zero", 0, 0, 1, 20},
		{"defaults for negative", -1, -5, 1, 20},
		{"valid values pass through", 3, 50, 3, 50},
		{"page size clamped to max", 1, 200, 1, 100},
		{"page size at max boundary", 1, 100, 1, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ps := NormalizePagination(tt.page, tt.pageSize)
			if p != tt.wantPage {
				t.Errorf("page = %d, want %d", p, tt.wantPage)
			}
			if ps != tt.wantPageSize {
				t.Errorf("pageSize = %d, want %d", ps, tt.wantPageSize)
			}
		})
	}
}

// TestNewResultNilData ensures nil data is replaced with an empty slice.
func TestNewResultNilData(t *testing.T) {
	r := NewResult[string](nil, 0, 1, 20)
	if r.Data == nil {
		t.Fatal("expected non-nil data slice")
	}
	if len(r.Data) != 0 {
		t.Fatalf("expected empty data, got %d elements", len(r.Data))
	}
}

// TestNewResultTotalPages verifies total page computation.
func TestNewResultTotalPages(t *testing.T) {
	r := NewResult[string]([]string{"a", "b"}, 45, 1, 20)
	if r.TotalPages != 3 {
		t.Errorf("totalPages = %d, want 3", r.TotalPages)
	}
	if r.Total != 45 {
		t.Errorf("total = %d, want 45", r.Total)
	}
}

// TestNewResultZeroTotal verifies zero total gives zero pages.
func TestNewResultZeroTotal(t *testing.T) {
	r := NewResult[string]([]string{}, 0, 1, 20)
	if r.TotalPages != 0 {
		t.Errorf("totalPages = %d, want 0", r.TotalPages)
	}
}
