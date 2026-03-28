# Integration Events

The marketing system produces and consumes integration events for cross-module communication. Events are serialized as `bus.Message` over the core messaging bus.

## Produced Events

### `campaign.v1.delivery`

Published by the campaign module per-recipient during send fan-out.

**Payload:**

| Field | Type | JSON | Description |
|---|---|---|---|
| `campaignId` | `string` | `campaignId` | Campaign that triggered the delivery |
| `contactId` | `string` | `contactId` | Target contact |
| `channel` | `string` | `channel` | Delivery channel (e.g., `"email"`) |
| `status` | `string` | `status` | Delivery outcome |
| `templateVersion` | `int` | `templateVersion` | Always `1` |
| `occurredAt` | `time.Time` | `occurredAt` | Event timestamp |

**Status values:**

| Status | Meaning |
|---|---|
| `submitted_to_provider` | Email dispatched successfully to SES |
| `failed` | Email dispatch failed |
| `skipped_ineligible` | Contact had no email address |

**Envelope:**

Each event is wrapped in an `IntegrationEvent` envelope:

| Field | Type | Description |
|---|---|---|
| `id` | `string` | Random 32-hex-char event ID |
| `topic` | `string` | `"campaign.v1.delivery"` |
| `schemaVersion` | `string` | `"v1"` |
| `occurredAt` | `time.Time` | Event timestamp |
| `payload` | `any` | Serialized delivery payload |
| `metadata` | `map[string]string` | Transport metadata |

**Metadata keys:** `campaign_id`, `contact_id`, `status`, `schema_version`, `produced_at`.

## Consumed Events

The analytics module ingests integration events from other domains to keep ClickHouse synchronized:

| Topic | Source Module | Action |
|---|---|---|
| `contacts.v1.created` | Contacts | Upsert contact snapshot |
| `contacts.v1.updated` | Contacts | Upsert contact snapshot |
| `orders.v1.created` | Orders | Upsert order fact + item facts |
| `orders.v1.updated` | Orders | Upsert order fact + item facts |
| `orders.v1.status.updated` | Orders | Upsert order fact + item facts |
| `membership.v1.changed` | Membership | Insert membership event |
| `campaign.v1.delivery` | Campaign | Insert campaign event |

All handlers are fail-open for transient errors (retried by the bus) and non-retriable for invalid/malformed payloads (discarded permanently).

## Event Flow During Campaign Send

```
Campaign Send Fan-out
         │
         ├──▶ EmailSender.Send() ─── per contact ──▶ SES
         │
         └──▶ Publisher.Publish("campaign.v1.delivery")
                     │
                     ▼
              Messaging Bus
                     │
                     ▼
              Analytics Module
                     │
                     ▼
              ClickHouse: campaign_events
                     │
                     ▼
              (used by segment filters, RFM refresh, etc.)
```

## Modules Without Events

- **Segment module** — Pure CRUD + query delegation, no events.
- **Email module** — No integration events. Status updates arrive via SES/SNS webhooks, not through the messaging bus.
