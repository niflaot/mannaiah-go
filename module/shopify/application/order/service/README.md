# Shopify Order Sync Service

This package validates Shopify order access, maps Shopify orders into normalized mainstream order payloads, creates or refreshes the linked contact first, and pushes mainstream changes back to Shopify. Outbound order events update an existing linked Shopify order, or pre-sync the order contact and create a missing Shopify order when no order link exists.

## Key methods / endpoints / events
- `service.NewService(cfg, source, contactTarget, target, logger, breakers...)`
- `(*OrderSyncService).ValidateIntegration(ctx)`
- `(*OrderSyncService).SyncOrderByID(ctx, trigger, id)`
- `service.NewUpserter(ordersService, links)`
- `service.NewMainstreamUpdateService(destination, links, logger, breakers...)`
- `(*OrderSyncService).SetMainstreamBackfill(source, handler)`
- `(*MainstreamUpdateService).SetContactResolver(source, handler)`
