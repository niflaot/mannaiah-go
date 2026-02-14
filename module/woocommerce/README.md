# WooCommerce Module

`module/woocommerce` provides WooCommerce integration capabilities for the monolith with DDD + hexagonal boundaries and future microservice extraction in mind.

## Scope
- WooCommerce adapter and sync orchestration.
- Contact synchronization from WooCommerce orders into the contacts module.
- Order synchronization from WooCommerce orders into the orders module.
- Contact `createdAt` alignment with oldest WooCommerce order dates per email.
- Manual and scheduled sync triggers.
- Integration events for sync lifecycle visibility.
- Configurable circuit-breaker protection for WooCommerce source outages.

## Architecture
- `runtime/`: module composition root wiring (constructor, lifecycle, source/bootstrap helpers, module OpenAPI artifact).
- `port/`: provider-agnostic contracts.
- `application/`: application-layer feature namespace.
- `application/contact`: contact feature namespace package.
- `application/contact/service`: contact sync use case orchestration.
- `application/contact/event`: contact integration event contracts/builders.
- `adapter/woocommerce`: WooCommerce SDK adapter (`github.com/jmolboy/woocommerce-go`).
- `adapter/contacts`: contacts upsert adapter via contacts application service.
- `adapter/orders`: orders upsert adapter via orders + contacts application services.
- `adapter/http`: protected sync endpoint adapters.
- `adapter/event`: core messaging publication adapter.

## Key Methods / Endpoints / Events
- Methods:
  - `woocommerce.New(cfg, contactService, orderService, scheduler, logger, publishers...)`
  - `(*woocommerce.Module).Load(loader)`
  - `(*woocommerce.Module).Start(ctx)`
  - `(*woocommerce.Module).Stop(ctx)`
  - `(*woocommerce.Module).SetAuthorizer(authorizer)`
- Endpoints:
  - `POST /woo/sync/contacts`
  - `POST /woo/sync/orders`
- Events:
  - `woocommerce.v1.contacts.sync.started`
  - `woocommerce.v1.contacts.sync.completed`
  - `woocommerce.v1.contacts.sync.failed`
  - `woocommerce.v1.orders.sync.started`
  - `woocommerce.v1.orders.sync.completed`
  - `woocommerce.v1.orders.sync.failed`
