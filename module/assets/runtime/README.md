# assets/runtime

Composition root for the assets module: repository + service + HTTP adapter + OpenAPI spec.

## Key methods / endpoints / events
- Methods: `runtime.New(db, storage, publishers...)`, `(*runtime.Module).RegisterRoutes`, `(*runtime.Module).SetAuthorizer`, `(*runtime.Module).Load`, `runtime.OpenAPISpec()`
- Endpoints: `POST /assets`, `GET /assets`, `GET /assets/{id}`, `PATCH /assets/{id}`, `DELETE /assets/{id}`
- Events: lifecycle events emitted by application service through injected publisher.
