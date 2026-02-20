# Falabella Module

Falabella integration module for connectivity checks, product synchronization, and async feed status resolution backed by Falabella Seller Center API.

## Key methods / endpoints / events
- Methods: `falabella.New`, `(*falabella.Module).RegisterRoutes`, `(*falabella.Module).SetAuthorizer`, `(*falabella.Module).ConfigureSyncStatus`, `(*falabella.Module).ConfigureScheduler`, `(*falabella.Module).Start`, `(*falabella.Module).Stop`, `(*falabella.Module).Load`, `falabella.OpenAPISpec`
- Endpoints: `GET /falabella/brands`, `POST /falabella/sync/products`, `POST /falabella/sync/products/{id}`, `GET /falabella/sync/status/feed/{feedId}`, `GET /falabella/sync/status/product/{productId}`, `POST /falabella/sync/status/feed/{feedId}/resolve`
- Events: none
