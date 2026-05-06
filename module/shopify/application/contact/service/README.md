# Shopify Contact Sync Service

This package validates Shopify customer access, maps Shopify customers into normalized contact payloads, and upserts them into the mainstream contacts module.

## Key methods / endpoints / events
- `service.NewService(cfg, source, target, logger, breakers...)`
- `(*ContactSyncService).ValidateIntegration(ctx)`
- `(*ContactSyncService).SyncContactByID(ctx, trigger, id)`
- `service.BuildContactSyncCommand(customer)`
- `service.NewUpserter(contactsService, links)`
