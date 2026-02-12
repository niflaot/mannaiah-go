# products/adapter/http/product

Fiber HTTP handlers for product CRUD endpoints.

## Key methods / endpoints / events
- Methods: `NewHandler(service, authorizers...)`, `(*Handler).SetAuthorizer(authorizer)`, `(*Handler).RegisterRoutes(router)`
- Endpoints: `POST /products`, `GET /products`, `GET /products/:id`, `PATCH /products/:id`, `DELETE /products/:id`
- Events: none.
