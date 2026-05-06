# Shopify HTTP Adapter

This package exposes protected manual sync endpoints and the public Shopify webhook endpoint with signature verification and asynchronous processing.

## Key methods / endpoints / events
- `http.NewHandler(contactsService, ordersService, processor, deliveries, secret, authorizers...)`
- `(*Handler).RegisterRoutes(router)`
- `(*Handler).SetAuthorizer(authorizer)`
- `http.NewProcessor(workers, timeout, contactsService, ordersService, logger)`
- `VerifyWebhookSignature(secret, body, signature)`
