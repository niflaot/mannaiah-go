# Orders Module

`module/orders` provides normalized order management with contact linkage, item resolution, and status history.

## Key Methods / Endpoints / Events
- Methods:
  - `orders.New(db, customerSource, resolvers...)`
  - `orders.OpenAPISpec()`
- Endpoints:
  - `POST /orders`
  - `GET /orders`
  - `GET /orders/:id`
  - `PATCH /orders/:id/status`
- Events: none in this module yet.
