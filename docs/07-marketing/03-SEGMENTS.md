# Segments

Segments define reusable contact audiences through a composable filter DSL. When a campaign is sent, the segment is resolved into a list of contact IDs by delegating to the analytics engine's ClickHouse query builder.

## Domain Model

| Field | Type | Description |
|---|---|---|
| `id` | `UUID` | Primary key |
| `name` | `string` | Segment display name |
| `slug` | `string` | URL-safe identifier |
| `channel` | `string` | Channel context |
| `filters` | `[]Filter` | Array of filter clauses |

### Filter Structure

Each filter is a clause with a type, optional exclusion flag, optional legacy value, and a parameters map:

```json
{
  "type": "order_recency",
  "exclude": false,
  "parameters": { "days": 30 }
}
```

When `exclude: true`, the filter removes matching contacts instead of including them.

## Filter DSL Reference

### Contact Filters

| Type | Parameters | Description |
|---|---|---|
| `city_code_in` | Value: `[]string` | Contacts in specified city codes (legacy) |
| `city` | `codes: []string` | Contacts in specified city codes |
| `email_opt_in` | Value: `bool` | Contacts with email opt-in |
| `opt_in_status` | `channel: string`, `status: string` | Contacts with specific opt-in state |
| `metadata` | `key: string`, `value?: string` | Contacts with specific metadata key/value |

### Purchase Filters

| Type | Parameters | Description |
|---|---|---|
| `min_total_spend` | Value: `float64` | Contacts with total spend ≥ threshold |
| `purchased_sku` | `skus: []string` OR Value: `string` | Contacts who purchased specific SKUs |
| `order_recency` | `days: int > 0` | Contacts who ordered in last N days |
| `no_order_recency` | `days: int > 0` | Contacts with no orders in last N days |
| `category` | `pattern: string` | Contacts who purchased in a category pattern |
| `order_status` | `statuses: []string` | Contacts with orders in specific statuses |
| `first_purchase_only` | `enabled?: bool` | Contacts with exactly one order |
| `subscribed_no_buy` | `enabled?: bool` | Opted-in contacts with zero orders |
| `min_order_count` | `count: int > 0` OR Value: `int > 0` | Contacts with at least N distinct orders |

### Ranking Filters

| Type | Parameters | Description |
|---|---|---|
| `top_spenders` | `limit: int` OR `percentage: float64` | Top spenders by absolute count or percentage |

### RFM Filters

| Type | Parameters | Description |
|---|---|---|
| `rfm_group` | `slug: string` | Contacts matching an RFM group's conditions |
| `rfm_score` | `min?: int`, `max?: int` | Contacts within RFM total score range |
| `rfm_range` | `rMin/rMax/fMin/fMax/mMin/mMax` | Contacts within individual R/F/M score ranges |

**RFM group expansion**: When a segment contains an `rfm_group` filter, the system fetches the group's conditions and rewrites the clause to an equivalent `rfm_range` filter at resolution time.

### Affinity Filters

All affinity filters use **relative percentage thresholds** — a contact's score for a specific dimension is compared to their own maximum score for that dimension, not an absolute value.

| Type | Parameters | Description |
|---|---|---|
| `tag_affinity` | `tags: [{tag, minScorePct, relatedTags?}]` | Contact's relative tag affinity ≥ threshold% |
| `category_affinity` | `categories: [{categoryId, minScorePct}]` | Contact's relative category affinity ≥ threshold% |
| `variation_affinity` | `variations: [{name, value, minScorePct}]` | Contact's relative variation affinity ≥ threshold% |

Example — select contacts with ≥60% relative affinity for "backpack":

```json
{
  "type": "tag_affinity",
  "parameters": {
    "tags": [{ "tag": "backpack", "minScorePct": 60 }]
  }
}
```

This translates to the ClickHouse query:

```sql
maxIf(affinity_score, tag IN ('backpack')) * 100.0
  / nullIf(max(affinity_score), 0) >= 60
```

## Resolution

Segment resolution converts a filter set into a paginated list of contact IDs:

```
1. Validate all filters (type existence, required parameters)
2. Normalize filters (trim, clone params, drop empty)
3. Map domain filters → analytics SegmentFilter struct
4. Expand rfm_group → rfm_range (fetch group conditions from MySQL)
5. Delegate to analytics Resolver → ClickHouse query engine
6. Return paginated contact IDs
```

### Preview Count

`POST /segments/preview/count` accepts a filter array directly (without creating a segment) and returns the matching contact count. Useful for UI previews before saving a segment.

## API Endpoints

| Method | Path | Status | Description |
|---|---|---|---|
| `POST` | `/segments` | 201 | Create a segment |
| `GET` | `/segments` | 200 | List segments (paginated) |
| `GET` | `/segments/:id` | 200 | Get a single segment |
| `PATCH` | `/segments/:id` | 200 | Update a segment |
| `DELETE` | `/segments/:id` | 200 | Delete a segment |
| `POST` | `/segments/:id/resolve` | 200 | Resolve contact IDs (paginated) |
| `GET` | `/segments/:id/count` | 200 | Count matching contacts |
| `POST` | `/segments/preview/count` | 200 | Preview count from filters |

### Error Responses

| Condition | HTTP | Code |
|---|---|---|
| Invalid filter type or missing parameters | 400 | `invalid_payload` |
| Segment not found | 404 | `segment_not_found` |
| Analytics resolver not configured | 503 | `segment_backend_unavailable` |

## Database Schema

**Table: `segments`**

| Column | Type | Notes |
|---|---|---|
| `id` | VARCHAR (PK) | UUID, auto-generated |
| `name` | VARCHAR | |
| `slug` | VARCHAR | |
| `channel` | VARCHAR | |
| `filters_json` | TEXT | JSON-serialized `[]Filter` |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | |

Filters are stored as a single JSON column. The segment module has no ClickHouse tables — all analytical queries are delegated to the analytics module.
