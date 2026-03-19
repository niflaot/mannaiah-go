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

## Affinity DSL (Percentage Only)
- Absolute affinity thresholds are deprecated for segment creation/update.
- Use percentage thresholds only:
  - field: `minScorePct`
  - range: `0` to `100`
- Tag affinity supports related tags in the same rule:
  - optional field: `relatedTags` (string array)
  - matching is done over `tag + relatedTags` and the percentage check is relative to each contact's top affinity score in that affinity domain.
- Supported affinity rules:
  - `tag_affinity`:
    - `{"type":"tag_affinity","parameters":{"tags":[{"tag":"gimnasio","relatedTags":["deportivo","urbano"],"minScorePct":70}]}}`
  - `category_affinity`:
    - `{"type":"category_affinity","parameters":{"categories":[{"categoryId":"cat-123","minScorePct":60}]}}`
  - `variation_affinity`:
    - `{"type":"variation_affinity","parameters":{"variations":[{"name":"size","value":"grande","minScorePct":55}]}}`
- Exclusion works the same way with `exclude: true`:
  - `{"type":"tag_affinity","exclude":true,"parameters":{"tags":[{"tag":"casual","minScorePct":40}]}}`
