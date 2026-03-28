# Contacts — Domain Model

## Contact

The `Contact` struct is the root aggregate of the contacts domain.

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | UUID primary key |
| `DocumentType` | `DocumentType` | Legal identity document type |
| `DocumentNumber` | `string` | Document number matching the type |
| `LegalName` | `string` | Full legal/business name (mutually exclusive with `FirstName`+`LastName`) |
| `FirstName` | `string` | Person given name |
| `LastName` | `string` | Person family name |
| `Email` | `string` | Primary email address (unique, required) |
| `Phone` | `string` | Phone number |
| `Address` | `string` | Street address line 1 |
| `AddressExtra` | `string` | Address line 2 / apartment / suite |
| `CityCode` | `string` | City identifier |
| `Metadata` | `map[string]string` | Freeform key/value pairs (key ≤ 128 chars, value ≤ 2 048 chars) |
| `CreatedAt` | `time.Time` | Record creation timestamp |
| `UpdatedAt` | `time.Time` | Record last-updated timestamp |

### Document Types

| Value | Description |
|-------|-------------|
| `CC` | Cédula de ciudadanía |
| `CE` | Cédula de extranjería |
| `TI` | Tarjeta de identidad |
| `PAS` | Passport |
| `NIT` | Número de Identificación Tributaria |
| `OTHER` | Other / unspecified |

---

## Business Invariants

### Validation (`Contact.Validate()`)

The following rules are enforced before any create or update reaches the database:

1. `Email` is required.
2. A contact must have **either** a `LegalName` **or** both `FirstName` and `LastName` — never a mix,
   never all empty. Providing `LegalName` alongside `FirstName`/`LastName` is rejected.

### Normalisation (`Contact.Normalize()`)

Before persistence all string fields are trimmed of leading/trailing whitespace. Metadata keys
are sorted and trimmed to ensure deterministic storage and comparison.

### Uniqueness Constraints

The persistence layer enforces:

| Constraint | Error |
|-----------|-------|
| One contact per `Email` | `ErrDuplicateEmail` |
| One contact per `(DocumentType, DocumentNumber)` pair | `ErrDuplicateDocument` |

`ErrNotFound` is returned by repository methods when a lookup by ID yields no result.

---

## Metadata

The `Metadata` map is a general-purpose extension point. Any external system can attach arbitrary
key/value data to a contact (e.g. WooCommerce customer ID, Logto user ID, RFM segment label)
without requiring schema changes.

Storage: a dedicated child table `contact_metadata (id, contact_id, key, value)` with a composite
unique index `idx_contacts_metadata_contact_key(contact_id, key)`. Querying by `metadataKey` +
`metadataValue` in the list endpoint is fully indexed.

---

## Port Layer

### `port.Repository`

All persistence operations go through the `port.Repository` interface, keeping domain and adapter
layers decoupled.

```go
Create(ctx context.Context, c *domain.Contact) error
GetByID(ctx context.Context, id string) (*domain.Contact, error)
List(ctx context.Context, q ListQuery) ([]domain.Contact, int64, error)
Update(ctx context.Context, c *domain.Contact) error
Delete(ctx context.Context, id string) error
```

### `ListQuery`

| Field | Type | Description |
|-------|------|-------------|
| `Page` | `int` | 1-based page number |
| `Limit` | `int` | Records per page |
| `OrderBy` | `string` | Column to sort by |
| `OrderDir` | `string` | `"asc"` or `"desc"` |
| `Email` | `string` | Exact or partial email filter |
| `DocumentType` | `string` | Filter by document type |
| `DocumentNumber` | `string` | Exact document number filter |
| `ExcludeIDs` | `[]string` | IDs to exclude from results |
| `MetadataKey` | `string` | Filter contacts that have this metadata key |
| `MetadataValue` | `string` | Combined with `MetadataKey`: require exact value |
