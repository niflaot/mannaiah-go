# Orders — Integration Events

The orders module publishes integration events through the core messaging bus whenever order
state changes. Consumers subscribe to these topics to react asynchronously — the analytics
module, the campaign module, and external sync agents are typical recipients.

---

## Topics

| Topic | Trigger |
|-------|---------|
| `orders.v1.created` | A new order is created via `Create()` |
| `orders.v1.updated` | An order's items, address, charges, or comments are changed by `Update()`, `AddComment()`, `UpdateComment()`, or `DeleteComment()` |
| `orders.v1.status.updated` | A new status entry is appended by `UpdateStatus()` |

> All three topics carry the same `OrderEventPayload` envelope. The differentiator is
> the topic name, not the payload structure.

---

## Payload Schema

```json
{
  "id": "uuid",
  "identifier": "WC-1042",
  "realm": "woocommerce",
  "contactId": "contact-uuid",
  "source": "woocommerce_sync",
  "currentStatus": "CREATED",
  "latestStatus": {
    "status": "CREATED",
    "author": "ops-maria",
    "description": "Payment confirmed",
    "noteOwner": "ops-maria",
    "note": "Stripe payment ID: pi_xxx",
    "occurredAt": "2026-03-10T09:30:00Z"
  },
  "items": [
    {
      "sku": "SHIRT-001-RED-M",
      "alternateName": "Camiseta Roja M",
      "quantity": 2,
      "value": 89900,
      "productId": "product-uuid",
      "resolutionSource": "sku"
    }
  ],
  "shippingAddress": {
    "address": "Calle 80 # 23-45",
    "address2": "Apto 301",
    "phone": "+57 310 000 0000",
    "cityCode": "BOG"
  },
  "hasCustomShippingAddress": true,
  "shippingCharges": [
    { "methodId": "standard", "methodTitle": "Envío estándar", "price": 8900 }
  ],
  "metadata": { "woo_order_id": "1042" },
  "createdAt": "2026-03-10T08:00:00Z",
  "updatedAt": "2026-03-10T09:30:00Z"
}
```

### Payload Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string (UUID)` | Internal order UUID |
| `identifier` | `string` | External order reference |
| `realm` | `string` | Source realm (e.g. `"woocommerce"`) |
| `contactId` | `string` | UUID of the linked contact |
| `source` | `string` | Mutation source (see below) |
| `currentStatus` | `string` | Most recent status value |
| `latestStatus` | `StatusEntry` | Full latest status entry |
| `items` | `OrderEventItem[]` | Line item snapshot at event time |
| `shippingAddress` | `ShippingAddress` | Shipping destination |
| `hasCustomShippingAddress` | `bool` | Whether address was set explicitly |
| `shippingCharges` | `ShippingCharge[]` | Applied shipping methods |
| `metadata` | `object` | Arbitrary key-value pairs from the source system |
| `createdAt` | `string (RFC3339)` | When the order was first created |
| `updatedAt` | `string (RFC3339)` | When this change occurred |

> `latestStatus` reflects the status entry that triggered or was current at event time. For
> `orders.v1.status.updated` events this is the freshly appended entry. For
> `orders.v1.updated` events it is the most recent pre-existing status entry (unchanged).

---

## Source Constants

| Value | Meaning |
|-------|---------|
| `"api"` | Human operator via HTTP API (default when `source` is omitted from the request body) |
| `"woocommerce_sync"` | WooCommerce sync agent |

The `source` field may be set explicitly in the HTTP request body. When omitted, the
`X-Sync-Source` request header is checked next; if also absent, `"api"` is used.

---

## Topic Semantics

### `orders.v1.created`

Emitted once after a successful `Create()`. The payload represents the initial state of the
order immediately after persistence, before any subsequent status or comment changes.

**Common consumers:**
- Analytics — to start an order journey funnel entry
- Campaign — to trigger post-purchase automation

### `orders.v1.updated`

Emitted after any mutation that changes order content: item list, shipping address, shipping
charges, or comments. Status history changes do **not** fire this topic — they fire
`orders.v1.status.updated` instead.

If an update is submitted but the resulting state is identical to the stored state (idempotent
no-op), **no event is emitted**.

### `orders.v1.status.updated`

Emitted each time a new `StatusEntry` is appended via `UpdateStatus()`. Because status history
is append-only, every call to `UpdateStatus()` always results in this event.

**Common consumers:**
- Campaign — to trigger status-based automations (e.g. "Order on hold" email)
- Analytics — to record funnel stage transitions

---

## Trace Context Propagation

All events carry the OpenTelemetry `traceparent` metadata header so that downstream
consumers can continue the same distributed trace that originated the mutation. This enables
end-to-end trace correlation across HTTP → messaging boundaries without any additional
configuration in the consumer.
