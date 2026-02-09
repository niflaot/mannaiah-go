# Core Module

`module/core` provides shared runtime capabilities for all future modules.

## Packages
- `config`: Viper-based startup configuration loading and validation from `.env` and environment variables.
- `logger`: Zap logger construction and resolution helpers.
- `redis`: Redis key-value primitives with pattern scanning and batched retrieval.
- `database`: GORM bootstrap and reusable generic CRUD service primitives.

## Goals
- Deterministic startup configuration rules.
- Strict required-field validation with startup error logs.
- Reusable logging primitives with configurable format and level.

## Key Methods / Endpoints / Events
- Methods:
  - `config.NewLoader(envFile, startupLogger)`
  - `config.Load(envFile, startupLogger, targets...)`
  - `(*config.Loader).Load(targets...)`
  - `logger.New(settings)`
  - `logger.NewWithWriters(settings, output, errorOutput)`
  - `logger.Resolve(provided, settings)`
  - `redis.New(cfg, providedLogger)`
  - `redis.NewWithClient(client, providedLogger, scanCount, batchSize)`
  - `(*redis.Store).Get(ctx, key)`
  - `(*redis.Store).Set(ctx, key, value, ttl)`
  - `(*redis.Store).Delete(ctx, key)`
  - `(*redis.Store).Keys(ctx, pattern)`
  - `(*redis.Store).GetByPattern(ctx, pattern)`
  - `database.Open(cfg, providedLogger)`
  - `database.NewService[T](db)`
  - `(*database.Service[T]).Create(ctx, entity)`
  - `(*database.Service[T]).Read(ctx, id)`
  - `(*database.Service[T]).Find(ctx, query)`
  - `(*database.Service[T]).Update(ctx, id, updates)`
  - `(*database.Service[T]).Delete(ctx, id)`
- Endpoints: none in this module.
- Events: startup validation errors are emitted through Zap logger records.
