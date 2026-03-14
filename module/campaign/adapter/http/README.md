# Campaign HTTP Adapter Package

Exposes authenticated campaign CRUD and send endpoints for marketing operators.

## Key methods / endpoints / events
- Methods:
  - `NewHandler(service, authorizers...)`
  - `(*Handler).RegisterRoutes(router)`
  - `(*Handler).SetAuthorizer(authorizer)`
- Endpoints:
  - `POST /campaigns`
  - `GET /campaigns`
  - `GET /campaigns/:id`
  - `PATCH /campaigns/:id`
  - `DELETE /campaigns/:id`
  - `POST /campaigns/:id/send`
- Events: none.
