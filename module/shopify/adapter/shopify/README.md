# Shopify Admin Adapter

This package wraps the Shopify Admin REST API for targeted customer reads, targeted order reads, customer sync markers, outbound order creation, and outbound order status updates.
The client serializes Admin API calls per shop domain and retries throttled `429` responses with a conservative fallback delay so bulk syncs, webhooks, and integration-event consumers do not trip Shopify rate limits.

## Key methods / endpoints / events
- `shopify.NewClient(cfg)`
- `(*Client).Validate(ctx)`
- `(*Client).GetCustomer(ctx, id)`
- `(*Client).UpdateCustomerTags(ctx, id, tags)`
- `(*Client).AppendCustomerNote(ctx, id, note)`
- `(*Client).GetOrder(ctx, id)`
- `(*Client).CreateOrderFromMainstream(ctx, command)`
- `(*Client).UpdateOrderFromMainstream(ctx, shopifyID, command)`
