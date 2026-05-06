# Shopify Module

`module/shopify` integrates Shopify with Mannaiah in-process.

## Key methods / endpoints / events

- `shopify.New(...)`
- `GET /shopify/oauth/install`
- `GET /shopify/oauth/callback`
- `POST /shopify/sync/contacts`
- `POST /shopify/sync/orders`
- `POST /shopify/webhooks`
- `GET /shopify/ext/orders/:shopifyOrderId`
- `POST /shopify/ext/orders/:shopifyOrderId/sync`
- `GET /shopify/ext/contacts/:shopifyCustomerId`
- `POST /shopify/ext/contacts/:shopifyCustomerId/sync`
- order integration event consumer for `orders.v1.created`, `orders.v1.updated`, and `orders.v1.status.updated`

## Context and usage

The Shopify module now uses OAuth-backed, per-store installations persisted in `shopify_installations`.
Manual sync routes accept a targeted Shopify identifier and may optionally include `shopDomain` when multiple Shopify stores are installed.
Webhook ingestion resolves the emitting shop from `X-Shopify-Shop-Domain`, and Shopify Admin extension routes require a signed session token tied to one installed store.
