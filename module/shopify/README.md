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

## Context and usage

The Shopify module now uses OAuth-backed, per-store installations persisted in `shopify_installations`.
Manual sync routes accept a targeted Shopify identifier and may optionally include `shopDomain` when multiple Shopify stores are installed.
Webhook ingestion resolves the emitting shop from `X-Shopify-Shop-Domain`, and Shopify Admin extension routes require a signed session token tied to one installed store.
Contact synchronization imports Shopify customers into Mannaiah, persists `shopify_sync_links`, and deduplicates by Shopify ID, email, and document data before creating or updating local contacts.
Order synchronization imports Shopify orders into Mannaiah using the same link table, creates or refreshes the linked local contact first, and deduplicates by Shopify order ID before creating or updating local orders.
The module is intentionally Shopify-to-Mannaiah only: it does not subscribe to mainstream contact/order integration events and does not write customers, orders, tags, or notes back to Shopify.
