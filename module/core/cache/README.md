# Cache Package

`cache` provides provider-agnostic cache interfaces used by core and future modules.

## Features
- Abstract cache contract independent of Redis or any concrete backend.
- Unified key-value operations and pattern-based retrieval semantics.
- Provider lifecycle contract through `Close`.

## Usage Rules
- Depend on `cache.Store` in domain/application services.
- Inject concrete implementations (for example `redis.Store`) at startup boundaries.

## Key Methods / Endpoints / Events
- Methods:
  - `cache.Store.Ping(ctx)`
  - `cache.Store.Get(ctx, key)`
  - `cache.Store.Set(ctx, key, value, ttl)`
  - `cache.Store.Delete(ctx, key)`
  - `cache.Store.Keys(ctx, pattern)`
  - `cache.Store.GetByPattern(ctx, pattern)`
  - `cache.Store.Close()`
- Endpoints: none in this package.
- Events: implementation-specific cache errors are returned to callers.
