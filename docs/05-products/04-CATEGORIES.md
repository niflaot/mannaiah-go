# Products — Categories

The category system provides a hierarchical tree for organising products. Categories support
both **manual product pinning** and **rule-based membership** via tag, price, and category
cross-reference filters.

---

## Category Domain Type

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | UUID primary key |
| `Slug` | `string` | URL-safe identifier, unique |
| `Name` | `string` | Display name |
| `Description` | `string` | Optional description |
| `ParentID` | `*string` | Parent category ID (`nil` = root category) |
| `IncludeChildren` | `bool` | When `true`, all descendant categories contribute their products to this category's product list |
| `Filter` | `Filter` | Rule-based product inclusion criteria |
| `ProductIDs` | `[]string` | Manually pinned product IDs |
| `CreatedAt` / `UpdatedAt` | `time.Time` | Timestamps |

### Filter

| Field | Type | Description |
|-------|------|-------------|
| `Tags` | `[]string` | Products must carry **at least one** of these tags |
| `PriceRange` | `*PriceRange` | Products whose `price` falls within `Min`–`Max` (either bound is optional) |
| `CategoryRefs` | `[]string` | Pull in the pinned `ProductIDs` from other category IDs |

`PriceRange` — `{ Min *float64; Max *float64 }` (nil pointer = no bound on that side).

---

## Database Tables

| Table | Content |
|-------|---------|
| `categories` | `id`, `slug` (unique), `name`, `description`, `parent_id`, `include_children`, timestamps, `deleted_at` |
| `category_filter_tags` | `category_id`, `position`, `tag` |
| `category_filter_price_ranges` | `category_id` (unique), `min_price`, `max_price` |
| `category_filter_category_refs` | `category_id`, `ref_category_id` |
| `category_products` | `category_id`, `product_id`, `position` |

---

## Product Membership Resolution Algorithm

`GET /categories/:id/products` resolves product membership client-side (in Go) by running three
independent database query chains and union-ing the resulting product ID sets in memory. The final
list is then paginated.

### Step 1 — Collect target category scope

```
Load category by ID
If IncludeChildren = true:
    BFS all descendants via:
        SELECT id FROM categories WHERE parent_id = ? AND deleted_at IS NULL
    → category scope = [root] ∪ [all descendants]
Else:
    → category scope = [root]
```

### Step 2 — Collect product IDs per category

For each category in the scope, run `collectCategoryScopedProductIDs`:

#### Chain A — Pinned products

```
productIDSet ← union(cat.ProductIDs for each cat in scope)
```

#### Chain B — Tag filter (when `Filter.Tags` is non-empty)

```sql
-- 1. Find all products that carry at least one of the filter tags
SELECT product_tags.product_id
FROM product_tags
JOIN tags ON tags.id = product_tags.tag_id AND tags.deleted_at IS NULL
WHERE tags.name IN (<filter tags>)
GROUP BY product_tags.product_id

-- 2. Further restrict to price range (if set)
SELECT id FROM products
WHERE deleted_at IS NULL
  AND id IN (<results from step 1>)
  [AND price >= <min>]   -- only if PriceRange.Min is set
  [AND price <= <max>]   -- only if PriceRange.Max is set
```

#### Chain C — Price-only filter (when `Filter.Tags` is empty, `PriceRange` is set)

```sql
SELECT id FROM products
WHERE deleted_at IS NULL
  [AND price >= <min>]
  [AND price <= <max>]
```

#### Chain D — CategoryRefs

For each referenced category ID in `Filter.CategoryRefs`:
```sql
SELECT product_id FROM category_products WHERE category_id = <ref_id>
```

### Step 3 — Union and paginate

```
allIDs = deduplicated union of sets from Chains A, B, C, D  (in-memory Go map)
total  = len(allIDs)
page   = allIDs[ offset : offset+pageSize ]
→ load full Product objects for paged IDs via GetByIDs
```

> **Important:** The full product ID universe is materialised in memory before pagination.
> For very large catalogues, keep filter precision high to avoid unbounded ID sets.

---

## Resolution Examples

### Example 1 — Simple tag filter

```json
{
  "slug": "running",
  "filter": { "tags": ["running", "sport"] },
  "productIds": []
}
```

| Step | Result |
|------|--------|
| Chain A | (empty — no pinned IDs) |
| Chain B | All products tagged `"running"` OR `"sport"` |
| Chain C | (skipped — tags present) |
| Final | Union of B |

---

### Example 2 — Price range + tags

```json
{
  "slug": "premium-sport",
  "filter": {
    "tags": ["sport"],
    "priceRange": { "min": 100000, "max": 500000 }
  }
}
```

| Step | Result |
|------|--------|
| Chain A | (empty) |
| Chain B | Products tagged `"sport"` AND priced between 100 000–500 000 |
| Final | Chain B results |

---

### Example 3 — Pinned + cross-reference

```json
{
  "slug": "featured",
  "productIds": ["prod-001", "prod-002"],
  "filter": {
    "categoryRefs": ["cat-running-uuid", "cat-gym-uuid"]
  }
}
```

| Step | Result |
|------|--------|
| Chain A | `prod-001`, `prod-002` |
| Chain D | All pinned products from `cat-running-uuid` + `cat-gym-uuid` |
| Final | Union → `prod-001`, `prod-002` + all from referenced cats |

---

### Example 4 — Parent category with `IncludeChildren = true`

```
Sports (root, includeChildren=true)
├── Running        → productIds: [A, B], filter: tags: ["running"]
└── Gym Equipment  → productIds: [C],   filter: priceRange: {min:50000}
```

```
GET /categories/sports-uuid/products

Scope = [Sports, Running, Gym Equipment]

Running   → Chain A: [A, B] + Chain B: [all "running" tagged]
GymEquip  → Chain A: [C]    + Chain C: [all products >= 50 000]
Sports    → (no own filter)

Final = union of all above, deduplicated, paginated
```

---

## Port Layer

### `port/category.Repository`

```go
EnsureSchema(ctx context.Context) error
Create(ctx context.Context, c *Category) error
GetByID(ctx context.Context, id string) (*Category, error)
GetBySlug(ctx context.Context, slug string) (*Category, error)
Tree(ctx context.Context) ([]*Category, error)             // root-level only
ListChildren(ctx context.Context, parentID string) ([]*Category, error)
Update(ctx context.Context, c *Category) error
Delete(ctx context.Context, id string) error               // soft-delete; fails if category has children
ListProducts(ctx context.Context, q ListProductsQuery) (*ListProductsResult, error)
```

`ListProductsQuery` — `{ CategoryID string; Page, PageSize int }`  
`ListProductsResult` — `{ Products []*Product; Total int64 }`

---

## HTTP Endpoints

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| `POST` | `/categories` | `product:manage` | Create a category |
| `GET` | `/categories` | `product:view` | Get full category tree |
| `GET` | `/categories/:id` | `product:view` | Get a single category |
| `GET` | `/categories/:id/children` | `product:view` | Get direct children |
| `GET` | `/categories/:id/products` | `product:view` | Resolve product membership (paginated) |
| `PATCH` | `/categories/:id` | `product:manage` | Update a category |
| `DELETE` | `/categories/:id` | `product:manage` | Soft-delete (fails if has children) |
