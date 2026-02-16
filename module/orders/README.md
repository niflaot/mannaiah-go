# Orders Module

`module/orders` provides normalized order management with contact linkage, item resolution, and status history.

## Key Methods / Endpoints / Events
- Methods:
  - `orders.New(db, customerSource, resolvers...)`
  - `orders.NewWithPublisher(db, customerSource, publisher, resolvers...)`
  - `orders.OpenAPISpec()`
- Endpoints:
  - `POST /orders`
  - `GET /orders`
  - `GET /orders/:id`
  - `PATCH /orders/:id`
  - `PATCH /orders/:id/status`
  - `POST /orders/:id/comments`
  - `PATCH /orders/:id/comments/:commentId`
  - `DELETE /orders/:id/comments/:commentId`
- Events:
  - `orders.v1.created`
  - `orders.v1.updated`
  - `orders.v1.status.updated`
