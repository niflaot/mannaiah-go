# Products — Domain Model & Realm System

## Product

The `Product` struct is the root aggregate of the PIM domain.

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | UUID primary key (`_id` in JSON) |
| `SKU` | `string` | Seller-defined stock keeping unit, unique |
| `Price` | `*float64` | Base price in the default currency |
| `Tags` | `[]string` | Tag names associated with this product |
| `Gallery` | `[]GalleryItem` | Ordered media gallery |
| `Datasheets` | `[]Datasheet` | Channel-specific presentation data (see Realms section) |
| `Variations` | `[]string` | IDs of `Variation` entities that apply to this product |
| `Variants` | `[]Variant` | Specific variant combinations with optional SKU overrides |
| `CreatedAt` | `time.Time` | Creation timestamp |
| `UpdatedAt` | `time.Time` | Last update timestamp |
| `IsDeleted` | `bool` | Soft-delete flag |
| `DeletedAt` | `*time.Time` | Soft-delete timestamp |

---

## GalleryItem

A gallery item links a stored media asset to a product with positioning and scoping metadata.

| Field | Type | Description |
|-------|------|-------------|
| `AssetID` | `string` | Reference to an `assets` module asset |
| `Position` | `*int` | Display order in the main gallery (nil = unordered) |
| `VariationPosition` | `*int` | Position within a variation-specific gallery |
| `IsMain` | `bool` | Marks the primary product image |
| `IncludedRealms` | `[]string` | If empty, visible in all realms; otherwise restricted to listed realms |
| `VariationIDs` | `[]string` | Restrict this image to specific variation IDs |

---

## The Realm System

A **realm** is a named channel scope on a product. Each product carries a `Datasheets` slice where
every element addresses a distinct realm. This allows the same product to have different names,
descriptions, and attributes for different sales channels without duplicating the product graph.

### Datasheet

Each `Datasheet` is stored as rows in dedicated database tables (`product_datasheets`,
`product_datasheet_attributes`). Attributes are persisted individually as `(key, value_json)` pairs,
making them queryable and extensible without schema migrations.

| Field | Type | Description |
|-------|------|-------------|
| `Realm` | `string` | Realm identifier (e.g. `"default"`, `"falabella"`) |
| `Name` | `string` | Localised product name for this channel |
| `Description` | `string` | Full product description for this channel |
| `Attributes` | `map[string]any` | Channel-specific attributes (see per-realm tables below) |

### MergeDatasheets

When a product is updated via `PATCH /products/:id`, `MergeDatasheets(incoming)` applies the
following strategy **keyed by realm**:

| Condition | Result |
|-----------|--------|
| Incoming realm already exists on the product | `Name`, `Description`, and each attribute key are individually merged; incoming values override; keys absent in the update are preserved |
| Incoming realm does not exist yet | Datasheet is appended |
| Realm absent from the incoming slice | Left completely untouched |

This means a single `PATCH` can update only the `"falabella"` datasheet without affecting `"default"`.

---

## Realm Reference

### `"default"` Realm

The `"default"` realm is the canonical fallback used by any system that does not have a
channel-specific datasheet. It **must** be populated when a product is created.

**Consumers:** `module/campaign` forces `Realm = "default"` when building recommendation queries;
`module/analytics` loads it alongside all other realms for recommendation rendering.

**Attributes read by consumers:**

| Attribute Key | Type | Used By | Description |
|--------------|------|---------|-------------|
| `price` | `float64` | `module/campaign`, `module/analytics` | Displayed/filtered price in recommendation blocks |
| `url` | `string` | `module/campaign`, `module/analytics` | Product page URL for recommendation links |
| `<variationID>.url` | `string` | `module/campaign`, `module/analytics` | Variation-scoped product URL (key pattern: `"var-uuid.url"`) |

Additional common `"default"` attributes (not machine-read, but documented by convention):

| Attribute Key | Type | Description |
|--------------|------|-------------|
| `brand` | `string` | Product brand name |
| `model` | `string` | Product model / reference code |
| `material` | `string` | Primary material |
| `weight` | `number` | Product weight in grams |
| `dimensions` | `object` | `{ width, height, depth }` in centimetres |

There is no enforced schema beyond the keys above — teams can add product-specific fields freely.

### `"falabella"` Realm

The Falabella integration module reads this realm to build Seller Center XML submissions.
The realm name is configurable via `FALABELLA_PRODUCT_REALM` (default `"falabella"`).

**Consumer:** `module/falabella`

See [01-falabella/02-REALMS.md](01-falabella/02-REALMS.md) for the complete attribute contract.

### Gallery Realm Scoping

`GalleryItem.IncludedRealms` controls image visibility per channel:

| `IncludedRealms` value | Behaviour |
|-----------------------|-----------|
| `[]` (empty) | Image is visible in **all** realms |
| `["falabella"]` | Image is visible **only** when rendering the Falabella realm |
| `["default", "falabella"]` | Image is visible in both `default` and `falabella` realms |

The Falabella sync pipeline filters gallery items against the `"falabella"` realm before
submitting the image feed.

### Realm Consumer Map

| Realm | Writer | Reader | What is read |
|-------|--------|--------|--------------|
| `"default"` | Product API (manual) | `module/campaign`, `module/analytics` | `price`, `url`, `<variationID>.url`, gallery `IncludedRealms` |
| `"falabella"` | Product API (manual) | `module/falabella` | `Name`, `Description`, all `Attributes` (for Seller Center listing) |
| *(custom)* | Product API | External consumers | Free-form via `Datasheet.Attributes` |

> **Note on WooCommerce:** WooCommerce does **not** use product datasheets. The string `"woocommerce"`
> appears as the `Order.Realm` field (identifying order origin) but is not a product datasheet realm.

---

## Variant

A `Variant` represents one concrete sellable combination of variation values.

| Field | Type | Description |
|-------|------|-------------|
| `VariationIDs` | `[]string` | The variation IDs that define this combination |
| `SKU` | `string` | Optional SKU override for this specific combination |

---

## Port Layer

### `port/product.Repository`

```go
EnsureSchema(ctx context.Context) error
Create(ctx context.Context, p *Product) error
GetByID(ctx context.Context, id string) (*Product, error)
GetBySKU(ctx context.Context, sku string) (*Product, error)
GetByIDs(ctx context.Context, ids []string) ([]*Product, error)
List(ctx context.Context) ([]Product, error)
ListByTagsAndPrice(ctx context.Context, tags []string, minPrice, maxPrice *float64, page, pageSize int) ([]*Product, int64, error)
Update(ctx context.Context, p *Product) error
Delete(ctx context.Context, id string) error
```
