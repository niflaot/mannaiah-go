package http

import "testing"

// TestPaths verifies shipping path coverage.
func TestPaths(t *testing.T) {
	paths := Paths()
	if paths == nil {
		t.Fatalf("Paths() returned nil")
	}
	if paths.Find("/shipping/marks") == nil {
		t.Fatalf("missing /shipping/marks path")
	}
	if paths.Find("/shipping/batches/{id}/close") == nil {
		t.Fatalf("missing /shipping/batches/{id}/close path")
	}
}
