# Analytics Engine

The analytics engine provides real-time contact intelligence using a dual-database architecture: **MySQL** as the transactional source of truth and **ClickHouse** as the analytical compute backend. It powers RFM scoring, product affinity profiling, segment resolution, and product recommendations.

## Architecture

```
MySQL (transactional)                  ClickHouse (analytical)
┌──────────────────┐                  ┌──────────────────────┐
│ contacts         │───seed/events───▶│ contacts_snapshot     │
│ orders           │───seed/events───▶│ orders_fact           │
│ order_items      │───seed/events───▶│ order_items_fact      │
│ membership_stamps│───seed/events───▶│ membership_events     │
│ email_deliveries │───seed/events───▶│ campaign_events       │
│ product_tags     │───seed─────────▶│ product_taxonomy      │
│ product_variants │───seed─────────▶│ product_variation_tax. │
└──────────────────┘                  └──────────┬───────────┘
                                                 │ refresh
                                      ┌──────────▼───────────┐
                                      │ rfm_scores_mv        │
                                      │ tag_affinity_mv      │
                                      │ category_affinity_mv │
                                      │ variation_affinity_mv│
                                      └──────────────────────┘
```

Data flows from MySQL to ClickHouse through two mechanisms:

1. **Seed** — Full historical sync triggered via `POST /analytics/seed`. Reads all MySQL tables in batches of 1000 and upserts into ClickHouse.
2. **Integration events** — Incremental updates consumed from the messaging bus for real-time synchronization.

## ClickHouse Table Design

All fact tables use **ReplacingMergeTree** engines with `updated_at` as the version column. This means duplicate inserts for the same primary key are automatically deduplicated during merges. The `FINAL` keyword is used in queries to ensure correct reads before compaction completes.

| Table | Engine | ORDER BY | Purpose |
|---|---|---|---|
| `contacts_snapshot` | ReplacingMergeTree | `contact_id` | Contact dimension data |
| `orders_fact` | ReplacingMergeTree | `(contact_id, order_id)` | Order-level aggregations |
| `order_items_fact` | ReplacingMergeTree | `(contact_id, order_id, sku, product_id)` | Line-item detail for affinity |
| `membership_events` | MergeTree | `(contact_id, channel, occurred_at)` | Opt-in/opt-out tracking |
| `campaign_events` | MergeTree | `(campaign_id, contact_id, occurred_at)` | Delivery tracking |
| `product_taxonomy` | ReplacingMergeTree | `(product_id, tag, category_id)` | Product → tag/category mapping |
| `product_variation_taxonomy` | ReplacingMergeTree | `(product_id, variation_id)` | Product → variation mapping |

Materialized views (manually refreshed via TRUNCATE + INSERT SELECT):

| View | Engine | Computed From |
|---|---|---|
| `rfm_scores_mv` | ReplacingMergeTree | `orders_fact` |
| `tag_affinity_mv` | SummingMergeTree | `order_items_fact` × `product_taxonomy` |
| `category_affinity_mv` | SummingMergeTree | `order_items_fact` × `product_taxonomy` |
| `variation_affinity_mv` | SummingMergeTree | `order_items_fact` × `product_variation_taxonomy` |

## Event Consumption

The analytics module listens to the following integration events and ingests data incrementally:

| Topic | Action |
|---|---|
| `contacts.v1.created` | Upsert 1 contact snapshot |
| `contacts.v1.updated` | Upsert 1 contact snapshot |
| `orders.v1.created` | Upsert 1 order fact + N item facts |
| `orders.v1.updated` | Upsert 1 order fact + N item facts |
| `orders.v1.status.updated` | Upsert 1 order fact + N item facts |
| `membership.v1.changed` | Insert 1 membership event |
| `campaign.v1.delivery` | Insert 1 campaign event |

Non-retriable errors (missing IDs, malformed payloads) are permanently discarded. Transient failures are retried by the messaging bus.

## RFM Scoring

RFM (Recency, Frequency, Monetary) assigns each contact a score from 1–5 on three dimensions:

### Score Computation

The `rfm_scores_mv` view computes raw metrics from `orders_fact`:

```sql
SELECT contact_id,
       dateDiff('day', max(created_at), now())  AS recency_days,
       countDistinct(order_id)                   AS frequency,
       sum(total_value)                          AS monetary
FROM orders_fact FINAL
GROUP BY contact_id
```

### Band Configuration

Each dimension has configurable thresholds stored in MySQL `rfm_band_configs`:

| Dimension | Direction | Default Band5→Band2 Thresholds |
|---|---|---|
| Recency | Descending (fewer days = better) | 7, 30, 90, 180 days |
| Frequency | Ascending (more orders = better) | 10, 6, 3, 2 orders |
| Monetary | Ascending (more spend = better) | 1000, 500, 200, 50 currency |

**Descending** means a lower raw value maps to a higher score (recency: 5 days → score 5). **Ascending** means a higher raw value maps to a higher score (10 orders → score 5).

The scoring function applies `multiIf()` expressions in ClickHouse to convert raw metrics to 1–5 scores at query time:

```
Score 5: value < Band5Min (descending) or value >= Band5Min (ascending)
Score 4: Band5Min ≤ value < Band4Min (descending) or Band4Min ≤ value < Band5Min (ascending)
Score 3: Band4Min ≤ value < Band3Min
Score 2: Band3Min ≤ value < Band2Min
Score 1: everything else
```

### RFM Groups

RFM groups are named segments defined by score ranges. Each group has conditions that specify min/max bounds on R, F, and M scores:

```json
{
  "name": "Champions",
  "slug": "champions",
  "conditions": { "rMin": 4, "fMin": 4, "mMin": 4 }
}
```

A contact belongs to a group if their R, F, and M scores all fall within the specified bounds. Groups are used by the segment filter DSL (`rfm_group` filter type) — when a segment references an RFM group by slug, the conditions are expanded into `rfm_range` filters at resolution time.

### RFM Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/analytics/rfm/bands` | List all band configs |
| `PUT` | `/analytics/rfm/bands/:dimension` | Update thresholds for one dimension |
| `POST` | `/analytics/rfm/groups` | Create an RFM group |
| `GET` | `/analytics/rfm/groups` | List all groups |
| `GET` | `/analytics/rfm/groups/:id` | Get one group |
| `PUT` | `/analytics/rfm/groups/:id` | Update a group |
| `DELETE` | `/analytics/rfm/groups/:id` | Delete a group |
| `GET` | `/analytics/rfm/contacts/:contactId/score` | Score one contact |
| `POST` | `/analytics/rfm/contacts/score-batch` | Score up to 1000 contacts |
| `POST` | `/analytics/rfm/refresh` | Truncate + repopulate `rfm_scores_mv` |

## Product Affinity

Product affinity measures how strongly a contact is associated with product tags, categories, and variations based on their purchase history.

### Time-Decay Formula

All affinity scores use exponential time decay:

$$\text{affinity\_score} = \sum_{\text{purchases}} \text{item\_value} \times e^{-0.01 \times \Delta\text{days}}$$

where $\Delta\text{days} = \text{dateDiff('day', order\_created\_at, now())}$.

The decay constant 0.01 produces a half-life of approximately **69 days** — a purchase from 69 days ago contributes half the affinity of an identical purchase today. This ensures recent behavior dominates while historical patterns still have influence.

### Affinity Dimensions

| Dimension | Join Path | Grouped By |
|---|---|---|
| **Tag** | `order_items_fact` × `product_taxonomy` | `(contact_id, tag)` |
| **Category** | `order_items_fact` × `product_taxonomy` | `(contact_id, category_id)` |
| **Variation** | `order_items_fact` × `product_variation_taxonomy` | `(contact_id, variation_name, variation_value)` |

Each dimension also tracks `total_spent` (sum of raw item values) and `purchase_count` (number of line items).

### Tag Correlations

The MySQL `tag_correlations` table stores cross-sell relationships:

```
source_tag: "backpack"  →  target_tag: "laptop-sleeve"  (probability: 85)
source_tag: "backpack"  →  target_tag: "water-bottle"   (probability: 72)
```

The recommendation engine uses correlations to expand a contact's affinity tags into related tags. If a contact has high affinity for "backpack", the engine also considers products tagged "laptop-sleeve" and "water-bottle" as candidates.

### Affinity Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/analytics/affinity/contacts/:contactId` | Full affinity profile (tags + categories + variations) |
| `GET` | `/analytics/affinity/contacts/:contactId/tags` | Tag affinities only |
| `GET` | `/analytics/affinity/contacts/:contactId/categories` | Category affinities only |
| `GET` | `/analytics/affinity/contacts/:contactId/variations` | Variation affinities only |
| `POST` | `/analytics/affinity/refresh` | Refresh all 3 affinity materialized views |

## Product Recommendations

The recommendation engine produces personalized product lists for a contact by combining affinity data with catalog filtering:

### Resolution Pipeline

```
1. Load pinned products by ID
2. Build exclusion set (explicit excludes + pinned + purchased)
3. Load contact's top 20 tag affinities
4. Expand via tag_correlations → get related tags
5. Fetch product candidates matching base tags (union/intersection)
6. Score candidates by affinity (sum of matching tag scores)
7. Filter: realm, price range, categories, tags, variations
8. Combine: pinned products first + dynamic ranked results
9. Resolve display data: realm-specific name, price, image URL, product URL
```

Recommendation requests support the full ProductBlock filter DSL:

| Parameter | Type | Purpose |
|---|---|---|
| `baseTags` | `[]string` | Required product tag filters |
| `baseTagMode` | `"any"/"all"` | Union or intersection of base tags |
| `affinity` | `bool` | Enable contact affinity ranking |
| `minScore` | `float64` | Minimum affinity score threshold [0–100] |
| `categoryIds` | `[]string` | Category restriction |
| `excludeCategoryIds` | `[]string` | Category exclusion |
| `includeTags` / `excludeTags` | `[]string` | Tag inclusion/exclusion |
| `minPrice` / `maxPrice` | `*float64` | Price range |
| `excludePurchased` | `bool` | Remove already-purchased products |
| `realm` | `string` | Datasheet/gallery realm |
| `limit` | `int` | Max results [1–10] |
| `pinnedIds` | `[]string` | Always-first products |
| `excludeIds` | `[]string` | Excluded products |
| `filterVariationIds` | `[]string` | Restrict to specific variations |
| `preferVariationIds` | `[]string` | Bias gallery images toward these variations |

### Endpoint

```
GET /analytics/recommendations/contacts/:contactId?baseTags=backpack&affinity=true&limit=5
```

## Segment Resolution

The analytics module provides the `Resolver` interface consumed by the segment module:

```go
type Resolver interface {
    ResolveContacts(ctx, filter SegmentFilter, page, limit int) ([]string, error)
    CountContacts(ctx, filter SegmentFilter) (int64, error)
}
```

The ClickHouse query engine builds dynamic WHERE clauses from the `SegmentFilter` struct, supporting 20+ filter conditions including city codes, spend thresholds, order recency, membership status, metadata lookups, RFM score ranges, and affinity percentages.

Affinity-based segment filters use **relative** thresholds — a contact's score for a specific tag is compared against their own maximum tag score, not an absolute value:

```sql
maxIf(affinity_score, tag IN (?)) * 100.0 / nullIf(max(affinity_score), 0) >= ?
```

This means "contacts whose relative tag affinity for 'backpack' is at least 60% of their strongest tag affinity."

## Configuration

| Env Var | Type | Default | Purpose |
|---|---|---|---|
| `ANALYTICS_ENABLED` | `bool` | `false` | Master toggle |
| `ANALYTICS_CLICKHOUSE_DSN` | `string` | `""` | ClickHouse connection string |
| `ANALYTICS_CLICKHOUSE_MAX_OPEN_CONNS` | `int` | `10` | Connection pool max open |
| `ANALYTICS_CLICKHOUSE_MAX_IDLE_CONNS` | `int` | `5` | Connection pool max idle |
| `ANALYTICS_CLICKHOUSE_CONN_MAX_LIFETIME_MS` | `int64` | `600000` | Connection max lifetime (ms) |
| `ANALYTICS_CLICKHOUSE_BATCH_SIZE` | `int` | `1000` | Batch insert size |
| `ANALYTICS_CLICKHOUSE_FLUSH_INTERVAL_MS` | `int64` | `5000` | Flush interval (ms) |
| `ANALYTICS_CLICKHOUSE_MIGRATION_ENABLED` | `bool` | `true` | Auto-apply ClickHouse schema |
