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
  - subscribes to `contacts.v1.*`, `orders.v1.*`, `membership.v1.changed`, `campaign.v1.delivery` when enabled.
