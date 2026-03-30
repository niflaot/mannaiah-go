# Analytics Module

ClickHouse-first analytics module with live event ingestion, historical seed, and analytical resolver support for segment queries.

## Key methods / endpoints / events
- Methods:
  - `Module.QueryService()`
- Endpoints:
  - `GET /analytics/status`
  - `POST /analytics/seed`
  - `GET /analytics/affinity/contacts/:contactId`
  - `GET /analytics/affinity/contacts/:contactId/tags`
  - `GET /analytics/affinity/contacts/:contactId/categories`
  - `GET /analytics/affinity/contacts/:contactId/variations`
  - `POST /analytics/affinity/refresh`
- Events:
  - consumes `contacts.v1.*`, `orders.v1.*`, `membership.v1.changed`, `campaign.v1.delivery`.

## Affinity Refresh Cron
- `ANALYTICS_AFFINITY_REFRESH_ENABLED` enables scheduled refresh execution.
- `ANALYTICS_AFFINITY_REFRESH_CRON` defines cron spec (example every 30 minutes: `*/30 * * * *`).
- `ANALYTICS_AFFINITY_REFRESH_TIMEOUT_MS` defines per-run timeout in milliseconds.
