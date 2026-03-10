# assets/runtime

Composition root for the assets module: repository + service + HTTP adapter + OpenAPI spec.

## Key methods / endpoints / events
- Methods: `runtime.New(db, storage, publishers...)`, `runtime.NewWithConfig(cfg, db, storage, logger, publishers...)`, `(*runtime.Module).ConfigureScheduler`, `(*runtime.Module).Start`, `(*runtime.Module).Stop`, `(*runtime.Module).RegisterRoutes`, `(*runtime.Module).SetAuthorizer`, `(*runtime.Module).Load`, `runtime.OpenAPISpec()`
- Endpoints: `POST /assets`, `GET /assets`, `GET /assets/{id}`, `PATCH /assets/{id}`, `DELETE /assets/{id}`, `POST /assets/workers/jpg/run`, `POST /assets/folders`, `GET /assets/folders`, `GET /assets/folders/tree`, `GET /assets/folders/{id}`, `PATCH /assets/folders/{id}`, `DELETE /assets/folders/{id}`
- Events: lifecycle events emitted by application service through injected publisher.

## JPG Worker Config
- `ASSETS_JPG_WORKER_ENABLED`
- `ASSETS_JPG_WORKER_CRON`
- `ASSETS_JPG_WORKER_TAGS` (comma-separated tag names)
- `ASSETS_JPG_WORKER_BATCH_SIZE`
- `ASSETS_JPG_WORKER_JPEG_QUALITY`
- `ASSETS_JPG_WORKER_TIMEOUT_MS`
