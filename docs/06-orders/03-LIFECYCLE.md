# Orders — Lifecycle & Status System

---

## Order Statuses

An order can be in one of five states at any point in time.

| Status | Meaning |
|--------|---------|
| `PENDING` | Order has been received but has not started processing |
| `CREATED` | Order has been confirmed and is being prepared |
| `HOLD` | Order is temporarily paused (awaiting payment confirmation, stock check, etc.) |
| `COMPLETED` | Order has been fully fulfilled and shipped |
| `CANCELLED` | Order has been cancelled (either by the customer or operations) |

There is **no enforced finite state machine** — any transition between any two valid statuses is
permitted. The domain validates only that the incoming value is one of the five accepted constants.
Business-level transition rules (e.g. preventing re-opening a cancelled order) are enforced at
the operations process level, not in code.

---

## Status History — Source of Truth

Order status is tracked as an **append-only log** in `order_status_history`. The `CurrentStatus`
field on the `Order` object is a convenience snapshot derived from the last history entry; it is
never the authoritative source.

### StatusEntry

| Field | Type | Description |
|-------|------|-------------|
| `Status` | `Status` | The new status value |
| `Author` | `string` | Required — who made the change (user ID, system name, etc.) |
| `Description` | `string` | Optional human-readable reason for the transition |
| `NoteOwner` | `string` | Optional — associates an internal note to a specific owner |
| `Note` | `string` | Optional internal note text attached to this status transition |
| `OccurredAt` | `time.Time` | Timestamp of the transition |

The `position` column in `order_status_history` preserves insertion order,
so the complete lifecycle of an order is always fully auditable.

---

## Typical Lifecycle Examples

### Standard fulfilment

```
PENDING ──► CREATED ──► COMPLETED
```

| Status | Author | Description |
|--------|--------|-------------|
| `PENDING` | `woocommerce_sync` | WooCommerce order received |
| `CREATED` | `ops-agent` | Payment confirmed, preparing shipment |
| `COMPLETED` | `shipping-agent` | Dispatched and delivered |

---

### Order placed on hold

```
PENDING ──► HOLD ──► CREATED ──► COMPLETED
```

| Status | Author | Note |
|--------|--------|------|
| `PENDING` | `api` | |
| `HOLD` | `ops-maria` | Awaiting customer address confirmation |
| `CREATED` | `ops-maria` | Address confirmed |
| `COMPLETED` | `shipping-agent` | |

---

### Cancelled order

```
PENDING ──► CANCELLED
```

| Status | Author | Description |
|--------|--------|-------------|
| `PENDING` | `woocommerce_sync` | |
| `CANCELLED` | `customer-request` | Customer requested cancellation within 30 min |

---

## WooCommerce Source Guard

When an order's `Realm` is `"woocommerce"` (case-insensitive), the orders module enforces a
**write protection rule** to prevent the Mannaiah REST API from overwriting data that is the
authoritative responsibility of the WooCommerce sync pipeline.

The rule is evaluated on every `Update()` call using the `source` field from the request or the
`X-Sync-Source` HTTP header:

| `source` value | Mutation allowed? |
|----------------|-----------------|
| `woocommerce_sync` | ✓ Yes — sync agent is the legitimate owner |
| `woocommerce*` (any other prefix) | ✗ Silently no-op'd — treated as accidental override |
| `api` (default) | ✗ Silently no-op'd for woocommerce-realm orders |
| _(empty)_ | Falls back to `X-Sync-Source` header value |

This guard ensures WooCommerce orders are only mutated by the sync agent, while still allowing
status updates and comments (which use separate endpoints and are not subject to this guard).

---

## Idempotent Updates

`Update()` computes a `mutableStateSnapshot` before and after applying changes. If the snapshots
are identical (floating-point values compared with tolerance 0.000001), **no database write and no
event publish occur**. This makes repeated sync calls safe without creating spurious audit entries
or triggering downstream event consumers unnecessarily.
