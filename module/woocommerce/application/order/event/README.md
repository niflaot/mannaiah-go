# WooCommerce Order Event Package

`application/order/event` defines WooCommerce order sync integration event contracts and payload builders.

## Key Methods / Endpoints / Events
- Methods:
  - `event.ResolvePublisher(publisher)`
  - `event.NewSyncStartedEvent(trigger)`
  - `event.NewSyncCompletedEvent(summary)`
  - `event.NewSyncFailedEvent(summary, syncErr)`
- Endpoints: none in this package.
- Events:
  - `woocommerce.v1.orders.sync.started`
  - `woocommerce.v1.orders.sync.completed`
  - `woocommerce.v1.orders.sync.failed`
