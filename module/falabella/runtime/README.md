# Runtime Package

Composition root for Falabella module wiring (source adapter, service, HTTP adapter, and OpenAPI artifact).

## Key methods / endpoints / events
- Methods: `runtime.New`, `(*runtime.Module).RegisterRoutes`, `(*runtime.Module).SetAuthorizer`, `(*runtime.Module).Load`, `runtime.OpenAPISpec`
- Endpoints: `GET /falabella/brands`, `POST /falabella/sync/products`, `POST /falabella/sync/products/{id}`
- Events: none
