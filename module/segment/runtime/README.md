# Segment Runtime Package

Composition root for segment module wiring and OpenAPI registration.

## Key methods / endpoints / events
- Methods:
  - `New(cfg, db, resolver)`
  - `(*Module).Load(loader)`
  - `(*Module).Service()`
- Endpoints:
  - `POST /segments`
  - `GET /segments`
  - `GET /segments/:id`
  - `PATCH /segments/:id`
  - `DELETE /segments/:id`
  - `POST /segments/:id/resolve`
  - `GET /segments/:id/count`
- Events: none.
