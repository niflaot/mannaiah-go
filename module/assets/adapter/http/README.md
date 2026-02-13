# assets/adapter/http

Fiber HTTP adapter for asset upload and metadata CRUD endpoints.

## Key methods / endpoints / events
- Methods: `NewHandler(service, authorizer...)`, `(*Handler).RegisterRoutes`, `(*Handler).SetAuthorizer`
- Endpoints: `POST /assets`, `GET /assets`, `GET /assets/{id}`, `PATCH /assets/{id}`, `DELETE /assets/{id}`, `POST /assets/folders`, `GET /assets/folders`, `GET /assets/folders/{id}`, `PATCH /assets/folders/{id}`, `DELETE /assets/folders/{id}`
- Events: none directly; emitted by application service.
