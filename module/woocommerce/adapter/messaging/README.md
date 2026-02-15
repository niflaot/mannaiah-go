# WooCommerce Messaging Adapter Package

`module/woocommerce/adapter/messaging` consumes order integration events and forwards mainstream-origin order updates to WooCommerce.

## Key Methods / Endpoints / Events
- Methods:
  - `messaging.NewOrderConsumer(handler, logger)`
  - `(*messaging.OrderConsumer).Register(registrar)`
- Endpoints: none in this package.
- Events:
  - `orders.v1.created`
  - `orders.v1.updated`
  - `orders.v1.status.updated`
