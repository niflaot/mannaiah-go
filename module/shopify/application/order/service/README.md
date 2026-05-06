# Shopify Order Sync Service

This package validates Shopify order access, maps Shopify orders into normalized mainstream order payloads, creates or refreshes the linked contact first, and pushes mainstream changes back to Shopify. Outbound order events update an existing linked Shopify order, or create a missing Shopify order when the order contact is already linked to a Shopify customer.

## Key methods / endpoints / events
- `service.NewService(cfg, source, contactTarget, target, logger, breakers...)`
- `(*OrderSyncService).ValidateIntegration(ctx)`
- `(*OrderSyncService).SyncOrderByID(ctx, trigger, id)`
- `service.NewUpserter(ordersService, links)`
- `service.NewMainstreamUpdateService(destination, links, logger, breakers...)`
