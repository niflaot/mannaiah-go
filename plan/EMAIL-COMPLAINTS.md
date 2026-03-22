# Email Complaints/Bounces Plan

## Goal
Implement SES feedback handling so bounces/complaints are reflected in delivery status history and unsafe recipients are automatically opted out of email marketing.

## Backend Behavior (Implemented)

### Public endpoint
- Endpoint: `POST /email/webhooks/ses`
- Public (no JWT), protected by SNS signature verification (enabled by default).
- Accepts Amazon SNS envelope messages:
  - `SubscriptionConfirmation` -> backend auto-confirms via `SubscribeURL`.
  - `Notification` -> backend parses SES event payload.
  - `UnsubscribeConfirmation` -> acknowledged (no-op).

### Security checks
- SNS signature verification for incoming webhook messages.
- Signature supports SNS `SignatureVersion` `1` and `2`.
- Signing certificate is fetched only via `https://...amazonaws.com`.
- Optional TopicArn guard:
  - when configured, messages from other topics are rejected.

### SES event mapping to delivery status
- `Delivery` -> `delivered`
- `Complaint` -> `complained`
  - auto membership opt-out source: `ses_complaint`
- `Bounce`:
  - `bounceType=Transient` -> `failed_retryable`
    - schedules automatic retry after configured delay
  - `bounceType=Permanent` / `Undetermined` -> `bounced`
    - auto membership opt-out source: `ses_bounce_permanent`
- `Reject` / `RenderingFailure` -> `failed_permanent`

### Soft bounce retry policy
- Configurable delayed retry attempts for transient bounces.
- Retry submits delivery again and updates delivery to:
  - `submitted_to_provider` on retry submit success
  - `failed_retryable` on retry submit failure

### Membership auto-optout behavior
- Complaints and hard bounces call membership stamper:
  - channel: `email`
  - action: `opt_out`
  - source: `ses_complaint` or `ses_bounce_permanent`

## Required Environment Variables
- `EMAIL_WEBHOOK_SNS_VERIFY_SIGNATURE` (default `true`)
- `EMAIL_WEBHOOK_SNS_TOPIC_ARN` (optional but recommended)
- `EMAIL_WEBHOOK_SNS_REQUEST_TIMEOUT_MS` (default `5000`)
- `EMAIL_WEBHOOK_SOFT_BOUNCE_RETRY_DELAY_SECONDS` (default `300`)
- `EMAIL_WEBHOOK_SOFT_BOUNCE_MAX_RETRIES` (default `1`)

## SES/SNS Setup Steps
1. In SES, create/select a Configuration Set.
2. Add an Event Destination of type SNS.
3. Select event types at least:
   - `Delivery`
   - `Bounce`
   - `Complaint`
   - recommended: `Reject`, `RenderingFailure`
4. Create an SNS topic in the same region and attach policy to allow SES publish from your configuration set ARN.
5. Subscribe your backend endpoint to the SNS topic:
   - `https://<your-api-domain>/email/webhooks/ses`
6. Ensure endpoint is publicly reachable via HTTPS.
7. Set `EMAIL_SES_CONFIGURATION_SET` so outbound emails are sent with that config set.
8. Set webhook env vars above and restart service.
9. Verify subscription is confirmed in SNS (`SubscriptionArn` not `PendingConfirmation`).
10. Test flow with SES mailbox simulator addresses (bounce/complaint scenarios).

## Frontend Contract / UX Tasks
1. Delivery status labels
- Handle and display:
  - `submitted_to_provider`
  - `delivered`
  - `failed_retryable`
  - `failed_permanent`
  - `bounced`
  - `complained`
  - `opened`

2. Retry transparency
- For `failed_retryable`, show:
  - “temporary delivery issue”
  - “automatic retry scheduled/running”

3. Suppression visibility
- Add clear badges for:
  - `complained` => “recipient unsubscribed automatically”
  - `bounced` (hard) => “recipient unsubscribed automatically”

4. Optional timeline panel
- Show status history entries (`reason`, `occurredAt`) when available to explain complaints/bounces/retries.

5. Filtering
- Add filters for `bounced`, `complained`, `failed_retryable`, `failed_permanent` in sent-email tables.

## Ops Verification Checklist
- Webhook endpoint returns `200` for valid SNS notifications.
- Invalid signature returns `401`.
- Wrong topic ARN returns `403` (when topic guard enabled).
- Complaint/hard-bounce events create membership opt-out stamps.
- Transient bounces create retryable status and trigger retries as configured.
