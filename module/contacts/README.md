# Contacts Module

`module/contacts` provides DDD/hexagonal contact management use cases and HTTP adapters.

## Packages
- `domain`: contact entities and invariants.
- `port`: repository ports and query contracts.
- `application`: use-case services.
- `adapter/store`: GORM persistence adapter.
- `adapter/http`: HTTP endpoint adapter.
- `adapter/event`: integration event publisher adapter over core messaging bus.

## Key Methods / Endpoints / Events
- Methods:
  - `contacts.New(db, publishers...)`
  - `(*contacts.Module).Load(loader)`
  - `(*contacts.Module).OpenAPISpec() *openapi3.T`
  - `(*contacts.Module).RegisterRoutes(router)`
  - `(*contacts.Module).SetAuthorizer(authorizer)`
- Endpoints:
  - `POST /contacts`
  - `GET /contacts`
  - `GET /contacts/:id`
  - `PATCH /contacts/:id`
  - `DELETE /contacts/:id`
  - Conflict behavior: `POST/PATCH` return `409` when `email` or `(documentType, documentNumber)` already exists.
- Events:
  - domain events: `contacts.contact.created`, `contacts.contact.updated`
  - integration events: `contacts.v1.created`, `contacts.v1.updated`
