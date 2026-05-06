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
- contact integration event consumer for `contacts.v1.created` and `contacts.v1.updated`
- order integration event consumer for `orders.v1.created`, `orders.v1.updated`, and `orders.v1.status.updated`

## Context and usage

The Shopify module now uses OAuth-backed, per-store installations persisted in `shopify_installations`.
Manual sync routes accept a targeted Shopify identifier and may optionally include `shopDomain` when multiple Shopify stores are installed.
Webhook ingestion resolves the emitting shop from `X-Shopify-Shop-Domain`, and Shopify Admin extension routes require a signed session token tied to one installed store.
Contact synchronization now persists `shopify_sync_links`, stitches inbound-created links before outbound fan-out, and pushes mainstream contact changes back to Shopify without re-emitting equivalent webhook echoes.
Order synchronization uses the same link table: linked orders receive status/tag updates, while unlinked Mannaiah orders create Shopify orders only when the order contact already has a Shopify customer link so the created order is assigned to the correct customer.
