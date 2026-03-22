package http

import "testing"

// TestMapProductBlockRequestsEmptySlice verifies explicit empty arrays remain explicit empty slices.
func TestMapProductBlockRequestsEmptySlice(t *testing.T) {
	t.Parallel()

	mapped := mapProductBlockRequests([]productBlockRequest{})
	if mapped == nil {
		t.Fatalf("mapProductBlockRequests(empty) returned nil, want non-nil empty slice")
	}
	if len(mapped) != 0 {
		t.Fatalf("mapProductBlockRequests(empty) len = %d, want 0", len(mapped))
	}
}
