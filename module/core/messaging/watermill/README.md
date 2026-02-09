# Messaging Watermill Package

`watermill` provides the in-memory Watermill composition facade behind `messaging/bus` ports.

## Nested Packages
- `watermill/router`: in-memory platform and registrar wiring.
- `watermill/publisher`: `bus.Publisher` adapter.
- `watermill/middleware`: correlation, retry, and dead-letter middleware.
- `watermill/pubsub`: GoChannel pubsub factory.
- `watermill/logger`: Zap logger adapter.
- `watermill/internal/correlation`: internal context propagation helpers.

## Usage Rules
- Import Watermill only from this package.
- Keep domain/application modules dependent on `messaging/bus` interfaces only.
- Register module handlers via `bus.Registrar` in composition root wiring.

## Key Methods / Endpoints / Events
- Methods:
  - `watermill.NewInMemoryPlatform(cfg, providedLogger)`
  - `watermill.NewPublisherAdapter(publisher)`
  - `(*watermill.InMemoryPlatform).Publisher()`
  - `(*watermill.InMemoryPlatform).Registrar()`
  - `(*watermill.InMemoryPlatform).Run(ctx)`
  - `(*watermill.InMemoryPlatform).Running()`
  - `(*watermill.InMemoryPlatform).Close()`
- Endpoints: none in this package.
- Events: integration messages are routed and, on failure, dead-lettered to `topic + suffix`.
