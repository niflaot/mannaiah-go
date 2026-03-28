# Falabella — Sync Pipeline

This document describes the end-to-end flow that runs when a product (or a batch of products) is
submitted for Falabella synchronization.

---

## Entry Points

| Function | Description |
|----------|-------------|
| `SyncProduct(ctx, id)` | Sync a single product by ID |
| `SyncProducts(ctx, ids)` | Sync a batch of products concurrently |

`SyncProducts` dispatches work across `FALABELLA_PRODUCT_SYNC_WORKERS` goroutines (default `4`)
and collects results into a `Summary`.

### Summary

```json
{
  "executionId": "uuid",
  "requested": 10,
  "synced": 8,
  "skipped": 1,
  "failed": 1,
  "results": [
    { "productId": "...", "sku": "SHIRT-001", "status": "synced" },
    { "productId": "...", "sku": "SHIRT-002", "status": "failed", "error": "..." }
  ]
}
```

---

## Pipeline Steps

```
1. Fetch product from ProductCatalog
        │
        ▼
2. Locate "falabella" realm datasheet
        │
        ▼
3. mapProduct() — build SyncProductRequest
        │
        ▼
4. Variant expansion — mapVariantProduct() per Variant
        │
        ▼
5. source.SyncProduct() → Seller Center API (ProductCreate / ProductUpdate)
        │                                    returns FeedID
        ▼
6. Record SyncEntry (step=product, status=pending)
        │
        ▼
7. waitForProductFeedResolution() — poll GetFeedStatus up to N times
        │            ┌────────────┐
        │            │  Pending  │◄─ sleep(backoff × attempt)
        │            └────────────┘
        │            ┌──────────────┐
        └────────────► Finished OK │
                     └──────────────┘
                             │
                             ▼
               8. syncImagesAfterProductFeedResolved()
                             │
                     ┌───────┴────────────────────────────────┐
                     │  ImageTranscodeEnabled?                 │
                     │  Yes → route URLs via transcode proxy  │
                     │  No  → use asset URLs directly         │
                     └───────┬────────────────────────────────┘
                             │
                             ▼
               9. source.SyncProductImages() → Seller Center API (Image feed)
                             │
                             ▼
              10. Record SyncEntry (step=image, status=pending)
```

---

## Step 3 — mapProduct()

The mapper reads the product's `"falabella"` realm datasheet and constructs a
`SyncProductRequest`:

- **`Name`**, **`Description`** — from datasheet.
- **`Brand`** — from `attributes["Brand"]`; fallback `"GENERIC"`.
- **`Model`**, **`TaxClass`** — from attributes.
- **`PriceFalabella`**, **`SalePriceFalabella`**, **`SaleStartDateFalabella`**,
  **`SaleEndDateFalabella`** — pricing window.
- **`OperatorCode`** — from attributes, fallback to `FALABELLA_PRODUCT_OPERATOR_CODE` env var.
- **`Stock`**, **`Status`** — listing availability.
- **`CategoryID`**, **`GlobalIdentifier`**, **`AttributeSetID`** — from Falabella config env vars.
- All remaining attribute keys → extra `ProductData` entries.

---

## Step 4 — Variant Expansion

For each `Variant` in the product:

1. `mapVariantProduct()` builds a child `SyncProductRequest` with:
   - `ParentSKU` set to the parent product SKU.
   - `SKU` set to the variant's own SKU (or parent SKU if not overridden).
   - Variant-scoped attributes extracted using the `"<variantSKU>.<key>"` notation from the
     parent's datasheet attributes.
2. The variant entry is appended to the parent request's `Variants` slice.

---

## Step 7 — Feed Resolution (Polling)

| Config Variable | Default | Description |
|----------------|---------|-------------|
| `FALABELLA_PRODUCT_FEED_RESOLUTION_ATTEMPTS` | `6` | Maximum poll attempts |
| `FALABELLA_PRODUCT_FEED_RESOLUTION_BACKOFF_MS` | `1000` | Initial backoff ms (doubles each attempt) |
| `FALABELLA_PRODUCT_FEED_RESOLUTION_REQUEST_TIMEOUT_MS` | `5000` | Per-request timeout |

If all attempts are exhausted without a `Finished` status, the sync entry is marked `failed`
and an error is returned. The `FALABELLA_PRODUCT_FEED_RESOLUTION_ATTEMPTS` cap is enforced at
a maximum of `30`; `BACKOFF_MS` is capped at `30 000 ms`.

---

## Create vs Update

The Falabella module determines whether to call `ProductCreate` or `ProductUpdate` by checking
whether the product has any existing `SyncEntry` records with `status=finished` for `step=product`.
If prior successful sync records exist, the action is `update`; otherwise it is `create`.

---

## Circuit Breaker

All calls to the Seller Center API go through a `circuitbreaker.Service`. When the breaker is
open, `SyncProduct` returns immediately with `ErrUnavailable`, and the result is recorded as
`skipped` in the batch summary. See [07-CONFIG.md](07-CONFIG.md) for breaker configuration.
