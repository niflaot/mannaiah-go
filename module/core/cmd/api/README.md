# API Command

`cmd/api` bootstraps infrastructure and loads workspace modules into runtime.

## Key Methods / Endpoints / Events
- Methods:
  - `main()`
  - `run(ctx, envFile)`
  - `registerCoreStatusRoute(router)`
  - `waitForShutdown(ctx, db, httpServer, messaging, serverErrors, messagingErrors)`
  - `shutdownResources(db, httpServer, messaging)`
- Endpoints:
  - `GET /status`
  - `GET /check-auth`
  - `GET /openapi.json`
  - `GET /docs`
  - plus any module endpoints loaded at startup
  - includes `orders` endpoints when orders module is enabled in startup wiring
- Events:
  - starts core in-memory messaging platform and module integration-event publication pipeline
  - initializes auth module and injects authentication/authorization into module endpoints
