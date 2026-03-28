# Assets — Integration Events

All events use schema version `v1`. Each event envelope carries:

| Metadata key | Value |
|-------------|-------|
| `schema_version` | `"v1"` |
| `produced_at` | RFC3339 UTC timestamp |
| `aggregate_id` | Entity UUID |
| `correlation_id` | Forwarded from triggering request (if set) |
| `causation_id` | Forwarded from triggering request (if set) |

---

## Asset Events

### `assets.v1.created`

Published after a new asset is successfully persisted and its binary is stored.

```json
{
  "id": "a1b2c3d4-...",
  "key": "assets/a1b2c3d4-logo.png",
  "name": "Brand Logo",
  "originalName": "logo.png",
  "folderId": "f-uuid",
  "mimeType": "image/png",
  "size": 48220,
  "tags": [{"name":"web","color":"#3b82f6"}],
  "metadata": {"source":"manual"},
  "isDeleted": false,
  "createdAt": "2026-03-28T10:00:00Z",
  "updatedAt": "2026-03-28T10:00:00Z"
}
```

---

### `assets.v1.updated`

Published after any mutable field is changed on an existing asset, including after a
successful JPEG conversion (the `key`, `mimeType`, and `size` will change).

Payload is identical to `assets.v1.created`. Consumers can diff `createdAt` vs `updatedAt`
to determine if the event represents a fresh upload or an update.

---

### `assets.v1.deleted`

Published after an asset is soft-deleted.

Same payload structure with:
- `isDeleted: true`
- `deletedAt` set (non-null)

The binary is **not** removed from the object store at this point.

---

## Folder Events

### `asset_folders.v1.created`

Published after a new folder is persisted.

```json
{
  "id": "f-uuid",
  "name": "Product Images",
  "slug": "product-images",
  "parentFolderId": "",
  "tags": [{"name":"auto","color":"#10b981"}],
  "isDeleted": false,
  "createdAt": "2026-03-28T10:05:00Z",
  "updatedAt": "2026-03-28T10:05:00Z"
}
```

---

### `asset_folders.v1.updated`

Published after folder name, slug, parent, or tags change.

---

### `asset_folders.v1.deleted`

Published after a folder is soft-deleted. `isDeleted: true`.

---

## Event Consumers (known)

| Consumer Module | Topic | Purpose |
|----------------|-------|---------|
| Falabella | `assets.v1.updated` | Detect JPEG conversion completion to update product image ordering |
| Products | `assets.v1.deleted` | Optionally invalidate product image references |
