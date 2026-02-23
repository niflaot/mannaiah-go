# Telemetry Package

`telemetry` provides OpenTelemetry tracing and Prometheus instrumentation for the core runtime.

## Responsibilities
- Initialize process-wide tracer provider and propagation (`traceparent`).
- Expose Prometheus metrics handler for `/metrics`.
- Provide Fiber middleware for HTTP request tracing + metrics.
- Record dependency and messaging metrics with low-cardinality labels.
- Collect SQL connection-pool stats on a periodic ticker.

## Key Methods
- `telemetry.Init(ctx, cfg, logger)`
- `(*telemetry.Provider).Shutdown(ctx)`
- `(*telemetry.Provider).MetricsHandler()`
- `(*telemetry.Provider).MetricsPath()`
- `(*telemetry.Provider).HTTPMiddleware()`
- `(*telemetry.Provider).StartSQLStatsCollector(db)`
- `telemetry.StartSpan(ctx, tracerName, spanName, opts...)`
- `telemetry.EndSpan(span, err)`
- `telemetry.TraceparentFromContext(ctx)`
- `telemetry.ContextWithTraceparent(ctx, traceparent)`
- `telemetry.RecordDependency(dependency, operation, startedAt, err)`
- `telemetry.RecordMessaging(topic, operation, startedAt, err)`
- `telemetry.IncMessagingDLQ(topic)`

## Labels and Cardinality
- HTTP: `method`, `route`, `status_code`
- Dependency: `dependency`, `operation`, `result`
- Messaging: `topic`, `operation`, `result`

Never include IDs, emails, query strings, payload fragments, or secrets in labels/attributes.
