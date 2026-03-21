# Campaign Personalization — Frontend Integration Guide

> API version: **v2.6.0**
> Requires: Analytics module enabled (`ANALYTICS_ENABLED=true`), ClickHouse configured.

---

## Overview

The campaign personalization system lets you build dynamic email campaigns that render differently per recipient. Two orthogonal features are available:

| Feature | What it does |
|---|---|
| **Template variables** | Inject campaign-level strings into email copy |
| **Product blocks** | Inject per-contact affinity product recommendations (image, name, price) |

Both are declared when creating/updating a campaign and rendered automatically per contact during send.

---

## 1. Recommendation API (standalone)

Before wiring product blocks into campaigns you can test recommendation resolution directly.

### `GET /analytics/recommendations/contacts/:contactId`

Returns ranked product recommendations for one contact.

**Required permission:** `marketing:manage`

**Path parameters**

| Parameter | Type | Description |
|---|---|---|
| `contactId` | string | Recipient contact identifier |

**Query parameters**

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `baseTag` | string | yes | — | Only products carrying this tag are candidates |
| `categoryId` | string | no | — | Restrict candidates to one category |
| `realm` | string | no | `default` | Display realm for name/image resolution |
| `limit` | int | no | `3` | Max results returned (clamped to [1, 10]) |
| `affinity` | string | no | `false` | Set to `"true"` to enable affinity-driven filtering |
| `minScore` | float | no | `0` | Minimum affinity score percentile [0, 100] |

**Example request**

```http
GET /analytics/recommendations/contacts/c_abc123?baseTag=leather-goods&affinity=true&minScore=20&limit=3&realm=default
Authorization: Bearer <token>
```

**Example response**

```json
[
  {
    "id": "prod_001",
    "name": "Slim Leather Wallet",
    "price": 49.90,
    "imageUrl": "https://cdn.example.com/assets/wallet-main.jpg"
  },
  {
    "id": "prod_002",
    "name": "Card Holder",
    "price": 29.90,
    "imageUrl": "https://cdn.example.com/assets/cardholder.jpg"
  }
]
```

**Resolution logic (in order)**

1. If `affinity=true`, fetch up to 20 of the contact's highest-scoring product tags from ClickHouse (filtered by `minScore` percentile).
2. Expand those tags via `tag_correlations` (cross-sell expansion) to get a wider candidate tag set.
3. Query `products` that carry `baseTag`, optionally narrowing to products that also carry at least one expanded affinity tag.
4. If `categoryId` is set, further restrict to that category.
5. Rank survivors by summed affinity score of their product tags.
6. Resolve realm-aware display name (`product_datasheets[realm]`) and image URL (`product_gallery` visible in `realm`).

---

## 2. Campaign Template Variables

Use `templateVars` to inject campaign-wide custom strings into email bodies.

### Declaring variables on create/update

```json
PATCH /campaigns/:id
{
  "templateVars": {
    "promo_code": "SUMMER10",
    "hero_title": "Your summer picks are here"
  }
}
```

### Using variables in template bodies

Variables are available as `.Custom.<key>` inside `htmlBody` / `textBody`:

```html
<h1>{{ .Custom.hero_title }}</h1>
<p>Use code <strong>{{ .Custom.promo_code }}</strong> for 10% off.</p>
```

### Template function reference

| Function | Signature | Example |
|---|---|---|
| `upper` | `upper string → string` | `{{ upper .Contact.Name }}` |
| `lower` | `lower string → string` | `{{ lower .Custom.promo_code }}` |
| `default` | `default fallback val → string` | `{{ default "Friend" .Contact.Name }}` |
| `formatDate` | `formatDate *time.Time → string` | `{{ formatDate .Contact.LastSaleDate }}` |
| `formatPrice` | `formatPrice float64 → string` | `{{ formatPrice 49.90 }}` → `49.90` |

---

## 3. Campaign Product Blocks

Product blocks declare affinity-filtered product lists that are resolved per recipient during send. Each block renders into a named slot in the template context.

### Declaring product blocks on create/update

```json
PATCH /campaigns/:id
{
  "productBlocks": [
    {
      "id": "hero-products",
      "baseTag": "leather-goods",
      "useAffinity": true,
      "affinityMinScorePct": 25,
      "categoryId": "",
      "realm": "default",
      "limit": 3
    },
    {
      "id": "accessories",
      "baseTag": "accessories",
      "useAffinity": false,
      "categoryId": "cat_bags",
      "realm": "default",
      "limit": 2
    }
  ]
}
```

**ProductBlock fields**

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `id` | string | yes | — | Template key. Accessed as `.Products.<id>` |
| `baseTag` | string | yes | — | Only products with this tag are candidates |
| `useAffinity` | bool | no | `false` | Enable per-contact affinity expansion |
| `affinityMinScorePct` | float | no | `0` | Affinity score percentile threshold [0, 100] |
| `categoryId` | string | no | — | Restrict to one category |
| `realm` | string | no | `"default"` | Realm for name/image resolution |
| `limit` | int | no | `3` | Max products per block (clamped to [1, 10]) |

### Using product blocks in template bodies

Each block resolves into `[]TemplateProduct` accessible at `.Products.<id>`:

```html
{{ range .Products.hero-products }}
<div class="product-card">
  <img src="{{ .ImageURL }}" alt="{{ .Name }}">
  <p class="name">{{ .Name }}</p>
  <p class="price">${{ formatPrice .Price }}</p>
</div>
{{ else }}
<p>No recommendations available right now.</p>
{{ end }}
```

**TemplateProduct fields available in template**

| Field | Type | Description |
|---|---|---|
| `.ID` | string | Product identifier |
| `.Name` | string | Realm-resolved display name |
| `.Price` | float64 | Product price (0.0 when unset) |
| `.ImageURL` | string | Public URL of first realm-matched gallery image |

---

## 4. Contact Data in Templates

Per-contact data is injected automatically under `.Contact` during send:

| Field | Type | Description |
|---|---|---|
| `.Contact.Name` | string | Contact display name (falls back to email if empty) |
| `.Contact.Email` | string | Recipient email address |
| `.Contact.LastSaleDate` | `*time.Time` | Most recent purchase date, or `nil` if no purchases |

**Example usage**

```html
<p>Hi {{ default "there" .Contact.Name }},</p>
<p>
  {{ if .Contact.LastSaleDate }}
  Your last purchase was on {{ formatDate .Contact.LastSaleDate }}.
  {{ else }}
  Welcome — we haven't seen you in a while!
  {{ end }}
</p>
```

---

## 5. Full Create Request Example

```json
POST /campaigns
{
  "name": "Summer Leather Campaign",
  "slug": "summer-leather-2026",
  "channel": "email",
  "segmentId": "seg_leather_buyers",
  "subject": "Your leather picks for summer, {{ .Contact.Name }}",
  "htmlBody": "<h1>{{ .Custom.hero_title }}</h1><p>Hi {{ default \"there\" .Contact.Name }}, here are your picks:</p>{{ range .Products.hero-products }}<div><img src=\"{{ .ImageURL }}\"><b>{{ .Name }}</b> — ${{ formatPrice .Price }}</div>{{ end }}",
  "textBody": "{{ .Custom.hero_title }}\n\nHi {{ default \"there\" .Contact.Name }},\n{{ range .Products.hero-products }}- {{ .Name }} ${{ formatPrice .Price }}\n{{ end }}",
  "templateVars": {
    "hero_title": "Your summer leather picks"
  },
  "productBlocks": [
    {
      "id": "hero-products",
      "baseTag": "leather-goods",
      "useAffinity": true,
      "affinityMinScorePct": 20,
      "realm": "default",
      "limit": 3
    }
  ]
}
```

---

## 6. Behavior on Send

When `POST /campaigns/:id/send` is called:

1. The campaign fan-out resolves all segment contacts.
2. For each contact, the server:
   - Fetches contact name, email, and last sale date.
   - Resolves each product block (affinity lookup + catalog query).
   - Renders `htmlBody` and `textBody` with the per-contact `TemplateContext`.
3. Sends the rendered email via the configured provider.

**Fail-open policy:** If contact data fetch or product resolution fails for a contact, the raw unrendered template body is sent instead. The campaign is never aborted due to enrichment errors.

**Output cap:** Rendered bodies exceeding 2 MiB are not sent — the contact delivery is marked failed and logged.

---

## 7. Tag Correlation Setup (prerequisite for affinity)

For affinity expansion to yield useful product results, the `tag_correlations` table must be populated. Use the existing tag correlation API:

```json
POST /tags/correlations
{
  "sourceTag": "leather-goods",
  "targetTag": "wallets",
  "probability": 80,
  "notes": "Leather buyers strongly cross-sell to wallets"
}
```

Contacts who have purchased products tagged `leather-goods` will have that affinity, which expands to `wallets` — so products tagged `wallets` become candidates for their product blocks even if `baseTag` is `leather-goods`.

---

## 8. Error Reference

| HTTP status | Code | Meaning |
|---|---|---|
| 400 | `baseTag query parameter is required` | `baseTag` missing from recommendation endpoint |
| 401 | `unauthorized` | Missing or invalid `Authorization` header |
| 403 | `forbidden` | Token lacks `marketing:manage` permission |
| 404 | `campaign_not_found` | Campaign ID does not exist |
| 409 | `campaign_send_conflict` | Campaign is already processing or already sent |
| 500 | `internal_server_error` | Unexpected server error |
