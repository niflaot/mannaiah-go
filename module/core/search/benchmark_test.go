package search

import (
	"context"
	"testing"
	"time"
)

// BenchmarkScoreResults benchmarks in-memory relevance scoring.
func BenchmarkScoreResults(b *testing.B) {
	entities := make([]string, 100)
	for i := range entities {
		entities[i] = "entity_" + string(rune('a'+i%26))
	}
	extract := func(e string, _ string) string { return e }
	primary := []string{"name"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScoreResults(entities, "entity_a", primary, nil, extract)
	}
}

// BenchmarkCacheKeyGeneration benchmarks deterministic cache key computation.
func BenchmarkCacheKeyGeneration(b *testing.B) {
	inner := &mockRepo{}
	store := newMockStore()
	cfg := CacheConfig{Enabled: true, TTL: 60 * time.Second, KeyPrefix: "bench"}
	cached := NewCachedRepository[string](inner, store, cfg)

	q := Query{
		Term: "benchmark",
		Filters: []Filter{
			{Field: "status", Operator: OpEQ, Value: "active"},
			{Field: "created_at", Operator: OpGTE, Value: "2024-01-01"},
		},
		Sort:     []SortField{{Field: "created_at", Direction: Desc}},
		Page:     1,
		PageSize: 20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cached.cacheKey(q)
	}
}

// BenchmarkSpotlightFanout benchmarks concurrent spotlight fan-out.
func BenchmarkSpotlightFanout(b *testing.B) {
	providers := make([]SpotlightProvider, 9)
	for i := range providers {
		hits := make([]SpotlightHit, 10)
		for j := range hits {
			hits[j] = SpotlightHit{Type: "type", ID: "id", Score: float64(j)}
		}
		providers[i] = &fakeProvider{typeName: "type", hits: hits}
	}
	svc := NewSpotlightService(2*time.Second, providers...)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Search(ctx, "benchmark", nil, 10)
	}
}
