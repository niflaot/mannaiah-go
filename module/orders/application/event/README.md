# Orders Application Event Package

`module/orders/application/event` defines integration event contracts/builders for order runtime mutations.

## Key Methods / Endpoints / Events
- Methods:
  - `event.ResolveSource(source)`
  - `event.BuildOrderCreatedIntegrationEvent(entity, source)`
  - `event.BuildOrderUpdatedIntegrationEvent(entity, source)`
  - `event.BuildOrderStatusUpdatedIntegrationEvent(entity, source)`
  - `event.ResolvePublisher(publisher)`
- Endpoints: none in this package.
- Events:
  - `orders.v1.created`
  - `orders.v1.updated`
  - `orders.v1.status.updated`
