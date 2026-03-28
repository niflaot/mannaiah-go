# Email Delivery

The email module handles SES integration, delivery lifecycle tracking, SNS webhook processing, and open-rate tracking via invisible pixel injection.

## Architecture

```
Campaign Module                    Email Module                         AWS
┌──────────────┐    Send()     ┌───────────────┐    sesv2.SendEmail  ┌─────┐
│ Send Worker  │──────────────▶│ EmailService  │────────────────────▶│ SES │
└──────────────┘               └───────┬───────┘                     └──┬──┘
                                       │ persist                        │
                               ┌───────▼───────┐                       │ SNS
                               │ email_deliver. │◀──────────────────────┘
                               │ email_del_stat │     webhook
                               └───────────────┘
```

## Delivery Lifecycle

```
pending ─── Send() ──▶ submitted_to_provider
                            │
                 ┌──────────┼──────────┬──────────┐
                 ▼          ▼          ▼          ▼
            delivered    bounced   complained   failed_retryable
                 │                                    │
                 ▼                               soft-bounce retry
              opened                                  │
                                              submitted_to_provider
```

| Status | Meaning |
|---|---|
| `pending` | Created but not yet dispatched |
| `submitted_to_provider` | Sent to SES successfully |
| `delivered` | SES confirmed delivery to recipient's mail server |
| `bounced` | Hard bounce — address is permanently undeliverable |
| `complained` | Recipient marked the email as spam |
| `opened` | Recipient loaded the tracking pixel |
| `failed_retryable` | Transient failure, eligible for retry |
| `failed_permanent` | Permanent failure, no retry |

## SES Provider

The SES adapter uses AWS SES v2 SDK (`sesv2.SendEmailInput`) with `Simple` message type:

- **Credentials**: Static `AccessKeyID`/`SecretAccessKey` via config, or falls back to the default AWS credential chain when keys are blank.
- **Sender format**: `"SenderName <address>"` when `EMAIL_SES_FROM_NAME` is set.
- **Idempotency tagging**: Each email carries an `idempotency_key` SES message tag (non-alphanumeric characters sanitized to `_`).
- **One email per call**: No batching or parallelism at the module level. The `EMAIL_SES_MAX_SEND_RATE` config exists as a placeholder but is not consumed.

## Invisible Pixel (Open Tracking)

When `EMAIL_TRACKING_BASE_URL` is configured (or derived from the sender domain), the service injects a 1×1 transparent GIF image tag before `</body>` in the HTML body:

```html
<img src="https://app.example.com/email/track/open/{deliveryID}"
     width="1" height="1" style="display:none" alt="" />
```

When the recipient's email client loads the image:

1. `GET /email/track/open/:id` is called (no authentication required).
2. The service records the `opened` status with a `StatusEntry`.
3. Returns a 43-byte transparent GIF with `Cache-Control: no-cache, no-store, must-revalidate`.

**Pixel URL resolution**: uses `EMAIL_TRACKING_BASE_URL` if set; otherwise derives from the sender address domain as `https://{domain}`.

## SNS Webhook Processing

SES delivery notifications arrive via Amazon SNS. The webhook endpoint at `POST /email/webhooks/ses` is public but verified:

### Webhook Pipeline

1. **Decode**: Tries 4 strategies — raw JSON, embedded (string-wrapped) JSON, form-encoded, body-parser fallback.
2. **Verify**: Checks SNS message signature using the signing certificate (HTTPS + `*.amazonaws.com` + contains `sns` in hostname). Supports signature versions 1 (SHA-1) and 2 (SHA-256).
3. **Topic guard**: Validates `TopicARN` matches `EMAIL_WEBHOOK_SNS_TOPIC_ARN` if configured.
4. **Route by message type**:
   - `SubscriptionConfirmation` → GETs the `SubscribeURL` (HTTPS + `*.amazonaws.com` only)
   - `Notification` → Parses nested SES JSON and maps to delivery status
   - `UnsubscribeConfirmation` → Logged, no action

### SES Notification Mapping

| SES Event | Domain Status | Side Effect |
|---|---|---|
| `Delivery` | `delivered` | — |
| `Bounce` (permanent) | `bounced` | Opt-out via `MembershipStamper` |
| `Bounce` (transient) | `failed_retryable` | Schedule soft-bounce retry |
| `Complaint` | `complained` | Opt-out via `MembershipStamper` |
| `Reject` | `failed_permanent` | — |
| `Rendering Failure` | `failed_permanent` | — |

### Soft-Bounce Retry

Transient bounces trigger an automatic retry after a configurable delay (default 300 seconds). The retry:

1. Waits `EMAIL_WEBHOOK_SOFT_BOUNCE_RETRY_DELAY_SECONDS`.
2. Re-creates a new delivery with suffixed idempotency key: `originalKey:retry1`.
3. Re-injects the tracking pixel before re-sending.
4. Maximum retries: `EMAIL_WEBHOOK_SOFT_BOUNCE_MAX_RETRIES` (default 1).

## Database Schema

**Table: `email_deliveries`**

| Column | Type | Notes |
|---|---|---|
| `id` | VARCHAR (PK) | UUID |
| `contact_id` | VARCHAR | |
| `email` | VARCHAR | |
| `subject` | VARCHAR | |
| `html_body` | TEXT | |
| `text_body` | TEXT | |
| `idempotency_key` | VARCHAR | `campaignID:contactID` format |
| `provider` | VARCHAR | `"ses"` |
| `provider_message_id` | VARCHAR (nullable) | Set after SES confirms |
| `status` | VARCHAR | Current lifecycle state |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | |

**Table: `email_delivery_status_history`**

| Column | Type | Notes |
|---|---|---|
| `id` | VARCHAR (PK) | UUID |
| `delivery_id` | VARCHAR | FK to `email_deliveries.id` |
| `status` | VARCHAR | |
| `reason` | VARCHAR | Bounce/complaint reason |
| `occurred_at` | DATETIME | UTC |
| `created_at` | DATETIME | UTC |

## API Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/email/send` | Bearer JWT | Dispatch one email |
| `GET` | `/email/deliveries` | Bearer JWT | List deliveries by `?email=` |
| `GET` | `/email/deliveries/:id` | Bearer JWT | Get one delivery |
| `POST` | `/email/webhooks/ses` | Public (SNS sig) | Receive SES/SNS webhooks |
| `GET` | `/email/track/open/:id` | Public | Open-tracking pixel |

## Configuration

| Env Var | Default | Purpose |
|---|---|---|
| `EMAIL_ENABLED` | `false` | Master toggle |
| `EMAIL_PROVIDER` | `"ses"` | Provider label |
| `EMAIL_SES_REGION` | — | AWS SES region |
| `EMAIL_SES_ACCESS_KEY_ID` | — | Static AWS key |
| `EMAIL_SES_SECRET_ACCESS_KEY` | — | Static AWS secret |
| `EMAIL_SES_FROM_ADDRESS` | — | Sender address |
| `EMAIL_SES_FROM_NAME` | — | Sender display name |
| `EMAIL_SES_MAX_SEND_RATE` | 14 | Max send rate/sec (placeholder) |
| `EMAIL_TRACKING_BASE_URL` | — | Open-tracking pixel base URL |
| `EMAIL_WEBHOOK_SNS_TOPIC_ARN` | — | Expected SNS topic ARN |
| `EMAIL_WEBHOOK_SNS_VERIFY_SIGNATURE` | `true` | Enable signature verification |
| `EMAIL_WEBHOOK_SNS_REQUEST_TIMEOUT_MS` | 5000 | SNS HTTP timeout |
| `EMAIL_WEBHOOK_SOFT_BOUNCE_RETRY_DELAY_SECONDS` | 300 | Soft-bounce retry delay |
| `EMAIL_WEBHOOK_SOFT_BOUNCE_MAX_RETRIES` | 1 | Max soft-bounce retries |
