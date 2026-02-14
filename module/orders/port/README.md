# Orders Port Package

`module/orders/port` defines order persistence and external lookup contracts.

## Key Methods / Endpoints / Events
- Methods:
  - `port.Repository.Create(ctx, order)`
  - `port.Repository.GetByID(ctx, id)`
  - `port.Repository.List(ctx, query)`
  - `port.Repository.AppendStatus(ctx, id, entry)`
  - `port.CustomerSource.GetByID(ctx, id)`
  - `port.ProductResolver.Resolve(ctx, sku, alternateName)`
- Endpoints: none in this package.
- Events: none in this package.
