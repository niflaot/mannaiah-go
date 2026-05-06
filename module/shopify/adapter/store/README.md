# Shopify Store Adapter

This package persists Shopify synchronization links and webhook delivery idempotency rows.

## Key methods / endpoints / events
- `store.NewRepository(db)`
- `(*Repository).GetLinkByShopifyID(ctx, kind, shopifyID)`
- `(*Repository).GetLinkByMannaiahID(ctx, kind, mannaiahID)`
- `(*Repository).UpsertLink(ctx, input)`
- `(*Repository).CreateDeliveryIfAbsent(ctx, deliveryID, topic)`
