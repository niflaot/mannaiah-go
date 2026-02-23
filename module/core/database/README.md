# Database Package

`database` provides GORM initialization and a reusable generic CRUD service with soft-delete support.

## Features
- Configuration-driven DB bootstrapping with GORM.
- Driver support for `sqlite`, `postgres`, and `mysql`.
- Generic base model with timestamps and soft-delete metadata.
- Generic CRUD service with query-based `Find`.
- Extension-friendly service composition for domain-specific methods.
- Embedded startup migration runner (`database/migration`) based on versioned SQL files.

## Usage Rules
- Load `database.Config` through the shared core config loader.
- Prefer versioned SQL migrations under `database/migration/migrations` for schema changes.
- SQL migration files are maintained for `mysql` and `sqlite`.
- Extend `Service[T]` via composition for module-specific behavior.

## Key Methods / Endpoints / Events
- Methods:
  - `database.Open(cfg, providedLogger)`
  - `migration.FromDatabaseConfig(cfg)`
  - `migration.Apply(ctx, db, cfg, logger)`
  - `database.NewService[T](db)`
  - `(*database.Service[T]).Create(ctx, entity)`
  - `(*database.Service[T]).Read(ctx, id)`
  - `(*database.Service[T]).Find(ctx, query)`
  - `(*database.Service[T]).Paginate(ctx, query)`
  - `(*database.Service[T]).Update(ctx, id, updates)`
  - `(*database.Service[T]).Delete(ctx, id)`
- Endpoints: none in this package.
- Events: query errors and slow-query signals are emitted through Zap-backed GORM logs.
