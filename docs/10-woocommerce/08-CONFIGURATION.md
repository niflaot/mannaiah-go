# WooCommerce — Configuration

All configuration is supplied via environment variables. No defaults assume a live store —
syncs are disabled until explicitly enabled.

---

## Store Connection

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `WOOCOMMERCE_URL` | `string` | `""` | WooCommerce store base URL (e.g. `https://mystore.com`) |
| `WOOCOMMERCE_CONSUMER_KEY` | `string` | `""` | REST API consumer key |
| `WOOCOMMERCE_CONSUMER_SECRET` | `string` | `""` | REST API consumer secret |
| `WOOCOMMERCE_VERIFY_SSL` | `bool` | `true` | Verify TLS certificate. Set to `false` only for self-signed certificates in staging environments |
| `WOOCOMMERCE_REQUEST_TIMEOUT_MS` | `int` | `5000` | Per-request HTTP timeout (milliseconds) |
| `WOOCOMMERCE_VALIDATION_TIMEOUT_MS` | `int` | `3000` | Connectivity validation timeout (milliseconds) |

Authentication for raw HTTP calls (non-SDK) uses OAuth consumer key+secret appended as query
parameters (`?consumer_key=…&consumer_secret=…`).

---

## Sync Schedule

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `WOOCOMMERCE_SYNC_CONTACTS` | `bool` | `false` | Enable scheduled contact sync |
| `WOOCOMMERCE_SYNC_CONTACTS_CRON` | `string` | `0 0 * * *` | Cron expression for contact sync |
| `WOOCOMMERCE_SYNC_ORDERS` | `bool` | `false` | Enable scheduled order sync |
| `WOOCOMMERCE_SYNC_ORDERS_CRON` | `string` | `0 0 * * *` | Cron expression for order sync |

Both cron jobs default to midnight UTC daily. The HTTP trigger endpoints (`POST /woo/sync/contacts`
and `POST /woo/sync/orders`) work regardless of whether cron is enabled.

---

## Worker Pool

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `WOOCOMMERCE_SYNC_PAGE_SIZE` | `int` | `100` | Orders per API page during sync |
| `WOOCOMMERCE_SYNC_WORKERS` | `int` | `8` | Concurrent upsert goroutines |
| `WOOCOMMERCE_SYNC_TIMEOUT_MS` | `int` | `600000` | Total sync operation timeout (10 minutes) |

Increasing `WOOCOMMERCE_SYNC_WORKERS` speeds up upsert throughput but increases load on the
contacts and orders databases. The default of 8 is appropriate for stores up to ~50 000 orders.

---

## Circuit Breaker

The circuit breaker wraps all WooCommerce source API calls. When the failure threshold is
reached, subsequent calls fail immediately without attempting the network request, allowing
the system to degrade gracefully.

| Env Var | Type | Default | Description |
|---------|------|---------|-------------|
| `WOOCOMMERCE_CIRCUIT_BREAKER_ENABLED` | `bool` | `true` | Enable circuit breaker |
| `WOOCOMMERCE_CIRCUIT_BREAKER_MAX_REQUESTS` | `uint32` | `1` | Max requests allowed in half-open state |
| `WOOCOMMERCE_CIRCUIT_BREAKER_INTERVAL_MS` | `int` | `60000` | Closed-state counter reset interval (ms) |
| `WOOCOMMERCE_CIRCUIT_BREAKER_TIMEOUT_MS` | `int` | `30000` | Open-state recovery window (ms) |
| `WOOCOMMERCE_CIRCUIT_BREAKER_FAILURE_THRESHOLD` | `uint32` | `3` | Consecutive failures before circuit opens |

### Circuit Breaker State Machine

```
CLOSED ──(3 consecutive failures)──▶ OPEN
  (normal operation)                  (fast-fail, no API calls)
                                              │
                                     30 s timeout
                                              │
                                              ▼
                                         HALF-OPEN
                                    (1 trial request)
                                         │        │
                                   success      failure
                                         │        │
                                      CLOSED    OPEN
```

In the `OPEN` state, any sync triggered via cron or HTTP will fail immediately with a
`circuit breaker open` error. The `sync.failed` integration event is published with this
error payload.

---

## Tolerant JSON Decoding

WooCommerce's REST API sometimes returns numeric fields as strings (especially in custom meta
fields). The WooCommerce source adapter uses flexible decoders:

- `flexibleInt` — unmarshals both `123` and `"123"` as `int`
- `flexibleFloat64` — unmarshals both `12.5` and `"12.5"` as `float64`

These are applied at the API response deserialization layer so that sync failures due to
type mismatches in the WooCommerce API are avoided.

---

## Minimal Production Configuration Example

```dotenv
WOOCOMMERCE_URL=https://mystore.com
WOOCOMMERCE_CONSUMER_KEY=ck_abc123...
WOOCOMMERCE_CONSUMER_SECRET=cs_xyz789...
WOOCOMMERCE_VERIFY_SSL=true
WOOCOMMERCE_REQUEST_TIMEOUT_MS=5000

WOOCOMMERCE_SYNC_CONTACTS=true
WOOCOMMERCE_SYNC_CONTACTS_CRON=0 2 * * *

WOOCOMMERCE_SYNC_ORDERS=true
WOOCOMMERCE_SYNC_ORDERS_CRON=0 1 * * *

WOOCOMMERCE_SYNC_PAGE_SIZE=100
WOOCOMMERCE_SYNC_WORKERS=8
WOOCOMMERCE_SYNC_TIMEOUT_MS=600000

WOOCOMMERCE_CIRCUIT_BREAKER_ENABLED=true
WOOCOMMERCE_CIRCUIT_BREAKER_FAILURE_THRESHOLD=3
WOOCOMMERCE_CIRCUIT_BREAKER_TIMEOUT_MS=30000
```
