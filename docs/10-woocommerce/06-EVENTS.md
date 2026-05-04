# WooCommerce — Integration Events

The WooCommerce module publishes lifecycle events for every sync operation. These events
allow monitoring systems, dashboards, and other modules to react to sync state changes
without polling.

---

## Topics

| Topic | Trigger |
|-------|---------|
| `woocommerce.v1.contacts.sync.started` | `SyncContacts` begins |
| `woocommerce.v1.contacts.sync.completed` | Contact sync finishes successfully |
| `woocommerce.v1.contacts.sync.failed` | Contact sync terminates with a fatal error |
| `woocommerce.v1.orders.sync.started` | `SyncOrders` begins |
| `woocommerce.v1.orders.sync.completed` | Order sync finishes successfully |
| `woocommerce.v1.orders.sync.failed` | Order sync terminates with a fatal error |

---

## Payload Schema

All six topics share the same envelope structure.

```json
{
  "trigger": "cron",
  "processed": 847,
  "created": 15,
  "updated": 830,
  "unchanged": 1,
  "skipped": 0,
  "failed": 1,
  "error": ""
}
```

| Field | Type | Notes |
|-------|------|-------|
| `trigger` | `string` | `"cron"` or `"api"` |
| `processed` | `int` | Total records fetched from WooCommerce |
| `created` | `int` | Records newly created in Mannaiah |
| `updated` | `int` | Records updated in Mannaiah |
| `unchanged` | `int` | Records that matched existing state exactly (no write) |
| `skipped` | `int` | Records skipped due to invalid data (empty email, etc.) |
| `failed` | `int` | Records that failed upsert |
| `error` | `string` | Non-empty only on `*.failed` topics; describes the fatal error |

---

## Example Payloads

### `woocommerce.v1.orders.sync.completed`

```json
{
  "trigger": "cron",
  "processed": 1200,
  "created": 8,
  "updated": 1185,
  "unchanged": 5,
  "skipped": 2,
  "failed": 0,
  "error": ""
}
```

### `woocommerce.v1.contacts.sync.failed`

```json
{
  "trigger": "api",
  "processed": 0,
  "created": 0,
  "updated": 0,
  "unchanged": 0,
  "skipped": 0,
  "failed": 0,
  "error": "WooCommerce API unreachable: connection refused"
}
```

---

## Consumed Events

The WooCommerce module also **consumes** events from the orders module to drive the outbound
sync direction:

| Consumed Topic | Publisher | Handler |
|---------------|-----------|---------|
| `orders.v1.order.created` | `module/orders` | `MainstreamUpdateService.HandleOrderEvent` |
| `orders.v1.order.updated` | `module/orders` | `MainstreamUpdateService.HandleOrderEvent` |
| `orders.v1.order.status_updated` | `module/orders` | `MainstreamUpdateService.HandleOrderEvent` |

See [04-SYNC-OUTBOUND.md](04-SYNC-OUTBOUND.md) for the guard conditions and loop-prevention
mechanism.
