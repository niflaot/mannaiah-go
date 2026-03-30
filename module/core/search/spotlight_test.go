package search

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeProvider implements SpotlightProvider for testing.
type fakeProvider struct {
	typeName string
	hits     []SpotlightHit
	err      error
	delay    time.Duration
}

func (f *fakeProvider) SpotlightSearch(ctx context.Context, term string, limit int) ([]SpotlightHit, error) {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return f.hits, f.err
}

func (f *fakeProvider) SpotlightType() string { return f.typeName }

// TestSpotlightServiceSearch verifies basic spotlight fan-out and merge.
func TestSpotlightServiceSearch(t *testing.T) {
	p1 := &fakeProvider{
		typeName: "contact",
		hits: []SpotlightHit{
			{Type: "contact", ID: "1", Title: "John", Score: 1.0},
		},
	}
	p2 := &fakeProvider{
		typeName: "order",
		hits: []SpotlightHit{
			{Type: "order", ID: "2", Title: "ORD-001", Score: 0.5},
		},
	}

	svc := NewSpotlightService(2*time.Second, p1, p2)
	result := svc.Search(context.Background(), "john", nil, 5)

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if result.Results[0].Score < result.Results[1].Score {
		t.Error("results should be sorted by score descending")
	}
	if result.Meta.Term != "john" {
		t.Errorf("meta.term = %q, want %q", result.Meta.Term, "john")
	}
	if result.Meta.Counts["contact"] != 1 {
		t.Errorf("counts[contact] = %d, want 1", result.Meta.Counts["contact"])
	}
	if result.Meta.Counts["order"] != 1 {
		t.Errorf("counts[order] = %d, want 1", result.Meta.Counts["order"])
	}
}

// TestSpotlightServiceTypeFilter verifies type filtering.
func TestSpotlightServiceTypeFilter(t *testing.T) {
	p1 := &fakeProvider{typeName: "contact", hits: []SpotlightHit{{Type: "contact", ID: "1"}}}
	p2 := &fakeProvider{typeName: "order", hits: []SpotlightHit{{Type: "order", ID: "2"}}}

	svc := NewSpotlightService(2*time.Second, p1, p2)
	result := svc.Search(context.Background(), "test", []string{"contact"}, 5)

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result (type filter), got %d", len(result.Results))
	}
	if result.Results[0].Type != "contact" {
		t.Errorf("type = %q, want contact", result.Results[0].Type)
	}
}

// TestSpotlightServiceErrorProvider verifies error in one provider does not break others.
func TestSpotlightServiceErrorProvider(t *testing.T) {
	good := &fakeProvider{typeName: "contact", hits: []SpotlightHit{{Type: "contact", ID: "1", Score: 1.0}}}
	bad := &fakeProvider{typeName: "order", err: errors.New("db error")}

	svc := NewSpotlightService(2*time.Second, good, bad)
	result := svc.Search(context.Background(), "test", nil, 5)

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result (failed provider skipped), got %d", len(result.Results))
	}
}

// TestSpotlightServiceTimeout verifies slow providers are excluded.
func TestSpotlightServiceTimeout(t *testing.T) {
	fast := &fakeProvider{typeName: "contact", hits: []SpotlightHit{{Type: "contact", ID: "1"}}}
	slow := &fakeProvider{typeName: "order", hits: []SpotlightHit{{Type: "order", ID: "2"}}, delay: 5 * time.Second}

	svc := NewSpotlightService(100*time.Millisecond, fast, slow)
	result := svc.Search(context.Background(), "test", nil, 5)

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result (slow provider timed out), got %d", len(result.Results))
	}
	if result.Results[0].Type != "contact" {
		t.Errorf("type = %q, want contact", result.Results[0].Type)
	}
}

// TestSpotlightServiceAddProvider verifies runtime provider registration.
func TestSpotlightServiceAddProvider(t *testing.T) {
	svc := NewSpotlightService(2 * time.Second)
	svc.AddProvider(&fakeProvider{typeName: "contact", hits: []SpotlightHit{{Type: "contact", ID: "1"}}})
	svc.AddProvider(nil)

	result := svc.Search(context.Background(), "test", nil, 5)
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result after AddProvider, got %d", len(result.Results))
	}
}

// TestBuildTypeSet verifies type set construction.
func TestBuildTypeSet(t *testing.T) {
	m := buildTypeSet(nil)
	if m != nil {
		t.Error("expected nil for empty types")
	}

	m = buildTypeSet([]string{"a", "b"})
	if !m["a"] || !m["b"] {
		t.Errorf("expected both a and b in set, got %v", m)
	}
	if m["c"] {
		t.Error("unexpected key c in set")
	}
}

// TestSpotlightServiceDefaultTimeout verifies negative timeout defaults to 2s.
func TestSpotlightServiceDefaultTimeout(t *testing.T) {
	svc := NewSpotlightService(-1)
	if svc.timeout != 2*time.Second {
		t.Errorf("timeout = %v, want 2s", svc.timeout)
	}
}

// TestSpotlightServiceLimitClamping verifies limit normalization.
func TestSpotlightServiceLimitClamping(t *testing.T) {
	p := &fakeProvider{typeName: "contact", hits: []SpotlightHit{}}
	svc := NewSpotlightService(2*time.Second, p)

	result := svc.Search(context.Background(), "test", nil, 0)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	result = svc.Search(context.Background(), "test", nil, 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
