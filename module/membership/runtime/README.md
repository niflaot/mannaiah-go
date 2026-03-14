# Membership Runtime Package

Composition root for membership wiring and OpenAPI registration.

## Key methods / endpoints / events
- Methods:
  - `New(cfg, db, contacts, publishers...)`
  - `(*Module).Load(loader)`
  - `(*Module).Service()`
- Endpoints:
  - `POST /membership/optin`
  - `POST /membership/optout`
  - `POST /membership/stamp`
  - `GET /membership/status/:contactId`
  - `GET /membership/status/:contactId/:channel`
  - `GET /membership/status/:contactId/stamps`
  - `GET /membership/stamps/:contactId/:channel`
- Events:
  - `membership.v1.changed`
