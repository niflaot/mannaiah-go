# Shopify Runtime Package

`runtime` wires Shopify adapters, services, routes, and module lifecycle behavior.

## Key configuration
- `SHOPIFY_SYNC_MODE` controls direction: `shopify` imports Shopify to Mannaiah only, while `bidirectional` also enables Mannaiah-to-Shopify event consumers and manual backfills. Default: `shopify`.
- `SHOPIFY_ADMIN_RATE_LIMIT_INTERVAL_MS` controls per-shop Admin API request pacing and defaults to `1200`.
- `SHOPIFY_429_RETRY_DELAY_MS` controls fallback waits for Shopify `429` responses that do not include `Retry-After` and defaults to `5000`.
