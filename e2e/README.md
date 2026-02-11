# System E2E Package

`e2e` contains root-level system end-to-end scenarios for the assembled modular monolith.

## Scope
- Validates cross-module runtime behavior from a composition-root perspective.
- Keeps scenario files modular (`auth/events`, `config/redis`, `database`) instead of one large suite.
- Provides step-by-step traceability through Zap step logs.
- Includes black-box startup process validation by running `go run ./module/core/cmd/api`.
- Includes resilience scenarios for authentication, database-connection failures, messaging publication failures, and concurrent uniqueness races.
- Includes benchmark coverage for hot persistence paths under module-level benchmark suites.
- Includes scheduler configuration and execution validation for the core cron package.

## Key Methods / Endpoints / Events
- Methods:
  - `newStepTracer(t)`
  - `newContactsE2EHarness(t)`
  - `(*contactsE2EHarness).DoJSONRequest(t, method, path, token, body)`
  - `doJSONRequestRaw(server, method, path, token, body)`
  - `(*contactsE2EHarness).CloseDatabase(t)`
  - `(*contactsE2EHarness).SignToken(t, scopes)`
  - `(*contactsE2EHarness).AwaitCreatedEvent(t)`
  - `(*contactsE2EHarness).AwaitUpdatedEvent(t)`
  - `waitForCronRun(t, signal, timeout)`
- Endpoints:
  - `POST /contacts`
  - `GET /contacts`
  - `GET /contacts/:id`
  - `PATCH /contacts/:id`
  - `DELETE /contacts/:id`
- Events:
  - `contacts.v1.created`
  - `contacts.v1.updated`
