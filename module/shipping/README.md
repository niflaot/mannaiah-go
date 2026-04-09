# Shipping Module

![Latest Version](https://img.shields.io/badge/latest-v1.0.0-0A66C2)

`module/shipping` provides carrier-agnostic shipping capabilities (quotation, mark generation, batch dispatch grouping, and normalized tracking) under DDD + hexagonal boundaries.

## Scope
- Carrier adapter registry (`tcc`, `manual`).
- Shipping quotation orchestration with audit persistence.
- Shipping mark generation and void flows.
- Carrier artifact persistence for both shipping mark documents and shipping manifests.
- Dispatch batch grouping (create/add/remove/close).
- On-demand merged batch manifest PDF generation (cover summary page + carrier manifest pages) with 5-minute cache (Redis when configured + in-memory fallback).
- JSON-based batch-manifest cover template for editable labels/headers without code changes.
- Homogenized tracking API.
- Integration event publication for mark/batch/tracking lifecycle updates.

## Architecture
- `runtime/`: composition root wiring.
- `domain/`: shipping aggregates/value objects.
- `port/`: repository/provider/event contracts.
- `application/`: use-case orchestration namespaces.
- `adapter/store`: GORM repository implementations.
- `adapter/carrier`: carrier providers and registry.
- `adapter/http`: protected HTTP endpoints and OpenAPI path builders.
- `adapter/event`: core bus publisher adapter.

## Key Methods / Endpoints / Events
- Methods:
  - `shipping.New(cfg, db, publishers...)`
  - `(*shipping.Module).Load(loader)`
  - `(*shipping.Module).SetAuthorizer(authorizer)`
- Endpoints:
  - `POST /shipping/quotations`, `GET /shipping/quotations`
  - `POST /shipping/quotations/order` — auto-build packages from order products and request a quotation
  - `POST /shipping/quotations/order-packaging` — preview package allocation, COD value, destination city, and resolved shipment mode without carrier calls or persistence
  - `GET /shipping/quotations/order/:identifier?carrierId={carrierID}` — retrieve the latest non-expired quotation for an order and carrier
  - `POST /shipping/marks`, `GET /shipping/marks/:id`, `GET /shipping/marks/:id/related`, `GET /shipping/marks`, `PATCH /shipping/marks/:id/void`
  - `POST /shipping/batches`, `GET /shipping/batches/:id`, `GET /shipping/batches`, `POST /shipping/batches/:id/marks`, `POST /shipping/batches/marks`, `DELETE /shipping/batches/:id/marks/:markID`, `PATCH /shipping/batches/:id/close`, `GET /shipping/batches/:id/manifest-document`
  - `GET /shipping/tracking/:trackingNumber?carrier={carrierID}`
  - `GET /shipping/carriers`, `GET /shipping/carriers/:id`
- Events:
  - `shipping.v1.mark.generated`
  - `shipping.v1.mark.failed`
  - `shipping.v1.mark.voided`
  - `shipping.v1.batch.created`
  - `shipping.v1.batch.closed`
  - `shipping.v1.tracking.updated`

## TCC Environment Switch
- `SHIPPING_TCC_ENABLED=true` enables the TCC carrier adapter.
- `SHIPPING_TCC_SANDBOX=true` routes requests to `https://testsomos.tcc.com.co`.
- `SHIPPING_TCC_SANDBOX=false` routes requests to `https://somos.tcc.com.co`.
- `SHIPPING_TCC_SANDBOX_ACCESS_TOKEN` is used for sandbox requests.
- `SHIPPING_TCC_PRODUCTION_ACCESS_TOKEN` is used for production requests.
- `SHIPPING_TCC_COD_FEE_PERCENT` adds a COD fee percent to requested collection amounts (`recaudoproducto`).

## Batch Manifest Document
- `SHIPPING_BATCH_MANIFEST_CACHE_TTL_SECONDS` controls merged manifest cache TTL (default `300`).
- `SHIPPING_BATCH_MANIFEST_TEMPLATE_PATH` optionally points to a JSON template file for cover-page labels/headers.
- Default template source: `application/dispatch/service/templates/batch_manifest_cover.es.json`.

## Quotation Discount
- `SHIPPING_QUOTATION_DISCOUNT_PERCENT` applies a global percentage discount to all carrier quotations.
- Quotation responses expose:
  - `fullFreightCost` (carrier value before discount)
  - `discountPercent` (configured percent)
  - `discountedFreightCost` (value after discount)
  - `freightCost` (compatibility alias of `discountedFreightCost`)

## Order-Based Quotation
`POST /shipping/quotations/order` auto-builds packages from order products using the overlapping box-packing algorithm:

**Request:**
```json
{
  "orderIdentifier": "<internal-id or WooCommerce order number>",
  "carrierId": "tcc",
  "originCityCode": "11001000",
  "shipmentMode": "parcel"
}
```

**Algorithm:**
1. Resolves order by internal ID or external identifier (WooCommerce number).
2. Reads product shipping attributes from the `"default"` realm datasheet for each line item SKU:
   - `pweight` → weight (kg), `pheight` → height (cm), `pwidth` → width (cm), `plength` → length/depth (cm)
   - `price` → declared value (defaults to `1` if missing), `overlapped` → packing flag (defaults to `true` if missing)
3. Skips products with any missing dimension.
4. Applies box-packing: non-overlapped items are standalone boxes; overlapped items are nested (max 3 per box).
5. If all products are overlapped, the largest becomes the main box (emits `ALL_OVERLAPPED` warning).
6. COD is resolved from the order payment method: COD methods map to total order value; non-COD methods map to `0`.

**Warnings** (non-fatal, included in response `warnings[]`):
| Code | Meaning |
|---|---|
| `NO_PRODUCTS` | No valid products found — returns an error |
| `ALL_OVERLAPPED` | Every product is flagged as overlapped; largest item promoted to main box |
| `INVALID_CITY` | Carrier rejected the destination city code |

**Quotation TTL:** `SHIPPING_QUOTATION_TTL_MINUTES` (default: `10` minutes).

**Get latest quotation for order:** `GET /shipping/quotations/order/:identifier?carrierId={carrierID}`
— Returns the most recent non-expired quotation for the given order and carrier, or `404` if none found.

**Preview packaging only:** `POST /shipping/quotations/order-packaging`
— Returns the auto-packed `units`, `declaredValue`, `collectOnDeliveryAmount`, `destCityCode`, and normalized `shipmentMode` (`express` for one unit, `parcel` for two or more) without calling carrier quotation APIs and without storing quotation rows.

## Batch Mark Creation Modes
- `POST /shipping/batches/marks` accepts `batch` (required), `quotationId` (required), and `direct` (optional, default `false`).
- `direct=false` (or omitted): creates a `QUOTED` draft mark in the target batch and requires the batch to be `OPEN`.
- `direct=true`: creates and materializes the mark immediately and assigns it to the target batch even if the batch is `CLOSED`.
- During `direct=true` materialization and during `PATCH /shipping/batches/:id/close`, carrier guardrails run immediately before outbound carrier dispatch.
- Guardrail violations return HTTP `500` (`message=shipping_guardrail_violation`) and include `mark_id`, `order_id`, `rule`, and `request_preview` in the `error` field.
- `POST /shipping/batches/:id/marks` accepts optional manual fields `trackingNumber` and `customTrackingUrl` so operators can attach manual guide references and custom tracking links before batch close/materialization.

## COD Collection
- Mark create requests accept `collectOnDeliveryAmount`.
- Mark responses expose:
  - `collectOnDeliveryAmount` (requested amount)
  - `collectOnDeliveryFeePercent` (carrier-applied fee percent)
  - `collectOnDeliveryChargedAmount` (amount sent to carrier)
  - `documentType` / `documentRef` (shipping mark artifact)
  - `manifestType` / `manifestRef` (shipping manifest artifact when provided by carrier)
- Quotation create requests also accept `collectOnDeliveryAmount`.
- Quotation responses expose:
  - `collectOnDeliveryAmount` (requested amount)
  - `collectOnDeliveryFeePercent` (carrier-applied fee percent)
  - `collectOnDeliveryFeeAmount` (carrier-applied fee amount)
  - `collectOnDeliveryChargedAmount` (amount projected to carrier)

## TCC Guardrails (Pre-Dispatch)
- Guardrails execute right before `tcc` dispatch API calls (`grabardespacho7`), both for:
  - batch close (`PATCH /shipping/batches/:id/close`) over quoted marks
  - direct batch mark creation (`POST /shipping/batches/marks` with `direct=true`)
- Validation input always includes:
  - the normalized order context (from the mark)
  - the exact outbound request preview (serialized JSON payload)

### COD orders
- `formapago` must be `"2"`.
- `recaudoproducto` must exist and be `> 0`.
- `totalvalorproducto` must exist and be `> 0`.

### Non-COD orders
- `formapago` must be `"1"`.
- `recaudoproducto` must not exist in JSON payload.
- `totalvalorproducto` must not exist in JSON payload.

### COD amount sent to TCC
- TCC collects freight + COD from recipient.
- To avoid over-collection, sent COD is netted:
  - `recaudoproducto = collectOnDeliveryChargedAmount - quotedFreightCost`
- If this net value is not positive, guardrail blocks dispatch.
