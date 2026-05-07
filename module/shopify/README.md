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
- `GET /shopify/ext/contacts/:shopifyCustomerId`
- `orders.v1.updated`
- `orders.v1.status.updated`
- `shipping.v1.mark.generated`
- `shipping.v1.mark.voided`

## Context and usage

The Shopify module now uses OAuth-backed, per-store installations persisted in `shopify_installations`.
Manual sync routes accept a targeted Shopify identifier and may optionally include `shopDomain` when multiple Shopify stores are installed.
Webhook ingestion resolves the emitting shop from `X-Shopify-Shop-Domain`, and Shopify Admin extension routes require a signed session token tied to one installed store.
Contact synchronization imports Shopify customers into Mannaiah, persists `shopify_sync_links`, and deduplicates by Shopify ID, email, and document data before creating or updating local contacts.
Order synchronization imports Shopify orders into Mannaiah using the same link table, creates or refreshes the linked local contact first, and deduplicates by Shopify order ID before creating or updating local orders.
The module treats Shopify as the checkout source for Shopify-realm records and uses cron/webhooks to keep Mannaiah populated. Admin extension blocks are read-only; manual extension sync actions were intentionally removed.
For Shopify-realm orders, Mannaiah operational changes are written back only for order edits, status cancellation, fulfillment/tracking creation, and fulfillment cancellation. Customer notifications are disabled on write-back requests where Shopify exposes that control.
