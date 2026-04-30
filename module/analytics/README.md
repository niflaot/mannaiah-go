# Analytics Module

ClickHouse-first analytics module with live event ingestion and historical seed support for BI-oriented fact storage.

## Key methods / endpoints / events
- Methods:
  - `Module.QueryService()`
- Events:
  - consumes `contacts.v1.*`, `orders.v1.*`, `membership.v1.changed`.

## Runtime Surface

The module remains registered for ClickHouse connectivity, schema bootstrap, and event ingestion.
It intentionally exposes no public analytics HTTP routes in `v2.0.0`.
