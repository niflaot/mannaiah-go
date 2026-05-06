# Shopify Module

`module/shopify` integrates Shopify with Mannaiah in-process.

## Key methods / endpoints / events

- `shopify.New(...)`
- `POST /shopify/sync/contacts`
- `POST /shopify/sync/orders`
- `POST /shopify/webhooks`
- order integration event consumer for `orders.v1.created`, `orders.v1.updated`, and `orders.v1.status.updated`
