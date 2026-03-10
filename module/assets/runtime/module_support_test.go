package runtime

import (
	"testing"
	"time"
)

// TestResolveWorkerTags verifies worker-tag parsing behavior.
func TestResolveWorkerTags(t *testing.T) {
	resolved := resolveWorkerTags(" marketplaces,feeds, MARKETPLACES , ,ops ")
	if len(resolved) != 3 {
		t.Fatalf("len(resolveWorkerTags) = %d, want %d", len(resolved), 3)
	}
	if resolved[0] != "marketplaces" || resolved[1] != "feeds" || resolved[2] != "ops" {
		t.Fatalf("resolveWorkerTags() = %#v, want [marketplaces feeds ops]", resolved)
	}
}

// TestResolveWorkerTimeout verifies timeout normalization behavior.
func TestResolveWorkerTimeout(t *testing.T) {
	if resolved := resolveWorkerTimeout(0); resolved != defaultWorkerTimeout {
		t.Fatalf("resolveWorkerTimeout(0) = %v, want %v", resolved, defaultWorkerTimeout)
	}
	if resolved := resolveWorkerTimeout(1500); resolved != 1500*time.Millisecond {
		t.Fatalf("resolveWorkerTimeout(1500) = %v, want %v", resolved, 1500*time.Millisecond)
	}
}
