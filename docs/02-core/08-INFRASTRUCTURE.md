# Infrastructure Utilities

## Cron Scheduler

The `cron` package wraps `robfig/cron v3` and integrates with the core lifecycle so scheduled jobs
start and stop with the application.

```go
cron.Add("0 * * * *", func() {
    // runs every hour
})
```

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CRON_LOCATION` | `UTC` | IANA timezone for schedule evaluation |
| `CRON_WITH_SECONDS` | `false` | Allow 6-field cron expressions (with seconds) |

---

## Circuit Breaker

The `circuitbreaker` package provides `circuitbreaker.Service`, a named state machine that wraps
any fallible operation.

### States

| State | Behaviour |
|-------|-----------|
| `Closed` | Requests pass through normally |
| `HalfOpen` | A limited number of trial requests are allowed |
| `Open` | All requests fail immediately with `ErrUnavailable` |

### Fields

| Field | Description |
|-------|-------------|
| `Name` | Human-readable identifier (used in metrics/spans) |
| `MaxRequests` | Trial requests permitted in the HalfOpen state |
| `Interval` | Closed-state error-counter reset window |
| `Timeout` | Duration to stay Open before transitioning to HalfOpen |
| `FailureThreshold` | Consecutive failures required to trip to Open |

---

## Startup / Composition Root

Modules register themselves with the core composition root at startup:

```go
runtime.RegisterRoutes(module)    // mounts the module's HTTP routes
runtime.AddOpenAPISpec(module)    // merges the module's OpenAPI spec
```

The composition root automatically exposes:

| Endpoint | Description |
|----------|-------------|
| `GET /status` | Liveness check — returns `{"status":"ok"}` |
| `GET /openapi.json` | Aggregated OpenAPI 3.x specification |

---

## OpenAPI Aggregation

Each module provides its own OpenAPI spec artifact. Core merges all registered specs into a single
document served at `/openapi.json`:

```go
swagger.Document.Merge(moduleSpec)
```

Modules call `runtime.AddOpenAPISpec(module)` during initialization; core calls
`swagger.Document.Merge` for each registered spec after all modules have loaded.
