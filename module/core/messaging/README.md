# Messaging Module

`module/core/messaging` provides replaceable messaging abstractions and Watermill adapters for in-memory integration events.

## Packages
- `bus`: transport-agnostic envelope and ports.
- `platform`: runtime config and retry classification helpers.
- `watermill`: in-memory Watermill facade and nested adapter packages.

## Goals
- Preserve hexagonal boundaries across modules.
- Keep Watermill isolated to infrastructure adapters.
- Enable future broker replacement without changing module application/domain logic.

## Key Methods / Endpoints / Events
- Methods:
  - `bus.Publisher.Publish(ctx, msg)`
  - `bus.Registrar.AddHandler(topic, handler)`
  - `watermill.NewInMemoryPlatform(cfg, providedLogger)`
  - `(*watermill.InMemoryPlatform).Publisher()`
  - `(*watermill.InMemoryPlatform).Registrar()`
  - `(*watermill.InMemoryPlatform).Run(ctx)`
  - `(*watermill.InMemoryPlatform).Close()`
- Endpoints: none in this module.
- Events: integration event envelopes are routed between modules via topics and metadata contracts.
