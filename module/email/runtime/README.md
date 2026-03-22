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
  - `POST /email/webhooks/ses` (public SNS endpoint, signature-verified)
- Events: none.

## Configuration
- `EMAIL_WEBHOOK_SNS_VERIFY_SIGNATURE` (default `true`)
- `EMAIL_WEBHOOK_SNS_TOPIC_ARN` (optional expected TopicArn guard)
- `EMAIL_WEBHOOK_SNS_REQUEST_TIMEOUT_MS` (default `5000`)
- `EMAIL_WEBHOOK_SOFT_BOUNCE_RETRY_DELAY_SECONDS` (default `300`)
- `EMAIL_WEBHOOK_SOFT_BOUNCE_MAX_RETRIES` (default `1`)
