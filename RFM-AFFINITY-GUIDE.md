# RFM & Affinity Engine — Technical Guide

## Overview

The RFM and Affinity engine provides two complementary analytical layers built on top of ClickHouse:

- **RFM scoring** — classifies contacts by purchase recency, frequency, and monetary value into 1–5 band scores.
- **Affinity profiling** — ranks which product tags, categories, and variations a contact has the strongest purchase affinity toward, using time-decayed spend weights.

Both engines read from fact tables populated by the analytics seed endpoint and maintain their own ClickHouse projection tables that can be refreshed on demand.

---

## Data Flow

```
MySQL (orders, products, contacts)
        │
        ▼
POST /analytics/seed
        │
        ├── orders_fact            (one row per order)
        ├── order_items_fact       (one row per line item, with resolved product_id)
        ├── product_taxonomy       (product → tag + category mappings)
        └── product_variation_taxonomy  (product → variation name/value mappings)
        │
        ▼
POST /analytics/rfm/refresh        POST /analytics/affinity/refresh
        │                                  │
        ▼                                  ▼
rfm_scores_mv               tag_affinity_mv
(ReplacingMergeTree)        category_affinity_mv
                            variation_affinity_mv
                            (SummingMergeTree)
```

The seed endpoint must be run first. The two refresh endpoints then re-derive their projections from the seeded fact tables.

---

## RFM Engine

### What RFM measures

| Dimension | Raw value | Meaning |
|-----------|-----------|---------|
| Recency (R) | Days since last order | Lower is better — recent buyers score higher |
| Frequency (F) | Count of distinct orders | Higher is better |
| Monetary (M) | Sum of order values | Higher is better |

Each dimension produces a band score from **1 (worst) to 5 (best)**. The three scores are summed into an `rfmTotal` (3–15).

### Band thresholds

Thresholds are stored in the `rfm_band_configs` MySQL table and cached in-process for **5 minutes** with a `sync.RWMutex` guard.

Default thresholds:

| Dimension | Band 5 | Band 4 | Band 3 | Band 2 |
|-----------|--------|--------|--------|--------|
| Recency (days, descending) | ≤ 7 | ≤ 30 | ≤ 90 | ≤ 180 |
| Frequency (ascending) | ≥ 10 | ≥ 6 | ≥ 3 | ≥ 2 |
| Monetary (ascending) | ≥ 1000 | ≥ 500 | ≥ 200 | ≥ 50 |

Recency uses **descending** scoring (lower days → higher band). Frequency and Monetary use **ascending** scoring (higher value → higher band).

Thresholds are configurable per dimension via `PUT /analytics/rfm/bands/{dimension}`.

### Scoring query

Scores are computed on the fly from `rfm_scores_mv` using ClickHouse `multiIf` expressions. The query binds 12 threshold arguments (4 per dimension) and returns the raw measurements plus the three computed band scores in a single round-trip.

```sql
SELECT contact_id, recency_days, frequency, monetary,
       multiIf(recency_days <= ?, 5, recency_days <= ?, 4, ..., 1),  -- R
       multiIf(frequency    >= ?, 5, frequency    >= ?, 4, ..., 1),  -- F
       multiIf(monetary     >= ?, 5, monetary     >= ?, 4, ..., 1)   -- M
FROM rfm_scores_mv FINAL
WHERE contact_id = ?
```

`rfm_scores_mv` itself aggregates raw order data:

```sql
-- populated by POST /analytics/rfm/refresh
SELECT contact_id,
       toUInt32(dateDiff('day', max(created_at), now64(3))) AS recency_days,
       toUInt32(countDistinct(order_id))                    AS frequency,
       sum(total_value)                                     AS monetary,
       now64(3)                                             AS updated_at
FROM orders_fact FINAL
GROUP BY contact_id
```

### RFM Groups

Groups are named cohorts stored in the `rfm_groups` MySQL table. Each group has a set of optional band-score range conditions (`rMin`, `rMax`, `fMin`, `fMax`, `mMin`, `mMax`). A contact belongs to a group when all non-nil conditions are satisfied by their computed R/F/M scores.

Example — "Champions" group (high on all three dimensions):

```json
{
  "name": "Champions",
  "slug": "champions",
  "conditions": {
    "rMin": 4,
    "fMin": 4,
    "mMin": 4
  }
}
```

Example — "At Risk" group (previously active, now lapsing):

```json
{
  "name": "At Risk",
  "slug": "at-risk",
  "conditions": {
    "rMin": 2, "rMax": 3,
    "fMin": 3,
    "mMin": 3
  }
}
```

### API reference

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/analytics/rfm/bands` | List all band threshold configurations |
| `PUT` | `/analytics/rfm/bands/{dimension}` | Update thresholds for `recency`, `frequency`, or `monetary` |
| `POST` | `/analytics/rfm/groups` | Create a named RFM group |
| `GET` | `/analytics/rfm/groups` | List all RFM groups |
| `GET` | `/analytics/rfm/groups/{id}` | Get one RFM group |
| `PUT` | `/analytics/rfm/groups/{id}` | Update one RFM group |
| `DELETE` | `/analytics/rfm/groups/{id}` | Delete one RFM group |
| `GET` | `/analytics/rfm/contacts/{contactId}/score` | Compute RFM score for one contact |
| `POST` | `/analytics/rfm/contacts/score-batch` | Compute RFM scores for up to 1000 contacts |
| `POST` | `/analytics/rfm/refresh` | Truncate and repopulate `rfm_scores_mv` |

All endpoints require the `marketing:manage` bearer scope.

---

## Affinity Engine

### What affinity measures

Affinity captures **which product attributes a contact tends to buy**, weighted by spend and decayed over time. Recent purchases contribute more than old ones.

The decay formula applied per line item:

```
affinity_score += item_value × exp(-0.01 × days_since_purchase)
```

A purchase made today contributes its full value. A purchase made 69 days ago contributes ~50% of its value. A purchase made 230 days ago contributes ~10%.

Three affinity dimensions are tracked:

| Dimension | Joins on | Groups by |
|-----------|----------|-----------|
| Tag | `product_taxonomy.tag` | contact + tag |
| Category | `product_taxonomy.category_id` | contact + category_id |
| Variation | `product_variation_taxonomy.variation_name/value` | contact + variation_name + variation_value |

### Data prerequisites

Before affinity scores can be computed, products must have taxonomy data:

- **Tags and categories** — set on products via `POST/PATCH /products` with `tags` and `categoryId` fields. The seed endpoint writes these into `product_taxonomy`.
- **Variations** — set on products via `POST/PATCH /variations` and assigned to product variants. The seed endpoint writes name/value pairs into `product_variation_taxonomy`.

Order line items must also have a resolved `product_id`. The order resolver matches items by SKU → variant SKU → alternate name. Items that fail to resolve are stored with an empty `product_id` and are excluded from affinity calculations.

### Affinity tables

All three tables use `SummingMergeTree`, which allows incremental inserts: rows with the same ORDER BY key have their numeric columns summed during background merges. The `FINAL` modifier forces merge-time deduplication at query time.

```sql
-- tag_affinity_mv
ENGINE = SummingMergeTree()
ORDER BY (contact_id, tag)

-- category_affinity_mv
ENGINE = SummingMergeTree()
ORDER BY (contact_id, category_id)

-- variation_affinity_mv
ENGINE = SummingMergeTree()
ORDER BY (contact_id, variation_name, variation_value)
```

### Refresh behavior

`POST /analytics/affinity/refresh` runs three sequential TRUNCATE + INSERT operations (one per dimension) inside individual transactions. Because the time-decay function uses `now64(3)` at insert time, scores must be recomputed from scratch on each refresh to stay accurate — old rows cannot be incrementally updated.

### Querying affinity

Results are sorted by `affinity_score DESC`. Two optional query parameters control output:

- `limit` (default `10`) — maximum rows returned per dimension.
- `minScore` (default `0`) — excludes entries below this threshold, useful for filtering out noise.

The full profile endpoint (`GET /analytics/affinity/contacts/{contactId}`) returns all three dimensions in a single response.

### API reference

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/analytics/affinity/contacts/{contactId}` | Full affinity profile (tags + categories + variations) |
| `GET` | `/analytics/affinity/contacts/{contactId}/tags` | Tag affinity scores only |
| `GET` | `/analytics/affinity/contacts/{contactId}/categories` | Category affinity scores only |
| `GET` | `/analytics/affinity/contacts/{contactId}/variations` | Variation affinity scores only |
| `POST` | `/analytics/affinity/refresh` | Truncate and repopulate all affinity tables |

All endpoints require the `marketing:manage` bearer scope.

---

## Operational Guide

### First-time setup

1. Seed fact tables from existing MySQL data:
   ```
   POST /analytics/seed
   ```
2. Populate RFM projection:
   ```
   POST /analytics/rfm/refresh
   ```
3. Populate affinity projections:
   ```
   POST /analytics/affinity/refresh
   ```

### Keeping data fresh

- Re-run `/analytics/seed` after orders or product taxonomy changes.
- Re-run the two refresh endpoints after seeding to bring projections up to date.
- Band threshold changes take effect immediately on the next score request (cache expires within 5 minutes, or is invalidated immediately on `PUT /analytics/rfm/bands/{dimension}`).

### Troubleshooting empty scores

| Symptom | Likely cause |
|---------|-------------|
| `ScoreContact` returns `null` | Contact has no rows in `orders_fact` or `rfm_scores_mv` was not refreshed |
| Affinity returns empty arrays | Products have no taxonomy data, or `order_items_fact.product_id` is empty for this contact's orders |
| All contacts show score 1 on all dimensions | Band thresholds are misconfigured — values are all zero or too high |
| Affinity scores seem stale | Scores use `now64(3)` at insert time; refresh must be re-run to apply current decay weights |

### Band tuning

Use `ComputeMonetaryPercentiles` (available internally via the ClickHouse store) to derive data-driven thresholds based on your actual monetary distribution. The percentile query returns `[p20, p40, p60, p80]` which map naturally to Band2Min through Band5Min for the monetary dimension.
