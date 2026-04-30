# Analytics Engine

The analytics module in `v2.0.0` is a BI-oriented ingestion layer. MySQL remains the transactional source of truth, while ClickHouse stores denormalized fact data for future dashboards and reporting.

## What It Keeps

- ClickHouse schema bootstrap
- Historical seed support for contacts, orders, order items, membership events, and product taxonomy
- Real-time ingestion from integration events:
  - `contacts.v1.created`
  - `contacts.v1.updated`
  - `orders.v1.created`
  - `orders.v1.updated`
  - `orders.v1.status.updated`
  - `membership.v1.changed`

## What Was Removed In `v2.0.0`

- Campaign event ingestion
- RFM scoring routes and persistence
- Affinity routes and refresh flows
- Segment resolution support
- Recommendation routes

The module remains registered so ClickHouse connectivity and fact ingestion continue to work, but it intentionally exposes no public analytics HTTP routes in this release.
