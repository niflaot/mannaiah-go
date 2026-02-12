# Redis Store Package

`store` provides the Redis-backed implementation of the core cache store contract.

## Responsibilities
- Build Redis clients from config.
- Implement key/value cache operations (`Get`, `Set`, `Delete`, `Keys`, `GetByPattern`).
- Apply optional circuit-breaker protection on Redis operations.
- Provide cache adapter wiring for provider-agnostic usage.

## Key Methods / Endpoints / Events
- Methods:
  - `store.New(cfg, providedLogger)`
  - `store.NewWithClient(client, providedLogger, scanCount, batchSize)`
  - `store.NewCache(cfg, providedLogger)`
  - `(*store.Store).Ping(ctx)`
  - `(*store.Store).Get(ctx, key)`
  - `(*store.Store).Set(ctx, key, value, ttl)`
  - `(*store.Store).Delete(ctx, key)`
  - `(*store.Store).Keys(ctx, pattern)`
  - `(*store.Store).GetByPattern(ctx, pattern)`
  - `(*store.Store).Close()`
- Endpoints: none in this package.
- Events: circuit-breaker and operation failures are logged through Zap.
