# Adapter / Store

Contains GORM-backed persistence for Falabella sync status entries.

## Key Types

- `Repository` — Implements `port.SyncStatusRepository` using GORM.
- Parent table: `falabella_sync_execution` with columns: `execution_id` (PK), `started_at`.
- Child table: `falabella_sync_status` with columns: `execution_id` (indexed), `feed_id` (PK), `product_id`, `sku`, `step`, `task`, `action`, `status`, `synced_at`, `resolved_at`.
- Link table: `falabella_sync_status_variation` with columns: `feed_id`, `variation_id` and composite PK (`feed_id`, `variation_id`).
- `ListPending` — Retrieves unresolved entries ordered by `synced_at ASC` with configurable limit (used by cron resolver).
- `EnsureSchema` — No-op at runtime; schema evolution is managed by versioned SQL migrations.

## Usage

```go
repo, err := store.NewRepository(db)
repo.EnsureSchema(ctx)
```
