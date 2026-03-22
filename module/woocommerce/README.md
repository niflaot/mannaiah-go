# WooCommerce Module

![Latest Version](https://img.shields.io/badge/latest-v2.9.10-0A66C2)

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
- `adapter/messaging`: cross-module order integration event consumer adapter.

## Key Methods / Endpoints / Events
- Methods:
  - `woocommerce.New(cfg, contactService, orderService, scheduler, logger, publishers...)`
  - `woocommerce.NewWithMessaging(cfg, contactService, orderService, scheduler, logger, registrar, publishers...)`
  - `(*woocommerce.Module).Load(loader)`
  - `(*woocommerce.Module).Start(ctx)`
  - `(*woocommerce.Module).Stop(ctx)`
  - `(*woocommerce.Module).SetAuthorizer(authorizer)`
- Endpoints:
  - `POST /woo/sync/contacts` (`?email=<contact@email>` optional targeted sync).
  - `POST /woo/sync/orders` (`?id=<woo_order_id>` optional targeted sync).
- Events:
  - `woocommerce.v1.contacts.sync.started`
  - `woocommerce.v1.contacts.sync.completed`
  - `woocommerce.v1.contacts.sync.failed`
  - `woocommerce.v1.orders.sync.started`
  - `woocommerce.v1.orders.sync.completed`
  - `woocommerce.v1.orders.sync.failed`
  - Consumes:
    - `orders.v1.created`
    - `orders.v1.updated`
    - `orders.v1.status.updated`
