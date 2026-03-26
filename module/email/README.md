# Email Module

Transactional email delivery module with provider abstraction and status tracking.

## Key methods / endpoints / events
- Methods:
  - `Module.Service()`
- Endpoints:
  - `POST /email/send`
  - `GET /email/deliveries?email=<recipient_email>`
  - `GET /email/deliveries/:id`
  - `POST /email/webhooks/ses` (public SNS endpoint, signature-verified)
- Events: none.
