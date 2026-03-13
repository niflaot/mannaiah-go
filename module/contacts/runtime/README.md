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
  - `POST /contacts/optin` (updates circle opt-in metadata by email)
  - `POST /contacts/optout` (updates circle opt-in metadata by email)
  - `GET /contacts/{id}`
  - `PATCH /contacts/{id}`
  - `DELETE /contacts/{id}`
- Events:
  - delegated through contacts application/event adapters
