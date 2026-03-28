# Assets — HTTP API

All endpoints require a valid bearer token.

---

## Asset Endpoints

### Create Asset

```
POST /assets
Permission: assets:manage
Content-Type: multipart/form-data
```

**Form fields**

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `file` | binary | **Yes** | Max 10 MB |
| `name` | string | No | Display name; defaults to filename |
| `folderId` | string | No | Parent folder UUID |
| `tags` | string (JSON array) | No | e.g. `[{"name":"web","color":"#3b82f6"}]` |
| `metadata` | string (JSON object) | No | e.g. `{"source":"import"}` |

**Response** `201 Created`

```json
{
  "_id": "a1b2c3d4-...",
  "key": "assets/a1b2c3d4-banner.png",
  "name": "Black Friday Banner",
  "originalName": "banner.png",
  "folderId": "f-uuid",
  "mimeType": "image/png",
  "size": 204800,
  "tags": [{"name":"promo","color":"#ef4444"}],
  "metadata": {"campaign": "bf-2026"},
  "createdAt": "2026-11-01T09:00:00Z",
  "updatedAt": "2026-11-01T09:00:00Z",
  "isDeleted": false
}
```

**Upload flow overview:**
1. Storage availability is checked — `503` if unavailable.
2. File is read into memory and validated (≤ 10 MB).
3. If `folderId` is provided, its existence is verified.
4. A UUID is generated and the storage key `assets/<uuid>-<originalName>` is constructed.
5. Domain entity is normalised and validated.
6. Binary is uploaded to the object store.
7. Database record is created in a transaction (with rollback if the DB write fails).
8. `assets.v1.created` event is published.

---

### List Assets

```
GET /assets
Permission: assets:view
```

**Query parameters**

| Param | Default | Notes |
|-------|---------|-------|
| `page` | `1` | |
| `limit` | `10` | |
| `filters` | `""` | Free-text filter |

**Response** `200 OK`

```json
{
  "data": [ /* Asset[] */ ],
  "meta": { "page": 1, "limit": 10, "total": 84 }
}
```

---

### Get Asset

```
GET /assets/:id
Permission: assets:view
```

**Response** `200 OK` with the full `Asset` object, or `404 Not Found`.

---

### Update Asset

```
PATCH /assets/:id
Permission: assets:manage
```

Updates mutable fields. All fields are optional.

```json
{
  "name": "Updated Display Name",
  "folderId": "new-folder-uuid",
  "tags": [{"name":"print","color":"#8b5cf6"}],
  "metadata": {"source": "manual_upload"}
}
```

> Pass an empty string for `folderId` to detach the asset from its current folder.

Tags and metadata are **replaced** — not merged. Pass the complete desired list each time.

**Response** `200 OK` with the updated `Asset`.

Write serialisation: an in-process keyed lock (`asset:<id>`) is acquired before writing,
preventing concurrent modifications to the same entity.

---

### Delete Asset

```
DELETE /assets/:id
Permission: assets:manage
```

Performs a soft delete. The asset record remains in the database (`deleted_at` is set and
`is_deleted = true`) but it will not appear in list results. The object-store binary is
**not** deleted.

**Response** `200 OK`
```json
{ "status": "deleted" }
```

---

### Trigger JPG Worker

```
POST /assets/workers/jpg/run
Permission: product:manage
```

Manually triggers the JPEG conversion worker for a single batch run. Useful for on-demand
conversion or testing. See [05-JPG-WORKER.md](05-JPG-WORKER.md) for full pipeline details.

**Query parameters** (all optional, override module config)

| Param | Default | Notes |
|-------|---------|-------|
| `tags` | from config | Comma-separated tag names; only assets with these tags are eligible |
| `batchSize` | `100` | Max assets per run |
| `jpegQuality` | `90` | JPEG encoder quality (1–100) |

**Response** `200 OK`

```json
{
  "scanned": 120,
  "converted": 105,
  "skipped": 10,
  "failed": 5,
  "tags": ["web", "promo"],
  "batchSize": 100,
  "jpegQuality": 90
}
```

---

## Folder Endpoints

### Create Folder

```
POST /assets/folders
Permission: assets:manage
```

```json
{
  "name": "Product Images",
  "parentFolderId": "f-uuid-or-omit-for-root",
  "tags": [{"name":"auto","color":"#10b981"}]
}
```

Slug is auto-generated from `name` if not provided. Conflicts with an existing slug under
the same parent return `409`.

**Response** `201 Created` with the `Folder` object.

---

### List Folders

```
GET /assets/folders
Permission: assets:view
```

**Query parameters**

| Param | Notes |
|-------|-------|
| `page`, `limit` | Pagination |
| `filters` | Free-text filter |
| `parentFolderId` | Filter to one parent scope |

**Response** `200 OK`
```json
{
  "data": [ /* Folder[] */ ],
  "meta": { "page": 1, "limit": 10, "total": 12 }
}
```

---

### Get Folder Tree

```
GET /assets/folders/tree
Permission: assets:view
```

Returns the full hierarchy as a nested structure. Assembled in-memory from a single DB query.
No depth limit.

**Response** `200 OK`
```json
{
  "data": [
    {
      "id": "f1", "name": "Products", "slug": "products",
      "children": [
        { "id": "f2", "name": "Shirts", "slug": "shirts", "children": [] }
      ]
    }
  ]
}
```

---

### Get Folder

```
GET /assets/folders/:id
Permission: assets:view
```

**Response** `200 OK` with the `Folder` object (no `children`) or `404`.

---

### Update Folder

```
PATCH /assets/folders/:id
Permission: assets:manage
```

```json
{
  "name": "Renamed Folder",
  "parentFolderId": "new-parent-uuid",
  "tags": [{"name":"archived","color":"#6b7280"}]
}
```

Slug is automatically rebuilt if `name` changes. Cycle detection is enforced. Slug
uniqueness is re-checked within the new parent scope.

**Response** `200 OK` with the updated `Folder`.

---

### Delete Folder

```
DELETE /assets/folders/:id
Permission: assets:manage
```

Soft-deletes the folder. All assets previously in this folder are **detached** (their
`folder_id` is set to NULL) — they are not deleted.

**Response** `200 OK`
```json
{ "status": "deleted" }
```

---

## Error Reference

| HTTP | Code | Trigger |
|------|------|---------|
| `400` | `file_required` | Missing `file` field in multipart |
| `400` | `file_too_large` | File exceeds 10 MB |
| `400` | `invalid_asset` | Domain validation failure (tags, metadata, size) |
| `400` | `invalid_asset_id` | Empty asset ID |
| `400` | `invalid_asset_name` | Empty asset name |
| `400` | `invalid_folder_id` | Empty folder ID |
| `400` | `invalid_folder_name` | Empty folder name |
| `400` | `invalid_folder_parent` | Self-reference or ancestor cycle |
| `400` | `invalid_jpg_worker_tags` | Worker triggered with no tags |
| `401` | `unauthorized` | Missing or invalid JWT |
| `403` | `forbidden` | Insufficient scope |
| `404` | `asset_not_found` | Asset ID not found |
| `404` | `asset_folder_not_found` | Folder ID not found |
| `409` | `asset_folder_already_exists` | Slug collision within same parent |
| `503` | `storage_unavailable` | Object store backend is unavailable |
