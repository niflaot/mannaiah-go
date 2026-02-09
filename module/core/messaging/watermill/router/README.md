# Watermill Router Package

`watermill/router` wires router lifecycle, handler registration, retry, and DLQ behavior.

## Responsibilities
- Build in-memory messaging platform runtime.
- Expose abstract `bus.Publisher` and `bus.Registrar` ports.
- Register handlers with retry/recover/DLQ middleware chain.

## Key Methods / Endpoints / Events
- Methods:
  - `router.NewInMemoryPlatform(cfg, providedLogger)`
  - `(*router.InMemoryPlatform).Publisher()`
  - `(*router.InMemoryPlatform).Registrar()`
  - `(*router.InMemoryPlatform).Run(ctx)`
  - `(*router.InMemoryPlatform).Running()`
  - `(*router.InMemoryPlatform).Close()`
- Endpoints: none in this package.
- Events: integration messages are consumed and can be dead-lettered on failure.
