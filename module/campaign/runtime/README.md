# Campaign Runtime Package

Provides campaign module composition-root wiring: repository + service + HTTP adapter registration and OpenAPI artifact exposure.

## Key methods / endpoints / events
- Methods:
  - `New(cfg, db, resolver, sender)`
  - `(*Module).Load(loader)`
  - `(*Module).SetAuthorizer(authorizer)`
  - `(*Module).SetSyncRecorder(recorder)`
  - `(*Module).Service()`
- Endpoints:
  - `POST /campaigns`
  - `GET /campaigns`
  - `GET /campaigns/:id`
  - `PATCH /campaigns/:id`
  - `DELETE /campaigns/:id`
  - `POST /campaigns/:id/send`
- Events: none.
