# Search Caching & Performance

## Cache Architecture

The search module provides an optional Redis-backed cache-aside decorator via `CachedRepository[T]`.

### How It Works

```
Request → CachedRepository → Cache hit?
                              ├── Yes → Return cached result
                              └── No  → Repository.Search()
                                         │
                                         ▼
                                    Store in Redis (TTL)
                                         │
                                         ▼
                                    Return result
```

### Cache Configuration

```go
type CacheConfig struct {
    Enabled   bool          // Enable cache (default: false)
    TTL       time.Duration // Cache entry TTL (default: 60s)
    KeyPrefix string        // Redis key prefix (e.g. "contacts")
}
```

### Cache Key Generation

Keys use deterministic SHA256 hashing of the serialized query:

```
search:<prefix>:<sha256(json(query))>
```

Example: `search:contacts:a1b2c3d4e5f6...`

This ensures:
- Identical queries produce identical keys (cache hit)
- Different queries never collide
- Prefix-based invalidation is straightforward

### Invalidation Strategy

- **TTL-based:** Entries expire automatically after the configured TTL
- **Prefix purge:** Use `cache.Store.GetByPattern("search:<prefix>:*")` + `Delete` to flush a resource's cache
- **Full purge:** Delete all keys matching `search:*`

### Enabling Cache per Resource

In `main.go`, wrap any `Repository[T]` with `CachedRepository`:

```go
contactCfg := coresearch.CacheConfig{
    Enabled:   true,
    TTL:       30 * time.Second,
    KeyPrefix: "contacts",
}
cachedContactRepo := coresearch.NewCachedRepository(contactSearchRepo, cacheStore, contactCfg)
router.Get("/search/contacts", coresearch.SearchHandlerFunc(cachedContactRepo))
```

---

## Performance Characteristics

### Database Indexes

Migration `000041` (MySQL) / `000040` (SQLite) adds B-tree indexes on all text-searchable and frequently filtered columns:

| Table | Indexed Columns |
|-------|----------------|
| contacts | first_name, last_name, email, document_number, phone, legal_name |
| orders | identifier, realm, contact_id, payment_method |
| products | sku |
| categories | name, slug |
| shipping_marks | tracking_number, order_id, carrier_id, status, dispatch_batch_id |
| tags | name |
| variations | name, value, definition |

### Query Optimization

- **Text search:** `LIKE %term%` uses the index for prefix matches and falls back to scan for infix matches. For large datasets (>1M rows), consider the Elasticsearch migration path.
- **LIKE escaping:** Special characters (`%`, `_`) in user input are escaped to prevent wildcard injection.
- **Pagination:** Count and data queries are split — `COUNT(*)` on the base query, `LIMIT/OFFSET` on the paginated query.
- **Sort validation:** Only declared sortable fields are accepted, preventing arbitrary column access.

### Spotlight Performance

- Providers execute concurrently via goroutines
- 2-second hard timeout per provider
- In-memory scoring is O(n log n) per provider (sort by score)
- Total spotlight latency ≈ max(provider latency) + merge overhead

### Benchmarking

Run search benchmarks:

```bash
cd module/core
go test -bench=BenchmarkSearch -benchmem ./search/...
```

### Scaling Recommendations

| Data Volume | Strategy |
|-------------|----------|
| < 100K rows | SQL LIKE + indexes (current) |
| 100K–1M rows | SQL LIKE + Redis cache (enable CachedRepository) |
| > 1M rows | Elasticsearch adapter (see migration path) |
