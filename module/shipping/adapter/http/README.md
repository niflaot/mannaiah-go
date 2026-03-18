# Shipping HTTP Adapter Package

`adapter/http` exposes shipping quote endpoints over the core HTTP abstractions.

## Key Methods / Endpoints / Events
- Methods:
  - `http.NewHandler(service, authorizers...)`
  - `(*http.Handler).RegisterRoutes(router)`
  - `(*http.Handler).SetAuthorizer(authorizer)`
- Endpoints:
  - `POST /shipping/quotes`
- Events: none in this package.
