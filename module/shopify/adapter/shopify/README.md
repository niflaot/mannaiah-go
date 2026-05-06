# Shopify Admin Adapter

This package wraps the Shopify Admin REST API for targeted customer reads, targeted order reads, and outbound order status updates.

## Key methods / endpoints / events
- `shopify.NewClient(cfg)`
- `(*Client).Validate(ctx)`
- `(*Client).GetCustomer(ctx, id)`
- `(*Client).GetOrder(ctx, id)`
- `(*Client).UpdateOrderFromMainstream(ctx, shopifyID, command)`
