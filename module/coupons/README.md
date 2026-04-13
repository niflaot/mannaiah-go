# Coupons Module

Provides coupon creation, management, and usage tracking with WooCommerce synchronization support.

## Key methods / endpoints / events

### REST Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/coupons` | Create a coupon (auto-generates a code when none provided) |
| `GET` | `/coupons` | List coupons with optional filters (`origin`, `active`, `code`, `limit`, `offset`) |
| `GET` | `/search/coupons` | Search coupons with optional filters (`term`, `email`, `contact`, `code`, `origin`, `discountType`, `limit`, `offset`) |
| `GET` | `/coupons/:id` | Get coupon by ID |
| `GET` | `/coupons/code/:code` | Get coupon by code |
| `PUT` | `/coupons/:id` | Update coupon (replaces assignment/scope lists wholesale) |
| `DELETE` | `/coupons/:id` | Soft-delete coupon |
| `POST` | `/coupons/:id/usage` | Record a coupon redemption for an order |

### Integration Events

| Topic | Description |
|-------|-------------|
| `coupons.v1.coupon.created` | Emitted when a coupon is created |
| `coupons.v1.coupon.updated` | Emitted when a coupon is updated |
| `coupons.v1.coupon.deleted` | Emitted when a coupon is soft-deleted |
| `coupons.v1.coupon.used` | Emitted when a coupon redemption is recorded |

## Domain concepts

- **Code**: Unique, uppercase. Auto-generated using a charset that avoids ambiguous characters (0/O, 1/I/L), formatted as `XXXX-XXXX-XXXX`.
- **Origin**: Tracks where the coupon was created (e.g., `manual`, `campaign`, `woocommerce`).
- **Discount types**: `fixed` (currency amount) or `percentage` (0–100).
- **Assignation**: Optional list of `assignedEmails` or `assignedContactIds` that may redeem the coupon. Empty = any user.
- **Usage limits**: `maxUsagesGlobal` (total redemptions) and `maxUsagesPerEmail`. Both nil = unlimited.
- **Scope**: `includedProductIds`, `includedCategoryIds`, `includedTagIds`. Empty = applies to all.
  - **Note**: Tag filtering is enforced by this system only. WooCommerce does not natively restrict coupons by product tags.
- **WooCommerce sync**: `woocommerceId` links the coupon to its WooCommerce counterpart for deduplication.

## Search endpoint contract

### Request

`GET /search/coupons`

Supported query filters:

| Query | Match type | Description |
|-------|------------|-------------|
| `term` | partial | Free-text match across coupon code, origin, assigned emails, and assigned contact identifiers |
| `email` | partial | Match within assigned email values |
| `contact` | partial | Match within assigned contact identifier values |
| `code` | partial | Match within coupon code |
| `origin` | exact | Match coupon origin |
| `discountType` | exact | Match `fixed` or `percentage` |
| `limit` | exact | Page size (default `50`) |
| `offset` | exact | Zero-based pagination offset (default `0`) |

### Response

Successful responses return HTTP `200` with the following JSON shape:

```json
{
  "items": [
    {
      "id": "coupon_123",
      "code": "SAVE-2026",
      "origin": "woocommerce",
      "discountType": "percentage",
      "discountAmount": 10,
      "maxUsagesGlobal": 100,
      "maxUsagesPerEmail": 1,
      "active": true,
      "expiresAt": "2026-12-31T23:59:59Z",
      "assignedEmails": ["user@example.com"],
      "assignedContactIds": ["contact_123"],
      "includedProductIds": ["product_1"],
      "includedCategoryIds": ["category_1"],
      "includedTagIds": ["tag_1"],
      "wooCommerceId": 42,
      "createdAt": "2026-04-13T12:00:00Z",
      "updatedAt": "2026-04-13T12:00:00Z"
    }
  ],
  "total": 1
}
```

## WooCommerce limitations

When syncing coupons to WooCommerce, the following fields have no native WooCommerce equivalent:

| Field | WooCommerce support | Fallback |
|-------|---------------------|---------|
| `includedTagIds` | ❌ Not supported | Stored in WC meta_data only; not enforced at checkout |
| `assignedContactIds` | ❌ Not supported | Contact emails are resolved and synced to `email_restrictions` |
| `origin` | ❌ Not supported | Stored in WC meta_data |

## Performance notes

- Usage counting uses indexed `COUNT(*)` queries on `coupon_usages(coupon_id)` and `coupon_usages(coupon_id, email)`.
- Idempotency for per-order application is enforced via the unique index `(coupon_id, order_id)`.
- For very high-traffic coupons, usage counts can be cached in Redis (not enabled by default).
