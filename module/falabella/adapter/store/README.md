# Adapter / Store

Contains GORM-backed persistence for Falabella sync status entries.

## Key Types

- `Repository` — Implements `port.SyncStatusRepository` using GORM.
- Parent table: `falabella_sync_execution` with columns: `execution_id` (PK), `started_at`.
- Child table: `falabella_sync_status` with columns: `execution_id` (indexed), `feed_id` (PK), `product_id`, `sku`, `step`, `action`, `status`, `synced_at`, `resolved_at`.
- `ListPending` — Retrieves unresolved entries ordered by `synced_at ASC` with configurable limit (used by cron resolver).
- `EnsureSchema` — Auto-migrates with legacy schema detection: drops old tables that have the removed `id` column.

## Usage

```go
repo, err := store.NewRepository(db)
repo.EnsureSchema(ctx)
```
