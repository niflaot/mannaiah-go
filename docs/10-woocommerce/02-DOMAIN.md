# WooCommerce — Domain Types

This module defines no persistent entities of its own. All types here are value objects
used as data transfer contracts between WooCommerce's REST API and Mannaiah's domain ports.

---

## Source Types (WooCommerce → Mannaiah)

### `WooOrder`

The raw representation of a WooCommerce order after API fetch and normalisation.

| Field | Type | Notes |
|-------|------|-------|
| `ID` | `int` | WooCommerce internal order ID |
| `BillingEmail` | `string` | Required; orders with blank email are skipped |
| `BillingFirstName` / `BillingLastName` | `string` | |
| `BillingCompany` | `string` | Used as `LegalName` for B2B contacts |
| `BillingPhone` | `string` | Raw format, normalised to `+57…` |
| `BillingPhoneNormalized` | `string` | After `normalizePhone()` |
| `BillingAddress1` / `BillingAddress2` | `string` | |
| `BillingCity` | `string` | Human city name, resolved to a DANE city code |
| `BillingCityCode` | `string` | After `citycode.Resolve()` |
| `ShippingFirstName` / `ShippingLastName` / `ShippingCompany` | `string` | |
| `ShippingPhone` | `string` | |
| `ShippingAddressLine1` / `ShippingAddressLine2` | `string` | |
| `ShippingCityCode` | `string` | |
| `PaymentMethod` | `string` | |
| `Status` | `string` | WooCommerce raw status (e.g. `"processing"`) |
| `CreatedAt` | `time.Time` | |
| `Items` | `[]WooOrderItem` | |
| `ShippingCharges` | `[]WooOrderShippingCharge` | |
| `Comments` | `[]WooOrderComment` | |
| `Metadata` | `map[string]string` | All WooCommerce order meta keys |

### `WooOrderItem`

| Field | Type |
|-------|------|
| `SKU` | `string` |
| `Name` | `string` |
| `Quantity` | `int` |
| `UnitPrice` | `float64` |
| `TotalPrice` | `float64` |
| `ProductID` | `int` |
| `VariationID` | `int` |

Items with an empty `SKU` **and** empty `Name` are silently skipped during mapping.

### `WooOrderShippingCharge`

| Field | Type |
|-------|------|
| `Name` | `string` |
| `MethodID` | `string` |
| `Total` | `float64` |

### `WooOrderComment`

| Field | Type |
|-------|------|
| `Author` | `string` |
| `Content` | `string` |
| `CreatedAt` | `time.Time` |

---

## Port Commands (Inbound)

### `ContactSyncCommand`

Passed to `ContactSyncTarget.UpsertByEmail()`.

| Field | Type | Notes |
|-------|------|-------|
| `Email` | `string` | Lookup key |
| `FirstName` / `LastName` | `string` | |
| `LegalName` | `string` | From `BillingCompany` |
| `Phone` | `string` | Normalised E.164-ish (`+57…`) |
| `Address` / `AddressExtra` | `string` | |
| `CityCode` | `string` | Resolved DANE code |
| `DocumentType` / `DocumentNumber` | `string` | From `_billing_document` meta key |
| `CreatedAt` | `*time.Time` | Oldest order date for this email |
| `Metadata` | `map[string]string` | Includes `integration.source = "woocommerce"` |

### `OrderSyncCommand`

Passed to `OrderSyncTarget.UpsertByIdentifier()`.

| Field | Type | Notes |
|-------|------|-------|
| `Identifier` | `string` | `strconv.Itoa(wooOrder.ID)` |
| `Realm` | `string` | Always `"woocommerce"` |
| `Status` | `string` | Normalised domain status |
| `PaymentMethod` | `string` | |
| `CreatedAt` | `*time.Time` | |
| `Contact` | `ContactSyncCommand` | Linked contact |
| `ShippingAddress` | `*OrderSyncShippingAddress` | nil if no custom shipping |
| `ShippingCharges` | `[]OrderSyncShippingCharge` | |
| `Items` | `[]OrderSyncItem` | Non-empty-named/SKU items only |
| `Metadata` | `map[string]string` | Includes `integration.source = "woocommerce_sync"` |
| `Comments` | `[]OrderSyncComment` | `customer_note` from WooCommerce |

---

## Port Commands (Outbound)

### `MainstreamOrderUpdateCommand`

Passed to `OrderDestination.UpdateOrderFromMainstream()`. Built from a
`orders.v1.*` event payload.

| Field | Type | Notes |
|-------|------|-------|
| `Identifier` | `string` | WooCommerce numeric order ID (must parse as int) |
| `Status` | `string` | Normalised WooCommerce status string |
| `ShippingAddress` | `*OrderSyncShippingAddress` | |
| `ShippingCharges` | `[]OrderSyncShippingCharge` | |
| `Items` | `[]OrderSyncItem` | |

---

## Phone Normalisation

`normalizePhone(raw)` converts any Colombian phone format to `+57XXXXXXXXXX`:

| Input | Output |
|-------|--------|
| `"+57 310 123 4567"` | `"+573101234567"` |
| `"57 310 123 4567"` | `"+573101234567"` |
| `"310 123 4567"` | `"+573101234567"` |
| `"3101234567"` | `"+573101234567"` |

1. Strip spaces and `+`.
2. Strip leading `57` if present.
3. Prepend `+57`.

---

## City Code Resolution

`citycode.Resolve(cityName)` maps a human-readable Colombian city name to its 5-digit DANE
code using an internal lookup map. Examples:

| Input | Output |
|-------|--------|
| `"Bogotá"` / `"bogota"` | `"11001"` |
| `"Medellín"` | `"05001"` |
| `"Cali"` | `"76001"` |
| _(unrecognised)_ | `""` (empty string) |

Unresolved city names result in a blank `CityCode`. The sync still proceeds — the city code
is not required for upsert.
