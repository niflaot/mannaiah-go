# Shipping — Integration Events

All events are published via the core messaging bus. Downstream modules subscribe to these
topics to react asynchronously to shipping state changes. The WooCommerce module, for
example, uses `shipping.v1.mark.generated` to write tracking information back to a
WooCommerce order.

---

## Topics

| Topic | Trigger |
|-------|---------|
| `shipping.v1.mark.generated` | Carrier successfully assigned a tracking number (direct or batch close) |
| `shipping.v1.mark.failed` | Carrier rejected the mark submission |
| `shipping.v1.mark.voided` | Mark voided locally by an operator |
| `shipping.v1.batch.created` | New dispatch batch opened |
| `shipping.v1.batch.closed` | Batch closed after carrier submission |
| `shipping.v1.tracking.updated` | Tracking history queried and cached |

---

## Payloads

### `shipping.v1.mark.generated`

```json
{
  "markId": "m-uuid",
  "orderId": "order-uuid",
  "carrierId": "tcc",
  "trackingNumber": "TCC-98765",
  "documentRef": "https://somos.tcc.com.co/guias/TCC-98765.pdf",
  "shipmentMode": "parcel",
  "dispatchBatchId": "b-uuid"
}
```

`dispatchBatchId` is `null` when the mark was generated directly (outside batch flow).

---

### `shipping.v1.mark.failed`

```json
{
  "markId": "m-uuid",
  "orderId": "order-uuid",
  "carrierId": "tcc",
  "failureReason": "invalid destination city code",
  "dispatchBatchId": "b-uuid"
}
```

---

### `shipping.v1.mark.voided`

```json
{
  "markId": "m-uuid",
  "orderId": "order-uuid",
  "carrierId": "tcc",
  "reason": "Incorrect recipient address"
}
```

---

### `shipping.v1.batch.created`

```json
{
  "batchId": "b-uuid",
  "carrierId": "tcc",
  "createdBy": "ops-maria"
}
```

---

### `shipping.v1.batch.closed`

```json
{
  "batchId": "b-uuid",
  "carrierId": "tcc",
  "marksCreated": 14,
  "marksFailed": 2,
  "closedAt": "2026-03-28T15:00:00Z"
}
```

---

### `shipping.v1.tracking.updated`

```json
{
  "carrierId": "tcc",
  "trackingNumber": "TCC-98765",
  "globalStatus": "COMPLETED",
  "eventCount": 5
}
```

---

## Trace Context Propagation

All events carry the OpenTelemetry `traceparent` metadata header, enabling downstream
consumers to continue the same distributed trace that originated the shipping operation.
