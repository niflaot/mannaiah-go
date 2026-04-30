# Search Module

The Search module provides a unified, performant, and extensible search infrastructure for all Mannaiah resources. It powers both per-resource filtered/sorted search endpoints and a cross-resource spotlight search.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Query DSL](#query-dsl)
4. [Endpoints](#endpoints)
5. [Spotlight Search](#spotlight-search)
6. [Caching](#caching)
7. [Adding a New Searchable Resource](#adding-a-new-searchable-resource)
8. [Elasticsearch Migration Path](#elasticsearch-migration-path)

---

## Overview

The search infrastructure is built on these principles:

- **Generic**: A single `Repository[T]` interface and `SearchHandlerFunc[T]` handler work for any GORM model.
- **Configurable**: Each resource declares its own `Descriptor` specifying which fields are text-searchable, filterable (with allowed operators), and sortable.
- **Cacheable**: An opt-in `CachedRepository[T]` decorator provides Redis-backed cache-aside with configurable TTL and per-resource key prefixes.
- **Extensible**: Adding a new searchable resource requires only implementing one file — the search repository adapter.

### Supported Resources

| Resource    | Endpoint              | Text Fields                                           |
|-------------|----------------------|-------------------------------------------------------|
| Contacts    | `GET /search/contacts`    | first_name, last_name, legal_name, email, document_number, phone |
| Orders      | `GET /search/orders`      | identifier, payment_method                            |
| Products    | `GET /search/products`    | sku                                                   |
| Categories  | `GET /search/categories`  | name, slug, description                               |
| Variations  | `GET /search/variations`  | name, value                                           |
| Tags        | `GET /search/tags`        | name                                                  |
| Shipping    | `GET /search/shipping`    | tracking_number, order_id, observations               |
| Spotlight   | `GET /search`             | All of the above (concurrent fan-out)                 |

---

## Architecture

```
HTTP Request
    │
    ▼
ParseQuery(ctx)          ── extracts term, filters, sort, pagination
    │
    ▼
SearchHandlerFunc[T]     ── generic handler wired to a Repository[T]
    │
    ▼
CachedRepository[T]      ── optional cache-aside (Redis)
    │
    ▼
Repository[T]            ── GORM adapter per resource
    │
    ▼
BuildGORMQuery(tx, q, d) ── translates Query + Descriptor into GORM scopes
    │
    ▼
Database (MySQL/SQLite)
```

### Key Packages

| Package | Path | Purpose |
|---------|------|---------|
| `search` | `module/core/search/` | Core types, interfaces, query builder, HTTP parser, cache, spotlight |
| Per-resource adapters | `module/<mod>/adapter/search/` | Resource-specific `Repository[T]` + `SpotlightProvider` implementations |

---

## Query DSL

All resource search endpoints accept the same query parameter DSL.

### Text Search

```
GET /search/contacts?term=john
```

The `term` parameter performs a case-insensitive `LIKE %term%` search across all text fields defined in the resource's Descriptor. Results are OR-matched — a hit on any field qualifies the row.

### Filtering

Filters use the bracket notation `filter[field]` or `filter[field.op]`:

| Syntax | Operator | Example |
|--------|----------|---------|
| `filter[status]=ACTIVE` | `eq` (exact match) | Equals |
| `filter[status.in]=ACTIVE,PAUSED` | `in` | IN set |
| `filter[name.like]=john` | `like` | LIKE %value% |
| `filter[created_at.gte]=2024-01-01` | `gte` | >= |
| `filter[created_at.lte]=2024-12-31` | `lte` | <= |
| `filter[created_at.gt]=2024-01-01` | `gt` | > |
| `filter[created_at.lt]=2024-12-31` | `lt` | < |
| `filter[price.between]=10,50` | `between` | BETWEEN a AND b |

**Only operators listed in the resource Descriptor's `FilterableFields` map are accepted.** Unsupported operators are silently ignored.

### Sorting

```
GET /search/orders?sort=created_at:desc,identifier:asc
```

Comma-separated `field:direction` pairs. Direction is `asc` or `desc`. Only fields listed in the Descriptor's `SortableFields` are accepted.

### Pagination

| Parameter | Default | Max | Description |
|-----------|---------|-----|-------------|
| `page` | 1 | — | 1-based page number |
| `pageSize` | 20 | 100 | Items per page |

### Combined Example

```
GET /search/shipping?term=ABC123&filter[status.in]=DISPATCHED,IN_TRANSIT&filter[carrier_id]=uuid-here&sort=created_at:desc&page=1&pageSize=25
```

---

## Endpoints

### Resource Search Response

All resource search endpoints return the same envelope:

```json
{
  "data": [ ... ],
  "total": 142,
  "page": 1,
  "pageSize": 20,
  "totalPages": 8
}
```

| Field | Type | Description |
|-------|------|-------------|
| `data` | `array` | Page of matched entities |
| `total` | `integer` | Total matching rows (before pagination) |
| `page` | `integer` | Current page |
| `pageSize` | `integer` | Requested page size |
| `totalPages` | `integer` | Computed total pages |

### Error Responses

| Status | Condition |
|--------|-----------|
| `400` | Invalid filter operator, malformed between value, etc. |

---

## Spotlight Search

The spotlight endpoint provides a unified cross-resource search with relevance scoring.

### Request

```
GET /search?term=john&types=contact,order&limit=10
```

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `term` | Yes | — | Search term |
| `types` | No | all | Comma-separated resource types to include |
| `limit` | No | 10 | Max results per provider (max: 50) |

### Response

```json
{
  "results": [
    {
      "type": "contact",
      "id": "uuid-123",
      "title": "John Doe",
      "subtitle": "john@example.com",
      "matchedField": "first_name",
      "score": 1.0
    },
    {
      "type": "order",
      "id": "uuid-456",
      "title": "ORD-2024-001",
      "subtitle": "",
      "matchedField": "identifier",
      "score": 0.7
    }
  ],
  "meta": {
    "term": "john",
    "tookMs": 45,
    "counts": {
      "contact": 3,
      "order": 1
    }
  }
}
```

### Scoring Algorithm

Results are scored by match quality:

| Match Type | Score |
|------------|-------|
| Exact match on primary field | 1.0 |
| Prefix match on primary field | 0.7 |
| Contains match on primary field | 0.4 |
| Exact match on secondary field | 0.5 |
| Prefix match on secondary field | 0.35 |
| Contains match on secondary field | 0.2 |

Results are sorted by score descending, then merged across all providers.

### Concurrency

Each spotlight provider executes concurrently with a 2-second timeout. Slow providers are excluded from results without failing the overall request.

---

## Caching

The `CachedRepository[T]` decorator wraps any `Repository[T]` with Redis-backed caching.

### Configuration

```go
cfg := search.CacheConfig{
    Enabled:   true,           // default: false
    TTL:       60 * time.Second, // default: 60s
    KeyPrefix: "contacts",     // per-resource prefix
}

cachedRepo := search.NewCachedRepository(repo, redisStore, cfg)
```

### Cache Key Strategy

Cache keys are deterministic SHA256 hashes of the full query (term + filters + sort + pagination), prefixed with the resource key:

```
search:contacts:sha256(<query-json>)
```

This ensures:
- Identical queries always hit cache
- Different queries never collide
- Cache invalidation is straightforward (delete by prefix)

### Disabling Cache

Set `Enabled: false` (the default) and `CachedRepository` becomes a transparent pass-through.

---

## Adding a New Searchable Resource

To add search for a new resource (e.g., "invoices"):

### 1. Create the adapter

Create `module/invoices/adapter/search/repository.go`:

```go
package search

import (
    "gorm.io/gorm"
    coresearch "mannaiah/module/core/search"
    "mannaiah/module/invoices/adapter/store"
)

type Repository struct {
    db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
    return &Repository{db: db}
}

func (r *Repository) descriptor() coresearch.Descriptor {
    return coresearch.Descriptor{
        TextFields: []string{"number", "customer_name"},
        FilterableFields: map[string][]coresearch.Operator{
            "status":     {coresearch.OpEq, coresearch.OpIn},
            "created_at": {coresearch.OpGte, coresearch.OpLte, coresearch.OpBetween},
        },
        SortableFields: []string{"number", "created_at", "amount"},
        DefaultSort:    coresearch.SortField{Field: "created_at", Direction: coresearch.DESC},
    }
}

func (r *Repository) Search(ctx context.Context, q coresearch.Query) (*coresearch.Result[store.InvoiceRecord], error) {
    desc := r.descriptor()
    base, paginated := coresearch.BuildGORMQuery(r.db.WithContext(ctx).Model(&store.InvoiceRecord{}), q, desc)

    var total int64
    base.Count(&total)

    var items []store.InvoiceRecord
    paginated.Find(&items)

    return coresearch.NewResult(items, total, q.Page, q.PageSize), nil
}
```

### 2. (Optional) Add SpotlightProvider

```go
func (r *Repository) SpotlightSearch(ctx context.Context, term string, limit int) ([]coresearch.SpotlightHit, error) {
    q := coresearch.Query{Term: term, Page: 1, PageSize: limit}
    result, err := r.Search(ctx, q)
    if err != nil {
        return nil, err
    }
    hits := make([]coresearch.SpotlightHit, 0, len(result.Data))
    for _, inv := range result.Data {
        hits = append(hits, coresearch.SpotlightHit{
            Type:  "invoice",
            ID:    inv.ID,
            Title: inv.Number,
        })
    }
    scored := coresearch.ScoreResults(hits, term,
        func(h coresearch.SpotlightHit) string { return h.Title },
        nil,
    )
    out := make([]coresearch.SpotlightHit, len(scored))
    for i, s := range scored {
        h := s.Entity
        h.Score = s.Score
        h.MatchedField = s.MatchedField
        out[i] = h
    }
    return out, nil
}

func (r *Repository) SpotlightType() string { return "invoice" }
```

### 3. Wire in main.go

```go
invoiceSearchRepo := invoicesearch.NewRepository(db)
spotlightService.Add(invoiceSearchRepo)
router.Get("/invoices/search", coresearch.SearchHandlerFunc(invoiceSearchRepo))
```

### 4. Add migration indexes

Add indexes for the new text/filter fields in the next migration version.

---

## Elasticsearch Migration Path

The current implementation uses GORM + SQL `LIKE` queries. The architecture is designed for a future migration to Elasticsearch:

1. **The `Repository[T]` interface is the adapter boundary.** Swapping from GORM to Elasticsearch requires only implementing a new adapter — no handler or routing changes.

2. **The `Descriptor` struct maps cleanly to ES mappings.** Text fields → `text` type with analyzers, filterable fields → `keyword` type, sortable fields → sort mappings.

3. **The `Query` struct is engine-agnostic.** It translates to SQL WHERE clauses today and to ES query DSL tomorrow.

4. **Migration strategy:**
   - Phase 1 (current): SQL LIKE + indexes. Good for < 1M rows per resource.
   - Phase 2: Add ES indexing as a secondary adapter. Dual-write during migration.
   - Phase 3: Switch read path to ES adapter. SQL becomes write-only source of truth.
   - Phase 4: Remove SQL read path.
