# Orders Runtime Package

`module/orders/runtime` wires order adapters, services, routes, and OpenAPI specs.

## Key Methods / Endpoints / Events
- Methods:
  - `runtime.New(db, customerSource, publisher, resolvers...)`
  - `(*runtime.Module).RegisterRoutes(router)`
  - `(*runtime.Module).SetAuthorizer(authorizer)`
  - `(*runtime.Module).Service()`
  - `(*runtime.Module).Load(loader)`
  - `runtime.OpenAPISpec()`
- Endpoints:
  - `POST /orders`
  - `GET /orders`
  - `GET /orders/:id`
  - `PATCH /orders/:id`
  - `PATCH /orders/:id/status`
  - `POST /orders/:id/comments`
- Events:
  - `orders.v1.created`
  - `orders.v1.updated`
  - `orders.v1.status.updated`
