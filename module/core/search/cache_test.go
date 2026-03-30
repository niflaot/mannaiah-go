package search

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockStore is a minimal in-memory implementation of cache.Store for testing.
type mockStore struct {
	mu   sync.Mutex
	data map[string]string
}

func newMockStore() *mockStore {
	return &mockStore{data: make(map[string]string)}
}

func (m *mockStore) Ping(_ context.Context) error { return nil }
func (m *mockStore) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	if !ok {
		return "", errors.New("miss")
	}
	return v, nil
}
func (m *mockStore) Set(_ context.Context, key, value string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}
func (m *mockStore) Delete(_ context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[key]; ok {
		delete(m.data, key)
		return 1, nil
	}
	return 0, nil
}
func (m *mockStore) Keys(_ context.Context, pattern string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var keys []string
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}
func (m *mockStore) GetByPattern(_ context.Context, _ string) (map[string]string, error) {
	return nil, nil
}
func (m *mockStore) Close() error { return nil }

// mockRepo is a search repository for testing.
type mockRepo struct {
	calls int
}

func (r *mockRepo) Search(_ context.Context, q Query) (*Result[string], error) {
	r.calls++
	return NewResult([]string{"item1", "item2"}, 2, q.Page, q.PageSize), nil
}

// TestCachedRepositoryDisabled verifies pass-through when cache is disabled.
func TestCachedRepositoryDisabled(t *testing.T) {
	inner := &mockRepo{}
	cached := NewCachedRepository[string](inner, newMockStore(), DefaultCacheConfig())

	ctx := context.Background()
	q := Query{Term: "test", Page: 1, PageSize: 20}

	_, err := cached.Search(ctx, q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = cached.Search(ctx, q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inner.calls != 2 {
		t.Errorf("inner.calls = %d, want 2 (cache disabled)", inner.calls)
	}
}

// TestCachedRepositoryEnabled verifies cache hit on second call.
func TestCachedRepositoryEnabled(t *testing.T) {
	inner := &mockRepo{}
	store := newMockStore()
	cfg := CacheConfig{Enabled: true, TTL: 60 * time.Second, KeyPrefix: "test"}
	cached := NewCachedRepository[string](inner, store, cfg)

	ctx := context.Background()
	q := Query{Term: "test", Page: 1, PageSize: 20}

	r1, err := cached.Search(ctx, q)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	r2, err := cached.Search(ctx, q)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if inner.calls != 1 {
		t.Errorf("inner.calls = %d, want 1 (second call should hit cache)", inner.calls)
	}
	if r1.Total != r2.Total {
		t.Errorf("cached result differs: total %d vs %d", r1.Total, r2.Total)
	}
}

// TestCachedRepositoryDifferentQueries verifies different queries miss cache.
func TestCachedRepositoryDifferentQueries(t *testing.T) {
	inner := &mockRepo{}
	store := newMockStore()
	cfg := CacheConfig{Enabled: true, TTL: 60 * time.Second, KeyPrefix: "test"}
	cached := NewCachedRepository[string](inner, store, cfg)

	ctx := context.Background()
	q1 := Query{Term: "alpha", Page: 1, PageSize: 20}
	q2 := Query{Term: "beta", Page: 1, PageSize: 20}

	_, _ = cached.Search(ctx, q1)
	_, _ = cached.Search(ctx, q2)

	if inner.calls != 2 {
		t.Errorf("inner.calls = %d, want 2 (different queries)", inner.calls)
	}
}

// TestCachedRepositoryNilStore verifies pass-through when store is nil.
func TestCachedRepositoryNilStore(t *testing.T) {
	inner := &mockRepo{}
	cfg := CacheConfig{Enabled: true, TTL: 60 * time.Second, KeyPrefix: "test"}
	cached := NewCachedRepository[string](inner, nil, cfg)

	ctx := context.Background()
	q := Query{Term: "test", Page: 1, PageSize: 20}

	_, err := cached.Search(ctx, q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.calls != 1 {
		t.Errorf("inner.calls = %d, want 1", inner.calls)
	}
}

// TestCacheKeyDeterministic verifies same query produces same cache key.
func TestCacheKeyDeterministic(t *testing.T) {
	inner := &mockRepo{}
	store := newMockStore()
	cfg := CacheConfig{Enabled: true, TTL: 60 * time.Second, KeyPrefix: "det"}
	cached := NewCachedRepository[string](inner, store, cfg)

	q := Query{
		Term:     "test",
		Filters:  []Filter{{Field: "status", Operator: OpEQ, Value: "active"}},
		Sort:     []SortField{{Field: "created_at", Direction: Desc}},
		Page:     1,
		PageSize: 20,
	}

	k1 := cached.cacheKey(q)
	k2 := cached.cacheKey(q)

	if k1 != k2 {
		t.Errorf("cache keys differ for identical query: %q vs %q", k1, k2)
	}
}

// TestNormalizeQueryForKey verifies canonical form of queries.
func TestNormalizeQueryForKey(t *testing.T) {
	q := Query{
		Term:     "  Hello  ",
		Filters:  []Filter{{Field: "b", Operator: OpEQ, Value: "2"}, {Field: "a", Operator: OpEQ, Value: "1"}},
		Sort:     []SortField{{Field: "created_at", Direction: Desc}},
		Page:     1,
		PageSize: 20,
	}

	n := normalizeQueryForKey(q)
	if n.Term != "hello" {
		t.Errorf("term = %q, want %q", n.Term, "hello")
	}

	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	n2 := normalizeQueryForKey(q)
	data2, _ := json.Marshal(n2)
	if string(data) != string(data2) {
		t.Errorf("normalized forms differ")
	}
}

// TestInvalidateRemovesEntries verifies Invalidate clears cached entries.
func TestInvalidateRemovesEntries(t *testing.T) {
	inner := &mockRepo{}
	store := newMockStore()
	cfg := CacheConfig{Enabled: true, TTL: 60 * time.Second, KeyPrefix: "inv"}
	cached := NewCachedRepository[string](inner, store, cfg)

	ctx := context.Background()
	q := Query{Term: "test", Page: 1, PageSize: 20}
	_, _ = cached.Search(ctx, q)

	if len(store.data) == 0 {
		t.Fatal("expected cached entry after search")
	}

	cached.Invalidate(ctx)

	if len(store.data) != 0 {
		t.Errorf("expected empty store after invalidate, got %d entries", len(store.data))
	}
}
