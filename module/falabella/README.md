# Falabella Module

Falabella integration module for connectivity checks and product synchronization backed by `falabella-go`.

## Key methods / endpoints / events
- Methods: `falabella.New`, `(*falabella.Module).RegisterRoutes`, `(*falabella.Module).SetAuthorizer`, `(*falabella.Module).Load`, `falabella.OpenAPISpec`
- Endpoints: `GET /falabella/brands`, `POST /falabella/sync/products`, `POST /falabella/sync/products/{id}`
- Events: none
