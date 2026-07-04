# Address Syncing Plan

## Current Findings

- Shipping quotation edits can change `destCityCode` without changing the stored order shipping city. Batch marks, rotulus PDFs, and checklist PDFs must therefore treat the quotation or explicit recipient override as the dispatch snapshot, not re-read stale order city values.
- Shopify order sync currently maps city text through `module/core/citycode.Resolve` during contact/order mapping. The resolver now wraps `github.com/flockstore/lib-go-cities/platform` so ambiguous, duplicated, incongruent, and low-confidence matches fail instead of guessing.
- Shopify order imports do not currently carry source `updated_at`, ETags, or a content hash. Existing-order imports call `orders.Update` with items, shipping address, shipping charges, and coupons every time, so an old external sync can overwrite an operator repair.
- Existing `shopify_sync_links` track IDs, last known status, and last synced time, but not source version, content hash, city-resolution state, or dirty-field ownership.
- Shopify write-back exists for order item edits and fulfillment, but there is no dedicated address-update destination for propagating corrected shipping city/address data back to Shopify.

## Goals

- Never guess an unsafe city code.
- Persist city-resolution failures so operators can repair them.
- Show unresolved address/city states in frontend order and freight-quotation flows before carrier calls.
- Preserve operator repairs across Shopify resyncs unless Shopify has a newer conflicting version.
- Propagate approved city/address corrections back to Shopify so the external source stops reintroducing bad data.
- Keep hot city lookups fast and deterministic.

## Phase 1: Dispatch Snapshot Correctness

- Treat dispatch marks as immutable operational snapshots.
- When creating a mark from a quotation, use this precedence:
  - explicit recipient override from the request,
  - quotation `destCityCode` / request snapshot `destCityCode`,
  - order-source destination only as missing-field enrichment.
- Frontend add-to-batch fallback payloads must use the edited quotation destination when the recipient form has not supplied a city override.
- Regression coverage:
  - order city Medellín (`05001`), edited quotation Armenia (`05059`), generated mark/checklist/rotulus city must be `05059`;
  - address/name/phone can still be enriched from the order when the quotation only changed destination city.

## Phase 2: City Resolution Failure Persistence

- Add an address-resolution table through core migrations:
  - `id`,
  - `source_type` (`shopify_order`, `shopify_contact`, `manual_order_update`, `shipping_quotation`),
  - `source_id`,
  - `field_path` (`shipping_address.city`, `contact.default_address.city`),
  - `raw_city`,
  - `raw_department`,
  - `resolved_city_code`,
  - `status` (`resolved`, `failed`, `manually_resolved`, `pushed_to_shopify`),
  - `reason` (`LOW_THRESHOLD`, `DUPLICATED`, `INCONGRUENT`, `AMBIGUOUS`),
  - `suggestions_json`,
  - `threshold`,
  - `resolved_by`,
  - `resolved_at`,
  - `created_at`,
  - `updated_at`.
- Add uniqueness on `(source_type, source_id, field_path)` so each current failure has one repair row.
- Store suggestions from `citycode.ResolveDetailed` as small JSON because they are operator guidance, not queryable business state.
- Keep orders/contacts with unresolved cities in a controlled state:
  - do not persist `-1` as a normal valid city for new Shopify imports;
  - either omit the shipping city and store a resolution failure, or store a blocked/error metadata flag until the operator repairs it.
- Expose read endpoints for unresolved address issues and repair actions.
- Frontend:
  - show a blocking alert on affected order detail and freight quotation rows;
  - show suggestions for duplicated/ambiguous cities;
  - let the operator choose a city code or edit city/department text;
  - after repair, re-run package/quotation flow using the fixed code.

## Phase 3: Source Version And Idempotent Sync

- Extend Shopify source order payloads to include `updated_at` from REST/GraphQL.
- Extend `shopify_sync_links` or add a companion version table with:
  - `source_updated_at`,
  - `source_hash`,
  - `last_applied_hash`,
  - `dirty_fields_json`,
  - `last_local_write_at`.
- Build a canonical Shopify order sync hash from fields that Mannaiah imports:
  - line items,
  - shipping address,
  - shipping charges,
  - discounts,
  - payment gateway names,
  - financial/fulfillment statuses.
- Before updating an existing order:
  - if `source_updated_at` and `source_hash` match the last applied values, skip the update and only touch `last_synced_at`;
  - if local dirty fields exist, do not overwrite those fields from Shopify unless Shopify has a strictly newer `updated_at` and the field is not protected;
  - if Shopify is newer and conflicts with a local repair, create an address-resolution conflict instead of silently overwriting.
- Keep status sync separate from mutable order/address sync. Status can still append history when changed, but item/address updates should be hash-gated.
- Tests:
  - repeated identical Shopify sync does not call `orders.Update`;
  - local shipping city repair survives a repeated Shopify sync with the same source version;
  - newer Shopify source version creates a conflict for protected local city fields;
  - status-only changes still append status without rewriting address fields.

## Phase 4: Shopify Write-Back For Address Repairs

- Add a Shopify destination method dedicated to shipping-address repair.
- Preferred implementation:
  - use Shopify Admin GraphQL if the target API supports order shipping-address update for the installed API version;
  - otherwise use REST order update with the minimal `shipping_address` payload.
- The write-back payload should include:
  - address lines,
  - city display name resolved from city code,
  - province/department when available,
  - phone,
  - zip if already present.
- Trigger write-back only after explicit operator confirmation or a configured trusted auto-repair policy.
- On success:
  - mark resolution row as `pushed_to_shopify`;
  - update sync link `source_updated_at` / hash after refetching Shopify order;
  - clear local dirty flag for the pushed fields.
- On failure:
  - keep the local repair active;
  - store write-back error and retry eligibility;
  - show non-blocking frontend warning that Shopify still differs.

## Phase 5: Rollout

- Add migrations first, behind feature flags:
  - `SHOPIFY_CITY_MATCH_THRESHOLD` default `0.80`;
  - `SHOPIFY_ADDRESS_REPAIR_WRITEBACK_ENABLED` default `false`;
  - `SHOPIFY_ORDER_VERSION_GUARD_ENABLED` default `true` after backfill.
- Backfill:
  - scan existing Shopify-linked orders with missing/invalid city codes;
  - run `ResolveDetailed`;
  - mark safe resolutions;
  - create failure rows for duplicated/ambiguous/incongruent/low-threshold records.
- Deploy order:
  - backend resolver + dispatch snapshot fix;
  - persistence/API for failures;
  - frontend alerts and repair flow;
  - sync version guard;
  - Shopify write-back.

## Open Decisions

- Confirm whether `Armenia` without department should default to Antioquia (`05059`) or remain blocked as duplicated. The library intentionally blocks it today unless department evidence exists.
- Decide whether quotation-only city edits should automatically update the order record, or only the shipping mark snapshot. Operationally, the mark snapshot fix is safer immediately; long term, explicit repair should update order, resolution state, and optionally Shopify together.
- Decide conflict priority when a Shopify customer edits shipping address after a warehouse operator has repaired it locally.
