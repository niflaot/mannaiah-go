# Core API Command

`cmd/coreapi` bootstraps infrastructure and loads workspace modules into runtime.

## Key Methods / Endpoints / Events
- Methods:
  - `main()`
  - `run(ctx, envFile)`
  - `registerCoreStatusRoute(router)`
  - `waitForShutdown(ctx, db, httpServer, messaging, serverErrors, messagingErrors)`
  - `shutdownResources(db, httpServer, messaging)`
- Endpoints:
  - `GET /status`
  - `GET /openapi.json`
  - `GET /docs`
  - plus any module endpoints loaded at startup
- Events:
  - starts core in-memory messaging platform and module integration-event publication pipeline
  - initializes auth module and injects authentication/authorization into module endpoints
