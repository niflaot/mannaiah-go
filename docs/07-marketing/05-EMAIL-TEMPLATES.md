# Email Templates

Campaign emails are rendered using **Go `text/template`** syntax. Each email is rendered per-contact, allowing personalization through contact data, custom variables, and dynamically resolved product blocks.

## Template Context

The rendering context (`TemplateContext`) provides three top-level objects:

```go
type TemplateContext struct {
    Contact  ContactTemplateData
    Custom   map[string]string
    Products map[string][]TemplateProduct
}
```

### `.Contact`

Per-contact personalization data, resolved from the contact database at send time:

| Field | Type | Example | Description |
|---|---|---|---|
| `.Contact.Name` | `string` | `"María"` | Short display name (first-name preference) |
| `.Contact.FullName` | `string` | `"María García"` | Complete display name |
| `.Contact.FirstName` | `string` | `"María"` | First word before first space |
| `.Contact.Email` | `string` | `"maria@example.com"` | Actual recipient email |
| `.Contact.LastSaleDate` | `*time.Time` | — | Most recent purchase date, or nil |

If the contact has no name data, the email address is used as fallback for `.Name` and `.FullName`.

### `.Custom`

Campaign-level custom variables plus auto-injected values:

```json
{
  "brand": "My Store",
  "unsubscribe_url": "https://app.example.com/unsubscribe?token=..."
}
```

The `unsubscribe_url` is automatically injected when the unsubscribe URL configuration is set on the campaign service. It generates a signed, time-limited URL (default TTL: 30 days).

### `.Products`

A map of product block ID → resolved products. Each product block in the campaign configuration is resolved per-contact using the affinity engine:

| Field | Type | Description |
|---|---|---|
| `.ID` | `string` | Product identifier |
| `.Name` | `string` | Realm-resolved display name |
| `.Price` | `float64` | Product price |
| `.ImageURL` | `string` | Public image URL |
| `.URL` | `string` | Realm-scoped product detail URL |

## Template Functions

| Function | Signature | Example |
|---|---|---|
| `formatDate` | `*time.Time → string` | `{{ formatDate .Contact.LastSaleDate }}` → `"2024-01-15"` |
| `formatPrice` | `float64 → string` | `{{ formatPrice .Price }}` → `"29990.00"` |
| `default` | `(fallback, val) → string` | `{{ default "Estimado/a" .Contact.Name }}` |
| `upper` | `string → string` | `{{ upper .Contact.Name }}` → `"MARÍA"` |
| `lower` | `string → string` | `{{ lower .Custom.brand }}` → `"my store"` |

## Template Examples

### Basic Personalization

```html
<h1>Hola {{ default "Estimado/a" .Contact.Name }}!</h1>
<p>Tenemos ofertas especiales para ti.</p>
{{ if .Contact.LastSaleDate }}
  <p>Tu última compra fue el {{ formatDate .Contact.LastSaleDate }}.</p>
{{ end }}
<p><a href="{{ .Custom.unsubscribe_url }}">Desuscribirse</a></p>
```

### Product Block Loop

```html
{{ with index .Products "hero_products" }}
<table>
  {{ range . }}
  <tr>
    <td><img src="{{ .ImageURL }}" width="200" /></td>
    <td>
      <strong>{{ .Name }}</strong><br/>
      ${{ formatPrice .Price }}<br/>
      <a href="{{ .URL }}">Ver producto</a>
    </td>
  </tr>
  {{ end }}
</table>
{{ end }}
```

### Multiple Blocks

A campaign can have several product blocks, each with different filter criteria:

```html
<h2>Recomendados para ti</h2>
{{ with index .Products "personalized" }}
  {{ range . }}
    <div>{{ .Name }} - ${{ formatPrice .Price }}</div>
  {{ end }}
{{ end }}

<h2>Ofertas en mochilas</h2>
{{ with index .Products "backpacks" }}
  {{ range . }}
    <div>{{ .Name }} - ${{ formatPrice .Price }}</div>
  {{ end }}
{{ end }}
```

## Link Rewriting

After template rendering, all `http://` and `https://` links in the HTML output are rewritten to append UTM tracking parameters:

| Parameter | Value |
|---|---|
| `utm_source` | `email` |
| `utm_medium` | `campaign` |
| `utm_campaign` | Campaign slug |
| `utm_id` | Campaign ID |

Non-HTTP links (`mailto:`, `tel:`, `#`) are left unchanged.

Example:
```
Before: <a href="https://store.example.com/products/123">
After:  <a href="https://store.example.com/products/123?utm_source=email&utm_medium=campaign&utm_campaign=summer-sale&utm_id=abc-123">
```

## Product Blocks

Product blocks define how products are selected and filtered for each block ID in the template. They are stored as a JSON array in the `product_blocks` column of the campaigns table.

### ProductBlock Schema

| Field | Type | Default | Description |
|---|---|---|---|
| `id` | `string` | — | Block key in template `Products` map (required) |
| `baseTags` | `[]string` | — | Product base tag filters (at least one source required) |
| `baseTagMode` | `"any"/"all"` | `"any"` | Union or intersection of base tags |
| `useAffinity` | `bool` | `false` | Enable contact-affinity-driven ranking |
| `affinityMinScorePct` | `float64` | `0` | Minimum affinity score threshold [0, 100] |
| `categoryIds` | `[]string` | — | Category restriction (id/slug/name) |
| `excludeCategoryIds` | `[]string` | — | Category exclusion list |
| `includeTags` | `[]string` | — | Require at least one tag match |
| `excludeTags` | `[]string` | — | Exclude products with any of these tags |
| `minPrice` | `*float64` | — | Minimum price filter |
| `maxPrice` | `*float64` | — | Maximum price filter |
| `excludePurchasedProducts` | `bool` | `false` | Remove products the contact already bought |
| `limit` | `int` | — | Max products [1, 10] |
| `pinnedProductIds` | `[]string` | — | Always-first products (supports `id\|variation_id`) |
| `excludeProductIds` | `[]string` | — | Excluded products (supports `id\|variation_id`) |
| `filterVariationIds` | `[]string` | — | Restrict to products with specific variations |
| `preferVariationIds` | `[]string` | — | Bias gallery images toward these variations |

### Block Resolution

During template rendering, each product block is resolved through the affinity product provider:

1. Skip blocks with no `id` or no product source (no `baseTags` and no `pinnedProductIds`).
2. Call `AffinityProductProvider.GetProducts(ctx, contactID, block)`.
3. The provider maps the block to a `RecommendationQuery` and delegates to the analytics recommendation service.
4. Results are placed into `TemplateContext.Products[block.ID]`.
5. Resolution is **fail-open** — if a block fails, it is silently skipped and the template renders without it.

### Example Block Configuration

```json
[
  {
    "id": "hero_products",
    "baseTags": ["backpack", "travel"],
    "baseTagMode": "any",
    "useAffinity": true,
    "affinityMinScorePct": 30,
    "excludePurchasedProducts": true,
    "limit": 4,
    "pinnedProductIds": ["prod-123"]
  },
  {
    "id": "budget_picks",
    "baseTags": ["accessories"],
    "maxPrice": 15000,
    "excludeCategoryIds": ["premium"],
    "limit": 3
  }
]
```

## Rendering Limits

- Maximum rendered output size: **2 MiB**. Templates exceeding this limit produce `ErrOutputTooLarge`.
- Product block limit: **1–10** products per block.
