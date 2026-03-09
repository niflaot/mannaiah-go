# Domain / Sync

Contains Falabella sync status domain entities and feed response models.

## Key Types

- `SyncEntry` — Represents a persisted sync status entry tracking product synchronization lifecycle, including linked `VariationIDs`. Uses `FeedID` (the Falabella-assigned feed UUID) as the natural primary key.
- `SyncAction` — Typed enum for sync operations (`create` / `update`).
- `SyncStatus` — Typed enum for feed resolution states (`pending` / `finished` / `failed`).
- `FeedResponse` — XML-mapped model for parsing Falabella FeedStatus API responses.
- `ActionResponse` — Parsed model for Falabella ProductCreate/ProductUpdate response values including warnings.
- `Warning` — Represents a WarningDetail element from Falabella sync responses.
- `FeedDetail` — Parsed feed detail values including record counts and errors.
- `FeedError` — Per-record error values from Falabella feed processing.
