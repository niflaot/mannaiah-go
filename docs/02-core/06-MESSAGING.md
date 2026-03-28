# Messaging

The messaging system provides a durable, in-process event bus with retry semantics, dead-letter
queuing, and distributed trace propagation.

## Architecture

```
Publisher ──► InMemoryPlatform ──► [retry loop] ──► Handler
                                                        │
                                              (error)   ▼
                                                      DLQ topic
```

The transport is Watermill's `GoChannel` backend. Events are dispatched asynchronously; a
configurable retry strategy re-delivers failed messages before routing to the dead-letter queue.

## Trace Propagation

Every published message includes a `traceparent` metadata key conforming to the W3C Trace Context
spec. Handlers extract the trace context before processing, ensuring a single distributed trace
spans HTTP requests, event publishing, and downstream consumers.

## Key Types

| Type | Description |
|------|-------------|
| `Publisher` | Publishes messages to a named topic |
| `Registrar` | Subscribes handlers to topics |
| `Message` | Envelope with payload, metadata, and optional DLQ |
| `Handler` | `func(ctx, msg) error` consumer contract |

## Metadata Keys

| Key | Description |
|-----|-------------|
| `MetadataEventID` | Unique event identifier |
| `MetadataTraceparent` | W3C traceparent header value |
| `MetadataEventTopic` | Original topic name |
| `MetadataDLQOriginalTopic` | Topic before DLQ routing |
| `MetadataDLQOriginalError` | Error that triggered DLQ routing |
| `MetadataDLQAttempt` | Retry count before DLQ routing |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MESSAGING_BUFFER_SIZE` | `1024` | In-memory channel buffer depth |
| `MESSAGING_RETRY_MAX` | `3` | Max delivery attempts per message |
| `MESSAGING_RETRY_INITIAL_INTERVAL_MS` | `500` | Initial retry back-off |
| `MESSAGING_RETRY_MAX_INTERVAL_MS` | `5000` | Maximum retry back-off |
| `MESSAGING_RETRY_MULTIPLIER` | `2.0` | Back-off multiplier |
| `MESSAGING_DLQ_SUFFIX` | `.dlq` | Suffix appended to topic name for DLQ |
