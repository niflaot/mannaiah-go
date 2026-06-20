# Shipping — HTTP API

All endpoints require a valid bearer token. No permission scope segmentation is currently
applied at the HTTP layer — authentication is the guard (subject to change).

---

## Quotation Endpoints

### Request a Quotation

```
POST /shipping/quotations
```

```json
{
  "carrierId": "tcc",
  "orderId": "order-uuid",
  "originCityCode": "11001",
  "destCityCode": "05001",
  "shipmentMode": "parcel",
  "declaredValue": 150000,
  "collectOnDeliveryAmount": 0,
  "units": [
    {
      "description": "Camiseta roja M",
      "packageType": "1",
      "dimensions": {
        "heightCM": 5, "widthCM": 25, "depthCM": 30,
        "realWeightKG": 0.3
      }
    }
  ]
}
```

`shipmentMode`: `"parcel"` or `"express"`.  
Quotation is stored with TTL = 24 h.

**Response** `201 Created`

```json
{
  "id": "q-uuid",
  "carrierId": "tcc",
  "orderId": "order-uuid",
  "originCityCode": "11001",
  "destCityCode": "05001",
  "freightCost": 12500,
  "estimatedDays": 2,
  "currencyCode": "COP",
  "expiresAt": "2026-03-29T10:00:00Z",
  "collectOnDeliveryAmount": 0,
  "collectOnDeliveryFeePercent": 0,
  "collectOnDeliveryFeeAmount": 0,
  "collectOnDeliveryChargedAmount": 0
}
```

---

### List Quotations

```
GET /shipping/quotations?orderID=<order-uuid>
```

Returns all non-expired quotations for a given order.

**Response** `200 OK`

```json
{ "data": [ /* QuotationResult[] */ ] }
```

---

### Preview Order Packaging (No Carrier Call)

```
POST /shipping/quotations/order-packaging
```

```json
{
  "orderIdentifier": "1024554",
  "carrierId": "tcc",
  "originCityCode": "11001"
}
```

Returns order packaging preview without calling carrier quotation APIs and without persisting quotation rows.

**Response** `200 OK`

```json
{
  "orderId": "9cd1e0ccacf39f0f1088ef81eb0d166a",
  "orderIdentifier": "1024554",
  "carrierId": "tcc",
  "originCityCode": "11001",
  "destCityCode": "11001",
  "declaredValue": 311000,
  "collectOnDeliveryAmount": 311000,
  "shipmentMode": "parcel",
  "units": [
    {
      "description": "7709738583238",
      "packageType": "CAJA",
      "dimensions": {
        "heightCm": 5,
        "widthCm": 40,
        "depthCm": 30,
        "realWeightKg": 1,
        "volumetricWeightKg": 2.4,
        "declaredValueCop": 157000
      }
    }
  ],
  "warnings": []
}
```

---

## Mark Endpoints

### Generate Mark (Direct)

```
POST /shipping/marks
```

Generates a label immediately by calling the carrier API. Use this path when you do not need
batch coordination.

```json
{
  "orderId": "order-uuid",
  "carrierId": "tcc",
  "quotationId": "q-uuid",
  "shipmentMode": "parcel",
  "declaredValue": 150000,
  "paymentForm": "CTA",
  "collectOnDeliveryAmount": 0,
  "observations": "Frágil",
  "sender": {
    "name": "Bodega Central",
    "legalName": "",
    "id": "", "idType": "",
    "addressLine": "Cra 15 #80-30",
    "cityCode": "11001",
    "phone": "3001234567",
    "email": "bodega@example.com"
  },
  "recipient": {
    "name": "Juan García",
    "addressLine": "Av El Dorado 92-34",
    "cityCode": "05001",
    "phone": "3109876543"
  },
  "units": [
    {
      "description": "Camiseta roja M",
      "packageType": "1",
      "dimensions": { "heightCM": 5, "widthCM": 25, "depthCM": 30, "realWeightKG": 0.3 }
    }
  ]
}
```

If `carrierId` has `RequiresBalanceCheck = true`, balance is verified before the carrier call.

**Response** `201 Created`

```json
{
  "id": "m-uuid",
  "orderId": "order-uuid",
  "carrierId": "tcc",
  "status": "GENERATED",
  "trackingNumber": "TCC-98765",
  "documentType": "LINK",
  "documentRef": "https://somos.tcc.com.co/guias/TCC-98765.pdf",
  "shipmentMode": "parcel",
  "createdAt": "...",
  "updatedAt": "..."
}
```

On carrier rejection: `status = "FAILED"`, `failureReason` is set, still `201 Created` —
the mark record is always persisted.

---

### List Marks

```
GET /shipping/marks
```

**Query parameters**

| Param | Notes |
|-------|-------|
| `orderID` | Filter by order UUID |
| `batchID` | Filter by dispatch batch UUID |
| `page`, `limit` | Pagination |

**Response** `200 OK`

```json
{ "data": [/* ShippingMark[] */], "meta": { "page": 1, "limit": 20, "total": 83 } }
```

---

### Get Mark

```
GET /shipping/marks/:id
```

**Response** `200 OK` with the full `ShippingMark` object or `404`.

---

### Get Related Marks

```
GET /shipping/marks/:id/related
```

Returns marks sharing the same `OrderID` or `DispatchBatchID` as the given mark (excluding
the mark itself). Sorted by `CreatedAt DESC`. Useful for showing shipment history for an
order.

**Response** `200 OK`

```json
{ "data": [/* ShippingMark[] */] }
```

---

### Void Mark

```
PATCH /shipping/marks/:id/void
{ "reason": "Incorrect recipient address" }
```

Sets `status = VOIDED`. **No carrier API call is made.** Out-of-band cancellation with the
carrier is the operator's responsibility.

**Response** `200 OK` with updated mark.

---

### Get Order Dispatch

```
GET /shipping/orders/:orderID/dispatch
```

Returns the highest-priority active shipping mark for the given order. Priority:
`QUOTED (3) > CREATED (2) > GENERATED (1)`. `FAILED`, `VOIDED`, and `REMOVED` marks
are excluded.

**Response** `200 OK` with the active `ShippingMark` or `404` if none found.

---

## Batch Endpoints

### Create Batch

```
POST /shipping/batches
{ "carrierId": "tcc", "createdBy": "ops-maria" }
```

**Response** `201 Created`
```json
{ "id": "b-uuid", "carrierId": "tcc", "status": "OPEN", "createdBy": "ops-maria", "markIDs": [], "createdAt": "..." }
```

---

### List Batches

```
GET /shipping/batches?carrierId=tcc&status=OPEN&page=1&limit=20
```

**Response** `200 OK`
```json
{ "data": [/* DispatchBatch[] */], "meta": { ... } }
```

---

### Get Batch

```
GET /shipping/batches/:id
```

**Response** `200 OK` with the full `DispatchBatch` or `404`.

---

### Add Mark to Batch

```
POST /shipping/batches/:id/marks
```

Body is the same as `POST /shipping/marks` minus `carrierId` (inherited from batch).
For manual carriers, you can also provide:
- `trackingNumber` (optional): operator-provided guide/reference.
- `customTrackingUrl` (optional): full tracking link override used in customer communications.

Creates a mark with `status = QUOTED` and `DispatchBatchID = batch.ID`.

**Errors:** `400` if batch is closed, `400` if carrier mismatch.

**Response** `200 OK` with the updated `DispatchBatch` (including the new mark ID in `markIDs`).

---

### Create Batch Mark (Draft or Direct)

```
POST /shipping/batches/marks
```

`batch` and `quotationId` are always required.

`direct=false` (default or omitted): creates a `QUOTED` draft mark and requires the batch to be open.  
`direct=true`: creates and materializes the mark immediately and assigns it to the batch even when the batch is closed.

```json
{
  "batch": "b-uuid",
  "direct": true,
  "quotationId": "q-uuid"
}
```

**Response** `201 Created` with the `ShippingMark` object.

If `direct=true`, guardrails are evaluated right before the carrier dispatch call. On violation,
the endpoint returns `500` with:
- `message`: `shipping_guardrail_violation`
- `error`: includes `mark_id`, `order_id`, `rule`, and `request_preview`.

---

### Remove Mark from Batch

```
DELETE /shipping/batches/:id/marks/:markId
```

Permanently deletes the mark. Only `QUOTED` marks may be removed.

**Response** `200 OK` with the updated `DispatchBatch`.

---

### Close Batch

```
PATCH /shipping/batches/:id/close
```

Submits all `QUOTED` marks to the carrier.

Guardrails are evaluated right before each carrier dispatch call:
- Non-guardrail carrier failures are logged and processing continues for the rest of marks.
- Guardrail violations abort batch close and return `500`.

**Response** `200 OK` with the closed `DispatchBatch`.

On guardrail violation:
- `message`: `shipping_guardrail_violation`
- `error`: includes `mark_id`, `order_id`, `rule`, and `request_preview`.

---

### Get Manifest Document

```
GET /shipping/batches/:id/manifest-document
```

Returns a merged PDF of all `CREATED` mark labels plus a summary page. Result is Redis-cached.

**Response** `200 OK` with `Content-Type: application/pdf` body.

---

### Get Mark Rotulus Document

```
GET /shipping/marks/:id/rotulus-document
```

Returns a half-letter PDF rotulus for the mark. The QR payload is signed with HMAC and the rendered PDF is cacheable.

**Permissions:** any of `shipping:generate`, `shipping:quotations`, or `order:view`.

**Response** `200 OK` with `Content-Type: application/pdf` body.

```http
GET /shipping/marks/:id/document
```

Returns the carrier shipping-label PDF for one mark. The backend stamps a compact `CONTENIDO` footer using order-summary item labels, including quantities and local product variation labels resolved from variant SKUs when available.

**Response** `200 OK` with `Content-Type: application/pdf` body.

---

## Tracking Endpoint

### Get Tracking History

```
GET /shipping/tracking/:trackingNumber?carrier=tcc
```

Fetches tracking history from the carrier and caches the result in Redis.

**Response** `200 OK`

```json
{
  "carrierId": "tcc",
  "trackingNumber": "TCC-98765",
  "globalStatus": "COMPLETED",
  "lastUpdate": "2026-03-28T14:30:00Z",
  "history": [
    { "date": "2026-03-28T08:00:00Z", "code": "1001", "text": "En origen", "city": "Bogotá", "status": "ORIGIN" },
    { "date": "2026-03-28T14:30:00Z", "code": "3000", "text": "Entregado", "city": "Medellín", "status": "COMPLETED" }
  ]
}
```

---

## Carrier Endpoints

### List Carriers

```
GET /shipping/carriers
```

Returns all registered carriers.

```json
{ "data": [
  { "id": "tcc", "name": "TCC", "type": "API", "active": true, "requiresBalanceCheck": true },
  { "id": "manual", "name": "Manual", "type": "MANUAL", "active": true, "requiresBalanceCheck": false }
]}
```

---

### Get Carrier

```
GET /shipping/carriers/:id
```

**Response** `200 OK` with the `Carrier` object or `404`.
