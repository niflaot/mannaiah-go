# Orders — HTTP API

All order endpoints require a valid bearer token. The `order:*` family of scopes also requires
`contact:view` and `product:view` as cross-domain dependencies — see
[03-auth/01-AUTH.md](../03-auth/01-AUTH.md) for the full dependency table.

The optional `X-Sync-Source` header identifies the calling system and governs WooCommerce
write-guard logic. Pass `woocommerce_sync` from the sync agent; omit from human API calls.

---

## Create Order

```
POST /orders
Permissions: order:manage, contact:view, product:view
```

**Request body**

```json
{
  "identifier": "WC-1042",
  "realm": "woocommerce",
  "contactId": "contact-uuid",
  "paymentMethod": "stripe",
  "items": [
    {
      "sku": "SHIRT-001-RED-M",
      "alternateName": "Camiseta Roja M",
      "quantity": 2,
      "value": 89900
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
  "source": "woocommerce_sync"
}
```

- If `hasCustomShippingAddress` is `false` or omitted, the shipping address is derived from
  the linked contact's billing address automatically.
- Line items are resolved to Mannaiah product IDs in parallel (see [02-DOMAIN.md](02-DOMAIN.md)).

**Response** — `201 Created` with the full `Order` object.

---

## List Orders

```
GET /orders
Permissions: order:view, contact:view, product:view
```

**Query parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | `int` | Page number (default `1`) |
| `limit` | `int` | Records per page (default `20`) |
| `realm` | `string` | Filter by realm (e.g. `"woocommerce"`) |
| `contactId` | `string` | Filter by contact UUID |
| `identifier` | `string` | Filter by external order identifier |
| `status` | `string` | Filter by current status value |

**Response** — `200 OK`

```json
{
  "data": [ /* Order[] */ ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 540,
    "totalPages": 27
  }
}
```

---

## Get Order

```
GET /orders/:id
Permissions: order:view, contact:view, product:view
```

**Response** — `200 OK` with the full `Order` object or `404 Not Found`.

---

## Update Order

```
PATCH /orders/:id
Permissions: order:manage, contact:view, product:view
```

Updates items, shipping address, and/or shipping charges. Does not affect status history or
comments.

```json
{
  "items": [
    { "sku": "SHIRT-001-RED-L", "quantity": 1, "value": 89900 }
  ],
  "shippingAddress": {
    "address": "Carrera 15 # 80-30",
    "cityCode": "BOG"
  },
  "shippingCharges": [
    { "methodId": "express", "methodTitle": "Envío exprés", "price": 15000 }
  ],
  "source": "api"
}
```

**WooCommerce guard:** If the order's `Realm` is `"woocommerce"` and `source` is not
`"woocommerce_sync"`, the call is silently no-op'd (returns the unmodified order).

**Idempotency:** If the resulting state is identical to the current state, no write or event is
emitted.

**Response** — `200 OK` with the updated `Order` object.

---

## Update Order Status

```
PATCH /orders/:id/status
Permissions: order:triage, contact:view, product:view
```

Appends a new entry to `StatusHistory`. Does not replace prior entries.

```json
{
  "status": "HOLD",
  "author": "ops-maria",
  "description": "Awaiting payment confirmation",
  "noteOwner": "ops-maria",
  "note": "Called customer — expects payment by EOD",
  "source": "api"
}
```

**Response** — `200 OK` with the updated `Order` object (including the new status history entry).

---

## Add Comment

```
POST /orders/:id/comments
Permissions: order:triage, contact:view, product:view
```

```json
{
  "author": "ops-maria",
  "comment": "Customer requested express shipping upgrade.",
  "internal": false,
  "source": "api"
}
```

**Response** — `200 OK` with the updated `Order` object.

---

## Update Comment

```
PATCH /orders/:id/comments/:commentId
Permissions: order:triage, contact:view, product:view
```

```json
{
  "author": "ops-maria",
  "comment": "Customer confirmed: standard shipping is fine.",
  "internal": true
}
```

**Response** — `200 OK` with the updated `Order` object or `404` if `commentId` not found.

---

## Delete Comment

```
DELETE /orders/:id/comments/:commentId
Permissions: order:triage, contact:view, product:view
```

**Response** — `200 OK` with the updated `Order` object.

---

## Order Object Schema

```json
{
  "id": "uuid",
  "identifier": "WC-1042",
  "realm": "woocommerce",
  "contactId": "contact-uuid",
  "currentStatus": "CREATED",
  "statusHistory": [
    {
      "status": "PENDING",
      "author": "woocommerce_sync",
      "description": "Order synced from WooCommerce",
      "noteOwner": "",
      "note": "",
      "occurredAt": "2026-03-10T08:00:00Z"
    },
    {
      "status": "CREATED",
      "author": "ops-maria",
      "description": "Payment confirmed",
      "noteOwner": "ops-maria",
      "note": "Stripe payment ID: pi_xxx",
      "occurredAt": "2026-03-10T09:30:00Z"
    }
  ],
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
  "comments": [],
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
  "paymentMethod": "stripe",
  "metadata": { "woo_order_id": "1042" },
  "createdAt": "2026-03-10T08:00:00Z",
  "updatedAt": "2026-03-10T09:30:00Z"
}
```

---

## Error Reference

| HTTP Status | Condition |
|-------------|-----------|
| `400` | Validation failure (missing items, invalid status, quantity ≤ 0) |
| `401` | Missing or invalid bearer token |
| `403` | Token lacks required scope(s) |
| `404` | Order or comment not found |
| `409` | Duplicate `(realm, identifier)` combination |
| `500` | Internal server error |
