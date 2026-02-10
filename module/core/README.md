# Core Module

`module/core` provides shared runtime capabilities for all future modules.

## Packages
- `config`: Viper-based startup configuration loading and validation from `.env` and environment variables.
- `logger`: Zap logger construction and resolution helpers.
- `cache`: Provider-agnostic cache interface contracts.
- `messaging`: Replaceable integration messaging contracts and Watermill in-memory adapters.
- `redis`: Redis key-value primitives with pattern scanning and batched retrieval.
- `database`: GORM bootstrap and reusable generic CRUD service primitives.
- `http`: Fiber server setup with `CORE_HOST`/`CORE_PORT`-authoritative address resolution and Zap request logging.
- `swagger`: OpenAPI aggregation and documentation route exposure.
- `startup`: module-loading runtime helpers for composition roots.

## Goals
- Deterministic startup configuration rules.
- Strict required-field validation with startup error logs.
- Reusable logging primitives with configurable format and level.

## Key Methods / Endpoints / Events
- Methods:
  - `config.NewLoader(envFile, startupLogger)`
  - `config.Load(envFile, startupLogger, targets...)`
  - `config.LoadWith(provider, targets...)`
  - `(*config.Loader).Load(targets...)`
  - `cache.Store.Get(ctx, key)`
  - `cache.Store.Set(ctx, key, value, ttl)`
  - `cache.Store.Delete(ctx, key)`
  - `cache.Store.Keys(ctx, pattern)`
  - `cache.Store.GetByPattern(ctx, pattern)`
  - `bus.Publisher.Publish(ctx, msg)`
  - `bus.Registrar.AddHandler(topic, handler)`
  - `watermill.NewInMemoryPlatform(cfg, providedLogger)`
  - `(*watermill.InMemoryPlatform).Publisher()`
  - `(*watermill.InMemoryPlatform).Registrar()`
  - `(*watermill.InMemoryPlatform).Run(ctx)`
  - `(*watermill.InMemoryPlatform).Close()`
  - `logger.New(settings)`
  - `logger.NewWithWriters(settings, output, errorOutput)`
  - `logger.Resolve(provided, settings)`
  - `redis.New(cfg, providedLogger)`
  - `redis.NewCache(cfg, providedLogger)`
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
  - `(*database.Service[T]).Paginate(ctx, query)`
  - `(*database.Service[T]).Update(ctx, id, updates)`
  - `(*database.Service[T]).Delete(ctx, id)`
  - `http.New(cfg, providedLogger)`
  - `http.NewWithCore(cfg, coreCfg, providedLogger)`
  - `http.AddressFrom(cfg, coreCfg)`
  - `http.NewAppError(status, message, cause)`
  - `swagger.NewDocument(info)`
  - `(*swagger.Document).Merge(spec *openapi3.T)`
  - `(*swagger.Document).Build() *openapi3.T`
  - `swagger.RegisterRoute(router, path, document)`
  - `startup.NewRuntime(server, document)`
  - `(*startup.Runtime).RegisterRoutes(register)`
  - `(*startup.Runtime).AddOpenAPISpec(spec *openapi3.T)`
  - `(*startup.Runtime).ExposeOpenAPI(path)`
  - `(*startup.Runtime).ExposeOpenAPIUI(path, specPath, title)`
  - `startup.CoreSpec() *openapi3.T`
  - `(*http.Server).RegisterRoutes(register)`
  - `(*http.Server).MountRoutes(prefix, register)`
  - `(*http.Server).Register(register)`
  - `(*http.Server).Mount(prefix, register)`
  - `(*http.Server).Start()`
  - `(*http.Server).Shutdown(ctx)`
- Endpoints:
  - startup compositions commonly expose `/status`, `/openapi.json`, and `/docs`
- Events: startup validation errors are emitted through Zap logger records.
