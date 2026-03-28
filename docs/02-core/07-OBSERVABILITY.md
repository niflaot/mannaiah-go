# Observability

Mannaiah exposes three observability signals: structured logs, distributed traces, and Prometheus
metrics.

## Structured Logging

The `logger` package wraps Uber Zap. Log mode is controlled by `LOG_FORMAT`:

- `pretty` — human-readable coloured output (development)
- `json` — structured JSON lines (production/log aggregators)

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Minimum log level (`debug`, `info`, `warn`, `error`) |
| `LOG_FORMAT` | `pretty` | Output format (`pretty` or `json`) |

---

## Distributed Tracing

Tracing uses the OpenTelemetry Go SDK. Spans are exported via OTLP over gRPC to a configured
collector endpoint.

Trace context propagates using the **W3C `traceparent`** header across:
- Inbound/outbound HTTP requests
- Published and consumed messaging events
- Dependency calls (database, cache, storage)

The exporter is **fail-open**: if the OTLP backend is unreachable, the service starts and handles
requests normally.

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | _(empty)_ | OTLP gRPC collector endpoint |
| `OTEL_SERVICE_NAME` | `mannaiah` | Service name reported on spans |
| `OTEL_ENVIRONMENT` | `development` | Deployment environment tag |

---

## Prometheus Metrics

The core module exposes `GET /metrics` for Prometheus scraping. Only **low-cardinality** labels
must be used; PII, raw IDs, query strings, and payload fragments must never appear in metric
attributes.

> **Important:** `HTTP_PREFORK=false` is required when Prometheus is enabled. The prefork process
> model creates multiple OS processes that cannot share the in-process metric registry.

### Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_ENABLED` | `true` | Enable `/metrics` endpoint |
| `METRICS_NAMESPACE` | `mannaiah` | Metric name prefix |
