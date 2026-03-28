# Campaigns

A campaign represents a single email dispatch targeting a segment of contacts. It carries the template, subject line, product block configuration, and send state.

## Domain Model

| Field | Type | Description |
|---|---|---|
| `id` | `UUID` | Primary key |
| `name` | `string` | Human-readable campaign name |
| `slug` | `string` | URL-safe identifier, must match `^[a-z0-9-]+$` |
| `channel` | `string` | Target channel (defaults to `"email"`) |
| `segmentId` | `UUID` | Foreign key to the target segment |
| `subject` | `string` | Email subject line |
| `htmlBody` | `string` | Go template HTML body |
| `textBody` | `string` | Go template plain-text body |
| `status` | `enum` | `PLANNED`, `PROCESSING`, `SENT`, `FAILED` |
| `totalRecipients` | `int` | Resolved audience size |
| `sentCount` | `int` | Successful deliveries |
| `failedCount` | `int` | Failed deliveries |
| `templateVars` | `map[string]string` | Custom key-value pairs for template rendering |
| `productBlocks` | `[]ProductBlock` | Product recommendation block configurations |

## Lifecycle

```
   ┌─────────┐   create   ┌─────────┐
   │ (none)  │───────────▶│ PLANNED │◀──────────┐
   └─────────┘            └────┬────┘           │
                               │ send           │ update (on failure)
                          ┌────▼──────┐         │
                          │PROCESSING │         │
                          └────┬──────┘         │
                     ┌─────────┴─────────┐      │
                     ▼                   ▼      │
                ┌────────┐         ┌────────┐   │
                │  SENT  │         │ FAILED │───┘
                └────────┘         └────────┘
```

- **PLANNED** — Initial state. The campaign can be edited and its template tested.
- **PROCESSING** — Send in progress. An in-memory mutex prevents duplicate sends. Updates are blocked.
- **SENT** — At least one recipient succeeded. Terminal state; updates are blocked.
- **FAILED** — All recipients failed. The campaign can be edited and re-sent.

## Send Pipeline

When `POST /campaigns/:id/send` is called:

1. Validate the campaign exists and is in `PLANNED` or `FAILED` status.
2. Acquire the in-memory send guard (prevents concurrent sends for the same campaign).
3. Set status to `PROCESSING`, reset counters.
4. Launch async goroutine for fan-out:
   - Paginate through `SegmentResolver.ResolveSegment(segmentID, page, 1000)` to collect all contact IDs.
   - Batch-resolve emails via `SegmentResolver.ResolveEmails(contactIDs)`.
   - Skip contacts with no email (marks `skipped_ineligible`).
   - Fan out via bounded worker pool (default **8 workers**, configurable).
   - Each worker: render per-contact template → send via `EmailSender` → publish delivery event.
5. Return the campaign immediately with status `PROCESSING` (HTTP 202).

### Idempotency

Each email send uses the idempotency key `campaignID:contactID`. This prevents duplicate deliveries if the send is retried.

## Test Send

`POST /campaigns/:id/test` sends a single email to a specified address without modifying campaign status or counters. It uses a random UUID idempotency key (`test:campaignID:uuid`) and runs in **strict** mode — template rendering errors are returned to the caller instead of silently skipped.

## API Endpoints

| Method | Path | Status | Description |
|---|---|---|---|
| `POST` | `/campaigns` | 201 | Create a campaign |
| `GET` | `/campaigns` | 200 | List campaigns (paginated) |
| `GET` | `/campaigns/:id` | 200 | Get a single campaign |
| `PATCH` | `/campaigns/:id` | 200 | Update a campaign |
| `DELETE` | `/campaigns/:id` | 200 | Delete a campaign |
| `POST` | `/campaigns/:id/send` | 202 | Start campaign send |
| `POST` | `/campaigns/:id/test` | 202 | Send test email |
| `GET` | `/campaigns/:id/deliveries` | 200 | List delivery results (paginated) |

### Query Parameters

- `GET /campaigns` — `page` (default 1), `limit` (default 20)
- `GET /campaigns/:id/deliveries` — `page` (default 1), `limit` (default 50)

### Error Responses

| Condition | HTTP | Code |
|---|---|---|
| Missing/invalid name, slug, email, template | 400 | `invalid_payload` / `invalid_template` |
| Campaign not found | 404 | `campaign_not_found` |
| Send while PROCESSING/SENT | 409 | `campaign_send_conflict` |
| Email sender not configured | 503 | `email_sender_not_configured` |
| Email sender unavailable | 503 | `email_sender_unavailable` |

## Database Schema

**Table: `campaigns`**

| Column | Type | Notes |
|---|---|---|
| `id` | VARCHAR (PK) | UUID |
| `name` | VARCHAR | |
| `slug` | VARCHAR | Regex-validated |
| `channel` | VARCHAR | Defaults to `"email"` |
| `segment_id` | VARCHAR | FK to segments |
| `subject` | VARCHAR | |
| `html_body` | TEXT | Go template |
| `text_body` | TEXT | Go template |
| `status` | VARCHAR | Lifecycle state |
| `total_recipients` | INT | |
| `sent_count` | INT | |
| `failed_count` | INT | |
| `template_vars` | JSON/TEXT | Default `"{}"` |
| `product_blocks` | JSON/TEXT | Default `"[]"` |
| `created_at` | DATETIME | |
| `updated_at` | DATETIME | |

Product blocks and template vars are stored as JSON strings with NOT NULL constraints and valid defaults.
