# Products — Tags & Correlation Graph

The tag system serves two purposes: **classifying products** for category filters and discovery,
and **expressing weighted cross-sell relationships** between tags to power recommendation engines
and campaign audience targeting.

---

## Tag

A `Tag` is a simple named label attached to products.

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `uint` | Auto-increment primary key |
| `Name` | `string` | Unique tag name |
| `CreatedAt` / `UpdatedAt` | `time.Time` | Timestamps |
| `DeletedAt` | `*time.Time` | Soft-delete timestamp |

### Auto-registration

Tags are auto-registered: any tag name in `product.Tags` is implicitly created via `EnsureAll(names)`
before the product is persisted. There is no manual "create tag" endpoint — tags emerge from product
data. Soft-deleted tags remain on existing products; re-assigning a deleted tag name to a new product
restores (un-deletes) the tag automatically.

---

## TagCorrelation

A `TagCorrelation` records how likely a shopper interested in `SourceTag` is also interested in
`TargetTag`. It is the foundation for the cross-sell recommendation and campaign segmentation
features.

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `uint` | Auto-increment primary key |
| `SourceTag` | `string` | Lexicographically smaller tag in the pair (normalised) |
| `TargetTag` | `string` | Lexicographically larger tag in the pair (normalised) |
| `Probability` | `float64` | Cross-sell weight, range **0.00 – 100.00** |
| `Notes` | `string` | Optional human-readable rationale |
| `CreatedAt` / `UpdatedAt` | `time.Time` | Timestamps |

### Lexicographic Normalisation

When a correlation is created, if `sourceTag > targetTag` alphabetically, the two values are
**swapped** before storage. This ensures the pair `(A, B)` and `(B, A)` always become the same
database row, preventing duplicates and making bidirectional lookup trivial.

```
POST {sourceTag: "sport", targetTag: "running"}
  → stored as: source="running", target="sport"  (r < s)

POST {sourceTag: "running", targetTag: "sport"}
  → stored as: source="running", target="sport"  (same row → ErrDuplicateCorrelation)
```

---

## How the Correlation Graph Is Built

**The correlation graph is entirely human-curated.** There is no automatic computation from
purchase history or clickstream data inside Mannaiah. Each `TagCorrelation` row is created by a
marketing operator via the API with an explicit `Probability` value.

The intended workflow is:

1. Extract co-purchase or co-view data from an external analytics source (e.g. ClickHouse analytics
   module, BigQuery, or manual review).
2. Decide the cross-sell probability for each tag pair based on that analysis.
3. `POST /tags/correlations` with the pair and probability.
4. Any recommendation-oriented consumer reads the correlation graph when building ranked product suggestions
   to score and rank products.

### Uniqueness Constraint

The database enforces `UNIQUE(source_tag, target_tag)` via `idx_tag_correlations_pair`. Attempting
to create a duplicate pair (in either order) returns `ErrDuplicateCorrelation`.

---

## Correlation Graph Examples

### Example 1 — Building a simple graph

A store sells running gear and gym equipment. Based on purchase data analysis:

| Source → Target | Probability | Notes |
|----------------|-------------|-------|
| `running` → `sport` | 91.0 | Almost all running buyers also browse sport |
| `running` → `gym` | 67.5 | Running buyers frequently cross-shop gym equipment |
| `gym` → `sport` | 84.0 | Gym buyers consistently engage with sport content |
| `yoga` → `sport` | 72.0 | Inferred from co-view sessions |
| `yoga` → `gym` | 55.0 | Moderate cross-sell potential |

After normalisation the stored pairs are symmetrical — `ListCorrelationsBySource("running")` also
returns rows where `running` appears as `target_tag`, so the bidirectional graph is queryable from
either side.

```
          91.0
running ←──────────→ sport
   │  \                  │
67.5   \                84.0
   │    \                │
   ↓     ↘               ↓
  gym  ←──────────→  (implicit)
         55.0
         yoga
```

### Example 2 — Updating a probability

After a seasonal campaign, `running → gym` crosses 70%:

```http
PATCH /tags/correlations/4
{ "probability": 71.5, "notes": "Updated after Q1 2026 campaign results" }
```

### Example 3 — Querying from a seed tag

```http
GET /tags/correlations/source/running
```

Returns all correlations where `running` appears as either `source_tag` or `target_tag`:

```json
[
  { "id": 1, "sourceTag": "running", "targetTag": "sport",   "probability": 91.0 },
  { "id": 2, "sourceTag": "gym",     "targetTag": "running", "probability": 67.5 }
]
```

The recommendation engine uses this to find related tags and discover products carrying those tags,
ranked by probability descending.

---

## Port Layer

### `port/tag.Repository`

```go
EnsureAll(ctx context.Context, names []string) error
List(ctx context.Context) ([]Tag, error)
SoftDelete(ctx context.Context, name string) error            // cascades deletion from product_tags
ListCorrelations(ctx context.Context) ([]TagCorrelation, error)
ListCorrelationsBySource(ctx context.Context, sourceTag string) ([]TagCorrelation, error)
CreateCorrelation(ctx context.Context, c *TagCorrelation) error
UpdateCorrelation(ctx context.Context, id uint, probability *float64, notes *string) (*TagCorrelation, error)
DeleteCorrelation(ctx context.Context, id uint) error         // hard delete
```

`ListCorrelationsBySource` queries `WHERE source_tag = ? OR target_tag = ?` so both directions of a
normalised pair are returned for any given tag name.

---

## HTTP Endpoints

### Tags

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| `GET` | `/tags` | `product:tags` | List all non-deleted tags |
| `DELETE` | `/tags/:name` | `marketing:manage` | Soft-delete a tag; cascades `product_tags` |

### Correlations

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| `GET` | `/tags/correlations` | `marketing:manage` | List all correlations |
| `GET` | `/tags/correlations/source/:tag` | `marketing:manage` | All correlations for a given tag (both directions) |
| `POST` | `/tags/correlations` | `marketing:manage` | Create a correlation |
| `PATCH` | `/tags/correlations/:id` | `marketing:manage` | Update probability or notes |
| `DELETE` | `/tags/correlations/:id` | `marketing:manage` | Hard-delete a correlation |

> Correlation routes are registered **before** `/tags/:name` to prevent the path segment
> `correlations` from being parsed as a tag name.

### Create Correlation — Request Body

```json
{
  "sourceTag": "sport",
  "targetTag": "running",
  "probability": 91.0,
  "notes": "High co-purchase rate per Q4 2025 data"
}
```

Validation rules:
- `sourceTag` and `targetTag` are required and must differ (`ErrSelfCorrelation`).
- `probability` must be in `[0.0, 100.0]`.
- The pair (after normalisation) must not already exist (`ErrDuplicateCorrelation`).
