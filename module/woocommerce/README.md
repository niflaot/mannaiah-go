# WooCommerce Module

`module/woocommerce` provides WooCommerce integration capabilities for the monolith with DDD + hexagonal boundaries and future microservice extraction in mind.

## Scope
- WooCommerce adapter and sync orchestration.
- Contact synchronization from WooCommerce orders into the contacts module.
- Manual and scheduled sync triggers.
- Integration events for sync lifecycle visibility.
- Configurable circuit-breaker protection for WooCommerce source outages.

## Architecture
- `runtime/`: module composition root wiring (constructor, lifecycle, source/bootstrap helpers, module OpenAPI artifact).
- `port/`: provider-agnostic contracts.
- `application/`: application-layer feature namespace.
- `application/contact`: contacts feature use cases and integration event mapping.
- `adapter/woocommerce`: WooCommerce SDK adapter (`github.com/jmolboy/woocommerce-go`).
- `adapter/contacts`: contacts upsert adapter via contacts application service.
- `adapter/http`: protected sync endpoint adapters.
- `adapter/event`: core messaging publication adapter.

## Key Methods / Endpoints / Events
- Methods:
  - `woocommerce.New(cfg, contactService, scheduler, logger, publishers...)`
  - `(*woocommerce.Module).Load(loader)`
  - `(*woocommerce.Module).Start(ctx)`
  - `(*woocommerce.Module).Stop(ctx)`
  - `(*woocommerce.Module).SetAuthorizer(authorizer)`
- Endpoints:
  - `POST /woo/sync/contacts`
- Events:
  - `woocommerce.v1.contacts.sync.started`
  - `woocommerce.v1.contacts.sync.completed`
  - `woocommerce.v1.contacts.sync.failed`
