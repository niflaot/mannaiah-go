# Mannaiah Go

[![Build Status](https://ci.niflaot.dev/api/badges/niflaot/mannaiah-go/status.svg)](https://ci.niflaot.dev/niflaot/mannaiah-go)
![Latest Version](https://img.shields.io/badge/latest-v3.0.3-0A66C2)

Mannaiah Go is a modular monolith built with Go, DDD, and hexagonal architecture. The repository is organized as a container workspace with independent modules under `module/`, composed by the `core` runtime.

Frontend integration is documented through the aggregated API docs at `/docs` and `/openapi.json`.

## Architecture

- `module/core`: shared runtime foundation (config, HTTP server, logging, database, messaging, cron, swagger aggregation).
- `module/auth`: authentication and authorization integration.
- `module/contacts`: contact domain.
- `module/orders`: order domain.
- `module/products`: product domain.
- `module/assets`: asset/storage domain.
- `module/falabella`: Falabella integration module.
- `module/syncrecord`: centralized sync execution registry and query API.
- `module/membership`: auditable consent/membership stamping module.
- `module/analytics`: ClickHouse analytics ingestion/bootstrap module for BI fact data.
- `module/email`: optional email delivery tracking and webhook module.
- `module/shipping`: carrier-agnostic shipping module (quotation, mark generation, dispatch batches, tracking).
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

### Analytics Bootstrap

- `ANALYTICS_ENABLED=true` enables ClickHouse analytics and integration consumers.
- Run `POST /analytics/seed` once to backfill ClickHouse fact tables from transactional data.
- Analytics no longer exposes CRM-facing routes as of `v2.0.0`; the retained module is write-focused infrastructure for BI.

### Email Tracking Pixel

- `EMAIL_TRACKING_BASE_URL` defines the public base URL for open-tracking pixel injection.
- When empty, runtime falls back to `https://<sender-domain>` from `EMAIL_SES_FROM_ADDRESS` / `EMAIL_SENDER_ADDRESS`.

### Assets JPG Worker

Use these env vars to convert tagged assets to `.jpg` through scheduled jobs:

- `ASSETS_JPG_WORKER_ENABLED`
- `ASSETS_JPG_WORKER_CRON`
- `ASSETS_JPG_WORKER_TAGS` (comma-separated tag names, for example `marketplaces,feeds`)
- `ASSETS_JPG_WORKER_BATCH_SIZE`
- `ASSETS_JPG_WORKER_JPEG_QUALITY`
- `ASSETS_JPG_WORKER_TIMEOUT_MS`

### Shipping (TCC)

- `SHIPPING_TCC_ENABLED=true` enables the TCC provider.
- `SHIPPING_TCC_SANDBOX=true` targets sandbox (`https://testsomos.tcc.com.co`).
- `SHIPPING_TCC_SANDBOX=false` targets production (`https://somos.tcc.com.co`).
- `SHIPPING_TCC_SANDBOX_ACCESS_TOKEN` is used when `SHIPPING_TCC_SANDBOX=true`.
- `SHIPPING_TCC_PRODUCTION_ACCESS_TOKEN` is used when `SHIPPING_TCC_SANDBOX=false`.
- `SHIPPING_TCC_COD_FEE_PERCENT` applies TCC COD fee percent over requested collection amount.
- `SHIPPING_QUOTATION_DISCOUNT_PERCENT` applies a global percentage discount to carrier quotations and exposes both full and discounted values in API responses.

## Testing

### Module Unit Tests

```bash
for module in module/core module/auth module/contacts module/orders module/assets module/products module/falabella module/syncrecord module/membership module/analytics module/email module/shipping; do
  (cd "$module" && go test ./...)
done
```

### Root E2E Tests

```bash
go test ./e2e -v -count=1
```

### Performance Benchmark

```bash
cd module/core
go test ./search -run '^$' -bench BenchmarkSpotlightFanout -benchmem -benchtime=100x -count=1
```

## Docker

### Build API

```bash
docker build -t mannaiah-go:local .
```

### Run API

```bash
docker run --rm -p 8080:8080 --env-file .env mannaiah-go:local
```

### Build Wiki (Docs)

```bash
docker build -f Dockerfile.docs -t mannaiah-go-wiki:local .
```

### Run Wiki

```bash
docker run --rm -p 3001:3001 mannaiah-go-wiki:local
```

Set `WIKI_PORT` in `.env` to change the host port used for the docs container.
Set `WIKI_ENABLED=true` to include the wiki in your deployment stack.

## Wiki / Documentation

The internal knowledge base is built with [Fumadocs](https://fumadocs.dev/) and lives in the `docs/`
directory as a Next.js 16 application.

### Local Development

```bash
cd docs
npm install
npm run dev
```

The wiki listens on `http://localhost:3001` by default (controlled by `WIKI_PORT`).

### Build for Production

```bash
cd docs
npm run build
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `WIKI_ENABLED` | `false` | Enable the wiki deployment in CI/CD and container stacks. |
| `WIKI_PORT` | `3001` | Port the docs container listens on. |

## CI/CD

- CI/CD is managed by Drone via `.drone.yml`.
- Validation pipeline runs module tests, e2e tests, and performance benchmark checks.
- Docker images are published to Nexus registry:
  - Registry: `docker.niflaot.dev`
  - API repository: `fl-docker/mannaiah-go`
  - Wiki repository: `fl-docker/mannaiah-go-wiki` (published when `WIKI_ENABLED=true`)
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
