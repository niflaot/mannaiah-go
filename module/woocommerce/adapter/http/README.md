# WooCommerce HTTP Adapter Package

`adapter/http` exposes WooCommerce sync endpoints using core HTTP abstractions.

## Responsibilities
- Register protected sync routes.
- Enforce authorization requirements (`contacts:manage`, `orders:manage`).
- Map sync and integration errors into standard API error payloads.

## Key Methods / Endpoints / Events
- Methods:
  - `http.NewHandler(contactsService, ordersService, authorizers...)`
  - `(*http.Handler).RegisterRoutes(router)`
  - `(*http.Handler).SetAuthorizer(authorizer)`
- Endpoints:
  - `POST /woo/sync/contacts`
  - `POST /woo/sync/orders`
- Events: none in this package.
