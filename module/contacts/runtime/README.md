# Contacts Runtime Package

`runtime` is the contacts module composition root responsible for schema setup, adapter wiring, route registration, and module OpenAPI exposure.

## Responsibilities
- Build contacts module dependencies from infrastructure inputs.
- Ensure contacts schema availability on startup.
- Register contacts HTTP routes.
- Expose module-level OpenAPI spec for aggregation.

## Key Methods / Endpoints / Events
- Methods:
  - `runtime.New(db, publishers...)`
  - `(*runtime.Module).RegisterRoutes(router)`
  - `(*runtime.Module).Service()`
  - `(*runtime.Module).SetAuthorizer(authorizer)`
  - `(*runtime.Module).OpenAPISpec()`
  - `(*runtime.Module).Load(loader)`
- Endpoints:
  - `POST /contacts`
  - `GET /contacts` (supports metadata query filters: `metadataKey`, `metadataValue`)
  - `POST /contacts/optin` (sets `flock_checker_circle_optin=yes`, accepted-at stamps, clears rejected-at stamps)
  - `POST /contacts/optout` (sets `flock_checker_circle_optin=no`, rejected-at stamps, clears accepted-at stamps)
  - `GET /contacts/{id}`
  - `PATCH /contacts/{id}`
  - `DELETE /contacts/{id}`
- Events:
  - delegated through contacts application/event adapters
