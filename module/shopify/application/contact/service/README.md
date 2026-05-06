# Shopify Contact Sync Service

This package validates Shopify customer access, maps Shopify customers into normalized contact payloads, upserts them into the mainstream contacts module, and pushes mainstream contact events back to Shopify with loop-safe link stitching. Bulk contact sync imports Shopify customers first, then backfills local contacts into Shopify using link/email deduplication.

## Key methods / endpoints / events
- `service.NewService(cfg, source, target, logger, breakers...)`
- `(*ContactSyncService).ValidateIntegration(ctx)`
- `(*ContactSyncService).SyncContactByID(ctx, trigger, id)`
- `(*ContactSyncService).SetMainstreamBackfill(source, handler)`
- `service.BuildContactSyncCommand(customer)`
- `service.NewUpserter(contactsService, links, customerDestination, logger)`
- `service.NewMainstreamUpdateService(source, destination, links, logger, breakers...)`
- `(*MainstreamContactUpdateService).HandleContactEvent(ctx, payload)`
