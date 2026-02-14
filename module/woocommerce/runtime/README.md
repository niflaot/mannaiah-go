# WooCommerce Runtime Package

`runtime` is the WooCommerce composition root responsible for wiring adapters, application services, routes, startup lifecycle, scheduler hooks, and OpenAPI artifacts.

## Responsibilities
- Build module dependencies from config and injected external services.
- Register WooCommerce HTTP routes.
- Start and stop scheduler lifecycle hooks for cron sync jobs.
- Expose module-level OpenAPI documents.
- Keep integration validation and fail-open route behavior consistent.

## Key Methods / Endpoints / Events
- Methods:
  - `runtime.New(cfg, contactService, orderService, scheduler, logger, publishers...)`
  - `(*runtime.Module).RegisterRoutes(router)`
  - `(*runtime.Module).SetAuthorizer(authorizer)`
  - `(*runtime.Module).OpenAPISpec()`
  - `(*runtime.Module).Load(loader)`
  - `(*runtime.Module).Start(ctx)`
  - `(*runtime.Module).Stop(ctx)`
- Endpoints:
  - `POST /woo/sync/contacts` (registered via HTTP adapter)
  - `POST /woo/sync/orders` (registered via HTTP adapter)
- Events:
  - delegates lifecycle event emission to `application/contact/service`
  - delegates lifecycle event emission to `application/order/service`
