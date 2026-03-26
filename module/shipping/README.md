# Shipping Module

![Latest Version](https://img.shields.io/badge/latest-v1.0.0-0A66C2)

`module/shipping` provides carrier-agnostic shipping capabilities (quotation, mark generation, batch dispatch grouping, and normalized tracking) under DDD + hexagonal boundaries.

## Scope
- Carrier adapter registry (`tcc`, `manual`).
- Shipping quotation orchestration with audit persistence.
- Shipping mark generation and void flows.
- Dispatch batch grouping (create/add/remove/close).
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
  - `POST /shipping/marks`, `GET /shipping/marks/:id`, `GET /shipping/marks/:id/related`, `GET /shipping/marks`, `PATCH /shipping/marks/:id/void`
  - `POST /shipping/batches`, `GET /shipping/batches/:id`, `GET /shipping/batches`, `POST /shipping/batches/:id/marks`, `DELETE /shipping/batches/:id/marks/:markID`, `PATCH /shipping/batches/:id/close`
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

## Quotation Discount
- `SHIPPING_QUOTATION_DISCOUNT_PERCENT` applies a global percentage discount to all carrier quotations.
- Quotation responses expose:
  - `fullFreightCost` (carrier value before discount)
  - `discountPercent` (configured percent)
  - `discountedFreightCost` (value after discount)
  - `freightCost` (compatibility alias of `discountedFreightCost`)

## COD Collection
- Mark create requests accept `collectOnDeliveryAmount`.
- Mark responses expose:
  - `collectOnDeliveryAmount` (requested amount)
  - `collectOnDeliveryFeePercent` (carrier-applied fee percent)
  - `collectOnDeliveryChargedAmount` (amount sent to carrier)
- Quotation create requests also accept `collectOnDeliveryAmount`.
- Quotation responses expose:
  - `collectOnDeliveryAmount` (requested amount)
  - `collectOnDeliveryFeePercent` (carrier-applied fee percent)
  - `collectOnDeliveryFeeAmount` (carrier-applied fee amount)
  - `collectOnDeliveryChargedAmount` (amount projected to carrier)
