# Asset Transfer Guide

This document describes the new asset-folder capabilities and the OpenAPI-facing contract for frontend integration.

## Authentication
- Protected endpoints require `Authorization: Bearer <jwt>`.
- Required scopes:
  - Assets read: `assets:read`
  - Assets create: `assets:create`
  - Assets update: `assets:update`
  - Assets delete: `assets:delete`

## Data Rules
- Tags (for assets and folders):
  - Max: `5`
  - `name`: lowercase, `[a-z0-9][a-z0-9_-]{0,31}`
  - `color`: lowercase hex `#rrggbb`
- Asset metadata:
  - JSON object `map[string]string`
  - Key length max `128`
  - Value length max `2048`
- Folder delete is soft-delete and detaches assets (`folderId` becomes empty on linked assets).
- Asset delete is soft-delete (metadata row), no folder impact.

## Endpoints

### Asset Endpoints

1. `POST /assets`
- Content-Type: `multipart/form-data`
- Fields:
  - `file` (required)
  - `name` (optional)
  - `folderId` (optional)
  - `tags` (optional JSON string, example: `[{"name":"cover","color":"#00aa11"}]`)
  - `metadata` (optional JSON string, example: `{"alt":"home hero"}`)

2. `GET /assets?page=1&limit=10&filters=...`

3. `GET /assets/{id}`

4. `PATCH /assets/{id}`
- JSON body (all optional):
```json
{
  "name": "Hero 2026",
  "folderId": "folder-id",
  "tags": [
    {"name": "cover", "color": "#00aa11"}
  ],
  "metadata": {
    "alt": "homepage hero"
  }
}
```

5. `DELETE /assets/{id}`

### Folder Endpoints

1. `POST /assets/folders`
```json
{
  "name": "Catalog",
  "tags": [
    {"name": "hero", "color": "#ff0000"}
  ]
}
```

2. `GET /assets/folders?page=1&limit=10&filters=...`

3. `GET /assets/folders/{id}`

4. `PATCH /assets/folders/{id}`
```json
{
  "name": "Catalog 2026",
  "tags": [
    {"name": "catalog", "color": "#ffaa00"}
  ]
}
```

5. `DELETE /assets/folders/{id}`

## Error Contract
The API maps errors to:
```json
{
  "message": "translatable_message",
  "error": "detailed error"
}
```

Common `message` values:
- `invalid_asset`
- `invalid_asset_id`
- `invalid_asset_name`
- `invalid_folder_id`
- `invalid_folder_name`
- `asset_not_found`
- `asset_folder_not_found`
- `storage_unavailable`

## Product Integration
- Product gallery keeps referencing `assetId` only.
- Product creation/update validates referenced assets through assets service (`Exists` check).
- Folder/tag/metadata additions do not break product payload compatibility.

## Events
- Asset events:
  - `assets.v1.created`
  - `assets.v1.updated`
  - `assets.v1.deleted`
- Folder events:
  - `asset_folders.v1.created`
  - `asset_folders.v1.updated`
  - `asset_folders.v1.deleted`

## Websocket Evaluation
- Current recommendation: websocket is not required yet.
- Reasons:
  - Integration events already exist for backend-to-backend flows.
  - Frontend can use standard fetch + refetch/polling on mutation.
- Add websocket/SSE only when you need collaborative real-time UI updates across multiple clients (e.g., shared media manager with live folder/asset updates).
