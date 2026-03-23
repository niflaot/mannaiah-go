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

### [v1.0.1] - 2026-03-23
- Orders module: added `payment_method` field stored on order records.
  - New `payment_method` column on `orders` table (`000026_order_payment_method` migration, MySQL + SQLite).
  - `Order` domain, `CreateCommand`, and repository mapper updated to carry and persist payment method values.
  - HTTP `POST /orders` request accepts optional `paymentMethod` string field.
  - OpenAPI `Order` response schema and `OrderCreate` request schema updated with `paymentMethod` property.
  - WooCommerce sync chain updated end-to-end: `WooOrder`, `rawOrderPayload` (raw-decode path), `mapSDKOrder` (SDK path), `OrderSyncCommand`, `mapOrderToCommand`, and `toCreateCommand` all propagate `payment_method`.
  - Unit tests added: raw order decode, `mapOrderToCommand`, and `toCreateCommand` payment method propagation.
- Version references set to `v1.0.1` (`module/core/telemetry/config.go`, `module/core/cmd/api/main.go`, `module/core/startup/runtime.go`, `.env.example`).
- `README.md` and `module/woocommerce/README.md` latest badge updated to `v1.0.1`.

### [v1.0.0] - 2026-03-22
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
