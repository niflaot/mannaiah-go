# WooCommerce HTTP Adapter Package

`adapter/http` exposes WooCommerce sync endpoints using core HTTP abstractions.

## Responsibilities
- Register protected sync routes.
- Enforce authorization requirements (`contacts:manage`).
- Map sync and integration errors into standard API error payloads.

## Key Methods / Endpoints / Events
- Methods:
  - `http.NewHandler(service, authorizers...)`
  - `(*http.Handler).RegisterRoutes(router)`
  - `(*http.Handler).SetAuthorizer(authorizer)`
- Endpoints:
  - `POST /woo/sync/contacts`
- Events: none in this package.
