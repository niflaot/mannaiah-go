# Contacts HTTP Adapter Package

`adapter/http` maps HTTP endpoints to contact application use cases.

## Key Methods / Endpoints / Events
- Methods:
  - `http.NewHandler(service, authorizers...)`
  - `(*http.Handler).SetAuthorizer(authorizer)`
  - `(*http.Handler).RegisterRoutes(router)`
- Endpoints:
  - `POST /contacts`
  - `GET /contacts`
  - `GET /contacts/:id`
  - `PATCH /contacts/:id`
  - `DELETE /contacts/:id`
- Events: triggers application flows that emit `contacts.v1.created` and `contacts.v1.updated`.
