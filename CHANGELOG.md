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

### [v2.0.5] - 2026-03-14
- Membership consent model changes:
  - remove `POST /membership/migrate` route and OpenAPI path.
  - add `channel=all` consent flow support and default `/membership/optin`/`/membership/optout` to `all`.
  - `GET /membership/status/{contactId}` now returns channel-agnostic status payload (`contactId` + `statuses[]`).
  - status resolution now reads latest effective rows from `membership_stamps` only (no snapshot-table dependency).
- WooCommerce consent mapping changes:
  - Circle checker metadata is no longer persisted to Mannaiah contact metadata.
  - Circle checker still drives membership stamping (`opt_in`/`opt_out`) through `channel=all`.
  - Privacy checker metadata continues to persist on contact metadata with date fields.
- OpenAPI response documentation improvements for the 2.0+ modules:
  - `membership`, `analytics`, `segment`, `campaign`, and `syncrecord` now expose concrete JSON response schemas instead of generic `Success` descriptions.
- Schema rollout:
  - add migration `000015_membership_drop_status_snapshot` (MySQL + SQLite) to drop `membership_status` snapshot table and keep stamps as source of truth.
- Bump release references and badges to `v2.0.5`:
  - `.env.example`
  - `module/core/telemetry/config.go`
  - `module/core/cmd/api/main.go`
  - `module/core/startup/runtime.go`
  - updated module OpenAPI spec versions
  - `README.md` and `module/woocommerce/README.md`

### [v2.0.4] - 2026-03-14
- Fix startup regression from `v2.0.3` affecting root E2E startup flow:
  - `module/segment/runtime/config.go` default now sets `SEGMENT_ENABLED=false`.
  - `.env.example` now sets `SEGMENT_ENABLED=false`.
- Bump release references and badges to `v2.0.4`:
  - `.env.example`
  - `module/core/telemetry/config.go`
  - `module/core/cmd/api/main.go`
  - `module/core/startup/runtime.go`
  - module OpenAPI specs for analytics/membership/syncrecord/email/campaign/segment
  - `README.md` and `module/woocommerce/README.md`
- Add frontend integration guide copy `MANUAL-v2.0.4.md` and update root `README.md` link.

### [v2.0.3] - 2026-03-14
- Bump release references and badges to `v2.0.3` across runtime docs/spec defaults:
  - `.env.example`
  - `module/core/telemetry/config.go`
  - `module/core/cmd/api/main.go`
  - `module/core/startup/runtime.go`
  - module OpenAPI specs for analytics/membership/syncrecord/email/campaign/segment
  - `README.md` and `module/woocommerce/README.md`
- Add frontend integration guide `MANUAL-v2.0.3.md` and link it from root `README.md`.
- Publish ClickHouse-first marketing BI stack updates from the 2.0 line:
  - analytics ingestion + seed + ClickHouse schema bootstrap,
  - segment resolver hard dependency on analytics backend,
  - campaign delivery integration events,
  - contacts consent endpoint removal in favor of membership routes.

### [v2.0.2] - 2026-03-14
- Add frontend-facing manual with complete contracts and use-case flows for analytics, membership, segments, campaigns, email, and sync monitoring.
- Remove planning artifacts `plan/MARKETING.md` and `plan/SYNC-RECORD.md` after implementation handoff.
- Harden analytics/segment behavior for ClickHouse-first BI compliance:
  - segment resolution now depends on analytics resolver and no longer falls back to MySQL query paths.
  - startup guard: `SEGMENT_ENABLED=true` requires `ANALYTICS_ENABLED=true`.
- Expand analytics module capabilities:
  - ClickHouse schema migrations embedded under `module/analytics/adapter/clickhouse/migrations/*.up.sql`.
  - runtime applies ClickHouse schema on startup when enabled.
  - event consumers wired for:
    - `contacts.v1.created`
    - `contacts.v1.updated`
    - `orders.v1.created`
    - `orders.v1.updated`
    - `orders.v1.status.updated`
    - `membership.v1.changed`
    - `campaign.v1.delivery`
  - analytics seed now backfills ClickHouse data (contacts snapshot, orders facts, order item facts, membership events, campaign events derived from delivery history).
- Remove legacy contacts consent endpoints from contacts module:
  - deleted `POST /contacts/optin` and `POST /contacts/optout`.
  - consent writes now flow through membership endpoints/module only.
- Expand segment filter DSL mapping/validation to support BI-oriented filters:
  - `city`
  - `order_recency`
  - `no_order_recency`
  - `category`
  - `top_spenders` (limit or percentage)
  - `first_purchase_only`
  - `subscribed_no_buy`
  - `opt_in_status`
  - `metadata`
  - plus existing compatibility filters (`city_code_in`, `min_total_spend`, `email_opt_in`, `purchased_sku`).

### [v2.0.0] - 2026-03-14
- Bump service/version references and badges to `v2.0.0`.
- Add centralized sync execution registry module (`module/syncrecord`):
  - normalized schema (`sync_runs`, `sync_run_errors`) with retention cleanup cron.
  - endpoints:
    - `GET /syncrecord/runs`
    - `GET /syncrecord/runs/:id`
    - `GET /syncrecord/stats`
  - wire recorder integration into WooCommerce, Falabella, Assets, Membership migration, Analytics seed, and Campaign send.
- Add auditable membership module (`module/membership`):
  - immutable `membership_stamps` + latest `membership_status` snapshot tables.
  - endpoints:
    - `POST /membership/optin`
    - `POST /membership/optout`
    - `POST /membership/stamp`
    - `GET /membership/status/:contactId`
    - `GET /membership/status/:contactId/:channel`
    - `GET /membership/status/:contactId/stamps`
    - `GET /membership/stamps/:contactId/:channel`
    - `POST /membership/migrate`
  - publish `membership.v1.changed` integration events.
- Add optional analytics module (`module/analytics`) for marketing:
  - endpoints:
    - `GET /analytics/status`
    - `POST /analytics/seed`
  - ClickHouse adapter wiring and optional startup integration.
- Add segment module (`module/segment`):
  - segment CRUD and resolution endpoints:
    - `POST /segments`
    - `GET /segments`
    - `GET /segments/:id`
    - `PATCH /segments/:id`
    - `DELETE /segments/:id`
    - `POST /segments/:id/resolve`
- Add optional email delivery module (`module/email`):
  - endpoints:
    - `POST /email/send`
    - `POST /email/webhooks/ses`
  - delivery/status-history persistence and webhook-driven status updates.
- Add campaign module (`module/campaign`):
  - endpoints:
    - `POST /campaigns`
    - `GET /campaigns`
    - `GET /campaigns/:id`
    - `PATCH /campaigns/:id`
    - `DELETE /campaigns/:id`
    - `POST /campaigns/:id/send`
  - asynchronous fan-out send orchestration with sync-run recording.
- Modify contacts and WooCommerce consent flows to delegate membership stamping.
- Add SQL migrations:
  - `000010_sync_record_schema`
  - `000011_membership_schema`
  - `000012_email_delivery_schema`
  - `000013_campaign_schema`
  - `000014_segment_schema`

### [v1.3.7] - 2026-03-14
- Bump service/version references and badges to `v1.3.7`.
- Fix WooCommerce contact duplicate-document upsert recovery:
  - add deterministic document lookup filters (`documentType`, `documentNumber`) in contacts list queries.
  - use direct `(documentType, documentNumber)` lookup fallback on duplicate create collisions.
  - treat raw SQL duplicate-key driver messages (for example MySQL `Error 1062`) as duplicate-retryable fallback cases.
- Fix WooCommerce inbound order status/comment mutation policy:
  - allow `source=woocommerce_sync` mutations for Woo orders so status transitions are applied during `/woo/sync/orders`.
  - keep Woo loop suppression for other Woo-origin sources.
- Add stamp-aware circle opt-in lifecycle behavior:
  - `POST /contacts/optin` now sets:
    - `flock_checker_circle_optin=yes`
    - `flock_checker_circle_optin_accepted_at`
    - `flock_checker_circle_optin_accepted_at_utc`
    - and clears `flock_checker_circle_optin_rejected_at*`.
  - `POST /contacts/optout` now sets:
    - `flock_checker_circle_optin=no`
    - `flock_checker_circle_optin_rejected_at`
    - `flock_checker_circle_optin_rejected_at_utc`
    - and clears `flock_checker_circle_optin_accepted_at*`.
  - Woo checker sync now maps `flock_checker_circle_optin=no` into rejected-at metadata and clears accepted-at stamps.
  - Duplicate-order metadata merge now prefers latest checker values per email and normalizes circle opt-in yes/no stamp cleanup.
- Add/extend unit coverage for:
  - Woo duplicate-document fallback by document filters and raw SQL duplicate messages.
  - Woo status update allowlist for `woocommerce_sync`.
  - opt-in/opt-out stamp transitions and checker metadata merge precedence.

### [v1.3.6] - 2026-03-13
- Bump service/version references and badges to `v1.3.6`.
- Fix WooCommerce contact sync metadata mapping (`POST /woo/sync/contacts`) to propagate checker consent keys from order metadata into contact metadata:
  - `flock_checker_<key>`
  - `flock_checker_<key>_accepted_at`
  - `flock_checker_<key>_accepted_at_utc`
- Add fallback accepted-at timestamp generation for checker values set to `yes` when source order metadata omits accepted-at fields.
- Fix duplicate-document contact sync behavior:
  - when contact creation fails with duplicate document constraints, fallback lookup now searches existing contacts by `(documentType, documentNumber)` and updates that contact instead of failing the sync upsert.
- Add/extend unit coverage for checker metadata propagation and duplicate-document fallback update flows.

### [v1.3.5] - 2026-03-13
- Bump service/version references and badges to `v1.3.5`.
- Extend WooCommerce order-to-contact sync metadata extraction for checker payloads:
  - `flock_checker_<key>`
  - `flock_checker_<key>_accepted_at`
  - `flock_checker_<key>_accepted_at_utc`
- Preserve checker metadata dynamically for custom keys (for example, `flock_checker_terminos_extra`).
- Ensure `flock_checker_circle_optin=yes` syncs accepted-at metadata even when timestamps are absent in order metadata (fallback to order creation timestamp).
- Add protected contact consent helper endpoints (requires `contacts:manage`):
  - `POST /contacts/optin` (by email)
  - `POST /contacts/optout` (by email)
- Persist opt-in/out actions in contact metadata keys:
  - `flock_checker_circle_optin`
  - `flock_checker_circle_optin_accepted_at`
  - `flock_checker_circle_optin_accepted_at_utc`
- Extend contacts OpenAPI/runtime/docs and add unit coverage for new consent routes and Woo checker metadata mapping.

### [v1.3.4] - Pending Release
- Bump service/version references and badges to `v1.3.4`.
- Add explicit gallery sort fields for drag-and-drop ordering:
  - `gallery[].position` (global image order)
  - `gallery[].variationPosition` (variation-scoped image order)
- Persist gallery and variation image positions in products schema and API responses.
- Enforce deterministic Falabella image payload ordering:
  - variation-linked images by `variationPosition` (fallback to `position`)
  - then shared images by `position`
  - stable fallback by source order
- Add SQL migrations `000009_products_gallery_variation_position` (MySQL + SQLite) with rollback files.
- Add unit and e2e coverage for position persistence and Falabella image ordering.

### [v1.3.3] - Pending Release
- Bump service/version references and badges to `v1.3.3`.
- Add Falabella product-feed resolution backoff gate before image sync dispatch.
- Block image sync when latest `ProductCreate`/`ProductUpdate` feed is not `Finished` with `FailedRecords=0`.
- Add Falabella feed-resolution backoff env configuration:
  - `FALABELLA_PRODUCT_FEED_RESOLUTION_ATTEMPTS`
  - `FALABELLA_PRODUCT_FEED_RESOLUTION_BACKOFF_MS`
  - `FALABELLA_PRODUCT_FEED_RESOLUTION_REQUEST_TIMEOUT_MS`
- Record product feed status entries before image sync attempts to keep cron feed-resolution visibility even when image sync is blocked.
- Add unit and e2e coverage for feed-resolution gating and backoff behavior.

### [v1.3.2] - Pending Release
- Bump service/version references and badges to `v1.3.2`.
- Add manual JPG worker trigger endpoint while keeping scheduled worker behavior:
  - `POST /assets/workers/jpg/run`
  - optional query overrides: `tags`, `batchSize`, `jpegQuality`
- Route executes the same JPG conversion flow used by the scheduled worker and returns execution counters.
- Extend assets OpenAPI documentation and runtime/module READMEs with manual worker endpoint details.
- Add unit and e2e coverage for manual JPG worker route behavior.

### [v1.3.1] - 2026-03-09
- Bump service/version references and badges to `v1.3.1`.
- Add assets JPG worker with cron lifecycle and env-driven configuration:
  - `ASSETS_JPG_WORKER_ENABLED`
  - `ASSETS_JPG_WORKER_CRON`
  - `ASSETS_JPG_WORKER_TAGS` (comma-separated)
  - `ASSETS_JPG_WORKER_BATCH_SIZE`
  - `ASSETS_JPG_WORKER_JPEG_QUALITY`
  - `ASSETS_JPG_WORKER_TIMEOUT_MS`
- Convert selected tagged assets to `image/jpeg`, upload `.jpg` replacements, update asset binary metadata, and delete previous source objects.
- Add storage download capability to assets/core storage ports and S3 adapter (`Download`).
- Add SQL migrations `000008_assets_jpg_worker_tag_index` (MySQL + SQLite) with rollback files.
- Add unit and e2e coverage for JPG worker conversion/replacement flow.

### [v1.3.0] - 2026-03-09
- Bump service/version references and badges to `v1.3.0`.
- Add Falabella image-transcode support so image sync can send JPG-only URLs to Falabella (`GET /falabella/images/transcoded`).
- Add Falabella transcode runtime config:
  - `FALABELLA_PRODUCT_IMAGE_TRANSCODE_ENABLED`
  - `FALABELLA_PRODUCT_IMAGE_TRANSCODE_PUBLIC_BASE_URL`
  - `FALABELLA_PRODUCT_IMAGE_TRANSCODE_ALLOWED_PREFIXES`
  - `FALABELLA_PRODUCT_IMAGE_TRANSCODE_TIMEOUT_MS`
- Expose feed task metadata to distinguish sync results:
  - `task=data` for product data feeds
  - `task=image` for image feeds
- Include task metadata in sync summary feed results, sync-status API responses, and feed resolve responses.
- Add SQL migrations `000007_falabella_sync_task` (MySQL + SQLite) with rollback files.
- Extend Falabella OpenAPI documentation and module READMEs with transcode + execution status endpoints.
- Stop forcing `X-Ray-ID` from telemetry middleware and keep access-log `trace_id` empty when no active trace span exists.

### [v1.2.4] - 2026-03-09
- Bump service/version references and badges to `v1.2.4`.
- Persist Falabella feed-to-variation links for sync entries (`variationIds`) and expose them in sync status responses.
- Add SQL migrations `000006_falabella_sync_variations` (MySQL + SQLite) with rollback files.

### [v1.2.3] - 2026-03-08
- Bump service/version references and badges to `v1.2.3`.
- Add access-log correlation fields `ray_id` and `trace_id`.
- Separate WooCommerce cron sync timeout from startup validation timeout.
- Add `WOOCOMMERCE_SYNC_TIMEOUT_MS` (default `600000`) for scheduled sync execution budget.
- Keep `WOOCOMMERCE_VALIDATION_TIMEOUT_MS` focused on startup/integration validation.

### [v1.2.1] - 2026-02-24
- Align `X-Ray-ID` with active OpenTelemetry `trace_id` for request correlation.

### [v1.2.0] - 2026-02-23
- Route OpenTelemetry exporter/runtime errors through Zap logging.
- Stabilize telemetry behavior for production startup/runtime.

### [v1.1.0] - 2026-02-23
- Improve WooCommerce/core logging behavior to reduce noisy access logs.
- Add Falabella and production readiness improvements.

### [v1.0.0] - 2026-02-16
- Initial tagged release line.
- Add targeted WooCommerce contact synchronization by email.
