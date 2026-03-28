# Product Affinity & Recommendations

The product affinity system builds per-contact profiles based on purchase history and uses them to power personalized product recommendations in campaigns.

## Affinity Computation

### Time-Decay Formula

All affinity scores use exponential time decay to weight recent purchases more heavily:

$$\text{affinity\_score} = \sum_{\text{purchases}} \text{item\_value} \times e^{-0.01 \times \Delta\text{days}}$$

where $\Delta\text{days}$ is the number of days between the purchase and now.

The decay constant **0.01** produces a half-life of approximately **69 days**:

| Days Ago | Weight Factor | Interpretation |
|---|---|---|
| 0 | 1.000 | Full weight |
| 7 | 0.932 | 93% weight |
| 30 | 0.741 | 74% weight |
| 69 | 0.501 | ~50% weight (half-life) |
| 180 | 0.165 | 17% weight |
| 365 | 0.026 | 3% weight |

### Three Affinity Dimensions

| Dimension | Data Join | Grouped By | ClickHouse Table |
|---|---|---|---|
| **Tag Affinity** | `order_items_fact` × `product_taxonomy` | `(contact_id, tag)` | `tag_affinity_mv` |
| **Category Affinity** | `order_items_fact` × `product_taxonomy` | `(contact_id, category_id)` | `category_affinity_mv` |
| **Variation Affinity** | `order_items_fact` × `product_variation_taxonomy` | `(contact_id, variation_name, variation_value)` | `variation_affinity_mv` |

Each row stores: `affinity_score`, `total_spent` (raw monetary sum), and `purchase_count` (line item count).

### Affinity Profile Example

For a contact who bought:
- A "travel backpack" ($50) 10 days ago (tags: `backpack`, `travel`)
- A "laptop sleeve" ($20) 45 days ago (tags: `laptop-sleeve`, `accessories`)
- A "travel water bottle" ($15) 5 days ago (tags: `water-bottle`, `travel`)

Tag affinity scores:
- `travel`: $50 × e^{-0.1} + 15 × e^{-0.05}$ = $45.24 + 14.27$ = **59.51**
- `backpack`: $50 × e^{-0.1}$ = **45.24**
- `water-bottle`: $15 × e^{-0.05}$ = **14.27**
- `laptop-sleeve`: $20 × e^{-0.45}$ = **12.75**
- `accessories`: $20 × e^{-0.45}$ = **12.75**

The contact has strongest affinity for `travel` and `backpack` products.

## Tag Correlations

Cross-sell relationships are stored in MySQL (`tag_correlations` table):

```
source_tag    │ target_tag     │ probability
──────────────┼────────────────┼──────────────
backpack      │ laptop-sleeve  │ 85
backpack      │ water-bottle   │ 72
travel        │ sunglasses     │ 65
laptop-sleeve │ cable-organizer│ 55
```

Correlations enable the recommendation engine to **expand** a contact's known affinities into related product categories. If a contact loves "backpack" products, the engine will also consider "laptop-sleeve" and "water-bottle" as recommendation candidates even if the contact has never purchased them.

### Correlation Graph

```
          ┌──85%──▶ laptop-sleeve ──55%──▶ cable-organizer
backpack ─┤
          └──72%──▶ water-bottle

travel ────65%──▶ sunglasses
```

Correlations are directional (A→B doesn't imply B→A) and deduplicated by target tag, ordered by probability descending.

## Recommendation Engine

The recommendation endpoint at `GET /analytics/recommendations/contacts/:contactId` produces a personalized product list.

### Resolution Pipeline

```
1. Parse query parameters → RecommendationQuery
2. Normalize query (deduplicate, merge baseTag→baseTags, clamp limit 1–10)
3. Load pinned products by ID from catalog
4. Build exclusion set:
     explicit excludeIds
   + pinned product IDs (prevent duplicates)
   + purchased product IDs (if excludePurchased=true)
5. If affinity enabled:
     a. Load contact's top 20 tag affinities from ClickHouse
     b. Expand via tag_correlations → gather related tags
     c. Merge expanded tags with baseTags
6. Fetch product candidates from catalog:
     - baseTags union ("any") or intersection ("all")
     - Apply category include/exclude
     - Apply tag include/exclude
     - Apply price range
     - Apply variation filtering
7. Score candidates:
     productAffinityScore = Σ(affinityScore for each tag the product carries)
8. Sort by affinity score descending
9. Combine: pinned products first + dynamic results up to limit
10. Resolve display data per product:
      - Realm-specific price, name
      - Image URL (prefer specified variation, then any realm-visible gallery)
      - Product URL (variation-scoped preferred)
```

### Example

Contact has high affinity for `backpack` (score: 45) and `travel` (score: 59).

Query: `baseTags=backpack&affinity=true&excludePurchased=true&limit=4`

1. Expand `backpack` via correlations → also consider `laptop-sleeve`, `water-bottle`.
2. Fetch products tagged any of: `backpack`, `laptop-sleeve`, `water-bottle`.
3. Score each product by overlapping tag affinities.
4. A product tagged `[backpack, travel]` scores 45 + 59 = 104.
5. A product tagged `[laptop-sleeve]` scores 12.75 (from contact's direct affinity).
6. Return top 4 by score.

### Variation-Preferred Images

When `preferVariationIds` is specified, the engine biases gallery image selection toward those variations:

1. Look for gallery entries matching a preferred variation ID.
2. If found, use that image URL.
3. If not found, fall back to any realm-visible gallery image.

This ensures that if a contact has variation affinity for "Color: Blue", the recommendation image shows the blue variant.

## Relative Affinity in Segments

When segments use affinity filter types (`tag_affinity`, `category_affinity`, `variation_affinity`), the thresholds are **relative** to each contact's own maximum:

```sql
-- "contacts with ≥60% relative affinity for tag 'backpack'"
maxIf(affinity_score, tag IN ('backpack'))
  * 100.0
  / nullIf(max(affinity_score), 0)
  >= 60
```

This means a contact with scores `{backpack: 45, travel: 59}` has:
- Relative backpack affinity: 45/59 × 100 = **76.3%** → passes 60% threshold
- If threshold were 80%, this contact would NOT pass

This relative approach ensures that active and inactive buyers are compared fairly — a low-volume buyer with most purchases in "backpack" can still qualify.

## API Reference

### Affinity Queries

| Method | Path | Description |
|---|---|---|
| `GET` | `/analytics/affinity/contacts/:contactId` | Full profile (tags + categories + variations) |
| `GET` | `/analytics/affinity/contacts/:contactId/tags` | Tag affinities |
| `GET` | `/analytics/affinity/contacts/:contactId/categories` | Category affinities |
| `GET` | `/analytics/affinity/contacts/:contactId/variations` | Variation affinities |
| `POST` | `/analytics/affinity/refresh` | Refresh all 3 materialized views |

### Recommendations

| Method | Path | Permission | Description |
|---|---|---|---|
| `GET` | `/analytics/recommendations/contacts/:contactId` | `marketing:manage` | Personalized product recommendations |

#### Path parameters

| Parameter | Description |
|---|---|
| `contactId` | Contact identifier. Used to load affinity scores and (optionally) the purchased-product history. |

#### Query parameters

| Parameter | Type | Default | Description |
|---|---|---|---|
| `baseTags` | `string` (CSV) | — | **Required** (unless `pinnedIds` is set). Comma-separated product tags that define the candidate pool. |
| `baseTag` | `string` | — | Deprecated single-tag alias. Merged into `baseTags` as the first entry. |
| `baseTagMode` | `"any"` \| `"all"` | `"any"` | `"any"` — candidates carry at least one base tag (union). `"all"` — candidates carry every base tag (intersection). |
| `affinity` | `bool` | `false` | When `true`, loads the contact's top-20 tag affinities from ClickHouse, expands them via `tag_correlations`, and adds the related tags to the candidate query. Candidates are then ranked by their accumulated affinity score. |
| `minScore` | `float` [0–100] | `0` | Minimum **relative** affinity score percentile required for a tag to be considered during affinity expansion. `0` keeps all tags. |
| `excludePurchased` | `bool` | `false` | When `true`, looks up all product IDs the contact has purchased (up to 2000) and excludes them from results. |
| `limit` | `int` [1–10] | `3` | Maximum number of products returned. Values outside [1, 10] are clamped. |
| `realm` | `string` | `"default"` | Selects which product datasheet to use for name, price, and gallery image resolution. Use the realm slug (e.g. `"falabella"`, `"default"`). Products with no datasheet for the requested realm are silently dropped. |
| `seed` | `int64` | `0` | Non-zero value enables deterministic pseudo-random shuffling of equally-scored candidates. Produces repeatable but varied orderings — typical usage is `hash(campaignId + contactId)` for frontend/backend parity. |
| `pinnedIds` | `string` (CSV) | — | Comma-separated product IDs always returned first, bypassing the base-tag filter and affinity ranking. Supports scoped syntax `<productId>\|<variationId>` to force a specific variation for URL/image resolution. |
| `excludeIds` | `string` (CSV) | — | Comma-separated product IDs never returned. Supports scoped syntax `<productId>\|<variationId>` to block a specific variation from URL/image candidate selection without removing the whole product. |
| `categoryId` | `string` | — | Restrict candidates to one category identifier. |
| `categoryIds` | `string` (CSV) | — | Restrict candidates to multiple category identifiers (merged with `categoryId`). |
| `excludeCategoryIds` | `string` (CSV) | — | Remove candidates that belong to any of these categories. |
| `includeTags` | `string` (CSV) | — | Additional OR tag filter: candidates must carry at least one of these tags (applied after base-tag lookup). |
| `excludeTags` | `string` (CSV) | — | Remove candidates carrying at least one of these tags. |
| `minPrice` | `float` | — | Minimum price filter. Applied against the realm datasheet price. Products without a realm price are excluded. |
| `maxPrice` | `float` | — | Maximum price filter. When both `minPrice` and `maxPrice` are set and `minPrice` > `maxPrice`, they are silently swapped. |
| `filterVariationIds` | `string` (CSV) | — | Restrict candidates to products that carry at least one of these variation IDs (e.g. only show products available in black). |
| `preferVariationIds` | `string` (CSV) | — | Bias gallery image selection toward images linked to these variation IDs (e.g. prefer the blue-variant photo). Falls back to the first realm-visible image when no match is found. |

#### Response

Returns a JSON array of `RecommendedProduct` objects (empty array `[]` when no candidates pass filters):

```json
[
  {
    "id": "abc123",
    "name": "Mochila de Viaje 40L",
    "price": 89900,
    "imageUrl": "https://cdn.example.com/products/abc123.jpg",
    "url": "https://store.example.com/products/mochila-40l?variant=black"
  }
]
```

| Field | Type | Description |
|---|---|---|
| `id` | `string` | Product identifier. |
| `name` | `string` | Display name from the matching realm datasheet. Falls back to the first available datasheet name if no realm match. |
| `price` | `float64` | Price from the realm datasheet's price attribute. |
| `imageUrl` | `string` | Public URL of the best matching gallery image. Prefers images linked to `preferVariationIds`; falls back to the first realm-visible image. Empty string when no image is available. |
| `url` | `string` | Realm-scoped product detail URL, variation-scoped when possible. Omitted when no URL is resolvable. |

#### Example requests

Minimum viable — top 3 backpack products by affinity for contact `c_01`:
```
GET /analytics/recommendations/contacts/c_01?baseTags=backpack&affinity=true
```

Campaign widget — 5 products, exclude purchased, seed-stable, blue-variants preferred, falabella realm:
```
GET /analytics/recommendations/contacts/c_01
  ?baseTags=backpack,travel
  &affinity=true
  &excludePurchased=true
  &limit=5
  &seed=87234
  &realm=falabella
  &preferVariationIds=var_blue,var_navy
```

Pinned product first (always show `prod_featured`), then affinity-ranked backpacks:
```
GET /analytics/recommendations/contacts/c_01
  ?pinnedIds=prod_featured
  &baseTags=backpack
  &affinity=true
  &limit=4
```

## Materialized View Refresh

All affinity views require manual refresh via `POST /analytics/affinity/refresh`, which truncates and repopulates all three views:
- `tag_affinity_mv` from `order_items_fact × product_taxonomy`
- `category_affinity_mv` from `order_items_fact × product_taxonomy`
- `variation_affinity_mv` from `order_items_fact × product_variation_taxonomy`
