# Segment Module

Stores reusable audience segment definitions and resolves contact IDs through the analytics resolver (ClickHouse-backed).

## Key methods / endpoints / events
- Methods:
  - `Module.Service()`
- Endpoints:
  - `POST /segments`
  - `GET /segments`
  - `GET /segments/:id`
  - `PATCH /segments/:id`
  - `DELETE /segments/:id`
  - `POST /segments/:id/resolve`
  - `GET /segments/:id/count`
- Events: none.
