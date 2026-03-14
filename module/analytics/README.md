# Analytics Module

ClickHouse-first analytics module with live event ingestion, historical seed, and analytical resolver support for segment queries.

## Key methods / endpoints / events
- Methods:
  - `Module.QueryService()`
- Endpoints:
  - `GET /analytics/status`
  - `POST /analytics/seed`
- Events:
  - consumes `contacts.v1.*`, `orders.v1.*`, `membership.v1.changed`, `campaign.v1.delivery`.
