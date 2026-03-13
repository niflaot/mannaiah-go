# Contacts HTTP Adapter Package

`adapter/http` maps HTTP endpoints to contact application use cases.

## Key Methods / Endpoints / Events
- Methods:
  - `http.NewHandler(service, authorizers...)`
  - `(*http.Handler).SetAuthorizer(authorizer)`
  - `(*http.Handler).RegisterRoutes(router)`
- Endpoints:
  - `POST /contacts`
  - `GET /contacts` (supports `metadataKey` and `metadataValue` query filters)
  - `POST /contacts/optin` (updates `flock_checker_circle_optin*` metadata by email)
  - `POST /contacts/optout` (updates `flock_checker_circle_optin*` metadata by email)
  - `GET /contacts/:id`
  - `PATCH /contacts/:id`
  - `DELETE /contacts/:id`
  - Contact create/update payloads support `metadata` (`map[string]string`).
- Events: triggers application flows that emit `contacts.v1.created` and `contacts.v1.updated`.
