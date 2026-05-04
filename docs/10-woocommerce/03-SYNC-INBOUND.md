# WooCommerce — Inbound Sync (Woo → Mannaiah)

The inbound sync reads orders and contacts from the WooCommerce REST API and upserts them
into Mannaiah's contacts and orders modules. It runs on a configurable cron schedule or can
be triggered manually via HTTP.

---

## Trigger Mechanisms

| Trigger | Contacts | Orders |
|---------|----------|--------|
| Cron | `WOOCOMMERCE_SYNC_CONTACTS_CRON` (default `0 0 * * *`) | `WOOCOMMERCE_SYNC_ORDERS_CRON` (default `0 0 * * *`) |
| HTTP | `POST /woo/sync/contacts` | `POST /woo/sync/orders` |
| Targeted | `POST /woo/sync/contacts?email=x` | `POST /woo/sync/orders?id=1042` |

The cron jobs are registered during `module.Start(ctx)` and deregistered on `module.Stop(ctx)`.

---

## Connectivity Validation

Before any sync operation, `ValidateIntegration()` is called:

```
GET /wp-json/wc/v3/orders?per_page=1
```

This call uses a short deadline (`WOOCOMMERCE_VALIDATION_TIMEOUT_MS`, default 3 000 ms). If
it fails, the sync is aborted and a `sync.failed` event is published.

---

## Pagination + Worker Pool

Full sync operations (contact or order) use this pipeline:

```
page 1 ──▶ fetch from WooCommerce (per_page=100, orderby=id asc)
page 2 ──▶ fetch  ↓
...         ...
page N ──▶ fetch (hasNext = false → stop)

For each page:
  map WooOrders → port commands
  dispatch to worker pool (8 goroutines)
    each worker calls UpsertByEmail / UpsertByIdentifier
    accumulates created/updated/unchanged/failed/skipped counts
```

**Configuration:**
- Page size: `WOOCOMMERCE_SYNC_PAGE_SIZE` (default `100`)
- Workers: `WOOCOMMERCE_SYNC_WORKERS` (default `8`)
- Total timeout: `WOOCOMMERCE_SYNC_TIMEOUT_MS` (default `600 000` ms = 10 min)

---

## Contact Sync (`SyncContacts`)

Contact records are **derived from order billing information**. There is no separate
WooCommerce customers endpoint used. Every unique `BillingEmail` across all orders becomes a
contact.

### Per-contact steps

1. Map `WooOrder.Billing*` fields → `ContactSyncCommand` (see [05-FIELD-MAPPING.md](05-FIELD-MAPPING.md)).
2. Call `ContactSyncTarget.UpsertByEmail(cmd)`:
   - If no contact exists for that email: **created**.
   - If contact already exists: fields are merged, **updated**.
   - If resulting state equals current state: **unchanged** (no write).
3. **Membership stamping** — after upsert, read `flock_checker_circle_optin` meta key:
   - `"yes"` → `MembershipStamper.StampByEmail(opt_in, channel="all", source="woocommerce_sync")`
   - `"no"` → `MembershipStamper.StampByEmail(opt_out, channel="all", source="woocommerce_sync")`
   - Timestamp is sourced from `_accepted_at_utc` (RFC3339) or `_accepted_at`
     (parsed with `America/Bogota` timezone).
4. Increment `SyncSummary` counter.

### Skip conditions

| Condition | Action |
|-----------|--------|
| `BillingEmail` is blank | Skipped |
| Upsert returns duplicate conflict on re-fetch | Second attempt by document number |
| Any unmapped error | Failed |

---

## Order Sync (`SyncOrders`)

Each order sync worker:
1. Upserts the contact (same as above) to ensure the linked contact exists.
2. Resolves the contact's internal Mannaiah ID.
3. Maps the full `WooOrder` → `OrderSyncCommand` (see [05-FIELD-MAPPING.md](05-FIELD-MAPPING.md)).
4. Sets `source = "woocommerce_sync"` in metadata — this is the guard key that prevents
   the WooCommerce source guard from blocking writes.
5. Calls `OrderSyncTarget.UpsertByIdentifier(realm="woocommerce", identifier=wooID, cmd)`:
   - Creates a new order if `(realm, identifier)` doesn't exist.
   - Updates the order if it does (using idempotency logic — no write if identical).

### Order status priority protection

The domain order service protects against status **downgrades**. If the WooCommerce status
maps to a lower-priority domain status than what the order currently holds, the status update
is silently discarded.

---

## Targeted Sync

### Single contact by email

```
POST /woo/sync/contacts?email=customer@example.com
```

Pages through WooCommerce orders filtered by billing email until found. If no orders exist for
that email, the contact is not created. This is equivalent to a full sync but scoped to a
single customer.

### Single order by WooCommerce ID

```
POST /woo/sync/orders?id=1042
```

Calls `source.GetOrderByID(1042)` directly (single API call, no pagination). Upserts contact
then order.

---

## Sync Run Recording

All sync operations (full or targeted) record their results via `SyncRecorder`:

```
StartRun(ctx, kind="woocommerce.contacts" / "woocommerce.orders", trigger)
  → runID

... on completion ...
CompleteRun(ctx, runID, processed, created, updated, failed, skipped)
  or
FailRun(ctx, runID, ..., []SyncError)
```

Records are queryable via the `module/syncrecord` module.

---

## Resilience

| Failure scenario | Behaviour |
|-----------------|-----------|
| WooCommerce API returns `5xx` | Circuit breaker trips; sync aborts with `sync.failed` event |
| Individual order mapping fails (no email, no items) | **Skipped** — sync continues |
| Individual upsert fails | **Logged and counted as Failed** — sync continues |
| Context deadline exceeded | Remaining workers drain in-flight requests; partial results returned |

---

## Example: Full Contact Sync Flow

```
ContactSyncService.SyncContacts(ctx, trigger="cron")
  │
  ├─ ValidateIntegration()
  │   GET /wp-json/wc/v3/orders?per_page=1 → 200 ✅
  │
  ├─ Publish: woocommerce.v1.contacts.sync.started
  │
  ├─ source.ListOrders(page=1, perPage=100, orderBy=id, asc)
  │   ← 100 orders (emails: A, B, C ... Z, some repeated)
  │
  │   Map 100 WooOrders → unique ContactSyncCommands (deduplicate by email)
  │
  │   Worker pool (8):
  │     Worker 1: UpsertByEmail(A) → created ✅
  │     Worker 2: UpsertByEmail(B) → updated (phone changed) ✅
  │     Worker 3: UpsertByEmail(C) → unchanged (no diff) ✅
  │     Worker 4: UpsertByEmail(D) → failed (DB error) ❌
  │     ...
  │
  ├─ source.ListOrders(page=2) ... → hasNext=false, stop
  │
  ├─ Publish: woocommerce.v1.contacts.sync.completed
  │   { processed: 847, created: 15, updated: 830, unchanged: 1, skipped: 0, failed: 1 }
  │
  └─ SyncRecorder.CompleteRun(...)
```
