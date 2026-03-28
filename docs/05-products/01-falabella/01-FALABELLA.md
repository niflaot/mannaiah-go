# Falabella Integration

The `falabella` module integrates Mannaiah with the **Falabella Seller Center API**. It reads
product data through the `ProductCatalog` port (backed by `module/products`), transforms it into
Falabella's XML-over-HTTP format, and tracks every sync operation through the `SyncEntry` ledger.

## Contents

| File | Description |
|------|-------------|
| [01-FALABELLA.md](01-FALABELLA.md) | This overview |
| [02-REALMS.md](02-REALMS.md) | Falabella realm attributes — required and optional fields |
| [03-SYNC-PIPELINE.md](03-SYNC-PIPELINE.md) | Full product sync pipeline, variants, and image upload |
| [04-STATUS-TRACKING.md](04-STATUS-TRACKING.md) | SyncEntry model, feed polling, and cron resolution |
| [05-IMAGE-TRANSCODING.md](05-IMAGE-TRANSCODING.md) | JPEG transcode proxy for Falabella image requirements |
| [06-API.md](06-API.md) | HTTP endpoints reference |
| [07-CONFIG.md](07-CONFIG.md) | All `FALABELLA_*` environment variables |

## Key Concepts

- **Feed**: Falabella's async processing model. Every product or image submission returns a
  `FeedID`. The caller must poll `GetFeedStatus(feedID)` to determine success or failure.
- **Realm**: Product channel scope. The Falabella module reads the `"falabella"` realm from a
  product's `Datasheets` slice. See [02-REALMS.md](02-REALMS.md) for the attribute contract.
- **SyncEntry**: Mannaiah's internal record for one Falabella feed. Tracks step (`product` or
  `image`), status (`pending` / `finished` / `failed`), and resolution timestamps.
- **Circuit Breaker**: All Seller Center API calls go through a configurable gobreaker instance
  that opens on threshold failures and returns `ErrUnavailable` during the open window.
