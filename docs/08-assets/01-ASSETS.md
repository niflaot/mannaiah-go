# Assets

The assets module provides a cloud-agnostic binary file store with folder organisation,
tag-based classification, and an automated JPEG conversion worker. It is a pure infrastructure
service: other modules (products, Falabella) reference asset IDs; the assets module does not
reference any business domain.

---

## Architecture Overview

```
HTTP Layer (Fiber)
        │
        ▼
Application Service
  ├── Asset ops (Create / Get / List / Update / Delete)
  ├── Folder ops (Create / Get / List / Tree / Update / Delete)
  └── JPG Worker (batch JPEG conversion via cron or HTTP trigger)
        │
        ├── Storage Port → Core storage backend (S3 / GCS / MinIO / local)
        ├── Repository Port → SQL GORM store (MySQL/PostgreSQL)
        └── Event Publisher Port → Core messaging bus
```

The module is storage-backend-agnostic. The concrete adapter (`CoreStoreAdapter`) delegates
to whatever object-store the core module registers — the assets application layer never
touches S3 or GCS directly.

---

## Table of Contents

| File | Contents |
|------|---------|
| [02-DOMAIN.md](02-DOMAIN.md) | `Asset` and `Tag` domain types, validation rules, storage key format |
| [03-FOLDERS.md](03-FOLDERS.md) | Folder hierarchy, slug generation, tree assembly |
| [04-API.md](04-API.md) | All HTTP endpoints, request/response schemas, error reference |
| [05-JPG-WORKER.md](05-JPG-WORKER.md) | JPEG conversion pipeline, cron schedule, rollback logic |
| [06-EVENTS.md](06-EVENTS.md) | Integration events, payload schemas, metadata envelope |

---

## Quick-Start Example

### Upload a file

```bash
curl -X POST https://api.example.com/assets \
  -H "Authorization: Bearer <token>" \
  -F "file=@/path/to/logo.png" \
  -F "name=Brand Logo" \
  -F 'tags=[{"name":"web","color":"#3b82f6"}]'
```

Response:

```json
{
  "_id": "a1b2c3d4-...",
  "key": "assets/a1b2c3d4-logo.png",
  "name": "Brand Logo",
  "originalName": "logo.png",
  "mimeType": "image/png",
  "size": 48220,
  "tags": [{ "name": "web", "color": "#3b82f6" }],
  "metadata": {},
  "createdAt": "2026-03-28T10:00:00Z",
  "updatedAt": "2026-03-28T10:00:00Z",
  "isDeleted": false
}
```

---

## Permission Scopes

| Scope | Operations granted |
|-------|--------------------|
| `assets:view` | List/get assets and folders |
| `assets:manage` | Create, update, delete assets and folders |
| `product:manage` | Trigger the JPG conversion worker |
