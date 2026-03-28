# Products — Variations

A **Variation** captures one dimension along which a product differs — for example, colour or
size. Products reference variations by ID; the combination of several variation IDs on a `Variant`
describes one specific sellable configuration.

## Variation Domain Type

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | UUID primary key (`_id` in JSON) |
| `Name` | `string` | Human-readable dimension name (e.g. `"Color"`, `"Talla"`) |
| `Definition` | `Definition` | Rendering hint for the UI |
| `Value` | `string` | The specific value of this dimension (e.g. `"Rojo"`, `"XL"`) |
| `CreatedAt` | `time.Time` | Creation timestamp |
| `UpdatedAt` | `time.Time` | Last update timestamp |
| `IsDeleted` | `bool` | Soft-delete flag |
| `DeletedAt` | `*time.Time` | Soft-delete timestamp |

### Definition Values

| Value | Description |
|-------|-------------|
| `COLOR` | Rendered as a colour swatch |
| `SIZE` | Rendered as a size selector |
| `TEXT` | Rendered as a text label |

---

## Relationship to Products

A product's `Variations []string` field lists all the variation IDs that are _available_ for
that product. The `Variants []Variant` field then identifies _actual_ combinations:

```json
{
  "variations": ["var-red", "var-blue", "var-m", "var-l"],
  "variants": [
    { "variationIDs": ["var-red", "var-m"], "sku": "SHIRT-RED-M" },
    { "variationIDs": ["var-red", "var-l"], "sku": "SHIRT-RED-L" },
    { "variationIDs": ["var-blue", "var-m"], "sku": "" }
  ]
}
```

When mapping to Falabella, each `Variant` becomes a child product entry with `ParentSKU` set to
the parent product's SKU. Variant-scoped attributes can be stored in the product's datasheet using
the key notation `"<variantSKU>.<attributeName>"` (e.g. `"SHIRT-RED-M.color"`).

---

## Port Layer

### `port/variation.Repository`

```go
EnsureSchema(ctx context.Context) error
Create(ctx context.Context, v *Variation) error
GetByID(ctx context.Context, id string) (*Variation, error)
List(ctx context.Context) ([]Variation, error)
Update(ctx context.Context, v *Variation) error
Delete(ctx context.Context, id string) error
```

---

## HTTP Endpoints

All variation endpoints require a valid bearer token.

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| `POST` | `/variations` | `product:manage` | Create a variation |
| `GET` | `/variations` | `product:view` | List all variations |
| `GET` | `/variations/:id` | `product:view` | Get a single variation |
| `PATCH` | `/variations/:id` | `product:manage` | Update a variation |
| `DELETE` | `/variations/:id` | `product:manage` | Soft-delete a variation |

### Variation Object Schema

```json
{
  "_id": "uuid",
  "name": "Color",
  "definition": "COLOR",
  "value": "Rojo",
  "createdAt": "2026-01-01T00:00:00Z",
  "updatedAt": "2026-01-01T00:00:00Z",
  "isDeleted": false,
  "deletedAt": null
}
```
