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
  - `migrations/postgres/*.sql`
  - `migrations/sqlite/*.sql`
- Supports `mysql`, `postgres`, and `sqlite` migration drivers.
- Uses `ErrNoChange` as a non-failure state.
- Applies best-effort timeout and graceful stop signaling.

## Baseline strategy
- Each driver directory includes `000001_baseline` as a no-op anchor migration for controlled versioning.
- Future schema changes must add numbered `*.up.sql` and `*.down.sql` pairs.

## Assets folder uniqueness migration
- `000002_assets_folder_parent_slug_index` removes the legacy global folder-slug uniqueness index (`idx_asset_folders_slug`).
- Parent-scoped uniqueness is enforced by the application schema (`idx_asset_folders_parent_slug`) during module schema setup.
