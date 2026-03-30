# Search Package (`module/core/search`)

Core search infrastructure providing generic, cacheable, and extensible search
for all Mannaiah resources with cross-resource spotlight capability.

## Key Methods / Endpoints / Events

### Types & Interfaces

- `Repository[T]` — Port interface each module implements for search
- `Query` — Unified search input (term, filters, sort, pagination)  
- `Result[T]` — Paginated response envelope
- `Descriptor` — Declares searchable/filterable/sortable fields per resource
- `SpotlightProvider` — Interface for modules contributing to spotlight search
- `SpotlightService` — Concurrent fan-out spotlight orchestrator

### Core Functions

- `BuildGORMQuery(tx, query, desc)` — Translates Query + Descriptor → GORM chain
- `ParseQuery(ctx)` — Extracts search query from HTTP request parameters
- `SearchHandlerFunc[T](repo)` — Generic HTTP handler for resource search
- `SpotlightHandlerFunc(svc)` — HTTP handler for spotlight search
- `ScoreResults[T](entities, term, primary, secondary, extract)` — In-memory relevance scoring
- `NewCachedRepository[T](inner, store, config)` — Cache-aside decorator
- `OpenAPISpec()` — Returns kin-openapi spec for all search endpoints

### Endpoints (registered in main.go)

| Method | Path | Handler |
|--------|------|---------|
| GET | `/search/contacts` | `SearchHandlerFunc` |
| GET | `/search/orders` | `SearchHandlerFunc` |
| GET | `/search/products` | `SearchHandlerFunc` |
| GET | `/search/categories` | `SearchHandlerFunc` |
| GET | `/search/variations` | `SearchHandlerFunc` |
| GET | `/search/tags` | `SearchHandlerFunc` |
| GET | `/search/shipping` | `SearchHandlerFunc` |
| GET | `/search/campaigns` | `SearchHandlerFunc` |
| GET | `/search/segments` | `SearchHandlerFunc` |
| GET | `/search` | `SpotlightHandlerFunc` |

## Usage

See [docs/11-search/](../../docs/11-search/) for full documentation.
