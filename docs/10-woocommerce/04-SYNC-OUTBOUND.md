# WooCommerce — Outbound Sync (Mannaiah → WooCommerce)

When an order in Mannaiah changes (status update, item edit, shipping change), the
WooCommerce module pushes those changes back to the originating WooCommerce store. This
keeps the store's order view consistent with Mannaiah's authoritative state.

---

## Trigger

The outbound path is **event-driven**. The module subscribes to three topics on the
integration event bus:

| Topic | Source |
|-------|--------|
| `orders.v1.order.created` | `module/orders` |
| `orders.v1.order.updated` | `module/orders` |
| `orders.v1.order.status_updated` | `module/orders` |

All three topics are handled by the same handler:
`MainstreamUpdateService.HandleOrderEvent(ctx, payload)`.

---

## Guard Conditions

Three conditions must **all** be true for the outbound write to proceed:

| # | Condition | Purpose |
|---|-----------|---------|
| 1 | `payload.Realm == "woocommerce"` | Only WooCommerce-realm orders are mirrored back |
| 2 | `event.source != "woocommerce_sync"` | **Loop prevention** — events originating from the inbound sync are never pushed back |
| 3 | `payload.Identifier` parses as an integer | WooCommerce order IDs are integers; non-numeric identifiers are from other realms |

If any condition fails, the event is silently acknowledged and no WooCommerce API call is made.

---

## Loop Prevention in Detail

Without the source guard, the following feedback loop would occur:

```
WooCommerce order #1042 changed
  → Woo → Mannaiah inbound sync
  → Mannaiah order updated
  → orders.v1.order.updated event emitted (source="woocommerce_sync")
  → HandleOrderEvent checks: source == "woocommerce_sync" → STOP ✅
```

If the guard were absent:

```
  ... → Mannaiah order updated → outbound handler fires
  → PUT /wc/v3/orders/1042 → WooCommerce updated
  → WooCommerce emits webhook (if configured) → inbound sync again
  → ∞ loop
```

The `source` field in the `OrderEventPayload` is the single authoritative guard. It is set
to `"api"` by the HTTP handler and to `"woocommerce_sync"` by the order upsert path.

---

## Outbound API Call

When all guards pass, the handler:

1. Calls `destination.Validate(ctx)` — verifies connectivity (same check as inbound sync).
2. Maps `OrderEventPayload` → `MainstreamOrderUpdateCommand` (see [05-FIELD-MAPPING.md](05-FIELD-MAPPING.md)).
3. Calls `destination.UpdateOrderFromMainstream(cmd)`.

### WooCommerce API call

```
PUT /wp-json/wc/v3/orders/{identifier}
```

The diff sent to WooCommerce:

```json
{
  "status": "processing",
  "line_items": [
    { "product_id": 456, "quantity": 2, "subtotal": "89900", "total": "179800" }
  ],
  "shipping_lines": [
    { "method_id": "standard", "method_title": "Envío estándar", "total": "8900" }
  ],
  "shipping": {
    "first_name": "Juan",
    "last_name": "García",
    "address_1": "Calle 80 # 23-45",
    "city": "Bogotá",
    "phone": "+573101234567"
  }
}
```

**Product ID resolution:** Line items carry Mannaiah's internal product IDs. Before building
the WooCommerce payload, the adapter calls:

```
GET /wp-json/wc/v3/products?sku={sku}
```

once per unique SKU to resolve the WooCommerce `product_id`. Results are cached per sync
operation (not persisted). If a SKU cannot be resolved, the item is skipped.

---

## Status Mapping (Domain → WooCommerce)

`normalizeWooStatus` is the inverse of the inbound status mapping:

| Domain Status | WooCommerce status |
|--------------|-------------------|
| `PENDING` | `"pending-payment"` |
| `CREATED` | `"processing"` |
| `HOLD` | `"on-hold"` |
| `COMPLETED` | `"completed"` |
| `CANCELLED` | `"cancelled"` |

---

## Error Handling

| Scenario | Behaviour |
|----------|-----------|
| WooCommerce unreachable | Circuit breaker trips; event handler returns error; event bus may retry |
| Invalid identifier (non-numeric) | Guard condition 3 catches this before any API call |
| SKU not found in WooCommerce | Item skipped in line_items patch |
| WooCommerce returns `4xx` | Error logged; no retry at handler level |

---

## Example: Operator Updates Order Status to COMPLETED

```
ops-laura  (via Mannaiah HTTP API)
  │
  ├─ PATCH /orders/order-uuid/status
  │   { status: "COMPLETED", author: "ops-laura", source: "api" }
  │
  │   orders module:
  │     Append StatusEntry { status: COMPLETED, source: "api" }
  │     Publish orders.v1.order.status_updated {
  │       realm: "woocommerce",
  │       identifier: "1042",
  │       latestStatus: { status: "COMPLETED", ... },
  │       source: "api"
  │     }
  │
  │   WooCommerce module (HandleOrderEvent):
  │     Guard 1: realm == "woocommerce" ✅
  │     Guard 2: source "api" ≠ "woocommerce_sync" ✅
  │     Guard 3: identifier "1042" is numeric ✅
  │
  │     destination.Validate() ← GET /wp-json/wc/v3/orders?per_page=1 → 200
  │
  │     PUT /wp-json/wc/v3/orders/1042
  │       { "status": "completed" }
  │
  │     WooCommerce order #1042 → status: completed ✅
```

---

## Example: WooCommerce Sync Should NOT Trigger Push-Back

```
[Cron tick or manual Woo→Mannaiah sync]

OrderSyncService.SyncOrders(...)
  ├─ Maps WooOrder #1042 → OrderSyncCommand { source: "woocommerce_sync", ... }
  ├─ UpsertByIdentifier(realm="woocommerce", id="1042", cmd)
  │   → Mannaiah order updated
  │   → orders module publishes: orders.v1.order.updated { source: "woocommerce_sync" }
  │
  │   HandleOrderEvent:
  │     Guard 2: source == "woocommerce_sync" → STOP ✅  (no WooCommerce API call)
```
