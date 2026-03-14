# Membership Module

Auditable consent membership stamping resolved from immutable stamps.

## Key methods / endpoints / events
- Methods:
  - `Module.Service()`
  - `Module.Recorder()`
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
