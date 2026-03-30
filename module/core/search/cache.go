package search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"mannaiah/module/core/cache"
)

// CacheConfig configures search result caching behavior.
type CacheConfig struct {
	// Enabled controls whether search caching is active.
	Enabled bool
	// TTL is the time-to-live for cached search results.
	TTL time.Duration
	// KeyPrefix is prepended to all cache keys.
	KeyPrefix string
}

// DefaultCacheConfig returns a production-ready default cache configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:   false,
		TTL:       60 * time.Second,
		KeyPrefix: "search",
	}
}

// CachedRepository wraps a SearchRepository with cache-aside reads.
type CachedRepository[T any] struct {
	inner  Repository[T]
	store  cache.Store
	config CacheConfig
}

// NewCachedRepository creates a cache-decorated search repository.
func NewCachedRepository[T any](inner Repository[T], store cache.Store, config CacheConfig) *CachedRepository[T] {
	return &CachedRepository[T]{
		inner:  inner,
		store:  store,
		config: config,
	}
}

// Search checks the cache first; on miss, delegates to the inner repository and caches the result.
func (r *CachedRepository[T]) Search(ctx context.Context, query Query) (*Result[T], error) {
	if !r.config.Enabled || r.store == nil {
		return r.inner.Search(ctx, query)
	}

	key := r.cacheKey(query)

	cached, err := r.store.Get(ctx, key)
	if err == nil && cached != "" {
		var result Result[T]
		if jsonErr := json.Unmarshal([]byte(cached), &result); jsonErr == nil {
			return &result, nil
		}
	}

	result, err := r.inner.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	if encoded, jsonErr := json.Marshal(result); jsonErr == nil {
		_ = r.store.Set(ctx, key, string(encoded), r.config.TTL)
	}

	return result, nil
}

// Invalidate removes all cached entries for the configured resource prefix.
func (r *CachedRepository[T]) Invalidate(ctx context.Context) {
	if !r.config.Enabled || r.store == nil {
		return
	}
	pattern := r.config.KeyPrefix + ":*"
	keys, err := r.store.Keys(ctx, pattern)
	if err != nil {
		return
	}
	for _, k := range keys {
		_, _ = r.store.Delete(ctx, k)
	}
}

// cacheKey builds a deterministic cache key from the search query.
func (r *CachedRepository[T]) cacheKey(query Query) string {
	normalized := normalizeQueryForKey(query)
	data, _ := json.Marshal(normalized)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%s:%s", r.config.KeyPrefix, hex.EncodeToString(hash[:]))
}

// cacheableQuery is a serialization-friendly representation of a search query for key hashing.
type cacheableQuery struct {
	Term     string   `json:"t"`
	Filters  []string `json:"f"`
	Sort     []string `json:"s"`
	Page     int      `json:"p"`
	PageSize int      `json:"ps"`
}

// normalizeQueryForKey builds a canonical representation of the query for caching.
func normalizeQueryForKey(query Query) cacheableQuery {
	filters := make([]string, 0, len(query.Filters))
	for _, f := range query.Filters {
		filters = append(filters, fmt.Sprintf("%s:%s:%v", f.Field, f.Operator, f.Value))
	}
	sort.Strings(filters)

	sorts := make([]string, 0, len(query.Sort))
	for _, s := range query.Sort {
		sorts = append(sorts, fmt.Sprintf("%s:%s", s.Field, s.Direction))
	}

	return cacheableQuery{
		Term:     strings.ToLower(strings.TrimSpace(query.Term)),
		Filters:  filters,
		Sort:     sorts,
		Page:     query.Page,
		PageSize: query.PageSize,
	}
}
