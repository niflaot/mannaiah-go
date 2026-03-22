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
  - `POST /campaigns/:id/test`
- Events: none.

## Configuration
- `MN_PUBLIC_URL`: Public frontend base URL used to build `.Custom.unsubscribe_url` as `${MN_PUBLIC_URL}/public/marketing/optout/{token}`.
- `MN_MARKETING_OPTOUT_SECRET`: HMAC secret used to sign unsubscribe tokens.
- `MN_MARKETING_OPTOUT_TOKEN_TTL_HOURS`: Expiration window in hours for generated unsubscribe tokens (default `720`).
