# Shopify Runtime Package

`runtime` wires Shopify adapters, services, routes, and module lifecycle behavior.

## Key configuration
- `SHOPIFY_ADMIN_RATE_LIMIT_INTERVAL_MS` controls per-shop Admin API request pacing and defaults to `600`.
- `SHOPIFY_429_RETRY_DELAY_MS` controls fallback waits for Shopify `429` responses that do not include `Retry-After` and defaults to `1100`.
