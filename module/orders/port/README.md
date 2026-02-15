# Orders Port Package

`module/orders/port` defines order persistence and external lookup contracts.

## Key Methods / Endpoints / Events
- Methods:
  - `port.Repository.Create(ctx, order)`
  - `port.Repository.Update(ctx, order)`
  - `port.Repository.GetByID(ctx, id)`
  - `port.Repository.List(ctx, query)`
  - `port.Repository.AppendStatus(ctx, id, entry)`
  - `port.IntegrationEventPublisher.Publish(ctx, event)`
  - `port.CustomerSource.GetByID(ctx, id)`
  - `port.ProductResolver.Resolve(ctx, sku, alternateName)`
- Endpoints: none in this package.
- Events:
  - `orders.v1.created`
  - `orders.v1.updated`
  - `orders.v1.status.updated`
