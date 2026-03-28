# Products — HTTP API

All product endpoints require a valid bearer token. Permissions follow the `product:*` scope
family.

---

## Products

### Create a Product

```
POST /products
Permission: product:edit
```

```json
{
  "sku": "SHIRT-001",
  "price": 89900,
  "tags": ["ropa", "sport"],
  "gallery": [
    {
      "assetId": "asset-uuid",
      "position": 1,
      "isMain": true,
      "includedRealms": [],
      "variationIds": []
    }
  ],
  "datasheets": [
    {
      "realm": "default",
      "name": "Camiseta Deportiva",
      "description": "Camiseta de alto rendimiento.",
      "attributes": {
        "brand": "Flock Sport",
        "material": "Polyester 100%"
      }
    },
    {
      "realm": "falabella",
      "name": "Camiseta Deportiva Flock",
      "description": "Camiseta de alto rendimiento para actividades físicas.",
      "attributes": {
        "Brand": "Flock Sport",
        "Model": "FS-001",
        "TaxClass": "IVA19",
        "PriceFalabella": 89900,
        "Stock": 50,
        "Status": "active"
      }
    }
  ],
  "variations": ["var-red", "var-blue", "var-m", "var-l"],
  "variants": [
    { "variationIDs": ["var-red", "var-m"], "sku": "SHIRT-001-RED-M" },
    { "variationIDs": ["var-red", "var-l"], "sku": "SHIRT-001-RED-L" }
  ]
}
```

**Response** — `201 Created` with the full `Product` object.

---

### List Products

```
GET /products
Permission: product:view
```

**Query parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `tags` | `string` | Comma-separated tag names to filter by |
| `page` | `int` | Page number (default `1`) |
| `pageSize` | `int` | Records per page (default `20`) |

**Response** — `200 OK`

```json
{
  "data": [ /* Product[] */ ],
  "total": 340,
  "page": 1,
  "pageSize": 20
}
```

---

### Get by ID

```
GET /products/:id
Permission: product:view
```

---

### Get by SKU

```
GET /products/sku/:sku
Permission: product:view
```

---

### Update a Product

```
PATCH /products/:id
Permission: product:edit
```

Partial updates are supported. Datasheets are merged by realm (see
[02-DOMAIN.md](02-DOMAIN.md#mergedatasheets)).

---

### Delete a Product

```
DELETE /products/:id
Permission: product:manage
```

**Response** — `200 OK`.

---

## Error Reference

| HTTP Status | Condition |
|-------------|-----------|
| `400` | Validation failure (missing SKU, invalid gallery asset IDs) |
| `401` | Missing or invalid bearer token |
| `403` | Token lacks required scope |
| `404` | Product / variation / category not found |
| `409` | Duplicate SKU |
| `500` | Internal server error |
