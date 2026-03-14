# Analytics Module

Optional analytics sidecar module with ClickHouse health and seed endpoints.

## Key methods / endpoints / events
- Methods:
  - `Module.QueryService()`
- Endpoints:
  - `GET /analytics/status`
  - `POST /analytics/seed`
- Events:
  - consumes `contacts.v1.*`, `orders.v1.*`, `membership.v1.changed`, `campaign.v1.delivery` when configured.
