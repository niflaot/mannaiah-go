# module/assets

Assets module for uploading files to object storage and managing metadata.

## Key methods / endpoints / events
- Methods: `assets.New(db, storage, publishers...)`, `(*assets.Module).RegisterRoutes`, `(*assets.Module).SetAuthorizer`, `(*assets.Module).Load`, `(*assets.Module).Service`, `assets.OpenAPISpec()`
- Endpoints: `POST /assets`, `GET /assets`, `GET /assets/{id}`, `PATCH /assets/{id}`, `DELETE /assets/{id}`
- Events: `assets.v1.created`, `assets.v1.updated`, `assets.v1.deleted`
