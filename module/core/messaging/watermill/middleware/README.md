# Watermill Middleware Package

`watermill/middleware` provides transport-level Watermill middleware behavior.

## Responsibilities
- Correlation id creation and propagation.
- Retry policy with non-retriable classification support.
- Dead-letter queue (DLQ) publishing and metadata enrichment.

## Key Methods / Endpoints / Events
- Methods:
  - `middleware.AddRouterMiddlewares(router)`
  - `middleware.Correlation(next)`
  - `middleware.NewRetry(cfg, logger)`
  - `middleware.NewDLQ(topic, suffix, publisher)`
  - `middleware.ShouldRetry(params)`
- Endpoints: none in this package.
- Events: failed messages are published to `topic + suffix` with `dlq_*` metadata.
