# Shipping Runtime Package

`runtime` is the shipping module composition root responsible for adapter wiring, route registration, and module OpenAPI exposure.

## Responsibilities
- Build shipping quote dependencies from configuration values.
- Keep endpoints available in OpenAPI even when integration config is invalid.
- Register shipping quote HTTP routes.
- Expose module-level OpenAPI spec for aggregation.

## Key Methods / Endpoints / Events
- Methods:
  - `runtime.New(cfg, logger)`
  - `(*runtime.Module).RegisterRoutes(router)`
  - `(*runtime.Module).SetAuthorizer(authorizer)`
  - `(*runtime.Module).OpenAPISpec()`
  - `(*runtime.Module).Load(loader)`
- Endpoints:
  - `POST /shipping/quotes`
- Events: none in this package.
