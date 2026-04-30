# Search Query DSL Reference

Complete reference for the search query parameter DSL supported by all `/<resource>/search` endpoints.

## Parameters

### `term` — Free Text Search

**Type:** `string`  
**Required:** No  
**Example:** `?term=john`

Performs a case-insensitive `LIKE %term%` against all text fields configured in the resource Descriptor. Multiple text fields are OR-joined — a row matches if any text field contains the term.

Special LIKE characters (`%`, `_`) in the term are automatically escaped.

---

### `filter[field]` — Exact Match Filter

**Type:** `string`  
**Required:** No  
**Example:** `?filter[status]=ACTIVE`

Applies a `WHERE field = value` condition. The field must be declared in the resource's `FilterableFields` map with `OpEq`.

---

### `filter[field.op]` — Operator Filter

**Type:** `string`  
**Required:** No

| Operator | SQL | Example | Notes |
|----------|-----|---------|-------|
| `eq` | `= ?` | `filter[status.eq]=ACTIVE` | Same as `filter[status]=ACTIVE` |
| `like` | `LIKE ?` | `filter[name.like]=john` | Wraps in `%value%` |
| `in` | `IN (?)` | `filter[status.in]=ACTIVE,PAUSED` | Comma-separated values |
| `gt` | `> ?` | `filter[price.gt]=100` | Greater than |
| `gte` | `>= ?` | `filter[created_at.gte]=2024-01-01` | Greater than or equal |
| `lt` | `< ?` | `filter[price.lt]=500` | Less than |
| `lte` | `<= ?` | `filter[created_at.lte]=2024-12-31` | Less than or equal |
| `between` | `BETWEEN ? AND ?` | `filter[price.between]=10,50` | Two comma-separated values |

**Validation:** Only operators declared in the resource's `FilterableFields` map are accepted. Undeclared operators are silently skipped.

---

### `sort` — Sort Order

**Type:** `string`  
**Required:** No  
**Example:** `?sort=created_at:desc,name:asc`

Comma-separated list of `field:direction` pairs.

- **direction:** `asc` or `desc` (case-insensitive)
- **Validation:** Only fields listed in `SortableFields` are accepted
- **Default:** If omitted, the resource's `DefaultSort` is used

---

### `page` — Page Number

**Type:** `integer`  
**Required:** No  
**Default:** `1`

1-based page index. Values < 1 are normalized to 1.

---

### `pageSize` — Page Size

**Type:** `integer`  
**Required:** No  
**Default:** `20`  
**Max:** `100`

Number of items per page. Clamped to [1, 100].

---

## Per-Resource Filter Reference

### Contacts (`/search/contacts`)

| Field | Operators |
|-------|-----------|
| `first_name` | eq, like |
| `last_name` | eq, like |
| `email` | eq, like |
| `document_number` | eq |
| `phone` | eq |
| `created_at` | gte, lte, between |

### Orders (`/search/orders`)

| Field | Operators |
|-------|-----------|
| `realm` | eq |
| `contact_id` | eq |
| `payment_method` | eq |
| `created_at` | gte, lte, between |

### Products (`/search/products`)

| Field | Operators |
|-------|-----------|
| `sku` | eq |
| `price` | gte, lte, gt, lt, between |
| `created_at` | gte, lte, between |

### Categories (`/search/categories`)

| Field | Operators |
|-------|-----------|
| `parent_id` | eq |
| `created_at` | gte, lte, between |

### Variations (`/search/variations`)

| Field | Operators |
|-------|-----------|
| `definition` | eq, in |
| `created_at` | gte, lte, between |

### Tags (`/search/tags`)

| Field | Operators |
|-------|-----------|
| `created_at` | gte, lte, between |

### Shipping (`/search/shipping`)

| Field | Operators |
|-------|-----------|
| `carrier_id` | eq |
| `status` | eq, in |
| `order_id` | eq |
| `dispatch_batch_id` | eq |
| `shipment_mode` | eq |
| `declared_value` | gte, lte, gt, lt, between |
| `created_at` | gte, lte, between |

---

## Complex Query Examples

### Search contacts by name with date range

```
GET /search/contacts?term=maria&filter[created_at.gte]=2024-01-01&filter[created_at.lte]=2024-06-30&sort=last_name:asc
```

### Search shipped orders in specific price range

```
GET /search/shipping?filter[status.in]=DISPATCHED,DELIVERED&filter[declared_value.between]=100,500&sort=created_at:desc&page=2&pageSize=50
```

### Spotlight search across contacts and orders

```
GET /search?term=ORD-2024&types=contact,order&limit=5
```
