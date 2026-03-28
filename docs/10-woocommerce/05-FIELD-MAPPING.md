# WooCommerce — Field Mapping

Complete field-by-field mapping tables for both inbound (WooCommerce → Mannaiah) and
outbound (Mannaiah → WooCommerce) transformations.

---

## Inbound: `WooOrder.Billing*` → `ContactSyncCommand`

| Source (`WooOrder`) | Target (`ContactSyncCommand`) | Transformation |
|---------------------|-------------------------------|----------------|
| `BillingEmail` | `Email` | Required; skip contact if blank |
| `BillingFirstName` | `FirstName` | As-is |
| `BillingLastName` | `LastName` | As-is |
| `BillingCompany` | `LegalName` | As-is |
| `BillingPhone` | `Phone` | `normalizePhone()` → `+57XXXXXXXXXX` |
| `BillingCity` | `CityCode` | `citycode.Resolve(city)` → 5-digit DANE code |
| `BillingAddress1` | `Address` | As-is |
| `BillingAddress2` | `AddressExtra` | As-is |
| `Metadata["_billing_document"]` | `DocumentNumber` | Only when present |
| _(constant)_ | `DocumentType` | `"CC"` (when `DocumentNumber` is present) |
| `CreatedAt` (earliest order for email) | `CreatedAt` | Oldest order date across all orders for this email |
| _(constant)_ | `Metadata["integration.source"]` | `"woocommerce"` |
| `ID` | `Metadata["integration.woocommerce.oldest_order_id"]` | `strconv.Itoa(order.ID)` |
| `Metadata["flock_checker_*"]` | `Metadata["flock_checker_*"]` | Copied verbatim |

---

## Inbound: `WooOrder` → `OrderSyncCommand`

| Source (`WooOrder`) | Target (`OrderSyncCommand`) | Transformation |
|---------------------|------------------------------|----------------|
| `ID` | `Identifier` | `strconv.Itoa(order.ID)` |
| _(constant)_ | `Realm` | `"woocommerce"` |
| `Status` | `Status` | Status mapping (see below) |
| `PaymentMethod` | `PaymentMethod` | As-is |
| `CreatedAt` | `CreatedAt` | As-is |
| `Billing*` | `Contact` | Full `ContactSyncCommand` mapping (table above) |
| `Shipping*` | `ShippingAddress` | See shipping address mapping (below) |
| `ShippingCharges` | `ShippingCharges` | Direct mapping |
| `Items` | `Items` | Skip items where SKU is blank AND Name is blank |
| _(constant)_ | `Metadata["integration.source"]` | `"woocommerce_sync"` |
| `Metadata` (all keys) | `Metadata` | Copied verbatim (merged) |
| `Comments[0].Content` | `Comments[0].Comment` | Author forced to `"system"` |

---

## Inbound: `WooOrder.Shipping*` → `OrderSyncShippingAddress`

| Source | Target | Transformation |
|--------|--------|----------------|
| `ShippingFirstName` + `" "` + `ShippingLastName` | `Name` | Concatenated |
| `ShippingAddressLine1` | `Address` | As-is |
| `ShippingAddressLine2` | `Address2` | As-is |
| `ShippingPhone` | `Phone` | As-is |
| `ShippingCityCode` | `CityCode` | Pre-resolved from WooCommerce order fetch |

If all shipping fields are blank, `ShippingAddress` is set to `nil`. The `hasCustomShippingAddress`
flag in the order is set to `true` only when shipping fields differ from billing fields.

---

## Inbound: WooCommerce Status → Domain Status

| WooCommerce `status` | Domain `Status` |
|---------------------|----------------|
| `"pending-payment"` | `PENDING` |
| `"processing"` | `CREATED` |
| `"on-hold"` | `HOLD` |
| `"completed"` | `COMPLETED` |
| `"cancelled"` | `CANCELLED` |
| `"canceled"` | `CANCELLED` |
| `"failed"` | `CANCELLED` |
| _(anything else)_ | `CREATED` |

---

## Inbound: `WooOrderItem` → `OrderSyncItem`

| Source | Target | Notes |
|--------|--------|-------|
| `SKU` | `SKU` | |
| `Name` | `AlternateName` | Used as fallback when SKU resolution fails |
| `Quantity` | `Quantity` | |
| `UnitPrice` | `Value` | Per-unit price |
| _(skipped)_ | — | When both `SKU` and `Name` are blank |

---

## Outbound: `OrderEventPayload` → `MainstreamOrderUpdateCommand`

| Source (`OrderEventPayload`) | Target (`MainstreamOrderUpdateCommand`) | Transformation |
|------------------------------|------------------------------------------|----------------|
| `Identifier` | `Identifier` | Must parse as integer (guard condition) |
| `LatestStatus.Status` | `Status` | `normalizeWooStatus()` — domain → WooCommerce |
| `ShippingAddress` | `ShippingAddress` | Direct |
| `ShippingCharges` | `ShippingCharges` | Direct |
| `Items` | `Items` | Direct |

---

## Outbound: Domain Status → WooCommerce Status

| Domain `Status` | WooCommerce `status` |
|----------------|---------------------|
| `PENDING` | `"pending-payment"` |
| `CREATED` | `"processing"` |
| `HOLD` | `"on-hold"` |
| `COMPLETED` | `"completed"` |
| `CANCELLED` | `"cancelled"` |

---

## Outbound: Line Item SKU Resolution

Before building the WooCommerce PUT payload, the adapter resolves internal SKUs to
WooCommerce product IDs:

```
For each unique SKU in Items:
  GET /wp-json/wc/v3/products?sku={sku}
    ← [{ id: 456, name: "...", ... }]
    cache: sku → product_id (in-memory, per operation only)

If SKU not found: item is excluded from line_items in the WooCommerce request
```

This is per-operation only — no cross-request SKU cache exists today.
