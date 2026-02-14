# Orders HTTP Adapter Package

`module/orders/adapter/http` exposes order endpoints over Fiber through core HTTP abstractions.

## Key Methods / Endpoints / Events
- Methods:
  - `http.NewHandler(service, authorizers...)`
  - `(*http.Handler).SetAuthorizer(authorizer)`
  - `(*http.Handler).RegisterRoutes(router)`
  - `(*http.Handler).mapError(err)`
- Endpoints:
  - `POST /orders`
  - `GET /orders`
  - `GET /orders/:id`
  - `PATCH /orders/:id/status`
- Events: none in this package.
