# Swagger Package

`swagger` centralizes OpenAPI aggregation and route exposure for core and modules.

## Key Methods / Endpoints / Events
- Methods:
  - `swagger.NewDocument(info)`
  - `(*swagger.Document).Merge(spec *openapi3.T)`
  - `(*swagger.Document).Build() *openapi3.T`
  - `swagger.RegisterRoute(router, path, document)`
  - `swagger.RegisterUIRoute(router, path, specPath, title)`
- Endpoints:
  - typically exposes aggregated OpenAPI JSON at `/openapi.json`
  - optionally exposes Swagger UI HTML at `/docs`
- Events: none in this package.
