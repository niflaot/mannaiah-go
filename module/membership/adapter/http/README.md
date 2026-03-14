# Membership HTTP Adapter

Exposes consent stamp and status endpoints.

## Key methods / endpoints / events
- Methods:
  - `NewHandler(...)`
  - `(*Handler).RegisterRoutes(...)`
- Endpoints:
  - `POST /membership/stamp`
  - `GET /membership/status/:contactId`
  - `GET /membership/status/:contactId/stamps`
  - `POST /membership/migrate`
- Events: none.
