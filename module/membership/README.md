# Membership Module

Auditable consent membership stamping with status snapshots.

## Key methods / endpoints / events
- Methods:
  - `Module.Service()`
  - `Module.Recorder()`
- Endpoints:
  - `POST /membership/stamp`
  - `GET /membership/status/:contactId`
  - `GET /membership/status/:contactId/stamps`
  - `POST /membership/migrate`
- Events:
  - `membership.v1.changed`
