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

| Method | Path | Description |
|---|---|---|
| `GET` | `/analytics/recommendations/contacts/:contactId` | Personalized product recommendations |

Query parameters: `baseTags`, `baseTagMode`, `categoryId`, `categoryIds`, `excludeCategoryIds`, `includeTags`, `excludeTags`, `minPrice`, `maxPrice`, `excludePurchased`, `realm`, `limit`, `affinity`, `minScore`, `pinnedIds`, `excludeIds`, `filterVariationIds`, `preferVariationIds`.

## Materialized View Refresh

All affinity views require manual refresh via `POST /analytics/affinity/refresh`, which truncates and repopulates all three views:
- `tag_affinity_mv` from `order_items_fact × product_taxonomy`
- `category_affinity_mv` from `order_items_fact × product_taxonomy`
- `variation_affinity_mv` from `order_items_fact × product_variation_taxonomy`
