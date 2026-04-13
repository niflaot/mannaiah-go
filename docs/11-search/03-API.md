# Search API Endpoints

Detailed endpoint documentation for all search routes.

## Resource Search Endpoints

All resource search endpoints follow identical request/response conventions.

### `GET /<resource>/search`

**Common Parameters:** See [02-DSL.md](02-DSL.md) for complete DSL reference.

**Response:** `200 OK`

```json
{
  "data": [],
  "total": 0,
  "page": 1,
  "pageSize": 20,
  "totalPages": 0
}
```

### Available Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/search/contacts` | Search contacts by name, email, phone, document |
| `GET` | `/search/orders` | Search orders by identifier, payment method |
| `GET` | `/search/products` | Search products by SKU |
| `GET` | `/search/categories` | Search categories by name, slug, description |
| `GET` | `/search/variations` | Search product variations by name, value |
| `GET` | `/search/tags` | Search product tags by name |
| `GET` | `/search/shipping` | Search shipping marks by tracking, order |
| `GET` | `/search/campaigns` | Search campaigns by name, slug, subject |
| `GET` | `/search/coupons` | Search coupons by code, origin, assignment, or linked contact name |
| `GET` | `/search/segments` | Search segments by name, slug |

### Coupon-Specific Filter

Coupons support the standard `term`, `page`, `pageSize`, and `sort` parameters plus one exact-match filter:

| Query | Description |
|-------|-------------|
| `filter[discountType]` | Match `fixed` or `percentage` coupons |

---

## Spotlight Endpoint

### `GET /search`

Cross-resource concurrent search with relevance scoring.

**Parameters:**

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `term` | string | Yes | — | Search term |
| `types` | string | No | all | Comma-separated provider types (e.g. `contact,order`) |
| `limit` | integer | No | 10 | Max results per provider (max: 50) |

**Response:** `200 OK`

```json
{
  "results": [
    {
      "type": "contact",
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "John Doe",
      "subtitle": "john@example.com",
      "matchedField": "first_name",
      "score": 1.0
    }
  ],
  "meta": {
    "term": "john",
    "tookMs": 32,
    "counts": {
      "contact": 5,
      "order": 2
    }
  }
}
```

**Provider Types:**

| Type | Source |
|------|--------|
| `contact` | Contacts |
| `order` | Orders |
| `product` | Products |
| `category` | Categories |
| `variation` | Variations |
| `tag` | Tags |
| `shipping_mark` | Shipping marks |
| `campaign` | Campaigns |
| `segment` | Segments |

**Behavior:**
- Each provider runs concurrently with a 2-second timeout
- Slow providers are excluded from results without error
- Results are merged and sorted by `score` descending
- The `counts` map shows how many hits each provider returned

---

## OpenAPI Specification

All search endpoints are documented in the OpenAPI spec available at:

```
GET /openapi.json
GET /docs
```

The search spec is merged into the aggregated document automatically at startup.
