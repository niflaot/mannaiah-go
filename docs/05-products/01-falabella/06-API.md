# Falabella — HTTP API

All Falabella endpoints are mounted under `/falabella`. Most require a valid bearer token.

---

## Sync Endpoints

### Sync a Single Product

```
POST /falabella/sync/products/:id
Permission: product:manage
```

Triggers the full sync pipeline for one product. Returns a `Summary` object.

**Response**

```json
{
  "executionId": "uuid",
  "requested": 1,
  "synced": 1,
  "skipped": 0,
  "failed": 0,
  "results": [
    {
      "productId": "uuid",
      "sku": "SHIRT-001",
      "status": "synced"
    }
  ]
}
```

---

### Sync a Batch of Products

```
POST /falabella/sync/products
Permission: product:manage
```

**Request body**

```json
{
  "productIds": ["uuid1", "uuid2", "uuid3"]
}
```

Dispatches work across `FALABELLA_PRODUCT_SYNC_WORKERS` goroutines. Returns a `Summary` with
per-product results.

---

## Status Endpoints

### Get Feed Status

```
GET /falabella/sync/status/feed/:feedId
Permission: product:view
```

Returns the `SyncEntry` for the given Falabella feed ID.

---

### Get Execution

```
GET /falabella/sync/status/execution/:executionId
Permission: product:view
```

Returns the `SyncExecution` envelope.

---

### List Feeds for Execution

```
GET /falabella/sync/status/execution/:executionId/feeds
Permission: product:view
```

Returns all `SyncEntry` records for the given execution.

---

### List Feeds for Product

```
GET /falabella/sync/status/product/:productId
Permission: product:view
```

Returns all historical `SyncEntry` records for a Mannaiah product ID (most recent first).

---

### Manually Resolve a Feed

```
POST /falabella/sync/status/feed/:feedId/resolve
Permission: product:manage
```

Immediately calls `GetFeedStatus` for the given feed and updates the `SyncEntry`. Useful for
unblocking stuck pending entries without waiting for the next cron tick.

---

## Image Transcode Proxy

```
GET /falabella/images/transcoded?src=<url>
Permission: public (no token required)
```

See [05-IMAGE-TRANSCODING.md](05-IMAGE-TRANSCODING.md) for full details.

---

## Brands Reference

```
GET /falabella/brands
Permission: product:view
```

Returns the list of brands approved by Falabella's Seller Center. Use this to validate the
`Brand` attribute before submitting a sync.

---

## Error Reference

| HTTP Status | Condition |
|-------------|-----------|
| `400` | Invalid request body or disallowed transcode URL prefix |
| `401` | Missing or invalid bearer token |
| `403` | Token lacks required scope |
| `404` | Product or feed not found |
| `503` | Falabella circuit breaker is open (`ErrUnavailable`) |
| `500` | Internal server error |
