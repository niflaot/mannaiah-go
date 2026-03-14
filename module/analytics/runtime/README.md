# Analytics Runtime Package

Composition root for analytics wiring and optional ClickHouse backend lifecycle.

## Key methods / endpoints / events
- Methods:
  - `New(cfg, db, registrar)`
  - `(*Module).Load(loader)`
  - `(*Module).QueryService()`
  - `(*Module).Stop()`
- Endpoints:
  - `GET /analytics/status`
  - `POST /analytics/seed`
- Events: none.
