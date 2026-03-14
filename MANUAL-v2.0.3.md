# Mannaiah v2.0.3 User Manual (Marketing + BI)

This guide is for frontend automation/AI and product teams integrating the marketing and BI functions delivered in 2.0+.

## Scope

- Analytics (ClickHouse-backed)
- Membership consent timeline
- Segments and audience resolution
- Campaign planning and send orchestration
- Email delivery tracking
- Sync-run observability

## Important behavior changes

- `POST /contacts/optin` and `POST /contacts/optout` were removed from Contacts.
- Consent operations must use Membership endpoints.
- Segment resolution is ClickHouse-first via Analytics resolver (no SQL fallback path).

## Auth model

- Most marketing routes require Bearer auth with scope `marketing:manage`.
- SES webhook route is provider-facing and does not use marketing auth:
  - `POST /email/webhooks/ses`

## Required environment setup

Enable full BI + marketing stack:

- `ANALYTICS_ENABLED=true`
- `ANALYTICS_CLICKHOUSE_DSN=clickhouse://<user>:<pass>@<host>:9000/<db>`
- `ANALYTICS_CLICKHOUSE_MIGRATION_ENABLED=true`
- `SEGMENT_ENABLED=true`
- `MEMBERSHIP_ENABLED=true`
- `CAMPAIGN_ENABLED=true`
- `EMAIL_ENABLED=true`
- `SYNC_RECORD_ENABLED=true`

Startup guard:

- If `SEGMENT_ENABLED=true` and `ANALYTICS_ENABLED=false`, service startup fails.

## End-to-end data flow

1. Core modules persist transactional data (contacts/orders/membership/email/campaigns) in primary DB.
2. Analytics module builds BI model in ClickHouse using:
- `POST /analytics/seed` (initial backfill)
- event consumers (`contacts.v1.*`, `orders.v1.*`, `membership.v1.changed`, `campaign.v1.delivery`)
3. Segment module resolves audiences from ClickHouse.
4. Campaign module resolves contacts + emails from segments and sends asynchronously.
5. Delivery outcomes are recorded and published for analytics/event-driven tracking.

## Route manual

### Analytics

- `GET /analytics/status`
- `POST /analytics/seed`

`GET /analytics/status` response:

```json
{
  "enabled": true,
  "backendHealthy": true,
  "error": ""
}
```

`POST /analytics/seed` response:

```json
{
  "contacts": 12000,
  "orders": 43000,
  "orderItems": 91000,
  "membershipEvents": 8400,
  "campaignEvents": 15300
}
```

### Membership

- `POST /membership/optin`
- `POST /membership/optout`
- `POST /membership/stamp`
- `GET /membership/status/:contactId`
- `GET /membership/status/:contactId/:channel`
- `GET /membership/status/:contactId/stamps`
- `GET /membership/stamps/:contactId/:channel`
- `POST /membership/migrate`

Request (`optin` / `optout`):

```json
{
  "contactId": "<optional>",
  "email": "<optional if contactId is omitted>",
  "channel": "email",
  "source": "checkout",
  "occurredAt": "2026-03-14T15:00:00Z"
}
```

Request (`stamp`):

```json
{
  "contactId": "...",
  "email": "...",
  "channel": "email",
  "action": "opt_in",
  "source": "campaign_footer",
  "occurredAt": "2026-03-14T15:00:00Z"
}
```

Status response:

```json
{
  "contactId": "...",
  "channel": "email",
  "action": "opt_in",
  "source": "checkout",
  "occurredAt": "2026-03-14T15:00:00Z",
  "updatedAt": "2026-03-14T15:00:01Z"
}
```

### Segments

- `POST /segments`
- `GET /segments`
- `GET /segments/:id`
- `PATCH /segments/:id`
- `DELETE /segments/:id`
- `POST /segments/:id/resolve?page=1&limit=1000`
- `GET /segments/:id/count`

Create/update body:

```json
{
  "name": "High value subscribed",
  "slug": "high-value-subscribed",
  "channel": "email",
  "filters": [
    {"type": "city", "parameters": {"codes": ["BOG", "MDE"]}},
    {"type": "min_total_spend", "value": 500000},
    {"type": "opt_in_status", "parameters": {"channel": "email", "status": "opt_in"}}
  ]
}
```

Resolve response:

```json
{
  "segmentId": "...",
  "contactIds": ["c1", "c2", "c3"]
}
```

Count response:

```json
{
  "segmentId": "...",
  "count": 1234
}
```

### Campaigns

- `POST /campaigns`
- `GET /campaigns`
- `GET /campaigns/:id`
- `PATCH /campaigns/:id`
- `DELETE /campaigns/:id`
- `POST /campaigns/:id/send`

Create body:

```json
{
  "name": "Weekend promo",
  "slug": "weekend-promo",
  "channel": "email",
  "segmentId": "seg_123",
  "subject": "Weekend promo",
  "htmlBody": "<h1>Hello</h1>",
  "textBody": "Hello"
}
```

Campaign status flow:

- `PLANNED` -> `PROCESSING` -> `SENT` or `FAILED`

Send endpoint returns `202` and runs asynchronously.

### Email

- `POST /email/send`
- `GET /email/deliveries/:id`
- `POST /email/webhooks/ses`

Send body:

```json
{
  "contactId": "<optional>",
  "email": "user@example.com",
  "subject": "Hi",
  "htmlBody": "<p>Hi</p>",
  "textBody": "Hi",
  "idempotencyKey": "campaign123:contact456"
}
```

Delivery statuses:

- `pending`
- `submitted_to_provider`
- `failed_retryable`
- `failed_permanent`
- `delivered`
- `bounced`
- `complained`

### Sync Record

- `GET /syncrecord/runs`
- `GET /syncrecord/runs/:id`
- `GET /syncrecord/stats?windowHours=24`

Supported list filters:

- `kind`, `trigger`, `status`
- `startedAfter`, `startedBefore` (RFC3339)
- `page`, `limit`

Key run kinds to monitor:

- `analytics.seed`
- `membership.migration`
- `campaign.send`
- `woocommerce.contacts`
- `woocommerce.orders`

## Segment filter DSL

Supported filters:

1. `city`
- `parameters.codes: string[]`

2. `order_recency`
- `parameters.days: int`

3. `no_order_recency`
- `parameters.days: int`

4. `category`
- `parameters.pattern: string`

5. `top_spenders`
- `parameters.limit: int` or `parameters.percentage: float`

6. `first_purchase_only`
- optional `parameters.enabled: bool` (defaults to true)

7. `subscribed_no_buy`
- optional `parameters.enabled: bool` (defaults to true)

8. `opt_in_status`
- `parameters.channel: string`
- `parameters.status: string` (`opt_in` or `opt_out`)

9. `metadata`
- `parameters.key: string`
- optional `parameters.value: string`

Compatibility filters still accepted:

- `city_code_in`
- `min_total_spend`
- `email_opt_in`
- `purchased_sku`

## Deployment/migration operations

### SQL migrations (inside repo, using current `.env`)

```bash
go run ./module/core/cmd/migrate --operation version
go run ./module/core/cmd/migrate --operation up
```

Dirty state recovery example (`dirty version 10`):

```bash
go run ./module/core/cmd/migrate --operation force --force-version 9
go run ./module/core/cmd/migrate --operation up
```

### ClickHouse migration in `fl-mannaiah` container

ClickHouse schema is auto-applied by API startup (no separate migration CLI in runtime image). Ensure envs are present, then seed:

```bash
printenv | grep '^ANALYTICS_'
curl -sS -X POST http://127.0.0.1:8080/analytics/seed \
  -H "Authorization: Bearer $DEV_AUTH_TOKEN" \
  -H "Content-Type: application/json"
curl -sS http://127.0.0.1:8080/analytics/status \
  -H "Authorization: Bearer $DEV_AUTH_TOKEN"
```

If `backendHealthy=false`, fix DSN/env and restart container.

## BI tables in ClickHouse

- `contacts_snapshot`
- `orders_fact`
- `order_items_fact`
- `membership_events`
- `campaign_events`

Segment resolution uses these tables as source.
