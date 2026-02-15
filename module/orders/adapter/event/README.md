# Orders Event Adapter Package

`module/orders/adapter/event` publishes order integration events over the core message bus.

## Key Methods / Endpoints / Events
- Methods:
  - `event.NewPublisher(publisher)`
  - `(*event.Publisher).Publish(ctx, integrationEvent)`
- Endpoints: none in this package.
- Events:
  - `orders.v1.created`
  - `orders.v1.updated`
  - `orders.v1.status.updated`
