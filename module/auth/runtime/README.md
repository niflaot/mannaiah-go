# auth/runtime

Composition root for auth verifier, service, HTTP adapter, and OpenAPI spec.

## Key methods / endpoints / events
- Methods: `runtime.New(cfg, coreEnvironment, logger)`, `(*runtime.Module).Require`, `(*runtime.Module).RegisterRoutes`, `(*runtime.Module).Load`, `runtime.OpenAPISpec()`
- Endpoints: `GET /check-auth`
- Events: none.
