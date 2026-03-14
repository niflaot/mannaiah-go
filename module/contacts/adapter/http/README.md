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
  - `POST /contacts/optin` (sets `flock_checker_circle_optin=yes`, accepted-at stamps, clears rejected-at stamps)
  - `POST /contacts/optout` (sets `flock_checker_circle_optin=no`, rejected-at stamps, clears accepted-at stamps)
  - `GET /contacts/:id`
  - `PATCH /contacts/:id`
  - `DELETE /contacts/:id`
  - Contact create/update payloads support `metadata` (`map[string]string`).
- Events: triggers application flows that emit `contacts.v1.created` and `contacts.v1.updated`.
