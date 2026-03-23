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
  - `POST /shipping/marks`, `GET /shipping/marks/:id`, `GET /shipping/marks`, `PATCH /shipping/marks/:id/void`
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
