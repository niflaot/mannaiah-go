# RFM Scoring

RFM (Recency, Frequency, Monetary) is a customer segmentation technique that scores each contact on three dimensions from 1 to 5, producing a composite score from 3 to 15.

## Dimensions

| Dimension | Metric | Direction | Meaning of Score 5 |
|---|---|---|---|
| **Recency** | Days since last order | Descending | Purchased very recently |
| **Frequency** | Distinct order count | Ascending | Many orders |
| **Monetary** | Total order value | Ascending | High spend |

**Descending** means a lower raw value produces a higher score (fewer days since last purchase = better). **Ascending** means a higher raw value produces a higher score.

## Band Configuration

Each dimension has 4 threshold values that divide contacts into 5 bands. Thresholds are stored in MySQL (`rfm_band_configs`) and applied at ClickHouse query time via `multiIf()` expressions.

### Default Thresholds

| Dimension | Band5 Min | Band4 Min | Band3 Min | Band2 Min | Direction |
|---|---|---|---|---|---|
| Recency | 7 days | 30 days | 90 days | 180 days | Descending |
| Frequency | 10 orders | 6 orders | 3 orders | 2 orders | Ascending |
| Monetary | 1000 | 500 | 200 | 50 | Ascending |

### Scoring Examples

**Recency (descending):**

| Raw Value | Score | Band |
|---|---|---|
| 3 days | 5 | ≤ 7 days |
| 15 days | 4 | 8–30 days |
| 60 days | 3 | 31–90 days |
| 120 days | 2 | 91–180 days |
| 200 days | 1 | > 180 days |

**Frequency (ascending):**

| Raw Value | Score | Band |
|---|---|---|
| 12 orders | 5 | ≥ 10 |
| 8 orders | 4 | 6–9 |
| 4 orders | 3 | 3–5 |
| 2 orders | 2 | 2 |
| 1 order | 1 | < 2 |

## RFM Groups

Groups are named presets that match contacts by R/F/M score ranges:

```json
{
  "name": "Champions",
  "slug": "champions",
  "description": "High-value recent buyers",
  "conditions": {
    "rMin": 4, "rMax": 5,
    "fMin": 4, "fMax": 5,
    "mMin": 4, "mMax": 5
  }
}
```

A contact matches a group if all specified conditions are satisfied (unspecified bounds are unconstrained). Common group archetypes:

| Group | R | F | M | Description |
|---|---|---|---|---|
| Champions | 4–5 | 4–5 | 4–5 | Best customers |
| Loyal | — | 4–5 | — | Frequent buyers regardless of recency/spend |
| At Risk | 1–2 | 3–5 | 3–5 | Were good customers, now inactive |
| Hibernating | 1 | 1–2 | 1–2 | Low across all dimensions |
| New Customers | 4–5 | 1 | — | Recent first-time buyers |

### Group Expansion in Segments

When a segment filter references `rfm_group` by slug, the system:

1. Fetches the group from MySQL.
2. Maps conditions to an `rfm_range` filter with equivalent `rMin/rMax/fMin/fMax/mMin/mMax` parameters.
3. If the group has empty conditions and `exclude: false` — the clause is dropped (matches everyone).
4. If the group has empty conditions and `exclude: true` — it emits a clause that matches nobody.

## Materialized View Refresh

RFM scores are computed from the `orders_fact` ClickHouse table and stored in `rfm_scores_mv`. The refresh operation is **not automatic** — it must be triggered manually:

```
POST /analytics/rfm/refresh
```

This truncates and repopulates the view:

```sql
INSERT INTO rfm_scores_mv
SELECT contact_id,
       dateDiff('day', max(created_at), now()) AS recency_days,
       countDistinct(order_id)                  AS frequency,
       sum(total_value)                         AS monetary,
       now64(3)                                 AS updated_at
FROM orders_fact FINAL
GROUP BY contact_id
```

Band scoring happens at query time, not at refresh time, so updating band thresholds takes immediate effect without a new refresh.

## Monetary Percentiles

The `rfm_scores_mv` table supports quantile computation for monetary analysis:

```sql
quantile(0.2)(monetary), quantile(0.4)(monetary),
quantile(0.6)(monetary), quantile(0.8)(monetary)
```

These percentiles can inform band threshold tuning.

## API Reference

| Method | Path | Description |
|---|---|---|
| `GET` | `/analytics/rfm/bands` | List all band configurations |
| `PUT` | `/analytics/rfm/bands/:dimension` | Update thresholds for one dimension |
| `POST` | `/analytics/rfm/groups` | Create an RFM group |
| `GET` | `/analytics/rfm/groups` | List all groups |
| `GET` | `/analytics/rfm/groups/:id` | Get one group |
| `PUT` | `/analytics/rfm/groups/:id` | Update a group |
| `DELETE` | `/analytics/rfm/groups/:id` | Delete a group |
| `GET` | `/analytics/rfm/contacts/:contactId/score` | Score one contact |
| `POST` | `/analytics/rfm/contacts/score-batch` | Score up to 1000 contacts |
| `POST` | `/analytics/rfm/refresh` | Truncate + repopulate `rfm_scores_mv` |
