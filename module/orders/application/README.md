# Orders Application Package

`module/orders/application` contains order use-case orchestration.

## Key Methods / Endpoints / Events
- Methods:
  - `application.NewService(repository, customerSource, resolvers...)`
  - `(*application.OrderService).Create(ctx, command)`
  - `(*application.OrderService).Get(ctx, id)`
  - `(*application.OrderService).List(ctx, query)`
  - `(*application.OrderService).UpdateStatus(ctx, id, command)`
- Endpoints: none in this package.
- Events: none in this package.
