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

### [v3.0.0] - 2026-05-07
- Removed the WooCommerce integration module from runtime, workspace, docs, route registration, and CI validation.
- Kept historical sync-record querying for existing `woocommerce.contacts` and `woocommerce.orders` runs, plus historical order `realm`/`source` values for origin visibility.
- Shopify inbound contact and order sync now resolves Shopify address city names into Mannaiah municipality city codes using the shared core city-code resolver.
- Release metadata updated to `v3.0.0`.

### [v2.0.0] - 2026-04-30
- Removed the legacy marketing and commerce surface:
  - deleted `module/campaign`, `module/coupons`, `module/segment`, `module/storefront`, and `module/woocommerce`
  - removed their API routes, bootstrap wiring, workspace entries, and CI coverage
  - removed WooCommerce/storefront/CRM-focused root E2E coverage that no longer applies
- Orders now retain coupon usage as direct order metadata:
  - added `couponCode`, `couponDiscountAmount`, and `couponDiscountType` to the order model and HTTP/OpenAPI surface
  - removed the `order_applied_coupons` runtime dependency
- Analytics was reduced to BI infrastructure only:
  - removed campaign-event ingestion
  - removed CRM-facing OpenAPI routes from the module surface
  - retained ClickHouse schema bootstrap plus fact-table ingestion for contacts, orders, order items, membership events, and taxonomy data
- Release metadata updated to `v2.0.0`.

### [v1.4.0] - 2026-04-14
- Storefront module: renderable and static-page management feature added (`module/storefront`).
  - Domain: `Renderable` root rows hold the current draft snapshot (`kind`, metadata JSON, content JSON, `draft` state), while immutable `RenderableVersion` rows are created only on publish and rollback.
  - Versioning: publish creates timestamped immutable snapshots; rollback clones a historical version into a fresh published version with a new timestamp and source-version linkage.
  - Static pages: one-to-one bindings from pages to renderables with title, unique URL, and SEO-tags JSON managed separately from renderable version history.
  - HTTP adapter: CRUD and listing endpoints added under `/storefront/renderable*` and `/storefront/page*`, all protected by `storefront:manage`.
  - OpenAPI: request/response schemas and path documentation added for renderables, published versions, rollbacks, and static pages.
  - Persistence: migration `000046_storefront_renderables` added for MySQL and SQLite with indexed renderable roots, published snapshots, and static-page bindings.
  - Performance/storage: draft edits remain on the root renderable row; only published or rollback snapshots create version rows, with compacted JSON payload persistence and indexed version retrieval.
  - Validation: added unit coverage for renderable publish/rollback and static-page conflicts, handler coverage for permission enforcement and payload decoding, plus end-to-end lifecycle coverage for renderables and static pages.
- Release metadata updated to `v1.4.0`.

### [v1.3.0] - 2026-04-12
- Coupons module: full coupon management feature added (`module/coupons`).
  - Domain: `Coupon` aggregate with fixed/percentage discount types, per-email and global usage limits, expiry, assigned emails/contacts, product/category/tag scope, and WooCommerce deduplication key.
  - Application: `Service` with create, update, delete, get-by-id/code/woocommerce-id, list, and record-usage use-cases; integration events published on every mutation and redemption.
  - HTTP adapter: REST endpoints under `/coupons` for CRUD and `/coupons/:id/usage` for redemption recording; migrated to `corehttp.Context` with structured `AppError` responses.
  - Store adapter: GORM-backed repository with PostgreSQL JSONB columns for list fields, soft-delete, and coupon-usage table.
  - Event adapter: Watermill bus publisher bridge.
  - OpenAPI spec: coupon endpoints documented under the `Coupons` tag with full request/response schemas.
  - WooCommerce coupon sync: `SyncCoupons` cron job pages WooCommerce coupons and upserts them via `CouponSyncTarget`; `client_coupons.go` implements `ListCoupons`, `GetCouponByID`, and `UpsertCoupon`; `couponWooSyncAdapter` bridges the woo sync target to the coupons service.
  - Order sync: `coupon_lines` now extracted from raw WooCommerce order payloads and stored as `AppliedCoupon` records on orders via `order_applied_coupons` join table.
  - DB migrations: `000044` (coupons + coupon_usages tables) and `000045` (order_applied_coupons table).
  - New env vars: `WOOCOMMERCE_SYNC_COUPONS`, `WOOCOMMERCE_SYNC_COUPONS_CRON`.
- Release metadata updated to `v1.3.0`.

### [v1.2.0] - 2026-04-08
- Shipping manual flow: operator-entered tracking details can now be staged in manual batches without quotation-derived payloads.
  - `POST /shipping/batches/{id}/marks` already accepts `trackingNumber`, `customTrackingUrl`, and `observations`; manual batches now also tolerate sparse payloads by defaulting shipment mode and a placeholder package unit server-side.
  - Manual carrier labels stored in `observations` are now preferred in transactional shipping emails, while structured carriers keep their existing lookup behavior.
- Shipping rotulus: the on-demand label now renders in the top half of a letter sheet with a dynamic `Pedido #...` title, a non-cropped signed QR, and expanded recipient shipping fields (`address`, `address 2`, `phone`, `city`) sourced from the order adapter when available.
- Shipping rotulus: city codes are now rendered as municipality names from the embedded city dataset, the title is uppercase, and label prefixes render in uppercase bold while their values remain normal weight.
- Release metadata updated to `v1.2.0`.

### [v1.0.0] - 2026-03-26
- Shipping batches: merged manifest print document added.
  - New endpoint: `GET /shipping/batches/{id}/manifest-document`.
  - Document is generated on-demand for closed batches and cached in-memory for `5` minutes.
  - Output is `application/pdf` in letter format with:
    - cover summary page (logo + batch metadata: id, generation timestamp, carrier, quantity)
    - detail table headers: `tracking number`, `recipient`, `order #`, `city`, `items`
    - merged carrier manifest PDF pages appended after the cover page.
  - Manifest fetch failures are handled per-document (skipped with warning) without failing full output.
  - Optional order-summary resolver wired from core composition root to use order public identifiers and item labels in cover rows.

### [v1.0.0] - 2026-03-26
- Shipping marks: carrier manifest persistence added without breaking batch close on missing manifest.
  - `ShippingMark` now stores both artifact pairs:
    - mark document: `documentType` / `documentRef`
    - shipping manifest: `manifestType` / `manifestRef`
  - TCC adapter mapping updated:
    - main mark document URL resolved from label/guide fields (`urlrotulos`/`urlguia`) and is now required for success.
    - manifest URL resolved from `urlrelacionenvio` and is optional (missing manifest does not fail mark materialization).
  - Store mapping and update paths now persist manifest fields.
  - OpenAPI schema for shipping marks updated to expose `manifestType` and `manifestRef`.
  - Migration `000033_shipping_mark_manifest` added (MySQL + SQLite): `manifest_type`, `manifest_ref` on `shipping_marks`.

### [v1.0.0] - 2026-03-26
- Email API: recipient-delivery listing endpoint added.
  - `GET /email/deliveries?email=<recipient_email>` returns all deliveries sent to the provided email, ordered by newest first.
  - Email service/repository contracts and store implementation updated for recipient-email filtering.
  - Email HTTP adapter, OpenAPI path/spec, and unit test coverage updated.

### [v1.0.0] - 2026-03-25
- Shipping API: related-marks endpoint added.
  - `GET /shipping/marks/{id}/related` returns marks related by `orderId` and/or `dispatchBatchId` (excluding self), sorted by newest first.
  - Shipping HTTP adapter, service, OpenAPI path/spec, and tests updated.
- Shipping event integration: mark-generated order auto-completion added in core runtime.
  - New consumer on `shipping.v1.mark.generated` resolves `orderId` and appends `COMPLETED` status with source `shipping_mark_generated`.
  - Completion update flows through existing order integration events (`orders.v1.status.updated`) to keep downstream consumers decoupled.
- WooCommerce mainstream update path: status propagation added.
  - `MainstreamOrderUpdateCommand` now includes `status`.
  - Woo adapter raw update payload now sets Woo order `status` when provided.
  - Mapping added from mainstream/domain statuses to Woo statuses (`created->processing`, `hold->on-hold`, `completed->completed`, etc.).
  - Unit and e2e coverage extended to verify completed-status propagation.
- Transactional shipping email flow added on shipping-mark generation.
  - New embedded template folder: `module/core/cmd/api/transactional/templates/`.
  - New template: `shipping_dispatched.html.tmpl`, rendered with the same Go-template renderer used for campaign templates.
  - New core consumer listens to `shipping.v1.mark.generated` and sends a transactional email with idempotency key `shipping_mark_dispatched:<markId>`.
  - Template data includes shipping number, public order number, carrier/tracking values, tracking CTA (`https://rastreo.flockstore.co`), WhatsApp help CTA, billing/shipping fallbacks, payment method, and ordered items.
  - Product image selection prefers default-realm gallery assets and variation-specific images when SKU/variation mapping is available.

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
- Shipping: mark `failureReason` field added — carrier error message is now persisted on marks that transition to `FAILED` status (both standalone `Generate` and batch-close `Materialize` paths). Exposed in `ShippingMark` response and OpenAPI spec.
  - Migration `000030_shipping_mark_failure_reason` (MySQL + SQLite): `failure_reason TEXT NOT NULL DEFAULT ''` on `shipping_marks`.
- Shipping: quotation expiration enforcement added.
  - `SHIPPING_QUOTATION_TTL_HOURS` env var (default: `24`) controls how long quotations remain valid.
  - `ExpiresAt` is now always set at quote time when the provider does not supply one.
  - Background cleanup goroutine (1h interval) purges expired quotation rows automatically.
  - `QuotationRepository.DeleteExpired` port method and store implementation added.
- Shipping: batch-close materialization errors are now logged (`mark_id`, `order_id`, `error`) via zap instead of being silently discarded.
- Shipping quotation: freight discount removed.
  - `SHIPPING_QUOTATION_DISCOUNT_PERCENT` env var removed; prices are used as-is from the carrier.
  - `fullFreightCost`, `discountPercent`, `discountedFreightCost` removed from `QuotationResult`, `QuotationRecord`, quotation response schema, and quotation list schema.
  - `full_freight_cost`, `discount_percent`, `discounted_freight_cost` columns dropped from `shipping_quotations`; migration `000031_shipping_quotation_remove_discount` (MySQL + SQLite).
  - `normalizeDiscountPercent` and `applyDiscount` helpers removed from quotation service.
- Shipping TCC: per-mode account numbers + configurable unit declaration.
  - `SHIPPING_TCC_ACCOUNT_NUMBER` split into `SHIPPING_TCC_PARCEL_ACCOUNT_NUMBER` and `SHIPPING_TCC_EXPRESS_ACCOUNT_NUMBER`; TCC `cuentaremitente` is now selected per shipment mode at both quotation and dispatch time.
  - `SHIPPING_TCC_DECLARATION` env var added as default `dicecontener` (unit contents description); per-unit `description` field takes precedence when provided.
- Shipping TCC: structured logging added to all API calls.
  - Request body logged at debug level before each outbound call.
  - HTTP errors, read failures, decode failures, empty responses, and carrier rejections all emit structured `zap.Error` events with path, status, latency, and body.
  - Success path logs at info with status and latency.
  - `zap.ReplaceGlobals(logger)` wired in `main.go` so global logger is live at startup.
- Shipping TCC: `ParseResultCode("")` now returns 0 (success) — TCC returns HTTP 200 with empty `codigoresultado` on successful dispatch; was incorrectly treated as rejection.
- Shipping TCC: dispatch rejection error now falls back to `remesas[0].mensajeresultado` when top-level `mensajeresultado` is empty.
- Shipping batch: fixed 404 on `POST /shipping/batches/{id}/marks` caused by MySQL returning `RowsAffected=0` when `AddMark` tried to UPDATE `dispatch_batch_id` to its already-set value; mark is now created without `dispatch_batch_id` so the update performs a real NULL→value transition.
- Shipping: `GET /shipping/orders/{orderID}/dispatch` utility endpoint added to check order dispatch provisioning status.
  - Returns `orderId`, `provisioned` (bool), `markId`, `batchId`, `status` for the highest-priority active mark of the order.
  - Priority: `QUOTED` > `CREATED` > `GENERATED`; `VOIDED`, `REMOVED`, `FAILED`, `PENDING` marks are excluded.
  - OpenAPI spec updated: `orderDispatch` response schema and path registered under `/shipping/orders/{orderID}/dispatch`.

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
