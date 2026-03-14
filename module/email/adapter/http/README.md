# Email HTTP Adapter

Exposes send and SES webhook endpoints.

## Key methods / endpoints / events
- Methods:
  - `NewHandler(...)`
  - `(*Handler).RegisterRoutes(...)`
- Endpoints:
  - `POST /email/send`
  - `GET /email/deliveries/:id`
  - `POST /email/webhooks/ses`
- Events: none.
