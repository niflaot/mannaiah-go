# Segment HTTP Adapter

Exposes segment CRUD and resolution endpoints.

## Key methods / endpoints / events
- Methods:
  - `NewHandler(...)`
  - `(*Handler).RegisterRoutes(...)`
- Endpoints:
  - `POST /segments`
  - `GET /segments`
  - `GET /segments/:id`
  - `PATCH /segments/:id`
  - `DELETE /segments/:id`
  - `POST /segments/:id/resolve`
  - `GET /segments/:id/count`
- Events: none.
