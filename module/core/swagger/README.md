# Swagger Package

`swagger` centralizes OpenAPI aggregation and route exposure for core and modules.

## Key Methods / Endpoints / Events
- Methods:
  - `swagger.NewDocument(info)`
  - `(*swagger.Document).Merge(spec *openapi3.T)`
  - `(*swagger.Document).Build() *openapi3.T`
  - `swagger.RegisterRoute(router, path, document)`
- Endpoints:
  - typically exposes aggregated OpenAPI JSON at `/openapi.json`
- Events: none in this package.
