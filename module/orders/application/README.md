# Orders Application Package

`module/orders/application` contains order use-case orchestration.

## Key Methods / Endpoints / Events
- Methods:
  - `application.NewService(repository, customerSource, resolvers...)`
  - `application.NewServiceWithPublisher(repository, customerSource, publisher, resolvers...)`
  - `(*application.OrderService).Create(ctx, command)`
  - `(*application.OrderService).Update(ctx, id, command)`
  - `(*application.OrderService).Get(ctx, id)`
  - `(*application.OrderService).List(ctx, query)`
  - `(*application.OrderService).UpdateStatus(ctx, id, command)`
  - `(*application.OrderService).AddComment(ctx, id, command)`
  - `(*application.OrderService).UpdateComment(ctx, id, commentID, command)`
  - `(*application.OrderService).DeleteComment(ctx, id, commentID, command)`
- Endpoints: none in this package.
- Events:
  - `orders.v1.created`
  - `orders.v1.updated`
  - `orders.v1.status.updated`
