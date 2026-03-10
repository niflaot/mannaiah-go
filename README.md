# Mannaiah Go

[![Build Status](https://ci.momlesstomato.dev/api/badges/flockstore/mannaiah-go/status.svg)](https://ci.momlesstomato.dev/flockstore/mannaiah-go)
![Latest Version](https://img.shields.io/badge/latest-v1.3.2-0A66C2)

Mannaiah Go is a modular monolith built with Go, DDD, and hexagonal architecture. The repository is organized as a container workspace with independent modules under `module/`, composed by the `core` runtime.

## Architecture

- `module/core`: shared runtime foundation (config, HTTP server, logging, database, messaging, cron, swagger aggregation).
- `module/auth`: authentication and authorization integration.
- `module/contacts`: contact domain.
- `module/orders`: order domain.
- `module/products`: product domain.
- `module/assets`: asset/storage domain.
- `module/falabella`: Falabella integration module.
- `module/woocommerce`: WooCommerce integration module.
- `e2e/`: root end-to-end validation flows.

## Key Runtime Endpoints

- `GET /status`: health/status endpoint.
- `GET /metrics`: Prometheus metrics endpoint (protect this at ingress/network layer).
- `GET /openapi.json`: aggregated OpenAPI document from core + modules.
- `GET /docs`: API documentation UI.

## Local Development

### Prerequisites

- Go `1.25.5`
- Docker (optional, for containerized runs)

### Start Locally

```bash
cp .env.example .env
go run ./module/core/cmd/api
```

The API listens on `CORE_HOST:CORE_PORT` (`0.0.0.0:8080` by default).

### Assets JPG Worker

Use these env vars to convert tagged assets to `.jpg` through scheduled jobs:

- `ASSETS_JPG_WORKER_ENABLED`
- `ASSETS_JPG_WORKER_CRON`
- `ASSETS_JPG_WORKER_TAGS` (comma-separated tag names, for example `marketplaces,feeds`)
- `ASSETS_JPG_WORKER_BATCH_SIZE`
- `ASSETS_JPG_WORKER_JPEG_QUALITY`
- `ASSETS_JPG_WORKER_TIMEOUT_MS`

## Testing

### Module Unit Tests

```bash
for module in module/core module/auth module/contacts module/orders module/assets module/products module/falabella module/woocommerce; do
  (cd "$module" && go test ./...)
done
```

### Root E2E Tests

```bash
go test ./e2e -v -count=1
```

### WooCommerce Benchmark

```bash
cd module/woocommerce
go test ./application/contact/service -run '^$' -bench BenchmarkProcessCommands -benchmem -benchtime=100x -count=1
```

## Docker

### Build

```bash
docker build -t mannaiah-go:local .
```

### Run

```bash
docker run --rm -p 8080:8080 --env-file .env mannaiah-go:local
```

## CI/CD

- CI/CD is managed by Drone via `.drone.yml`.
- Validation pipeline runs module tests, e2e tests, and WooCommerce benchmark checks.
- Docker images are published to Nexus registry:
  - Registry: `docker.momlesstomato.dev`
  - Repository: `fl-docker/mannaiah-go`
- Drone secrets required for publish:
  - `nexus_username`
  - `nexus_password`

## Observability

- Metrics:
  - Prometheus exposition is available at `GET /metrics`.
  - Keep `HTTP_PREFORK=false` when you need single-process metric accuracy.
  - Restrict `/metrics` to internal scrapers using proxy/network controls.
- Tracing:
  - Distributed tracing uses OpenTelemetry with OTLP gRPC export support.
  - Configure collector endpoint with `TELEMETRY_OTLP_ENDPOINT`.
  - Trace context propagation uses W3C `traceparent` across HTTP and messaging.

### Telemetry Environment Variables

- `TELEMETRY_ENABLED`
- `TELEMETRY_SERVICE_NAME`
- `TELEMETRY_SERVICE_VERSION`
- `TELEMETRY_TRACES_ENABLED`
- `TELEMETRY_TRACES_EXPORTER`
- `TELEMETRY_OTLP_ENDPOINT`
- `TELEMETRY_OTLP_INSECURE`
- `TELEMETRY_TRACES_SAMPLER`
- `TELEMETRY_TRACES_SAMPLER_RATIO`
- `TELEMETRY_METRICS_ENABLED`
- `TELEMETRY_METRICS_PATH`
- `TELEMETRY_DB_STATS_INTERVAL_MS`
