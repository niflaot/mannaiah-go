# Changelog

This file is the release source of truth for:
- version bump locations in code/docs
- Docker image publication rules
- version history registry

## Release Workflow

### 1) Update version references
Update all of these when releasing `vX.Y.Z`:
- `.env.example`: `TELEMETRY_SERVICE_VERSION=vX.Y.Z`
- `module/core/telemetry/config.go`: `defaultServiceVersion` and `Config.ServiceVersion` default tag
- `module/core/cmd/api/main.go`: Swagger document version (`"X.Y.Z"`)
- `module/core/startup/runtime.go`: `CoreSpec()` OpenAPI version (`"X.Y.Z"`)
- `README.md`: latest badge (`latest-vX.Y.Z`)
- `module/woocommerce/README.md`: latest badge (`latest-vX.Y.Z`)

### 2) Commit and merge
- Commit version changes on your release branch.
- Merge the release branch to `main`.

### 3) Create and push release tag
- Create annotated tag:
  - `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
- Push main and tag:
  - `git push origin main`
  - `git push origin vX.Y.Z`

## Docker Publish Rules (.drone.yml)

- Docker publish runs on:
  - `push` to `main` (`latest` + `${DRONE_COMMIT_SHA}`)
  - `tag` events (`${DRONE_TAG}` + `${DRONE_COMMIT_SHA}`)
- `.drone.yml` does not hardcode release version numbers.
- Repository target:
  - `docker.momlesstomato.dev/fl-docker/mannaiah-go`

## Image Acceptance Criteria

A new release image is accepted only if all are true:
1. Git tag `vX.Y.Z` exists in remote (`origin`).
2. Drone `validate` pipeline passed for the tagged commit.
3. Drone `docker-publish` pipeline passed for the tag event.
4. Nexus contains:
   - `docker.momlesstomato.dev/fl-docker/mannaiah-go:vX.Y.Z`
   - `docker.momlesstomato.dev/fl-docker/mannaiah-go:<commit-sha>`
5. Pull smoke test succeeds:
   - `docker pull docker.momlesstomato.dev/fl-docker/mannaiah-go:vX.Y.Z`

## Version Registry

Keep newest entries on top. Add one section per version.

### [v1.0.0] - 2026-03-24
- Shipping quotation: discounted freight-cost fields added to quotation result.
  - `fullFreightCost`, `discountPercent`, `discountedFreightCost` added to `QuotationResult` and `port.QuotationRecord`.
  - `SHIPPING_QUOTATION_DISCOUNT_PERCENT` env var configures the discount percentage applied at quotation time.
  - `quotationModel` stores `full_freight_cost`, `discount_percent`, `discounted_freight_cost`; migration `000023_shipping_quotation_discount` (MySQL + SQLite).
- Shipping TCC: dual access tokens (sandbox / production) + configurable COD fee.
  - `SHIPPING_TCC_SANDBOX_ACCESS_TOKEN` and `SHIPPING_TCC_PRODUCTION_ACCESS_TOKEN` replace the single token env var.
  - `SHIPPING_TCC_COD_FEE_PERCENT` configures the COD surcharge percentage for TCC; applied in quotation and mark generation.
  - `collectOnDeliveryFeePercent`, `collectOnDeliveryFeeAmount`, `collectOnDeliveryChargedAmount` added to `QuotationResult` and `ShippingMark`.
  - `collect_on_delivery_fee_percent` and `collect_on_delivery_charged_amount` columns added; migration `000024_shipping_mark_cod_fee` (MySQL + SQLite).
  - `quotation_cod_fee_percent`, `quotation_cod_fee_amount`, `quotation_cod_charged_amount` columns added to `shipping_quotations`; migration `000025_shipping_quotation_cod_fee` (MySQL + SQLite).
- Shipping: `shipmentMode` required field added to quotation, mark, and draft-mark requests.
  - Valid values: `parcel` (TCC business unit 1) and `express` (TCC business unit 2).
  - `SHIPPING_TCC_BUSINESS_UNIT` global env var removed; mode is now per-request.
  - `ErrInvalidShipmentMode` domain error added and mapped to HTTP 400 `invalid_payload`.
  - `shipment_mode VARCHAR(16) NOT NULL DEFAULT 'parcel'` column added to `shipping_marks`; migration `000029_shipping_mark_shipment_mode` (MySQL + SQLite).
  - OpenAPI spec updated: `shipmentMode` enum (`parcel`/`express`) added to quotation request, mark request, draft mark request, and `shippingMark` response schemas.
- Docker DNS: added `dns: [8.8.8.8, 1.1.1.1]` to `docker-compose.yml` mannaiah service to ensure Go's pure-Go DNS resolver can reach TCC's Oracle WAAS endpoint (`somos.tcc.com.co`).

### [v1.0.0] - 2026-03-23
- Release train reset:
  - New tag baseline starts again at `v1.0.0`.
- New shipping module (`module/shipping`) added with DDD + hexagonal structure:
  - Quotation flow (`POST/GET /shipping/quotations`).
  - Shipping mark flow (`POST /shipping/marks`, `PATCH /shipping/marks/{id}/void`, list/get endpoints).
  - Dispatch batch flow (`POST /shipping/batches`, add/remove marks, close batch, list/get endpoints).
  - Tracking flow (`GET /shipping/tracking/{trackingNumber}`).
  - Carrier catalog flow (`GET /shipping/carriers`, `GET /shipping/carriers/{id}`).
  - Carrier adapters:
    - `tcc` (quotation, mark generation, tracking mapping aligned to TCC plugin payload shapes).
    - `manual` fallback provider.
  - Shipping integration events:
    - `shipping.v1.mark.generated`
    - `shipping.v1.mark.failed`
    - `shipping.v1.mark.voided`
    - `shipping.v1.batch.created`
    - `shipping.v1.batch.closed`
    - `shipping.v1.tracking.updated`
- Database migrations added for shipping persistence:
  - MySQL + SQLite `000022_shipping_schema` (`dispatch_batches`, `shipping_marks`, `shipping_mark_units`, `shipping_quotations`).
- Runtime/bootstrap integration:
  - Core startup now loads `shipping.Config`, initializes module, authorizer, and registers routes/spec.
  - Workspace/build integration updated (`go.work`, root/core `go.mod`, `.drone.yml` module sweep).
- TCC carrier contract updates:
  - Base URLs are hardcoded by mode (`SHIPPING_TCC_SANDBOX=true|false`):
    - sandbox: `https://testsomos.tcc.com.co`
    - production: `https://somos.tcc.com.co`
  - Guide generation endpoint switched to `/api/clientes/remesas/grabardespacho7`.
  - Tracking request/response mapping aligned with `consultarestatusremesasv3` (`remesas[]` + `respuesta`).
- Docs and release metadata updates:
  - Root `README.md` and `module/woocommerce/README.md` latest badge set to `v1.0.0`.
  - Core OpenAPI version references set to `1.0.0` (`module/core/cmd/api/main.go`, `module/core/startup/runtime.go`).
  - Telemetry default service version set to `v1.0.0` (`module/core/telemetry/config.go`, `.env.example`).
- Shipping dispatch batch: `name` field removed; `created_by` field added.
  - Migration `000027_dispatch_batch_created_by` (MySQL + SQLite): drops `name`, adds `created_by VARCHAR(255) NOT NULL DEFAULT 'system'` on `dispatch_batches`.
  - `DispatchBatch` domain, `CreateBatchCommand`, store model, store mapping, and event builder updated: `Name` removed, `CreatedBy` added.
  - HTTP `POST /shipping/batches` now derives `created_by` from the JWT token owner (`Claims.Subject`), or `"system"` for dev-bypass credentials or internal callers.
  - `Authorizer` interface extended with `Subject()` method; `auth.runtime.Module` implements it (returns `"system"` for dev-admin or auth errors).
  - OpenAPI `DispatchBatch` response schema and `BatchCreate` request schema updated accordingly.
- Shipping draft mark workflow added:
  - New `MarkStatus` values: `QUOTED`, `CREATED`, `REMOVED`.
  - Draft marks are created with `QUOTED` status in a batch; real carrier submission only occurs at batch close.
  - `POST /shipping/batches/{id}/marks` now accepts a full mark payload (sender, recipient, units, optional quotation reference and quoted freight cost) and returns a `ShippingMark` with `status: QUOTED`.
  - `PATCH /shipping/batches/{id}/close` materializes all `QUOTED` marks via the carrier provider before closing; failed marks transition to `FAILED` status without blocking the close.
  - A JSON snapshot of the mark is captured immediately before carrier submission and stored in `draftSnapshot` for before/after audit trail.
  - `Void()` changed to soft-only: no carrier API call is made; status is updated locally only.
  - `MarkMaterializer` port defined in dispatch service package to avoid circular imports with mark service.
  - `ErrMarkNotDraft` domain error added for removal of non-QUOTED marks.
  - Migration `000028_shipping_mark_draft` (MySQL + SQLite): adds `quotation_id VARCHAR(64) NULL`, `quoted_freight_cost DECIMAL(15,2) NOT NULL DEFAULT 0`, `draft_snapshot TEXT NOT NULL DEFAULT ''` on `shipping_marks`.
  - OpenAPI spec updated: `draftMarkRequest` schema, `shippingMark` extended with `quotationId`, `quotedFreightCost`, `draftSnapshot`.
- Orders module: `payment_method` field added to order records.
  - Migration `000026_order_payment_method` (MySQL + SQLite): `payment_method VARCHAR(128) NOT NULL DEFAULT ''` on `orders` table.
  - `Order` domain, `CreateCommand`, repository mapper, and HTTP `POST /orders` handler accept and persist payment method.
  - OpenAPI `Order` response schema and `OrderCreate` request schema include `paymentMethod`.
  - WooCommerce sync chain propagates `payment_method` end-to-end (raw decode, SDK path, `OrderSyncCommand`, `mapOrderToCommand`, `toCreateCommand`).
