# WooCommerce Integration

The WooCommerce module bridges a WooCommerce store and Mannaiah's domain. The integration
is **bidirectional**:

```
WooCommerce ──(cron / HTTP trigger)──▶ Mannaiah       [INBOUND sync]
Mannaiah    ──(event bus)────────────▶ WooCommerce    [OUTBOUND update]
```

The canonical source of truth for business data is **Mannaiah**. WooCommerce is treated as
an external data source for contacts and orders; the sync gate enforces that WooCommerce
cannot overwrite updates originating from Mannaiah operators.

---

## Sync Directions at a Glance

| Direction | What is synced | Trigger |
|-----------|---------------|---------|
| **Woo → Mannaiah** (inbound) | Customer contacts (billing details) | Cron, HTTP trigger |
| **Woo → Mannaiah** (inbound) | Orders | Cron, HTTP trigger |
| **Mannaiah → Woo** (outbound) | Order status, items, shipping | `orders.v1.*` events |

---

## Table of Contents

| File | Contents |
|------|---------|
| [02-DOMAIN.md](02-DOMAIN.md) | All WooCommerce domain types and port commands |
| [03-SYNC-INBOUND.md](03-SYNC-INBOUND.md) | Woo→Mannaiah contact and order sync (cron, HTTP, pagination, worker pool) |
| [04-SYNC-OUTBOUND.md](04-SYNC-OUTBOUND.md) | Mannaiah→Woo event-driven reverse push, loop prevention |
| [05-FIELD-MAPPING.md](05-FIELD-MAPPING.md) | Field-by-field mapping tables for contacts and orders |
| [06-EVENTS.md](06-EVENTS.md) | Integration events published by this module |
| [07-API.md](07-API.md) | HTTP endpoints (manual sync trigger) |
| [08-CONFIGURATION.md](08-CONFIGURATION.md) | All environment variables and circuit breaker settings |

---

## Architecture Overview

```
                       ┌──────────────────────────────────┐
                       │         WooCommerce Store         │
                       │  /wp-json/wc/v3/orders           │
                       │  /wp-json/wc/v3/products         │
                       └────────────────┬─────────────────┘
                                        │ REST (OAuth)
                              ┌─────────▼──────────┐
                              │  WooSource Adapter  │
                              │  (circuit breaker)  │
                              └─────────┬──────────┘
                                        │
              ┌─────────────────────────▼───────────────────────────┐
              │              WooCommerce Module                       │
              │                                                       │
              │  ContactSyncService       OrderSyncService            │
              │    paginate + map           paginate + map            │
              │    worker pool (8)          worker pool (8)           │
              │    upsert contacts          upsert contacts + orders  │
              │    stamp membership         record sync run           │
              │                                                       │
              │  MainstreamUpdateService                              │
              │    handles orders.v1.* events                        │
              │    → WooDestination.UpdateOrderFromMainstream         │
              └──────┬──────────────────────────────┬────────────────┘
                     │                              │
           ┌─────────▼──────┐           ┌──────────▼─────────┐
           │ module/contacts │           │ module/orders       │
           │ module/membership│          │ module/syncrecord   │
           └─────────────────┘           └────────────────────┘
```

---

## Dependency Modules (Ports)

| Port | Backed By | Purpose |
|------|-----------|---------|
| `ContactSyncTarget` | `module/contacts` | Upsert contacts |
| `OrderSyncTarget` | `module/orders` | Upsert orders |
| `OrderDestination` | `module/orders` | Update order from event |
| `MembershipStamper` | `module/membership` | Stamp opt-in/opt-out |
| `SyncRecorder` | `module/syncrecord` | Track sync run history |

WooCommerce has **no database tables of its own**. All persistent state lives in the target
modules.

---

## Quick Reference

### Manually trigger a full sync

```bash
# Contacts
curl -X POST https://api.example.com/woo/sync/contacts \
  -H "Authorization: Bearer <token>"

# Orders
curl -X POST https://api.example.com/woo/sync/orders \
  -H "Authorization: Bearer <token>"
```

### Targeted sync (single record)

```bash
# Sync one customer
curl -X POST "https://api.example.com/woo/sync/contacts?email=customer@example.com" \
  -H "Authorization: Bearer <token>"

# Sync one WooCommerce order
curl -X POST "https://api.example.com/woo/sync/orders?id=1042" \
  -H "Authorization: Bearer <token>"
```

Both return a `SyncSummary`:

```json
{
  "trigger": "api",
  "processed": 1,
  "created": 0,
  "updated": 1,
  "unchanged": 0,
  "skipped": 0,
  "failed": 0
}
```
