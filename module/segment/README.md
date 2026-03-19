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

## Filter DSL Behavior
- Each filter entry supports:
  - `type` (required)
  - `exclude` (optional, default `false`)
  - `value` (optional)
  - `parameters` (optional)
- Resolution semantics:
  - All filter entries are combined with `AND`.
  - `exclude: false` applies the filter normally.
  - `exclude: true` negates the full filter condition.
- Order status scoping:
  - `order_status` filters define status scope for order-dependent filters.
  - Included statuses map to `IN (...)`.
  - Excluded statuses map to `NOT IN (...)`.
- Range-window example:
  - include buyers in last 90 days: `{"type":"order_recency","parameters":{"days":90}}`
  - exclude buyers in last 30 days: `{"type":"order_recency","exclude":true,"parameters":{"days":30}}`
  - Combined result: contacts with purchases between day 31 and day 90.
