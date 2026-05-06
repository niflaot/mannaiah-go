# Shopify Admin Adapter

This package wraps the Shopify Admin REST API for targeted customer reads, targeted order reads, customer search, order listing, webhook registration, and OAuth token exchange.
The client serializes Admin API calls per shop domain and retries throttled `429` responses with a conservative fallback delay so bulk syncs and webhooks do not trip Shopify rate limits.

## Key methods / endpoints / events
- `shopify.NewClient(cfg)`
- `(*Client).Validate(ctx)`
- `(*Client).GetCustomer(ctx, id)`
- `(*Client).FindCustomerByEmail(ctx, email)`
- `(*Client).ListCustomers(ctx, sinceID, limit)`
- `(*Client).GetOrder(ctx, id)`
- `(*Client).ListOrders(ctx, sinceID, limit)`
- `(*Client).RegisterWebhooks(ctx, shopDomain, accessToken, address)`
