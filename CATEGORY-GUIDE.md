# Category Integration Guide

This guide covers the product taxonomy (categories) feature introduced in v2.2.0, intended for frontend and API consumer integration.

---

## Category Data Model

```json
{
  "id": "string",
  "slug": "string (unique, URL-friendly)",
  "name": "string",
  "description": "string (optional)",
  "parentId": "string | null",
  "includeChildren": "boolean",
  "filter": {
    "tags": ["string"],
    "priceRange": {
      "min": "number | null",
      "max": "number | null"
    },
    "categoryRefs": ["string"]
  },
  "productIds": ["string"],
  "createdAt": "ISO 8601 timestamp",
  "updatedAt": "ISO 8601 timestamp"
}
```

---

## Filter Types

Categories support three filter mechanisms that combine to define which products appear in a category:

### 1. Tag Filter (`filter.tags`)
Products must have **at least one** of these tags to be included.

```json
{ "filterTags": ["tech", "sale"] }
```

### 2. Price Range Filter (`filter.priceRange`)
Only products with a `price` value within the specified range are included. Combine with tag filter for tag+price matching.

```json
{ "filterMinPrice": 10.0, "filterMaxPrice": 200.0 }
```

### 3. Category Reference Filter (`filter.categoryRefs`)
Products pinned to the referenced categories are also included in this category's product set.

```json
{ "filterCategoryRefs": ["cat-id-1", "cat-id-2"] }
```

### 4. Manually Pinned Products (`productIds`)
Products explicitly pinned to this category, regardless of tags or price.

```json
{ "productIds": ["product-id-1", "product-id-2"] }
```

### Filter Combination Rules
- The product set is the **union** of all matching filter results plus pinned products.
- A category with **no filters and no pinned products** returns an **empty product list**.
- If `includeChildren` is `true`, products from descendant categories are also included in the union.

---

## Endpoints

All category endpoints are protected by JWT bearer authentication.

### Base URL
```
/categories
```

---

### POST /categories
Create a new category.

**Permission:** `product:manage`

**Request Body:**
```json
{
  "slug": "electronics",
  "name": "Electronics",
  "description": "Electronic products",
  "parentId": null,
  "includeChildren": false,
  "filterTags": ["tech"],
  "filterMinPrice": 10.0,
  "filterMaxPrice": 1000.0,
  "filterCategoryRefs": [],
  "productIds": []
}
```

**Response:** `201 Created`
```json
{
  "id": "...",
  "slug": "electronics",
  "name": "Electronics",
  ...
}
```

**curl example:**
```bash
curl -X POST https://api.example.com/categories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"slug":"electronics","name":"Electronics"}'
```

---

### GET /categories
Get all root-level categories (the category tree).

**Permission:** `product:view`

**Response:** `200 OK`
```json
{
  "data": [
    { "id": "...", "slug": "electronics", "name": "Electronics", ... }
  ]
}
```

**curl example:**
```bash
curl https://api.example.com/categories \
  -H "Authorization: Bearer $TOKEN"
```

---

### GET /categories/:id
Get a specific category by ID.

**Permission:** `product:view`

**Response:** `200 OK` — category object, or `404` if not found.

**curl example:**
```bash
curl https://api.example.com/categories/CAT_ID \
  -H "Authorization: Bearer $TOKEN"
```

---

### GET /categories/:id/children
Get the direct children of a category.

**Permission:** `product:view`

**Response:** `200 OK`
```json
{
  "data": [
    { "id": "...", "slug": "laptops", "name": "Laptops", "parentId": "...", ... }
  ]
}
```

**curl example:**
```bash
curl https://api.example.com/categories/CAT_ID/children \
  -H "Authorization: Bearer $TOKEN"
```

---

### GET /categories/:id/products
Get paginated products belonging to a category.

**Permission:** `product:view`

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page`     | int  | `1`     | 1-based page number |
| `pageSize` | int  | `20`    | Items per page |

**Response:** `200 OK`
```json
{
  "data": [
    { "_id": "...", "sku": "SKU-1", "price": 49.99, "tags": ["tech"], ... }
  ],
  "total": 42,
  "page": 1,
  "pageSize": 20
}
```

**curl example:**
```bash
curl "https://api.example.com/categories/CAT_ID/products?page=1&pageSize=20" \
  -H "Authorization: Bearer $TOKEN"
```

---

### PATCH /categories/:id
Update a category. All fields are optional — only provided fields are updated.

**Permission:** `product:manage`

**Request Body (all optional):**
```json
{
  "slug": "new-slug",
  "name": "New Name",
  "description": "Updated description",
  "parentId": "parent-cat-id",
  "includeChildren": true,
  "filterTags": ["tech", "gadget"],
  "filterMinPrice": 5.0,
  "filterMaxPrice": 500.0,
  "filterCategoryRefs": ["cat-id-1"],
  "productIds": ["prod-id-1", "prod-id-2"]
}
```

**Response:** `200 OK` — updated category object.

**curl example:**
```bash
curl -X PATCH https://api.example.com/categories/CAT_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated Name"}'
```

---

### DELETE /categories/:id
Delete a category. Categories with children cannot be deleted.

**Permission:** `product:manage`

**Response:** `200 OK`
```json
{ "status": "deleted" }
```

**Error Responses:**
- `404` — Category not found.
- `409` — Category has children (`category_has_children`).

**curl example:**
```bash
curl -X DELETE https://api.example.com/categories/CAT_ID \
  -H "Authorization: Bearer $TOKEN"
```

---

## Permissions

| Scope           | Access |
|-----------------|--------|
| `product:view`  | Read endpoints (GET) |
| `product:manage`| Write endpoints (POST, PATCH, DELETE) |

---

## Hierarchy Rules

- A category with `parentId: null` is a **root** category.
- Nested categories can be created by setting `parentId` to an existing category ID.
- A category **cannot** be its own parent (circular parent validation).
- Categories with children **cannot be deleted** — delete all children first.
- Setting `includeChildren: true` makes a category's product listing include products from all descendant categories (recursive).

---

## Products: Tags and Price

Products support two new fields used by category filters:

```json
{
  "_id": "...",
  "sku": "SKU-1",
  "price": 49.99,
  "tags": ["tech", "sale"],
  ...
}
```

- `price` — optional float64 value used by price range filters.
- `tags` — string array used by tag filters.

These fields can be set on create or update:
```bash
curl -X POST https://api.example.com/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"sku":"SKU-1","price":49.99,"tags":["tech","sale"]}'
```

---

## Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | `invalid_payload` | Malformed JSON body |
| 400 | `invalid_category_id` | Empty or missing category ID |
| 400 | `invalid_category` | Slug or name is missing |
| 400 | `circular_category_parent` | Category references itself as parent |
| 401 | `unauthorized` | Missing or invalid auth token |
| 403 | `forbidden` | Insufficient permissions |
| 404 | `category_not_found` | Category does not exist |
| 409 | `category_slug_conflict` | Slug already in use |
| 409 | `category_has_children` | Cannot delete a parent with children |
| 500 | `internal_server_error` | Unexpected server error |
