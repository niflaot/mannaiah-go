# Shopify Order Sync Service

This package validates Shopify order access, maps Shopify orders into normalized mainstream order payloads, and creates or refreshes the linked local contact before upserting the local order.

## Key methods / endpoints / events
- `service.NewService(cfg, source, contactTarget, target, logger, breakers...)`
- `(*OrderSyncService).ValidateIntegration(ctx)`
- `(*OrderSyncService).SyncOrderByID(ctx, trigger, id)`
- `service.NewUpserter(ordersService, links)`
