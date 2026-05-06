# Shopify Order Sync Service

This package validates Shopify order access, maps Shopify orders into normalized mainstream order payloads, creates or refreshes the linked contact first, and pushes mainstream status changes back to Shopify.

## Key methods / endpoints / events
- `service.NewService(cfg, source, contactTarget, target, logger, breakers...)`
- `(*OrderSyncService).ValidateIntegration(ctx)`
- `(*OrderSyncService).SyncOrderByID(ctx, trigger, id)`
- `service.NewUpserter(ordersService, links)`
- `service.NewMainstreamUpdateService(destination, links, logger, breakers...)`
