# Contacts — Integration Events

The contacts module publishes integration events to the in-process message bus whenever a contact
is created or updated. Other modules subscribe to these topics to propagate changes without
direct coupling.

---

## Published Topics

### `contacts.v1.created`

Emitted after a new contact is persisted successfully.

**Payload** — `ContactEventPayload`

```json
{
  "id": "uuid",
  "documentType": "CC",
  "documentNumber": "1234567890",
  "legalName": "",
  "firstName": "Ana",
  "lastName": "García",
  "email": "ana.garcia@example.com",
  "phone": "+57 310 000 0000",
  "address": "Calle 123 # 45-67",
  "addressExtra": "Apto 8",
  "cityCode": "BOG",
  "metadata": { "woo_customer_id": "42" },
  "createdAt": "2026-01-15T10:00:00Z",
  "updatedAt": "2026-01-15T10:00:00Z"
}
```

---

### `contacts.v1.updated`

Emitted after a contact is updated successfully.

**Payload** — same `ContactEventPayload` shape as above, reflecting the state _after_ the update.

---

## Event Envelope

Events are wrapped in a standard `IntegrationEvent` envelope before publishing:

| Field | Description |
|-------|-------------|
| `ID` | Unique event UUID |
| `Topic` | Topic name (e.g. `contacts.v1.created`) |
| `SchemaVersion` | Payload schema version string |
| `OccurredAt` | RFC3339 timestamp of when the event was raised |
| `CorrelationID` | Propagated from the originating HTTP request (if present) |
| `CausationID` | ID of the command/event that caused this event |
| `Payload` | The `ContactEventPayload` object |
| `Metadata` | Additional string key/value pairs (schema_version, produced_at, etc.) |

The adapter layer serialises the payload to JSON and injects the W3C `traceparent` metadata key so
trace context propagates across the message boundary into consumer handlers.

---

## Subscribing to Contact Events

Any module that needs to react to contact lifecycle changes should declare a handler in its own
`adapter/event/` package and register it with the core messaging bus:

```go
bus.Subscribe("contacts.v1.created", handler)
bus.Subscribe("contacts.v1.updated", handler)
```

The handler receives a `bus.Message` where `Payload` is the raw JSON-encoded `ContactEventPayload`.
Extract trace context from `msg.Metadata["traceparent"]` before starting work to maintain
distributed trace continuity.
