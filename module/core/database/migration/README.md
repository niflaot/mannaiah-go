# Database Migration Package

Provides startup-safe, embedded SQL migration execution using `golang-migrate`.

## Key methods / endpoints / events
- Methods:
  - `migration.FromDatabaseConfig(cfg)`
  - `migration.Apply(ctx, db, cfg, logger)`
- Endpoints: none.
- Events: none.

## Behavior
- Reads migration files from embedded driver-specific directories:
  - `migrations/mysql/*.sql`
  - `migrations/sqlite/*.sql`
- Supports `mysql`, `sqlite`, and `sqlite3` migration drivers.
- Uses `ErrNoChange` as a non-failure state.
- Applies best-effort timeout and graceful stop signaling.
- Supports operations: `up`, `down`, `version`, `force`.

## Dedicated command
- Run migrations with the dedicated CLI command:
  - `go run ./module/core/cmd/migrate --operation up`
  - `go run ./module/core/cmd/migrate --operation down --steps 1`
  - `go run ./module/core/cmd/migrate --operation version`
  - `go run ./module/core/cmd/migrate --operation force --force-version <version>`
- By default, command execution enforces migrations (`DB_MIGRATIONS_ENABLED` is ignored unless `--respect-env-enabled=true`).

## Baseline strategy
- MySQL migrations include `000001_baseline` as a no-op anchor migration for controlled versioning.
- Future schema changes must add numbered `*.up.sql` and `*.down.sql` pairs.

## Assets folder uniqueness migration
- `000002_assets_folder_parent_slug_index` removes the legacy global folder-slug uniqueness index (`idx_asset_folders_slug`).
- Parent-scoped uniqueness is enforced by SQL schema migration (`000003_assets_schema`).

## Module schema migrations
- `000003_assets_schema` provisions assets schema.
- `000004_products_orders_schema` provisions products, variations, and orders schemas.
- `000005_contacts_falabella_schema` provisions contacts and Falabella sync-status schemas.

## Source of truth
- Runtime modules do not execute `AutoMigrate` for production schema changes.
- Schema evolution must be delivered through versioned SQL migrations in this package.
