# HTTP Adapter Package

Inbound Fiber HTTP handlers for Falabella endpoints.

## Key methods / endpoints / events
- Methods: `http.NewHandler`, `(*http.Handler).SetAuthorizer`, `(*http.Handler).RegisterRoutes`
- Endpoints: `GET /falabella/images/transcoded`, `GET /falabella/brands`, `POST /falabella/sync/products`, `POST /falabella/sync/products/{id}`, `GET /falabella/sync/status/feed/{feedId}`, `GET /falabella/sync/status/execution/{executionId}`, `GET /falabella/sync/status/execution/{executionId}/feeds`, `GET /falabella/sync/status/product/{productId}`, `POST /falabella/sync/status/feed/{feedId}/resolve`
- Events: none
