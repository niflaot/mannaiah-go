# WooCommerce Order Service Package

`application/order/service` orchestrates WooCommerce order synchronization without embedding domain logic.

## Key Methods / Endpoints / Events
- Methods:
  - `service.NewService(cfg, source, target, publisher, logger, breakers...)`
  - `(*service.OrderSyncService).ValidateIntegration(ctx)`
  - `(*service.OrderSyncService).SyncOrders(ctx, trigger)`
- Endpoints: none in this package.
- Events:
  - `woocommerce.v1.orders.sync.started`
  - `woocommerce.v1.orders.sync.completed`
  - `woocommerce.v1.orders.sync.failed`
