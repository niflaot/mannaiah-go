# Redis Package

`redis` provides high-performance Redis key-value operations for core and module integrations.

## Features
- URL and auth-driven connection configuration via config tags.
- Shared client-based operations for key lifecycle and retrieval.
- Pattern-based key discovery using `SCAN` to avoid blocking `KEYS`.
- Batched `MGET` retrieval for efficient pattern fetches.
- Zap-based error logging and configurable logger injection.
- Implements provider-agnostic cache contracts from `module/core/cache`.

## Usage Rules
- Load `redis.Config` with the shared `config` loader.
- Reuse a single `Store` instance per process.
- Depend on `cache.Store` interfaces in domain services.
- Prefer `GetByPattern` only for bounded operational patterns.

## Key Methods / Endpoints / Events
- Methods:
  - `redis.New(cfg, providedLogger)`
  - `redis.NewCache(cfg, providedLogger)`
  - `redis.NewWithClient(client, providedLogger, scanCount, batchSize)`
  - `(*redis.Store).Ping(ctx)`
  - `(*redis.Store).Get(ctx, key)`
  - `(*redis.Store).Set(ctx, key, value, ttl)`
  - `(*redis.Store).Delete(ctx, key)`
  - `(*redis.Store).Keys(ctx, pattern)`
  - `(*redis.Store).GetByPattern(ctx, pattern)`
  - `(*redis.Store).Close()`
- Endpoints: none in this package.
- Events: Redis operation failures are emitted through Zap error logs.
