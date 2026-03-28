# Falabella — Realm Attributes

When a product is synced to Falabella, the module locates the datasheet whose `Realm` matches
`FALABELLA_PRODUCT_REALM` (default `"falabella"`) and reads its `Attributes` map. The keys below
are the exact strings the mapper looks for.

---

## Required Attributes

The following attributes **must** be present in the `"falabella"` datasheet for a product create
or update to succeed. Missing required values will cause the sync to fail at the mapping stage
before any API call is made.

| Attribute Key | Type | Description |
|--------------|------|-------------|
| `Name` | `string` | Product title shown on Falabella (max 255 chars) |
| `Description` | `string` | Long-form product description |
| `Stock` | `int` | Available inventory quantity |
| `Status` | `string` | `"active"` or `"inactive"` |

---

## Strongly Recommended Attributes

These attributes have fallback values but should be set explicitly for listing quality.

| Attribute Key | Type | Fallback | Description |
|--------------|------|----------|-------------|
| `Brand` | `string` | `"GENERIC"` | Brand name (must match a Falabella-approved brand) |
| `Model` | `string` | `""` | Manufacturer model number |
| `OperatorCode` | `string` | `FALABELLA_PRODUCT_OPERATOR_CODE` env value | Business-unit operator code (e.g. `"FACO"`) |

---

## Pricing Attributes

| Attribute Key | Type | Description |
|--------------|------|-------------|
| `PriceFalabella` | `float64` | Regular price on Falabella |
| `SalePriceFalabella` | `float64` | Promotional/sale price (optional) |
| `SaleStartDateFalabella` | `string` | RFC3339 sale start date |
| `SaleEndDateFalabella` | `string` | RFC3339 sale end date |

When `SalePriceFalabella` is set alongside the date range, Falabella applies the sale price
automatically within that window.

---

## Tax Attribute

| Attribute Key | Type | Description |
|--------------|------|-------------|
| `TaxClass` | `string` | Falabella tax class code (e.g. `"IVA19"`, `"EXEMPT"`) |

---

## Extra ProductData Attributes

Any key in `Attributes` that is not one of the reserved keys listed above is passed through to
Falabella as an extra `ProductData` attribute in the XML payload. This is the extension point for
category-specific mandatory attributes that Falabella requires (e.g. `"Género"`, `"Talla"`,
`"Material"`).

Example:

```json
{
  "realm": "falabella",
  "name": "Camiseta Deportiva",
  "description": "Camiseta de alto rendimiento.",
  "attributes": {
    "Brand": "Flock Sport",
    "Model": "FS-001",
    "TaxClass": "IVA19",
    "PriceFalabella": 89900,
    "Stock": 50,
    "Status": "active",
    "Género": "Hombre",
    "Material": "Polyester 100%"
  }
}
```

---

## Variant-Scoped Attributes

For products with variants, variant-specific attribute values can be stored in the parent product's
datasheet using the key notation `"<variantSKU>.<attributeKey>"`:

```json
{
  "attributes": {
    "SHIRT-RED-M.color": "Rojo",
    "SHIRT-RED-M.size": "M",
    "SHIRT-BLUE-M.color": "Azul",
    "SHIRT-BLUE-M.size": "M"
  }
}
```

The mapper extracts these keys for each variant during the variant expansion step of the sync
pipeline, applying them as variant-level `ProductData` entries in the Falabella XML request.

---

## Global Falabella Identifiers

These values are set centrally via environment variables and apply to all products synced to
Falabella. They are not stored per-product.

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `FALABELLA_PRODUCT_CATEGORY_ID` | `1638` | Falabella category ID |
| `FALABELLA_PRODUCT_GLOBAL_IDENTIFIER` | `G08010305` | Global product identifier |
| `FALABELLA_PRODUCT_ATTRIBUTE_SET_ID` | `5` | Attribute set ID |
| `FALABELLA_PRODUCT_OPERATOR_CODE` | `FACO` | Operator / business-unit code |
