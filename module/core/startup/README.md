# Startup Package

`startup` provides module bootstrapping helpers for core composition roots.

## Key Methods / Endpoints / Events
- Methods:
  - `startup.NewRuntime(server, document)`
  - `(*startup.Runtime).RegisterRoutes(register)`
  - `(*startup.Runtime).AddOpenAPISpec(spec *openapi3.T)`
  - `(*startup.Runtime).ExposeOpenAPI(path)`
  - `(*startup.Runtime).ExposeOpenAPIUI(path, specPath, title)`
  - `startup.CoreSpec() *openapi3.T`
- Endpoints:
  - provides core OpenAPI specs for `/status`, `/openapi.json`, and `/docs`
- Events: none in this package.
