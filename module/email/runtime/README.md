# Email Runtime Package

Composition root for email module wiring and OpenAPI registration.

## Key methods / endpoints / events
- Methods:
  - `New(cfg, db)`
  - `(*Module).Load(loader)`
  - `(*Module).Service()`
  - `(*Module).SetMembershipStamper(...)`
- Endpoints:
  - `POST /email/send`
  - `GET /email/deliveries/:id`
  - `POST /email/webhooks/ses`
- Events: none.
