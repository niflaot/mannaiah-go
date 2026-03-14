# Membership HTTP Adapter

Exposes consent stamp and status endpoints.

## Key methods / endpoints / events
- Methods:
  - `NewHandler(...)`
  - `(*Handler).RegisterRoutes(...)`
- Endpoints:
  - `POST /membership/optin`
  - `POST /membership/optout`
  - `POST /membership/stamp`
  - `GET /membership/status/:contactId`
  - `GET /membership/status/:contactId/:channel`
  - `GET /membership/status/:contactId/stamps`
  - `GET /membership/stamps/:contactId/:channel`
- Events: none.
