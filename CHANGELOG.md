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

### [v2.9.16] - 2026-03-22
- Fix campaign product link rendering when product datasheet URLs are variation-scoped:
  - Recommendation URL resolution now supports scoped URL keys matched by both `product_variation_links` IDs and `product_variants` SKU tokens.
  - Scoped URL token normalization now accepts wrapped scopes such as `(sku).url`, `[sku].url`, `{sku}.url`, and quoted scopes.
  - When plain `url` is empty and scoped URLs exist, recommendation now falls back to a deterministic first scoped URL so template `.URL` is not empty.
  - Product catalog entries now include variant SKU tokens for recommendation URL candidate matching.
- Tests added/updated:
  - `module/analytics/application/recommendation/service_support_test.go`
  - `module/analytics/adapter/store/product_catalog_repository_support_test.go`
- Documentation/OpenAPI/version updates:
  - Core Swagger/OpenAPI and telemetry/default version references bumped to `v2.9.16` (`2.9.16` in OpenAPI `info.version` fields).
  - Root/module README latest version badges bumped to `v2.9.16`.

### [v2.9.15] - 2026-03-22
- Fix campaign/recommendation email product image resolution in runtime sends:
  - Core startup now wires a concrete analytics asset URL resolver into recommendation service after assets module startup.
  - Recommendation image URLs now resolve from assets service using metadata URL keys with priority and key-based fallback URL building.
  - Fallback public base URL for assets now resolves from `FALABELLA_PRODUCT_IMAGE_BASE_URL` or `STORAGE_ENDPOINT + STORAGE_BUCKET_NAME`.
- Tests added:
  - `module/core/cmd/api/analytics_asset_url_resolver_test.go` covering metadata priority, key-based URL fallback, lookup error behavior, and base URL fallback resolution.
- Documentation/OpenAPI/version updates:
  - Core Swagger/OpenAPI and telemetry/default version references bumped to `v2.9.15` (`2.9.15` in OpenAPI `info.version` fields).
  - Root/module README latest version badges bumped to `v2.9.15`.

### [v2.9.12] - 2026-03-22
- Campaign/frontend contract and template-population fixes:
  - `PATCH /campaigns/{id}` now preserves explicit empty `productBlocks: []` updates (clear semantics) instead of treating empty arrays as omitted fields.
  - Test-send strict personalization now fails with controlled `400 invalid_contact_personalization` when an explicit `contactId` cannot be resolved, preventing silent fallback discrepancies.
  - Template context name semantics updated: `.Contact.Name` now resolves as short-name (first-name preference), while `.Contact.FullName` keeps full display name.
- Product DSL enrichment and media resolution improvements:
  - Added `.URL` in product template items, resolved from product datasheet attribute key `url` (realm-aware with fallback).
  - Improved recommendation image URL resolution fallback chain:
    - asset resolver output,
    - `asset_metadata` URL keys (`falabella_url`, `public_url/publicUrl`, `cdn_url/cdnUrl`, `image_url`, `url`),
    - direct `asset_id` when already an absolute `http(s)` URL.
- Email tracking pixel fallback:
  - When `EMAIL_TRACKING_BASE_URL` is empty, runtime now falls back to `https://<sender-domain>` from configured sender email.
- Documentation and OpenAPI:
  - Added root spec file `FINAL-DSL-EMAILS.md` as backend DSL source of truth for frontend integration.
  - Campaign OpenAPI now documents create/update request bodies and campaign template fields (`templateVars`, `productBlocks`) and was bumped to `2.5.7`.
  - Analytics recommended-product schema now includes `url` and Analytics OpenAPI was bumped to `2.6.1`.
- Tests added/updated:
  - Campaign HTTP: explicit empty `productBlocks` mapping regression.
  - Campaign application: strict personalization failure behavior, short-name rendering semantics, product URL template rendering.
  - Email runtime: tracking-base-url fallback behavior.
  - Analytics recommendation: realm URL and gallery image URL fallback helpers.
- Release version bumped to `v2.9.12`.

### [v2.9.11] - 2026-03-22
- Prevent contact-email leakage in campaign template context:
  - Template context now sets `.Contact.Email` to the actual recipient email for the current send operation (test-send override email in `POST /campaigns/{id}/test`), instead of using contact-repository email data.
  - This keeps test-send personalization safe while preserving real-contact name resolution from `contactId`.
- Tests added:
  - `module/campaign/application/service_send_render_test.go`: verifies `.Contact.Email` resolves to the send target email in both HTML and text renders.
- OpenAPI/docs updates:
  - Campaign OpenAPI metadata bumped to `2.5.6`.
- Release version bumped to `v2.9.11`.

### [v2.9.10] - 2026-03-22
- Fix backend/frontend personalization discrepancy on campaign test-send:
  - Campaign runtime now wires a real contact-data provider from contacts service into campaign rendering (`SetContactDataProvider`), replacing noop fallback behavior.
  - `.Contact.Name` and `.Contact.Email` in test-send templates now resolve from the provided `contactId` when the contact exists, instead of defaulting to the override test email.
  - Campaign runtime now wires analytics-backed affinity product provider for campaign rendering (`SetAffinityProductProvider`) so dynamic product blocks resolve consistently with recommendation rules.
- Tests added:
  - `module/core/cmd/api/campaign_contact_data_adapter_test.go` verifies contact personalization mapping and display-name fallback behavior.
- OpenAPI/docs updates:
  - Campaign OpenAPI metadata bumped to `2.5.5`.
- Release version bumped to `v2.9.10`.

### [v2.9.9] - 2026-03-22
- Fix campaign test-send behavior for invalid template syntax:
  - `TestSend` now uses strict template rendering and returns a controlled validation error when template parse/execute fails, instead of silently delivering raw `{{...}}` content.
  - New domain sentinel `ErrInvalidTemplate` mapped by campaign HTTP adapter to `400` with message key `invalid_template`.
  - `POST /campaigns/{id}/test` OpenAPI `400` response description now includes invalid template syntax cases.
- Rendering robustness:
  - Keep product block source detection fix (`baseTag` / `baseTags` / `pinnedProductIds`) so modern block configs are resolved before template execution.
- Tests added:
  - `module/campaign/application/service_send_render_test.go`: strict-render invalid template regression.
  - `module/campaign/adapter/http/handler_error_test.go`: `ErrInvalidTemplate` → `400 invalid_template`.
- OpenAPI/docs updates:
  - Campaign OpenAPI metadata bumped to `2.5.4`.
- Release version bumped to `v2.9.9`.

### [v2.9.8] - 2026-03-22
- Fix campaign template-product block resolution when using modern block inputs:
  - `renderForContact` no longer skips product blocks when `BaseTag` is empty but `BaseTags` or `PinnedProductIDs` are present.
  - Product block source detection now accepts any of: non-empty `baseTag`, non-empty `baseTags`, or non-empty `pinnedProductIds`.
  - This prevents false-empty product collections that could trigger template execution failures and raw-template fallback behavior in delivered emails.
- Tests added:
  - `module/campaign/application/service_send_render_test.go` verifies rendering resolves products for both BaseTags-only and pinned-only block configurations.
- OpenAPI/docs updates:
  - Campaign OpenAPI metadata bumped to `2.5.3`.
- Release version bumped to `v2.9.8`.

### [v2.9.7] - 2026-03-22
- Fix campaign test-send error mapping for SES identity-verification failures:
  - SES `MessageRejected` errors caused by unverified sender/recipient identities are now classified as `domain.ErrSenderUnavailable` in campaign application flows.
  - `POST /campaigns/{id}/test` now returns controlled `503` (`email_sender_unavailable`) for this provider-unavailable/configuration state instead of `500`.
  - Async campaign fan-out send workers reuse the same sender error classification for consistent failure semantics.
- Tests added:
  - `module/campaign/application/sender_error_test.go` verifies SES unverified-identity detection and passthrough behavior for unrelated errors.
  - `module/campaign/adapter/http/handler_error_test.go` verifies `ErrSenderUnavailable` maps to HTTP `503` with the expected error code.
- OpenAPI/docs updates:
  - Campaign OpenAPI metadata bumped to `2.5.2`.
  - `POST /campaigns/{id}/test` `503` response description updated to include unavailable sender states.
- Release version bumped to `v2.9.7`.

### [v2.9.6] - 2026-03-22
- Finalize SES email-tag compatibility rollout for campaign sends:
  - Keep SES `idempotency_key` message-tag sanitization active so unsupported characters (for example `:` in `campaignID:contactID` and `test:campaignID:uuid`) are replaced with `_` before `SendEmail`.
  - Add regression tests in `module/email/adapter/ses/provider_test.go` covering campaign/test-send idempotency-key patterns and unsupported-character replacement.
- OpenAPI/docs updates:
  - Campaign OpenAPI metadata bumped to `2.5.1`.
  - `POST /campaigns/{id}/test` operation description now documents internal SES tag-value sanitization behavior.
- Release version bumped to `v2.9.6`.

### [v2.9.5] - 2026-03-22
- Fix SES `BadRequestException` caused by invalid characters in email tag values:
  - Idempotency keys (`campaignID:contactID`, `test:campaignID:uuid`) contain colons which SES rejects in tag values (only `[a-zA-Z0-9_\-.@]` allowed).
  - `sanitizeSESTagValue` now replaces any disallowed character with `_` before the tag is sent.
- Release version bumped to `v2.9.5`.

### [v2.9.4] - 2026-03-22
- Fix duplicate `provider_message_id` unique constraint violation on concurrent or repeated email sends:
  - `deliveryModel.ProviderMessageID` was `string`, causing `''` to be inserted for all deliveries before SES confirms. MySQL unique indexes reject duplicate empty strings but allow multiple `NULL`s.
  - Changed field to `*string`; empty strings now persist as `NULL` via `nullableString` helper; reads back via `derefString`.
- Release version bumped to `v2.9.4`.

### [v2.9.3] - 2026-03-22
- Fix SES static credentials not forwarded to AWS SDK:
  - `EMAIL_SES_ACCESS_KEY_ID` and `EMAIL_SES_SECRET_ACCESS_KEY` config fields were parsed but never passed to `ses.NewProvider`, causing the AWS SDK to fall through its credential chain to EC2 IMDS and time out (500 on all email send operations in non-EC2 environments).
  - `ses.Config` now includes `AccessKeyID` and `SecretAccessKey`; when both are non-empty, `awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(...))` is used instead of relying on the ambient credential chain.
- Release version bumped to `v2.9.3`.

### [v2.9.2] - 2026-03-22
- Add campaign test-send endpoint:
  - **`POST /campaigns/:id/test`** — renders the campaign template for a given `contactId` and delivers the result to an override `email` address. Does not modify campaign status, counters, or stats.
  - Request body: `{ "contactId": "...", "email": "..." }`. `contactId` is optional — omitting it uses the override email as the name fallback.
  - Response: `{ "email": "...", "subject": "...", "status": "submitted" }`.
  - Returns `400` when `email` is missing, `503` when the email sender is not configured.
  - Campaign OpenAPI bumped to `2.5.0`.
- Release version bumped to `v2.9.2`.

### [v2.9.1] - 2026-03-22
- Fix campaign `PATCH` returning `500` when `template_vars` or `product_blocks` is empty:
  - `marshalTemplateFields` now falls back to `"{}"` / `"[]"` instead of empty string `""` so MySQL `JSON NOT NULL` columns always receive valid JSON.
- Release version bumped to `v2.9.1`.

### [v2.9.0] - 2026-03-21
- Add contact name interpolation modes for campaign templates:
  - **`{{.Contact.FirstName}}`** — first word of the contact name before the first space (e.g. `"Juan"` from `"Juan Pérez"`).
  - **`{{.Contact.FullName}}`** — complete contact display name (e.g. `"Juan Pérez"`).
  - **`{{.Contact.Name}}`** — retained as-is for backward compatibility (identical to `FullName`).
  - Computed in `renderForContact` via `campaigntemplate.ExtractFirstName`; no fallback change: email is still the fallback when no name is available.
- Add email open tracking via invisible pixel:
  - **`GET /email/track/open/:id`** — public (no auth) endpoint that serves a 1×1 transparent GIF and records an `opened` status history entry for the delivery. Fail-open: unknown IDs produce no error response.
  - **`domain.StatusOpened`** — new `"opened"` status value added to the email delivery status set; recorded in `email_delivery_status_history` only (root delivery status is not mutated on open, per history-authoritative design).
  - **`EMAIL_TRACKING_BASE_URL`** — new env var on the email module. When set (e.g. `https://api.example.com`), the pixel `<img>` is injected just before `</body>` in every outbound HTML email. Empty by default (pixel disabled).
  - Pixel is injected into the HTML sent to the provider; the stored `html_body` in `email_deliveries` retains the original template output.
- Add UTM tracking parameters to all campaign email links:
  - All `http://` and `https://` `href` attributes in rendered campaign HTML bodies are rewritten to append `utm_source=email&utm_medium=campaign&utm_campaign={slug}&utm_id={campaignID}`.
  - Non-HTTP links (`mailto:`, `tel:`, anchor `#`) are left unchanged.
  - Applied after template rendering in `renderForContact`; raw text body is not modified.
  - Logic lives in `module/campaign/application/template/link_rewriter.go` (`RewriteLinks`).
- Email OpenAPI bumped to `2.1.0`; Campaign OpenAPI bumped to `2.4.0`.
- Release version bumped to `v2.9.0`.

### [v2.8.0] - 2026-03-21
- Add multi-tag union and intersection filtering for product recommendations:
  - **`baseTags`** (comma-separated HTTP param) / **`BaseTags []string`** (domain/campaign) replaces single `baseTag` as the primary tag selector. `baseTag` is retained for backward compatibility and merged into `BaseTags` during `Normalize()`.
  - **`baseTagMode`** controls matching semantics:
    - `"any"` (default) — union: products with **at least one** of the specified tags (e.g. `baseTags=offer-tier-1,offer-tier-2` returns products tagged with either).
    - `"all"` — intersection: products that carry **every** tag in `baseTags` (e.g. `baseTags=classic,offer-tier-1` returns only products tagged with both).
  - SQL: union uses `DISTINCT product_id WHERE tag_id IN ?`; intersection uses `GROUP BY product_id HAVING COUNT(DISTINCT tag_id) = N`.
  - If any tag in an `"all"` query does not exist, zero results are returned (no product can satisfy a missing tag).
  - `BaseTags` deduplicated in `Normalize()` to prevent HAVING count mismatch on duplicate input.
  - Port interface: `GetProductsByBaseTag` renamed to `GetProductsByBaseTags` with new `baseTags []string` and `baseTagMode string` parameters.
  - Analytics OpenAPI bumped to `2.6.0`; Campaign OpenAPI bumped to `2.3.0`.
  - Campaign `ProductBlock`: gains `baseTags` and `baseTagMode` JSON fields.
  - `plan/RECOMMENDATION-GUIDE.md` updated with multi-tag documentation and examples.
- Release version bumped to `v2.8.0`.

### [v2.7.0] - 2026-03-21
- Enforce realm-mandatory price and image on product recommendations, add variation filtering and preference:
  - **Mandatory realm price**: price is now resolved from `product_datasheet_attributes` where `key='price'` for the requested realm. Products with no realm price attribute are excluded from results. Base `products.price` is retained as a raw field but not used for display.
  - **Mandatory realm image**: the first gallery image visible in the requested realm is required. Products with no realm-visible gallery item (checked via `product_gallery_included_realms`) are excluded from results.
  - **Gallery realm visibility**: a gallery item is visible in realm R when `product_gallery_included_realms` has no rows for it (all-realms) or has a row with `realm = R`.
  - **`ProductGalleryEntry.VariationIDs`**: gallery item variation links loaded from `product_gallery_variations`; allow per-variation image selection.
  - **`ProductCatalogEntry.VariationIDs`**: product-level variation links loaded from `product_variation_links`.
  - **`ProductDatasheetEntry.Price *float64`**: realm-specific price parsed from `product_datasheet_attributes` (supports JSON number and JSON string representations).
  - **`RecommendationQuery.FilterVariationIDs`**: when non-empty, only products carrying at least one matching variation ID are candidates (SQL filter via `product_variation_links`). Applied to both `GetProductsByBaseTag` and `GetProductsByIDs`.
  - **`RecommendationQuery.PreferVariationIDs`**: when non-empty, image selection prefers gallery items linked to a matching variation before falling back to the first realm-visible item.
  - **`ProductCatalogStore` interface extended**: `GetProductsByBaseTag` gains `filterVariationIDs []string` parameter; `GetProductsByIDs` gains `filterVariationIDs []string` parameter.
  - **HTTP endpoint** `GET /analytics/recommendations/contacts/:contactId`: new `filterVariationIds` and `preferVariationIds` query params (comma-separated).
  - Analytics OpenAPI version bumped to `2.5.0`.
  - **Campaign `ProductBlock`**: gains `filterVariationIds` and `preferVariationIds` fields; passed through HTTP handler and affinity adapter.
  - Campaign OpenAPI version bumped to `2.2.0`.
  - Noop store updated for new interface signatures.
  - **`plan/RECOMMENDATION-GUIDE.md`** created with full recommendation system reference.
- Release version references bumped to `v2.7.0`.

### [v2.6.2] - 2026-03-21
- Fix product gallery query in recommendation catalog repository:
  - Table name was `product_gallery` (non-existent); corrected to `product_gallery_items`.
- Release version references bumped to `v2.6.2`.

### [v2.6.1] - 2026-03-20
- Extend campaign product blocks with pinned and excluded product controls:
  - **`pinnedProductIDs`** on `ProductBlock` and `RecommendationQuery`: specific product IDs always included first in the block result, loaded via a new `ProductCatalogStore.GetProductsByIDs` method, bypassing baseTag/affinity filtering. `BaseTag` is now optional when `pinnedProductIDs` is non-empty (enables editorial "exactly these products" blocks).
  - **`excludeProductIDs`** on `ProductBlock` and `RecommendationQuery`: product IDs that must never appear in results; applied at SQL level (`WHERE id NOT IN ?`) in `resolveProductIDs` and merged with loaded pinned IDs to prevent duplication between pinned and dynamic slots.
  - **`ProductCatalogStore.GetProductsByIDs`** (`port/product_catalog.go`): new interface method loading products by explicit ID list; confirms active records, restores input order, reuses shared `loadProductEntries` GORM helper.
  - **`RecommendationService.Recommend` rewritten**: (1) load pinned products; (2) build unified `excludeSet` = `ExcludeProductIDs` ∪ pinned IDs; (3) compute `dynamicLimit = Limit - len(pinned)`; (4) if `dynamicLimit > 0 && BaseTag != ""` run affinity → correlation → catalog with `excludeIDs`; (5) combine and cap at `Limit`.
  - **HTTP endpoint** `GET /analytics/recommendations/contacts/:contactId`: new `pinnedIds` and `excludeIds` query params (comma-separated); validation now returns 400 only when both `baseTag` and `pinnedIds` are empty.
  - **Noop store** updated: `noopProductCatalogStore.GetProductsByIDs` added.
  - Analytics OpenAPI version bumped to `2.4.1`.
  - Campaign HTTP handler: `productBlockRequest` extended with `pinnedProductIds` and `excludeProductIds`; `mapProductBlockRequests` propagates both fields.
  - Campaign OpenAPI version bumped to `2.1.1`.
- Release version references bumped to `v2.6.1`.

### [v2.6.0] - 2026-03-20
- Ship campaign personalization engine and channel-agnostic recommendation API:
  - **Analytics: RecommendationService** (`module/analytics/application/recommendation`):
    - 5-step resolution algorithm: contact tag affinity fetch → tag correlation expansion → product catalog filter → affinity-weighted ranking → realm-aware display data resolution.
    - `port.TagCorrelationStore` interface + GORM adapter (`tag_correlation_repository.go`) querying `tag_correlations WHERE source_tag IN ? AND probability > 0`.
    - `port.ProductCatalogStore` interface + GORM adapter (`product_catalog_repository.go`) resolving base tag → expanded affinity tags → category filter → datasheets + gallery with realm data.
    - `port.AssetURLResolver` interface + `NoopAssetURLResolver` default; consumers inject a concrete resolver.
    - `domain.RecommendationQuery` (normalized, Limit clamped to [1,10], Realm defaults to "default") and `domain.RecommendedProduct`.
  - **Analytics HTTP endpoint** `GET /analytics/recommendations/contacts/:contactId`:
    - Required query param `baseTag`; optional `categoryId`, `realm`, `limit`, `affinity` (enable contact-affinity expansion), `minScore` (affinity percentile threshold).
    - Returns `[]RecommendedProduct` (empty array when no candidates found).
    - Registered in `spec_recommendation.go` with `RecommendedProduct` OpenAPI component schema.
    - Analytics module OpenAPI version bumped to `2.4.0`.
  - **Campaign template personalization** (`module/campaign`):
    - `domain.ProductBlock` — stored with campaign as JSON; drives per-send product resolution per named block.
    - `domain.TemplateProduct`, `domain.ContactTemplateData`, `domain.TemplateContext` — template rendering data model.
    - `domain.Campaign.TemplateVars` (`map[string]string`) and `domain.Campaign.ProductBlocks` (`[]ProductBlock`) fields added.
    - `port.ContactDataProvider` interface + `NoopContactDataProvider`; provides `Name`, `Email`, `LastSaleDate` per contact.
    - `port.AffinityProductProvider` interface + `NoopAffinityProductProvider`; provides `[]TemplateProduct` per contact + block.
    - `application/template.Renderer` — Go `text/template` with fixed function allowlist (`formatDate`, `formatPrice`, `default`, `upper`, `lower`) and 2 MiB output cap (`ErrOutputTooLarge`).
    - `CampaignService.renderForContact` — per-contact enrichment: fetches contact data + resolves all product blocks, builds `TemplateContext`, renders `HTMLBody`/`TextBody`; fail-open on any enrichment error.
    - `CampaignService.SetContactDataProvider` and `SetAffinityProductProvider` setter methods; noops wired by default.
    - `adapter/affinity.ProductProvider` — thin adapter mapping `ProductBlock` → `domain.RecommendationQuery` → `RecommendationService.Recommend` → `[]TemplateProduct`.
    - Campaign store: `template_vars` and `product_blocks` columns persisted as JSON; marshal/unmarshal helpers added.
    - Campaign HTTP handler: `productBlockRequest` struct, `templateVars` and `productBlocks` accepted in create and update request bodies.
    - Campaign module OpenAPI version bumped to `2.1.0`.
  - **Database migrations** (`000021_campaign_template_fields`):
    - SQLite: `ALTER TABLE campaigns ADD COLUMN template_vars TEXT NOT NULL DEFAULT ''` + `ADD COLUMN product_blocks TEXT NOT NULL DEFAULT ''`.
    - MySQL: `ALTER TABLE campaigns ADD COLUMN template_vars JSON NOT NULL DEFAULT (JSON_OBJECT())` + `ADD COLUMN product_blocks JSON NOT NULL DEFAULT (JSON_ARRAY())`.
    - Down migrations provided for both dialects.
  - **Noop stores extended**: `noopTagCorrelationStore` and `noopProductCatalogStore` added to analytics runtime for ClickHouse/DB-absent environments.
- Release version references bumped to `v2.6.0`.

### [v2.5.3] - 2026-03-19
- Fix ClickHouse error 48 in affinity segment preview/count (third attempt):
  - ClickHouse converts `IN (SELECT ...)` with a window function inside into a JoinLogical step, which it also cannot handle when the outer query references `cs.contact_id` in other conditions.
  - Eliminated window functions entirely. Replaced with `GROUP BY contact_id HAVING maxIf(score, <tag_condition>) * 100.0 / nullIf(max(score), 0) >= ?`:
    - `maxIf(score, tag IN (?))` — max score among matching tags per contact (zero if no match)
    - `max(score)` — max score across all tags per contact (the normalization denominator)
    - `nullIf(..., 0)` — prevents division by zero; contacts with all-zero scores are excluded
  - The IN subquery is now a simple two-level GROUP BY with no window functions and no JOINs, which ClickHouse executes as a plain hash-based semi-join without any decorrelation.
  - Fix applied to all six affinity conditions: tag/category/variation in both `store_query_segment_affinity.go` and `store_query_segment_clauses.go`.
- Release version references bumped to `v2.5.3`.

### [v2.5.2] - 2026-03-19
- Fix ClickHouse error 48 ("Cannot decorrelate query, because 'JoinLogical' step is not supported") in affinity segment preview/count:
  - The v2.5.1 fix replaced window functions with CROSS JOIN inside correlated EXISTS subqueries, but ClickHouse's decorrelator also does not support JoinLogical steps inside EXISTS.
  - Replaced all six affinity EXISTS correlated subqueries with `cs.contact_id IN (SELECT contact_id FROM (...) WHERE ...)` — a non-correlated IN subquery.
  - The IN subquery is executed independently, so window functions (`max(score) OVER (PARTITION BY contact_id)`) work correctly without any decorrelation.
  - Semantics are identical: for each contact, the normalized affinity score for the matched tag/category/variation must meet the minimum percentage threshold.
  - Fix applied to both the DSL filter path (`store_query_segment_affinity.go`) and the clause path (`store_query_segment_clauses.go`).
- Release version references bumped to `v2.5.2`.

### [v2.5.1] - 2026-03-19
- Fix ClickHouse error 48 ("Cannot check Sorting plan step for correlated expressions") in affinity segment preview/count:
  - All three affinity EXISTS subqueries (`tag_affinity`, `category_affinity`, `variation_affinity`) used `max(score) OVER (PARTITION BY contact_id)` — a window function — inside correlated subqueries. ClickHouse cannot apply the sort step required by window functions when decorrelating EXISTS predicates.
  - Replaced the window function with a `CROSS JOIN` against a scalar `max()` aggregate subquery.
  - Pushed `contact_id = cs.contact_id` into the leaf-level `WHERE` clauses of both sides of the join, eliminating the outer correlation while preserving identical semantics.
  - Fix applied to both the DSL filter path (`store_query_segment_affinity.go`) and the clause path (`store_query_segment_clauses.go`).
- Release version references bumped to `v2.5.1`.

### [v2.5.0] - 2026-03-19
- Breaking DSL update for affinity segment filters:
  - Removed absolute affinity threshold semantics (`minScore`) for segment create/update payloads.
  - Added percentage-only thresholds (`minScorePct`, range `0..100`) for:
    - `tag_affinity`
    - `category_affinity`
    - `variation_affinity`
  - Added `relatedTags` support for `tag_affinity` rows to evaluate one rule over `tag + relatedTags`.
  - Percentage evaluation is relative per contact: matched item score is compared against that contact's maximum affinity score for the same domain.
- Segment service validation/mapping updates:
  - Affinity rows now require `minScorePct`.
  - Invalid/legacy affinity payloads are rejected during segment resolution/count/preview (fail-closed behavior).
- Query builder updates:
  - Clause-based and legacy affinity SQL builders now use percentage-relative threshold checks.
  - Tag affinity SQL now supports rule expansion over `relatedTags`.
- Segment OpenAPI updates:
  - Segment module API version bumped to `2.1.0`.
  - Filter parameter documentation clarifies `minScorePct` + `relatedTags` usage.
- Release version references bumped to `v2.5.0`.

### [v2.4.9] - 2026-03-19
- Add include/exclude segment DSL execution support across analytics resolution:
  - Segment `filters[]` now supports `exclude: true` to negate any supported filter clause.
  - Added clause-preserving mapping from segment service to analytics resolver, enabling mixed include/exclude filtering without dropping filter intent.
  - Added clause-based ClickHouse segment query builder handling include/exclude behavior for:
    - purchase windows (for example include last 90 days + exclude last 30 days),
    - product/category/tag/variation, city/country, affinity and propensity filters,
    - legacy and explicit order-status scoping with excluded status support.
  - Top-spender helper paths now honor included and excluded order-status scopes consistently.
- Fix segment filter validation regressions:
  - `first_purchase_only` and `subscribed_no_buy` no longer fail opt-in status validation.
  - `rfm_range` now validates `rMin/rMax/fMin/fMax/mMin/mMax` correctly.
- OpenAPI updates:
  - Segment module OpenAPI version bumped to `2.0.8`.
  - Segment filter schema now documents `filters[].exclude` as an optional boolean.
- Release version references bumped to `v2.4.9`.

### [v2.4.8] - 2026-03-19
- Fix ClickHouse affinity queries failing with "Aggregate function found in WHERE":
  - All three affinity store queries (`GetTagAffinity`, `GetCategoryAffinity`, `GetVariationAffinity`) used `WHERE affinity_score >= ?` but `affinity_score` is also an aggregate alias — ClickHouse resolves the alias in WHERE, not the raw column.
  - Moved the `minScore` filter from `WHERE` to `HAVING` in all three queries.

### [v2.4.7] - 2026-03-18
- Track affinity refresh runs in the sync registry:
  - Added `syncRecorder port.SyncRecorder` to `AffinityService` (defaults to `NoopSyncRecorder`).
  - Added `SetSyncRecorder` method to `AffinityService`.
  - `RefreshAll` now records `StartRun("analytics.affinity.refresh", "manual")`, `CompleteRun` on success, and `FailRun` with the failing MV name on error.
  - `Module.SetSyncRecorder` now forwards the recorder to both `AnalyticsService` and `AffinityService`.

### [v2.4.6] - 2026-03-18
- Fix analytics seed silently producing empty tag affinity results after v2.4.3 FK migration:
  - `seedProductTaxonomy` was reading `SELECT product_id, tag FROM product_tags` but the `tag` column was dropped in v2.4.3 — the error was swallowed and `product_taxonomy` in ClickHouse was left empty.
  - Updated query to JOIN through the canonical `tags` registry: `JOIN tags ON tags.id = product_tags.tag_id AND tags.deleted_at IS NULL`.

### [v2.4.5] - 2026-03-18
- Version bump — no functional changes; promotes the tag correlation unordered uniqueness fix (v2.4.4) as the current stable release.

### [v2.4.4] - 2026-03-18
- Enforce unordered uniqueness for tag correlation pairs:
  - `CreateCorrelation` normalizes the pair lexicographically before storing so `(A, B)` and `(B, A)` are always persisted as the same row; the existing DB unique constraint on `(source_tag, target_tag)` then rejects the duplicate naturally.
  - `ListCorrelationsBySource` now queries `WHERE source_tag = ? OR target_tag = ?` so all correlations involving a tag are returned regardless of which side it was stored on.

### [v2.4.3] - 2026-03-18
- Unify `product_tags` with the canonical `tags` registry via FK migration:
  - MySQL/SQLite migration 000020: backfill `tags` with any pre-existing product tag names, add `tag_id BIGINT NOT NULL FK → tags(id)` to `product_tags`, drop the `tag` string column; down migration reverses this.
  - Product store write path (`replaceProductTags`): resolves each tag name to its canonical `tag_id` before inserting; returns a descriptive error if a name is not present in the registry (guarded upstream by `EnsureAll`).
  - Product store read path (`loadProductAggregate`): JOINs `product_tags` with `tags WHERE deleted_at IS NULL` to resolve tag names; soft-deleted tags are automatically excluded from product reads.
  - `ListByTagsAndPrice`: subquery now JOINs `tags` on `tag_id` and filters by `tags.name IN ?` instead of `product_tags.tag IN ?`.
  - Category store (`repository_products.go`): tag filter subquery updated to JOIN through `tags` on `tag_id`.
  - Tag store `SoftDelete`: cascade deletes `product_tags WHERE tag_id = ?` using the tag's ID instead of matching by name string.
  - `repository_test.go`: added `seedTagsForTest` helper to pre-populate the canonical registry in unit tests that bypass the application layer.

### [v2.4.2] - 2026-03-18
- Fix two CI test failures introduced in v2.4.0/v2.4.1:
  - `handler_test.go`: `serviceMock` was missing `ListByTags` — added `listByTagsFn` field and nil-safe `ListByTags` implementation to satisfy the updated `Service` interface.
  - `module_test.go`: `TestNewRejectsNilDB` was asserting `productstore.ErrNilDB` but `New()` now creates the tag repository first, so the first nil-DB error is `tagstore.ErrNilDB` — updated import and assertion accordingly.
- Bump all release version references to `v2.4.2`.

### [v2.4.1] - 2026-03-18
- Fix OpenAPI spec for all `/tags` and `/tags/correlations/*` endpoints:
  - All success responses now carry full JSON body schemas instead of description-only stubs.
  - Added `TagListResponse`, `TagCorrelationListResponse`, `DeleteResponse` component schemas.
  - Added `jsonResponseBodyRef` helper shared across all tag spec operations.
  - `GET /tags` → `TagListResponse` (`{ "data": Tag[] }`).
  - `DELETE /tags/{name}` → `DeleteResponse` (`{ "status": string }`).
  - `GET /tags/correlations` → `TagCorrelationListResponse` (`{ "data": TagCorrelation[] }`).
  - `GET /tags/correlations/source/{tag}` → `TagCorrelationListResponse`.
  - `POST /tags/correlations` → `TagCorrelation`.
  - `PATCH /tags/correlations/{id}` → `TagCorrelation`.
  - `DELETE /tags/correlations/{id}` → `DeleteResponse`.

### [v2.4.0] - 2026-03-18
- Ship canonical tag registry, tag correlations, and `min_order_count` segment filter:
  - MySQL/SQLite migration 000019: `tags` canonical registry (soft-delete, name unique index) + `tag_correlations` (source/target/probability/notes, unique pair constraint).
  - Domain: `domain/tag.Tag` (soft-deletable) and `domain/tag.TagCorrelation` structs in products module.
  - Port: `port/tag.Repository` interface with `EnsureAll`, `SoftDelete`, and full correlation CRUD contracts.
  - GORM adapter: `adapter/store/tag.Repository` — `EnsureAll` creates new tags and reintegrates soft-deleted ones; `SoftDelete` cascades to `product_tags`; hard-delete for correlations.
  - Application service: `application/tag.TagService` implements `Service` + `TagRegistrar` interface.
  - `TagRegistrar` integrated into `ProductService`: `Create` and `Update` now call `tagRegistrar.EnsureAll` before persistence so the canonical registry stays in sync.
  - HTTP endpoints: `GET /tags` (`products:read`), `DELETE /tags/:name` (`marketing:manage`), `GET /tags/correlations`, `GET /tags/correlations/source/:tag`, `POST /tags/correlations`, `PATCH /tags/correlations/:id`, `DELETE /tags/correlations/:id` (all correlation routes require `marketing:manage`).
  - OpenAPI spec: new `tags` tag group, `Tag`, `TagCorrelation`, `CreateTagCorrelationDto`, `UpdateTagCorrelationDto` schemas; all `/tags/*` path items documented.
  - Segment filter `min_order_count`: `SegmentFilter.MinOrderCount`, `toAnalyticsFilter` case in segment service, `appendMinOrderCountCondition` ClickHouse `EXISTS/HAVING countDistinct` subquery.

### [v2.3.9] - 2026-03-17
- Swap gallery realm logic from `excludedRealms` (opt-out) to `includedRealms` (opt-in):
  - Domain: `GalleryItem.ExcludedRealms` renamed to `IncludedRealms`; empty list means visible in all realms.
  - MySQL/SQLite migration 000018: creates `product_gallery_included_realms`, drops `product_gallery_excluded_realms`.
  - Repository: `productGalleryExcludedRealmRecord` → `productGalleryIncludedRealmRecord`; read/write paths updated.
  - Falabella port: `CatalogImage.ExcludedRealms` → `IncludedRealms`.
  - Falabella catalog adapter: propagates `IncludedRealms` to port layer.
  - Falabella mapper: `isRealmExcluded` replaced with `isRealmIncluded` (flipped logic — image is skipped when realm is not in `IncludedRealms` and list is non-empty).
  - OpenAPI spec: `excludedRealms` → `includedRealms` in gallery-item schema.
  - Seed script: `scripts/seed_falabella_included_realms.sql` — adds all existing gallery items to the `falabella` included realm.

### [v2.3.0] - 2026-03-16
- Ship RFM scoring + tag/category/variation affinity engine:
  - ClickHouse migrations 000006–000011: `product_taxonomy`, `rfm_scores_mv`, `tag_affinity_mv`, `category_affinity_mv`, `product_variation_taxonomy`, `variation_affinity_mv`.
  - MySQL/SQLite migration 000017: `rfm_groups`, `rfm_band_configs`, `rfm_group_conditions`.
  - Domain layer: `RFMScore`, `RFMBandConfig`, `RFMGroup`, `RFMGroupConditions`, `TagAffinity`, `CategoryAffinity`, `VariationAffinity`, `AffinityProfile`.
  - Port layer: `RFMStore`, `RFMGroupRepository`, `AffinityStore`, `TaxonomyStore`.
  - Application services: `rfm.RFMService` (with 5-minute band cache), `affinity.AffinityService`.
  - ClickHouse adapters: `store_rfm.go`, `store_affinity.go`, `store_taxonomy.go`.
  - GORM adapter: `rfm_group_repository.go` with `SeedDefaultBands`.
  - Segment query extensions: RFM score/range and tag/category/variation affinity EXISTS filters.
  - Segment filter DSL: new types `rfm_group`, `rfm_score`, `rfm_range`, `tag_affinity`, `category_affinity`, `variation_affinity`.
  - HTTP endpoints: `/analytics/rfm/*` and `/analytics/affinity/*` (including `variations`).
  - Taxonomy seed: `seedProductTaxonomy` and `seedVariationTaxonomy` run as part of `POST /analytics/seed`.
  - Noop stores for ClickHouse-absent environments.
  - E2E tests for RFM group CRUD and affinity profile endpoints.
- Bump release references to `v2.3.0`.

### [v2.2.2] - 2026-03-16
- Fix variant-SKU order item resolution in analytics seed:
  - `orders/adapter/products/resolver.go`: added `findByVariantSKU` step between parent-SKU and alternate-name lookups; queries `product_variants.sku` directly.
  - `resolution_source` is set to `"variant_sku"` for rows matched via variant SKU.
  - To recover existing unresolved rows: `TRUNCATE TABLE order_items_fact` then `POST /analytics/seed`.
- Bump release references and badges to `v2.2.2`.

### [v2.2.1] - 2026-03-16
- Homogenize all product permission scopes to `products:` prefix:
  - Products handler: `products:create`, `products:read`, `products:update`, `products:delete`.
  - Variations handler: migrated from `variations:*` to `products:create/read/update/delete`.
  - Categories handler: migrated from `product:view/manage` to `products:read/manage`.
  - All e2e tests updated to match.
- Bump release references and badges to `v2.2.1`.

### [v2.2.0] - 2026-03-16
- Ship product taxonomy with categories, tags, and price filters:
  - **Tags on Products**: `Tags []string` and `Price *float64` added to `module/products/domain/product/product.go`.
  - **product_tags table**: new `productTagRecord` and `replaceProductTags` in `module/products/adapter/store/product/repository_tags.go`; write/read paths updated.
  - **Port extensions**: `GetByIDs` and `ListByTagsAndPrice` added to `module/products/port/product/repository.go` and implemented in store adapter.
  - **Category domain**: `module/products/domain/category/category.go` — `Category`, `Filter`, `PriceRange` aggregate with `Normalize()` / `Validate()`.
  - **Category port**: `module/products/port/category/repository.go` — `Repository` interface, `ListProductsQuery`, `ListProductsResult`, sentinel errors.
  - **Category service**: `module/products/application/category/service.go` — `CategoryService` with `Create`, `Get`, `GetBySlug`, `Tree`, `Children`, `Update`, `Delete`, `ListProducts`; comprehensive mock-based tests.
  - **Category store adapter**: `module/products/adapter/store/category/repository.go` (CRUD + relations) and `repository_products.go` (union-based product resolution with tag/price/ref/pinned/children support); SQLite in-memory tests.
  - **Category HTTP handler**: `module/products/adapter/http/category/handler.go` — 7 routes (`POST /categories`, `GET /categories`, `GET /categories/:id`, `GET /categories/:id/children`, `GET /categories/:id/products`, `PATCH /categories/:id`, `DELETE /categories/:id`) with `product:view` / `product:manage` permissions; handler tests.
  - **OpenAPI spec**: `module/products/runtime/spec_categories.go` — full OpenAPI 3.0 spec for all 7 category endpoints with schemas `CreateCategoryDto`, `UpdateCategoryDto`, `Category`.
  - **Runtime wiring**: `module/products/runtime/module.go` updated to wire `categoryRepository`, `categoryService`, `categoryHandler`; `CategoryService()` accessor added; spec updated with category paths, schemas, and tag.
  - **Database migrations**: `000016_product_taxonomy_schema` for MySQL and SQLite — `product_tags`, `categories`, `category_filter_tags`, `category_filter_price_ranges`, `category_filter_category_refs`, `category_products`, and `ALTER TABLE products ADD COLUMN price`.
  - **E2E tests**: `e2e/categories_e2e_test.go` (full lifecycle) and `e2e/categories_tags_e2e_test.go` (tag/price/children filtering).
  - **CATEGORY-GUIDE.md**: frontend integration guide with data model, filter types, all endpoints, permissions, hierarchy rules, and error codes.
- Bump release references and badges to `v2.2.0`:
  - `.env.example`, `module/core/telemetry/config.go`, `README.md`.

### [v2.1.0] - 2026-03-15
- Add `GET /campaigns/:id/deliveries` endpoint returning paginated email delivery rows for a campaign:
  - New `DeliveryRow` struct and `DeliveryReader` port interface in `module/campaign/port/integration.go`.
  - `ListByCampaignID` added to `module/email/port/repository.go` and implemented in `module/email/adapter/store/repository.go` (queries by `idempotency_key LIKE '{campaignID}:%'`).
  - `Repository()` accessor exposed from `module/email/runtime/module.go`.
  - `ListDeliveries` use-case added to campaign application service; `DeliveryListResult` / `DeliveryEntry` types added.
  - HTTP handler `listDeliveries` registered at `GET /campaigns/:id/deliveries` (protected by `marketing:manage`).
  - `campaignDeliveryReaderAdapter` wired in `module/core/cmd/api/main.go`.
  - OpenAPI schema `CampaignDeliveryRow` and `CampaignDeliveryList` + `listDeliveriesOperation` added to spec.
- Bump release references and badges to `v2.1.0`:
  - `.env.example`, `module/core/telemetry/config.go`, `module/core/cmd/api/main.go`, `module/core/startup/runtime.go`, `README.md`, `module/woocommerce/README.md`.

### [v2.0.11] - 2026-03-15
- Fix ClickHouse `FINAL alias` ordering in all segment subqueries:
  - In ClickHouse the table alias must precede `FINAL` — `FROM table alias FINAL`, not `FROM table FINAL alias`.
  - Fixed `orders_fact of FINAL` and `order_items_fact oi FINAL` in all EXISTS / NOT EXISTS subqueries inside `buildSegmentWhere` (MinTotalSpend, PurchasedSKU, CategoryPattern, OrderRecency, NoOrderRecency, FirstPurchase, SubscribedNoBuy).
- Refactor analytics module to comply with 250-line file-size rule:
  - `adapter/clickhouse/store_query.go` (351 lines) split into:
    - `store_query.go` — `ResolveContacts` / `CountContacts` public methods.
    - `store_query_segment.go` — `buildSegmentWhere` and per-filter condition builders.
    - `store_query_support.go` — top-spender resolution, `orderStatusFragment`, `makePlaceholders`.
  - `application/service.go` (713 lines) split into:
    - `service.go` — interface, struct, constructor, ingest and resolve methods.
    - `service_seed.go` — `Seed` orchestration and sync-record lifecycle.
    - `service_seed_contacts.go` — contact batch reader and snapshot builder.
    - `service_seed_orders.go` — order/item batch reader and fact payload builder.
    - `service_seed_membership.go` — membership stamp batch reader.
    - `service_seed_campaign.go` — campaign delivery event batch reader.
  - All files now under 235 lines.
- Bump release references and badges to `v2.0.11`:
  - `.env.example`, `module/core/telemetry/config.go`, `module/core/cmd/api/main.go`, `module/core/startup/runtime.go`, `README.md`, `module/woocommerce/README.md`

### [v2.0.10] - 2026-03-15
- Fix ClickHouse syntax error in segment contact queries:
  - `FROM contacts_snapshot FINAL cs` is invalid in ClickHouse — the alias must precede `FINAL`.
  - Fixed both `ResolveContacts` and `CountContacts` queries to `FROM contacts_snapshot cs FINAL WHERE`.
  - Symptom: any segment filter that used `cs.` prefixed conditions (e.g. `city_code`) triggered a syntax error at position 66.
- Bump release references and badges to `v2.0.10`:
  - `.env.example`
  - `module/core/telemetry/config.go`
  - `module/core/cmd/api/main.go`
  - `module/core/startup/runtime.go`
  - `README.md` and `module/woocommerce/README.md`

### [v2.0.9] - 2026-03-15
- Add WooCommerce billing/shipping city name → Colombian municipality code resolution:
  - New `internal/citycode` package embeds 1119-entry `cities.json` at compile time via `go:embed`.
  - `Resolve(name string) string` pipeline: pass-through if already numeric → exact map lookup (O(1)) → unique-prefix fallback (handles "Bogota" → "Bogota D.C.") → Levenshtein fuzzy at 80 % similarity → `"-1"` sentinel.
  - Accent/diacritic stripping via NFD decomposition (`golang.org/x/text/unicode/norm`) before lookup.
  - `IsNumericCode(value string) bool` guards the update path so contacts with an already-resolved numeric code are never overwritten.
  - Wired into `application/contact/service/service_mapping.go`, both contact and shipping paths in `application/order/service/service_mapping.go`, and `adapter/contacts/upserter.go` `updateExisting`.
  - Shipping address empty-check uses raw string values before resolution to avoid `-1` sentinel triggering non-empty detection.
- Bump release references and badges to `v2.0.9`:
  - `.env.example`
  - `module/core/telemetry/config.go`
  - `module/core/cmd/api/main.go`
  - `module/core/startup/runtime.go`
  - `README.md` and `module/woocommerce/README.md`

### [v2.0.8] - 2026-03-15
- Add segment preview-count endpoint (`POST /segments/preview/count`):
  - Accepts the same `filters` array as segment create without persisting anything.
  - Validates filters and runs the count query against the analytics backend.
  - Returns `{ "count": <int> }` — no segment ID needed.
  - Registered before parameterised routes to avoid `preview` being matched as `:id`.
  - OpenAPI spec updated with `SegmentPreviewCount` schema and `previewCountOperation`.
- Bump release references and badges to `v2.0.8`:
  - `.env.example`
  - `module/core/telemetry/config.go`
  - `module/core/cmd/api/main.go`
  - `module/core/startup/runtime.go`
  - `README.md` and `module/woocommerce/README.md`

### [v2.0.7] - 2026-03-15
- Add `order_status` segment filter to scope all order-related subqueries by `current_status`:
  - `domain.SegmentFilter.OrderStatuses []string` field added.
  - All `orders_fact` subqueries (`min_total_spend`, `order_recency`, `no_order_recency`, `first_purchase_only`, `subscribed_no_buy`, `top_spenders`) now apply an `IN (...)` clause when statuses are set.
  - `order_items_fact` subqueries (`purchased_sku`, `category`) apply a nested correlated EXISTS against `orders_fact` for status filtering.
  - Segment filter type `"order_status"` with `parameters.statuses` wired through segment service mapping and validation.
- Harden analytics event consumer resilience:
  - JSON unmarshal failures now return `platform.NonRetriable` — bad payloads skip retry backoff and go directly to DLQ.
  - Missing required fields (contact id, order id, membership channel/action, campaign id) also marked as non-retriable.
  - Transient ClickHouse write failures (connection drops, timeouts) remain retriable as before.
- Bump release references and badges to `v2.0.7`:
  - `.env.example`
  - `module/core/telemetry/config.go`
  - `module/core/cmd/api/main.go`
  - `module/core/startup/runtime.go`
  - `README.md` and `module/woocommerce/README.md`

### [v2.0.6] - 2026-03-15
- Fix MySQL reserved-word syntax error in analytics seed contact metadata query:
  - `SELECT key,value FROM contact_metadata` failed on MySQL because `key` is a reserved keyword.
  - Fixed by passing a single raw SQL string with backtick-escaped column to GORM `Select`: `"contact_id, \`key\`, value"`.
- Bump release references and badges to `v2.0.6`:
  - `.env.example`
  - `module/core/telemetry/config.go`
  - `module/core/cmd/api/main.go`
  - `module/core/startup/runtime.go`
  - `README.md` and `module/woocommerce/README.md`

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
