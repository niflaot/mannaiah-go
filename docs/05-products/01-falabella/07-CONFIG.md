# Falabella — Configuration

All Falabella configuration is injected via environment variables prefixed `FALABELLA_`.

---

## Seller Center API Connection

| Variable | Default | Description |
|----------|---------|-------------|
| `FALABELLA_URL` | `https://sellercenter-api.falabella.com` | Seller Center base URL |
| `FALABELLA_USER_ID` | `""` | API user identifier |
| `FALABELLA_API_KEY` | `""` | API authentication key |
| `FALABELLA_USER_AGENT` | `""` | HTTP User-Agent header |
| `FALABELLA_API_VERSION` | `1.0` | API version string |
| `FALABELLA_REQUEST_TIMEOUT_MS` | `5000` | Per-request HTTP timeout |
| `FALABELLA_VALIDATION_TIMEOUT_MS` | `3000` | Startup connectivity validation timeout |

---

## Circuit Breaker

| Variable | Default | Description |
|----------|---------|-------------|
| `FALABELLA_CIRCUIT_BREAKER_ENABLED` | `true` | Enable circuit breaker on Seller Center calls |
| `FALABELLA_CIRCUIT_BREAKER_FAILURE_THRESHOLD` | `3` | Consecutive failures before breaker opens |
| `FALABELLA_CIRCUIT_BREAKER_TIMEOUT_MS` | `30000` | Duration the breaker stays open (ms) |
| `FALABELLA_CIRCUIT_BREAKER_INTERVAL_MS` | `60000` | Closed-state failure counter reset window (ms) |
| `FALABELLA_CIRCUIT_BREAKER_MAX_REQUESTS` | `1` | Trial requests allowed in HalfOpen state |

---

## Product Mapping

| Variable | Default | Description |
|----------|---------|-------------|
| `FALABELLA_PRODUCT_REALM` | `falabella` | Datasheet realm name to read for Falabella fields |
| `FALABELLA_PRODUCT_CATEGORY_ID` | `1638` | Falabella category ID applied to all products |
| `FALABELLA_PRODUCT_GLOBAL_IDENTIFIER` | `G08010305` | Global product identifier |
| `FALABELLA_PRODUCT_ATTRIBUTE_SET_ID` | `5` | Attribute set ID |
| `FALABELLA_PRODUCT_OPERATOR_CODE` | `FACO` | Business-unit operator code (fallback if not in attributes) |

---

## Sync Workers & Feed Resolution

| Variable | Default | Max | Description |
|----------|---------|-----|-------------|
| `FALABELLA_PRODUCT_SYNC_WORKERS` | `4` | — | Concurrent goroutines for batch sync |
| `FALABELLA_PRODUCT_FEED_RESOLUTION_ATTEMPTS` | `6` | `30` | Maximum feed poll attempts during inline resolution |
| `FALABELLA_PRODUCT_FEED_RESOLUTION_BACKOFF_MS` | `1000` | `30000` | Initial backoff between poll attempts (doubles each attempt) |
| `FALABELLA_PRODUCT_FEED_RESOLUTION_REQUEST_TIMEOUT_MS` | `5000` | `30000` | Timeout for each `GetFeedStatus` call |

---

## Image Pipeline

| Variable | Default | Description |
|----------|---------|-------------|
| `FALABELLA_PRODUCT_IMAGE_BASE_URL` | `""` | Base URL prepended to S3 asset keys to form full image URLs |
| `FALABELLA_PRODUCT_IMAGE_TRANSCODE_ENABLED` | `false` | Route image URLs through the JPEG transcode proxy before submitting to Falabella |
| `FALABELLA_PRODUCT_IMAGE_TRANSCODE_PUBLIC_BASE_URL` | `""` | Public base URL of this Mannaiah instance (used to construct proxy URLs) |
| `FALABELLA_PRODUCT_IMAGE_TRANSCODE_ALLOWED_PREFIXES` | `""` | Comma-separated allowed source URL prefixes (security allowlist) |
| `FALABELLA_PRODUCT_IMAGE_TRANSCODE_TIMEOUT_MS` | `15000` | Source image fetch timeout for the transcode proxy |

---

## Background Cron

| Variable | Default | Description |
|----------|---------|-------------|
| `FALABELLA_SYNC_STATUS_CRON` | `*/5 * * * *` | Cron expression for background feed resolution |
| `FALABELLA_SYNC_STATUS_BATCH_SIZE` | `50` | Maximum pending feed entries resolved per cron tick |
