# WooCommerce — HTTP API

The WooCommerce module exposes two HTTP endpoints that allow operators to manually trigger
sync operations, bypassing the scheduled cron jobs.

---

## Sync Contacts

```
POST /woo/sync/contacts
Permission: contact:sync
```

Triggers a full contact sync from WooCommerce. All contacts are derived from billing
information across all WooCommerce orders.

**Response** `200 OK`

```json
{
  "trigger": "api",
  "processed": 1200,
  "created": 5,
  "updated": 1193,
  "unchanged": 2,
  "skipped": 0,
  "failed": 0
}
```

---

## Sync Single Contact by Email

```
POST /woo/sync/contacts?email=customer@example.com
Permission: contact:sync
```

Pages through WooCommerce orders filtered by billing email. Syncs only the single contact
matching that email. Useful for re-syncing a specific customer after a data correction.

**Response** `200 OK` — same `SyncSummary` structure; `processed = 1` on success.

---

## Sync Orders

```
POST /woo/sync/orders
Permission: order:sync
```

Triggers a full order sync. Each order upserts its linked contact first, then upserts the
order record.

**Response** `200 OK`

```json
{
  "trigger": "api",
  "processed": 3500,
  "created": 12,
  "updated": 3485,
  "unchanged": 3,
  "skipped": 0,
  "failed": 0
}
```

---

## Sync Single Order by WooCommerce ID

```
POST /woo/sync/orders?id=1042
Permission: order:sync
```

Fetches only order `#1042` via `GET /wp-json/wc/v3/orders/1042`. Upserts contact then order.

**Response** `200 OK` — `processed = 1` on success.

---

## Error Responses

| HTTP | Condition |
|------|-----------|
| `401` | Missing or invalid bearer token |
| `403` | Insufficient scope |
| `503` | WooCommerce integration validation failed (store unreachable) |
| `500` | Fatal sync error (returned in the `error` body field and published as `*.failed` event) |

---

## SyncSummary Schema

| Field | Type | Description |
|-------|------|-------------|
| `trigger` | `string` | Always `"api"` for HTTP-triggered syncs |
| `processed` | `int` | Total records fetched |
| `created` | `int` | Newly created |
| `updated` | `int` | Updated (state changed) |
| `unchanged` | `int` | No-op (idempotent) |
| `skipped` | `int` | Invalid / incomplete records |
| `failed` | `int` | Upsert failures |
