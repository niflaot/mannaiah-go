# Marketing

The marketing domain encompasses **campaigns**, **segments**, **RFM analytics**, **email delivery**, and **product recommendations**. It is split across four Go modules that collaborate through ports and integration events:

| Module | Responsibility |
|---|---|
| `module/campaign` | Campaign lifecycle, template rendering, product blocks, send fan-out |
| `module/segment` | Segment CRUD, filter DSL, contact resolution |
| `module/analytics` | ClickHouse compute: RFM, affinity, recommendations, segment query engine |
| `module/email` | SES delivery, webhook processing, open tracking |

## Data Flow

```
                    ┌───────────┐
                    │ Campaign  │
                    │  Module   │
                    └─────┬─────┘
                          │ send
              ┌───────────┼───────────┐
              ▼           ▼           ▼
        ┌──────────┐ ┌────────┐ ┌──────────┐
        │ Segment  │ │ Email  │ │ Analytics│
        │ Resolver │ │ Sender │ │ Affinity │
        └────┬─────┘ └────┬───┘ └────┬─────┘
             │             │          │
             ▼             ▼          ▼
        ClickHouse     Amazon SES  ClickHouse
        (contacts)     (delivery)  (affinity MVs)
```

When a campaign is sent:

1. The campaign service paginates through the segment (via `SegmentResolver`) to collect all contact IDs.
2. For each contact, it resolves personalization data and product block recommendations (via `AffinityProductProvider`).
3. It renders the Go template with contact data + custom vars + resolved products.
4. UTM tracking links are appended to all HTTP links in the HTML body.
5. The rendered email is dispatched through the email sender with an idempotency key of `campaignID:contactID`.
6. A `campaign.v1.delivery` integration event is published per recipient.

## Permissions

All marketing endpoints require Bearer JWT authentication with the `marketing:manage` permission.

## Subsequent Pages

- [02-CAMPAIGNS.md](02-CAMPAIGNS.md) — Campaign lifecycle and API
- [03-SEGMENTS.md](03-SEGMENTS.md) — Segment CRUD and filter DSL
- [04-RFM.md](04-RFM.md) — RFM scoring, bands, and groups
- [05-EMAIL-TEMPLATES.md](05-EMAIL-TEMPLATES.md) — Template rendering DSL and product blocks
- [06-EMAIL-DELIVERY.md](06-EMAIL-DELIVERY.md) — SES integration, webhooks, open tracking
- [07-AFFINITY.md](07-AFFINITY.md) — Product affinity, correlations, and recommendations
- [08-EVENTS.md](08-EVENTS.md) — Integration events
