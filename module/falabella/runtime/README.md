# Runtime Package

Composition root for Falabella module wiring (source adapter, service, HTTP adapter, sync status persistence, and OpenAPI artifact).

## Key methods / endpoints / events
- Methods: `runtime.New`, `(*runtime.Module).RegisterRoutes`, `(*runtime.Module).SetAuthorizer`, `(*runtime.Module).ConfigureSyncStatus`, `(*runtime.Module).ConfigureScheduler`, `(*runtime.Module).Start`, `(*runtime.Module).Stop`, `(*runtime.Module).Load`, `runtime.OpenAPISpec`
- Endpoints: `GET /falabella/brands`, `POST /falabella/sync/products`, `POST /falabella/sync/products/{id}`, `GET /falabella/sync/status/feed/{feedId}`, `GET /falabella/sync/status/product/{productId}`, `POST /falabella/sync/status/feed/{feedId}/resolve`
- Events: none
