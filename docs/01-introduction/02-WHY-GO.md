# Why Go?

## Performance Without Ceremony

Go compiles to native machine code with minimal runtime overhead. For a service that acts as an
integration hub — receiving webhooks, syncing product catalogues, processing async messaging, and
serving REST APIs — raw throughput and low latency matter. Go delivers this without JVM tuning,
GC configuration, or interpreter warm-up time.

## Concurrency as a First-Class Citizen

Mannaiah operates many concurrent workflows simultaneously:

- JPG transcoding workers scanning batches of assets on a schedule.
- Marketplace sync pipelines that fan out across thousands of product records.
- Async messaging handlers processing integration events from multiple modules.
- Cron-based sweep tasks running alongside live HTTP traffic.

Go's goroutine model and channel primitives make expressing this concurrency idiomatic and readable.
A goroutine starts at roughly 2 KB of stack, so Mannaiah can run thousands of them concurrently
without memory pressure. The `sync` and `context` packages provide coordination without third-party
dependencies.

## Deployment Simplicity

Go produces a single, statically-linked binary. The production Dockerfile copies one executable:

```dockerfile
COPY --from=builder /out/mannaiah-api /usr/local/bin/mannaiah-api
```

No runtime installation, no shared library resolution, no virtual environment to activate.
The container image is small, reproducible, and hard to break.

## Type Safety and Explicit Error Handling

Go's type system catches integration mismatches at compile time. In a codebase that translates
between multiple marketplace data models (Falabella SKU formats, WooCommerce structures, internal
PIM schemas), incorrect mappings are compile-time errors, not silent data corruption in production.

Go's explicit error handling returns `(T, error)` rather than throwing exceptions, which forces
every caller to reason about failure paths. For a system integrating with unreliable external APIs,
this is a feature, not a limitation.

## Standard Library Depth

Go's standard library covers the majority of what Mannaiah needs: HTTP server/client, JSON
marshalling, cryptographic primitives, context propagation, and concurrency utilities. This reduces
the dependency surface area, lowering supply-chain risk and upgrade friction.

## Ecosystem Fit

The observability and persistence ecosystem is mature and well-suited to Mannaiah's requirements:

- **OpenTelemetry Go SDK** — distributed tracing and metrics.
- **Prometheus `client_golang`** — Prometheus metrics exposition.
- **GORM** — SQL persistence with multi-driver support (SQLite for development, PostgreSQL/MySQL for production).
- **Watermill** — pub/sub messaging and async event routing.
- **Fiber** — high-throughput HTTP serving.

All operate via explicit adapters in the hexagonal architecture, meaning they can be swapped
without touching domain logic.

## Summary

Go was selected because it aligns with the operational profile of Mannaiah: a concurrent, I/O-heavy
integration hub that needs to be deployed as a small, reliable binary and maintained by a small,
focused team. The language's simplicity keeps cognitive overhead low; its concurrency model matches
the problem domain; its tooling makes testing, profiling, and cross-compilation straightforward.
