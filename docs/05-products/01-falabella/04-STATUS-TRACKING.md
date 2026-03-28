# Falabella — Sync Status Tracking

Every Falabella API submission produces a `FeedID`. Because the Seller Center processes feeds
asynchronously, Mannaiah tracks the lifecycle of each feed internally through a `SyncEntry` model
and resolves pending entries both inline (during sync) and on a background cron schedule.

---

## Domain Types

### SyncEntry

One `SyncEntry` corresponds to one Falabella feed (either a product data feed or an image feed).

| Field | Type | Description |
|-------|------|-------------|
| `ExecutionID` | `string` | UUID of the batch execution this entry belongs to |
| `ProductID` | `string` | Mannaiah product ID |
| `SKU` | `string` | Product SKU at sync time |
| `VariationIDs` | `[]string` | Variation IDs included in this entry |
| `FeedID` | `string` | Falabella-assigned feed identifier |
| `Step` | `SyncStep` | `"product"` or `"image"` |
| `Task` | `SyncTask` | `"data"` or `"image"` |
| `Action` | `SyncAction` | `"create"` or `"update"` |
| `Status` | `SyncStatus` | `"pending"`, `"finished"`, or `"failed"` |
| `SyncedAt` | `time.Time` | When the feed was submitted |
| `ResolvedAt` | `*time.Time` | When the feed status was resolved (nil if pending) |

### SyncExecution

A `SyncExecution` groups all `SyncEntry` records for a single sync run.

| Field | Type | Description |
|-------|------|-------------|
| `ExecutionID` | `string` | UUID primary key |
| `StartedAt` | `time.Time` | Execution start time |

---

## Status Lifecycle

```
Submit feed to Falabella API
          │
          ▼
    [SyncEntry: pending]
          │
          ├─── Inline resolution (waitForProductFeedResolution)
          │    Polls up to N times during the sync call
          │
          └─── Background resolution (cron every 5 min)
               ResolvePendingFeeds(limit=50)
                     │
               GetFeedStatus(feedID)
                     │
               ┌─────┴─────┐
          finished       failed
               │              │
    UpdateStatus(finished)   UpdateStatus(failed)
    ResolvedAt = now         ResolvedAt = now
```

---

## Port Layer

### `port.SyncStatusRepository`

```go
EnsureSchema(ctx context.Context) error
CreateExecution(ctx context.Context, e *SyncExecution) error
Create(ctx context.Context, e *SyncEntry) error
GetExecutionByID(ctx context.Context, executionID string) (*SyncExecution, error)
GetByFeedID(ctx context.Context, feedID string) (*SyncEntry, error)
ListByExecutionID(ctx context.Context, executionID string) ([]SyncEntry, error)
GetByProductID(ctx context.Context, productID string) ([]SyncEntry, error)
ListPending(ctx context.Context, limit int) ([]SyncEntry, error)
UpdateStatus(ctx context.Context, feedID string, status SyncStatus, resolvedAt *time.Time) error
```

---

## Background Cron

| Config Variable | Default | Description |
|----------------|---------|-------------|
| `FALABELLA_SYNC_STATUS_CRON` | `*/5 * * * *` | Cron expression for feed resolution runs |
| `FALABELLA_SYNC_STATUS_BATCH_SIZE` | `50` | Maximum pending entries resolved per cron tick |

On each tick, `ResolvePendingFeeds(limit)` fetches up to `limit` entries with `status=pending`,
calls `GetFeedStatus` for each, and updates records to `finished` or `failed` based on the
Falabella response.

---

## HTTP Endpoints

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| `GET` | `/falabella/sync/status/feed/:feedId` | `product:view` | Get a single SyncEntry by feed ID |
| `GET` | `/falabella/sync/status/execution/:executionId` | `product:view` | Get a SyncExecution |
| `GET` | `/falabella/sync/status/execution/:executionId/feeds` | `product:view` | List all entries for an execution |
| `GET` | `/falabella/sync/status/product/:productId` | `product:view` | List all entries for a product |
| `POST` | `/falabella/sync/status/feed/:feedId/resolve` | `product:manage` | Manually trigger resolution for a feed |

---

## FeedDetail Response Shape

When a feed is resolved, the Falabella XML response is parsed into:

```json
{
  "feedId": "abc123",
  "status": "Finished",
  "action": "ProductCreate",
  "totalRecords": 3,
  "processedRecords": 3,
  "failedRecords": 0,
  "errors": []
}
```

If `failedRecords > 0`, the `errors` array contains objects with `code`, `message`, and
`sellerSku` fields identifying which SKU failed and why.
