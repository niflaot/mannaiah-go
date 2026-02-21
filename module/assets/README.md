# module/assets

Assets module for uploading files to object storage and managing metadata, tags, and nested logical folders.

## Key methods / endpoints / events
- Methods: `assets.New(db, storage, publishers...)`, `(*assets.Module).RegisterRoutes`, `(*assets.Module).SetAuthorizer`, `(*assets.Module).Load`, `(*assets.Module).Service`, `assets.OpenAPISpec()`
- Endpoints: `POST /assets`, `GET /assets`, `GET /assets/{id}`, `PATCH /assets/{id}`, `DELETE /assets/{id}`, `POST /assets/folders`, `GET /assets/folders`, `GET /assets/folders/tree`, `GET /assets/folders/{id}`, `PATCH /assets/folders/{id}`, `DELETE /assets/folders/{id}`
- Events: `assets.v1.created`, `assets.v1.updated`, `assets.v1.deleted`, `asset_folders.v1.created`, `asset_folders.v1.updated`, `asset_folders.v1.deleted`
