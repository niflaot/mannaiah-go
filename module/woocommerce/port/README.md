# WooCommerce Port Package

`port` defines provider-agnostic interfaces used by WooCommerce application services.

## Contracts
- Order source retrieval (`OrderSource`).
- Contact upsert target behavior (`ContactSyncTarget`).
- Integration event publication (`IntegrationEventPublisher`).

## Key Methods / Endpoints / Events
- Methods:
  - `port.OrderSource.Validate(ctx)`
  - `port.OrderSource.ListOrders(ctx, page, pageSize)`
  - `port.ContactSyncTarget.UpsertByEmail(ctx, command)`
  - `port.IntegrationEventPublisher.Publish(ctx, event)`
- Endpoints: none in this package.
- Events: transport event envelopes for sync lifecycle publication.
