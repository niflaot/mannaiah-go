# Circuit Breaker Package

`circuitbreaker` provides a reusable abstraction over `github.com/sony/gobreaker` for fault isolation and fail-fast behavior on unstable dependencies.

## Responsibilities
- Build configurable circuit breaker instances.
- Expose an abstract breaker interface for adapters and services.
- Provide state and open-error detection helpers.

## Key Methods / Endpoints / Events
- Methods:
  - `circuitbreaker.New(cfg, logger)`
  - `circuitbreaker.NewBreaker(cfg, logger)`
  - `(*circuitbreaker.Service).Execute(operation)`
  - `(*circuitbreaker.Service).State()`
  - `(*circuitbreaker.Service).IsOpenError(err)`
- Endpoints: none in this package.
- Events: none in this package.
