# products/runtime

Module composition root for product wiring and OpenAPI artifact exposure.

## Key methods / endpoints / events
- Methods: `runtime.New(db)`, `(*runtime.Module).RegisterRoutes(router)`, `(*runtime.Module).Load(loader)`, `runtime.OpenAPISpec()`
- Endpoints: `/products`, `/products/:id`, `/variations`, `/variations/:id`
- Events: no integration events are emitted yet.
