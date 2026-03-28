# Orders — Domain Model

---

## Order

The `Order` struct is the root aggregate of the orders domain. All sub-entities — items, status
history, comments, shipping — are owned by the order and never exist independently.

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Internal UUID |
| `Identifier` | `string` | External order reference (e.g. WooCommerce order number). Unique per `Realm`. |
| `Realm` | `string` | Source system namespace (e.g. `"woocommerce"`). Together with `Identifier` forms the global unique key. |
| `ContactID` | `string` | Linked `contacts` module customer |
| `Items` | `[]Item` | One or more order line items |
| `CurrentStatus` | `Status` | Latest status snapshot (derived from the last `StatusHistory` entry) |
| `StatusHistory` | `[]StatusEntry` | Append-only status change log — source of truth for order state |
| `Comments` | `[]Comment` | Append-only comment thread |
| `ShippingAddress` | `ShippingAddress` | Resolved delivery address |
| `HasCustomShippingAddress` | `bool` | When `false`, shipping address is automatically populated from the linked contact's billing address |
| `ShippingCharges` | `[]ShippingCharge` | Shipping method lines |
| `PaymentMethod` | `string` | Payment method token or label |
| `Metadata` | `map[string]string` | Freeform extension data (key ≤ 128, value ≤ 2 048 chars) |
| `CreatedAt` / `UpdatedAt` | `time.Time` | Timestamps |

---

## Item

Each line item represents one product SKU and quantity.

| Field | Type | Description |
|-------|------|-------------|
| `SKU` | `string` | Seller SKU from the source channel |
| `AlternateName` | `string` | Fallback lookup name (used when SKU alone does not resolve to a product) |
| `Quantity` | `int` | Must be > 0 |
| `Value` | `float64` | Unit price (≥ 0 after normalisation) |
| `ProductID` | `string` | Resolved Mannaiah product UUID (empty if unresolved) |
| `ResolutionSource` | `ItemResolutionSource` | How the product was matched |

### ItemResolutionSource

| Value | Description |
|-------|-------------|
| `"sku"` | Product matched by exact SKU lookup |
| `"alternate_name"` | SKU lookup failed; product matched by fallback alternate name |
| `"unresolved"` | No product match found — item order is tracked but not linked to PIM |

**Parallel resolution:** Up to 8 goroutines run simultaneously to resolve items, calling
`ProductResolver.Resolve(sku, alternateName)` per item. The fallback chain is: SKU → alternate name
→ unresolved. Every item is resolved regardless of success; the order is always persisted.

---

## ShippingAddress

| Field | Type | Description |
|-------|------|-------------|
| `Address` | `string` | Street address line 1 |
| `Address2` | `string` | Address line 2 / apartment |
| `Phone` | `string` | Contact phone for delivery |
| `CityCode` | `string` | City identifier |

**Billing fallback:** When `HasCustomShippingAddress = false`, the shipping address is
transparently populated from the linked contact's billing address on every read and write path
via `CustomerSource.GetByID`. The persisted `order_shipping_addresses` row is only created when
`HasCustomShippingAddress = true`.

---

## ShippingCharge

| Field | Type | Description |
|-------|------|-------------|
| `MethodID` | `string` | Carrier/method identifier |
| `MethodTitle` | `string` | Human-readable method name |
| `Price` | `float64` | Charge amount |

---

## Database Tables

| Table | Purpose |
|-------|---------|
| `orders` | Root: `id`, `identifier`, `realm`, `contact_id`, `payment_method`, timestamps |
| `order_items` | Line items: `position`, `sku`, `alternate_name`, `quantity`, `value`, `product_id`, `resolution_source` |
| `order_status_history` | Append-only status log: `position`, `status`, `author`, `description`, `note_owner`, `note`, `occurred_at` |
| `order_comments` | Comment thread: `author`, `comment`, `internal`, `occurred_at` |
| `order_shipping_addresses` | Optional 1:1 per order (only when `has_custom_shipping_address`) |
| `order_shipping_charges` | Shipping method lines: `position`, `method_id`, `method_title`, `price` |
| `order_metadata` | Freeform key-value per order |
| `order_item_metadata` | Freeform key-value per item |

**Key unique constraint:** `idx_orders_realm_identifier` on `(realm, identifier)` — guarantees
channel-scoped order ID uniqueness.

---

## Port Layer

### `port.Repository`

```go
Create(ctx context.Context, o *Order) error
Update(ctx context.Context, o *Order) error
GetByID(ctx context.Context, id string) (*Order, error)
List(ctx context.Context, q ListQuery) ([]Order, int64, error)
AppendStatus(ctx context.Context, id string, s StatusEntry) (*Order, error)
AppendComment(ctx context.Context, id string, c Comment) (*Order, error)
UpdateComment(ctx context.Context, id string, commentID string, c Comment) (*Order, error)
DeleteComment(ctx context.Context, id string, commentID string) (*Order, error)
```

`ListQuery` fields: `Page`, `Limit`, `Realm`, `ContactID`, `Identifier`, `Status`

Port errors: `ErrNotFound`, `ErrCommentNotFound`, `ErrDuplicateIdentifier`

### `port.CustomerSource`

```go
GetByID(ctx context.Context, id string) (*Customer, error)
```

`Customer` — `{ ID, Address, AddressExtra, Phone, CityCode string }`

### `port.ProductResolver`

```go
Resolve(ctx context.Context, sku string, alternateName string) (*ProductResolution, error)
```

`ProductResolution` — `{ ProductID string; MatchedBy string }` (`"sku"` or `"alternate_name"`)
